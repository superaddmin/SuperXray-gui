# SuperXray UI/UX Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the Vue 3 frontend closer to the approved digital-banking visual reference while fixing mobile navigation, action density, and accessibility issues.

**Architecture:** Keep all changes inside `frontend/src` and source-level tests. Do not change legacy APIs, Xray data models, CoreManager behavior, or generated `web/ui` except through the normal frontend build.

**Tech Stack:** Vue 3, Vite, TypeScript, Ant Design Vue 4, Node test runner, CSS media queries.

**Latest update (2026-05-16):** Follow-up UI audit fixes are complete for the Xray information architecture, Gateway MVP placement, Chinese i18n cleanup, Header token override, and mobile drawer focus/touch behavior. Updated screenshots are stored in `docs/assets/xray-mvp-desktop.png` and `docs/assets/xray-mvp-mobile.png`.

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

### Task 6: Xray Workspace And Mobile Polish

**Files:**
- Modify: `frontend/src/views/XrayView.vue`
- Modify: `frontend/src/views/InboundsView.vue`
- Modify: `frontend/src/layouts/MainLayout.vue`
- Modify: `frontend/src/i18n/messages.ts`
- Modify: `frontend/src/styles/app.css`
- Modify: `frontend/tests/*.test.ts`
- Modify: `README.md`
- Modify: `README.zh_CN.md`
- Modify: `docs/ui-ux-audit.md`
- Add: `docs/assets/xray-mvp-desktop.png`
- Add: `docs/assets/xray-mvp-mobile.png`

- [x] Add source tests that require Xray workspace navigation and Gateway MVP placement before the template editor.
- [x] Add source tests that require Gateway MVP actions and labels to use i18n keys.
- [x] Add source tests for Header glass token override and mobile drawer close-focus behavior.
- [x] Move Gateway Egress MVP above the long Xray template editor.
- [x] Add the Xray workspace navigation card with anchors for runtime, Gateway, template, outbound tools, structured config, DNS policy, and protocol tools.
- [x] Replace remaining high-priority mixed English labels in the Xray/Gateway first viewport with Chinese i18n text.
- [x] Override `.app-header.ant-layout-header` so Ant default layout background cannot replace the approved glass token.
- [x] Add an explicit mobile drawer close button, focus it after drawer open, and keep compact controls at touch-safe dimensions.
- [x] Capture updated desktop and mobile screenshots for documentation.

**Browser evidence:**
- Desktop 1440px: Header computed background is `rgba(6, 17, 31, 0.84)` and Gateway MVP appears before the template editor.
- Mobile 375px: no horizontal overflow; workspace buttons are 44px tall; drawer close control is 44x44 and receives focus with aria-label `关闭导航`.
