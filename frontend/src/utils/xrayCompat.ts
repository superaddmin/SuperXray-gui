import type { JsonObject, JsonValue } from '@/types/api';

export interface OutboundEditorForm {
  tag: string;
  protocol: string;
  sendThrough: string;
  settingsJson: string;
  streamSettingsJson: string;
  proxySettingsJson: string;
  muxJson: string;
}

export interface RoutingRuleEditorForm {
  type: string;
  outboundTag: string;
  balancerTag: string;
  domainText: string;
  ipText: string;
  sourceText: string;
  userText: string;
  inboundTagText: string;
  protocolText: string;
  attrsJson: string;
  networkText: string;
  portText: string;
  sourcePortText: string;
}

export interface DnsServerEditorForm {
  address: string;
  domainsText: string;
  expectIPsText: string;
  skipFallback: boolean;
  clientIP: string;
  queryStrategy: string;
}

export interface FakeDnsEditorForm {
  ipPool: string;
  poolSize: number;
}

export interface BalancerEditorForm {
  tag: string;
  strategy: string;
  selectorText: string;
  fallbackTag: string;
}

export interface ReverseEditorForm {
  type: 'bridge' | 'portal';
  tag: string;
  domain: string;
  bridgeOutboundTag: string;
  bridgeReplyOutboundTag: string;
  portalInboundTagsText: string;
}

export interface DnsPresetOption {
  name: string;
  family: boolean;
  data: string[];
}

export interface DnsPolicyForm {
  enableDNS: boolean;
  dnsTag: string;
  dnsClientIp: string;
  dnsStrategy: string;
  dnsDisableCache: boolean;
  dnsDisableFallback: boolean;
  dnsDisableFallbackIfMatch: boolean;
  dnsEnableParallelQuery: boolean;
  dnsUseSystemHosts: boolean;
}

export interface RuntimePolicyForm {
  freedomStrategy: string;
  routingStrategy: string;
  logLevel: string;
  accessLog: string;
  errorLog: string;
  dnsLog: boolean;
  maskAddressLog: string;
  statsInboundUplink: boolean;
  statsInboundDownlink: boolean;
  statsOutboundUplink: boolean;
  statsOutboundDownlink: boolean;
}

export interface ObservatoryForm {
  observatoryEnable: boolean;
  observatoryJson: string;
  burstObservatoryEnable: boolean;
  burstObservatoryJson: string;
}

export interface OutboundRow extends JsonObject {
  key: number;
  tag: string;
  protocol: string;
  sendThrough: string;
  address: string;
}

export interface RoutingRuleRow extends JsonObject {
  key: number;
  type: string;
  outboundTag: string;
  balancerTag: string;
  domainText: string;
  ipText: string;
  sourceText: string;
  userText: string;
  inboundTagText: string;
  protocolText: string;
  networkText: string;
  portText: string;
  sourcePortText: string;
}

export interface DnsServerRow extends JsonObject {
  key: number;
  address: string;
  domainsText: string;
  expectIPsText: string;
  skipFallback: boolean;
  clientIP: string;
  queryStrategy: string;
}

export interface FakeDnsRow extends JsonObject {
  key: number;
  ipPool: string;
  poolSize: number;
}

export interface BalancerRow extends JsonObject {
  key: number;
  tag: string;
  strategy: string;
  selectorText: string;
  fallbackTag: string;
}

export interface ReverseRow extends JsonObject {
  key: number;
  type: 'bridge' | 'portal';
  tag: string;
  domain: string;
  bridgeOutboundTag: string;
  bridgeReplyOutboundTag: string;
  portalInboundTagsText: string;
}

export const DNS_PRESET_OPTIONS: DnsPresetOption[] = [
  {
    name: 'Google DNS',
    family: false,
    data: ['8.8.8.8', '8.8.4.4', '2001:4860:4860::8888', '2001:4860:4860::8844'],
  },
  {
    name: 'Cloudflare DNS',
    family: false,
    data: ['1.1.1.1', '1.0.0.1', '2606:4700:4700::1111', '2606:4700:4700::1001'],
  },
  {
    name: 'Adguard DNS',
    family: false,
    data: ['94.140.14.14', '94.140.15.15', '2a10:50c0::ad1:ff', '2a10:50c0::ad2:ff'],
  },
  {
    name: 'Adguard Family DNS',
    family: true,
    data: ['94.140.14.14', '94.140.15.15', '2a10:50c0::ad1:ff', '2a10:50c0::ad2:ff'],
  },
  {
    name: 'Cloudflare Family DNS',
    family: true,
    data: ['1.1.1.3', '1.0.0.3', '2606:4700:4700::1113', '2606:4700:4700::1003'],
  },
];

export function moveArrayItem<T>(items: T[], from: number, to: number): T[] {
  const next = items.slice();
  if (from < 0 || to < 0 || from >= next.length || to >= next.length) {
    return next;
  }
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}

export function getOutboundRows(config: JsonValue | null | undefined): OutboundRow[] {
  const template = asObject(config);
  const outbounds = asObjectArray(template.outbounds);
  return outbounds.map((outbound, index) => ({
    ...outbound,
    key: index,
    tag: stringValue(outbound.tag) || `outbound-${index + 1}`,
    protocol: stringValue(outbound.protocol),
    sendThrough: stringValue(outbound.sendThrough),
    address: resolveOutboundAddress(outbound),
  }));
}

export function upsertOutbound(
  config: JsonValue | null | undefined,
  index: number | null,
  form: OutboundEditorForm,
): JsonObject {
  const template = cloneObject(config);
  const outbounds = asObjectArray(template.outbounds);
  const outbound: JsonObject = {
    tag: form.tag.trim(),
    protocol: form.protocol.trim(),
  };
  setObjectField(outbound, 'settings', parseJsonObject(form.settingsJson));
  setObjectField(outbound, 'streamSettings', parseJsonObject(form.streamSettingsJson));
  setObjectField(outbound, 'proxySettings', parseJsonObject(form.proxySettingsJson));
  setObjectField(outbound, 'mux', parseJsonObject(form.muxJson));
  setStringField(outbound, 'sendThrough', form.sendThrough);

  if (index === null || index < 0 || index >= outbounds.length) {
    outbounds.push(outbound);
  } else {
    outbounds[index] = outbound;
  }

  template.outbounds = outbounds;
  return template;
}

export function deleteOutboundAt(config: JsonValue | null | undefined, index: number): JsonObject {
  const template = cloneObject(config);
  const outbounds = asObjectArray(template.outbounds);
  outbounds.splice(index, 1);
  template.outbounds = outbounds;
  return template;
}

export function getRoutingRuleRows(config: JsonValue | null | undefined): RoutingRuleRow[] {
  const rules = getRoutingRules(config);
  return rules.map((rule, index) => ({
    ...rule,
    key: index,
    type: stringValue(rule.type) || 'field',
    outboundTag: stringValue(rule.outboundTag),
    balancerTag: stringValue(rule.balancerTag),
    domainText: joinStringArray(rule.domain),
    ipText: joinStringArray(rule.ip),
    sourceText: joinStringArray(rule.source),
    userText: joinStringArray(rule.user),
    inboundTagText: joinStringArray(rule.inboundTag),
    protocolText: joinStringArray(rule.protocol),
    networkText: joinStringArray(rule.network),
    portText: formatScalar(rule.port),
    sourcePortText: formatScalar(rule.sourcePort),
  }));
}

export function upsertRoutingRule(
  config: JsonValue | null | undefined,
  index: number | null,
  form: RoutingRuleEditorForm,
): JsonObject {
  const template = cloneObject(config);
  const routing = asObject(template.routing);
  const rules = asObjectArray(routing.rules);
  const rule: JsonObject = {
    type: form.type.trim() || 'field',
  };

  setStringField(rule, 'outboundTag', form.outboundTag);
  setStringField(rule, 'balancerTag', form.balancerTag);
  setArrayField(rule, 'domain', form.domainText);
  setArrayField(rule, 'ip', form.ipText);
  setArrayField(rule, 'source', form.sourceText);
  setArrayField(rule, 'user', form.userText);
  setArrayField(rule, 'inboundTag', form.inboundTagText);
  setArrayField(rule, 'protocol', form.protocolText);
  setArrayField(rule, 'network', form.networkText);
  setScalarField(rule, 'port', form.portText);
  setScalarField(rule, 'sourcePort', form.sourcePortText);
  setObjectField(rule, 'attrs', parseJsonObject(form.attrsJson));

  if (index === null || index < 0 || index >= rules.length) {
    rules.push(rule);
  } else {
    rules[index] = rule;
  }

  routing.rules = rules;
  template.routing = routing;
  return template;
}

export function deleteRoutingRuleAt(
  config: JsonValue | null | undefined,
  index: number,
): JsonObject {
  const template = cloneObject(config);
  const routing = asObject(template.routing);
  const rules = asObjectArray(routing.rules);
  rules.splice(index, 1);
  routing.rules = rules;
  template.routing = routing;
  return template;
}

export function getDnsServerRows(config: JsonValue | null | undefined): DnsServerRow[] {
  const template = asObject(config);
  const dns = asObject(template.dns);
  const servers = arrayValue(dns.servers);
  return servers.map((server, index) => {
    if (typeof server === 'string') {
      return {
        key: index,
        address: server,
        domainsText: '',
        expectIPsText: '',
        skipFallback: false,
        clientIP: '',
        queryStrategy: '',
      };
    }
    const row = asObject(server);
    return {
      ...row,
      key: index,
      address: stringValue(row.address),
      domainsText: joinStringArray(row.domains),
      expectIPsText: joinStringArray(row.expectIPs),
      skipFallback: Boolean(row.skipFallback),
      clientIP: stringValue(row.clientIP),
      queryStrategy: stringValue(row.queryStrategy),
    };
  });
}

export function upsertDnsServer(
  config: JsonValue | null | undefined,
  index: number | null,
  form: DnsServerEditorForm,
): JsonObject {
  const template = cloneObject(config);
  const dns = asObject(template.dns);
  const servers = arrayValue(dns.servers).map((server) =>
    typeof server === 'string' ? server : asObject(server),
  );
  const server: JsonObject = {
    address: form.address.trim(),
  };
  setArrayField(server, 'domains', form.domainsText);
  setArrayField(server, 'expectIPs', form.expectIPsText);
  if (form.skipFallback) {
    server.skipFallback = true;
  }
  setStringField(server, 'clientIP', form.clientIP);
  setStringField(server, 'queryStrategy', form.queryStrategy);

  if (index === null || index < 0 || index >= servers.length) {
    servers.push(server);
  } else {
    servers[index] = server;
  }

  dns.servers = servers as JsonValue[];
  if (!dns.queryStrategy) {
    dns.queryStrategy = 'UseIP';
  }
  template.dns = dns;
  return template;
}

export function deleteDnsServerAt(config: JsonValue | null | undefined, index: number): JsonObject {
  const template = cloneObject(config);
  const dns = asObject(template.dns);
  const servers = arrayValue(dns.servers);
  servers.splice(index, 1);
  dns.servers = servers;
  template.dns = dns;
  return template;
}

export function getFakeDnsRows(config: JsonValue | null | undefined): FakeDnsRow[] {
  const template = asObject(config);
  const rows = arrayValue(template.fakedns).map((item) => asObject(item));
  return rows.map((item, index) => ({
    ...item,
    key: index,
    ipPool: stringValue(item.ipPool),
    poolSize: numberValue(item.poolSize),
  }));
}

export function getBalancerRows(config: JsonValue | null | undefined): BalancerRow[] {
  const template = asObject(config);
  const routing = asObject(template.routing);
  const rows = asObjectArray(routing.balancers);
  return rows.map((item, index) => ({
    ...item,
    key: index,
    tag: stringValue(item.tag),
    strategy: stringValue(asObject(item.strategy).type) || 'random',
    selectorText: joinStringArray(item.selector),
    fallbackTag: stringValue(item.fallbackTag),
  }));
}

export function getReverseRows(config: JsonValue | null | undefined): ReverseRow[] {
  const template = asObject(config);
  const reverse = asObject(template.reverse);
  const routingRules = getRoutingRules(config);
  const rows: ReverseRow[] = [];

  asObjectArray(reverse.bridges).forEach((item, index) => {
    const tag = stringValue(item.tag);
    const relatedRules = routingRules.filter((rule) => joinStringArray(rule.inboundTag) === tag);
    rows.push({
      ...item,
      key: rows.length + index,
      type: 'bridge',
      tag,
      domain: stringValue(item.domain),
      bridgeOutboundTag: stringValue(relatedRules[0]?.outboundTag),
      bridgeReplyOutboundTag: stringValue(relatedRules[1]?.outboundTag),
      portalInboundTagsText: '',
    });
  });

  asObjectArray(reverse.portals).forEach((item, index) => {
    const tag = stringValue(item.tag);
    const relatedRules = routingRules.filter((rule) => stringValue(rule.outboundTag) === tag);
    rows.push({
      ...item,
      key: rows.length + index,
      type: 'portal',
      tag,
      domain: stringValue(item.domain),
      bridgeOutboundTag: '',
      bridgeReplyOutboundTag: '',
      portalInboundTagsText: joinStringArray(relatedRules[0]?.inboundTag),
    });
  });

  return rows;
}

export function upsertFakeDns(
  config: JsonValue | null | undefined,
  index: number | null,
  form: FakeDnsEditorForm,
): JsonObject {
  const template = cloneObject(config);
  const items = arrayValue(template.fakedns).map((item) => asObject(item));
  const fakeDns: JsonObject = {
    ipPool: form.ipPool.trim(),
    poolSize: Math.max(0, Math.trunc(form.poolSize || 0)),
  };
  if (index === null || index < 0 || index >= items.length) {
    items.push(fakeDns);
  } else {
    items[index] = fakeDns;
  }
  template.fakedns = items;
  return template;
}

export function deleteFakeDnsAt(config: JsonValue | null | undefined, index: number): JsonObject {
  const template = cloneObject(config);
  const items = arrayValue(template.fakedns);
  items.splice(index, 1);
  template.fakedns = items;
  return template;
}

export function upsertBalancer(
  config: JsonValue | null | undefined,
  index: number | null,
  form: BalancerEditorForm,
): JsonObject {
  const template = cloneObject(config);
  const routing = asObject(template.routing);
  const balancers = asObjectArray(routing.balancers);
  const existing = index === null || index < 0 || index >= balancers.length ? null : balancers[index];
  const oldTag = existing ? stringValue(existing.tag) : '';
  const balancer: JsonObject = {
    tag: form.tag.trim(),
    selector: splitList(form.selectorText),
  };
  setStringField(balancer, 'fallbackTag', form.fallbackTag);
  if (form.strategy.trim() && form.strategy.trim() !== 'random') {
    balancer.strategy = {
      type: form.strategy.trim(),
    };
  }

  if (index === null || index < 0 || index >= balancers.length) {
    balancers.push(balancer);
  } else {
    balancers[index] = balancer;
  }
  routing.balancers = balancers;
  template.routing = routing;

  if (oldTag && oldTag !== form.tag.trim()) {
    const rules = getRoutingRules(template).map((rule) => {
      const nextRule = { ...rule };
      if (stringValue(nextRule.balancerTag) === oldTag) {
        nextRule.balancerTag = form.tag.trim();
      }
      return nextRule;
    });
    routing.rules = rules;
    template.routing = routing;
  }

  return syncObservatorySelectors(template);
}

export function deleteBalancerAt(config: JsonValue | null | undefined, index: number): JsonObject {
  const template = cloneObject(config);
  const routing = asObject(template.routing);
  const balancers = asObjectArray(routing.balancers);
  const removed = balancers.splice(index, 1)[0];
  if (balancers.length > 0) {
    routing.balancers = balancers;
  } else {
    delete routing.balancers;
  }
  const removedTag = stringValue(removed?.tag);
  if (removedTag) {
    routing.rules = getRoutingRules(template).map((rule) => {
      const nextRule = { ...rule };
      if (stringValue(nextRule.balancerTag) === removedTag) {
        delete nextRule.balancerTag;
      }
      return nextRule;
    });
  }
  template.routing = routing;
  return syncObservatorySelectors(template);
}

export function upsertReverse(
  config: JsonValue | null | undefined,
  index: number | null,
  form: ReverseEditorForm,
): JsonObject {
  let template = cloneObject(config);
  if (index !== null && index >= 0) {
    template = deleteReverseAt(template, index);
  }

  const reverse = asObject(template.reverse);
  const collectionKey = `${form.type}s`;
  const items = asObjectArray(reverse[collectionKey]);
  items.push({
    tag: form.tag.trim(),
    domain: form.domain.trim(),
  });
  reverse[collectionKey] = items;
  template.reverse = reverse;

  const routing = asObject(template.routing);
  const rules = getRoutingRules(template);
  rules.push(...buildReverseRules(form));
  routing.rules = rules;
  template.routing = routing;
  return template;
}

export function deleteReverseAt(config: JsonValue | null | undefined, index: number): JsonObject {
  const template = cloneObject(config);
  const rows = getReverseRows(template);
  const row = rows[index];
  if (!row) {
    return template;
  }

  const reverse = asObject(template.reverse);
  const collectionKey = `${row.type}s`;
  const items = asObjectArray(reverse[collectionKey]).filter(
    (item) => !(stringValue(item.tag) === row.tag && stringValue(item.domain) === row.domain),
  );
  if (items.length > 0) {
    reverse[collectionKey] = items;
  } else {
    delete reverse[collectionKey];
  }
  if (Object.keys(reverse).length > 0) {
    template.reverse = reverse;
  } else {
    delete template.reverse;
  }

  const routing = asObject(template.routing);
  routing.rules = getRoutingRules(template).filter((rule) => {
    if (row.type === 'bridge') {
      const inboundTag = arrayValue(rule.inboundTag);
      return !(inboundTag.length === 1 && inboundTag[0] === row.tag);
    }
    return stringValue(rule.outboundTag) !== row.tag;
  });
  template.routing = routing;
  return template;
}

export function getDnsPolicyForm(config: JsonValue | null | undefined): DnsPolicyForm {
  const template = asObject(config);
  const dns = asObject(template.dns);
  return {
    enableDNS: Object.keys(dns).length > 0,
    dnsTag: stringValue(dns.tag),
    dnsClientIp: stringValue(dns.clientIp),
    dnsStrategy: stringValue(dns.queryStrategy) || 'UseIP',
    dnsDisableCache: Boolean(dns.disableCache),
    dnsDisableFallback: Boolean(dns.disableFallback),
    dnsDisableFallbackIfMatch: Boolean(dns.disableFallbackIfMatch),
    dnsEnableParallelQuery: Boolean(dns.enableParallelQuery),
    dnsUseSystemHosts: Boolean(dns.useSystemHosts),
  };
}

export function applyDnsPolicyForm(
  config: JsonValue | null | undefined,
  form: DnsPolicyForm,
): JsonObject {
  const template = cloneObject(config);
  if (!form.enableDNS) {
    delete template.dns;
    delete template.fakedns;
    return template;
  }

  const dns = asObject(template.dns);
  dns.servers = Array.isArray(dns.servers) ? dns.servers : [];
  dns.queryStrategy = form.dnsStrategy || 'UseIP';
  setStringField(dns, 'tag', form.dnsTag);
  setStringField(dns, 'clientIp', form.dnsClientIp);
  setBooleanField(dns, 'disableCache', form.dnsDisableCache);
  setBooleanField(dns, 'disableFallback', form.dnsDisableFallback);
  setBooleanField(dns, 'disableFallbackIfMatch', form.dnsDisableFallbackIfMatch);
  setBooleanField(dns, 'enableParallelQuery', form.dnsEnableParallelQuery);
  setBooleanField(dns, 'useSystemHosts', form.dnsUseSystemHosts);
  template.dns = dns;
  return template;
}

export function applyDnsPreset(
  config: JsonValue | null | undefined,
  presetData: string[],
): JsonObject {
  const template = cloneObject(config);
  const dns = asObject(template.dns);
  dns.servers = presetData.slice();
  if (!dns.queryStrategy) {
    dns.queryStrategy = 'UseIP';
  }
  template.dns = dns;
  return template;
}

export function getRuntimePolicyForm(config: JsonValue | null | undefined): RuntimePolicyForm {
  const template = asObject(config);
  const routing = asObject(template.routing);
  const log = asObject(template.log);
  const policy = asObject(template.policy);
  const system = asObject(policy.system);
  const direct = getDirectFreedomOutbound(template);
  const directSettings = asObject(direct?.settings);
  return {
    freedomStrategy: stringValue(directSettings.domainStrategy) || 'AsIs',
    routingStrategy: stringValue(routing.domainStrategy) || 'AsIs',
    logLevel: stringValue(log.loglevel) || 'warning',
    accessLog: stringValue(log.access),
    errorLog: stringValue(log.error),
    dnsLog: Boolean(log.dnsLog),
    maskAddressLog: stringValue(log.maskAddress),
    statsInboundUplink: Boolean(system.statsInboundUplink),
    statsInboundDownlink: Boolean(system.statsInboundDownlink),
    statsOutboundUplink: Boolean(system.statsOutboundUplink),
    statsOutboundDownlink: Boolean(system.statsOutboundDownlink),
  };
}

export function applyRuntimePolicyForm(
  config: JsonValue | null | undefined,
  form: RuntimePolicyForm,
): JsonObject {
  const template = cloneObject(config);
  const routing = asObject(template.routing);
  routing.domainStrategy = form.routingStrategy || 'AsIs';
  template.routing = routing;

  const log = asObject(template.log);
  log.loglevel = form.logLevel || 'warning';
  setStringField(log, 'access', form.accessLog);
  setStringField(log, 'error', form.errorLog);
  setBooleanField(log, 'dnsLog', form.dnsLog);
  setStringField(log, 'maskAddress', form.maskAddressLog);
  template.log = log;

  const policy = asObject(template.policy);
  const system = asObject(policy.system);
  system.statsInboundUplink = form.statsInboundUplink;
  system.statsInboundDownlink = form.statsInboundDownlink;
  system.statsOutboundUplink = form.statsOutboundUplink;
  system.statsOutboundDownlink = form.statsOutboundDownlink;
  policy.system = system;
  template.policy = policy;

  const outbounds = asObjectArray(template.outbounds);
  const directIndex = outbounds.findIndex(
    (outbound) => stringValue(outbound.protocol) === 'freedom' && stringValue(outbound.tag) === 'direct',
  );
  if (directIndex === -1) {
    outbounds.push({
      protocol: 'freedom',
      tag: 'direct',
      settings: {
        domainStrategy: form.freedomStrategy || 'AsIs',
      },
    });
  } else {
    const direct = outbounds[directIndex];
    const settings = asObject(direct.settings);
    settings.domainStrategy = form.freedomStrategy || 'AsIs';
    direct.settings = settings;
    outbounds[directIndex] = direct;
  }
  template.outbounds = outbounds;

  return template;
}

export function getObservatoryForm(config: JsonValue | null | undefined): ObservatoryForm {
  const template = asObject(config);
  const observatory = asObject(template.observatory);
  const burstObservatory = asObject(template.burstObservatory);
  return {
    observatoryEnable: Object.keys(observatory).length > 0,
    observatoryJson: Object.keys(observatory).length > 0 ? JSON.stringify(observatory, null, 2) : '',
    burstObservatoryEnable: Object.keys(burstObservatory).length > 0,
    burstObservatoryJson:
      Object.keys(burstObservatory).length > 0 ? JSON.stringify(burstObservatory, null, 2) : '',
  };
}

export function applyObservatoryForm(
  config: JsonValue | null | undefined,
  form: ObservatoryForm,
): JsonObject {
  const template = cloneObject(config);
  if (form.observatoryEnable) {
    template.observatory = parseJsonObject(form.observatoryJson) || {};
  } else {
    delete template.observatory;
  }
  if (form.burstObservatoryEnable) {
    template.burstObservatory = parseJsonObject(form.burstObservatoryJson) || {};
  } else {
    delete template.burstObservatory;
  }
  return template;
}

function getRoutingRules(config: JsonValue | null | undefined): JsonObject[] {
  const template = asObject(config);
  const routing = asObject(template.routing);
  return asObjectArray(routing.rules);
}

function buildReverseRules(form: ReverseEditorForm): JsonObject[] {
  const domainRule: JsonObject = {
    type: 'field',
    domain: [`full:${form.domain.trim()}`],
  };
  const passRule: JsonObject = {
    type: 'field',
  };

  if (form.type === 'bridge') {
    domainRule.inboundTag = [form.tag.trim()];
    passRule.inboundTag = [form.tag.trim()];
    setStringField(domainRule, 'outboundTag', form.bridgeOutboundTag);
    setStringField(passRule, 'outboundTag', form.bridgeReplyOutboundTag);
    return [domainRule, passRule];
  }

  const inboundTags = splitList(form.portalInboundTagsText);
  if (inboundTags.length > 0) {
    domainRule.inboundTag = inboundTags;
    passRule.inboundTag = inboundTags;
  }
  domainRule.outboundTag = form.tag.trim();
  passRule.outboundTag = form.tag.trim();
  return [domainRule, passRule];
}

function getDirectFreedomOutbound(template: JsonObject): JsonObject | null {
  return asObjectArray(template.outbounds).find(
    (outbound) => stringValue(outbound.protocol) === 'freedom' && stringValue(outbound.tag) === 'direct',
  ) || null;
}

function syncObservatorySelectors(template: JsonObject): JsonObject {
  const next = cloneObject(template);
  const balancers = getBalancerRows(next);
  const leastPings = balancers.filter((row) => row.strategy === 'leastPing');
  const leastLoads = balancers.filter((row) =>
    ['leastLoad', 'roundRobin', 'random'].includes(row.strategy),
  );

  if (leastPings.length > 0) {
    const observatory = asObject(next.observatory);
    observatory.subjectSelector = uniqueList(leastPings.flatMap((row) => splitList(row.selectorText)));
    next.observatory = observatory;
  } else {
    delete next.observatory;
  }

  if (leastLoads.length > 0) {
    const burstObservatory = asObject(next.burstObservatory);
    burstObservatory.subjectSelector = uniqueList(
      leastLoads.flatMap((row) => splitList(row.selectorText)),
    );
    next.burstObservatory = burstObservatory;
  } else {
    delete next.burstObservatory;
  }

  return next;
}

function resolveOutboundAddress(outbound: JsonObject): string {
  const settings = asObject(outbound.settings);
  const servers = arrayValue(settings.servers);
  const firstServer = servers.length > 0 ? asObject(servers[0]) : null;
  if (firstServer) {
    const address = stringValue(firstServer.address);
    const port = formatScalar(firstServer.port);
    return address && port ? `${address}:${port}` : address || port || '-';
  }
  const address = stringValue(settings.address);
  return address || stringValue(outbound.sendThrough) || '-';
}

function cloneObject(value: JsonValue | null | undefined): JsonObject {
  return JSON.parse(JSON.stringify(asObject(value))) as JsonObject;
}

function asObject(value: JsonValue | null | undefined): JsonObject {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as JsonObject)
    : {};
}

function asObjectArray(value: JsonValue | undefined): JsonObject[] {
  return arrayValue(value).map((item) => asObject(item));
}

function arrayValue(value: JsonValue | undefined): JsonValue[] {
  return Array.isArray(value) ? value : [];
}

function stringValue(value: JsonValue | undefined): string {
  return typeof value === 'string' ? value : '';
}

function numberValue(value: JsonValue | undefined): number {
  return typeof value === 'number' ? value : 0;
}

function joinStringArray(value: JsonValue | undefined): string {
  return arrayValue(value)
    .filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
    .join('\n');
}

function formatScalar(value: JsonValue | undefined): string {
  if (typeof value === 'string' || typeof value === 'number') {
    return String(value);
  }
  return '';
}

function parseJsonObject(text: string): JsonObject | null {
  const trimmed = text.trim();
  if (!trimmed || trimmed === '{}') {
    return null;
  }
  const parsed = JSON.parse(trimmed) as JsonValue;
  return asObject(parsed);
}

function splitList(text: string): string[] {
  return text
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function uniqueList(items: string[]): string[] {
  return Array.from(new Set(items.filter((item) => item.trim().length > 0)));
}

function setStringField(target: JsonObject, key: string, value: string) {
  const trimmed = value.trim();
  if (trimmed) {
    target[key] = trimmed;
  }
}

function setArrayField(target: JsonObject, key: string, value: string) {
  const items = splitList(value);
  if (items.length > 0) {
    target[key] = items;
  }
}

function setScalarField(target: JsonObject, key: string, value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return;
  }
  const number = Number(trimmed);
  target[key] = Number.isFinite(number) && `${number}` === trimmed ? number : trimmed;
}

function setObjectField(target: JsonObject, key: string, value: JsonObject | null) {
  if (value && Object.keys(value).length > 0) {
    target[key] = value;
  }
}

function setBooleanField(target: JsonObject, key: string, value: boolean) {
  target[key] = value;
}
