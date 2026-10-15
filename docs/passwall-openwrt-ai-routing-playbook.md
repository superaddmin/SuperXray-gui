# OpenWrt / Passwall AI 路由实战手册

> 相关总览入口：[服务器部署 + OpenWrt 路由 + AI 出口治理统一总览](operations-ai-routing-overview.md)

> 适用场景：主 WiFi 默认直连、AI / GFW 域名代理、USA WiFi 全代理的双网络结构。
> 核对日期：2026-10-15。OpenWrt、Passwall 和 SSID 配置属于外部运行环境，本仓库不安装这些组件、不写入 UCI/ACL。当前 SuperXray 提供 Xray 节点、订阅及模板配置；下文命令是现场验证步骤。

## 1. 推荐拓扑

### 主 WiFi

- 网段：`192.168.5.0/24`
- 目标：普通流量直连，AI / GFW 域名走稳定代理节点

### USA WiFi

- 网段：`192.168.52.0/24`
- 目标：保持单独全代理，用于美国出口专网

### 历史排查案例

原文保留过一次名为 USA 的 SSID 实际按域名分流的现场记录，但没有可复核的采样时间和脱敏证据文件，因此不能视为当前部署状态。可复用的排查方法是：

- 记录实际 SSID、客户端网段和网关链，不能套用上表的规划地址。
- 比较普通端点与 AI 目标的出口；`chatgpt.com/cdn-cgi/trace` 使用 GET 才能读取 `ip` 与 `colo`，单次 IP 不能证明长期固定或国家属性。
- ICMP 路径与 TCP/UDP 代理路径分别判断，traceroute 不代表 HTTPS 出口。
- 客户端未启用系统代理或 TUN 只能排除部分本机路径；需结合 OpenWrt ACL 计数与进程确认是否由网关接管，不能只凭客户端状态断定。

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

### OpenWrt

```bash
uci show passwall
iptables -t nat -vnL PSW
iptables -t mangle -vnL PSW
cat /tmp/etc/passwall/var
iwinfo
cat /tmp/dhcp.leases
```

`PSW` 链及 `/tmp/etc/passwall/var` 依赖 Passwall 版本；fw4/nftables 环境使用 `nft list ruleset` 核对实际链和计数。采样前后从目标 WiFi 产生请求，观察计数差值；`uci show` 和进程参数可能含节点凭据，分享前脱敏。

### 服务器

```bash
docker logs --tail 100 <container>
curl https://api.ipify.org
ss -lntp
```

systemd 部署改用 `systemctl is-active x-ui`、`journalctl -u x-ui -n 100 --no-pager`。服务器直连 ipify 只验证基础网络；还需实际客户端经目标节点连通。订阅用 GET 拉取当前 `subPath`/`subJsonPath`/`subClashPath` 并核对 `/diagnose`，不要以 HEAD 的 404 判故障。修改 ACL 或节点前在受控位置备份 UCI，变更后同时复测客户端和路由器；故障时恢复对应配置并重复相同探测。

## 6. 常见陷阱

- `uci` 改了不等于出口变了
- SOCKS 成功不等于透明代理成功
- 页面能开不等于模型接口可用
- 不要在未确认当前网段/SSID前判断策略是否正确
