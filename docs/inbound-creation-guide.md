# 入站规则创建填写教程

> 目标读者：面板管理员
> 适用场景：在 Web 面板的“入站列表”中点击“添加入站”，创建新的 Xray 入站规则。

---

## 1. 创建前准备

创建入站前先确认三件事：

1. 服务器防火墙、安全组或云厂商端口已放行。
2. 计划使用的端口没有被其他入站或系统服务占用。
3. 已确定协议、传输方式和安全方式，例如 `VLESS + TCP + Reality`、`VLESS + WebSocket + TLS`、`Trojan + TLS`、`Shadowsocks`。

如果只是验证面板是否能创建入站，建议先用默认 `vless`，只修改备注、端口和客户端 Email，其他高级项保持默认。确认可以保存后，再逐步开启 Reality、TLS、WebSocket、Sniffing、IP 限制等高级功能。

---

## 2. 按页面顺序填写

### 2.1 基础信息

| 字段 | 是否必填 | 填写规则 |
|------|----------|----------|
| 启用 | 否 | 打开后创建成功会尝试同步到正在运行的 Xray；关闭后只保存配置。 |
| 备注 | 建议填写 | 用于在入站列表中识别规则，例如 `vless-reality-443`。 |
| 协议 | 必填 | 创建后一般不能直接改协议；选错建议删除后重新创建。 |
| 监听 IP | 可留空 | 留空或 `0.0.0.0` 表示监听所有 IPv4 地址；只想本机访问可填 `127.0.0.1`。 |
| 端口 | 必填 | 范围 `1-65535`，同一监听地址下必须唯一。 |
| 总流量 | 可为 `0` | `0` 表示入站整体不限流量；客户端仍可单独限流。 |
| 周期流量重置 | 可保持默认 | `never` 表示不自动重置；也可选择 hourly/daily/weekly/monthly。 |
| 到期时间 | 可留空 | 留空表示入站永不过期。 |

常见错误：

- `Port already exists`：端口已被其他入站占用，换端口或删除旧入站。
- 端口填 `443`、`8443`、`2053` 等常用端口前，先确认系统服务和反向代理没有占用。

### 2.2 连接方式

这一组决定客户端能否看到某些后续字段，尤其是 VLESS 的 Flow。

| 字段 | 填写规则 |
|------|----------|
| 传输 | 常见值是 `TCP (RAW)`、`WebSocket`、`gRPC`、`HTTPUpgrade`、`XHTTP`、`mKCP`。服务端和客户端必须一致。 |
| 安全 | 可选 `none`、`Reality`、`TLS`。Reality 只在 VLESS/Trojan 且部分传输方式下显示。 |

Flow 的真实显示条件：

1. 协议必须是 `vless`。
2. 传输必须是 `TCP (RAW)`。
3. 安全必须选择 `Reality` 或 `TLS`。

只有三项同时满足时，客户端区域才会显示 `Flow` 下拉框。它不是按钮，也不会在 WebSocket、gRPC、XHTTP 等传输下显示。

### 2.3 客户端

多用户协议会在“添加入站”时默认创建一个客户端。常见多用户协议包括 VMess、VLESS、Trojan、Shadowsocks 多用户模式和 Hysteria。WireGuard 使用 `Peer` 管理客户端，字段位置不同，但订阅识别也依赖 Email、启用状态和订阅 ID。

| 字段 | 适用协议 | 填写规则 |
|------|----------|----------|
| 启用 | 全部客户端协议 | 关闭后该客户端不会生效。 |
| Email | 多用户协议 | 必填，建议全局唯一；可用字母、数字、点和短横线。 |
| Email | WireGuard Peer | 用于订阅备注和后续识别；建议填写唯一值。 |
| ID | VMess / VLESS | 必须是 UUID。点击同步图标可自动生成。 |
| Password | Trojan / Shadowsocks | 必填。点击同步图标可自动生成。 |
| Auth Password | Hysteria | 必填。点击同步图标可自动生成。 |
| PrivateKey / PublicKey | WireGuard Peer | 必须是有效 WireGuard key；点击同步图标可自动生成。 |
| Security | VMess | 通常保持 `auto`。 |
| Subscription / Sub ID | 开启订阅后显示、WireGuard Peer | 订阅 ID，留默认随机值即可；WireGuard 订阅按 Peer 的 `subId` 匹配。 |
| Telegram ChatID | 开启 Telegram Bot 后显示 | 只能填数字；不知道就留空或填 `0`。 |
| Comment | 可选 | 客户端备注。 |
| IP Limit | 开启 IP 限制后显示 | `0` 表示不限 IP 数量。 |
| Flow | 仅 VLESS + TCP + TLS/Reality 显示 | Reality/Vision 常用 `xtls-rprx-vision`；不确定时先选 `none`。 |
| 客户端总流量 | 可为 `0` | `0` 表示该客户端不限流量。 |
| 延迟启动 | 可选 | 打开后按“天数”计算首次使用后的到期时间。 |
| 到期时间 | 可留空 | 留空表示该客户端永不过期。 |
| 续期天数 | 设置到期时间后显示 | 到期后自动续期的天数，`0` 表示不自动续期。 |

关键规则：

- VMess/VLESS 的 `ID` 必须是合法 UUID。
- VLESS 的 `Flow` 只能用于 TCP + TLS/Reality。
- Trojan/Shadowsocks 的 `Password` 不能为空。
- Shadowsocks 2022 method 的服务端 key 和客户端 key 必须是匹配算法长度的 Base64。
- Hysteria 的 `Auth Password` 不能为空。
- WireGuard Peer 的 key 和 Allowed IPs 必须有效，否则订阅导出会跳过该 Peer。
- Email 不要重复；重复会影响流量统计、订阅和客户端检索。
- Telegram ChatID 必须是数字，不要填文字、空格或带引号的内容。

### 2.4 协议高级

不同协议会显示不同高级项：

| 区域 | 说明 |
|------|------|
| VLESS Authentication / decryption / encryption | 这是 VLESS 协议级认证/加密字段，不是 Reality/TLS 的“安全”。普通 VLESS + Reality/TLS 场景通常保持默认。 |
| Fallbacks | TCP 且未启用 VLESS Authentication 时显示；用于把不匹配流量转发到其他服务。不会配置时不要添加。 |
| Vision Seed | 选择 VLESS Flow 为 `xtls-rprx-vision` 或 `xtls-rprx-vision-udp443` 后显示；不理解用途时保持默认。 |
| Shadowsocks Method | 需要和客户端支持的加密方式一致。 |
| Hysteria 参数 | 偏 UDP/QUIC 场景，需确认运营商和防火墙允许 UDP。 |

### 2.5 传输高级

| 传输 | 重点字段 |
|------|----------|
| TCP (RAW) | 通常不需要额外路径；HTTP 伪装只在明确需要时配置。 |
| WebSocket | Path 必须以 `/` 开头；Host 通常填域名。 |
| gRPC | Service Name 要和客户端一致。 |
| HTTPUpgrade / XHTTP | Path、Host、Header 要和客户端一致。 |
| mKCP | 网络环境敏感；如果运营商限制 UDP，创建成功也可能无法连接。 |
| Sockopt / TCP Masks / External Proxy | 都属于高级网络选项；没有明确部署需求时保持默认。 |

填写传输字段时，最重要的是“服务端和客户端一致”。Path、Host、SNI、Service Name、Reality Short ID 中任意一项不一致，都可能导致入站创建成功但客户端无法连接。

### 2.6 TLS / Reality 高级

| 安全方式 | 填写规则 |
|----------|----------|
| TLS | SNI 要匹配域名；证书路径或证书内容必须真实可用；私钥和证书必须成对。 |
| Reality | Target、SNI、Private Key、Public Key、Short IDs 必须成组匹配。点击 `Get New Cert` 可生成 X25519 密钥。 |

Reality 常见填写顺序：

1. 安全选择 `Reality`。
2. Target 填目标站点，例如 `www.microsoft.com:443`。
3. SNI 填目标域名，例如 `www.microsoft.com`。
4. 点击 `Get New Cert` 生成私钥和公钥。
5. Short IDs 保持随机值或点击同步图标重新生成。
6. 如果需要 Vision，再回到客户端区域把 Flow 选为 `xtls-rprx-vision`。

### 2.7 Sniffing

Sniffing 可保持默认。确认入站能创建并能连接后，再按需启用 HTTP/TLS/QUIC 识别、routeOnly、metadataOnly、排除域名或排除 IP。

---

## 3. 推荐最小创建流程

### 3.1 新手验证流程

1. 打开“入站列表”。
2. 点击“添加入站”。
3. 协议保持默认 `vless`。
4. 备注填写 `test-vless`。
5. 端口填写一个未占用端口，例如 `8443`。
6. 传输保持 `TCP (RAW)`。
7. 安全先保持 `none`。
8. 客户端 Email 填写 `test@example`。
9. ID 保持默认自动生成。
10. Telegram ChatID 留空或填 `0`。
11. 客户端总流量保持 `0`。
12. 到期时间留空。
13. 点击“创建”。

如果这一步成功，再逐步开启 TLS、Reality、WebSocket、IP 限制、订阅、流量重置等高级功能。不要一次性打开很多高级项，否则排查会困难。

### 3.2 VLESS + Reality + Vision

1. 协议选择 `vless`。
2. 端口填写已放行端口，例如 `443`。
3. 传输选择 `TCP (RAW)`。
4. 安全选择 `Reality`。
5. 客户端 Email 填唯一值，ID 保持自动生成。
6. 客户端 Flow 选择 `xtls-rprx-vision`。
7. Reality 区域填写 Target、SNI，点击 `Get New Cert` 生成密钥。
8. Short IDs 保持随机值或重新生成。
9. Sniffing 先保持默认。

### 3.3 VLESS + TLS + WebSocket

1. 协议选择 `vless`。
2. 传输选择 `WebSocket`。
3. 安全选择 `TLS`。
4. 客户端 Email 填唯一值，ID 保持自动生成。
5. WS Path 填写以 `/` 开头的路径，例如 `/vless`。
6. Host 填写域名，例如 `example.com`。
7. TLS SNI 填写同一个域名。
8. 证书路径填服务器上真实存在的证书文件和私钥文件。

注意：WebSocket 场景不会显示 Flow，这是符合规则的。

### 3.4 Trojan + TLS

1. 协议选择 `trojan`。
2. 端口填写已放行端口。
3. 安全选择 `TLS`。
4. 客户端 Password 保持自动生成或填写强密码。
5. SNI 和证书路径按域名证书填写。
6. 不确定 Fallbacks 时先不要添加。

### 3.5 Shadowsocks

1. 协议选择 `shadowsocks`。
2. Method 保持默认或选择客户端支持的加密方式。
3. Password 使用随机生成值。
4. Network 一般保持 `tcp,udp`。
5. 如果开启多用户模式，每个客户端必须有独立 Email 和 Password。
6. 如果选择 `2022-*` method，不要手工随意改短 Password；它是 Base64 key，长度必须匹配算法。

### 3.6 WireGuard

1. 协议选择 `wireguard`。
2. SecretKey / PublicKey 使用同步图标生成。
3. 每个 Peer 填写 Email、启用状态和 Sub ID；订阅服务会按 Peer 的 Sub ID 导出。
4. Peer PrivateKey / PublicKey 使用同步图标生成。
5. Allowed IPs 至少保留一个有效 CIDR，例如 `10.0.0.2/32`。
6. 开启订阅后，普通订阅会返回 WireGuard `.conf`；JSON 订阅会返回 Xray WireGuard outbound；Clash 订阅会返回 Mihomo WireGuard proxy。

---

## 4. 常见报错与判断

### `Port already exists`

端口已被当前面板中的其他入站使用，或同一监听地址下已有相同端口。换端口或删除旧入站。

### `Duplicate email`

客户端 Email 已存在。每个客户端 Email 建议全局唯一。

### `empty client ID`

当前协议的客户端关键字段为空：

- VMess/VLESS：检查 ID。
- Trojan/Shadowsocks：检查 Password。
- Hysteria：检查 Auth Password。

### `uuid is invalid` / `flow requires tcp with tls or reality`

VMess/VLESS 客户端 ID 不是合法 UUID，或 VLESS Flow 用在了 WebSocket、gRPC、XHTTP、无 TLS/Reality 等不支持组合。重新生成 ID，或把传输切回 TCP 并启用 TLS/Reality。

### `shadowsocks server key` / `shadowsocks client key`

Shadowsocks 2022 method 的服务端 Password 或客户端 Password 不是匹配长度的 Base64 key。点击同步图标重新生成，不要把普通短密码填进 2022 method。

### `json: cannot unmarshal string into Go value of type []model.Client`

如果是在标准“添加入站”页面填写后出现这个错误，高概率是旧版本后端解析 Bug，不是 Flow 没填，也不是普通填写错误。

原因是旧后端把整个 `settings` 都按 `[]model.Client` 解析，但 VLESS settings 中除了 `clients` 数组，还会有 `decryption`、`encryption` 等字符串字段，解析到这些字符串时就会报错。

正确后端行为是只解析 `settings.clients`：

```json
{
  "clients": [
    {
      "id": "00000000-0000-4000-8000-000000000000",
      "email": "test@example",
      "enable": true,
      "tgId": 0
    }
  ],
  "decryption": "none",
  "encryption": "none"
}
```

如果仍复现：

1. 确认后端已经更新到修复版本。
2. 强制刷新浏览器页面，避免旧静态资源缓存。
3. 打开浏览器开发者工具，检查 `/panel/api/inbounds/add` 请求中的 `settings` 字段。
4. 如果是手工导入 JSON，确认 `clients` 是数组，不是带引号的 JSON 字符串。

---

## 5. 创建后检查清单

- 入站列表能看到新规则。
- 端口、协议、备注和启用状态符合预期。
- 客户端 Email、ID/Password/Auth 不为空。
- 如果开启 TLS 或 Reality，客户端链接中的 SNI、Host、Path、Short ID 与服务端一致。
- 如果开启订阅，订阅链接能拉取到该客户端。
- 如果启用后无法连接，先检查防火墙、端口监听、Xray 日志，再检查客户端配置。
