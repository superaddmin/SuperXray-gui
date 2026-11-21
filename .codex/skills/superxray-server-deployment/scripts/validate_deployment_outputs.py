#!/usr/bin/env python3
"""Validate a redacted SuperXray deployment manifest without printing secrets."""

from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import unquote, urlparse


def error(message: str) -> int:
    print(f"ERROR: {message}", file=sys.stderr)
    return 1


def require(mapping: dict, key: str, path: str):
    value = mapping.get(key)
    if value is None or value == "":
        raise ValueError(f"{path}.{key} is required")
    return value


def placeholder(value, prefix: str = "") -> bool:
    return isinstance(value, str) and re.fullmatch(
        rf"<{prefix}[a-z][a-z0-9_]*>", value, re.IGNORECASE | re.ASCII
    ) is not None


def pick_entry(mapping: dict, *names: str):
    for name in names:
        value = mapping.get(name)
        if isinstance(value, dict):
            return value
    raise ValueError(f"subscriptions must contain one of: {', '.join(names)}")


def validate(manifest: dict, *, require_redacted: bool = True) -> list[str]:
    server = require(manifest, "server", "manifest")
    service = require(manifest, "service", "manifest")
    subscriptions = require(manifest, "subscriptions", "manifest")
    if not all(isinstance(value, dict) for value in (server, service, subscriptions)):
        raise ValueError("server, service and subscriptions must be objects")
    socks5 = manifest.get("socks5") or subscriptions.get("socks5LongTerm") or subscriptions.get("socks5")
    if not isinstance(socks5, dict):
        raise ValueError("socks5 must be an object")
    if require_redacted and "host" in server and not placeholder(server["host"]):
        raise ValueError("server.host must be redacted")

    arch = require(server, "arch", "server")
    if arch not in {"amd64", "arm64", "<amd64|arm64>"}:
        raise ValueError("server.arch must be amd64 or arm64")
    if require(service, "systemd", "service") not in {"active", "<active|inactive|failed>"}:
        raise ValueError("service.systemd must be active for a delivered deployment")
    require(service, "xray", "service")
    ports = require(service, "ports", "service")
    if not isinstance(ports, dict):
        raise ValueError("service.ports must be an object")

    now = datetime.now(timezone.utc)
    summaries: list[str] = []
    expiry_values: dict[str, int] = {}
    three_entry = pick_entry(subscriptions, "threeMonth", "3-month", "clashThreeMonth")
    six_entry = pick_entry(subscriptions, "sixMonth", "6-month", "clashSixMonth")
    for name, entry in (("threeMonth", three_entry), ("sixMonth", six_entry)):
        expiry_ms = require(entry, "expiryTimeMs", f"subscriptions.{name}")
        if type(expiry_ms) is not int or expiry_ms <= 0:
            raise ValueError(f"subscriptions.{name}.expiryTimeMs must be a positive integer in milliseconds")
        expiry_values[name] = expiry_ms
        expires_utc = require(entry, "expiresUtc", f"subscriptions.{name}")
        try:
            parsed = datetime.fromisoformat(str(expires_utc).replace("Z", "+00:00"))
        except ValueError as exc:
            raise ValueError(f"subscriptions.{name}.expiresUtc must be an ISO 8601 timestamp") from exc
        if parsed.tzinfo is None:
            raise ValueError(f"subscriptions.{name}.expiresUtc must include a timezone")
        try:
            parsed = parsed.astimezone(timezone.utc)
            parsed_ms = round(parsed.timestamp() * 1000)
        except (ValueError, OSError, OverflowError) as exc:
            raise ValueError(f"subscriptions.{name}.expiresUtc is outside the supported date range") from exc
        if abs(parsed_ms - expiry_ms) > 1000:
            raise ValueError(f"subscriptions.{name}.expiresUtc does not match expiryTimeMs")
        if parsed <= now:
            raise ValueError(f"subscriptions.{name} is already expired")
        url = entry.get("clashUrl") or entry.get("clash") or entry.get("clashSubscription")
        if not url:
            raise ValueError(f"subscriptions.{name}.clashUrl is required")
        redacted_url = placeholder(url, "REDACTED_")
        if not isinstance(url, str) or not (url.startswith(("https://", "http://")) or redacted_url):
            raise ValueError(f"subscriptions.{name}.clashUrl must be an HTTP(S) URL or redacted placeholder")
        if require_redacted and not redacted_url:
            raise ValueError(f"subscriptions.{name}.clashUrl must be redacted")
        diagnostic = require(entry, "diagnostic", f"subscriptions.{name}")
        if not isinstance(diagnostic, dict):
            raise ValueError(f"subscriptions.{name}.diagnostic must be an object")
        output_nodes = diagnostic.get("outputNodes")
        skipped_nodes = diagnostic.get("skippedNodes")
        for count in (output_nodes, skipped_nodes):
            if type(count) is not int and not (isinstance(count, float) and count.is_integer()):
                raise ValueError(f"subscriptions.{name}.diagnostic counts must be integers")
        if output_nodes < 1 or skipped_nodes != 0:
            raise ValueError(f"subscriptions.{name}.diagnostic must have outputNodes >= 1 and skippedNodes == 0")
        summaries.append(f"{name}=ok")

    if expiry_values["sixMonth"] <= expiry_values["threeMonth"]:
        raise ValueError("subscriptions.sixMonth.expiryTimeMs must be later than threeMonth.expiryTimeMs")

    host = require(socks5, "host", "socks5")
    port = require(socks5, "port", "socks5")
    username = require(socks5, "username", "socks5")
    password = require(socks5, "password", "socks5")
    uri = require(socks5, "uri", "socks5")
    if require_redacted and not all(placeholder(value) for value in (host, username, password)):
        raise ValueError("socks5 host, username and password must be redacted")
    if not isinstance(port, (int, str)) or not str(port).isascii() or not str(port).isdigit() or not 1 <= int(port) <= 65535:
        raise ValueError("socks5.port must be between 1 and 65535")
    if not str(uri).startswith("socks5://"):
        raise ValueError("socks5.uri must use the socks5:// scheme")
    try:
        parsed_uri = urlparse(str(uri))
        uri_port = parsed_uri.port
    except ValueError as exc:
        raise ValueError("socks5.uri must contain a valid host and port") from exc
    if parsed_uri.hostname is None or parsed_uri.hostname.lower() != str(host).lower() or uri_port != int(port):
        raise ValueError("socks5.uri host/port must match socks5.host/port")
    if parsed_uri.username is None or parsed_uri.password is None:
        raise ValueError("socks5.uri must contain username and password")
    if require_redacted and (parsed_uri.path not in ("", "/") or parsed_uri.query or parsed_uri.fragment):
        raise ValueError("socks5.uri must not contain extra data in a redacted manifest")
    if unquote(parsed_uri.username) != str(username) or unquote(parsed_uri.password) != str(password):
        raise ValueError("socks5.uri credentials must match socks5.username/password")
    return summaries + ["socks5=ok"]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--allow-secrets", action="store_true", help="validate a protected local manifest containing real values")
    args = parser.parse_args()
    try:
        manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
        if not isinstance(manifest, dict):
            return error("manifest root must be a JSON object")
        summaries = validate(manifest, require_redacted=not args.allow_secrets)
    except (OSError, UnicodeError, json.JSONDecodeError, RecursionError):
        return error("cannot read manifest as UTF-8 JSON")
    except (ValueError, TypeError) as exc:
        return error(str(exc))
    print("deployment_manifest=ok")
    for summary in summaries:
        print(summary)
    print("credentials=redacted-output-only")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
