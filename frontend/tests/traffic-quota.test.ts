import assert from 'node:assert/strict';
import test from 'node:test';

import { bytesToGb, resolveTotalBytes } from '../src/utils/trafficQuota.ts';

test('editing remarks preserves exact stored quota bytes across repeated saves', () => {
  for (const originalBytes of [0, 1, 1024, 1234567, 1234567890, Number.MAX_SAFE_INTEGER]) {
    const displayedGB = bytesToGb(originalBytes);
    const saved = resolveTotalBytes(originalBytes, displayedGB);
    assert.equal(saved, originalBytes);
    assert.equal(resolveTotalBytes(saved, bytesToGb(saved)), originalBytes);
  }
});

test('tiny existing quotas remain finite when their GB display rounds to zero', () => {
  assert.equal(bytesToGb(1), 0);
  assert.equal(bytesToGb(1234567), 0);
  assert.equal(resolveTotalBytes(1, 0), 1);
  assert.equal(resolveTotalBytes(1234567, 0), 1234567);
});

test('explicit quota edits override the original bytes including unlimited', () => {
  assert.equal(resolveTotalBytes(1234567890, 2, true), 2147483648);
  assert.equal(resolveTotalBytes(1234567890, 2), 2147483648);
  assert.equal(resolveTotalBytes(1234567890, 0, true), 0);
  assert.equal(resolveTotalBytes(1, 0, true), 0);
  assert.equal(resolveTotalBytes(1, 0.5, true), 536870912);
});

test('new inbound and client quotas convert GB without inheriting a prior edit snapshot', () => {
  assert.equal(resolveTotalBytes(null, 1.25), 1342177280);
  assert.equal(resolveTotalBytes(undefined, 0), 0);
  assert.equal(resolveTotalBytes(null, 0.01, true), 10737418);
});
