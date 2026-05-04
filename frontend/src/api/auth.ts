import { legacyEndpoints } from './endpoints';
import { postForm, type ApiRequestOptions } from './request';
import type { FormRecord } from '@/types/api';

interface LoginPayload {
  password: string;
  twoFactorCode?: string;
  username: string;
}

export function login(payload: LoginPayload, options?: ApiRequestOptions) {
  const form: FormRecord = {
    password: payload.password,
    twoFactorCode: payload.twoFactorCode,
    username: payload.username,
  };
  return postForm<void>(legacyEndpoints.auth.login, form, options);
}

export function getTwoFactorEnabled(options?: ApiRequestOptions) {
  return postForm<boolean>(legacyEndpoints.auth.twoFactorEnabled, {}, options);
}
