---
name: superxray-release-cicd
description: Verify and maintain the SuperXray-gui release CI/CD pipeline. Use when preparing a release, reviewing tag/main push readiness, changing GitHub Actions release workflows, validating semantic version and CHANGELOG policy, packaging multi-architecture artifacts, or diagnosing failed GitHub Releases deployments.
---

# SuperXray Release CI/CD

## Purpose

Use this skill to keep SuperXray-gui releases reproducible and safe. The repository's event trigger is GitHub Actions; this skill provides the release gate, workflow checks, deployment policy, and rollback playbook that an agent should apply before code is merged or tagged.

## Quick Gate

From the repository root, run the executable gate:

```powershell
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --install-tools
```

In GitHub Actions or Linux shells:

```bash
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --install-tools
```

The gate checks Git status, semantic versioning, `CHANGELOG.md`, workflow trigger policy, GitHub Actions linting, Go formatting, `go vet`, `staticcheck`, tests, race tests, `govulncheck`, `gosec`, shell syntax, TOML translations, and a local binary version check.

## Trigger Policy

- Pull requests and pushes to `main` must run validation, tests, static analysis, security scans, and build checks.
- Only tags matching `vX.Y.Z` or `vX.Y.Z-prerelease` may publish GitHub Releases.
- A release tag must match `config/version` without the leading `v`.
- `CHANGELOG.md` must contain `## [X.Y.Z]` for the version being released.
- Release notes must be generated from that CHANGELOG section or by GitHub release-note generation. Do not publish blank releases.
- Linux `amd64` and `arm64` artifacts are mandatory and are the only default binary release targets.
- Binary release builds must use Ubuntu runners with CGO enabled: native `gcc` for amd64 and `gcc-aarch64-linux-gnu` for arm64.
- Do not publish Windows or legacy Linux architecture artifacts unless the release policy is explicitly changed.
- Docker images may be pushed to GHCR on release tags after the binary release gate succeeds.

## Deployment Steps

1. Confirm `git status --porcelain` is clean.
2. Confirm `config/version`, docs, README references, and `CHANGELOG.md` agree.
3. Run `release_gate.py --install-tools`.
4. Push changes to `main`; wait for CI and CodeQL.
5. Create an annotated semantic version tag:

```bash
git tag -a vX.Y.Z -m "release: vX.Y.Z"
git push origin vX.Y.Z
```

6. Watch `.github/workflows/release.yml` and `.github/workflows/docker.yml`.
7. Verify GitHub Releases contains at least:

```text
x-ui-linux-amd64.tar.gz
x-ui-linux-arm64.tar.gz
release notes for vX.Y.Z
```
8. Verify GHCR contains the optional multi-arch image `ghcr.io/superaddmin/superxray-gui` for `linux/amd64` and `linux/arm64` when `docker.yml` runs.

## Rollback

- If the gate fails before a tag is pushed, do not release. Fix code or release metadata and rerun the gate.
- If a tag workflow fails before assets are published, fix the blocker and rerun the workflow for the same tag.
- If a release is partially published, delete the draft/prerelease release and rerun after fixing the failed job.
- If a stable release is already public and broken, publish a patch tag (`vX.Y.(Z+1)`) instead of rewriting history.
- Only delete a remote tag when no production users could have consumed it:

```bash
git push origin :refs/tags/vX.Y.Z
git tag -d vX.Y.Z
```

## References

- Read `references/release-policy.md` before changing release workflow behavior.
- Use `scripts/release_gate.py` as the deterministic check. Patch the script when project release policy changes instead of duplicating checks in ad hoc commands.
