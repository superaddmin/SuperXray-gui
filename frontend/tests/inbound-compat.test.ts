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
  mergeXhttpSettings,
  mergeSubscriptionEndpointDefaults,
  normalizeFinalmaskForXray,
  normalizeInboundStreamCompatibility,
  normalizeTunSettings,
  normalizeWireguardSettings,
  restoreInboundClients,
  resolveXhttpHost,
  resolveXhttpExtraSettings,
  resolveHysteriaUdpHop,
  separateInboundClients,
  validateHysteriaQuicFormInput,
  validateFinalmaskForXray,
  validateShadowsocksMethod,
  validateXhttpFormInput,
  validateXhttpSessionSettings,
  validateTunSettings,
} from '../src/utils/inboundCompat.ts';
import { protocolSupportsShareLink } from '../src/schemas/protocolRegistry.ts';

test('26.10.10 stream aliases migrate without dropping unknown fields', () => {
  const stream = normalizeInboundStreamCompatibility({
    xhttpSettings: {
      sessionPlacement: 'header',
      sessionKey: 'legacy-session',
      sessionIDTable: 'Base62',
      sessionIDLength: '8-16',
      localExtension: { keep: true },
    },
    realitySettings: { dest: 'example.com:443', settings: { publicKey: 'pk' } },
    sockopt: { v6only: true, localSockopt: 'keep' },
  });

  assert.equal((stream.xhttpSettings as Record<string, unknown>).sessionIDPlacement, 'header');
  assert.equal((stream.xhttpSettings as Record<string, unknown>).sessionIDKey, 'legacy-session');
  assert.equal((stream.xhttpSettings as Record<string, unknown>).sessionPlacement, undefined);
  assert.equal((stream.realitySettings as Record<string, unknown>).target, 'example.com:443');
  assert.equal((stream.sockopt as Record<string, unknown>).V6Only, true);
  assert.deepEqual((stream.sockopt as Record<string, unknown>).localSockopt, 'keep');
});

test('26.10.10 finalmask conversion is idempotent and keeps unknown masks', () => {
  const input = {
    finalmask: {
      udp: [{ type: 'custom', settings: { keep: true } }],
      quicParams: { udpHop: { ports: '20000-30000', interval: '5-10' } },
    },
  };
  const first = normalizeFinalmaskForXray(input);
  const second = normalizeFinalmaskForXray(first);
  assert.deepEqual(second, first);
  const finalmask = first.finalmask as Record<string, unknown>;
  const udp = finalmask.udp as unknown[];
  assert.equal(finalmask.quicParams, undefined);
  assert.deepEqual(udp[1], {
    type: 'udphop',
    settings: { mode: 'intervalRemote', remotePorts: '20000-30000', interval: '5-10' },
  });
  assert.deepEqual(udp[0], { type: 'custom', settings: { keep: true } });
});

test('26.10.10 WireGuard drops removed workers while preserving extension fields', () => {
  const settings = normalizeWireguardSettings({
    workers: 4,
    peers: [],
    localExtension: { keep: true },
  });
  assert.equal(settings.workers, undefined);
  assert.deepEqual(settings.localExtension, { keep: true });
});

test('26.10.10 XHTTP session table and length validation is actionable', () => {
  assert.match(validateXhttpSessionSettings({ sessionIDTable: '非ASCII' }), /ASCII/);
  assert.match(validateXhttpSessionSettings({ sessionIDLength: '0-8' }), /positive/);
  assert.equal(validateXhttpSessionSettings({ sessionIDLength: '16-8' }), '');
  assert.equal(validateXhttpSessionSettings({ sessionIDLength: '8-16' }), '');
  assert.equal(validateXhttpSessionSettings({ extra: { sessionIDLength: 16 } }), '');
  assert.match(
    validateXhttpSessionSettings({ extra: { sessionIDLength: '2147483648' } }),
    /length or range/,
  );
  assert.match(validateXhttpSessionSettings({ extra: { sessionIDTable: '非ASCII' } }), /ASCII/);
});

test('26.10.10 rejects removed Shadowsocks cipher names explicitly', () => {
  assert.match(validateShadowsocksMethod('none'), /removed/);
  assert.match(validateShadowsocksMethod('plain'), /removed/);
  assert.equal(validateShadowsocksMethod('aes-256-gcm'), '');
  assert.equal(validateShadowsocksMethod('aes-128-gcm'), '');
  assert.equal(validateShadowsocksMethod('xchacha20-poly1305'), '');
  assert.equal(validateShadowsocksMethod('chacha20-ietf-poly1305'), '');
});

test('26.10.10 finalmask rejects incompatible settings and keeps UTF-8 password semantics', () => {
  assert.match(
    validateFinalmaskForXray({ security: 'reality', finalmask: { tcp: [{ type: 'header' }] } }),
    /REALITY/,
  );
  assert.match(
    validateFinalmaskForXray({
      finalmask: { udp: [{ type: 'salamander', settings: { password: 'abc' } }] },
    }),
    /4 bytes/,
  );
  assert.equal(
    validateFinalmaskForXray({
      finalmask: { udp: [{ type: 'salamander', settings: { password: '密码' } }] },
    }),
    '',
  );
  assert.match(
    validateFinalmaskForXray({
      finalmask: { tcp: [{ type: 'xmc', settings: { username: 'user' } }] },
    }),
    /UUID/,
  );
  assert.match(
    validateFinalmaskForXray({
      finalmask: { udp: [{ type: 'xdns', settings: { domains: ['example.com:mx'] } }] },
    }),
    /unsupported/,
  );
});

test('26.10.10 xDNS migration preserves unknown fields and handles explicit TXT records', () => {
  const input = {
    finalmask: {
      udp: [
        {
          type: 'xdns',
          settings: {
            domains: ['example.com:txt', { name: 'second.example', types: [1], vendor: true }],
            resolvers: [{ type: 'udp', settings: { addr: '127.0.0.1:53' }, vendor: true }],
          },
        },
      ],
    },
  };
  const migrated = normalizeFinalmaskForXray(input);
  const mask = (migrated.finalmask?.udp as Array<Record<string, unknown>>)[0];
  const settings = mask.settings as Record<string, unknown>;
  assert.deepEqual(settings.domains, [
    { names: ['example.com'], types: [16], edns0: 1232 },
    { names: ['second.example'], types: [1], vendor: true },
  ]);
  assert.equal((settings.resolvers as Array<Record<string, unknown>>)[0].vendor, true);
  assert.deepEqual(normalizeFinalmaskForXray(migrated), migrated);
  assert.equal(input.finalmask.udp[0].settings.domains[0], 'example.com:txt');
});

test('26.10.10 migrated UDP hop remains editable without duplicating or losing other masks', () => {
  const original = {
    finalmask: {
      udp: [
        { type: 'salamander', settings: { password: 'test-password' } },
        {
          type: 'udphop',
          settings: {
            mode: 'intervalRemote',
            remotePorts: '20000-30000',
            interval: 30,
            vendor: true,
          },
        },
      ],
    },
  };
  assert.equal(resolveHysteriaUdpHop(original.finalmask).ports, '20000-30000');
  const input = {
    ...HYSTERIA_QUIC_DEFAULTS,
    quicParamsEnabled: true,
    udpHopEnabled: true,
    ports: '40000-45000',
    interval: '5-10',
  };
  const migrated = normalizeFinalmaskForXray(applyHysteriaFinalmaskUdpHop(original, input));
  const masks = migrated.finalmask?.udp as Array<Record<string, unknown>>;
  assert.equal(masks.length, 2);
  assert.deepEqual(masks[1].settings, {
    mode: 'intervalRemote',
    remotePorts: '40000-45000',
    interval: '5-10',
    vendor: true,
  });
  const disabled = normalizeFinalmaskForXray(
    applyHysteriaFinalmaskUdpHop(migrated, { ...input, udpHopEnabled: false }),
  );
  assert.deepEqual(disabled.finalmask?.udp, [original.finalmask.udp[0]]);
});

test('26.10.10 validates XMC profiles instead of obsolete root credentials', () => {
  const profile = {
    username: 'Player_1',
    uuid: '00000000-0000-4000-8000-000000000001',
    texturesValue: 'fixture-value',
    texturesSignature: 'fixture-signature',
  };
  const stream = { finalmask: { tcp: [{ type: 'xmc', settings: { profiles: [profile] } }] } };
  assert.equal(validateFinalmaskForXray(stream), '');
  for (const invalid of [
    { ...profile, username: 'x' },
    { ...profile, uuid: 'invalid' },
    { ...profile, texturesValue: '' },
    { ...profile, texturesSignature: '' },
  ]) {
    assert.match(
      validateFinalmaskForXray({
        finalmask: { tcp: [{ type: 'xmc', settings: { profiles: [invalid] } }] },
      }),
      /profile/i,
    );
  }
});

test('26.10.10 preserves old xDNS resolver names and each record type', () => {
  const input = {
    finalmask: {
      udp: [
        {
          type: 'xdns',
          settings: {
            domains: ['same.example:a'],
            resolvers: [
              'same.example:txt+udp://127.0.0.1:53',
              'other.example:aaaa+udp://127.0.0.2:53',
            ],
          },
        },
      ],
    },
  };
  const got = normalizeFinalmaskForXray(input);
  const settings = (got.finalmask?.udp as Array<Record<string, unknown>>)[0].settings as Record<
    string,
    unknown
  >;
  assert.deepEqual(settings.domains, [
    { names: ['same.example'], types: [1], edns0: 1232 },
    { names: ['same.example'], types: [16], edns0: 1232 },
    { names: ['other.example'], types: [28], edns0: 1232 },
  ]);
  assert.deepEqual(normalizeFinalmaskForXray(got), got);
});

test('26.10.10 rejects invalid or conflicting UDP hops without changing the input', () => {
  const cases = [
    { quicParams: { udpHop: true } },
    { quicParams: { udpHop: { interval: 30 } } },
    {
      quicParams: { udpHop: { ports: '1000', vendor: 1 } },
      udp: [
        { type: 'udphop', settings: { mode: 'intervalRemote', remotePorts: '1000', vendor: 2 } },
      ],
    },
    {
      udp: [
        { type: 'udphop', settings: { mode: 'intervalRemote', remotePorts: '1000' } },
        { type: 'udphop', settings: { mode: 'intervalRemote', remotePorts: '2000' } },
      ],
    },
  ];
  for (const finalmask of cases) {
    const input = { finalmask };
    assert.match(validateFinalmaskForXray(input), /udp.?hop|UDP hop/i);
    assert.deepEqual(normalizeFinalmaskForXray(input), input);
  }
});

test('26.10.10 deduplicates identical hops and preserves legacy extension types', () => {
  const hop = {
    type: 'udphop',
    settings: { mode: 'intervalRemote', remotePorts: '1000', interval: 30, vendor: { keep: true } },
  };
  const input = {
    finalmask: {
      quicParams: { udpHop: { ports: '1000', interval: 30, vendor: { keep: true } } },
      udp: [
        hop,
        { type: 'custom' },
        {
          settings: {
            vendor: { keep: true },
            interval: 30,
            remotePorts: '1000',
            mode: 'intervalRemote',
          },
          type: 'udphop',
        },
      ],
    },
  };
  const got = normalizeFinalmaskForXray(input);
  assert.equal(validateFinalmaskForXray(input), '');
  assert.deepEqual(got.finalmask?.udp, [{ type: 'custom' }, hop]);
  assert.deepEqual(normalizeFinalmaskForXray(got), got);
});

test('26.10.10 rejects non-integer session length types', () => {
  for (const value of [false, true, null, [], {}, 1.5, 2147483648, '1-2147483648']) {
    assert.notEqual(
      validateXhttpSessionSettings({ sessionIDLength: value }),
      '',
      JSON.stringify(value),
    );
  }
});

test('26.10.10 xDNS mixed aliases keep names and invalid domains cannot be overwritten', () => {
  const input = {
    finalmask: {
      udp: [
        {
          type: 'xdns',
          settings: {
            domains: [
              { name: 'legacy.test', names: ['new.test'], vendor: true },
              { names: ['same.test'], types: [1], edns0: 1232 },
            ],
            resolvers: ['same.test:a+udp://127.0.0.1:53'],
          },
        },
      ],
    },
  };
  const got = normalizeFinalmaskForXray(input);
  const settings = (got.finalmask?.udp as Array<Record<string, unknown>>)[0].settings as Record<
    string,
    unknown
  >;
  assert.deepEqual(settings.domains, [
    { names: ['new.test', 'legacy.test'], vendor: true },
    { names: ['same.test'], types: [1], edns0: 1232 },
  ]);
  const invalid = {
    finalmask: {
      udp: [
        {
          type: 'xdns',
          settings: {
            domains: { vendor: true },
            resolvers: ['same.test:a+udp://127.0.0.1:53'],
          },
        },
      ],
    },
  };
  assert.match(validateFinalmaskForXray(invalid), /domains must be an array/);
  assert.deepEqual(normalizeFinalmaskForXray(invalid), invalid);
});

test('advanced inbound settings isolate clients and restore them without data loss', () => {
  const originalClients = [
    { id: 'client-1', email: 'first@example.com', localExtension: { keep: true } },
    { id: 'client-2', email: 'second@example.com' },
  ];
  const separated = separateInboundClients({
    clients: originalClients,
    decryption: 'none',
    localSetting: { enabled: true },
  });

  assert.deepEqual(separated.editorSettings, {
    decryption: 'none',
    localSetting: { enabled: true },
  });
  assert.deepEqual(separated.clients, originalClients);
  assert.deepEqual(
    restoreInboundClients({ ...separated.editorSettings, encryption: 'none' }, separated.clients),
    {
      clients: originalClients,
      decryption: 'none',
      encryption: 'none',
      localSetting: { enabled: true },
    },
  );
});

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

function defaultXhttpFormInput() {
  return {
    path: '/xhttp',
    host: 'xhttp.example.com',
    mode: 'packet-up',
    noSSEHeader: true,
    scMaxBufferedPosts: 8,
    scMaxEachPostBytes: '1000000',
    scStreamUpServerSecs: '20-80',
    xPaddingBytes: '100-1000',
    xPaddingObfsMode: true,
    xPaddingKey: 'x_padding',
    xPaddingHeader: 'X-Padding',
    xPaddingPlacement: 'header',
    xPaddingMethod: 'tokenish',
    uplinkHTTPMethod: 'POST',
    sessionPlacement: 'header',
    sessionKey: 'x_session',
    seqPlacement: 'query',
    seqKey: 'x_seq',
    uplinkDataPlacement: 'header',
    uplinkDataKey: 'x_data',
    uplinkChunkSize: 4096,
    xmuxEnabled: true,
    xmuxMaxConcurrency: '16-32',
    xmuxMaxConnections: '',
    xmuxCMaxReuseTimes: '10',
    xmuxHMaxRequestTimes: '600-900',
    xmuxHMaxReusableSecs: '1800-3000',
    xmuxHKeepAlivePeriod: 30,
  } as const;
}

test('resolveXhttpExtraSettings migrates legacy root fields without overriding nested values', () => {
  const extra = resolveXhttpExtraSettings({
    xPaddingBytes: '100-1000',
    sessionKey: 'legacy-session',
    extra: { sessionKey: 'nested-session', localExtra: { enabled: true } },
  });

  assert.equal(extra.xPaddingBytes, '100-1000');
  assert.equal(extra.sessionIDKey, 'nested-session');
  assert.equal(extra.sessionKey, undefined);
  assert.deepEqual(extra.localExtra, { enabled: true });
});

test('resolveXhttpHost preserves root and nested legacy Host headers', () => {
  assert.equal(resolveXhttpHost({ headers: { Host: 'root.example.com' } }), 'root.example.com');
  assert.equal(
    resolveXhttpHost({ extra: { headers: { host: 'nested.example.com' } } }),
    'nested.example.com',
  );
  assert.equal(
    resolveXhttpHost({
      host: 'direct.example.com',
      extra: { headers: { Host: 'nested.example.com' } },
    }),
    'direct.example.com',
  );

  const nestedLegacy = {
    path: '/legacy',
    mode: 'packet-up',
    extra: { headers: { host: 'roundtrip.example.com', 'X-Local': 'keep' } },
  };
  const merged = mergeXhttpSettings(nestedLegacy, {
    ...defaultXhttpFormInput(),
    host: resolveXhttpHost(nestedLegacy),
  });
  assert.deepEqual((merged.extra as Record<string, unknown>).headers, {
    host: 'roundtrip.example.com',
    'X-Local': 'keep',
  });
});

test('mergeXhttpSettings writes v1.11.4 extra schema and preserves unknown local fields', () => {
  const result = mergeXhttpSettings(
    {
      path: '/old',
      mode: 'auto',
      headers: { Host: 'old.example.com' },
      xPaddingBytes: 'legacy-padding',
      rootExtension: { enabled: true },
      extra: { localExtra: { keep: 'yes' } },
    },
    defaultXhttpFormInput(),
  );

  assert.equal(result.path, '/xhttp');
  assert.equal(result.host, 'xhttp.example.com');
  assert.equal(result.headers, undefined);
  assert.deepEqual(result.rootExtension, { enabled: true });
  assert.deepEqual(result.extra, {
    localExtra: { keep: 'yes' },
    headers: { Host: 'xhttp.example.com' },
    xPaddingBytes: '100-1000',
    noSSEHeader: true,
    scMaxBufferedPosts: 8,
    scMaxEachPostBytes: '1000000',
    scStreamUpServerSecs: '20-80',
    xPaddingObfsMode: true,
    xPaddingKey: 'x_padding',
    xPaddingHeader: 'X-Padding',
    xPaddingPlacement: 'header',
    xPaddingMethod: 'tokenish',
    uplinkHTTPMethod: 'POST',
    sessionIDPlacement: 'header',
    sessionIDKey: 'x_session',
    seqPlacement: 'query',
    seqKey: 'x_seq',
    uplinkDataPlacement: 'header',
    uplinkDataKey: 'x_data',
    uplinkChunkSize: 4096,
    xmux: {
      maxConcurrency: '16-32',
      cMaxReuseTimes: '10',
      hMaxRequestTimes: '600-900',
      hMaxReusableSecs: '1800-3000',
      hKeepAlivePeriod: 30,
    },
  });
});

test('mergeXhttpSettings preserves hidden conditional values while removing only disabled XMUX', () => {
  const input = defaultXhttpFormInput();
  const result = mergeXhttpSettings(
    {
      extra: {
        xPaddingKey: 'old',
        sessionKey: 'old',
        seqKey: 'old',
        uplinkDataKey: 'old',
        xmux: { maxConcurrency: 'old' },
        localExtra: 'keep',
      },
    },
    {
      ...input,
      mode: 'stream-up',
      xPaddingObfsMode: false,
      sessionPlacement: 'path',
      seqPlacement: 'path',
      uplinkDataPlacement: '',
      xmuxEnabled: false,
    },
  );

  const extra = result.extra as Record<string, unknown>;
  assert.equal(extra.xPaddingKey, 'x_padding');
  assert.equal(extra.sessionIDKey, 'x_session');
  assert.equal(extra.sessionKey, undefined);
  assert.equal(extra.seqKey, 'x_seq');
  assert.equal(extra.uplinkDataKey, 'x_data');
  assert.equal(extra.xmux, undefined);
  assert.equal(extra.localExtra, 'keep');
});

test('validateXhttpFormInput rejects unsupported placements and control characters', () => {
  const input = defaultXhttpFormInput();
  assert.equal(validateXhttpFormInput(input), '');
  assert.match(validateXhttpFormInput({ ...input, sessionPlacement: 'body' }), /session placement/);
  assert.match(validateXhttpFormInput({ ...input, path: '/x\nhttp' }), /control characters/);
  assert.equal(validateXhttpFormInput({ ...input, xPaddingPlacement: 'cookie' }), '');
  assert.equal(validateXhttpFormInput({ ...input, xPaddingPlacement: 'query' }), '');
  assert.equal(validateXhttpFormInput({ ...input, uplinkDataPlacement: 'auto' }), '');
  assert.equal(validateXhttpFormInput({ ...input, uplinkDataPlacement: 'cookie' }), '');
  assert.match(
    validateXhttpFormInput({ ...input, uplinkDataPlacement: 'query' }),
    /uplink data placement/,
  );
  assert.match(
    validateXhttpFormInput({ ...input, mode: 'stream-up', uplinkHTTPMethod: 'GET' }),
    /only supported in packet-up mode/,
  );
  assert.match(
    validateXhttpFormInput({ ...input, mode: 'stream-up', uplinkDataPlacement: 'header' }),
    /only supported in packet-up mode/,
  );
  assert.match(
    validateXhttpFormInput({ ...input, xmuxMaxConcurrency: 'abc' }),
    /integer or integer range/,
  );
  assert.match(validateXhttpFormInput({ ...input, xPaddingBytes: '0-100' }), /greater than 0/);
});

test('validateHysteriaQuicFormInput matches Xray-core QUIC boundaries', () => {
  const input = {
    ...HYSTERIA_QUIC_DEFAULTS,
    quicParamsEnabled: true,
    udpHopEnabled: false,
    ports: '',
    interval: '',
  };
  assert.equal(validateHysteriaQuicFormInput(input), '');
  assert.equal(
    validateHysteriaQuicFormInput({
      ...input,
      initStreamReceiveWindow: 0,
      maxStreamReceiveWindow: 16_384,
      initConnectionReceiveWindow: 16_384,
      maxConnectionReceiveWindow: 0,
      maxIdleTimeout: 4,
      maxIncomingStreams: 8,
    }),
    '',
  );
  assert.match(
    validateHysteriaQuicFormInput({ ...input, initStreamReceiveWindow: 16_383 }),
    /at least 16384/,
  );
  assert.match(validateHysteriaQuicFormInput({ ...input, maxIdleTimeout: 3 }), /between 4 and 120/);
  assert.match(
    validateHysteriaQuicFormInput({ ...input, maxIdleTimeout: 121 }),
    /between 4 and 120/,
  );
  assert.match(validateHysteriaQuicFormInput({ ...input, maxIncomingStreams: 7 }), /at least 8/);
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
  assert.match(link.searchParams.get('fm') || '', /"type":"udphop"/);
  assert.match(link.searchParams.get('fm') || '', /"remotePorts":"40000-45000"/);
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
