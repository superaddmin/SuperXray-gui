# SuperXray Fintech Ops UI Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将新 Vue 3 UI 优化为与参考页一致的 Fintech Ops Console 风格，并刷新 `web/ui/` 嵌入式构建产物。

**Architecture:** 以 `frontend/src/styles/app.css` 和 Ant Design Vue token 为设计系统入口，通过少量共享组件样式 API 承载页面差异。业务数据流、后端 API、旧 UI 回退入口和 Xray/CoreManager 生命周期保持不变。

**Tech Stack:** Vue 3、Vite、TypeScript、Pinia、Vue Router、Ant Design Vue 4、`@fontsource/*` 自托管字体、Playwright 浏览器验收。

---

## 执行约束

- 当前工作区已有未提交的 `frontend/` 与 `web/ui/` 改动，本计划在当前工作区继续实施，不新建 worktree，避免丢失这些上下文。
- 不执行 `git commit`，最终只汇报变更与验证结果。
- 不回滚、不覆盖与本任务无关的既有改动。
- 不修改 `web/html/`、旧 `web/assets/`、Go 后端 API、数据库模型、订阅输出、旧 UI 回退路由。
- 不新增外部运行时 CDN；字体通过 npm 包进入 Vite 构建产物。

## File Structure

- Modify `frontend/package.json` and `frontend/package-lock.json`: add self-hosted font packages.
- Modify `frontend/src/main.ts`: import selected font weights before app styles.
- Modify `frontend/src/App.vue`: align Ant Design Vue token with fintech palette and fonts.
- Modify `frontend/src/styles/app.css`: replace global visual system, layout, components, Ant Design overrides, responsive and focus states.
- Modify `frontend/src/layouts/MainLayout.vue`: add shell labels and route-aware class hooks without changing routing.
- Modify `frontend/src/components/PageHeader.vue`: support optional description/status slots while preserving current props.
- Modify `frontend/src/components/StatusTile.vue`: add optional tone prop and semantic class names.
- Modify `frontend/src/components/AppStatusBar.vue`: render compact status strip using existing store data.
- Modify `frontend/src/views/LoginView.vue`: add fintech login copy structure and class hooks.
- Modify `frontend/src/views/DashboardView.vue`: add dashboard command header and panel class hooks.
- Modify `frontend/src/views/CoreInstancesView.vue`, `InboundsView.vue`, `XrayView.vue`, `LogsView.vue`, `SettingsView.vue`: add page-specific class hooks only where needed.
- Generate `web/ui/`: refreshed Vite build output.

## Task 1: Font Packages And Theme Tokens

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Modify: `frontend/src/main.ts`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Install self-hosted fonts**

Run:

```powershell
cd frontend
npm install @fontsource/dm-sans @fontsource/space-grotesk
```

Expected: `package.json` and `package-lock.json` include both packages, with no audit vulnerabilities reported by npm install.

- [ ] **Step 2: Import only required font weights**

In `frontend/src/main.ts`, keep Ant Design reset first, then add font imports before `./styles/app.css`:

```ts
import 'ant-design-vue/dist/reset.css';
import '@fontsource/dm-sans/400.css';
import '@fontsource/dm-sans/500.css';
import '@fontsource/dm-sans/700.css';
import '@fontsource/space-grotesk/600.css';
import '@fontsource/space-grotesk/700.css';
import './styles/app.css';
```

- [ ] **Step 3: Update Ant Design token palette**

In `frontend/src/App.vue`, set token values to the approved palette:

```ts
token: {
  borderRadius: 16,
  colorBgBase: '#0a0e27',
  colorBgContainer: '#0f1635',
  colorBgElevated: '#10183a',
  colorBgLayout: '#0a0e27',
  colorBorder: 'rgba(30, 38, 80, 0.72)',
  colorBorderSecondary: 'rgba(30, 38, 80, 0.48)',
  colorError: '#ef4444',
  colorInfo: '#0080ff',
  colorLink: '#0080ff',
  colorPrimary: '#39ff14',
  colorPrimaryHover: '#7dff6a',
  colorSuccess: '#39ff14',
  colorText: '#ffffff',
  colorTextDescription: '#8b92b3',
  colorTextSecondary: '#aab2d5',
  colorTextTertiary: '#70789f',
  colorWarning: '#f7931a',
  fontFamily:
    '"DM Sans", Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
}
```

- [ ] **Step 4: Verify type compatibility**

Run:

```powershell
cd frontend
npm run typecheck
```

Expected: exit code 0.

## Task 2: Global Fintech Visual System

**Files:**
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Replace root design tokens**

Define the approved variables:

```css
:root {
  --brand-bg: #0a0e27;
  --brand-bg-deep: #070b22;
  --brand-shell: rgba(7, 11, 34, 0.88);
  --brand-glass: rgba(15, 22, 53, 0.82);
  --brand-glass-strong: rgba(16, 24, 58, 0.94);
  --brand-card: rgba(15, 22, 53, 0.82);
  --brand-card-hover: rgba(18, 28, 67, 0.92);
  --brand-border: rgba(30, 38, 80, 0.72);
  --brand-border-soft: rgba(30, 38, 80, 0.46);
  --brand-blue: #0080ff;
  --brand-blue-soft: #66b5ff;
  --brand-green: #39ff14;
  --brand-green-soft: #7dff6a;
  --brand-red: #ef4444;
  --brand-amber: #f7931a;
  --brand-ink: #ffffff;
  --brand-muted: #8b92b3;
  --brand-muted-soft: #70789f;
  --brand-terminal: #070b22;
  --brand-code: #dbeafe;
  --brand-shadow: rgba(0, 0, 0, 0.38);
}
```

- [ ] **Step 2: Add reference grid background**

Update `body`, `.app-shell`, and `.login-page` to use:

```css
background:
  linear-gradient(rgba(30, 38, 80, 0.28) 1px, transparent 1px),
  linear-gradient(90deg, rgba(30, 38, 80, 0.28) 1px, transparent 1px),
  radial-gradient(circle at 78% 18%, rgba(57, 255, 20, 0.12), transparent 28%),
  radial-gradient(circle at 18% 8%, rgba(0, 128, 255, 0.16), transparent 26%),
  var(--brand-bg);
background-size:
  32px 32px,
  32px 32px,
  auto,
  auto,
  auto;
```

- [ ] **Step 3: Restyle global Ant Design controls**

Update CSS overrides for `.ant-btn`, `.ant-btn-primary`, `.ant-input`, `.ant-select-selector`, `.ant-table`, `.ant-modal-content`, `.ant-drawer-content`, `.ant-tabs`, `.ant-tag`, `.ant-alert` so:

```css
.ant-btn-primary {
  background: var(--brand-green);
  color: #03110a;
}

.ant-btn-dangerous,
.ant-btn-dangerous:hover,
.ant-btn-dangerous:focus {
  color: #fecaca;
}

.ant-table-tbody > tr:hover > td {
  background: rgba(0, 128, 255, 0.1) !important;
}
```

- [ ] **Step 4: Preserve accessibility utilities**

Keep and verify these behaviors remain in `app.css`:

```css
:where(button, a, input, textarea, select, .ant-btn, .ant-select-selector, .ant-input):focus-visible {
  outline: 2px solid var(--brand-green);
  outline-offset: 2px;
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}
```

## Task 3: Shared Shell And Components

**Files:**
- Modify: `frontend/src/layouts/MainLayout.vue`
- Modify: `frontend/src/components/PageHeader.vue`
- Modify: `frontend/src/components/StatusTile.vue`
- Modify: `frontend/src/components/AppStatusBar.vue`
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Add route-aware shell class**

In `MainLayout.vue`, add the current route name as a class hook:

```vue
<ALayout class="app-shell" :class="`route-${String(route.name || 'dashboard')}`">
```

- [ ] **Step 2: Extend `PageHeader` without breaking callers**

Use this template shape:

```vue
<header class="page-header">
  <div class="page-header-copy">
    <p v-if="eyebrow" class="page-eyebrow">{{ eyebrow }}</p>
    <h1>{{ title }}</h1>
    <p v-if="description" class="page-description">{{ description }}</p>
  </div>
  <div class="page-header-actions">
    <slot />
  </div>
</header>
```

Add prop:

```ts
description?: string;
```

- [ ] **Step 3: Add semantic tone to `StatusTile`**

Use this prop shape:

```ts
withDefaults(
  defineProps<{
    hint: string;
    label: string;
    tone?: 'danger' | 'info' | 'neutral' | 'success' | 'warning';
    value: string;
  }>(),
  { tone: 'info' },
);
```

Bind class:

```vue
<ACard class="status-tile" :class="`status-tile-${tone}`" :bordered="false">
```

- [ ] **Step 4: Restyle status bar**

Keep existing store logic, but render status tags with stable classes:

```vue
<ATag class="status-tag status-tag-runtime" :color="xrayStatusColor">{{ xrayStatusLabel }}</ATag>
<ATag class="status-tag status-tag-phase">Phase 10</ATag>
```

- [ ] **Step 5: Run lint after component API changes**

Run:

```powershell
cd frontend
npm run lint
```

Expected: exit code 0.

## Task 4: Page-Level Class Hooks And Polish

**Files:**
- Modify: `frontend/src/views/LoginView.vue`
- Modify: `frontend/src/views/DashboardView.vue`
- Modify: `frontend/src/views/CoreInstancesView.vue`
- Modify: `frontend/src/views/InboundsView.vue`
- Modify: `frontend/src/views/XrayView.vue`
- Modify: `frontend/src/views/LogsView.vue`
- Modify: `frontend/src/views/SettingsView.vue`
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Add page descriptions**

Use `PageHeader` descriptions:

```vue
<PageHeader eyebrow="Overview" title="Dashboard" description="Live Xray health, traffic, clients, and geo maintenance.">
```

Apply equivalent concise descriptions to Core Instances, Inbounds, Xray, Logs, and Settings.

- [ ] **Step 2: Add page root classes**

Use a second class on each root section:

```vue
<section class="page-stack dashboard-page">
```

Equivalent classes:

```text
core-page
inbounds-page
xray-page
logs-page
settings-page
```

- [ ] **Step 3: Use `StatusTile` tones where semantic state is obvious**

Examples:

```vue
<StatusTile label="Xray State" :value="xrayStateLabel" :hint="xrayVersionLabel" tone="success" />
<StatusTile label="CPU" :value="formatPercent(status?.cpu)" :hint="cpuHint" tone="info" />
```

Use `warning` only for stopped/experimental/static risk indicators and `danger` only for error paths.

- [ ] **Step 4: Restyle page-specific panels**

Add CSS for:

```css
.section-toolbar,
.geo-panel,
.log-viewport,
.code-preview,
.json-editor,
.settings-feature-panel,
.drawer-summary,
.toolbar-grid {
  border-color: var(--brand-border-soft);
}
```

Ensure `.toolbar-grid` becomes one column at mobile width and does not force page-level horizontal scroll.

- [ ] **Step 5: Run format check**

Run:

```powershell
cd frontend
npm run format
```

Expected: exit code 0. If it fails only due formatting, run `npm run format:write`, then rerun `npm run format`.

## Task 5: Build, Embedded Assets, Browser Verification

**Files:**
- Modify/generated: `web/ui/**`

- [ ] **Step 1: Build frontend and refresh embedded UI**

Run:

```powershell
cd frontend
npm run build
```

Expected: exit code 0 and `web/ui/` assets regenerated.

- [ ] **Step 2: Run backend compatibility checks**

Run from repo root:

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

Expected: exit code 0 for each command.

- [ ] **Step 3: Launch or reuse a local preview**

For Vite-only visual verification:

```powershell
cd frontend
npm run dev -- --host 127.0.0.1
```

Expected: Vite prints a localhost URL. Use Browser/Playwright to inspect the login or app shell. If the panel requires runtime config, verify static render and build output instead of authenticated flows.

- [ ] **Step 4: Capture responsive screenshots**

Check viewports:

```text
375x812
768x1024
1024x768
1440x900
```

Expected:

- No page-level horizontal scroll on mobile.
- No overlapping text or clipped buttons.
- Primary actions are green, dangerous actions remain red.
- Console has no CSP, font, or CSS loading errors.

- [ ] **Step 5: Record E2E status**

Run if local panel service is available:

```powershell
npm run e2e
```

Expected: pass when `SUPERXRAY_E2E_BASE_URL` points to a running isolated panel. If it fails with `ERR_CONNECTION_REFUSED`, record it as environment blockage.

## Self-Review Checklist

- The plan covers the approved spec: visual system, self-hosted fonts, shared components, core pages, responsive checks, build output, CSP safety, rollback boundaries.
- No task changes old UI templates or backend data behavior.
- Component prop additions are backward-compatible.
- Verification includes frontend, backend, browser, and E2E status.
- Current dirty worktree is acknowledged; no commit step is included.
