import { legacyEndpoints } from './endpoints';
import { downloadFile, getJson, postForm, type ApiRequestOptions, uploadForm } from './request';

import type {
  PanelLogRequest,
  ServerStatus,
  XrayAccessLogEntry,
  XrayLogRequest,
} from '@/types/server';

export function getServerStatus(options?: ApiRequestOptions): Promise<ServerStatus | null> {
  return getJson<ServerStatus | null>(legacyEndpoints.server.status, options);
}

export function getXrayVersions(options?: ApiRequestOptions): Promise<string[]> {
  return getJson<string[]>(legacyEndpoints.server.xrayVersions, options);
}

function restartXrayService(options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.server.restartXray, {}, options);
}

export function startXrayService(options?: ApiRequestOptions): Promise<void> {
  return restartXrayService(options);
}

export function stopXrayService(options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.server.stopXray, {}, options);
}

export function installXrayVersion(version: string, options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.server.installXray(version), {}, options);
}

export function updateGeofile(fileName?: string, options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.server.updateGeofile(fileName), {}, options);
}

export async function getPanelLogs(
  request: PanelLogRequest,
  options?: ApiRequestOptions,
): Promise<string[]> {
  const logs = await postForm<string[] | null>(
    legacyEndpoints.server.logs(request.count),
    {
      level: request.level,
      syslog: request.syslog,
    },
    options,
  );
  return logs ?? [];
}

export async function getXrayLogs(
  request: XrayLogRequest,
  options?: ApiRequestOptions,
): Promise<XrayAccessLogEntry[]> {
  const logs = await postForm<XrayAccessLogEntry[] | null>(
    legacyEndpoints.server.xrayLogs(request.count),
    {
      filter: request.filter,
      showDirect: request.showDirect,
      showBlocked: request.showBlocked,
      showProxy: request.showProxy,
    },
    options,
  );
  return logs ?? [];
}

export function downloadDatabase(options?: ApiRequestOptions): Promise<Blob> {
  return downloadFile(legacyEndpoints.server.database, options);
}

export function importDatabase(file: File, options?: ApiRequestOptions): Promise<string> {
  const body = new FormData();
  body.append('db', file);
  return uploadForm<string>(legacyEndpoints.server.importDatabase, body, options);
}
