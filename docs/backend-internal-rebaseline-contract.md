# 后端 internal 重基线契约

> **执行阶段：** `docs/superpowers/plans/2026-06-25-3x-ui-backend-architecture-rebaseline-and-xray-v26-3-27-sync.md` Phase 1
>
> **适用范围：** SuperXray-gui 当前 Go 后端，含 `web/`、`sub/`、`database/`、`xray/`、`core/`、`config/`、`main.go`
>
> **版本约束：** 运行时 Xray-core release tag 固定为 `v26.3.27`。该 release tag 不等同于 `github.com/xtls/xray-core` Go module 版本。

本文是后端重基线的第一份可执行契约。它不要求一次性搬迁目录，而是先固定目标 `internal/` 边界、依赖方向、API 契约和迁移验收口径。后续代码迁移必须以本文为约束，避免在 controller、service、database、xray runtime 之间继续扩大隐式耦合。

## 1. 当前事实基线

| 当前入口 | 当前职责 | 迁移判断 |
|---|---|---|
| `main.go` | CLI、日志初始化、DB 初始化、Web/Sub server 生命周期、信号处理 | 入口应薄化，保留装配和生命周期，不继续承载业务命令细节 |
| `config/` | 本地路径、版本、日志级别、环境变量读取 | 可直接演进为 `internal/config` 的本地配置层 |
| `database/` | SQLite/GORM 初始化、AutoMigrate、seeders、全局 DB | 先补版本化迁移，再拆 repository |
| `database/model/` | `User`、`Inbound`、`Setting`、流量与 Geo 模型 | 当前仍是活跃写模型，不能直接替换 |
| `web/web.go` | Gin engine、中间件、Session、locale、WebSocket、cron、UI、Xray 启停 | 应拆成 `internal/web` 的装配层和 `internal/service` 的业务层 |
| `web/controller/` | 页面/API 控制器、参数绑定、统一响应、鉴权入口 | 应迁入 `internal/controller`，禁止继续下沉业务计算 |
| `web/service/` | 设置、用户、入站、Xray、服务器、Geo、TG Bot 等业务逻辑 | 应迁入 `internal/service`，并通过接口依赖 database/xray/notifier |
| `web/job/` | Xray 状态、流量、IP、重置、LDAP、告警等定时任务 | 调度归 `internal/web`，业务动作归 `internal/service` |
| `web/middleware/` | 安全头、CSRF、Host 校验、重定向 | 应归 `internal/web/middleware` 或 `internal/web` |
| `web/ui.go`、`web/ui/` | Vue 静态资源 go:embed 托管和 runtime config 注入 | 保留在 `internal/web` 作为发布装配边界 |
| `sub/` | 独立订阅 HTTP 服务、URI/JSON/Clash 输出、诊断 | 业务生成归 `internal/subscription`，HTTP server 装配归 `internal/web` |
| `xray/` | Xray config、进程、gRPC API、traffic、client traffic | 应归 `internal/xray`，禁止直接依赖 Gin 或 DB |
| `core/` | 多核心抽象和 sing-box 实验适配 | Phase 10 前只读/实验边界保持不变，不接管 legacy Xray 生命周期 |

## 2. 目标 internal 包边界

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

新增接口必须优先进入版本化命名空间，建议路径为 `/api/v1/*`。标准响应：

```json
{
  "success": true,
  "data": {},
  "error": null,
  "code": "OK",
  "requestId": "req_..."
}
```

错误响应：

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "invalid request payload",
    "details": {}
  },
  "code": "VALIDATION_FAILED",
  "requestId": "req_..."
}
```

### 3.3 错误码分层

| 范围 | 示例 | HTTP 状态 | 说明 |
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

当前事实：`InboundService` 直接操作 `model.Inbound` 和 `xray.XrayAPI`，写入后通过 `XrayService.SetToNeedRestart` 或 controller 广播触发刷新。

迁移要求：先抽 `InboundRepository` 与 `XrayRuntime` 接口，再迁移 controller。不能先改表结构。

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

当前事实：设置更新和 Xray 模板保存主要由 `SettingService`、`XraySettingController`、`XrayService.RestartXray` 组合完成，缺少显式 diff 分类。

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
| `history_of_seeders` | seeders | 保留兼容，引入 `schema_migrations` 后只作历史记录 |
| `custom_geo_resources` | `CustomGeoService` | 归 resource repository，保留 SSRF/path 校验 |

### 5.2 Phase 2 迁移准备

Phase 2 开始前必须先补两张管理表：

```text
schema_migrations(version, name, checksum, applied_at, duration_ms, status)
migration_events(id, version, direction, started_at, finished_at, error, backup_path)
```

验收口径：

1. 空库启动能自动创建当前所有表和管理表。
2. 旧库启动不会丢失 `inbounds.settings`、client stats、settings KV。
3. migration 可重复执行，重复执行只读状态，不重复写数据。
4. 任何 destructive migration 必须先创建备份路径并记录到 `migration_events`。

## 6. Xray-core v26.3.27 集成约束

| 链路 | 约束 |
|---|---|
| release tag | `v26.3.27` 只用于运行时二进制下载、Docker 初始化、release 打包、用户文档 |
| Go module | 保持 `go.mod` 的合法 module 版本，不把 `v26.3.27` 写成 module 依赖 |
| 进程管理 | `Start/Stop/Restart/Reload/Validate/Health` 作为目标动作集 |
| gRPC | stats 读取和 handler 控制分离，所有调用必须带 timeout 和错误映射 |
| 模板 | 输入为 typed struct，输出 JSON 必须校验后写入 runtime config |
| 回滚 | 启动、校验、reload、restart 失败时保留上一份可用 config 和 release |

当前 `xray.Process` 仍直接写 `config.json` 并启动外部二进制。Phase 4 迁移时先抽接口，不直接让 `core.Manager` 接管 legacy Xray 生命周期。

## 7. Phase 1 验收清单

| 验收项 | 状态 | 证据 |
|---|---|---|
| `internal/` 目标目录图与职责说明 | 完成 | 本文第 2 节 |
| 每个模块唯一 owner 和边界 | 完成 | 本文第 2 节 owner 表 |
| API 响应包、错误码、版本策略 | 完成 | 本文第 3 节 |
| 登录、入站、流量、订阅、热重载数据流 | 完成 | 本文第 4 节 |
| 当前 DB 模型与 ORM 迁移方向 | 完成 | 本文第 5 节 |
| Xray-core `v26.3.27` release 与 Go module 分离说明 | 完成 | 本文第 6 节 |

## 8. 下一阶段执行队列

Phase 2 从数据层开始，建议最小步骤如下：

1. 新增 `database` 迁移管理表和幂等迁移 runner，暂不改变业务表。
2. 为 `User`、`Inbound`、`Setting`、`Traffic` 建 repository interface 和 GORM 实现。
3. 把 `xray.ClientTraffic` 的持久化模型从 runtime 包依赖中拆出来，先做类型别名或适配器过渡。
4. 给空库、旧库、重复迁移、失败回滚补集成测试。
5. 只有 Phase 2 验收通过后，再进入 API/middleware 重构。
