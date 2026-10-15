# `MHSanaei/3x-ui v3.10.2` 集成记录

本记录对应 `integration/3x-ui-v3.10.2` 分支，按本项目[上游同步策略](upstream-sync-policy.md)审查固定上游版本并选择性移植。产品版本仍为 `3.4.3`。本次工作保留 Vue、SQLite 入站 JSON 模型、现有 API 和部署边界；完整历史清单表示比较范围，不表示 973 个上游提交的全部功能或修复均已合入。

## 1. 固定基线与证据边界

| 项目 | 固定值 |
| --- | --- |
| 上游仓库 / 标签 | `MHSanaei/3x-ui` / `v3.10.2`，本地 ref 为 `upstream-v3.10.2` |
| 上游提交 | `608ef4e4010bf83e2e78328e08f77efc051ad4d7` |
| 共同基线 | `b5ef412b8db999a6b981e1550762551d7b06540f`（`v3.3.1`） |
| 任务开始时本地提交 | `d87b7c157f3c764156f0d1346cb2302517daff66` |
| 本地独有 / 上游独有提交 | 131 / 973 |
| 共同基线 → 本地文件差异 | 1184 条 |
| 共同基线 → 上游文件差异 | 1978 条 |
| 本地 → 上游直接树差异 | 2522 条 |
| 本项目版本 | `config/version`：`3.4.3` |
| 目标运行核心 | `Xray v26.10.10`，提交 `701af60772cd` |
| Go 模块中的核心固定版本 | `github.com/xtls/xray-core v1.260327.1-0.20261010092107-701af60772cd` |
| Go / gRPC 固定依赖 | `go 1.27.2` / `google.golang.org/grpc v1.86.0-dev` |

清单均从以上不可变提交生成，不包含本轮尚未提交的工作区修改：

- [本地独有提交清单](upstream-3x-ui-v3.10.2-local-commits.tsv)：131 条完整 commit ID、日期、原始标题。
- [上游独有提交清单](upstream-3x-ui-v3.10.2-upstream-commits.tsv)：973 条完整 commit ID、日期、原始标题。
- [两侧文件差异清单](upstream-3x-ui-v3.10.2-file-manifest.tsv)：3162 条，分别记录 `base → local` 和 `base → upstream`，包含状态、重命名前路径、目标路径、领域、处理策略。
- [本地与上游直接树差异](upstream-3x-ui-v3.10.2-tree-diff.tsv)：2522 条，用于查看两者最终文件树差异。

TSV 使用真实制表符并保留原始提交标题；标题中的英语来自 Git 历史。`review_state=inventory_only` 表示“已完整盘点，未凭此认定逐行语义审查完成”。文件领域与策略由路径路由得到，只决定后续审查方向，不能替代调用链分析。下文有来源和落点的条目才是本轮具体处理记录。三点比较 `upstream...local` 仅显示共同基线到本地一侧，不能当作双侧完整清单。

可复现的只读命令（PowerShell）：

```powershell
$base = 'b5ef412b8db999a6b981e1550762551d7b06540f'
$local = 'd87b7c157f3c764156f0d1346cb2302517daff66'
$upstream = '608ef4e4010bf83e2e78328e08f77efc051ad4d7'
git merge-base $local $upstream
git rev-list --count "$upstream..$local"
git rev-list --count "$local..$upstream"
git log --format='%H%x09%ad%x09%s' --date=short "$upstream..$local"
git log --format='%H%x09%ad%x09%s' --date=short "$local..$upstream"
git -c core.quotepath=false diff --name-status --find-renames $base $local
git -c core.quotepath=false diff --name-status --find-renames $base $upstream
git -c core.quotepath=false diff --name-status --find-renames $local $upstream
```

## 2. 本地定制保护矩阵

131 条本地历史包含产品功能、供应方移植、合并基线、生成物、发布与文档提交，并非 131 个独立功能。以下矩阵按活跃职责归并；完整来源仍以提交清单为准。

| 本地定制 / 来源示例 | 当前落点 | 本轮处理与影响 |
| --- | --- | --- |
| Vue UI-first、旧 HTML UI 退役：`c542bc7c6`、`446016ebd` | `frontend/src`、`web/ui.go`、`web/ui` | 保留 Vue 3 / Vite 入口及静态资源部署；仅将协议字段适配到现有表单，不恢复 legacy UI，不引入上游 React 构建目录。 |
| 入站表单、批量客户端、协议工具、分享及订阅入口：`5bf752383`、`149308d9a`、`88457ae8d` | `InboundsView.vue`、`inboundCompat.ts`、`sub/` | 保留 JSON 客户端保护、批量/克隆/导出流程；新字段纳入读写边界和回归。 |
| Gateway Egress MVP、HTTP/Mixed 代理账号与 URI：`c79561ce8`、`b87b4adc3`、`380c9b37b`、`cb7867a6a` | `core/`、`frontend/src/utils/gatewayEgressMvp.ts`、入站/订阅服务 | 保留现有 MVP 端口、监听地址与输出契约，不新增生产 `egress_*` 表/API。 |
| 住宅出口池、AI 分流、WARP Matrix：`f4f852cdc`、`5366eada4` | `frontend/src/utils/xrayCompat.ts`、`XrayView.vue`、WARP 服务 | 保留本地路由规则、balancer 和出站编排；不接入上游自动 WARP rotation 或新 panel egress 模型。 |
| CoreManager / `default-xray` 只读视图与实验 sing-box：`c542bc7c6`、`0caa8d648`、`1ba803621` | `core/`、`web/service/core_service.go` | 保留 Phase gate，legacy Xray 仍由 `web/service/xray.go` 管理生命周期。 |
| 安全边界、健康与指标 API：`ddc2837a7`、`30e116c53`、`fc932ce77` | `web/middleware/`、`web/session/`、`web/controller/` | 保留 CSP、CSRF、session nonce、登录下载、路径/URL 检查及指标基数限制；新增可信代理策略和 TOTP 修改确认。 |
| 流量失败重试与 Geo 原子回滚：`a0aa4086d`、`aa2b81df6`、`82eb0261c` | `web/job/xray_traffic_job.go`、CustomGeo 服务、`logger/` | 保留事务失败的 pending 增量和逐项确认，不用上游统计写模型覆盖；外部通知增加有界超时。 |
| `alireza0/x-ui v1.11.4` 已有移植：`b32798947`、`7d998f747`、`74ae3cf40`、`95f9daa42` | 自签证书服务、Vue TLS/TUN/Hysteria/XHTTP 表单、订阅矩阵 | 保留自签证书、TLS 内联证书、TUN 兼容和 panelProxy SOCKS5H；新核心适配不撤销已有字段与测试。 |
| Vue OpenAPI 文档及 stale gate：`06872d1b7`、`8fc01334d` | `docs/openapi/`、`tools/openapiexport`、`frontend/public/openapi.json` | 保留本地路由、envelope 和文档生成链；不引入上游 React 文档站。 |
| 品牌、中文运维、Linux 双架构与发布门禁：`b0d61e52a`、`d8d171ece`、`7ddef7a83`、`15f5661a0` | `install.sh`、`update.sh`、`x-ui.sh`、Docker、`.github/`、`.codex/` | 保留 SuperXray 名称、`3.4.3` 版本、amd64/arm64 资产、部署路径与秘密扫描；只同步必要工具链和核心校验。 |
| 部署交付与路由排障知识：`506d755b8`、`d87b7c157` | `.codex/skills`、`docs/` | 保留现有部署技能与 OpenWrt/Passwall 说明，不用上游文档覆盖。 |

## 3. 本轮上游来源与适配落点

以下是按行为适配，不代表整条提交原样 cherry-pick。一个上游提交可能还包含未采用的 React、规范化数据库或节点功能。

| 上游来源 | 本轮行为 | 本项目落点 / 验证入口 |
| --- | --- | --- |
| `73a028f52`、`82c771118` | 对齐 Xray v26.10.10 与 Go 依赖；本地 Vue 安全补丁独立选取，未复制 React lock | `go.mod/go.sum`、Docker、release workflow、`frontend/package*.json` |
| `fea3c94b1`、`539bcc897`、`041476a31` | XHTTP `sessionID*` 别名、table/length、订阅字段；保留本地 `extra` 合并方式 | `util/xraycompat`、`inboundCompat.ts`、`InboundsView.vue`、`sub/` 相关测试 |
| `66a9a788f`、`7db92d631` | Reality `dest` 兼容；拒绝 Reality 与非空 TCP finalmask 组合 | `util/xraycompat`、Vue 兼容层与回归 |
| `62423cacd`、`73a028f52`、`ec0a57fc4`、`6321297f6`、`0466e8b0b` | xDNS / UDP Hop 结构、客户端投影、重复/冲突处理、Salamander 校验 | `util/xraycompat`、入站生成、JSON/URI/Clash 订阅、Vue 回归 |
| `39ce7cbc2`、`bc424f096`、`826e29e2d` | DNS outbound 旧规则及 qType 0、freedom 策略位置迁移 | `util/xraycompat/compat.go`、`database/db.go`、迁移测试 |
| `4c177f0cf`、`fea6a20f7`、`2bb29468d` | SS-2022 runtime Account、cipher 白名单、API 初始化防御和 gRPC deadline | `xray/api.go`、`xray/api_test.go`、`xray/log_writer.go` |
| `523a593ca`、`ad5f2a28c`、`baef3cdd0` | 原子配置写入、生命周期同步；在旧核心停止前做完整候选配置预检 | `xray/process.go`、`xray/validate.go`、`web/service/xray.go`、真实核心/生命周期测试 |
| `2bb851dd5` | 上游核心归档 SHA-256 校验；追加本地完整配置与 staged assets 预检、失败恢复 | `web/service/xray_update.go`、`server.go`、Docker/release 资产校验与回归 |
| `18b337d13`、`23511108b` | 关闭被替换的连接池、数据库 owner-only 权限；候选 DB / 租约 / 回滚按本地架构实现 | `database/db.go`、`maintenance.go`、`web/service/database_import.go` 与迁移/导入测试 |
| `0f6e1ae8d`、`d6cddaff1`、`7fe9932d7` | JSON 本地 listener 绑定 loopback、Vision mux 限制、TLS pin 字符串、Clash 标量安全引用 | `sub/default.json`、`subJsonService.go`、`sub/clash_yaml.go` 与订阅矩阵 |
| `ad288a7ec`、`169cd86e0` | 可信代理才可提供 forwarded URL；不再将 X-Real-IP 当作订阅服务器主机 | `util/trustedproxy`、`sub/subService.go`；扩展到限流、日志、session/CSRF scheme |
| `c8a3a2d72` | 替换已启用的 TOTP secret 必须验证旧认证器；本地同时保护禁用与首次启用路径 | `web/controller/setting.go`、`web/service/setting.go`、Vue Settings 与 TOTP 测试 |
| `f4b7b08e0` | LDAP 空目录不触发全部自动删除；本地额外修正配置 tags/Truthy 默认值混用 | `web/job/ldap_sync_job.go`、对应回归 |
| `5e1cb7693`，按本地调用链适配 | 缺字段订阅及截断日志不再 panic；创建入站不复用外部主键；四条入站/客户端写路径提交失败完整回滚；保留未知字段与大整数；监听通配地址冲突检查 | `sub/*Service.go`、`server.go`、`inbound.go`、`database/repository.go`、`web/entity` 及回归 |
| `5e1cb7693`，按本地 Telegram 自助契约适配 | 非管理员回调限制为本人订阅操作，长回调解码后再授权；保护管理员向导状态，延迟消息删除不清除新状态 | `tgbot.go`、`tgbot_callback_auth_test.go` |
| `5e1cb7693`，适配 Vue 编辑器 | 未编辑配额时保持原始字节数，避免备注编辑使小配额变为无限；显式修改和设置零仍生效 | `InboundsView.vue`、`trafficQuota.ts` 与行为测试，覆盖入站/客户端/默认客户端 |
| `bedbe04bf` 与本地重启调度补强 | cron panic 恢复并释放数据库租约/非重叠令牌；手动停机不会被自动重启，失败重启保留待重试标记 | `web/web.go`、`cron_test.go`、`xray.go`、`xray_runtime_test.go` |
| 本地补强，未认定等价于某条完整上游提交 | 流量外部通知 3 秒超时；候选 DB 完整生成后预检；维护租约排空与失败恢复 | `xray_traffic_job.go`、`database_import.go`、`database_lease.go`、`web/web.go` |

未列入上表的上游历史条目不能自动标记为“已合入”或“已无风险”。例如批量审计提交 `5e1cb7693` 含多个修复，不能凭其中一个本地修复宣称整批问题均已处理。

## 4. 明确保留或未采用的上游变化

| 上游范围 | 原因与处理策略 |
| --- | --- |
| `internal/` 目录重构、React 19 / Ant Design / TypeScript 7 / oxc、React 文档站 | 保留当前 Go 包布局、Vue 3 / TypeScript 6 / ESLint / Prettier 与 `frontend/src → web/ui`；可复用协议语义，不复用 UI 文件树。 |
| normalized clients / client_inbounds / managed Hosts、PostgreSQL 及其迁移/导出格式 | 会改变 `database/model.Inbound` JSON 客户端、外键、权限和 API 契约；本轮只升级现有 SQLite 数据，不提供跨库迁移器。 |
| 新 node tree、mTLS node sync、normalized 统计汇总与跨节点 bulk apply | 与当前节点/统计模型不同；保留本地节点功能，不导入依赖新模型的同步路径。 |
| MASQUE、XDRIVE、原生 TUIC、AmneziaWG、MTProto/MTG/PIA 新运行时和 UI | 不在本项目现有生产协议/生命周期范围；升级核心不等于新增面板协议支持。高级模板仍须通过目标核心校验。 |
| 上游 panel egress bridge、subscription-based outbound 自动刷新、WARP 自动轮换 | 保留 panelProxy、Gateway MVP、WARP Matrix 与本地更新触发方式，避免扩大外部内容自动写配置和 sidecar 生命周期。 |
| PostgreSQL CLI/安装器、AWS golden image、上游域名/品牌/版本与发布矩阵 | 保留中文运维脚本、现有路径、Linux amd64/arm64 资产和 SuperXray 版本管理。 |
| 上游 TOTP 相邻时间窗口（`d440c2b93`）等独立行为扩展 | 当前 TOTP 修改确认验证当前窗口；本轮未宣称实现相邻窗口容错，需保证系统时间同步。 |

## 5. Xray 配置迁移与失败语义

- 保留未知 JSON 字段及大整数表示；`xray.Config.Extra` 保存已建模字段以外的顶层内容。兼容层针对已知历史字段转换，不把未知字段自动当作有效 Xray schema。最终核心预检可能明确拒绝无法运行的扩展。
- `streamSettings.method → network`；Reality `dest → target`；sockopt 接受 `V6Only/v6only`，后端按目标核心字段写法归一。**Reality 禁止任何非空 TCP finalmask**，空数组允许；不把冲突配置静默删减为可运行配置。
- XHTTP `sessionPlacement/sessionKey → sessionIDPlacement/sessionIDKey`，根与 `extra` 都兼容。保留 `sessionIDTable/sessionIDLength`；单值、范围、数字类型、ASCII 和边界按照目标核心模型校验。服务端运行时去掉 XMUX/download 等客户端专属投影，持久化和订阅仍保留。
- 旧 `quicParams.udpHop` 转为 `udp` 中 `type=udphop`、`mode=intervalRemote`、`remotePorts`。保留扩展；相同 hop 去重并置于末尾，不同配置的重复 hop、缺失 ports 或冲突字段拒绝迁移。入站 server 运行配置移除客户端 remote-hop 投影；客户端订阅/分享继续携带跳跃配置。Vue 表单兼容旧形态及当前可编辑形态。
- xDNS 旧字符串域名（A/AAAA/TXT）、`name`、旧 resolver 转为 `names/types/addrs`；未知 record suffix、空名称和无法明确转换的 resolver 返回错误。DNS outbound 的 `nonIPQuery/blockTypes` 变为显式规则，已有规则优先；数字 `qType: 0` 转为字符串以免被解释为默认匹配。
- WireGuard 删除已移除的 `workers`；旧 outbound `domainStrategy` 迁到当前策略位置，`remoteDNS: ["local"]` 转为本地解析策略，其余 remote DNS 必须是 IP。freedom `targetStrategy/domainStrategy` 迁入 `streamSettings.sockopt.domainStrategy`，按核心优先级保留有效旧值；`proxySettings.tag` 迁为 `dialerProxy`。
- Hysteria 缺少 version 时补 `2`，明确拒绝其他版本；旧 VMess 客户端 `none/zero` security 转为 `auto`。Shadowsocks 只拒绝已移除的 `none/plain`，AES-GCM / ChaCha20 / XChaCha20 保持；SS-2022 runtime Account 使用对应类型而非普通 SS Account。
- Salamander 密码至少 4 个 UTF-8 字节；XMC 要求目标核心所需完整 profile，Clash 输出过滤其不支持的 XMC。TLS/Reality 密钥、证书、传输协议组合的最终合法性由生成检查和目标核心共同验证。缺少材料不会由迁移自动生成并替换用户原身份。
- TLS `pinnedPeerCertSha256` 旧字符串数组转为逗号分隔字符串，保持证书 pin 的顺序和内容；目标核心已移除的 `allowInsecure: true` 明确拒绝，要求管理员配置有效证书、受信任 CA 或证书 pin，不能以静默删除字段改变原有验证语义。
- 旧入站 tag 保持稳定，本轮不执行 `0.0.0.0` 相关 tag 更名，避免破坏路由/统计引用。兼容转换在副本上进行，失败返回原始 JSON，不提交一部分字段；数据库迁移、入站/模板更新及 runtime 生成的原子性由对应事务和候选预检保证。

### 候选配置、数据库导入与并发

Xray restart 在停止旧核心前生成**完整候选配置**并执行 `run -test -config <private-file.json>`。临时文件保留 `.json` 扩展以便目标核心识别格式。核心升级仅接受 `v26.10.10` 及以上稳定版本；先核对归档 `.dgst`，再使用 staged binary 及 staged assets 检查当前完整配置；成功后才替换，启动失败尝试恢复旧文件和旧 runtime。`run -test` 不实际绑定监听端口，端口占用等运行错误仍需启动检查和回滚路径覆盖。

ImportDB 使用独立候选文件及连接，完成 schema/历史迁移、订阅/入站所需迁移和完整 Xray 配置生成后调用真实核心预检。此时原库仍是活动库；迁移/预检失败不切换。替换前 checkpoint、关闭原连接并保留备份；打开候选或恢复 runtime 失败时恢复原数据库，若恢复本身失败则返回包含恢复路径的明确错误，不宣称恢复必然成功。启动、CLI 与导入共用事务 hook，将 schema、seed、兼容转换、历史迁移和孤儿清理放在同一事务；故障触发器回归验证失败不部分提交、原活动连接继续可用。旧 MultiDomain 迁移保留现有 externalProxy 和未知数字，冲突拒绝而不覆盖。

`database.AcquireReadLease` 是进程内共享读租约；ImportDB 排空受控访问者后取得独占维护租约，锁序为 database lease → `runtimeMaintenanceMu`。HTTP middleware、cron job wrapper 和 Telegram 四个消息/回调工作单元获取读租约。它不是跨进程数据库锁；检测到仍在使用的 WAL/SHM/journal 时拒绝替换，需停止外部数据库写入者。四条入站/客户端写路径另行串行化读改写及提交后的核心同步，防止并发追加丢更新。

### 权限与可信代理契约

`POST /panel/setting/update` 保持原设置 payload/envelope，新增仅供请求验证的 `twoFactorCode`，不持久化该验证码。首次启用 TOTP 验证新认证器；禁用或替换已启用 TOTP 验证旧认证器。空 token 表示保留当前 token，禁用成功后清除存储 token；校验失败整次设置更新失败。登录、CSRF 与旧密码修改校验继续生效。

`SUPERXRAY_TRUSTED_PROXY_CIDRS` 为逗号分隔 CIDR；未设置时信任 `127.0.0.0/8,::1/128`，显式空值不信任任何代理，非法 CIDR 使服务器配置失败。只信任直接 TCP peer 命中的 forwarded headers，X-Forwarded-For 从右向左寻找首个不可信地址。策略用于面板/订阅的 scheme、host、限流和审计身份。反向代理在容器或其他主机时，应配置它实际来源的最小 CIDR 并重启；未配置可能生成内部 URL 或无法识别外部 HTTPS。`X-Real-IP` 仅表示客户端地址，不能用于订阅服务器主机名。

## 6. 验证记录

验证环境：Windows / PowerShell，Go 1.27.2（CGO / MinGW）、Node 22.23.2 / npm 10.9.8，以及 Docker Desktop Linux amd64 / QEMU arm64。真实核心为官方 v26.10.10，Windows amd64 与容器 Linux 归档 SHA-256 均与官方 `.dgst` 一致。没有推送、发布或修改真实业务服务。

| 验证范围 | 命令 / 入口 | 本轮最终结果 |
| --- | --- | --- |
| Go 单元/集成 | `go test ./...` | 通过；也覆盖 node_modules 中无测试的 flatted Go 包。单独一轮 CustomGeo URL 测试因 example.com DNS 超时失败，随后全量运行通过 |
| Go 静态检查 | `gofmt -l .`、`go vet ./...`、`tmp/audit-tools/staticcheck.exe ./...` | 通过；staticcheck 固定 v0.8.2，Go 1.27.2 构建 |
| 竞态 / 维护边界 | `go test -race -shuffle=on ./...` | 最终全量通过；含迁移、并发追加、Telegram 权限、订阅和本地定制回归 |
| Linux Docker 全量测试 | `golang:1.27.2-bookworm`，`CGO_ENABLED=1`，设置 `SUPERXRAY_TEST_XRAY_BINARY` 后运行 `go test -race -shuffle=on ./... && go vet ./...` | 全部通过，exit 0；包含真实 Linux Xray 26.10.10 的配置、启动与失败恢复测试 |
| 后端构建 | `go build -o bin/SuperXray.exe ./main.go` | 通过，`-v` 为 `3.4.3` |
| Linux 生产编译 | 在 `golang:1.27.2-bookworm` 中以 `CGO_ENABLED=1` 执行 `go build -trimpath -ldflags '-w -s' -o /out/x-ui-linux-amd64 main.go`；Go 1.27.2 / Zig 0.16.0 以 `CGO_ENABLED=1 GOOS=linux GOARCH=arm64` 编译 | 两架构均通过；arm64 为静态 musl ELF64 AArch64，并在 Alpine / QEMU 中执行 `-v` 返回 `3.4.3` |
| Docker / 发布脚本 | `docker buildx build --network host --platform linux/amd64 --load -t superxray:upstream-v3.10.2-amd64 .`；arm64 增加 `--build-arg GOPROXY=https://goproxy.cn,direct` 并替换平台/标签；shell 语法与 `actionlint` | 两架构完整镜像均构建成功，包含 GCC / CGO 编译和 Xray 归档 SHA-256 校验；版本、登录页、live/ready、真实配置 `run -test` 均通过；arm64 运行验证使用 QEMU |
| 数据库迁移/失败恢复 | `database/*migration*test.go`、`maintenance_test.go`、`web/service/database_*test.go` | 通过：重复迁移、故障回滚、候选连接隔离、外部代理无损合并、备份恢复失败保留路径 |
| 真实核心矩阵 / 回滚 | `$env:SUPERXRAY_TEST_XRAY_BINARY='<verified xray.exe>'; go test ./web/service -run '^TestRealXray' -count=1 -v` | 通过：6 类 TCP/WS-TLS/gRPC-TLS/旧 XHTTP-TLS/Reality/Hysteria-UDP Hop 配置，另验证 schema 失败保旧进程、占用端口恢复原配置和监听；未设置变量时这些测试 skip |
| 前端依赖 / 类型 / lint / 测试 | `npm ci`、`npm run typecheck`、`npm run lint`、`npm run test` | 通过；最终 144/144 测试，lint 0 errors，保留已有 Vue 样式 warnings |
| 格式 / 生产构建 | `npm run format`、`npm run build`、检查 `web/ui` 生成物 | 前端全目录 Prettier 与生产构建通过，已更新受版本管理的 `web/ui`；按后续修复要求补齐此前 11 个文件的既有格式问题 |
| 浏览器 E2E | 根目录 `npm run e2e`，开启 `SUPERXRAY_E2E_RESTART=1`、`SUPERXRAY_E2E_MUTATION=1`、`SUPERXRAY_E2E_IMPORT_DB=1`，配置隔离测试订阅 | 19/19 通过，0 failed / 0 skipped，耗时 1.7 分钟；使用 Docker 全新 tmpfs 数据库、仅 loopback 随机端口和真实 Xray 26.10.10，覆盖登录、协议编辑、权限/CSRF、备份、重启、DB 回灌及订阅 |
| 项目及发布元数据 | `actionlint`、`release_gate.py --ci --metadata-only`、`scripts/secret_scan.py`、项目配置 9 项测试与校验、3 个技能格式校验 | 均通过；metadata-only 不等于完整发布门禁 |
| 依赖安全 | 前端 `npm audit --omit=dev` / `npm audit`；`govulncheck -show verbose ./...` | 两种 npm audit 均 0；Go 可达/已导入包漏洞 0，模块级 GO-2026-5932 为未使用的 openpgp 包，官方无修复版 |
| 本文机器清单 | 固定 refs 重算并与 TSV 行数/字段比较；`git diff --check` | 清单行数/字段复核通过，diff 与 UTF-8 检查通过 |

gosec v2.29.0 自带的 x/tools v0.49.0 无法解析本次工具链的类型信息。仅在 ignored `tmp/audit-tools/gosec-build` 中固定 x/tools v0.51.0 重建扫描器，未修改项目依赖或全局工具；`gosec-go127.exe -quiet ./...` 通过。动态路径/执行告警逐项审核后只在固定发布资产、私有临时目录及管理员配置路径处写明 `#nosec` 理由。

集成验证时的产物（ignored，不提交二进制；以下哈希属于该次构建）：

- `bin/SuperXray.exe`：SHA-256 `7801d8dd1bcdaf07062662cef790e17bdbde2b4c0d2db7a0f7521131cb21b298`。
- `tmp/linux-build/x-ui-linux-amd64`：SHA-256 `80dee6dea71d462bb9b7cbc91cf74718f8889fdf1c1d91eed39254ad048bd53d`。
- `tmp/SuperXray-gui-arm64`：SHA-256 `337c7d9ecd2e848ba628551e81c7c97f78b0843d62626e6b522015dbd0fe6029`；64,275,864 bytes，静态 musl AArch64，Docker / QEMU 运行版本检查通过。
- `tmp/linux-build/x-ui-linux-arm64`：从最终 arm64 Docker 镜像提取，SHA-256 `da7cc1a6143eda93a153dd0a8d0b56225c03b3415dd1fd9e38b167ffbf58bf9d`，由 Alpine GCC / CGO 构建。

2026-10-15 仓库清理将这些本地构建产物和验证工具移至仓库外的 `F:/SuperXray-gui-cleanup-20261015-v3102/`，保留相对路径及搬迁清单。上述路径用于描述历史构建，不表示文件仍在工作区；清理后的布局、重建与恢复办法见 [本轮仓库审计](repository-audit-2026-10-15.md)。本机 Docker 镜像未因文件整理而移除。

最终本地镜像（未推送）：

- `superxray:upstream-v3.10.2-amd64`：`linux/amd64`，镜像 digest `sha256:7c10f9bea78cdb5f09affad2524df74f86b5d4bb4e080ef6fe8d3db72b49a75d`。
- `superxray:upstream-v3.10.2-arm64`：`linux/arm64`，镜像 digest `sha256:e75275a708822dbe334dbca1672ff6bbf5a16dd1aa37a902afa5756cdf49b43f`。

两个镜像中的面板 `-v` 均为 `3.4.3`，核心均为 `26.10.10` 且架构匹配。使用 `docker run --rm --platform linux/<arch> --entrypoint /app/x-ui <image> -v` 检查版本；现有默认入口不转发命令行参数。两个架构的隔离面板均返回登录页及 live/ready HTTP 200，ready 显示 Xray running，生成配置通过核心 `run -test`；本轮所有测试容器均已停止并清理。

集成验证时 Linux 编译实际命令如下；清理后复现需先重新创建 `tmp/linux-build` 输出目录。Windows Go 下载缓存是 junction，首次直接挂载 `pkg/mod` 导致 `cache/download: file exists`；追加其真实目录挂载后成功。WSL 原生工具链下载另遇到 `proxy.golang.org` TLS handshake timeout，没有降低 TLS 校验。

```powershell
$downloadCache = (Get-Item -LiteralPath C:/Users/www/go/pkg/mod/cache/download).Target
docker run --rm --mount type=bind,source=F:/SuperXray-gui,target=/src,readonly --mount type=bind,source=F:/SuperXray-gui/tmp/linux-build,target=/out --mount type=bind,source=C:/Users/www/go/pkg/mod,target=/go/pkg/mod --mount "type=bind,source=$downloadCache,target=/go/pkg/mod/cache/download" --workdir /src --env CGO_ENABLED=1 golang:1.27.2-bookworm go build -trimpath -ldflags '-w -s' -o /out/x-ui-linux-amd64 main.go
docker run --rm --mount type=bind,source=F:/SuperXray-gui/tmp/linux-build,target=/out,readonly --entrypoint /out/x-ui-linux-amd64 golang:1.27.2-bookworm -v
```

已修复前端全目录 Prettier 的 11 个既有失败文件：`src/api/cores.ts`、`src/stores/core.ts`、`src/utils/gatewayEgressMvp.ts`、`src/utils/xrayCompat.ts`、`src/utils/xrayProtocolTools.ts`、`src/views/ApiDocsView.vue`、`tests/form-styles.test.ts`、`tests/gateway-egress-mvp.test.ts`、`tests/openapi-docs.test.ts`、`tests/xray-compat.test.ts`、`tests/xray-protocol-tools.test.ts`（均相对 `frontend/`）。独立审查逐字对比各文件与 `Prettier.format(HEAD 内容, 当前项目配置)`，11/11 一致，确认仅格式变化。随后前端全目录 format、typecheck、lint（0 errors）和 144/144 单测再次通过。

Docker 下载修复保留官方默认 `GOPROXY=https://proxy.golang.org,direct`，允许通过构建参数显式选择代理；`go.mod/go.sum` 单独下载，模块缓存加锁，编译缓存按架构隔离，编译使用 `-mod=readonly`。核心和 Geo 下载使用有限重试、20 秒连接超时及 300 秒单次超时，保留 HTTPS 和 Xray 解压前 SHA-256 校验。未修改 CI 发布行为，具体构建与排障命令见 [部署文档](deployment.md)。

本机默认 BuildKit 网络曾阻塞 Alpine 下载；`--network host` 后恢复。arm64 的 Go 模块下载还遇到 `storage.googleapis.com` TLS 超时，显式代理参数解决。一次 Geo 下载因 Docker DNS 返回空记录而失败退出，确认 DNS 恢复后重跑成功；缓存使重跑 Go 编译降为约 14 秒，后续瞬时连接超时由有限重试恢复。没有关闭 TLS 或依赖完整性校验，也未把临时 IP 固定到代码。上述命令验证了本机可用构建路径，不代表任何网络环境均无需配置代理。

E2E 补装了 Playwright 所需 Chromium，并修正 `tests/e2e/legacy-panel.spec.ts` 的 4 个旧登录定位器和 1 个设置保存响应监听路径；独立审查确认未新增 skip、吞错或削弱断言。回灌后原测试凭据仍可登录，备份 SQLite `integrity_check=ok`、`foreign_key_check` 为 0 条，suite 创建记录全部清理。核心处于 running，容器内生成配置再次 `run -test` 成功。测试容器及其 tmpfs 数据库/日志已销毁；浏览器报告原位于 ignored `playwright-report/index.html`，现已随仓库清理移至上述隔离目录，不提交含测试数据的产物。

复现完整 E2E 需为隔离面板设置 `SUPERXRAY_E2E_BASE_URL`、`SUPERXRAY_E2E_USERNAME`、`SUPERXRAY_E2E_PASSWORD`、`SUPERXRAY_E2E_SUB_URL`，并打开表中的三个写入/重启/回灌开关后运行 `npm run e2e`。这些值只通过当前进程环境传入，不写入文档或受版本管理的文件。

最终独立代码、Go 与安全复核在本次改动范围内未发现剩余 CRITICAL/HIGH。验证结论限于本机隔离数据和上述环境，不替代真实生产数据、原生 arm64 主机及线上 CI 发布验收。

## 7. 兼容性限制与人工处理

- 不支持直接导入上游 PostgreSQL、normalized clients/hosts 或其专用 dump；请使用本项目兼容的 SQLite 备份，导入前保留原 DB 和匹配旧版本二进制。
- 已移除 cipher、冲突 UDP Hop、非法 xDNS / WireGuard DNS、Reality + TCP finalmask、缺失 XMC profile 或 TLS/Reality 材料需要管理员明确修正；不会为“迁移成功”静默改变访问凭据或丢弃冲突配置。
- 新核心可能拒绝本面板不认识但以前被忽略的 JSON 字段；未知字段保留不代表目标 schema 接受。必须对真实生产模板和数据做完整预检，并在目标 Linux/容器环境验证 geo 文件、文件权限、网卡和监听端口。
- gRPC 固定为目标 Xray 依赖要求的 `v1.86.0-dev`，`go.sum` 固定校验和；这是明确版本而非 `latest`，但仍是开发版本。升级到稳定版本需重新验证 protobuf/HandlerService API 和 runtime 用户增删，不在本轮无依据替换。
- `.dgst` 缺失/格式不支持的核心归档会失败关闭；离线或定制核心需人工校验资产并遵循受控安装流程。候选校验依赖可执行的目标核心及其 assets，核心缺失或版本不匹配会阻止导入/启动。
- DB lease 只覆盖协作入口，SQLite 不应同时由其他进程写入。若错误提示恢复失败，应停止服务，检查返回的备份路径，再恢复原库/核心并重新预检；不要在仍有连接时移动数据库。
- 模板入站的 `port` 仍沿用本项目整数模型；字符串端口范围会返回反序列化错误，未借此变更扩大数据库/API 契约。未知入站扩展字段现可保留并参与配置变更检测。
- 进程启动观察窗口为 500ms，不代表外部网络、证书续期、真实代理握手和长期健康均已验证。Clash 不支持的 XMC 节点和目标 Xray 已删除 `allowInsecure=true` 的 JSON 节点不输出；URI/Mihomo 仍按各自能力处理，管理员应改为可信证书或 pin。
- 本轮 E2E 在独立 Docker tmpfs 中运行并销毁，无真实业务数据挂载；历史 ignored `tmp/` 验证产物及浏览器报告不纳入 Git。
- 本次没有建立上游全部 973 条提交逐条完成审查的声明。未单独映射的重要修复、新协议、节点与 PostgreSQL 功能仍需后续在本地边界内独立评估。
