# Contributing

Thank you for contributing to SuperXray. Keep changes focused, follow the nearest repository instructions, and use current source files as the authority.

## Setup

Requirements:

- Go `1.26.4` and a C toolchain for CGO/SQLite builds.
- Node.js and npm compatible with the committed lockfiles.

```powershell
Copy-Item .env.example .env
go mod download

# Root Playwright/E2E dependencies
npm ci

# Vue frontend dependencies
Set-Location frontend
npm ci
Set-Location ..
```

Start the Go service with `go run ./main.go`. For Vue development, run `npm run dev` from `frontend/`.

## Minimum verification

Run the smallest set that covers the changed area.

### Go/backend

`node_modules` can contain unrelated Go fixtures after `npm ci`. Build the package array from `go list` and filter those import paths before full tests.

```powershell
$goPackages = @(go list ./... | Where-Object { $_ -notmatch '/node_modules/' })
if ($LASTEXITCODE -ne 0 -or $goPackages.Count -eq 0) { throw 'go list failed or returned no project packages' }
go test $goPackages
go vet $goPackages
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin/SuperXray.exe ./main.go
```

### Vue frontend

```powershell
Set-Location frontend
npm run typecheck
npm run lint
npm run test
npm run build
Set-Location ..
```

Run `npm run e2e` from the repository root when a browser workflow is affected.

## Documentation and secrets

- Update README or `docs/` content when behavior, API names, parameters, configuration, or deployment steps change.
- Verify examples against `go.mod`, `frontend/package.json`, `.codex/project.toml`, and the current implementation.
- Keep credentials, private keys, tokens, cookies, live databases, subscription identifiers, and server audit artifacts out of commits.
- Run the repository secret scan before submitting changes:

```powershell
python scripts/secret_scan.py
```

## Change notes

Describe the problem, solution, impact, verification commands and results, risks, and rollback path. Avoid unrelated refactors, dependency upgrades, generated-file edits, or bulk formatting.
