# Phase 10+ Egress Governance Design

## 1. Purpose

This document defines the Phase 10+ design for a first-class SuperXray-gui egress governance subsystem. It is intentionally separate from the Xray-compatible MVP plan because this design introduces new backend state, APIs, scheduled probes, and management UI.

The design may be implemented only after Phase 10+ review accepts the new data model and operational risk. It must not be mixed into the MVP that only generates Xray-compatible config.

## 2. Phase Boundary

Allowed in this design phase after approval:
- New backend schema for egress governance.
- New backend services for probe scheduling, result persistence, switch event logging, and Gateway export.
- New UI page or Xray subpage for egress group management.
- Optional CoreManager integration for future non-Xray cores after the existing Xray lifecycle boundary is explicitly accepted.

Still forbidden unless separately approved:
- Migrating old `model.Inbound`.
- Replacing legacy Xray lifecycle with CoreManager lifecycle.
- Making `experimental-sing-box` a default production core.
- Writing secrets into logs, frontend-visible JSON, screenshots, or exported manifests.
- Allowing arbitrary probe URLs that create SSRF risk.

## 3. Domain Model

### 3.1 `egress_groups`

Purpose: logical routing pools such as `openai-egress`, `anthropic-egress`, `gemini-egress`, `region-us`, and `region-jp`.

Suggested fields:

| Field | Type | Required | Notes |
|---|---|---:|---|
| `id` | integer | yes | Primary key |
| `group_key` | string | yes | Stable unique key, e.g. `openai-egress` |
| `name` | string | yes | Admin display name |
| `kind` | string enum | yes | `platform`, `region`, `fallback` |
| `platform` | string nullable | no | `openai`, `anthropic`, `gemini`, or null |
| `region_code` | string nullable | no | ISO-like region code such as `US`, `JP`, `HK` |
| `expected_country_code` | string nullable | no | Expected exit country for health classification |
| `strategy` | string enum | yes | `primary_backup`, `weighted`, `manual` |
| `sticky_scope` | string enum | yes | `account`, `platform`, `region`, `none` |
| `rule_version` | string | yes | Version of the domain/routing rules applied to this group |
| `status` | string enum | yes | `active`, `inactive`, `draining` |
| `health_status` | string enum | yes | `healthy`, `warn`, `critical`, `unknown` |
| `created_at` | timestamp | yes | Server time |
| `updated_at` | timestamp | yes | Server time |

Indexes:
- Unique index on `group_key`.
- Index on `(kind, platform, region_code)`.
- Index on `health_status`.

### 3.2 `egress_nodes`

Purpose: concrete outbounds or external network paths inside an egress group.

Suggested fields:

| Field | Type | Required | Notes |
|---|---|---:|---|
| `id` | integer | yes | Primary key |
| `group_id` | integer | yes | FK to `egress_groups.id` |
| `node_key` | string | yes | Stable unique key, e.g. `openai-us-wireguard-1` |
| `name` | string | yes | Admin display name |
| `core_instance_id` | string nullable | no | Future CoreManager instance reference |
| `xray_outbound_tag` | string nullable | no | Existing Xray outbound tag for Xray-backed nodes |
| `protocol` | string | yes | `wireguard`, `vless`, `trojan`, `socks`, `http`, etc. |
| `expected_country_code` | string | yes | Expected exit country |
| `expected_asn` | string nullable | no | Optional ASN expectation |
| `expected_isp` | string nullable | no | Optional ISP expectation |
| `priority` | integer | yes | Lower number wins for `primary_backup` |
| `weight` | integer | yes | Used for `weighted` |
| `status` | string enum | yes | `active`, `inactive`, `draining`, `quarantined` |
| `health_status` | string enum | yes | `healthy`, `warn`, `critical`, `unknown` |
| `last_exit_ip_masked` | string nullable | no | Masked display value only |
| `last_exit_ip_hash` | string nullable | no | Stable comparison without leaking full IP |
| `last_country_code` | string nullable | no | Last detected country |
| `last_checked_at` | timestamp nullable | no | Latest probe time |
| `last_error_class` | string nullable | no | Redacted class such as `dns`, `tls`, `timeout`, `http_status` |
| `last_error_redacted` | string nullable | no | No credentials or request bodies |
| `created_at` | timestamp | yes | Server time |
| `updated_at` | timestamp | yes | Server time |

Indexes:
- Unique index on `node_key`.
- Index on `group_id`.
- Index on `(group_id, priority, status)`.
- Index on `health_status`.

### 3.3 `egress_probe_results`

Purpose: append-only probe history for country checks, API reachability, DNS, and stream stability.

Suggested fields:

| Field | Type | Required | Notes |
|---|---|---:|---|
| `id` | integer | yes | Primary key |
| `group_id` | integer | yes | FK to `egress_groups.id` |
| `node_id` | integer | yes | FK to `egress_nodes.id` |
| `probe_type` | string enum | yes | `exit_ip`, `country`, `platform_api`, `dns`, `stream` |
| `target` | string | yes | Probe target hostname or logical target |
| `success` | boolean | yes | Probe result |
| `latency_ms` | integer nullable | no | End-to-end probe latency |
| `http_status` | integer nullable | no | HTTP status when applicable |
| `exit_ip_masked` | string nullable | no | Masked IP |
| `exit_ip_hash` | string nullable | no | Hash for IP change detection |
| `country_code` | string nullable | no | Detected country |
| `asn` | string nullable | no | Detected ASN |
| `isp` | string nullable | no | Detected ISP |
| `error_class` | string nullable | no | `dns`, `connect`, `tls`, `timeout`, `http_status`, `country_mismatch` |
| `error_redacted` | string nullable | no | Redacted short error |
| `checked_at` | timestamp | yes | Probe time |
| `expires_at` | timestamp | yes | Retention boundary |

Indexes:
- Index on `(node_id, checked_at DESC)`.
- Index on `(group_id, probe_type, checked_at DESC)`.
- Index on `(success, checked_at DESC)`.

Retention:
- Keep high-resolution probe rows for 7 days.
- Keep rolled-up hourly aggregates for 90 days.
- Never store request bodies, prompts, API keys, cookies, or full Authorization headers.

### 3.4 `egress_switch_events`

Purpose: audit every automatic or manual switch decision.

Suggested fields:

| Field | Type | Required | Notes |
|---|---|---:|---|
| `id` | integer | yes | Primary key |
| `group_id` | integer | yes | FK to `egress_groups.id` |
| `from_node_id` | integer nullable | no | Previous node |
| `to_node_id` | integer nullable | no | New node |
| `reason` | string enum | yes | `probe_failure`, `country_mismatch`, `manual`, `release`, `recovery`, `rollback` |
| `triggered_by` | string enum | yes | `system`, `admin`, `release` |
| `triggered_by_user_id` | integer nullable | no | Admin ID when manual |
| `previous_group_health` | string | yes | Previous group health |
| `next_group_health` | string | yes | New group health |
| `rollout_percent` | integer | yes | `0` to `100` |
| `message_redacted` | string | yes | Human-readable audit summary |
| `created_at` | timestamp | yes | Event time |

Indexes:
- Index on `(group_id, created_at DESC)`.
- Index on `(reason, created_at DESC)`.
- Index on `triggered_by_user_id`.

## 4. API Design

All endpoints live under `/panel/api/egress/*` and require the same authenticated admin session and CSRF protections as existing panel mutation APIs.

### 4.1 Groups

```text
GET    /panel/api/egress/groups
POST   /panel/api/egress/groups
GET    /panel/api/egress/groups/:id
PUT    /panel/api/egress/groups/:id
POST   /panel/api/egress/groups/:id/disable
POST   /panel/api/egress/groups/:id/enable
GET    /panel/api/egress/groups/:id/summary
```

Response shape for `GET /panel/api/egress/groups`:

```json
{
  "success": true,
  "obj": [
    {
      "id": 1,
      "group_key": "openai-egress",
      "name": "OpenAI Egress",
      "kind": "platform",
      "platform": "openai",
      "region_code": "US",
      "expected_country_code": "US",
      "strategy": "primary_backup",
      "sticky_scope": "account",
      "rule_version": "2026-05-16-openai-v1",
      "status": "active",
      "health_status": "healthy"
    }
  ]
}
```

### 4.2 Nodes

```text
GET    /panel/api/egress/groups/:group_id/nodes
POST   /panel/api/egress/groups/:group_id/nodes
GET    /panel/api/egress/nodes/:id
PUT    /panel/api/egress/nodes/:id
POST   /panel/api/egress/nodes/:id/drain
POST   /panel/api/egress/nodes/:id/quarantine
POST   /panel/api/egress/nodes/:id/activate
```

Mutation rules:
- `xray_outbound_tag` must reference an existing Xray outbound tag when the node is Xray-backed.
- `core_instance_id` must be optional until CoreManager production lifecycle is accepted.
- Credentials must be referenced by a secret handle or existing Xray config, not copied into `egress_nodes`.

### 4.3 Probes

```text
GET    /panel/api/egress/nodes/:id/probes
POST   /panel/api/egress/nodes/:id/probes/run
GET    /panel/api/egress/groups/:group_id/probes/latest
```

Allowed probe target classes:
- Fixed exit IP detection endpoint configured by administrator.
- Platform API reachability targets from versioned allowlist.
- DNS targets from versioned allowlist.
- Stream stability endpoint from versioned allowlist.

Probe requests must reject:
- Link-local, loopback, private, and metadata IP ranges unless explicitly running a local-only safety check.
- User-submitted arbitrary URLs.
- Request bodies containing real prompt or user payload.

### 4.4 Switch Events

```text
GET    /panel/api/egress/groups/:group_id/switch-events
POST   /panel/api/egress/groups/:group_id/manual-switch
POST   /panel/api/egress/groups/:group_id/rollback
```

Manual switch request:

```json
{
  "to_node_id": 12,
  "rollout_percent": 5,
  "reason": "manual",
  "message": "Canary OpenAI traffic to backup US node"
}
```

### 4.5 Gateway Export

```text
GET /panel/api/egress/gateway-manifest.csv
GET /panel/api/egress/gateway-manifest.json
```

CSV fields:

```csv
name,protocol,host,port,platform,region_code,expected_country_code,egress_group,health_status,notes
```

Export rules:
- Host must be `127.0.0.1` or `::1`.
- Full exit IP is not exported.
- Credentials are exported only if the admin explicitly requests a secret-bearing export and confirms the danger prompt.

## 5. UI Design

Recommended placement: a new "Egress" route after Xray, or an "Egress Governance" tab inside Xray for the first Phase 10+ iteration.

Views:
- **Groups Overview:** cards for platform groups and region groups, with health, active node, expected country, rule version, and last probe time.
- **Group Detail:** node table, current route strategy, sticky scope, health timeline, latest probe summary.
- **Node Detail Drawer:** expected country/ASN, backend outbound tag, latest exit IP masked value, recent errors, probe history.
- **Switch Events:** audit table with reason, from/to node, rollout percent, operator, and timestamp.
- **Gateway Manifest:** CSV/JSON preview, copy, download, and "only loopback hosts" validation badge.

Dangerous controls:
- Manual switch requires typed confirmation when moving production traffic.
- Rollback requires showing previous active node and last healthy probe time.
- Quarantine requires reason text.
- Secret-bearing export requires a second confirmation and must be disabled by default.

## 6. Probe and Health State Machine

Node state:

```text
unknown -> healthy -> warn -> critical -> quarantined
critical -> recovering -> healthy
```

Default transitions:
- `warn`: 3 consecutive platform or DNS probe failures.
- `critical`: 5 consecutive failures, country mismatch, or stream stability failure above configured threshold.
- `quarantined`: manual action or repeated country mismatch.
- `recovering`: 3 consecutive successful probes after critical.
- `healthy`: stable probes for 10 minutes after recovery.

Group health:
- `healthy`: at least one active healthy node and no country mismatch.
- `warn`: primary is degraded but backup is healthy.
- `critical`: no healthy node is available for the group.
- `unknown`: no recent probe data.

## 7. Security Requirements

- All mutation endpoints require CSRF validation.
- Logs may include platform, group key, node key, target hostname, status code, latency, and redacted error class.
- Logs must not include prompt text, request body, Authorization header, Cookie, API key, OAuth token, proxy password, WireGuard private key, or full exit IP.
- Probe targets must be allowlisted to avoid SSRF.
- Gateway-facing ports must be validated as loopback-only before a node or group can become `healthy`.
- UI must render logs and config previews as text, never through `v-html`.

## 8. Migration and Rollback

Migration sequence after approval:

1. Add tables with no active routing behavior.
2. Add read-only APIs and UI listing empty state.
3. Add node/group CRUD behind an admin-only feature flag.
4. Add manual probe execution.
5. Add scheduled probe execution.
6. Add Gateway manifest export.
7. Add manual switch and rollback audit events.
8. Add automated health state transitions.
9. Consider automated routing changes only after separate production-readiness review.

Rollback:
- Disable feature flag to hide UI and stop scheduler.
- Keep tables for audit unless a release explicitly requires dropping them.
- Existing Xray config remains source of truth until automated routing changes are approved.
- If automated changes are approved later, each generated config write must store a pre-change snapshot.

## 9. Test Strategy

Backend:
- Unit tests for schema validation, target allowlist, IP masking, health transitions, and event logging.
- Controller tests for auth, CSRF, invalid IDs, forbidden probe targets, and secret redaction.
- Scheduler tests using fake clock and fake probe client.

Frontend:
- Source tests for route/page presence, no `v-html`, manifest columns, dangerous confirmation text, and status badges.
- Component tests or browser checks for 375px, 768px, and 1440px layouts.

E2E:
- Create a test group and node.
- Run a fake probe.
- Verify latest health appears in UI.
- Export Gateway manifest and confirm loopback host.
- Trigger manual switch and rollback with audit event visible.

Validation commands:

```powershell
cd frontend
npm run test
npm run typecheck
npm run lint
npm run build
```

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

## 10. Acceptance Criteria

- `egress_groups`, `egress_nodes`, `egress_probe_results`, and `egress_switch_events` have reviewed schema and indexes.
- APIs expose group/node/probe/event operations with auth and CSRF protections.
- Probe targets are allowlisted and reject SSRF-prone destinations.
- Full credentials and full exit IP values are not logged or exported by default.
- UI lets administrators inspect health and export Gateway manifest without editing raw JSON.
- Rollback and manual switch events are auditable.
- Existing Xray lifecycle and `model.Inbound` remain unchanged until a separate migration is approved.
