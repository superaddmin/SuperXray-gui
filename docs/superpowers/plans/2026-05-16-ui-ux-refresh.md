# SuperXray UI/UX Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the Vue 3 frontend closer to the approved digital-banking visual reference while fixing mobile navigation, action density, and accessibility issues.

**Architecture:** Keep all changes inside `frontend/src` and source-level tests. Do not change legacy APIs, Xray data models, CoreManager behavior, or generated `web/ui` except through the normal frontend build.

**Tech Stack:** Vue 3, Vite, TypeScript, Ant Design Vue 4, Node test runner, CSS media queries.

---

### Task 1: Regression Tests

**Files:**
- Modify: `frontend/tests/form-styles.test.ts`
- Modify: `frontend/tests/inbounds-view.test.ts`
- Create: `frontend/tests/layout-accessibility.test.ts`

- [x] Add tests that require mobile layout primitives, digital-banking color tokens, and card radius constraints.
- [x] Add tests that require the Inbounds page header to expose primary actions and a more-actions menu instead of keeping every action flat.
- [x] Add tests that require the language toggle accessible name to include the visible label.
- [x] Run `npm run test -- --test-name-pattern "mobile|digital banking|language|more actions"` from `frontend` and confirm the new tests fail.

### Task 2: Visual System

**Files:**
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/styles/app.css`

- [x] Replace neon-green primary tokens with banking-blue primary tokens and reserve green for success.
- [x] Reduce card and panel radius to operational-console proportions.
- [x] Keep contrast at WCAG AA or better for text, muted text, and focus rings.
- [x] Keep reduced-motion handling and existing CSP-compatible Ant Design nonce usage.

### Task 3: Mobile Layout

**Files:**
- Modify: `frontend/src/layouts/MainLayout.vue`
- Modify: `frontend/src/styles/app.css`

- [x] Add a mobile drawer navigation controlled by the existing menu button.
- [x] Hide the persistent sider below the mobile breakpoint so content uses the full viewport width.
- [x] Keep desktop sider behavior unchanged.
- [x] Ensure the header status bar does not overflow at 375px.

### Task 4: Action Density

**Files:**
- Modify: `frontend/src/views/InboundsView.vue`
- Modify: `frontend/src/styles/app.css`

- [x] Keep `Refresh`, `Refresh Activity`, and `New Inbound` visible as primary page actions.
- [x] Move imports, exports, gateway templates, and destructive bulk actions into a `More actions` dropdown.
- [x] Preserve all existing handler functions and legacy-compatible behavior.
- [x] Keep destructive actions visually separated inside the menu.

### Task 5: Verification

**Files:**
- No source files unless verification reveals a defect.

- [x] Run `cd frontend; npm run test`.
- [x] Run `cd frontend; npm run typecheck`.
- [x] Run `cd frontend; npm run lint`.
- [x] Run `cd frontend; npm run build`.
- [x] Start Vite locally and capture 375px, 768px, and 1440px screenshots for Dashboard and Inbounds.
- [x] Run Lighthouse snapshot on Dashboard and resolve any new accessibility regressions.
- [x] Run Lighthouse snapshot on Inbounds and resolve filter/accessible-name regressions.
