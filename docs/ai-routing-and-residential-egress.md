# AI 平台智能分流与住宅出口运行手册

> **目标读者**：需要在 SuperXray 中为 OpenAI/ChatGPT、Anthropic/Claude、Google/Gemini 等 AI 平台配置专用出口的运维人员
> **适用版本**：`v3.0.14`
> **相关文档**：[部署指南](deployment.md) | [系统架构设计](architecture.md) | [入站创建教程](inbound-creation-guide.md)

---

## 1. 目标与边界

本文档描述一套基于 Xray 原生能力的域名级智能分流方案：

- AI 平台相关域名通过专用住宅 SOCKS 出站访问。
- 非 AI 平台流量保持 `direct` 或既有常规线路，不占用住宅出口资源。
- 住宅出口通过 `observatory` 做健康探测，并由 `leastPing` Balancer 选择可用出口。
- AI 规则命中但住宅出口不可用时，默认 `fallbackTag=blocked`，避免自动降级到非住宅出口。
- DNS 对 AI 域名使用显式 DoH 解析和 `skipFallback`，降低 DNS 污染和错误回退风险。

重要边界：

- 本方案是“合规可控的域名级分流与失败保护”，不承诺规避平台风控，也不承诺账号绝对安全。
- WebRTC 泄漏主要发生在终端浏览器或客户端侧；面板侧可约束 Xray DNS 和路由，但不能替代终端浏览器隐私配置。
- AI 平台域名会持续变化，域名清单需要按日志和实际访问定期维护。
- 文档示例必须使用占位符，不应写入真实服务器 IP、订阅 ID、UUID、私钥、公钥或住宅代理密码。

---

## 2. 推荐拓扑

```text
客户端
  -> VLESS + TCP + Reality 入站（例如 inbound-443）
    -> Xray 路由规则
      -> AI 域名：ai-residential-balancer
          -> us-residential-socks（高纯净度住宅 SOCKS 出口）
          -> blocked（住宅出口不可用时失败关闭）
      -> 其他流量：direct / 既有常规线路
```

推荐先保证 VLESS Reality 入站自身稳定，再叠加分流：

| 项目 | 推荐值 |
|------|--------|
| 入站协议 | `vless` |
| 传输 | `tcp` |
| 安全层 | `reality` |
| 入站 tag | `inbound-443` 或其他稳定 tag |
| Reality target | `<REALITY_TARGET_HOST>:443` |
| Reality serverNames | 至少包含 `<REALITY_TARGET_HOST>` |
| Sniffing | 建议开启 `http`、`tls`、`quic`，必要时包含 `fakedns` |

如果日志出现 `please fill in a valid value for "target"`，说明 Reality `target` 为空或无效，需要先修复入站配置并重启 Xray。

---

## 3. 出站配置

在 **Xray → Outbounds** 或 Xray JSON 中新增住宅 SOCKS 出站。示例：

```json
{
  "tag": "us-residential-socks",
  "protocol": "socks",
  "settings": {
    "servers": [
      {
        "address": "<RESIDENTIAL_SOCKS_HOST>",
        "port": 1086,
        "users": [
          {
            "user": "<SOCKS_USERNAME>",
            "pass": "<SOCKS_PASSWORD>"
          }
        ]
      }
    ]
  }
}
```

保留默认 `direct` 和 `blocked` 出站：

```json
[
  {
    "tag": "direct",
    "protocol": "freedom",
    "settings": {
      "domainStrategy": "AsIs"
    }
  },
  {
    "tag": "blocked",
    "protocol": "blackhole",
    "settings": {}
  }
]
```

安全要求：

- `<SOCKS_PASSWORD>` 只写入服务器实际配置，不写入 Git、截图、Issue 或通用文档。
- 不要添加 `inboundTag -> us-residential-socks` 的全量兜底规则，否则所有流量都会占用住宅出口。
- SOCKS 出站连通性应使用面板的 Outbound Test 或临时测试配置验证，不要只依赖客户端主观体感。

---

## 4. AI 域名清单

基础域名清单建议放在 AI 路由规则和 AI DNS server 的 `domains` 中。该清单是可维护基线，不代表覆盖未来新增的全部域名。

### 4.1 OpenAI / ChatGPT

```text
domain:openai.com
domain:chatgpt.com
domain:oaistatic.com
domain:oaiusercontent.com
domain:openaiapi-site.azureedge.net
domain:openaicom-api-bdcpf8c6d2e9atf6.z01.azurefd.net
domain:auth0.openai.com
```

### 4.2 Anthropic / Claude

```text
domain:anthropic.com
domain:claude.ai
domain:claudeusercontent.com
domain:console.anthropic.com
```

### 4.3 Google AI / Gemini

```text
domain:gemini.google.com
domain:aistudio.google.com
domain:ai.google.dev
domain:generativelanguage.googleapis.com
domain:makersuite.google.com
domain:bard.google.com
domain:googleapis.com
domain:googleusercontent.com
domain:gstatic.com
domain:googleusercontent.cn
domain:google.com
domain:googleapis.cn
```

维护建议：

- 如果只希望 Google AI 走住宅出口，不希望所有 Google 搜索、YouTube 或 Gmail 走住宅出口，可以移除 `domain:google.com`、`domain:googleusercontent.com`、`domain:gstatic.com` 等宽域名，再通过日志补充更精确域名。
- 如果客户端使用 QUIC/HTTP3，需确认 sniffing 能识别目标域名；否则可在客户端禁用 QUIC 或使用 DNS/FakeDNS 辅助分流。
- 对平台新增域名，先添加到 DNS server 的 `domains`，再添加到 routing rule，保存后重启 Xray 验证。

---

## 5. 路由、Balancer 与健康检测

### 5.1 路由规则

将 AI 域名绑定到 Balancer：

```json
{
  "type": "field",
  "inboundTag": [
    "inbound-443"
  ],
  "domain": [
    "domain:openai.com",
    "domain:chatgpt.com",
    "domain:oaistatic.com",
    "domain:oaiusercontent.com",
    "domain:anthropic.com",
    "domain:claude.ai",
    "domain:claudeusercontent.com",
    "domain:gemini.google.com",
    "domain:aistudio.google.com",
    "domain:generativelanguage.googleapis.com",
    "domain:ai.google.dev",
    "domain:googleapis.com"
  ],
  "balancerTag": "ai-residential-balancer"
}
```

规则顺序建议：

1. 面板内部 API 规则保持在最前。
2. 私网、BT 阻断规则保留。
3. AI 域名规则放在普通直连或兜底规则之前。
4. 不添加捕获所有域名或所有入站的住宅出口规则。

### 5.2 Balancer

```json
{
  "tag": "ai-residential-balancer",
  "selector": [
    "us-residential-socks"
  ],
  "strategy": {
    "type": "leastPing"
  },
  "fallbackTag": "blocked"
}
```

`fallbackTag=blocked` 是失败关闭策略：当住宅出口不可用或探测失败时，AI 平台请求不会自动落到 `direct`，从而避免出口身份漂移。若业务更重视可用性，可把 fallback 改为常规代理 tag，但必须接受出口变化带来的风控风险。

### 5.3 Observatory

```json
{
  "subjectSelector": [
    "us-residential-socks"
  ],
  "probeURL": "https://www.google.com/generate_204",
  "probeInterval": "1m",
  "enableConcurrency": true
}
```

说明：

- `probeURL` 用于出站健康探测，不代表所有 AI 平台的真实延迟。
- 如果住宅供应商对并发或探测频率敏感，可把 `probeInterval` 调整为 `2m` 或更长。
- 新 UI 和旧 UI 均可保存 Observatory JSON；使用 `leastPing` Balancer 时需确认 `subjectSelector` 覆盖住宅出站 tag。

---

## 6. DNS 策略

推荐在 Xray JSON 中配置 AI 专用 DNS server：

```json
{
  "queryStrategy": "UseIPv4",
  "servers": [
    {
      "address": "https://1.1.1.1/dns-query",
      "domains": [
        "domain:openai.com",
        "domain:chatgpt.com",
        "domain:anthropic.com",
        "domain:claude.ai",
        "domain:gemini.google.com",
        "domain:aistudio.google.com",
        "domain:generativelanguage.googleapis.com",
        "domain:googleapis.com"
      ],
      "skipFallback": true
    },
    "localhost",
    "1.1.1.1"
  ]
}
```

DNS 设计原则：

- AI 域名优先走显式 DoH，减少被系统 DNS 或运营商 DNS 改写的概率。
- `skipFallback=true` 避免 AI 域名解析失败后回落到其他 DNS server。
- `UseIPv4` 适合 IPv6 质量不稳定或住宅出口仅保证 IPv4 质量的场景。
- 终端客户端如支持 `socks5h`，应优先让域名在代理侧解析，减少本地 DNS 泄漏。

---

## 7. 部署步骤

1. 备份当前数据库和 Xray 配置：

```bash
cp -a /etc/x-ui/x-ui.db "/root/x-ui-db-$(date +%F-%H%M%S).db"
cp -a /usr/local/x-ui/bin/config.json "/root/xray-config-$(date +%F-%H%M%S).json" 2>/dev/null || true
```

2. 在 **Xray → Outbounds** 添加 `us-residential-socks`，使用占位符对应的真实住宅 SOCKS 主机、端口、用户名和密码。
3. 在 **Xray → Routing** 添加 AI 域名规则，`balancerTag` 指向 `ai-residential-balancer`。
4. 在 **Xray → Balancers** 添加 `ai-residential-balancer`，selector 包含 `us-residential-socks`，策略选择 `leastPing`，fallback 选择 `blocked`。
5. 在 **Xray → DNS** 添加 AI 专用 DoH server，并启用 `skipFallback`。
6. 在 **Xray → Observatory** 写入健康检测 JSON。
7. 保存配置并重启 Xray。
8. 在面板日志中确认 Xray Running，无 `failed to build`、`invalid`、`target`、`balancer` 等错误。

---

## 8. 验证清单

在服务器上验证基础状态：

```bash
x-ui status
systemctl is-active x-ui
journalctl -u x-ui -n 100 --no-pager
ss -tulpen | grep -E 'x-ui|xray|:2096|:443'
```

验证 Xray 配置包含预期对象：

```bash
jq '.outbounds[] | select(.tag=="us-residential-socks")' /usr/local/x-ui/bin/config.json
jq '.routing.balancers[] | select(.tag=="ai-residential-balancer")' /usr/local/x-ui/bin/config.json
jq '.routing.rules[] | select(.balancerTag=="ai-residential-balancer")' /usr/local/x-ui/bin/config.json
jq '.observatory' /usr/local/x-ui/bin/config.json
```

验证订阅服务：

```bash
curl -kI "https://<SUB_HOST>:2096/sub/<SUB_ID>"
curl -kI "https://<SUB_HOST>:2096/clash/<SUB_ID>"
```

验证原则：

- `direct` 出站测试应能访问 `https://www.google.com/generate_204`，用于判断服务器基础网络。
- `us-residential-socks` 出站测试应返回 HTTP `204` 或可接受的 2xx/3xx 状态，并记录延迟。
- AI 域名访问慢时，先比较 `direct` 与住宅出站延迟，再判断是住宅出口质量、远端平台、DNS、客户端链路还是入站 Reality 质量问题。
- 如果命中 AI 域名后住宅出口不可用，请求应失败关闭，而不是自动直连。

---

## 9. 终端侧泄漏防护建议

面板侧配置只能约束服务器上的 Xray 行为。终端侧仍需按客户端能力配置：

- 浏览器禁用或限制 WebRTC 本地 IP 暴露。
- 优先使用支持远端解析的代理模式，例如 SOCKS5H 或 TUN/FakeDNS。
- 避免同一浏览器会话在直连和住宅出口之间频繁切换。
- 保持系统时间、时区、语言、DNS 和出口区域策略一致，减少账号侧异常信号。
- 不在客户端日志、订阅导入记录或截图中暴露 `subId`、UUID、Reality 私钥、公钥和住宅出口凭据。

---

## 10. 常见故障

| 现象 | 直接排查点 | 处理方式 |
|------|------------|----------|
| AI 平台仍走直连 | 路由规则未命中、sniffing 未开启、域名缺失 | 开启 TLS/HTTP/QUIC sniffing，补充域名，确认规则在兜底规则之前 |
| 所有流量都走住宅出口 | 存在 `inboundTag -> us-residential-socks` 全量规则 | 删除全量规则，仅保留 AI 域名规则 |
| Xray 启动失败 | JSON 语法、Reality `target`、Balancer tag、出站 tag 错误 | 查看面板 Xray 日志，按首个 `failed to build` 错误修复 |
| Google 或 ChatGPT 很慢 | 住宅出口延迟高、供应商限速、客户端到服务器链路抖动 | 用 Outbound Test 对比 `direct` 与住宅出口；更换住宅节点或降低并发 |
| 订阅导出为空 | 订阅开关未启用、客户端无 `subId`、公开 URI 为空 | 在设置页填充默认订阅 URI，确认客户端启用并存在 `subId` |
| Clash/Mihomo 导入失败 | 协议或传输不被 Clash 输出覆盖 | 改用 `/json/<subId>` 或客户端手动配置，优先使用 TCP/WS/gRPC 主路径 |
| DNS 泄漏 | 终端本地解析、客户端未使用远端 DNS | 使用 SOCKS5H/TUN/FakeDNS，浏览器和客户端侧关闭本地 DNS 泄漏路径 |

---

## 11. 回滚方案

出现不可接受的延迟、平台失败或 Xray 启动问题时：

1. 在面板中停用 AI 域名路由规则，保留住宅 SOCKS 出站配置。
2. 或将 `ai-residential-balancer` 的 selector 临时改为空，并观察 Xray 是否启动。
3. 如需完全回滚，删除 `us-residential-socks` 出站、AI 路由规则、`ai-residential-balancer`、对应 Observatory 和 AI 专用 DNS server。
4. 恢复备份的 `/usr/local/x-ui/bin/config.json` 或数据库后重启：

```bash
systemctl restart x-ui
x-ui restart-xray
journalctl -u x-ui -n 100 --no-pager
```

回滚后必须重新验证：Xray Running、普通客户端可连通、非 AI 流量未误走住宅出口、订阅链接仍可访问。
