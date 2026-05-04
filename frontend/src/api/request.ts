import { message as antMessage } from 'ant-design-vue';
import axios, { type AxiosRequestConfig } from 'axios';

import type { ApiErrorDetails, FormRecord, FormValue, LegacyResponse } from '@/types/api';
import { getRuntimeConfig } from '@/types/runtime';

export interface ApiRequestOptions {
  notifyOnError?: boolean;
  redirectOnUnauthorized?: boolean;
}

export class ApiError<T = unknown> extends Error {
  readonly status?: number;
  readonly endpoint?: string;
  readonly response?: LegacyResponse<T> | T;
  readonly sessionExpired: boolean;

  constructor(message: string, details: ApiErrorDetails<T> = {}) {
    super(message);
    this.name = 'ApiError';
    this.status = details.status;
    this.endpoint = details.endpoint;
    this.response = details.response;
    this.sessionExpired = details.sessionExpired ?? false;
  }
}

const runtimeConfig = getRuntimeConfig();

const defaultOptions: Required<ApiRequestOptions> = {
  notifyOnError: true,
  redirectOnUnauthorized: true,
};

const apiClient = axios.create({
  baseURL: runtimeConfig.apiBasePath,
  timeout: 20_000,
  withCredentials: true,
  headers: {
    'X-Requested-With': 'XMLHttpRequest',
  },
});

apiClient.interceptors.request.use((config) => {
  if (runtimeConfig.csrfToken) {
    config.headers.set('X-CSRF-Token', runtimeConfig.csrfToken);
  }
  return config;
});

function normalizeEndpoint(endpoint: string): string {
  return endpoint.replace(/^\/+/, '');
}

export function getApiErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'Request failed';
}

export async function getJson<T>(endpoint: string, options?: ApiRequestOptions): Promise<T> {
  return requestLegacy<T>(
    {
      method: 'GET',
      url: normalizeEndpoint(endpoint),
    },
    options,
  );
}

export async function postForm<T>(
  endpoint: string,
  data: FormRecord = {},
  options?: ApiRequestOptions,
): Promise<T> {
  return requestLegacy<T>(
    {
      method: 'POST',
      url: normalizeEndpoint(endpoint),
      data: toUrlEncodedForm(data),
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    },
    options,
  );
}

export async function uploadForm<T>(
  endpoint: string,
  data: FormData,
  options?: ApiRequestOptions,
): Promise<T> {
  return requestLegacy<T>(
    {
      method: 'POST',
      url: normalizeEndpoint(endpoint),
      data,
    },
    options,
  );
}

export async function downloadFile(endpoint: string, options?: ApiRequestOptions): Promise<Blob> {
  const mergedOptions = { ...defaultOptions, ...options };
  try {
    const response = await apiClient.request<Blob>({
      method: 'GET',
      url: normalizeEndpoint(endpoint),
      responseType: 'blob',
    });
    return response.data;
  } catch (error) {
    const apiError = toApiError(error, endpoint);
    handleApiError(apiError, mergedOptions);
    throw apiError;
  }
}

async function requestLegacy<T>(
  config: AxiosRequestConfig,
  options?: ApiRequestOptions,
): Promise<T> {
  const mergedOptions = { ...defaultOptions, ...options };
  try {
    const response = await apiClient.request<LegacyResponse<T>>(config);
    return unwrapLegacyResponse(response.data, normalizeEndpoint(config.url || ''));
  } catch (error) {
    const apiError = toApiError(error, config.url || '');
    handleApiError(apiError, mergedOptions);
    throw apiError;
  }
}

function unwrapLegacyResponse<T>(response: LegacyResponse<T>, endpoint: string): T {
  if (!isLegacyResponse<T>(response)) {
    return response as T;
  }

  if (!response.success) {
    throw new ApiError(response.msg || 'Request failed', {
      endpoint,
      response,
    });
  }

  return response.obj;
}

function toApiError(error: unknown, fallbackEndpoint: string): ApiError {
  if (error instanceof ApiError) {
    return error;
  }

  if (axios.isAxiosError(error)) {
    const status = error.response?.status;
    const endpoint = normalizeEndpoint(String(error.config?.url || fallbackEndpoint));
    const responseData = error.response?.data as Partial<LegacyResponse<unknown>> | undefined;
    const responseMessage = typeof responseData?.msg === 'string' ? responseData.msg : '';
    const sessionExpired = status === 401 || (status === 404 && endpoint.startsWith('panel/api/'));

    return new ApiError(responseMessage || error.message || 'Request failed', {
      status,
      endpoint,
      response: error.response?.data,
      sessionExpired,
    });
  }

  return new ApiError(getApiErrorMessage(error), {
    endpoint: normalizeEndpoint(fallbackEndpoint),
  });
}

function handleApiError(error: ApiError, options: Required<ApiRequestOptions>) {
  if (error.sessionExpired && options.redirectOnUnauthorized) {
    redirectToLogin();
    return;
  }
  if (options.notifyOnError) {
    void antMessage.error(error.message);
  }
}

function redirectToLogin() {
  const loginPath = `${getRuntimeConfig().basePath}panel/login`;
  if (window.location.pathname !== loginPath) {
    window.location.assign(loginPath);
  }
}

function isLegacyResponse<T>(value: unknown): value is LegacyResponse<T> {
  return Boolean(value && typeof value === 'object' && 'success' in value && 'obj' in value);
}

function toUrlEncodedForm(data: FormRecord): URLSearchParams {
  const body = new URLSearchParams();

  Object.entries(data).forEach(([key, value]) => {
    appendFormValue(body, key, value);
  });

  return body;
}

function appendFormValue(body: URLSearchParams, key: string, value: FormValue) {
  if (value === null || value === undefined) {
    return;
  }
  if (Array.isArray(value)) {
    value.forEach((item) => body.append(key, String(item)));
    return;
  }
  body.append(key, String(value));
}
