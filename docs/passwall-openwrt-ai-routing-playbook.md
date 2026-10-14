# OpenWrt / Passwall AI 路由实战手册

> 相关总览入口：[服务器部署 + OpenWrt 路由 + AI 出口治理统一总览](operations-ai-routing-overview.md)

> 适用场景：主 WiFi 默认直连、AI / GFW 域名代理、USA WiFi 全代理的双网络结构。
> 代码核对日期：2026-10-14。SuperXray-gui 不配置 OpenWrt、Passwall 或 WiFi；下述网络结构是运维规划，实际策略依赖目标固件、Passwall 版本和运行规则。现场案例保留原始记录，本轮未连接设备重新实测。

## 1. 推荐拓扑

### 主 WiFi

- 网段：`192.168.5.0/24`
- 目标：普通流量直连，AI / GFW 域名走稳定代理节点

### USA WiFi

- 网段：`192.168.52.0/24`
- 目标：保持单独全代理，用于美国出口专网

### 实测现状

> 历史现场案例：以下观察只适用于当时环境。缺少客户端代理进程不能单独证明代理一定由 OpenWrt 承担，需结合网关进程、ACL/规则计数交叉验证；trace 返回的出口 IP/colo 也不能单独证明出口长期固定或属于住宅网络。

上表是规划目标，现场可能不同，排障前先按实测值核对（本节数据来自一次实际排查）：

- SSID：`USA_Los Angeles`，客户端地址 `192.168.111.0/24`
- 网关链：`192.168.111.1` → `192.168.1.1` → 运营商 CGNAT `100.64.64.1`；ICMP 直出、不经代理
- 分流行为：**按域名分流，不是全代理** —— 国内端点回显本地 ISP 出口，AI / 境外域名走隧道，隧道出口为固定公网 IP（`chatgpt.com/cdn-cgi/trace` 可直接确认 `ip` 与 `colo`）
- 客户端侧没有任何代理进程、没有 TUN/TAP 适配器、系统代理为关闭，说明代理在 OpenWrt 网关上生效，客户端排查只能验证"出口是否变了"，不能验证 ACL

排查时不要假定 SSID 名称与分流策略一致（名为 USA 的网络实测仍是分流），一律以客户端实测出口和 OpenWrt 运行态为准。

## 2. 关键概念

### 显式 SOCKS

客户端显式填写代理地址，流量只在应用层进入代理。

### 透明 REDIRECT

iptables NAT 将命中的 TCP 流量重定向到本地代理端口。

### TPROXY

常用于 UDP/更完整的透明代理路径，命中后仍保留原始目标信息。

### DNS 劫持 / 分流

DNS 不只是“解析”，还决定哪些域名会被视为代理目标。DNS 与 ACL 规则必须一起看。

## 3. 推荐排障顺序

1. 客户端确认当前 SSID / 网段 / 默认出口。
2. OpenWrt 确认当前 ACL、`iptables` 命中、Passwall 运行缓存。
3. 服务器确认节点本身是否可用。

## 4. 为什么模型接口域名要单独补齐

AI 产品通常不止一个首页域名，还会拆分：

- API
- 控制台
- 静态资源
- IDE 插件代理
- 账户系统

如果只代理首页域名，常见后果是：

- 页面能打开，API 超时
- Cursor / Copilot 登录正常但补全失败
- OpenRouter 控制台正常但请求不稳定

因此应补齐高价值模型接口域名，而不是把 `github.com`、`x.com` 这类整站一刀切走代理。

## 5. 验证模板

### 客户端

```bash
curl --connect-timeout 5 --max-time 30 https://api.ipify.org
curl --connect-timeout 5 --max-time 30 -sS -o /dev/null -w 'HTTP %{http_code}\n' https://api.openai.com/v1/models
curl --connect-timeout 5 --max-time 30 https://chatgpt.com/cdn-cgi/trace
```

`401`/`403` 等响应须区分网络可达、授权失败和平台策略拒绝；基础探测不验证模型推理。trace 需要 GET 正文才能读取 `ip` 与 `colo`。

### OpenWrt

```bash
uci show passwall
iptables -t nat -vnL PSW
iptables -t mangle -vnL PSW
cat /tmp/etc/passwall/var
iwinfo
cat /tmp/dhcp.leases
```

这些命令适用于相应 Passwall/iptables 布局。fw4/nftables 环境还需 `nft list ruleset`；没有 `PSW` 链或上述缓存文件时先定位实际版本和规则，不能据此直接判定故障。完整 UCI、运行参数、租约和日志可能含代理密码或设备信息，分享前只保留脱敏摘要。

### 服务器

```bash
docker logs --tail 100 <container>
curl https://api.ipify.org
ss -lntp
```

## 6. 常见陷阱

- `uci` 改了不等于出口变了
- SOCKS 成功不等于透明代理成功
- 页面能开不等于模型接口可用
- 不要在未确认当前网段/SSID前判断策略是否正确
