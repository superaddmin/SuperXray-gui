# Gateway 生图超时治理执行手册

> 适用范围：SuperXray-gui 负责的 Xray 侧 Gateway-facing 出口、Gateway 登记清单、测试服务器验证和生产灰度发布。
> 当前边界：只使用既有 Xray 模板保存路径和 legacy Xray 生命周期，不新增 `egress_*` 表，不迁移 `model.Inbound`，不让 CoreManager 接管生产 Xray。

## 1. 执行目标

把生图请求的 80 秒级超时拆成可观测链路，并优先修复最可能的阻塞点：

```text
Client
  -> Super-Code-Gateway
  -> Gateway 代理选择
  -> manifestHost:11801/11802/11803/11901/11981
  -> Xray gateway-* SOCKS inbound
  -> platform egress outbound
  -> AI 平台 API / 后端模型推理
  -> 响应返回
```

测试服务器通过后，才允许把同一变更以灰度方式复制到生产服务器。

## 2. 多角色审查闸门

执行前必须完成三类审查：

| 角色 | 必须确认 |
|---|---|
| 架构审查 | 方案不突破当前阶段边界；Gateway Egress MVP 仍只生成 Xray 模板和 CSV；真实出口由现有 Xray outbound 承载 |
| 安全审查 | Gateway-facing 端口不公网暴露；非 loopback 监听必须有源地址限制；CSV 不含代理密码、token、UUID、私钥；日志不记录 Prompt、Authorization、Cookie |
| 验证审查 | 测试服务器能收集端口监听、容器可达性、socks5h timing、Xray 统计、回滚证据 |

任一审查发现阻断项时，不进入测试服务器执行。

生产前的硬阻断项：

- `gateway-*` SOCKS inbound 使用 `noauth` 时，若监听地址不是 `127.0.0.1` / `::1`，必须提供防火墙、安全组或容器网络策略证据，只允许 Gateway 容器 IP 或网段访问。
- Docker bridge 场景不得把 Gateway CSV 的 `manifestHost` 写成 `127.0.0.1`，除非 Gateway 与 Xray 处于同一网络命名空间。
- `openai-egress`、`anthropic-egress`、`gemini-egress` 不得保留 `_gatewayEgressMvp` 的 `freedom` 占位出口。
- `outboundTestUrl`、`observatory.probeURL`、Gateway 健康检查 URL 必须固定 allowlist，禁止访问 localhost、RFC1918、link-local、metadata 和可疑内网域名。
- 主 Xray 与 Gateway 日志不得记录 Prompt、Authorization、Cookie、API Key、代理密码或私钥。
- 测试服务器必须完成“备份、部署、验证、回滚、恢复验证”的闭环后，才允许进入生产灰度。

## 3. 测试服务器前置条件

测试服务器需要准备：

- SuperXray-gui / x-ui 管理面板可登录。
- Gateway 测试实例或测试容器名称。
- 选定网络策略：
  - 同网络命名空间：`listenHost=127.0.0.1`，`manifestHost=127.0.0.1`。
  - Docker bridge + 宿主机 x-ui：`listenHost` 为宿主机受控监听地址，`manifestHost` 为 Gateway 容器可达地址，例如 `host.docker.internal` 或宿主网桥地址。
- `openai-egress`、`anthropic-egress`、`gemini-egress` 的真实 outbound 配置。不得保留 MVP 生成的 `freedom` 占位。
- 测试账号或测试 API Key。验证脚本本身不需要也不会打印 API Key。
- 可回滚的 Xray 配置备份和数据库备份。
- 若使用 Docker bridge 或受控内网地址，准备 Gateway 容器 IP/网段和对应防火墙规则证据。

## 4. 测试服务器实施步骤

### 4.1 生成 Gateway-facing Xray 入口

在新 UI 的 Xray 页面执行：

1. 在 Gateway 出口 MVP 中设置 `listenHost`、`manifestHost` 和策略标签。
2. 生成 Xray 配置。
3. 保存 Xray 模板。
4. 手动重启 Xray。

预期生成端口：

| 端口 | 用途 | 出口组 |
|---:|---|---|
| `11801` | OpenAI | `openai-egress` |
| `11802` | Anthropic | `anthropic-egress` |
| `11803` | Gemini | `gemini-egress` |
| `11901` | US 区域 | `region-us` |
| `11981` | JP 区域 | `region-jp` |

### 4.2 替换真实出口

检查 Xray 模板中平台出口：

- `openai-egress`
- `anthropic-egress`
- `gemini-egress`

这些 outbound 不得继续是带 `_gatewayEgressMvp` 标记的 `freedom` 占位。生产前至少应有主出口；推荐每个平台 1 主 1 备，并通过手工或后续治理层实现切换。

### 4.3 启用统计、DNS 与 Observatory

在 Xray 模板中启用：

- `policy.system.statsOutboundUplink=true`
- `policy.system.statsOutboundDownlink=true`
- AI 域名 DNS 使用可信 DoH 或出口侧解析。
- `observatory` 或 `burstObservatory` 覆盖真实 outbound tag。

SOCKS 场景下，Gateway 侧登记协议应使用 `socks5h`，避免在 Gateway 容器内提前本地解析 AI 域名。

### 4.4 Gateway 超时和连接池策略

Gateway 侧建议把超时拆开配置：

| 阶段 | 建议值 |
|---|---:|
| 连接池等待 | `1s` 到 `2s` |
| 代理 TCP connect | `3s` 到 `5s` |
| SOCKS/HTTP CONNECT | `5s` |
| TLS handshake | `5s` |
| 请求头/首字节等待 | 按平台设置，建议单独记录 |
| 生图任务总超时 | 同步请求 `120s` 到 `300s`；更推荐异步任务 |

原则：不得让请求在“拿连接/连接代理”阶段消耗完整 80 秒。若 `connect` 或 `CONNECT` 超过阈值，应快速失败并切换备用出口或返回明确错误。

### 4.5 生图异步化

测试服务器上若 Gateway 支持异步生图，应采用：

```text
POST /image/jobs
  -> 返回 job_id
GET /image/jobs/:job_id
  -> 查询状态、结果或错误
```

同步链路只用于短请求和状态查询；长时间模型推理不应占用 Gateway 入口连接池。

## 5. 只读验证脚本

在测试服务器仓库目录运行：

```bash
chmod +x scripts/gateway-egress-validate.sh

GATEWAY_EGRESS_HOST=host.docker.internal \
GATEWAY_CONTAINER=<gateway-container-name> \
GATEWAY_ALLOWED_SOURCE_CIDR=<gateway-container-cidr> \
XRAY_CONFIG=/usr/local/x-ui/bin/config.json \
scripts/gateway-egress-validate.sh
```

同网络命名空间可省略 `GATEWAY_CONTAINER`，并使用：

```bash
GATEWAY_EGRESS_HOST=127.0.0.1 \
XRAY_CONFIG=/usr/local/x-ui/bin/config.json \
scripts/gateway-egress-validate.sh
```

生产预检查应显式启用生产模式：

```bash
GATEWAY_SECURITY_MODE=production \
GATEWAY_EGRESS_HOST=<gateway-reachable-host> \
GATEWAY_CONTAINER=<gateway-container-name> \
GATEWAY_ALLOWED_SOURCE_CIDR=<gateway-container-cidr> \
XRAY_CONFIG=/usr/local/x-ui/bin/config.json \
scripts/gateway-egress-validate.sh
```

脚本只做只读检查：

- `ss` 检查 Gateway-facing 端口监听。
- 检查通配监听、非 loopback 无认证 SOCKS、源地址限制证据。
- `jq` 检查 gateway inbounds、blocked 保护规则、outbound 统计、observatory 和占位 outbound。
- 检查 `observatory` / `burstObservatory` 探测 URL 是否指向明显私网或本地地址。
- `curl --socks5-hostname` 输出 `connect/tls/ttfb/total` timing。
- 使用 `https://example.com` 做负向探测，确认平台入口不会退化成通用开放代理。
- 可选 `docker exec` 从 Gateway 容器内验证 `manifestHost` 可达。

## 6. 通过判据

测试服务器必须全部满足：

- `11801/11802/11803/11901/11981` 或评审后的等价端口处于监听状态。
- 端口未监听在 `0.0.0.0`、`::` 或公网地址。
- 非 loopback 监听已提供 Gateway 容器网段源限制证据。
- Gateway 容器内访问 `manifestHost:11801` 可达。
- Xray 模板中不存在平台 `freedom` 占位出口。
- outbound 统计已启用。
- `observatory` 或 `burstObservatory` 已配置。
- 探测 URL 不指向 localhost、私网、link-local 或 metadata 地址。
- OpenAI 专用入口访问非 OpenAI 域名失败或被 blocked。
- OpenAI/Anthropic/Gemini 探测的 `connect` 不接近 80 秒。
- 若总耗时仍高，但 `connect` 和 `tls` 正常，应转入 Gateway 上游请求和模型推理排队排查。
- 回滚步骤已演练或至少命令已准备。

## 7. 生产部署闸门

生产部署前必须具备：

- 测试服务器验证脚本零失败。
- 测试 Gateway 生图流量连续通过，且超时率下降。
- 生产 Xray 配置和数据库已备份。
- Gateway CSV 中 `manifestHost` 已确认对生产 Gateway 可达。
- 防火墙仅允许 Gateway 所在主机或容器网段访问 Gateway-facing 端口。
- `GATEWAY_SECURITY_MODE=production` 的只读验证脚本零失败。
- Xray `access` 关闭或写入受控路径，`dnsLog=false`，`maskAddress` 至少为 `half`，Gateway 日志完成敏感字段脱敏。
- 生产变更窗口和回滚负责人明确。

生产发布顺序：

1. 导入或保存 Xray 模板。
2. 替换真实 outbound。
3. 重启 Xray。
4. 导入 Gateway 代理清单。
5. 只给测试账号或 5% 以下流量启用。
6. 运行只读验证脚本。
7. 观察 Gateway 5xx、timeout、connect timing、Xray outbound stats。
8. 达标后逐步扩大流量。

## 8. 回滚

触发回滚：

- Gateway 5xx 或 timeout 上升。
- 代理端口误暴露到公网。
- 出口国家或平台规则不匹配。
- `connect`、`tls`、`ttfb` 任一阶段出现持续异常。
- Xray 重启失败或配置无法加载。

回滚动作：

1. Gateway 切回旧 proxy_id 或旧区域代理绑定。
2. 恢复上一版 Xray 模板或数据库备份。
3. 重启 Xray。
4. 保留验证脚本输出、Gateway 日志、Xray 日志和 timing 结果。

## 9. 需要人工提供的信息

执行远端测试和生产部署前，需要提供：

- 测试服务器 SSH 连接方式。
- 生产服务器 SSH 连接方式。
- Gateway 测试容器名或部署方式。
- 选定的 `listenHost` 与 `manifestHost`。
- 真实出口 outbound 的脱敏结构或由运维手工替换完成的确认。
- 允许的变更窗口和回滚联系人。
