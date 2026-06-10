import pathlib
import unittest

import scripts.secret_scan as secret_scan


class SecretScanTests(unittest.TestCase):
    def test_missing_git_index_file_is_skipped(self) -> None:
        root = pathlib.Path.cwd()
        missing = root / "web" / "assets" / "deleted-by-retirement.js"

        self.assertEqual(secret_scan.scan_file(root, missing), [])


if __name__ == "__main__":
    unittest.main()
