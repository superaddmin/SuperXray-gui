# SuperXray UI-First Phase Gates

## Phase Map

| Phase | Scope | Hard gate |
|---|---|---|
| 0 | Inventory and E2E baseline | No core behavior changes |
| 1 | New Vue 3/Vite shell | No old API semantics changes |
| 2 | Go static integration | Legacy `/panel` remains intact |
| 3 | API SDK/types | Components do not hardcode API URLs |
| 4 | Read-only Dashboard/logs/config | No `v-html`; no writes |
| 5 | Xray lifecycle/config | Old UI can still read saved config |
| 6 | Inbounds/clients | New writes are legacy-compatible |
| 7 | Settings/subscription/backup | Subscription output matches legacy |
| 8 | Gray switch | `/panel/legacy` works |
| 9 | Security closeout | New UI CSP/CSRF/XSS gates pass |
| 10.1 | default-xray read-only | No behavior change |
| 10.2 | CoreManager wraps Xray | Old and new APIs behave the same |
| 10.3 | sing-box experimental | Does not affect Xray |
| 10.4 | Capability schema | Schema versioned and backend validated |
| 10.5 | SubscriptionNode | Legacy Xray subscription unchanged |

## Forbidden Before Phase 10

- Creating `proxy_inbounds` or `proxy_clients` as active write paths.
- Moving old inbounds out of `model.Inbound`.
- Routing Xray startup through a new supervisor.
- Adding sing-box UI or lifecycle actions.
- Removing the legacy UI.

## Mandatory Checks

Backend:

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

Frontend:

```powershell
cd frontend
npm run typecheck
npm run lint
npm run build
```

E2E must cover the user-visible flow changed by the phase.
