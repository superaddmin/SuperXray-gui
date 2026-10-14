# Server Inbound And Subscription Audit Template

> This template is intentionally redacted. Do not store real subscription URLs, `subId`, client UUIDs, panel base paths, private key paths, proxy usernames, proxy passwords, emails, cookies, tokens, or database contents in the repository.
> 代码核对日期：2026-10-14。本文仅为脱敏审查模板，不是服务器实测报告；事实依据为 `sub/sub.go`、`sub/subController.go`、`sub/subscription_diagnostic.go`、`web/service/xray.go` 和 `frontend/src/utils/gatewayEgressMvp.ts`。

## Scope

- Server: `<SERVER_IP>`
- Time: `<UTC_TIMESTAMP>`
- Mode: read-only unless the change window explicitly allows mutation
- Operator: `<OPERATOR>`
- Deployment / version: `<systemd|docker>` / `<PANEL_VERSION_AND_XRAY_VERSION>`
- Evidence time and vantage point: `<CLIENT|ROUTER|SERVER>` / `<TIMESTAMP>`

## Service State

- x-ui service: `<active|inactive|failed>`
- Web port: `<WEB_PORT>`
- Subscription port: `<SUB_PORT>`
- Inbound port: `<INBOUND_PORT>`
- Panel and Xray process state are recorded separately: `<PANEL_STATE>` / `<XRAY_STATE>`
- Actual config / DB paths: `<REDACTED_LOCAL_PATHS>`

## Subscription State

- `subEnable`: `<true|false>`
- `subPath`: `<SUB_PATH>`
- `subClashEnable`: `<true|false>`
- `subJsonEnable`: `<true|false>`
- `subJsonPath` / `subClashPath`: `<SUB_JSON_PATH>` / `<SUB_CLASH_PATH>`
- Subscription URL: `<REDACTED_SUBSCRIPTION_URL>`
- Request format / target: `<uri|json|clash|wireguard>` / `<TARGET_CLIENT>`
- GET status / output node count: `<HTTP_STATUS>` / `<COUNT>`
- Diagnose summary / skipped reasons: `<REDACTED_AGGREGATES>`

订阅服务器独立于面板的 `webBasePath`。URI、JSON、Clash 路径分别由 `subPath`、`subJsonPath`、`subClashPath` 配置，JSON 与 Clash 的路由分别由 `subJsonEnable`、`subClashEnable` 控制；三个入口均注册 GET 与 `/:subid/diagnose`，没有 HEAD 路由。对实际入口执行 GET，并在本地核对内容及诊断；不要仅凭 HTTP 200 判定协议可用，不要把完整订阅响应附入公开报告。

## Inbound Summary

- tag: `<INBOUND_TAG>`
- protocol: `<PROTOCOL>`
- network: `<NETWORK>`
- security: `<SECURITY>`
- enabled: `<true|false>`
- client count: `<COUNT>`

## Egress Summary

- residential outbound tag: `<OUTBOUND_TAG>`
- country probe: `<COUNTRY_CODE>`
- exit IP: `<REDACTED_EXIT_IP_OR_PUBLIC_ONLY_IF_APPROVED>`
- timing samples: `<CONNECT_TLS_TTFB_TOTAL>`
- Gateway listen / manifest host reachable from its network namespace: `<REDACTED_RESULT>`
- Placeholder outbounds replaced / blocked rule and order verified: `<RESULT>`
- Positive and negative target probes: `<REDACTED_RESULTS>`

Gateway CSV 的 `manual-check` 和国家标签不是健康检测结果；实际出口、来源限制和规则命中需另行验证，保持客户端/中间设备/服务器至少两个视角的证据。

## Traffic Summary

- inbound total: `<BYTES>`
- outbound total: `<BYTES>`
- client total: `<BYTES>`

## Findings

- First blocker:
- Risk:
- Recommendation:
- Rollback:
- Not verified / next validation command:
