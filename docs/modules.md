# 核心模块解析

> **目标读者**：后端 / 前端维护者
> **适用版本**：`v3.5.0`
> **源码核对日期**：2026-10-15
> **事实来源**：`main.go`、`config/`、`database/`、`web/`、`sub/`、`core/`、`frontend/src`
> **相关文档**：[系统架构设计](architecture.md) | [API 接口说明](api.md) | [开发者贡献指南](development.md)

---

## 1. `main.go`

`main.go` 是进程入口，负责 CLI 分发、环境加载、数据库初始化和服务生命周期。

### 1.1 CLI

| 命令 | 行为 |
|---|---|
| 无参数 / `run` | 启动 Web Server 和 Sub Server |
| `setting` | 修改面板端口、路径、用户、密码、TG Bot 等设置 |
| `migrate` | 执行数据库迁移相关命令 |
| `cert` | 管理证书 |
| `-v` | 输出 `config.GetVersion()` |

### 1.2 服务启动

`runWebServer` 的关键链路：

1. 根据进程环境中的 `XUI_DEBUG` / `XUI_LOG_LEVEL` 初始化日志级别。
2. 读取 `.env`；这一步晚于 logger 初始化，仅写入 `.env` 的日志变量不会回溯修改已初始化级别。
3. 调用 `service.InitDB(config.GetDBPath())`，统一执行数据库和历史配置迁移；提交后才发布连接。
4. 创建并启动 `web.Server`。
5. 创建并启动 `sub.Server`；如果 `subEnable=false`，订阅服务直接不监听。
6. 监听 `SIGHUP`、`SIGTERM`、`SIGUSR1`。

---

## 2. `config/`

`config/config.go` 只负责本地路径、版本和日志级别，不读取数据库设置。

| 函数 | 环境变量 | 默认行为 |
|---|---|---|
| `GetVersion()` | 无 | 读取 `config/version`，当前 `3.5.0` |
| `GetAssetVersion()` | 构建注入 `buildHash` | 返回 `version.buildHash`；未注入时 `config.init()` 使用启动初始化时的十六进制 UnixNano 填充 hash |
| `GetName()` | 无 | 读取 `config/name`，当前 `x-ui` |
| `IsDebug()` | `XUI_DEBUG` | 值为 `true` 时启用 debug |
| `GetLogLevel()` | `XUI_LOG_LEVEL` | debug 模式固定 `debug`，否则默认 `info` |
| `GetBinFolderPath()` | `XUI_BIN_FOLDER` | 默认 `bin` |
| `GetDBFolderPath()` | `XUI_DB_FOLDER` | Windows 默认可执行文件目录；识别为 Go 临时构建路径时回退当前工作目录；其他平台默认 `/etc/x-ui` |
| `GetDBPath()` | `XUI_DB_FOLDER` | `<DBFolder>/<name>.db` |
| `GetLogFolder()` | `XUI_LOG_FOLDER` | Windows 默认 `./log`，其他平台默认 `/var/log/x-ui` |

代码中没有旧版的 bin/log path getter；文档和代码应统一使用上表中的目录函数。

---

## 3. `database/`

### 3.1 初始化

生产启动、CLI 和导入通过 `web/service.InitDB(dbPath)` 进入 `database.InitDBWithMigration`：

1. 创建数据库目录（`0750`），创建并保护数据库文件（`0600`）。
2. 按 `config.IsDebug()` 决定 GORM logger。
3. 使用 SQLite driver 打开数据库。
4. 在同一事务中执行 `initModels()`/`AutoMigrate` 和 `recordBaselineMigration()`，幂等记录 baseline 到 `schema_migrations`、`migration_events`。
5. 同一事务内为无用户的空库创建 `admin/admin` 管理员（存储 bcrypt），运行密码 seeder（已有 bcrypt 不重复哈希），记录 `HistoryOfSeeders`。
6. 同一事务内调用 `normalizeStoredXrayConfig` 转换入站及模板兼容字段，再执行 service 提供的历史入站字段/客户端流量迁移与孤立统计清理；任何步骤失败全部回滚。
7. 事务提交后原子替换全局连接，关闭旧连接。候选库入口 `PrepareDBWithMigration` 不发布全局连接，由调用方关闭。

`database.InitDB` 是不带业务迁移 hook 的底层入口，不能代替生产 `service.InitDB`。`BackupSQLite` 使用 `VACUUM INTO` 创建一致性快照；`ValidateSQLiteDB` 单独检查文件完整性。`database/maintenance.go` 的读租约与独占维护锁协调请求、cron 和数据库替换，不能替代跨进程 SQLite 锁。

### 3.2 迁移模型

`initModels()` 当前迁移：

```go
&model.User{}
&model.Inbound{}
&model.OutboundTraffics{}
&model.Setting{}
&model.InboundClientIps{}
&xray.ClientTraffic{}
&model.HistoryOfSeeders{}
&model.CustomGeoResource{}
&model.SchemaMigration{}
&model.MigrationEvent{}
```

### 3.3 核心模型

`database/model/model.go` 定义：

| 模型 | 作用 |
|---|---|
| `User` | 面板账户 |
| `Inbound` | 当前唯一活跃 Xray 入站写模型 |
| `OutboundTraffics` | outbound tag 流量统计 |
| `InboundClientIps` | 客户端 IP 记录 |
| `HistoryOfSeeders` | 种子历史 |
| `Setting` | 数据库键值设置 |
| `CustomGeoResource` | 自定义 Geo 资源 |
| `SchemaMigration` | 数据库迁移版本记录，幂等记录 baseline 迁移 |
| `MigrationEvent` | 当前 baseline up/applied 事件，不是通用失败事件日志或 down runner |
| `Client` | 嵌入 `Inbound.Settings` JSON 的客户端结构，不是独立表 |

`database/repository.go` 已提供 `UserRepository`、`SettingRepository`、`InboundRepository` 和 GORM 实现；`NewRepositories` 聚合这三个边界。`UserService`、`SettingService`、`InboundService` 支持构造注入，未注入时从全局 DB 惰性创建实现。入站事务和 Xray 副作用仍由 service 协调；Repository 当前仍接受 `*gorm.DB` 并引用 `xray.ClientTraffic`，尚未迁入目标 `internal/` 层。

支持的 `Protocol` 常量：

```go
vmess, vless, tunnel, http, trojan, shadowsocks,
mixed, wireguard, tun, hysteria, hysteria2
```

`IsHysteria` 同时接受 `hysteria` 和 `hysteria2`，因为 UI 默认把 Hysteria v2 存为 `hysteria`，外部导入可能携带 `hysteria2`。

---

## 4. `web/` Server

### 4.1 服务器职责

`web.Server` 负责：

- Web 面板 HTTP/HTTPS 监听。
- 新 Vue UI 托管。
- 面板 API、WebSocket 和静态资源。
- Cron 后台任务。
- Telegram Bot 启停。
- 自定义 Geo 启动检查。

### 4.2 路由组织

主要路由入口：

| 文件 | 作用 |
|---|---|
| `web/web.go` | 创建 Gin engine、注册中间件、控制器和 WebSocket |
| `web/ui.go` | 注册新 UI `/panel/*`、`/panel/ui/*` 和 `/panel/assets/*path` |
| `web/controller/xui.go` | 初始化设置和 Xray 控制器，提供 `/panel/csrf-token` |
| `web/controller/api.go` | 注册 `/panel/api/*` |
| `web/controller/v1.go` | 注册公开 `/api/v1/health/live`、`/api/v1/health/ready`、`/api/v1/metrics` |
| `web/controller/setting.go` | 注册 `/panel/setting/*` |
| `web/controller/xray_setting.go` | 注册 `/panel/xray/*` |

### 4.3 中间件

`web/web.go` 和相关中间件组合提供：

| 中间件 | 作用 |
|---|---|
| `SecurityHeadersMiddleware` | CSP、frame、content-type、referrer 安全头 |
| `trustedproxy.Configure` | Web/Sub 共用可信代理 CIDR；默认仅回环，显式空值禁用转发头信任 |
| `DatabaseLeaseMiddleware` | 请求持有数据库读租约；导入 handler 自行持有维护锁，等待其他请求/任务退出 |
| `RequestIDMiddleware` | 校验/生成 `X-Request-ID`，写入响应头和 Gin context |
| `StructuredRequestLoggerMiddleware` | 记录请求与进程内指标，未知路径归并为 `<unmatched>` |
| `DomainValidatorMiddleware` | 可选 Host 白名单 |
| `gzip.Gzip` | HTTP 压缩 |
| `sessions.Sessions("SuperXray", store)` | Cookie Session |
| base path 注入 | 给控制器提供 `base_path` |
| 静态资源缓存 | 构建产物长缓存，入口 HTML no-store/no-cache |
| `locale.LocalizerMiddleware` | Web 国际化 |
| `RedirectMiddleware` | 兼容旧路径重定向 |
| `CSRFMiddleware` | API 状态变更防护 |

### 4.4 版本化观测接口

`V1Controller` 的三个 GET 接口公开访问，每客户端固定窗口默认每分钟 120 次。`web/entity.APIResponse` 提供 `success/requestId/data/error` 响应，`RequestIDMiddleware` 和 `StructuredRequestLoggerMiddleware` 在 Web engine 级注册。`HealthService` 的 live/ready 都只是只读快照，Xray 未运行仍返回 HTTP 200；`RequestMetricsStore` 统计面板请求，每次重建 Web 路由器时重新创建，进程重启或 SIGHUP 重启服务都会清零。具体返回与容量限制见 [API 接口说明](api.md) 和 [运行时可靠性](runtime-reliability.md)。

---

## 5. Controller 层

### 5.1 `BaseController`

`web/controller/base.go` 当前只提供：

- `checkLogin(c)`：页面鉴权中间件。Ajax 未登录返回 `401` JSON，普通页面重定向到 `base_path`。
- `I18nWeb(c, key, params...)`：从 Gin context 取 `I18n` 函数。

没有 `isLogin` 或 `getI18nWebFunc` 方法。

### 5.2 `IndexController`

| 方法 | 路由 | 说明 |
|---|---|---|
| `index` | `GET /` | 根页面；按登录状态跳转 |
| `login` | `POST /login` | 登录并写 Session/CSRF |
| `logout` | `GET /logout` | 清除 Session |
| `getTwoFactorEnable` | `POST /getTwoFactorEnable` | 返回 2FA 开关 |

### 5.3 `XUIController`

负责旧设置/Xray 控制器初始化，不再注册旧 HTML 页面：

| 路由 | 说明 |
|---|---|
| `/panel/setting/*` | 面板设置控制器 |
| `/panel/xray/*` | Xray 设置与控制器 |
| `/panel/csrf-token` | 已登录 SPA 获取当前 Session CSRF token |

`/panel/legacy*` 已退役，不再注册路由。

### 5.4 `APIController`

`web/controller/api.go` 创建 `/panel/api` 分组：

```go
api := g.Group("/panel/api")
api.Use(a.checkAPIAuth)
api.Use(middleware.CSRFMiddleware())
```

`checkAPIAuth` 未登录返回 `404`，再执行 CSRF。子控制器：

- `InboundController`：`/panel/api/inbounds`
- `ServerController`：`/panel/api/server`
- `CoreController`：`/panel/api/cores`
- `CustomGeoController`：`/panel/api/custom-geo`
- `BackuptoTgbot`：`POST /panel/api/backuptotgbot`
- `SettingController` / `XraySettingController`：`/panel/api/setting/*`、`/panel/api/xray/*`，与兼容组分别创建控制器实例

### 5.5 `InboundController`

职责：

- Inbound CRUD。
- 客户端添加、更新、删除、按 email 删除。
- 跨入站复制客户端。
- 客户端流量和 IP 记录。
- 在线客户端和最后在线时间。
- 单个 Inbound JSON 导入。
- 变更后按需设置 Xray 重启标记并广播 WebSocket。

特别注意：

- `/import` 只解析表单字段 `data` 中的单个 `model.Inbound` JSON。
- `/updateClientTraffic/:email` 使用 JSON Body `{upload, download}`。
- `copyClients` 读取 `sourceInboundId`、重复表单字段 `clientEmails` 和 `flow`，路径 `:id` 是目标 Inbound。

### 5.6 `SettingController`

路径前缀 `/panel/setting`（通过 `XUIController` 注册）和 `/panel/api/setting`（通过 `APIController` 注册）。控制器内部启用 CSRF，因此两组均要求登录且 POST 必须带 token；未登录分别使用页面鉴权和 API 隐藏式 404。两者分别创建实例并复用相同实现。以下路由适用于两个路径前缀：

| 路由 | 方法 | 说明 |
|---|---|---|
| `/all` | POST | 返回 `entity.AllSetting` |
| `/defaultSettings` | POST | 根据 Host 计算默认设置 |
| `/update` | POST | `ShouldBind(updateSettingForm)` 绑定 `AllSetting` 与请求专用 `twoFactorCode`，调用 `UpdateAllSetting` |
| `/updateUser` | POST | 字段为 `oldUsername`、`oldPassword`、`newUsername`、`newPassword` |
| `/restartPanel` | POST | 延迟重启面板 |
| `/getDefaultJsonConfig` | GET | 默认 Xray JSON |

服务层保存入口是 `UpdateAllSetting`。`/all` 不返回现有 TOTP 密钥；启用需新验证码，禁用/替换需旧验证码。验证码只属于请求；其他设置和安全字段在同一事务保存，接口为整体更新。

### 5.7 `XraySettingController`

路径前缀 `/panel/xray`（通过 `XUIController` 注册）和 `/panel/api/xray`（通过 `APIController` 注册）。控制器内部启用 CSRF，因此两组均要求登录且 POST 必须带 token。两者分别创建实例并复用相同实现。以下路由适用于两个路径前缀：

- 读取和保存 Xray JSON 模板。
- 读取 outbound 流量。
- 重置 outbound 流量。
- 使用服务端保存的测试 URL 测试 outbound。
- 操作 WARP 和 NordVPN。

`POST /panel/xray/` 的 `obj` 是字符串化 JSON，其中包含 `xraySetting`、`inboundTags` 和 `outboundTestUrl`。

### 5.8 `ServerController`

职责：

- 状态缓存与 CPU 历史。
- Xray 版本列表、安装、停止、重启。
- 配置和数据库下载。
- 面板日志、Xray 日志读取。
- Geo 文件更新。
- 数据库导入。
- UUID、X25519、ML-DSA-65、ML-KEM-768、VLESS enc、ECH 生成。

`NewServerController` 会调用 `startTask()`，向全局 Cron 注册每 2 秒状态刷新任务，并通过 WebSocket 广播 `status`。

### 5.9 `CoreController`

路径前缀 `/panel/api/cores`：

| 路由 | 说明 |
|---|---|
| `GET /instances` | 列出实例 |
| `GET /instances/:id` | 获取实例详情 |
| `GET /instances/:id/status` | 获取状态 |
| `POST /instances/:id/validate` | 校验配置 |
| `POST /instances/:id/start` | 启动 |
| `POST /instances/:id/stop` | 停止 |
| `POST /instances/:id/restart` | 重启 |

`default-xray` 是只读 legacy 实例；`experimental-sing-box` 是实验性外部二进制实例。

### 5.10 `CustomGeoController`

提供自定义 Geo 资源列表、别名、添加、更新、删除、单项下载和全部更新。写操作继续走 `/panel/api` 的登录和 CSRF 链。

---

## 6. Service 层

### 6.1 `SettingService`

`web/service/setting.go` 以 `settings` 表为存储。核心职责：

- 聚合数据库键值为 `entity.AllSetting`。
- 提供 Web/Sub/TG/LDAP/订阅/Xray 相关 getter。
- `GetDefaultSettings(host)` 生成默认设置和订阅 URI。
- `UpdateAllSetting(allSetting)` 保存设置并做格式校验。
- `GetXrayConfigTemplate()`、`GetDefaultXrayConfig()`、`SetXrayOutboundTestUrl()` 等 Xray 设置辅助。
- `twoFactorToken` 读取脱敏、变更验证和全设置事务保存；`webBasePath` 空值或 `/` 规范化为 `/super/`。

### 6.2 `InboundService`

项目中最大的业务服务之一。职责：

- Inbound CRUD、导入和克隆所需基础操作。
- 客户端增删改、批量复制和流量重置。
- 与 Xray API 协同更新运行时 inbound。
- 客户端 IP 记录、在线状态和最后在线时间。
- 分享链接和订阅辅助数据。

该服务仍以 `model.Inbound` 和嵌入式 clients JSON 为核心。

新增/导入分配新主键并清除上传流量记录的关联主键；更新保留原入站归属、统计与未知 JSON 扩展。并发写操作有服务层协调，数据库写入失败会回滚对应事务并标记运行态重新同步；数据库事务不把已经执行的 Xray gRPC 副作用变成原子操作。

### 6.3 `XrayService`

legacy Xray 进程生命周期服务：

- `RestartXray()`
- `StopXray()`
- `GetXrayTraffic()`
- `GetXrayResult()`
- `SetToNeedRestart()`
- `IsNeedRestartAndSetFalse()`

CoreManager 当前没有接管这些 legacy lifecycle API。

`GetXrayConfig` 合并模板与启用入站、规范化旧字段，按客户端启用/到期/配额过滤并保留核心扩展；生成路径可调用 `AddTraffic(nil,nil)` 检查配额和到期。候选预检使用 `getXrayConfig(false)`，不写流量。重启在停止旧进程前执行真实核心 `run -test -config`，启动失败尝试恢复旧进程；手工停止后后台不会自行启动。

`XraySettingService` 保存前解开历史响应包装，预检合并入站后的完整模板。`xray.Process` 同目录临时文件同步后替换配置，并在启动失败/500ms 内退出时恢复旧文件。`xray/validate.go` 设置 15 秒校验超时，错误不回传可能含密钥的核心原始诊断。

### 6.4 `ServerService`

提供服务器状态和运维能力：

- CPU、内存、磁盘、网络 IO 采集。
- CPU 历史聚合。
- Xray 版本查询与安装。
- 日志读取。
- 数据库导入、导出和安全校验。
- Geo 文件更新。
- 密钥和证书生成工具。

核心安装只接受 `v26.10.10` 及以后正式版本，在私有 staging 目录用候选核心与候选 Geo 预检，再替换并保留恢复副本至启动成功。实现分布于 `server.go` 与 `xray_update.go`。

数据库导入实现分布于 `server.go` 与 `database_import.go`：64 MiB 限额、SQLite 完整性检查、候选完整迁移及核心预检；维护锁排空请求/任务后替换，打开或启动失败恢复原库。核心原本停止时保持停止；恢复失败保留原库路径供人工处理。

### 6.5 `OutboundService`

负责 outbound 流量统计和连通性测试。`TestOutbound` 会：

1. 解析待测试 outbound。
2. 创建临时 Xray config。
3. 启动临时 Xray 进程。
4. 通过本地测试 inbound 访问服务端保存的测试 URL。
5. 返回 `success`、`delay`、`statusCode` 或 `error`。

### 6.6 `UserService`

提供本地 bcrypt 密码校验、LDAP 登录、TOTP 2FA 和用户更新。

### 6.7 `TgbotService`

实现 Telegram Bot 命令、登录通知、统计报告、数据库备份、回调哈希防重放和二维码相关功能。

### 6.8 `WarpService` / `NordService`

- `warp.go`：WARP 数据、配置、注册和 license。
- `nord.go`：国家、服务器、凭据和 key 管理。

新 UI 的 WARP Matrix 会在 Xray JSON 模板层生成多个 WARP outbound 和规则，不新增数据库模型。

### 6.9 `CustomGeoService`

负责 Geo 资源验证、下载、本地缓存、别名规范化、保留别名、防 SSRF 和路径安全。

### 6.10 `CoreService`

`web/service/core_service.go` 是 `/panel/api/cores` 与 `core/` 包之间的适配层。初始化时注册：

| 实例 | 适配器 | 能力 |
|---|---|---|
| `default-xray` | `defaultXrayAdapter` | read only，生命周期不走 CoreManager |
| `experimental-sing-box` | `core/singbox.Adapter` | external binary，validate/start/stop/restart |

sing-box 默认路径来自 `SUPERXRAY_SING_BOX_*` 环境变量，未设置时回退到 `config.GetBinFolderPath()` 和 `config.GetLogFolder()`。

---

## 7. `core/`

`core/types.go` 定义：

| 类型 | 说明 |
|---|---|
| `CoreType` | `xray`、`sing-box` 等核心类型 |
| `State` | `unknown`、`running`、`stopped`、`error`、`not-installed`、`not-configured` |
| `Capabilities` | read/write/validate/start/stop/restart/lifecycleViaCoreManager |
| `Status` | 核心运行状态 |
| `Instance` | API 返回的核心实例视图 |
| `Adapter` | Manager 调用的核心适配接口 |

`core/manager.go` 负责注册 Adapter、列出实例、查状态和分发生命周期操作。它拒绝重复 ID，找不到实例或能力不支持时返回明确错误。

`core/singbox/adapter.go` 是当前唯一外部核心适配器：通过 `sing-box check -c <config>` 校验配置，并用外部进程执行 start/stop/restart。

---

## 8. `web/websocket/`

### 8.1 Hub

真实结构：

```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
    ctx        context.Context
    cancel     context.CancelFunc
    workerPoolSize int
}
```

它使用普通 map 加读写锁；`broadcast` 传递的是已序列化字节。

### 8.2 Notifier

`notifier.go` 提供：

| 函数 | payload |
|---|---|
| `BroadcastStatus(status any)` | `status` |
| `BroadcastTraffic(traffic any)` | `traffic` |
| `BroadcastInbounds(inbounds any)` | `inbounds` |
| `BroadcastOutbounds(outbounds any)` | `outbounds` |
| `BroadcastNotification(title, message, level string)` | `{title,message,level}` |
| `BroadcastXrayState(state, errorMsg string)` | `{state,errorMsg}` |
| `BroadcastInvalidate(dataType MessageType)` | `{type:<original>}` |

时间戳由 `time.Now().UnixMilli()` 生成。

---

## 9. `sub/`

`sub.Server` 是独立 HTTP 服务。它在 `subEnable=true` 时启动，按数据库设置读取：

源码默认 `subEnable=true`、JSON 关闭、Clash 开启；三个订阅路径默认 `/sub/`、`/json/`、`/clash/`，独立于面板 `/super/` 前缀。

- `subPath`
- `subJsonPath`
- `subClashPath`
- `subJsonEnable`
- `subClashEnable`
- `subEncrypt`
- `subShowInfo`
- JSON fragment/noises/mux/rules
- 订阅标题、支持 URL、公告、Happ 路由规则

`SUBController` 注册：

| 路由 | 条件 | 说明 |
|---|---|---|
| `<subPath>:subid` | 始终 | URI/Base64，WireGuard Peer 输出配置文本 |
| `<subPath>:subid/diagnose` | 始终 | URI 订阅诊断 |
| `<subJsonPath>:subid` | `subJsonEnable=true` | Xray JSON 订阅 |
| `<subJsonPath>:subid/diagnose` | `subJsonEnable=true` | JSON 订阅诊断 |
| `<subClashPath>:subid` | `subClashEnable=true` | Clash/Mihomo YAML |
| `<subClashPath>:subid/diagnose` | `subClashEnable=true` | Clash 订阅诊断 |

`target` 查询参数可把 `/sub/:subid` 自动路由到 JSON 或 Clash 输出；目标格式未启用时回退 URI 输出。

当前订阅控制器不提供 HTML 页面；`Accept: text/html`、`html`、`view` 参数不改变输出格式。HTTP/Mixed 代理账户的分享与订阅由 `sub/proxy_account_subscription.go`、前端 `inboundCompat.ts` 处理，WireGuard 由 `sub/wireguard_subscription.go` 处理。

---

## 10. `frontend/src`

### 10.1 API

| 文件 | 职责 |
|---|---|
| `api/endpoints.ts` | 集中定义 legacy API 路径 |
| `api/request.ts` | Axios client、CSRF token、通用响应 unwrap、登录过期跳转 |
| `api/websocket.ts` | 打开 `basePath + "ws"` 并解析消息 |
| `api/server.ts` / `inbounds.ts` / `xray.ts` / `settings.ts` / `cores.ts` / `auth.ts` / `customGeo.ts` | 业务 API SDK |

`requestLegacy` 只在 `success=false` 时抛出 `ApiError`；`401` 或 `/panel/api/*` 的 `404` 被视为 session expired。

### 10.2 Stores

| Store | 作用 |
|---|---|
| `app` | locale、侧边栏折叠、runtime config |
| `server` | status、WebSocket、`status/xray_state/invalidate` 消息处理 |
| `core` | core instances、选中实例和 lifecycle action 状态 |

### 10.3 Views

当前新 UI 页面：

- `LoginView`
- `DashboardView`
- `LogsView`
- `CoreInstancesView`
- `XrayView`
- `InboundsView`
- `SettingsView`
- `ApiDocsView`（读取需登录的 `/panel/api/openapi.json`）
- `NotFoundView`

### 10.4 Xray 兼容工具

`frontend/src/utils/xrayCompat.ts` 和 `xrayProtocolTools.ts` 在前端结构化编辑 Xray JSON：

- outbound CRUD。
- Residential IP Pool，识别 tag 包含 `residential` 的 socks outbound。
- AI residential routing，生成 `ai-residential` balancer、TCP 域名规则和 UDP blocked 规则。
- routing、DNS server、FakeDNS、balancer、reverse。
- DNS policy、runtime policy、observatory、burst observatory。
- WARP Matrix 和协议工具输出。

这些工具只修改模板 JSON；当前 Vue SDK 的读取与保存分别走 `/panel/api/xray/` 与 `/panel/api/xray/update`，兼容 `/panel/xray/*` 路径继续存在。

### 10.5 Inbound 兼容工具

`frontend/src/utils/inboundCompat.ts` 和 `schemas/protocolRegistry.ts` 提供：

- 可编辑协议注册表：VMess、VLESS、Tunnel、HTTP、Trojan、Shadowsocks、Mixed、WireGuard、Tun、Hysteria、Hysteria2。
- 协议能力：clients、stream、tls、sniffing、shareLink。
- 分享链接生成：vmess、vless、trojan、shadowsocks、hysteria/hysteria2、wireguard、HTTP/Mixed 代理账户。
- 客户端订阅链接生成：`subURI`、可选 `subJsonURI`、可选 `subClashURI`。
- 批量客户端生成，数量限制为 1 到 500。

---

## 11. `logger/` 与 `util/`

`logger/logger.go` 提供控制台/syslog 与文件日志，并保留内存缓冲给日志 API 使用。

常用工具包：

| 目录 | 说明 |
|---|---|
| `util/crypto` | bcrypt 密码哈希 |
| `util/ldap` | LDAP 认证 |
| `util/random` | 随机字符串和数字 |
| `util/json_util` | JSON raw message 辅助 |
| `util/reflect_util` | 反射工具 |
| `util/common` | 错误、格式化、多错误合并 |
| `util/sys` | 跨平台系统和进程工具 |
| `util/pathutil` | 受限路径打开和根路径保护 |
| `util/trustedproxy` | 可信直接代理边界、外部 scheme/host 与客户端 IP 推导 |
| `util/xraycompat` | 保留未知字段的旧配置转换、XHTTP/Reality/TLS/FinalMask/DNS/WireGuard 兼容及运行时投影 |
