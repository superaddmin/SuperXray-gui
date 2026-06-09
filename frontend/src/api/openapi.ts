import { getRuntimeConfig } from '@/types/runtime';
import type { OpenAPIDocument } from '@/types/openapi';

export class OpenAPIDocumentLoadError extends Error {
  readonly sessionExpired: boolean;
  readonly status: number;

  constructor(status: number) {
    super(`OpenAPI document request failed: ${status}`);
    this.name = 'OpenAPIDocumentLoadError';
    this.status = status;
    this.sessionExpired = status === 404;
  }
}

export function getOpenAPIDocumentURL(): string {
  const runtime = getRuntimeConfig();
  const basePath = runtime.basePath.endsWith('/') ? runtime.basePath : `${runtime.basePath}/`;
  return `${basePath}panel/api/openapi.json`;
}

export async function fetchOpenAPIDocument(): Promise<OpenAPIDocument> {
  const response = await fetch(getOpenAPIDocumentURL(), {
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      'X-Requested-With': 'XMLHttpRequest',
    },
  });

  if (!response.ok) {
    const error = new OpenAPIDocumentLoadError(response.status);
    if (error.sessionExpired) {
      redirectToLogin();
    }
    throw error;
  }

  return (await response.json()) as OpenAPIDocument;
}

function redirectToLogin() {
  const loginPath = `${getRuntimeConfig().basePath}panel/login`;
  if (window.location.pathname !== loginPath) {
    window.location.assign(loginPath);
  }
}
