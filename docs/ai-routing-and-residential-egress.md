# AI 平台智能分流与住宅出口运行手册

> **目标读者**：需要在 SuperXray 中为 OpenAI/ChatGPT、Anthropic/Claude、Google/Gemini 等 AI 平台配置专用出口的运维人员
> **适用版本**：`v3.4.3`
> **代码核对日期**：2026-10-15；当前分支纳入上游 `3x-ui v3.10.2`，目标 Xray 为 `v26.10.10`，项目版本仍由 `config/version` 声明为 `3.4.3`。服务器、住宅供应商和 OpenWrt 的运行状态需在目标环境重新验证。
> **事实来源**：`frontend/src/views/XrayView.vue`、`frontend/src/utils/xrayCompat.ts`、`frontend/src/utils/gatewayEgressMvp.ts`、`web/controller/xray_setting.go`
> **相关文档**：[系统架构设计](architecture.md) | [API 接口说明](api.md) | [入站创建教程](inbound-creation-guide.md) | [服务器部署 + OpenWrt 路由 + AI 出口治理统一总览](operations-ai-routing-overview.md)

---

## 1. 当前实现边界

本文描述的是 **Xray JSON 模板级** 分流方案。新 UI 的 Xray 页面可以结构化编辑 Residential IP Pool、AI Routing、DNS、Routing、Balancer、Observatory、Gateway Egress MVP，但保存时仍写入现有 Xray 配置模板：

```text
XrayView
  -> frontend utils 修改 JSON
  -> POST /panel/api/xray/update
  -> XraySettingService.SaveXraySetting
  -> 兼容转换 + 合并已启用入站 + 已安装 Xray run -test
  -> 校验成功后持久化模板
  -> 用户显式重启 legacy XrayService
```

保存目标是现有 `settings` 中的 `xrayTemplateConfig`。服务端解开旧响应包装、执行 `util/xraycompat` 兼容转换，并用候选模板与当前启用的数据库入站生成完整配置；随后调用已安装核心的 `run -test -config`（15 秒超时）。失败时不会写入候选模板。校验不启动监听，不能证明端口、出口、DNS 或 AI 平台实际可用；缺少核心、证书或 Geo 文件也会使保存失败。

保存不会自动重启。Vue 保存成功后显示确认框；确认后才调用既有 `/panel/api/server/restartXrayService`。重启先预检再停止旧核心，新进程启动失败会尝试恢复旧进程；恢复失败仍需人工处理。下文面板路由均相对部署的 `webBasePath`。实现见 `web/service/xray_setting.go`、`web/service/xray.go`、`xray/validate.go`。

当前 Vue SDK 使用 `/panel/api/xray/update`；旧 `/panel/xray/update` 是保留的兼容入口，两者的状态变更请求均要求登录和 CSRF token。

不会发生：

- 不新增数据库表。
- 不新增后端出口模型。
- 不创建 `proxy_inbounds` / `proxy_clients`。
- 不让 CoreManager 接管 legacy Xray。
- 不把 sing-box 设为默认出口治理层。

重要边界：

- 这是合规可控的域名级路由和失败保护，不承诺规避平台风控。
- AI 域名清单会变化，需要按日志和实际访问维护。
- WebRTC、浏览器指纹、客户端 DNS 泄漏主要发生在终端侧，面板无法替代终端隐私配置。
- 文档示例必须使用占位符，不应写入真实服务器 IP、订阅 ID、UUID、私钥、公钥或住宅代理密码。

---

## 2. Residential IP Pool

新 UI 的 **Residential IP Pool** 不是后端模型，而是识别和编辑 Xray 模板中的 `socks` outbound。

识别规则来自 `xrayCompat.ts`：

```text
protocol == "socks"
tag 转小写后包含 "residential"
```

推荐手工或通过 UI 生成：

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

保留默认失败保护出站：

```json
{
  "tag": "blocked",
  "protocol": "blackhole"
}
```

安全要求：

- 不要把住宅代理密码写入 Git、Issue、截图或共享文档。
- 不要添加“所有入站 / 所有域名 -> residential”的兜底规则。
- 测试住宅 outbound 时使用 Xray 页的 Outbound Test；测试 URL 由服务端保存的 `xrayOutboundTestUrl` 决定，不接受请求临时传入任意 URL。
- Residential 表单保存会按已知 SOCKS 字段重建该 outbound；需要保留 `streamSettings` 等额外字段时，使用原始模板或完整 outbound JSON 编辑器并核对差异。

---

## 3. AI Residential Routing

点击 **Apply AI Routing** 时，前端 `applyAiResidentialRouting` 会：

1. 收集当前模板中所有 Residential IP Pool outbound tag。
2. 创建或替换 balancer：

```json
{
  "tag": "ai-residential",
  "selector": [
    "us-residential-socks"
  ],
  "strategy": {
    "type": "random"
  }
}
```

3. 添加 TCP 域名规则，走 `ai-residential`：

```json
{
  "type": "field",
  "balancerTag": "ai-residential",
  "network": "tcp",
  "domain": [
    "domain:openai.com",
    "domain:chatgpt.com",
    "domain:oaistatic.com",
    "domain:oaiusercontent.com",
    "domain:sora.com",
    "domain:openai.azure.com",
    "domain:anthropic.com",
    "domain:claude.ai",
    "domain:claude.com",
    "domain:aistudio.google.com",
    "domain:generativelanguage.googleapis.com",
    "domain:makersuite.google.com",
    "domain:gemini.google.com",
    "domain:cloudcode-pa.googleapis.com",
    "domain:aiplatform.googleapis.com",
    "domain:openrouter.ai",
    "domain:x.ai",
    "domain:grok.com",
    "domain:perplexity.ai",
    "domain:groq.com",
    "domain:mistral.ai",
    "domain:cohere.com",
    "domain:fireworks.ai",
    "domain:together.xyz",
    "domain:huggingface.co",
    "domain:cursor.com",
    "domain:cursor.sh",
    "domain:githubcopilot.com",
    "domain:codeium.com",
    "domain:windsurf.com"
  ]
}
```

4. 添加 UDP 域名规则，走 `blocked`：

```json
{
  "type": "field",
  "outboundTag": "blocked",
  "network": "udp",
  "domain": [
    "domain:openai.com",
    "domain:chatgpt.com",
    "domain:oaistatic.com",
    "domain:oaiusercontent.com",
    "domain:sora.com",
    "domain:openai.azure.com",
    "domain:anthropic.com",
    "domain:claude.ai",
    "domain:claude.com",
    "domain:aistudio.google.com",
    "domain:generativelanguage.googleapis.com",
    "domain:makersuite.google.com",
    "domain:gemini.google.com",
    "domain:cloudcode-pa.googleapis.com",
    "domain:aiplatform.googleapis.com",
    "domain:openrouter.ai",
    "domain:x.ai",
    "domain:grok.com",
    "domain:perplexity.ai",
    "domain:groq.com",
    "domain:mistral.ai",
    "domain:cohere.com",
    "domain:fireworks.ai",
    "domain:together.xyz",
    "domain:huggingface.co",
    "domain:cursor.com",
    "domain:cursor.sh",
    "domain:githubcopilot.com",
    "domain:codeium.com",
    "domain:windsurf.com"
  ]
}
```

5. 如果模板缺少 `blocked` outbound，会自动补一个 blackhole。

规则顺序：

- 原有 `outboundTag=api` 的规则保留在前。
- AI TCP 和 UDP 规则插在 API 规则之后。
- 其他已有规则跟在后面。

默认不设置 balancer `fallbackTag`。住宅出口不可用时，请求应失败关闭，而不是自动直连。

这是生成配置的目标行为，必须以实际域名命中和故障演练验证。无 residential outbound 时操作返回原模板；已有 `blocked` tag 会保留，生成器不检查它是否确实为 `blackhole`。与 Gateway MVP 同时使用时，先应用 AI Routing，再生成 Gateway 配置，使 Gateway 专属规则位于最前；之后再次应用 AI Routing 会把共享 AI 规则前置，可能改变 Gateway 的专属出口选择，保存前必须重新检查顺序。

维护域名清单时必须同步改三处，否则会出现重复或残留规则：

- `frontend/src/utils/xrayCompat.ts` 的 `AI_RESIDENTIAL_DOMAINS`。
- 同文件的 `isAiResidentialDomainCandidate` 关键词表：它决定哪些旧规则会被识别并替换，新增域名若不命中关键词就不会被清理。
- `frontend/tests/xray-compat.test.ts` 中断言域名清单的用例。

不要加入 `domain:google.com`、`domain:googleapis.com` 这类整域规则，副作用过大；Google 侧只加具体 API 子域。

---

## 4. DNS 建议

AI Routing 只负责 Xray routing；DNS 仍需单独配置。推荐在 Xray 页的 DNS Servers / DNS Policy 中维护 AI 域名解析。

示例：

```json
{
  "queryStrategy": "UseIPv4",
  "disableFallbackIfMatch": true,
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
        "domain:makersuite.google.com"
      ],
      "skipFallback": true
    },
    "localhost",
    "1.1.1.1"
  ]
}
```

原则：

- AI 域名优先使用显式 DoH。
- 对 AI 专用 DNS server 开启 `skipFallback`。
- IPv6 质量不稳定时优先 `UseIPv4`。
- 终端客户端支持 `socks5h`、TUN 或 FakeDNS 时，优先让域名在代理侧解析。

---

## 5. Observatory

Xray 页可编辑 `observatory` 和 `burstObservatory` JSON。基础示例：

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

- `probeURL` 只用于健康探测，不代表 AI 平台真实延迟。
- 住宅供应商限制探测频率时，把 `probeInterval` 调整为 `2m` 或更长。
- 如手工启用 `leastPing`，确认 balancer selector 与 observatory subject selector 覆盖相同 outbound tag。

---

## 6. Gateway Egress MVP

Xray 页的 **Gateway Egress MVP** 用于生成供 Super-Code-Gateway 或同类网关登记的本机代理入口和 CSV 清单。它仍然只修改 Xray JSON 模板，不新增后端 API、数据库或 CoreManager 功能。

### 6.1 默认网络策略

来自 `gatewayEgressMvp.ts`：

```json
{
  "listenHost": "127.0.0.1",
  "manifestHost": "127.0.0.1",
  "strategyLabel": "same-network"
}
```

Host 校验会拒绝：

- 空值。
- 通配监听：`0.0.0.0`、`::`、`[::]`、`*`。
- 带协议、路径、逗号或空白的值。

该校验只检查字符串形态，不验证 DNS、真实连通性、是否公网地址、端口冲突或防火墙。`noauth` 入口使用非 loopback 地址时，必须另行限制来源。生成器按 tag 替换同名 inbound，不检查其他 tag 是否占用相同端口；同名 outbound 与已有 `blocked` 会原样保留，需人工确认它们是预期的真实出口和拒绝出口。

### 6.2 生成的 profiles

| key | port | platform | region | egressGroup |
|---|---:|---|---|---|
| `openai-us-primary` | `11801` | `openai` | `US` | `openai-egress` |
| `anthropic-us-primary` | `11802` | `anthropic` | `US` | `anthropic-egress` |
| `gemini-us-primary` | `11803` | `gemini` | `US` | `gemini-egress` |
| `region-us-primary` | `11901` | 空 | `US` | `region-us` |
| `region-jp-primary` | `11981` | 空 | `JP` | `region-jp` |

### 6.3 生成的 Xray inbound

每个 profile 生成一个本地 SOCKS inbound：

```json
{
  "tag": "gateway-openai-us-primary",
  "listen": "127.0.0.1",
  "port": 11801,
  "protocol": "socks",
  "settings": {
    "auth": "noauth",
    "accounts": [],
    "udp": false,
    "ip": "127.0.0.1"
  }
}
```

### 6.4 生成的 placeholder outbound

每个 profile 会确保存在对应 outbound；默认是 `freedom` 占位，必须在生产使用前替换成真实 VPN/代理出站：

```json
{
  "tag": "openai-egress",
  "protocol": "freedom",
  "settings": {},
  "_gatewayEgressMvp": {
    "profile": "openai-us-primary",
    "expectedCountryCode": "US",
    "note": "Replace this placeholder with the real VPN/proxy outbound before production use."
  }
}
```

### 6.5 生成的 routing

平台 profile 会生成域名规则：

| platform | domains |
|---|---|
| `openai` | `domain:api.openai.com`、`domain:chatgpt.com`、`domain:chat.openai.com`、`domain:openai.com`、`domain:oaistatic.com`、`domain:oaiusercontent.com`、`domain:sora.com`、`domain:openai.azure.com` |
| `anthropic` | `domain:api.anthropic.com`、`domain:claude.ai`、`domain:anthropic.com`、`domain:claude.com` |
| `gemini` | `domain:generativelanguage.googleapis.com`、`domain:cloudcode-pa.googleapis.com`、`domain:aiplatform.googleapis.com`、`domain:aistudio.google.com`、`domain:gemini.google.com` |

区域 profile 只按 inboundTag 绑定到对应 egress group。

平台规则是 inboundTag 与域名同时匹配，随后才到拒绝保护；区域规则没有域名条件，会接受该区域入口上的任意目标，最终保护规则无法限制它。区域入口应只对受信任 Gateway 开放，并按业务需要增加目标限制。所有生成规则及保护规则前置于保留的既有规则；重新生成只替换 `_gatewayEgressMvp=true` 的旧规则。

最后会添加一条保护规则，把所有 Gateway 生成 inbound 未匹配流量送到 `blocked`：

```json
{
  "type": "field",
  "inboundTag": [
    "gateway-openai-us-primary",
    "gateway-anthropic-us-primary",
    "gateway-gemini-us-primary",
    "gateway-region-us-primary",
    "gateway-region-jp-primary"
  ],
  "outboundTag": "blocked",
  "_gatewayEgressMvp": true
}
```

### 6.6 CSV manifest

前端可复制或下载 CSV：

```csv
name,protocol,host,port,platform,region_code,expected_country_code,egress_group,health_status,notes
openai-us-primary,socks5h,127.0.0.1,11801,openai,US,US,openai-egress,manual-check,OpenAI MVP local exit (same-network)
```

CSV 只是登记清单，不是后端状态；`health_status` 固定为 `manual-check`。

CSV 按当前表单参数生成，不会读取运行中的 Xray 配置；`expected_country_code` 和 `region_code` 只是标签，不执行国家检测或出口绑定。生成器不会替换已存在的同名 outbound，也不会自动登记 Gateway。导出后逐项对照已保存模板与实际监听。

---

## 7. 推荐部署步骤

1. 备份数据库和当前 Xray 配置：

```bash
sqlite3 /etc/x-ui/x-ui.db ".backup '/root/x-ui-backup.db'"
cp -a /usr/local/x-ui/bin/config.json "/root/xray-config-$(date +%F-%H%M%S).json" 2>/dev/null || true
```

上述路径适用于安装脚本的 systemd 布局；先确认实际数据库和配置路径，备份文件使用受限权限。也可使用面板鉴权的数据库备份接口；不要在运行中直接复制 SQLite 主文件作为完整备份，尤其存在 WAL 时。

2. 在 Xray 页添加或确认 `direct`、`blocked`、住宅 SOCKS outbound。
3. 使用 Residential IP Pool 保存一个或多个 tag 包含 `residential` 的 socks outbound。
4. 点击 Apply AI Routing，检查生成的 `ai-residential` balancer、TCP AI 域名规则和 UDP blocked 规则。
5. 视需求添加 DNS server、DNS policy、Observatory。
6. 保存 Xray 模板；候选完整配置须通过当前安装核心的预检，未通过时保持旧模板并先处理错误。
7. 手动重启 Xray。
8. 查看 Xray 日志，确认没有 `failed to build`、`invalid`、`target`、`balancer` 等错误。

Gateway Egress MVP 的部署顺序：

1. 在 Gateway Egress MVP 区域确认 `listenHost` 和 `manifestHost`。
2. 点击生成配置。
3. 把 placeholder outbound 替换为真实出口。
4. 保存模板并重启 Xray，核对五个监听、规则与真实出口。
5. 从 Gateway 所在网络命名空间完成正反向探测，再复制或下载 CSV manifest 登记；不要把尚未生效的入口交付给业务调用。

---

## 8. 验证清单

基础状态：

```bash
x-ui status
systemctl is-active x-ui
journalctl -u x-ui -n 100 --no-pager
ss -tulpen | grep -E 'x-ui|xray|:2096|:443|:11801|:11802|:11803'
```

Xray 配置对象：

```bash
jq '.outbounds[] | select(.tag=="us-residential-socks") | {tag,protocol}' /usr/local/x-ui/bin/config.json
jq '.routing.balancers[] | select(.tag=="ai-residential")' /usr/local/x-ui/bin/config.json
jq '.routing.rules[] | select(.balancerTag=="ai-residential")' /usr/local/x-ui/bin/config.json
jq '.routing.rules[] | select(._gatewayEgressMvp==true)' /usr/local/x-ui/bin/config.json
```

订阅状态：

```bash
curl --connect-timeout 5 --max-time 30 -sS -o /dev/null -w 'HTTP %{http_code}\n' "https://<SUB_HOST>:<SUB_PORT>/<SUB_PATH>/<SUB_ID>"
curl --connect-timeout 5 --max-time 30 -sS -o /dev/null -w 'HTTP %{http_code}\n' "https://<SUB_HOST>:<SUB_PORT>/<SUB_JSON_PATH>/<SUB_ID>"
curl --connect-timeout 5 --max-time 30 -sS -o /dev/null -w 'HTTP %{http_code}\n' "https://<SUB_HOST>:<SUB_PORT>/<SUB_CLASH_PATH>/<SUB_ID>"
```

订阅路由注册的是 GET，不能用 HEAD 的 `404` 判断订阅故障；`subPath`、`subJsonPath`、`subClashPath`、协议与端口按实际设置替换，示例中的路径占位符不含首尾 `/`。JSON/Clash 路由分别受启用开关控制。不要将带真实订阅 ID 的命令或完整响应公开；HTTP 成功仍需在本地核对可输出节点和目标客户端兼容性。

操作判定：

- `direct` 测试可用于判断服务器基础网络。
- residential outbound 测试可用于判断住宅出口是否可用。
- AI 域名命中后如果住宅出口不可用，应失败关闭，不应自动直连。
- Gateway MVP 的 `freedom` placeholder 必须替换后才算真实出口配置。

---

## 9. 常见故障

| 现象 | 直接排查点 | 处理方式 |
|---|---|---|
| AI 平台仍走直连 | 域名未命中、sniffing 未启用、规则顺序在兜底之后 | 补域名，启用 TLS/HTTP/QUIC sniffing，移动规则到普通兜底之前 |
| 所有流量都走住宅出口 | 存在全量 inboundTag 或 domain 规则 | 删除全量规则，只保留 AI 域名规则 |
| Xray 启动失败 | JSON 语法、outbound tag、balancer tag、Reality target、当前 Xray 不支持字段 | 查看 Xray 日志第一条 `failed to build` 错误 |
| Residential IP Pool 为空 | outbound tag 不含 `residential` 或 protocol 不是 `socks` | 调整 tag 或协议 |
| Outbound Test 失败 | 住宅代理不可达、认证错误、测试 URL 不通 | 用服务端日志和供应商控制台核对 |
| Gateway 端口不可连 | `listenHost` 为 127.0.0.1 但 Gateway 与 Xray 不在同一网络命名空间 | 使用共享网络命名空间，或改用可达的受控地址并限制来源；同一宿主机上的普通容器也不共享 loopback |
| Gateway 流量被 blocked | platform 域名清单未覆盖目标 | 补充 routing rule 或使用区域 profile |
| 订阅为空 | `subEnable`、`subId`、公开 URI、协议能力不匹配 | 用 `subPath`、`subJsonPath`、`subClashPath` 配置的入口追加 `:subid/diagnose` 排查 |
| DNS 泄漏 | 终端本地解析、客户端未使用远端 DNS | 使用 SOCKS5H/TUN/FakeDNS，限制浏览器本地 DNS 路径 |

---

## 10. 回滚

最小回滚会让原 AI 流量重新落到既有后续路由，可能变为直连。先暂停对应 Gateway 代理绑定或确认该行为可接受，再执行：

1. 禁用或删除 AI 域名路由规则。
2. 保留 residential outbound，以便后续排查。
3. 保存模板并重启 Xray。

完全回滚优先恢复同一份已知可用模板，保留其引用的 outbound、balancer、DNS 和 Observatory，避免逐条删除造成悬空引用。若必须恢复数据库，按 [部署指南](deployment.md#10-备份与恢复) 停止写入并保留原文件；数据库导入使用面板候选迁移与预检路径。恢复模板并重启后再恢复 Gateway 绑定：

```bash
systemctl restart x-ui
x-ui restart-xray
journalctl -u x-ui -n 100 --no-pager
```

回滚后确认：

- Xray Running。
- 普通客户端可连通。
- 非 AI 流量未误走住宅出口。
- 订阅链接仍可访问。

---

## 11. OpenWrt / Passwall 场景的分流职责拆分

当 SuperXray 节点被接入 OpenWrt / Passwall 这类中间设备时，建议把“主 WiFi”和“专用代理 WiFi”职责彻底拆开：

### 11.1 推荐结构

- **主 WiFi / `lan`**：默认直连，只把 AI / GFW 域名送入 `<PRIMARY_PROXY_HOST>`。
- **USA WiFi / `us`**：规划为独立美国出口网络，绑定 `<US_PROXY_HOST>`；是否全代理需以现场 ACL 和运行态确认。

这种结构的好处是：

- 普通网页和下载不会被无谓代理拖慢。
- AI / 模型接口路径更稳定，便于统一治理。
- 日常工作网络与专用美国网络互不干扰。

### 11.2 不要混淆三条路径

1. **显式 SOCKS 出口**：通过 `curl -x socks5h://...` 验证，只代表应用主动走代理。
2. **透明 REDIRECT / TPROXY**：由 `iptables` 和 Passwall 运行态接管，必须看 ACL 命中计数。
3. **DNS 劫持 / 分流**：决定域名是否会被视为代理目标。若 DNS 与代理列表脱节，就会出现“首页能开、模型接口失败”。

### 11.3 模型接口域名为什么要单独维护

AI 平台通常不止一个首页域名，还会拆分为：

- API 域名
- 控制台域名
- 静态资源域名
- IDE 插件/代理域名
- 账户系统域名

如果只代理首页域名，常见问题是：

- 页面可访问，但模型请求超时
- Cursor / Copilot 登录正常，但补全失败
- OpenRouter 控制台正常，但 API 不稳定

因此建议优先维护“高价值、低副作用”的模型接口域名列表，而不是把 `github.com`、`x.com` 这类大域名整站代理。

### 11.4 运行态验证优先级

1. 客户端确认当前 SSID、网段和默认出口。
2. OpenWrt 确认 `iptables -vnL` 计数、`/tmp/etc/passwall/var`、Passwall 运行进程。
3. 服务器确认节点本身可用和真实出口。

如果三者不一致，优先相信运行态计数和进程参数，而不是 UI 设置页的静态显示。更多实战排障步骤见 [OpenWrt / Passwall AI 路由实战手册](passwall-openwrt-ai-routing-playbook.md)。
