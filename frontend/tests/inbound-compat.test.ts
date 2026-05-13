import assert from 'node:assert/strict';
import test from 'node:test';

import { buildClientSubscriptionLinks, generateBulkClientProfiles } from '../src/utils/inboundCompat.ts';

test('buildClientSubscriptionLinks returns enabled subscription endpoints for a client subId', () => {
  const links = buildClientSubscriptionLinks(
    { subId: 'client-sub-id' },
    {
      subEnable: true,
      subJsonEnable: true,
      subClashEnable: true,
      subURI: 'https://example.com/sub/',
      subJsonURI: 'https://example.com/json/',
      subClashURI: 'https://example.com/clash/',
    },
  );

  assert.deepEqual(links, [
    { label: 'URI', url: 'https://example.com/sub/client-sub-id' },
    { label: 'JSON', url: 'https://example.com/json/client-sub-id' },
    { label: 'Clash', url: 'https://example.com/clash/client-sub-id' },
  ]);
});

test('buildClientSubscriptionLinks omits disabled or empty subscription endpoints', () => {
  const links = buildClientSubscriptionLinks(
    { subId: 'client-sub-id' },
    {
      subEnable: true,
      subJsonEnable: false,
      subClashEnable: true,
      subURI: 'https://example.com/sub/',
      subJsonURI: '',
      subClashURI: '',
    },
  );

  assert.deepEqual(links, [{ label: 'URI', url: 'https://example.com/sub/client-sub-id' }]);
});

test('buildClientSubscriptionLinks returns empty when subscription is disabled or subId missing', () => {
  assert.deepEqual(
    buildClientSubscriptionLinks(
      { subId: '' },
      {
        subEnable: true,
        subJsonEnable: true,
        subClashEnable: true,
        subURI: 'https://example.com/sub/',
        subJsonURI: 'https://example.com/json/',
        subClashURI: 'https://example.com/clash/',
      },
    ),
    [],
  );

  assert.deepEqual(
    buildClientSubscriptionLinks(
      { subId: 'client-sub-id' },
      {
        subEnable: false,
        subJsonEnable: true,
        subClashEnable: true,
        subURI: 'https://example.com/sub/',
        subJsonURI: 'https://example.com/json/',
        subClashURI: 'https://example.com/clash/',
      },
    ),
    [],
  );
});

test('generateBulkClientProfiles creates sequential client emails and unique ids', () => {
  const profiles = generateBulkClientProfiles({
    protocol: 'vless',
    quantity: 3,
    firstIndex: 7,
    emailPrefix: 'team-',
    emailPostfix: '@example.com',
    flow: 'xtls-rprx-vision',
    limitIp: 2,
    totalGB: 10,
    expiryTime: 1234567890,
    reset: 30,
  });

  assert.equal(profiles.length, 3);
  assert.deepEqual(
    profiles.map((item) => item.email),
    ['team-7@example.com', 'team-8@example.com', 'team-9@example.com'],
  );
  assert.ok(profiles.every((item) => item.id && item.id.length > 0));
  assert.ok(profiles.every((item) => item.subId && item.subId.length > 0));
  assert.ok(profiles.every((item) => item.flow === 'xtls-rprx-vision'));
  assert.ok(profiles.every((item) => item.limitIp === 2));
  assert.ok(profiles.every((item) => item.totalGB === 10));
  assert.ok(profiles.every((item) => item.expiryTime === 1234567890));
});
