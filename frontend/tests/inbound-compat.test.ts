import assert from 'node:assert/strict';
import test from 'node:test';

import {
  applyPanelDefaultTlsCertificate,
  applyHysteriaFinalmaskUdpHop,
  buildClientSubscriptionLinks,
  buildInboundShareLinks,
  defaultStreamSettings,
  generateBulkClientProfiles,
  mergeSubscriptionEndpointDefaults,
} from '../src/utils/inboundCompat.ts';

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

test('mergeSubscriptionEndpointDefaults fills enabled blank subscription URIs', () => {
  const settings = mergeSubscriptionEndpointDefaults(
    {
      subEnable: true,
      subJsonEnable: false,
      subClashEnable: true,
      subURI: '',
      subJsonURI: '',
      subClashURI: '',
    },
    {
      subURI: 'https://example.com/sub/',
      subJsonURI: 'https://example.com/json/',
      subClashURI: 'https://example.com/clash/',
    },
  );

  assert.deepEqual(settings, {
    subEnable: true,
    subJsonEnable: false,
    subClashEnable: true,
    subURI: 'https://example.com/sub/',
    subJsonURI: '',
    subClashURI: 'https://example.com/clash/',
  });
});

test('applyPanelDefaultTlsCertificate fills empty HY2 TLS certificate file paths', () => {
  const stream = applyPanelDefaultTlsCertificate(
    {
      network: 'hysteria',
      security: 'tls',
      tlsSettings: {
        certificates: [],
      },
    },
    {
      certFile: '/etc/superxray/cert.pem',
      keyFile: '/etc/superxray/key.pem',
    },
  );

  assert.deepEqual(stream.tlsSettings?.certificates, [
    {
      certificateFile: '/etc/superxray/cert.pem',
      keyFile: '/etc/superxray/key.pem',
      oneTimeLoading: false,
      usage: 'encipherment',
      buildChain: false,
    },
  ]);
});

test('applyPanelDefaultTlsCertificate preserves existing TLS certificates', () => {
  const stream = applyPanelDefaultTlsCertificate(
    {
      tlsSettings: {
        certificates: [
          {
            certificateFile: '/custom/cert.pem',
            keyFile: '/custom/key.pem',
          },
        ],
      },
    },
    {
      certFile: '/etc/superxray/cert.pem',
      keyFile: '/etc/superxray/key.pem',
    },
  );

  assert.deepEqual(stream.tlsSettings?.certificates, [
    {
      certificateFile: '/custom/cert.pem',
      keyFile: '/custom/key.pem',
    },
  ]);
});

test('applyPanelDefaultTlsCertificate preserves existing inline TLS certificate content', () => {
  const stream = applyPanelDefaultTlsCertificate(
    {
      tlsSettings: {
        certificates: [
          {
            certificate: ['-----BEGIN CERTIFICATE-----', 'MIIB', '-----END CERTIFICATE-----'],
            key: ['-----BEGIN PRIVATE KEY-----', 'MIIB', '-----END PRIVATE KEY-----'],
          },
        ],
      },
    },
    {
      certFile: '/etc/superxray/cert.pem',
      keyFile: '/etc/superxray/key.pem',
    },
  );

  assert.deepEqual(stream.tlsSettings?.certificates, [
    {
      certificate: ['-----BEGIN CERTIFICATE-----', 'MIIB', '-----END CERTIFICATE-----'],
      key: ['-----BEGIN PRIVATE KEY-----', 'MIIB', '-----END PRIVATE KEY-----'],
    },
  ]);
});

test('defaultStreamSettings keeps Hysteria2 on h3 without uTLS fingerprint', () => {
  const stream = defaultStreamSettings('hysteria2');
  const tlsSettings = stream.tlsSettings as Record<string, unknown>;
  const tlsClientSettings = tlsSettings.settings as Record<string, unknown>;

  assert.deepEqual(tlsSettings.alpn, ['h3']);
  assert.equal(tlsClientSettings.fingerprint, '');
});


test('applyHysteriaFinalmaskUdpHop writes UDP Hop without dropping salamander obfs', () => {
  const stream = applyHysteriaFinalmaskUdpHop(
    {
      finalmask: {
        udp: [
          {
            type: 'salamander',
            settings: { password: 'obfs-pass' },
          },
        ],
      },
    },
    {
      quicParamsEnabled: true,
      udpHopEnabled: true,
      ports: '40000:45000',
      interval: '5:10',
    },
  );

  assert.deepEqual(stream.finalmask, {
    udp: [
      {
        type: 'salamander',
        settings: { password: 'obfs-pass' },
      },
    ],
    quicParams: {
      udpHop: { ports: '40000-45000', interval: '5-10' },
    },
  });
});

test('applyHysteriaFinalmaskUdpHop removes only udpHop when disabled', () => {
  const stream = applyHysteriaFinalmaskUdpHop(
    {
      finalmask: {
        udp: [{ type: 'salamander', settings: { password: 'obfs-pass' } }],
        quicParams: {
          congestion: 'bbr',
          udpHop: { ports: '40000-45000', interval: '5-10' },
        },
      },
    },
    {
      quicParamsEnabled: true,
      udpHopEnabled: false,
      ports: '40000-45000',
      interval: '5-10',
    },
  );

  assert.deepEqual(stream.finalmask, {
    udp: [{ type: 'salamander', settings: { password: 'obfs-pass' } }],
    quicParams: { congestion: 'bbr' },
  });
});

test('applyHysteriaFinalmaskUdpHop removes all quicParams when QUIC Params is disabled', () => {
  const stream = applyHysteriaFinalmaskUdpHop(
    {
      finalmask: {
        udp: [{ type: 'salamander', settings: { password: 'obfs-pass' } }],
        quicParams: {
          congestion: 'bbr',
          udpHop: { ports: '40000-45000', interval: '5-10' },
        },
      },
    },
    {
      quicParamsEnabled: false,
      udpHopEnabled: true,
      ports: '40000-45000',
      interval: '5-10',
    },
  );

  assert.deepEqual(stream.finalmask, {
    udp: [{ type: 'salamander', settings: { password: 'obfs-pass' } }],
  });
});
test('buildInboundShareLinks exports Hysteria2 UDP hop ports without default fp', () => {
  const links = buildInboundShareLinks({
    protocol: 'hysteria',
    remark: 'hy2-hop',
    listen: '203.0.113.20',
    port: 443,
    settings: JSON.stringify({
      version: 2,
      clients: [{ email: 'hy2@example.com', auth: 'hy2-auth', enable: true }],
    }),
    streamSettings: JSON.stringify({
      network: 'hysteria',
      security: 'tls',
      tlsSettings: {
        serverName: 'hy2.example',
        alpn: ['h3'],
        settings: { fingerprint: '' },
      },
      finalmask: {
        udp: [
          {
            type: 'salamander',
            settings: { password: 'obfs-pass' },
          },
        ],
        quicParams: {
          udpHop: { ports: '40000-45000', interval: '5-10' },
        },
      },
    }),
  } as never);

  assert.equal(links.length, 1);
  const link = new URL(links[0]);
  assert.equal(link.searchParams.get('alpn'), 'h3');
  assert.equal(link.searchParams.get('mport'), '40000-45000');
  assert.equal(link.searchParams.get('obfs'), 'salamander');
  assert.equal(link.searchParams.get('obfs-password'), 'obfs-pass');
  assert.match(link.searchParams.get('fm') || '', /"udpHop"/);
  assert.equal(link.searchParams.has('fp'), false);
});

test('buildInboundShareLinks exports single-user Shadowsocks links like legacy UI', () => {
  const links = buildInboundShareLinks({
    protocol: 'shadowsocks',
    remark: 'single-ss',
    listen: '203.0.113.10',
    port: 8388,
    settings: JSON.stringify({
      method: '2022-blake3-chacha20-poly1305',
      password: 'server-secret',
      network: 'tcp,udp',
      clients: [],
    }),
    streamSettings: JSON.stringify({ network: 'tcp', security: 'none', externalProxy: [] }),
  } as never);

  assert.equal(links.length, 1);
  assert.match(links[0], /^ss:\/\//);
  assert.match(links[0], /203\.0\.113\.10:8388/);
  assert.match(links[0], /#single-ss/);
});

test('buildInboundShareLinks preserves external proxy export rows', () => {
  const links = buildInboundShareLinks({
    protocol: 'vless',
    remark: 'edge',
    listen: '0.0.0.0',
    port: 443,
    settings: JSON.stringify({
      clients: [{ id: '11111111-1111-4111-8111-111111111111', email: 'alice' }],
      decryption: 'none',
    }),
    streamSettings: JSON.stringify({
      network: 'tcp',
      security: 'reality',
      externalProxy: [{ remark: 'cdn', dest: 'cdn.example.com', port: 8443, forceTls: 'same' }],
      realitySettings: {
        settings: { publicKey: 'pub', fingerprint: 'chrome', spiderX: '/' },
        serverNames: ['www.apple.com'],
        shortIds: ['abcd'],
      },
    }),
  } as never);

  assert.equal(links.length, 1);
  assert.match(links[0], /cdn\.example\.com:8443/);
  assert.match(links[0], /#edge-alice-cdn/);
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
