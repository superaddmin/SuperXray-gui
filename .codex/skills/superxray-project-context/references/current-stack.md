# SuperXray Current Stack Reference

## Structure

```text
main.go                 CLI, env, DB init, web/sub server lifecycle
config/                 version/name/path helpers
database/               SQLite/GORM init and models
web/                    panel server, controllers, services, middleware, jobs, websocket, legacy UI
frontend/               Vue 3/Vite source and tests
web/ui/                 generated Vite build output embedded by Go
sub/                    subscription server and protocol outputs
xray/                   legacy Xray process/API/traffic integration
core/                   CoreManager and experimental adapters
tests/e2e/              Playwright acceptance journeys
.github/                release/docker/codeql/arm64 workflows
.codex/                 project-local AI governance, agents, workflows, skills
```

## Core Dependencies

Backend:

- Gin: API routing and middleware.
- GORM + SQLite: persistent panel state.
- Xray-core + gRPC: Xray config/API/process integration.
- robfig/cron: periodic jobs.
- gorilla/websocket: real-time traffic and status updates.
- telego, ldap, gotp: Telegram, LDAP, two-factor flows.

Frontend:

- Vue 3.5, Vite 8, TypeScript 6.
- Pinia, Vue Router, Ant Design Vue 4, Axios.
- Node test through `frontend/package.json`, covering TS tests and legacy JS model tests.

Release:

- `.github/workflows/release.yml`: Linux amd64/arm64 binary packages.
- `.github/workflows/docker.yml`: GHCR multi-arch images.
- `.github/workflows/test-arm64.yml`: ARM64 cross compile and QEMU execution.
- `.codex/skills/superxray-release-cicd/scripts/release_gate.py`: metadata and release policy gate.

## Current Phase Facts

- Phase 9 security closeout is still the main stabilization lane.
- Phase 10 has a recorded risk acceptance for minimal backend runtime entry.
- `default-xray` is a read-only view over legacy Xray status.
- `experimental-sing-box` is an isolated external adapter, not the production default.
- CI/second-environment reproduction remains a blocker for full Phase 10 confidence.

## Compatibility Contracts

- New UI writes must be readable by legacy UI and old APIs.
- Active inbound writes use `database/model.Inbound`.
- Client data remains embedded in `Inbound.Settings` JSON.
- Subscription output reads the legacy model.
- Gateway Egress MVP writes Xray config template data and CSV manifest only.
