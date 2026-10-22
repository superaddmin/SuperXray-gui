# 上游同步策略与落地雷达

> 代码核对日期：2026-10-15；历史上游雷达快照日期：2026-08-11
> 上游仓库：`MHSanaei/3x-ui`、`alireza0/x-ui`
> 当前固定审查目标：`MHSanaei/3x-ui v3.10.2` = `608ef4e4010bf83e2e78328e08f77efc051ad4d7`；共同基线仍为 `v3.3.1` = `b5ef412b8db999a6b981e1550762551d7b06540f`。另一供应方基线：`alireza0/x-ui v1.11.4` = `73b050c95e3fb25128baedc8668d72a45a7b5b6f`。固定目标不代表上游最新版本，也不代表所有上游功能均已采用。
> 当前项目版本：`3.6.0`；Go：`1.27.2`；运行核心：Xray `v26.10.21`（兼容基线 `v26.10.10`）

## 当前 v3.10.2 审查入口

本次固定上游标签、共同基线、完整提交列表、文件差异和实际处理结论统一记录在 [v3.10.2 集成记录](upstream-3x-ui-v3.10.2-integration.md) 及其机器清单中。该记录明确区分已适配、保留本地实现、未采用和仍需人工处理的场景；本文件的历史雷达不再作为本次集成结果的事实源。

以下是当前工作区已存在的适配摘要，详细来源、转换规则及测试环境以该集成记录为准：

| 范围 | 当前处理 |
|---|---|
| 依赖与运行核心 | Go 1.27.2、Xray 运行时 v26.10.21（兼容基线 v26.10.10）和其 gRPC API 依赖固定在 `go.mod/go.sum`；Vue 依赖按本地工具链更新，不复制上游 React lock |
| 协议与旧配置 | `util/xraycompat` 归一 XHTTP sessionID、Reality dest、sockopt、UDP Hop、xDNS、DNS outbound、WireGuard/freedom 策略及 TLS pin；未知字段保留，冲突或废弃字段需明确报错，不能静默丢弃 |
| 配置和核心替换 | 完整候选配置预检、原子配置写入、归档 SHA-256 校验、候选核心/资产验证及失败恢复；不改变 legacy Xray 生命周期所有权 |
| 数据库 | SQLite schema/seed/历史配置转换共用事务；导入独立候选库、完整配置预检、维护租约与失败恢复；保留 JSON 客户端模型 |
| 安全与数据保护 | TOTP 修改确认、可信代理 CIDR、LDAP 空目录保护、Telegram 回调授权、入站写入主键/并发/未知字段保护、Vue 配额原始字节保留 |
| 本地交付 | 保留 Vue、OpenAPI 生成链、中文部署脚本及双架构；Docker 依赖缓存/可配置下载代理/有界重试已验证，发布不由本次文档核对触发 |

此前隔离验证覆盖 Windows/Linux Go 测试、amd64/arm64 镜像和 19 条 Docker 功能 E2E；arm64 运行使用 QEMU，不能替代原生 arm64/生产环境验收。历史 973 条提交与文件清单用于完整盘点，`inventory_only` 不是逐项语义审查完成标记。

## 1. 同步原则

SuperXray-gui 已在上游 3x-ui 基础上加入 Vue 3 新 UI、旧 HTML UI 退役、Phase 9 安全收口、Gateway Egress MVP 与风险接受的最小 CoreManager/sing-box 后端入口。后续同步上游时采用“基线记录 + 选择性移植 + 本地门禁验证”的策略，不直接用上游文件覆盖本项目。

必须保留的本地边界：

- 不重新引入 `/panel/legacy/`、`web/html`、`web/assets` 作为生产 UI。
- 不让 CoreManager 接管 legacy Xray 的 start/stop/restart。
- 不把活跃写模型从 `database/model.Inbound` 迁移到 `proxy_inbounds` / `proxy_clients`。
- 不新增生产 `egress_*` 数据库/API。
- 不把 experimental sing-box 提升为生产默认。
- 不使用 `v-html`、`innerHTML`、`insertAdjacentHTML` 渲染日志、配置、订阅或外部内容。

## 2. 历史已同步内容（截至 2026-08-11）

以下是 2026-08-11 之前的同步/衍生落地记录。版本号描述的是当时状态，当前依赖以文首及锁文件为准；历史“测试通过”不代表本次文档核对重新执行或远端已部署：

1. 记录 `MHSanaei/3x-ui upstream/main` 的合并基线。
2. 移植 finalmask fragment 空/零长度防崩溃修复。
3. 同步 Go 版本到 `1.26.5`，并更新 Xray-core、gRPC、fasthttp、gopsutil、telego、validator、`x/crypto` 等依赖。
4. 补齐 release gate：`go.mod` 的 `go` 指令必须与 `.codex/project.toml` 的 `[stack.backend].version` 一致。
5. 补齐 settings 空值 fallback，避免非空默认配置被空字符串持久化后导致启动或读取失败。
6. 增强 Clash/Mihomo 订阅 routing rules，支持复用订阅设置中的行分隔规则并保留 `MATCH,PROXY` 兜底。
7. 增强订阅 diagnose 输出，返回支持格式和协议能力元数据。
8. 扩展 default-xray CoreManager lifecycle 拒绝回归测试，确保 `Validate/Start/Stop/Restart` 均不接管 legacy 生命周期。
9. 针对上游 `3x-ui v3_3_0` 的 WARP 出站修复，新增 `panelProxy` 设置、HTTP/HTTPS/SOCKS5 出站客户端构造器，并让 WARP API 请求使用面板代理。
10. 在 Vue Settings 页面暴露 `Panel Outbound Proxy`，保持新 UI 与 legacy setting form 的 `panelProxy` 字段一致。
11. 增强 release metadata gate，加入 `docs/openapi/panel-api.yaml` -> `frontend/public/openapi.json` 生成物陈旧检查，避免 OpenAPI 文档和 Vite public 输入漂移。
12. 对上游订阅相关修复执行本地回归确认：Clash rules、VLESS encryption、public/fallback、Shadowrocket、routing/IPLimit 相关聚焦测试通过。
13. 选择性同步 `3x-ui v3.3.1` 的 Xray 日志路径安全修复：运行时生成配置时将 `log.access` / `log.error` 限定到面板日志目录下的 basename，并补充回归测试覆盖绝对路径、`..` 穿越与 `none` / 空值禁用语义。
14. 同步 `golang.org/x/net v0.56.0` 及随动 `golang.org/x/exp`、`golang.org/x/tools` 更新。
15. 引入项目本地 `.gitattributes`，仅约束 Shell 脚本、OpenAPI 生成物、前端测试和 `web/ui/openapi.json` 为 LF；不引入上游 React 生成目录约束。
16. 按功能移植 `alireza0/x-ui v1.11.4` 的自签名证书、Hysteria QUIC、TUN、XHTTP/XMUX 与 Go 依赖更新；逐字段保留旧配置和未知 JSON 扩展。
17. 评估后保留本地 JSON 显式默认值、客户端高级 JSON 兼容入口、WARP Matrix 与 `3.4.3` 版本号，不采用上游面向旧 HTML UI 的删除或全局收缩改动。
18. 保持 `alireza0-xui` 远程只读，设置 `remote.alireza0-xui.pushurl=DISABLED`；完整集成记录见 `docs/upstream-alireza0-v1.11.4-integration.md`。

## 3. 历史上游雷达及当前处理

已记录的上游 `3x-ui v3.3.1` 重点包括（历史候选及决策，不是当前远程变更扫描）：

- `80e168787`：将 Xray `log.access` / `log.error` 约束在面板日志目录，修复 GHSA-jm48 任意文件写入风险；本项目已按当前 `web/service` 架构移植。
- `eee652c4a` / `cd46730bb`：更新 `golang.org/x/net` 与前端 lock；本项目已同步 Go 依赖，前端 lock 因技术栈不同暂不采用。
- `41645255f`：上游大规模迁移到 `internal/` 布局；本项目暂不采用，避免破坏当前 Go 包路径、Phase 9/10 边界和发布脚本。
- 上游前端已是 React 19 + Ant Design 6，新增大量 TSX 页面与 `internal/web/dist` 输出；本项目保持 Vue 3 / Vite / `web/ui`，不直接覆盖 `frontend/src`。
- `6b16d8c37`、`7bcc5830c`：Xray gRPC live apply 与 online-stats API；本项目暂缓，需单独评审旧 Xray 生命周期、IP limit 与 CoreManager 边界。
- `ca4f32e3d`、`8578b229c`：`panelProxy` 升级为 outbound/balancer based panel egress；本项目当前保留已落地的 `panelProxy`，后续如迁移必须做 SSRF/凭据/路由回滚评审。
- `3092326d9`：用 Xray-core 原生 geodata auto-update 替代 custom geo manager；本项目暂缓，避免直接删除现有 CustomGeo API/UI 与运维流程。
- `2a7342baa`、`cc65f3716`、`f1a4286e2`：入站分享地址策略和订阅排序；适合作为 Vue UI + 订阅服务后续 P1 移植候选。
- `4002be4ad`、`d04cb1097`：当时列为 WireGuard 字段/per-peer comment 候选。当前已有 WireGuard peer 编辑与协议导出，v3.10.2 兼容层另处理 workers、DNS 和策略转换；但 `buildClientPayloadFromEditor` 对 WireGuard 删除 `comment`，不能把共享表单中出现备注框当作 per-peer comment 已支持。两条完整提交仍需逐项比较。
- `0766e1668`、`60da6bed1`：当时列为 XHTTP XMUX 与默认值候选。XMUX 已在 alireza0 v1.11.4 功能移植中落地，v3.10.2 又适配 sessionID、extra 和服务端/客户端投影；移除指纹化默认值不作为本项目已完整采用的结论。
- `6c1594693`、`5eec17848`：MTProto domain-fronting 与通过 Xray routing 出站；需 sidecar 生命周期与路由安全评审后再进入。
- `57e966175`、`2db48174b`、`f88f53cd7`、`65fa40b81`：fail2ban、BBR、update webBasePath restart、API token CLI 修复；属于脚本/CLI 面，需单独按本项目 install/update/x-ui.sh 本地化策略审查后移植。

历史 `3x-ui v3_3_0` 重点包括：

- `a32c6803d`：WARP API 请求走 `panelProxy`，解决受限网络下 Cloudflare WARP 注册/授权请求不可达问题。
- `d9ccf157c`：新增手动/自动 WARP IP rotation；本项目只先落地低风险代理出站，不直接引入自动轮换 job。
- `1ca5924a` / `f8e89cc84`：MTProto/mtg sidecar、日志可见性与孤儿进程清理；本项目暂缓，避免引入新的生产 sidecar 生命周期。
- `0daedd3d`：subscription-based outbounds auto-update；本项目暂缓，避免引入未经 Phase gate 的外部订阅自动写配置路径。
- `abf6b879`：custom subscription pages；本项目暂缓，需先完成模板/XSS/外部内容渲染审查。
- `e6c1ce9a`、`1c74b995`、`9f31d7d0`、`be8bd4e22`：node tree、多跳归因、access.log 同步与流量 reset 传播；本项目暂缓，避免越过当前 legacy model 与 CoreManager 边界。
- `98ba8803`、`46684dd1`、`2b4e199a`、`e409bc30` 等订阅/Clash/IP limit 修复：本项目优先通过本地订阅矩阵测试确认，必要时再按 TDD 小步移植。
- `a014c017`、`83799d71`、`e56f6c63`：OpenAPI schema/examples/server base path 优化；本项目保持 Vue 自有 OpenAPI 方案，并新增生成物 stale gate。

## 4. 优先级策略

### P0：持续保护与优先移植

- 安全/崩溃修复：已落地的 finalmask、settings fallback、订阅空值防御、TOTP 和代理信任边界持续回归，新发现修复优先按本地调用链适配。
- 依赖与元数据漂移门禁：`go.mod`、`.codex/project.toml`、release gate。
- 订阅 diagnose：只暴露聚合能力和跳过原因，不暴露 client UUID、真实 sub URL、token 或代理凭据。
- WARP API 面板代理：只允许管理员配置的 `panelProxy` 影响面板自身出站请求；无效代理必须回退直连并记录 warning。
- OpenAPI stale gate：生成物不一致时 metadata-only gate 必须失败。

### P1：订阅与协议增强

- Clash/Mihomo routing rules 已有本地实现，后续增量继续维护输出语义和安全引用。
- XHTTP/sockopt/Reality 等已有 v3.10.2 兼容规则；后续 ECH 或其他字段增量仍需补本项目新 UI/旧 API/订阅矩阵测试，不能因核心升级推定面板全部支持。
- Shadowrocket base64 行为需单独评估当前客户端兼容性，不能破坏已有 Generic URI 输出。
- Settings UI 可以暴露与 legacy 设置兼容的低风险字段，但必须通过 `frontend/tests/settings-view.test.ts` 覆盖字段绑定。

### P2：CoreManager 与供应链门禁

- default-xray 继续保持 read-only 与 legacy lifecycle owner。
- experimental sing-box 只能在显式 binary/config 存在时作为实验入口使用。
- 二进制下载、迁移脚本和 DB dump/import 继续遵守允许范围、SHA256、私有临时目录和维护锁；已有核心升级/导入验证与恢复流程不得被新上游实现绕过。
- MTProto/mtg sidecar、OutboundSubscription 自动刷新、node tree/access.log 跨节点同步暂缓；进入生产前必须单独完成生命周期、权限、XSS/SSRF、回滚和旧模型兼容评审。

## 5. 标准执行流程

先核对远程名称与供应方 URL；下列 `upstream` 只适用于已配置的 MHSanaei/3x-ui。另一个供应方使用 `alireza0-xui` 和独立 `vendor/alireza0-xui/*` tag，不混用两个供应方的基线。远程读取和发布执行是不同操作，本文命令不会自动授权推送。

```powershell
git remote -v
git fetch upstream main --prune
git rev-parse upstream/main
git log --oneline HEAD..upstream/main
git log --oneline upstream/main..HEAD
```

如上游有新提交：

1. 先分类为安全修复、协议输出、UI/i18n、脚本/发布、数据库/迁移、节点同步。
2. 对每个候选提交写最小回归测试，先确认 RED。
3. 只移植必要代码，不覆盖本地 Phase 9/10 边界。
4. 跑聚焦验证：

```powershell
go test ./sub ./web/service ./web/controller ./core/... -count=1
python -m unittest discover -s .codex/skills/superxray-release-cicd/tests -p test_release_gate.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```

5. 发布前跑全量验证和 secret scan。

当前落地证据索引：

| 功能 | 源码与回归入口 |
|---|---|
| 日志路径限制与新核心日志兼容 | `web/service/xray.go`、`web/service/xray_log_path_test.go`、`web/service/xray_logs_compat_test.go` |
| 设置默认值与 panelProxy | `web/service/setting.go`、`web/service/setting_test.go`、`util/netproxy/`、`web/service/warp_security_test.go` |
| 订阅规则和诊断 | `sub/subClashService.go`、`sub/subscription_diagnostic.go`、`sub/subscription_output_matrix_test.go` |
| legacy 生命周期保护 | `web/service/core_service.go`、`web/service/core_service_test.go` |
| 自签证书与上游协议表单 | `web/service/self_signed_cert.go`、`web/service/self_signed_cert_test.go`、`frontend/src/utils/inboundCompat.ts` |
| Xray 配置兼容与预检 | `util/xraycompat/`、`xray/validate.go`、`web/service/xray_setting.go`、`web/service/xray_real_core_test.go` |
| 数据库迁移、候选导入与恢复 | `database/db.go`、`database/maintenance.go`、`web/service/database_import.go` 与对应测试 |
| TOTP 与可信代理 | `web/service/setting_totp_test.go`、`util/trustedproxy/`、`frontend/tests/settings-view.test.ts` |
| 配额与本地 UI 保护 | `frontend/src/utils/trafficQuota.ts`、`frontend/tests/traffic-quota.test.ts`、`frontend/tests/inbounds-view.test.ts` |
| Go 与 OpenAPI 元数据门禁 | `go.mod`、`.codex/skills/superxray-release-cicd/scripts/release_gate.py`、`tools/openapiexport/` |

版本与依赖以 `config/version`、`go.mod`、`frontend/package.json` 和各自 lock 文件为事实源；供应方基线 tag 不替代产品版本。历史候选的外部实现和当前供应方发布状态需要重新抓取后审查。

## 6. 回滚策略

- 单个同步点必须保持小提交；如验证失败，用 `git revert <commit>` 回滚该同步点。
- 生产部署前必须备份 SQLite DB、旧二进制/容器 tag、配置目录和 systemd unit。
- 若订阅输出异常，优先回滚订阅服务相关提交，不回滚无关 UI 或 CoreManager 代码。
- 若 CoreManager 相关 API 异常，必须确认 legacy Xray lifecycle 不受影响；禁止通过 CoreManager 强行重启 default-xray。
