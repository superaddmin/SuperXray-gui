---
name: superxray-server-deployment
description: "Deploy and verify SuperXray on a Linux server, then produce the standard delivery text for long-term, three-month and six-month Clash/Mihomo subscriptions plus one long-term authenticated SOCKS5 connection. Use for server rollout, post-deployment checks, subscription delivery, or deployment incident retrospectives."
---

# SuperXray 服务器部署与交付

## 目的

把“代码已推送”与“服务器已可用”分开验证。每次部署都要留下可复现的运行态证据，并输出一份脱敏交付摘要，至少包含：

- 长期、三个月和六个月的 Clash/Mihomo 订阅入口、到期时间和诊断计数；
- 一个长期 SOCKS5 的主机、端口、用户名、密码和 URI 格式；
- 面板、Xray、订阅服务、监听端口、TLS 握手和实际代理出口状态；
- 首个错误、回滚点和未验证项。

真实服务器 IP、订阅 ID、订阅 URL、用户名、密码、UUID、面板路径、Cookie、私钥和原始日志只能放在受保护的本地运维目录或密码管理器中，不得写入仓库、`.codex`、Issue、截图或公开报告。

## 何时使用

- 推送 `main` 后把新版本部署到一台或多台 Linux 服务器；
- 需要一次性生成三个月/六个月订阅和通用 SOCKS5 参数；
- 线上部署后检查服务、端口、证书、订阅内容、Xray 链路和日志；
- 复盘部署失败、订阅地址漂移、证书异常、端口冲突或 SSH/脚本问题。

## 执行顺序

1. **划定范围和回滚点**：先记录当前 commit、版本、目标架构、部署方式、服务器快照/数据库备份位置和未提交的无关改动。不要为了“清理工作区”回滚用户已有改动。
2. **核对发布产物**：读取 `config/version`、`CHANGELOG.md` 和 Release 资产；目标服务器先执行 `uname -m`，将 `x86_64` 映射到 `amd64`，将 `aarch64/arm64` 映射到 `arm64`。不要把错误架构的面板或 Xray 二进制上传到服务器。
3. **安全连接服务器**：优先使用 SSH key；密码只能由交互式提示或受保护的凭据注入提供，不能写入命令行、脚本、日志或 Git。固定并核对主机指纹；Windows OpenSSH 遇到 `known_hosts` 权限错误时使用临时受保护的 known-hosts 文件，不能为了省事关闭主机校验。
4. **安装/更新**：备份 `/etc/x-ui/x-ui.db`、证书和当前二进制；复制 Release 包或源码构建产物；写入 `/etc/default/x-ui`；运行 `x-ui migrate` 和安全的用户名、密码、随机 `webBasePath` 配置；改动 systemd unit 后执行 `systemctl daemon-reload`，再 `enable --now x-ui`。
5. **创建主节点与期限订阅**：在同一个 VLESS（TLS Vision 或 Reality）入站内创建三个 client：长期（`expiryTime=0`）、三个月、六个月，各自拥有独立 `subId`。`expiryTime` 是 Unix 毫秒；有效期按日历月 `AddDate(0, 3, 0)` / `AddDate(0, 6, 0)` 计算，同时记录 UTC 可读时间。不要把三个月/六个月放到 mixed/SOCKS5 入站：当前代码不会读取 `mixed` 入站 `settings.accounts[].expiryTime`。
   生成时记录同一个 `issuedAt`，分别计算 `issuedAt.AddDate(0, 3, 0)` 和 `issuedAt.AddDate(0, 6, 0)`，为每个期限分配独立客户端/subId，保存 Unix 毫秒后再用 GET 和 diagnose 回读核对。
6. **生成长期 SOCKS5**：使用 `protocol: "mixed"` 入站和 `settings.accounts[]` 账号，只创建一个长期账号，确认端口唯一、`listen` 与防火墙策略一致。标准交付只输出连接参数，不输出 SOCKS5 订阅 URL；输出配置字段和 URI：

   ```text
   协议: SOCKS5
   主机: <SOCKS_HOST>
   端口: <SOCKS_PORT>
   用户名: <SOCKS_USERNAME>
   密码: <SOCKS_PASSWORD>
   URI: socks5://<URLENCODED_USERNAME>:<URLENCODED_PASSWORD>@<SOCKS_HOST>:<SOCKS_PORT>
   ```

   通用代理客户端使用上述五项；它不需要 Clash/Mihomo 订阅 URL。密码含 `@`、`:`, `/` 或 `%` 时只对 URI 中的用户名/密码做 URL 编码，表单字段仍填写原始值。支持远程 DNS 的客户端可使用 `socks5h` 语义。
7. **验证服务端运行态**：在服务器执行 `systemctl is-active x-ui.service`、`ss -lntp`、`journalctl -u x-ui.service --since "5 min ago"`，分别核对面板、订阅和 Xray TCP 监听；Mixed 设置 `udp: true` 时追加 `ss -lunp` 和云安全组/本机防火墙 UDP 检查。不要把 `x-ui` active 当成 Xray 入站可用的唯一证据。
8. **验证公网订阅**：订阅入口只保证 GET 和 `/<subId>/diagnose`，没有 HEAD 路由。用实际 GET 请求解析 Clash/Mihomo 内容中的 `type`、`server`、`port`、`username`、`password`，再检查诊断 `outputNodes >= 1`、`skippedNodes == 0`。订阅服务独立于 `webBasePath`；反向代理的 `/sub/`、`/json/`、`/clash/` 都必须传递 `X-Forwarded-Host`，并优先固定 `subURI/subJsonURI/subClashURI`，避免生成的节点主机漂移到客户端 IP。
9. **验证实际代理链路**：通用 SOCKS5 至少从客户端运行一次 `curl --socks5-hostname` 到 `https://api.ipify.org`，出口应与预期服务器一致；再选择一个稳定 HTTPS 目标检查 HTTP 状态。Google 单一探测超时只能记录为该目标/线路结果，不能据此宣称 SOCKS5 服务失效。
10. **形成标准交付信息**：先调用 `scripts/render_subscription_delivery.py --manifest <private-delivery.json>` 生成“标准订阅交付格式”文本，再用 `scripts/validate_deployment_outputs.py` 检查脱敏 manifest；公开报告只保留占位符、状态、计数、到期时间和验证命令，不复制完整订阅响应、账号密码或原始日志。

## 标准订阅交付格式

所有新部署必须按以下顺序和字段输出。示例只使用占位符，真实值只写入 `.local/ops/<deployment-id>/`：

```text
面板安装完成！   <COUNTRY> <SERVER_IP>

https://<PANEL_HOST>:<WEB_PORT><WEB_BASE_PATH>/
用户名: <PANEL_USERNAME>
密码: <PANEL_PASSWORD>

URI/Base64:
https://<PUBLIC_HOST>:<SUB_PORT>/sub/<MAIN_SUB_ID>

Xray JSON:
https://<PUBLIC_HOST>:<SUB_PORT>/json/<MAIN_SUB_ID>

Clash/Mihomo:
https://<PUBLIC_HOST>:<SUB_PORT>/clash/<MAIN_SUB_ID>

SOCKS5 订阅
协议: SOCKS5
主机: <SOCKS_HOST>
端口: <SOCKS_PORT>
用户名: <SOCKS_USERNAME>
密码: <SOCKS_PASSWORD>
SOCKS5 订阅链接: https://<PUBLIC_HOST>:<SUB_PORT>/clash/<SOCKS_SUB_ID>
三个月有效期（UTC 到期: <YYYY-MM-DD HH:MM:SS>） https://<PUBLIC_HOST>:<SUB_PORT>/clash/<THREE_MONTH_SUB_ID>
六个月有效期（UTC 到期: <YYYY-MM-DD HH:MM:SS>） https://<PUBLIC_HOST>:<SUB_PORT>/clash/<SIX_MONTH_SUB_ID>
```

- URI/JSON/Clash 三段固定使用长期主节点的 `subId`；三个月/六个月行使用各自期限 client 的 `/clash/<subId>`；
- 有可用证书时所有链接使用 `https://`，并先开启 `subJsonEnable`，确保 `/json/` 返回 200；
- SOCKS5 段先给协议/主机/端口/用户名/密码，再给该长期账号自己的 Clash 订阅链接（`/clash/<SOCKS_SUB_ID>`），不附带限时 SOCKS5 链接；
- UTC 到期时间由 `expiryTimeMs / 1000` 转换，格式固定为 `YYYY-MM-DD HH:MM:SS`；
- 交付前逐项验证 6 个 URL（面板、/sub/、/json/、长期 /clash/、SOCKS5 /clash/、3/6 月 /clash/）均为 HTTPS 200，并至少实测一次 VLESS 与 SOCKS5 代理出口。

## 期限订阅契约

当前订阅实现的关键事实：

- 普通客户端/入站的 `expiryTime` 由服务端过滤，单位为 Unix milliseconds；
- Mixed/HTTP 的 `settings.accounts[]` 目前按 `subId` 匹配账号，但不读取账号级 `expiryTime`；
- 标准部署把长期、三个月、六个月放在同一个 VLESS 入站的三个独立 client 中；SOCKS5 只有一个长期账号，不生成限时 SOCKS5 订阅，也不依赖 mixed 账号级到期；
- 到期时间应同时输出 `expiryTimeMs` 与 ISO 8601 UTC 字符串，并在验证时确认仍晚于当前时间；
- Clash 与 Mihomo 使用同一 `/clash/<subId>` YAML 入口，URI、JSON、Clash 端点均用 GET 验证。

## 常见踩坑与修正

| 现象 | 首个检查 | 修正 |
| --- | --- | --- |
| 面板 active 但节点不可用 | Xray 进程、入站端口、Xray 日志 | 分开检查 `systemctl`、`ps`、`ss` 和 Xray 重启结果 |
| `systemctl` 提示 unit changed on disk | `systemctl status` 首行 | `systemctl daemon-reload` 后再重启 |
| 重启日志出现 `xray is not running` | 随后进程与监听端口 | 若随后 Xray 进程和端口正常，把它记录为重启竞态；若持续出现再排查配置/权限 |
| 订阅返回 200 但客户端无节点 | GET 正文与 diagnose | 解析协议字段和 `outputNodes/skippedNodes`，不要只看状态码 |
| 订阅节点主机变成客户端 IP | 反代请求头与 `ResolveRequest` 回退链 | 三个订阅 location 均设置 `X-Forwarded-Host`，并固定公开订阅 URI |
| 用 HEAD 检查订阅失败 | 路由定义 | 改用 GET；订阅服务没有 HEAD 路由 |
| 三/六个月链接到期不一致 | `expiryTime` 单位和计算方式 | 写入 Unix 毫秒，按日历月计算，分别核对 UTC 时间 |
| Mixed 多账号没有分别到期 | `accounts[].expiryTime` 是否被读取 | 目前拆成独立入站/subId，或先扩展代码并补测试 |
| 本地 Xray 测试启动失败 | 本地端口占用 | 启动前探测空闲端口；不要固定复用已被占用的端口 |
| 证书菜单显示成功但 HTTPS 失败 | 实际 TLS 握手 | 用 `curl -fsS` 或 `openssl s_client` 验证证书链、SAN 和域名/IP 一致性 |
| 服务器能开面板但公网订阅失败 | 安全组、防火墙、监听地址 | 同时检查云安全组、本机防火墙、`ss` 和公网 GET |
| PowerShell 远程命令引号/`printf` 异常 | 本地 shell 与远端 shell | 显式使用 `ssh ... "bash -s"` 或 heredoc；避免把 PowerShell 语法当 Bash 执行 |

## 最小验证命令

服务器端：

```bash
uname -m
systemctl is-active x-ui.service
ss -lntp | grep -E ':(<WEB_PORT>|<SUB_PORT>|<SOCKS_PORT>|<XRAY_PORT>)'
# 仅当 SOCKS/Mixed 或其他入站启用 UDP 时执行
ss -lunp | grep -E ':(<SOCKS_PORT>|<XRAY_PORT>)'
journalctl -u x-ui.service --since '5 min ago' --no-pager
```

客户端/公网端：

```bash
curl -fsS "https://<PUBLIC_HOST>:<SUB_PORT>/clash/<SUB_ID>"
curl -fsS "https://<PUBLIC_HOST>:<SUB_PORT>/clash/<SUB_ID>/diagnose"
read -r -p 'SOCKS username: ' SOCKS_USERNAME
read -r -s -p 'SOCKS password: ' SOCKS_PASSWORD; printf '\\n'
curl --socks5-hostname '<SOCKS_HOST>:<SOCKS_PORT>' --proxy-user "${SOCKS_USERNAME}:${SOCKS_PASSWORD}" https://api.ipify.org
unset SOCKS_USERNAME SOCKS_PASSWORD
```

上面的账号输入使用交互式读取；不要把真实密码替换进命令文本、Shell 历史或脚本。

提交前：

```powershell
python .codex/skills/superxray-server-deployment/scripts/validate_deployment_outputs.py --manifest <redacted-manifest.json>
python scripts/secret_scan.py
python .codex/skills/superxray-project-context/scripts/validate_codex_config.py
```

## 回滚

部署失败时先停止新进程，恢复上一版 `/usr/local/x-ui`、systemd unit、`/etc/x-ui/x-ui.db` 和证书备份，执行 `systemctl daemon-reload && systemctl restart x-ui`，然后重复服务端、订阅、公网代理三组验证。不要通过删除数据库、重置账号或删除远程 tag 来掩盖部署失败。

## 参考

- `references/deployment-runbook.md`：脱敏交付 manifest、期限链接和 SOCKS5 输出模板。
- `docs/deployment.md`：安装、备份、证书、防火墙、Docker 与日常巡检。
- `docs/server-inbound-subscription-audit.template.md`：服务器订阅审计字段和脱敏边界。
- `sub/proxy_account_subscription.go`、`sub/subClashService.go`、`sub/subService.go`：订阅与 Mixed/SOCKS5 输出实现。
- `.codex/workflows/network-routing-debug-checklist.md`：客户端/中间设备/服务器双视角验证。
- `.codex/skills/superxray-release-cicd/SKILL.md`：Release 资产和回滚门禁。
