# 运行态网络与代理排障地图

## 三视角证据模型

### 1. 客户端视角

适用问题：

- 默认出口是谁
- AI/API 是否可访问
- DNS 是否超时
- 普通网页与模型接口是否分化

最小命令：

```powershell
ipconfig
curl.exe https://api.ipify.org
curl.exe -I https://api.openai.com/v1/models
curl.exe -I https://chatgpt.com/cdn-cgi/trace
nslookup api.openai.com <gateway-ip>
```

### 2. 中间设备（OpenWrt / Passwall）视角

适用问题：

- ACL 是否创建
- 规则是否命中
- 透明代理与默认链是否串扰
- DNS 劫持与代理域名表是否一致

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

### 3. 服务器视角

适用问题：

- 节点本身是否可用
- 服务器真实出口是谁
- 程序是否运行、监听是否正确

最小命令：

```bash
docker exec <container> /app/x-ui -v
docker logs <container> --tail 100
curl https://api.ipify.org
ss -lntp
```

## 排障顺序

1. 先确认客户端当前网段 / SSID / 默认网关。
2. 再确认中间设备 ACL 与命中计数。
3. 再确认服务器节点本身是否健康。
4. 最后才修改静态配置。

## 常见错误推理

- “节点切了，所以出口就变了” -> 错。先看 `ipify` 与 ACL 计数。
- “SOCKS 出口没问题，所以透明代理也没问题” -> 错。它们不是同一条路径。
- “Google/首页能开，所以模型接口也没问题” -> 错。AI 需要单独域名集合验证。
