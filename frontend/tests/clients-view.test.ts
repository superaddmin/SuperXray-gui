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
