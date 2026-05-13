import assert from 'node:assert/strict';
import test from 'node:test';

import {
  DNS_PRESET_OPTIONS,
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
  getReverseRows,
  getRoutingRuleRows,
  getRuntimePolicyForm,
  moveArrayItem,
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
