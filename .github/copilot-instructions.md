# SuperXray Repository Instructions

Use repository facts instead of copying architecture descriptions into this file.

## Required orientation

Read these sources before changing code or documentation:

1. `AGENTS.md` instructions supplied by the active workspace.
2. `.codex/project.toml` for the current stack, phase, source-of-truth map, and hard gates.
3. `.codex/governance.toml` and `.codex/routing.toml` for task boundaries, ownership, and verification.
4. `.codex/context/project-map.md` for the compact project map.
5. The files directly affected by the task and the relevant entry in `.codex/workflows/verification-matrix.md`.

For release work, also read `.github/agentic-workflows/release.md`. For implementation status, use `plans/STATUS.md` rather than historical reports under `docs/superpowers/`.

## Current facts

- Go module: `github.com/superaddmin/SuperXray-gui/v2`.
- Go version: `1.26.4`, defined by `go.mod`.
- Backend: Gin, GORM, SQLite, Xray-core integration, subscription service, and background jobs.
- Frontend source: `frontend/src` using Vue 3.5, Vite 8, TypeScript 6, Pinia 3, Vue Router 4, Ant Design Vue 4, and Axios.
- Frontend build output: `web/ui`, embedded and served by `web/ui.go` at `/panel/` with `/panel/ui/` compatibility.
- `web/html`, `web/assets`, and `/panel/legacy*` are retired and must not be restored.

Do not hand-edit generated `web/ui` assets. Change `frontend/src`, run the frontend checks, and rebuild through `frontend/package.json` when generated output is part of the requested change.

## Non-negotiable boundaries

- Keep active Xray writes on `database/model.Inbound` until an explicit architecture gate changes that contract.
- Do not route the legacy Xray lifecycle through CoreManager before Phase 10.2 approval.
- Do not render logs, configuration, subscriptions, or external content with `v-html`, `innerHTML`, or `insertAdjacentHTML`.
- State-changing APIs require CSRF protection; downloads, imports, URLs, paths, and executable inputs require the existing security checks.
- Do not commit credentials, private keys, live databases, subscription identifiers, cookies, tokens, or full server audit artifacts.
- Preserve project-native tooling and avoid unrelated refactors, dependency upgrades, or bulk formatting.

## Verification

Choose the smallest relevant set, then run the broader gate when the change crosses domains.

After npm dependencies are installed, `go list ./...` can discover third-party Go fixtures under `node_modules`. Full Go verification therefore filters those import paths first.

```powershell
# Go
$goPackages = @(go list ./... | Where-Object { $_ -notmatch '/node_modules/' })
if ($LASTEXITCODE -ne 0 -or $goPackages.Count -eq 0) { throw 'go list failed or returned no project packages' }
go test $goPackages
go vet $goPackages
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin/SuperXray.exe ./main.go

# Frontend
Set-Location frontend
npm ci
npm run typecheck
npm run lint
npm run test
npm run build
Set-Location ..

# Repository secrets
python scripts/secret_scan.py
```

Use `npm run e2e` from the repository root only for flows that require browser coverage. Record the commands actually run and their results; never invent verification output.
