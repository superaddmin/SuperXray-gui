import { legacyEndpoints } from './endpoints';
import { getJson, postForm, type ApiRequestOptions } from './request';

import type { CustomGeoForm, CustomGeoResource } from '@/types/customGeo';
import type { FormRecord } from '@/types/api';

export function listCustomGeo(options?: ApiRequestOptions): Promise<CustomGeoResource[]> {
  return getJson<CustomGeoResource[]>(legacyEndpoints.customGeo.list, options);
}

export function addCustomGeo(data: CustomGeoForm, options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.customGeo.add, customGeoToForm(data), options);
}

export function updateCustomGeo(
  id: number,
  data: CustomGeoForm,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(legacyEndpoints.customGeo.update(id), customGeoToForm(data), options);
}

export function deleteCustomGeo(id: number, options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.customGeo.delete(id), {}, options);
}

export function downloadCustomGeo(id: number, options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.customGeo.download(id), {}, options);
}

export function updateAllCustomGeo(options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.customGeo.updateAll, {}, options);
}

function customGeoToForm(data: CustomGeoForm): FormRecord {
  return {
    alias: data.alias,
    type: data.type,
    url: data.url,
  };
}
