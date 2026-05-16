import type { JsonObject, JsonValue } from '@/types/api';

export type GatewayEgressPlatform = 'openai' | 'anthropic' | 'gemini';

export interface GatewayEgressMvpProfile {
  egressGroup: string;
  expectedCountryCode: string;
  key: string;
  notes: string;
  platform: GatewayEgressPlatform | '';
  port: number;
  regionCode: 'US' | 'JP';
}

export interface GatewayEgressMvpNetworkStrategy {
  listenHost: string;
  manifestHost: string;
  strategyLabel: string;
}

export interface GatewayEgressMvpPreview {
  listenHost: string;
  manifestHost: string;
  platforms: GatewayEgressPlatform[];
  ports: number[];
  profileCount: number;
  regions: Array<'US' | 'JP'>;
  strategyLabel: string;
}

export const DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY: GatewayEgressMvpNetworkStrategy = {
  listenHost: '127.0.0.1',
  manifestHost: '127.0.0.1',
  strategyLabel: 'same-network',
};

export const GATEWAY_EGRESS_MVP_PROFILES: GatewayEgressMvpProfile[] = [
  {
    key: 'openai-us-primary',
    port: 11801,
    platform: 'openai',
    regionCode: 'US',
    expectedCountryCode: 'US',
    egressGroup: 'openai-egress',
    notes: 'OpenAI MVP local exit',
  },
  {
    key: 'anthropic-us-primary',
    port: 11802,
    platform: 'anthropic',
    regionCode: 'US',
    expectedCountryCode: 'US',
    egressGroup: 'anthropic-egress',
    notes: 'Anthropic MVP local exit',
  },
  {
    key: 'gemini-us-primary',
    port: 11803,
    platform: 'gemini',
    regionCode: 'US',
    expectedCountryCode: 'US',
    egressGroup: 'gemini-egress',
    notes: 'Gemini MVP local exit',
  },
  {
    key: 'region-us-primary',
    port: 11901,
    platform: '',
    regionCode: 'US',
    expectedCountryCode: 'US',
    egressGroup: 'region-us',
    notes: 'US region MVP local exit',
  },
  {
    key: 'region-jp-primary',
    port: 11981,
    platform: '',
    regionCode: 'JP',
    expectedCountryCode: 'JP',
    egressGroup: 'region-jp',
    notes: 'JP region MVP local exit',
  },
];

const PLATFORM_DOMAINS: Record<GatewayEgressPlatform, string[]> = {
  openai: ['domain:api.openai.com', 'domain:chatgpt.com', 'domain:chat.openai.com'],
  anthropic: ['domain:api.anthropic.com', 'domain:claude.ai'],
  gemini: [
    'domain:generativelanguage.googleapis.com',
    'domain:cloudcode-pa.googleapis.com',
    'domain:aiplatform.googleapis.com',
  ],
};

const WILDCARD_HOSTS = new Set(['0.0.0.0', '::', '[::]', '*']);

export function buildGatewayEgressMvpPreview(
  strategy?: Partial<GatewayEgressMvpNetworkStrategy>,
): GatewayEgressMvpPreview {
  const network = normalizeGatewayEgressMvpNetworkStrategy(strategy);
  return {
    ...network,
    profileCount: GATEWAY_EGRESS_MVP_PROFILES.length,
    ports: GATEWAY_EGRESS_MVP_PROFILES.map((profile) => profile.port),
    platforms: Array.from(
      new Set(
        GATEWAY_EGRESS_MVP_PROFILES.filter(
          (profile): profile is GatewayEgressMvpProfile & { platform: GatewayEgressPlatform } =>
            profile.platform !== '',
        ).map((profile) => profile.platform),
      ),
    ),
    regions: Array.from(new Set(GATEWAY_EGRESS_MVP_PROFILES.map((profile) => profile.regionCode))),
  };
}

export function mergeGatewayEgressMvpConfig(
  source: JsonValue | null | undefined,
  strategy?: Partial<GatewayEgressMvpNetworkStrategy>,
): JsonObject {
  const network = normalizeGatewayEgressMvpNetworkStrategy(strategy);
  const next = cloneObject(source);
  const inbounds = asArray(next.inbounds);
  const outbounds = asArray(next.outbounds);
  const routing = isJsonObject(next.routing) ? { ...next.routing } : {};
  const rules = asArray(routing.rules);

  for (const profile of GATEWAY_EGRESS_MVP_PROFILES) {
    upsertByTag(inbounds, buildInbound(profile, network.listenHost));
    ensureOutbound(outbounds, buildPlaceholderOutbound(profile));
  }

  const generatedInboundTags = GATEWAY_EGRESS_MVP_PROFILES.map((profile) =>
    gatewayInboundTag(profile),
  );
  const generatedRules = GATEWAY_EGRESS_MVP_PROFILES.flatMap((profile) => buildRules(profile));
  const preservedRules = rules.filter((rule) => !isGeneratedGatewayRule(rule));

  routing.domainStrategy = typeof routing.domainStrategy === 'string' ? routing.domainStrategy : 'AsIs';
  routing.rules = [
    ...generatedRules,
    {
      type: 'field',
      inboundTag: generatedInboundTags,
      outboundTag: 'blocked',
      _gatewayEgressMvp: true,
    },
    ...preservedRules,
  ];

  if (!outbounds.some((outbound) => isJsonObject(outbound) && outbound.tag === 'blocked')) {
    outbounds.push({ tag: 'blocked', protocol: 'blackhole' });
  }

  next.inbounds = inbounds;
  next.outbounds = outbounds;
  next.routing = routing;
  return next;
}

export function buildGatewayEgressManifestCsv(
  strategy?: Partial<GatewayEgressMvpNetworkStrategy>,
): string {
  const network = normalizeGatewayEgressMvpNetworkStrategy(strategy);
  const header =
    'name,protocol,host,port,platform,region_code,expected_country_code,egress_group,health_status,notes';
  const rows = GATEWAY_EGRESS_MVP_PROFILES.map((profile) =>
    [
      profile.key,
      'socks5h',
      network.manifestHost,
      String(profile.port),
      profile.platform,
      profile.regionCode,
      profile.expectedCountryCode,
      profile.egressGroup,
      'manual-check',
      `${profile.notes} (${network.strategyLabel})`,
    ].map(csvCell).join(','),
  );
  return [header, ...rows].join('\n');
}

export function normalizeGatewayEgressMvpNetworkStrategy(
  strategy: Partial<GatewayEgressMvpNetworkStrategy> = {},
): GatewayEgressMvpNetworkStrategy {
  return {
    listenHost: normalizeHost(
      strategy.listenHost === undefined
        ? DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY.listenHost
        : strategy.listenHost,
      'listenHost',
    ),
    manifestHost: normalizeHost(
      strategy.manifestHost === undefined
        ? DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY.manifestHost
        : strategy.manifestHost,
      'manifestHost',
    ),
    strategyLabel:
      typeof strategy.strategyLabel === 'string' && strategy.strategyLabel.trim()
        ? strategy.strategyLabel.trim()
        : DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY.strategyLabel,
  };
}

function buildInbound(profile: GatewayEgressMvpProfile, listenHost: string): JsonObject {
  return {
    tag: gatewayInboundTag(profile),
    listen: listenHost,
    port: profile.port,
    protocol: 'socks',
    settings: {
      auth: 'noauth',
      accounts: [],
      udp: false,
      ip: listenHost,
    },
  };
}

function buildPlaceholderOutbound(profile: GatewayEgressMvpProfile): JsonObject {
  return {
    tag: profile.egressGroup,
    protocol: 'freedom',
    settings: {},
    _gatewayEgressMvp: {
      profile: profile.key,
      expectedCountryCode: profile.expectedCountryCode,
      note: 'Replace this placeholder with the real VPN/proxy outbound before production use.',
    },
  };
}

function buildRules(profile: GatewayEgressMvpProfile): JsonObject[] {
  if (!profile.platform) {
    return [
      {
        type: 'field',
        inboundTag: [gatewayInboundTag(profile)],
        outboundTag: profile.egressGroup,
        _gatewayEgressMvp: true,
      },
    ];
  }

  return [
    {
      type: 'field',
      inboundTag: [gatewayInboundTag(profile)],
      domain: PLATFORM_DOMAINS[profile.platform],
      outboundTag: profile.egressGroup,
      _gatewayEgressMvp: true,
    },
  ];
}

function gatewayInboundTag(profile: GatewayEgressMvpProfile): string {
  return `gateway-${profile.key}`;
}

function normalizeHost(value: string, fieldName: string): string {
  const host = value.trim();
  if (!host) {
    throw new Error(`${fieldName} is required`);
  }
  if (WILDCARD_HOSTS.has(host)) {
    throw new Error(`${fieldName} cannot be a wildcard host`);
  }
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(host) || /[\s,/]/.test(host)) {
    throw new Error(`${fieldName} must be a host without protocol, path, comma, or whitespace`);
  }
  return host;
}

function upsertByTag(items: JsonValue[], item: JsonObject): void {
  const tag = String(item.tag || '');
  const index = items.findIndex((candidate) => isJsonObject(candidate) && candidate.tag === tag);
  if (index >= 0) {
    items[index] = item;
    return;
  }
  items.push(item);
}

function ensureOutbound(items: JsonValue[], item: JsonObject): void {
  const tag = String(item.tag || '');
  if (!items.some((candidate) => isJsonObject(candidate) && candidate.tag === tag)) {
    items.push(item);
  }
}

function isGeneratedGatewayRule(value: JsonValue): boolean {
  return isJsonObject(value) && value._gatewayEgressMvp === true;
}

function cloneObject(value: JsonValue | null | undefined): JsonObject {
  return JSON.parse(JSON.stringify(isJsonObject(value) ? value : {})) as JsonObject;
}

function asArray(value: JsonValue | undefined): JsonValue[] {
  return Array.isArray(value) ? [...value] : [];
}

function isJsonObject(value: unknown): value is JsonObject {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function csvCell(value: string): string {
  return /[",\n\r]/.test(value) ? `"${value.replaceAll('"', '""')}"` : value;
}
