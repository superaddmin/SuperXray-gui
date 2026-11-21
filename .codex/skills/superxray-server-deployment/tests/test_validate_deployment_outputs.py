import json
import runpy
import subprocess
import sys
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "validate_deployment_outputs.py"
RENDERER = SCRIPT.with_name("render_subscription_delivery.py")


class DeploymentManifestTest(unittest.TestCase):
    def test_renderer_errors_hide_invalid_field_values(self):
        functions = runpy.run_path(str(RENDERER))
        marker = "private-fixture-marker"
        calls = [
            ("utc_text", ({"expiresUtc": marker},)),
            ("sub_url", ({"subId": "demo"}, marker + ":bad", 2096, "clash")),
            ("panel_base", ("https://[" + marker + "]/",)),
        ]
        for name, args in calls:
            with self.subTest(name=name), self.assertRaises(ValueError) as raised:
                functions[name](*args)
            self.assertNotIn(marker, str(raised.exception))

    def test_invalid_subscription_container_has_readable_error(self):
        validate = runpy.run_path(str(SCRIPT))["validate"]
        with self.assertRaisesRegex(ValueError, "subscriptions must be objects"):
            validate({"server": {}, "service": {}, "subscriptions": ["invalid"]})

    def test_render_rejects_out_of_range_port_and_expiry(self):
        functions = runpy.run_path(str(RENDERER))
        for invalid_port in (0, 65536, True, "1.5"):
            with self.subTest(port=invalid_port), self.assertRaises(ValueError):
                functions["port_number"]({"port": invalid_port}, "port", 1080)
        self.assertEqual(functions["port_number"]({}, "port", 1080), 1080)
        with self.assertRaisesRegex(ValueError, "date range"):
            functions["utc_text"]({"expiryTimeMs": 10**1000})

    def test_generated_subscription_urls_handle_ipv6(self):
        sub_url = runpy.run_path(str(RENDERER))["sub_url"]
        for host in ("2001:db8::1", "[2001:db8::1]", "example.org"):
            authority = "[2001:db8::1]" if ":" in host else host
            with self.subTest(host=host):
                self.assertEqual(sub_url({"subId": "demo"}, host, 2096, "clash"), f"https://{authority}:2096/clash/demo")

    def test_render_accepts_direct_urls_and_keeps_them_out_of_warnings(self):
        urls = {kind + "Url": "http://example.org/" + kind for kind in ("clash", "json", "uri")}
        manifest = {
            "host": "example.org",
            "panel": {"url": "https://example.org/custompanel/", "username": "fixture", "password": "fixture"},
            "subscriptions": {"main": urls, **{
                key: {"clashUrl": urls["clashUrl"], "expiresUtc": "2030-01-01T00:00:00Z"}
                for key in ("threeMonth", "sixMonth")
            }},
            "socks5": {"username": "fixture", "password": "fixture", "clashUrl": urls["clashUrl"]},
        }
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "manifest.json"
            path.write_text(json.dumps(manifest), encoding="utf-8")
            result = subprocess.run([sys.executable, "-X", "utf8", str(RENDERER), "--manifest", str(path)],
                                    capture_output=True, encoding="utf-8")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(manifest["panel"]["url"], result.stdout)
        self.assertIn(urls["clashUrl"], result.stdout)
        self.assertIn("WARNING", result.stderr)
        self.assertNotIn("example.org", result.stderr)
        panel_base = runpy.run_path(str(RENDERER))["panel_base"]
        self.assertEqual(panel_base("https://example.org/base/panel"), "https://example.org/base/")

    def test_valid_manifest_is_accepted_without_printing_password(self):
        three_month_expiry = datetime.now(timezone.utc) + timedelta(days=90)
        six_month_expiry = datetime.now(timezone.utc) + timedelta(days=180)
        manifest = {
            "server": {"host": "<SERVER_IP>", "arch": "amd64"},
            "service": {"systemd": "active", "xray": "running", "ports": {"subscription": 2096}},
            "subscriptions": {
                name: {
                    "expiryTimeMs": int((three_month_expiry if name == "threeMonth" else six_month_expiry).timestamp() * 1000),
                    "expiresUtc": (three_month_expiry if name == "threeMonth" else six_month_expiry).isoformat().replace("+00:00", "Z"),
                    "clashUrl": "<REDACTED_SUBSCRIPTION_URL>",
                    "diagnostic": {"outputNodes": 1, "skippedNodes": 0},
                }
                for name in ("threeMonth", "sixMonth")
            },
            "socks5": {
                "host": "<SOCKS_HOST>",
                "port": 1080,
                "username": "<SOCKS_USERNAME>",
                "password": "<SOCKS_PASSWORD>",
                "uri": "socks5://<SOCKS_USERNAME>:<SOCKS_PASSWORD>@<SOCKS_HOST>:1080",
            },
        }
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "manifest.json"
            path.write_text(json.dumps(manifest), encoding="utf-8")
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--manifest", str(path)],
                capture_output=True,
                text=True,
                check=False,
            )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("deployment_manifest=ok", result.stdout)
        self.assertNotIn("<SOCKS_PASSWORD>", result.stdout)
        self.assertNotIn("<SOCKS_HOST>", result.stdout)
        validate = runpy.run_path(str(SCRIPT))["validate"]
        self.assertEqual(validate(manifest, require_redacted=False), ["threeMonth=ok", "sixMonth=ok", "socks5=ok"])
        marker = "private-fixture-marker"
        tests = (
            (manifest["server"], "host", marker),
            (manifest["socks5"], "username", "<SOCKS_USERNAME>" + marker),
            (manifest["socks5"], "uri", manifest["socks5"]["uri"] + "?" + marker),
            (manifest["socks5"], "uri", manifest["socks5"]["uri"].replace("1080", marker)),
            (manifest["subscriptions"]["threeMonth"], "clashUrl", "<REDACTED_URL>" + marker),
            (manifest["subscriptions"]["threeMonth"], "expiresUtc", marker),
            (manifest["subscriptions"]["threeMonth"], "expiryTimeMs", 10**1000),
        )
        for entry, key, value in tests:
            original = entry[key]
            entry[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError) as raised:
                validate(manifest)
            self.assertNotIn(marker, str(raised.exception))
            entry[key] = original
        manifest["server"]["host"] = "<server_ip>"
        self.assertIn("socks5=ok", validate(manifest))
        for invalid_count in (float("nan"), float("inf"), 1.5, True):
            manifest["subscriptions"]["threeMonth"]["diagnostic"]["outputNodes"] = invalid_count
            with self.subTest(count=invalid_count), self.assertRaisesRegex(ValueError, "counts must be integers"):
                validate(manifest)

    def test_invalid_diagnostic_is_rejected(self):
        expiry = datetime.now(timezone.utc) + timedelta(days=90)
        manifest = {
            "server": {"host": "<SERVER_IP>", "arch": "amd64"},
            "service": {"systemd": "active", "xray": "running", "ports": {}},
            "subscriptions": {
                name: {
                    "expiryTimeMs": int((expiry + timedelta(days=90 if name == "sixMonth" else 0)).timestamp() * 1000),
                    "expiresUtc": (expiry + timedelta(days=90 if name == "sixMonth" else 0)).isoformat().replace("+00:00", "Z"),
                    "clashUrl": "<REDACTED_SUBSCRIPTION_URL>",
                    "diagnostic": "invalid",
                }
                for name in ("threeMonth", "sixMonth")
            },
            "socks5": {
                "host": "<SOCKS_HOST>",
                "port": 1080,
                "username": "<SOCKS_USERNAME>",
                "password": "<SOCKS_PASSWORD>",
                "uri": "socks5://<SOCKS_USERNAME>:<SOCKS_PASSWORD>@<SOCKS_HOST>:1080",
            },
        }
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "invalid.json"
            path.write_text(json.dumps(manifest), encoding="utf-8")
            result = subprocess.run([sys.executable, str(SCRIPT), "--manifest", str(path)], capture_output=True, text=True, check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("diagnostic must be an object", result.stderr)


if __name__ == "__main__":
    unittest.main()
