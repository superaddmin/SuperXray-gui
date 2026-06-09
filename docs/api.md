# API 接口说明

> **目标读者**：集成开发者 / 前端维护者
> **适用版本**：`v3.0.22`
> **事实来源**：`web/controller/*`、`web/ui.go`、`web/middleware/security.go`、`web/websocket/*`、`sub/*`、`frontend/src/api/*`
> **相关文档**：[系统架构设计](architecture.md) | [核心模块解析](modules.md) | [部署指南](deployment.md) | [Panel OpenAPI](openapi/panel-api.yaml)

> **文档职责**：本文保留为面向维护者的人类说明；`docs/openapi/panel-api.yaml` 是 Vue 面板 API 的机器契约，用于路由漂移测试，并通过 `tools/openapiexport` 生成 `frontend/public/openapi.json`，再由 Vite 输出到 `web/ui/openapi.json`。

---

## 1. 全局约定

### 1.1 Base Path

Web 面板所有路由都挂在配置项 `webBasePath` 下。文档中的路径以源码默认 `webBasePath=/` 展示；如果运行时配置为 `/xui/`，则：

| 文档路径 | 实际路径示例 |
|---|---|
| `/login` | `/xui/login` |
| `/panel/api/server/status` | `/xui/panel/api/server/status` |
| `/ws` | `/xui/ws` |

新 UI 运行时由 `web/ui.go` 注入：

```js
window.__SUPERXRAY_UI_CONFIG__ = {
  apiBasePath: "/",
  basePath: "/",
  uiBasePath: "/panel/",
  csrfToken: "<session csrf token>",
  cspNonce: "<request nonce>",
  version: "3.0.22.<asset-hash>"
}
```

### 1.2 认证与会话

面板使用 Cookie Session，Cookie 名为 `SuperXray`。会话 Cookie 为 `HttpOnly`、`SameSite=Lax`，在 HTTPS 请求或 `X-Forwarded-Proto: https` 下设置 `Secure`。

认证流程：

1. `POST /login` 登录并写入 Session。
2. 新 UI 访问 `/panel/*` 页面时由服务端注入 CSRF token。
3. 后续 API 请求携带 Cookie；`/panel/api/*` 未登录统一返回 `404 Not Found`，用于隐藏 API 存在性。
4. 普通页面未登录会重定向到登录页；Ajax 页面 API 可能返回 `401` 或通用 JSON 错误，取决于控制器所在路由组。

### 1.3 CSRF

`/panel/api/*`、`/panel/setting/*`、`/panel/xray/*` 均启用 `CSRFMiddleware`。安全方法 `GET`、`HEAD`、`OPTIONS`、`TRACE` 直接放行；其他方法必须满足：

- 请求头 `X-CSRF-Token` 或 `X-XSRF-Token` 等于当前 Session 内的 token。
- 如果请求带 `Origin` 或 `Referer`，其 scheme 和 host 必须与当前请求同源。

前端 Axios 默认还会发送 Ajax 识别头，但该头不是 CSRF 凭证。

CSRF 失败响应：

```json
{
  "success": false,
  "msg": "CSRF validation failed"
}
```

HTTP 状态码为 `403`。

### 1.4 通用响应结构

大多数面板 API 使用 `web/entity.Msg`：

```json
{
  "success": true,
  "msg": "",
  "obj": {}
}
```

错误时：

```json
{
  "success": false,
  "msg": "错误描述",
  "obj": null
}
```

文件下载、订阅输出和部分诊断端点不包裹该结构，见对应章节。

### 1.5 请求编码

现有 Go 控制器混用 `ShouldBind`、`PostForm`、`FormFile` 和 JSON Body：

| 类型 | 典型端点 | Content-Type |
|---|---|---|
| 表单绑定 | `/panel/setting/update`、`/panel/api/inbounds/add` | `application/x-www-form-urlencoded` 或 `multipart/form-data` |
| 显式表单字段 | `/panel/xray/update`、`/panel/api/server/logs/:count` | `application/x-www-form-urlencoded` |
| JSON Body | `/panel/api/inbounds/updateClientTraffic/:email` | `application/json` |
| 文件上传 | `/panel/api/server/importDB` | `multipart/form-data` |

---

## 2. 页面路由

### 2.1 新 Vue 3 UI

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/panel/login` | 新 UI 登录页 |
| `GET` | `/panel` | 已登录后重定向到 `/panel/` |
| `GET` | `/panel/` | Dashboard |
| `GET` | `/panel/dashboard` | Dashboard 兼容路由 |
| `GET` | `/panel/logs` | 日志页 |
| `GET` | `/panel/cores` | Core 实例页 |
| `GET` | `/panel/xray` | Xray 配置与工具页 |
| `GET` | `/panel/inbounds` | Inbounds 管理页 |
| `GET` | `/panel/settings` | 设置页 |
| `GET` | `/panel/docs` | API 文档页 |
| `GET` | `/panel/assets/*path` | 新 UI 构建资源 |
| `GET` | `/panel/ui` | 兼容入口，重定向到 `/panel/ui/` |
| `GET` | `/panel/ui/*path` | 新 UI 兼容入口 |

### 2.2 Legacy UI

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/panel/legacy` | 重定向到 `/panel/legacy/` |
| `GET` | `/panel/legacy/` | 旧版 Dashboard |
| `GET` | `/panel/legacy/inbounds` | 旧版 Inbounds |
| `GET` | `/panel/legacy/settings` | 旧版 Settings |
| `GET` | `/panel/legacy/xray` | 旧版 Xray 配置 |

---

## 3. 认证 API

### POST `/login`

登录本地用户或 LDAP 用户；启用 2FA 时会校验 TOTP。

请求字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `username` | string | 是 | 用户名 |
| `password` | string | 是 | 密码 |
| `twoFactorCode` | string | 否 | 2FA 验证码，启用后必填 |

成功响应：

```json
{
  "success": true,
  "msg": "",
  "obj": null
}
```

### POST `/getTwoFactorEnable`

无需登录，返回当前是否启用 2FA。

```json
{
  "success": true,
  "msg": "",
  "obj": false
}
```

### GET `/logout`

清除 Session 并跳转到登录页。

---

## 4. 设置 API

路径前缀：`/panel/setting`。所有非安全方法需要 CSRF token。

### POST `/panel/setting/all`

返回 `entity.AllSetting`。主要字段如下：

| 字段 | 类型 | 说明 |
|---|---|---|
| `webListen` / `webDomain` / `webPort` | string / string / int | Web 面板监听、域名校验和端口 |
| `webCertFile` / `webKeyFile` | string | Web HTTPS 证书路径 |
| `webBasePath` | string | Web 路由前缀，保存时强制首尾 `/` |
| `sessionMaxAge` | int | Session 最大分钟数 |
| `pageSize` / `expireDiff` / `trafficDiff` | int | 列表分页、到期和流量提醒阈值 |
| `remarkModel` / `datepicker` / `timeLocation` | string | 备注模板、日期格式和时区 |
| `tgBot*` / `tgRunTime` / `tgCpu` / `tgLang` | mixed | Telegram Bot 配置 |
| `twoFactorEnable` / `twoFactorToken` | bool / string | 2FA 配置 |
| `subEnable` / `subListen` / `subPort` / `subPath` | mixed | 订阅服务开关、监听和 URI 路径 |
| `subURI` / `subJsonURI` / `subClashURI` | string | 公开订阅入口 |
| `subJsonEnable` / `subClashEnable` | bool | JSON 和 Clash 输出开关 |
| `subJsonFragment` / `subJsonNoises` / `subJsonMux` / `subJsonRules` | string | JSON 订阅增强片段 |
| `ldap*` | mixed | LDAP 登录和同步配置 |

### POST `/panel/setting/defaultSettings`

按当前请求 Host 计算默认设置，尤其用于新装环境补齐订阅 URI。

### POST `/panel/setting/update`

提交 `entity.AllSetting` 表单字段，调用 `SettingService.UpdateAllSetting` 保存。服务端会校验监听地址、端口、证书、时区和订阅路径格式。

### POST `/panel/setting/updateUser`

更新当前登录用户的用户名和密码。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `oldUsername` | string | 是 | 当前用户名 |
| `oldPassword` | string | 是 | 当前密码 |
| `newUsername` | string | 是 | 新用户名，不能为空 |
| `newPassword` | string | 是 | 新密码，不能为空 |

### POST `/panel/setting/restartPanel`

3 秒后重启面板进程。

### GET `/panel/setting/getDefaultJsonConfig`

返回默认 Xray JSON 配置对象。

---

## 5. Xray 设置 API

路径前缀：`/panel/xray`。非安全方法需要 CSRF token。

### POST `/panel/xray/`

返回 Xray 模板、Inbound tag 和出站测试 URL。注意 `obj` 是一个字符串，字符串内容本身是 JSON：

```json
{
  "success": true,
  "msg": "",
  "obj": "{\"xraySetting\":\"{...}\",\"inboundTags\":[\"inbound-443\"],\"outboundTestUrl\":\"https://www.google.com/generate_204\"}"
}
```

前端 `frontend/src/api/xray.ts` 会二次解析该字符串。

### POST `/panel/xray/update`

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `xraySetting` | string | 是 | 完整 Xray JSON 模板字符串 |
| `outboundTestUrl` | string | 否 | 保存到服务端设置中的出站测试 URL；为空时使用 `https://www.google.com/generate_204` |

保存模板不会迁移数据模型；它仍写入现有 Xray 模板设置。

### GET `/panel/xray/getDefaultJsonConfig`

返回默认 Xray JSON 配置对象。

### GET `/panel/xray/getOutboundsTraffic`

返回 `model.OutboundTraffics` 列表：

```json
{
  "success": true,
  "msg": "",
  "obj": [
    {
      "id": 1,
      "tag": "direct",
      "up": 0,
      "down": 0,
      "total": 0
    }
  ]
}
```

### GET `/panel/xray/getXrayResult`

返回当前 XrayService 保存的运行结果或错误摘要。

### POST `/panel/xray/resetOutboundsTraffic`

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `tag` | string | 是 | 要重置的 outbound tag；`-alltags-` 表示全部 |

### POST `/panel/xray/testOutbound`

测试出站连通性。服务端使用已保存的 `xrayOutboundTestUrl`，不会接受客户端传入任意测试 URL，避免 SSRF。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `outbound` | string | 是 | 单个 outbound JSON |
| `allOutbounds` | string | 否 | outbound JSON 数组，用于解析 `sockopt.dialerProxy` 等依赖 |

成功响应 `obj`：

```json
{
  "success": true,
  "delay": 123,
  "statusCode": 204
}
```

失败时 `obj.success=false` 并包含 `error`。

### POST `/panel/xray/warp/:action`

| action | 表单字段 | 说明 |
|---|---|---|
| `data` | 无 | 获取 WARP 数据 |
| `del` | 无 | 删除 WARP 数据 |
| `config` | 无 | 获取 WARP 配置 |
| `reg` | `privateKey`, `publicKey` | 注册 WARP |
| `license` | `license` | 设置 WARP License |

### POST `/panel/xray/nord/:action`

| action | 表单字段 | 说明 |
|---|---|---|
| `countries` | 无 | 获取国家列表 |
| `servers` | `countryId` | 获取服务器列表 |
| `reg` | `token` | 获取 NordVPN 凭据 |
| `setKey` | `key` | 保存密钥 |
| `data` | 无 | 获取已保存数据 |
| `del` | 无 | 删除已保存数据 |

---

## 6. Inbound API

路径前缀：`/panel/api/inbounds`。所有端点都要求登录；非安全方法要求 CSRF token。

### 6.1 Inbound CRUD

| 方法 | 路径 | 请求 | 响应 |
|---|---|---|---|
| `GET` | `/list` | 无 | 当前用户所有 Inbound |
| `GET` | `/get/:id` | `id` path int | 单个 Inbound |
| `POST` | `/add` | `model.Inbound` 表单/JSON 绑定 | 保存后的 Inbound |
| `POST` | `/update/:id` | `model.Inbound` 表单/JSON 绑定 | 通用响应 |
| `POST` | `/del/:id` | `id` path int | 通用响应 |
| `POST` | `/import` | `data` 表单字段 | 通用响应 |

`/import` 的 `data` 是单个 `model.Inbound` JSON 字符串，不是数组：

```json
{
  "remark": "vless-443",
  "enable": true,
  "port": 443,
  "protocol": "vless",
  "settings": "{\"clients\":[]}",
  "streamSettings": "{\"network\":\"tcp\"}",
  "sniffing": "{\"enabled\":true}",
  "tag": "inbound-443"
}
```

Inbound 活跃写模型仍是 `database/model.Inbound`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | int | 主键 |
| `up` / `down` / `total` / `allTime` | int64 | 流量统计和限制 |
| `remark` | string | 备注 |
| `enable` | bool | 是否启用 |
| `expiryTime` | int64 | 过期时间戳 |
| `trafficReset` / `lastTrafficResetTime` | string / int64 | 流量重置策略 |
| `listen` / `port` | string / int | 监听地址与端口 |
| `protocol` | string | Xray 入站协议 |
| `settings` | string | 协议设置 JSON 字符串 |
| `streamSettings` | string | 传输设置 JSON 字符串 |
| `tag` | string | 唯一 tag |
| `sniffing` | string | sniffing JSON 字符串 |
| `clientStats` | array | 关联的 `xray.ClientTraffic` 统计 |

`model.Client` 不独立建表，存放在 `Inbound.Settings` 的 clients JSON 内。

### 6.2 Client 管理

| 方法 | 路径 | 请求 |
|---|---|---|
| `POST` | `/addClient` | `model.Inbound` 绑定，包含目标 inbound 和客户端 settings |
| `POST` | `/:id/copyClients` | `sourceInboundId`、重复字段 `clientEmails`、`flow` |
| `POST` | `/:id/delClient/:clientId` | 删除指定 client ID |
| `POST` | `/:id/delClientByEmail/:email` | 按 email 删除客户端 |
| `POST` | `/updateClient/:clientId` | `model.Inbound` 绑定，替换指定客户端 |

`copyClients` 复制方向为：从 `sourceInboundId` 读取客户端，复制到路径参数 `:id` 指定的目标 Inbound。

### 6.3 流量 API

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/getClientTraffics/:email` | 按 email 查询客户端流量 |
| `GET` | `/getClientTrafficsById/:id` | 按 Inbound ID 查询客户端流量 |
| `POST` | `/:id/resetClientTraffic/:email` | 重置单客户端流量 |
| `POST` | `/resetAllTraffics` | 重置全部 Inbound 流量 |
| `POST` | `/resetAllClientTraffics/:id` | 重置指定 Inbound 下所有客户端；新 UI 也会传 `-1` 表示全局 |
| `POST` | `/delDepletedClients/:id` | 删除指定 Inbound 下已耗尽客户端 |
| `POST` | `/updateClientTraffic/:email` | 手动增加客户端流量 |

`updateClientTraffic` 使用 JSON Body：

```json
{
  "upload": 1024,
  "download": 2048
}
```

### 6.4 IP 与在线状态

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/clientIps/:email` | 获取客户端 IP 记录 |
| `POST` | `/clearClientIps/:email` | 清除客户端 IP 记录 |
| `POST` | `/onlines` | 返回当前在线客户端列表 |
| `POST` | `/lastOnline` | 返回客户端最后在线时间映射 |

---

## 7. Server API

路径前缀：`/panel/api/server`。

### 7.1 状态与工具

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/status` | 返回缓存的服务器状态；后台每 2 秒刷新 |
| `GET` | `/cpuHistory/:bucket` | 返回 CPU 历史聚合，允许 `2/30/60/120/180/300` |
| `GET` | `/getXrayVersion` | 返回可安装 Xray 版本列表，控制器缓存 60 秒 |
| `GET` | `/getConfigJson` | 返回当前 Xray 运行配置 JSON |
| `GET` | `/getDb` | 下载当前 SQLite 数据库，文件名 `x-ui.db` |
| `GET` | `/getNewUUID` | 生成 UUID |
| `GET` | `/getNewX25519Cert` | 生成 X25519 密钥对 |
| `GET` | `/getNewmldsa65` | 生成 ML-DSA-65 密钥 |
| `GET` | `/getNewmlkem768` | 生成 ML-KEM-768 密钥 |
| `GET` | `/getNewVlessEnc` | 生成 VLESS encryption key |

### 7.2 Xray 与 Geo 操作

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/stopXrayService` | 停止 legacy XrayService |
| `POST` | `/restartXrayService` | 重启 legacy XrayService；新 UI 的“启动”也走该端点 |
| `POST` | `/installXray/:version` | 安装指定 Xray 版本 |
| `POST` | `/updateGeofile` | 更新全部 Geo 文件 |
| `POST` | `/updateGeofile/:fileName` | 更新指定 Geo 文件，文件名由服务层校验 |

`updateGeofile/:fileName` 只接受安全文件名，不允许路径分隔符或遍历片段。

### 7.3 日志

日志端点参数是 POST 表单字段，不是 query string。

`POST /panel/api/server/logs/:count`

| 字段 | 类型 | 说明 |
|---|---|---|
| `level` | string | 面板日志级别过滤 |
| `syslog` | string/bool | 是否包含系统日志 |

`POST /panel/api/server/xraylogs/:count`

| 字段 | 类型 | 说明 |
|---|---|---|
| `filter` | string | 关键字过滤 |
| `showDirect` | string/bool | 是否显示 direct/freedom 流量 |
| `showBlocked` | string/bool | 是否显示 blocked/blackhole 流量 |
| `showProxy` | string/bool | 是否显示代理流量 |

### 7.4 数据库导入

`POST /panel/api/server/importDB`

请求为 multipart，文件字段名固定为 `db`。服务端校验：

- 请求体最大为 `service.MaxImportDBFileSize + 1MB`。
- 文件大小必须大于等于 16 bytes，且不超过 `service.MaxImportDBFileSize`。
- 文件名必须等于 `filepath.Base(filename)`，且匹配 `^[a-zA-Z0-9_\-.]+$`。
- 扩展名只允许 `.db`、`.sqlite`、`.sqlite3`。
- 服务层会继续做 SQLite 格式和导入安全校验。

### 7.5 ECH

`POST /panel/api/server/getNewEchCert`

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `sni` | string | 是 | ECH 证书使用的 SNI |

---

## 8. Core API

路径前缀：`/panel/api/cores`。该 API 是当前最小 Phase 10 运行时入口，只暴露实例视图和受控生命周期能力，不迁移 `model.Inbound`。

### 8.1 实例列表

`GET /panel/api/cores/instances`

当前注册两个实例：

| ID | coreType | mode | source | lifecycleOwner | 能力 |
|---|---|---|---|---|---|
| `default-xray` | `xray` | `legacy` | `legacy-inbound-table` | `legacy-xray-service` | 只读；不支持 validate/start/stop/restart |
| `experimental-sing-box` | `sing-box` | `experimental` | `external-binary` | `core-manager` | 支持 validate/start/stop/restart；不支持写配置 |

实例字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` / `name` / `displayName` | string | 实例标识 |
| `coreType` | string | `xray` 或 `sing-box` |
| `mode` | string | `legacy` 或 `experimental` |
| `source` | string | 实例来源 |
| `lifecycleOwner` | string | 生命周期负责方 |
| `status` | object | `state`、`version`、`pid`、`error` 等 |
| `capabilities` | object | `read/write/validate/start/stop/restart/lifecycleViaCoreManager` |
| `writeSupported` | bool | 是否支持通过该 API 写配置 |
| `managerAttached` | bool | 是否挂在 CoreManager 生命周期 |
| `experimentalOnly` | bool | 是否仅作为实验实例 |

状态值包括：`unknown`、`running`、`stopped`、`error`、`not-installed`、`not-configured`。

### 8.2 实例详情与状态

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/instances/:id` | 返回单个实例及当前状态 |
| `GET` | `/instances/:id/status` | 只返回状态 |

### 8.3 生命周期操作

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/instances/:id/validate` | 校验配置 |
| `POST` | `/instances/:id/start` | 启动实例 |
| `POST` | `/instances/:id/stop` | 停止实例 |
| `POST` | `/instances/:id/restart` | 重启实例 |

`default-xray` 对这些操作返回生命周期不支持；legacy Xray 启停仍走 `/panel/api/server/stopXrayService` 与 `/panel/api/server/restartXrayService`。

`experimental-sing-box` 路径来自：

- `SUPERXRAY_SING_BOX_BINARY`，默认 `config.GetBinFolderPath()/sing-box(.exe)`。
- `SUPERXRAY_SING_BOX_CONFIG`，默认 `config.GetBinFolderPath()/sing-box-config.json`。
- `SUPERXRAY_SING_BOX_LOG_FOLDER`，默认 `config.GetLogFolder()`。

---

## 9. Custom Geo API

路径前缀：`/panel/api/custom-geo`。

| 方法 | 路径 | 请求 | 说明 |
|---|---|---|---|
| `GET` | `/list` | 无 | 列出自定义 Geo 资源 |
| `GET` | `/aliases` | 无 | 返回可用别名 |
| `POST` | `/add` | `type`, `alias`, `url` | 新增资源 |
| `POST` | `/update/:id` | `type`, `alias`, `url` | 更新资源 |
| `POST` | `/delete/:id` | `id` path int | 删除资源 |
| `POST` | `/download/:id` | `id` path int | 下载/刷新单个资源 |
| `POST` | `/update-all` | 无 | 批量更新资源 |

`type` 当前用于区分 `geoip` / `geosite`，服务层负责 URL、Host、防 SSRF、别名和本地路径校验。

---

## 10. 其他面板 API

### POST `/panel/api/backuptotgbot`

调用 Telegram Bot 服务向管理员发送数据库备份。该操作要求登录、CSRF token 和已启用 TG Bot。

---

## 11. WebSocket

### 11.1 连接

连接地址为当前 `webBasePath + "ws"`：

```text
ws://<host>:<webPort>/<webBasePath>ws
wss://<host>:<webPort>/<webBasePath>ws
```

例如默认路径为 `/ws`，`webBasePath=/xui/` 时为 `/xui/ws`。连接必须携带已登录 Session Cookie。

### 11.2 消息结构

```json
{
  "type": "status",
  "payload": {},
  "time": 1778153425000
}
```

`time` 是 Unix milliseconds。

### 11.3 消息类型

| type | payload | 来源 |
|---|---|---|
| `status` | `service.Status` | `ServerController` 每 2 秒刷新并广播 |
| `traffic` | `{traffics, clientTraffics, onlineClients, lastOnlineMap}` | `XrayTrafficJob` 每 10 秒采集 |
| `inbounds` | Inbound 列表 | Inbound 变更后广播 |
| `notification` | `{title,message,level}` | 服务端事件通知 |
| `xray_state` | `{state,errorMsg}` | legacy Xray 停止/重启结果 |
| `outbounds` | outbound 流量统计 | outbound 更新场景 |
| `invalidate` | `{type:"inbounds"}` 等 | payload 超过 10MB 或显式轻量刷新 |

`Hub.broadcast` 内部传输的是已序列化 `[]byte`，不是 `Message` 指针；客户端只需按上述 JSON 结构解析。

---

## 12. 订阅服务

订阅服务由 `sub.Server` 独立监听，只有 `subEnable=true` 时启动。它不使用面板登录认证，但可配置独立监听地址、端口、证书和域名校验。

默认路径来自设置：

| 设置 | 默认语义 | 路由 |
|---|---|---|
| `subPath` | URI/Base64 订阅 | `/sub/:subid` |
| `subJsonPath` | Xray JSON 订阅 | `/json/:subid`，仅 `subJsonEnable=true` |
| `subClashPath` | Clash/Mihomo YAML | `/clash/:subid`，仅 `subClashEnable=true` |

每种启用格式都有诊断端点：

```text
GET /sub/:subid/diagnose
GET /json/:subid/diagnose
GET /clash/:subid/diagnose
```

### 12.1 GET `/sub/:subid`

行为：

- 默认返回协议链接列表；`subEncrypt=true` 时返回 Base64 编码，否则返回明文。
- 当 `Accept` 包含 `text/html`、`?html=1` 或 `?view=html` 时，渲染可视化订阅页面。
- 当存在 `target` 查询参数时，会按目标 profile 自动选择格式；例如 `target=xray` 优先 JSON，`target=mihomo` 或 `target=stash` 优先 Clash。对应格式未启用时回退到 URI 输出。

### 12.2 GET `/json/:subid`

返回完整 Xray 客户端 JSON 配置，包含订阅设置中的 Fragment、Noises、Mux 和自定义规则。

### 12.3 GET `/clash/:subid`

返回 Clash/Mihomo YAML，Content-Type 为 `application/yaml; charset=utf-8`。

### 12.4 通用订阅响应头

订阅成功时按配置写入：

| Header | 说明 |
|---|---|
| `Subscription-Userinfo` | `upload=...; download=...; total=...; expire=...` |
| `Profile-Update-Interval` | 订阅更新间隔 |
| `Profile-Title` | `base64:<title>` |
| `Support-Url` | 支持 URL |
| `Profile-Web-Page-Url` | 当前订阅页面 URL 或配置值 |
| `Announce` | `base64:<announce>` |
| `Routing-Enable` | Happ 路由开关 |
| `Routing` | Happ 路由规则 |
