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

  return {
    clients: [],
  };
}

export function defaultStreamSettings(
  protocol?: XrayEditableInboundProtocol,
): InboundStreamSettings {
  if (protocol === 'wireguard') {
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
          fingerprint: 'chrome',
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

export function buildClientShareLink(inbound: Inbound, client: InboundClient): string {
  if (!getClientPrimaryId(inbound.protocol, client)) {
    return '';
  }

  if (inbound.protocol === 'vmess') {
    return buildVmessShareLink(inbound, client);
  }
  if (inbound.protocol === 'vless') {
    return buildVlessShareLink(inbound, client);
  }
  if (inbound.protocol === 'trojan') {
    return buildTrojanShareLink(inbound, client);
  }
  if (inbound.protocol === 'shadowsocks') {
    return buildShadowsocksShareLink(inbound, client);
  }
  if (inbound.protocol === 'hysteria' || inbound.protocol === 'hysteria2') {
    return buildHysteriaShareLink(inbound, client);
  }
  if (inbound.protocol === 'wireguard') {
    return buildWireguardShareLink(inbound, client);
  }
  return '';
}

function buildVmessShareLink(inbound: Inbound, client: InboundClient): string {
  const stream = parseInboundStreamSettings(inbound);
  const payload = {
    v: '2',
    ps: client.email || inbound.remark || inbound.tag,
    add: resolveInboundHost(inbound),
    port: String(inbound.port),
    id: client.id,
    aid: '0',
    scy: client.security || 'auto',
    net: stream.network || 'tcp',
    type: 'none',
    host: '',
    path: '',
    tls: stream.security === 'tls' ? 'tls' : '',
    sni: '',
  };

  return `vmess://${base64Utf8(JSON.stringify(payload))}`;
}

function buildVlessShareLink(inbound: Inbound, client: InboundClient): string {
  const stream = parseInboundStreamSettings(inbound);
  const params = new URLSearchParams();
  params.set('encryption', 'none');
  params.set('security', stream.security || 'none');
  params.set('type', stream.network || 'tcp');
  if (client.flow) {
    params.set('flow', client.flow);
  }

  const label = encodeURIComponent(client.email || inbound.remark || inbound.tag);
  return `vless://${client.id}@${resolveInboundHost(inbound)}:${inbound.port}?${params.toString()}#${label}`;
}

function buildTrojanShareLink(inbound: Inbound, client: InboundClient): string {
  if (!client.password) {
    return '';
  }

  const stream = parseInboundStreamSettings(inbound);
  const params = new URLSearchParams();
  params.set('security', stream.security || 'none');
  params.set('type', stream.network || 'tcp');

  const label = encodeURIComponent(client.email || inbound.remark || inbound.tag);
  return `trojan://${encodeURIComponent(client.password)}@${resolveInboundHost(inbound)}:${inbound.port}?${params.toString()}#${label}`;
}

function buildShadowsocksShareLink(inbound: Inbound, client: InboundClient): string {
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

  const label = encodeURIComponent(client.email || inbound.remark || inbound.tag);
  const userInfo = base64UrlUtf8(passwordParts.join(':'));
  return `ss://${userInfo}@${resolveInboundHost(inbound)}:${inbound.port}#${label}`;
}

function buildHysteriaShareLink(inbound: Inbound, client: InboundClient): string {
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

  const label = encodeURIComponent(client.email || inbound.remark || inbound.tag);
  return `hysteria2://${encodeURIComponent(client.auth)}@${resolveInboundHost(inbound)}:${inbound.port}?${params.toString()}#${label}`;
}

function buildWireguardShareLink(inbound: Inbound, client: InboundClient): string {
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

  const url = new URL(`wireguard://${resolveInboundHost(inbound)}:${inbound.port}`);
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
  url.hash = encodeURIComponent(client.email || inbound.remark || inbound.tag);
  return url.toString();
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

function hasSettingsText(
  source: Pick<Inbound, 'settings'> | InboundSettings,
): source is Pick<Inbound, 'settings'> {
  return typeof (source as Pick<Inbound, 'settings'>).settings === 'string';
}

function isInboundClient(value: unknown): value is InboundClient {
  return Boolean(value && typeof value === 'object' && 'email' in value);
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
