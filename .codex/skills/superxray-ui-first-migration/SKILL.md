---
name: superxray-ui-first-migration
description: Use when working on SuperXray-gui UI-first migration, Xray parity, legacy UI fallback, CSP/CSRF hardening, Gateway Egress MVP, CoreManager/default-xray/sing-box Phase 10 gates, or plans under plans/01-strategy and plans/04-ui-first-execution.
---

# SuperXray UI First Migration

Use this skill to keep SuperXray-gui work aligned with the approved route:

```text
UI first
  -> Xray parity and stability
  -> legacy UI fallback
  -> Phase 9 security closeout
  -> risk-accepted minimal CoreManager/default-xray/sing-box backend entry
  -> Phase 10.2+ lifecycle and capability expansion only after gates pass
```

## Required Context

Read only the sections needed for the task:

- Current status: `plans/STATUS.md`
- Main plan: `plans/01-strategy/ui-first-xray-stable-multi-core-roadmap.md`
- Backend plan: `plans/02-architecture/backend-multi-core-architecture-plan.md`
- UI design: `plans/03-ui-design/multi-core-ui-design-plan.md`
- Phase gate quick reference: `references/phase-gates.md`
- Phase 10 risk acceptance: `plans/04-ui-first-execution/phase-10-entry-gate-assessment.md`
- default-xray ADR: `plans/04-ui-first-execution/phase-10a-default-xray-readonly-adr.md`

## Workflow

1. Identify the phase before changing files.
2. Check the phase gate in `references/phase-gates.md` and `plans/STATUS.md`.
3. State allowed files, forbidden files, owner agent, reviewers, and verification.
4. Keep UI phases on existing Xray APIs and old data models.
5. For Phase 10 work, distinguish risk-accepted minimal backend entry from still-blocked lifecycle/model migration.
6. Run the phase's required checks.
7. Record rollback instructions.

## Non-Negotiable Gates

- Do not migrate active `model.Inbound` writes to `proxy_inbounds` or `proxy_clients`.
- Do not route old Xray lifecycle through CoreManager before Phase 10.2 approval.
- Do not promote experimental sing-box to production default.
- Do not write data from the new UI that the legacy UI cannot read.
- Do not remove legacy UI before the new UI passes parity, E2E, release, and rollback gates.
- Do not render logs, config previews, subscriptions, imports, or external content with `v-html`, `innerHTML`, or `insertAdjacentHTML`.
- Do not turn Gateway Egress MVP docs into production `egress_*` database/API without Phase 10+ design approval.

## Current Allowed Phase 10 Exception

The repository has accepted a limited risk exception:

- `CoreManager` can exist as a minimal backend registry.
- `default-xray` can be exposed as a read-only instance view.
- `experimental-sing-box` can exist as an isolated external binary adapter.

This exception does not allow:

- Legacy Xray start/stop/restart through CoreManager.
- Active inbound/client schema migration.
- sing-box production default routing.
- Capability-driven dynamic form writes to new production tables.

## Role Routing

- Planning and phase judgment: `superxray-ui-program-manager`
- Vue 3/Vite implementation: `superxray-frontend-migrator`
- Go embed, base path, CSP routing: `superxray-go-integration`
- Xray services and jobs: `superxray-backend-service-guardian`
- CoreManager/sing-box boundary: `superxray-core-runtime-architect`
- Database contracts: `superxray-database-steward`
- Protocol/subscription/Gateway MVP: `superxray-subscription-protocol-specialist`
- CSP/CSRF/XSS/download/import checks: `superxray-security-gate`
- Tests and coverage: `superxray-test-strategist`
- Browser journeys and artifacts: `superxray-e2e-gate`
- Release candidates: `superxray-release-gate`

## Validation

Backend:

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

Frontend:

```powershell
cd frontend
npm run typecheck
npm run lint
npm run test
npm run build
```

Security searches:

```powershell
rg "v-html|innerHTML|insertAdjacentHTML" web/html frontend/src -n
rg "unsafe-inline|unsafe-eval" frontend/src web/ui -n
rg "proxy_inbounds|proxy_clients" core web/controller web/middleware web/service database/model frontend/src web/ui -n
```

Run E2E for any phase that touches user-visible flows, login/session/base path behavior, mutation/import/restart paths, or legacy compatibility.
