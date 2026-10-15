import json
import subprocess
import sys
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "validate_deployment_outputs.py"


class DeploymentManifestTest(unittest.TestCase):
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
