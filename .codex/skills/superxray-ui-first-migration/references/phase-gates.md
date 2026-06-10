# SuperXray Phase Gates

## Current State

- Phase 9: security closeout and compatibility stabilization.
- Phase 10: risk-accepted minimal backend runtime entry only.
- Full Phase 10 confidence still needs CI or second-environment reproduction of the local non-skipped E2E acceptance.

## Phase Summary

| Phase | Theme | Gate |
| --- | --- | --- |
| 0 | Baseline and freeze | E2E baseline and parity checklist exist |
| 1 | Frontend shell | Vue/Vite shell, router, layout, Pinia |
| 2 | Go static integration | `/panel/`, `/panel/ui/`, embed, runtime config, CSP nonce |
| 3 | API SDK/types | Old API SDK and types centralized |
| 4 | Read-only Dashboard/logs/config | No HTML rendering sinks; no writes |
| 5 | Xray lifecycle/config | Old Xray APIs remain the lifecycle path |
| 6 | Inbounds/clients | New UI writes remain legacy-readable |
| 7 | Settings/subscription/backup | Old setting/server APIs remain compatible and CSRF-protected |
| 8 | Default entry gray switch | `/panel/` new UI, `/panel/ui/` compatibility entry |
| 9 | Security closeout | CSP/CSRF/download/import/XSS and compatibility checks |
| 10.1 | default-xray read-only | Read-only CoreInstances view, no lifecycle takeover |
| 10.2 | lifecycle gate | Requires explicit approval and old API behavior comparison |
| 10.3 | experimental sing-box | Isolated adapter, not production default |
| 10.4+ | capability/multi-core | Requires new data/model/API gates |

## Forbidden Until Explicit Approval

- Routing legacy Xray start/stop/restart through CoreManager before Phase 10.2.
- Creating `proxy_inbounds` or `proxy_clients` as active write paths.
- Migrating Client out of `Inbound.Settings` JSON.
- Reintroducing `/panel/legacy/`, `web/html`, or `web/assets` after legacy retirement.
- Promoting `experimental-sing-box` to production default.
- Creating production `egress_*` DB/API from Gateway MVP docs.
- Rendering untrusted content with `v-html`, `innerHTML`, or `insertAdjacentHTML`.

## Required Checks By Risk

- UI write path: frontend typecheck/lint/test/build plus old API/legacy readability evidence.
- Security path: Go security tests, sink searches, CSRF/CSP checks, secret scan.
- Core runtime path: `go test ./core/... ./web/service ./web/controller` plus forbidden schema/lifecycle search.
- Subscription path: `go test ./sub ./web/service` plus affected format matrix.
- Release path: release gate script plus Go/frontend/E2E status.
