# SuperXray-gui Release Policy

## Required Gates

Every releasable commit must satisfy:

- Clean Git worktree unless running in a temporary CI checkout.
- `config/version` is semantic version format: `X.Y.Z` or `X.Y.Z-prerelease`.
- Tag releases use `v` prefix and match `config/version`.
- `CHANGELOG.md` contains a section exactly named `## [X.Y.Z]`.
- `go.mod` and `go.sum` are tidy.
- GitHub Actions workflows pass `actionlint`.
- Go formatting, `go vet`, `staticcheck`, unit tests, and race tests pass.
- `govulncheck` reports no reachable vulnerabilities.
- `gosec` reports no unsuppressed findings. Suppressions must include a reason.
- Shell scripts parse with Bash.
- `web/translation/*.toml` files parse successfully.
- The built binary reports the same version as `config/version`.

## GitHub Actions Strategy

`release.yml` is the binary release workflow:

- Analyze first, then build.
- Publish GitHub Release assets only for pushed semantic tags.
- Generate release notes from `CHANGELOG.md`.
- Linux `amd64` and `arm64` artifacts are mandatory for Ubuntu server deployments.
- Do not build Windows or legacy Linux release artifacts unless the project release policy changes.
- Build Linux release binaries on Ubuntu runners with CGO enabled; use `gcc` for amd64 and `gcc-aarch64-linux-gnu` for arm64.
- Keep ARM64 execution checks in `test-arm64.yml`, where QEMU runs the built ARM64 binary.

`docker.yml` is the container release workflow:

- Trigger only on semantic tags or manual dispatch.
- Publish `linux/amd64` and `linux/arm64` images to `ghcr.io/superaddmin/superxray-gui`.
- Do not require Docker Hub credentials for the default release path.

`test-arm64.yml` is the ARM64 confidence workflow:

- Cross-compile the ARM64 binary with `gcc-aarch64-linux-gnu`.
- Execute the ARM64 binary under an Ubuntu ARM64 container via QEMU.
- Keep this workflow green before release.

`codeql.yml` is the security analysis workflow:

- Run on pull requests, branch pushes, and schedule.
- Do not ignore CodeQL failures for release candidates.

## Release Notes

Release notes must be generated from the matching CHANGELOG section. If GitHub's generated notes are also enabled, keep the CHANGELOG section as the human-authored source of truth and append generated commit/PR notes below it.

## Rollback Rules

- Prefer forward fixes through a patch release after a public stable release.
- Delete or recreate tags only for unpublished or internal prerelease tags.
- If upload jobs create a partial prerelease, delete the release assets and rerun the workflow after fixing the cause.
- Never overwrite a stable release asset without documenting why in the release notes.
