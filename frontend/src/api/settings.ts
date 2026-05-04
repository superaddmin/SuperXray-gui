import { legacyEndpoints } from './endpoints';
import { postForm, type ApiRequestOptions } from './request';

import type { FormRecord } from '@/types/api';
import type {
  PanelSettings,
  PanelSettingsUpdateForm,
  UserCredentialsUpdateForm,
} from '@/types/settings';

export function getAllSettings(options?: ApiRequestOptions): Promise<PanelSettings> {
  return postForm<PanelSettings>(legacyEndpoints.settings.all, {}, options);
}

export function getDefaultSettings(options?: ApiRequestOptions): Promise<PanelSettings> {
  return postForm<PanelSettings>(legacyEndpoints.settings.defaultSettings, {}, options);
}

export function updateSettings(
  data: PanelSettingsUpdateForm,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(legacyEndpoints.settings.update, settingsToForm(data), options);
}

export function updateUserCredentials(
  data: UserCredentialsUpdateForm,
  options?: ApiRequestOptions,
): Promise<void> {
  return postForm<void>(legacyEndpoints.settings.updateUser, { ...data }, options);
}

export function restartPanel(options?: ApiRequestOptions): Promise<void> {
  return postForm<void>(legacyEndpoints.settings.restartPanel, {}, options);
}

function settingsToForm(data: PanelSettingsUpdateForm): FormRecord {
  return { ...data };
}
