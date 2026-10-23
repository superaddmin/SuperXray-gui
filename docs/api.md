# API 接口说明

> **目标读者**：集成开发者 / 前端维护者
> **适用版本**：`v3.6.0`
> **源码核对日期**：2026-10-23（静态契约与现有回归测试；部署配置和外部服务需在目标环境验证）
> **事实来源**：`web/controller/*`、`web/ui.go`、`web/middleware/security.go`、`web/websocket/*`、`sub/*`、`frontend/src/api/*`
> **相关文档**：[系统架构设计](architecture.md) | [核心模块解析](modules.md) | [部署指南](deployment.md) | [Panel OpenAPI](openapi/panel-api.yaml) | [Versioned API OpenAPI](openapi/api-v1.yaml)

> **文档职责**：本文保留为面向维护者的人类说明；`docs/openapi/panel-api.yaml` 是 Vue 面板 API 的机器契约，用于路由漂移测试，并通过 `tools/openapiexport` 生成 `frontend/public/openapi.json`，再由 Vite 输出到 `web/ui/openapi.json`。`docs/openapi/api-v1.yaml` 是新增 `/api/v1/*` 版本化 API 的机器契约。

Panel OpenAPI 只覆盖 `/panel/api/*` 的 75 个操作，不含页面、认证、兼容组、WebSocket 和独立订阅服务。机器契约范围外的 20 个面板操作（根/登录/2FA/登出 4 个、CSRF token 1 个、设置 6 个、Xray 9 个）均在下文逐项说明。

---

## 1. 全局约定

### 1.1 Base Path

Web 面板业务路由挂在配置项 `webBasePath` 下。文档中的路径以 `webBasePath=/` 展示（约定为文档描述方便，非运行时默认值）；如果运行时配置为 `/xui/`，则：

源码默认 `webBasePath` 是 `/super/`，例如默认登录路径为 `/super/panel/login`、默认状态 API 为 `/super/panel/api/server/status`。安装或管理员可修改配置，实际值以当前数据库设置为准。

| 文档路径 | 实际路径示例 |
|---|---|
| `/login` | `/xui/login` |
| `/api/v1/health/live` | `/xui/api/v1/health/live` |
| `/api/v1/metrics` | `/xui/api/v1/metrics` |
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
  version: "3.6.0.<asset-hash>"
}
```

### 1.2 认证与会话

面板使用 Cookie Session，Cookie 名为 `SuperXray`，Path 为运行时 `webBasePath`，启用 `HttpOnly`、`SameSite=Lax`。服务器配置了面板证书和私钥时，会话默认带 `Secure`；登出时删除 Cookie 的 `Secure` 根据当前 TLS 请求或可信代理的 `X-Forwarded-Proto: https` 决定。单独由反向代理终止 HTTPS 的部署须核对实际 Cookie 属性。

Web 和 Sub Server 共用 `SUPERXRAY_TRUSTED_PROXY_CIDRS` 信任边界：未设置时仅信任 `127.0.0.0/8,::1/128`，显式空值不信任任何代理，非法 CIDR 阻止服务启动。只有可信直接连接方的转发头参与 CSRF scheme、限流客户端 IP、请求日志和订阅公开 URL 推导。反向代理还需透传正确的 `Host`；CSRF host 校验仍使用请求 `Host`。

认证流程：

1. `POST /login` 登录并写入 Session。
2. 新 UI 访问 `/panel/*` 页面时由服务端注入 CSRF token。
3. 后续 API 请求携带 Cookie；`/panel/api/*` 未登录统一返回 `404 Not Found`，用于隐藏 API 存在性。
4. 普通页面未登录会重定向到登录页；Ajax 页面 API 可能返回 `401` 或通用 JSON 错误，取决于控制器所在路由组。

`GET /panel/csrf-token` 可在登录后获取 token，返回通用响应的字符串 `obj`。该端点属于页面鉴权组，未登录 Ajax 返回 `401`，普通请求重定向到根路径。

### 1.3 CSRF

`/panel/api/*` 路由组启用 `CSRFMiddleware`。设置与 Xray 控制器也各自在自己的路由组启用该中间件，因此 `/panel/setting/*`、`/panel/xray/*` 兼容组的非安全方法同样要求 CSRF token。即使 POST 只是读取设置或模板，也需 token。安全方法 `GET`、`HEAD`、`OPTIONS`、`TRACE` 直接放行；其他方法必须满足：

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

业务错误通常仍返回 HTTP `200`，调用方必须检查 `success`，不能只判断状态码。文件下载、订阅输出和部分诊断端点不包裹该结构，见对应章节。

`/api/v1/*` 版本化 API 使用 `web/entity.APIResponse`，并通过 `X-Request-ID` 关联请求：

```json
{
  "success": true,
  "requestId": "req_0123456789abcdef0123456789abcdef",
  "data": {}
}
```

错误响应：

```json
{
  "success": false,
  "requestId": "req_0123456789abcdef0123456789abcdef",
  "error": {
    "code": "NOT_FOUND",
    "message": "route not found"
  }
}
```

当前错误码集合为 `UNAUTHORIZED`、`FORBIDDEN`、`NOT_FOUND`、`RATE_LIMITED`、`VALIDATION_ERROR`、`INTERNAL_ERROR`。其中 `/api/v1/*` 被限流时返回 HTTP `429`，错误码为 `RATE_LIMITED`，并通过 `Retry-After` 响应头提示建议重试秒数。

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
| `GET` | `/panel/api/openapi.json` | 新 UI OpenAPI 规范 JSON 文件，用于 API 文档页；需登录 |

辅助 `GET /.well-known/appspecific/com.chrome.devtools.json` 直接注册在 engine 上，不带 `webBasePath`，公开返回 HTTP 200 和空 JSON 对象；它不属于 Panel OpenAPI 的业务契约。

### 2.2 已退役 Legacy UI

旧 HTML UI 已退役，`web/html`、`web/assets` 和 `/panel/legacy*` 页面路由不再注册；默认 `NoRoute` 会返回 `404 Not Found`。新 Vue UI、登录接口、面板 API、`/panel/setting/*` 与 `/panel/xray/*` 控制器仍保留。

未登录访问根路径 `/` 时会重定向到 `/panel/login`，登录成功后进入 `/panel/`。

## 2.3 Versioned API

新增 `/api/v1` 路由面用于后续后端重基线迁移，不改变旧 `/panel/api` 响应字段。

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/api/v1/health/live` | 面板进程存活检查 |
| `GET` | `/api/v1/health/ready` | 面板就绪检查，并包含只读 Xray 运行时快照 |
| `GET` | `/api/v1/metrics` | 面板进程请求指标快照，返回统一响应壳 |

上述三个端点公开访问，不要求 Session 或 CSRF。`live` 和 `ready` 都返回 HTTP `200` 和只读运行时/指标快照，区别为 `data.status` 的 `live` / `ready`；Xray 停止或出错不会把 ready 改为 `503`。Xray 状态枚举是 `running`、`stop`、`error`，与 Core API 的 `stopped` 不同。这些探针没有执行数据库就绪、外部依赖可用性或订阅可用性检查。

成功响应示例：

```json
{
  "success": true,
  "requestId": "req_0123456789abcdef0123456789abcdef",
  "data": {
    "status": "ready",
    "checkedAt": "2026-06-26T00:00:00Z",
    "xray": {
      "state": "running",
      "version": "v26.10.21"
    },
    "metrics": {
      "requests": {
        "total": 3,
        "inFlight": 1,
        "byStatus": {
          "200": 2,
          "404": 1
        },
        "byRoute": {
          "GET /api/v1/health/live": {
            "total": 2,
            "averageLatencyMs": 12
          }
        },
        "averageLatencyMs": 18
      },
      "observedAt": "2026-06-26T00:00:00Z"
    }
  }
}
```

`/api/v1` 当前默认按客户端固定窗口限流为每分钟 120 次。超过限额时返回 HTTP `429`、统一错误壳和 `Retry-After` 响应头。未命中路由会返回 HTTP `404` 和统一错误壳；旧页面和 `/panel/api` 仍保持原有 `404` 行为。

`/api/v1/metrics` 返回与健康响应中 `metrics` 字段相同结构的只读快照。指标统计整个面板 HTTP 请求，存在进程内存中，Web Server 重新创建路由器（包括 SIGHUP 重启）后清零；未匹配路径归并为 `<unmatched>`，路由/状态维度上限和溢出规则见 [运行时可靠性](runtime-reliability.md)。响应生成时当前请求尚未完成，可能反映在 `inFlight` 中；连续请求的数值不保证相同。

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
  "msg": "<本地化登录成功提示>",
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

路径前缀：`/panel/setting`（通过 `XUIController` 注册）和 `/panel/api/setting`（通过 `APIController` 注册）。两组均要求登录与非安全方法的 CSRF token；页面兼容组未登录 Ajax 返回 401/普通请求重定向，API 组未登录返回 404。两者分别创建 `SettingController` 实例并复用同一实现。当前 Vue SDK 使用 `/panel/api/setting/*`。

### POST `/panel/setting/all`

返回 `entity.AllSetting`。主要字段如下：

| 字段 | 类型 | 说明 |
|---|---|---|
| `webListen` / `webDomain` / `webPort` | string / string / int | Web 面板监听、域名校验和端口 |
| `webCertFile` / `webKeyFile` | string | Web HTTPS 证书路径 |
| `webBasePath` | string | Web 路由前缀，保存时强制首尾 `/`；空值或 `/` 规范化为 `/super/` |
| `sessionMaxAge` | int | Session 最大分钟数 |
| `pageSize` / `expireDiff` / `trafficDiff` | int | 列表分页、到期和流量提醒阈值 |
| `remarkModel` / `datepicker` / `timeLocation` | string | 备注模板、日期格式和时区 |
| `tgBot*` / `tgRunTime` / `tgCpu` / `tgLang` | mixed | Telegram Bot 配置 |
| `twoFactorEnable` / `twoFactorToken` | bool / string | 2FA 开关；读取时 token 恒为空字符串，不返回现有密钥 |
| `subEnable` / `subListen` / `subPort` / `subPath` | mixed | 订阅服务开关、监听和 URI 路径 |
| `subURI` / `subJsonURI` / `subClashURI` | string | 公开订阅入口 |
| `subJsonEnable` / `subClashEnable` | bool | JSON 和 Clash 输出开关 |
| `subJsonFragment` / `subJsonNoises` / `subJsonMux` / `subJsonRules` | string | JSON 订阅增强片段 |
| `ldap*` | mixed | LDAP 登录和同步配置 |

### POST `/panel/setting/defaultSettings`

按当前请求 Host 计算默认设置，尤其用于新装环境补齐订阅 URI。

### POST `/panel/setting/update`

提交完整 `entity.AllSetting` 表单或 JSON，并可附带请求专用字段 `twoFactorCode`，调用 `SettingService.UpdateAllSetting` 保存。该接口是整体设置更新，不是 PATCH；应先读取当前设置再修改目标字段。服务端会校验监听地址、端口、证书、面板代理、时区和路径；Web/Sub 监听地址重叠时不允许使用相同端口。生产调用在同一数据库事务中保存所有字段，校验或写入失败不会部分更新。

2FA 变更规则：首次启用须提交新 Base32 密钥及新验证器的当前 6 位验证码；关闭或替换已启用验证器须提交旧验证器的当前验证码。新密钥规范化为大写、无 padding，长度须为 16–128 字符且解码后至少 10 bytes。空 `twoFactorToken` 表示保留现有密钥；关闭后清除存储密钥。未改变 2FA 的普通设置保存不要求验证码，`twoFactorCode` 不持久化，也不在设置读取响应中返回。

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

路径前缀：`/panel/xray`（通过 `XUIController` 注册）和 `/panel/api/xray`（通过 `APIController` 注册）。两组均要求登录与非安全方法的 CSRF token；两者分别创建 `XraySettingController` 实例并复用同一实现。当前 Vue SDK 使用 `/panel/api/xray/*`。

### POST `/panel/xray/`

返回 Xray 模板、Inbound tag 和出站测试 URL。注意 `obj` 是一个字符串，字符串内容本身是 JSON：

```json
{
  "success": true,
  "msg": "",
  "obj": "{\"xraySetting\":{\"inbounds\":[],\"outbounds\":[]},\"inboundTags\":[\"inbound-443\"],\"outboundTestUrl\":\"https://www.google.com/generate_204\"}"
}
```

前端 `frontend/src/api/xray.ts` 会二次解析该字符串；解析后的 `xraySetting` 是配置对象。示例仅展示包装结构，不是可直接运行的完整配置。

### POST `/panel/xray/update`

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `xraySetting` | string | 是 | 完整 Xray JSON 模板字符串 |
| `outboundTestUrl` | string | 否 | 保存到服务端设置中的出站测试 URL；为空时使用 `https://www.google.com/generate_204` |

保存时先解开历史响应包装、规范化兼容字段，再与当前启用入站合成完整配置，使用已安装的 Xray 执行 `run -test -config`。预检失败不会保存模板；缺少 Xray 二进制或依赖的证书/Geo 文件同样会失败。通过后写入现有 `xrayTemplateConfig` 设置；重启仍是独立操作。`outboundTestUrl` 在模板保存后单独写入，当前控制器不返回该写入的错误，因此不能把两个字段视为原子更新。

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
| `GET` | `/options` | 无 | 轻量级 Inbound 选项列表，用于选择器/API 客户端 |
| `GET` | `/get/:id` | `id` path int | 单个 Inbound |
| `POST` | `/add` | `model.Inbound` 表单/JSON 绑定 | 保存后的 Inbound |
| `POST` | `/update/:id` | `model.Inbound` 表单/JSON 绑定 | 保存后的 Inbound |
| `POST` | `/del/:id` | `id` path int | 通用响应 |
| `POST` | `/import` | `data` 表单字段 | 通用响应，`obj` 为新保存的 Inbound |

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

新增及导入均由服务端分配 Inbound 和流量记录主键、当前用户 ID 与派生 tag，不信任上传对象中的关联主键。更新以路径中的 ID 为准，保留既有归属和统计。clients 中未知扩展字段保留；传输字段兼容转换与失败规则见 [上游集成记录](upstream-3x-ui-v3.10.2-integration.md)。

流量字段使用 bytes，正值 `expiryTime` 与 `lastTrafficResetTime` 使用 Unix milliseconds。协议常量及 Vue 可编辑注册表为 `vmess`、`vless`、`tunnel`、`http`、`trojan`、`shadowsocks`、`mixed`、`wireguard`、`tun`、`hysteria`、`hysteria2`；`hysteria2` 生成 Xray 运行配置时规范化为 `hysteria`。WireGuard Peer 存在 `settings.peers` 中。`userId` 不对 JSON 暴露，新增时由当前 Session 写入。

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
| `GET` | `/getClientTrafficsById/:id` | 按 `settings.clients[].id` 字符串（通常 UUID）检索 email 对应的流量列表；不是 Inbound ID |
| `POST` | `/:id/resetClientTraffic/:email` | 重置单客户端流量 |
| `POST` | `/resetAllTraffics` | 重置全部 Inbound 流量 |
| `POST` | `/resetAllClientTraffics/:id` | 重置指定 Inbound 下所有客户端；新 UI 也会传 `-1` 表示全局 |
| `POST` | `/delDepletedClients/:id` | 删除指定 Inbound 下已耗尽客户端 |
| `POST` | `/updateClientTraffic/:email` | 按 email 覆盖客户端 `up`/`down` 当前值，不是增量累加 |

`updateClientTraffic` 使用 JSON Body：

```json
{
  "upload": 1024,
  "download": 2048
}
```

单位为 bytes；省略的字段按零绑定并覆盖对应计数，不支持只提交其中一个字段来保持另一个值。

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
| `GET` | `/checkUpdate` | 对比当前面板版本与 GitHub 最新稳定 Release；控制器缓存 10 分钟，只返回提示信息，不自动安装 |
| `GET` | `/getConfigJson` | 从数据库模板及启用入站重新生成配置对象，不是读取最后启动的配置文件；生成时会检查配额/到期状态 |
| `GET` | `/getDb` | 通过 `VACUUM INTO` 下载包含已提交 WAL 数据的一致性 SQLite 快照，文件名 `x-ui.db` |
| `GET` | `/getNewUUID` | 生成 UUID |
| `GET` | `/getNewX25519Cert` | 生成 X25519 密钥对 |
| `GET` | `/getNewmldsa65` | 生成 ML-DSA-65 密钥 |
| `GET` | `/getNewmlkem768` | 生成 ML-KEM-768 密钥 |
| `GET` | `/getNewVlessEnc` | 生成 VLESS encryption key |

`/checkUpdate` 读取 `XUI_RELEASE_REPO`（默认 `superaddmin/SuperXray-gui`）的 GitHub 最新稳定 Release，与 `config/version` 比较；只返回版本、Release 名称、链接和发布时间，不下载或安装资产。GitHub API 未认证限流为每小时 60 次，控制器缓存 10 分钟；网络失败返回 legacy 错误响应，前端不弹错误提示。

### 7.2 Xray 与 Geo 操作

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/stopXrayService` | 停止 legacy XrayService |
| `POST` | `/restartXrayService` | 重启 legacy XrayService；新 UI 的“启动”也走该端点 |
| `POST` | `/installXray/:version` | 安装 `v26.10.10` 及以后、符合 `vYY.M.D` 形式的正式版本；先用候选二进制和 Geo 文件校验当前配置，再替换并在失败时恢复 |
| `POST` | `/updateGeofile` | 更新全部 Geo 文件 |
| `POST` | `/updateGeofile/:fileName` | 更新指定 Geo 文件，文件名由服务层校验 |

`updateGeofile/:fileName` 只接受 `geoip.dat`、`geosite.dat`、`geoip_IR.dat`、`geosite_IR.dat`、`geoip_RU.dat`、`geosite_RU.dat`；省略参数时更新全部。控制器先检查安全 `.dat` 文件名，服务层再执行固定名单校验；下载、预检与恢复规则见 [运行时可靠性](runtime-reliability.md)。

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

- 请求体最大为 65 MiB（64 MiB 数据库文件 + 1 MiB multipart 开销）。
- 文件大小必须大于等于 16 bytes，且不超过 64 MiB（`service.MaxImportDBFileSize`）。
- 文件名必须等于 `filepath.Base(filename)`，且匹配 `^[a-zA-Z0-9_\-.]+$`。
- 扩展名只允许 `.db`、`.sqlite`、`.sqlite3`。
- 服务层继续做 SQLite 签名及 `integrity_check`，在私有候选库中执行完整迁移，再生成配置并调用本机 Xray 预检。缺少核心、证书或 Geo 文件会拒绝导入。

导入持有数据库维护锁，等待现有请求及定时任务结束；候选迁移不发布全局 DB。只有候选通过后才 checkpoint、停止原本正在运行的核心并替换文件。新库打开或启动失败时恢复原库和原运行状态；外部写入者留下 WAL/SHM/journal 时拒绝替换。成功后清理回滚文件，恢复失败则报告保留位置。原先停止的 Xray 在成功导入后仍保持停止。详细恢复边界见 [运行时可靠性](runtime-reliability.md)。

### 7.5 ECH

`POST /panel/api/server/getNewEchCert`

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|---|
| `sni` | string | 是 | ECH 证书使用的 SNI |

### 7.6 自签名证书

`POST /panel/api/server/getNewSelfSignedCert`

生成 ECDSA P-256 自签名 TLS 服务器证书，有效期一年，证书和私钥均以 PEM 格式返回。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `sni` | string | 否 | 逗号分隔的 SAN 名称，支持 DNS 名和 IP 地址；最多 16 个，总长不超过 1024 字符；为空时使用 `localhost, 127.0.0.1, ::1` |

成功响应示例：

```json
{
  "success": true,
  "msg": "",
  "obj": {
    "cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
    "key": "<PRIVATE_KEY_PEM>"
  }
}
```

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
| `status` | object | `state`、`version`、可选 `pid/errorMsg/binary/config/updatedAt` |
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

调用 Telegram Bot 服务向管理员发送数据库备份。该操作要求登录和 CSRF token；实际发送取决于 TG Bot 配置和服务状态，控制器本身不确认发送结果。

---

## 11. WebSocket

### 11.1 连接

连接地址为当前 `webBasePath + "ws"`：

```text
ws://<host>:<webPort>/<webBasePath>ws
wss://<host>:<webPort>/<webBasePath>ws
```

文档根路径示例为 `/ws`，源码默认路径为 `/super/ws`，`webBasePath=/xui/` 时为 `/xui/ws`。连接必须携带已登录 Session Cookie，未登录返回 401。

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

源码默认 `subEnable=true`、`subJsonEnable=false`、`subClashEnable=true`；订阅路径独立于 `webBasePath`，实际启用格式和入口以当前设置为准。标准服务器交付需要开启 `/json/` 时，先设置 `subJsonEnable=true` 并重启面板；完整交付格式见 [部署指南 9.4 节](deployment.md#94-标准订阅交付格式)。

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

诊断直接返回 JSON，包含 `subId/target/format/supportedFormats/supportedProtocols/totalInbounds/outputNodes/skippedNodes/warnings/skipReasons`；跳过项包含 `inboundId/protocol/reason`，不返回完整客户端凭证。数据库读取失败返回 500 和 `error`。普通格式生成失败或没有可输出内容则返回 HTTP 400、文本 `Error!`。

### 12.1 GET `/sub/:subid`

行为：

- 默认返回协议链接列表；`subEncrypt=true` 时返回 Base64 编码，否则返回明文。
- WireGuard Peer 会生成 WireGuard 配置文本，并进入同一明文/Base64 输出流程；当前没有 HTML 订阅页面，`Accept: text/html`、`?html=1`、`?view=html` 不切换输出格式。
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
