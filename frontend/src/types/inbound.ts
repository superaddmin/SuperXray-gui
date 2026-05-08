export type InboundProtocol =
  | 'vmess'
  | 'vless'
  | 'tunnel'
  | 'http'
  | 'trojan'
  | 'shadowsocks'
  | 'mixed'
  | 'wireguard'
  | 'tun'
  | 'hysteria'
  | 'hysteria2'
  | string;

export interface ClientTraffic {
  id: number;
  inboundId: number;
  enable: boolean;
  email: string;
  uuid: string;
  subId: string;
  up: number;
  down: number;
  allTime: number;
  expiryTime: number;
  total: number;
  reset: number;
  lastOnline: number;
}

export interface Inbound {
  id: number;
  up: number;
  down: number;
  total: number;
  allTime: number;
  remark: string;
  enable: boolean;
  expiryTime: number;
  trafficReset: string;
  lastTrafficResetTime: number;
  clientStats: ClientTraffic[];
  listen: string;
  port: number;
  protocol: InboundProtocol;
  settings: string;
  streamSettings: string;
  tag: string;
  sniffing: string;
}

export type InboundForm = Omit<Inbound, 'id' | 'clientStats' | 'tag'> & {
  id?: number;
  clientStats?: ClientTraffic[];
  tag?: string;
};

export interface InboundClientForm {
  id: number;
  settings: string;
}

export type XrayEditableInboundProtocol =
  | 'vmess'
  | 'vless'
  | 'tunnel'
  | 'http'
  | 'trojan'
  | 'shadowsocks'
  | 'mixed'
  | 'wireguard'
  | 'tun'
  | 'hysteria'
  | 'hysteria2';

export interface InboundSettings {
  clients?: InboundClient[];
  peers?: InboundClient[];
  decryption?: string;
  encryption?: string;
  fallbacks?: unknown[];
  version?: number;
  mtu?: number | number[];
  secretKey?: string;
  pubKey?: string;
  noKernelTun?: boolean;
  method?: string;
  network?: string;
  password?: string;
  ivCheck?: boolean;
  selectedAuth?: string;
  testseed?: number[];
  [key: string]: unknown;
}

export interface InboundStreamSettings {
  network?: string;
  security?: string;
  externalProxy?: unknown[];
  tcpSettings?: Record<string, unknown>;
  tlsSettings?: Record<string, unknown>;
  realitySettings?: Record<string, unknown>;
  kcpSettings?: Record<string, unknown>;
  wsSettings?: Record<string, unknown>;
  grpcSettings?: Record<string, unknown>;
  httpupgradeSettings?: Record<string, unknown>;
  xhttpSettings?: Record<string, unknown>;
  hysteriaSettings?: Record<string, unknown>;
  sockopt?: Record<string, unknown>;
  finalmask?: Record<string, unknown>;
  [key: string]: unknown;
}

export interface InboundSniffingSettings {
  enabled?: boolean;
  destOverride?: string[];
  metadataOnly?: boolean;
  routeOnly?: boolean;
  [key: string]: unknown;
}

export interface InboundClient {
  id?: string;
  email: string;
  password?: string;
  method?: string;
  auth?: string;
  privateKey?: string;
  publicKey?: string;
  preSharedKey?: string;
  allowedIPs?: string[];
  keepAlive?: number;
  security?: string;
  flow?: string;
  limitIp?: number;
  totalGB?: number;
  expiryTime?: number;
  enable?: boolean;
  tgId?: number;
  subId?: string;
  comment?: string;
  reset?: number;
  created_at?: number;
  updated_at?: number;
  [key: string]: unknown;
}
