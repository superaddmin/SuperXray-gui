import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync('frontend/src/views/InboundsView.vue', 'utf8');

test('new inbound form exposes default client flow fields for VLESS Vision', () => {
  assert.match(source, /title="Default Client"/);
  assert.match(source, /v-if="inboundVlessFlowVisible" label="Flow"/);
  assert.match(source, /v-model:value="inboundClientEditor\.flow"/);
  assert.match(source, /streamEditor\.network === 'tcp'/);
  assert.match(source, /streamEditor\.security === 'tls' \|\| streamEditor\.security === 'reality'/);
});

test('new inbound submit syncs default client into settings JSON', () => {
  assert.match(source, /if \(inboundClientSectionVisible\.value\) \{\s*applyInboundClientEditorToSettings\(\);\s*\}/);
  assert.match(source, /function applyInboundClientEditorToSettings\(\)/);
  assert.match(source, /settings\.clients = \[\{ \.\.\.existingClient, \.\.\.client \}, \.\.\.clients\.slice\(1\)\]/);
  assert.match(source, /client\.flow = editor\.flow \|\| ''/);
});

test('inbound modal is organized into reusable form sections', () => {
  assert.match(source, /import FormSection from '@\/components\/FormSection\.vue';/);
  assert.match(source, /class="responsive-modal-form"/);
  assert.match(source, /<FormSection\s+eyebrow="Inbound"\s+title="Basic Inbound"/);
  assert.match(
    source,
    /<FormSection\s+v-if="inboundEditor\.protocol === 'wireguard'"\s+eyebrow="Protocol"\s+title="WireGuard Settings"/,
  );
  assert.match(
    source,
    /<FormSection\s+v-if="protocolSupportsStream\(inboundEditor\.protocol\)"\s+eyebrow="Transport"\s+title="Transport Settings"/,
  );
  assert.match(
    source,
    /<FormSection\s+v-if="inboundClientSectionVisible"\s+eyebrow="Client"\s+title="Default Client"/,
  );
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
