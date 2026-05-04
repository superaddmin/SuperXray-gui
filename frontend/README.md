# SuperXray New UI

This directory contains the Phase 1 Vue 3/Vite frontend shell.

## Scope

- Vue 3 + Vite + TypeScript application shell.
- Vue Router, Pinia, Axios, and Ant Design Vue 4 dependencies.
- Relative Vite build output with `base: ""`.
- Production build output is written to `../web/ui` for Go embedding in Phase 2.
- Reserved routes for Dashboard, Logs, Xray, Inbounds, and Settings.
- Phase 3 API wrappers under `src/api` and legacy response types under `src/types`.
- Phase 4 read-only Dashboard, Logs, and Xray config preview.
- Phase 5 Xray lifecycle, version management, and legacy-compatible template editing.
- No migrated multi-core write flows.

## Commands

```powershell
npm install
npm run typecheck
npm run lint
npm run build
npm run dev
```

`npm run build` must run before Go release builds so that `web/ui` exists for `go:embed`.

## Runtime Config

Phase 2 should inject `window.__SUPERXRAY_UI_CONFIG__` before the Vite bundle:

```ts
window.__SUPERXRAY_UI_CONFIG__ = {
  apiBasePath: '/',
  basePath: '/',
  cspNonce: '',
  csrfToken: '',
  uiBasePath: '/panel/ui/',
  version: 'dev',
};
```

The new UI is designed to keep using the legacy Xray APIs until the parity gates are complete.
