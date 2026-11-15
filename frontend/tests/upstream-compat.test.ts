import assert from 'node:assert/strict';
import test from 'node:test';
import {
  buildInboundShareLinks,
  normalizeFinalmaskForXray,
  normalizeInboundStreamCompatibility,
  validateFinalmaskForXray,
} from '../src/utils/inboundCompat.ts';

test('upstream Hysteria sharing removes source IP filters without mutating storage', () => {
  const stream = {
    network: 'hysteria',
    security: 'tls',
    finalmask: {
      tcp: [{ type: 'filter', settings: { ips: ['192.0.2.0/24'] } }],
      udp: [
        { type: 'filter', settings: { ips: ['198.51.100.0/24'] } },
        { type: 'noise', settings: { script: 'noise.lua' } },
      ],
      vendor: true,
    },
  };
  const before = JSON.stringify(stream);
  const links = buildInboundShareLinks({
    protocol: 'hysteria',
    listen: '203.0.113.1',
    port: 443,
    settings: JSON.stringify({ clients: [{ email: 'fixture', auth: 'fixture-auth' }] }),
    streamSettings: before,
  } as never);
  const fm = JSON.parse(new URL(links[0]).searchParams.get('fm') || '{}');
  assert.equal(fm.tcp, undefined);
  assert.equal(fm.udp.length, 1);
  assert.equal(fm.udp[0].settings.script, 'noise.lua');
  assert.equal(fm.vendor, true);
  assert.equal(JSON.stringify(stream), before);
});

test('upstream TLS custom ciphers are shared with an explicit fingerprint policy', () => {
  for (const fingerprint of ['', 'chrome']) {
    const links = buildInboundShareLinks({
      protocol: 'vless',
      listen: '203.0.113.1',
      port: 443,
      settings: JSON.stringify({
        clients: [{ id: '00000000-0000-4000-8000-000000000001', email: 'fixture' }],
      }),
      streamSettings: JSON.stringify({
        network: 'tcp',
        security: 'tls',
        tlsSettings: {
          cipherSuites: 'TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256',
          settings: { fingerprint },
        },
      }),
    } as never);
    const params = new URL(links[0]).searchParams;
    assert.equal(params.get('cs'), 'TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256');
    assert.equal(params.get('fp'), fingerprint || 'unsafe');
  }
});

test('upstream VMess and external TLS overrides keep each endpoint isolated', () => {
  for (const protocol of ['vmess', 'vless', 'trojan', 'shadowsocks']) {
    const links = buildInboundShareLinks({
      protocol,
      listen: '203.0.113.1',
      port: 443,
      settings: JSON.stringify({
        method: 'chacha20-ietf-poly1305',
        clients: [
          {
            id: '00000000-0000-4000-8000-000000000001',
            email: 'fixture',
            password: 'fixture-pass',
          },
        ],
      }),
      streamSettings: JSON.stringify({
        network: 'tcp',
        security: 'tls',
        tlsSettings: { cipherSuites: 'base-suite', settings: { fingerprint: 'chrome' } },
        externalProxy: [
          { dest: 'first.example.test', port: 443, cipherSuites: 'host-suite', fingerprint: '' },
          { dest: 'second.example.test', port: 443 },
          { dest: 'plain.example.test', port: 80, forceTls: 'none' },
        ],
      }),
    } as never);
    const fields = links.map((link) =>
      protocol === 'vmess'
        ? JSON.parse(Buffer.from(link.slice('vmess://'.length), 'base64').toString())
        : Object.fromEntries(new URL(link).searchParams),
    );
    assert.equal(fields[0].cs, 'host-suite', protocol);
    assert.equal(fields[0].fp, 'unsafe', protocol);
    assert.equal(fields[1].cs, 'base-suite', protocol);
    assert.equal(fields[1].fp, 'chrome', protocol);
    assert.equal(fields[2].cs, undefined, protocol);
    assert.equal(fields[2].fp, undefined, protocol);
  }
});

test('upstream XDRIVE service alias preserves transport extensions', () => {
  const input = { network: 'xdrive', xdriveSettings: { service: 'Google Drive', vendor: true } };
  const output = normalizeInboundStreamCompatibility(input);
  assert.deepEqual(output.xdriveSettings, { service: 'google-drive', vendor: true });
  assert.equal(input.xdriveSettings.service, 'Google Drive');
});

test('upstream KCP mask conversion preserves extensions and rejects conflicts atomically', () => {
  const input = {
    finalmask: {
      udp: [
        { type: 'mkcp-aes128gcm', settings: { password: 'fixture-seed', vendor: true } },
        { type: 'header-dns', settings: { domain: 'fixture.test' } },
      ],
    },
  };
  const result = normalizeFinalmaskForXray(input);
  assert.deepEqual((result.finalmask as typeof input.finalmask).udp, [
    { type: 'mkcp-legacy', settings: { value: 'fixture-seed', vendor: true } },
    { type: 'mkcp-legacy', settings: { header: 'dns', value: 'fixture.test' } },
  ]);
  assert.equal(input.finalmask.udp[0].type, 'mkcp-aes128gcm');
  assert.deepEqual(normalizeFinalmaskForXray(result), result);
  const invalid = {
    finalmask: { udp: [{ type: 'mkcp-aes128gcm', settings: { password: 'old', value: 'new' } }] },
  };
  assert.deepEqual(normalizeFinalmaskForXray(invalid), invalid);
});

test('upstream empty AES password never migrates to a plaintext KCP mask', () => {
  for (const settings of [{}, { password: '' }, { value: 'new' }]) {
    const input = { finalmask: { udp: [{ type: 'mkcp-aes128gcm', settings }] } };
    assert.match(validateFinalmaskForXray(input), /password/i);
    assert.deepEqual(normalizeFinalmaskForXray(input), input);
  }
});
