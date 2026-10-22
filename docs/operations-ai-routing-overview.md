# 服务器部署 + OpenWrt 路由 + AI 出口治理统一总览

> **目标读者**：需要同时维护 SuperXray 服务器节点、OpenWrt/Passwall 分流、AI 平台出口质量与排障流程的开发者 / 运维人员  
> **适用版本**：`v3.5.0`
> **核对日期**：2026-10-22。本仓库实现服务器侧 Xray 模板、订阅和 Gateway CSV；默认运行时 Xray 为 `v26.10.21`（兼容基线 `v26.10.10`），模板保存会执行兼容转换及核心预检，仍须显式重启与运行态验证。OpenWrt/Passwall、SSID、网段及外部 Gateway 均为部署环境，以下拓扑与命令是验证模板，不能证明现场已经配置成功。
> 本文已合并原《OpenWrt / Passwall AI 路由实战手册》的独有内容；该文件不再单独维护。
> **相关文档**：[部署指南](deployment.md) | [AI 平台智能分流与住宅出口运行手册](ai-routing-and-residential-egress.md) | [Gateway 超时修复手册](gateway-egress-timeout-remediation-runbook.md)

---

## 1. 先理解系统分层

这类问题通常不是“单台服务器”问题，而是三层协同：

### 1.1 服务器层

职责：

- 运行 SuperXray 面板
- 维护 Xray/订阅服务
- 提供一个或多个可用出口节点

常见对象：

- `<SERVER_IP>`
- Web 面板
- Sub Server
- Xray / 容器 / systemd 服务

### 1.2 中间设备层

职责：

- OpenWrt / Passwall 做 ACL、透明代理、DNS 分流、SSID 分网

常见对象：

- `lan`
- `us`
- `Passwall`
- `iptables`
- `chinadns-ng`
- `dnsmasq`

### 1.3 终端层

职责：

- 实际连接某个 WiFi
- 观察默认出口
- 访问普通网站 / AI 平台 / IDE 插件

常见对象：

- 主 WiFi（`OpenWrt-5G` / `OpenWrt-2.4G`）
- USA WiFi（`USA-US`）
- `curl ipify`
- `OpenAI / ChatGPT / Claude / Cursor / Copilot`

---

## 2. 推荐职责边界

### 2.1 主 WiFi

- 网段示例：`192.168.5.0/24`，由 OpenWrt 管理员配置
- 目标：
  - 普通流量直连
  - AI / GFW 域名走 `<PRIMARY_PROXY_HOST>`

### 2.2 USA WiFi

- 网段示例：`192.168.52.0/24`，由 OpenWrt 管理员配置
- 目标：
  - 保持全代理
  - 独立承载美国出口需求，绑定 `<US_PROXY_HOST>`

### 2.3 为什么要分开

这些网段和 SSID 只是规划示例。历史上出现过名为 USA 的 SSID 实际仍按域名分流的现场记录，但没有可复核的采样时间和脱敏证据文件，因此不能视为当前部署状态。名称含 USA 不等于全代理，出口国家也不能由节点名称证明。

这样做的收益是：

- 普通网页/下载不被代理拖慢
- AI / 模型接口路径稳定
- 日常网络与美国代理网络互不污染
- 后续排障时可以快速判断问题属于哪一层

### 2.4 代理路径类型

排障前先确认流量走的是哪条路径，三类路径不能互相替代：

- **显式 SOCKS**：客户端或应用显式填写代理地址，流量在应用层进入代理；只代表该端口链路。
- **透明 REDIRECT**：iptables NAT 把命中的 TCP 流量重定向到本地代理端口，需要核对实际链与命中计数。
- **TPROXY**：常用于 UDP 或更完整的透明代理路径，命中后仍保留原始目标信息。
- **DNS 劫持 / 分流**：DNS 决定哪些域名进入代理集合；DNS 规则与 ACL 必须一起检查。

`uci` 配置正确、SOCKS 单次连通、页面首页可打开，都不能证明 WiFi 透明代理或模型接口链路已经生效。

---

## 3. 标准验证顺序

### 3.1 客户端视角

先回答：

- 当前连的是哪个 SSID？
- 当前 IP 属于哪个网段？
- 默认出口是谁？
- 普通网页与 AI 接口是否一致？

最小命令：

```bash
curl --connect-timeout 5 --max-time 30 https://api.ipify.org
curl --connect-timeout 5 --max-time 30 -sS -o /dev/null -w 'HTTP %{http_code}\n' https://api.openai.com/v1/models
curl --connect-timeout 5 --max-time 30 https://chatgpt.com/cdn-cgi/trace
nslookup api.openai.com <gateway-ip>
```

记录实际 SSID、客户端网段和网关链，不能套用上文规划地址。ICMP 路径与 TCP/UDP 代理路径要分别判断，traceroute 不代表 HTTPS 出口；客户端未启用系统代理或 TUN 只能排除部分本机路径，仍需结合 OpenWrt ACL 计数确认是否由网关接管。

### 3.2 中间设备视角

再回答：

- 命中的 ACL 是哪一条？
- 是显式 SOCKS、透明 REDIRECT 还是 TPROXY？
- DNS 劫持与代理表是否一致？

最小命令：

```bash
uci show passwall
iptables -t nat -vnL PSW
iptables -t mangle -vnL PSW
cat /tmp/etc/passwall/var
ps w | grep -E 'passwall|xray run|chinadns-ng'
iwinfo
cat /tmp/dhcp.leases
```

上述链名和缓存路径适用于相应 Passwall/iptables 版本；不存在时先核对安装版本与运行进程，fw4/nftables 环境使用 `nft list ruleset` 核对实际链和计数。采样前后应从目标 WiFi 产生请求，观察计数差值，而不是只看一次静态输出。`uci show` 与进程参数可能包含节点凭据，只在本地受控环境查看，分享前脱敏。

### 3.3 服务器视角

最后确认：

- 节点本身是不是好的？
- 服务器真实出口是谁？
- 面板/Xray 是否正常运行？

最小命令：

```bash
docker exec <container> /app/x-ui -v
docker logs --tail 100 <container>
curl https://api.ipify.org
ss -lntp
```

这是 Docker 部署示例；systemd 部署使用 `systemctl is-active x-ui`、`journalctl -u x-ui -n 100 --no-pager` 与实际安装路径。服务器直连 IP 正常只能证明基础网络，还需通过目标 inbound/outbound 验证真实客户端路径。API 返回 `401` 等通常仅证明 HTTP 可达，不代表账号、模型调用或流式请求成功；GET trace 才能读取回显的 `ip`/`colo`。

### 3.4 订阅与诊断

订阅必须用 GET 拉取当前 `subPath`、`subJsonPath`、`subClashPath` 的实际内容并核对对应 `/diagnose` 输出；不要以 HEAD 的 404 或单次空响应判定订阅故障。路径随订阅设置变化，不能直接套用文档示例。

---

## 4. 常见场景速查

### 场景 A：服务器部署后，怎么确认节点本身没问题？

先看服务器层：

```bash
curl https://api.ipify.org
docker logs --tail 100 <container>
ss -lntp
```

如果服务器自己出口正常、服务正常，再去看 OpenWrt 或终端。

### 场景 B：主 WiFi 分流是否真的生效？

看中间设备层：

```bash
iptables -t nat -vnL PSW
iptables -t mangle -vnL PSW
cat /tmp/etc/passwall/var
```

重点找：

- `br-lan`
- `match-set ... black/gfw`
- `redir port`
- 对应 `ACL_*_tcp_node`

### 场景 C：USA WiFi 是否还在走独立全代理？

看 OpenWrt 本地显式 SOCKS 或对应 `us` ACL：

```bash
curl --connect-timeout 5 --max-time 30 -x socks5h://<OPENWRT_SOCKS_HOST>:<OPENWRT_SOCKS_PORT> https://api.ipify.org
uci show passwall | grep "@acl_rule"
```

显式 SOCKS 结果只代表该端口链路；还要从实际 USA WiFi 客户端产生请求并观察对应透明代理 ACL 计数，才能判定 WiFi 分流策略。

### 场景 D：页面能开，但模型接口不行

不要只测首页。分开测：

```bash
curl --connect-timeout 5 --max-time 30 https://chatgpt.com/cdn-cgi/trace
curl --connect-timeout 5 --max-time 30 -sS -o /dev/null -w 'HTTP %{http_code}\n' https://api.openai.com/v1/models
curl --connect-timeout 5 --max-time 30 -sS -o /dev/null -w 'HTTP %{http_code}\n' https://api.anthropic.com/v1/messages
curl --connect-timeout 5 --max-time 30 -sS -o /dev/null -w 'HTTP %{http_code}\n' 'https://generativelanguage.googleapis.com/$discovery/rest?version=v1beta'
```

上述无凭据 GET 仅用于网络可达性，trace 响应可读出口字段；401/403/404/405 应结合 TLS、响应头和平台行为解释，不能当作业务验收通过。OpenRouter、Cursor、Copilot 等需要另测实际产品 API/插件链路，首页 HEAD 不能替代模型请求。

### 场景 E：节点切了，但出口没变

优先检查：

1. 客户端是否还在原网段
2. ACL 是否真实命中
3. `/tmp/etc/passwall/var` 是否绑定了新节点
4. 默认 SOCKS 与透明代理是不是两条不同链

---

## 5. 为什么模型接口要单独治理

AI 产品往往不只一个域名，而是拆成：

- 主站
- API
- 控制台
- 静态资源
- IDE 插件代理
- 账户系统

如果只代理首页域名，会出现：

- 首页能开，API 超时
- Cursor / Copilot 登录正常，但补全失败
- OpenRouter 控制台正常，但请求不稳

因此建议维护“高价值、低副作用”的模型接口域名集合，而不是把 `github.com`、`x.com` 等整站全量代理。

---

## 6. 什么时候该怀疑哪一层

### 优先怀疑客户端

如果：

- 当前 SSID/网段不符合预期
- `ipify` 与目标设计不符
- 本地 DNS 直接超时

### 优先怀疑中间设备

如果：

- `uci` 已改，但 `iptables` 没命中
- `proxy_host` 已补，但黑名单/GFW 命中不增长
- SOCKS 成功，但透明链不通

### 优先怀疑服务器

如果：

- 服务器自己 `ipify` 异常
- Xray / 容器未启动
- 端口没监听
- 日志出现 `failed to build`、`invalid`、`listen`、`tls` 错误

---

## 7. 常见误判

- `uci` 改了 ≠ 出口变了
- SOCKS 成功 ≠ 透明代理成功
- 页面能开 ≠ 模型接口可用
- 服务器好 ≠ OpenWrt 路由就正确
- 静态配置对 ≠ 运行态路径对
- SSID 名称含 USA ≠ 该网络一定全代理
- traceroute / ICMP 通 ≠ HTTPS 出口正确
- 单次 `ipify` 结果 ≠ 出口长期固定或国家属性

---

## 8. 最终建议

对这类任务，统一采用下面策略：

1. 先分清任务域
2. 再跑三视角验证
3. 最后才改配置
4. 每次改动必须带回滚点

修改 ACL 或节点前，在受控位置备份 UCI；变更后同时复测客户端和路由器。故障时恢复对应配置，并重复同一组探测，避免用不同的测试条件判断是否恢复。

延伸阅读：

- [AI 平台智能分流与住宅出口运行手册](ai-routing-and-residential-egress.md)
- [Gateway 超时修复手册](gateway-egress-timeout-remediation-runbook.md)
- [部署指南](deployment.md)
