type XrayProcessState = 'running' | 'stop' | 'error' | string;

interface ResourceUsage {
  current: number;
  total: number;
}

export interface ServerStatus {
  cpu: number;
  cpuCores: number;
  logicalPro: number;
  cpuSpeedMhz: number;
  mem: ResourceUsage;
  swap: ResourceUsage;
  disk: ResourceUsage;
  xray: {
    state: XrayProcessState;
    errorMsg: string;
    version: string;
  };
  uptime: number;
  loads: number[];
  tcpCount: number;
  udpCount: number;
  netIO: {
    up: number;
    down: number;
  };
  netTraffic: {
    sent: number;
    recv: number;
  };
  publicIP: {
    ipv4: string;
    ipv6: string;
  };
  appStats: {
    threads: number;
    mem: number;
    uptime: number;
  };
}

export interface XrayAccessLogEntry {
  DateTime: string;
  FromAddress: string;
  ToAddress: string;
  Inbound: string;
  Outbound: string;
  Email: string;
  Event: number;
}

export interface PanelLogRequest {
  count: number;
  level: string;
  syslog: boolean;
}

export interface XrayLogRequest {
  count: number;
  filter: string;
  showDirect: boolean;
  showBlocked: boolean;
  showProxy: boolean;
}
