import type { CoreType } from '@/types/core';
import type { XrayEditableInboundProtocol } from '@/types/inbound';

export type ProtocolCategory = 'proxy' | 'tunnel' | 'transparent' | 'peer';

export interface ProtocolCapabilities {
  clients: boolean;
  stream: boolean;
  tls: boolean;
  sniffing: boolean;
  shareLink: boolean;
}

export interface ProtocolRegistryEntry {
  protocol: XrayEditableInboundProtocol;
  label: string;
  category: ProtocolCategory;
  color: string;
  coreTypes: CoreType[];
  capabilities: ProtocolCapabilities;
}

export const xrayProtocolRegistry: Record<XrayEditableInboundProtocol, ProtocolRegistryEntry> = {
  vmess: {
    protocol: 'vmess',
    label: 'VMess',
    category: 'proxy',
    color: 'blue',
    coreTypes: ['xray'],
    capabilities: { clients: true, stream: true, tls: true, sniffing: true, shareLink: true },
  },
  vless: {
    protocol: 'vless',
    label: 'VLESS',
    category: 'proxy',
    color: 'green',
    coreTypes: ['xray'],
    capabilities: { clients: true, stream: true, tls: true, sniffing: true, shareLink: true },
  },
  tunnel: {
    protocol: 'tunnel',
    label: 'Tunnel',
    category: 'tunnel',
    color: 'lime',
    coreTypes: ['xray'],
    capabilities: { clients: false, stream: false, tls: false, sniffing: true, shareLink: false },
  },
  http: {
    protocol: 'http',
    label: 'HTTP',
    category: 'proxy',
    color: 'volcano',
    coreTypes: ['xray'],
    capabilities: { clients: false, stream: false, tls: false, sniffing: true, shareLink: false },
  },
  trojan: {
    protocol: 'trojan',
    label: 'Trojan',
    category: 'proxy',
    color: 'purple',
    coreTypes: ['xray'],
    capabilities: { clients: true, stream: true, tls: true, sniffing: true, shareLink: true },
  },
  shadowsocks: {
    protocol: 'shadowsocks',
    label: 'Shadowsocks',
    category: 'proxy',
    color: 'cyan',
    coreTypes: ['xray'],
    capabilities: { clients: true, stream: true, tls: false, sniffing: true, shareLink: true },
  },
  mixed: {
    protocol: 'mixed',
    label: 'Mixed',
    category: 'proxy',
    color: 'magenta',
    coreTypes: ['xray'],
    capabilities: { clients: false, stream: false, tls: false, sniffing: true, shareLink: false },
  },
  wireguard: {
    protocol: 'wireguard',
    label: 'WireGuard',
    category: 'peer',
    color: 'geekblue',
    coreTypes: ['xray'],
    capabilities: { clients: true, stream: false, tls: false, sniffing: false, shareLink: true },
  },
  tun: {
    protocol: 'tun',
    label: 'Tun',
    category: 'transparent',
    color: 'orange',
    coreTypes: ['xray'],
    capabilities: { clients: false, stream: false, tls: false, sniffing: false, shareLink: false },
  },
  hysteria: {
    protocol: 'hysteria',
    label: 'Hysteria2',
    category: 'proxy',
    color: 'gold',
    coreTypes: ['xray'],
    capabilities: { clients: true, stream: true, tls: true, sniffing: true, shareLink: true },
  },
  hysteria2: {
    protocol: 'hysteria2',
    label: 'Hysteria2 Legacy Alias',
    category: 'proxy',
    color: 'gold',
    coreTypes: ['xray'],
    capabilities: { clients: true, stream: true, tls: true, sniffing: true, shareLink: true },
  },
};

export const xrayEditableProtocols = Object.keys(
  xrayProtocolRegistry,
) as XrayEditableInboundProtocol[];

/** 判断协议是否已被新 UI 注册为可编辑协议。 */
export function isRegisteredEditableProtocol(
  protocol: string,
): protocol is XrayEditableInboundProtocol {
  return protocol in xrayProtocolRegistry;
}

/** 获取协议注册信息，未知协议返回空值。 */
export function getProtocolRegistryEntry(protocol: string): ProtocolRegistryEntry | undefined {
  return isRegisteredEditableProtocol(protocol) ? xrayProtocolRegistry[protocol] : undefined;
}

/** 判断协议是否支持客户端列表管理。 */
export function protocolSupportsClients(protocol: string): boolean {
  return Boolean(getProtocolRegistryEntry(protocol)?.capabilities.clients);
}

/** 判断协议是否支持 Stream Settings 表单。 */
export function protocolSupportsStream(protocol: string): boolean {
  return Boolean(getProtocolRegistryEntry(protocol)?.capabilities.stream);
}

/** 判断协议是否支持分享链接生成。 */
export function protocolSupportsShareLink(protocol: string): boolean {
  return Boolean(getProtocolRegistryEntry(protocol)?.capabilities.shareLink);
}
