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

    def make_gate(self, root: pathlib.Path, tag: str = "v3.0.8"):
        args = argparse.Namespace(tag=tag, install_tools=False, allow_dirty=True, ci=False)
        return release_gate.Gate(root, args)

    def make_metadata_gate(self, root: pathlib.Path, tag: str = "v3.0.8"):
        args = argparse.Namespace(tag=tag, install_tools=False, allow_dirty=True, ci=True, metadata_only=True)
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

    def workflow_files(self, release_workflow: str) -> dict[str, str]:
        files = self.base_files()
        files.update(
            {
                ".github/workflows/release.yml": release_workflow,
                ".github/workflows/docker.yml": (
                    "platforms: linux/amd64,linux/arm64\n"
                    "image: ghcr.io/superaddmin/superxray-gui\n"
                ),
                ".github/workflows/test-arm64.yml": (
                    "platform linux/arm64\n"
                    "gcc-aarch64-linux-gnu\n"
                ),
                ".github/workflows/codeql.yml": "github/codeql-action/analyze\n",
                ".github/agentic-workflows/release.md": (
                    "GitHub Agentic Workflow: Release\n"
                    "release_gate.py\n"
                    "vX.Y.Z\n"
                    "x-ui-linux-amd64.tar.gz\n"
                    "x-ui-linux-arm64.tar.gz\n"
                    "ghcr.io/superaddmin/superxray-gui\n"
                    "--force-with-lease\n"
                ),
                ".github/copilot-instructions.md": ".github/agentic-workflows/release.md\n",
                "Dockerfile": 'RUN chmod +x DockerInit.sh && ./DockerInit.sh "$TARGETARCH"\n',
            }
        )
        return files

    def valid_release_workflow(self) -> str:
        return "\n".join(
            [
                "on:",
                "  push:",
                "    tags:",
                '      - "v*.*.*"',
                "    paths:",
                '      - ".codex/**"',
                '      - ".github/agentic-workflows/**"',
                "jobs:",
                "  analyze:",
                "    steps:",
                "      - name: Validate release metadata",
                "        run: python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only",
                "      - name: Run secret scan",
                "        run: python scripts/secret_scan.py",
                "      - uses: actions/setup-go@v6",
                "      - run: sudo apt-get install -y gcc-aarch64-linux-gnu",
                "      - run: GOARCH=\"$goarch\" go build",
                "      - name: Run OpenAPI contract gate",
                "        run: go test ./web/controller -run 'TestV1OpenAPIRoutesStayInSyncWithGoRoutes|TestV1OpenAPIResponseContract|TestV1OpenAPIIncludesMetricsEndpoint' -count=1",
                "      - run: echo x-ui-linux-${{ matrix.platform }}.tar.gz",
                "      - run: echo '- amd64'",
                "      - run: echo '- arm64'",
                "      - run: echo 'Generate release notes'",
                "      - run: echo '$0 ~ \"^## \\\\[\" version \"\\\\]([[:space:]]|$)\"'",
                "      - run: echo 'id: release_notes'",
                "      - run: echo 'body: ${{ steps.release_notes.outputs.body }}'",
            ]
        )

    def test_release_metadata_ignores_historical_superpowers_reports(self) -> None:
        files = self.base_files()
        files[
            "docs/superpowers/reports/2026-05-09-v303-new-ui-functional-comparison.md"
        ] = "Historical baseline: v3.0.3.\n"
        self.make_gate(self.make_repo(files)).check_release_metadata()

    def test_release_metadata_ignores_historical_superpowers_plans_and_specs(self) -> None:
        files = self.base_files()
        files["docs/superpowers/plans/2026-06-07-p0-p2.md"] = "Historical project v3.0.7.\n"
        files["docs/superpowers/specs/2026-05-01-design.md"] = "Historical design v3.0.3.\n"
        self.make_gate(self.make_repo(files)).check_release_metadata()

    def test_release_metadata_still_rejects_stale_readme_references(self) -> None:
        files = self.base_files()
        files["README.zh_CN.md"] = "Install v3.0.7.\n"
        with self.assertRaisesRegex(RuntimeError, "stale version references"):
            self.make_gate(self.make_repo(files)).check_release_metadata()

    def test_release_metadata_allows_upstream_3x_ui_version_references(self) -> None:
        files = self.base_files()
        files["docs/upstream-sync-policy.md"] = (
            "> 当前上游基线：`3x-ui upstream tag v3.0.7` = `abcdef123`\n"
            "- 选择性同步 `3x-ui v3.0.7` 的安全修复。\n"
            "- Historical upstream 3x-ui v3.0.3 radar entry.\n"
        )

        self.make_gate(self.make_repo(files)).check_release_metadata()

    def test_release_metadata_does_not_match_current_version_prefixes(self) -> None:
        files = {
            "CHANGELOG.md": (
                "# CHANGELOG\n\n"
                "## [3.0.10]\n\n- Current.\n\n"
                "## [3.0.9]\n\n- Previous.\n\n"
                "## [3.0.1]\n\n- Historical.\n"
            ),
            "config/version": "3.0.10\n",
            "README.md": "Install v3.0.10.\n",
        }
        self.make_gate(self.make_repo(files), tag="v3.0.10").check_release_metadata()

    def test_release_workflow_runs_reusable_metadata_gate(self) -> None:
        release_workflow = (REPO_ROOT / ".github" / "workflows" / "release.yml").read_text(
            encoding="utf-8"
        )
        self.assertIn("release_gate.py --ci --metadata-only", release_workflow)

    def test_release_workflow_includes_codex_paths_for_metadata_changes(self) -> None:
        release_workflow = (REPO_ROOT / ".github" / "workflows" / "release.yml").read_text(
            encoding="utf-8"
        )
        self.assertIn('".codex/**"', release_workflow)

    def test_release_workflow_includes_secret_scan_gate(self) -> None:
        release_workflow = (REPO_ROOT / ".github" / "workflows" / "release.yml").read_text(
            encoding="utf-8"
        )
        self.assertIn("python scripts/secret_scan.py", release_workflow)

    def test_release_workflow_includes_openapi_contract_gate(self) -> None:
        release_workflow = (REPO_ROOT / ".github" / "workflows" / "release.yml").read_text(
            encoding="utf-8"
        )
        self.assertIn("Run OpenAPI contract gate", release_workflow)
        self.assertIn("TestV1OpenAPIIncludesMetricsEndpoint", release_workflow)

    def test_release_workflow_rejects_missing_codex_path_filter(self) -> None:
        release_workflow = self.valid_release_workflow().replace('      - ".codex/**"\n', "")

        with self.assertRaisesRegex(RuntimeError, r"\.codex/\*\*"):
            self.make_gate(self.make_repo(self.workflow_files(release_workflow))).check_workflows()

    def test_release_workflow_rejects_missing_secret_scan_gate(self) -> None:
        release_workflow = self.valid_release_workflow().replace(
            "      - name: Run secret scan\n        run: python scripts/secret_scan.py\n",
            "",
        )

        with self.assertRaisesRegex(RuntimeError, "Run secret scan"):
            self.make_gate(self.make_repo(self.workflow_files(release_workflow))).check_workflows()

    def test_release_workflow_rejects_missing_openapi_contract_gate(self) -> None:
        release_workflow = self.valid_release_workflow().replace(
            "      - name: Run OpenAPI contract gate\n"
            "        run: go test ./web/controller -run 'TestV1OpenAPIRoutesStayInSyncWithGoRoutes|TestV1OpenAPIResponseContract|TestV1OpenAPIIncludesMetricsEndpoint' -count=1\n",
            "",
        )

        with self.assertRaisesRegex(RuntimeError, "OpenAPI contract gate"):
            self.make_gate(self.make_repo(self.workflow_files(release_workflow))).check_workflows()

    def test_project_go_version_metadata_rejects_drift(self) -> None:
        files = self.base_files()
        files["go.mod"] = "module example.test/superxray\n\ngo 1.26.4\n"
        files[".codex/project.toml"] = '[stack.backend]\nversion = "1.26.3"\n'

        with self.assertRaisesRegex(RuntimeError, "Go version drift"):
            self.make_gate(self.make_repo(files)).check_project_go_version_metadata()

    def test_metadata_only_run_rejects_project_go_version_drift(self) -> None:
        files = self.base_files()
        files["go.mod"] = "module example.test/superxray\n\ngo 1.26.4\n"
        files[".codex/project.toml"] = '[stack.backend]\nversion = "1.26.3"\n'

        gate = self.make_metadata_gate(self.make_repo(files))

        self.assertEqual(gate.run(), 1)
        self.assertTrue(
            any("project_go_version_metadata" in failure for failure in gate.failures),
            gate.failures,
        )

    def test_project_go_version_metadata_accepts_matching_versions(self) -> None:
        files = self.base_files()
        files["go.mod"] = "module example.test/superxray\n\ngo 1.26.4\n"
        files[".codex/project.toml"] = '[stack.backend]\nversion = "1.26.4"\n'

        self.make_gate(self.make_repo(files)).check_project_go_version_metadata()

    def openapi_files(self, committed_json: str) -> dict[str, str]:
        files = self.base_files()
        files.update(
            {
                "go.mod": "module example.test/superxray\n\ngo 1.26.4\n",
                ".codex/project.toml": '[stack.backend]\nversion = "1.26.4"\n',
                "docs/openapi/panel-api.yaml": "openapi: 3.1.0\ninfo:\n  title: Test\npaths: {}\n",
                "frontend/public/openapi.json": committed_json,
                "tools/openapiexport/main.go": "package main\n",
            }
        )
        return files

    def install_fake_openapi_export(self, gate: release_gate.Gate, generated_json: str) -> None:
        def fake_command(command: list[str], *, capture: bool = False) -> str:
            self.assertEqual(command[:3], ["go", "run", "./tools/openapiexport"])
            self.assertIn("-out", command)
            out_path = pathlib.Path(command[command.index("-out") + 1])
            out_path.write_text(generated_json, encoding="utf-8")
            return ""

        gate.command = fake_command

    def test_openapi_generated_metadata_accepts_matching_normalized_json(self) -> None:
        gate = self.make_gate(
            self.make_repo(
                self.openapi_files('{"paths":{},"info":{"version":"3.0.8"},"openapi":"3.1.0"}\n')
            )
        )
        self.install_fake_openapi_export(
            gate,
            '{\n  "openapi": "3.1.0",\n  "info": {"version": "3.0.8"},\n  "paths": {}\n}\n',
        )

        gate.check_openapi_generated_metadata()

    def test_openapi_generated_metadata_rejects_stale_committed_json(self) -> None:
        gate = self.make_gate(
            self.make_repo(
                self.openapi_files('{"openapi":"3.1.0","info":{"version":"3.0.7"},"paths":{}}\n')
            )
        )
        self.install_fake_openapi_export(
            gate,
            '{"openapi":"3.1.0","info":{"version":"3.0.8"},"paths":{}}\n',
        )

        with self.assertRaisesRegex(RuntimeError, "frontend/public/openapi.json is stale"):
            gate.check_openapi_generated_metadata()

    def test_metadata_only_run_rejects_openapi_generated_drift(self) -> None:
        gate = self.make_metadata_gate(
            self.make_repo(
                self.openapi_files('{"openapi":"3.1.0","info":{"version":"3.0.7"},"paths":{}}\n')
            )
        )
        self.install_fake_openapi_export(
            gate,
            '{"openapi":"3.1.0","info":{"version":"3.0.8"},"paths":{}}\n',
        )

        self.assertEqual(gate.run(), 1)
        self.assertTrue(
            any("openapi_generated_metadata" in failure for failure in gate.failures),
            gate.failures,
        )


if __name__ == "__main__":
    unittest.main()
