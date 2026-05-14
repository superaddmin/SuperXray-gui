from __future__ import annotations

import argparse
import importlib.util
import pathlib
import subprocess
import tempfile
import unittest


SCRIPT_PATH = pathlib.Path(__file__).resolve().parents[1] / "scripts" / "release_gate.py"
REPO_ROOT = pathlib.Path(__file__).resolve().parents[4]
SPEC = importlib.util.spec_from_file_location("release_gate", SCRIPT_PATH)
assert SPEC is not None and SPEC.loader is not None
release_gate = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release_gate)


class ReleaseMetadataTests(unittest.TestCase):
    def make_repo(self, files: dict[str, str]) -> pathlib.Path:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        root = pathlib.Path(tmp.name)
        for relative, content in files.items():
            path = root / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content, encoding="utf-8")
        subprocess.run(["git", "init"], cwd=root, check=True, stdout=subprocess.DEVNULL)
        subprocess.run(["git", "add", "."], cwd=root, check=True, stdout=subprocess.DEVNULL)
        return root

    def make_gate(self, root: pathlib.Path):
        args = argparse.Namespace(tag="v3.0.8", install_tools=False, allow_dirty=True, ci=False)
        return release_gate.Gate(root, args)

    def base_files(self) -> dict[str, str]:
        return {
            "CHANGELOG.md": (
                "# CHANGELOG\n\n"
                "## [3.0.8]\n\n- Current.\n\n"
                "## [3.0.7]\n\n- Previous.\n\n"
                "## [3.0.3]\n\n- Historical.\n"
            ),
            "config/version": "3.0.8\n",
            "README.md": "Install v3.0.8.\n",
        }

    def test_release_metadata_ignores_historical_superpowers_reports(self) -> None:
        files = self.base_files()
        files[
            "docs/superpowers/reports/2026-05-09-v303-new-ui-functional-comparison.md"
        ] = "Historical baseline: v3.0.3.\n"
        self.make_gate(self.make_repo(files)).check_release_metadata()

    def test_release_metadata_still_rejects_stale_readme_references(self) -> None:
        files = self.base_files()
        files["README.zh_CN.md"] = "Install v3.0.7.\n"
        with self.assertRaisesRegex(RuntimeError, "stale version references"):
            self.make_gate(self.make_repo(files)).check_release_metadata()

    def test_release_workflow_runs_reusable_metadata_gate(self) -> None:
        release_workflow = (REPO_ROOT / ".github" / "workflows" / "release.yml").read_text(
            encoding="utf-8"
        )
        self.assertIn("release_gate.py --ci --metadata-only", release_workflow)


if __name__ == "__main__":
    unittest.main()
