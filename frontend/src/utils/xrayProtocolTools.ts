import type { JsonObject, JsonValue } from '@/types/api';

export interface ProtocolToolPreset {
  value: string;
  label: string;
  runtime: 'xray' | 'sing-box';
  saveToXray: boolean;
}

export interface ProtocolToolInput {
  combo: string;
  mode?: 'quick' | 'fixed';
  originUrl?: string;
  tunnelName?: string;
  token?: string;
  server?: string;
  port?: number;
  uuid?: string;
  password?: string;
  sni?: string;
  publicKey?: string;
  shortId?: string;
  path?: string;
  host?: string;
  spiderX?: string;
  fingerprint?: string;
  modeName?: string;
  congestionControl?: string;
  tag?: string;
}

export interface ProtocolToolResult {
  combo: string;
  runtime: 'xray' | 'sing-box' | 'unknown';
  saveToXray: boolean;
  summary?: string;
  shareLink?: string;
  clientOutbound?: string;
  singBoxOutbound?: string;
  command?: string;
  systemd?: string;
  compose?: string;
  notice?: string;
}

export interface WarpDataPayload {
  private_key: string;
  client_id: string;
}

export interface WarpConfigPayload {
  interface: {
    addresses: {
      v4?: string;
      v6?: string;
    };
  };
  peers: Array<{
    public_key: string;
    endpoint: {
      host: string;
    };
  }>;
}

export interface WarpBaseSettings extends JsonObject {
  mtu: number;
  secretKey: string;
  address: string[];
  reserved: number[];
  peers: Array<{
    publicKey: string;
    endpoint: string;
  }>;
  noKernelTun: boolean;
}

export interface WarpMatrixOption {
  tag: string;
  label: string;
  domainStrategy: string;
  rule?: JsonObject;
}

export const PROTOCOL_TOOL_PRESETS: ProtocolToolPreset[] = [
  { value: 'vless-reality-vision', label: 'VLESS Reality Vision', runtime: 'xray', saveToXray: true },
  { value: 'vless-xhttp-reality', label: 'VLESS XHTTP Reality Vision', runtime: 'xray', saveToXray: true },
  { value: 'vless-ws-tls', label: 'VLESS WS TLS', runtime: 'xray', saveToXray: true },
  { value: 'trojan-tcp-tls', label: 'Trojan TCP TLS', runtime: 'xray', saveToXray: true },
  { value: 'shadowsocks-2022', label: 'Shadowsocks 2022', runtime: 'xray', saveToXray: true },
  { value: 'hysteria2-tls', label: 'Hysteria2 TLS', runtime: 'xray', saveToXray: true },
  { value: 'tuic-singbox', label: 'TUIC sing-box', runtime: 'sing-box', saveToXray: false },
  { value: 'anytls-singbox', label: 'AnyTLS sing-box', runtime: 'sing-box', saveToXray: false },
];

export const WARP_MATRIX_OPTIONS: WarpMatrixOption[] = [
  { tag: 'warp', label: 'WARP default', domainStrategy: 'ForceIP' },
  { tag: 'warp-ipv4', label: 'WARP IPv4', domainStrategy: 'ForceIPv4' },
  { tag: 'warp-ipv6', label: 'WARP IPv6', domainStrategy: 'ForceIPv6' },
  {
    tag: 'warp-openai',
    label: 'WARP OpenAI',
    domainStrategy: 'ForceIP',
    rule: { type: 'field', outboundTag: 'warp-openai', domain: ['geosite:openai'] },
  },
];

export function generateProtocolToolArgo(
  input: Pick<ProtocolToolInput, 'mode' | 'originUrl' | 'tunnelName' | 'token'>,
): ProtocolToolResult {
  const mode = cleanString(input.mode, 'quick');
  const originUrl = cleanString(input.originUrl, 'http://localhost:2053');
  if (mode === 'fixed') {
    const token = cleanString(input.token, '<CLOUDFLARE_TUNNEL_TOKEN>');
    const tunnelName = cleanString(input.tunnelName, 'superxray');
    return {
      combo: 'argo-fixed',
      runtime: 'unknown',
      saveToXray: false,
      command: `cloudflared tunnel run --token ${token}`,
      systemd: [
        '[Unit]',
        `Description=Cloudflare Tunnel for ${tunnelName}`,
        'After=network-online.target',
        'Wants=network-online.target',
        '',
        '[Service]',
        'TimeoutStartSec=0',
        'Type=simple',
        `ExecStart=/usr/local/bin/cloudflared tunnel run --token ${token}`,
        'Restart=on-failure',
        'RestartSec=5s',
        '',
        '[Install]',
        'WantedBy=multi-user.target',
      ].join('\n'),
      compose: [
        'services:',
        '  cloudflared:',
        '    image: cloudflare/cloudflared:latest',
        '    restart: unless-stopped',
        `    command: tunnel run --token ${token}`,
      ].join('\n'),
      notice: 'Token is generated into this output only and is not submitted to the backend by Protocol Tools.',
    };
  }

  return {
    combo: 'argo-quick',
    runtime: 'unknown',
    saveToXray: false,
    command: `cloudflared tunnel --url ${originUrl}`,
    notice: 'Quick Tunnels are for testing and development. Use a fixed tunnel for production.',
  };
}

export function generateProtocolToolCombo(input: ProtocolToolInput): ProtocolToolResult {
  const combo = cleanString(input.combo, 'vless-reality-vision');
  switch (combo) {
    case 'vless-reality-vision': {
      const outbound = buildVlessOutbound(input, 'tcp', 'reality');
      return {
        combo,
        runtime: 'xray',
        saveToXray: true,
        summary: 'VLESS over TCP + Reality + Vision',
        clientOutbound: jsonText(outbound),
        shareLink: buildVlessShareLink(input, 'tcp', 'reality'),
      };
    }
    case 'vless-xhttp-reality': {
      const outbound = buildVlessOutbound(input, 'xhttp', 'reality');
      return {
        combo,
        runtime: 'xray',
        saveToXray: true,
        summary: 'VLESS over XHTTP + Reality + Vision',
        clientOutbound: jsonText(outbound),
        shareLink: buildVlessShareLink(input, 'xhttp', 'reality'),
      };
    }
    case 'vless-ws-tls': {
      const outbound = buildVlessOutbound(input, 'ws', 'tls');
      (outbound.settings as JsonObject).vnext = [
        {
          address: cleanString(input.server, 'example.com'),
          port: asInt(input.port, 443),
          users: [{ id: cleanString(input.uuid, defaultUuid()), encryption: 'none', flow: '' }],
        },
      ];
      return {
        combo,
        runtime: 'xray',
        saveToXray: true,
        summary: 'VLESS over WebSocket + TLS',
        clientOutbound: jsonText(outbound),
        shareLink: buildVlessShareLink(input, 'ws', 'tls'),
      };
    }
    case 'trojan-tcp-tls': {
      const outbound = buildTrojanOutbound(input);
      const server = cleanString(input.server, 'example.com');
      const port = asInt(input.port, 443);
      const password = encodeURIComponent(cleanString(input.password, 'change-me'));
      const sni = cleanString(input.sni, server);
      return {
        combo,
        runtime: 'xray',
        saveToXray: true,
        summary: 'Trojan over TCP + TLS',
        clientOutbound: jsonText(outbound),
        shareLink: `trojan://${password}@${server}:${port}?security=tls&type=tcp&sni=${encodeURIComponent(sni)}#trojan`,
      };
    }
    case 'shadowsocks-2022': {
      const outbound = buildShadowsocks2022Outbound(input);
      return {
        combo,
        runtime: 'xray',
        saveToXray: true,
        summary: 'Shadowsocks 2022 outbound template',
        clientOutbound: jsonText(outbound),
        shareLink: 'ss://<base64(method:server-key:client-key)>@<server>:<port>#shadowsocks-2022',
      };
    }
    case 'hysteria2-tls': {
      const outbound = buildHysteria2Outbound(input);
      const server = cleanString(input.server, 'example.com');
      const port = asInt(input.port, 443);
      const password = encodeURIComponent(cleanString(input.password, 'change-me'));
      const sni = cleanString(input.sni, server);
      return {
        combo,
        runtime: 'xray',
        saveToXray: true,
        summary: 'Hysteria2 over TLS',
        clientOutbound: jsonText(outbound),
        shareLink: `hysteria2://${password}@${server}:${port}?security=tls&sni=${encodeURIComponent(sni)}&alpn=h3#hysteria2`,
      };
    }
    case 'tuic-singbox':
      return {
        combo,
        runtime: 'sing-box',
        saveToXray: false,
        summary: 'TUIC external sing-box outbound',
        singBoxOutbound: jsonText({
          type: 'tuic',
          tag: cleanString(input.tag, 'tuic-out'),
          server: cleanString(input.server, 'example.com'),
          server_port: asInt(input.port, 443),
          uuid: cleanString(input.uuid, defaultUuid()),
          password: cleanString(input.password, 'change-me'),
          congestion_control: cleanString(input.congestionControl, 'cubic'),
        }),
        notice: 'TUIC is not supported as an Xray inbound in this panel build; use this as an external sing-box config snippet.',
      };
    case 'anytls-singbox':
      return {
        combo,
        runtime: 'sing-box',
        saveToXray: false,
        summary: 'AnyTLS external sing-box outbound',
        singBoxOutbound: jsonText({
          type: 'anytls',
          tag: cleanString(input.tag, 'anytls-out'),
          server: cleanString(input.server, 'example.com'),
          server_port: asInt(input.port, 443),
          password: cleanString(input.password, 'change-me'),
        }),
        notice: 'AnyTLS is not supported as an Xray inbound in this panel build; use this as an external sing-box config snippet.',
      };
    default:
      return {
        combo,
        runtime: 'unknown',
        saveToXray: false,
        notice: `Unknown combo: ${combo}`,
      };
  }
}

export function buildWarpMatrixBaseSettings(
  warpData: WarpDataPayload,
  warpConfig: WarpConfigPayload,
): WarpBaseSettings {
  const peer = warpConfig.peers[0];
  const address: string[] = [];
  if (warpConfig.interface.addresses.v4) {
    address.push(`${warpConfig.interface.addresses.v4}/32`);
  }
  if (warpConfig.interface.addresses.v6) {
    address.push(`${warpConfig.interface.addresses.v6}/128`);
  }

  return {
    mtu: 1420,
    secretKey: warpData.private_key,
    address,
    reserved: decodeWarpClientId(warpData.client_id),
    peers: [
      {
        publicKey: peer.public_key,
        endpoint: peer.endpoint.host,
      },
    ],
    noKernelTun: false,
  };
}

export function applyWarpMatrixToTemplate(
  templateSettings: JsonValue | null | undefined,
  baseSettings: WarpBaseSettings,
  selectedTags: string[],
): JsonObject {
  const template = cloneObject(templateSettings);
  const currentOutbounds = asObjectArray(template.outbounds);
  const outbounds = currentOutbounds.filter((outbound) => !isWarpTag(stringValue(outbound.tag)));
  outbounds.push(
    ...selectedTags
      .map((tag) => WARP_MATRIX_OPTIONS.find((option) => option.tag === tag))
      .filter(Boolean)
      .map((option) => ({
        tag: option!.tag,
        protocol: 'wireguard',
        settings: {
          ...cloneObject(baseSettings),
          domainStrategy: option!.domainStrategy,
        },
      })),
  );

  const routing = asObject(template.routing);
  const currentRules = asObjectArray(routing.rules).filter(
    (rule) => !isWarpTag(stringValue(rule.outboundTag)),
  );
  const nextRules = selectedTags
    .map((tag) => WARP_MATRIX_OPTIONS.find((option) => option.tag === tag)?.rule)
    .filter(Boolean)
    .map((rule) => cloneObject(rule));

  routing.rules = [...currentRules, ...nextRules];
  template.outbounds = outbounds;
  template.routing = routing;
  return template;
}

function buildVlessOutbound(input: ProtocolToolInput, network: string, security: string): JsonObject {
  const server = cleanString(input.server, 'example.com');
  const port = asInt(input.port, 443);
  const uuid = cleanString(input.uuid, defaultUuid());
  const sni = cleanString(input.sni, server);
  const streamSettings: JsonObject = {
    network,
    security,
  };

  if (security === 'reality') {
    streamSettings.realitySettings = {
      serverName: sni,
      fingerprint: cleanString(input.fingerprint, 'chrome'),
      publicKey: cleanString(input.publicKey, 'PUBLIC_KEY'),
      shortId: cleanString(input.shortId, ''),
      spiderX: cleanString(input.spiderX, '/'),
    };
  } else if (security === 'tls') {
    streamSettings.tlsSettings = {
      serverName: sni,
      alpn: ['h2', 'http/1.1'],
    };
  }

  if (network === 'xhttp') {
    streamSettings.xhttpSettings = {
      path: cleanString(input.path, '/xhttp'),
      host: cleanString(input.host, sni),
      mode: cleanString(input.modeName, 'auto'),
    };
  } else if (network === 'ws') {
    streamSettings.wsSettings = {
      path: cleanString(input.path, '/'),
      host: cleanString(input.host, sni),
    };
  }

  return {
    protocol: 'vless',
    tag: cleanString(input.tag, 'proxy'),
    settings: {
      vnext: [
        {
          address: server,
          port,
          users: [
            {
              id: uuid,
              encryption: 'none',
              flow: 'xtls-rprx-vision',
            },
          ],
        },
      ],
    },
    streamSettings,
  };
}

function buildVlessShareLink(input: ProtocolToolInput, network: string, security: string): string {
  const server = cleanString(input.server, 'example.com');
  const port = asInt(input.port, 443);
  const uuid = cleanString(input.uuid, defaultUuid());
  const sni = cleanString(input.sni, server);
  const query = new URLSearchParams({
    type: network,
    security,
    encryption: 'none',
    sni,
  });
  if (security === 'reality') {
    query.set('flow', 'xtls-rprx-vision');
    query.set('fp', cleanString(input.fingerprint, 'chrome'));
    query.set('pbk', cleanString(input.publicKey, 'PUBLIC_KEY'));
    query.set('sid', cleanString(input.shortId, ''));
    query.set('spx', cleanString(input.spiderX, '/'));
  }
  if (network === 'xhttp') {
    query.set('path', cleanString(input.path, '/xhttp'));
    query.set('mode', cleanString(input.modeName, 'auto'));
  }
  if (network === 'ws') {
    query.set('path', cleanString(input.path, '/'));
    query.set('host', cleanString(input.host, sni));
  }
  return `vless://${uuid}@${server}:${port}?${query.toString()}#${encodeURIComponent(cleanString(input.tag, 'vless'))}`;
}

function buildTrojanOutbound(input: ProtocolToolInput): JsonObject {
  const server = cleanString(input.server, 'example.com');
  const sni = cleanString(input.sni, server);
  return {
    protocol: 'trojan',
    tag: cleanString(input.tag, 'proxy'),
    settings: {
      servers: [
        {
          address: server,
          port: asInt(input.port, 443),
          password: cleanString(input.password, 'change-me'),
        },
      ],
    },
    streamSettings: {
      network: 'tcp',
      security: 'tls',
      tlsSettings: {
        serverName: sni,
        alpn: ['h2', 'http/1.1'],
      },
    },
  };
}

function buildShadowsocks2022Outbound(input: ProtocolToolInput): JsonObject {
  return {
    protocol: 'shadowsocks',
    tag: cleanString(input.tag, 'proxy'),
    settings: {
      servers: [
        {
          address: cleanString(input.server, 'example.com'),
          port: asInt(input.port, 443),
          method: cleanString(input.modeName, '2022-blake3-aes-256-gcm'),
          password: cleanString(input.password, 'SERVER_KEY:CLIENT_KEY'),
        },
      ],
    },
  };
}

function buildHysteria2Outbound(input: ProtocolToolInput): JsonObject {
  const server = cleanString(input.server, 'example.com');
  const sni = cleanString(input.sni, server);
  return {
    protocol: 'hysteria',
    tag: cleanString(input.tag, 'proxy'),
    settings: {
      version: 2,
      address: server,
      port: asInt(input.port, 443),
    },
    streamSettings: {
      network: 'hysteria',
      security: 'tls',
      tlsSettings: {
        serverName: sni,
        alpn: ['h3'],
      },
      hysteriaSettings: {
        version: 2,
        auth: cleanString(input.password, 'change-me'),
      },
    },
  };
}

function decodeWarpClientId(clientId: string): number[] {
  const decoded = atob(clientId);
  return Array.from(decoded).map((char) => char.charCodeAt(0));
}

function isWarpTag(tag: string): boolean {
  return tag === 'warp' || tag.startsWith('warp-');
}

function cleanString(value: unknown, fallback = ''): string {
  if (value === null || value === undefined) {
    return fallback;
  }
  const text = String(value).trim();
  return text.length > 0 ? text : fallback;
}

function asInt(value: unknown, fallback: number): number {
  const parsed = Number.parseInt(String(value ?? ''), 10);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function jsonText(value: unknown): string {
  return JSON.stringify(value, null, 2);
}

function defaultUuid(): string {
  return '11111111-1111-4111-8111-111111111111';
}

function cloneObject(value: unknown): JsonObject {
  return JSON.parse(JSON.stringify(asObject(value))) as JsonObject;
}

function asObject(value: unknown): JsonObject {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as JsonObject)
    : {};
}

function asObjectArray(value: JsonValue | undefined): JsonObject[] {
  return Array.isArray(value) ? value.map((item) => asObject(item)) : [];
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value : '';
}
