---
name: superxray-ui-first-migration
description: Use when working on SuperXray-gui UI-first migration, Xray parity in the new Vue 3/Vite UI, new/legacy UI routing, CSP/CSRF hardening, phase-gated CoreManager entry, or reviewing plans under plans/01-strategy/ui-first-xray-stable-multi-core-roadmap.md.
---

# SuperXray UI First Migration

## Overview

Use this skill to keep SuperXray-gui work aligned with the approved route:

```text
UI first
  -> Xray parity and stability
  -> legacy UI fallback
  -> CoreManager default-xray
  -> sing-box experimental
  -> capability schema and multi-core expansion
```

## Required Context

Read only the sections needed for the task:

- Main plan: `plans/01-strategy/ui-first-xray-stable-multi-core-roadmap.md`
- Backend plan: `plans/02-architecture/backend-multi-core-architecture-plan.md`
- UI design: `plans/03-ui-design/multi-core-ui-design-plan.md`
- Phase gate quick reference: `references/phase-gates.md`

## Workflow

1. Identify the phase before changing files.
2. Check the phase gate in `references/phase-gates.md`.
3. State allowed files and forbidden files.
4. Keep UI phases on existing Xray APIs and old data models.
5. Run the phase's required checks.
6. Record rollback instructions.

## Non-Negotiable Gates

- Do not migrate `model.Inbound` during phases 0-9.
- Do not add sing-box during phases 0-9.
- Do not route old Xray lifecycle through CoreManager before phase 10.2.
- Do not write data from the new UI that the legacy UI cannot read.
- Do not render logs, config previews, or subscriptions with `v-html`.
- Do not remove the legacy UI before the new UI passes Xray parity E2E.

## Role Routing

- Planning and phase judgment: `superxray-ui-program-manager`
- Vue 3/Vite implementation: `superxray-frontend-migrator`
- Go embed, base path, CSP routing: `superxray-go-integration`
- CSP/CSRF/XSS/download/import checks: `superxray-security-gate`
- Browser journeys and artifacts: `superxray-e2e-gate`
- Release candidates: `superxray-release-gate`

## Validation

Backend:

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

Frontend from phase 1 onward:

```powershell
cd frontend
npm run typecheck
npm run lint
npm run build
```

Run E2E for any phase that touches user-visible flows.
