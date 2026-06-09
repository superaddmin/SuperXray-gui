# OpenWrt 主 WiFi / USA WiFi AI-GFW 分流设计

**日期：** 2026-06-09

## 目标

将 OpenWrt 当前的代理结构重构为：

- 主 WiFi（OpenWrt-2.4G / OpenWrt-5G，`lan` / `192.168.5.0/24`）以 `85.155.178.115` 为主，仅对 AI / GFW 相关域名走代理，其余流量直连。
- USA WiFi（USA-US / USA-US-2G，`us` / `192.168.52.0/24`）保持单独网段，继续通过 `35.87.239.230` 进行全代理访问。
- 变更必须可回滚、在线可验证，并避免再次引入 DNS 超时或默认 SOCKS 复用导致的出口漂移。

## 当前事实

### OpenWrt 网络角色

- `lan`：`192.168.5.1/24`
- `us`：`192.168.52.1/24`（USA-US）
- `jp`：`192.168.51.1/24`
- 当前客户端 `DESKTOP-9UPVU52` 在 `192.168.52.193`，即位于 `us` 网段。

### 当前 Passwall 状态

- `@global[0].tcp_node='wyv8VRj6'` -> `35.87.239.230`
- `P54yqxIj.node='wyv8VRj6'` -> 本地 SOCKS `1081`
- `@acl_rule[0].interface='us'`
- `@acl_rule[0].tcp_node='t838Jhr9'` -> `35.87.239.230`
- `us` 当前是 TCP/UDP 全代理。
- OpenClash 配置启用但服务 inactive，当前实际生效的是 Passwall。

### 已知问题

- 直接把 `us` 的 ACL 切到 `85.155.178.115` 不能稳定改善体验。
- 把默认 SOCKS / 全局链一起切到 `85.155.178.115` 时，OpenWrt 本地 SOCKS 出口确实变成 85，但透明代理路径出现 DNS 超时和本机访问异常。
- 因此现有问题不是单一节点切换问题，而是当前透明代理链路与默认链/ACL/DNS 耦合过深。

## 设计原则

1. **主 WiFi 和 USA WiFi 职责分离**：
   - `lan` 承担日常主网络
   - `us` 保持独立美国代理 WiFi
2. **主 WiFi 默认直连**：
   - 不再对 `lan` 做全代理
   - 避免普通网站和下载被无谓拖慢
3. **AI / GFW 域名走 85**：
   - 主 WiFi 在命中代理域名时转发到 `85.155.178.115`
4. **USA WiFi 继续走 35**：
   - 保持其独立用途，不混入主 WiFi 路由
5. **DNS 采用最小侵入方案**：
   - 不再把主 WiFi 所有查询一刀切送入全局代理链
   - 只对代理域名走 Passwall 的代理 DNS 逻辑

## 目标结构

### 主 WiFi (`lan`)

- 出口默认：本地原生线路
- AI / GFW 域名：Passwall -> `85.155.178.115`
- 默认 `ipify`：应为本地原生出口
- `openai.com` / `chatgpt.com` / `anthropic.com` 等：应可走 85

### USA WiFi (`us`)

- 出口：保持 `35.87.239.230`
- 使用场景：单独 USA 代理 WiFi
- `ipify`：应继续是 `35.87.239.230`

## 代理域名集

主 WiFi 仅代理 AI / GFW 相关域名。首批最小集合：

- `openai.com`
- `chatgpt.com`
- `oaistatic.com`
- `oaiusercontent.com`
- `anthropic.com`
- `claude.ai`
- `aistudio.google.com`
- `generativelanguage.googleapis.com`
- `makersuite.google.com`
- `gemini.google.com`
- `x.ai`
- `grok.com`
- `perplexity.ai`

实现上优先复用 Passwall 现有 proxy/gfw 列表入口；若需补充则追加到自定义代理域名列表，而不是覆盖项目默认规则集。

## 实施方式

1. 备份 `/etc/config/passwall`
2. 读取 Passwall 当前 `lan` / `us` ACL、代理列表、自定义域名列表入口
3. 为 `lan` 增加一条新的 ACL：
   - `interface='lan'`
   - `sources='192.168.5.0/24'`
   - 节点使用 `85.155.178.115`
   - `tcp_proxy_mode` / `udp_proxy_mode` 使用“仅代理列表/仅 GFW”模式，而非 `proxy`
4. 保留 `us` ACL 指向 `35.87.239.230`
5. 不改动 OpenClash，仅作用于 Passwall
6. 在线验证；若出现 DNS 异常或主 WiFi 普通网站异常，立即回滚

## 验证标准

### 主 WiFi 验证

- `ipify`：不是 `35.87.239.230`，也不应固定为 `85.155.178.115`
- 普通网站直连：`google generate_204` 正常
- AI / GFW：`api.openai.com` / `chatgpt.com` 可访问
- 下载速度：相对当前全量透明代理状态应改善

### USA WiFi 验证

- `ipify`：仍为 `35.87.239.230`
- 作为独立代理 WiFi 保持可用

## 回滚

若主 WiFi 出现以下任一情况则立即回滚：

- DNS 超时
- 普通网站普遍打不开
- AI 域名访问更差
- Passwall 生成的透明代理链路不稳定

回滚方式：恢复变更前的 `/etc/config/passwall` 备份并重启 Passwall。
