#!/usr/bin/env python3
"""Render the standard SuperXray subscription delivery text from a protected manifest.

The manifest may come from .local/ops/<deployment-id>/delivery.json. Real values are
printed only to stdout; use --redact to render placeholders for docs/tests.
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import urlparse


def fail(message: str) -> int:
    print(f"ERROR: {message}", file=sys.stderr)
    return 1


def first(mapping: dict, *keys: str):
    for key in keys:
        value = mapping.get(key)
        if value not in (None, ""):
            return value
    raise KeyError(keys[0])


def panel_base(url: str) -> str:
    parsed = urlparse(url)
    if not parsed.scheme or not parsed.netloc:
        raise ValueError("panel.url must be an absolute URL")
    path = parsed.path
    if path.endswith("panel/"):
        path = path[: -len("panel/")]
    if not path.endswith("/"):
        path += "/"
    return f"{parsed.scheme}://{parsed.netloc}{path}"


def utc_text(subscription: dict) -> str:
    expiry_ms = subscription.get("expiryTimeMs")
    if isinstance(expiry_ms, int) and expiry_ms > 0:
        return datetime.fromtimestamp(expiry_ms / 1000, timezone.utc).strftime("%Y-%m-%d %H:%M:%S")
    expires = subscription.get("expiresUtc")
    if not expires:
        raise ValueError("subscription requires expiryTimeMs or expiresUtc")
    parsed = datetime.fromisoformat(str(expires).replace("Z", "+00:00"))
    if parsed.tzinfo is None:
        raise ValueError("expiresUtc must include a timezone")
    return parsed.astimezone(timezone.utc).strftime("%Y-%m-%d %H:%M:%S")


def sub_url(subscription: dict, host: str, port: int, kind: str) -> str:
    direct = subscription.get(kind) or subscription.get(f"{kind}Subscription")
    if direct:
        return str(direct)
    sub_id = subscription.get("subId") or subscription.get("subID")
    if not sub_id:
        raise ValueError(f"subscription requires subId or {kind} URL")
    path = {"clash": "clash", "uri": "sub", "json": "json"}[kind]
    return f"https://{host}:{port}/{path}/{sub_id}"


def render(manifest: dict, *, redact: bool) -> str:
    host = str(first(manifest, "host", "serverIp", "server_ip"))
    country = str(manifest.get("country") or "")
    panel = manifest.get("panel") or {}
    subscriptions = manifest.get("subscriptions") or {}
    socks = manifest.get("socks5") or subscriptions.get("socks5LongTerm") or subscriptions.get("socks5") or {}

    main = first(subscriptions, "main", "longTerm", "clashLongTerm")
    three = first(subscriptions, "threeMonth", "3-month", "clashThreeMonth")
    six = first(subscriptions, "sixMonth", "6-month", "clashSixMonth")

    panel_url = panel_base(str(first(panel, "url", "panelURL", "panelUrl")))
    username = str(first(panel, "username", "user"))
    password = str(first(panel, "password", "pass"))
    sub_port = int((manifest.get("ports") or {}).get("subscription") or 2096)
    socks_host = str(socks.get("host") or host)
    socks_port = int(socks.get("port") or 1080)
    socks_user = str(first(socks, "username", "user"))
    socks_pass = str(first(socks, "password", "pass"))
    socks_clash = socks.get("clash") or socks.get("clashSubscription")
    if not socks_clash:
        socks_sub_id = socks.get("subId") or socks.get("subID")
        if not socks_sub_id:
            raise ValueError("socks5 requires a clash URL or subId")
        socks_clash = f"https://{host}:{sub_port}/clash/{socks_sub_id}"
    main_sub_id = str(first(main, "subId", "subID"))
    three_sub_id = str(first(three, "subId", "subID"))
    six_sub_id = str(first(six, "subId", "subID"))

    values = {
        "COUNTRY": country,
        "SERVER_IP": host,
        "PANEL_URL": panel_url,
        "PANEL_USERNAME": username,
        "PANEL_PASSWORD": password,
        "SUB_PORT": str(sub_port),
        "MAIN_SUB_ID": main_sub_id,
        "URI_URL": sub_url(main, host, sub_port, "uri"),
        "JSON_URL": sub_url(main, host, sub_port, "json"),
        "CLASH_URL": sub_url(main, host, sub_port, "clash"),
        "SOCKS_HOST": socks_host,
        "SOCKS_PORT": str(socks_port),
        "SOCKS_USERNAME": socks_user,
        "SOCKS_PASSWORD": socks_pass,
        "SOCKS5_CLASH_URL": str(socks_clash),
        "THREE_MONTH_SUB_ID": three_sub_id,
        "SIX_MONTH_SUB_ID": six_sub_id,
        "THREE_MONTH_URL": sub_url(three, host, sub_port, "clash"),
        "SIX_MONTH_URL": sub_url(six, host, sub_port, "clash"),
        "THREE_MONTH_UTC": utc_text(three),
        "SIX_MONTH_UTC": utc_text(six),
    }
    if redact:
        placeholders = {
            "COUNTRY": "<COUNTRY>",
            "SERVER_IP": "<SERVER_IP>",
            "PANEL_URL": "<PANEL_URL>",
            "PANEL_USERNAME": "<PANEL_USERNAME>",
            "PANEL_PASSWORD": "<PANEL_PASSWORD>",
            "SUB_PORT": "<SUB_PORT>",
            "MAIN_SUB_ID": "<MAIN_SUB_ID>",
            "URI_URL": "<URI_SUBSCRIPTION_URL>",
            "JSON_URL": "<JSON_SUBSCRIPTION_URL>",
            "CLASH_URL": "<CLASH_SUBSCRIPTION_URL>",
            "SOCKS_HOST": "<SOCKS_HOST>",
            "SOCKS_PORT": "<SOCKS_PORT>",
            "SOCKS_USERNAME": "<SOCKS_USERNAME>",
            "SOCKS_PASSWORD": "<SOCKS_PASSWORD>",
            "SOCKS5_CLASH_URL": "<SOCKS5_CLASH_URL>",
            "THREE_MONTH_SUB_ID": "<THREE_MONTH_SUB_ID>",
            "SIX_MONTH_SUB_ID": "<SIX_MONTH_SUB_ID>",
            "THREE_MONTH_URL": "<THREE_MONTH_CLASH_URL>",
            "SIX_MONTH_URL": "<SIX_MONTH_CLASH_URL>",
        }
        values.update(placeholders)

    header = "面板安装完成！   " + " ".join(
        part for part in (values["COUNTRY"], values["SERVER_IP"]) if part
    )
    lines = [
        header,
        "",
        values["PANEL_URL"],
        f"用户名: {values['PANEL_USERNAME']}",
        f"密码: {values['PANEL_PASSWORD']}",
        "",
        "URI/Base64:",
        values["URI_URL"],
        "",
        "Xray JSON:",
        values["JSON_URL"],
        "",
        "Clash/Mihomo:",
        values["CLASH_URL"],
        "",
        "SOCKS5 订阅",
        "协议: SOCKS5",
        f"主机: {values['SOCKS_HOST']}",
        f"端口: {values['SOCKS_PORT']}",
        f"用户名: {values['SOCKS_USERNAME']}",
        f"密码: {values['SOCKS_PASSWORD']}",
        f"SOCKS5 订阅链接: {values['SOCKS5_CLASH_URL']}",
        f"三个月有效期（UTC 到期: {values['THREE_MONTH_UTC']}） {values['THREE_MONTH_URL']}",
        f"六个月有效期（UTC 到期: {values['SIX_MONTH_UTC']}） {values['SIX_MONTH_URL']}",
    ]
    for url in (values["URI_URL"], values["JSON_URL"], values["CLASH_URL"], values["SOCKS5_CLASH_URL"], values["THREE_MONTH_URL"], values["SIX_MONTH_URL"]):
        if not redact and not str(url).startswith("https://"):
            print(f"WARNING: non-HTTPS subscription URL: {url}", file=sys.stderr)
    return "\n".join(lines) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--redact", action="store_true", help="render placeholders only")
    args = parser.parse_args()
    try:
        manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
        if not isinstance(manifest, dict):
            return fail("manifest root must be a JSON object")
        print(render(manifest, redact=args.redact), end="")
    except (OSError, json.JSONDecodeError, KeyError, ValueError, TypeError) as exc:
        return fail(str(exc))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
