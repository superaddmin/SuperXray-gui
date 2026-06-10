# Phase 10 Legacy UI 退场执行记录

日期：2026-06-11

## 结论

旧 HTML UI 已完成退场执行：`web/html`、`web/assets` 和 `/panel/legacy*` 不再作为生产入口、回退入口或订阅可视页存在。新 Vue UI 保留 `/panel/` 默认入口与 `/panel/ui/` 兼容入口；登录接口、面板 API、设置控制器、Xray 控制器、订阅 URI/Base64/JSON/Clash/Mihomo/WireGuard/diagnose 输出继续保留。

本记录取代 2026-05-04 的“暂不能删除”分析结论。后续同步上游或排障时，不应把旧 HTML UI 当作必须保留的阶段门禁；需要恢复旧 UI 只能走显式回滚评审。

## 已执行退场范围

| 范围 | 当前状态 |
| --- | --- |
| `web/html/**` | 已删除，不再参与 Go template 渲染 |
| `web/assets/**` | 已删除，不再作为旧静态目录挂载 |
| `/panel/legacy*` | 已从 `web/controller/xui.go` 移除 |
| Go 旧模板链路 | `web/web.go` 不再 embed/load HTML 模板，不再 `SetHTMLTemplate` |
| 旧 assets 挂载 | `web/web.go` 与 `sub/sub.go` 不再 `StaticFS` 挂载旧资源 |
| 订阅 HTML 页面 | 浏览器 `Accept: text/html` 订阅请求也返回纯文本订阅内容 |
| Legacy 宽 CSP | 已移除 `unsafe-inline` / `unsafe-eval` 例外，统一 nonce CSP |
| 旧 sidebar/template 测试 | 已删除旧模板渲染相关测试 |

## 保留能力

| 能力 | 保留方式 |
| --- | --- |
| 新 Vue UI | `frontend/src` 构建到 `web/ui`，Go embed 托管 |
| 登录 | `POST /login`、`POST /getTwoFactorEnable` 与 `/panel/login` 新 UI 保留 |
| API | `/panel/api/**` 保留，继续走登录与 CSRF 保护 |
| 设置 | `/panel/api/setting/*` 与兼容 `/panel/setting/*` 控制器语义保留 |
| Xray | `/panel/api/xray/*` 与兼容 `/panel/xray/*` 控制器语义保留 |
| Xray 生命周期 | 仍走既有 XrayService，不由 CoreManager 接管 |
| 订阅输出 | URI/Base64、Xray JSON、Clash/Mihomo、WireGuard、diagnose 保留 |
| 数据模型 | 活跃写模型仍是 `database/model.Inbound` 与旧 JSON 字段 |
| QRious | 作为新 UI 静态资源迁移到 `frontend/public/assets/qrcode/` 与 `web/ui/assets/qrcode/` |

## 验收与回归要求

发布前至少验证：

```powershell
rg "go:embed assets|go:embed html|EmbeddedHTML|EmbeddedAssets|LoadHTMLFiles|SetHTMLTemplate|StaticFS\(|c\.HTML\(|/panel/legacy|panel/legacy|web/assets|web/html" web sub frontend tests -n -g '!*node_modules*' -g '!web/ui/**'
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
cd frontend
npm run typecheck
npm run lint
npm run test
npm run build
cd ..
python scripts/secret_scan.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```

允许存在的旧名词：历史计划、历史审计报告、`legacyEndpoints`（表示旧 API/数据模型兼容）和 E2E 文件名中的历史命名；不得存在旧 HTML UI 的运行时挂载、模板渲染、旧 assets 静态目录或 `/panel/legacy*` 路由。

## 回滚边界

若发布后需要恢复旧 HTML UI，不能直接从上游覆盖 `web/web.go`、`web/html`、`web/assets` 或 `xui.go`。必须先提交回滚评审，说明：

1. 为什么新 UI、旧 API或订阅纯文本输出不能满足运行需求。
2. 是否会重新引入 `unsafe-inline` / `unsafe-eval` CSP 例外。
3. 如何验证登录、设置、Xray、订阅和自定义 base path。
4. 如何在下一补丁版本再次移除旧 UI。
