interface SuperXrayUiConfig {
  apiBasePath?: string;
  basePath?: string;
  cspNonce?: string;
  csrfToken?: string;
  uiBasePath?: string;
  version?: string;
}

declare global {
  interface Window {
    __SUPERXRAY_UI_CONFIG__?: SuperXrayUiConfig;
  }
}

interface RuntimeConfig {
  apiBasePath: string;
  basePath: string;
  cspNonce: string;
  csrfToken: string;
  uiBasePath: string;
  version: string;
}

const fallbackConfig: RuntimeConfig = {
  apiBasePath: '/',
  basePath: '/',
  cspNonce: '',
  csrfToken: '',
  uiBasePath: '/',
  version: 'dev',
};

function normalizePath(value: string): string {
  if (!value) {
    return '/';
  }
  const withLeadingSlash = value.startsWith('/') ? value : `/${value}`;
  return withLeadingSlash.endsWith('/') ? withLeadingSlash : `${withLeadingSlash}/`;
}

export function getRuntimeConfig(): RuntimeConfig {
  const config = window.__SUPERXRAY_UI_CONFIG__ || {};

  return {
    apiBasePath: normalizePath(config.apiBasePath || config.basePath || fallbackConfig.apiBasePath),
    basePath: normalizePath(config.basePath || fallbackConfig.basePath),
    cspNonce: config.cspNonce || fallbackConfig.cspNonce,
    csrfToken: config.csrfToken || fallbackConfig.csrfToken,
    uiBasePath: normalizePath(config.uiBasePath || fallbackConfig.uiBasePath),
    version: config.version || fallbackConfig.version,
  };
}

export function hasInjectedRuntimeConfig(): boolean {
  return Boolean(window.__SUPERXRAY_UI_CONFIG__);
}
