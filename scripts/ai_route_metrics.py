#!/usr/bin/env python3
"""Summarize redacted, per-request AI routing measurements from a UTF-8 CSV."""

from __future__ import annotations

import argparse
import csv
import json
import math
import sys
from collections import defaultdict
from pathlib import Path
from typing import TextIO


GROUP_FIELDS = ("service", "scenario", "kind", "connection", "variant")
TIMINGS = ("connect_ms", "tls_ready_ms", "ttfb_ms", "first_event_ms")
CHOICES = {
    "service": {"claude", "chatgpt", "gemini", "antigravity"},
    "kind": {"request", "stream"},
    "connection": {"cold", "warm", "unknown"},
    "variant": {"A", "B"},
    "outcome": {"complete", "failed", "interrupted", "cancelled"},
}

USAGE = """Input header (one row per initiated business request):
  service,scenario,variant,connection,kind,outcome,connect_ms,tls_ready_ms,ttfb_ms,first_event_ms

Use service claude/chatgpt/gemini/antigravity and a short workload label like
ide-completion. Alternate baseline A and candidate B in the same network window,
with the same client, model, input/output budget and concurrency per scenario.
Use cold/warm for observed new/reused connections, or unknown. Measure IDE
operations in the IDE itself. Include timeouts and cancellations in the input.

Timing fields are cumulative milliseconds from request start to TCP connection,
completed TLS handshake, first HTTP response byte and first meaningful model
output, respectively. For curl convert time_connect/time_appconnect/
time_starttransfer from seconds. Leave unobserved timings blank.

A complete request has a valid business result. A complete stream also ends
normally after output. Mark failures before first output as failed, failures
after output as interrupted, and intentional cancellation as cancelled.
Use kind=request for non-streaming requests, or stream for streaming requests.

Success uses all attempts. Stream interruption uses completed and interrupted
streams with observed output. Cancellations are counted separately. Rate
intervals are 95% Wilson intervals. P95 uses nearest rank on completed requests
with the respective timing. Each timing includes its own sample count."""


def rate(successes: int, total: int) -> dict:
    if not total:
        return {"n": total, "count": successes, "pct": None, "ci95_pct": None}
    p = successes / total
    z = 1.959963984540054
    denominator = 1 + z**2 / total
    center = (p + z**2 / (2 * total)) / denominator
    radius = z * math.sqrt(p * (1 - p) / total + z**2 / (4 * total**2)) / denominator
    bounds = [max(0.0, center - radius), min(1.0, center + radius)]
    return {
        "n": total,
        "count": successes,
        "pct": round(100 * p, 3),
        "ci95_pct": [round(100 * b, 3) for b in bounds],
    }


def read_timings(row: dict, line: int) -> dict[str, float]:
    timings = {}
    for field in TIMINGS:
        if not row[field].strip():
            continue
        try:
            value = float(row[field])
        except ValueError:
            raise ValueError(f"line {line}: invalid {field}") from None
        if not math.isfinite(value) or value < 0:
            raise ValueError(f"line {line}: invalid {field}")
        timings[field] = value
    elapsed = list(timings.values())
    if elapsed != sorted(elapsed):
        raise ValueError(f"line {line}: cumulative timings must be nondecreasing")
    if "first_event_ms" in timings and (row["kind"] != "stream" or row["outcome"] == "failed"):
        raise ValueError(f"line {line}: unexpected first_event_ms")
    return timings


def measurements(source: TextIO) -> list[dict]:
    reader = csv.DictReader(source, strict=True)
    required = {*GROUP_FIELDS, "outcome", *TIMINGS}
    header = reader.fieldnames or []
    if set(header) != required or len(header) != len(required):
        raise ValueError("expected the CSV header shown in --help")
    groups = defaultdict(list)
    for row in reader:
        line = reader.line_num
        if None in row or None in row.values():
            raise ValueError(f"line {line}: incorrect number of fields")
        for field, options in CHOICES.items():
            if row[field] not in options:
                raise ValueError(f"line {line}: invalid {field}")
        if not row["scenario"].strip():
            raise ValueError(f"line {line}: missing scenario")
        if row["kind"] == "request" and row["outcome"] == "interrupted":
            raise ValueError(f"line {line}: interrupted applies only to streams")
        key = tuple(row[field] for field in GROUP_FIELDS)
        groups[key].append((row["outcome"], read_timings(row, line)))
    if not groups:
        raise ValueError("CSV has no request samples")

    report = []
    for key, samples in sorted(groups.items()):
        group = dict(zip(GROUP_FIELDS, key))
        complete = [timings for outcome, timings in samples if outcome == "complete"]
        interrupted = sum(outcome == "interrupted" for outcome, _ in samples)
        group["success"] = rate(len(complete), len(samples))
        group["failed"] = sum(outcome == "failed" for outcome, _ in samples)
        group["cancelled"] = sum(outcome == "cancelled" for outcome, _ in samples)
        if group["kind"] == "stream":
            group["stream_interruption"] = rate(interrupted, len(complete) + interrupted)
        latency = {}
        for field in TIMINGS:
            values = sorted(t[field] for t in complete if field in t)
            rank = math.ceil(0.95 * len(values))
            latency[field] = {"n": len(values), "p95": values[rank - 1] if values else None}
        group["completed_latency_ms"] = latency
        report.append(group)
    return report


def main() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__, epilog=USAGE, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("csv", type=Path, help="redacted UTF-8 measurement CSV")
    args = parser.parse_args()
    try:
        with args.csv.open(encoding="utf-8-sig", newline="") as source:
            report = measurements(source)
    except (OSError, ValueError, csv.Error) as exc:
        parser.exit(2, f"{parser.prog}: {exc}\n")
    json.dump(report, sys.stdout, indent=2)
    print()


if __name__ == "__main__":
    main()
