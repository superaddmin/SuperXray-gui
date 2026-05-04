import type { JsonValue } from './api';

export interface XraySettingPayload {
  xraySetting: JsonValue;
  inboundTags: JsonValue;
  outboundTestUrl: string;
}

export interface XraySettingUpdateForm {
  xraySetting: string;
  outboundTestUrl: string;
}

export type XrayCommandResult = string;

export interface OutboundTraffic {
  down: number;
  id: number;
  tag: string;
  total: number;
  up: number;
}

export interface OutboundTestResult {
  delay: number;
  error?: string;
  statusCode?: number;
  success: boolean;
}
