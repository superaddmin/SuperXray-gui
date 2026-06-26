# 服务器部署 + OpenWrt 路由 + AI 出口治理统一总览

> **目标读者**：需要同时维护 SuperXray 服务器节点、OpenWrt/Passwall 分流、AI 平台出口质量与排障流程的开发者 / 运维人员  
> **适用版本**：`v3.4.1`
> **相关文档**：[部署指南](deployment.md) | [AI 平台智能分流与住宅出口运行手册](ai-routing-and-residential-egress.md) | [OpenWrt / Passwall AI 路由实战手册](passwall-openwrt-ai-routing-playbook.md)

---

## 1. 先理解系统分层

这类问题通常不是“单台服务器”问题，而是三层协同：

### 1.1 服务器层

职责：

- 运行 SuperXray 面板
- 维护 Xray/订阅服务
- 提供一个或多个可用出口节点

常见对象：

- `85.155.178.115`
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

### 主 WiFi

- 网段：通常是 `192.168.5.0/24`
- 目标：
  - 普通流量直连
  - AI / GFW 域名走稳定代理节点（例如 `85.155.178.115`）

### USA WiFi

- 网段：通常是 `192.168.52.0/24`
- 目标：
  - 保持全代理
  - 独立承载美国出口需求（例如 `35.87.239.230`）

### 为什么要分开

这样做的收益是：

- 普通网页/下载不被代理拖慢
- AI / 模型接口路径稳定
- 日常网络与美国代理网络互不污染
- 后续排障时可以快速判断问题属于哪一层

---

## 3. 标准验证顺序

## 3.1 客户端视角

先回答：

- 当前连的是哪个 SSID？
- 当前 IP 属于哪个网段？
- 默认出口是谁？
- 普通网页与 AI 接口是否一致？

最小命令：

```bash
curl https://api.ipify.org
curl -I https://api.openai.com/v1/models
curl -I https://chatgpt.com/cdn-cgi/trace
nslookup api.openai.com <gateway-ip>
```

## 3.2 中间设备视角

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

## 3.3 服务器视角

最后确认：

- 节点本身是不是好的？
- 服务器真实出口是谁？
- 面板/Xray 是否正常运行？

最小命令：

```bash
docker exec <container> /app/x-ui -v
docker logs <container> --tail 100
curl https://api.ipify.org
ss -lntp
```

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
curl -x socks5h://127.0.0.1:1081 https://api.ipify.org
uci show passwall | grep "@acl_rule"
```

### 场景 D：页面能开，但模型接口不行

不要只测首页。分开测：

```bash
curl -I https://chatgpt.com/cdn-cgi/trace
curl -I https://api.openai.com/v1/models
curl -I https://claude.ai
curl -I https://openrouter.ai
curl -I https://cursor.com
curl -I https://models.github.ai
```

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

---

## 8. 最终建议

对这类任务，统一采用下面策略：

1. 先分清任务域
2. 再跑三视角验证
3. 最后才改配置
4. 每次改动必须带回滚点

如果任务涉及 OpenWrt / Passwall / AI 路由，优先阅读：

- [OpenWrt / Passwall AI 路由实战手册](passwall-openwrt-ai-routing-playbook.md)
- [AI 平台智能分流与住宅出口运行手册](ai-routing-and-residential-egress.md)
- [部署指南](deployment.md)
