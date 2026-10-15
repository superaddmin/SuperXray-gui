import type {
  Inbound,
  InboundClient,
  InboundSettings,
  InboundSniffingSettings,
  InboundStreamSettings,
  XrayEditableInboundProtocol,
} from '@/types/inbound';

const WILDCARD_LISTENS = new Set(['', '0.0.0.0', '::', '::0']);
const SHADOWSOCKS_DEFAULT_METHOD = 'chacha20-ietf-poly1305';
const SHADOWSOCKS_2022_PREFIX = '2022-';
const SHADOWSOCKS_2022_AES_128_GCM = '2022-blake3-aes-128-gcm';
const SHADOWSOCKS_2022_SINGLE_USER = '2022-blake3-chacha20-poly1305';
const SHADOWSOCKS_DEFAULT_KEY_BYTES = 32;
const REMOVED_SHADOWSOCKS_METHODS = new Set(['none', 'plain']);

export const HYSTERIA_QUIC_DEFAULTS = {
  initStreamReceiveWindow: 8_388_608,
  maxStreamReceiveWindow: 8_388_608,
  initConnectionReceiveWindow: 20_971_520,
  maxConnectionReceiveWindow: 20_971_520,
  maxIdleTimeout: 30,
  maxIncomingStreams: 1_024,
} as const;

export interface SubscriptionEndpointSettings {
  subEnable: boolean;
  subJsonEnable: boolean;
  subClashEnable: boolean;
  subURI: string;
  subJsonURI: string;
  subClashURI: string;
}

export interface PanelDefaultTlsCertificate {
  certFile?: string;
  keyFile?: string;
}

export interface SubscriptionLinkItem {
  label: string;
  url: string;
}

export interface HysteriaUdpHopFormInput {
  quicParamsEnabled: boolean;
  udpHopEnabled: boolean;
  ports: string;
  interval: string;
  initStreamReceiveWindow: number;
  maxStreamReceiveWindow: number;
  initConnectionReceiveWindow: number;
  maxConnectionReceiveWindow: number;
  maxIdleTimeout: number;
  maxIncomingStreams: number;
}

export interface XhttpFormInput {
  path: string;
  host: string;
  mode: string;
  noSSEHeader: boolean;
  scMaxBufferedPosts: number;
  scMaxEachPostBytes: string;
  scStreamUpServerSecs: string;
  xPaddingBytes: string;
  xPaddingObfsMode: boolean;
  xPaddingKey: string;
  xPaddingHeader: string;
  xPaddingPlacement: string;
  xPaddingMethod: string;
  uplinkHTTPMethod: string;
  sessionPlacement: string;
  sessionKey: string;
  sessionIDTable?: string;
  sessionIDLength?: string;
  seqPlacement: string;
  seqKey: string;
  uplinkDataPlacement: string;
  uplinkDataKey: string;
  uplinkChunkSize: number;
  xmuxEnabled: boolean;
  xmuxMaxConcurrency: string;
  xmuxMaxConnections: string;
  xmuxCMaxReuseTimes: string;
  xmuxHMaxRequestTimes: string;
  xmuxHMaxReusableSecs: string;
  xmuxHKeepAlivePeriod: number;
}

export function mergeSubscriptionEndpointDefaults(
  settings: SubscriptionEndpointSettings,
  defaults: Partial<SubscriptionEndpointSettings>,
): SubscriptionEndpointSettings {
  return {
    ...settings,
    subURI: settings.subURI || (settings.subEnable ? defaults.subURI || '' : ''),
    subJsonURI: settings.subJsonURI || (settings.subJsonEnable ? defaults.subJsonURI || '' : ''),
    subClashURI:
      settings.subClashURI || (settings.subClashEnable ? defaults.subClashURI || '' : ''),
  };
}

interface ShareLinkBuildOptions {
  address?: string;
  port?: number;
  forceTls?: string;
  remark?: string;
}

interface ShareLinkTarget extends ShareLinkBuildOptions {
  address: string;
  port: number;
  remark: string;
}

interface ProxyAccount {
  user?: string;
  pass?: string;
  subId?: string;
  remark?: string;
  email?: string;
  comment?: string;
  enable?: boolean;
  enabled?: boolean;
}

export interface BulkClientGenerationInput {
  protocol: XrayEditableInboundProtocol;
  quantity: number;
  firstIndex: number;
  emailPrefix: string;
  emailPostfix: string;
  flow?: string;
  security?: string;
  shadowsocksMethod?: string;
  limitIp: number;
  totalGB: number;
  expiryTime: number;
  reset: number;
}

export const SHADOWSOCKS_METHOD_OPTIONS = [
  SHADOWSOCKS_DEFAULT_METHOD,
  'chacha20-poly1305',
  'xchacha20-ietf-poly1305',
  SHADOWSOCKS_2022_AES_128_GCM,
  '2022-blake3-aes-256-gcm',
  SHADOWSOCKS_2022_SINGLE_USER,
];

export function parseInboundSettings(inbound: Pick<Inbound, 'settings'>): InboundSettings {
  return parseJsonObject<InboundSettings>(inbound.settings, { clients: [] });
}

export function separateInboundClients(settings: Record<string, unknown>): {
  editorSettings: Record<string, unknown>;
  clients: unknown[];
} {
  const editorSettings = { ...settings };
  const clients = Array.isArray(editorSettings.clients) ? [...editorSettings.clients] : [];
  delete editorSettings.clients;
  return { editorSettings, clients };
}

export function restoreInboundClients(
  editorSettings: Record<string, unknown>,
  clients: readonly unknown[],
): Record<string, unknown> {
  return {
    ...editorSettings,
    clients: [...clients],
  };
}

export function parseInboundStreamSettings(
  inbound: Pick<Inbound, 'streamSettings'>,
): InboundStreamSettings {
  const parsed = parseJsonObject<InboundStreamSettings>(
    inbound.streamSettings,
    defaultStreamSettings(),
  );
  return normalizeInboundStreamCompatibility(parsed);
}

/**
 * Normalize persisted stream settings before they reach the Vue editor.
 *
 * Xray 26.10.10 renamed XHTTP's session fields, canonicalized sockopt's
 * V6Only spelling, and accepts REALITY's historical `dest` alias.  Keep the
 * conversion at the JSON boundary so every editor, share-link builder, and
 * import path sees the same values while unrelated/unknown fields survive.
 */
export function normalizeInboundStreamCompatibility(
  stream: InboundStreamSettings,
): InboundStreamSettings {
  const next: InboundStreamSettings = { ...stream };

  if (isRecord(next.xhttpSettings)) {
    next.xhttpSettings = normalizeXhttpCompatibility(next.xhttpSettings);
  }

  if (isRecord(next.realitySettings)) {
    const reality = { ...next.realitySettings };
    if (!stringValue(reality.target).trim() && stringValue(reality.dest).trim()) {
      reality.target = reality.dest;
    }
    next.realitySettings = reality;
  }

  if (isRecord(next.sockopt)) {
    next.sockopt = normalizeSockoptCompatibility(next.sockopt);
  }

  return next;
}

/** Migrate legacy XHTTP session keys without dropping sessionID table/range. */
export function normalizeXhttpCompatibility(
  settings: Record<string, unknown>,
): Record<string, unknown> {
  const next = { ...settings };
  if (
    (next.sessionIDPlacement === undefined || next.sessionIDPlacement === '') &&
    typeof next.sessionPlacement === 'string'
  ) {
    next.sessionIDPlacement = next.sessionPlacement;
  }
  if (
    (next.sessionIDKey === undefined || next.sessionIDKey === '') &&
    typeof next.sessionKey === 'string'
  ) {
    next.sessionIDKey = next.sessionKey;
  }
  delete next.sessionPlacement;
  delete next.sessionKey;
  if (isRecord(next.extra)) {
    next.extra = normalizeXhttpCompatibility(next.extra);
  }
  return next;
}

/** Fold lowercase imports into Xray's canonical sockopt key. */
export function normalizeSockoptCompatibility(
  settings: Record<string, unknown>,
): Record<string, unknown> {
  const next = { ...settings };
  if (next.V6Only === undefined && next.v6only !== undefined) {
    next.V6Only = Boolean(next.v6only);
  }
  delete next.v6only;
  return next;
}

/**
 * Xray 26.6.22 removed the WireGuard `workers` setting.  It is invalid on
 * current cores, so drop only that known obsolete key and retain every other
 * extension in the embedded settings JSON.
 */
export function normalizeWireguardSettings(settings: InboundSettings): InboundSettings {
  if (!Object.hasOwn(settings, 'workers')) {
    return settings;
  }
  const next = { ...settings };
  delete next.workers;
  return next;
}

/** Convert pre-26.9.9 QUIC udpHop and pre-26.9.30 xDNS forms to 26.10.10. */
export function normalizeFinalmaskForXray(stream: InboundStreamSettings): InboundStreamSettings {
  const finalmask = stream.finalmask;
  if (!isRecord(finalmask)) {
    return stream;
  }
  // Keep the original editable document when conversion would lose data.
  if (validateFinalmaskForXray(stream)) return stream;

  const nextFinalmask: Record<string, unknown> = { ...finalmask };
  const quicParams = isRecord(nextFinalmask.quicParams) ? { ...nextFinalmask.quicParams } : null;
  const { masks: udpMasks } = migrateUdpHopMasks(finalmask);

  if (quicParams && Object.hasOwn(quicParams, 'udpHop')) {
    delete quicParams.udpHop;
    if (Object.keys(quicParams).length > 0) nextFinalmask.quicParams = quicParams;
    else delete nextFinalmask.quicParams;
  }

  if (udpMasks.length > 0) {
    nextFinalmask.udp = udpMasks.map((entry) => normalizeXdnsMask(entry));
  }
  const result: InboundStreamSettings = { ...stream, finalmask: nextFinalmask };
  if (Object.keys(nextFinalmask).length === 0) delete result.finalmask;
  return result;
}

/** Validate combinations that Xray 26.10.10 rejects without mutating JSON. */
export function validateFinalmaskForXray(stream: InboundStreamSettings): string {
  const finalmask = stream.finalmask;
  if (!isRecord(finalmask)) return '';
  const hopError = migrateUdpHopMasks(finalmask).error;
  if (hopError) return hopError;
  if (stringValue(stream.security).toLowerCase() === 'reality') {
    const tcp = Array.isArray(finalmask.tcp) ? finalmask.tcp : [];
    if (tcp.length > 0) return 'REALITY cannot be combined with a TCP finalmask';
  }
  const udp = Array.isArray(finalmask.udp) ? finalmask.udp : [];
  for (const raw of udp) {
    const mask = asRecord(raw);
    if (String(mask.type).toLowerCase() === 'salamander') {
      const password = stringValue(asRecord(mask.settings).password);
      if (new TextEncoder().encode(password).length < 4)
        return 'Salamander finalmask password must be at least 4 bytes';
    }
  }
  const tcp = Array.isArray(finalmask.tcp) ? finalmask.tcp : [];
  for (const raw of tcp) {
    const mask = asRecord(raw);
    if (String(mask.type).toLowerCase() !== 'xmc') continue;
    const settings = asRecord(mask.settings);
    const profiles = Array.isArray(settings.profiles) ? settings.profiles : [];
    if (profiles.length === 0) {
      return 'XMC finalmask requires profiles with username, UUID, texturesValue and texturesSignature';
    }
    for (const rawProfile of profiles) {
      const profile = asRecord(rawProfile);
      const uuid = stringValue(profile.uuid);
      const canonical = '[\\da-f]{8}-[\\da-f]{4}-[\\da-f]{4}-[\\da-f]{4}-[\\da-f]{12}';
      if (
        !/^[A-Za-z0-9_]{3,16}$/.test(stringValue(profile.username)) ||
        !new RegExp(`^(?:[\\da-f]{32}|(?:urn:uuid:)?${canonical}|\\{${canonical}\\})$`, 'i').test(
          uuid,
        ) ||
        !stringValue(profile.texturesValue) ||
        !stringValue(profile.texturesSignature)
      ) {
        return 'XMC finalmask profile is incomplete or invalid';
      }
    }
  }
  const xdns = udp.filter((raw) => String(asRecord(raw).type).toLowerCase() === 'xdns');
  for (const raw of xdns) {
    const settings = asRecord(raw).settings;
    if (!isRecord(settings)) return 'xDNS settings must be an object';
    const domains = settings.domains;
    for (const entry of Array.isArray(domains) ? domains : []) {
      if (isRecord(entry) && Object.hasOwn(entry, 'name')) {
        if (!stringValue(entry.name).trim())
          return 'Legacy xDNS domain name must be a non-empty string';
        if (Object.hasOwn(entry, 'names') && !Array.isArray(entry.names)) {
          return 'xDNS domain names must be an array to migrate name';
        }
      }
      if (typeof entry !== 'string') continue;
      if (!legacyXdnsDomain(entry)) {
        return `xDNS domain record type is unsupported: ${entry}`;
      }
    }
    for (const entry of Array.isArray(settings.resolvers) ? settings.resolvers : []) {
      if (typeof entry === 'string') {
        if (Object.hasOwn(settings, 'domains') && !Array.isArray(settings.domains)) {
          return 'xDNS domains must be an array to migrate legacy resolvers';
        }
        const marker = entry.indexOf('+udp://');
        if (
          marker <= 0 ||
          !legacyXdnsDomain(entry.slice(0, marker)) ||
          !entry.slice(marker + 7).trim()
        ) {
          return 'Legacy xDNS resolver must use name[:type]+udp://address';
        }
      } else if (
        isRecord(entry) &&
        !Object.hasOwn(entry, 'addrs') &&
        Object.hasOwn(entry, 'type')
      ) {
        if (
          !['udp', 'tcp'].includes(stringValue(entry.type).toLowerCase()) ||
          !stringValue(asRecord(entry.settings).addr).trim()
        ) {
          return 'Legacy xDNS resolver requires udp/tcp and settings.addr';
        }
      }
    }
  }
  return '';
}

function migrateUdpHopMasks(finalmask: Record<string, unknown>): {
  masks: unknown[];
  error: string;
} {
  const masks: unknown[] = Array.isArray(finalmask.udp) ? [...finalmask.udp] : [];
  const qp = asRecord(finalmask.quicParams);
  if (Object.hasOwn(qp, 'udpHop')) {
    const hop = asRecord(qp.udpHop);
    if (!stringValue(hop.ports).trim()) return { masks, error: 'Legacy udpHop requires ports' };
    if (Object.hasOwn(hop, 'mode') && hop.mode !== 'intervalRemote') {
      return { masks, error: 'Legacy udpHop mode conflicts with intervalRemote' };
    }
    if (Object.hasOwn(hop, 'remotePorts') && !equalJson(hop.remotePorts, hop.ports)) {
      return { masks, error: 'Legacy udpHop ports conflict with remotePorts' };
    }
    if (Object.hasOwn(finalmask, 'udp') && !Array.isArray(finalmask.udp)) {
      return { masks, error: 'Finalmask udp must be an array to migrate udpHop' };
    }
    const settings: Record<string, unknown> = {
      ...hop,
      mode: 'intervalRemote',
      remotePorts: hop.ports,
    };
    delete settings.ports;
    masks.push({ type: 'udphop', settings });
  }
  let hop: unknown;
  const kept: unknown[] = [];
  for (const mask of masks) {
    if (stringValue(asRecord(mask).type).toLowerCase() !== 'udphop') {
      kept.push(mask);
    } else {
      if (hop !== undefined && !equalJson(hop, mask)) {
        return {
          masks,
          error:
            'Multiple UDP hop masks have different settings; choose one configuration before upgrading',
        };
      }
      hop = mask;
    }
  }
  if (hop !== undefined) kept.push(hop);
  return { masks: kept, error: '' };
}

function equalJson(left: unknown, right: unknown): boolean {
  if (left === right) return true;
  if (Array.isArray(left) && Array.isArray(right)) {
    return (
      left.length === right.length && left.every((value, index) => equalJson(value, right[index]))
    );
  }
  if (isRecord(left) && isRecord(right)) {
    return (
      Object.keys(left).length === Object.keys(right).length &&
      Object.keys(left).every(
        (key) => Object.hasOwn(right, key) && equalJson(left[key], right[key]),
      )
    );
  }
  return false;
}

function legacyXdnsDomain(value: string): Record<string, unknown> | null {
  const trimmed = value.trim();
  const separator = trimmed.lastIndexOf(':');
  const suffix = separator >= 0 ? trimmed.slice(separator + 1).toLowerCase() : '';
  if (separator >= 0 && !['a', 'aaaa', 'txt'].includes(suffix)) return null;
  const name = (separator >= 0 ? trimmed.slice(0, separator) : trimmed).replace(/^\.+|\.+$/g, '');
  if (!name || name.includes(':')) return null;
  return { names: [name], types: [suffix === 'a' ? 1 : suffix === 'aaaa' ? 28 : 16], edns0: 1232 };
}

function normalizeXdnsMask(value: unknown): unknown {
  if (
    !isRecord(value) ||
    String(value.type).toLowerCase() !== 'xdns' ||
    !isRecord(value.settings)
  ) {
    return value;
  }
  const settings = { ...value.settings };
  const domains = Array.isArray(settings.domains) ? settings.domains : [];
  const upgradedDomains = domains.flatMap((entry) => {
    if (isRecord(entry)) {
      if (typeof entry.name === 'string') {
        const names = Array.isArray(entry.names) ? [...entry.names] : [];
        if (!names.includes(entry.name)) names.push(entry.name);
        const next: Record<string, unknown> = { ...entry, names };
        delete next.name;
        return [next];
      }
      return [entry];
    }
    if (typeof entry !== 'string') return [entry];
    return [legacyXdnsDomain(entry) ?? entry];
  });

  const resolvers = Array.isArray(settings.resolvers) ? settings.resolvers : [];
  const upgradedResolvers = resolvers.flatMap((entry) => {
    if (isRecord(entry) && !Object.hasOwn(entry, 'addrs') && typeof entry.type === 'string') {
      const resolverSettings = { ...asRecord(entry.settings) };
      const addr = stringValue(resolverSettings.addr);
      if (!addr) return [entry];
      const next: Record<string, unknown> = {
        ...entry,
        addrs: [`${entry.type.toLowerCase()}://${addr}`],
      };
      delete next.type;
      delete resolverSettings.addr;
      if (Object.keys(resolverSettings).length) next.settings = resolverSettings;
      else delete next.settings;
      return [next];
    }
    if (typeof entry === 'string') {
      const marker = entry.indexOf('+udp://');
      const domain = marker > 0 ? legacyXdnsDomain(entry.slice(0, marker)) : null;
      if (domain) {
        if (!upgradedDomains.some((existing) => equalJson(existing, domain)))
          upgradedDomains.push(domain);
        return [{ addrs: [`udp://${entry.slice(marker + 7)}`] }];
      }
      return [entry];
    }
    return [entry];
  });
  if (upgradedDomains.length > 0) settings.domains = upgradedDomains;
  if (resolvers.length > 0) settings.resolvers = upgradedResolvers;
  return { ...value, settings };
}

export function parseInboundSniffingSettings(
  inbound: Pick<Inbound, 'sniffing'>,
): InboundSniffingSettings {
  return parseJsonObject<InboundSniffingSettings>(inbound.sniffing, defaultSniffingSettings());
}

export function getInboundClients(
  inbound: Pick<Inbound, 'settings' | 'protocol'>,
): InboundClient[] {
  const settings = parseInboundSettings(inbound);
  if (inbound.protocol === 'wireguard') {
    const normalized = normalizeWireguardSettings(settings);
    return Array.isArray(normalized.peers) ? normalized.peers.filter(isWireguardPeer) : [];
  }
  return Array.isArray(settings.clients) ? settings.clients.filter(isInboundClient) : [];
}

export function defaultInboundSettings(protocol: XrayEditableInboundProtocol): InboundSettings {
  if (protocol === 'vless') {
    return {
      clients: [],
      decryption: 'none',
      encryption: 'none',
    };
  }

  if (protocol === 'trojan') {
    return {
      clients: [],
      fallbacks: [],
    };
  }

  if (protocol === 'shadowsocks') {
    return {
      method: SHADOWSOCKS_DEFAULT_METHOD,
      network: 'tcp,udp',
      clients: [],
      ivCheck: false,
    };
  }

  if (protocol === 'hysteria' || protocol === 'hysteria2') {
    return {
      version: 2,
      clients: [],
    };
  }

  if (protocol === 'wireguard') {
    return defaultWireguardSettings();
  }

  if (protocol === 'tunnel') {
    return {
      address: '',
      port: 0,
      portMap: [],
      network: 'tcp,udp',
      followRedirect: false,
    };
  }

  if (protocol === 'mixed') {
    return {
      auth: 'password',
      accounts: [defaultProxyAccount()],
      udp: false,
      ip: '127.0.0.1',
    };
  }

  if (protocol === 'http') {
    return {
      accounts: [defaultProxyAccount()],
      allowTransparent: false,
    };
  }

  if (protocol === 'tun') {
    return {
      name: 'xray0',
      mtu: 1500,
      gateway: ['10.0.0.1/16'],
      dns: [],
      userLevel: 0,
      autoSystemRoutingTable: [],
      autoOutboundsInterface: 'auto',
    };
  }

  return {
    clients: [],
  };
}

export function defaultStreamSettings(
  protocol?: XrayEditableInboundProtocol,
): InboundStreamSettings {
  if (
    protocol === 'wireguard' ||
    protocol === 'tunnel' ||
    protocol === 'mixed' ||
    protocol === 'http' ||
    protocol === 'tun'
  ) {
    return {};
  }

  if (protocol === 'hysteria' || protocol === 'hysteria2') {
    return {
      network: 'hysteria',
      security: 'tls',
      externalProxy: [],
      tlsSettings: {
        serverName: '',
        minVersion: '1.2',
        maxVersion: '1.3',
        cipherSuites: '',
        rejectUnknownSni: false,
        disableSystemRoot: false,
        enableSessionResumption: false,
        certificates: [],
        alpn: ['h3'],
        settings: {
          fingerprint: '',
          echConfigList: '',
        },
      },
      hysteriaSettings: {
        protocol: 'hysteria',
        version: 2,
        auth: '',
        udpIdleTimeout: 60,
      },
    };
  }

  return {
    network: 'tcp',
    security: 'none',
    externalProxy: [],
    tcpSettings: {
      acceptProxyProtocol: false,
      header: {
        type: 'none',
      },
    },
  };
}

export function applyPanelDefaultTlsCertificate(
  stream: InboundStreamSettings,
  defaults: PanelDefaultTlsCertificate,
): InboundStreamSettings {
  const certFile = (defaults.certFile || '').trim();
  const keyFile = (defaults.keyFile || '').trim();
  if (!certFile || !keyFile) {
    return stream;
  }

  const tlsSettings = asRecord(stream.tlsSettings);
  const certificates = Array.isArray(tlsSettings.certificates) ? tlsSettings.certificates : [];
  if (certificates.some(hasUsableTlsCertificateEntry)) {
    return stream;
  }

  return {
    ...stream,
    tlsSettings: {
      ...tlsSettings,
      certificates: [
        {
          certificateFile: certFile,
          keyFile,
          oneTimeLoading: false,
          usage: 'encipherment',
          buildChain: false,
        },
      ],
    },
  };
}

export function applyHysteriaFinalmaskUdpHop(
  stream: InboundStreamSettings,
  input: HysteriaUdpHopFormInput,
): InboundStreamSettings {
  const next: InboundStreamSettings = { ...stream };
  const finalmask: Record<string, unknown> = { ...asRecord(stream.finalmask) };
  const quicParams: Record<string, unknown> = { ...asRecord(finalmask.quicParams) };
  const existingHop = { ...resolveHysteriaUdpHop(finalmask) };
  delete existingHop.mode;
  delete existingHop.remotePorts;
  const ports = normalizeHopRange(input.ports);
  const interval = normalizeHopRange(input.interval);
  // The form owns the historical intervalRemote hop only. Keep other modes
  // and unrelated finalmask extensions in the advanced JSON editor intact.
  if (Array.isArray(finalmask.udp)) {
    const masks = finalmask.udp.filter((entry) => !isEditableUdpHopMask(entry));
    if (masks.length > 0) finalmask.udp = masks;
    else delete finalmask.udp;
  }

  if (!input.quicParamsEnabled) {
    delete finalmask.quicParams;
    if (Object.keys(finalmask).length > 0) {
      next.finalmask = finalmask;
    } else {
      delete next.finalmask;
    }
    return next;
  }

  quicParams.initStreamReceiveWindow = nonNegativeInteger(
    input.initStreamReceiveWindow,
    HYSTERIA_QUIC_DEFAULTS.initStreamReceiveWindow,
  );
  quicParams.maxStreamReceiveWindow = nonNegativeInteger(
    input.maxStreamReceiveWindow,
    HYSTERIA_QUIC_DEFAULTS.maxStreamReceiveWindow,
  );
  quicParams.initConnectionReceiveWindow = nonNegativeInteger(
    input.initConnectionReceiveWindow,
    HYSTERIA_QUIC_DEFAULTS.initConnectionReceiveWindow,
  );
  quicParams.maxConnectionReceiveWindow = nonNegativeInteger(
    input.maxConnectionReceiveWindow,
    HYSTERIA_QUIC_DEFAULTS.maxConnectionReceiveWindow,
  );
  quicParams.maxIdleTimeout = nonNegativeInteger(
    input.maxIdleTimeout,
    HYSTERIA_QUIC_DEFAULTS.maxIdleTimeout,
  );
  quicParams.maxIncomingStreams = nonNegativeInteger(
    input.maxIncomingStreams,
    HYSTERIA_QUIC_DEFAULTS.maxIncomingStreams,
  );

  if (input.quicParamsEnabled && input.udpHopEnabled && ports) {
    const hop: Record<string, unknown> = { ...existingHop, ports };
    if (interval) hop.interval = interval;
    else delete hop.interval;
    quicParams.udpHop = hop;
    finalmask.quicParams = quicParams;
    next.finalmask = finalmask;
    return next;
  }

  delete quicParams.udpHop;
  if (Object.keys(quicParams).length > 0) {
    finalmask.quicParams = quicParams;
  } else {
    delete finalmask.quicParams;
  }
  if (Object.keys(finalmask).length > 0) {
    next.finalmask = finalmask;
  } else {
    delete next.finalmask;
  }
  return next;
}

/** Read both historical and current intervalRemote hop settings for the form. */
export function resolveHysteriaUdpHop(finalmask: unknown): Record<string, unknown> {
  const settings = asRecord(finalmask);
  const masks = Array.isArray(settings.udp) ? settings.udp : [];
  const current = masks.find(isEditableUdpHopMask);
  if (current) {
    const hop = asRecord(asRecord(current).settings);
    return { ...hop, ports: hop.remotePorts };
  }
  return asRecord(asRecord(settings.quicParams).udpHop);
}

function isEditableUdpHopMask(value: unknown): boolean {
  const mask = asRecord(value);
  return (
    String(mask.type).toLowerCase() === 'udphop' &&
    String(asRecord(mask.settings).mode).toLowerCase() === 'intervalremote'
  );
}

export function validateHysteriaQuicFormInput(input: HysteriaUdpHopFormInput): string {
  if (!input.quicParamsEnabled) {
    return '';
  }

  const integerChecks: Array<[string, number]> = [
    ['initial stream receive window', input.initStreamReceiveWindow],
    ['max stream receive window', input.maxStreamReceiveWindow],
    ['initial connection receive window', input.initConnectionReceiveWindow],
    ['max connection receive window', input.maxConnectionReceiveWindow],
    ['max idle timeout', input.maxIdleTimeout],
    ['max incoming streams', input.maxIncomingStreams],
  ];
  for (const [label, value] of integerChecks) {
    if (!Number.isSafeInteger(value) || value < 0) {
      return `Hysteria QUIC ${label} must be a non-negative integer`;
    }
  }

  const windowChecks: Array<[string, number]> = [
    ['initial stream receive window', input.initStreamReceiveWindow],
    ['max stream receive window', input.maxStreamReceiveWindow],
    ['initial connection receive window', input.initConnectionReceiveWindow],
    ['max connection receive window', input.maxConnectionReceiveWindow],
  ];
  for (const [label, value] of windowChecks) {
    if (value > 0 && value < 16_384) {
      return `Hysteria QUIC ${label} must be 0 or at least 16384`;
    }
  }
  if (input.maxIdleTimeout !== 0 && (input.maxIdleTimeout < 4 || input.maxIdleTimeout > 120)) {
    return 'Hysteria QUIC max idle timeout must be 0 or between 4 and 120';
  }
  if (input.maxIncomingStreams > 0 && input.maxIncomingStreams < 8) {
    return 'Hysteria QUIC max incoming streams must be 0 or at least 8';
  }
  return '';
}

const XHTTP_LEGACY_EXTRA_KEYS = [
  'headers',
  'scMaxBufferedPosts',
  'scMaxEachPostBytes',
  'scStreamUpServerSecs',
  'noSSEHeader',
  'serverMaxHeaderBytes',
  'xPaddingBytes',
  'xPaddingObfsMode',
  'xPaddingKey',
  'xPaddingHeader',
  'xPaddingPlacement',
  'xPaddingMethod',
  'uplinkHTTPMethod',
  'sessionPlacement',
  'sessionKey',
  'seqPlacement',
  'seqKey',
  'uplinkDataPlacement',
  'uplinkDataKey',
  'uplinkChunkSize',
  'noGRPCHeader',
  'scMinPostsIntervalMs',
  'xmux',
  'downloadSettings',
] as const;

const XHTTP_SESSION_EXTRA_KEYS = [
  'sessionIDPlacement',
  'sessionIDKey',
  'sessionIDTable',
  'sessionIDLength',
] as const;

export function resolveXhttpExtraSettings(
  settings: Record<string, unknown>,
): Record<string, unknown> {
  const extra = normalizeXhttpCompatibility({ ...asRecord(settings.extra) });
  for (const key of XHTTP_LEGACY_EXTRA_KEYS) {
    if (extra[key] === undefined && settings[key] !== undefined) {
      extra[key] = settings[key];
    }
  }
  for (const key of XHTTP_SESSION_EXTRA_KEYS) {
    if (extra[key] === undefined && settings[key] !== undefined) {
      extra[key] = settings[key];
    }
  }
  return normalizeXhttpCompatibility(extra);
}

export function resolveXhttpHost(settings: Record<string, unknown>): string {
  const directHost = stringValue(settings.host).trim();
  if (directHost) {
    return directHost;
  }
  const headers = asRecord(resolveXhttpExtraSettings(settings).headers);
  const hostKey = Object.keys(headers).find((key) => key.toLowerCase() === 'host');
  return hostKey ? stringValue(headers[hostKey]).trim() : '';
}

export function mergeXhttpSettings(
  settings: Record<string, unknown>,
  input: XhttpFormInput,
): Record<string, unknown> {
  const next = { ...settings };
  const extra = resolveXhttpExtraSettings(settings);
  for (const key of [...XHTTP_LEGACY_EXTRA_KEYS, ...XHTTP_SESSION_EXTRA_KEYS]) {
    delete next[key];
  }

  next.path = input.path.trim() || '/';
  setOptionalString(next, 'host', input.host);
  next.mode = input.mode || 'auto';

  const headers = { ...asRecord(extra.headers) };
  const hostHeaderKey = Object.keys(headers).find((key) => key.toLowerCase() === 'host');
  if (hostHeaderKey) {
    setOptionalString(headers, hostHeaderKey, input.host);
  }
  if (Object.keys(headers).length > 0) {
    extra.headers = headers;
  } else {
    delete extra.headers;
  }

  setOptionalNonNegativeInteger(extra, 'scMaxBufferedPosts', input.scMaxBufferedPosts);
  setOptionalString(extra, 'scMaxEachPostBytes', input.scMaxEachPostBytes);
  setOptionalString(extra, 'scStreamUpServerSecs', input.scStreamUpServerSecs);
  setOptionalBoolean(extra, 'noSSEHeader', input.noSSEHeader);
  setOptionalString(extra, 'xPaddingBytes', input.xPaddingBytes);
  setOptionalBoolean(extra, 'xPaddingObfsMode', input.xPaddingObfsMode);
  setOptionalString(extra, 'xPaddingKey', input.xPaddingKey);
  setOptionalString(extra, 'xPaddingHeader', input.xPaddingHeader);
  setOptionalString(extra, 'xPaddingPlacement', input.xPaddingPlacement);
  setOptionalString(extra, 'xPaddingMethod', input.xPaddingMethod);
  setOptionalString(extra, 'uplinkHTTPMethod', input.uplinkHTTPMethod.toUpperCase());
  setOptionalString(extra, 'sessionIDPlacement', input.sessionPlacement);
  setOptionalString(extra, 'sessionIDKey', input.sessionKey);
  if (input.sessionIDTable !== undefined) {
    setOptionalString(extra, 'sessionIDTable', input.sessionIDTable);
  }
  if (input.sessionIDLength !== undefined) {
    setOptionalString(extra, 'sessionIDLength', input.sessionIDLength);
  }
  setOptionalString(extra, 'seqPlacement', input.seqPlacement);
  setOptionalString(extra, 'seqKey', input.seqKey);
  setOptionalString(extra, 'uplinkDataPlacement', input.uplinkDataPlacement);
  setOptionalString(extra, 'uplinkDataKey', input.uplinkDataKey);
  setOptionalNonNegativeInteger(extra, 'uplinkChunkSize', input.uplinkChunkSize);

  if (input.xmuxEnabled) {
    const xmux = { ...asRecord(extra.xmux) };
    setOptionalString(xmux, 'maxConcurrency', input.xmuxMaxConcurrency);
    setOptionalString(xmux, 'maxConnections', input.xmuxMaxConnections);
    if (input.xmuxMaxConnections.trim()) {
      delete xmux.maxConcurrency;
    } else if (input.xmuxMaxConcurrency.trim()) {
      delete xmux.maxConnections;
    }
    setOptionalString(xmux, 'cMaxReuseTimes', input.xmuxCMaxReuseTimes);
    setOptionalString(xmux, 'hMaxRequestTimes', input.xmuxHMaxRequestTimes);
    setOptionalString(xmux, 'hMaxReusableSecs', input.xmuxHMaxReusableSecs);
    setOptionalNonNegativeInteger(xmux, 'hKeepAlivePeriod', input.xmuxHKeepAlivePeriod);
    extra.xmux = xmux;
  } else {
    delete extra.xmux;
  }

  if (Object.keys(extra).length > 0) {
    next.extra = extra;
  } else {
    delete next.extra;
  }
  return next;
}

/** Validate v26.10.10 XHTTP session table/range when present in imported JSON. */
export function validateXhttpSessionSettings(settings: Record<string, unknown>): string {
  const normalized = resolveXhttpExtraSettings(settings);
  if (
    normalized.sessionIDLength !== undefined &&
    typeof normalized.sessionIDLength !== 'string' &&
    typeof normalized.sessionIDLength !== 'number'
  ) {
    return 'XHTTP sessionIDLength must be a positive length or range';
  }
  const table = stringValue(normalized.sessionIDTable).trim();
  const length =
    normalized.sessionIDLength === undefined ? '' : String(normalized.sessionIDLength).trim();
  if (table && Array.from(table).some((character) => character.charCodeAt(0) >= 128)) {
    return 'XHTTP sessionIDTable must contain only ASCII characters';
  }
  if (!length) return '';
  const range = parseCoreInt32Range(length);
  if (!range) {
    return 'XHTTP sessionIDLength must be a positive length or range';
  }
  if (range[0] <= 0 || range[1] <= 0) {
    return 'XHTTP sessionIDLength range must be positive';
  }
  return '';
}

export function validateXhttpFormInput(input: XhttpFormInput): string {
  if (hasControlCharacter(input.path) || hasControlCharacter(input.host)) {
    return 'XHTTP path and host must not contain control characters';
  }
  if (!['auto', 'packet-up', 'stream-up', 'stream-one'].includes(input.mode)) {
    return `Unsupported XHTTP mode: ${input.mode}`;
  }
  const optionChecks: Array<[string, string, readonly string[]]> = [
    [
      'padding placement',
      input.xPaddingPlacement,
      ['', 'cookie', 'header', 'query', 'queryInHeader'],
    ],
    ['padding method', input.xPaddingMethod, ['', 'repeat-x', 'tokenish']],
    ['uplink HTTP method', input.uplinkHTTPMethod, ['', 'POST', 'PUT', 'GET']],
    ['session placement', input.sessionPlacement, ['', 'path', 'header', 'cookie', 'query']],
    ['sequence placement', input.seqPlacement, ['', 'path', 'header', 'cookie', 'query']],
    ['uplink data placement', input.uplinkDataPlacement, ['', 'auto', 'body', 'cookie', 'header']],
  ];
  for (const [label, value, allowed] of optionChecks) {
    if (!allowed.includes(value)) {
      return `Unsupported XHTTP ${label}: ${value}`;
    }
  }
  if (input.uplinkHTTPMethod === 'GET' && input.mode !== 'packet-up') {
    return 'XHTTP uplink HTTP method GET is only supported in packet-up mode';
  }
  if (['cookie', 'header'].includes(input.uplinkDataPlacement) && input.mode !== 'packet-up') {
    return `XHTTP uplink data placement ${input.uplinkDataPlacement} is only supported in packet-up mode`;
  }
  const integerChecks: Array<[string, number]> = [
    ['max buffered posts', input.scMaxBufferedPosts],
    ['uplink chunk size', input.uplinkChunkSize],
    ['XMUX keep-alive period', input.xmuxHKeepAlivePeriod],
  ];
  for (const [label, value] of integerChecks) {
    if (!Number.isSafeInteger(value) || value < 0) {
      return `XHTTP ${label} must be a non-negative integer`;
    }
  }
  if (input.uplinkChunkSize > 2_147_483_647) {
    return 'XHTTP uplink chunk size must fit a signed 32-bit integer';
  }
  const rangeChecks: Array<[string, string, boolean]> = [
    ['each post bytes', input.scMaxEachPostBytes, false],
    ['stream-up server seconds', input.scStreamUpServerSecs, false],
    ['padding bytes', input.xPaddingBytes, true],
    ['XMUX max concurrency', input.xmuxMaxConcurrency, false],
    ['XMUX max connections', input.xmuxMaxConnections, false],
    ['XMUX max reuse times', input.xmuxCMaxReuseTimes, false],
    ['XMUX max request times', input.xmuxHMaxRequestTimes, false],
    ['XMUX max reusable seconds', input.xmuxHMaxReusableSecs, false],
  ];
  for (const [label, value, positive] of rangeChecks) {
    const trimmed = value.trim();
    if (!trimmed) {
      continue;
    }
    const range = parseCoreInt32Range(trimmed);
    if (!range) {
      return `XHTTP ${label} must be an integer or integer range`;
    }
    if (positive && (range[0] <= 0 || range[1] <= 0)) {
      return `XHTTP ${label} values must be greater than 0`;
    }
  }
  const textChecks = [
    input.scMaxEachPostBytes,
    input.scStreamUpServerSecs,
    input.xPaddingBytes,
    input.xPaddingKey,
    input.xPaddingHeader,
    input.sessionKey,
    input.seqKey,
    input.uplinkDataKey,
    input.xmuxMaxConcurrency,
    input.xmuxMaxConnections,
    input.xmuxCMaxReuseTimes,
    input.xmuxHMaxRequestTimes,
    input.xmuxHMaxReusableSecs,
  ];
  if (textChecks.some((value) => value.length > 256 || hasControlCharacter(value))) {
    return 'XHTTP text values must be at most 256 characters without control characters';
  }
  return '';
}

export function normalizeTunSettings(settings: InboundSettings): InboundSettings {
  const normalized = { ...settings };
  const rawMtu = normalized.mtu ?? normalized.MTU;
  const mtu = Array.isArray(rawMtu) ? rawMtu[0] : rawMtu;
  const autoOutboundsInterface = stringValue(normalized.autoOutboundsInterface).trim();

  normalized.name = stringValue(normalized.name).trim() || 'xray0';
  normalized.mtu = mtu === undefined || mtu === null || mtu === '' ? 1500 : Number(mtu);
  normalized.gateway = normalizeStringList(normalized.gateway ?? normalized.Gateway);
  normalized.dns = normalizeStringList(normalized.dns ?? normalized.DNS);
  normalized.userLevel =
    normalized.userLevel === undefined || normalized.userLevel === null
      ? 0
      : Number(normalized.userLevel);
  normalized.autoSystemRoutingTable = normalizeStringList(normalized.autoSystemRoutingTable);
  if (autoOutboundsInterface) {
    normalized.autoOutboundsInterface = autoOutboundsInterface;
  } else {
    delete normalized.autoOutboundsInterface;
  }
  delete normalized.MTU;
  delete normalized.Gateway;
  delete normalized.DNS;
  return normalized;
}

export function validateTunSettings(settings: InboundSettings): string {
  const name = stringValue(settings.name).trim();
  if (!name || name.length > 128 || hasControlCharacter(name)) {
    return 'TUN interface name must be 1-128 characters without control characters';
  }

  const mtu = Number(settings.mtu);
  if (!Number.isInteger(mtu) || mtu < 1 || mtu > 9_000) {
    return 'TUN MTU must be an integer between 1 and 9000';
  }

  const userLevel = Number(settings.userLevel);
  if (!Number.isInteger(userLevel) || userLevel < 0 || userLevel > 4_294_967_295) {
    return 'TUN user level must be an unsigned 32-bit integer';
  }

  for (const gateway of normalizeStringList(settings.gateway)) {
    if (!isValidIpPrefix(gateway)) {
      return `Invalid TUN gateway CIDR: ${gateway}`;
    }
  }
  for (const dns of normalizeStringList(settings.dns)) {
    if (!isValidIpAddress(dns)) {
      return `Invalid TUN DNS address: ${dns}`;
    }
  }
  for (const route of normalizeStringList(settings.autoSystemRoutingTable)) {
    if (!isValidIpPrefix(route)) {
      return `Invalid TUN system route CIDR: ${route}`;
    }
  }

  const outboundInterface = stringValue(settings.autoOutboundsInterface).trim();
  if (outboundInterface.length > 128 || hasControlCharacter(outboundInterface)) {
    return 'TUN outbound interface must be at most 128 characters without control characters';
  }
  return '';
}

export function defaultSniffingSettings(): InboundSniffingSettings {
  return {
    enabled: false,
    destOverride: ['http', 'tls', 'quic', 'fakedns'],
    metadataOnly: false,
    routeOnly: false,
  };
}

export function stringifyJson(value: unknown): string {
  return JSON.stringify(value, null, 2);
}

function defaultProxyAccount(): { user: string; pass: string; subId: string } {
  return {
    user: randomLowerToken(10),
    pass: randomLowerToken(10),
    subId: randomLowerToken(16),
  };
}

function parseJsonObject<T extends object>(text: string, fallback: T): T {
  if (!text.trim()) {
    return fallback;
  }
  try {
    const parsed = JSON.parse(text) as unknown;
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as T;
    }
  } catch {
    return fallback;
  }
  return fallback;
}

export function resolveInboundHost(inbound: Pick<Inbound, 'listen'>): string {
  const listen = inbound.listen?.trim();
  if (listen && !WILDCARD_LISTENS.has(listen)) {
    return listen;
  }
  return window.location.hostname || '127.0.0.1';
}

export function getInboundNetwork(inbound: Pick<Inbound, 'streamSettings'>): string {
  return parseInboundStreamSettings(inbound).network || 'tcp';
}

export function getInboundSecurity(inbound: Pick<Inbound, 'streamSettings'>): string {
  return parseInboundStreamSettings(inbound).security || 'none';
}

export function getShadowsocksMethod(source: Pick<Inbound, 'settings'> | InboundSettings): string {
  const settings = hasSettingsText(source) ? parseInboundSettings(source) : source;
  const method = settings.method || SHADOWSOCKS_DEFAULT_METHOD;
  return normalizeShadowsocksMethod(String(method));
}

/** Xray-core no longer accepts these legacy Shadowsocks cipher names. */
export function validateShadowsocksMethod(method: string): string {
  const normalized = normalizeShadowsocksMethod(method);
  if (REMOVED_SHADOWSOCKS_METHODS.has(normalized)) {
    return `Shadowsocks cipher ${normalized} was removed from the target Xray core; choose a supported method`;
  }
  return '';
}

export function isShadowsocks2022Method(method: string): boolean {
  return normalizeShadowsocksMethod(method).startsWith(SHADOWSOCKS_2022_PREFIX);
}

export function isSingleUserShadowsocks2022(method: string): boolean {
  return normalizeShadowsocksMethod(method) === SHADOWSOCKS_2022_SINGLE_USER;
}

function shadowsocksKeyBytes(method: string): number {
  return normalizeShadowsocksMethod(method) === SHADOWSOCKS_2022_AES_128_GCM
    ? 16
    : SHADOWSOCKS_DEFAULT_KEY_BYTES;
}

export function generateShadowsocksPassword(method: string): string {
  if (isShadowsocks2022Method(method)) {
    return base64Bytes(randomBytes(shadowsocksKeyBytes(method)));
  }
  return base64UrlBytes(randomBytes(SHADOWSOCKS_DEFAULT_KEY_BYTES));
}

export function getClientPrimaryId(protocol: string, client: InboundClient): string {
  if (protocol === 'trojan') {
    return String(client.password || '');
  }
  if (protocol === 'shadowsocks') {
    return String(client.email || '');
  }
  if (protocol === 'hysteria' || protocol === 'hysteria2') {
    return String(client.auth || '');
  }
  if (protocol === 'wireguard') {
    return String(client.publicKey || '');
  }
  return String(client.id || '');
}

export function buildClientShareLink(
  inbound: Inbound,
  client: InboundClient,
  options: ShareLinkBuildOptions = {},
): string {
  if (!getClientPrimaryId(inbound.protocol, client)) {
    return '';
  }

  if (inbound.protocol === 'vmess') {
    return buildVmessShareLink(inbound, client, options);
  }
  if (inbound.protocol === 'vless') {
    return buildVlessShareLink(inbound, client, options);
  }
  if (inbound.protocol === 'trojan') {
    return buildTrojanShareLink(inbound, client, options);
  }
  if (inbound.protocol === 'shadowsocks') {
    return buildShadowsocksShareLink(inbound, client, options);
  }
  if (inbound.protocol === 'hysteria' || inbound.protocol === 'hysteria2') {
    return buildHysteriaShareLink(inbound, client, options);
  }
  if (inbound.protocol === 'wireguard') {
    return buildWireguardShareLink(inbound, client, options);
  }
  return '';
}

export function buildInboundShareLinks(inbound: Inbound): string[] {
  if (inbound.protocol === 'http' || inbound.protocol === 'mixed') {
    return buildProxyAccountShareLinks(inbound);
  }

  const targetsForClient = (clientEmail = '') => buildShareLinkTargets(inbound, clientEmail);

  if (
    inbound.protocol === 'shadowsocks' &&
    isSingleUserShadowsocks2022(getShadowsocksMethod(inbound))
  ) {
    return targetsForClient()
      .map((target) => buildSingleUserShadowsocksShareLink(inbound, target))
      .filter(hasText);
  }

  return getInboundClients(inbound)
    .flatMap((client) =>
      targetsForClient(client.email).map((target) => buildClientShareLink(inbound, client, target)),
    )
    .filter(hasText);
}

export function buildClientSubscriptionLinks(
  client: Pick<InboundClient, 'subId'>,
  settings: SubscriptionEndpointSettings,
): SubscriptionLinkItem[] {
  const subId = String(client.subId || '').trim();
  if (!settings.subEnable || !subId) {
    return [];
  }

  const links: SubscriptionLinkItem[] = [];
  const uri = appendSubscriptionId(settings.subURI, subId);
  if (uri) {
    links.push({ label: 'URI', url: uri });
  }

  if (settings.subJsonEnable) {
    const jsonUri = appendSubscriptionId(settings.subJsonURI, subId);
    if (jsonUri) {
      links.push({ label: 'JSON', url: jsonUri });
    }
  }

  if (settings.subClashEnable) {
    const clashUri = appendSubscriptionId(settings.subClashURI, subId);
    if (clashUri) {
      links.push({ label: 'Clash', url: clashUri });
    }
  }

  return links;
}

export function generateBulkClientProfiles(input: BulkClientGenerationInput): InboundClient[] {
  const quantity = Math.max(1, Math.min(500, Math.trunc(input.quantity || 1)));
  const firstIndex = Math.max(1, Math.trunc(input.firstIndex || 1));
  const prefix = input.emailPrefix || '';
  const postfix = input.emailPostfix || '';

  return Array.from({ length: quantity }, (_, index) => {
    const email = `${prefix}${firstIndex + index}${postfix}`;
    const client: InboundClient = {
      email,
      limitIp: Math.max(0, Number(input.limitIp || 0)),
      totalGB: Math.max(0, Number(input.totalGB || 0)),
      expiryTime: Math.max(0, Number(input.expiryTime || 0)),
      enable: true,
      subId: randomLowerToken(16),
      reset: Math.max(0, Number(input.reset || 0)),
    };

    if (input.protocol === 'vmess') {
      client.id = randomUuid();
      client.security = input.security || 'auto';
      return client;
    }

    if (input.protocol === 'vless') {
      client.id = randomUuid();
      client.flow = input.flow || '';
      return client;
    }

    if (input.protocol === 'trojan') {
      client.password = randomLowerToken(32);
      return client;
    }

    if (input.protocol === 'shadowsocks') {
      const method = normalizeShadowsocksMethod(
        input.shadowsocksMethod || SHADOWSOCKS_DEFAULT_METHOD,
      );
      client.password = generateShadowsocksPassword(method);
      if (!isShadowsocks2022Method(method)) {
        client.method = method;
      }
      return client;
    }

    if (input.protocol === 'hysteria' || input.protocol === 'hysteria2') {
      client.auth = randomLowerToken(32);
      return client;
    }

    return client;
  });
}

function buildVmessShareLink(
  inbound: Inbound,
  client: InboundClient,
  options: ShareLinkBuildOptions,
): string {
  const stream = parseInboundStreamSettings(inbound);
  const security = resolveShareSecurity(stream, options.forceTls);
  const payload = {
    v: '2',
    ps: shareLinkRemark(inbound, client, options),
    add: shareLinkAddress(inbound, options),
    port: String(shareLinkPort(inbound, options)),
    id: client.id,
    aid: '0',
    scy: client.security || 'auto',
    net: stream.network || 'tcp',
    type: 'none',
    host: '',
    path: '',
    tls: security === 'tls' ? 'tls' : '',
    sni: '',
  };

  return `vmess://${base64Utf8(JSON.stringify(payload))}`;
}

function buildVlessShareLink(
  inbound: Inbound,
  client: InboundClient,
  options: ShareLinkBuildOptions,
): string {
  const stream = parseInboundStreamSettings(inbound);
  const params = new URLSearchParams();
  params.set('encryption', 'none');
  appendSecurityParams(params, stream, resolveShareSecurity(stream, options.forceTls));
  params.set('type', stream.network || 'tcp');
  if (client.flow) {
    params.set('flow', client.flow);
  }

  const label = encodeURIComponent(shareLinkRemark(inbound, client, options));
  return `vless://${client.id}@${shareLinkAddress(inbound, options)}:${shareLinkPort(
    inbound,
    options,
  )}?${params.toString()}#${label}`;
}

function buildTrojanShareLink(
  inbound: Inbound,
  client: InboundClient,
  options: ShareLinkBuildOptions,
): string {
  if (!client.password) {
    return '';
  }

  const stream = parseInboundStreamSettings(inbound);
  const params = new URLSearchParams();
  appendSecurityParams(params, stream, resolveShareSecurity(stream, options.forceTls));
  params.set('type', stream.network || 'tcp');

  const label = encodeURIComponent(shareLinkRemark(inbound, client, options));
  return `trojan://${encodeURIComponent(client.password)}@${shareLinkAddress(
    inbound,
    options,
  )}:${shareLinkPort(inbound, options)}?${params.toString()}#${label}`;
}

function buildShadowsocksShareLink(
  inbound: Inbound,
  client: InboundClient,
  options: ShareLinkBuildOptions,
): string {
  if (!client.password) {
    return '';
  }

  const settings = parseInboundSettings(inbound);
  const method = normalizeShadowsocksMethod(client.method || getShadowsocksMethod(settings));
  if (!method) {
    return '';
  }

  const passwordParts = [method];
  if (isShadowsocks2022Method(method) && settings.password) {
    passwordParts.push(String(settings.password));
  }
  passwordParts.push(client.password);

  const label = encodeURIComponent(shareLinkRemark(inbound, client, options));
  const userInfo = base64UrlUtf8(passwordParts.join(':'));
  return `ss://${userInfo}@${shareLinkAddress(inbound, options)}:${shareLinkPort(
    inbound,
    options,
  )}#${label}`;
}

function buildSingleUserShadowsocksShareLink(
  inbound: Inbound,
  options: ShareLinkBuildOptions,
): string {
  const settings = parseInboundSettings(inbound);
  const method = normalizeShadowsocksMethod(getShadowsocksMethod(settings));
  const serverPassword = String(settings.password || '');
  if (!method || !serverPassword) {
    return '';
  }

  const label = encodeURIComponent(shareLinkRemark(inbound, undefined, options));
  const userInfo = base64UrlUtf8(`${method}:${serverPassword}`);
  return `ss://${userInfo}@${shareLinkAddress(inbound, options)}:${shareLinkPort(
    inbound,
    options,
  )}#${label}`;
}

function buildHysteriaShareLink(
  inbound: Inbound,
  client: InboundClient,
  options: ShareLinkBuildOptions,
): string {
  if (!client.auth) {
    return '';
  }

  const stream = parseInboundStreamSettings(inbound);
  const tlsSettings = asRecord(stream.tlsSettings);
  const tlsClientSettings = asRecord(tlsSettings.settings);
  const params = new URLSearchParams();
  params.set('security', 'tls');
  setParamIfPresent(params, 'sni', stringValue(tlsSettings.serverName));
  setParamIfPresent(params, 'fp', stringValue(tlsClientSettings.fingerprint));
  const alpn = arrayOrCsv(tlsSettings.alpn);
  if (alpn) {
    params.set('alpn', alpn);
  }
  appendHysteriaFinalMaskParams(params, stream.finalmask);

  const label = encodeURIComponent(shareLinkRemark(inbound, client, options));
  return `hysteria2://${encodeURIComponent(client.auth)}@${shareLinkAddress(
    inbound,
    options,
  )}:${shareLinkPort(inbound, options)}?${params.toString()}#${label}`;
}

function buildWireguardShareLink(
  inbound: Inbound,
  client: InboundClient,
  options: ShareLinkBuildOptions,
): string {
  if (!client.privateKey) {
    return '';
  }

  const settings = parseInboundSettings(inbound);
  const serverPublicKey =
    settings.pubKey ||
    (settings.secretKey ? generateWireguardKeypair(settings.secretKey).publicKey : '');
  if (!serverPublicKey) {
    return '';
  }

  const url = new URL(
    `wireguard://${shareLinkAddress(inbound, options)}:${shareLinkPort(inbound, options)}`,
  );
  url.username = client.privateKey;
  url.searchParams.set('publickey', serverPublicKey);
  const allowedIp = client.allowedIPs?.find((item) => item.trim());
  if (allowedIp) {
    url.searchParams.set('address', normalizeAllowedIp(allowedIp));
  }
  if (settings.mtu) {
    url.searchParams.set('mtu', String(settings.mtu));
  }
  if (client.preSharedKey) {
    url.searchParams.set('presharedkey', client.preSharedKey);
  }
  if (client.keepAlive) {
    url.searchParams.set('keepalive', String(client.keepAlive));
  }
  url.hash = encodeURIComponent(shareLinkRemark(inbound, client, options));
  return url.toString();
}

function buildProxyAccountShareLinks(inbound: Inbound): string[] {
  const settings = parseInboundSettings(inbound);
  const accounts = Array.isArray(settings.accounts)
    ? settings.accounts.map(normalizeProxyAccount).filter(Boolean)
    : [];
  if (accounts.length === 0 && settings.subId) {
    accounts.push({});
  }
  return accounts
    .filter((account) => account.enable !== false && account.enabled !== false)
    .map((account) => buildProxyAccountShareLink(inbound, account))
    .filter(hasText);
}

function normalizeProxyAccount(value: unknown): ProxyAccount {
  const account = asRecord(value);
  return {
    user: stringValue(account.user || account.username),
    pass: stringValue(account.pass || account.password),
    subId: stringValue(account.subId || account.subID || account.sub_id),
    remark: stringValue(account.remark || account.name),
    email: stringValue(account.email),
    comment: stringValue(account.comment),
    enable: typeof account.enable === 'boolean' ? account.enable : undefined,
    enabled: typeof account.enabled === 'boolean' ? account.enabled : undefined,
  };
}

function buildProxyAccountShareLink(inbound: Inbound, account: ProxyAccount): string {
  const scheme = inbound.protocol === 'mixed' ? 'socks5' : 'http';
  const auth =
    account.user || account.pass
      ? `${encodeURIComponent(account.user || '')}${
          account.pass ? `:${encodeURIComponent(account.pass)}` : ''
        }@`
      : '';
  const remark =
    account.remark ||
    account.email ||
    account.comment ||
    inbound.remark ||
    inbound.tag ||
    `inbound-${inbound.port}`;
  return `${scheme}://${auth}${shareLinkAddress(inbound, {})}:${shareLinkPort(
    inbound,
    {},
  )}#${encodeURIComponent(remark)}`;
}

function buildShareLinkTargets(inbound: Inbound, clientEmail: string): ShareLinkTarget[] {
  const stream = parseInboundStreamSettings(inbound);
  const proxies = Array.isArray(stream.externalProxy)
    ? stream.externalProxy.map(normalizeExternalProxy).filter((proxy) => proxy.address)
    : [];

  if (proxies.length === 0) {
    return [
      {
        address: resolveInboundHost(inbound),
        port: inbound.port,
        forceTls: 'same',
        remark: formatShareRemark(inbound, clientEmail),
      },
    ];
  }

  return proxies.map((proxy) => ({
    address: proxy.address,
    port: proxy.port || inbound.port,
    forceTls: proxy.forceTls || 'same',
    remark: formatShareRemark(inbound, clientEmail, proxy.remark),
  }));
}

function normalizeExternalProxy(value: unknown): {
  address: string;
  port: number;
  forceTls: string;
  remark: string;
} {
  const proxy = asRecord(value);
  return {
    address: stringValue(proxy.dest).trim(),
    port: Number(proxy.port || 0),
    forceTls: stringValue(proxy.forceTls || 'same'),
    remark: stringValue(proxy.remark).trim(),
  };
}

function formatShareRemark(inbound: Inbound, clientEmail = '', proxyRemark = ''): string {
  const inboundRemark = inbound.remark || inbound.tag || `inbound-${inbound.port}`;
  return [inboundRemark, clientEmail, proxyRemark]
    .map((part) => part.trim())
    .filter((part) => part.length > 0)
    .join('-');
}

function shareLinkAddress(inbound: Inbound, options: ShareLinkBuildOptions): string {
  return options.address || resolveInboundHost(inbound);
}

function shareLinkPort(inbound: Inbound, options: ShareLinkBuildOptions): number {
  return Number(options.port || inbound.port);
}

function shareLinkRemark(
  inbound: Inbound,
  client: InboundClient | undefined,
  options: ShareLinkBuildOptions,
): string {
  return (
    options.remark || client?.email || inbound.remark || inbound.tag || `inbound-${inbound.port}`
  );
}

function resolveShareSecurity(stream: InboundStreamSettings, forceTls: string | undefined): string {
  if (forceTls && forceTls !== 'same') {
    return forceTls;
  }
  return stream.security || 'none';
}

function appendSecurityParams(
  params: URLSearchParams,
  stream: InboundStreamSettings,
  security: string,
) {
  params.set('security', security || 'none');
  if (security === 'tls') {
    appendTlsParams(params, stream);
    return;
  }
  if (security === 'reality') {
    appendRealityParams(params, stream);
  }
}

function appendTlsParams(params: URLSearchParams, stream: InboundStreamSettings) {
  const tlsSettings = asRecord(stream.tlsSettings);
  const tlsClientSettings = asRecord(tlsSettings.settings);
  setParamIfPresent(params, 'fp', stringValue(tlsClientSettings.fingerprint));
  setParamIfPresent(params, 'sni', stringValue(tlsSettings.serverName));
  const alpn = arrayOrCsv(tlsSettings.alpn);
  if (alpn) {
    params.set('alpn', alpn);
  }
  setParamIfPresent(params, 'ech', stringValue(tlsClientSettings.echConfigList));
}

function appendRealityParams(params: URLSearchParams, stream: InboundStreamSettings) {
  const realitySettings = asRecord(stream.realitySettings);
  const realityClientSettings = asRecord(realitySettings.settings);
  setParamIfPresent(params, 'pbk', stringValue(realityClientSettings.publicKey));
  setParamIfPresent(params, 'fp', stringValue(realityClientSettings.fingerprint));
  setParamIfPresent(params, 'sni', firstValue(realitySettings.serverNames));
  setParamIfPresent(params, 'sid', firstValue(realitySettings.shortIds));
  setParamIfPresent(params, 'spx', stringValue(realityClientSettings.spiderX));
  setParamIfPresent(params, 'pqv', stringValue(realityClientSettings.mldsa65Verify));
}

function defaultWireguardSettings(): InboundSettings {
  const server = generateWireguardKeypair();
  return {
    mtu: 1420,
    secretKey: server.privateKey,
    pubKey: server.publicKey,
    peers: [generateWireguardPeer(0)],
    noKernelTun: false,
  };
}

export function generateWireguardPeer(index = 0): InboundClient {
  const keypair = generateWireguardKeypair();
  return {
    email: `wg-${randomLowerToken(8)}`,
    privateKey: keypair.privateKey,
    publicKey: keypair.publicKey,
    allowedIPs: [`10.0.0.${index + 2}/32`],
    keepAlive: 0,
    enable: true,
    subId: randomLowerToken(16),
  };
}

export function generateWireguardPresharedKey(): string {
  return base64Bytes(randomBytes(32));
}

export function generateWireguardKeypair(secretKey = ''): {
  privateKey: string;
  publicKey: string;
} {
  return WireguardKeyUtil.generateKeypair(secretKey);
}

function base64Utf8(value: string): string {
  const bytes = new TextEncoder().encode(value);
  return base64Bytes(bytes);
}

function base64UrlUtf8(value: string): string {
  const bytes = new TextEncoder().encode(value);
  return base64UrlBytes(bytes);
}

function base64Bytes(bytes: Uint8Array): string {
  let binary = '';
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary);
}

function base64UrlBytes(bytes: Uint8Array): string {
  return base64Bytes(bytes).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function normalizeShadowsocksMethod(method: string): string {
  return method.trim().toLowerCase().replaceAll('_', '-');
}

function randomBytes(length: number): Uint8Array {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return bytes;
}

function defaultWireguardPeerFromValue(value: InboundClient): InboundClient {
  return {
    ...value,
    email: value.email || value.publicKey || 'wireguard-peer',
    allowedIPs: Array.isArray(value.allowedIPs) ? value.allowedIPs : [],
    enable: value.enable !== false,
  };
}

function isWireguardPeer(value: unknown): value is InboundClient {
  if (!value || typeof value !== 'object') {
    return false;
  }
  const peer = defaultWireguardPeerFromValue(value as InboundClient);
  Object.assign(value, peer);
  return Boolean(peer.publicKey || peer.privateKey || peer.email);
}

function setParamIfPresent(params: URLSearchParams, key: string, value: string) {
  if (value) {
    params.set(key, value);
  }
}

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function appendHysteriaFinalMaskParams(params: URLSearchParams, finalmask: unknown) {
  const normalizedFinalmask = normalizeFinalmaskForXray({
    finalmask: asRecord(finalmask),
  }).finalmask;
  const serialized = serializeShareableFinalMask(normalizedFinalmask);
  if (serialized) {
    params.set('fm', serialized);
  }

  const obfsPassword = hysteriaSalamanderPassword(normalizedFinalmask);
  if (obfsPassword) {
    params.set('obfs', 'salamander');
    params.set('obfs-password', obfsPassword);
  }

  const hopPorts = hysteriaHopPorts(normalizedFinalmask);
  if (hopPorts) {
    params.set('mport', hopPorts);
  }
}

function hysteriaSalamanderPassword(finalmask: unknown): string {
  const udpMasks = asRecord(finalmask).udp;
  if (!Array.isArray(udpMasks)) {
    return '';
  }

  for (const rawMask of udpMasks) {
    const mask = asRecord(rawMask);
    if (mask.type !== 'salamander') {
      continue;
    }
    const password = stringValue(asRecord(mask.settings).password).trim();
    if (password) {
      return password;
    }
  }
  return '';
}

function hysteriaHopPorts(finalmask: unknown): string {
  const raw = asRecord(finalmask);
  const quicParams = asRecord(raw.quicParams);
  const udpHop = asRecord(quicParams.udpHop);
  const legacyPorts = stringValue(udpHop.ports).trim();
  if (legacyPorts) {
    return legacyPorts;
  }
  const masks = Array.isArray(raw.udp) ? raw.udp : [];
  for (const mask of masks) {
    const candidate = asRecord(mask);
    if (String(candidate.type).toLowerCase() !== 'udphop') continue;
    const settings = asRecord(candidate.settings);
    const ports = stringValue(settings.remotePorts || settings.ports).trim();
    if (ports) return ports;
  }
  return '';
}

function normalizeHopRange(value: string): string {
  return value.trim().replace(/^(\d+)\s*:\s*(\d+)$/, '$1-$2');
}

function parseCoreInt32Range(value: string): [number, number] | null {
  const match = /^(-?\d+)(?:-(-?\d+))?$/.exec(value);
  if (!match) {
    return null;
  }
  const left = Number(match[1]);
  const right = Number(match[2] ?? match[1]);
  if (
    !Number.isSafeInteger(left) ||
    !Number.isSafeInteger(right) ||
    left < -2_147_483_648 ||
    left > 2_147_483_647 ||
    right < -2_147_483_648 ||
    right > 2_147_483_647
  ) {
    return null;
  }
  return left <= right ? [left, right] : [right, left];
}

function normalizeStringList(value: unknown): string[] {
  const values = Array.isArray(value)
    ? value
    : typeof value === 'string'
      ? value.split(/[\n,]/)
      : [];
  return [
    ...new Set(
      values
        .filter((item): item is string => typeof item === 'string')
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  ];
}

function nonNegativeInteger(value: unknown, fallback: number): number {
  const number = Number(value);
  return Number.isSafeInteger(number) && number >= 0 ? number : fallback;
}

function setOptionalString(target: Record<string, unknown>, key: string, value: string) {
  const trimmed = value.trim();
  if (trimmed) {
    target[key] = trimmed;
  } else {
    delete target[key];
  }
}

function setOptionalBoolean(target: Record<string, unknown>, key: string, value: boolean) {
  if (value) {
    target[key] = true;
  } else {
    delete target[key];
  }
}

function setOptionalNonNegativeInteger(
  target: Record<string, unknown>,
  key: string,
  value: number,
) {
  if (Number.isSafeInteger(value) && value > 0) {
    target[key] = value;
  } else {
    delete target[key];
  }
}

function hasControlCharacter(value: string): boolean {
  return Array.from(value).some((character) => {
    const codePoint = character.codePointAt(0) || 0;
    return codePoint <= 31 || codePoint === 127;
  });
}

function isValidIpPrefix(value: string): boolean {
  const separator = value.lastIndexOf('/');
  if (separator <= 0 || separator !== value.indexOf('/')) {
    return false;
  }
  const address = value.slice(0, separator);
  const prefixText = value.slice(separator + 1);
  if (!/^\d+$/.test(prefixText) || !isValidIpAddress(address)) {
    return false;
  }
  const prefix = Number(prefixText);
  return prefix >= 0 && prefix <= (address.includes(':') ? 128 : 32);
}

function isValidIpAddress(value: string): boolean {
  return isValidIpv4(value) || isValidIpv6(value);
}

function isValidIpv4(value: string): boolean {
  const parts = value.split('.');
  return (
    parts.length === 4 &&
    parts.every(
      (part) =>
        /^\d{1,3}$/.test(part) &&
        (part === '0' || !part.startsWith('0')) &&
        Number(part) >= 0 &&
        Number(part) <= 255,
    )
  );
}

function isValidIpv6(value: string): boolean {
  if (!value.includes(':') || value.includes('%') || (value.match(/::/g) || []).length > 1) {
    return false;
  }

  const compressed = value.includes('::');
  const halves = value.split('::');
  const groups = halves.flatMap((half) => (half ? half.split(':') : []));
  let groupCount = 0;
  for (let index = 0; index < groups.length; index += 1) {
    const group = groups[index];
    if (group.includes('.')) {
      if (index !== groups.length - 1 || !isValidIpv4(group)) {
        return false;
      }
      groupCount += 2;
      continue;
    }
    if (!/^[0-9a-fA-F]{1,4}$/.test(group)) {
      return false;
    }
    groupCount += 1;
  }
  return compressed ? groupCount < 8 : groupCount === 8;
}

function serializeShareableFinalMask(finalmask: unknown): string {
  if (!hasShareableFinalMaskValue(finalmask)) {
    return '';
  }
  try {
    return JSON.stringify(finalmask);
  } catch {
    return '';
  }
}

function hasShareableFinalMaskValue(value: unknown): boolean {
  if (value == null) {
    return false;
  }
  if (Array.isArray(value)) {
    return value.some(hasShareableFinalMaskValue);
  }
  if (typeof value === 'object') {
    return Object.values(value).some(hasShareableFinalMaskValue);
  }
  if (typeof value === 'string') {
    return value.trim().length > 0;
  }
  return true;
}

function hasUsableTlsCertificateEntry(value: unknown): boolean {
  const certificate = asRecord(value);
  const certificateFile = stringValue(certificate.certificateFile).trim();
  const keyFile = stringValue(certificate.keyFile).trim();
  if (certificateFile && keyFile) {
    return true;
  }

  const inlineCertificate = Array.isArray(certificate.certificate)
    ? certificate.certificate.some((line) => typeof line === 'string' && line.trim().length > 0)
    : stringValue(certificate.certificate).trim().length > 0;
  const inlineKey = Array.isArray(certificate.key)
    ? certificate.key.some((line) => typeof line === 'string' && line.trim().length > 0)
    : stringValue(certificate.key).trim().length > 0;
  return inlineCertificate && inlineKey;
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function arrayOrCsv(value: unknown): string {
  if (Array.isArray(value)) {
    return value
      .filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
      .join(',');
  }
  return stringValue(value);
}

function firstValue(value: unknown): string {
  if (Array.isArray(value)) {
    return (
      value.find((item): item is string => typeof item === 'string' && item.trim().length > 0) || ''
    );
  }
  return (
    stringValue(value)
      .split(',')
      .find((item) => item.trim().length > 0)
      ?.trim() || ''
  );
}

function normalizeAllowedIp(value: string): string {
  const trimmed = value.trim();
  if (!trimmed || trimmed.includes('/')) {
    return trimmed;
  }
  return trimmed.includes(':') ? `${trimmed}/128` : `${trimmed}/32`;
}

function randomLowerToken(length: number): string {
  const alphabet = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let token = '';
  for (let index = 0; index < length; index += 1) {
    token += alphabet[Math.floor(Math.random() * alphabet.length)];
  }
  return token;
}

function randomUuid(): string {
  if (crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (token) => {
    const value = Math.floor(Math.random() * 16);
    const digit = token === 'x' ? value : (value & 0x3) | 0x8;
    return digit.toString(16);
  });
}

function appendSubscriptionId(base: string, subId: string): string {
  const trimmed = base.trim();
  if (!trimmed) {
    return '';
  }
  return `${trimmed}${subId}`;
}

function hasSettingsText(
  source: Pick<Inbound, 'settings'> | InboundSettings,
): source is Pick<Inbound, 'settings'> {
  return typeof (source as Pick<Inbound, 'settings'>).settings === 'string';
}

function isInboundClient(value: unknown): value is InboundClient {
  return Boolean(value && typeof value === 'object' && 'email' in value);
}

function hasText(value: string): boolean {
  return value.trim().length > 0;
}

// Ported from the legacy UI so generated WireGuard keys stay legacy-compatible.
class WireguardKeyUtil {
  static gf(init?: number[]): Float64Array {
    const result = new Float64Array(16);
    if (init) {
      for (let index = 0; index < init.length; index += 1) {
        result[index] = init[index];
      }
    }
    return result;
  }

  static pack(output: Uint8Array, input: Float64Array) {
    let bit: number;
    const m = WireguardKeyUtil.gf();
    const t = WireguardKeyUtil.gf();
    for (let index = 0; index < 16; index += 1) {
      t[index] = input[index];
    }
    WireguardKeyUtil.carry(t);
    WireguardKeyUtil.carry(t);
    WireguardKeyUtil.carry(t);
    for (let round = 0; round < 2; round += 1) {
      m[0] = t[0] - 0xffed;
      for (let index = 1; index < 15; index += 1) {
        m[index] = t[index] - 0xffff - ((m[index - 1] >> 16) & 1);
        m[index - 1] &= 0xffff;
      }
      m[15] = t[15] - 0x7fff - ((m[14] >> 16) & 1);
      bit = (m[15] >> 16) & 1;
      m[14] &= 0xffff;
      WireguardKeyUtil.cswap(t, m, 1 - bit);
    }
    for (let index = 0; index < 16; index += 1) {
      output[2 * index] = t[index] & 0xff;
      output[2 * index + 1] = t[index] >> 8;
    }
  }

  static carry(output: Float64Array) {
    for (let index = 0; index < 16; index += 1) {
      output[(index + 1) % 16] += (index < 15 ? 1 : 38) * Math.floor(output[index] / 65536);
      output[index] &= 0xffff;
    }
  }

  static cswap(left: Float64Array, right: Float64Array, bit: number) {
    const c = ~(bit - 1);
    for (let index = 0; index < 16; index += 1) {
      const t = c & (left[index] ^ right[index]);
      left[index] ^= t;
      right[index] ^= t;
    }
  }

  static add(output: Float64Array, left: Float64Array, right: Float64Array) {
    for (let index = 0; index < 16; index += 1) {
      output[index] = (left[index] + right[index]) | 0;
    }
  }

  static subtract(output: Float64Array, left: Float64Array, right: Float64Array) {
    for (let index = 0; index < 16; index += 1) {
      output[index] = (left[index] - right[index]) | 0;
    }
  }

  static multmod(output: Float64Array, left: Float64Array, right: Float64Array) {
    const t = new Float64Array(31);
    for (let leftIndex = 0; leftIndex < 16; leftIndex += 1) {
      for (let rightIndex = 0; rightIndex < 16; rightIndex += 1) {
        t[leftIndex + rightIndex] += left[leftIndex] * right[rightIndex];
      }
    }
    for (let index = 0; index < 15; index += 1) {
      t[index] += 38 * t[index + 16];
    }
    for (let index = 0; index < 16; index += 1) {
      output[index] = t[index];
    }
    WireguardKeyUtil.carry(output);
    WireguardKeyUtil.carry(output);
  }

  static invert(output: Float64Array, input: Float64Array) {
    const c = WireguardKeyUtil.gf();
    for (let index = 0; index < 16; index += 1) {
      c[index] = input[index];
    }
    for (let index = 253; index >= 0; index -= 1) {
      WireguardKeyUtil.multmod(c, c, c);
      if (index !== 2 && index !== 4) {
        WireguardKeyUtil.multmod(c, c, input);
      }
    }
    for (let index = 0; index < 16; index += 1) {
      output[index] = c[index];
    }
  }

  static clamp(bytes: Uint8Array) {
    bytes[31] = (bytes[31] & 127) | 64;
    bytes[0] &= 248;
  }

  static generatePublicKey(privateKey: Uint8Array): Uint8Array {
    let bit: number;
    const z = new Uint8Array(32);
    const a = WireguardKeyUtil.gf([1]);
    const b = WireguardKeyUtil.gf([9]);
    const c = WireguardKeyUtil.gf();
    const d = WireguardKeyUtil.gf([1]);
    const e = WireguardKeyUtil.gf();
    const f = WireguardKeyUtil.gf();
    const constant = WireguardKeyUtil.gf([0xdb41, 1]);
    const nine = WireguardKeyUtil.gf([9]);
    for (let index = 0; index < 32; index += 1) {
      z[index] = privateKey[index];
    }
    WireguardKeyUtil.clamp(z);
    for (let index = 254; index >= 0; index -= 1) {
      bit = (z[index >>> 3] >>> (index & 7)) & 1;
      WireguardKeyUtil.cswap(a, b, bit);
      WireguardKeyUtil.cswap(c, d, bit);
      WireguardKeyUtil.add(e, a, c);
      WireguardKeyUtil.subtract(a, a, c);
      WireguardKeyUtil.add(c, b, d);
      WireguardKeyUtil.subtract(b, b, d);
      WireguardKeyUtil.multmod(d, e, e);
      WireguardKeyUtil.multmod(f, a, a);
      WireguardKeyUtil.multmod(a, c, a);
      WireguardKeyUtil.multmod(c, b, e);
      WireguardKeyUtil.add(e, a, c);
      WireguardKeyUtil.subtract(a, a, c);
      WireguardKeyUtil.multmod(b, a, a);
      WireguardKeyUtil.subtract(c, d, f);
      WireguardKeyUtil.multmod(a, c, constant);
      WireguardKeyUtil.add(a, a, d);
      WireguardKeyUtil.multmod(c, c, a);
      WireguardKeyUtil.multmod(a, d, f);
      WireguardKeyUtil.multmod(d, b, nine);
      WireguardKeyUtil.multmod(b, e, e);
      WireguardKeyUtil.cswap(a, b, bit);
      WireguardKeyUtil.cswap(c, d, bit);
    }
    WireguardKeyUtil.invert(c, c);
    WireguardKeyUtil.multmod(a, a, c);
    WireguardKeyUtil.pack(z, a);
    return z;
  }

  static generatePrivateKey(): Uint8Array {
    const privateKey = randomBytes(32);
    WireguardKeyUtil.clamp(privateKey);
    return privateKey;
  }

  static keyFromBase64(encoded: string): Uint8Array {
    const binary = atob(encoded);
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index);
    }
    return bytes;
  }

  static generateKeypair(secretKey = ''): { privateKey: string; publicKey: string } {
    const privateKey = secretKey
      ? WireguardKeyUtil.keyFromBase64(secretKey)
      : WireguardKeyUtil.generatePrivateKey();
    const publicKey = WireguardKeyUtil.generatePublicKey(privateKey);
    return {
      privateKey: secretKey || base64Bytes(privateKey),
      publicKey: base64Bytes(publicKey),
    };
  }
}
