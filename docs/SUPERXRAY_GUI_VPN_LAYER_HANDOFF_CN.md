# SuperXray-gui VPN 层出口治理交接文档

> 面向对象：SuperXray-gui 项目组、VPN/网络出口负责人、运维负责人、安全负责人。  
> 交接方：Super-Code-Gateway 项目组。  
> 文档日期：2026-05-16。  
> 目标：把由 SuperXray-gui 负责的 VPN 层、代理入口、平台专属出口、国家/地区出口和监控验收要求完整拆出，作为跨项目实施依据。

## 0. SuperXray-gui 项目内落地裁定（2026-05-16）

本交接文档作为跨项目目标、边界和验收清单保留，但不能作为当前仓库一次性全量实现规格。结合 SuperXray-gui 现阶段 UI-first / Xray parity 边界，执行拆分如下：

- MVP 只做 Xray 兼容配置生成与本机端口登记，不新增数据库模型，不接管 CoreManager，不碰 sing-box 生产路径。实施计划见：`docs/superpowers/plans/2026-05-16-vpn-egress-mvp-xray-compatible.md`。
- Phase 10+ 才评审完整出口治理子系统，包含 `egress_groups`、`egress_nodes`、`egress_probe_results`、`egress_switch_events` 的 schema/API/UI 与探测、切换、审计能力。设计草案见：`docs/superpowers/specs/2026-05-16-phase10-egress-governance-design.md`。
- 若下文愿景描述与本节阶段裁定冲突，以本节和上述两个拆分文档为当前执行边界。

2026-05-16 已批准 MVP 进入实施，但必须把“网络可达 host 策略”作为生成前置条件：

- 生成器必须区分 `listenHost` 与 `manifestHost`。`listenHost` 是 Xray inbound 实际监听地址，`manifestHost` 是 Super-Code-Gateway CSV 代理登记地址。
- 默认同网络命名空间场景可使用 `listenHost=127.0.0.1`、`manifestHost=127.0.0.1`。
- 当前 Docker bridge + 宿主机 x-ui 形态下，`manifestHost` 必须替换为 Gateway 容器实际可达的 host，例如 `host.docker.internal`、宿主网桥地址或经评审确认的服务名。
- 当前阶段暂缓 `egress_*` 数据库与 `/panel/api/egress/*` API，仅保留 Phase 10+ 设计评审。
- 不接管 CoreManager，不碰 sing-box 生产路径；`default-xray` 仍是只读观察实例，不支持通过 CoreManager 生命周期接管，代码边界见 `web/service/core_service.go:62` 起的 `default-xray` 实例定义。

### 0.1 实际服务器运行态审查摘要

2026-05-16 已对当前同服服务器做只读运行态审查，结论如下：

| 检查项 | 当前服务器事实 | 对本文档的影响 |
|---|---|---|
| 部署拓扑 | `Super-Code-Gateway` 运行在 Docker 容器中，`SuperXray-gui/x-ui` 以 systemd 服务运行在宿主机 | 两者不是同一网络命名空间，不能直接把所有 `127.0.0.1` 示例当成当前部署可用地址 |
| Gateway 可消费入口 | 未发现 `1080/8081/11801/11802/11803/11901/11981` 等 SOCKS/HTTP 入口监听 | MVP 必须先生成并落地 Gateway-facing Xray inbound |
| 现有 Xray 入站 | 当前只有 `*:443` VLESS Reality 客户端入口、`127.0.0.1:62789` API、`127.0.0.1:11111` metrics | `inbound-443` 是客户端代理入口，不是 Gateway 本机出口治理入口 |
| AI 分流现状 | 已有 OpenAI/ChatGPT、Anthropic/Claude、Google/Gemini 域名命中共享 `ai-residential` balancer | 仅说明已有客户端侧 AI 分流雏形，不等于平台独立出口组已经完成 |
| 出口组现状 | 当前只有 `us-residential-socks` 一个 AI 住宅 SOCKS 出站参与 balancer | 仍需拆出或生成 `openai-egress`、`anthropic-egress`、`gemini-egress` 与地区出口 |
| Gateway 数据状态 | Gateway 已有 `US`、`JP` 区域，但 `proxies` 和 `account_region_proxy_bindings` 当前为空 | SuperXray-gui 必须输出可导入/可登记的代理清单，Gateway 才能绑定账号区域 |
| DNS 与健康检查 | 当前 Xray `dns` 为 `null`，未启用 `observatory` / `burstObservatory` | DNS、探测、熔断、切换和审计属于 Phase 10+ 或后续增强验收 |

因此，本文档当前必须按“两层交付”理解：MVP 先解决 Xray 兼容配置生成、本机入口和 Gateway 登记清单；完整出口治理平台能力进入 Phase 10+。

### 0.2 新 UI 落地状态（2026-05-16）

MVP 入口已经完成前端落地并在 Xray 页前置：

- Xray 页新增 **Xray 工作区** 导航，`Gateway 出口`、`运行控制`、`模板编辑`、`出站工具`、`结构化配置`、`DNS 策略`、`协议工具` 可直接跳转。
- **Gateway 出口 MVP** 面板位于模板编辑器之前，避免入口埋在长配置页后段。
- 面板提供 `listenHost`、`manifestHost`、网络策略标签输入，以及“生成 Xray 配置 / 复制登记清单 / 下载登记清单”操作。
- 当前仍只生成 Xray 兼容配置与 CSV，不新增数据库模型，不新增 API，不接管 CoreManager，不触碰 sing-box 生产路径。
- 中文界面首屏、Gateway MVP 操作和移动端表单已经改为显式 i18n 文案；顶部 Header 玻璃背景 token 已覆盖 Ant 默认样式。
- 移动端抽屉提供显式关闭按钮，打开后自动聚焦“关闭导航”，工作区导航按钮保持 44px 高度且无横向溢出。

桌面视图：

![Xray 工作区与 Gateway 出口 MVP](assets/xray-mvp-desktop.png)

移动视图：

![Gateway 出口 MVP 移动端表单](assets/xray-mvp-mobile.png)

## 1. 背景

Super-Code-Gateway 已经具备 AI API 网关、账号管理、代理登记、账号区域标签和区域代理池能力。网关侧可以在账号创建和编辑时绑定：

- 手动代理：`proxy_id`
- 账号区域：`region_code`
- 区域代理池：`account_region_proxy_bindings`

前端联动点位于：

- `frontend/src/components/account/CreateAccountModal.vue`
  - `ProxySelector` 负责选择账号手动代理。
  - `form.region_code` 负责选择账号所在国家或地区。
  - 创建账号时会提交 `proxy_id` 与 `region_code`。

因此，Super-Code-Gateway 侧的职责是“选择账号、选择代理配置、发起上游请求”。实际 VPN、代理节点、平台域名分流、国家地区出口、DNS 策略和出口健康由 SuperXray-gui 负责。

本交接文档要求 SuperXray-gui 将 VPN 层能力产品化、配置化、可监控化，使网关可以把本机代理当成稳定、可审计、可回滚的出口资源使用。

## 2. 边界划分

### 2.1 Super-Code-Gateway 负责

- 提供统一 API 入口。
- 管理用户、API Key、分组、订阅、账号和计费。
- 管理上游 AI 平台账号。
- 管理代理记录，包括协议、主机、端口、账号密码、启停状态。
- 管理账号区域，例如 `US`、`JP`、`HK`。
- 管理区域和代理的绑定关系。
- 在请求选定账号后，根据账号手动代理或区域代理池解析最终代理 URL。
- 记录网关层请求、错误、账号、平台、延迟和用量。

### 2.2 SuperXray-gui 负责

- 启动和维护本机代理入口。
- 管理 VPN、代理、隧道、WARP、住宅/ISP 出口等网络节点。
- 按平台建立专属出口组：
  - OpenAI / ChatGPT
  - Anthropic / Claude
  - Google / Gemini
- 按国家和地区建立出口组，例如 `region-us`、`region-jp`、`region-hk`。
- 维护域名规则、IP 规则和 DNS 策略。
- 维护主备出口、健康检查、熔断、灰度回切和回滚。
- 防止本机代理端口暴露公网。
- 输出供 Gateway 登记的本机代理地址、协议、出口地区、健康状态和验证结果。

### 2.3 明确非目标

- 不在 Super-Code-Gateway 中直接启动 VPN 进程。
- 不把整机默认路由强行切到 VPN。
- 不提供伪造身份、绕过平台限制、规避检测或违反平台条款的方案。
- 不把敏感代理凭据写入日志、前端、截图或公开文档。
- 不要求用户终端直连 OpenAI、Claude 或 Gemini 官方服务。

## 3. 总体目标

SuperXray-gui 需要交付一套“本机出口治理层”，使 Gateway 能够通过本机代理稳定访问主流 AI 平台，并具备以下能力：

1. 平台专属出口  
   OpenAI、Anthropic、Google/Gemini 可以分别绑定不同出口组。

2. 国家/地区出口  
   `US`、`JP`、`HK` 等区域账号能够命中对应地区出口。

3. 本机代理隔离  
   Gateway 只连接 `127.0.0.1` 上的 HTTP/SOCKS5 入口，不直接理解 VPN 隧道细节。

4. 出口稳定性  
   同一账号、同一区域、同一平台在短时间内不频繁漂移出口国家或 IP 段。

5. 可观测与可回滚  
   每个出口组都有健康检查、最近出口 IP、国家、延迟、失败原因、切换记录和回滚入口。

6. 安全收敛  
   本机代理不公网暴露，DNS 不泄漏，凭据不泄漏，敏感日志脱敏。

## 4. 推荐拓扑

```text
Client / SDK / CLI
  -> HTTPS
  -> Super-Code-Gateway
  -> 账号调度
  -> 手动 proxy_id 或 region_code 解析
  -> 127.0.0.1:1080 / 127.0.0.1:8081
  -> SuperXray-gui 本机代理入口
  -> 平台/地区出口组
  -> OpenAI / Anthropic / Google Gemini 上游
```

Super-Code-Gateway 看到的是一个普通代理：

```text
socks5h://127.0.0.1:1080
http://127.0.0.1:8081
```

SuperXray-gui 内部负责把该入口继续分流到：

```text
openai-egress
anthropic-egress
gemini-egress
region-us
region-jp
region-hk
fallback-stable
```

### 4.1 Docker 同服部署地址裁定

上面的 `127.0.0.1` 拓扑只在 Gateway 与 SuperXray-gui 处于同一网络命名空间时成立。当前实测服务器中，Gateway 在 Docker bridge 网络内，SuperXray-gui/x-ui 在宿主机网络内，因此：

- Gateway 容器内的 `127.0.0.1` 指向 Gateway 容器自身。
- 宿主机 Xray 的 `127.0.0.1:<port>` 默认不会被 Gateway 容器访问到。
- 在未完成网络映射前，Gateway 中登记 `socks5h://127.0.0.1:11801` 不会命中宿主机 Xray。

当前项目允许以下四种落地方式，实施前必须选定一种并写入代理登记清单：

| 方案 | `listenHost` | `manifestHost` | 使用条件 |
|---|---|---|---|
| Gateway 使用 host network | `127.0.0.1` | `127.0.0.1` | 适合单机 Linux，但会改变容器隔离和端口映射 |
| Docker `host-gateway` | Docker 网桥可达地址或受控内网地址 | `host.docker.internal` 或宿主网关地址 | 推荐用于保持 Docker bridge 的部署，防火墙只允许 Gateway 容器网段访问 |
| SuperXray-gui 容器化 | 容器内受控监听地址或服务绑定地址，当前 MVP 禁止 `0.0.0.0` 通配监听 | `superxray` 或同网络服务名 | 推荐用于长期可维护部署；如确需通配监听，必须进入 Phase 10+ 或专项安全评审 |
| Gateway sidecar 代理 | `127.0.0.1` | `127.0.0.1` | 可控性强，但需要调整部署编排 |

MVP 文档中生成的 `127.0.0.1` 清单是“同命名空间默认模板”。如果生产部署继续采用当前 Docker bridge + 宿主机 x-ui 形态，最终导出的 Gateway CSV 必须替换为实际可达 host，并附带端口安全说明；当前 MVP 生成器会拒绝空 host、通配 host 和带协议/path 的 host。

生成器必须把 `listenHost` 和 `manifestHost` 分开处理：`listenHost` 写入 Xray inbound 的 `listen`/`settings.ip`，`manifestHost` 写入 Gateway CSV 的 `host` 字段。禁止用一个 `host` 字段同时承担监听和登记语义。

## 5. 本机代理入口要求

### 5.1 必须监听本机地址

必须：

```text
SOCKS5: 127.0.0.1:1080
HTTP:   127.0.0.1:8081
```

禁止：

```text
0.0.0.0:1080
0.0.0.0:8081
公网 IP:1080
公网 IP:8081
```

### 5.2 建议入口命名

```text
gateway-socks5-in
gateway-http-in
```

### 5.3 协议要求

- SOCKS5 必须支持远端 DNS 解析能力。
- Gateway 登记 SOCKS5 时推荐使用 `socks5h` 语义。
- HTTP 入口必须支持 `CONNECT` 隧道。
- 入口必须支持长连接和流式响应，不得主动短超时断开。
- 入口不得要求公网访问白名单，仅允许本机访问。

### 5.4 端口安全验收

SuperXray-gui 项目组需要提供以下命令输出或等价截图：

```powershell
Get-NetTCPConnection -State Listen |
  Where-Object { $_.LocalPort -in 1080,8081 } |
  Select-Object LocalAddress,LocalPort,OwningProcess,State
```

验收条件：

- `LocalAddress` 只能是 `127.0.0.1` 或 `::1`。
- 不允许出现 `0.0.0.0`。
- 不允许公网 IP 监听。

## 6. 出口组设计

### 6.1 平台出口组

至少需要建立以下平台出口组：

| 出口组 | 用途 | 最低节点要求 | 切换策略 |
|---|---|---:|---|
| `openai-egress` | OpenAI API、ChatGPT 相关服务 | 1 主 1 备 | 主故障后切备，不频繁抖动 |
| `anthropic-egress` | Anthropic API、Claude 相关服务 | 1 主 1 备 | 主故障后切备 |
| `gemini-egress` | Gemini API、Google AI Studio、Code Assist | 1 主 1 备 | 主故障后切备 |
| `fallback-stable` | 灾备兜底 | 1 | 人工确认后启用 |

### 6.2 地区出口组

至少支持按国家和地区扩展：

| 出口组 | 用途 | 示例区域 |
|---|---|---|
| `region-us` | 美国区账号出口 | `US` |
| `region-jp` | 日本区账号出口 | `JP` |
| `region-hk` | 香港区账号出口 | `HK` |
| `region-sg` | 新加坡区账号出口 | `SG` |
| `region-eu` | 欧洲区域出口 | `DE`、`FR`、`NL` 等 |

地区出口组必须有清晰的出口国家检测结果。若出口 IP 检测国家与组名不一致，默认不得标记为健康。

### 6.3 组内节点字段

SuperXray-gui 内部每个出口节点建议维护以下字段：

| 字段 | 含义 |
|---|---|
| `node_id` | 节点唯一 ID |
| `name` | 管理端显示名称 |
| `protocol` | 节点协议，例如 WireGuard、VLESS、Trojan、HTTP CONNECT、SOCKS5 |
| `listen_profile` | 对应本机入口或内部出站 profile |
| `egress_group` | 所属出口组 |
| `country_code` | 预期出口国家或地区 |
| `expected_asn` | 可选，预期 ASN |
| `expected_isp` | 可选，预期 ISP |
| `priority` | 主备优先级 |
| `weight` | 同优先级权重 |
| `status` | active、inactive、draining |
| `health_status` | healthy、warn、critical、unknown |
| `last_exit_ip` | 最近检测到的出口 IP |
| `last_checked_at` | 最近检测时间 |
| `last_error` | 最近错误摘要，必须脱敏 |

## 7. 平台域名分流要求

### 7.1 OpenAI / ChatGPT

OpenAI 平台出口组至少覆盖：

```text
api.openai.com
chatgpt.com
chat.openai.com
auth.openai.com
platform.openai.com
oaiusercontent.com
oaistatic.com
openai.com
```

说明：

- `api.openai.com` 是 Gateway API 转发的核心目标。
- ChatGPT Web 相关域名是否启用由业务场景决定；若 SuperXray-gui 同时服务浏览器场景，应单独纳入规则。
- OpenAI 域名规则需要有版本号和更新时间，便于后续跟踪变更。

### 7.2 Anthropic / Claude

Anthropic 平台出口组至少覆盖：

```text
api.anthropic.com
anthropic.com
claude.ai
console.anthropic.com
```

说明：

- `api.anthropic.com` 是 Gateway API 转发的核心目标。
- Claude Web 域名是否启用由业务场景决定。

### 7.3 Google / Gemini

Google/Gemini 平台出口组至少覆盖：

```text
generativelanguage.googleapis.com
cloudcode-pa.googleapis.com
aiplatform.googleapis.com
oauth2.googleapis.com
www.googleapis.com
accounts.google.com
ai.google.dev
gemini.google.com
```

说明：

- `generativelanguage.googleapis.com` 对应 AI Studio / Gemini API。
- `cloudcode-pa.googleapis.com` 对应 Gemini Code Assist。
- `aiplatform.googleapis.com` 对应 Vertex AI。
- OAuth 和资源管理相关 Google 域名需要与 token 刷新、项目识别链路一起验证。

### 7.4 域名规则维护要求

SuperXray-gui 需要提供：

- 平台域名规则文件。
- 规则来源说明。
- 规则更新时间。
- 规则变更审查记录。
- 回滚到上一版规则的方式。

不得只使用单条全局规则：

```text
MATCH -> PROXY
```

必须能证明平台流量和普通流量被区分处理。

## 8. 国家与地区路由策略

### 8.1 Gateway 到 SuperXray-gui 的映射方式

当前 Gateway 通过代理记录表达出口：

```text
protocol: socks5h
host: 127.0.0.1
port: 1080
```

或者：

```text
protocol: http
host: 127.0.0.1
port: 8081
```

在 SuperXray-gui 侧，需要支持把不同本机入口或不同认证参数映射到不同出口组。推荐两种方式：

### 8.2 方案 A：多本机端口映射

示例：

| Gateway 代理 | SuperXray-gui 出口 |
|---|---|
| `socks5h://127.0.0.1:11801` | `openai-egress` |
| `socks5h://127.0.0.1:11802` | `anthropic-egress` |
| `socks5h://127.0.0.1:11803` | `gemini-egress` |
| `socks5h://127.0.0.1:11901` | `region-us` |
| `socks5h://127.0.0.1:11981` | `region-jp` |

优点：

- Gateway 只需登记不同本机端口。
- 路由关系直观。
- 易于灰度和回滚。

缺点：

- 本机端口较多，需要统一端口规划。

### 8.3 方案 B：单入口加内部域名规则

示例：

```text
socks5h://127.0.0.1:1080
```

SuperXray-gui 根据目标域名判断：

```text
api.openai.com -> openai-egress
api.anthropic.com -> anthropic-egress
generativelanguage.googleapis.com -> gemini-egress
```

优点：

- Gateway 代理记录少。
- 运维入口简单。

缺点：

- 仅靠域名难以表达“账号区域”。
- 如果一个平台需要多个地区出口，仍需要额外 profile 或端口。

### 8.4 推荐方案

建议采用“多本机端口 + 平台域名规则”的混合模式：

```text
平台专属端口
  -> 平台出口组
  -> 平台域名规则二次校验

地区专属端口
  -> 地区出口组
  -> 出口国家检测二次校验
```

这样 Gateway 可把账号区域绑定到明确的本机代理端口，而 SuperXray-gui 仍然可以在内部防止目标域名误分流。

## 9. DNS 策略

### 9.1 基本原则

- 平台域名解析必须与实际出口地区一致。
- SOCKS5 场景优先远端 DNS 解析。
- 避免本地 DNS 解析出与出口地区不一致的结果。
- 避免国内 DNS 污染影响 AI 平台域名。
- DNS 查询日志不得记录用户敏感请求内容。

### 9.2 推荐策略

| 场景 | 策略 |
|---|---|
| Gateway 使用 SOCKS5H | 域名交给 SuperXray-gui 或出口端解析 |
| Gateway 使用 HTTP CONNECT | SuperXray-gui 内部按域名建立隧道 |
| 平台域名 | 使用可信远端 DNS 或出口所在地 DNS |
| 本地域名和内网地址 | 禁止走平台出口 |
| 私网地址 | 默认阻断，除非明确白名单 |

### 9.3 DNS 验收

SuperXray-gui 项目组需提供：

- 每个平台域名的解析路径说明。
- DNS 是否在本地、代理端或远端出口解析。
- DNS 泄漏检查结果。
- 异常 DNS 时的降级策略。

## 10. 健康检查与探测

### 10.1 基础连通性探测

每个出口节点必须周期性探测：

```text
出口 IP
国家/地区
ASN/ISP
基础延迟
最近错误
```

可使用多个低敏探测目标，避免单点误判。

### 10.2 平台可达性探测

每个平台出口组必须探测对应 API 目标：

| 平台 | 探测目标 | 可接受结果 |
|---|---|---|
| OpenAI | `https://api.openai.com/v1/models` | `401` 可视为网络可达 |
| Anthropic | `https://api.anthropic.com/v1/messages` | `401`、`404`、`405`、`400` 可视为网络可达 |
| Gemini | `https://generativelanguage.googleapis.com/$discovery/rest?version=v1beta` | `200` 可视为网络可达 |

注意：

- 探测不得使用生产用户请求体。
- 探测不得泄漏真实上游 API Key。
- 401/403 等状态需要区分“网络可达但未授权”和“出口不可用”。

### 10.3 流式稳定性探测

AI 服务大量使用长连接和流式输出。SuperXray-gui 应增加：

- 长连接保持测试。
- 60 秒、180 秒、600 秒分档连接保持测试。
- 中途断流率统计。
- 入口到出口的连接复用情况统计。

### 10.4 熔断策略

建议状态机：

```text
healthy
  -> warn
  -> critical
  -> quarantined
  -> recovering
  -> healthy
```

触发建议：

- 连续 3 次平台探测失败：进入 `warn`。
- 连续 5 次失败或 5 分钟不可用：进入 `critical`。
- 出口国家与预期国家不一致：进入 `critical`。
- 出口 IP 在短窗口内频繁变化：进入 `warn` 或 `critical`。
- 人工确认风险节点：进入 `quarantined`。

恢复建议：

- 连续 3 次探测恢复后进入 `recovering`。
- `recovering` 期间只承载少量新请求。
- 稳定 10 分钟后回到 `healthy`。

## 11. 出口粘性与切换原则

### 11.1 粘性目标

- 同一上游账号尽量固定命中同一出口组。
- 同一账号区域尽量固定命中同一国家/地区出口。
- 同一平台请求不应在短时间内跨国家漂移。
- 流式请求进行中不得强制切换出口。

### 11.2 切换规则

允许切换：

- 主出口不可用。
- 出口健康进入 `critical`。
- 出口国家检测不匹配。
- 管理员手动切换。
- 灰度发布计划明确要求切换。

禁止切换：

- 单次请求失败立即跨国家切换。
- 流式请求中途强制切换。
- 未经健康检查的新节点直接承载生产流量。
- 同一账号在短时间内频繁轮换出口 IP。

### 11.3 回切规则

- 主出口恢复后不得立即全量回切。
- 先进入 `recovering`。
- 灰度比例建议：`5% -> 20% -> 50% -> 100%`。
- 每阶段至少观察 10 分钟。
- 如果错误率高于备用出口，立即停止回切。

## 12. 与 Gateway 的配置对接

### 12.1 Gateway 需要从 SuperXray-gui 获得的信息

每个可登记出口需要给出：

| 字段 | 示例 |
|---|---|
| `name` | `xray-openai-us-primary` |
| `protocol` | `socks5h` |
| `host` | `127.0.0.1` |
| `port` | `11801` |
| `username` | 可选 |
| `password` | 可选 |
| `platform` | `openai`、`anthropic`、`gemini` 或空 |
| `region_code` | `US`、`JP`、`HK` 等 |
| `expected_country_code` | `US` |
| `health_status` | `healthy` |
| `last_exit_ip` | 脱敏展示或仅管理员可见 |
| `notes` | 运维备注 |

### 12.2 Gateway 代理登记建议

平台专属代理：

```text
openai-us-primary
  protocol = socks5h
  host = 127.0.0.1
  port = 11801

anthropic-us-primary
  protocol = socks5h
  host = 127.0.0.1
  port = 11802

gemini-us-primary
  protocol = socks5h
  host = 127.0.0.1
  port = 11803
```

地区专属代理：

```text
region-us-primary
  protocol = socks5h
  host = 127.0.0.1
  port = 11901

region-jp-primary
  protocol = socks5h
  host = 127.0.0.1
  port = 11981
```

### 12.3 账号创建联动

`frontend/src/components/account/CreateAccountModal.vue` 创建账号时已经包含：

- `proxy_id`
- `region_code`

Gateway 侧运行时规则：

```text
账号手动 proxy_id 存在
  -> 使用该代理

账号无 proxy_id 且 region_code 存在
  -> 使用该区域绑定的代理池

账号无 proxy_id 且无 region_code
  -> 保持旧行为
```

SuperXray-gui 需要保证这些代理端口背后的出口组稳定可用。

## 13. 配置示例

以下为逻辑示例，不限定 SuperXray-gui 必须使用某一种内核。无论底层是 Xray、sing-box、mihomo、Clash Meta 或其他实现，都需要提供等价能力。

### 13.1 入口示例

```yaml
inbounds:
  - tag: gateway-openai-socks
    type: socks
    listen: 127.0.0.1
    listen_port: 11801

  - tag: gateway-anthropic-socks
    type: socks
    listen: 127.0.0.1
    listen_port: 11802

  - tag: gateway-gemini-socks
    type: socks
    listen: 127.0.0.1
    listen_port: 11803

  - tag: gateway-region-jp-socks
    type: socks
    listen: 127.0.0.1
    listen_port: 11981
```

### 13.2 出口组示例

```yaml
outbounds:
  - tag: openai-egress-primary
    type: wireguard
    meta:
      platform: openai
      country_code: US

  - tag: openai-egress-backup
    type: socks
    meta:
      platform: openai
      country_code: US

  - tag: anthropic-egress-primary
    type: wireguard
    meta:
      platform: anthropic
      country_code: US

  - tag: gemini-egress-primary
    type: socks
    meta:
      platform: gemini
      country_code: US

  - tag: region-jp-primary
    type: vless
    meta:
      region_code: JP
```

### 13.3 路由示例

```yaml
route:
  rules:
    - inbound: gateway-openai-socks
      domain:
        - api.openai.com
        - chatgpt.com
        - chat.openai.com
      outbound: openai-egress-primary

    - inbound: gateway-anthropic-socks
      domain:
        - api.anthropic.com
        - claude.ai
      outbound: anthropic-egress-primary

    - inbound: gateway-gemini-socks
      domain:
        - generativelanguage.googleapis.com
        - cloudcode-pa.googleapis.com
        - aiplatform.googleapis.com
      outbound: gemini-egress-primary

    - inbound: gateway-region-jp-socks
      outbound: region-jp-primary

    - inbound:
        - gateway-openai-socks
        - gateway-anthropic-socks
        - gateway-gemini-socks
        - gateway-region-jp-socks
      outbound: reject
```

关键要求：

- 平台入口必须只允许匹配的平台域名进入对应出口。
- 地区入口可以按业务需要只承载 AI 平台域名，也可以承载 Gateway 指定目标，但必须记录策略。
- 未匹配流量默认拒绝，不要静默走 DIRECT。

## 14. 安全要求

### 14.1 网络暴露面

必须关闭公网暴露：

- `1080`
- `8081`
- 平台专属本机代理端口
- 地区专属本机代理端口
- 数据库端口
- Redis 端口
- Gateway 内部管理端口

### 14.2 凭据安全

- 节点凭据不得写入普通日志。
- 导出配置时必须脱敏。
- 截图、工单、日报中不得出现真实代理密码、token、私钥。
- 本机配置文件权限应限制为服务账号可读。

### 14.3 请求日志

允许记录：

- 平台
- 出口组
- 出口节点 ID
- 目标域名
- 状态码
- 延迟
- 错误分类
- 脱敏后的出口 IP

禁止记录：

- 用户 Prompt。
- 完整请求体。
- Authorization header。
- Cookie。
- 上游 API Key。
- OAuth token。

## 15. 监控指标

SuperXray-gui 至少需要暴露或可查询以下指标：

| 指标 | 说明 |
|---|---|
| `egress_probe_success` | 出口探测是否成功 |
| `egress_probe_latency_ms` | 出口探测延迟 |
| `egress_ip_change_total` | 出口 IP 变化次数 |
| `egress_country_mismatch_total` | 出口国家不匹配次数 |
| `egress_switch_total` | 出口切换次数 |
| `egress_active_connections` | 活跃连接数 |
| `egress_stream_interrupt_total` | 流式连接中断数 |
| `egress_target_error_total` | 按目标平台聚合的错误数 |
| `egress_dns_error_total` | DNS 错误数 |
| `egress_recovering_nodes` | 恢复中节点数量 |

建议日志字段：

```json
{
  "time": "2026-05-16T10:00:00+08:00",
  "event": "egress_probe",
  "egress_group": "openai-egress",
  "node_id": "openai-us-primary",
  "platform": "openai",
  "expected_country_code": "US",
  "actual_country_code": "US",
  "latency_ms": 182,
  "status": "healthy",
  "error": ""
}
```

## 16. 告警规则

### 16.1 P1 告警

- `openai-egress` 主备均不可用。
- `anthropic-egress` 主备均不可用。
- `gemini-egress` 主备均不可用。
- 本机代理端口误监听到 `0.0.0.0`。
- 出口国家与账号区域不一致且仍承载生产流量。
- 大量请求静默走 DIRECT。

### 16.2 P2 告警

- 单平台出口连续 5 分钟错误率高于 20%。
- 出口 IP 在 30 分钟内变化超过 2 次。
- 单出口延迟高于 24 小时均值 2 倍并持续 10 分钟。
- DNS 错误连续出现。
- 流式中断率持续升高。

### 16.3 P3 告警

- 备用出口长期未探测。
- 健康检查数据超过 10 分钟未更新。
- 规则文件超过 30 天未审查。
- 出口组中存在 inactive 节点但仍被路由引用。

## 17. 发布与回滚流程

### 17.1 发布前检查

- 备份当前 SuperXray-gui 配置。
- 校验配置语法。
- 校验本机端口监听地址。
- 校验平台域名规则。
- 校验地区出口国家。
- 校验主备节点均可探测。
- 准备一键回滚配置。

### 17.2 灰度发布

建议顺序：

```text
测试端口
  -> 单平台测试账号
  -> 单区域测试账号
  -> 低流量生产账号
  -> 单平台 10%
  -> 单平台 50%
  -> 全量
```

### 17.3 回滚条件

出现以下任一情况，应立即回滚：

- 平台探测持续失败。
- 出口国家不匹配。
- Gateway 侧 5xx 或 timeout 明显上升。
- 流式中断明显上升。
- 本机代理端口误暴露。
- DNS 解析异常导致目标平台不可达。

### 17.4 回滚动作

- 切回上一版 SuperXray-gui 路由配置。
- 切回旧的本机代理端口映射。
- 保留故障期间日志和探测结果。
- 通知 Gateway 项目组暂停新增区域代理绑定。

## 18. 联调计划

### 18.1 联调环境

需要准备：

- 一台同服部署机器。
- Super-Code-Gateway。
- SuperXray-gui。
- PostgreSQL 与 Redis。
- 至少 3 个本机代理端口：
  - OpenAI 测试出口
  - Anthropic 测试出口
  - Gemini 测试出口
- 至少 2 个地区出口：
  - `US`
  - `JP`

### 18.2 Gateway 侧操作

1. 在管理后台创建代理：
   - `openai-us-primary`
   - `anthropic-us-primary`
   - `gemini-us-primary`
   - `region-us-primary`
   - `region-jp-primary`
2. 分别执行代理测试和质量检查。
3. 创建账号区域 `US`、`JP`。
4. 为 `US`、`JP` 绑定对应代理。
5. 创建测试账号，设置 `region_code`。
6. 调用 `/v1/messages`、`/v1/responses`、`/chat/completions`、`/v1beta/models`。

### 18.3 SuperXray-gui 侧验证

需要确认：

- 收到来自 Gateway 的本机代理连接。
- 平台入口命中对应平台出口组。
- 地区入口命中对应地区出口组。
- 出口 IP 与预期国家一致。
- 主出口断开后，新请求切到备用出口。
- 流式请求不中途被强制切换。
- 所有切换均有审计日志。

## 19. 验收清单

### 19.1 MVP 必须通过

MVP 只验收 Xray 兼容配置生成、本机入口和 Gateway 对接清单，不验收完整治理平台能力。

- [ ] 可在 Xray 模板中生成 Gateway-facing SOCKS/HTTP inbound。
- [ ] 生成的默认 inbound 只监听 `127.0.0.1` 或实施方案指定的受控内网地址。
- [ ] 生成器支持 `listenHost` 与 `manifestHost` 两个独立参数，并禁止混用。
- [ ] 生成端口包含 `11801`、`11802`、`11803`、`11901`、`11981` 或经评审确认的等价规划。
- [ ] 生成 OpenAI、Anthropic、Gemini 平台域名规则，并避免全局 `MATCH -> PROXY`。
- [ ] 生成未匹配流量的最终拒绝规则，避免本机入口变成通用开放代理。
- [ ] 生成 Gateway 代理登记 CSV，包含协议、host、端口、平台、区域、预期国家和出口组。
- [ ] 已按第 4.1 节选择 Docker 同服网络方案，CSV 中的 `manifestHost` 对 Gateway 容器实际可达。
- [ ] Gateway 可登记并访问 SuperXray-gui 生成的本机代理入口。
- [ ] OpenAI、Anthropic、Gemini 基础网络探测可达，失败时有明确错误类型。
- [ ] 代理凭据、token、UUID、私钥、公钥和住宅出口密码不进入日志、截图、CSV 或前端可见状态。
- [ ] Xray 配置保存前可回退到旧模板，保存后可通过既有 x-ui/Xray 流程重启验证。

### 19.2 Phase 10+ 必须通过

以下能力不作为 MVP 验收项，只在 Phase 10+ 出口治理子系统立项后验收：

- [ ] OpenAI、Anthropic、Gemini 至少各有独立出口组。
- [ ] 至少支持 `US`、`JP` 两个地区出口组。
- [ ] 出口 IP 和国家检测有记录。
- [ ] 出口国家不匹配时不会被标记为健康。
- [ ] 节点健康检查、熔断、切换和回切有持久化记录。
- [ ] 主备出口切换可演练。
- [ ] 回滚流程可演练。
- [ ] 支持按出口组查看健康状态。
- [ ] 支持按节点查看最近错误。
- [ ] 支持出口 IP 变化告警。
- [ ] 支持规则版本管理。

### 19.3 建议通过

- [ ] 支持多本机端口映射不同平台和地区。
- [ ] 支持配置语法校验。
- [ ] 支持灰度切换和延迟回切。
- [ ] 支持导出脱敏运行态审查报告，便于 Gateway 项目组交叉验证。

## 20. 交付物清单

SuperXray-gui 项目组需要交付：

1. VPN/代理出口组配置文件。
2. 平台域名规则文件。
3. 国家/地区出口组配置文件。
4. 本机代理端口规划表。
5. 出口健康检查脚本或内置任务说明。
6. 出口 IP 与国家检测报告。
7. 主备切换演练报告。
8. 回滚操作说明。
9. 安全检查报告。
10. 与 Gateway 对接的代理登记清单。

## 21. 代理登记清单模板

```csv
name,protocol,host,port,platform,region_code,expected_country_code,egress_group,health_status,notes
openai-us-primary,socks5h,127.0.0.1,11801,openai,US,US,openai-egress,healthy,OpenAI 主出口
anthropic-us-primary,socks5h,127.0.0.1,11802,anthropic,US,US,anthropic-egress,healthy,Anthropic 主出口
gemini-us-primary,socks5h,127.0.0.1,11803,gemini,US,US,gemini-egress,healthy,Gemini 主出口
region-jp-primary,socks5h,127.0.0.1,11981,,JP,JP,region-jp,healthy,日本区账号出口
```

## 22. 工单模板

标题：

```text
实现 Super-Code-Gateway 同服部署的 SuperXray-gui VPN 出口治理层
```

背景：

```text
Super-Code-Gateway 已支持账号手动代理和账号区域代理池。现在需要 SuperXray-gui 提供本机代理入口、平台专属出口组、国家/地区出口组、健康检查、熔断、监控和回滚能力，使 Gateway 可以稳定、合规地通过本机代理访问 OpenAI、Anthropic、Google/Gemini 等平台。
```

范围：

```text
- 本机 SOCKS5/HTTP 入口
- 平台出口组
- 地区出口组
- 域名规则
- DNS 策略
- 健康检查
- 主备切换
- 安全加固
- Gateway 代理登记清单
```

验收：

```text
- 本机代理不公网暴露
- OpenAI/Anthropic/Gemini 平台出口可独立配置
- US/JP 地区出口可独立配置
- 出口 IP 与国家检测可审计
- 主备切换和回滚演练通过
- Gateway 通过登记代理可完成平台 API 探测
```

## 23. 风险与注意事项

| 风险 | 影响 | 处理方式 |
|---|---|---|
| 本机代理公网暴露 | 变成开放代理，被滥用 | 强制监听 `127.0.0.1`，防火墙巡检 |
| DNS 泄漏 | 域名解析与出口地区不一致 | 使用远端 DNS 或代理端解析 |
| 出口频繁变化 | 账号风险和体验波动 | 粘性绑定、主备策略、延迟回切 |
| 平台域名规则过期 | 请求误分流或失败 | 规则版本化、定期审查 |
| 探测误判 | 错误切换或错误熔断 | 多目标探测、错误分类 |
| 凭据泄漏 | 安全事故 | 日志脱敏、权限收敛 |
| 全局 MATCH 规则 | 普通流量和 AI 流量混用 | 平台和地区规则精细化 |

## 24. 参考依据

### 24.1 内部依据

- `OPENAI_GATEWAY_TOPOLOGY_CN.md`
- `docs/OPENAI_ACCESS_STABILITY_PLAN_CN.md`
- `docs/superpowers/specs/2026-05-14-account-region-proxy-routing-design.md`
- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/api/admin/accountRegions.ts`
- `frontend/src/types/index.ts`

### 24.2 外部官方依据

- OpenAI API Reference：`https://platform.openai.com/docs/api-reference`
- Anthropic Messages API：`https://docs.anthropic.com/en/api/messages-examples`
- Google Gemini API Reference：`https://ai.google.dev/api/rest/generativelanguage`

说明：AI 平台域名和 API 路径可能随官方产品调整而变化。SuperXray-gui 项目组应将平台域名规则做成版本化配置，并在发布前以官方文档和实际探测结果双重校验。

## 25. 最终交接结论

SuperXray-gui 项目组需要把 VPN 层从“可连接的代理工具”升级为“可治理的出口平台”：

- 对 Gateway 暴露稳定的本机代理入口。
- 对内部维护平台和地区出口组。
- 对运维提供健康、熔断、切换、回滚和审计。
- 对安全保证端口、凭据、DNS 和日志边界。

当前阶段的可执行结论分两步：

1. MVP 阶段先交付 Xray 兼容的 Gateway-facing 本机入口、平台/地区端口规划、域名规则和 Gateway CSV。该阶段不承诺完整健康检查、熔断、审计和自动切换。
2. Phase 10+ 阶段再建设完整出口治理子系统，补齐出口组、节点、探测结果、切换事件、规则版本和回滚审计。

只有当第 4.1 节的 Docker 同服网络方案完成并通过 Gateway 连通性验证后，Super-Code-Gateway 才能登记 SuperXray-gui 提供的代理地址，并通过账号 `proxy_id` 或 `region_code` 选择出口，实现按账号地区和 AI 平台的稳定出口访问。
