# OpenWrt AI/GFW Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 OpenWrt 主 WiFi 重构为“AI / GFW 域名走 85.155.178.115，其余直连”，同时保留 USA WiFi 全代理走 35.87.239.230。

**Architecture:** 通过 Passwall 新增/调整 `lan` ACL 实现主 WiFi 规则分流，复用现有 `us` ACL 保持 USA WiFi 独立全代理。优先修改 Passwall 现有自定义代理域名列表入口，避免碰 OpenClash 和全局透明代理默认链。

**Tech Stack:** OpenWrt 24.10.3, Passwall, Xray, UCI, SSH, curl

---

### Task 1: 盘点 Passwall 规则入口与域名列表载体

**Files:**
- Read: `/etc/config/passwall`
- Read: `/tmp/etc/passwall/*`
- Output doc: `docs/superpowers/specs/2026-06-09-openwrt-ai-gfw-routing-design.md`

- [ ] **Step 1: 读取 ACL、全局节点、默认 SOCKS 和自定义域名入口**

Run:

```bash
uci show passwall | grep -E '^passwall\.@acl_rule\[|^passwall\.@global\[0\]|^passwall\.P54yqxIj\.|proxy_list|gfw|domains|domain_list|rule_list'
```

Expected: 能看到 `lan/us` ACL、默认节点、SOCKS 节点和代理域名列表相关键。

- [ ] **Step 2: 读取运行时规则，确认 lan 当前未被全代理接管**

Run:

```bash
iptables -t nat -vnL PSW
iptables -t mangle -vnL PSW
```

Expected: 可区分 `br-us` 与默认链命中，确认主 WiFi 后续要新增或调整的精确位置。

- [ ] **Step 3: 记录关键节点映射**

必须确认：

```text
cfg641c7e -> 85.155.178.115
wyv8VRj6 -> 35.87.239.230
t838Jhr9 -> 35.87.239.230
```

- [ ] **Step 4: 提交一次现场记录（如有本地文档更新）**

```bash
git add docs/superpowers/specs/2026-06-09-openwrt-ai-gfw-routing-design.md docs/superpowers/plans/2026-06-09-openwrt-ai-gfw-routing-implementation.md
git commit -m "docs: add openwrt ai gfw routing design and plan"
```

### Task 2: 备份并构建主 WiFi 的 lan ACL

**Files:**
- Backup: `/etc/config/passwall`
- Modify: `/etc/config/passwall`

- [ ] **Step 1: 生成远端备份**

Run:

```bash
mkdir -p /root/codex-backups
cp -a /etc/config/passwall /root/codex-backups/passwall.before-lan-ai-gfw-$(date +%Y%m%d-%H%M%S)
ls -ltr /root/codex-backups | tail -n 3
```

Expected: 新备份文件出现。

- [ ] **Step 2: 新增或启用一条 lan ACL，默认直连，仅代理列表走 85**

Pseudo-UCI target:

```bash
uci add passwall acl_rule
uci set passwall.@acl_rule[-1].enabled='1'
uci set passwall.@acl_rule[-1].interface='lan'
uci set passwall.@acl_rule[-1].sources='192.168.5.0/24'
uci set passwall.@acl_rule[-1].remarks='WiFi_Main_AI_GFW_via_85'
uci set passwall.@acl_rule[-1].use_global_config='0'
uci set passwall.@acl_rule[-1].tcp_node='cfg641c7e'
uci set passwall.@acl_rule[-1].udp_node='tcp'
uci set passwall.@acl_rule[-1].tcp_proxy_mode='<proxy-list-only mode>'
uci set passwall.@acl_rule[-1].udp_proxy_mode='<proxy-list-only mode or disabled-safe mode>'
uci set passwall.@acl_rule[-1].dns_mode='xray'
uci set passwall.@acl_rule[-1].v2ray_dns_mode='tcp'
uci set passwall.@acl_rule[-1].remote_dns='1.1.1.1'
uci set passwall.@acl_rule[-1].filter_proxy_ipv6='1'
uci set passwall.@acl_rule[-1].chn_list='direct'
uci set passwall.@acl_rule[-1].use_direct_list='1'
uci set passwall.@acl_rule[-1].use_proxy_list='1'
uci set passwall.@acl_rule[-1].use_block_list='1'
uci set passwall.@acl_rule[-1].use_gfw_list='1'
uci set passwall.@acl_rule[-1].dns_shunt='chinadns-ng'
uci commit passwall
```

Expected: lan ACL 存在，节点指向 `cfg641c7e`，不使用 `proxy` 全代理模式。

- [ ] **Step 3: 不改动现有 us ACL**

Run:

```bash
uci show passwall | grep -E '^passwall\.@acl_rule\[[0-9]+\]\.(interface|sources|remarks|tcp_node)='
```

Expected: `us` ACL 仍指向 `t838Jhr9`，未被覆盖。

### Task 3: 补充 AI 域名到代理列表入口

**Files:**
- Modify: `/etc/config/passwall` 或 Passwall 识别的自定义代理域名文件/section

- [ ] **Step 1: 找到 Passwall 实际使用的代理域名自定义入口**

Run:

```bash
uci show passwall | grep -Ei 'proxy.*domain|domain.*proxy|rule.*domain|host.*list'
grep -R -n -Ei 'openai|chatgpt|anthropic|claude|gemini' /etc/config /usr/share/passwall /tmp/etc/passwall 2>/dev/null
```

Expected: 找到可追加自定义代理域名的 section 或文件入口。

- [ ] **Step 2: 追加最小 AI 域名集合**

Domain set:

```text
openai.com
chatgpt.com
oaistatic.com
oaiusercontent.com
anthropic.com
claude.ai
aistudio.google.com
generativelanguage.googleapis.com
makersuite.google.com
gemini.google.com
x.ai
grok.com
perplexity.ai
```

Expected: 这些域名进入代理列表，不覆盖原有列表。

- [ ] **Step 3: 保存并重启 Passwall**

Run:

```bash
uci commit passwall
/etc/init.d/passwall restart
sleep 8
```

Expected: Passwall 正常启动，无明显错误日志。

### Task 4: 在线验证主 WiFi 与 USA WiFi 行为

**Files:**
- Read: `/etc/config/passwall`
- Read: `/tmp/etc/passwall/*`

- [ ] **Step 1: 验证 USA WiFi 保持 35 出口**

Run on OpenWrt:

```bash
curl -sS --max-time 20 -x socks5h://127.0.0.1:1081 https://api.ipify.org
```

Expected: `35.87.239.230`

- [ ] **Step 2: 验证 85 节点本地可用**

Run on OpenWrt using the 85-specific runtime or direct generated socks path if exposed.

Expected: `85.155.178.115`

- [ ] **Step 3: 从主 WiFi 客户端验证默认出口不是 35**

Run:

```powershell
curl.exe -sS --max-time 15 https://api.ipify.org
```

Expected: 不是 `35.87.239.230`；也不要求固定为 85。

- [ ] **Step 4: 验证普通网站直连正常**

Run:

```powershell
curl.exe -I -L -sS --max-time 12 https://www.google.com/generate_204
```

Expected: `HTTP/1.1 204`。

- [ ] **Step 5: 验证 AI 站点可访问**

Run:

```powershell
curl.exe -I -L -sS --max-time 12 https://api.openai.com/v1/models
curl.exe -I -L -sS --max-time 12 https://chatgpt.com/cdn-cgi/trace
```

Expected: OpenAI 返回 `401`；ChatGPT 至少能建立可用响应（可能是 403 challenge，但不应 DNS 超时）。

- [ ] **Step 6: 验证下载速度相对当前透明全代理状态改善**

Run:

```powershell
curl.exe -L -sS --max-time 45 -o NUL -w 'time=%{time_total} speed_Bps=%{speed_download} http=%{http_code} size=%{size_download}\n' 'https://speed.cloudflare.com/__down?bytes=5000000'
```

Expected: 不应退化到全代理异常状态；如明显更差，则回滚。

### Task 5: 回滚预案

**Files:**
- Restore: `/root/codex-backups/passwall.before-lan-ai-gfw-*`

- [ ] **Step 1: 若出现 DNS 超时或主 WiFi 异常，立即恢复备份**

Run:

```bash
cp -a /root/codex-backups/passwall.before-lan-ai-gfw-<timestamp> /etc/config/passwall
/etc/init.d/passwall restart
sleep 8
```

Expected: 网络恢复到变更前状态。

- [ ] **Step 2: 回滚后复核**

Run:

```powershell
nslookup api.ipify.org 192.168.5.1
curl.exe -I -L -sS --max-time 12 https://www.google.com/generate_204
```

Expected: DNS 正常，Google 204 正常。
