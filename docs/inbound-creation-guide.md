# 入站规则创建填写教程

> 目标读者：面板管理员。源码核对日期：2026-10-15，项目版本：`v3.6.0`；当前分支纳入上游 `v3.10.2`，默认运行时 Xray：`v26.10.21`（兼容基线 `v26.10.10`）。
> 事实来源：`frontend/src/views/InboundsView.vue`、`frontend/src/schemas/protocolRegistry.ts`、`frontend/src/utils/inboundCompat.ts`、`frontend/src/utils/realitySettings.ts`、`web/controller/inbound.go`、`web/service/protocol_validation.go`、`database/repository.go`。

打开运行时 `webBasePath + "panel/inbounds"`，点击 `New Inbound`（中文界面对应“添加入站”）。页面使用当前 Vue 表单，下文保留源码中的英文标签便于对照。编辑已有入站时 Protocol 禁用。

## 1. 创建前准备

确认服务器端口、防火墙和安全组已放行，端口没有被系统服务或其他入站占用，客户端支持所选协议和传输。面板保存成功只表示配置已接受；连接可用性还要检查 Xray 运行状态、监听端口和客户端配置。

当前协议选择器包含 VMess、VLESS、Tunnel、HTTP、Trojan、Shadowsocks、Mixed、WireGuard、Tun、Hysteria2 和 Hysteria2 Legacy Alias。Hysteria2 常规保存为 `hysteria` 且 `settings.version=2`，导入别名 `hysteria2` 也可识别，生成 Xray 配置时统一为 `hysteria`。

## 2. Basic Inbound

| 字段 | 填写规则 |
| --- | --- |
| Protocol | 新建时默认 `vless`；编辑时不可改变。 |
| Remark | 用于识别入站，可留空。 |
| Listen | 留空生成 Xray 配置时使用 `0.0.0.0`；仅本机访问填写 `127.0.0.1`。 |
| Port | UI 接受 `1..65535`，默认随机端口。 |
| Traffic Limit GB | UI 按 GiB 换算为 bytes 保存，`0` 表示入站整体不限量；编辑时没有修改配额输入会保留原始 bytes，避免显示精度导致额度变化。 |
| Expiry Timestamp | Unix milliseconds，`0` 表示不过期；当前是数字输入框。 |
| Traffic Reset | 文本输入，默认 `never`；可用 `never/hourly/daily/weekly/monthly`。 |
| Enable | 默认打开；打开后后端尝试同步 Xray，失败时设置需要重启标志。关闭时保存禁用配置。 |

端口冲突判断同时考虑监听地址。通配监听（空、`0.0.0.0`、`::`、`::0`）与任意相同端口冲突；指定地址与同地址或通配监听的相同端口冲突。该判断检查面板数据库，不能替代系统端口检查。

## 3. 协议与 Transport Settings

VMess、VLESS、Trojan、Shadowsocks、Hysteria 支持 Transport Settings。HTTP、Mixed、Tunnel、WireGuard、Tun 不显示该表单。协议注册表中的 TLS 能力标记与表单显示并非完整的 Xray 支持矩阵，组合最终须通过后端和目标核心校验。

| Network | 主要字段 |
| --- | --- |
| `tcp` | TCP Proxy Protocol、TCP Header（`none/http`）。 |
| `ws` | WS Path、WS Host、WS Heartbeat Period、WS Proxy Protocol。 |
| `grpc` | gRPC Service、gRPC Authority、gRPC Multi Mode。 |
| `httpupgrade` | HTTPUpgrade Path、Host、Proxy Protocol。 |
| `xhttp` | Path、Host、Mode、HTTP Header/Padding、Session/Sequence/Data Placement、Session ID Table/Length、XMux；与客户端参数一致。 |
| `kcp` | MTU、TTI、上下行容量、窗口等 mKCP 参数。 |
| `hysteria` | Hysteria 专用 UDP/QUIC 传输，仅 Hysteria 协议显示。 |

普通协议的 Security 下拉包含 `none/tls/reality`；Hysteria 固定为 `hysteria` Network 和 `tls` Security。选项可见不表示每种组合都能在目标核心工作。

结构化区的 `Sync JSON` 从 JSON 读取字段，`Apply` 将字段写回 JSON；提交时也会应用当前结构化字段。Sockopt 包含域名策略、标记、TCP 参数、Dialer Proxy、Interface Name、Trusted X-Forwarded-For 等高级项，按实际部署填写。

### 3.1 TLS / Reality

TLS 使用 Certificate File + Key File，或者 Inline Certificate + Inline Private Key，成对填写真实证书与私钥。两种都填写时，构建逻辑优先采用内联内容。TLS SNI、ALPN、Fingerprint、Min/Max Version、ECH、Reject Unknown SNI、Disable System Root、Session Resumption 都在 Transport Settings 中。

`Generate Self-Signed Certificate` 调用面板证书 API，并填充内联证书和私钥；客户端需要相应的信任配置。Hysteria 新建/提交还可从面板默认 TLS 证书路径补齐空的证书字段；已有字段不会被默认值覆盖。

Reality 填写 Target（`host:port`）、Server Names（逗号分隔）、Private Key 和 Short IDs；Public Key、SpiderX、ML-DSA-65 相关字段用于客户端兼容信息。已登录可调用 `GET /panel/api/server/getNewX25519Cert` 获取密钥；当前入站 Reality 表单没有旧 UI 的 `Get New Cert` 按钮。

Short ID 按 Xray 要求填写合法十六进制，客户端 SNI、公钥和 Short ID 必须与服务端一致。当前 `realitySettings.ts` 仅清理空白/列表、补默认目标和名称，并检查私钥与 Short IDs 非空；目标和名称格式、密钥与十六进制有效性还须由实际核心验收。

目标核心已移除 `tlsSettings.allowInsecure=true`；旧配置必须改为可信证书、`pinnedPeerCertSha256` 或 `verifyPeerCertByName`，不能通过忽略证书错误来继续升级。旧证书 pin 数组由兼容层转换为逗号字符串。Reality 不能与 TCP finalmask 共用；冲突会明确报错，不会静默删除安全设置。

### 3.2 WireGuard / Tun / 其他协议

WireGuard 显示 MTU、noKernelTun、Server Private Key、Server Public Key；`Generate` 生成服务端密钥。Peer 存放于 Settings JSON 的 `peers`，已有入站可在客户端详情中添加、编辑 Peer。新建提交要求服务端密钥和至少一个 Peer。

Peer 使用 Email、Enable、Sub ID、Private/Public Key、Allowed IPs（CIDR）、可选 Pre-shared Key 和 Keep Alive。订阅按 Peer 的 `subId` 匹配；无效 key/CIDR 会使生成失败或 Peer 被跳过。普通订阅生成 WireGuard 配置文本并遵循明文/Base64 设置，JSON 生成 Xray outbound，Clash 生成 Mihomo proxy。

Tun 显示 Interface Name、MTU、Gateway CIDRs、DNS Addresses、User Level、Auto System Route CIDRs、Outbounds Interface。创建前确认目标系统、权限与核心支持 TUN；其他协议的未结构化字段在 Settings JSON 中编辑。

## 4. Default Client

VMess、VLESS、Trojan、Shadowsocks、Hysteria 的客户端通过独立表单管理；WireGuard 使用 Peer，不显示 Default Client。HTTP/Mixed 使用 `settings.accounts`，不显示普通多用户客户端表单。

| 字段 | 填写规则 |
| --- | --- |
| Email | 不能为空，保持唯一，后端重复检查按大小写不敏感比较。 |
| UUID | VMess/VLESS，合法 UUID；`Generate` 重新生成。 |
| Password | Trojan/Shadowsocks，`Generate` 生成对应协议凭证。Shadowsocks 2022 AES method 使用规定长度的 Base64 key。 |
| Auth | Hysteria 客户端认证，不能为空；可点击 `Generate`。 |
| Security | VMess，默认 `auto`。 |
| Flow | Default Client 仅在 VLESS + `tcp` + TLS/Reality 显示；常用 `xtls-rprx-vision`。 |
| Method | Shadowsocks 客户端加密方式，须与服务端兼容。2022 单用户 method 不支持普通 clients 操作。 |
| Traffic Limit GB | 换算为客户端 `totalGB` bytes 保存，`0` 表示不限量。 |
| Expiry Timestamp | Unix milliseconds，`0` 表示不过期。 |
| IP Limit | `0` 表示不限 IP，实际限制还依赖后端 IP 检查链。 |
| Reset Days | 对应客户端 `reset`，到期自动续期天数，`0` 表示不续期。 |
| Sub ID | 默认随机生成，无需等订阅开关开启才显示。 |
| Enable / Comment | 控制客户端启用状态与备注。 |

客户端独立编辑弹窗的 Flow 对所有 VLESS 入站可见，后端仍校验组合。当前表单没有旧版 Telegram ChatID 或延迟启动开关；新客户端 payload 的 `tgId` 固定为 `0`。

客户端记录从高级 Settings JSON 中隔离；提交时恢复 `settings.clients`，避免表单和 JSON 同时编辑同一数组。`Reset` 从已保护的记录加载，`Apply` 更新当前默认客户端；已有其他客户端保留。

## 5. Advanced JSON

页面提供 Settings JSON、Stream Settings JSON、Sniffing JSON 的编辑与 Format。必须填写 JSON 对象；有独立客户端表单的协议不能在 Settings JSON 中手工加入 `clients`，否则提示 `Manage clients with the client form instead of Settings JSON`。

VLESS decryption/encryption、Trojan fallbacks、Shadowsocks 服务端 method/network 等未结构化协议项应在 Settings JSON 中编辑。当前没有独立的旧版 VLESS Authentication、Fallbacks 或 Vision Seed 面板。Sniffing 使用 JSON，支持项必须符合实际 Xray schema。

### 5.1 旧配置兼容与高级字段

后端保存/读取、数据库升级与配置生成使用 `util/xraycompat`；前端编辑器也兼容相应旧字段。未知 JSON 字段按兼容规则保留，不代表目标核心一定支持：

- `streamSettings.method` 在没有 `network` 时转换为 `network`；Reality `dest` 转为 `target`；sockopt `V6Only` 转为 `v6only`，已有新字段优先。
- XHTTP `sessionPlacement/sessionKey` 转为 `sessionIDPlacement/sessionIDKey`，旧 `extra` 中可提升的字段转换到目标结构；Session ID Length 接受正数或正数范围，Table 要求 ASCII。
- Hysteria UDP Hop 从 `finalmask.quicParams.udpHop` 转为 `finalmask.udp` 内 `type=udphop`、`mode=intervalRemote` 和 `remotePorts`；前端原有端口/间隔表单仍可读写。
- xDNS 旧域名/解析器形式转换为目标核心结构；Salamander 密码至少 4 bytes；XMC 需要完整 profiles。无法无损合并的新旧字段或无效结构会报错。
- VMess `none/zero` 规范为 `auto`；WireGuard 废弃 `workers` 删除；Hysteria 缺省 version 补 `2`，显式非 2 拒绝。Shadowsocks `none/plain` 已废弃，需要管理员和客户端共同更换 AEAD 凭证。

数据库迁移失败会回滚事务。入站表单保存成功后仍需检查动态添加结果或重启核心；Xray 模板保存和重启会做完整配置预检，实际监听与客户端握手仍须单独验证。详细规则与回归依据见 [上游集成记录](upstream-3x-ui-v3.10.2-integration.md)。

## 6. 最小创建流程

1. 点击 New Inbound，保持默认 VLESS、随机可用端口、Enable。
2. 填 Remark 和唯一 Default Client Email，保持已生成 UUID。
3. 验证保存链路时使用 `tcp + none`，流量和到期为 `0`；实际交付按部署需要配置 TLS/Reality。
4. 点击弹窗确认按钮。检查列表、详情和 Xray result，必要时在 Xray 页重启。
5. 导出分享链接/订阅链接，从实际客户端检查连接、日志和流量。

VLESS + Reality + Vision 使用 `tcp`、`reality`，填写 Reality 参数后选择 Vision Flow。VLESS + WebSocket + TLS 使用 `ws`、`tls`、相同 Path/Host/SNI 和真实证书，Flow 留空。Trojan + TLS 填正确 Password 与证书。Shadowsocks 在 Settings JSON 中确认服务端 method，并使用匹配凭证。Hysteria2 要求 `settings.version=2`、`hysteria + tls`、可用证书和 Auth，同时放行 UDP。

## 7. 常见报错与创建后检查

| 错误 | 检查方向 |
| --- | --- |
| `Port already exists` | 核对监听地址和其他入站端口，另行检查系统端口占用。 |
| `Duplicate email` | 换用唯一 email，包括仅大小写不同的重复值。 |
| `empty client ID` | 核对 UUID、Trojan Password、Shadowsocks Email、Hysteria Auth；协议校验还会检查所需凭证。 |
| `uuid is invalid` / `flow requires tcp with tls or reality` | UUID 格式和 Flow 的传输/安全组合。 |
| `shadowsocks server key` / `shadowsocks client key` | 服务端和客户端 key 的 Base64 编码与长度。 |
| `Hysteria2 requires TLS certificate file paths or inline certificate content` | 证书和私钥都要齐全。 |
| `WireGuard requires at least one peer` | 确保 Settings JSON 中有 Peer，并核对 key/CIDR。 |
| `json: cannot unmarshal string into Go value of type []model.Client` | 当前 `parseInboundClients` 只解析 `settings.clients`；检查运行二进制版本和导入结构，`clients` 应是数组。 |

手工调用 API 或 Import JSON 时，外层 `settings/streamSettings/sniffing` 是 JSON 字符串，`settings.clients` 内层是数组。导入 `data` 是单个 Inbound 对象，不是数组。完整接口见 [API 接口说明](api.md)。

确认列表字段与实际监听一致，凭证和订阅 ID 正确；TLS/Reality/Path/Host/SNI/Short ID 与客户端一致；订阅格式在 Settings 中启用且实际可拉取。检查日志、流量和出口。入站仍写入 `database/model.Inbound`，Core Instances 页面不会接管该 Xray 生命周期。
