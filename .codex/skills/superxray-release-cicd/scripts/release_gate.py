#!/usr/bin/env python3
"""Release readiness gate for SuperXray-gui."""

from __future__ import annotations

import argparse
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import tomllib


SEMVER = re.compile(r"^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$")
TAG_SEMVER = re.compile(r"^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$")
SECRET_PATTERNS = [
    re.compile(r"AKIA[0-9A-Z]{16}"),
    re.compile(r"ghp_[A-Za-z0-9_]{30,}"),
    re.compile(r"sk-[A-Za-z0-9]{20,}"),
    re.compile(r"-----BEGIN (?:RSA|DSA|EC|OPENSSH) PRIVATE KEY-----"),
]


class Gate:
    def __init__(self, root: pathlib.Path, args: argparse.Namespace) -> None:
        self.root = root
        self.args = args
        self.failures: list[str] = []
        self.warnings: list[str] = []
        self.env = os.environ.copy()
        go_bin = self._go_bin()
        if go_bin:
            self.env["PATH"] = str(go_bin) + os.pathsep + self.env.get("PATH", "")
        mingw_bin = pathlib.Path(r"C:\msys64\mingw64\bin")
        if mingw_bin.exists():
            self.env["PATH"] = str(mingw_bin) + os.pathsep + self.env.get("PATH", "")

    def _go_bin(self) -> pathlib.Path | None:
        try:
            result = subprocess.run(
                ["go", "env", "GOPATH"],
                cwd=self.root,
                text=True,
                capture_output=True,
                check=True,
                env=os.environ.copy(),
            )
            return pathlib.Path(result.stdout.strip()) / "bin"
        except Exception:
            return None

    def run(self) -> int:
        checks = [
            self.check_clean_worktree,
            self.check_release_metadata,
            self.check_workflows,
            self.check_actionlint,
            self.check_go_mod_tidy,
            self.check_go_format,
            self.check_shell_syntax,
            self.check_translations,
            self.check_go_vet,
            self.check_staticcheck,
            self.check_tests,
            self.check_race_tests,
            self.check_govulncheck,
            self.check_gosec,
            self.check_binary_version,
            self.check_secret_patterns,
        ]
        for check in checks:
            name = check.__name__.replace("check_", "")
            print(f"==> {name}")
            try:
                check()
            except Exception as exc:  # noqa: BLE001 - gate should continue and summarize all blockers.
                self.failures.append(f"{name}: {exc}")
                print(f"FAIL {name}: {exc}")
        self.print_summary()
        return 1 if self.failures else 0

    def command(self, command: list[str], *, capture: bool = False) -> str:
        result = subprocess.run(
            command,
            cwd=self.root,
            text=True,
            capture_output=capture,
            env=self.env,
        )
        if result.returncode != 0:
            output = (result.stdout or "") + (result.stderr or "")
            raise RuntimeError(f"{' '.join(command)} failed with {result.returncode}\n{output[-4000:]}")
        return result.stdout if capture else ""

    def ensure_tool(self, name: str, install: list[str] | None = None) -> str:
        found = shutil.which(name, path=self.env.get("PATH"))
        if found:
            return found
        if install and self.args.install_tools:
            self.command(install)
            found = shutil.which(name, path=self.env.get("PATH"))
            if found:
                return found
        raise RuntimeError(f"required tool not found: {name}")

    def bash(self) -> str:
        candidates = [
            r"C:\msys64\usr\bin\bash.exe",
            r"C:\Program Files\Git\bin\bash.exe",
            r"C:\Program Files\Git\usr\bin\bash.exe",
            "/usr/bin/bash",
            "/bin/bash",
            shutil.which("bash", path=self.env.get("PATH")),
        ]
        for candidate in candidates:
            if candidate and pathlib.Path(candidate).exists():
                return candidate
        raise RuntimeError("bash not found")

    def check_clean_worktree(self) -> None:
        if self.args.allow_dirty:
            self.warnings.append("dirty worktree allowed by --allow-dirty")
            return
        status = self.command(["git", "status", "--porcelain"], capture=True)
        if status.strip():
            raise RuntimeError("working tree has uncommitted changes")

    def version(self) -> str:
        version = (self.root / "config" / "version").read_text(encoding="utf-8").strip()
        if not SEMVER.match(version):
            raise RuntimeError(f"config/version is not semantic version: {version}")
        return version

    def release_tag(self) -> str | None:
        if self.args.tag:
            return self.args.tag
        if os.environ.get("GITHUB_REF_TYPE") == "tag":
            return os.environ.get("GITHUB_REF_NAME")
        ref = os.environ.get("GITHUB_REF", "")
        if ref.startswith("refs/tags/"):
            return ref.removeprefix("refs/tags/")
        return None

    def check_release_metadata(self) -> None:
        version = self.version()
        changelog = self.root / "CHANGELOG.md"
        if not changelog.exists():
            raise RuntimeError("CHANGELOG.md is missing")
        changelog_text = changelog.read_text(encoding="utf-8")
        if f"## [{version}]" not in changelog_text:
            raise RuntimeError(f"CHANGELOG.md missing section ## [{version}]")
        stale_result = subprocess.run(
            ["git", "grep", "-n", r"2\.9\.3\|v2\.9\.3", "--", "README*.md", "docs", ".github", "config"],
            cwd=self.root,
            text=True,
            capture_output=True,
            env=self.env,
        )
        if stale_result.returncode not in (0, 1):
            raise RuntimeError(stale_result.stderr or stale_result.stdout)
        stale = stale_result.stdout
        if stale.strip():
            raise RuntimeError(f"stale version references found:\n{stale}")
        tag = self.release_tag()
        if tag:
            if not TAG_SEMVER.match(tag):
                raise RuntimeError(f"release tag is not semantic: {tag}")
            if tag.removeprefix("v") != version:
                raise RuntimeError(f"tag {tag} does not match config/version {version}")

    def check_workflows(self) -> None:
        release = self._read(".github/workflows/release.yml")
        docker = self._read(".github/workflows/docker.yml")
        arm64 = self._read(".github/workflows/test-arm64.yml")
        codeql = self._read(".github/workflows/codeql.yml")
        required_release_tokens = [
            "v*.*.*",
            "Validate release metadata",
            "Generate release notes",
            "body_path: release-notes.md",
            "actions/setup-go",
            "gcc-aarch64-linux-gnu",
            "GOARCH=\"$goarch\"",
            "x-ui-linux-${{ matrix.platform }}.tar.gz",
            "- amd64",
            "- arm64",
        ]
        for token in required_release_tokens:
            if token not in release:
                raise RuntimeError(f"release.yml missing required token: {token}")
        forbidden_release_tokens = [
            "build-windows",
            "windows-latest",
            "BOOTLIN_ARCH",
            "- armv7",
            "- armv6",
            "- armv5",
            "- 386",
            "- s390x",
        ]
        for token in forbidden_release_tokens:
            if token in release:
                raise RuntimeError(f"release.yml contains unsupported release target/toolchain: {token}")
        for token in ["linux/amd64", "linux/arm64"]:
            if token not in docker:
                raise RuntimeError(f"docker.yml missing platform: {token}")
        if "ghcr.io/superaddmin/superxray-gui" not in docker:
            raise RuntimeError("docker.yml must publish the lower-case GHCR image name")
        if "DOCKER_HUB_TOKEN" in docker or "hsanaeii/3x-ui" in docker:
            raise RuntimeError("docker.yml must not require Docker Hub credentials for releases")
        if "platform linux/arm64" not in arm64:
            raise RuntimeError("test-arm64.yml must verify arm64 under QEMU")
        if "gcc-aarch64-linux-gnu" not in arm64:
            raise RuntimeError("test-arm64.yml must cross-compile arm64 with gcc-aarch64-linux-gnu")
        if "golang:1.26-alpine" in arm64:
            raise RuntimeError("test-arm64.yml must not compile Go inside the emulated arm64 container")
        if "github/codeql-action/analyze" not in codeql:
            raise RuntimeError("codeql.yml must run CodeQL analysis")

    def _read(self, relative: str) -> str:
        path = self.root / relative
        if not path.exists():
            raise RuntimeError(f"missing {relative}")
        return path.read_text(encoding="utf-8")

    def check_go_mod_tidy(self) -> None:
        self.command(["go", "mod", "tidy", "-diff"])

    def check_actionlint(self) -> None:
        self.ensure_tool(
            "actionlint",
            ["go", "install", "github.com/rhysd/actionlint/cmd/actionlint@latest"],
        )
        self.command(["actionlint"])

    def check_go_format(self) -> None:
        output = self.command(["gofmt", "-l", "."], capture=True)
        if output.strip():
            raise RuntimeError(f"unformatted Go files:\n{output}")

    def check_shell_syntax(self) -> None:
        bash = self.bash()
        for script in ["install.sh", "x-ui.sh", "update.sh", "DockerEntrypoint.sh", "DockerInit.sh"]:
            self.command([bash, "-n", script])

    def check_translations(self) -> None:
        for path in sorted((self.root / "web" / "translation").glob("*.toml")):
            tomllib.loads(path.read_text(encoding="utf-8"))

    def check_go_vet(self) -> None:
        self.command(["go", "vet", "./..."])

    def check_staticcheck(self) -> None:
        self.ensure_tool("staticcheck")
        self.command(["staticcheck", "./..."])

    def check_tests(self) -> None:
        self.command(["go", "test", "./..."])

    def check_race_tests(self) -> None:
        self.command(["go", "test", "-race", "./..."])

    def check_govulncheck(self) -> None:
        self.ensure_tool(
            "govulncheck",
            ["go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"],
        )
        self.command(["govulncheck", "./..."])

    def check_gosec(self) -> None:
        self.ensure_tool(
            "gosec",
            ["go", "install", "github.com/securego/gosec/v2/cmd/gosec@latest"],
        )
        self.command(["gosec", "-quiet", "./..."])

    def check_binary_version(self) -> None:
        version = self.version()
        suffix = ".exe" if os.name == "nt" else ""
        with tempfile.TemporaryDirectory() as temp_dir:
            output = pathlib.Path(temp_dir) / f"superxray-gui-verify{suffix}"
            self.command(["go", "build", "-ldflags=-s -w", "-o", str(output), "main.go"])
            reported = subprocess.run(
                [str(output), "-v"],
                cwd=self.root,
                text=True,
                capture_output=True,
                env=self.env,
                check=True,
            ).stdout.strip()
        if reported != version:
            raise RuntimeError(f"binary reports {reported}, expected {version}")

    def check_secret_patterns(self) -> None:
        include_suffixes = {
            ".go",
            ".sh",
            ".yml",
            ".yaml",
            ".md",
            ".toml",
            ".json",
            ".env",
            ".example",
        }
        excluded_dirs = {".git", "dist", "log", "node_modules"}
        findings: list[str] = []
        for path in self.root.rglob("*"):
            if not path.is_file():
                continue
            if any(part in excluded_dirs for part in path.relative_to(self.root).parts):
                continue
            if path.name == "SuperXray-gui" or path.suffix.lower() not in include_suffixes:
                continue
            try:
                text = path.read_text(encoding="utf-8")
            except UnicodeDecodeError:
                continue
            for pattern in SECRET_PATTERNS:
                if pattern.search(text):
                    findings.append(str(path.relative_to(self.root)))
        if findings:
            raise RuntimeError("possible secrets found in: " + ", ".join(sorted(set(findings))))

    def print_summary(self) -> None:
        print("\nRelease gate summary")
        print("====================")
        if self.warnings:
            print("Warnings:")
            for warning in self.warnings:
                print(f"- {warning}")
        if self.failures:
            print("Failures:")
            for failure in self.failures:
                print(f"- {failure}")
        else:
            print("PASS: repository satisfies the SuperXray release gate.")


def find_root(start: pathlib.Path) -> pathlib.Path:
    for candidate in [start, *start.parents]:
        if (candidate / "go.mod").exists() and (candidate / ".github" / "workflows").exists():
            return candidate
    raise RuntimeError("could not find SuperXray-gui repository root")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run SuperXray-gui release readiness checks.")
    parser.add_argument("--ci", action="store_true", help="Run in CI mode; currently informational.")
    parser.add_argument("--install-tools", action="store_true", help="Install govulncheck/gosec when missing.")
    parser.add_argument("--allow-dirty", action="store_true", help="Allow uncommitted changes while developing.")
    parser.add_argument("--tag", help="Release tag to validate, for example v2.9.4.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = find_root(pathlib.Path.cwd().resolve())
    return Gate(root, args).run()


if __name__ == "__main__":
    sys.exit(main())
