export type CoreType = 'xray' | 'sing-box' | string;

export type CoreState =
  | 'unknown'
  | 'running'
  | 'stopped'
  | 'error'
  | 'not-installed'
  | 'not-configured'
  | string;

export interface CoreCapabilities {
  read: boolean;
  write: boolean;
  validate: boolean;
  start: boolean;
  stop: boolean;
  restart: boolean;
  lifecycleViaCoreManager: boolean;
}

export interface CoreStatus {
  state: CoreState;
  version: string;
  errorMsg?: string;
  pid?: number;
  binary?: string;
  config?: string;
  updatedAt?: string;
}

export interface CoreInstance {
  id: string;
  name: string;
  displayName: string;
  coreType: CoreType;
  mode: string;
  source: string;
  lifecycleOwner: string;
  status: CoreStatus;
  capabilities: CoreCapabilities;
  writeSupported: boolean;
  managerAttached: boolean;
  experimentalOnly: boolean;
}

export interface CoreLifecycleResult {
  state: CoreState;
  msg?: string;
  errorMsg?: string;
  pid?: number;
}
