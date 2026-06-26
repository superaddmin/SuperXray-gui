# 3x-ui 后端架构重基线与 Xray-core v26.3.27 同步实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **基线来源：** `MHSanaei/3x-ui` 最新 `main` 分支，commit `e4b881e58a8ac99804c8a4261df09aa7366b62f2`（2026-06-25）。
>
> **强制约束：** 本计划必须把运行时 Xray-core release 升级到 `v26.3.27`，并在文档中明确“release tag”与“Go module 版本”是两条不同的链路；不要把 `v26.3.27` 误写成 Go module 依赖版本。

**Goal:** 基于 upstream 最新 main 分支，对后端架构做一次可执行的重基线梳理，明确 `api/config/database/web/xray/subscription/service/controller` 的职责边界、数据流、ORM 映射、认证权限、流量与订阅链路、路由与中间件、热重载、静态资源嵌入和发布策略；同时把仓库运行时 Xray-core release、安装脚本、发布工作流和文档统一到 `v26.3.27`。

**Architecture:** 采用“先锁版本与契约，再拆分模块边界，随后迁移业务与数据层，最后收口安全、可观测性和发布流水线”的顺序。当前仓库仍以顶层 `web/`、`sub/`、`database/`、`core/`、`main.go` 为主要入口，因此本计划的目标不是一次性硬搬目录，而是先形成 `internal/` 目标结构，再用薄适配层逐步切换。

**Tech Stack:** Go 1.26.4、Gin、GORM、SQLite、robfig/cron、WebSocket、Go `embed`、Xray-core release `v26.3.27`、GitHub Actions、Docker、Markdown。

**当前执行状态（2026-06-26）：** Phase 0 已完成运行时 Xray-core `v26.3.27` 发布链路对齐；Phase 1 已落地架构契约文档 [docs/backend-internal-rebaseline-contract.md](../../backend-internal-rebaseline-contract.md)，用于约束后续 `internal/` 包边界、API 契约、数据流和迁移顺序；Phase 2 已落地迁移元数据表、baseline 记录和首批 GORM repository 边界，SettingService/UserService 及 InboundService 读路径、第二批查询路径、第三批邮箱唯一性读取路径已收口，不改变现有业务表。

---

## 1. 现有架构剖析与关键设计决策

### 1.1 上游 baseline 的模块切分

上游 `internal/` 的语义边界可以归纳为：

| 模块 | 主要职责 | 边界约束 |
|---|---|---|
| `internal/api` | 请求/响应 DTO、错误码、响应包、版本化契约 | 不碰数据库和进程管理 |
| `internal/config` | 默认值、环境变量、启动参数、运行时配置快照 | 不承载业务规则 |
| `internal/database` | 连接、迁移、Repository、模型映射 | 不直接做 HTTP 绑定 |
| `internal/controller` | 路由处理、参数绑定、鉴权门禁、输出响应 | 只做运输层编排 |
| `internal/service` | 用户、节点、流量、订阅、通知、审计等业务逻辑 | 不直接操作 Gin 上下文 |
| `internal/web` | HTTP Server、middleware、websocket、cron、静态资源 | 不写业务规则 |
| `internal/xray` | Xray 进程管理、gRPC API、stats、template 渲染、热重载触发 | 不直接侵入 controller |
| `internal/subscription` | 订阅格式生成、host 重写、格式能力矩阵、诊断输出 | 只读路径，不做写库 |

### 1.2 当前仓库的现实映射

| 当前目录 | 对应能力 | 计划中的演进方向 |
|---|---|---|
| `main.go` | 启动、信号处理、服务装配 | 保持入口薄化，只保留装配与生命周期 |
| `web/` | server、controller、service、middleware、job、session、websocket、静态 UI | 拆成 `internal/web`、`internal/controller`、`internal/service` |
| `sub/` | 订阅生成、输出矩阵、诊断、订阅服务 | 收敛为 `internal/subscription` |
| `database/` | GORM 模型与迁移 | 收敛为 `internal/database` + `repository` |
| `core/` | Core manager / sing-box 适配 | 作为兼容层保留，后续再决定是否纳入 `internal/xray` |
| `web/ui/` | 前端静态资源嵌入产物 | 保持 go:embed 发布路径，不再手工维护生成文件 |

### 1.3 关键设计决策

1. Xray-core 仍按“外部子进程 + gRPC API”管理，不把核心协议逻辑内嵌进面板进程。
2. 数据库仍以 SQLite + GORM 为主，先做版本化迁移和仓储分层，不在本计划里切换数据库引擎。
3. 浏览器端继续保留 Session + CSRF 路线，机器访问侧补充 JWT/API Token/作用域控制。
4. 订阅服务保持只读输出，不在订阅请求里顺手写库或触发高风险副作用。
5. 运行时 release tag 与 Go module 版本分离管理。`v26.3.27` 只用于下载和发布包，不等价于 `go.mod` 里的 module 版本。

### 1.4 核心数据流

#### 登录与鉴权

`Browser -> middleware -> session / CSRF -> controller -> service -> database`

#### 入站保存

`UI -> API DTO -> controller -> service -> repository -> template render -> xray reload/restart`

#### 流量与计费

`xray stats / access log -> collector job -> traffic ledger -> quota / expiry evaluator -> websocket / notify`

#### 订阅分发

`request -> subscription controller/service -> formatter -> raw/json/clash output -> host rewrite -> response`

#### 配置热重载

`settings update -> config snapshot diff -> apply live fields -> restart-needed fields -> xray reload or process restart`

### 1.5 数据库模型与 ORM 映射

建议把 ORM 映射整理成“主实体 + 聚合根 + 只读视图”三层：

| 领域 | 目标实体 | 映射策略 |
|---|---|---|
| 用户与权限 | `User`、`ApiToken`、`Role`（如引入） | 明确区分 UI session 与机器 token |
| 入站与客户端 | `Inbound`、`Client`、`ClientRecord`、`ClientGroup` | 继续允许 JSON 列承载 Xray 配置片段，但对外暴露 typed DTO |
| 流量与账本 | `ClientTraffic`、`OutboundTraffics`、`NodeClientTraffic`、`ClientGlobalTraffic` | 统一为“账本 + 聚合报表”两套读模型 |
| 节点与订阅 | `Node`、`OutboundSubscription`、`ClientExternalLink` | 订阅输出只读，不与编辑表单混写 |
| 系统设置 | `Setting`、`HistoryOfSeeders`、`InboundClientIps` | 保持键值对兼容，但迁移到版本化设置 snapshot |

## 2. 目标系统边界与技术选型

### 2.1 明确纳入的范围

- 面板认证、权限、会话、API token、JWT。
- 用户管理、节点管理、入站管理、流量统计、订阅服务、通知告警。
- Xray 进程生命周期、配置模板、gRPC 交互、热重载、健康检查。
- 面板 API 路由注册、middleware 管线、静态资源嵌入与发布。
- 数据库版本化迁移、备份、回滚、兼容读取。

### 2.2 明确排除的范围

- 不在本计划中替换 Xray-core 协议实现。
- 不把本地面板改造成新的支付/清结算系统；计费仅指配额、流量、到期和告警。
- 不把 `v26.3.27` 当成 Go module 版本去强行改 import path。
- 不在第一阶段推翻现有 UI 技术栈。

### 2.3 技术选型论证

- 继续用 Gin：路由、中间件、绑定和 JSON 响应与当前代码契合度最高。
- 继续用 GORM + SQLite：现有数据模型大量依赖 JSON 列和轻量迁移，短期最稳。
- 继续用 Go `embed`：静态 UI 和 release 包最容易保持一致。
- Xray 仍以进程外部化方式接入：更利于版本替换、隔离崩溃和独立重启。
- 安全认证采用“Session + CSRF + JWT/API Token”并行：浏览器友好，自动化访问也有短令牌路径。

## 3. `internal/` 目标包结构方案

### 3.1 目标目录

```text
internal/
  api/
  config/
  controller/
  database/
  service/
  subscription/
  web/
  xray/
```

### 3.2 逐模块职责边界

#### `internal/api`

- 定义请求/响应 DTO、错误码、分页结构、统一响应包。
- 定义 API 版本号、兼容策略和字段废弃策略。
- 不允许引用 GORM model 和 Gin context。

#### `internal/config`

- 负责默认值、环境变量、配置快照、热重载标记。
- 区分 `restart-required`、`xray-reload-required`、`live-update` 三类配置。

#### `internal/database`

- 管理 DB 初始化、迁移、事务、Repository。
- GORM model 只描述持久化结构，不写路由逻辑。

#### `internal/controller`

- 负责路由注册、参数绑定、认证门禁、错误映射、响应输出。
- 只做输入输出编排，不写核心业务。

#### `internal/service`

- 承载用户、节点、流量、订阅、通知、审计等业务规则。
- 通过接口依赖 repository、xray client 和 notifier。

#### `internal/subscription`

- 生成 Base64、URI、JSON、Clash/Mihomo、诊断输出。
- 支持 host rewrite、format capability matrix、client filtering。

#### `internal/web`

- 装配 Gin Engine、middleware、websocket、cron、静态资源、健康检查。
- 只负责 HTTP server 生命周期，不持有复杂业务规则。

#### `internal/xray`

- 封装 binary path、config path、log path、grpc client、stats client、restart logic。
- 负责模板渲染、配置验证、热重载和进程守护。

### 3.3 迁移原则

1. 新功能优先进入 `internal/*`，旧顶层包只保留兼容入口。
2. 先迁移动作少、依赖低的模块，再迁移高耦合模块。
3. 先做“写路径”抽象，再做“读路径”优化。
4. 每次迁移必须有回归测试覆盖原路径和新路径。

## 4. 核心业务与接口设计路线

### 4.1 用户管理

- 目标接口：注册/登录/登出/改密/2FA/LDAP 同步/角色变更/API token。
- 认证顺序：Session 或 JWT 解析 -> 用户状态检查 -> 权限检查 -> CSRF（仅浏览器态写请求）。
- 验收：密码校验、2FA、过期会话、禁用用户、token 吊销都能被测试覆盖。

### 4.2 节点与入站管理

- 目标接口：节点列表、入站增删改查、克隆、启停、标签迁移、客户端导出。
- 入站写入流程：DTO 校验 -> 模型归一化 -> 持久化 -> 触发 xray template render -> 按需 reload/restart。
- 验收：JSON 结构合法、非法协议组合被拒绝、重启策略可预测。

### 4.3 流量统计与计费

- 流量来源统一为 Xray stats、access log、客户端 IP 事件和节点级聚合。
- 计费定义为“流量记账 + 套餐配额 + 到期时间 + 重置策略”，不直接混入支付系统。
- 验收：断点续算、周期重置、超额告警、异常回退均有测试。

### 4.4 订阅服务

- 输出格式：URI/Base64、JSON、Clash/Mihomo、诊断 JSON。
- 订阅服务必须支持 host 重写、用户过滤、协议能力矩阵、不可达节点跳过。
- 验收：同一客户端在不同格式下输出一致，敏感字段不泄漏。

### 4.5 通知与告警

- 统一接口：`Notify(event)`、`Warn(message)`、`Critical(message)`。
- 适配 Telegram、邮件、Webhook、日志告警。
- 验收：失败重试、限流、脱敏和幂等策略明确。

## 5. Xray-core v26.3.27 集成层与模板引擎抽象

### 5.1 版本更新要求

- `DockerInit.sh`、`.github/workflows/release.yml`、README 与部署文档统一下载 `v26.3.27` release。
- release 资产、脚本注释和用户文档都要同步更新，不允许继续出现 `v26.4.25` 或 `v26.6.22` 的运行时链路。
- 如果未来要切换 Go module major path 到 `/v26`，单独开技术课题，不与本计划合并。

### 5.2 进程管理

- 统一管理 binary、config、log、stats、restart reason。
- 提供 `Start/Stop/Restart/Reload/Validate/Health` 六类动作。
- 对外暴露只读状态和最近一次失败原因，不泄漏敏感凭据。

### 5.3 模板引擎

- 入站/出站/路由/DNS/observatory/fakeDNS/balancer/reverse 均走模板渲染。
- 模板输入必须是 typed struct，不允许 controller 直接拼接 JSON 字符串。
- 模板输出先校验 JSON schema，再写入运行时配置。

### 5.4 gRPC 交互

- 通过本地 gRPC client 与 Xray API 交互。
- 所有变更都要有 timeout、context cancel、错误映射和回滚路径。
- 统计类读取与控制类写入分离，避免一个接口同时承担太多职责。

## 6. API 契约规范

### 6.1 请求/响应模型

- 所有面板 API 统一使用版本化 DTO。
- 响应包统一包含 `success/data/error/code/requestId` 一类字段。
- 对外错误码分层：`4xx` 业务输入、`5xx` 系统错误、`6xx` 迁移/依赖/外部核心错误。

### 6.2 版本化管理

- 新增接口走 `v1`，破坏性变更走 `v2` 或独立命名空间。
- 废弃字段先灰度双写，再标记 deprecated，最后移除。
- OpenAPI、前端 SDK、测试数据三者必须同步。

### 6.3 中间件管线

- 推荐顺序：`request-id -> logger -> recovery -> security headers -> session -> auth -> rate limit -> csrf -> domain validator -> handler`。
- 公共只读接口与登录后接口分开挂载，订阅/健康检查等低风险路径不要被高开销中间件误伤。

## 7. 安全加固方案

- TLS：默认启用，证书加载、续期和回滚可自动化。
- JWT：用于机器访问或短期授权，浏览器仍以 Session 为主。
- 速率限制：按 IP、用户、token、路由组分级。
- 输入校验：所有 JSON、路径、Host、URL、文件名都必须白名单校验。
- SSRF/路径遍历防护：外部 URL、下载源和文件写入都走 allowlist。
- CSRF：状态变更请求强制校验。
- 日志脱敏：UUID、token、密码、Cookie、证书路径不得裸写入日志。

## 8. 可观测性方案

- 结构化日志：统一字段 `requestId/userId/ip/path/duration/error`。
- 指标暴露：请求耗时、成功率、Xray 状态、流量、重启次数、订阅命中率。
- 健康检查：区分 `live`、`ready`、`xray healthy`、`db healthy`。
- 告警：将 xray 进程退出、DB 迁移失败、订阅异常、流量采集失败纳入告警规则。

## 9. 测试与 CI/CD 方案

### 9.1 单元测试

- DTO 校验。
- repository / service 纯逻辑。
- xray template renderer。
- subscription formatter。
- auth / permission / token logic。

### 9.2 集成测试

- DB migration from empty DB and old DB。
- Xray mock / gRPC mock。
- API 路由与 middleware 链路。
- 流量采集、订阅输出、配置热重载。

### 9.3 E2E

- 登录、创建入站、修改配置、重启 Xray、查看流量、导出订阅、下载备份。
- UI 静态资源嵌入后，验证生产包和本地开发包的行为一致。

### 9.4 CI/CD

- 格式化、单测、集成测、静态检查、release gate、secret scan、OpenAPI stale gate。
- release 包必须显式绑定 `v26.3.27` Xray release。

## 10. 容器化与部署拓扑

- 单机部署：Web 面板 + Xray child process + SQLite。
- 容器部署：面板容器和数据卷分离，Xray release 与 Geo 数据在构建阶段装配。
- 多实例部署：面板可横向扩展，但 DB 和 Xray 控制权必须避免并发写冲突。
- 升级策略：先备份 DB 与配置，再更新 release 包，再滚动重启。

## 11. 分阶段交付里程碑

### Phase 0: 版本基线与发布链路对齐

**前置依赖：** 已确认上游 release `v26.3.27` 可用，且仓库内所有下载入口已盘点完毕。

- [x] 更新 `DockerInit.sh`、`.github/workflows/release.yml`、`README.zh_CN.md`、`docs/architecture.md`、`docs/development.md` 的 Xray 版本引用。
- [x] 在架构文档里写明“运行时 release tag”与“Go module 版本”分离。
- [x] 保留一条清晰的回滚策略：如果 `v26.3.27` 下载/启动失败，先回退到上一稳定 release。

**验收标准：**

- `rg -n "v26\\.4\\.25"` 只剩历史说明，不再出现在运行时下载链路。
- release workflow 与安装脚本都指向 `v26.3.27`。
- 文档不再把 release tag 误写成 Go module 版本。

**执行证据（2026-06-26）：**

- `DockerInit.sh`、`.github/workflows/release.yml`、`README.zh_CN.md`、`docs/architecture.md`、`docs/development.md` 已指向 Xray-core release `v26.3.27`。
- 上游 `XTLS/Xray-core` tag `v26.3.27` 已确认存在，Linux `amd64` / `arm64` release 资产可访问。
- `go.mod` 保持合法 Go module 版本链路，未把 release tag `v26.3.27` 误写为 module 依赖。

### Phase 1: 架构与契约盘点

**前置依赖：** Phase 0 完成。

- [x] 形成 `internal/` 目标目录图与职责说明。
- [x] 定义 API 响应包、错误码、版本策略。
- [x] 画清数据流：登录、入站、流量、订阅、热重载。

**验收标准：**

- 每个模块都有唯一 owner 和边界。
- 新增接口可以明确落到 controller/service/repository 的哪一层。

**执行证据（2026-06-26）：**

- 新增 [docs/backend-internal-rebaseline-contract.md](../../backend-internal-rebaseline-contract.md)，固定当前顶层包到目标 `internal/` 包的迁移映射。
- 文档明确了 `internal/api`、`internal/config`、`internal/database`、`internal/controller`、`internal/service`、`internal/subscription`、`internal/web`、`internal/xray` 的 owner、输入输出和禁止事项。
- 文档定义了 legacy API 兼容响应、新 `/api/v1` 响应包、错误码分层、版本策略、核心数据流、数据库模型映射和 Xray-core `v26.3.27` 集成约束。

### Phase 2: 数据层与迁移框架

**前置依赖：** Phase 1 完成。

- [x] 引入版本化迁移表、迁移记录和回滚记录。
- [x] 建立 GORM model 与业务 DTO 转换前置边界（repository interface）。
- [x] 启动首批 service 直连 GORM 收口（SettingService/UserService -> repository）。
- [ ] 继续分批迁移剩余 service/controller 的 DTO adapter 与直连 GORM 调用。
  - [x] InboundService 首批读路径（`GetInbounds` / `GetAllInbounds` / `GetInbound`）已迁到 `InboundRepository`。
  - [x] InboundService 第二批查询路径（`GetInboundOptions` / `GetInboundsByTrafficReset` / `checkPortExist`）已迁到 `InboundRepository`。
  - [x] InboundService 第三批邮箱唯一性读取路径（`getAllEmails` / `checkEmailExistForInbound` / `checkEmailsExistForClients`）已迁到 `InboundRepository`。
- [x] 启动旧表兼容读取验证，避免一刀切改表。

**验收标准：**

- 空库和旧库都能启动。
- 迁移可重复执行且幂等。

**执行证据（2026-06-26）：**

- 新增 `model.SchemaMigration` 和 `model.MigrationEvent`，由 `database.InitDB` 自动创建 `schema_migrations` 与 `migration_events`。
- `database.InitDB` 写入 baseline 记录 `202606260001 / baseline-auto-migrate`，用于标记当前 AutoMigrate 基线。
- 新增 `database/db_migration_test.go`，覆盖空库初始化后的迁移元数据表与 baseline 记录。
- 新增 `database/repository.go`，定义 `Repositories`、`UserRepository`、`SettingRepository`、`InboundRepository` 及首批 GORM 实现，继续读取 `users`、`settings`、`inbounds`、`client_traffics` 当前表和模型；`UserRepository` 已覆盖按用户名读取、保存和凭证更新，`InboundRepository` 已覆盖列表、单条、options、traffic reset、端口冲突与客户端邮箱查询。
- 新增 `database/repository_test.go`，覆盖默认用户读取/凭证更新、设置新增/更新、入站单条读取、按用户/全量读取、options 轻量记录、traffic reset 过滤、端口冲突判断、客户端邮箱提取以及列表路径 `ClientStats` preload；单条 `Inbounds.Get` 保持旧 `GetInbound` 不预加载统计的语义，确保 repository 边界不绕开旧模型关系。
- `web/service/setting.go` 已通过 `database.SettingRepository` 访问设置数据；`SettingService{}` 零值仍自动回落到全局 DB 仓储，避免一次性改动所有初始化入口。
- `web/service/setting_test.go` 新增 fake repository 用例，验证 `getString/setString/GetAllSetting/ResetSettings` 走 repository 边界。
- `web/service/user.go` 已通过 `database.UserRepository` 访问用户数据；`UserService{}` 零值保持兼容，`CheckUser`、`UpdateUser`、`UpdateFirstUser` 保留原认证、LDAP 与 2FA 语义。
- 新增 `web/service/user_test.go`，用 fake repository 验证 `GetFirstUser/CheckUser/UpdateUser/UpdateFirstUser` 的用户读写边界。
- `web/service/inbound.go` 已新增 `NewInboundService` 与 `inbounds()` repository accessor，`GetInbounds`、`GetAllInbounds`、`GetInbound` 切到 `database.InboundRepository`；列表读取继续执行 `ClientStats` 的 UUID/SubId 回填，`InboundService{}` 零值仍回落到全局 DB 仓储。
- `web/service/inbound.go` 第二批迁移已将 `GetInboundOptions`、`GetInboundsByTrafficReset`、`checkPortExist` 切到 `database.InboundRepository`；options 派生字段由 repository record adapter 计算，service 只映射现有 API DTO。
- `web/service/inbound.go` 第三批迁移已将 `getAllEmails` 切到 `database.InboundRepository.ListClientEmails`，使 `checkEmailExistForInbound`、`checkEmailsExistForClients` 和复制客户端路径共享同一 repository 邮箱读取边界。
- `web/service/inbound_test.go` 新增 fake inbound repository 用例，验证 InboundService 首批、第二批和第三批读/查询路径调用 repository 边界、按用户筛选、全量读取、单条读取、options 元数据、traffic reset 过滤、端口冲突调用、邮箱唯一性读取以及大小写不敏感的客户端统计回填。
- 当前步骤未改动 `users`、`inbounds`、`settings`、`client_traffics` 等业务表语义，旧写路径继续兼容。

### Phase 3: API / middleware 重构

**前置依赖：** Phase 1、Phase 2 完成。

- [ ] 路由分组重整为 public/auth/admin/api/ws。
- [ ] 中间件管线固化，加入 JWT、速率限制和统一错误映射。
- [ ] 统一请求/响应 DTO 和错误码。

**验收标准：**

- 401/403/404/429/5xx 行为清晰且可测试。
- 前端和脚本调用不会因字段变更而无提示失败。

### Phase 4: Xray 集成层与模板引擎

**前置依赖：** Phase 2、Phase 3 完成。

- [ ] 封装 Xray process manager、gRPC client、template renderer、health checker。
- [ ] 把入站/出站/路由/DNS/observatory 配置统一进模板层。
- [ ] 热重载与重启策略分层处理。

**验收标准：**

- 配置变更可预期地触发 reload/restart。
- 模板输出可校验，失败时有清晰回滚。

### Phase 5: 业务服务层

**前置依赖：** Phase 3、Phase 4 完成。

- [ ] 用户、节点、流量、订阅、通知、审计服务接口定稿。
- [ ] 每个 service 只依赖接口，不直接依赖 Gin。
- [ ] 把统计与告警逻辑收口到服务层。

**验收标准：**

- 业务服务可独立单测。
- controller 不再承载核心业务计算。

### Phase 6: 安全与可观测性

**前置依赖：** Phase 3、Phase 5 完成。

- [ ] TLS、JWT、CSRF、限流、输入校验、日志脱敏全部落地。
- [ ] 结构化日志、指标暴露、健康检查和告警规则补齐。

**验收标准：**

- 安全头与鉴权行为都有回归测试。
- 健康检查和指标能用于发布判断。

### Phase 7: 测试、CI/CD 与容器化

**前置依赖：** Phase 0 至 Phase 6 完成。

- [ ] 补齐单元、集成和 E2E 覆盖。
- [ ] release gate、secret scan、OpenAPI stale gate 进入 CI。
- [ ] 容器镜像与 release 包发布一致化。

**验收标准：**

- 主分支的 CI 结果能直接代表可发布性。
- release 包中包含正确版本的 Xray release。

## 12. 风险缓解清单

| 风险 | 影响 | 缓解 |
|---|---|---|
| Xray release 版本升级失败 | 面板启动后无法拉起代理核心 | 保留上一版本 release 包与回滚脚本 |
| 模块拆分过快 | controller/service 互相引用混乱 | 先建接口，再迁移实现 |
| 迁移改表引发数据丢失 | 历史入站、订阅、流量记录受损 | 版本化迁移 + 备份 + 幂等回滚 |
| 认证体系改动引入登录失败 | 管理面板不可用 | Session 与 JWT 双路径并行，先灰度后切换 |
| 订阅输出兼容性回退 | 客户端导入失败 | 保持旧格式输出，同时新增格式能力测试 |
| 静态资源嵌入与发布不一致 | 线上页面与源码不一致 | CI 构建产物哈希校验 |

## 13. 完成定义

当以下条件都满足时，本计划视为完成：

1. 仓库内所有运行时 Xray release 入口都已对齐 `v26.3.27`。
2. `internal/` 目标边界已经形成，且当前顶层包有明确迁移路径。
3. API、数据库、认证、流量、订阅、热重载和静态资源发布都具备可执行的验收标准。
4. CI/CD 和容器化发布链路能稳定产出可部署制品。
