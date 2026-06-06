# 核心模块解析

> **目标读者**：后端 / 前端维护者
> **适用版本**：`v3.0.19`
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

1. 读取 `.env`。
2. 根据 `XUI_DEBUG` / `XUI_LOG_LEVEL` 初始化日志级别。
3. 调用 `database.InitDB(config.GetDBPath())`。
4. 创建并启动 `web.Server`。
5. 创建并启动 `sub.Server`；如果 `subEnable=false`，订阅服务直接不监听。
6. 监听 `SIGHUP`、`SIGTERM`、`SIGUSR1`。

---

## 2. `config/`

`config/config.go` 只负责本地路径、版本和日志级别，不读取数据库设置。

| 函数 | 环境变量 | 默认行为 |
|---|---|---|
| `GetVersion()` | 无 | 读取 `config/version`，当前 `3.0.19` |
| `GetAssetVersion()` | 构建注入 `buildHash` | `version.buildHash`，未注入时使用启动时间戳 |
| `GetName()` | 无 | 读取 `config/name`，当前 `x-ui` |
| `IsDebug()` | `XUI_DEBUG` | 值为 `true` 时启用 debug |
| `GetLogLevel()` | `XUI_LOG_LEVEL` | debug 模式固定 `debug`，否则默认 `info` |
| `GetBinFolderPath()` | `XUI_BIN_FOLDER` | 默认 `bin` |
| `GetDBFolderPath()` | `XUI_DB_FOLDER` | Windows 默认可执行文件目录，其他平台默认 `/etc/x-ui` |
| `GetDBPath()` | `XUI_DB_FOLDER` | `<DBFolder>/<name>.db` |
| `GetLogFolder()` | `XUI_LOG_FOLDER` | Windows 默认 `./log`，其他平台默认 `/var/log/x-ui` |

代码中没有旧版的 bin/log path getter；文档和代码应统一使用上表中的目录函数。

---

## 3. `database/`

### 3.1 初始化

`database.InitDB(dbPath string)`：

1. 创建数据库目录，权限 `0750`。
2. 按 `config.IsDebug()` 决定 GORM logger。
3. 使用 SQLite driver 打开数据库。
4. 调用 `initModels()` 执行 `AutoMigrate`。
5. 如果 `users` 表为空，创建默认 `admin/admin`，密码以 bcrypt 保存。
6. 执行 `runSeeders`，把历史明文用户密码迁移到 bcrypt，并记录 `HistoryOfSeeders`。

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
| `Client` | 嵌入 `Inbound.Settings` JSON 的客户端结构，不是独立表 |

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
- 新 Vue UI 和 legacy UI 托管。
- 面板 API、WebSocket 和静态资源。
- Cron 后台任务。
- Telegram Bot 启停。
- 自定义 Geo 启动检查。

### 4.2 路由组织

主要路由入口：

| 文件 | 作用 |
|---|---|
| `web/web.go` | 创建 Gin engine、注册中间件、控制器、WebSocket、legacy 静态资源 |
| `web/ui.go` | 注册新 UI `/panel/*`、`/panel/ui/*` 和 `/panel/assets/*path` |
| `web/controller/xui.go` | 注册 legacy UI `/panel/legacy/*` |
| `web/controller/api.go` | 注册 `/panel/api/*` |
| `web/controller/setting.go` | 注册 `/panel/setting/*` |
| `web/controller/xray_setting.go` | 注册 `/panel/xray/*` |

### 4.3 中间件

`web/web.go` 和相关中间件组合提供：

| 中间件 | 作用 |
|---|---|
| `SecurityHeadersMiddleware` | CSP、frame、content-type、referrer 安全头 |
| `DomainValidatorMiddleware` | 可选 Host 白名单 |
| `gzip.Gzip` | HTTP 压缩 |
| `sessions.Sessions("SuperXray", store)` | Cookie Session |
| base path 注入 | 给模板和控制器提供 `base_path` |
| 静态资源缓存 | 构建产物长缓存，入口 HTML no-store/no-cache |
| `locale.LocalizerMiddleware` | Web 国际化 |
| `RedirectMiddleware` | 兼容旧路径重定向 |
| `CSRFMiddleware` | API 状态变更防护 |

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

负责 legacy 页面和旧设置/Xray 控制器初始化：

| 路由 | 说明 |
|---|---|
| `/panel/legacy` | 重定向到 `/panel/legacy/` |
| `/panel/legacy/` | legacy Dashboard |
| `/panel/legacy/inbounds` | legacy Inbounds |
| `/panel/legacy/settings` | legacy Settings |
| `/panel/legacy/xray` | legacy Xray |

`/panel/setting/*` 和 `/panel/xray/*` 仍在该控制器初始化链路中挂载。

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

路径前缀 `/panel/setting`，使用 CSRF：

| 路由 | 方法 | 说明 |
|---|---|---|
| `/all` | POST | 返回 `entity.AllSetting` |
| `/defaultSettings` | POST | 根据 Host 计算默认设置 |
| `/update` | POST | `ShouldBind(entity.AllSetting)` 后调用 `UpdateAllSetting` |
| `/updateUser` | POST | 字段为 `oldUsername`、`oldPassword`、`newUsername`、`newPassword` |
| `/restartPanel` | POST | 延迟重启面板 |
| `/getDefaultJsonConfig` | GET | 默认 Xray JSON |

服务层保存入口是 `UpdateAllSetting`。

### 5.7 `XraySettingController`

路径前缀 `/panel/xray`，使用 CSRF：

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

### 6.2 `InboundService`

项目中最大的业务服务之一。职责：

- Inbound CRUD、导入和克隆所需基础操作。
- 客户端增删改、批量复制和流量重置。
- 与 Xray API 协同更新运行时 inbound。
- 客户端 IP 记录、在线状态和最后在线时间。
- 分享链接和订阅辅助数据。

该服务仍以 `model.Inbound` 和嵌入式 clients JSON 为核心。

### 6.3 `XrayService`

legacy Xray 进程生命周期服务：

- `RestartXray()`
- `StopXray()`
- `GetXrayTraffic()`
- `GetXrayResult()`
- `SetToNeedRestart()`
- `IsNeedRestartAndSetFalse()`

CoreManager 当前没有接管这些 legacy lifecycle API。

### 6.4 `ServerService`

提供服务器状态和运维能力：

- CPU、内存、磁盘、网络 IO 采集。
- CPU 历史聚合。
- Xray 版本查询与安装。
- 日志读取。
- 数据库导入、导出和安全校验。
- Geo 文件更新。
- 密钥和证书生成工具。

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
| `<subPath>:subid` | 始终 | URI/Base64 或 HTML 订阅页 |
| `<subPath>:subid/diagnose` | 始终 | URI 订阅诊断 |
| `<subJsonPath>:subid` | `subJsonEnable=true` | Xray JSON 订阅 |
| `<subJsonPath>:subid/diagnose` | `subJsonEnable=true` | JSON 订阅诊断 |
| `<subClashPath>:subid` | `subClashEnable=true` | Clash/Mihomo YAML |
| `<subClashPath>:subid/diagnose` | `subClashEnable=true` | Clash 订阅诊断 |

`target` 查询参数可把 `/sub/:subid` 自动路由到 JSON 或 Clash 输出；目标格式未启用时回退 URI 输出。

---

## 10. `frontend/src`

### 10.1 API

| 文件 | 职责 |
|---|---|
| `api/endpoints.ts` | 集中定义 legacy API 路径 |
| `api/request.ts` | Axios client、CSRF token、通用响应 unwrap、登录过期跳转 |
| `api/websocket.ts` | 打开 `basePath + "ws"` 并解析消息 |
| `api/server.ts` / `inbounds.ts` / `xray.ts` / `settings.ts` / `core.ts` | 业务 API SDK |

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
- `NotFoundView`

### 10.4 Xray 兼容工具

`frontend/src/utils/xrayCompat.ts` 和 `xrayProtocolTools.ts` 在前端结构化编辑 Xray JSON：

- outbound CRUD。
- Residential IP Pool，识别 tag 包含 `residential` 的 socks outbound。
- AI residential routing，生成 `ai-residential` balancer、TCP 域名规则和 UDP blocked 规则。
- routing、DNS server、FakeDNS、balancer、reverse。
- DNS policy、runtime policy、observatory、burst observatory。
- WARP Matrix 和协议工具输出。

这些工具只修改模板 JSON；保存仍走 `/panel/xray/update`。

### 10.5 Inbound 兼容工具

`frontend/src/utils/inboundCompat.ts` 和 `schemas/protocolRegistry.ts` 提供：

- 可编辑协议注册表：VMess、VLESS、Tunnel、HTTP、Trojan、Shadowsocks、Mixed、WireGuard、Tun、Hysteria、Hysteria2。
- 协议能力：clients、stream、tls、sniffing、shareLink。
- 分享链接生成：vmess、vless、trojan、shadowsocks、hysteria、wireguard。
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
