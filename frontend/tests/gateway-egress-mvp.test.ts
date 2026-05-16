import assert from 'node:assert/strict';
import test from 'node:test';

import {
  GATEWAY_EGRESS_MVP_PROFILES,
  buildGatewayEgressManifestCsv,
  buildGatewayEgressMvpPreview,
  mergeGatewayEgressMvpConfig,
} from '../src/utils/gatewayEgressMvp.ts';

test('gateway egress mvp profiles keep deterministic ports', () => {
  assert.deepEqual(
    GATEWAY_EGRESS_MVP_PROFILES.map((profile) => [profile.key, profile.port, profile.platform]),
    [
      ['openai-us-primary', 11801, 'openai'],
      ['anthropic-us-primary', 11802, 'anthropic'],
      ['gemini-us-primary', 11803, 'gemini'],
      ['region-us-primary', 11901, ''],
      ['region-jp-primary', 11981, ''],
    ],
  );
});

test('gateway egress mvp merge defaults to same-network loopback hosts', () => {
  const merged = mergeGatewayEgressMvpConfig({
    inbounds: [{ tag: 'existing-in', listen: '127.0.0.1', port: 10000, protocol: 'socks' }],
    outbounds: [{ tag: 'direct', protocol: 'freedom' }],
    routing: { domainStrategy: 'AsIs', rules: [] },
  });

  const inbounds = merged.inbounds as Array<{ listen: string; port: number; tag: string }>;
  const rules = (merged.routing as { rules: Array<{ outboundTag?: string; domain?: string[] }> })
    .rules;

  assert.ok(inbounds.some((inbound) => inbound.tag === 'gateway-openai-us-primary'));
  assert.ok(
    inbounds
      .filter((inbound) => inbound.tag.startsWith('gateway-'))
      .every((inbound) => inbound.listen === '127.0.0.1'),
  );
  assert.ok(rules.some((rule) => rule.domain?.includes('domain:api.openai.com')));
  assert.ok(rules.some((rule) => rule.domain?.includes('domain:api.anthropic.com')));
  assert.ok(rules.some((rule) => rule.domain?.includes('domain:generativelanguage.googleapis.com')));
  assert.equal(rules.at(-1)?.outboundTag, 'blocked');
});

test('gateway egress mvp manifest can use docker reachable host separately from listen host', () => {
  const csv = buildGatewayEgressManifestCsv({
    listenHost: '127.0.0.1',
    manifestHost: 'host.docker.internal',
    strategyLabel: 'docker-host-gateway',
  });

  assert.match(
    csv,
    /^name,protocol,host,port,platform,region_code,expected_country_code,egress_group,health_status,notes/m,
  );
  assert.match(
    csv,
    /openai-us-primary,socks5h,host\.docker\.internal,11801,openai,US,US,openai-egress,manual-check,OpenAI MVP local exit \(docker-host-gateway\)/,
  );
  assert.doesNotMatch(csv, /openai-us-primary,socks5h,127\.0\.0\.1,11801/);
});

test('gateway egress mvp merge writes reviewed listen host into generated inbounds', () => {
  const merged = mergeGatewayEgressMvpConfig(
    { inbounds: [], outbounds: [], routing: { rules: [] } },
    {
      listenHost: '172.17.0.1',
      manifestHost: 'host.docker.internal',
      strategyLabel: 'docker-host-gateway',
    },
  );

  const inbounds = merged.inbounds as Array<{ listen: string; settings?: { ip?: string } }>;
  assert.ok(inbounds.every((inbound) => inbound.listen === '172.17.0.1'));
  assert.ok(inbounds.every((inbound) => inbound.settings?.ip === '172.17.0.1'));
});

test('gateway egress mvp rejects wildcard or empty hosts', () => {
  assert.throws(
    () => buildGatewayEgressManifestCsv({ listenHost: '127.0.0.1', manifestHost: '0.0.0.0' }),
    /manifestHost cannot be a wildcard host/,
  );
  assert.throws(
    () => mergeGatewayEgressMvpConfig({}, { listenHost: ' ', manifestHost: '127.0.0.1' }),
    /listenHost is required/,
  );
});

test('gateway egress mvp preserves existing real outbound definitions', () => {
  const merged = mergeGatewayEgressMvpConfig({
    outbounds: [
      {
        tag: 'openai-egress',
        protocol: 'socks',
        settings: { servers: [{ address: '10.0.0.10', port: 1080 }] },
      },
    ],
  });

  const outbounds = merged.outbounds as Array<{ protocol: string; tag: string }>;
  assert.equal(outbounds.find((outbound) => outbound.tag === 'openai-egress')?.protocol, 'socks');
});

test('gateway egress mvp preview includes network strategy', () => {
  const preview = buildGatewayEgressMvpPreview({
    listenHost: '127.0.0.1',
    manifestHost: 'host.docker.internal',
    strategyLabel: 'docker-host-gateway',
  });

  assert.equal(preview.profileCount, 5);
  assert.deepEqual(preview.ports, [11801, 11802, 11803, 11901, 11981]);
  assert.deepEqual(preview.platforms, ['openai', 'anthropic', 'gemini']);
  assert.deepEqual(preview.regions, ['US', 'JP']);
  assert.equal(preview.listenHost, '127.0.0.1');
  assert.equal(preview.manifestHost, 'host.docker.internal');
  assert.equal(preview.strategyLabel, 'docker-host-gateway');
});
