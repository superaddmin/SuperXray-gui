# v3.7.3 上游逐项对照与集成记录

核对日期：2026-11-15。产品版本：`3.7.3`。本轮选择性适配两个上游稳定版区间的更新，逐项记录 130 条提交；“逐项审查”表示每条有本地状态、处理决定和依据，不表示所有上游功能均已合并。上游稳定版之后的 `main` 提交不在本轮范围。

## 固定范围与证据

| 项目 | 起点 | 固定目标 | 增量提交数 |
|---|---|---|---:|
| 本地 SuperXray | `v3.7.2` 的提交 `5f494f90f436361508ed22827c4b5e1a890c65b9` | 本次 `v3.7.3` 发布提交 | — |
| `MHSanaei/3x-ui` | `v3.10.2`：`608ef4e4010bf83e2e78328e08f77efc051ad4d7` | `v3.11.0`：`c621ba9014a6aa21fd7c9ecf45655e44e24ebf13` | 68 |
| `alireza0/x-ui` | `v1.11.4`：`73b050c95e3fb25128baedc8668d72a45a7b5b6f` | `v1.15.0`：`807e7e24362021908ec9e10a4400caa8142214fb` | 62 |

逐项事实源为 [3x-ui 对照表](upstream-v3.7.3-3x-review.tsv)和 [x-ui 对照表](upstream-v3.7.3-alireza-review.tsv)。两表保留审查时的本地状态，处理决定及语义证据更新为本轮实际结果；同一功能可能对应多个提交，不能把“采用提交数”当作功能数。供应方版本号、产品版本号和运行核心版本分别维护。

本地引用可复核范围，标签不自动跟随上游移动：

```powershell
git rev-parse 'v3.7.2^{}' upstream-v3.10.2 upstream-v3.11.0
git rev-parse vendor/alireza0-xui/v1.11.4 vendor/alireza0-xui/v1.15.0
git rev-list --count upstream-v3.10.2..upstream-v3.11.0
git rev-list --count vendor/alireza0-xui/v1.11.4..vendor/alireza0-xui/v1.15.0
```

最后两项应分别为 `68`、`62`。历史全量盘点仍保留在 [v3.10.2 集成记录](upstream-3x-ui-v3.10.2-integration.md)和 [v1.11.4 集成记录](upstream-alireza0-v1.11.4-integration.md)，不能将历史 `inventory_only` 行推定为本轮完整采用。

## 实际吸收

| 更新及上游提交 | v3.7.3 行为 | 源码与回归 |
|---|---|---|
| 首次账号安全 `1df8ab29` | 新库保持无用户，管理员使用本地 CLI 初始化；安装器继续识别未初始化状态并显示其生成的账号，面板启动日志不输出初始凭据，已有账号保留 | `database/db.go`、`main.go`；`database/db_bootstrap_test.go`、`main_bootstrap_test.go`；[初始化说明](upstream-v3.7.3-bootstrap.md) |
| WireGuard 防注入 `0faad4f4`、订阅缓存 `0729fc6d` | 每一行拒绝控制字符；所有订阅格式、诊断和错误响应加 `no-store` | `sub/wireguard_subscription.go`、`sub/subController.go`；`sub/upstream_security_test.go` |
| 配额重置 `aedd8fac`、`1dbb3743` | 仅恢复配额自动关闭且未过期的入站；周期重置同步运行态，核心手动停止时保持停止 | `database/repository.go`、`web/job/periodic_traffic_reset_job.go`；`database/traffic_reset_test.go`、`web/job/periodic_traffic_reset_job_test.go` |
| cron 兼容 `982d996e` | 五字段和六字段计划共用可选秒解析器，保留配置时区、描述符和 `CRON_TZ` 解析能力 | `web/web.go`；`web/cron_schedule_test.go` 覆盖五/六字段下一触发时刻及非法表达式 |
| 正常关停 `ad470681` | Web/Sub 使用独立 5 秒关闭上下文，忽略已关闭监听器与未运行核心的预期状态 | `web/web.go`、`sub/sub.go`、`web/service/xray.go`；`sub/shutdown_test.go` |
| 公网 IP 缓存 `be253141` | 状态读取不等待外网；成功缓存 5 分钟，任一地址族失败后 1 分钟重试，保留最后成功地址且避免重复并发刷新 | `web/service/public_ip_cache.go`；`web/service/public_ip_cache_test.go` |
| TLS 输出 `2b759f39`、`c3d4a153` | Go/Vue 分享保留 `cs`，JSON 保留 `cipherSuites`；仅未指定指纹且自定义套件时补 `unsafe`，每个外部代理独立覆盖，明文链接移除 TLS 参数 | `sub/subService.go`、`sub/subJsonService.go`、`frontend/src/utils/inboundCompat.ts`；Go/JSON/前端 upstream 回归 |
| SS HTTP 伪装 `0b90dc2b` | Shadowsocks TCP HTTP 根路径映射 Mihomo `obfs`；不兼容路径和其他协议不映射 | `sub/subClashService.go`；`sub/upstream_protocol_test.go` |
| XDRIVE `61b129d5`、`28246a30`、`2ed88999` | 校验 VLESS、decryption/encryption、flow/testpre、普通出站 mux 与字段范围；服务别名归一；JSON 订阅补 mux，缺客户端 encryption 时跳过 | `util/xraycompat/modern.go`、`sub/subJsonService.go`；`util/xraycompat/upstream_compat_test.go`、`sub/upstream_json_compat_test.go` |
| Lua 互斥 `ccb92c70`、`2f458668` | 拒绝路由脚本与 rules/domainStrategy、DNS 脚本与互斥 DNS 字段的组合，不静默删除管理员配置 | `util/xraycompat/modern.go`；`util/xraycompat/upstream_compat_test.go` |
| KCP/finalmask `f15c8ede`、`53443f4a` | Go/TS 原子转换旧 KCP mask 到 `mkcp-legacy`；冲突及空 AES 口令报错；客户端 JSON/Vue 输出去除源 IP filter，数据库原值与其他扩展保留 | `util/xraycompat/modern.go`、`frontend/src/utils/inboundCompat.ts`；Go/JSON/前端 upstream 回归 |

其中首次账号修复吸收上游的安全目标，采用本地显式 CLI 初始化；不复制上游生成并记录随机账号的方式。关停修复没有修改 systemd `KillMode`。cron 改动没有新增设置层的计划校验接口。

依赖按本地工具链小步更新：Vite `8.2.1 → 8.2.6`，直接版本范围收敛为 `~8.2.6`；follow-redirects 间接锁定 `1.16.0 → 1.16.3`。生产依赖 `npm audit --omit=dev` 为 0 漏洞；全量审核仍有 8 项来自 typescript-eslint/minimatch 开发工具链的中危告警，未使用强制跨主版本修复。

首次完整发布门禁另发现 Go 标准库、brotli 和 compress 可达安全告警，因此 Go 工具链提升至 `1.27.4`，brotli `1.2.6 → 1.2.7`、compress `1.20.1 → 1.20.2`。这是本地安全门禁驱动的最小补丁，不代表复制上游完整依赖图；更新后需重新执行受影响的门禁。

## 已有覆盖与保留本地

| 上游内容 | 本地判断与依据 |
|---|---|
| 可信代理地址解析 `d50c419e` | `util/trustedproxy` 已从 XFF 右端跳过可信代理，不受旧左端优先行为影响；保留本地信任边界 |
| CSRF 拒绝审计 `3051aebb` | `StructuredRequestLoggerMiddleware` 已以 info 记录完成请求的状态码、method、路由模板和可信客户端 IP；避免重复专用日志，原始秘密路径不进入日志 |
| 高级 JSON 格式 `5a135d2a` | Vue 编辑器保留输入原文，仅显式格式化或保存时归一 |
| Shadowsocks 2022 API/密钥 `c4ae0ecd`、`571afa0d` | 已使用账户类型及按方法校验密钥；不引入会静默更换凭据的自动修复迁移 |
| 日志竞态 `81ef9478` | 已有 `logBufferMu` 与 logger 指针保护 |
| 页面语言 `cc47b081`、`be9a6fcb` | 已同步 `document.documentElement.lang` 和登录语言；该结论不扩大为全站无障碍验收 |
| Xray 核心与旧转换 | 运行时已固定 `v26.10.21`，原 UDP Hop/xDNS/WireGuard 转换、最低核心版本门禁和原子替换已有覆盖；Go API 类型依赖仍为 `v26.10.10` 基线，未全量升级 |
| OpenAPI/CI、SQLite 与客户端模型 | 继续采用本地 YAML 生成链、stale gate、Linux 双架构、SQLite 与嵌入式 JSON clients；不复制上游独立 clients/hosts/PostgreSQL 迁移 |

## 明确缺口与暂缓项

两表逐条列出理由；以下为会影响功能预期的主要缺口：

- **节点、Host 与 Token 模型**：node-admin scope、节点合并/心跳通知、Host CIDR/SNI 池/排序/客户绑定、按 subId 的新 ClientView API、索引化 Token 查询等依赖本地未采用的表与契约。补丁版不引入这些活跃模型。
- **性能重构**：Telegram 独立 clients 索引、串行流量 writer、全局流量心跳、订阅缓存、gRPC 连接池和跨节点 IP 合并需要按本地模型另行设计；不能以 Go 依赖已存在推定已实现。
- **Telegram**：可编辑屏幕、inline 列表、论坛主题与换库迁移均未采用；保留 telego 和既有回调所有权防护。
- **新协议与完整向导**：MASQUE、WARP MASQUE、TUIC、AmneziaWG 内核/用户态数据面未接入；XDRIVE/Google Drive、Lua、Finalmask 新字段、TLS 预设/ECH 辅助器与 Mojang 查询未提供完整专用 Vue 向导。高级 JSON 透传和 schema 校验不等于真实外部连接验收。
- **订阅扩展**：远程 Clash DNS/sniffer 模板、Mihomo XHTTP reuse-settings、完整分享导入器等未采用；本地没有对应拉取信任契约或传输能力。
- **运维与 UI**：LXC 特定监控、与 fail2ban 解耦的 IP 采集、全局 cron 流量重置、完整无人值守安装、全站无障碍增量审计和新增语言均暂缓。
- **构建/依赖**：不采用上游 React/Vue2 补丁、PostgreSQL CI 或跳过版本变更测试的策略。Go 已因漏洞修复提升至 `1.27.4`，Xray API 依赖整图升级暂缓；当前 Docker 支持的 amd64/arm64 映射正确，但未知架构默认分支改为拒绝的额外防御尚未实施，不新增其他发布架构。

## 升级兼容性与回滚

保留 Vue UI、既有 API、`database/model.Inbound` 活跃写模型、客户端 JSON、Xray 生命周期与只读 default-xray/CoreManager 边界；不恢复旧 HTML UI。

1. 升级前停止写入并备份 SQLite 数据库、配置、证书和旧二进制或容器 tag；不要将这些运行数据提交到仓库。
2. 数据库自动迁移新增内部 `inbounds.quota_disabled`，非空默认 `false`，不暴露给 JSON/表单。升级前已关闭且没有原因标记的入站继续关闭，即使历史原因可能是配额耗尽；管理员核实未过期和额度后手动启用一次。
3. 周期/单入站重置只恢复新版本明确标记为配额关闭的入站。手动操作清除内部标记，避免之后被周期任务重新开启；到期关闭不会自动恢复。
4. 新库没有默认账号；手工二进制/Docker 部署须使用同一数据库目录执行 CLI。已有库保持账号，历史默认凭据不会自动旋转，应主动修改。
5. 旧 KCP 配置在事务迁移/归一层转换，冲突或空 AES 口令会明确失败并回滚，不能通过删除字段绕过错误。XDRIVE/Lua 约束也可能使原先存储的冲突配置被拒绝，按报错调整后再保存或启动。
6. 需要回退时先停止新版本，再恢复旧二进制/容器和升级前数据库/配置备份。旧二进制忽略新增列，但没有配额恢复语义；已有归一转换和升级后写入不能仅靠替换二进制视作完整回滚。未初始化的空库回退旧版本可能重新生成固定默认账号，必须重新核对账号安全。

稳定发布后优先发布补丁修复，不重写已公开标签与资产。发布前失败则修复门禁后再创建标签，遵循[发布策略](../.codex/skills/superxray-release-cicd/references/release-policy.md)。

## 验证记录

以下为本轮本地执行证据；提交前完整门禁与隔离真实核心 E2E 均已通过，远端发布结果单独核实，不能以历史验证替代：

| 验证 | 结果与范围 |
|---|---|
| 初始化与迁移 `go test ./database ./web/service . -count=1` | 通过；新库、重复初始化、安装器检测、显式 CLI、已有账户和候选导入覆盖见初始化记录 |
| 新增上游聚焦回归 | Go 订阅、安全、协议、兼容、配额、cron、关停与公网 IP 缓存通过；相关 race 通过 |
| 真实 Xray `TestRealXray` 与新增配置验证 | 使用 `v26.10.21` 通过运行时恢复/迁移 fixture，并通过 27 项 XDRIVE/Lua `run -test` 配置验证；不表示 Google Drive/MASQUE 等外部业务链路已实测 |
| `go vet ./...`、`staticcheck ./...` | 通过 |
| 前端 `npm run typecheck`、`npm run test` | 通过，单测 `158/158` |
| 前端 `npm run lint` | 0 error；176 条既有 Vue 格式 warning 保留 |
| 前端 `npm audit --omit=dev` / 完整 `npm audit` | 生产依赖 0 漏洞；完整审核剩 8 项 moderate 开发依赖告警 |
| 最终提交前完整发布门禁 | `release_gate.py --tag v3.7.3 --allow-dirty` 通过：元数据、tidy、全量 Go test/race、vet、staticcheck、govulncheck、gosec、二进制版本及秘密扫描均通过；`--allow-dirty` 仅允许尚未提交的发布工作区，不表示最终发布可跳过干净工作区与 CI 检查 |
| Go 漏洞与安全复核 | `govulncheck` 可达漏洞为 0；另有 3 项位于 required modules 中但未被当前代码调用的漏洞，不影响扫描通过结果。安全复核无 CRITICAL/HIGH 项 |
| 前端 `npm run build` | 通过，Vite `8.2.6`；OpenAPI `3.7.3` 已生成并嵌入 `web/ui` |
| 文档与审查清单 | 两份 TSV 与固定 `git rev-list` 集合完全一致，共 68/62 条且无重复；新增记录/初始化/索引链接有效，UTF-8 无 BOM 与 LF 校验、`git diff --check`、秘密扫描通过 |
| 隔离真实核心 E2E | `21 passed / 0 failed / 0 skipped / 0 flaky`，耗时 68.312 秒，退出码 0；Go `1.27.4`、面板 `3.7.3`、真实 Xray `26.10.21`；变更、重启与数据库导入开关全部开启，重启和导入实际执行，合成 VLESS 订阅返回 HTTP 200 且含 `no-store`；测试结束进程残留为 0 |
| GitHub Release/双架构资产/GHCR | 发布后核实 [v3.7.3 Release](https://github.com/superaddmin/SuperXray-gui/releases/tag/v3.7.3)及对应 [Actions](https://github.com/superaddmin/SuperXray-gui/actions)；本地门禁通过不证明远端发布已成功 |

复核入口：

```powershell
go test ./database ./web/service ./web/job ./web ./sub ./util/xraycompat . -count=1
go test -race ./database ./web/service ./web/job ./web ./sub ./util/xraycompat -count=1
go test ./web/service -run TestRealXray -count=1 -v
go vet ./...
staticcheck ./...
python scripts/secret_scan.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --tag v3.7.3 --allow-dirty
```

真实核心测试须设置 `SUPERXRAY_TEST_XRAY_BINARY`，未设置会跳过，跳过不能记为通过。完整发布入口仍为 `release_gate.py --install-tools`，前端依次执行 typecheck、lint、test、build；E2E 需隔离数据库和真实核心环境。

本次 E2E 在仓库外临时工作目录调用仓库的 Playwright CLI，并用 `--config` 指向仓库配置，等价执行 `npm run e2e` 对应测试，避免读取真实 `.env`。`SUPERXRAY_E2E_MUTATION`、`SUPERXRAY_E2E_RESTART`、`SUPERXRAY_E2E_IMPORT_DB` 均开启；账号、数据库、合成订阅及测试产物均位于隔离临时环境，没有将真实凭据或运行数据写入仓库。该结果验证本机隔离链路，远端 CI、双架构发布资产和 GHCR 仍须按实际运行结果验收。
