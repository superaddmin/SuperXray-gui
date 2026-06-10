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

export function parseInboundStreamSettings(
  inbound: Pick<Inbound, 'streamSettings'>,
): InboundStreamSettings {
  return parseJsonObject<InboundStreamSettings>(inbound.streamSettings, defaultStreamSettings());
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
    return Array.isArray(settings.peers) ? settings.peers.filter(isWireguardPeer) : [];
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
      mtu: [1500, 1280],
      gateway: [],
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
  const ports = normalizeHopRange(input.ports);
  const interval = normalizeHopRange(input.interval);

  if (!input.quicParamsEnabled) {
    delete finalmask.quicParams;
    if (Object.keys(finalmask).length > 0) {
      next.finalmask = finalmask;
    } else {
      delete next.finalmask;
    }
    return next;
  }

  if (input.quicParamsEnabled && input.udpHopEnabled && ports) {
    quicParams.udpHop = interval ? { ports, interval } : { ports };
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

function defaultProxyAccount(): { user: string; pass: string } {
  return {
    user: randomLowerToken(10),
    pass: randomLowerToken(10),
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

function appendHysteriaFinalMaskParams(params: URLSearchParams, finalmask: unknown) {
  const serialized = serializeShareableFinalMask(finalmask);
  if (serialized) {
    params.set('fm', serialized);
  }

  const obfsPassword = hysteriaSalamanderPassword(finalmask);
  if (obfsPassword) {
    params.set('obfs', 'salamander');
    params.set('obfs-password', obfsPassword);
  }

  const hopPorts = hysteriaHopPorts(finalmask);
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
  const quicParams = asRecord(asRecord(finalmask).quicParams);
  const udpHop = asRecord(quicParams.udpHop);
  return stringValue(udpHop.ports).trim();
}

function normalizeHopRange(value: string): string {
  return value.trim().replace(/^(\d+)\s*:\s*(\d+)$/, '$1-$2');
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
