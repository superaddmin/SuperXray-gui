# Xray Compat Protocol Combos Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an Xray-compatible protocol tools page, Argo command/config generator, external-only TUIC/AnyTLS evaluation output, and a selectable WARP outbound matrix.

**Architecture:** Add a focused browser-side helper module for deterministic generation and tests. Wire the helper into the existing Xray settings page and WARP modal without changing backend storage or unsupported Xray protocols. Keep generated TUIC/AnyTLS output external-only so Xray template validation remains safe.

**Tech Stack:** Go templates, Vue.js, Ant Design Vue, browser JavaScript, Node.js built-in test runner, existing Xray template JSON model.

---

## File Structure

- Create `web/assets/js/model/protocol_tools.js`: pure generation helpers for Argo, protocol presets, and WARP matrix operations.
- Create `web/assets/js/model/protocol_tools.test.js`: Node tests for all helpers.
- Create `web/html/settings/xray/protocol_tools.html`: Xray settings tab content for command/config generation.
- Modify `web/html/xray.html`: load `protocol_tools.js`, add the new settings tab, and initialize `protocolTool` state/methods.
- Modify `web/html/modals/warp_modal.html`: expose selectable WARP matrix items and apply helper-generated outbounds/rules.
- Modify `docs/modules.md`: mention the new Protocol Tools UI and WARP matrix behavior.

## Task 1: Generator Tests First

**Files:**
- Create: `web/assets/js/model/protocol_tools.test.js`

- [ ] **Step 1: Write tests for Argo generation, protocol presets, and WARP matrix**

Use this test shape:

```javascript
const assert = require('node:assert/strict');
const test = require('node:test');
const {
  ProtocolToolGenerator,
  WarpMatrixBuilder,
} = require('./protocol_tools.js');

test('generates quick tunnel command', () => {
  const result = ProtocolToolGenerator.generateArgo({
    mode: 'quick',
    originUrl: 'http://localhost:2053',
  });
  assert.equal(result.mode, 'quick');
  assert.match(result.command, /cloudflared tunnel --url http:\/\/localhost:2053/);
  assert.equal(result.externalProxy.dest, '<trycloudflare-host>');
});

test('marks tuic and anytls as external only', () => {
  for (const combo of ['tuic-singbox', 'anytls-singbox']) {
    const result = ProtocolToolGenerator.generateCombo({
      combo,
      server: 'example.com',
      port: 443,
      uuid: '11111111-1111-4111-8111-111111111111',
      password: 'secret',
      sni: 'example.com',
    });
    assert.equal(result.saveToXray, false);
    assert.equal(result.runtime, 'sing-box');
    assert.match(result.notice, /not supported as an Xray inbound/);
  }
});

test('generates xray vless reality vision combo', () => {
  const result = ProtocolToolGenerator.generateCombo({
    combo: 'vless-reality-vision',
    server: 'example.com',
    port: 443,
    uuid: '11111111-1111-4111-8111-111111111111',
    publicKey: 'pub',
    shortId: 'abcd',
    sni: 'www.microsoft.com',
  });
  assert.equal(result.saveToXray, true);
  const outbound = JSON.parse(result.clientOutbound);
  assert.equal(outbound.protocol, 'vless');
  assert.equal(outbound.streamSettings.security, 'reality');
  assert.equal(outbound.settings.vnext[0].users[0].flow, 'xtls-rprx-vision');
});

test('builds warp matrix and preserves non-warp config', () => {
  const base = {
    mtu: 1420,
    secretKey: 'private',
    address: ['172.16.0.2/32', '2606:4700:110:abcd::1/128'],
    reserved: [1, 2, 3],
    peers: [{ publicKey: 'peer', endpoint: '162.159.192.1:2408' }],
    noKernelTun: false,
  };
  const existing = {
    outbounds: [{ tag: 'direct', protocol: 'freedom' }, { tag: 'warp-old', protocol: 'wireguard' }],
    routing: { rules: [{ outboundTag: 'direct', domain: ['geosite:cn'] }, { outboundTag: 'warp-old', domain: ['geosite:openai'] }] },
  };
  const result = WarpMatrixBuilder.applyMatrix(existing, base, ['warp', 'warp-ipv4', 'warp-openai']);
  assert.deepEqual(result.outbounds.map(o => o.tag), ['direct', 'warp', 'warp-ipv4', 'warp-openai']);
  assert.equal(result.routing.rules.length, 2);
  assert.equal(result.routing.rules[1].outboundTag, 'warp-openai');
});
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
node --test web/assets/js/model/protocol_tools.test.js
```

Expected: failure because `protocol_tools.js` does not exist.

## Task 2: Pure Generator Module

**Files:**
- Create: `web/assets/js/model/protocol_tools.js`

- [ ] **Step 1: Implement `ProtocolToolGenerator` and `WarpMatrixBuilder`**

Implement these exported globals and CommonJS exports:

```javascript
const ProtocolToolGenerator = {
  generateArgo(input) {},
  generateCombo(input) {},
  comboPresets: [],
};

const WarpMatrixBuilder = {
  matrixOptions: [],
  buildOutbounds(baseSettings, selectedTags) {},
  buildRules(selectedTags) {},
  applyMatrix(templateSettings, baseSettings, selectedTags) {},
};

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { ProtocolToolGenerator, WarpMatrixBuilder };
}
if (typeof window !== 'undefined') {
  window.ProtocolToolGenerator = ProtocolToolGenerator;
  window.WarpMatrixBuilder = WarpMatrixBuilder;
}
```

Generation rules:

- `generateArgo({mode:'quick', originUrl})` returns command `cloudflared tunnel --url <originUrl>`.
- `generateArgo({mode:'fixed', originUrl, token, tunnelName})` returns command `cloudflared tunnel run --token <token>`, a systemd unit, a Compose service, and `externalProxy`.
- `generateCombo({combo:'vless-reality-vision', ...})` returns Xray outbound JSON with `protocol: "vless"`, `security: "reality"`, and flow `xtls-rprx-vision`.
- `generateCombo({combo:'vless-xhttp-reality', ...})` returns Xray outbound JSON with `network: "xhttp"`, `security: "reality"`, and flow `xtls-rprx-vision`.
- `generateCombo({combo:'tuic-singbox'|'anytls-singbox', ...})` returns `saveToXray: false`, `runtime: "sing-box"`, and a sing-box JSON snippet.
- `WarpMatrixBuilder.applyMatrix` removes only outbound tags exactly `warp` or beginning with `warp-`, removes routing rules with those outbound tags, then appends selected WARP outbounds/rules.

- [ ] **Step 2: Run generator tests**

Run:

```powershell
node --test web/assets/js/model/protocol_tools.test.js
```

Expected: all tests pass.

## Task 3: Protocol Tools Page

**Files:**
- Create: `web/html/settings/xray/protocol_tools.html`
- Modify: `web/html/xray.html`

- [ ] **Step 1: Add the template**

Create a tab body with three sections:

- Argo Tunnel generator form.
- Protocol combo selector and output.
- Support matrix table.

Vue bindings use:

```html
<a-select v-model="protocolTool.combo">
<a-textarea v-model="protocolTool.generated" :auto-size="{ minRows: 12, maxRows: 24 }"></a-textarea>
<a-button type="primary" icon="code" @click="generateProtocolToolOutput">Generate</a-button>
```

- [ ] **Step 2: Load helper script and add Xray tab**

In `web/html/xray.html`, add:

```html
<script src="{{ .base_path }}assets/js/model/protocol_tools.js?{{ .cur_ver }}"></script>
```

Add a tab pane:

```html
<a-tab-pane key="tpl-protocol-tools" force-render="true">
  <template #tab>
    <a-icon type="tool"></a-icon>
    <span>Protocol Tools</span>
  </template>
  {{ template "settings/xray/protocol_tools" . }}
</a-tab-pane>
```

Add Vue state and method:

```javascript
protocolTool: {
  mode: 'combo',
  argoMode: 'quick',
  originUrl: 'http://localhost:2053',
  tunnelToken: '',
  tunnelName: 'superxray',
  combo: 'vless-reality-vision',
  server: 'example.com',
  port: 443,
  uuid: '11111111-1111-4111-8111-111111111111',
  password: 'change-me',
  sni: 'www.microsoft.com',
  publicKey: '',
  shortId: '',
  generated: '',
},
generateProtocolToolOutput() {
  const result = this.protocolTool.mode === 'argo'
    ? ProtocolToolGenerator.generateArgo({
        mode: this.protocolTool.argoMode,
        originUrl: this.protocolTool.originUrl,
        token: this.protocolTool.tunnelToken,
        tunnelName: this.protocolTool.tunnelName,
      })
    : ProtocolToolGenerator.generateCombo(this.protocolTool);
  this.protocolTool.generated = JSON.stringify(result, null, 2);
}
```

- [ ] **Step 3: Verify existing frontend tests still pass**

Run:

```powershell
node --test web/assets/js/model/inbound.test.js web/assets/js/model/inbound_form_help.test.js web/assets/js/model/protocol_tools.test.js
```

Expected: all tests pass.

## Task 4: WARP Matrix UI

**Files:**
- Modify: `web/html/modals/warp_modal.html`

- [ ] **Step 1: Replace single-outbound-only controls with matrix controls**

Keep existing `addOutbound`, `resetOutbound`, and `delOutbound` for backwards compatibility. Add:

```html
<a-checkbox-group v-model="warpMatrix.selected" :options="warpMatrix.options"></a-checkbox-group>
<a-button type="primary" icon="cluster" @click="applyMatrix" :loading="warpModal.confirmLoading">Apply Matrix</a-button>
```

- [ ] **Step 2: Use helper-generated matrix**

Add modal data:

```javascript
warpMatrix: {
  selected: ['warp'],
  options: WarpMatrixBuilder.matrixOptions.map(option => ({ label: option.label, value: option.tag })),
},
```

Add method:

```javascript
applyMatrix() {
  const next = WarpMatrixBuilder.applyMatrix(
    app.templateSettings,
    warpModal.warpOutbound.settings,
    this.warpMatrix.selected
  );
  app.templateSettings.outbounds = next.outbounds;
  app.templateSettings.routing = next.routing;
  app.outboundSettings = JSON.stringify(app.templateSettings.outbounds);
  app.routingRuleSettings = JSON.stringify(app.templateSettings.routing.rules || []);
  warpModal.close();
}
```

- [ ] **Step 3: Run JS tests**

Run:

```powershell
node --test web/assets/js/model/protocol_tools.test.js
```

Expected: all tests pass.

## Task 5: Documentation and Full Verification

**Files:**
- Modify: `docs/modules.md`

- [ ] **Step 1: Document the new page**

Add a short section under Xray settings modules:

```markdown
### Protocol Tools

Xray 设置页提供 Protocol Tools 标签页，用于生成 Argo Tunnel 命令、Xray 兼容协议组合配置和 sing-box 外部协议配置。TUIC 与 AnyTLS 标记为 external-only，不会写入 Xray 入站协议。
```

- [ ] **Step 2: Run verification**

Run:

```powershell
node --test web/assets/js/model/inbound.test.js web/assets/js/model/inbound_form_help.test.js web/assets/js/model/protocol_tools.test.js
$env:PATH='C:\msys64\mingw64\bin;' + $env:PATH
go test ./...
git diff --check
```

Expected:

- Node tests pass.
- Go tests pass.
- `git diff --check` has no whitespace errors.

- [ ] **Step 3: Commit**

Run:

```powershell
git add docs/superpowers/specs/2026-05-01-xray-compat-protocol-combos-design.md docs/superpowers/plans/2026-05-01-xray-compat-protocol-combos.md web/assets/js/model/protocol_tools.js web/assets/js/model/protocol_tools.test.js web/html/settings/xray/protocol_tools.html web/html/xray.html web/html/modals/warp_modal.html docs/modules.md
git commit -m "feat: 增加 Xray 协议组合工具"
```

## Self-Review

- Spec coverage: Argo generation is covered by Task 2 and Task 3; protocol support evaluation is covered by Task 2 and Task 3; WARP matrix is covered by Task 2 and Task 4; docs and verification are covered by Task 5.
- Placeholder scan: no unresolved placeholder wording is required for implementation. Example values like `example.com` and `change-me` are deliberate UI defaults and test fixtures.
- Type consistency: helper names are consistently `ProtocolToolGenerator` and `WarpMatrixBuilder`; UI state is consistently `protocolTool`; WARP selected matrix tags match the helper output tags.
