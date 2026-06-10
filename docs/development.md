# 开发者贡献指南

> **目标读者**：贡献者 / 维护者 / 自动化 Agent
> **适用版本**：`v3.3.0`
> **事实来源**：`go.mod`、`.env.example`、`frontend/package.json`、`.github/workflows/*`、`.codex/project.toml`
> **相关文档**：[系统架构设计](architecture.md) | [核心模块解析](modules.md) | [API 接口说明](api.md) | [部署指南](deployment.md)

---

## 1. 开发原则

当前项目不是单一 Go 仓库，而是混合栈仓库：

- Go/Gin/GORM/SQLite 后端。
- Vue 3/Vite/TypeScript 新 UI。
- Legacy HTML/JS UI。
- 独立订阅服务。
- Xray 进程、外部 sing-box 实验适配器、Geo 资源、Telegram Bot、LDAP、WARP/Nord 集成。

变更前必须先判断任务域。不要把 UI 迁移、CoreManager、多内核、订阅、发布脚本和数据库模型混成一次重构。

当前硬边界：

- `database/model.Inbound` 仍是 active Xray 写模型。
- 不创建 `proxy_inbounds` / `proxy_clients` 活跃写路径。
- legacy Xray 生命周期不通过 CoreManager 接管。
- `/panel/legacy/*` 保留为回退入口。
- 日志、配置、订阅和外部内容不得使用 `v-html` / `innerHTML` 渲染。

---

## 2. 环境要求

### 2.1 Go

`go.mod` 声明：

```text
go 1.26.4
```

本地构建需要 CGO，因为项目使用 SQLite。

常用命令：

```bash
go mod download
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

Linux 发布包由 GitHub Actions 使用交叉编译和对应交叉编译器构建；本地开发优先使用当前平台的正常 Go 工具链。

### 2.2 Node / Frontend

新 UI 在 `frontend/`，脚本来自 `frontend/package.json`：

| 命令 | 作用 |
|---|---|
| `npm run dev` | Vite dev server |
| `npm run build` | `vue-tsc -b && vite build` |
| `npm run preview` | 预览构建产物 |
| `npm run test` | 运行 TS 单测和 legacy JS 单测 |
| `npm run typecheck` | `vue-tsc -b --noEmit` |
| `npm run lint` | ESLint |
| `npm run format` | Prettier check |
| `npm run format:write` | Prettier write |

`npm run test` 实际执行：

```bash
cd .. && node --test --experimental-strip-types frontend/tests/*.test.ts web/assets/js/model/*.test.js web/assets/js/util/*.test.js
```

### 2.3 环境变量

`.env.example` 当前只包含：

```text
XUI_DEBUG=false
XUI_LOG_LEVEL=info
XUI_DB_FOLDER=/etc/x-ui
XUI_LOG_FOLDER=/var/log/x-ui
XUI_BIN_FOLDER=bin
```

本地开发可创建 `.env`：

```bash
XUI_DEBUG=true
XUI_LOG_LEVEL=debug
XUI_DB_FOLDER=x-ui
XUI_LOG_FOLDER=x-ui
XUI_BIN_FOLDER=bin
```

实验 sing-box Core API 可额外设置：

```bash
SUPERXRAY_SING_BOX_BINARY=bin/sing-box
SUPERXRAY_SING_BOX_CONFIG=bin/sing-box-config.json
SUPERXRAY_SING_BOX_LOG_FOLDER=x-ui
```

未设置时，sing-box 默认读取 `config.GetBinFolderPath()` 下的二进制和配置，并使用 `config.GetLogFolder()`。

---

## 3. 本地启动

### 3.1 后端直跑

```bash
go run main.go
```

首次数据库为空时会创建默认用户 `admin/admin`。生产部署必须立即修改默认账户、端口、base path 和 HTTPS 设置；一键安装脚本通常会生成随机安全值。

### 3.2 前端开发

新 UI 源码位于 `frontend/src`，构建产物嵌入到 `web/ui`。常规开发有两种路径：

1. 改 Go/API/legacy UI：直接跑后端即可。
2. 改 Vue 3 新 UI：在 `frontend/` 运行 `npm run dev` 或 `npm run build`，再由 Go 托管构建产物。

如果改动影响后端注入 runtime config、base path、CSP 或静态资源缓存，需要同时验证 Go 托管路径：

```bash
go test ./web ./web/locale
go build -o bin/SuperXray.exe ./main.go
```

---

## 4. 目录拓扑

```text
SuperXray-gui/
├── main.go                         # CLI 与服务入口
├── go.mod / go.sum                 # Go module 与依赖锁
├── .env.example                    # 环境变量示例
├── Dockerfile / docker-compose.yml # 容器构建与编排
├── install.sh / update.sh          # 安装与更新脚本
├── config/                         # 版本、名称、路径与日志级别
├── core/                           # Core 类型、Manager、sing-box 实验适配器
├── database/                       # SQLite 初始化与 GORM 模型
├── frontend/                       # Vue 3/Vite/TypeScript 新 UI
│   ├── package.json
│   ├── src/
│   │   ├── api/                    # API SDK、request、WebSocket
│   │   ├── assets/                 # 静态资源（logo、字体等）
│   │   ├── components/
│   │   ├── i18n/                   # DOM 翻译器与国际化消息
│   │   ├── layouts/
│   │   ├── router/
│   │   ├── schemas/                # 协议注册表
│   │   ├── stores/                 # Pinia stores
│   │   ├── styles/                 # 全局样式
│   │   ├── types/
│   │   ├── utils/                  # Xray/Inbound 兼容工具
│   │   └── views/
│   └── tests/
├── logger/                         # 日志系统
├── sub/                            # 独立订阅服务
├── util/                           # crypto/ldap/path/sys/common 等工具
├── web/                            # Web 面板主体
│   ├── controller/                 # Index/API/Inbounds/Server/Xray/Settings/Core/Geo
│   ├── service/                    # 业务服务（inbound/xray/server/setting/core/tgbot/nord/warp/custom_geo）
│   ├── middleware/                 # 安全头、CSRF、域名校验、重定向
│   ├── websocket/                  # Hub 与 notifier
│   ├── html/                       # Legacy Go templates
│   ├── assets/                     # Legacy 静态资源与 JS 测试
│   ├── ui/                         # 新 UI build 输出
│   ├── translation/                # go-i18n TOML 翻译文件
│   ├── locale/                     # 国际化中间件
│   ├── session/                    # Cookie Session 管理
│   ├── job/                        # Cron 后台任务
│   ├── network/                    # 自动 HTTPS 监听与连接
│   ├── entity/                     # API 响应实体
│   ├── global/                     # 全局变量与接口
│   ├── web.go                      # Gin engine、中间件、路由注册
│   ├── ui.go                       # 新 UI 托管与 runtime config 注入
│   ├── cron_test.go
│   ├── server_security_test.go
│   ├── sidebar_component_test.go
│   ├── ui_test.go
│   └── service/config.json         # Xray 默认模板
├── xray/                           # Xray API 和进程集成
├── docs/                           # 技术文档
├── plans/                          # 路线图、治理和架构规划
│   ├── 00-governance/
│   ├── 01-strategy/
│   ├── 02-architecture/
│   └── 03-ui-design/
└── .github/workflows/              # release/docker/codeql/cache cleanup
```

文档体系规划入口是 `plans/00-governance/documentation-system-plan.md`。`docs/superpowers/*` 是历史规格、计划和评审材料，除非任务明确要求，不应把它们改写成当前运行手册。

---

## 5. 代码规范

### 5.1 Go

- 使用 `gofmt`，不要引入新的格式化风格。
- Controller 只做参数绑定、鉴权和响应，不承载长业务逻辑。
- Service 复用现有模型和工具函数，不绕过 `model.Inbound` 兼容层。
- 生成物和 runtime 文件不要手工打长期补丁，应回源到定义或生成入口。
- 新增安全敏感 API 必须有认证、CSRF、输入校验和测试。

### 5.2 Vue 3 / TypeScript

- API 路径集中放在 `frontend/src/api/endpoints.ts`。
- 通过 `request.ts` 发请求，让 CSRF、Cookie、错误处理和登录跳转统一。
- 页面状态优先使用 Pinia store 或局部 reactive/ref，不在组件里散落全局变量。
- 不使用 `v-html` 渲染日志、配置、订阅或外部内容。
- Xray JSON 结构化编辑应复用 `frontend/src/utils/xrayCompat.ts` / `xrayProtocolTools.ts`。
- Inbound 表单和分享链接应复用 `frontend/src/utils/inboundCompat.ts` 与协议注册表。

### 5.3 文档

- 架构、API、模块职责必须以源码为准。
- 变更 API 参数、响应体、环境变量、脚本或发布资产时，同步更新 `docs/`。
- 历史计划文档保留历史语境；当前运行指南放在根级 `docs/*.md`。

---

## 6. 测试与验证

### 6.1 最小后端验证

按改动范围选择最小相关集：

```bash
go test ./web/locale ./web
go test ./web/controller ./web/middleware ./web/service
go test ./database/model ./sub ./xray
```

大范围后端或发布前执行：

```bash
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

### 6.2 前端验证

改动 `frontend/src`、`frontend/tests` 或 legacy JS 工具时执行：

```bash
cd frontend
npm run typecheck
npm run lint
npm run test
npm run build
```

### 6.3 安全相关验证

涉及 CSRF、CSP、日志渲染、上传、下载、路径、URL 或凭据时，至少补/跑对应测试：

- `web/middleware/security_test.go`
- `web/server_security_test.go`
- `web/controller/server_security_test.go`
- `web/service/server_security_test.go`
- `web/service/custom_geo_test.go`
- `web/service/warp_security_test.go`
- 前端日志/配置/订阅渲染相关测试

### 6.4 常见测试文件

| 区域 | 示例 |
|---|---|
| 模型 | `database/model/model_test.go`、`protocol_test.go` |
| Web | `web/cron_test.go`、`web/sidebar_component_test.go`、`web/server_security_test.go` |
| 中间件 | `web/middleware/security_test.go` |
| 控制器 | `web/controller/server_security_test.go` |
| 服务 | `web/service/*_test.go` |
| 任务 | `web/job/*_test.go` |
| 订阅 | `sub/*_test.go` |
| Xray | `xray/*_test.go` |
| 前端 | `frontend/tests/*.test.ts` |
| legacy JS | `web/assets/js/model/*.test.js`、`web/assets/js/util/*.test.js` |

### 6.5 运行态系统任务的证据优先级

当任务涉及服务器部署、OpenWrt/Passwall、代理分流、订阅出口、AI 平台接口或线上“可访问但功能异常”问题时，代码阅读不能替代运行态取证。建议遵循以下顺序：

1. **客户端视角**：确认当前网段、默认网关、实际出口与目标接口是否可访问。
2. **中间设备视角**：确认 ACL、`iptables -vnL` 计数、`/tmp/etc/passwall/var`、Passwall/Xray 运行进程。
3. **服务器视角**：确认容器/进程、端口监听、真实出口和最近日志。
4. **静态配置**：最后再核对 `uci`、模板 JSON、部署脚本和文档预期。

网络类任务至少保留双视角证据（客户端 + 中间设备，或中间设备 + 服务器）再下结论。不要在未确认实际出口前评价节点优劣，也不要把显式 SOCKS 与透明代理/TPROXY 路径混为一谈。

---

## 7. UI 与 Core 阶段门禁

当前允许的工作：

- 新 UI 与 legacy UI 兼容维护。
- Xray parity、Inbounds、Settings、Logs、Dashboard、订阅导出体验修复。
- CSP/CSRF/XSS/上传下载安全收口。
- `/panel/api/cores` 最小实例视图和 experimental sing-box 生命周期。
- Xray JSON 模板层的 Gateway Egress MVP 和 AI residential routing。

当前禁止的工作：

- 删除 legacy UI fallback。
- 把 legacy Xray lifecycle 改由 CoreManager 控制。
- 新建 active `proxy_inbounds` / `proxy_clients` 写路径。
- 把新 UI 写成旧 UI 无法读取的数据格式。
- 让 sing-box 成为生产默认核心。

---

## 8. 提交流程

### 8.1 Commit message

推荐中文说明或英文 conventional commit 均可，保持简洁：

```text
docs(api): 同步 CSRF 与 WebSocket API 文档
fix(inbound): 修复客户端复制的表单字段处理
feat(ui): 增加入站订阅导出入口
test(security): 添加数据库导入文件名校验回归
```

### 8.2 PR 描述

PR 至少包含：

- 背景/问题
- 方案概述
- 变更点
- 影响范围
- 验证方式与结果
- 风险与回滚

---

## 9. 发布流程

当前 Release workflow：`.github/workflows/release.yml`。

触发：

- `main` 相关路径变更。
- tag push。

关键步骤：

1. `release_gate.py --ci --metadata-only` 校验版本、CHANGELOG 和 release metadata。
2. 设置 Go。
3. 针对 Linux `amd64` / `arm64` 构建 `xui-release`。
4. 下载 Xray release `v26.4.25`。
5. 下载 Geo 数据文件。
6. 打包：
   - `x-ui-linux-amd64.tar.gz`
   - `x-ui-linux-arm64.tar.gz`
7. tag push 时上传 GitHub Release。

发布前推荐：

```bash
git status --short
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
cd frontend
npm run typecheck
npm run lint
npm run test
npm run build
cd ..
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```

版本号必须与 `config/version` 和 tag `vX.Y.Z` 保持一致。

---

## 10. 回滚思路

| 变更类型 | 首选回滚 |
|---|---|
| 新 UI 页面问题 | 使用 `/panel/legacy/*` 回退 |
| Xray JSON 模板问题 | 恢复数据库备份或旧模板，再重启 Xray |
| Inbound 写入问题 | 保持 `model.Inbound` 兼容，必要时用 legacy UI 修正 |
| sing-box 实验问题 | 停止 `experimental-sing-box`，不影响 `default-xray` |
| 数据库导入失败 | 服务层应保持原 DB 不被破坏 |
| Release 资产问题 | 检查 tag、`config/version`、CHANGELOG、资产名和 `install.sh` 下载路径 |
