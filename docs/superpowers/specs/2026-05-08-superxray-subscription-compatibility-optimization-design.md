# SuperXray 订阅兼容性优化设计方案

## 背景

SuperXray-gui 当前订阅能力已经覆盖 Base64 / Plain URI、Xray JSON、Clash/Mihomo YAML 与 WireGuard 配置输出。经前置梳理确认，后端入站规则支持 VMess、VLESS、Trojan、Shadowsocks、Hysteria、Hysteria2、WireGuard、Tunnel、HTTP、Mixed、Tun；其中只有可作为客户端节点或 peer 配置导入的协议进入订阅输出。

现有主要风险集中在订阅生成稳定性、跨客户端格式兼容、Host 来源可信边界、错误可诊断性和大量客户端场景性能。该方案用于指导后续按阶段实现，保持现有订阅路径兼容，同时逐步建立标准化、可测试、可诊断的订阅输出体系。

## 目标

- 为 Windows、macOS、iOS、Android、Linux/Headless 等平台提供明确的订阅入口推荐。
- 标准化 URI、Xray JSON、Clash/Mihomo YAML、WireGuard 配置输出。
- 修复已知稳定性问题，避免畸形配置导致 panic 或整份订阅失败。
- 降低 Host Header 污染、共享请求态串号、JSON map alias 等安全和一致性风险。
- 提供订阅诊断能力，让管理员了解节点被跳过或降级的原因。
- 为后续客户端 profile、缓存、ETag、性能优化和 UI 向导打基础。

## 非目标

- 不把 Tunnel、HTTP、Mixed、Tun 强行加入客户端订阅输出。
- 不破坏已有 `/sub/<subid>`、`/json/<subid>`、`/clash/<subid>` 路径。
- 不在第一阶段修改数据库模型。
- 不在第一阶段引入大型第三方依赖。
- 不牺牲标准格式来迁就单一客户端私有行为。

## 当前能力矩阵

| 协议 | 入站配置 | Base64 / URI | Xray JSON | Clash/Mihomo | WireGuard 配置 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| VMess | 支持 | 支持 | 支持 | 支持 | 不适用 | 主路径完整 |
| VLESS | 支持 | 支持 | 支持 | 支持 | 不适用 | Reality/XTLS 依赖客户端能力 |
| Trojan | 支持 | 支持 | 支持 | 支持 | 不适用 | TLS/SNI/ALPN 需正确配置 |
| Shadowsocks | 支持 | 支持 | 支持 | 支持 | 不适用 | 加密方法需与客户端兼容 |
| Hysteria | 支持 | 支持 | 支持 | 支持 | 不适用 | 需要 TLS 与客户端 v1 支持 |
| Hysteria2 | 支持 | 支持 | 支持 | 支持 | 不适用 | 需要 TLS 与客户端 v2 支持 |
| WireGuard | 支持 | 独立配置文本 | 支持 | 支持 | 支持 | 使用 peer 语义 |
| Tunnel | 支持 | 不支持 | 不支持 | 不支持 | 不适用 | 入站规则，不是订阅节点 |
| HTTP | 支持 | 不支持 | 不支持 | 不支持 | 不适用 | 本地/服务端代理入站 |
| Mixed | 支持 | 不支持 | 不支持 | 不支持 | 不适用 | HTTP + SOCKS 混合入站 |
| Tun | 支持 | 不支持 | 不支持 | 不支持 | 不适用 | 透明代理/系统路由能力 |

## 平台与客户端适配策略

### Windows

| 客户端 | 推荐入口 | 推荐格式 | 重点协议 |
| --- | --- | --- | --- |
| v2rayN | `/sub/<subid>` | Base64 / URI | VMess、VLESS、Trojan、Shadowsocks、Hysteria2 |
| NekoRay / NekoBox | `/sub/<subid>` 或 `/json/<subid>` | URI / Xray JSON | VLESS Reality、Hysteria2、Trojan |
| Clash Verge / Mihomo GUI | `/clash/<subid>` | Clash/Mihomo YAML | VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard |
| WireGuard 官方客户端 | WireGuard 导出 | `.conf` | WireGuard |

Windows 默认推荐 v2rayN 与 Mihomo 系两条路径。Reality 参数需要完整输出 `pbk`、`sid`、`sni`、`fp`、`flow`、`spx`。WireGuard 不复用普通代理分享语义，应提供独立配置复制、下载或二维码。

### macOS

| 客户端 | 推荐入口 | 推荐格式 | 重点协议 |
| --- | --- | --- | --- |
| ClashX Meta / Mihomo Party | `/clash/<subid>` | Clash/Mihomo YAML | VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard |
| NekoRay / NekoBox | `/sub/<subid>` 或 `/json/<subid>` | URI / Xray JSON | VLESS Reality、Hysteria2 |
| V2rayU / V2rayXS | `/sub/<subid>` | Base64 / URI | VMess、VLESS、Trojan |
| WireGuard 官方客户端 | WireGuard 导出 | `.conf` | WireGuard |

macOS 用户应优先展示 Clash/Mihomo 链接，同时提示当前 Clash 输出普通代理协议只稳定覆盖 TCP、WebSocket、gRPC。

### iOS

| 客户端 | 推荐入口 | 推荐格式 | 重点协议 |
| --- | --- | --- | --- |
| Shadowrocket | `/sub/<subid>` | Base64 / URI | VMess、VLESS、Trojan、Shadowsocks、Hysteria2 |
| Stash | `/clash/<subid>` | Clash/Mihomo YAML | VLESS、Trojan、Shadowsocks、Hysteria2 |
| Loon | 后续扩展 | Loon 配置 | Trojan、Shadowsocks、VMess |
| Quantumult X | 后续扩展 | QuanX 配置 | Trojan、Shadowsocks、VMess |
| WireGuard 官方客户端 | WireGuard 导出 | `.conf` / QR | WireGuard |

iOS 是兼容性敏感平台。URI 参数必须稳定排序、严格 URL encode、过滤空参数。复杂 Reality/Hysteria2 节点应提示优先使用支持度更好的客户端或 Clash/Mihomo 配置。

### Android

| 客户端 | 推荐入口 | 推荐格式 | 重点协议 |
| --- | --- | --- | --- |
| v2rayNG | `/sub/<subid>` | Base64 / URI | VMess、VLESS、Trojan、Shadowsocks |
| NekoBox | `/sub/<subid>` 或 `/json/<subid>` | URI / Xray JSON | VLESS Reality、Hysteria2 |
| Clash Meta for Android | `/clash/<subid>` | Clash/Mihomo YAML | VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard |
| Hiddify Next | `/sub/<subid>` 或 `/clash/<subid>` | URI / Clash | VLESS、Hysteria2、Trojan |
| WireGuard 官方客户端 | WireGuard 导出 | `.conf` / QR | WireGuard |

Android 应按客户端类型推荐，而不是只按系统推荐。NekoBox 与 Hiddify 适合作为 Reality / Hysteria2 新协议推荐客户端。

### Linux / Headless / 路由器

| 客户端 | 推荐入口 | 推荐格式 |
| --- | --- | --- |
| xray-core | `/json/<subid>` | Xray JSON |
| mihomo | `/clash/<subid>` | Clash/Mihomo YAML |
| sing-box | 后续扩展 `/singbox/<subid>` | sing-box JSON |
| WireGuard | WireGuard 导出 | `.conf` |
| OpenWrt 插件 | `/clash/<subid>` 或 `/sub/<subid>` | Clash / URI |

Linux/Headless 场景需要更强的 HTTP 缓存、稳定 JSON/YAML 输出和可诊断提示，后续可增加 sing-box 输出。

## 链接格式标准化

### 统一中间模型

后续应建立 `NormalizedNode` 作为订阅输出中间层：

```text
Inbound + Client / Peer
        ↓
NormalizedNode
        ↓
URI Encoder / Xray JSON Encoder / Clash Encoder / WireGuard Encoder
```

`NormalizedNode` 应包含协议、名称、地址、端口、用户身份、传输层、安全层、TLS/Reality、Hysteria、WireGuard、客户端能力、warning 列表等字段。

### URI 规则

- 所有 query 参数必须 URL encode。
- fragment/remark 必须 URL encode。
- 空参数不输出。
- 参数顺序固定，便于快照测试。
- Host 优先使用 canonical origin，不能无条件信任请求 Host。
- IPv6 使用标准括号格式。

### Xray JSON 规则

- 每个 outbound 使用统一 schema。
- `streamSettings` 必须深拷贝，禁止 externalProxy 复用 map。
- 单节点字段缺失时跳过节点并记录 warning，不 panic。
- WireGuard peer 输出与普通 client 输出分开处理。

### Clash/Mihomo YAML 规则

- 使用 Mihomo 兼容字段。
- 节点名必须全局唯一。
- 不支持的传输需要过滤并记录 warning。
- 普通代理协议当前优先稳定支持 TCP、WebSocket、gRPC。
- Hysteria/Hysteria2/WireGuard 使用独立构建路径。

## 错误处理设计

| 等级 | 示例 | 行为 |
| --- | --- | --- |
| Fatal | subId 不存在、订阅服务关闭 | 返回 404/403 或现有兼容错误 |
| NodeSkip | 单个节点字段缺失 | 跳过节点，继续生成 |
| FormatUnsupported | Clash 不支持某传输 | 过滤节点，记录 warning |
| SoftWarning | 客户端可能不支持 Reality/Hysteria | 输出节点，诊断中提示 |

第一阶段应先保证订阅生成器不因单个畸形节点 panic。后续增加诊断接口：

```text
/sub/<subid>/diagnose
/json/<subid>/diagnose
/clash/<subid>/diagnose
```

诊断结果包含总入站数、输出节点数、跳过节点数、warning 列表与跳过原因。

## 性能优化设计

- 入口处一次性解析 inbound settings。
- 构建 `email -> client`、`subId -> client`、`email -> traffic`、`peerSubId -> peer` 索引。
- 避免协议生成函数重复 JSON parse 和线性查找。
- 后续引入短 TTL 缓存，缓存 key 包含 subId、格式、canonical origin、客户端 profile、订阅设置版本。
- 支持 `ETag`、`Last-Modified`、`Cache-Control`、`Subscription-Userinfo`、`Profile-Update-Interval`。

## UI/UX 改进设计

- 在客户端详情中提供“选择客户端”导入向导。
- 按平台推荐 Windows、macOS、iOS、Android、Linux/Headless 链接。
- Inbounds 列表展示 `URI`、`Xray JSON`、`Clash`、`WireGuard`、`No Subscription`、`Clash Limited` 等能力标签。
- WireGuard 使用 peer 语言，提供复制配置、下载 `.conf`、二维码。
- 空订阅时在浏览器/诊断视图展示可读原因。

## 分阶段实施计划

### 阶段 1：订阅安全与稳定性修复

- 修复 SubService 共享请求态，避免并发请求串号。
- 修复 Host Header / X-Forwarded-Host 可信边界。
- 修复 unchecked type assertion 与未检查数组下标。
- 修复 JSON externalProxy map alias。
- 补充畸形配置、externalProxy、并发订阅测试。

### 阶段 2：统一订阅中间模型

- 定义 `NormalizedNode`。
- 先迁移 VMess、VLESS、Trojan、Shadowsocks。
- 再迁移 Hysteria/Hysteria2。
- 最后迁移 WireGuard。
- 保持现有接口路径兼容。

### 阶段 3：客户端 Profile 适配

- 定义 `generic`、`v2rayn`、`shadowrocket`、`stash`、`mihomo`、`xray`、`wireguard` profile。
- 支持 `?target=` 参数，但默认行为保持兼容。
- UI 根据平台和客户端展示推荐链接。

### 阶段 4：诊断接口与 UI 提示

- 增加订阅诊断接口。
- 增加“测试订阅生成”按钮。
- 增加协议能力标签。
- 空订阅输出可读原因。

### 阶段 5：缓存与性能优化

- 降低重复 JSON parse。
- 增加短 TTL 缓存。
- 支持 ETag / Last-Modified。
- 增加大客户端数 benchmark。

## 测试策略

- URI 快照测试：验证参数顺序、URL encode、空参数过滤。
- JSON schema 测试：验证 Xray outbound 结构稳定。
- Clash YAML 快照测试：验证 Mihomo 输出稳定。
- 畸形配置测试：验证不 panic，只跳过单节点。
- externalProxy 测试：验证不会 map alias 串改。
- WireGuard peer 测试：验证 peer 字段完整。
- 并发测试：验证不同 Host 不串号。
- benchmark：验证大量 client 场景订阅生成耗时。

## 效果评估指标

| 类别 | 指标 | 目标 |
| --- | --- | --- |
| 稳定性 | 订阅接口 5xx | 明显下降 |
| 稳定性 | panic/recover 日志 | 目标为 0 |
| 兼容性 | v2rayN 导入成功率 | ≥ 95% |
| 兼容性 | Shadowrocket 导入成功率 | ≥ 90% |
| 兼容性 | Clash/Mihomo 导入成功率 | ≥ 95% |
| 兼容性 | WireGuard 导入成功率 | ≥ 95% |
| 性能 | 100 节点订阅生成耗时 | 本地基准 < 100ms |
| 性能 | 1000 节点订阅生成耗时 | 本地基准 < 500ms |
| 体验 | 空订阅可诊断比例 | 明显提升 |
| 体验 | 错误格式链接复制次数 | 明显下降 |

## 第一阶段落地优先级

第一阶段按风险和改动可控性排序：

1. 修复 JSON externalProxy map alias，并补充测试。
2. 增加 safe getter，处理 externalProxy 字段缺失和类型错误，避免 panic。
3. 逐步消除 SubService `address` / `datepicker` 请求态共享，先从链接生成路径开始。
4. 收敛 Host 来源，优先支持配置 canonical URI，严格校验请求派生 Host。
5. 增加订阅输出矩阵回归测试，避免破坏现有客户端路径。
