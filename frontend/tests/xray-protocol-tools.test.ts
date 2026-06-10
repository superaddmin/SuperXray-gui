import assert from 'node:assert/strict';
import test from 'node:test';

import {
  PROTOCOL_TOOL_PRESETS,
  WARP_MATRIX_OPTIONS,
  applyWarpMatrixToTemplate,
  buildWarpMatrixBaseSettings,
  generateProtocolToolArgo,
  generateProtocolToolCombo,
} from '../src/utils/xrayProtocolTools.ts';

test('protocol tool presets include xray and external combo rows', () => {
  assert.ok(PROTOCOL_TOOL_PRESETS.some((item) => item.value === 'vless-reality-vision'));
  assert.ok(PROTOCOL_TOOL_PRESETS.some((item) => item.value === 'tuic-singbox'));
});

test('generateProtocolToolArgo supports quick and fixed tunnel modes', () => {
  const quick = generateProtocolToolArgo({
    mode: 'quick',
    originUrl: 'http://127.0.0.1:2053',
    tunnelName: '',
    token: '',
  });
  assert.match(quick.command, /cloudflared tunnel --url http:\/\/127\.0\.0\.1:2053/);

  const fixed = generateProtocolToolArgo({
    mode: 'fixed',
    originUrl: 'http://127.0.0.1:2053',
    tunnelName: 'superxray',
    token: 'eyJ.test-token',
  });
  assert.match(fixed.command, /cloudflared tunnel run --token eyJ\.test-token/);
  assert.match(fixed.systemd || '', /ExecStart=\/usr\/local\/bin\/cloudflared tunnel run --token eyJ\.test-token/);
});

test('generateProtocolToolCombo returns xray-compatible outbound for vless reality vision', () => {
  const result = generateProtocolToolCombo({
    combo: 'vless-reality-vision',
    server: 'example.com',
    port: 443,
    uuid: '11111111-1111-4111-8111-111111111111',
    password: 'secret',
    sni: 'www.microsoft.com',
    publicKey: 'pub',
    shortId: 'abcd',
    path: '/xhttp',
    tag: 'proxy',
  });

  assert.equal(result.saveToXray, true);
  assert.equal(result.runtime, 'xray');
  assert.match(result.shareLink || '', /^vless:\/\//);
  assert.doesNotThrow(() => JSON.parse(result.clientOutbound || '{}'));
});

test('generateProtocolToolCombo keeps Hysteria2 on h3 without uTLS fingerprint', () => {
  const result = generateProtocolToolCombo({
    combo: 'hysteria2-tls',
    server: 'hy2.example',
    port: 443,
    password: 'hy2/auth=with padding',
    sni: 'hy2.example',
    fingerprint: 'chrome',
    tag: 'proxy',
  });

  const outbound = JSON.parse(result.clientOutbound || '{}');
  const tlsSettings = outbound.streamSettings?.tlsSettings;

  assert.equal(tlsSettings?.serverName, 'hy2.example');
  assert.deepEqual(tlsSettings?.alpn, ['h3']);
  assert.equal(tlsSettings?.settings?.fingerprint, undefined);
  assert.equal(outbound.streamSettings?.hysteriaSettings?.auth, 'hy2/auth=with padding');
  assert.match(result.shareLink || '', /hysteria2:\/\/hy2%2Fauth%3Dwith%20padding@/);
  assert.match(result.shareLink || '', /alpn=h3/);
  assert.doesNotMatch(result.shareLink || '', /fp=chrome/);
});

test('warp matrix options include openai-specific routing', () => {
  assert.ok(WARP_MATRIX_OPTIONS.some((item) => item.tag === 'warp-openai'));
});

test('buildWarpMatrixBaseSettings derives wireguard settings from warp payloads', () => {
  const settings = buildWarpMatrixBaseSettings(
    { private_key: 'private-key', client_id: 'AQID' },
    {
      interface: {
        addresses: {
          v4: '172.16.0.2',
          v6: '2606:4700:110:abcd::1',
        },
      },
      peers: [{ public_key: 'peer-public', endpoint: { host: '162.159.192.1:2408' } }],
    },
  );

  assert.deepEqual(settings.address, ['172.16.0.2/32', '2606:4700:110:abcd::1/128']);
  assert.deepEqual(settings.reserved, [1, 2, 3]);
  assert.equal(settings.peers[0].publicKey, 'peer-public');
});

test('applyWarpMatrixToTemplate preserves non-warp config and replaces old warp outputs', () => {
  const result = applyWarpMatrixToTemplate(
    {
      outbounds: [
        { tag: 'direct', protocol: 'freedom' },
        { tag: 'warp-old', protocol: 'wireguard' },
      ],
      routing: {
        rules: [
          { outboundTag: 'direct', domain: ['geosite:cn'] },
          { outboundTag: 'warp-old', domain: ['geosite:openai'] },
        ],
      },
    },
    {
      mtu: 1420,
      secretKey: 'private',
      address: ['172.16.0.2/32'],
      reserved: [1, 2, 3],
      peers: [{ publicKey: 'peer', endpoint: '162.159.192.1:2408' }],
      noKernelTun: false,
    },
    ['warp', 'warp-openai'],
  );

  assert.deepEqual(result.outbounds.map((row: { tag: string }) => row.tag), ['direct', 'warp', 'warp-openai']);
  assert.equal(result.routing.rules.length, 2);
});

test('applyWarpMatrixToTemplate does not make plain WARP the global default route', () => {
  const result = applyWarpMatrixToTemplate(
    {
      outbounds: [{ tag: 'direct', protocol: 'freedom' }],
      routing: {
        rules: [{ outboundTag: 'direct', domain: ['geosite:cn'] }],
      },
    },
    {
      mtu: 1420,
      secretKey: 'private',
      address: ['172.16.0.2/32'],
      reserved: [1, 2, 3],
      peers: [{ publicKey: 'peer', endpoint: '162.159.192.1:2408' }],
      noKernelTun: false,
    },
    ['warp'],
  );

  assert.deepEqual(result.outbounds.map((row: { tag: string }) => row.tag), ['direct', 'warp']);
  assert.deepEqual(result.routing.rules, [{ outboundTag: 'direct', domain: ['geosite:cn'] }]);
});
