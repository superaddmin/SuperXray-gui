# SuperXray Vue UI

`frontend/` contains the active panel frontend. It is a Vue 3.5 and TypeScript 6 application built with Vite 8.

## Architecture

- Application source: `frontend/src`.
- Unit tests: `frontend/tests/*.test.ts`, executed with Node's test runner.
- Main libraries: Vue Router 4, Pinia 3, Ant Design Vue 4, and Axios.
- Routes: login, dashboard, logs, core instances, Xray, inbounds, settings, API docs, and the not-found view.
- API wrappers live in `src/api`; shared protocol and compatibility logic lives in `src/schemas` and `src/utils`.
- `vite.config.ts` writes production output to `../web/ui` with a relative asset base.
- `web/ui.go` embeds `web/ui` and serves the application at `/panel/`, with `/panel/ui/` retained as a compatible route.
- The Go server injects `window.__SUPERXRAY_UI_CONFIG__` at request time, including API/base paths, CSP nonce, CSRF token, UI base path, and version.

`frontend/src` is the source of truth. Do not hand-edit generated files under `web/ui`. The retired `web/html` and `web/assets` directories are not part of the current UI.

## Commands

Run commands from this directory:

```powershell
npm ci
npm run dev
npm run typecheck
npm run lint
npm run format
npm run test
npm run build
npm run preview
```

`npm run build` first runs `npm run gen:openapi`, then `vue-tsc -b` and `vite build`. Vite clears and recreates `web/ui`, so include generated output only when the requested change requires it.

## Development and verification

- Use `npm run dev` for the Vite development server.
- Run `npm run typecheck`, `npm run lint`, and `npm run test` for frontend source changes.
- Run `npm run build` when validating the embedded production bundle or changing generated OpenAPI output.
- If a change affects runtime config injection, base paths, CSP, CSRF, static caching, or Go route mounting, also run from the repository root:

```powershell
go test ./web ./web/locale
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin/SuperXray.exe ./main.go
```

The UI continues to use the existing Go APIs. Active Xray writes remain on the legacy-compatible `database/model.Inbound` contract until the project phase gates explicitly change it.
