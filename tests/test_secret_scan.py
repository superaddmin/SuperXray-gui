import pathlib
import subprocess
import sys
import tempfile
import unittest

import scripts.secret_scan as secret_scan


class SecretScanTests(unittest.TestCase):
    @staticmethod
    def initialize_git_repository(root: pathlib.Path) -> None:
        subprocess.run(
            ["git", "init", "--quiet"],
            cwd=root,
            check=True,
            text=True,
            encoding="utf-8",
            capture_output=True,
        )
        (root / "go.mod").write_text("module example.test/fixture\n\ngo 1.22\n", encoding="utf-8")

    @staticmethod
    def git(root: pathlib.Path, *args: str) -> None:
        subprocess.run(
            ["git", *args],
            cwd=root,
            check=True,
            text=True,
            encoding="utf-8",
            capture_output=True,
        )

    @staticmethod
    def run_cli(root: pathlib.Path, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(pathlib.Path(secret_scan.__file__).resolve()), *args],
            cwd=root,
            text=True,
            encoding="utf-8",
            errors="replace",
            capture_output=True,
        )

    @staticmethod
    def write_private_key(path: pathlib.Path) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(
            "-----BEGIN " + "RSA PRIVATE KEY-----\nfixture\n-----END " + "RSA PRIVATE KEY-----\n",
            encoding="utf-8",
        )

    def test_pem_private_key_is_scanned(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            private_key = root / "fixture.pem"
            private_key.write_text(
                "-----BEGIN " + "RSA PRIVATE KEY-----\nfixture\n-----END " + "RSA PRIVATE KEY-----\n",
                encoding="utf-8",
            )

            self.assertTrue(secret_scan.should_scan(root, private_key))
            self.assertTrue(
                any(": private-key:" in finding for finding in secret_scan.scan_file(root, private_key))
            )

    def test_key_private_key_is_scanned(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            private_key = root / "fixture.key"
            private_key.write_text(
                "-----BEGIN "
                + "OPENSSH PRIVATE KEY-----\nfixture\n-----END "
                + "OPENSSH PRIVATE KEY-----\n",
                encoding="utf-8",
            )

            self.assertTrue(secret_scan.should_scan(root, private_key))
            self.assertTrue(
                any(": private-key:" in finding for finding in secret_scan.scan_file(root, private_key))
            )

    def test_pkcs8_private_key_headers_are_scanned(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            for prefix in ("", "ENCRYPTED "):
                with self.subTest(prefix=prefix):
                    private_key = root / f"fixture-{prefix.strip() or 'plain'}.pem"
                    private_key.write_text(
                        "-----BEGIN "
                        + prefix
                        + "PRIVATE KEY-----\nfixture\n-----END "
                        + prefix
                        + "PRIVATE KEY-----\n",
                        encoding="utf-8",
                    )

                    self.assertTrue(secret_scan.should_scan(root, private_key))
                    self.assertTrue(
                        any(
                            ": private-key:" in finding
                            for finding in secret_scan.scan_file(root, private_key)
                        )
                    )

    def test_inline_short_private_key_fixture_is_skipped(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            fixture = root / "fixture.ts"
            begin = "-----BEGIN " + "PRIVATE KEY-----"
            end = "-----END " + "PRIVATE KEY-----"
            fixture.write_text(f"key: ['{begin}', 'MIIB', '{end}']\n", encoding="utf-8")

            self.assertEqual(secret_scan.scan_file(root, fixture), [])

    def test_binary_key_containers_are_reported_without_text_decoding(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            for suffix in (".jks", ".keystore", ".p12", ".pfx"):
                with self.subTest(suffix=suffix):
                    certificate = root / f"fixture{suffix}"
                    certificate.write_bytes(b"\xff\x00\xfe\x01")

                    self.assertTrue(secret_scan.should_scan(root, certificate))
                    findings = secret_scan.scan_file(root, certificate)
                    self.assertEqual(len(findings), 1)
                    self.assertIn(": binary-key-container:", findings[0])

    def test_binary_private_key_material_is_reported_without_text_decoding(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            for suffix in (".pem", ".key", ".pk8"):
                with self.subTest(suffix=suffix):
                    private_key = root / f"fixture{suffix}"
                    private_key.write_bytes(b"\xff\x00\xfe\x01")

                    self.assertTrue(secret_scan.should_scan(root, private_key))
                    findings = secret_scan.scan_file(root, private_key)
                    self.assertEqual(len(findings), 1)
                    self.assertIn(": binary-key-material:", findings[0])

    def test_cli_default_excludes_ignored_pem_but_all_reports_it(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            self.initialize_git_repository(root)
            (root / ".gitignore").write_text("*.pem\n", encoding="utf-8")
            private_key = root / "ignored.pem"
            self.write_private_key(private_key)

            listed = {path.relative_to(root).as_posix() for path in secret_scan.git_files(root)}
            default_result = self.run_cli(root)
            all_result = self.run_cli(root, "--all")

            self.assertNotIn("ignored.pem", listed)
            self.assertEqual(default_result.returncode, 0, default_result.stderr)
            self.assertEqual(all_result.returncode, 1, all_result.stderr)
            self.assertIn("ignored.pem:1: private-key:", all_result.stdout)
            self.assertIn("tracked and unignored untracked files", default_result.stdout)

    def test_cli_default_reports_untracked_pem(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            self.initialize_git_repository(root)
            private_key = root / "untracked.pem"
            self.write_private_key(private_key)

            listed = {path.relative_to(root).as_posix() for path in secret_scan.git_files(root)}
            result = self.run_cli(root)

            self.assertIn("untracked.pem", listed)
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn("untracked.pem:1: private-key:", result.stdout)

    def test_cli_default_reports_force_tracked_ignored_pem(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            self.initialize_git_repository(root)
            (root / ".gitignore").write_text("*.pem\n", encoding="utf-8")
            private_key = root / "tracked.pem"
            self.write_private_key(private_key)
            self.git(root, "add", "-f", "tracked.pem")

            listed = {path.relative_to(root).as_posix() for path in secret_scan.git_files(root)}
            result = self.run_cli(root)

            self.assertIn("tracked.pem", listed)
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn("tracked.pem:1: private-key:", result.stdout)

    def test_cli_default_reports_force_tracked_binary_key(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            self.initialize_git_repository(root)
            (root / ".gitignore").write_text("*.key\n", encoding="utf-8")
            private_key = root / "tracked.key"
            private_key.write_bytes(b"\xff\x00\xfe\x01")
            self.git(root, "add", "-f", "tracked.key")

            listed = {path.relative_to(root).as_posix() for path in secret_scan.git_files(root)}
            result = self.run_cli(root)

            self.assertIn("tracked.key", listed)
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn("tracked.key: binary-key-material:", result.stdout)

    def test_cli_default_reports_force_tracked_binary_key_container(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            self.initialize_git_repository(root)
            (root / ".gitignore").write_text("*.p12\n", encoding="utf-8")
            key_container = root / "tracked.p12"
            key_container.write_bytes(b"\xff\x00\xfe\x01")
            self.git(root, "add", "-f", "tracked.p12")

            listed = {path.relative_to(root).as_posix() for path in secret_scan.git_files(root)}
            result = self.run_cli(root)

            self.assertIn("tracked.p12", listed)
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn("tracked.p12: binary-key-container:", result.stdout)

    def test_cli_reports_token_in_tracked_web_ui_javascript(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            self.initialize_git_repository(root)
            javascript = root / "web" / "ui" / "fixture.js"
            javascript.parent.mkdir(parents=True)
            token = "gh" + "p_" + ("A" * 32)
            javascript.write_text(f"const token = '{token}';\n", encoding="utf-8")
            self.git(root, "add", "web/ui/fixture.js")

            result = self.run_cli(root)

            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn("web/ui/fixture.js:1: github-token:", result.stdout)

    def test_cli_all_pass_message_describes_scope(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            self.initialize_git_repository(root)

            result = self.run_cli(root, "--all")

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("all non-skipped workspace files", result.stdout)

    def test_missing_git_index_file_is_skipped(self) -> None:
        root = pathlib.Path.cwd()
        missing = root / "web" / "assets" / "deleted-by-retirement.js"

        self.assertEqual(secret_scan.scan_file(root, missing), [])


if __name__ == "__main__":
    unittest.main()
