# API 接口说明

> **目标读者**：集成开发者  
> **适用版本**：`v2.9.8`
> **相关文档**：[核心模块解析](modules.md) | [系统架构设计](architecture.md) | [部署指南](deployment.md)

---

## 1. 概述

### 1.1 认证方式

SuperXray 使用 **Cookie-based Session** 认证。所有需要认证的 API 端点通过 Session Cookie 验证身份。

**认证流程**：

1. 调用 `POST /login` 获取 Session Cookie
2. 后续请求自动携带 Cookie
3. 未认证的 API 请求返回 `404 Not Found`（隐藏 API 端点存在性）

### 1.2 请求/响应格式

- **请求格式**：`application/json` 或 `application/x-www-form-urlencoded`
- **响应格式**：`application/json`
- **基础路径**：所有 API 路径基于配置的 `webBasePath`（源码默认 `/`；一键安装通常会随机生成）

### 1.3 通用响应结构

```json
// 成功响应
{
    "success": true,
    "msg": "",
    "obj": { ... }
}

// 错误响应
{
    "success": false,
    "msg": "错误描述信息",
    "obj": null
}
```

### 1.4 错误码定义

| HTTP 状态码 | 含义 |
|-------------|------|
| `200` | 请求成功 |
| `404` | 未认证或资源不存在（未认证时统一返回 404） |
| `500` | 服务器内部错误 |

---

## 2. 认证接口

### POST /login

用户登录，获取 Session Cookie。

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 用户名 |
| `password` | string | 是 | 密码 |
| `twoFactorCode` | string | 否 | TOTP 验证码（启用 2FA 时必填） |

**请求示例**：

```json
{
    "username": "admin",
    "password": "admin"
}
```

**响应示例**：

```json
{
    "success": true,
    "msg": "",
    "obj": null
}
```

### POST /getTwoFactorEnable

获取双因素认证启用状态（无需认证）。

**响应示例**：

```json
{
    "success": true,
    "obj": false
}
```

### GET /logout

用户登出，清除 Session。

**响应**：重定向到登录页面。

---

## 3. 面板页面路由

以下路由返回 HTML 页面（需要认证）：

| 方法 | 路径 | 功能 |
|------|------|------|
| `GET` | `/panel/` | 面板首页（状态仪表盘） |
| `GET` | `/panel/inbounds` | Inbounds 管理页 |
| `GET` | `/panel/settings` | 面板设置页 |
| `GET` | `/panel/xray` | Xray 配置页 |

---

## 4. 设置管理 API

### POST /panel/setting/all

获取所有面板设置。

**响应示例**：

```json
{
    "success": true,
    "obj": {
        "webListen": "",
        "webPort": 2053,
        "webCertFile": "",
        "webKeyFile": "",
        "webBasePath": "/",
        "sessionMaxAge": 360,
        "pageSize": 25,
        "timeLocation": "Local",
        "tgBotEnable": false,
        "tgBotToken": "",
        "tgBotChatId": "",
        "subEnable": true,
        "subPort": 2096,
        "subPath": "/sub/",
        "subJsonPath": "/json/",
        "subClashPath": "/clash/",
        "twoFactorEnable": false,
        "ldapEnable": false
    }
}
```

### POST /panel/setting/defaultSettings

获取默认设置值。

### POST /panel/setting/update

更新所有设置。

**请求参数**：与 `getAllSetting` 返回的结构体相同。

### POST /panel/setting/updateUser

更新用户名和密码。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 新用户名 |
| `password` | string | 是 | 新密码 |

### POST /panel/setting/restartPanel

重启面板服务。

### GET /panel/setting/getDefaultJsonConfig

获取默认 Xray JSON 配置模板。

---

## 5. Xray 配置 API

### POST /panel/xray/

获取 Xray 配置模板、Inbound 标签列表和测试 URL。

### POST /panel/xray/update

更新 Xray 配置。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `xraySetting` | string | 是 | Xray JSON 配置字符串 |
| `outboundTestUrl` | string | 否 | 出站测试 URL |

### GET /panel/xray/getDefaultJsonConfig

获取默认 Xray JSON 配置。

### GET /panel/xray/getOutboundsTraffic

获取所有 Outbound 的流量统计。

**响应示例**：

```json
{
    "success": true,
    "obj": [
        {
            "tag": "direct",
            "up": 1024000,
            "down": 2048000,
            "total": 3072000
        }
    ]
}
```

### GET /panel/xray/getXrayResult

获取 Xray 运行结果（错误信息）。

### POST /panel/xray/resetOutboundsTraffic

重置指定 Outbound 的流量统计。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tag` | string | 是 | Outbound 标签 |

### POST /panel/xray/testOutbound

测试 Outbound 连通性。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `outbound` | string | 是 | Outbound 配置 JSON |
| `allOutbounds` | string | 否 | 所有 Outbound 配置 |

### WARP 操作 API

```
POST /panel/xray/warp/:action
```

| action | 说明 |
|--------|------|
| `data` | 获取 WARP 配置数据 |
| `del` | 删除 WARP 配置 |
| `config` | 获取 WARP 配置 |
| `reg` | 注册 WARP |
| `license` | 设置 WARP License |

### NordVPN 操作 API

```
POST /panel/xray/nord/:action
```

| action | 说明 |
|--------|------|
| `countries` | 获取可用国家列表 |
| `servers` | 获取服务器列表 |
| `reg` | 注册 NordVPN |
| `setKey` | 设置密钥 |
| `data` | 获取配置数据 |
| `del` | 删除配置 |

---

## 6. Inbound 管理 API

### 6.1 Inbound CRUD

#### GET /panel/api/inbounds/list

获取当前用户的所有 Inbound 列表。

**响应示例**：

```json
{
    "success": true,
    "obj": [
        {
            "id": 1,
            "up": 1024000,
            "down": 2048000,
            "total": 0,
            "remark": "vless-tcp",
            "enable": true,
            "expiryTime": 0,
            "listen": "",
            "port": 443,
            "protocol": "vless",
            "settings": "{...}",
            "streamSettings": "{...}",
            "tag": "inbound-443",
            "sniffing": "{...}"
        }
    ]
}
```

#### GET /panel/api/inbounds/get/:id

获取指定 ID 的 Inbound。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `id` | path | int | Inbound ID |

#### POST /panel/api/inbounds/add

添加新的 Inbound。

**请求参数**：Inbound 表单数据（JSON），包含 `protocol`、`port`、`settings`、`streamSettings` 等。

#### POST /panel/api/inbounds/del/:id

删除指定 Inbound。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `id` | path | int | Inbound ID |

#### POST /panel/api/inbounds/update/:id

更新指定 Inbound。

#### POST /panel/api/inbounds/import

批量导入 Inbound。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `data` | string | 是 | Inbound JSON 数组字符串 |

### 6.2 客户端管理

#### POST /panel/api/inbounds/addClient

添加客户端到 Inbound。

**请求参数**：包含 Inbound ID 和 Client 配置的表单数据。

#### POST /panel/api/inbounds/:id/delClient/:clientId

删除指定客户端。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `id` | path | int | Inbound ID |
| `clientId` | path | string | 客户端 ID |

#### POST /panel/api/inbounds/:id/delClientByEmail/:email

通过 Email 删除客户端。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `id` | path | int | Inbound ID |
| `email` | path | string | 客户端 Email |

#### POST /panel/api/inbounds/updateClient/:clientId

更新客户端配置。

#### POST /panel/api/inbounds/:id/copyClients

复制客户端到另一个 Inbound。

| 参数 | 类型 | 说明 |
|------|------|------|
| `sourceInboundId` | int | 源 Inbound ID |
| `clientEmails` | []string | 要复制的客户端 Email 列表 |
| `flow` | string | 流控设置 |

### 6.3 流量管理

#### GET /panel/api/inbounds/getClientTraffics/:email

获取指定客户端的流量统计。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `email` | path | string | 客户端 Email |

**响应示例**：

```json
{
    "success": true,
    "obj": [
        {
            "email": "user@example.com",
            "up": 512000,
            "down": 1024000,
            "total": 10737418240,
            "expiryTime": 1735689600000,
            "enable": true
        }
    ]
}
```

#### GET /panel/api/inbounds/getClientTrafficsById/:id

按 Inbound ID 获取客户端流量。

#### POST /panel/api/inbounds/:id/resetClientTraffic/:email

重置指定客户端的流量。

#### POST /panel/api/inbounds/resetAllTraffics

重置所有 Inbound 的流量统计。

#### POST /panel/api/inbounds/resetAllClientTraffics/:id

重置指定 Inbound 下所有客户端的流量。

#### POST /panel/api/inbounds/updateClientTraffic/:email

手动更新客户端流量。

| 参数 | 类型 | 说明 |
|------|------|------|
| `email` | path | 客户端 Email |
| `upload` | int64 | 上传增量（字节） |
| `download` | int64 | 下载增量（字节） |

#### POST /panel/api/inbounds/delDepletedClients/:id

删除指定 Inbound 下已耗尽（流量用完或过期）的客户端。

### 6.4 IP 管理

#### POST /panel/api/inbounds/clientIps/:email

获取指定客户端的 IP 记录。

#### POST /panel/api/inbounds/clearClientIps/:email

清除指定客户端的 IP 记录。

### 6.5 在线状态

#### POST /panel/api/inbounds/onlines

获取当前在线的客户端列表。

#### POST /panel/api/inbounds/lastOnline

获取客户端最后在线时间。

---

## 7. 服务器管理 API

### GET /panel/api/server/status

获取服务器状态信息。

**响应示例**：

```json
{
    "success": true,
    "obj": {
        "cpu": 15.5,
        "mem": { "current": 512, "total": 2048 },
        "disk": { "current": 1024, "total": 20480 },
        "xray": { "state": "running", "errorMsg": "" },
        "uptime": 86400,
        "nets": []
    }
}
```

### GET /panel/api/server/cpuHistory/:bucket

获取 CPU 历史数据。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `bucket` | path | int | 采样间隔（秒）：2/30/60/120/180/300 |

### GET /panel/api/server/getXrayVersion

获取可用的 Xray 版本列表。

### GET /panel/api/server/getConfigJson

获取当前 Xray 运行配置 JSON。

### GET /panel/api/server/getDb

下载数据库文件（`x-ui.db`）。

### GET /panel/api/server/getNewUUID

生成新的 UUID。

### GET /panel/api/server/getNewX25519Cert

生成 X25519 密钥对。

### GET /panel/api/server/getNewmldsa65

生成 ML-DSA-65 后量子签名密钥。

### GET /panel/api/server/getNewmlkem768

生成 ML-KEM-768 后量子封装密钥。

### GET /panel/api/server/getNewVlessEnc

生成 VLESS 加密密钥。

### POST /panel/api/server/stopXrayService

停止 Xray 服务。

### POST /panel/api/server/restartXrayService

重启 Xray 服务。

### POST /panel/api/server/installXray/:version

安装指定版本的 Xray。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `version` | path | string | Xray 版本号 |

### POST /panel/api/server/updateGeofile

更新 GeoIP 和 GeoSite 数据文件。

### POST /panel/api/server/updateGeofile/:fileName

更新指定的 Geo 文件。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `fileName` | path | string | 文件名（仅允许 `[a-zA-Z0-9_\-.]`） |

### POST /panel/api/server/logs/:count

获取应用日志。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `count` | path | int | 日志条数 |
| `level` | query | string | 日志级别过滤 |
| `syslog` | query | bool | 是否包含系统日志 |

### POST /panel/api/server/xraylogs/:count

获取 Xray 运行日志。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `count` | path | int | 日志条数 |
| `filter` | query | string | 日志过滤关键词 |

### POST /panel/api/server/importDB

导入数据库文件。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `db` | file | 是 | 数据库文件（multipart upload） |

### POST /panel/api/server/getNewEchCert

生成 ECH（Encrypted Client Hello）证书。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `sni` | string | 是 | 服务器名称 |

---

## 8. 自定义 Geo 资源 API

### GET /panel/api/custom-geo/list

列出所有自定义 Geo 资源。

### GET /panel/api/custom-geo/aliases

获取 Geo 资源别名列表。

### POST /panel/api/custom-geo/add

添加自定义 Geo 资源。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 资源类型（geoip/geosite） |
| `alias` | string | 是 | 资源别名 |
| `url` | string | 是 | 资源下载 URL |

### POST /panel/api/custom-geo/update/:id

更新指定 Geo 资源。

### POST /panel/api/custom-geo/delete/:id

删除指定 Geo 资源。

### POST /panel/api/custom-geo/download/:id

下载/刷新指定 Geo 资源。

### POST /panel/api/custom-geo/update-all

批量更新所有 Geo 资源。

---

## 9. WebSocket 接口

### 连接方式

```
ws://<host>:<port>/ws
```

通过 HTTP Upgrade 建立 WebSocket 连接。

### 消息类型定义

服务端推送的 JSON 消息格式：

```json
{
    "type": "status",
    "data": { ... }
}
```

| type | data 内容 | 触发频率 |
|------|----------|---------|
| `status` | 服务器状态对象（CPU/内存/磁盘/网络） | @every 2s |
| `traffic` | 流量增量数据 | @every 10s |
| `inbounds` | Inbound 列表 | 变更时 |
| `notification` | 通知消息字符串 | 事件触发 |
| `xray_state` | Xray 运行状态 | 状态变化时 |
| `outbounds` | Outbound 流量统计 | 变更时 |
| `invalidate` | 无数据（刷新信号） | 操作触发 |

### 实时数据格式

#### status 消息

```json
{
    "type": "status",
    "data": {
        "cpu": 15.5,
        "mem": { "current": 536870912, "total": 2147483648 },
        "disk": { "current": 1073741824, "total": 21474836480 },
        "xray": { "state": "running" },
        "uptime": 86400,
        "nets": [
            { "name": "eth0", "sent": 1048576, "recv": 2097152 }
        ]
    }
}
```

#### traffic 消息

```json
{
    "type": "traffic",
    "data": {
        "inbounds": [
            { "tag": "inbound-443", "up": 1024, "down": 2048 }
        ],
        "clients": [
            { "email": "user@example.com", "up": 512, "down": 1024 }
        ]
    }
}
```

---

## 10. 订阅服务 API

订阅服务运行在独立端口（默认 `2096`），无需认证。

### GET /sub/:subid

获取 Base64 编码的订阅链接。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `subid` | path | string | 客户端订阅 ID |

**响应**：

- 如果 `subEncrypt=true`：返回 Base64 编码的协议链接列表
- 如果 `subEncrypt=false`：返回 HTML 页面展示

**Base64 解码后内容示例**：

```
vless://uuid@server:443?type=tcp&security=tls&sni=example.com#remark
vmess://base64(json)
trojan://password@server:443?type=tcp&security=tls#remark
```

### GET /json/:subid

获取 JSON 格式的 Xray 客户端配置。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `subid` | path | string | 客户端订阅 ID |

**响应**：完整的 Xray 客户端 JSON 配置，包含 Fragment/Noises/Mux 等高级设置。

### GET /clash/:subid

获取 Clash/Mihomo 格式的 YAML 配置。

| 参数 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `subid` | path | string | 客户端订阅 ID |

**响应**：YAML 格式的 Clash 代理配置。

---

## 11. 其他 API

### POST /panel/api/backuptotgbot

发送数据库备份到 Telegram Bot 管理员（需启用 TG Bot）。该接口会产生副作用，必须使用已登录会话和同源 AJAX 请求。
