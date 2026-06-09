# 网络 / 路由 / 代理问题排障清单

## 适用范围

- OpenWrt / Passwall / 透明代理
- AI / GFW 域名分流
- 服务器节点出口与订阅节点验证
- 线上“能访问但功能异常”的模型接口问题

## Step 1: 先锁定当前任务域

在动手前明确当前主问题属于哪一类：

- 发布 / 构建
- 服务器部署
- 服务器运行态
- OpenWrt/Passwall 分流
- 客户端体验
- 文档/治理

跨域时先写明暂挂项，不要把多个问题混成一个结论。

## Step 2: 先看客户端真实出口

```powershell
curl.exe https://api.ipify.org
ipconfig
```

如果连当前网段 / 网关 / 出口都没确认，不要开始评估节点质量。

## Step 3: 再看中间设备命中

```bash
uci show passwall
iptables -t nat -vnL PSW
iptables -t mangle -vnL PSW
cat /tmp/etc/passwall/var
```

未看到计数增长前，不要声称规则已生效。

## Step 4: 最后看服务器

```bash
curl https://api.ipify.org
docker logs --tail 100 <container>
ss -lntp
```

服务器健康不代表客户端路径就正确。

## Step 5: 网络类结论至少要双视角

必须满足以下任一组合：

- 客户端 + 中间设备
- 中间设备 + 服务器

## 禁忌

- 不要用单次 `curl -x socks5h://...` 结论覆盖透明代理链。
- 不要用“页面可打开”替代“模型接口可用”。
- 不要在未确认默认出口前评价节点优劣。
- 不要先改一堆配置再回头找首个阻塞点。
