import { legacyEndpoints } from './endpoints';
import { getJson, postForm, type ApiRequestOptions } from './request';

import type { Inbound, InboundClientForm, InboundForm } from '@/types/inbound';

export function listInbounds(options?: ApiRequestOptions): Promise<Inbound[]> {
  return getJson<Inbound[]>(legacyEndpoints.inbounds.list, options);
}

export function addInbound(data: InboundForm, options?: ApiRequestOptions): Promise<Inbound> {
  return postForm<Inbound>(legacyEndpoints.inbounds.add, inboundToForm(data), options);
}

export function updateInbound(
  id: number,
  data: InboundForm,
  options?: ApiRequestOptions,
): Promise<Inbound> {
  return postForm<Inbound>(legacyEndpoints.inbounds.update(id), inboundToForm(data), options);
}

export function deleteInbound(id: number, options?: ApiRequestOptions): Promise<number> {
  return postForm<number>(legacyEndpoints.inbounds.delete(id), {}, options);
}

export function importInbound(data: string, options?: ApiRequestOptions): Promise<Inbound> {
  return postForm<Inbound>(legacyEndpoints.inbounds.import, { data }, options);
}

export function addInboundClient(
  data: InboundClientForm,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(
    legacyEndpoints.inbounds.addClient,
    {
      id: data.id,
      settings: data.settings,
    },
    options,
  );
}

export function updateInboundClient(
  clientId: string,
  data: InboundClientForm,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(
    legacyEndpoints.inbounds.updateClient(clientId),
    {
      id: data.id,
      settings: data.settings,
    },
    options,
  );
}

export function deleteInboundClient(
  id: number,
  clientId: string,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(legacyEndpoints.inbounds.deleteClient(id, clientId), {}, options);
}

export function resetInboundClientTraffic(
  id: number,
  email: string,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(legacyEndpoints.inbounds.resetClientTraffic(id, email), {}, options);
}

export function resetAllInboundTraffics(options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.inbounds.resetAllTraffics, {}, options);
}

export function resetAllInboundClientTraffics(
  id: number,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(legacyEndpoints.inbounds.resetAllClientTraffics(id), {}, options);
}

export function deleteDepletedInboundClients(
  id: number,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(legacyEndpoints.inbounds.deleteDepletedClients(id), {}, options);
}

export function getOnlineClients(options?: ApiRequestOptions): Promise<string[]> {
  return postForm<string[]>(legacyEndpoints.inbounds.onlines, {}, options);
}

export function getClientsLastOnline(options?: ApiRequestOptions): Promise<Record<string, number>> {
  return postForm<Record<string, number>>(legacyEndpoints.inbounds.lastOnline, {}, options);
}

export function getClientIps(
  email: string,
  options?: ApiRequestOptions,
): Promise<string | string[]> {
  return postForm<string | string[]>(legacyEndpoints.inbounds.clientIps(email), {}, options);
}

export function clearClientIps(email: string, options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.inbounds.clearClientIps(email), {}, options);
}

function inboundToForm(data: InboundForm) {
  return {
    id: data.id,
    up: data.up,
    down: data.down,
    total: data.total,
    allTime: data.allTime,
    remark: data.remark,
    enable: data.enable,
    expiryTime: data.expiryTime,
    trafficReset: data.trafficReset,
    lastTrafficResetTime: data.lastTrafficResetTime,
    listen: data.listen,
    port: data.port,
    protocol: data.protocol,
    settings: data.settings,
    streamSettings: data.streamSettings,
    tag: data.tag,
    sniffing: data.sniffing,
  };
}
