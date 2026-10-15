# 后端 internal 重基线契约

> **文档性质：** 本文保留后端重基线的目标设计，并标注当前实现；`internal/` 目录图是待迁移目标，不是现存目录。多内核架构路线见 `plans/02-architecture/backend-multi-core-architecture-plan.md`。
>
> **源码核对日期：** 2026-10-15，依据 `database/db.go`、`database/repository.go`、`web/entity/api.go`、`web/controller/v1.go`、`web/service/*` 与相关测试。
>
> **适用范围：** SuperXray-gui 当前 Go 后端，含 `web/`、`sub/`、`database/`、`xray/`、`core/`、`config/`、`main.go`
>
> **版本约束：** 仓库安装/打包默认 Xray-core release tag 为 `v26.10.10`，面板安装器允许该版本及以后符合 `vYY.M.D` 形式的正式版本。release tag 不等同于 `github.com/xtls/xray-core` Go module 版本。

本文是后端重基线的第一份可执行契约。它不要求一次性搬迁目录，而是先固定目标 `internal/` 边界、依赖方向、API 契约和迁移验收口径。后续代码迁移必须以本文为约束，避免在 controller、service、database、xray runtime 之间继续扩大隐式耦合。

## 1. 当前事实基线

| 当前入口 | 当前职责 | 迁移判断 |
|---|---|---|
| `main.go` | CLI、日志初始化、DB 初始化、Web/Sub server 生命周期、信号处理 | 入口应薄化，保留装配和生命周期，不继续承载业务命令细节 |
| `config/` | 本地路径、版本、日志级别、环境变量读取 | 可直接演进为 `internal/config` 的本地配置层 |
| `database/` | SQLite/GORM 初始化、统一迁移事务、原子发布全局 DB、baseline 记录、typed repository、维护租约 | baseline、User/Setting/Inbound repository 与候选迁移已实现；继续收敛事务/runtime 耦合 |
| `database/model/` | `User`、`Inbound`、`Setting`、流量与 Geo 模型 | 当前仍是活跃写模型，不能直接替换 |
| `web/web.go` | Gin engine、中间件、Session、locale、WebSocket、cron、UI、Xray 启停 | 应拆成 `internal/web` 的装配层和 `internal/service` 的业务层 |
| `web/controller/` | 页面/API 控制器、参数绑定、统一响应、鉴权入口 | 应迁入 `internal/controller`，禁止继续下沉业务计算 |
| `web/service/` | 设置、用户、入站、Xray、服务器、Geo、TG Bot 等业务逻辑 | User/Setting/Inbound 已支持 repository 注入；迁入 `internal/service` 尚未完成 |
| `web/job/` | Xray 状态、流量、IP、重置、LDAP、告警等定时任务 | 调度归 `internal/web`，业务动作归 `internal/service` |
| `web/middleware/` | 安全头、CSRF、Host 校验、重定向 | 应归 `internal/web/middleware` 或 `internal/web` |
| `web/ui.go`、`web/ui/` | Vue 静态资源 go:embed 托管和 runtime config 注入 | 保留在 `internal/web` 作为发布装配边界 |
| `sub/` | 独立订阅 HTTP 服务、URI/JSON/Clash 输出、诊断 | 业务生成归 `internal/subscription`，HTTP server 装配归 `internal/web` |
| `xray/` | Xray config、进程、gRPC API、traffic、client traffic | 应归 `internal/xray`，禁止直接依赖 Gin 或 DB |
| `core/` | 多核心抽象和 sing-box 实验适配 | Phase 10 前只读/实验边界保持不变，不接管 legacy Xray 生命周期 |

## 2. 目标 internal 包边界

本节及第 4 节流程图描述迁移目标。当前源码仍使用根级 `web/`、`database/`、`sub/`、`xray/` 等包，不能据此假定已实现以下依赖隔离。

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

| 目标包 | Owner | 输入 | 输出 | 禁止事项 |
|---|---|---|---|---|
| `internal/api` | API 契约 owner | HTTP DTO、OpenAPI、错误码、分页、响应包 | typed request/response、error mapping | 不引用 Gin、GORM model、Xray process |
| `internal/config` | Runtime config owner | 环境变量、本地路径、DB 设置快照 | immutable snapshot、reload diff | 不做业务决策，不写数据库 |
| `internal/database` | Persistence owner | GORM DB、migration、repository | model、repository interface、transaction | 不引用 Gin，不触发 Xray restart |
| `internal/controller` | Transport owner | Gin context、DTO、auth context | HTTP response、WebSocket push trigger | 不直接读写 GORM，不直接拼 Xray JSON |
| `internal/service` | Domain owner | repository、xray client、notifier、clock | 用户、节点、流量、订阅、告警业务动作 | 不依赖 Gin context，不持有 HTTP session |
| `internal/subscription` | Subscription owner | client/inbound read model、format option | URI/Base64、JSON、Clash、diagnostic | 不写库，不改运行时配置 |
| `internal/web` | HTTP/runtime owner | Gin engine、middleware、cron、embed FS | web server、sub server、ws、health | 不写核心业务规则 |
| `internal/xray` | Xray integration owner | config template、binary path、gRPC endpoint | process state、stats、reload/restart result | 不依赖 controller/session，不读取业务表 |

### 2.1 依赖方向

允许方向：

```text
main -> internal/web -> internal/controller -> internal/service
main -> internal/config
internal/service -> internal/database
internal/service -> internal/xray
internal/service -> internal/subscription
internal/web -> internal/api
internal/controller -> internal/api
```

约束：

1. `controller` 只能做参数绑定、认证上下文提取、调用 service、映射响应。
2. `service` 只能依赖接口，不能接收或保存 `*gin.Context`。
3. `database` 只描述持久化和事务，不主动广播 WebSocket，不主动重启 Xray。
4. `xray` 只处理进程、gRPC、模板渲染和 stats，不读取 `settings` 或 `inbounds` 表。
5. `subscription` 是只读生成链路，订阅请求不得产生数据库写入或 Xray restart。
6. `web` 负责 server lifecycle、middleware、cron、embed 和 websocket hub，不放业务规则。

## 3. API 契约

### 3.1 兼容层

现有 `/panel/api/*`、`/panel/setting/*`、`/panel/xray/*` 继续保持 legacy 响应：

```json
{
  "success": true,
  "msg": "",
  "obj": {}
}
```

该结构由 `web/entity.Msg` 和 `web/controller/util.go` 支撑。重构期间不能改变字段名、HTTP 状态行为或前端 unwrap 逻辑。

### 3.2 新契约

当前已实现公开 `GET /api/v1/health/live`、`GET /api/v1/health/ready` 和 `GET /api/v1/metrics`，统一响应来自 `web/entity.APIResponse`。新增接口的目标路径为 `/api/v1/*`。当前成功响应：

```json
{
  "success": true,
  "requestId": "req_...",
  "data": {}
}
```

错误响应：

```json
{
  "success": false,
  "requestId": "req_...",
  "error": {
    "code": "NOT_FOUND",
    "message": "route not found"
  }
}
```

`data/error/requestId` 使用 `omitempty`；当前没有顶层 `code`。请求 ID 由 engine 中间件提供，错误的 `details` 仅在有实际数据时输出。机器契约见 [Versioned API OpenAPI](openapi/api-v1.yaml)。健康接口返回只读快照，Xray 停止或出错不改变 HTTP 200；这不等于完整依赖就绪检查。

### 3.3 当前错误码与后续分层目标

当前枚举仅有 `UNAUTHORIZED`、`FORBIDDEN`、`NOT_FOUND`、`RATE_LIMITED`、`VALIDATION_ERROR`、`INTERNAL_ERROR`。V1 路由未匹配返回 `NOT_FOUND/404`，每客户端每分钟默认 120 次固定窗口限流返回 `RATE_LIMITED/429` 与 `Retry-After`；其他枚举为已定义的可复用错误类型，不能视为已经存在对应业务 API。

以下领域错误分层是后续设计候选，尚未由 `APIErrorCode` 实现；落地时须同时调整代码、OpenAPI 和调用方：

| 目标范围 | 候选示例 | 目标 HTTP 状态 | 说明 |
|---|---|---|---|
| `AUTH_*` | `AUTH_UNAUTHENTICATED`、`AUTH_FORBIDDEN`、`AUTH_CSRF_FAILED` | 401/403/404 | 登录、权限、CSRF、隐藏 API |
| `VALIDATION_*` | `VALIDATION_FAILED`、`VALIDATION_UNSUPPORTED_PROTOCOL` | 400/422 | DTO 和业务输入校验 |
| `RESOURCE_*` | `RESOURCE_NOT_FOUND`、`RESOURCE_CONFLICT` | 404/409 | 主键、tag、email、订阅 id 冲突 |
| `RATE_*` | `RATE_LIMITED` | 429 | IP、用户、token、路由组限流 |
| `DB_*` | `DB_UNAVAILABLE`、`DB_MIGRATION_FAILED` | 500 | 数据库连接、迁移、事务失败 |
| `XRAY_*` | `XRAY_PROCESS_FAILED`、`XRAY_GRPC_FAILED`、`XRAY_TEMPLATE_INVALID` | 502/503/500 | Xray 进程、gRPC、模板、reload/restart |
| `SUB_*` | `SUB_NOT_FOUND`、`SUB_FORMAT_DISABLED`、`SUB_UNSUPPORTED_PROTOCOL` | 400/404 | 订阅输出与格式能力矩阵 |
| `CONFIG_*` | `CONFIG_RELOAD_FAILED`、`CONFIG_RESTART_REQUIRED` | 409/500 | 配置热重载与重启判定 |

### 3.4 版本策略

1. Legacy API 以现有路径保留，OpenAPI 继续记录事实契约。
2. 新 API 进入 `/api/v1`，破坏性变化只能进入 `/api/v2` 或新 resource 名称。
3. DTO 不直接暴露 GORM model，尤其不能把 `model.Inbound` 作为新 API 的长期请求模型。
4. 字段废弃采用“兼容读取、双写或派生输出、标记 deprecated、删除”的顺序。
5. 每个新 endpoint 必须同步 OpenAPI、前端 SDK 类型和路由漂移测试。

## 4. 核心数据流

### 4.1 登录与鉴权

```text
Browser
  -> internal/web middleware
  -> session / csrf / auth context
  -> internal/controller auth handler
  -> internal/service user service
  -> internal/database user repository
```

当前事实：`IndexController.login` 写 Session，`BaseController.checkLogin` 和 `APIController.checkAPIAuth` 执行登录门禁，`CSRFMiddleware` 保护状态变更请求。

迁移要求：Session 与 CSRF 保持浏览器路径，JWT/API Token 只能作为新增机器访问路径，不能破坏现有前端登录。

### 4.2 入站保存

```text
UI
  -> DTO validation
  -> controller
  -> service inbound command
  -> repository transaction
  -> xray template render
  -> reload/restart decision
  -> websocket invalidate / traffic refresh
```

当前事实：`InboundService` 通过 `InboundRepository` 读写 `model.Inbound`，仍持有 `xray.XrayAPI` 并自行协调 GORM 事务与 Xray 副作用；写入后通过 `XrayService.SetToNeedRestart` 或 controller 广播触发刷新。

迁移要求：继续减少 repository 的 `*gorm.DB`/`xray.ClientTraffic` 耦合，抽出 `XrayRuntime` 接口，再迁移 controller。不能先改表结构。

### 4.3 流量统计与计费

```text
cron job
  -> xray stats gRPC / access log parser
  -> traffic service
  -> traffic ledger repository
  -> quota / expiry evaluator
  -> websocket / notification
```

当前事实：`XrayTrafficJob`、`CheckClientIpJob`、`PeriodicTrafficResetJob`、`StatsNotifyJob` 分散在 `web/job`，业务逻辑依赖 `web/service` 和 `xray`。

迁移要求：调度保留在 `internal/web`，采集和计费规则收敛到 `internal/service/traffic`。

### 4.4 订阅分发

```text
subscription request
  -> subscription web handler
  -> subscription service read model
  -> formatter capability matrix
  -> URI / Base64 / JSON / Clash / diagnostic
  -> response headers
```

当前事实：`sub.Server` 独立监听，`SUBController` 注册 URI、JSON、Clash 和 diagnose 路由，`SubService` 按 `subId` 读取可用入站。

迁移要求：HTTP 监听与 TLS 归 `internal/web`，订阅生成归 `internal/subscription`。订阅链路只读，不允许写库。

### 4.5 配置热重载

```text
settings update
  -> config snapshot diff
  -> classify live-update / xray-reload / restart-required
  -> xray validate
  -> reload or restart
  -> health check
  -> rollback on failure
```

当前事实：设置更新和 Xray 模板保存主要由 `SettingService`、`XraySettingController`、`XrayService.RestartXray` 组合完成，缺少显式 diff 分类。模板保存已在只读候选视图合并入站并调用本机 Xray 预检；重启先预检再停旧进程，启动失败尝试恢复旧进程。全局设置事务与模板/出站测试 URL 的两次写入不是同一个事务。

迁移要求：Phase 4 前先定义 `ConfigChangeSet`，把“立即生效、Xray reload、面板重启”分成不同结果。

### 4.6 静态资源嵌入与发布

```text
frontend build
  -> web/ui assets
  -> Go embed
  -> runtime config injection
  -> release workflow package
```

当前事实：新 Vue UI 构建产物由 `web/ui.go` 托管，release workflow 负责 Linux amd64/arm64 包装。

迁移要求：`internal/web` 接管 embed 注册后，必须保持 `/panel/`、`/panel/assets/*`、runtime config、CSP nonce 与缓存行为一致。

## 5. 数据库与迁移路线

### 5.1 当前模型映射

| 表/模型 | 当前 owner | 目标处理 |
|---|---|---|
| `users` / `model.User` | `database/model`、`UserService` | Phase 2 后迁出 auth DTO，保留表结构兼容 |
| `inbounds` / `model.Inbound` | `database/model`、`InboundService` | 继续作为活跃写模型，先加 repository，不改表 |
| embedded clients / `model.Client` | `Inbound.Settings` JSON | 新 DTO typed 化，但持久化继续兼容 JSON |
| `client_traffics` / `xray.ClientTraffic` | `xray` + `database.InitDB` | 拆出 persistence model，避免 `database` 依赖 runtime xray 包 |
| `outbound_traffics` | `model.OutboundTraffics` | 归 traffic repository |
| `settings` | `model.Setting`、`SettingService` | 保持 KV 兼容，引入 typed snapshot |
| `inbound_client_ips` | `model.InboundClientIps` | 归 traffic/security read model |
| `history_of_seeders` | seeders | 仍用于判断密码 seeder 是否已执行；与 baseline 管理表并存 |
| `custom_geo_resources` | `CustomGeoService` | 归 resource repository，保留 SSRF/path 校验 |

### 5.2 迁移管理表（已落地）

两张管理表已在当前代码中实现，由 `database/db.go` 的 `initModels()` 和 `recordBaselineMigration()` 维护：

```text
schema_migrations(version, name, checksum, applied_at, duration_ms, status)
migration_events(id, version, direction, started_at, finished_at, status, error, backup_path)
```

对应模型为 `model.SchemaMigration` 和 `model.MigrationEvent`。生产 `service.InitDB` 通过 `InitDBWithMigration` 将 `AutoMigrate`、baseline、管理员和密码 seeder、Xray JSON 兼容转换、历史入站及孤立流量清理放进同一事务。`recordBaselineMigration()` 幂等记录 baseline 版本 `202606260001`，`FirstOrCreate` 避免同一版本及 up 事件重复插入；失败回滚后不发布候选连接。重复初始化仍执行 `AutoMigrate`，管理表仍只是 baseline 标记，没有通用版本队列或 down runner，也不会为每次失败单独留下持久化事件。

候选导入复用 `PrepareDBWithMigration`，不替换全局 DB；完整配置通过本机 Xray 校验后才替换在线数据库。替换持有维护锁，失败时恢复旧库及原运行状态；外部进程仍占有 WAL/SHM/journal 时拒绝替换。当前已经具备这些具体失败恢复机制，不能与尚未实现的通用 migration runner 混淆。

验收口径（当前实现已有前 3 项及迁移事务/导入失败恢复回归）：

1. 空库启动能自动创建当前所有表和管理表。
2. 旧库启动不会丢失 `inbounds.settings`、client stats、settings KV。
3. baseline 记录可重复执行，不重复插入管理记录；不能将整个 InitDB 称为只读操作。
4. 任何 destructive migration 必须先创建备份路径并记录到 `migration_events`（后续 destructive migration 需遵循此约束）。

## 6. Xray-core v26.10.10 集成约束

| 链路 | 约束 |
|---|---|
| release tag | `v26.10.10` 只用于运行时二进制下载、Docker 初始化、release 打包、用户文档 |
| Go module | 保持 `go.mod` 的合法 module 版本，不把 `v26.10.10` 写成 module 依赖 |
| 进程管理 | `Start/Stop/Restart/Reload/Validate/Health` 作为目标动作集 |
| gRPC | stats 读取和 handler 控制分离，所有调用必须带 timeout 和错误映射 |
| 模板 | 当前为 typed 外层结合 `RawMessage` 和未知字段保留；兼容层规范化，完整配置由真实核心校验后写入 runtime config |
| 回滚 | 启动、校验、reload、restart 失败时保留上一份可用 config 和 release |

当前 `xray.Process` 已使用同目录临时文件、`Sync` 和 rename 写 `config.json`，启动失败/500ms 初始窗口内退出时恢复旧文件；`XrayService` 停旧核心前预检，失败启动时恢复旧进程。核心升级在 staging 使用候选核心和 Geo 校验，替换/启动失败回滚文件。以上是单进程失败恢复，不是断电一致性或长期健康保证。Phase 4 迁移时保留这些机制并先抽接口，不直接让 `core.Manager` 接管 legacy Xray 生命周期。

## 7. 重基线设计文档核对清单

此处“完成”表示设计内容已写入本文，不表示目标包迁移、运行时动作集或阶段门禁已通过；本文的重基线分步编号也不替代 UI-first Phase 9/10 门禁。

| 验收项 | 状态 | 证据 |
|---|---|---|
| `internal/` 目标目录图与职责说明 | 完成 | 本文第 2 节 |
| 每个模块唯一 owner 和边界 | 完成 | 本文第 2 节 owner 表 |
| API 响应包、错误码、版本策略 | 完成 | 本文第 3 节 |
| 登录、入站、流量、订阅、热重载数据流 | 完成 | 本文第 4 节 |
| 当前 DB 模型与 ORM 迁移方向 | 完成 | 本文第 5 节 |
| Xray-core `v26.10.10` release 与 Go module 分离说明 | 完成 | 本文第 6 节 |

## 8. 下一阶段执行队列

数据层后续最小步骤如下：

1. **已实现 baseline 记录**：`SchemaMigration`、`MigrationEvent` 与 `recordBaselineMigration()`；通用版本队列和回滚 runner 尚未实现。
2. **已实现三个 repository**：`User`、`Inbound`、`Setting` 接口及 GORM 实现，并接入相应服务；独立 Traffic repository 尚未实现。
3. 把 `xray.ClientTraffic` 的持久化模型从 runtime 包依赖中拆出来，先做类型别名或适配器过渡。
4. 保留数据库空库/旧数据/重复 baseline、兼容转换事务回滚及 `database_import_test.go`、`database_migration_candidate_test.go` 的候选隔离/恢复回归；后续 destructive migration 仍需专门设计备份与失败恢复。
5. `/api/v1` 与请求 ID/日志/限流中间件已落地；后续扩大迁移仍须完成对应数据层和兼容性验收，保持 `default-xray` 只读与旧模型写路径边界。
