# VPN Egress MVP Xray-Compatible Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a minimal, reviewable VPN egress MVP that generates Xray-compatible local proxy entries and a Gateway registration list without adding new database models, taking over CoreManager, or promoting sing-box to a production path. The generator must require an explicit network-reachability strategy before operators use the manifest in Super-Code-Gateway.

**Architecture:** The MVP stays inside the existing Xray template workflow. It generates deterministic inbound profiles, routing rules, DNS policy hints, and a CSV registration manifest for Super-Code-Gateway, then lets operators apply the generated JSON through the existing Xray template save path. Xray inbound listening uses `listenHost`; Gateway CSV registration uses `manifestHost`. The default same-network strategy is `127.0.0.1` for both, while Docker bridge deployments must set a Gateway-reachable `manifestHost`. It does not create `egress_*` tables, does not change `model.Inbound`, does not add a new scheduler, and does not route old Xray lifecycle through CoreManager.

**Tech Stack:** Vue 3, TypeScript, Ant Design Vue, existing Xray template API, Node test runner, existing Go backend only for current Xray template persistence.

**Implementation status (2026-05-16):** MVP frontend implementation is complete and now appears near the top of the Xray page, before the long template editor. The UI exposes separate `listenHost` and `manifestHost` inputs, copy/download manifest actions, and deterministic JP/US registration rows. Updated screenshots are available at `docs/assets/xray-mvp-desktop.png` and `docs/assets/xray-mvp-mobile.png`.

![Xray Gateway Egress MVP desktop](../../assets/xray-mvp-desktop.png)

---

## Phase Gate

Current project phase is UI-first Phase 9 / Phase 10 risk-accepted boundary. This MVP is allowed only because it uses the existing Xray template editor and legacy-compatible Xray JSON fields.

Approved implementation decision:
- MVP is approved only for Xray-compatible config generation and Gateway manifest export.
- The generator must expose two host concepts:
  - `listenHost`: the address written into generated Xray inbound objects.
  - `manifestHost`: the address written into Gateway CSV rows.
- Same-network defaults are `listenHost=127.0.0.1` and `manifestHost=127.0.0.1`.
- Docker bridge + host x-ui deployments must choose a reachable `manifestHost` before importing CSV rows into Gateway.
- Egress database tables and `/panel/api/egress/*` APIs remain deferred to Phase 10+ design review.
- `default-xray` remains a read-only CoreManager observation instance; this MVP must not call CoreManager lifecycle methods. The source boundary is `web/service/core_service.go:62`, where `default-xray` is defined with `LifecycleViaCoreManager: false`.
- Server-side egress governance remains deferred: `egress_groups`, `egress_nodes`, `egress_probe_results`, `egress_switch_events`, and `/panel/api/egress/*` are still Phase 10+ design-review items only.

Allowed:
- Add frontend utilities that generate Xray JSON snippets and Gateway CSV rows.
- Add source-level tests for generated Xray config and manifest output.
- Add a Xray page panel that previews and applies generated Xray template changes through the existing `updateXraySetting` path.
- Use only Xray-compatible protocols already represented in the current template model: `socks`, `http`, `freedom`, `blackhole`, `wireguard`, `vless`, `trojan`, and `socks` outbounds where already supported by Xray.

Forbidden:
- Do not add `egress_groups`, `egress_nodes`, `egress_probe_results`, or `egress_switch_events` tables in this MVP.
- Do not create `proxy_inbounds` or `proxy_clients`.
- Do not migrate `model.Inbound`.
- Do not call CoreManager for old Xray start/stop/restart.
- Do not add sing-box production config, sing-box lifecycle UI, or sing-box generated routing.
- Do not store proxy passwords or node private keys in a new frontend-visible registry.

## File Map

- Create `frontend/src/utils/gatewayEgressMvp.ts`  
  Owns deterministic MVP profile definitions, Xray JSON merge helpers, route generation, controlled listen-host validation, manifest-host rendering, and Gateway CSV rendering.

- Modify `frontend/src/views/XrayView.vue`  
  Adds a compact "Gateway Egress MVP" panel to preview generated profiles, apply generated Xray template changes, and copy/download the registration manifest. It reuses the existing `parsedConfig`, `configText`, and `confirmSave` flow.

- Modify `frontend/src/styles/app.css`  
  Adds small layout classes for manifest preview and profile cards, following existing card/table styles.

- Create `frontend/tests/gateway-egress-mvp.test.ts`  
  Locks the generation contract: separate `listenHost` and `manifestHost`, deterministic ports, platform domain rules, final reject rule, preserved existing config, and CSV manifest shape.

- Modify `frontend/tests/xray-view.test.ts`  
  Locks that XrayView exposes the MVP panel and calls the utility functions without scattering generated config inline.

- Optional documentation update after implementation: `docs/SUPERXRAY_GUI_VPN_LAYER_HANDOFF_CN.md`  
  Add links to this MVP plan and the Phase 10+ design only after the implementation direction is accepted.

## MVP Profile Contract

The first iteration ships these fixed local profiles:

| Key | Port | Platform | Region | Group | Purpose |
|---|---:|---|---|---|---|
| `openai-us-primary` | `11801` | `openai` | `US` | `openai-egress` | OpenAI API testing and Gateway registration |
| `anthropic-us-primary` | `11802` | `anthropic` | `US` | `anthropic-egress` | Anthropic API testing and Gateway registration |
| `gemini-us-primary` | `11803` | `gemini` | `US` | `gemini-egress` | Gemini API testing and Gateway registration |
| `region-us-primary` | `11901` | empty | `US` | `region-us` | Region based account routing |
| `region-jp-primary` | `11981` | empty | `JP` | `region-jp` | Region based account routing |

All generated inbounds must listen on `127.0.0.1` by default, or on a reviewed controlled host chosen for the deployment. Gateway manifest rows must use `manifestHost`, which may differ from `listenHost` when Gateway runs inside Docker bridge. Every platform inbound gets platform-domain allow rules followed by a final reject rule for unmatched traffic from generated gateway inbounds.

## Network Host Contract

The MVP generator accepts this strategy object:

```ts
export interface GatewayEgressMvpNetworkStrategy {
  listenHost: string;
  manifestHost: string;
  strategyLabel: string;
}
```

Rules:
- `listenHost` writes to Xray `inbounds[*].listen` and `settings.ip`.
- `manifestHost` writes to CSV `host`.
- Both values are trimmed before use.
- Empty host, wildcard host, and public catch-all addresses such as `0.0.0.0` are rejected.
- The same-network default is `{ listenHost: '127.0.0.1', manifestHost: '127.0.0.1', strategyLabel: 'same-network' }`.
- Docker bridge production use must select a reviewed reachable manifest host before importing the CSV into Gateway.

## Task 1: Generator Contract Tests

**Files:**
- Create: `frontend/tests/gateway-egress-mvp.test.ts`
- Create: `frontend/src/utils/gatewayEgressMvp.ts`

- [ ] **Step 1: Write the failing generator tests**

Create `frontend/tests/gateway-egress-mvp.test.ts`:

```ts
import assert from 'node:assert/strict';
import test from 'node:test';

import {
  DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY,
  GATEWAY_EGRESS_MVP_PROFILES,
  buildGatewayEgressManifestCsv,
  buildGatewayEgressMvpPreview,
  mergeGatewayEgressMvpConfig,
} from '../src/utils/gatewayEgressMvp.ts';

test('gateway egress mvp profiles are deterministic', () => {
  assert.deepEqual(
    GATEWAY_EGRESS_MVP_PROFILES.map((profile) => [profile.key, profile.port]),
    [
      ['openai-us-primary', 11801],
      ['anthropic-us-primary', 11802],
      ['gemini-us-primary', 11803],
      ['region-us-primary', 11901],
      ['region-jp-primary', 11981],
    ],
  );
});

test('gateway egress mvp separates listenHost and manifestHost', () => {
  const strategy = {
    listenHost: '172.18.0.1',
    manifestHost: 'host.docker.internal',
    strategyLabel: 'docker-host-gateway',
  };
  const merged = mergeGatewayEgressMvpConfig({}, strategy);
  const csv = buildGatewayEgressManifestCsv(strategy);
  const inbounds = merged.inbounds as Array<{ listen: string; port: number; tag: string }>;

  assert.ok(inbounds.some((inbound) => inbound.listen === '172.18.0.1'));
  assert.match(csv, /openai-us-primary,socks5h,host\.docker\.internal,11801/);
  assert.doesNotMatch(csv, /172\.18\.0\.1,11801/);
});

test('gateway egress mvp merge adds local inbounds and a final reject rule', () => {
  const base = {
    inbounds: [{ tag: 'existing-in', listen: '127.0.0.1', port: 10000, protocol: 'socks' }],
    outbounds: [{ tag: 'direct', protocol: 'freedom' }],
    routing: { domainStrategy: 'AsIs', rules: [] },
  };

  const merged = mergeGatewayEgressMvpConfig(base, DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY);
  const inbounds = merged.inbounds as Array<{ listen: string; port: number; tag: string }>;
  const rules = (merged.routing as { rules: Array<{ inboundTag?: string[]; outboundTag?: string; domain?: string[] }> }).rules;

  assert.ok(inbounds.some((inbound) => inbound.tag === 'gateway-openai-us-primary'));
  assert.ok(inbounds.every((inbound) => inbound.listen !== '0.0.0.0'));
  assert.ok(rules.some((rule) => rule.domain?.includes('domain:api.openai.com')));
  assert.ok(rules.some((rule) => rule.domain?.includes('domain:api.anthropic.com')));
  assert.ok(rules.some((rule) => rule.domain?.includes('domain:generativelanguage.googleapis.com')));
  assert.equal(rules.at(-1)?.outboundTag, 'blocked');
});

test('gateway egress mvp manifest uses socks5h registration rows', () => {
  const csv = buildGatewayEgressManifestCsv(DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY);

  assert.match(csv, /^name,protocol,host,port,platform,region_code,expected_country_code,egress_group,health_status,notes/m);
  assert.match(csv, /openai-us-primary,socks5h,127\.0\.0\.1,11801,openai,US,US,openai-egress,manual-check,OpenAI MVP local exit/);
  assert.match(csv, /region-jp-primary,socks5h,127\.0\.0\.1,11981,,JP,JP,region-jp,manual-check,JP region MVP local exit/);
  assert.doesNotMatch(csv, /0\.0\.0\.0/);
});

test('gateway egress mvp preview summarizes generated profiles', () => {
  const preview = buildGatewayEgressMvpPreview();

  assert.equal(preview.profileCount, 5);
  assert.deepEqual(preview.ports, [11801, 11802, 11803, 11901, 11981]);
  assert.deepEqual(preview.platforms, ['openai', 'anthropic', 'gemini']);
  assert.deepEqual(preview.regions, ['US', 'JP']);
});
```

- [ ] **Step 2: Run the test to confirm RED**

Run from `frontend`:

```powershell
npm run test -- frontend/tests/gateway-egress-mvp.test.ts
```

Expected: FAIL because `frontend/src/utils/gatewayEgressMvp.ts` does not exist.

- [ ] **Step 3: Add the generator utility**

Create `frontend/src/utils/gatewayEgressMvp.ts`:

```ts
import type { JsonObject, JsonValue } from '@/types/api';

export type GatewayEgressPlatform = 'openai' | 'anthropic' | 'gemini';

export interface GatewayEgressMvpProfile {
  egressGroup: string;
  expectedCountryCode: string;
  key: string;
  notes: string;
  platform: GatewayEgressPlatform | '';
  port: number;
  regionCode: 'US' | 'JP';
}

export interface GatewayEgressMvpNetworkStrategy {
  listenHost: string;
  manifestHost: string;
  strategyLabel: string;
}

export interface GatewayEgressMvpPreview {
  platforms: GatewayEgressPlatform[];
  ports: number[];
  profileCount: number;
  regions: Array<'US' | 'JP'>;
}

export const DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY: GatewayEgressMvpNetworkStrategy = {
  listenHost: '127.0.0.1',
  manifestHost: '127.0.0.1',
  strategyLabel: 'same-network',
};

export const GATEWAY_EGRESS_MVP_PROFILES: GatewayEgressMvpProfile[] = [
  {
    key: 'openai-us-primary',
    port: 11801,
    platform: 'openai',
    regionCode: 'US',
    expectedCountryCode: 'US',
    egressGroup: 'openai-egress',
    notes: 'OpenAI MVP local exit',
  },
  {
    key: 'anthropic-us-primary',
    port: 11802,
    platform: 'anthropic',
    regionCode: 'US',
    expectedCountryCode: 'US',
    egressGroup: 'anthropic-egress',
    notes: 'Anthropic MVP local exit',
  },
  {
    key: 'gemini-us-primary',
    port: 11803,
    platform: 'gemini',
    regionCode: 'US',
    expectedCountryCode: 'US',
    egressGroup: 'gemini-egress',
    notes: 'Gemini MVP local exit',
  },
  {
    key: 'region-us-primary',
    port: 11901,
    platform: '',
    regionCode: 'US',
    expectedCountryCode: 'US',
    egressGroup: 'region-us',
    notes: 'US region MVP local exit',
  },
  {
    key: 'region-jp-primary',
    port: 11981,
    platform: '',
    regionCode: 'JP',
    expectedCountryCode: 'JP',
    egressGroup: 'region-jp',
    notes: 'JP region MVP local exit',
  },
];

const PLATFORM_DOMAINS: Record<GatewayEgressPlatform, string[]> = {
  openai: ['domain:api.openai.com', 'domain:chatgpt.com', 'domain:chat.openai.com'],
  anthropic: ['domain:api.anthropic.com', 'domain:claude.ai'],
  gemini: [
    'domain:generativelanguage.googleapis.com',
    'domain:cloudcode-pa.googleapis.com',
    'domain:aiplatform.googleapis.com',
  ],
};

export function buildGatewayEgressMvpPreview(): GatewayEgressMvpPreview {
  return {
    profileCount: GATEWAY_EGRESS_MVP_PROFILES.length,
    ports: GATEWAY_EGRESS_MVP_PROFILES.map((profile) => profile.port),
    platforms: GATEWAY_EGRESS_MVP_PROFILES.filter(
      (profile): profile is GatewayEgressMvpProfile & { platform: GatewayEgressPlatform } =>
        profile.platform !== '',
    ).map((profile) => profile.platform),
    regions: ['US', 'JP'],
  };
}

export function mergeGatewayEgressMvpConfig(
  source: JsonValue | null | undefined,
  networkStrategy: GatewayEgressMvpNetworkStrategy = DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY,
): JsonObject {
  const normalizedStrategy = normalizeNetworkStrategy(networkStrategy);
  const next: JsonObject = isJsonObject(source) ? structuredClone(source) : {};
  const inbounds = Array.isArray(next.inbounds) ? [...next.inbounds] : [];
  const outbounds = Array.isArray(next.outbounds) ? [...next.outbounds] : [];
  const routing = isJsonObject(next.routing) ? { ...next.routing } : {};
  const rules = Array.isArray(routing.rules) ? [...routing.rules] : [];

  for (const profile of GATEWAY_EGRESS_MVP_PROFILES) {
    upsertByTag(inbounds, buildInbound(profile, normalizedStrategy));
    upsertByTag(outbounds, buildPlaceholderOutbound(profile));
  }

  const generatedInboundTags = GATEWAY_EGRESS_MVP_PROFILES.map((profile) => gatewayInboundTag(profile));
  const generatedRules = GATEWAY_EGRESS_MVP_PROFILES.flatMap((profile) => buildRules(profile));
  const preservedRules = rules.filter((rule) => !isGeneratedGatewayRule(rule));

  routing.domainStrategy = routing.domainStrategy || 'AsIs';
  routing.rules = [
    ...generatedRules,
    {
      type: 'field',
      inboundTag: generatedInboundTags,
      outboundTag: 'blocked',
      _gatewayEgressMvp: true,
    },
    ...preservedRules,
  ];

  if (!outbounds.some((outbound) => isJsonObject(outbound) && outbound.tag === 'blocked')) {
    outbounds.push({ tag: 'blocked', protocol: 'blackhole' });
  }

  next.inbounds = inbounds;
  next.outbounds = outbounds;
  next.routing = routing;
  return next;
}

export function buildGatewayEgressManifestCsv(
  networkStrategy: GatewayEgressMvpNetworkStrategy = DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY,
): string {
  const normalizedStrategy = normalizeNetworkStrategy(networkStrategy);
  const header =
    'name,protocol,host,port,platform,region_code,expected_country_code,egress_group,health_status,notes';
  const rows = GATEWAY_EGRESS_MVP_PROFILES.map((profile) =>
    [
      profile.key,
      'socks5h',
      normalizedStrategy.manifestHost,
      String(profile.port),
      profile.platform,
      profile.regionCode,
      profile.expectedCountryCode,
      profile.egressGroup,
      'manual-check',
      profile.notes,
    ].join(','),
  );
  return [header, ...rows].join('\n');
}

function buildInbound(profile: GatewayEgressMvpProfile, networkStrategy: GatewayEgressMvpNetworkStrategy): JsonObject {
  return {
    tag: gatewayInboundTag(profile),
    listen: networkStrategy.listenHost,
    port: profile.port,
    protocol: 'socks',
    settings: {
      auth: 'noauth',
      accounts: [],
      udp: false,
      ip: networkStrategy.listenHost,
    },
  };
}

function normalizeNetworkStrategy(strategy: GatewayEgressMvpNetworkStrategy): GatewayEgressMvpNetworkStrategy {
  const listenHost = strategy.listenHost.trim();
  const manifestHost = strategy.manifestHost.trim();
  if (!listenHost || !manifestHost || listenHost === '0.0.0.0' || manifestHost === '0.0.0.0') {
    throw new Error('Gateway egress MVP requires explicit non-wildcard listenHost and manifestHost.');
  }
  return {
    listenHost,
    manifestHost,
    strategyLabel: strategy.strategyLabel.trim() || 'custom',
  };
}

function buildPlaceholderOutbound(profile: GatewayEgressMvpProfile): JsonObject {
  return {
    tag: profile.egressGroup,
    protocol: 'freedom',
    settings: {},
    _gatewayEgressMvp: {
      profile: profile.key,
      expectedCountryCode: profile.expectedCountryCode,
      note: 'Replace this placeholder with the real VPN/proxy outbound before production use.',
    },
  };
}

function buildRules(profile: GatewayEgressMvpProfile): JsonObject[] {
  if (!profile.platform) {
    return [
      {
        type: 'field',
        inboundTag: [gatewayInboundTag(profile)],
        outboundTag: profile.egressGroup,
        _gatewayEgressMvp: true,
      },
    ];
  }

  return [
    {
      type: 'field',
      inboundTag: [gatewayInboundTag(profile)],
      domain: PLATFORM_DOMAINS[profile.platform],
      outboundTag: profile.egressGroup,
      _gatewayEgressMvp: true,
    },
  ];
}

function gatewayInboundTag(profile: GatewayEgressMvpProfile): string {
  return `gateway-${profile.key}`;
}

function upsertByTag(items: JsonValue[], item: JsonObject): void {
  const tag = String(item.tag || '');
  const index = items.findIndex((candidate) => isJsonObject(candidate) && candidate.tag === tag);
  if (index >= 0) {
    items[index] = item;
    return;
  }
  items.push(item);
}

function isGeneratedGatewayRule(value: JsonValue): boolean {
  return isJsonObject(value) && value._gatewayEgressMvp === true;
}

function isJsonObject(value: JsonValue | unknown): value is JsonObject {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}
```

- [ ] **Step 4: Run generator tests to confirm GREEN**

Run from `frontend`:

```powershell
npm run test -- frontend/tests/gateway-egress-mvp.test.ts
```

Expected: PASS.

## Task 2: Xray View MVP Panel

**Files:**
- Modify: `frontend/src/views/XrayView.vue`
- Modify: `frontend/tests/xray-view.test.ts`
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Extend the Xray view source test**

Add this test to `frontend/tests/xray-view.test.ts`:

```ts
test('xray view exposes gateway egress mvp generator without backend schema changes', () => {
  assert.match(source, /Gateway Egress MVP/);
  assert.match(source, /mergeGatewayEgressMvpConfig/);
  assert.match(source, /buildGatewayEgressManifestCsv/);
  assert.match(source, /listenHost/);
  assert.match(source, /manifestHost/);
  assert.match(source, /applyGatewayEgressMvp/);
  assert.match(source, /gatewayEgressManifestCsv/);
  assert.doesNotMatch(source, /egress_groups|egress_nodes|CoreManager|sing-box production/);
});
```

- [ ] **Step 2: Run the test to confirm RED**

Run from `frontend`:

```powershell
npm run test -- frontend/tests/xray-view.test.ts
```

Expected: FAIL because the Xray view does not expose the MVP panel yet.

- [ ] **Step 3: Import the MVP helpers**

In `frontend/src/views/XrayView.vue`, ensure `reactive` is included in the existing Vue imports, then add the MVP utility imports near the existing utility imports:

```ts
import { reactive } from 'vue';

import {
  DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY,
  buildGatewayEgressManifestCsv,
  buildGatewayEgressMvpPreview,
  mergeGatewayEgressMvpConfig,
} from '@/utils/gatewayEgressMvp';
```

Add these computed values and action near existing Xray template computed helpers:

```ts
const gatewayEgressMvpPreview = computed(() => buildGatewayEgressMvpPreview());
const gatewayEgressNetworkStrategy = reactive({ ...DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY });
const gatewayEgressManifestCsv = computed(() => buildGatewayEgressManifestCsv(gatewayEgressNetworkStrategy));

function applyGatewayEgressMvp() {
  const merged = mergeGatewayEgressMvpConfig(parsedConfig.value, gatewayEgressNetworkStrategy);
  configText.value = JSON.stringify(merged, null, 2);
  void message.success('Gateway egress MVP config generated. Review and save the Xray template to apply it.');
}
```

- [ ] **Step 4: Add the panel markup**

In `frontend/src/views/XrayView.vue`, add this card after the existing Xray template editor card and before outbound tools:

```vue
<ACard class="work-panel gateway-egress-mvp-panel" :bordered="false">
  <FormSection
    eyebrow="Gateway"
    title="Gateway Egress MVP"
    description="Generate Xray SOCKS5 inbounds and a Super-Code-Gateway registration manifest without creating new backend models."
  >
    <div class="gateway-egress-host-grid">
      <AInput v-model:value="gatewayEgressNetworkStrategy.listenHost" addon-before="listenHost" />
      <AInput v-model:value="gatewayEgressNetworkStrategy.manifestHost" addon-before="manifestHost" />
    </div>
    <template #actions>
      <ASpace wrap>
        <AButton @click="applyGatewayEgressMvp">Generate Xray Config</AButton>
        <AButton @click="copyText(gatewayEgressManifestCsv)">Copy Manifest</AButton>
        <AButton @click="downloadText('gateway-egress-mvp.csv', gatewayEgressManifestCsv)">
          Download Manifest
        </AButton>
      </ASpace>
    </template>

    <div class="gateway-egress-mvp-grid">
      <div class="client-link-card">
        <strong>{{ gatewayEgressMvpPreview.profileCount }}</strong>
        <p class="muted-text">local proxy profiles</p>
      </div>
      <div class="client-link-card">
        <strong>{{ gatewayEgressMvpPreview.ports.join(', ') }}</strong>
        <p class="muted-text">reserved loopback ports</p>
      </div>
      <div class="client-link-card">
        <strong>{{ gatewayEgressMvpPreview.platforms.join(', ') }}</strong>
        <p class="muted-text">platform exits</p>
      </div>
      <div class="client-link-card">
        <strong>{{ gatewayEgressMvpPreview.regions.join(', ') }}</strong>
        <p class="muted-text">region exits</p>
      </div>
    </div>

    <pre class="code-preview compact-preview mt-16">{{ gatewayEgressManifestCsv }}</pre>
  </FormSection>
</ACard>
```

- [ ] **Step 5: Add panel CSS**

Add to `frontend/src/styles/app.css` near the existing card/grid utilities:

```css
.gateway-egress-host-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.gateway-egress-mvp-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.gateway-egress-mvp-grid strong {
  display: block;
  color: var(--brand-ink);
  font-family: var(--font-heading);
  font-size: 20px;
  line-height: 1.2;
  overflow-wrap: anywhere;
}

@media (max-width: 760px) {
  .gateway-egress-host-grid,
  .gateway-egress-mvp-grid {
    grid-template-columns: 1fr;
  }
}
```

- [ ] **Step 6: Run Xray view tests**

Run from `frontend`:

```powershell
npm run test -- frontend/tests/xray-view.test.ts frontend/tests/gateway-egress-mvp.test.ts
```

Expected: PASS.

## Task 3: Regression Verification

**Files:**
- No source files unless verification reveals a defect.

- [ ] **Step 1: Run full frontend tests**

Run:

```powershell
cd frontend
npm run test
```

Expected: all tests pass.

- [ ] **Step 2: Run frontend typecheck**

Run:

```powershell
cd frontend
npm run typecheck
```

Expected: exit code 0.

- [ ] **Step 3: Run frontend lint**

Run:

```powershell
cd frontend
npm run lint
```

Expected: exit code 0.

- [ ] **Step 4: Run production build**

Run:

```powershell
cd frontend
npm run build
```

Expected: exit code 0 and regenerated `web/ui/` assets.

- [ ] **Step 5: Run backend compatibility checks**

Run from repository root:

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

Expected: all commands exit 0. If local toolchain is missing, record the exact blocker and do not claim backend verification passed.

- [ ] **Step 6: Browser smoke test**

Start the local frontend:

```powershell
cd frontend
npm run dev -- --host 127.0.0.1 --port 5173
```

Verify:
- `/xray` renders the Gateway Egress MVP panel.
- 375px, 768px, and 1440px widths have no horizontal overflow.
- Clicking "Generate Xray Config" updates the Xray JSON editor but does not save until the existing save action is used.
- Manifest CSV uses the selected `manifestHost`; same-network defaults remain `127.0.0.1`.

## Rollback

Revert these files to remove the MVP without touching backend state:

```text
frontend/src/utils/gatewayEgressMvp.ts
frontend/src/views/XrayView.vue
frontend/src/styles/app.css
frontend/tests/gateway-egress-mvp.test.ts
frontend/tests/xray-view.test.ts
web/ui/**
```

If an operator saved the generated Xray template, restore the previous Xray template from the panel backup or from the pre-change `xrayTemplateConfig` snapshot. No database schema rollback is required because this MVP does not add tables.

## Acceptance Criteria

- Generated Xray inbound listen hosts default to `127.0.0.1` and can only be changed to a reviewed controlled host.
- Generated Gateway-facing manifest hosts use `manifestHost`, so Docker bridge deployments are not forced to register unusable `127.0.0.1` rows.
- Generated platform ports are deterministic: `11801`, `11802`, `11803`.
- Generated region ports are deterministic: `11901`, `11981`.
- Generated platform routes include platform domain allow rules.
- Generated gateway inbounds have a final unmatched `blocked` rule.
- Gateway registration manifest is available as CSV and copyable text.
- Existing Xray template save path remains the only write path.
- No new backend model, CoreManager takeover, or sing-box production path appears in the diff.
