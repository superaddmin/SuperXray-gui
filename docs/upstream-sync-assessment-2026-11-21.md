# 上游同步决策 · 2026-11-21

**执行结果：Xray 已通过独立验证，Linux 安装包与 Docker 镜像已锁定 `v26.11.9`。** 路由标签候选和首批五项修复已落入工作区。两个面板上游的最新正式版仍是上次评估的 v3.11.0 和 v1.15.0，本次候选来自其后 main 分支。[既有同步策略](upstream-sync-policy.md)

评估基线为 v3.7.4（`dc24d7448`）及当时未提交的源码。筛选了 3x-ui `c621ba901..5297fd3c8` 的 179 条和 alireza0/x-ui `807e7e243..b59efb3f8` 的 1 条提交标题。文末记录实施结果和隔离环境验收数据。[3x-ui 比较](https://github.com/MHSanaei/3x-ui/compare/c621ba901...5297fd3c8) · [alireza0/x-ui 比较](https://github.com/alireza0/x-ui/compare/807e7e243...b59efb3f8)

## 优先移植与验收

| 顺序 | 本地差异与改法 | 验收条件 |
| --- | --- | --- |
| 1. 登录 | 登录路由直接进入密码检查，可适配 `8e8541a5d`、`2e14a2a51` 的按来源限制、校验中计数和用户名长度限制。`7367ff75c` 适用于登录直接调用 TOTP 库的路径，可与设置页共用密钥规范化和校验。[路由](../web/controller/index.go#L45) · [登录](../web/controller/index.go#L59) · [TOTP](../web/service/user.go#L96) · [设置](../web/service/setting.go#L885) | 正常账号、已存展示格式密钥可登录，限额在并行校验时生效，错误配置返回认证失败。 |
| 2. 入站配额 | 批量重置只清计数，定时任务按 `quota_disabled` 恢复。Vue 单条重置回写整条入站，常规更新会清掉该标记。按 `eae4725d9` 统一三个入口的语义。[批量](../database/repository.go#L630) · [定时重置](../database/repository.go#L637) · [Vue 单条](../frontend/src/views/InboundsView.vue#L2883) · [标记写入](../web/service/inbound.go#L586) | 单条、批量、定时都只恢复配额停用且未过期的入站。手工停用和旧库无标记记录保持停用，重复重置结果一致。 |
| 3. 升级路径 | 安装和更新脚本按长度重置面板路径。移植 `09799ff73`，仅初始化空值或根路径。[安装](../install.sh#L670) · [更新](../update.sh#L646) | 重装、升级均保留非根短路径，设置读取失败时不写回。 |
| 4. VLESS 保存 | 将 `19f2f21d8` 纳入新增、编辑和导入验证，服务端启用解密时归一化空 fallbacks，保存前拒绝非空冲突。[入站协议校验](../web/service/protocol_validation.go#L14) | 合法旧配置可加载，冲突写入返回错误，数据库与运行核心继续使用原配置。 |

**启用 LDAP AutoDelete 的部署应把目录保护提至首批。** 本地只拦空结果，非空但显著缩小的目录仍可触发删除。采用 `ce44d2e77` 之前须先建立可信快照和收缩保护。用连续两轮部分结果验证均保留原基线及客户端，再用完整结果验证同步恢复。[自动删除入口](../web/job/ldap_sync_job.go#L169) · [上游提交](https://github.com/MHSanaei/3x-ui/commit/ce44d2e77)

## 后续适配

- **订阅与客户端。** VLESS ID 在直连与外部代理 URI 均直接拼接，按 `255c65846` 验证订阅和二维码解析为同一 ID。`2467b9d54` 的生成预算应以历史大订阅规模验收，覆盖 URI、JSON、Clash。部分更新 `e4935be86` 需先明确 API 契约，分别测试字段缺省、显式清空和入站专属 flow。[URI 两条路径](../sub/subService.go#L405) · [订阅组装](../sub/subController.go#L120) · [客户端更新](../web/service/inbound.go#L1219)
- **续期与日志。** 按 `351a01ff0` 同时限制新写入和旧库续期间隔，覆盖边界与过期追赶。`776ca554b`、`54339661a` 适用于访问日志读取，验收只保留最近请求的匹配行，单条超长记录之后仍可读取。[续期](../web/service/inbound.go#L1612) · [访问日志](../web/service/server.go#L856)
- **Xray 核心单独升级。** v26.11.9 已正式发布，本地分发 v26.10.21，Go API 类型固定于 v26.10.10 对应提交。先验证旧模板的 `loglevel` / `logLevel` 和 Freedom `finalRules` 中 `blockDelay` / `delay` 的转换与动作约束。在 Linux amd64、arm64 上以新旧核心运行配置预检、启动和流量统计，验证失败后还原核心、资源与匹配的配置，再决定是否调整默认版本。[正式发行](https://github.com/XTLS/Xray-core/releases/tag/v26.11.9) · [规则变更](https://github.com/MHSanaei/3x-ui/commit/a748f799c) · [分发](../.github/workflows/release.yml#L161) · [类型依赖](../go.mod#L24)

## 已覆盖与暂缓

当前未提交工作树已按协议、主机和端口校验 WebSocket Origin，覆盖 `b1108a9e8`。本地核心更新已在停机前下载、校验并预检候选核心，覆盖 `6d9e836d6`。保留对应回归即可。[WebSocket](../web/controller/websocket.go#L35) · [下载](../web/service/server.go#L690) · [核心预检与替换](../web/service/server.go#L783)

alireza0 的 DNS 和模板入站标签提示已按本地 Vue 编辑器移植。节点联邦、PostgreSQL、React 界面和独立于 Xray 的协议运行时暂缓，均需单独的架构决策。[标签输入](../frontend/src/views/XrayView.vue) · [上游提交](https://github.com/alireza0/x-ui/commit/b59efb3f8)

## 实施与验收

| 范围 | 已验证行为 |
| --- | --- |
| 路由编辑器 | 候选取自已加载的面板入站、当前未保存模板的入站、全局 DNS 和 DNS 服务器。手填与逗号粘贴均可用。编辑会保留历史标签、原有标量和未知字段。浏览器用合成模板验证编辑结果。 |
| 登录 | 五分钟窗口内，同一可信来源与用户名最多占用 5 个校验名额，同一来源合计 25 个。成功释放对应名额，失败到期释放。设置与登录均支持已有的分组、小写及补位 Base32 密钥，损坏的密钥返回校验失败。 |
| 入站配额 | 单条使用 `POST /panel/api/inbounds/resetTraffic/{id}`。单条、批量均在 SQLite 更新条件中校验用户归属，与定时任务共用计数清零及恢复条件。重复重置、过期、手工停用、旧库和其他用户入站均有回归。 |
| 升级与重装 | 读取 CLI 设置快照，保留 `/a/` 等非根路径。显式根路径或空值允许初始化，缺少数据库记录时沿用项目的默认路径 `/super/`。字段不完整、读取失败会中止。 |
| VLESS | 新增、编辑、导入都移除空回落数组；非空回落要求显式设置 `"decryption":"none"`，并校验目标、路径与 PROXY 协议版本。SQLite 测试覆盖失败后原记录保留。 |
| LDAP | 初次启用或重启时参考当前受管入站的账号，首份可信目录只建立基线。后续结果须包含上次可信账号的至少一半，奇数向上取整。空结果、持续缩水和等量换组均保留基线。使用临时 SQLite 连续验证 8 轮，完整目录恢复后仅删除确已离开的账号。 |

新设用户名上限为 256 字节。历史长用户名在 4096 字节的输入上限内仍可验证，实际登录还要能写入签名会话 cookie。测试覆盖 1500 字节旧账号的登录和会话回读。超过会话容量的已有账号可通过本机 `x-ui setting` 重设为短名。CLI 对凭据、端口和路径使用同一 SQLite 事务，强制路径写入失败时仍可使用原凭据和端口。

### Xray 核心与回退

官方发行 ZIP 的 SHA-256 同时与 GitHub 发行资产元数据及同版本的 `.dgst` 校验。Linux amd64 使用 WSL 2，Linux arm64 使用 QEMU 8.2.2。[官方发行](https://github.com/XTLS/Xray-core/releases/tag/v26.11.9)

| 制品 | v26.11.9 ZIP SHA-256 |
| --- | --- |
| Linux amd64 | `b651f7e1d9966ee2c1b1ee1bf8b52c72771824d3065dce96bbc8bcc19ed95f82` |
| Linux arm64 | `64ed6858982a7f5eb85d88b811089063bd4f17561f4c8f44e0f1dc8e6f6430be` |
| Windows amd64 | `1262522266337b824f07beb8e1375786b437683b3890105469088f8e17e0a69d` |

- 两种 Linux 架构共完成 20 次配置预检。18 次预期有效配置通过，2 次旧式 `allow.blockDelay` 在候选核心上按预期失败。两种架构各按旧版 → 候选 → 旧版执行，六次启动均能转发环回 HTTP 并经 gRPC 读取流量。
- 面板只在生成运行配置时移除旧核心在 `allow` 上忽略的 `blockDelay`。在 `block` 上采用两核都执行的 `blockDelay`，新核心可能输出该拼写的弃用提示。旧核心使用其默认规则日志行为。旧配置留在数据库中，可配合上版二进制、同版资源文件及配置快照回退。[运行配置转换](../util/xraycompat/freedom.go)
- Windows amd64 的候选核心通过项目现有的 Go gRPC 客户端读取上行 119 字节、下行 132 字节，清零后再次读取双向均为零。该接口兼容性实测通过，保留 `go.mod` 的现有类型依赖。[项目客户端](../xray)

### 项目回归

| 检查 | 结果 |
| --- | --- |
| Go | 按 CONTRIBUTING.md 排除 node_modules 示例后，32 个项目包的普通测试、竞态测试、`go vet` 和 Staticcheck 通过。 |
| Vue | 164 项单元测试通过。ESLint 零警告，Prettier、类型检查和生产构建通过。 |
| 浏览器 | 临时数据库、独立端口、官方 Windows 新核心下，24 项 Chromium 测试全部通过，涵盖单条重置、登录、核心重启和数据库导入。 |
| 运维与发行 | WSL 脚本回归独立重跑通过。一次并行运行遇到临时日志文件缺失。ShellCheck 0.9.0 对 8 份脚本按 style 级检查零告警。actionlint、发行元数据检查、凭据扫描与差异检查通过。 |

变更落在项目工作区，发行和生产部署另行进行。此工作区由 Git 忽略的 Windows 开发核心为 `v26.10.10`，其现有 `bin/config.json` 也已通过官方 `v26.11.9` 的只读预检。
