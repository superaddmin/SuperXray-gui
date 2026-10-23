# CLAUDE.md

This repository keeps project facts in shared source-of-truth files. Read those files instead of relying on a duplicated architecture summary here.

## Start here

1. Follow the active `AGENTS.md` instructions.
2. Read `.codex/project.toml`, `.codex/governance.toml`, `.codex/routing.toml`, and `.codex/context/project-map.md`.
3. Select verification from `.codex/workflows/verification-matrix.md`.
4. Read only the source files and active plans relevant to the task.

Current anchors:

- Module: `github.com/superaddmin/SuperXray-gui/v2`.
- Go: `1.27.2` from `go.mod`.
- Xray runtime distribution: `v26.10.21`; configuration compatibility baseline: `v26.10.10` (`701af60772cd`). The exact Go module pin and gRPC development-version pin are in `go.mod` / `go.sum`.
- Upstream integration scope, preserved customizations, and unresolved limits: `docs/upstream-3x-ui-v3.10.2-integration.md`. Inventory entries are not proof that every upstream change was ported.
- Vue source: `frontend/src`; scripts and versions: `frontend/package.json`.
- Vite output: `web/ui`; Go embedding and routes: `web/ui.go`.
- Legacy `web/html`, `web/assets`, and `/panel/legacy*` paths are retired.
- Current phase and exceptions: `plans/STATUS.md` and the phase references listed by `.codex/project.toml`.

## Working rules

- Make the smallest change in the routed owner domain.
- Preserve `database/model.Inbound` as the active Xray write model.
- Keep the legacy Xray lifecycle outside CoreManager until its phase gate changes.
- Preserve CSP/CSRF, authenticated downloads, safe import, path, URL, and execution checks.
- Keep secrets, private keys, runtime databases, subscription data, and audit artifacts out of Git.
- Update user-facing documentation when behavior or interfaces change.

## 服务器部署交付规范

- 新服务器部署统一遵循 `.codex/skills/superxray-server-deployment/SKILL.md`，并按 `references/deployment-runbook.md` 的“标准订阅交付格式”输出交付信息。
- 标准交付固定包含：面板 HTTPS 地址与账号、URI/Base64、Xray JSON、Clash/Mihomo 长期链接、SOCKS5 连接参数（协议/主机/端口/用户名/密码）及其 Clash 订阅链接、三个月与六个月 Clash/Mihomo 链接及 UTC 到期时间。
- 三个月/六个月订阅是 VLESS 主节点的独立 client；SOCKS5 只保留一个长期订阅，不生成限时 SOCKS5 链接。
- 有可用证书时，面板与订阅服务默认启用 HTTPS，并开启 `subJsonEnable`；交付前必须验证 `/sub/`、`/json/`、`/clash/` 三个入口均为 HTTPS 200。
- 真实主机、凭据、subId 和订阅 URL 只写入 `.local/ops/<deployment-id>/`，不得提交到 Git。

## Common verification

Installed npm dependencies can contain unrelated Go fixtures, so enumerate project packages and exclude import paths under `node_modules` before full Go tests.

```powershell
$goPackages = @(go list ./... | Where-Object { $_ -notmatch '/node_modules/' })
if ($LASTEXITCODE -ne 0 -or $goPackages.Count -eq 0) { throw 'go list failed or returned no project packages' }
go test $goPackages
go vet $goPackages
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin/SuperXray.exe ./main.go

Set-Location frontend
npm ci
npm run typecheck
npm run lint
npm run test
npm run build
Set-Location ..

python scripts/secret_scan.py
```

Release work additionally follows `.github/agentic-workflows/release.md` and its release gate. Report only commands that were actually executed.
