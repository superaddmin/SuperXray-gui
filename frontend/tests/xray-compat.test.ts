import assert from 'node:assert/strict';
import test from 'node:test';

import {
  DNS_PRESET_OPTIONS,
  AI_RESIDENTIAL_DOMAINS,
  applyAiResidentialRouting,
  applyDnsPolicyForm,
  applyDnsPreset,
  applyObservatoryForm,
  applyRuntimePolicyForm,
  deleteBalancerAt,
  deleteDnsServerAt,
  deleteFakeDnsAt,
  deleteOutboundAt,
  deleteReverseAt,
  deleteRoutingRuleAt,
  getBalancerRows,
  getDnsPolicyForm,
  getDnsServerRows,
  getFakeDnsRows,
  getObservatoryForm,
  getOutboundRows,
  getResidentialIpRows,
  getReverseRows,
  getRoutingRuleRows,
  getRuntimePolicyForm,
  moveArrayItem,
  upsertResidentialIpOutbound,
  upsertBalancer,
  upsertDnsServer,
  upsertFakeDns,
  upsertOutbound,
  upsertReverse,
  upsertRoutingRule,
} from '../src/utils/xrayCompat.ts';

test('upsertOutbound adds and updates outbound rows in template settings', () => {
  const template = { outbounds: [{ tag: 'direct', protocol: 'freedom' }] };
  const added = upsertOutbound(template, null, {
    tag: 'proxy',
    protocol: 'socks',
    sendThrough: '',
    settingsJson: '{\n  "servers": []\n}',
    streamSettingsJson: '{}',
    proxySettingsJson: '{}',
    muxJson: '{}',
  });
  assert.deepEqual(getOutboundRows(added).map((row) => row.tag), ['direct', 'proxy']);

  const updated = upsertOutbound(added, 1, {
    tag: 'proxy-2',
    protocol: 'socks',
    sendThrough: '0.0.0.0',
    settingsJson: '{\n  "servers": []\n}',
    streamSettingsJson: '{}',
    proxySettingsJson: '{}',
    muxJson: '{}',
  });
  assert.equal(getOutboundRows(updated)[1].tag, 'proxy-2');
});

test('routing helpers manage rules without destroying existing routing config', () => {
  const template = { routing: { domainStrategy: 'AsIs', rules: [] } };
  const next = upsertRoutingRule(template, null, {
    type: 'field',
    outboundTag: 'proxy',
    balancerTag: '',
    domainText: 'geosite:google',
    ipText: '',
    sourceText: '',
    userText: '',
    inboundTagText: '',
    protocolText: '',
    attrsJson: '{}',
    networkText: 'tcp',
    portText: '',
    sourcePortText: '',
  });
  assert.equal(getRoutingRuleRows(next).length, 1);
  assert.equal(next.routing.domainStrategy, 'AsIs');

  const deleted = deleteRoutingRuleAt(next, 0);
  assert.equal(getRoutingRuleRows(deleted).length, 0);
});

test('dns helpers create dns section on demand and preserve query strategy', () => {
  const template = { dns: { queryStrategy: 'UseIP', servers: [] } };
  const next = upsertDnsServer(template, null, {
    address: 'https://1.1.1.1/dns-query',
    domainsText: 'geosite:openai',
    expectIPsText: '1.1.1.1',
    skipFallback: false,
    clientIP: '',
    queryStrategy: '',
  });
  assert.equal(getDnsServerRows(next).length, 1);
  assert.equal(next.dns.queryStrategy, 'UseIP');

  const deleted = deleteDnsServerAt(next, 0);
  assert.equal(getDnsServerRows(deleted).length, 0);
});

test('fakedns helpers manage top-level fakedns array', () => {
  const template = {};
  const next = upsertFakeDns(template, null, {
    ipPool: '198.18.0.0/15',
    poolSize: 65535,
  });
  assert.equal(getFakeDnsRows(next).length, 1);

  const deleted = deleteFakeDnsAt(next, 0);
  assert.equal(getFakeDnsRows(deleted).length, 0);
});

test('moveArrayItem reorders arrays predictably', () => {
  assert.deepEqual(moveArrayItem(['a', 'b', 'c'], 2, 0), ['c', 'a', 'b']);
});

test('deleteOutboundAt removes selected outbound only', () => {
  const template = {
    outbounds: [
      { tag: 'direct', protocol: 'freedom' },
      { tag: 'proxy', protocol: 'socks' },
    ],
  };
  const next = deleteOutboundAt(template, 0);
  assert.deepEqual(getOutboundRows(next).map((row) => row.tag), ['proxy']);
});

test('upsertBalancer adds and updates routing balancers and preserves balancer tags', () => {
  const template = {
    routing: {
      rules: [{ type: 'field', balancerTag: 'old-balancer' }],
      balancers: [],
    },
  };
  const added = upsertBalancer(template, null, {
    tag: 'old-balancer',
    strategy: 'leastPing',
    selectorText: 'proxy-a\nproxy-b',
    fallbackTag: 'direct',
  });
  assert.equal(getBalancerRows(added).length, 1);
  assert.equal(getBalancerRows(added)[0].strategy, 'leastPing');

  const updated = upsertBalancer(added, 0, {
    tag: 'new-balancer',
    strategy: 'random',
    selectorText: 'proxy-a',
    fallbackTag: '',
  });
  assert.equal(getBalancerRows(updated)[0].tag, 'new-balancer');
  assert.equal(getRoutingRuleRows(updated)[0].balancerTag, 'new-balancer');

  const deleted = deleteBalancerAt(updated, 0);
  assert.equal(getBalancerRows(deleted).length, 0);
  assert.equal(getRoutingRuleRows(deleted)[0].balancerTag, '');
});

test('upsertReverse adds reverse entry and related routing rules', () => {
  const template = { routing: { rules: [] } };
  const added = upsertReverse(template, null, {
    type: 'bridge',
    tag: 'reverse-0',
    domain: 'reverse.example',
    bridgeOutboundTag: 'proxy',
    bridgeReplyOutboundTag: 'direct',
    portalInboundTagsText: '',
  });
  assert.equal(getReverseRows(added).length, 1);
  assert.equal(getRoutingRuleRows(added).length, 2);
  assert.equal(getReverseRows(added)[0].type, 'bridge');

  const updated = upsertReverse(added, 0, {
    type: 'portal',
    tag: 'portal-0',
    domain: 'portal.example',
    bridgeOutboundTag: '',
    bridgeReplyOutboundTag: '',
    portalInboundTagsText: 'api-in\napi-out',
  });
  assert.equal(getReverseRows(updated)[0].type, 'portal');
  assert.equal(getRoutingRuleRows(updated).length, 2);
  assert.equal(getRoutingRuleRows(updated)[0].outboundTag, 'portal-0');

  const deleted = deleteReverseAt(updated, 0);
  assert.equal(getReverseRows(deleted).length, 0);
  assert.equal(getRoutingRuleRows(deleted).length, 0);
});

test('dns preset options are available and can replace dns.servers', () => {
  assert.ok(DNS_PRESET_OPTIONS.some((item) => item.name === 'Cloudflare DNS'));
  const next = applyDnsPreset({}, DNS_PRESET_OPTIONS[0].data);
  assert.deepEqual((next.dns as { servers: string[] }).servers, DNS_PRESET_OPTIONS[0].data);
});

test('dns policy form reads and writes top-level dns switches', () => {
  const initial = getDnsPolicyForm({});
  assert.equal(initial.enableDNS, false);

  const next = applyDnsPolicyForm(
    {},
    {
      enableDNS: true,
      dnsTag: 'dns-in',
      dnsClientIp: '1.1.1.1',
      dnsStrategy: 'UseIPv4',
      dnsDisableCache: true,
      dnsDisableFallback: false,
      dnsDisableFallbackIfMatch: true,
      dnsEnableParallelQuery: true,
      dnsUseSystemHosts: false,
    },
  );

  const form = getDnsPolicyForm(next);
  assert.equal(form.enableDNS, true);
  assert.equal(form.dnsTag, 'dns-in');
  assert.equal(form.dnsStrategy, 'UseIPv4');
  assert.equal(form.dnsDisableCache, true);
  assert.equal(form.dnsEnableParallelQuery, true);
});

test('runtime policy form updates routing, log, policy.system, and direct freedom strategy', () => {
  const next = applyRuntimePolicyForm(
    {},
    {
      freedomStrategy: 'UseIPv6',
      routingStrategy: 'IPIfNonMatch',
      logLevel: 'debug',
      accessLog: './access.log',
      errorLog: './error.log',
      dnsLog: true,
      maskAddressLog: 'half',
      statsInboundUplink: true,
      statsInboundDownlink: true,
      statsOutboundUplink: false,
      statsOutboundDownlink: true,
    },
  );

  const form = getRuntimePolicyForm(next);
  assert.equal(form.freedomStrategy, 'UseIPv6');
  assert.equal(form.routingStrategy, 'IPIfNonMatch');
  assert.equal(form.logLevel, 'debug');
  assert.equal(form.dnsLog, true);
  assert.equal(form.statsInboundUplink, true);
  assert.equal(form.statsOutboundDownlink, true);
});

test('observatory form reads and writes observatory json blocks', () => {
  const next = applyObservatoryForm(
    {},
    {
      observatoryEnable: true,
      observatoryJson: '{\n  "subjectSelector": ["proxy-a"],\n  "probeURL": "https://www.google.com/generate_204"\n}',
      burstObservatoryEnable: true,
      burstObservatoryJson: '{\n  "subjectSelector": ["proxy-b"],\n  "pingConfig": { "destination": "https://www.google.com/generate_204" }\n}',
    },
  );
  const form = getObservatoryForm(next);
  assert.equal(form.observatoryEnable, true);
  assert.match(form.observatoryJson, /proxy-a/);
  assert.equal(form.burstObservatoryEnable, true);
  assert.match(form.burstObservatoryJson, /proxy-b/);
});

test('residential ip helper adds socks outbounds without exposing credentials in rows', () => {
  const next = upsertResidentialIpOutbound({}, null, {
    tag: 'residential-us-1',
    protocol: 'socks',
    server: '203.0.113.10',
    port: 1086,
    username: 'user-a',
    password: 'secret-a',
  });

  const rows = getResidentialIpRows(next);
  assert.equal(rows.length, 1);
  assert.equal(rows[0].tag, 'residential-us-1');
  assert.equal(rows[0].address, '203.0.113.10:1086');
  assert.doesNotMatch(JSON.stringify(rows[0]), /secret-a/);

  const outbound = getOutboundRows(next)[0];
  assert.equal(outbound.protocol, 'socks');
  assert.deepEqual((outbound.settings as { servers: unknown[] }).servers, [
    {
      address: '203.0.113.10',
      port: 1086,
      users: [{ user: 'user-a', pass: 'secret-a' }],
    },
  ]);
});

test('ai residential routing uses a balancer and keeps google domains narrowed to gemini APIs', () => {
  const template = upsertResidentialIpOutbound({}, null, {
    tag: 'residential-us-1',
    protocol: 'socks',
    server: '203.0.113.10',
    port: 1086,
    username: 'user-a',
    password: 'secret-a',
  });
  const next = applyAiResidentialRouting(
    upsertResidentialIpOutbound(template, null, {
      tag: 'residential-us-2',
      protocol: 'socks',
      server: '203.0.113.11',
      port: 1086,
      username: '',
      password: '',
    }),
  );

  const routing = next.routing as {
    balancers: Array<{ tag: string; selector: string[]; fallbackTag?: string }>;
    rules: Array<{ balancerTag?: string; outboundTag?: string; network?: string; domain?: string[] }>;
  };
  assert.deepEqual(routing.balancers[0], {
    tag: 'ai-residential',
    selector: ['residential-us-1', 'residential-us-2'],
    strategy: { type: 'random' },
  });
  assert.equal(routing.rules[0].balancerTag, 'ai-residential');
  assert.equal(routing.rules[0].network, 'tcp');
  assert.equal(routing.rules[1].outboundTag, 'blocked');
  assert.equal(routing.rules[1].network, 'udp');
  assert.ok(routing.rules[0].domain?.includes('domain:gemini.google.com'));
  assert.ok(routing.rules[0].domain?.includes('domain:generativelanguage.googleapis.com'));
  assert.ok(!routing.rules[0].domain?.includes('domain:google.com'));
  assert.ok(!routing.rules[0].domain?.includes('domain:googleapis.com'));
  assert.deepEqual(
    AI_RESIDENTIAL_DOMAINS.filter((domain) => domain.includes('google')),
    [
      'domain:aistudio.google.com',
      'domain:generativelanguage.googleapis.com',
      'domain:makersuite.google.com',
      'domain:gemini.google.com',
    ],
  );
});

test('ai residential routing replaces previous broad google rules for residential outbounds', () => {
  const template = {
    outbounds: [
      { tag: 'direct', protocol: 'freedom' },
      {
        tag: 'us-residential-socks',
        protocol: 'socks',
        settings: { servers: [{ address: '203.0.113.10', port: 1086 }] },
      },
      { tag: 'blocked', protocol: 'blackhole' },
    ],
    routing: {
      domainStrategy: 'AsIs',
      rules: [
        {
          type: 'field',
          outboundTag: 'us-residential-socks',
          network: 'tcp',
          domain: ['domain:openai.com', 'domain:google.com', 'domain:googleapis.com'],
        },
        {
          type: 'field',
          outboundTag: 'blocked',
          network: 'udp',
          domain: ['domain:openai.com', 'domain:google.com', 'domain:googleapis.com'],
        },
      ],
    },
  };

  const next = applyAiResidentialRouting(template);
  const rules = (next.routing as { rules: Array<{ domain?: string[] }> }).rules;
  assert.equal(rules.length, 2);
  assert.ok(rules.every((rule) => !rule.domain?.includes('domain:google.com')));
  assert.ok(rules.every((rule) => !rule.domain?.includes('domain:googleapis.com')));
  assert.ok(rules[0].domain?.includes('domain:gemini.google.com'));
});
