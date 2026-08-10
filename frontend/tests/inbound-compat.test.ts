import assert from 'node:assert/strict';
import test from 'node:test';

import {
  HYSTERIA_QUIC_DEFAULTS,
  applyPanelDefaultTlsCertificate,
  applyHysteriaFinalmaskUdpHop,
  buildClientSubscriptionLinks,
  buildInboundShareLinks,
  defaultInboundSettings,
  defaultStreamSettings,
  generateBulkClientProfiles,
  mergeSubscriptionEndpointDefaults,
  normalizeTunSettings,
  validateTunSettings,
} from '../src/utils/inboundCompat.ts';
import { protocolSupportsShareLink } from '../src/schemas/protocolRegistry.ts';

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

test('default proxy account settings include subscription id for HTTP and mixed', () => {
  for (const protocol of ['http', 'mixed'] as const) {
    const settings = defaultInboundSettings(protocol);
    const accounts = settings.accounts as Array<Record<string, unknown>>;
    assert.equal(Array.isArray(accounts), true);
    assert.equal(typeof accounts[0]?.subId, 'string');
    assert.ok(String(accounts[0]?.subId).length > 0);
  }
});

test('default TUN settings match the current Xray scalar MTU schema', () => {
  assert.deepEqual(defaultInboundSettings('tun'), {
    name: 'xray0',
    mtu: 1500,
    gateway: ['10.0.0.1/16'],
    dns: [],
    userLevel: 0,
    autoSystemRoutingTable: [],
    autoOutboundsInterface: 'auto',
  });
});

test('normalizeTunSettings keeps unknown local fields while migrating legacy aliases', () => {
  assert.deepEqual(
    normalizeTunSettings({
      name: ' xray-local ',
      MTU: [1280, 1500],
      Gateway: ['10.0.0.1/16', '10.0.0.1/16'],
      DNS: ['1.1.1.1', ' 2606:4700:4700::1111 '],
      userLevel: 2,
      autoSystemRoutingTable: ['0.0.0.0/0'],
      autoOutboundsInterface: ' eth0 ',
      localExtension: { enabled: true },
    }),
    {
      name: 'xray-local',
      mtu: 1280,
      gateway: ['10.0.0.1/16'],
      dns: ['1.1.1.1', '2606:4700:4700::1111'],
      userLevel: 2,
      autoSystemRoutingTable: ['0.0.0.0/0'],
      autoOutboundsInterface: 'eth0',
      localExtension: { enabled: true },
    },
  );
});

test('validateTunSettings accepts IPv4 and IPv6 values and rejects invalid network input', () => {
  const valid = normalizeTunSettings({
    name: 'xray0',
    mtu: 1500,
    gateway: ['10.0.0.1/16', 'fc00::1/64'],
    dns: ['1.1.1.1', '2606:4700:4700::1111'],
    userLevel: 0,
    autoSystemRoutingTable: ['0.0.0.0/0', '::/0'],
    autoOutboundsInterface: 'auto',
  });
  assert.equal(validateTunSettings(valid), '');
  assert.match(validateTunSettings(normalizeTunSettings({ ...valid, mtu: 0 })), /TUN MTU/);
  assert.match(
    validateTunSettings(normalizeTunSettings({ ...valid, userLevel: -1 })),
    /user level/,
  );
  assert.match(validateTunSettings({ ...valid, gateway: ['10.0.0.1/99'] }), /gateway CIDR/);
  assert.match(validateTunSettings({ ...valid, dns: ['1.1.1.999'] }), /DNS address/);
  assert.match(validateTunSettings({ ...valid, dns: ['010.0.0.1'] }), /DNS address/);
  assert.match(
    validateTunSettings({ ...valid, autoSystemRoutingTable: ['not-a-prefix'] }),
    /system route CIDR/,
  );
});

test('protocol registry marks HTTP and mixed proxy inbounds as shareable', () => {
  assert.equal(protocolSupportsShareLink('http'), true);
  assert.equal(protocolSupportsShareLink('mixed'), true);
});

test('buildInboundShareLinks exports HTTP and SOCKS5 proxy account links', () => {
  const httpLinks = buildInboundShareLinks({
    protocol: 'http',
    remark: 'http-proxy',
    listen: 'http.example.com',
    port: 8080,
    settings: JSON.stringify({
      accounts: [{ user: 'http-user', pass: 'http/pass with space', subId: 'sub-http' }],
    }),
    streamSettings: '{}',
  } as never);
  const socksLinks = buildInboundShareLinks({
    protocol: 'mixed',
    remark: 'socks-proxy',
    listen: 'socks.example.com',
    port: 1080,
    settings: JSON.stringify({
      accounts: [{ user: 'socks-user', pass: 'socks/pass with space', subId: 'sub-socks' }],
    }),
    streamSettings: '{}',
  } as never);

  assert.deepEqual(httpLinks, [
    'http://http-user:http%2Fpass%20with%20space@http.example.com:8080#http-proxy',
  ]);
  assert.deepEqual(socksLinks, [
    'socks5://socks-user:socks%2Fpass%20with%20space@socks.example.com:1080#socks-proxy',
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
      ...HYSTERIA_QUIC_DEFAULTS,
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
      ...HYSTERIA_QUIC_DEFAULTS,
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
      ...HYSTERIA_QUIC_DEFAULTS,
      quicParamsEnabled: true,
      udpHopEnabled: false,
      ports: '40000-45000',
      interval: '5-10',
    },
  );

  assert.deepEqual(stream.finalmask, {
    udp: [{ type: 'salamander', settings: { password: 'obfs-pass' } }],
    quicParams: { congestion: 'bbr', ...HYSTERIA_QUIC_DEFAULTS },
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
      ...HYSTERIA_QUIC_DEFAULTS,
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
