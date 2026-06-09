# OpenWrt / Passwall AI 路由实战手册

> 相关总览入口：[服务器部署 + OpenWrt 路由 + AI 出口治理统一总览](operations-ai-routing-overview.md)

> 适用场景：主 WiFi 默认直连、AI / GFW 域名代理、USA WiFi 全代理的双网络结构。

## 1. 推荐拓扑

### 主 WiFi

- 网段：`192.168.5.0/24`
- 目标：普通流量直连，AI / GFW 域名走稳定代理节点

### USA WiFi

- 网段：`192.168.52.0/24`
- 目标：保持单独全代理，用于美国出口专网

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
curl https://api.ipify.org
curl -I https://api.openai.com/v1/models
curl -I https://chatgpt.com/cdn-cgi/trace
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
