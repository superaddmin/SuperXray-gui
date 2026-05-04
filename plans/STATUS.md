# SuperXray-gui 技术计划与项目进度状态追踪报告

报告日期：2026-05-04
报告口径：以 `plans/` 文档、当前代码实现、测试结果和 UI-first 阶段门禁为准。
当前主线：UI 先行、Xray 稳定迁移、再接多内核。

## 1. 执行摘要

项目当前处于 **UI-first Phase 9 安全收口与 Phase 10 风险接受并行推进段**。Phase 0 到 Phase 5 已完成，Phase 6 已完成子阶段 **6a：VMess/VLESS 入站与客户端管理**、**6b：Trojan/Shadowsocks 入站与客户端管理**、**6c/6d：Hysteria2/WireGuard 与 StreamSettings 通用表单** 和 **6e：批量操作/分享导出/mutation E2E 基线**。Phase 7a 已新增 Settings、Subscription、Backup/Restore 新 UI 入口，并已在隔离面板验证设置保存与 SQLite 备份下载。Phase 8 已把新 UI 切到 `/panel/` 默认入口，并保留 `/panel/legacy/` 旧 UI 回退和 `/panel/ui/` 兼容入口。Phase 9 本轮已收口旧 UI HTML sink、session 级 CSRF token、未登录数据库/配置/日志读取、非法数据库导入、新 UI `style-src` nonce 收紧、新 UI 到 Legacy UI 的六类协议创建兼容矩阵、Settings 兼容抽检、VLESS/Trojan/Shadowsocks/Hysteria2 客户端与 WireGuard peer 编辑兼容抽检、在线/IP 管理入口、订阅输出矩阵和相关 E2E；本机隔离真实 Xray core 环境已完成 19 passed / 0 skipped / 0 failed 的完整 Playwright 验收。

多内核后端架构方向已经形成完整方案。Phase 10.1-10.5 已完成准入评估，Phase 10.1 `default-xray` 虚拟只读实例 ADR 已落地；当前已完成六类协议创建、订阅设置保存、VLESS/Trojan/Shadowsocks/Hysteria2 客户端编辑、WireGuard peer 编辑和在线/IP 管理的新旧 UI 自动兼容抽检，订阅输出矩阵已补 Go 层回归；仍缺第二环境/CI 复刻。2026-05-04 因上线部署需要，项目已记录 **风险接受/强制进入**，允许先实施最小 active CoreManager/sing-box 后端入口；仍禁止把旧 `model.Inbound` 迁移到 `proxy_inbounds/proxy_clients`，禁止 CoreManager 接管现有 Xray 生命周期，禁止修改旧订阅输出语义。

2026-05-04 已按方案 A 完成第一轮 UI 框架迁移收口，并继续补齐旧 UI 退场前门禁：新 Vue 3/Vite UI 的未使用文件、未使用导出和未使用依赖已清零，CodeMirror 依赖已移除；新 UI 已补登录页、Custom Geo/Geofile、Xray 出站工具、入站导入/批量工具、在线/IP 管理、2FA 设置和订阅公开链接入口；`/panel/legacy`、`web/html` 和 `web/assets` 继续作为受控回退与兼容验收边界保留。新二进制本地 E2E 19 passed，桌面/移动截图式响应式检查通过。

协议能力补齐计划已基本完成：主力协议校验、WireGuard 订阅导出、协议能力矩阵和相关测试均已在代码侧出现，并通过当前验证命令。

## 2. 总体里程碑状态

| 里程碑                        | 状态     | 完成度 | 当前说明                                                                                                                        |
| ----------------------------- | -------- | -----: | ------------------------------------------------------------------------------------------------------------------------------- |
| M0：旧 UI 行为冻结与 E2E 基线 | 已完成   |   100% | 已形成 Xray parity checklist 和 Playwright 基线；占位 `.env` 会自动跳过 E2E                                                     |
| M1：Vue 3/Vite 新 UI 工程化   | 已完成   |   100% | `frontend/` 工程、路由、布局、Pinia、Ant Design Vue、构建链路已完成                                                             |
| M2：Go 托管新 UI              | 已完成   |   100% | `/panel/ui/` 嵌入式 SPA 入口、运行时配置注入、严格 CSP 已完成                                                                   |
| M3：旧 API SDK 与类型层       | 已完成   |   100% | 集中 endpoints/request/types，组件不再散落硬编码旧 API URL                                                                      |
| M4：只读 Dashboard/日志/配置  | 已完成   |   100% | 日志和配置均按文本渲染，无 `v-html`                                                                                             |
| M5：Xray 生命周期与配置管理   | 已完成   |   100% | Start/Restart/Stop、版本安装、模板编辑保存走旧兼容端点                                                                          |
| M6：入站与客户端迁移          | 阶段完成 |    99% | 6a-6e 功能收尾已完成；本机隔离真实 core mutation E2E、六类协议创建兼容抽检和主力协议编辑兼容抽检已通过                          |
| M7：设置、订阅、备份恢复迁移  | 阶段完成 |    94% | Phase 7a 设置保存、SQLite 备份下载、DB 回灌导入、2FA 设置替代和订阅公开链接入口已在本机隔离真实 Xray core 环境通过              |
| M8：新 UI 默认入口灰度        | 阶段完成 |    82% | `/panel/` 新 UI 默认入口、`/panel/login` 新登录页、`/panel/legacy/` 回退、`/panel/ui/` 兼容入口已通过本地 E2E 和浏览器冒烟      |
| M9：安全收口                  | 推进中   |    99% | 无 HTML sink、session 级 CSRF token、未登录下载/读取、非法 DB 导入、style-src nonce、协议/设置兼容抽检和完整非跳过 E2E 已通过   |
| M10：多内核后端启动           | 强制进入 |    28% | 已记录风险接受/强制进入，并落地最小 `/panel/api/cores/*`、CoreManager 和 experimental sing-box 后端入口；旧 UI 退场前补门禁推进 |

## 3. 文档体系审查结论

### 3.1 当前文档分类结果

| 分类     | 文档                                                      | 核心价值                                              | 审查状态                         |
| -------- | --------------------------------------------------------- | ----------------------------------------------------- | -------------------------------- |
| 治理     | `00-governance/documentation-system-plan.md`              | 定义 `docs/` 长期文档体系和写作规则                   | 结构清晰，已迁入治理类           |
| 战略     | `01-strategy/ui-first-xray-stable-multi-core-roadmap.md`  | 裁定 UI-first 施工顺序、阶段门禁、风险矩阵            | 当前项目主计划，需每阶段结束更新 |
| 架构     | `02-architecture/backend-multi-core-architecture-plan.md` | 多内核后端抽象、CoreManager、Capability、数据模型设计 | 方向可行，但必须 Phase 10 后实施 |
| 架构     | `02-architecture/protocol-capability-completion-plan.md`  | 协议校验、WireGuard 订阅、协议能力矩阵                | 基本完成，进入维护状态           |
| UI 设计  | `03-ui-design/multi-core-ui-design-plan.md`               | 多内核 UI 信息架构、动态表单、安全 UI                 | 作为 Phase 10+ UI 蓝图保留       |
| 阶段记录 | `04-ui-first-execution/*`                                 | Phase 0-7a 的实际交付、验证、回滚                     | 已系统化归档                     |

### 3.2 重构后的查阅逻辑

管理层只需看：

1. `plans/STATUS.md`
2. `plans/01-strategy/ui-first-xray-stable-multi-core-roadmap.md`

开发实施优先看：

1. `plans/04-ui-first-execution/phase-09-security-closeout.md`
2. `plans/04-ui-first-execution/phase-08-default-entry-gray-switch.md`
3. `plans/04-ui-first-execution/phase-10-entry-gate-assessment.md`
4. `plans/04-ui-first-execution/phase-10a-default-xray-readonly-adr.md`
5. `plans/04-ui-first-execution/phase-07a-settings-subscription-backup.md`
6. `plans/04-ui-first-execution/phase-06e-inbounds-wrap-up.md`
7. `plans/04-ui-first-execution/phase-06c-06d-hysteria2-wireguard-stream.md`
8. `plans/04-ui-first-execution/phase-06b-inbounds-trojan-shadowsocks.md`
9. `plans/04-ui-first-execution/phase-06a-inbounds-vmess-vless.md`
10. `plans/04-ui-first-execution/phase-00-xray-parity-checklist.md`
11. `plans/04-ui-first-execution/phase-00-e2e-baseline.md`

架构审查优先看：

1. `plans/02-architecture/backend-multi-core-architecture-plan.md`
2. `plans/03-ui-design/multi-core-ui-design-plan.md`
3. `plans/02-architecture/protocol-capability-completion-plan.md`

## 4. 各计划核心内容与执行进度

### 4.1 文档体系规划

文档：`00-governance/documentation-system-plan.md`

核心内容：

- 规划 `docs/architecture.md`、`docs/deployment.md`、`docs/modules.md`、`docs/api.md`、`docs/development.md` 和中文 README。
- 定义读者、章节大纲、交叉引用和写作注意事项。

当前进度：

- `docs/architecture.md`、`docs/deployment.md`、`docs/modules.md`、`docs/api.md`、`docs/development.md`、`README.zh_CN.md` 已存在。
- 本次已将计划文档迁入治理目录，并新增 `plans/README.md` 与本状态报告。

风险：

- `docs/` 长期文档可能落后于 UI-first 新前端和新 `/panel/ui/` 路由。
- Phase 6/7 完成后，API 文档和用户指南需要同步新 UI 行为。

下一步行动：

- Phase 6 全量完成后更新 `docs/inbound-creation-guide.md` 和 `docs/api.md`。
- Phase 7 完成后更新部署、环境变量、备份恢复和订阅说明。

### 4.2 UI-first 主路线图

文档：`01-strategy/ui-first-xray-stable-multi-core-roadmap.md`

核心内容：

- 明确先新 UI 工程化，再迁移现有 Xray，最后接多内核。
- Phase 0-9 保持旧 API、旧数据模型和旧 Xray 生命周期。
- Phase 10 才允许 CoreManager/default-xray/sing-box/Capability Schema。

当前进度：

- Phase 0-5 已完成。
- Phase 6a、6b、6c/6d、6e 已完成，Phase 6 总体进入真实环境验收。
- Phase 7a 设置/订阅/备份基础迁移已完成，并通过本地隔离面板保存/备份下载验收。
- Phase 8 默认入口灰度已完成本地验收：新 UI 默认 `/panel/`，旧 UI 回退 `/panel/legacy/`，兼容入口 `/panel/ui/`。
- Phase 9 已完成一轮安全收口：HTML sink、session 级 CSRF token、未登录下载/读取、非法 DB 导入、新 UI `style-src` nonce 收紧、六类协议创建兼容矩阵、Settings 兼容抽检、VLESS/Trojan/Shadowsocks/Hysteria2 客户端和 WireGuard peer 编辑兼容抽检、在线/IP 管理入口和订阅输出矩阵已加入回归；本机隔离真实 Xray core E2E 已 19 passed / 0 skipped / 0 failed。
- Phase 10.1-10.5 已完成门禁评估；因第二环境/CI 复刻未完成，常规门禁仍未完全通过。
- 2026-05-04 已记录风险接受/强制进入：允许先做最小 active CoreManager/sing-box 后端入口，同时保留旧 Xray 生命周期和旧数据模型不变。

已完成里程碑：

- 新 Vue 3/Vite 前端工程。
- Go `/panel/ui/` 静态资源托管和运行时配置注入。
- 旧 API SDK 与类型层。
- 只读 Dashboard、日志中心和配置查看。
- Xray 生命周期、版本管理和配置模板编辑。
- VMess/VLESS/Trojan/Shadowsocks/Hysteria2/WireGuard 入站与客户端或 peer 核心操作。
- TCP、WS、gRPC、HTTPUpgrade、TLS、Reality 常用 StreamSettings 表单。
- 客户端/peer 批量选择、批量重置、批量删除和全量分享链接导出。
- Settings、Subscription、Backup/Restore 新 UI 基础入口。
- 新 UI 默认入口、旧 UI 回退入口和兼容入口。
- Phase 9 CSRF、下载鉴权、非法 DB 导入、旧模板 HTML sink 和新 UI CSP nonce 回归。

推进中任务：

- Phase 6 验收：订阅输出抽检已补 Go 层基础矩阵，等待第二环境复刻。
- Phase 7 验收：第二环境/CI 复刻数据库导入成功路径。
- Phase 9 验收：继续扩展上传/导入边界回归，并在第二环境/CI 复刻完整非跳过 E2E。

风险：

- Phase 6 表单复杂度高，旧 `settings/streamSettings/sniffing` JSON 字段容易出现旧 UI 不可读风险。
- Phase 7 涉及面板凭据、订阅输出和数据库导入，数据库导入仍必须在隔离测试实例完成首次成功路径验收。
- 真实 E2E 需要有效 `.env` 和隔离 Xray core；本机已解决，CI/第二测试环境仍需复刻。
- 过早进入 Phase 10 会叠加 UI 迁移、安全收口和后端架构改造风险。

下一步行动：

- 完成 Phase 9 剩余安全收口：更多上传/导入边界回归。
- 在 CI 或第二测试环境复刻本机 `SUPERXRAY_E2E_*`、Xray core 和隔离 DB/bin/log 目录。
- 风险接受后先实施最小 CoreManager/sing-box 后端入口，并继续补齐第二环境/CI 复刻。

### 4.3 后端多内核架构方案

文档：`02-architecture/backend-multi-core-architecture-plan.md`

核心内容：

- 设计 Core 接口、CoreRegistry、CoreManager、Capability Schema。
- 规划 `core_instances`、`proxy_inbounds`、`proxy_clients`、`core_assets`、`core_events` 等未来表。
- 定义 Neutral Model、ConfigBuilder、进程管理、统一日志、统一订阅、版本下载和安全要求。

当前进度：

- 方案完整，已被 UI-first 路线图裁决为 Phase 10 后实施。
- 因上线部署需要，Phase 10 已记录风险接受/强制进入，并已启动最小 active CoreManager/sing-box 后端入口。
- `proxy_inbounds/proxy_clients` 和旧 `model.Inbound` 迁移仍不准入。

已完成里程碑：

- 架构蓝图、接口模型、数据模型和迁移阶段设计已完成。
- 与 UI-first 主计划完成交叉验证。

推进中任务：

- Phase 10.1 `default-xray` 虚拟只读实例 ADR 已完成。
- 强制进入段已实施 `/panel/api/cores/*`、CoreManager 内存注册表和 `experimental-sing-box` 外部进程适配器。

风险：

- 直接落地 `proxy_inbounds/proxy_clients` 会破坏旧 UI 和旧订阅兼容。
- CoreManager 提前接管 Xray 生命周期会影响当前稳定性和回滚能力。
- Capability Schema 若先于 sing-box MVP 落地，容易过度抽象。

下一步行动：

- 在部署环境验证可回滚的最小 active CoreManager/sing-box 后端入口。
- Phase 10.2 前必须有完整旧 API 行为对照测试。
- 继续补齐 CI/第二环境复刻。

### 4.4 协议能力补齐计划

文档：`02-architecture/protocol-capability-completion-plan.md`

核心内容：

- P0：主力协议正确性和安全校验。
- P1：WireGuard 订阅和导出能力。
- P2：兼容性、安全和性能闭环。
- P3：协议矩阵与文档同步。

当前进度：

- 基本完成，进入维护状态。
- 已发现并验证相关文件：
  - `web/service/protocol_validation.go`
  - `web/service/protocol_validation_test.go`
  - `sub/wireguard_subscription.go`
  - `sub/wireguard_subscription_test.go`
  - `sub/protocol_capability.go`
  - `sub/protocol_capability_test.go`
  - `web/assets/js/model/inbound.test.js`
  - `docs/inbound-creation-guide.md`

验证结果：

- `go test ./sub ./web/service` 通过。
- `node --test web/assets/js/model/inbound.test.js` 通过，21 个测试全部通过。

风险：

- WireGuard/订阅相关能力已接入新 UI，仍需真实环境再次验证。
- 新 UI 当前 Phase 6a/6b/6c/6d 已覆盖 VMess/VLESS/Trojan/Shadowsocks/Hysteria2/WireGuard，尚未消费完整协议能力矩阵。

下一步行动：

- Phase 6 回归和真实 E2E 时，继续复用现有 WireGuard/Hysteria2 测试作为回归基线。
- 将协议能力矩阵纳入新 UI 动态表单或字段可见性设计。

### 4.5 多内核 UI 设计方案

文档：`03-ui-design/multi-core-ui-design-plan.md`

核心内容：

- 定义多内核 Dashboard、实例列表、创建向导、入站管理、用户管理、路由 DNS、配置预览、日志中心、订阅管理、版本管理。
- 推荐 Vue 3 + Vite + TypeScript + Pinia + Ant Design Vue 4。
- 规划动态表单引擎、Capability Schema、CSP、安全 UI 和响应式设计。

当前进度：

- 技术栈部分已经落地：Vue 3/Vite/TypeScript/Pinia/Ant Design Vue 4。
- 多内核页面、动态能力驱动、实例管理和 Capability Schema 尚未实施。

已完成里程碑：

- 新 UI 工程基础与部分单内核 Xray 页面已经按该技术栈推进。

推进中任务：

- 当前仍聚焦 Xray 等价迁移，不进入多内核 UI。

风险：

- 多内核 UI 设计中的动态表单和实例管理如果提前进入 Phase 6，会与旧 API 兼容目标冲突。
- UI 信息架构已完成 Phase 8 本地入口验收，生产灰度仍需保留旧 UI 快速回退。

下一步行动：

- Phase 9 继续吸收其安全设计原则。
- Phase 10 准入通过前不启动多内核 Dashboard 和实例管理页面。

## 5. UI-first 阶段执行明细

| 阶段                              | 状态     | 关键交付                                                                                                                | 当前风险                                                                  | 下一步                                       |
| --------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- | -------------------------------------------- |
| Phase 0：现状冻结/E2E 基线        | 已完成   | 旧入口盘点、Xray 等价验收清单、Playwright 基线                                                                          | 其他测试机仍需配置真实 `.env` 才能避免跳过                                | 复刻本机隔离 E2E 环境                        |
| Phase 1：新前端工程骨架           | 已完成   | `frontend/`、路由、布局、状态管理、CSP 基础设施                                                                         | `go test ./...` 会扫描 `frontend/node_modules` 中的 Go 示例包             | CI 可考虑收敛 Go 测试路径或排除 node_modules |
| Phase 2：Go 静态接入              | 已完成   | `/panel/ui/`、go:embed、runtime config、严格 CSP                                                                        | base path/反代路径仍需真实部署回归                                        | 在反向代理环境跑一次资源加载 E2E             |
| Phase 3：API SDK/类型层           | 已完成   | endpoints/request/types/server/inbounds/xray/settings/subscription                                                      | 类型定义需随旧 API 扩展持续同步                                           | Phase 6/7 继续补齐 SDK                       |
| Phase 4：只读 Dashboard/日志/配置 | 已完成   | Dashboard、Logs、VirtualLogViewer、Xray 配置预览                                                                        | 大日志和过滤体验需真实数据观察                                            | Phase 9 加日志 XSS 回归                      |
| Phase 5：Xray 生命周期/配置       | 已完成   | Start/Restart/Stop、版本安装、配置模板编辑保存                                                                          | 配置保存后真实重启 E2E 需显式开启                                         | 测试环境跑 `SUPERXRAY_E2E_RESTART=1`         |
| Phase 6a：VMess/VLESS 入站        | 已完成   | 入站列表、详情、VMess/VLESS 新增编辑、客户端操作、基础分享                                                              | 分享链接为基础能力                                                        | 已进入 Phase 6b                              |
| Phase 6b：Trojan/Shadowsocks 入站 | 已完成   | Trojan/Shadowsocks 新增编辑、客户端主键兼容、基础分享                                                                   | Shadowsocks 2022 单用户/多用户差异需真实环境回归                          | 已进入 Phase 6c/6d                           |
| Phase 6c/6d：Hysteria2/WireGuard  | 已完成   | Hysteria2 auth 客户端、WireGuard peers、StreamSettings 通用表单                                                         | Hysteria2 证书和 WireGuard 订阅需真实环境回归                             | 进入 Phase 6 收尾                            |
| Phase 6e：入站收尾                | 已完成   | 批量选择、批量重置/删除、分享链接导出、多协议 mutation E2E 基线                                                         | 本机 mutation E2E、六类协议创建兼容抽检和主力协议编辑兼容抽检已通过       | 扩展订阅输出兼容矩阵                         |
| Phase 6 全量                      | 阶段完成 | 完整入站和客户端闭环已落地                                                                                              | 旧 UI/订阅抽检待独立测试环境执行                                          | 保持回归，进入兼容验收                       |
| Phase 7：设置/订阅/备份           | 阶段完成 | Settings、Subscription、Backup/Restore Phase 7a 基础迁移，2FA 设置替代和订阅公开链接入口                                | DB 回灌导入已通过，CI/第二环境仍需复刻                                    | 在第二环境复刻导入/订阅验收                  |
| Phase 8：默认入口灰度             | 阶段完成 | `/panel/` 新 UI、`/panel/legacy/` 回退、`/panel/ui/` 兼容入口                                                           | 用户入口切换已本地通过，生产灰度仍需保留快速回滚                          | 保持 legacy 入口并进入 Phase 9 安全收口      |
| Phase 9：安全收口                 | 推进中   | 无 `v-html/innerHTML`、session 级 CSRF token、未登录下载/读取、非法 DB 导入、新 UI style-src nonce                      | 本机完整非跳过 E2E、协议/设置/编辑/在线 IP/订阅矩阵已通过，第二环境待复刻 | 完成第二环境复刻                             |
| Phase 10：多内核                  | 强制进入 | Phase 10.1-10.5 准入评估、风险接受记录、CoreManager 和 `experimental-sing-box` 后端入口已完成；旧 UI 退场前补门禁已推进 | CI/第二环境复刻未完成，active 入口需保持最小可回滚                        | 部署环境验证 `/panel/api/cores/*`            |

## 6. 技术风险矩阵

| 风险                         | 等级 | 影响                               | 当前缓解                                                                                                      | 后续动作                               |
| ---------------------------- | ---- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| Phase 6 协议表单遗漏         | 中   | 新 UI 创建的数据旧 UI 不可读       | 6a-6e 覆盖六类主力协议和批量操作，保持旧 JSON 字段                                                            | 真实验收阶段按用例补细节               |
| Phase 7 设置/备份写入风险    | 中   | 面板登录、订阅输出或数据库恢复异常 | Phase 7a 仅复用旧 API 和旧字段，设置保存、备份下载和 DB 回灌导入已在本机隔离 core 通过                        | 在第二环境或 CI 复刻导入验收           |
| E2E 真实环境缺失             | 中   | 管理层误判“已全量通过”             | 本机 `.env`、隔离 Xray core 和 19 条 E2E 已通过                                                               | 在 CI/第二测试机复刻                   |
| 旧 API 与新 SDK 类型漂移     | 中   | 页面错误、保存失败                 | endpoints/types 集中维护                                                                                      | 每新增 API 操作补类型和 E2E            |
| CSP/CSRF 收口不完整          | 中   | XSS/CSRF 风险残留                  | session 级 CSRF token、未登录下载/读取、非法 DB 导入、无 HTML sink、新 UI style-src nonce 和非跳过 E2E 已通过 | 继续扩展上传/导入边界回归              |
| 默认入口切换风险             | 中   | 用户无法回到旧 UI                  | `/panel/legacy/` 回退入口已通过本地 E2E 和浏览器冒烟                                                          | 生产灰度保留快速回滚到旧 UI            |
| 多内核提前接入               | 高   | Xray 稳定性下降，回滚困难          | 已记录风险接受/强制进入；本阶段限制为最小 CoreManager/sing-box 后端入口，不迁移旧模型、不接管 Xray 生命周期   | 实施后立即补测试、回滚记录和 CI 复刻   |
| 文档路径变化导致工具引用失效 | 中   | Agent/团队查找旧路径失败           | 本次已更新 `.codex` 相关引用                                                                                  | 后续新增计划必须同步 `plans/README.md` |

## 7. 下一步关键行动点

### P0：Phase 9 安全收口

1. 在 CI 或第二测试机复刻本机隔离 Xray core 安全 E2E。
2. 继续扩展上传/导入大小、文件名和内容校验回归。
3. 保持本机 `SUPERXRAY_E2E_RESTART=1`、`SUPERXRAY_E2E_IMPORT_DB=1`、`SUPERXRAY_E2E_MUTATION=1` 非跳过回归。
4. 已补新 Vue UI 创建六类主力协议禁用入站后 Legacy UI/旧 API 可读的兼容抽检记录，已补新 UI 保存订阅标题/公告后 Legacy Settings 可读记录，并已补 VLESS/Trojan/Shadowsocks/Hysteria2 客户端和 WireGuard peer 编辑后 Legacy UI/旧 API 可读记录；订阅输出矩阵已补 Go 层回归。
5. 保持 `web/html` 和新 UI 无 `v-html/innerHTML/insertAdjacentHTML`。
6. 保持新 UI `script-src`、`style-src` 无 `unsafe-inline/unsafe-eval`，并保留动态 `<style>` nonce 浏览器冒烟。

### P1：真实 Xray core 隔离验收

1. 本机已放入真实 `xray-windows-amd64.exe`、`geoip.dat`、`geosite.dat` 到隔离 `xray-bin`。
2. 本机已设置 `SUPERXRAY_E2E_RESTART=1` 并验证 Xray restart 成功路径。
3. 本机已设置 `SUPERXRAY_E2E_MUTATION=1` 并运行多协议写入基线。
4. 本机已设置 `SUPERXRAY_E2E_IMPORT_DB=1` 并验证数据库备份回灌导入成功路径。
5. 本机已设置 `SUPERXRAY_E2E_SUB_URL` 并完成订阅服务路径抽检。
6. 已在 `/panel/legacy/inbounds` 自动抽检旧 UI 读取新 UI 写入的六类主力协议禁用入站，并在 `/panel/legacy/settings` 自动抽检旧 UI 读取新 UI 保存的订阅标题/公告；已补 VLESS/Trojan/Shadowsocks/Hysteria2 客户端和 WireGuard peer 编辑后旧 UI/旧 API 可读，订阅输出矩阵已补 Go 层回归。

### P2：测试环境与发布准备

1. 配置真实 `.env`，替换 `<webBasePath>`、`<username>`、`<password>` 占位值。
2. 在测试环境运行 `npm run e2e`，至少完成默认低风险用例。
3. 在独立测试环境开启 `SUPERXRAY_E2E_MUTATION=1` 验证写入闭环。
4. 保持 `npm run format/typecheck/lint/build` 绿灯。

### P3：文档同步

1. Phase 6 完成后更新 `docs/inbound-creation-guide.md`。
2. Phase 7 完成后更新 `docs/api.md`、`docs/deployment.md` 和 `README.zh_CN.md`。
3. 每个阶段结束更新本文件的里程碑状态和风险矩阵。

### P4：后端多内核强制进入

1. 已实施最小 active CoreManager/sing-box 后端入口。
2. `default-xray` 只观察旧 Xray 状态，不接管现有 Xray 启停重启。
3. `experimental-sing-box` 仅在存在 binary 和显式配置文件时允许启动；缺失时返回明确错误。
4. 准备旧 API 行为对照测试，确保 CoreManager 包装不会改变旧行为。
5. 第二环境复刻通过前，不迁移旧 `model.Inbound`，不新增 `proxy_inbounds/proxy_clients` 写入路径。

## 8. 本次审查验证记录

已执行：

```powershell
cd frontend
npm run typecheck
npm run lint
npm run format
npm run build
cd ..
go test ./web/middleware ./web/controller ./web
go test ./core/... ./web/service ./web/controller
go test ./sub
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
rg 'v-html|innerHTML|insertAdjacentHTML' web\html frontend\src -n
rg 'v-html' web\html frontend\src web\ui -n
rg 'unsafe-inline|unsafe-eval' frontend\src web\ui -n
rg 'proxy_inbounds|proxy_clients' core web\controller web\middleware web\service database\model frontend\src web\ui -n
npm run e2e
```

结果：

- `npm run typecheck` 通过。
- `npm run lint` 通过。
- `npm run format` 通过。
- `npm run build` 通过，并重新生成 `web/ui` 产物。
- `go test ./web/middleware ./web/controller ./web` 通过。
- `go test ./core/... ./web/service ./web/controller` 通过。
- `go test ./sub` 通过，覆盖订阅链接/配置、JSON outbound、Clash/mihomo proxy 的协议输出矩阵。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `go build -o bin/SuperXray.exe ./main.go` 通过。
- Phase 10 强制进入后，`prettier --check plans/04-ui-first-execution/phase-10-entry-gate-assessment.md plans/04-ui-first-execution/phase-10a-default-xray-readonly-adr.md plans/STATUS.md` 通过。
- `web/html`、`frontend/src` 未发现 `v-html`、`innerHTML`、`insertAdjacentHTML`。
- `web/html`、`frontend/src`、`web/ui` 未发现 `v-html`。
- `frontend/src` 和 `web/ui` 未发现 `unsafe-inline` 或 `unsafe-eval`。
- `core`、`web/controller`、`web/middleware`、`web/service`、`database/model`、`frontend/src`、`web/ui` 未发现 `proxy_inbounds` 或 `proxy_clients`。
- `CoreManager` 和 `sing-box` 仅在强制进入允许范围内出现：`core/`、`web/service/core_service.go`、`web/controller/core.go`、`web/controller/api.go` 与 Phase 10 文档。
- 新 UI 退场前补门禁已覆盖：`/panel/login` 新登录页、Dashboard Geo Maintenance、Xray Outbound Tools、Inbounds Import JSON/Reset All Traffic、Settings Two Factor Setup 和 Subscription Public Links。
- 新 UI 在线/IP 管理已覆盖：Inbounds 页展示 Online Clients、Refresh Activity，详情抽屉提供 Online / IP Management、View IPs、Clear IPs。
- `npm run e2e` 在本机隔离真实 Xray core 环境通过：19 passed / 0 skipped / 0 failed。
- E2E 已开启 `SUPERXRAY_E2E_MUTATION=1`、`SUPERXRAY_E2E_RESTART=1`、`SUPERXRAY_E2E_IMPORT_DB=1` 和 `SUPERXRAY_E2E_SUB_URL`。
- 新旧 UI 兼容自动抽检已覆盖：新 Vue UI 创建 VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard 六类禁用入站后，Legacy UI 与旧入站列表 API 可读取并在测试结束清理。
- Settings 兼容自动抽检已覆盖：新 Vue UI 保存订阅标题/公告后，Legacy Settings UI 可读取，并在测试结束恢复原值。
- 编辑兼容自动抽检已覆盖：新 Vue UI 编辑 VLESS 入站、VLESS 客户端和 WireGuard peer 后，Legacy UI 可读取入站行和 VLESS 客户端展开行，旧入站列表 API 可读取并在测试结束清理。
- 扩展客户端编辑兼容自动抽检已覆盖：新 Vue UI 编辑 Trojan、Shadowsocks、Hysteria2 客户端后，Legacy UI 展开行和旧入站列表 API 可读取并在测试结束清理。
- 通过浏览器访问 `http://127.0.0.1:2073/phase7a/panel/`，新 UI 默认入口正常，控制台无 error/warning。
- 新 UI 浏览器检查显示 21 个动态 `<style>` 节点全部带 CSP nonce。
- 通过浏览器访问 `http://127.0.0.1:2073/phase7a/panel/legacy/`，旧 UI 回退入口正常，控制台无 error/warning。
- 页面截图已保存为 `superxray-phase8-default-ui.png` 和 `superxray-phase8-legacy-ui.png`。

说明：

- 本次继续推进 Phase 9 安全收口：旧 UI HTML sink、session 级 CSRF token、未登录下载和非法 DB 导入已加入回归。
- 本次继续推进 Phase 10 风险接受/强制进入：新增最小 active CoreManager、`default-xray` 观察实例和 `experimental-sing-box` 外部进程适配入口。
- 本次继续推进 Xray 等价迁移缺口：补齐新 UI 在线/IP 管理入口与订阅输出矩阵 Go 回归；CI/第二环境复刻仍是后续门禁。
- 所有设置写入仍走旧 `/panel/setting/*`，数据库备份恢复仍走旧 `/panel/api/server/*`，但相关 POST 已要求有效 CSRF token；未改动旧 `model.Inbound`、数据库结构、旧订阅输出或现有 Xray 生命周期。
- 新 UI `script-src` 和 `style-src` 已保持无 `unsafe-inline` 和 `unsafe-eval`；Ant Design Vue 动态样式通过 nonce bootstrap 运行。Legacy UI 因历史内联脚本和 Vue 2 模板仍保留宽 CSP，仅作为回退入口接受。
- 本机隔离运行环境已补齐真实 Xray core：`Xray 26.3.27`，restart、订阅服务路径和数据库导入成功路径均已非跳过通过；本轮验证后本地测试面板继续运行在 `http://127.0.0.1:2073/phase7a/panel/`，用于后续人工验收。
