#!/usr/bin/env python3
"""Scan tracked and untracked workspace files for high-risk operational secrets."""

from __future__ import annotations

import argparse
import os
import pathlib
import re
import subprocess
import sys
from urllib.parse import urlparse
from dataclasses import dataclass


SKIP_DIRS = {
    ".git",
    ".venv",
    "bin",
    "dist",
    "node_modules",
    "playwright-report",
    "release",
    "test-results",
}
SKIP_PATH_PREFIXES = {
    "web/ui/",
}
DOC_SUFFIXES = {".md", ".txt", ".env", ".toml", ".yaml", ".yml", ".sh", ".conf", ".cfg", ".ini"}

TEXT_SUFFIXES = {
    "",
    ".cfg",
    ".conf",
    ".env",
    ".example",
    ".go",
    ".ini",
    ".js",
    ".json",
    ".md",
    ".py",
    ".sh",
    ".toml",
    ".ts",
    ".txt",
    ".vue",
    ".yaml",
    ".yml",
}


@dataclass(frozen=True)
class Pattern:
    name: str
    regex: re.Pattern[str]
    guidance: str
    suffixes: frozenset[str] | None = None


PATTERNS = [
    Pattern(
        "private-key",
        re.compile(r"-----BEGIN (?:RSA|DSA|EC|OPENSSH|PRIVATE) PRIVATE KEY-----"),
        "Remove private keys from the repository and rotate the key.",
    ),
    Pattern(
        "github-token",
        re.compile(r"\bgh[pousr]_[A-Za-z0-9_]{30,}\b"),
        "Remove GitHub tokens and rotate the credential.",
    ),
    Pattern(
        "aws-access-key",
        re.compile(r"\bAKIA[0-9A-Z]{16}\b"),
        "Remove AWS access keys and rotate the credential.",
    ),
    Pattern(
        "openai-api-key",
        re.compile(r"\bsk-(?:proj-)?[A-Za-z0-9_-]{20,}\b"),
        "Remove OpenAI API keys and rotate the credential.",
    ),
    Pattern(
        "subscription-url",
        re.compile(r"https?://[^\s`\"']+/(?:sub|clash|json)/[A-Za-z0-9_-]{8,}", re.I),
        "Replace subscription URLs with placeholders.",
    ),
    Pattern(
        "sub-id",
        re.compile(r"\bsubId[：:\s]+[A-Za-z0-9_-]{10,}\b", re.I),
        "Replace subscription IDs with <SUB_ID>.",
        frozenset(DOC_SUFFIXES),
    ),
    Pattern(
        "client-uuid",
        re.compile(
            r"\b(?:client\s+UUID|uuid)[：:\s]+[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b",
            re.I,
        ),
        "Replace client UUIDs with <CLIENT_UUID>.",
    ),
    Pattern(
        "socks-uri-credential",
        re.compile(r"\bsocks5h?://[^:\s/@]+:[^@\s/]+@[A-Za-z0-9.-]+:\d{2,5}\b", re.I),
        "Replace proxy URIs with host/user/password placeholders.",
    ),
    Pattern(
        "proxy-credential-tuple",
        re.compile(r"\b(?:\d{1,3}\.){3}\d{1,3}:\d{2,5}:[A-Za-z0-9._-]{6,}:[A-Za-z0-9._-]{6,}\b"),
        "Replace host:port:user:password proxy tuples with placeholders.",
    ),
    Pattern(
        "explicit-proxy-password",
        re.compile(r"^\s*(?:[-*]\s*)?password[：:\s]+[`'\"]?[A-Za-z0-9._!@#$%^&*+=-]{8,}[`'\"]?\s*$", re.I),
        "Do not store real proxy or service passwords in docs/scripts.",
        frozenset(DOC_SUFFIXES),
    ),
]


def command(root: pathlib.Path, args: list[str]) -> str:
    result = subprocess.run(
        args,
        cwd=root,
        text=True,
        encoding="utf-8",
        errors="replace",
        capture_output=True,
    )
    if result.returncode not in (0, 1):
        raise RuntimeError(result.stderr or result.stdout)
    return result.stdout


def git_files(root: pathlib.Path) -> list[pathlib.Path]:
    output = command(root, ["git", "ls-files", "--cached", "--others", "--exclude-standard"])
    return [root / line for line in output.splitlines() if line.strip()]


def all_files(root: pathlib.Path) -> list[pathlib.Path]:
    return [path for path in root.rglob("*") if path.is_file()]


def should_scan(root: pathlib.Path, path: pathlib.Path) -> bool:
    try:
        rel = path.relative_to(root)
    except ValueError:
        return False
    if any(part in SKIP_DIRS for part in rel.parts):
        return False
    rel_posix = rel.as_posix()
    if any(rel_posix.startswith(prefix) for prefix in SKIP_PATH_PREFIXES):
        return False
    if rel_posix == "scripts/secret_scan.py":
        return False
    if path.suffix.lower() not in TEXT_SUFFIXES:
        return False
    return True


def scan_file(root: pathlib.Path, path: pathlib.Path) -> list[str]:
    if not path.exists():
        return []
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return []
    rel = path.relative_to(root).as_posix()
    findings: list[str] = []
    for lineno, line in enumerate(text.splitlines(), start=1):
        for pattern in PATTERNS:
            if pattern.suffixes is not None and path.suffix.lower() not in pattern.suffixes:
                continue
            lowered = line.lower()
            if pattern.name in {"subscription-url", "socks-uri-credential"} and (
                "example.com" in lowered
                or "example.org" in lowered
                or "localhost" in lowered
                or "127.0.0.1" in lowered
                or "192.168." in lowered
                or "10." in lowered
                or "<" in line
            ):
                continue
            if pattern.name == "socks-uri-credential" and socks_uri_looks_like_fixture(line):
                continue
            if pattern.name in {"sub-id", "explicit-proxy-password"} and "<" in line:
                continue
            if pattern.regex.search(line):
                findings.append(f"{rel}:{lineno}: {pattern.name}: {pattern.guidance}")
    return findings


def socks_uri_looks_like_fixture(line: str) -> bool:
    match = re.search(r"\bsocks5h?://[^\s`\"']+", line, re.I)
    if not match:
        return False
    parsed = urlparse(match.group(0))
    host = parsed.hostname or ""
    username = parsed.username or ""
    password = parsed.password or ""
    if host.startswith(("10.", "127.", "192.168.")) or host in {"localhost", "example.com", "example.org"}:
        return True
    if username in {"user", "username", "test"} and password in {"pass", "password", "secret", "test"}:
        return True
    return False


def find_root(start: pathlib.Path) -> pathlib.Path:
    for candidate in [start, *start.parents]:
        if (candidate / ".git").exists() and (candidate / "go.mod").exists():
            return candidate
    raise RuntimeError("could not find repository root")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Scan workspace text files for high-risk secrets.")
    parser.add_argument("--all", action="store_true", help="Scan all non-skipped files, including ignored files.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = find_root(pathlib.Path.cwd().resolve())
    files = all_files(root) if args.all else git_files(root)
    findings: list[str] = []
    for path in files:
        if should_scan(root, path):
            findings.extend(scan_file(root, path))
    if findings:
        print("Secret scan failed. High-risk findings:")
        for finding in findings:
            print(f"- {finding}")
        return 1
    print("PASS: no high-risk secrets found in scanned workspace files.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
