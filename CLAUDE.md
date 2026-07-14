# CLAUDE.md

This repository keeps project facts in shared source-of-truth files. Read those files instead of relying on a duplicated architecture summary here.

## Start here

1. Follow the active `AGENTS.md` instructions.
2. Read `.codex/project.toml`, `.codex/governance.toml`, `.codex/routing.toml`, and `.codex/context/project-map.md`.
3. Select verification from `.codex/workflows/verification-matrix.md`.
4. Read only the source files and active plans relevant to the task.

Current anchors:

- Module: `github.com/superaddmin/SuperXray-gui/v2`.
- Go: `1.26.5` from `go.mod`.
- Vue source: `frontend/src`; scripts and versions: `frontend/package.json`.
- Vite output: `web/ui`; Go embedding and routes: `web/ui.go`.
- Legacy `web/html`, `web/assets`, and `/panel/legacy*` paths are retired.
- Current phase and exceptions: `plans/STATUS.md` and the phase references listed by `.codex/project.toml`.

## Working rules

- Make the smallest change in the routed owner domain.
- Preserve `database/model.Inbound` as the active Xray write model.
- Keep the legacy Xray lifecycle outside CoreManager until its phase gate changes.
- Preserve CSP/CSRF, authenticated downloads, safe import, path, URL, and execution checks.
- Keep secrets, private keys, runtime databases, subscription data, and audit artifacts out of Git.
- Update user-facing documentation when behavior or interfaces change.

## Common verification

Installed npm dependencies can contain unrelated Go fixtures, so enumerate project packages and exclude import paths under `node_modules` before full Go tests.

```powershell
$goPackages = @(go list ./... | Where-Object { $_ -notmatch '/node_modules/' })
if ($LASTEXITCODE -ne 0 -or $goPackages.Count -eq 0) { throw 'go list failed or returned no project packages' }
go test $goPackages
go vet $goPackages
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin/SuperXray.exe ./main.go

Set-Location frontend
npm ci
npm run typecheck
npm run lint
npm run test
npm run build
Set-Location ..

python scripts/secret_scan.py
```

Release work additionally follows `.github/agentic-workflows/release.md` and its release gate. Report only commands that were actually executed.
