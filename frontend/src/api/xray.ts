import { legacyEndpoints } from './endpoints';
import { ApiError, getJson, postForm, type ApiRequestOptions } from './request';

import type { FormRecord } from '@/types/api';
import type {
  OutboundTestResult,
  OutboundTraffic,
  XrayCommandResult,
  XraySettingPayload,
  XraySettingUpdateForm,
} from '@/types/xray';

export async function getXraySetting(options?: ApiRequestOptions): Promise<XraySettingPayload> {
  const raw = await postForm<string>(legacyEndpoints.xray.setting, {}, options);
  return parseXraySettingPayload(raw);
}

export function updateXraySetting(
  data: XraySettingUpdateForm,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(
    legacyEndpoints.xray.update,
    {
      xraySetting: data.xraySetting,
      outboundTestUrl: data.outboundTestUrl,
    },
    options,
  );
}

export function getXrayResult(options?: ApiRequestOptions): Promise<XrayCommandResult> {
  return getJson<string>(legacyEndpoints.xray.result, options);
}

export function getOutboundsTraffic(options?: ApiRequestOptions): Promise<OutboundTraffic[]> {
  return getJson<OutboundTraffic[]>(legacyEndpoints.xray.outboundsTraffic, options);
}

export function resetOutboundsTraffic(tag: string, options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.xray.resetOutboundsTraffic, { tag }, options);
}

export function testOutbound(
  outbound: string,
  allOutbounds: string,
  options?: ApiRequestOptions,
): Promise<OutboundTestResult> {
  return postForm<OutboundTestResult>(
    legacyEndpoints.xray.testOutbound,
    { allOutbounds, outbound },
    options,
  );
}

export function runWarpAction(
  action: string,
  data: FormRecord = {},
  options?: ApiRequestOptions,
): Promise<string> {
  return postForm<string>(legacyEndpoints.xray.warp(action), data, options);
}

export function runNordAction(
  action: string,
  data: FormRecord = {},
  options?: ApiRequestOptions,
): Promise<string> {
  return postForm<string>(legacyEndpoints.xray.nord(action), data, options);
}

function parseXraySettingPayload(raw: string): XraySettingPayload {
  try {
    return JSON.parse(raw) as XraySettingPayload;
  } catch (error) {
    throw new ApiError('Failed to parse Xray settings response', {
      response: error,
    });
  }
}
