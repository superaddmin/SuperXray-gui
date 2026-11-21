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
from ipaddress import IPv6Address
from pathlib import Path
from urllib.parse import urlparse


SELF_CHECK_URL = "https://github.com/ryuu-restart/agent-route/blob/master/docs/DNS-WEBRTC.md"


def fail(message: str) -> int:
    print(f"ERROR: {message}", file=sys.stderr)
    return 1


def first(mapping: dict, *keys: str):
    for key in keys:
        value = mapping.get(key)
        if value not in (None, ""):
            return value
    raise KeyError(keys[0])


def as_object(value, name: str) -> dict:
    if not isinstance(value, dict):
        raise ValueError(f"{name} must be an object")
    return value


def port_number(mapping: dict, key: str, default: int) -> int:
    value = mapping.get(key, default)
    if not isinstance(value, (str, int)) or not str(value).isascii() or not str(value).isdigit():
        raise ValueError(f"{key} must be a port number")
    port = int(value)
    if not 1 <= port <= 65535:
        raise ValueError(f"{key} must be between 1 and 65535")
    return port


def panel_base(url: str) -> str:
    try:
        parsed = urlparse(url)
    except ValueError as exc:
        raise ValueError("panel.url must be an absolute URL") from exc
    if not parsed.scheme or not parsed.netloc:
        raise ValueError("panel.url must be an absolute URL")
    path = parsed.path.rstrip("/")
    if path.endswith("/panel"):
        path = path[:-len("/panel")]
    if not path.endswith("/"):
        path += "/"
    return f"{parsed.scheme}://{parsed.netloc}{path}"


def utc_text(subscription: dict) -> str:
    expiry_ms = subscription.get("expiryTimeMs")
    if type(expiry_ms) is int and expiry_ms > 0:
        try:
            return datetime.fromtimestamp(expiry_ms / 1000, timezone.utc).strftime("%Y-%m-%d %H:%M:%S")
        except (ValueError, OSError, OverflowError) as exc:
            raise ValueError("expiryTimeMs is outside the supported date range") from exc
    expires = subscription.get("expiresUtc")
    if not expires:
        raise ValueError("subscription requires expiryTimeMs or expiresUtc")
    try:
        parsed = datetime.fromisoformat(str(expires).replace("Z", "+00:00"))
    except ValueError as exc:
        raise ValueError("expiresUtc must be an ISO 8601 timestamp") from exc
    if parsed.tzinfo is None:
        raise ValueError("expiresUtc must include a timezone")
    try:
        return parsed.astimezone(timezone.utc).strftime("%Y-%m-%d %H:%M:%S")
    except (ValueError, OSError, OverflowError) as exc:
        raise ValueError("expiresUtc is outside the supported date range") from exc


def sub_url(subscription: dict, host: str, port: int, kind: str) -> str:
    direct = subscription.get(kind) or subscription.get(f"{kind}Subscription") or subscription.get(f"{kind}Url")
    if direct:
        return str(direct)
    sub_id = subscription.get("subId") or subscription.get("subID")
    if not sub_id:
        raise ValueError(f"subscription requires subId or {kind} URL")
    path = {"clash": "clash", "uri": "sub", "json": "json"}[kind]
    if ":" in host and not host.startswith("["):
        try:
            host = f"[{IPv6Address(host)}]"
        except ValueError as exc:
            raise ValueError("host must contain a valid IPv6 address") from exc
    return f"https://{host}:{port}/{path}/{sub_id}"


def render(manifest: dict, *, redact: bool) -> str:
    host = str(first(manifest, "host", "serverIp", "server_ip"))
    country = str(manifest.get("country") or "")
    panel = as_object(manifest.get("panel"), "panel")
    subscriptions = as_object(manifest.get("subscriptions"), "subscriptions")
    socks = as_object(
        manifest.get("socks5") or subscriptions.get("socks5LongTerm") or subscriptions.get("socks5"), "socks5"
    )

    main = as_object(first(subscriptions, "main", "longTerm", "clashLongTerm"), "subscriptions.main")
    three = as_object(first(subscriptions, "threeMonth", "3-month", "clashThreeMonth"), "subscriptions.threeMonth")
    six = as_object(first(subscriptions, "sixMonth", "6-month", "clashSixMonth"), "subscriptions.sixMonth")

    panel_url = panel_base(str(first(panel, "url", "panelURL", "panelUrl")))
    username = str(first(panel, "username", "user"))
    password = str(first(panel, "password", "pass"))
    sub_port = port_number(as_object(manifest.get("ports", {}), "ports"), "subscription", 2096)
    socks_host = str(socks.get("host") or host)
    socks_port = port_number(socks, "port", 1080)
    socks_user = str(first(socks, "username", "user"))
    socks_pass = str(first(socks, "password", "pass"))
    socks_clash = sub_url(socks, host, sub_port, "clash")

    values = {
        "COUNTRY": country,
        "SERVER_IP": host,
        "PANEL_URL": panel_url,
        "PANEL_USERNAME": username,
        "PANEL_PASSWORD": password,
        "URI_URL": sub_url(main, host, sub_port, "uri"),
        "JSON_URL": sub_url(main, host, sub_port, "json"),
        "CLASH_URL": sub_url(main, host, sub_port, "clash"),
        "SOCKS_HOST": socks_host,
        "SOCKS_PORT": str(socks_port),
        "SOCKS_USERNAME": socks_user,
        "SOCKS_PASSWORD": socks_pass,
        "SOCKS5_CLASH_URL": str(socks_clash),
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
            "URI_URL": "<URI_SUBSCRIPTION_URL>",
            "JSON_URL": "<JSON_SUBSCRIPTION_URL>",
            "CLASH_URL": "<CLASH_SUBSCRIPTION_URL>",
            "SOCKS_HOST": "<SOCKS_HOST>",
            "SOCKS_PORT": "<SOCKS_PORT>",
            "SOCKS_USERNAME": "<SOCKS_USERNAME>",
            "SOCKS_PASSWORD": "<SOCKS_PASSWORD>",
            "SOCKS5_CLASH_URL": "<SOCKS5_CLASH_URL>",
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
        "",
        f"终端侧 DNS/WebRTC 自查（第三方指南）: {SELF_CHECK_URL}",
    ]
    for field in ("URI_URL", "JSON_URL", "CLASH_URL", "SOCKS5_CLASH_URL", "THREE_MONTH_URL", "SIX_MONTH_URL"):
        if not redact and not values[field].startswith("https://"):
            print(f"WARNING: {field} uses a non-HTTPS subscription URL", file=sys.stderr)
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
    except (OSError, UnicodeError, json.JSONDecodeError, RecursionError):
        return fail("cannot read or render manifest as UTF-8 JSON")
    except (KeyError, ValueError, TypeError) as exc:
        return fail(str(exc))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
