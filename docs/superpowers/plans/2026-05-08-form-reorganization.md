# SuperXray Form Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重组 Inbounds 与 Settings 表单分区，统一表单容器、动作区和移动端行为，同时保留现有业务函数、字段绑定、JSON 同步和提交路径。

**Architecture:** 新增展示型 `FormSection.vue` 作为统一 section 外壳，Inbounds 与 Settings 只调整模板结构和 class hooks。`app.css` 提供公共表单布局、响应式弹窗、动作换行和 JSON 区样式，所有 API、payload、校验和提交函数保持现状。

**Tech Stack:** Vue 3、TypeScript、Ant Design Vue 4、Node `node:test` 源码结构测试、Vite、PowerShell。

---

## 执行约束

- 当前工作区已有未提交的 `frontend/package.json`、`frontend/src/views/InboundsView.vue`、`frontend/tests/inbounds-view.test.ts` 和 `web/ui/` 构建产物改动；本计划在当前工作区继续，不新建 worktree。
- 只修改计划列出的 `frontend/` 源码、测试、样式和本计划文件，不回滚不相关改动。
- 不修改后端 API、数据库模型、旧 UI、`model.Inbound`、CoreManager 或 sing-box 路径。
- 不重写 `submitInbound()`、`saveSettings()`、`confirmSave()`、`resetForm()`、`applySubscriptionDefaults()`、`confirmUpdateCredentials()`、`handleDbFileChange()`、`confirmRestartPanel()`。
- 每个生产代码改动前必须先写失败测试并观察失败。

## File Structure

- Create `frontend/src/components/FormSection.vue`: 展示型表单分区组件，提供 eyebrow/title/description props 与 actions/default slots。
- Modify `frontend/src/views/InboundsView.vue`: 使用 `FormSection` 重组入站编辑弹窗 Basic、WireGuard、Transport、Default Client、Advanced JSON 分区；保留现有 `v-model` 和函数调用。
- Modify `frontend/src/views/SettingsView.vue`: 使用 `FormSection` 重组 Panel、Security、Subscription、Formats、Telegram、LDAP、Backup tab 内部分区；保留所有字段和动作函数。
- Modify `frontend/src/styles/app.css`: 增加 `.form-section`、`.form-section-header`、`.form-actions`、`.form-json-stack`、`.responsive-modal-form`、三列/单列响应式规则，并保留旧类兼容。
- Modify `frontend/tests/inbounds-view.test.ts`: 扩展源码结构测试，锁定分区、默认客户端、JSON sync/apply 和提交路径。
- Create `frontend/tests/settings-view.test.ts`: 锁定 Settings 全字段、关键动作函数和分区结构。
- Create `frontend/tests/form-section.test.ts`: 锁定 `FormSection.vue` 为展示型组件，不引用业务 API。
- Create `frontend/tests/form-styles.test.ts`: 锁定公共表单类和移动端响应式规则。

## Task 1: RED tests for form reorganization

**Files:**
- Modify: `frontend/tests/inbounds-view.test.ts`
- Create: `frontend/tests/settings-view.test.ts`
- Create: `frontend/tests/form-section.test.ts`
- Create: `frontend/tests/form-styles.test.ts`

- [ ] **Step 1: Extend Inbounds source tests**

Add tests that require `FormSection`, `responsive-modal-form`, semantic form sections, Default Client sync/apply, Advanced JSON and unchanged submit sync:

```ts
test('inbound modal is organized into reusable form sections', () => {
  assert.match(source, /import FormSection from '@\/components\/FormSection\.vue';/);
  assert.match(source, /class="responsive-modal-form"/);
  assert.match(source, /<FormSection\s+eyebrow="Inbound"\s+title="Basic Inbound"/);
  assert.match(source, /<FormSection\s+v-if="inboundEditor\.protocol === 'wireguard'"\s+eyebrow="Protocol"\s+title="WireGuard Settings"/);
  assert.match(source, /<FormSection\s+v-if="protocolSupportsStream\(inboundEditor\.protocol\)"\s+eyebrow="Transport"\s+title="Transport Settings"/);
  assert.match(source, /<FormSection\s+v-if="inboundClientSectionVisible"\s+eyebrow="Client"\s+title="Default Client"/);
  assert.match(source, /<FormSection\s+eyebrow="Advanced"\s+title="Advanced JSON"/);
});

test('inbound form keeps default client and JSON action paths', () => {
  assert.match(source, /@click="syncInboundClientEditorFromSettings"/);
  assert.match(source, /@click="applyInboundClientEditorToSettings"/);
  assert.match(source, /@click="formatInboundJson\('settings'\)"/);
  assert.match(source, /@click="formatInboundJson\('streamSettings'\)"/);
  assert.match(source, /@click="formatInboundJson\('sniffing'\)"/);
  assert.match(source, /v-model="inboundEditor\.settings"/);
  assert.match(source, /v-model="inboundEditor\.streamSettings"/);
  assert.match(source, /v-model="inboundEditor\.sniffing"/);
});
```

- [ ] **Step 2: Add Settings source tests**

Create `frontend/tests/settings-view.test.ts`:

```ts
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync('frontend/src/views/SettingsView.vue', 'utf8');

test('settings view uses form sections for each settings workflow', () => {
  assert.match(source, /import FormSection from '@\/components\/FormSection\.vue';/);
  for (const title of [
    'Web Endpoint',
    'TLS Files',
    'Session and Display',
    'Thresholds and Naming',
    'Two Factor',
    'Credentials',
    'Feature Flags',
    'Server Endpoint',
    'Public URIs',
    'Metadata and TLS',
    'External Traffic',
    'Public Links',
    'Recommended Client Links',
    'Announce and Routing',
    'JSON Formats',
    'Bot Flags',
    'Bot Connection',
    'Runtime Rules',
    'LDAP Flags',
    'Connection',
    'User Mapping',
    'Sync Defaults',
    'Database Backup / Restore',
    'Panel Runtime',
  ]) {
    assert.match(source, new RegExp(`title="${title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}"`));
  }
});

test('settings view keeps all critical action flows', () => {
  for (const action of [
    '@click="loadSettings"',
    '@click="resetForm"',
    '@click="confirmSave"',
    '@click="generateTwoFactorToken"',
    '@click="disableTwoFactor"',
    '@click="confirmUpdateCredentials"',
    '@click="applySubscriptionDefaults"',
    '@click="copySubscriptionLinks"',
    '@click="copyRecommendedLinks"',
    '@click="downloadDb"',
    '@click="openDbFilePicker"',
    '@change="handleDbFileChange"',
    '@click="confirmRestartPanel"',
  ]) {
    assert.match(source, new RegExp(action.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
});

test('settings view keeps representative field bindings from every tab', () => {
  for (const binding of [
    'settings.webListen',
    'settings.webDomain',
    'settings.webPort',
    'settings.webBasePath',
    'settings.webCertFile',
    'settings.webKeyFile',
    'settings.sessionMaxAge',
    'settings.pageSize',
    'settings.expireDiff',
    'settings.trafficDiff',
    'settings.remarkModel',
    'settings.datepicker',
    'settings.timeLocation',
    'settings.twoFactorEnable',
    'settings.twoFactorToken',
    'credentials.oldUsername',
    'credentials.oldPassword',
    'credentials.newUsername',
    'credentials.newPassword',
    'settings.subEnable',
    'settings.subJsonEnable',
    'settings.subClashEnable',
    'settings.subEncrypt',
    'settings.subShowInfo',
    'settings.subEnableRouting',
    'settings.subTitle',
    'settings.subUpdates',
    'settings.subListen',
    'settings.subPort',
    'settings.subDomain',
    'settings.subPath',
    'settings.subURI',
    'settings.subJsonPath',
    'settings.subJsonURI',
    'settings.subClashPath',
    'settings.subClashURI',
    'settings.subSupportUrl',
    'settings.subProfileUrl',
    'settings.subCertFile',
    'settings.subKeyFile',
    'settings.externalTrafficInformEnable',
    'settings.externalTrafficInformURI',
    'settings.subAnnounce',
    'settings.subRoutingRules',
    'settings.subJsonFragment',
    'settings.subJsonNoises',
    'settings.subJsonMux',
    'settings.subJsonRules',
    'settings.tgBotEnable',
    'settings.tgBotBackup',
    'settings.tgBotLoginNotify',
    'settings.tgBotToken',
    'settings.tgBotChatId',
    'settings.tgBotProxy',
    'settings.tgBotAPIServer',
    'settings.tgRunTime',
    'settings.tgCpu',
    'settings.tgLang',
    'settings.ldapEnable',
    'settings.ldapUseTLS',
    'settings.ldapInvertFlag',
    'settings.ldapAutoCreate',
    'settings.ldapAutoDelete',
    'settings.ldapHost',
    'settings.ldapPort',
    'settings.ldapBindDN',
    'settings.ldapPassword',
    'settings.ldapBaseDN',
    'settings.ldapUserFilter',
    'settings.ldapUserAttr',
    'settings.ldapVlessField',
    'settings.ldapSyncCron',
    'settings.ldapFlagField',
    'settings.ldapTruthyValues',
    'settings.ldapInboundTags',
    'settings.ldapDefaultTotalGB',
    'settings.ldapDefaultExpiryDays',
    'settings.ldapDefaultLimitIP',
  ]) {
    assert.match(source, new RegExp(binding.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
});
```

- [ ] **Step 3: Add FormSection component tests**

Create `frontend/tests/form-section.test.ts`:

```ts
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync('frontend/src/components/FormSection.vue', 'utf8');

test('form section is presentational and slot based', () => {
  assert.match(source, /defineProps<\{\s*eyebrow\?: string;\s*title: string;\s*description\?: string;\s*\}>/);
  assert.match(source, /<slot name="actions" \/>/);
  assert.match(source, /<slot \/>/);
  assert.match(source, /class="form-section"/);
  assert.doesNotMatch(source, /from '@\/api\//);
  assert.doesNotMatch(source, /submitInbound|saveSettings|restartPanel|importDatabase/);
});
```

- [ ] **Step 4: Add CSS source tests**

Create `frontend/tests/form-styles.test.ts`:

```ts
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const css = readFileSync('frontend/src/styles/app.css', 'utf8');

test('form styles define shared section and responsive modal primitives', () => {
  for (const selector of [
    '.form-section',
    '.form-section-header',
    '.form-section-title',
    '.form-section-description',
    '.form-actions',
    '.form-json-stack',
    '.responsive-modal-form',
    '.form-grid--three',
  ]) {
    assert.match(css, new RegExp(selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
});

test('form styles collapse grids and actions on mobile', () => {
  assert.match(css, /@media \(max-width: 768px\)[\s\S]*\.responsive-modal-form/);
  assert.match(css, /@media \(max-width: 768px\)[\s\S]*\.form-section-header/);
  assert.match(css, /@media \(max-width: 768px\)[\s\S]*\.form-actions/);
  assert.match(css, /@media \(max-width: 768px\)[\s\S]*\.form-grid--three/);
});
```

- [ ] **Step 5: Run RED tests**

Run:

```powershell
cd frontend
npm run test -- frontend/tests/inbounds-view.test.ts frontend/tests/settings-view.test.ts frontend/tests/form-section.test.ts frontend/tests/form-styles.test.ts
```

Expected: FAIL because `FormSection.vue`, Settings sections, Inbounds section wrappers and new CSS selectors are not implemented yet.

## Task 2: Implement FormSection and shared CSS primitives

**Files:**
- Create: `frontend/src/components/FormSection.vue`
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Create the presentational FormSection component**

Add:

```vue
<template>
  <section class="form-section">
    <div class="form-section-header">
      <div class="form-section-heading">
        <p v-if="eyebrow" class="page-eyebrow">{{ eyebrow }}</p>
        <h3 class="form-section-title">{{ title }}</h3>
        <p v-if="description" class="form-section-description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="form-actions">
        <slot name="actions" />
      </div>
    </div>
    <slot />
  </section>
</template>

<script setup lang="ts">
defineProps<{
  eyebrow?: string;
  title: string;
  description?: string;
}>();
</script>
```

- [ ] **Step 2: Add shared form CSS**

Add selectors to `frontend/src/styles/app.css` near the existing `.form-grid` / `.settings-feature-panel` rules:

```css
.form-section {
  margin: 16px 0;
  padding: 18px;
  border: 1px solid var(--brand-border-soft);
  border-radius: 16px;
  background: rgba(7, 11, 34, 0.44);
}

.form-section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.form-section-heading {
  min-width: 0;
}

.form-section-title {
  margin: 0;
  color: var(--brand-ink);
  font-family: var(--font-heading);
  font-size: 18px;
  line-height: 1.25;
}

.form-section-description {
  max-width: 780px;
  margin: 6px 0 0;
  color: var(--brand-muted);
  font-size: 13px;
  line-height: 1.55;
}

.form-actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.form-grid--three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.form-json-stack {
  display: grid;
  gap: 14px;
}

.responsive-modal-form {
  max-width: 100%;
}
```

Add mobile rules in the existing `@media (max-width: 768px)` block:

```css
.responsive-modal-form {
  margin: 0 -4px;
}

.form-section {
  padding: 14px;
  border-radius: 12px;
}

.form-section-header {
  flex-direction: column;
  gap: 10px;
}

.form-actions {
  width: 100%;
  justify-content: flex-start;
}

.form-actions .ant-btn {
  min-height: 44px;
}

.form-grid--three {
  grid-template-columns: 1fr;
}
```

- [ ] **Step 3: Run component and style tests**

Run:

```powershell
cd frontend
npm run test -- frontend/tests/form-section.test.ts frontend/tests/form-styles.test.ts
```

Expected: PASS.

## Task 3: Reorganize Inbounds modal without changing business logic

**Files:**
- Modify: `frontend/src/views/InboundsView.vue`

- [ ] **Step 1: Import FormSection**

Add near component imports:

```ts
import FormSection from '@/components/FormSection.vue';
```

- [ ] **Step 2: Add responsive modal class**

Change the inbound modal form opening tag to:

```vue
<AForm class="responsive-modal-form" layout="vertical">
```

- [ ] **Step 3: Wrap Basic Inbound fields**

Replace the setup alert plus first `.form-grid` block with:

```vue
<AAlert
  class="mb-12"
  message="Inbound setup guide"
  description="Start with protocol, remark, listen address, and port. Keep 0.0.0.0 to accept connections on all interfaces; set traffic and expiry to 0 for no limit. Use the transport form for common TCP/WS/gRPC/TLS/Reality options, then click Sync JSON only when you need to inspect or manually adjust the raw legacy JSON."
  show-icon
  type="info"
/>

<FormSection
  eyebrow="Inbound"
  title="Basic Inbound"
  description="Protocol, listening address, limits, and enable state are saved through the existing inbound submit path."
>
  <div class="form-grid">
    <!-- move the existing Protocol, Remark, Listen, Port, Traffic Limit GB, Expiry Timestamp, Traffic Reset, Enable AFormItem nodes here unchanged -->
  </div>
</FormSection>
```

- [ ] **Step 4: Wrap WireGuard Settings**

Change the WireGuard section wrapper to:

```vue
<FormSection
  v-if="inboundEditor.protocol === 'wireguard'"
  eyebrow="Protocol"
  title="WireGuard Settings"
  description="Server keys and WireGuard-specific settings stay synchronized with the legacy settings JSON."
>
  <template #actions>
    <AButton size="small" @click="syncWireguardEditorFromSettings">Sync JSON</AButton>
    <AButton size="small" @click="applyWireguardEditorToSettings">Apply</AButton>
  </template>
  <div class="form-grid">
    <!-- move existing WireGuard AFormItem nodes unchanged -->
  </div>
</FormSection>
```

- [ ] **Step 5: Wrap Transport Settings**

Change the stream section wrapper to:

```vue
<FormSection
  v-if="protocolSupportsStream(inboundEditor.protocol)"
  eyebrow="Transport"
  title="Transport Settings"
  description="Network, security, TLS, Reality and sockopt controls continue to update the existing stream settings JSON."
>
  <template #actions>
    <AButton size="small" @click="syncStreamEditorFromSettings">Sync JSON</AButton>
    <AButton size="small" @click="applyStreamEditorToSettings">Apply</AButton>
  </template>
  <div class="form-grid">
    <!-- move existing streamEditor AFormItem nodes unchanged -->
  </div>
</FormSection>
```

- [ ] **Step 6: Wrap Default Client**

Change the default client wrapper to:

```vue
<FormSection
  v-if="inboundClientSectionVisible"
  eyebrow="Client"
  title="Default Client"
  description="Create the first client for protocols that require one. Apply keeps the form and raw settings JSON in sync."
>
  <template #actions>
    <AButton size="small" @click="syncInboundClientEditorFromSettings">Sync JSON</AButton>
    <AButton size="small" @click="applyInboundClientEditorToSettings">Apply</AButton>
  </template>
  <div class="form-grid client-form-grid">
    <!-- move existing default client AFormItem nodes unchanged -->
  </div>
  <AFormItem label="Comment">
    <AInput v-model:value="inboundClientEditor.comment" />
  </AFormItem>
</FormSection>
```

- [ ] **Step 7: Wrap Advanced JSON**

Place the three JSON editors inside:

```vue
<FormSection
  eyebrow="Advanced"
  title="Advanced JSON"
  description="Raw legacy JSON remains editable for compatibility and advanced Xray options."
>
  <div class="form-json-stack">
    <!-- move Settings JSON, Stream Settings JSON, Sniffing JSON blocks unchanged -->
  </div>
</FormSection>
```

- [ ] **Step 8: Run Inbounds tests**

Run:

```powershell
cd frontend
npm run test -- frontend/tests/inbounds-view.test.ts
```

Expected: PASS.

## Task 4: Reorganize Settings tabs without changing business logic

**Files:**
- Modify: `frontend/src/views/SettingsView.vue`

- [ ] **Step 1: Import FormSection**

Add near component imports:

```ts
import FormSection from '@/components/FormSection.vue';
```

- [ ] **Step 2: Rebuild Panel tab sections**

Keep the same `AForm layout="vertical"` and move existing fields into `FormSection` wrappers titled:

```vue
<FormSection eyebrow="Panel" title="Web Endpoint">
<FormSection eyebrow="Panel" title="TLS Files">
<FormSection eyebrow="Panel" title="Session and Display">
<FormSection eyebrow="Panel" title="Thresholds and Naming">
```

- [ ] **Step 3: Rebuild Security tab sections**

Use sections titled:

```vue
<FormSection eyebrow="Security" title="Two Factor">
<FormSection eyebrow="Security" title="Credentials">
```

Move Generate Token / Disable Two Factor / Update buttons into `#actions` slots and keep the same click handlers.

- [ ] **Step 4: Rebuild Subscription tab sections**

Use sections titled:

```vue
<FormSection eyebrow="Subscription" title="Feature Flags">
<FormSection eyebrow="Subscription" title="Server Endpoint">
<FormSection eyebrow="Subscription" title="Public URIs">
<FormSection eyebrow="Subscription" title="Metadata and TLS">
<FormSection eyebrow="Subscription" title="External Traffic">
<FormSection eyebrow="Subscription" title="Public Links">
<FormSection eyebrow="Subscription" title="Recommended Client Links">
<FormSection eyebrow="Subscription" title="Announce and Routing">
```

Keep Fill Defaults in the tab header and keep all copy/open handlers.

- [ ] **Step 5: Rebuild Formats, Telegram, LDAP and Backup sections**

Use sections titled:

```vue
<FormSection eyebrow="Formats" title="JSON Formats">
<FormSection eyebrow="Telegram" title="Bot Flags">
<FormSection eyebrow="Telegram" title="Bot Connection">
<FormSection eyebrow="Telegram" title="Runtime Rules">
<FormSection eyebrow="LDAP" title="LDAP Flags">
<FormSection eyebrow="LDAP" title="Connection">
<FormSection eyebrow="LDAP" title="User Mapping">
<FormSection eyebrow="LDAP" title="Sync Defaults">
<FormSection eyebrow="Backup" title="Database Backup / Restore">
<FormSection eyebrow="Backup" title="Panel Runtime">
```

Move Download / Import / Restart buttons into actions slots and keep hidden file input with `@change="handleDbFileChange"`.

- [ ] **Step 6: Run Settings tests**

Run:

```powershell
cd frontend
npm run test -- frontend/tests/settings-view.test.ts
```

Expected: PASS.

## Task 5: Full verification and UI evidence

**Files:**
- No source edits unless verification exposes an issue.

- [ ] **Step 1: Run focused tests**

Run:

```powershell
cd frontend
npm run test -- frontend/tests/inbounds-view.test.ts frontend/tests/settings-view.test.ts frontend/tests/form-section.test.ts frontend/tests/form-styles.test.ts
```

Expected: PASS.

- [ ] **Step 2: Run frontend checks**

Run:

```powershell
cd frontend
npm run typecheck
npm run lint
npm run build
```

Expected: all exit 0.

- [ ] **Step 3: Browser verification**

Start dev server if needed:

```powershell
cd frontend
npm run dev -- --host 127.0.0.1
```

Use browser automation to capture Inbounds and Settings at 375px, 768px, and 1280px. Verify:

- Inbounds modal has Basic, Transport/WireGuard, Default Client, Advanced JSON sections.
- Settings tabs show reorganized sections and all main actions are visible.
- No horizontal overflow in form areas at 375px.
- Local screenshots are saved under `tmp/form-reorganization-20260508/`.

- [ ] **Step 4: Review diff**

Run:

```powershell
git diff -- frontend/src/components/FormSection.vue frontend/src/views/InboundsView.vue frontend/src/views/SettingsView.vue frontend/src/styles/app.css frontend/tests/inbounds-view.test.ts frontend/tests/settings-view.test.ts frontend/tests/form-section.test.ts frontend/tests/form-styles.test.ts
```

Expected: diff only contains form organization, tests, and styles described in this plan.
