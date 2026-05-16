# SuperXray-gui 项目地图

## 架构概览

SuperXray-gui 是基于 Go/Gin 的 Xray-core 管理面板。主进程启动两个 HTTP 服务：

- Web Server：管理面板、旧 UI、新 Vue UI、REST API、WebSocket、设置、Xray 生命周期。
- Sub Server：订阅服务，输出 URI/Base64、Xray JSON、Clash/Mihomo、WireGuard 配置和 diagnose 信息。

核心请求链路：

```text
Browser/API Client
  -> Gin middleware
  -> web/controller
  -> web/service
  -> database/model or xray API or core Manager
```

## 关键目录

| 路径 | 说明 | 主责代理 |
| --- | --- | --- |
| `main.go` | CLI、服务启动、信号处理、TG Bot 停止顺序 | `superxray-backend-service-guardian` |
| `config/` | 版本、应用名、路径和环境变量 | `superxray-backend-service-guardian` |
| `database/` | SQLite/GORM 初始化、模型、seeders | `superxray-database-steward` |
| `web/controller/` | Gin controller、API 路由、Legacy 页面入口、Core API | `superxray-go-integration` |
| `web/service/` | 业务逻辑：Inbound、Setting、Server、Xray、TgBot、Core、Geo | `superxray-backend-service-guardian` |
| `web/middleware/` | CSP、CSRF、Domain、Redirect 等中间件 | `superxray-security-gate` |
| `web/job/` | cron 任务：流量、IP、日志、LDAP、通知 | `superxray-backend-service-guardian` |
| `web/websocket/` | WebSocket hub 和广播通知 | `superxray-backend-service-guardian` |
| `web/html`, `web/assets` | Legacy UI 与旧静态资源，当前仍是回退边界 | `superxray-go-integration` |
| `frontend/` | Vue 3/Vite 新 UI 源码 | `superxray-frontend-migrator` |
| `web/ui` | Vite 构建输出，Go 嵌入资源 | `superxray-frontend-migrator` |
| `sub/` | 订阅服务器与协议输出 | `superxray-subscription-protocol-specialist` |
| `xray/` | Xray 进程、gRPC API、配置片段 | `superxray-backend-service-guardian` |
| `core/` | CoreManager、default-xray、experimental sing-box | `superxray-core-runtime-architect` |
| `tests/e2e/` | Playwright 阶段验收 | `superxray-e2e-gate` |
| `.github/`, `Dockerfile`, `install.sh` | CI/CD、容器、安装更新 | `superxray-devops-cicd-maintainer` |
| `docs/`, `plans/`, `README*.md` | 长期文档、阶段状态、用户说明 | `superxray-docs-i18n-maintainer` |

## 当前业务主线

- Phase 0-8 已完成新 UI 壳、Go 静态接入、旧 API SDK、只读页面、Xray 生命周期、入站/客户端、设置/订阅/备份和默认入口灰度。
- Phase 9 聚焦安全收口：CSP/CSRF/XSS、下载鉴权、数据库导入安全、新旧 UI 兼容抽检。
- Phase 10 已有最小 CoreManager/sing-box 入口，但仍是风险接受后的受控入口，不得接管旧 Xray 主路径。

## 数据契约

- 活跃入站模型仍是 `database/model.Inbound`。
- 客户端配置嵌入在 `Inbound.Settings` JSON 中，不是独立表。
- `StreamSettings`、`Sniffing`、`Settings` JSON 必须保持旧 UI 和新 UI 可读。
- 订阅服务读取旧模型生成 URI/JSON/Clash/Mihomo/WireGuard 输出。

## 常用验证命令

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
cd frontend
npm run typecheck
npm run lint
npm run test
npm run build
cd ..
npm run e2e
```
