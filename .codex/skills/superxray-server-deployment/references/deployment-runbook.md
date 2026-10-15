# 部署交付 Runbook（脱敏模板）

本文只描述交付格式和验证顺序。真实服务器值必须写入 `.local/ops/<deployment-id>/` 或密码管理器；不要把真实值复制回本文。

## 脱敏交付 manifest

建议每台服务器生成一个本地 JSON，结构如下：

```json
{
  "server": {
    "host": "<SERVER_IP>",
    "arch": "<amd64|arm64>",
    "panelVersion": "<PANEL_VERSION>",
    "xrayVersion": "<XRAY_VERSION>"
  },
  "service": {
    "systemd": "<active|inactive|failed>",
    "xray": "<running|stopped|unknown>",
    "tlsVerified": "<true|false>",
    "ports": {
      "web": "<WEB_PORT>",
      "subscription": "<SUB_PORT>",
      "xray": "<XRAY_PORT>",
      "socks5": "<SOCKS_PORT>"
    }
  },
  "subscriptions": {
    "threeMonth": {
      "expiryTimeMs": "<UNIX_MILLISECONDS>",
      "expiresUtc": "<ISO8601_UTC>",
      "clashUrl": "<REDACTED_SUBSCRIPTION_URL>",
      "diagnostic": {"outputNodes": 1, "skippedNodes": 0}
    },
    "sixMonth": {
      "expiryTimeMs": "<UNIX_MILLISECONDS>",
      "expiresUtc": "<ISO8601_UTC>",
      "clashUrl": "<REDACTED_SUBSCRIPTION_URL>",
      "diagnostic": {"outputNodes": 1, "skippedNodes": 0}
    }
  },
  "socks5": {
    "host": "<SOCKS_HOST>",
    "port": "<SOCKS_PORT>",
    "username": "<SOCKS_USERNAME>",
    "password": "<SOCKS_PASSWORD>",
    "uri": "socks5://<SOCKS_USERNAME>:<SOCKS_PASSWORD>@<SOCKS_HOST>:<SOCKS_PORT>"
  }
}
```

`<REDACTED_SUBSCRIPTION_URL>`、`<SOCKS_PASSWORD>` 等占位符是故意保留的。若要在受保护的本地目录保存实际 manifest，可保留同样字段，但不能把该文件复制到 Git 工作树或交付报告。

默认校验器要求 manifest 脱敏；只有在受保护的本地运维目录中核对实际值时，才显式使用 `--allow-secrets`，且命令输出仍不会打印密码：

```powershell
python .codex/skills/superxray-server-deployment/scripts/validate_deployment_outputs.py --manifest <private-manifest.json> --allow-secrets
```

## 三个月/六个月检查表

1. 每个期限有独立 `subId` 或独立入站，不能把一个永久 SOCKS5 账号伪装成两个期限链接。
2. `expiryTimeMs` 是整数毫秒，不是秒，也不是字符串日期。
3. 到期时间按日历月计算：起始时间分别执行 `AddDate(0, 3, 0)` 和 `AddDate(0, 6, 0)`；同时记录 UTC 字符串。
4. 通过 GET 请求 `/clash/<subId>`，解析 YAML 节点；再 GET `/clash/<subId>/diagnose`，确认 `outputNodes` 和 `skippedNodes`。
5. Clash 与 Mihomo 使用同一个 Clash YAML URL；不要为两个客户端伪造两种不同的订阅协议。

## 通用 SOCKS5 检查表

Mixed 入站最小设置应满足：

```json
{
  "protocol": "mixed",
  "listen": "0.0.0.0",
  "port": "<SOCKS_PORT>",
  "settings": {
    "auth": "password",
    "accounts": [
      {
        "user": "<SOCKS_USERNAME>",
        "pass": "<SOCKS_PASSWORD>",
        "subId": "<SOCKS_SUB_ID>",
        "enable": true
      }
    ],
    "udp": true
  }
}
```

交付时同时给出表单字段和 URI。URI 中的凭据必须 URL 编码；客户端表单中的密码填写原始密码。用交互式账号输入配合 `curl --socks5-hostname` 验证远程 DNS 和实际出口，不能把真实密码写进命令历史或脚本：

```bash
read -r -p 'SOCKS username: ' SOCKS_USERNAME
read -r -s -p 'SOCKS password: ' SOCKS_PASSWORD; printf '\n'
curl --socks5-hostname '<SOCKS_HOST>:<SOCKS_PORT>' \
  --proxy-user "${SOCKS_USERNAME}:${SOCKS_PASSWORD}" https://api.ipify.org
unset SOCKS_USERNAME SOCKS_PASSWORD
```

## 服务器巡检证据

保留命令和脱敏结果：

```bash
systemctl is-active x-ui.service
ps -eo pid,comm,args | grep -E 'x-ui|xray' | grep -v grep
ss -lntp | grep -E ':(<WEB_PORT>|<SUB_PORT>|<SOCKS_PORT>|<XRAY_PORT>)'
# settings.udp=true 或协议需要 UDP 时追加
ss -lunp | grep -E ':(<SOCKS_PORT>|<XRAY_PORT>)'
journalctl -u x-ui.service --since '5 min ago' --no-pager
```

Xray 的“启动成功”日志可能以 warning 级别出现；重启竞态也可能短暂记录 `xray is not running`。以随后进程、监听端口和代理链路为准，并记录首个持续性错误。

## 反向代理要求

订阅服务的公开地址不能依赖请求来源 IP 推断。`/sub/`、`/json/`、`/clash/` 三个 location 都应设置：

```nginx
proxy_set_header Host $host;
proxy_set_header X-Forwarded-Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```

部署后优先把 `subURI`、`subJsonURI`、`subClashURI` 固定为公开域名/端口，再执行 GET 验证。只看 200 或只看面板页面不够。

## 回滚记录

记录以下内容但脱敏：

- 上一版本 commit、面板版本和 Xray 版本；
- 数据库/证书/二进制备份位置；
- 首个错误和最小复现命令；
- 回滚命令及回滚后的 `systemctl`、`ss`、订阅诊断和代理出口结果。
