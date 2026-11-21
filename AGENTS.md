# Repository Guidelines

## Project layout

- `main.go` starts the panel. Keep Gin handlers in `web/controller/`, business logic in `web/service/`, and GORM models in `database/model/`.
- `sub/` serves subscriptions; `xray/` manages the Xray process. Keep existing Xray lifecycle operations separate from `core/` (CoreManager). Use `database/model.Inbound` for active inbound writes.
- Edit Vue 3/TypeScript in `frontend/src/` and static assets in `frontend/public/`. Vite generates `web/ui/`, which Go embeds and serves at `/panel/`. Do not hand-edit the bundle or restore `/panel/legacy/`.

## Local development and builds

Use the Go version in `go.mod`, a C compiler for SQLite (CGO), and Node.js 22.12+ with npm.

- **Root:** `go mod download` fetches dependencies; `go run ./main.go` starts the panel.
- **Frontend:** In `frontend/`, `npm ci` installs UI dependencies and `npm run dev` starts Vite.
- **Build order:** Run `npm run build` in `frontend/` to regenerate the OpenAPI JSON and UI bundle, then `go build -o bin/SuperXray.exe ./main.go` from the root for a Windows binary.

## Style and naming

Format Go with `gofmt` (tabs). For Vue and TypeScript, use two spaces, single quotes, and semicolons. Name components in PascalCase (`FormSection.vue`) and utilities in camelCase (`xrayCompat.ts`). In `frontend/`, run `npm run format` to check Prettier formatting and `npm run lint` for ESLint.

## Testing

Cover changed behavior and add regression tests for bug fixes. CI has no percentage-based coverage gate.

- **Go:** Place `TestXxx` functions in `*_test.go` files beside the code. For service changes, run `go test ./web/service` and `go vet ./web/service`. For full runs, use the package filter in [CONTRIBUTING.md](CONTRIBUTING.md) to exclude `node_modules` fixtures.
- **UI:** In `frontend/`, run `npm run typecheck` and `npm test`. The latter runs `frontend/tests/*.test.ts` with Node's test runner.
- **Browser:** Set `SUPERXRAY_E2E_BASE_URL`, `SUPERXRAY_E2E_USERNAME`, and `SUPERXRAY_E2E_PASSWORD` for a running, isolated panel. From the root, run `npm ci`, `npx playwright install chromium`, then `npm run e2e` (`tests/e2e/*.spec.ts`). Tests skip without credentials.

## Commits and pull requests

Use focused commits with a type (`feat`, `fix`, `docs`, `chore`) and a short Chinese summary, e.g. `feat: 支持从客户端总览启停客户端`. Link related issues, describe affected areas, report verification results and rollback steps, and attach screenshots for visible UI changes. Update `docs/` when behavior changes. Run `python scripts/secret_scan.py`; never commit `.env`, live databases, or real subscription URLs.
