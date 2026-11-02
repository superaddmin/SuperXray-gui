import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync('frontend/src/views/ClientsView.vue', 'utf8');

test('clients overview masks credentials and subscription identifiers by default', () => {
  assert.match(source, /maskSecret\(record\.credential\)/);
  assert.match(source, /maskSecret\(record\.subId\)/);
  assert.match(source, /function maskSecret\(value: string\): string/);
  assert.match(source, /return value\.trim\(\) \? '••••••••' : ''/);
});

test('clients overview falls back to the stored client quota when traffic stats are absent', () => {
  assert.match(source, /const total = Number\(stats\?\.total \?\? client\.totalGB \?\? 0\)/);
});

test('clients overview exposes start and stop actions backed by inbound client updates', () => {
  assert.match(source, /title: 'Actions', key: 'actions'/);
  assert.match(source, /updateInboundClient/);
  assert.match(source, /updateInbound\(/);
  assert.match(source, /parseInboundSettings\(inbound\)/);
  assert.match(source, /const clientActionKey = ref\(''\)/);
  assert.match(source, /Boolean\(clientActionKey\.value\)/);
  assert.match(source, /:disabled="Boolean\(clientActionKey\)"/);
  assert.match(source, /\{\s*\.\.\.row\.client,\s*enable: enabled\s*\}/);
  assert.match(source, /settings\.peers = peers\.map/);
  assert.match(source, /record\.clientEnabled \? 'Stop' : 'Start'/);
});
