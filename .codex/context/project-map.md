# SuperXray-gui 项目地图

## 架构概览

SuperXray-gui 是基于 Go/Gin 的 Xray-core 管理面板。当前代码库处于“新 Vue 3 UI 默认入口 + legacy UI 回退 + Phase 9 安全收口 + 风险接受的最小 Phase 10 Core API 入口”状态。

主进程启动两个 HTTP 服务：

- Web Server：管理面板、新 Vue UI、legacy UI、REST API、WebSocket、设置、Xray 生命周期、后台任务。
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
| `web/service/` | Inbound、Setting、Server、Xray、TgBot、Core、Geo、WARP/Nord | `superxray-backend-service-guardian` |
| `web/middleware/` | CSP、CSRF、Domain、Redirect 等中间件 | `superxray-security-gate` |
| `web/job/` | cron 任务：流量、IP、日志、LDAP、通知 | `superxray-backend-service-guardian` |
| `web/websocket/` | WebSocket hub 和广播通知 | `superxray-backend-service-guardian` |
| `web/html`, `web/assets` | Legacy UI 与旧静态资源，当前仍是回退边界 | `superxray-go-integration` |
| `frontend/` | Vue 3/Vite/TypeScript 新 UI 源码 | `superxray-frontend-migrator` |
| `web/ui` | Vite 构建输出，Go 嵌入资源 | `superxray-frontend-migrator` |
| `sub/` | 订阅服务器与协议输出 | `superxray-subscription-protocol-specialist` |
| `xray/` | Xray 进程、gRPC API、配置片段和 traffic | `superxray-backend-service-guardian` |
| `core/` | CoreManager、default-xray、experimental sing-box | `superxray-core-runtime-architect` |
| `tests/e2e/` | Playwright 阶段验收 | `superxray-e2e-gate` |
| `.github/`, `Dockerfile`, `install.sh` | CI/CD、容器、安装更新 | `superxray-devops-cicd-maintainer` |
| `docs/`, `plans/`, `README*.md` | 长期文档、阶段状态、用户说明 | `superxray-docs-i18n-maintainer` |
| `.codex/` | 项目级 AI 协作配置、技能和路由 | `superxray-ui-program-manager` |

## 核心依赖

后端核心依赖：

- `github.com/gin-gonic/gin`：Web/API 路由。
- `gorm.io/gorm` + `gorm.io/driver/sqlite`：SQLite 数据访问。
- `github.com/xtls/xray-core` + `google.golang.org/grpc`：Xray API 类型与 gRPC 集成。
- `github.com/robfig/cron/v3`：后台任务。
- `github.com/gorilla/websocket`：实时广播。
- `github.com/mymmrac/telego`、`github.com/go-ldap/ldap/v3`、`github.com/xlzd/gotp`：TG Bot、LDAP、TOTP。

前端核心依赖：

- `vue`、`vue-router`、`pinia`：SPA、路由和状态。
- `ant-design-vue`、`@ant-design/icons-vue`：组件和图标。
- `axios`：旧 API SDK。
- `vite`、`vue-tsc`、`eslint`、`node --test`：构建、类型、检查和单测。

## 当前业务主线

- Phase 0-8 已完成新 UI 壳、Go 静态接入、旧 API SDK、只读页面、Xray 生命周期、入站/客户端、设置/订阅/备份和默认入口灰度。
- Phase 9 聚焦安全收口：CSP/CSRF/XSS、下载鉴权、数据库导入安全、新旧 UI 兼容抽检。
- 本机隔离真实 Xray core 环境已完成 19 条 Playwright 验收，仍缺 CI/第二环境复刻。
- Phase 10 已风险接受并强制进入最小后端入口：CoreManager 注册表、`default-xray` 只读观察、`experimental-sing-box` 外部适配器。
- Phase 10 不放宽旧模型迁移、旧订阅语义、旧 Xray 生命周期和 legacy UI 回退边界。

## 数据契约

- 活跃入站模型仍是 `database/model.Inbound`。
- 客户端配置嵌入在 `Inbound.Settings` JSON 中，不是独立表。
- `StreamSettings`、`Sniffing`、`Settings` JSON 必须保持旧 UI 和新 UI 可读。
- 订阅服务读取旧模型生成 URI/JSON/Clash/Mihomo/WireGuard 输出。
- `proxy_inbounds`、`proxy_clients`、`egress_*` 当前只能作为未来设计或文档概念，不是活跃写路径。

## 业务功能分区

- 面板与安全：登录、session、CSRF、CSP、设置、备份恢复、日志下载。
- Xray 运维：运行状态、启动/停止/重启、版本安装、配置模板、流量统计。
- 入站管理：VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard、客户端管理、批量操作。
- 订阅输出：URI/Base64、Xray JSON、Clash/Mihomo、WireGuard、diagnose。
- Gateway Egress MVP：生成 Xray-compatible SOCKS5 inbound 与 Gateway CSV manifest。
- 多核心入口：只读 `default-xray` 和实验 `sing-box`，不替代 Xray 主路径。
- 发布部署：Linux 二进制、GHCR 镜像、install/update/x-ui 脚本、Docker。

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
python scripts/secret_scan.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```
