---
name: superxray-project-context
description: "Use when scanning or modifying SuperXray-gui and Codex needs a current project map: directory structure, Go/Vue/legacy UI stack, core dependencies, business domains, phase gates, routing owners, and minimum verification commands."
---

# SuperXray Project Context

Use this skill as the first local orientation pass for SuperXray-gui tasks that involve code changes, `.codex` configuration, architecture analysis, agent routing, or broad project understanding.

## Quick Workflow

1. Read `.codex/project.toml`, `.codex/governance.toml`, `.codex/routing.toml`, and `.codex/context/project-map.md`.
2. Classify the task domain: frontend, Go integration, backend service, database, subscription/protocol, core runtime, security, E2E, DevOps/release, docs/i18n, or `.codex` governance.
3. Select the highest-priority route in `.codex/routing.toml`.
4. Read only the selected agent's `required_context` and the source files directly touched by the task.
5. Pick verification from `.codex/workflows/verification-matrix.md`.
6. If the task crosses phase boundaries, load `superxray-ui-first-migration`.
7. If the task touches release assets, workflows, versions, or `CHANGELOG.md`, load `superxray-release-cicd`.

## Project Facts

- Go module: `github.com/superaddmin/SuperXray-gui/v2`.
- Product: Xray-core web panel with new Vue UI, legacy UI fallback, subscription server, Gateway Egress MVP, and guarded multi-core runtime entry.
- Backend: Go 1.26.3, Gin, GORM, SQLite, Xray-core gRPC/API, robfig/cron, gorilla/websocket.
- Frontend: Vue 3.5, Vite 8, TypeScript 6, Pinia, Ant Design Vue 4, Axios.
- Legacy UI: `web/html` and `web/assets`, currently mounted under `/panel/legacy/`.
- New UI: `frontend/src`, built into `web/ui`, mounted at `/panel/` and compatible `/panel/ui/`.
- Subscription service: `sub/`, serving URI/Base64, Xray JSON, Clash/Mihomo, WireGuard config, and diagnose output.
- Release: Linux `amd64` and `arm64` binary assets plus optional GHCR multi-arch image.

## Business Domains

- Panel/security: login, session, CSRF, CSP, settings, backup/restore, logs.
- Xray runtime: legacy start/stop/restart, config template, version install, traffic stats.
- Inbounds: VMess, VLESS, Trojan, Shadowsocks, Hysteria2, WireGuard, clients, batch actions.
- Subscriptions: target-aware links, URI/JSON/Clash/Mihomo/WireGuard output, diagnostics.
- Gateway Egress MVP: Xray-compatible SOCKS5 inbound plus Gateway CSV manifest only.
- Core runtime: CoreManager, `default-xray` read-only view, `experimental-sing-box` external adapter.
- Release/deploy: GitHub Actions, Docker, install/update scripts, release gate.

## Non-Negotiable Boundaries

- Do not route legacy Xray start/stop/restart through CoreManager before Phase 10.2 approval.
- Do not migrate active writes away from `database/model.Inbound` to `proxy_inbounds` or `proxy_clients`.
- Do not remove `/panel/legacy/`, `web/html`, or `web/assets` before legacy retirement gates pass.
- Do not expand Gateway Egress MVP into production `egress_*` database/API without Phase 10+ design approval.
- Do not render logs, config previews, subscriptions, or external content with `v-html`, `innerHTML`, or `insertAdjacentHTML`.
- Do not store secrets, live database files, real subscription URLs, client UUIDs, panel paths, tokens, cookies, or full audit artifacts in the repository.

## Owner Selection

- `.codex/**`, plans, phase gates: `superxray-ui-program-manager`.
- `frontend/src/**`, `frontend/tests/**`, `web/ui/**`: `superxray-frontend-migrator`.
- `web/web.go`, `web/ui.go`, controllers, middleware, legacy routes: `superxray-go-integration`.
- `web/service/**`, `web/job/**`, `web/websocket/**`, `xray/**`, `util/**`: `superxray-backend-service-guardian`.
- `database/**`: `superxray-database-steward`.
- `sub/**`, protocol registry, compatibility utilities, Gateway MVP docs: `superxray-subscription-protocol-specialist`.
- `core/**`, Core API, CoreInstances view/types: `superxray-core-runtime-architect`.
- security-sensitive routes, imports, downloads, external calls, binary execution: `superxray-security-gate`.
- tests and verification matrix: `superxray-test-strategist`.
- Playwright flows: `superxray-e2e-gate`.
- `.github/**`, Docker, install/update scripts: `superxray-devops-cicd-maintainer`.
- version, changelog, release workflows/assets: `superxray-release-gate`.
- docs, i18n, README, plans status: `superxray-docs-i18n-maintainer`.

## Minimum Evidence Commands

Use read-only commands first:

```powershell
rg --files -g '!*node_modules*' -g '!web/ui/assets/**'
git status --short
rg "<symbol or route>" <target-paths> -n
```

Use the smallest relevant verification:

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
cd frontend; npm run typecheck; npm run lint; npm run test; npm run build
npm run e2e
python scripts/secret_scan.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```

## References

Read `.codex/context/project-map.md` for the compact map. Read `references/current-stack.md` only when a task needs a denser stack/dependency/business-domain summary without opening multiple docs.
