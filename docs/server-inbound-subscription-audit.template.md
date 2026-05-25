# Server Inbound And Subscription Audit Template

> This template is intentionally redacted. Do not store real subscription URLs, `subId`, client UUIDs, panel base paths, private key paths, proxy usernames, proxy passwords, emails, cookies, tokens, or database contents in the repository.

## Scope

- Server: `<SERVER_IP>`
- Time: `<UTC_TIMESTAMP>`
- Mode: read-only unless the change window explicitly allows mutation
- Operator: `<OPERATOR>`

## Service State

- x-ui service: `<active|inactive|failed>`
- Web port: `<WEB_PORT>`
- Subscription port: `<SUB_PORT>`
- Inbound port: `<INBOUND_PORT>`

## Subscription State

- `subEnable`: `<true|false>`
- `subPath`: `<SUB_PATH>`
- `subClashEnable`: `<true|false>`
- `subJsonEnable`: `<true|false>`
- Subscription URL: `<REDACTED_SUBSCRIPTION_URL>`

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

## Traffic Summary

- inbound total: `<BYTES>`
- outbound total: `<BYTES>`
- client total: `<BYTES>`

## Findings

- First blocker:
- Risk:
- Recommendation:
- Rollback:
