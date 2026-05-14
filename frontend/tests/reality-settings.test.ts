import assert from 'node:assert/strict';
import test from 'node:test';

import {
  normalizeRealityServerSettings,
  validateRealityServerSettings,
} from '../src/utils/realitySettings.ts';

test('normalizeRealityServerSettings fills blank target and server names', () => {
  const normalized = normalizeRealityServerSettings({
    target: '  ',
    serverNames: '',
    privateKey: ' server-private ',
    shortIds: ' abc123 , def456 ',
    publicKey: '',
    spiderX: '',
  });

  assert.equal(normalized.target, 'www.apple.com:443');
  assert.deepEqual(normalized.serverNames, ['www.apple.com']);
  assert.equal(normalized.privateKey, 'server-private');
  assert.deepEqual(normalized.shortIds, ['abc123', 'def456']);
  assert.equal(normalized.spiderX, '/');
});

test('validateRealityServerSettings rejects missing server secrets before saving', () => {
  const error = validateRealityServerSettings({
    target: 'www.apple.com:443',
    serverNames: 'www.apple.com',
    privateKey: '',
    shortIds: '',
  });

  assert.match(error || '', /private key/i);
});
