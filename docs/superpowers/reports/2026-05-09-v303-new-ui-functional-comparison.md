# 新 UI 与 V3.0.3 功能逻辑对比分析报告

日期：2026-05-09

## 1. 范围与方法

本报告对比对象：

- 旧版本基线：Git 标签 `v3.0.3`。
- 新 UI 当前态：当前工作区的 Vue 3 / Vite / TypeScript 前端，以及仍在复用的 Go 旧 API。
- 重点范围：登录、Dashboard、Logs、Xray、Inbounds、Settings、订阅、备份恢复、Custom Geo、旧 UI 回退和当前新增 Core Instances。

证据来源：

- 旧 UI 页面与逻辑：`v3.0.3:web/html/index.html`、`v3.0.3:web/html/inbounds.html`、`v3.0.3:web/html/xray.html`、`v3.0.3:web/html/settings.html`、`v3.0.3:web/assets/js/model/inbound.js`、`v3.0.3:web/assets/js/model/setting.js`。
- 旧 API：`v3.0.3:web/controller/inbound.go`、`v3.0.3:web/controller/server.go`、`v3.0.3:web/controller/setting.go`、`v3.0.3:web/controller/xray_setting.go`、`v3.0.3:sub/subController.go`。
- 新 UI 页面与 API SDK：`frontend/src/views/*.vue`、`frontend/src/api/*.ts`、`frontend/src/types/*.ts`、`frontend/src/utils/inboundCompat.ts`。
- 当前后端新增或增强：`web/controller/core.go`、`web/controller/server.go`、`web/controller/xui.go`、`sub/subController.go`、`sub/subscription_profile.go`、`sub/subscription_diagnostic.go`。
- 当前前端回归测试：`frontend/tests/inbounds-view.test.ts`、`frontend/tests/settings-view.test.ts`、`frontend/tests/form-styles.test.ts`。

阶段边界：

- 当前仍属于 UI-first 迁移路线，阶段门禁要求新 UI 写入继续兼容旧 Xray API 和旧数据模型。
- Phase 10 前禁止迁移 `model.Inbound`、禁止新增 sing-box 写路径、禁止移除旧 UI。
- 因此报告中的“未保留”特指“尚未在新 Vue UI 中提供等价图形化入口或完整流程”，不等于后端能力被删除；多数缺口仍可通过 `/panel/legacy` 旧 UI 或 JSON 编辑回退。

## 2. 总体结论

新 UI 已经保留并增强了日常核心闭环的大部分能力：

- 登录和 2FA 登录判断保留，并新增 Vue 表单校验、双语标题与语言切换。
- Dashboard 保留状态、流量、入站和客户端概要，新增 Custom Geo 资源管理入口和更清晰的状态卡片。
- Logs 保留面板日志、Xray 日志、级别/行数/直连/阻断/代理过滤，增强为纯文本虚拟滚动、复制、下载和自动跟随，消除了旧 UI `v-html` 日志渲染风险。
- Inbounds 已保留入站列表、搜索过滤、创建、编辑、启停、删除、导入 JSON、客户端增删改启停、重置流量、删除耗尽客户端、在线/最后在线/IP 记录和分享链接生成，并通过新表单分区增强可读性和移动端体验。
- Settings 已保留 V3.0.3 的绝大多数字段和保存、默认值、凭证更新、2FA、订阅、Telegram、LDAP、备份恢复、面板重启流程，并通过统一分区和推荐客户端链接增强体验。
- Xray 生命周期、版本安装、配置读写、出站流量统计、出站测试、Warp/Nord 数据动作保留；但旧版结构化高级 Xray 配置编辑器未完整移植。
- 订阅输出保留 URI、JSON、Clash/Mihomo、订阅页、Header 元数据；当前后端新增 `target` 自动格式选择和诊断端点，属于明显功能增强。
- 旧 UI 通过 `/panel/legacy/` 显式保留，是当前未迁移边缘功能的主要回退路径。

主要缺口集中在四类：

- Xray 结构化配置编辑器：旧版 Routing、Outbounds、Reverse、Balancers、DNS、FakeDNS、Protocol Tools、规则拖动/增删改等图形化能力，在新 UI 中主要退回为 JSON 编辑。
- Inbounds 边缘操作：旧版入站克隆、二维码、全量导出链接、全量导出订阅、单入站导出订阅、客户端批量导入、从其他入站复制客户端，在新 UI 当前态没有完整等价入口。
- 高级生成器与辅助表单：旧版 X25519、ML-DSA-65、ML-KEM-768、ECH、VLESS encryption、Reality 目标预设、FinalMask、External Proxy 等高级辅助没有完整新 UI 图形化入口，部分只能手动填 JSON。
- Settings 辅助体验：旧版安全配置告警、LDAP 入站标签多选辅助在新 UI 中没有完整保留；底层字段仍保留。

## 3. 功能分类总览

| 分类 | 功能域 | 当前结论 |
| --- | --- | --- |
| 已保留并增强 | 登录、日志、Settings 表单组织、订阅推荐链接、订阅诊断、Custom Geo、备份导入安全校验、旧 UI 回退、新 UI 工程化、Core Instances 只读视图 | 保留旧 API 或旧数据格式，同时提升交互、可维护性或安全性 |
| 已保留但未明显增强 | Xray 基础生命周期、Xray 版本安装、Xray 配置 JSON 保存、Settings 大多数字段、Inbounds 核心 CRUD、客户端基本 CRUD、入站 JSON 导入 | 功能可用，但主要是旧逻辑迁移或外观重组 |
| 未完整保留 | Xray 结构化高级编辑器、入站克隆/二维码/全量导出/客户端复制/批量客户端导入、Reality/ECH/VLESS 等高级生成按钮、CPU 历史图、Settings 安全告警、LDAP 标签多选 | 新 UI 尚无等价图形化入口；旧 UI 或 JSON 仍可回退 |
| 新增功能 | Vue 3/Vite/TS 工程化、API SDK、Pinia Store、严格新 UI 路由、虚拟日志、订阅 target/diagnose、Core Instances、表单公共组件、推荐客户端链接、统一响应式表单样式 | V3.0.3 不具备或不完整 |

## 4. 逐功能详细对比

### 4.1 登录与会话

旧版 V3.0.3：

- `web/controller/index.go` 提供 `/login`、`/logout`、`/getTwoFactorEnable`。
- `web/html/login.html` 使用用户名、密码和可选 `twoFactorCode` 提交登录。
- 后端负责校验空用户名、空密码、用户名密码、2FA，并写入 session。

新 UI 当前态：

- `frontend/src/api/auth.ts` 继续调用 `login` 和 `getTwoFactorEnable`。
- `frontend/src/views/LoginView.vue` 保留 username、password、twoFactorCode，并在 mounted 后加载 2FA 开关。
- 新增 Ant Design Vue 表单规则、loading 状态、错误提示、文档标题本地化和语言切换。

结论：已保留并增强。

增强点：

- 旧后端认证路径不变。
- 前端表单校验更明确。
- 中英文动态文案和标题增强了可用性。

### 4.2 Dashboard 与状态监控

旧版 V3.0.3：

- `index.html` 展示 CPU、内存、网络流量、磁盘、Swap、TCP/UDP、Public IP、Xray 状态和版本。
- 通过 `/panel/api/server/status` 获取状态。
- Dashboard 内直接提供 Restart Xray、Stop Xray、日志模态框、Xray 版本切换、Geo 更新、Custom Geo、备份导入、CPU History 图表。

新 UI 当前态：

- `DashboardView.vue` 继续展示 Xray state/version、CPU、Memory、Traffic、Inbounds、Clients、Panel Uptime、Connections、Public IP、Load、Disk、Swap。
- `server` store 使用 WebSocket 实时刷新状态。
- Geo 更新和 Custom Geo 管理已保留，并拆成 Dashboard 中的清晰表格和模态表单。
- Xray 启停、版本安装迁移到 `XrayView.vue`；备份导入迁移到 `SettingsView.vue` 的 Backup tab。

结论：核心状态能力已保留并部分增强。

已保留并增强：

- 状态卡片更清晰，Custom Geo 从旧版折叠模态区升级为 Dashboard 工作区。
- WebSocket 状态更新由 `stores/server.ts` 集中管理。

已保留但迁移入口：

- Xray 生命周期和版本管理从 Dashboard 转移到 Xray 页面。
- 备份恢复从 Dashboard 转移到 Settings Backup。

未完整保留：

- 旧 Dashboard 的 CPU History 图表和 `/panel/api/server/cpuHistory/:bucket` 可视化入口未在新 UI 中呈现；当前新 UI只显示即时 CPU 和负载摘要。

### 4.3 Logs 日志中心

旧版 V3.0.3：

- Dashboard 模态框读取面板日志和 Xray 日志。
- 面板日志支持行数、level、syslog。
- Xray 日志支持行数、关键词、Direct、Blocked、Proxy。
- 日志通过 `formattedLogs` 拼 HTML 后 `v-html` 渲染。

新 UI 当前态：

- `LogsView.vue` 独立成页面。
- 继续调用 `/panel/api/server/logs/:count` 和 `/panel/api/server/xraylogs/:count`。
- 保留行数、level、syslog、关键词、Direct、Blocked、Proxy。
- 新增 `VirtualLogViewer.vue`，日志行用 Vue 插值 `{{ item.line }}` 纯文本渲染。
- 支持 copy/download/auto follow。
- 当前 `rg v-html frontend/src web/ui` 未发现新 UI 使用 `v-html`。

结论：已保留并显著增强。

增强点：

- 从模态框升级为独立页面。
- 虚拟滚动降低大量日志 DOM 压力。
- 纯文本渲染降低日志 XSS 风险。

### 4.4 Xray 生命周期、版本和配置

旧版 V3.0.3：

- `xray.html` 提供保存 Xray 设置、重启 Xray、获取运行结果。
- Dashboard 也可停止、重启、安装 Xray 版本。
- `xray_setting.go` 提供 `/panel/xray/`、`/panel/xray/update`、`/panel/xray/getXrayResult`、`/panel/xray/getOutboundsTraffic`、`/panel/xray/resetOutboundsTraffic`、`/panel/xray/testOutbound`、`/panel/xray/warp/:action`、`/panel/xray/nord/:action`。

新 UI 当前态：

- `XrayView.vue` 保留刷新运行状态、Start/Restart、Stop、版本列表、安装版本、加载配置、格式化 JSON、保存配置、保存后重启确认。
- 保留 Xray Result、Outbounds Traffic、Reset All/Single Outbound Traffic、Test First Outbound、Warp/Nord provider data actions。
- `frontend/src/api/xray.ts` 继续走旧 `/panel/xray/*` 端点。

结论：基础能力已保留，但高级结构化编辑未完整保留。

已保留但未明显增强：

- Xray 启停/重启、版本安装、配置保存和出站流量重置仍是旧 API 的新 UI 包装。

已保留并部分增强：

- JSON 编辑提供格式化、复制、下载、保存后重启确认。
- Provider data 操作集中在 Xray 页面。

未完整保留：

- 旧版结构化模板页签未完整移植：Basic、Routing、Outbounds、Reverse、Balancers、DNS、FakeDNS、Protocol Tools、Advanced。
- 旧版图形化增删改能力未完整移植：`addOutbound/editOutbound/deleteOutbound`、`addReverse/editReverse/deleteReverse`、`addBalancer/editBalancer/deleteBalancer`、`addDNSServer/editDNSServer/deleteDNSServer`、`addFakedns/editFakedns/deleteFakedns`、`addRule/editRule/replaceRule/deleteRule`。
- 旧版 DNS Presets、Warp/Nord 模态配置和 Protocol Tools 组合生成器没有完整图形化等价入口。

未保留原因：

- 新 UI 当前实现选择先保留 Xray 配置 JSON 闭环，以避免阶段 6/7 期间重写复杂模板编辑器导致旧配置格式漂移。
- 旧 UI 仍通过 `/panel/legacy/xray` 保留，复杂结构化编辑可作为回退。
- 若要补齐，应作为单独 Phase 5 parity 子任务，按 Outbounds、Routing、DNS、Reverse、Balancers 分批迁移并补 E2E。

### 4.5 Inbounds 入站管理

旧版 V3.0.3：

- 入站列表展示协议、备注、地址、客户端、流量、到期、启用状态。
- 支持自动刷新、在线客户端、最后在线、客户端状态统计。
- 支持新增、编辑、删除、克隆、启停、导入入站。
- 支持全局重置入站流量、全局重置客户端流量、删除耗尽客户端。
- 支持单入站二维码、导出链接、导出订阅、导出入站 JSON、复制入站、复制客户端。
- 客户端支持新增、批量新增、编辑、删除、重置流量、复制到其他入站。
- 后端端点包括 `/panel/api/inbounds/add`、`update/:id`、`del/:id`、`import`、`addClient`、`updateClient/:clientId`、`/:id/delClient/:clientId`、`/:id/resetClientTraffic/:email`、`resetAllTraffics`、`resetAllClientTraffics/:id`、`delDepletedClients/:id`、`onlines`、`lastOnline`、`clientIps/:email`、`clearClientIps/:email`、`/:id/copyClients`、`updateClientTraffic/:email`、`/:id/delClientByEmail/:email`。

新 UI 当前态：

- `InboundsView.vue` 保留列表、协议过滤、状态过滤、搜索、启停、详情 Drawer。
- 保留新增、编辑、删除、导入 JSON、重置所有入站流量。
- 保留客户端列表、单客户端新增/编辑/删除/启停/重置。
- 保留选中客户端批量删除、选中客户端批量重置、重置单入站所有客户端、删除耗尽客户端。
- 保留在线/最后在线/IP 记录查看和清空。
- 保留客户端分享链接生成和导出选中客户端链接。
- 表单已重组为 Basic Inbound、WireGuard Settings、Transport Settings、Default Client、Advanced JSON。
- `frontend/tests/inbounds-view.test.ts` 锁定 Default Client、JSON sync/apply、format JSON 和提交前同步默认客户端到 settings JSON。

结论：核心 CRUD 与客户端管理已保留并增强；部分旧版边缘操作未完整保留。

已保留并增强：

- 入站与客户端详情从旧表格展开和多模态框，升级为列表加 Drawer 的任务组织。
- 新表单保留旧 `settings`、`streamSettings`、`sniffing` JSON，同时提供 Transport/Client/WireGuard 同步表单。
- WireGuard 客户端和服务端密钥可在前端生成，逻辑位于 `inboundCompat.ts` 与 `InboundsView.vue`。
- 客户端 IP 记录查看和清空保留，且展示更集中。

已保留但未明显增强：

- 新增/编辑/删除/启停入站仍调用旧 API。
- 客户端增删改、重置流量仍调用旧 API。

未完整保留：

- 入站克隆 `openCloneInbound`。
- 单入站二维码 `qrcode` 模态框。
- 全量导出所有入站分享链接 `exportAllLinks`。
- 全量导出订阅 `exportAllSubs`、单入站订阅导出 `exportSubs`。
- 单入站导出入站 JSON 的独立入口。
- 旧版客户端批量新增 `addBulkClient`。
- 从其他入站复制客户端 `copyClients`，对应后端 `/panel/api/inbounds/:id/copyClients` 当前仍存在，但新 UI 未封装到 `frontend/src/api/inbounds.ts`。
- `updateClientTraffic/:email`、`delClientByEmail/:email` 后端能力未出现在新 UI。
- 二维码视觉证据和扫码路径未在新 UI 提供。

未保留原因：

- 当前新 UI 优先迁移核心新增/编辑/删除/分享链接路径，降低协议表单重写风险。
- 批量导入、复制客户端和二维码涉及旧模态框、分享链接生成、选择器和跨入站规则，应单独补迁移与回归测试。
- 旧 UI 回退入口仍保留，短期可承接这些边缘操作。

### 4.6 Inbounds 协议表单和高级协议逻辑

旧版 V3.0.3：

- 协议覆盖 VMess、VLESS、Trojan、Shadowsocks、HTTP、Tunnel、Mixed、Tun、WireGuard、Hysteria/Hysteria2。
- 每个协议有独立模板。
- Stream 支持 TCP、mKCP、WS、gRPC、HTTPUpgrade、XHTTP、Hysteria、TLS、Reality、Sockopt、FinalMask、External Proxy。
- 提供多种生成器：UUID、Shadowsocks password、WireGuard keypair/psk、X25519、ML-DSA-65、ML-KEM-768、ECH、VLESS encryption 等。
- Reality 表单有目标预设、Short IDs 生成、证书/种子生成。

新 UI 当前态：

- `protocolRegistry.ts` 注册 VMess、VLESS、Tunnel、HTTP、Trojan、Shadowsocks、Mixed、WireGuard、Tun、Hysteria、Hysteria2。
- `InboundsView.vue` 支持基本字段、WireGuard、Transport、Default Client、Advanced JSON。
- Transport 表单覆盖 TCP、kcp、ws、grpc、httpupgrade、xhttp、tls、reality、hysteria、sockopt 的主要字段。
- Advanced JSON 保留手工编辑 `settings`、`streamSettings`、`sniffing`。

结论：主要协议和主要字段已保留；高级辅助功能未完整保留。

已保留并增强：

- 协议能力集中到 `protocolRegistry.ts`，可读性优于旧版散落判断。
- 表单分区和 JSON 同步路径降低字段堆叠复杂度。

已保留但依赖手工 JSON：

- External Proxy、FinalMask 等复杂结构可通过 Advanced JSON 保留或编辑，但没有完整图形化编辑器。
- 部分已存在的 `streamSettings` 子对象会被保留，但新表单不会生成完整旧版复杂结构。

未完整保留：

- X25519、ML-DSA-65、ML-KEM-768、ECH、VLESS encryption 等后端生成器在 `web/controller/server.go` 仍存在，但新 UI 未封装。
- Reality target 预设、Short IDs 快速生成、ECH 证书生成按钮未完整迁移。
- FinalMask 图形化 TCP/UDP mask 编辑器未迁移。
- External Proxy 图形化管理未迁移。

未保留原因：

- 这些能力属于高复杂度协议辅助，和入站核心 CRUD 解耦。
- 当前阶段采用 Advanced JSON 作为兼容安全网，避免新图形化表单写出旧 UI 不可读结构。

### 4.7 Settings 面板设置

旧版 V3.0.3：

- Panel General 保存 webListen、webDomain、webPort、webBasePath、TLS 文件、sessionMaxAge、pageSize、expireDiff、trafficDiff、remarkModel、datepicker、timeLocation。
- 顶部保存调用 `/panel/setting/update`，重启调用 `/panel/setting/restartPanel`。
- 页面包含安全告警：非 HTTPS、默认面板端口、默认 panel URI、默认 sub URI、默认 JSON URI。

新 UI 当前态：

- `SettingsView.vue` 保留全部上述字段。
- 使用 `loadSettings`、`resetForm`、`confirmSave`、`saveSettings` 调用旧 API。
- 表单拆为 Web Endpoint、TLS Files、Session and Display、Thresholds and Naming。
- `settings-view.test.ts` 锁定字段绑定和关键动作。

结论：字段与保存逻辑已保留并增强表单组织；安全告警未完整保留。

已保留并增强：

- 表单结构清晰，桌面/移动端共用 `FormSection` 和响应式网格。

未完整保留：

- 旧版 `securityAlerts` 对默认危险配置的实时提醒未在新 UI 中实现。

未保留原因：

- 本轮表单重组主要锁定字段和保存路径，未同步迁移旧版设置页的派生安全提示逻辑。

### 4.8 Settings 安全与凭证

旧版 V3.0.3：

- 支持修改用户名/密码。
- 开关 2FA 时使用 two factor modal 校验 TOTP。
- 支持生成和删除 twoFactorToken。

新 UI 当前态：

- 保留 Credentials 区：oldUsername、oldPassword、newUsername、newPassword。
- 保留 `confirmUpdateCredentials` 和 `updateCredentials`，继续调用 `/panel/setting/updateUser`。
- 保留 Two Factor 开关字段、token、setup URI、generate token、disable two factor。

结论：已保留并部分增强。

增强点：

- setup URI 可直接在页面生成，字段组织更清楚。

保留但行为需注意：

- 旧版开启/关闭 2FA 有 TOTP modal 校验步骤；新 UI 当前通过生成/禁用 token 和保存设置完成，未完全复刻旧弹窗交互。后端保存字段不变，但用户操作流程不完全一致。

### 4.9 Settings 订阅与格式

旧版 V3.0.3：

- 支持 URI、JSON、Clash/Mihomo、Encrypted、Show Info、Routing。
- 支持 subPath、subJsonPath、subClashPath、subURI、subJsonURI、subClashURI、title、updates、domain、listen、port、TLS 文件、support/profile/announce/routing headers。
- 支持 JSON fragment/noises/mux/rules 格式配置。
- 订阅页展示 URI、JSON、Clash 链接和二维码。

新 UI 当前态：

- 保留全部订阅字段和 JSON 格式字段。
- 新增 Public Links 只读输出、复制链接、打开 URI/JSON/Clash。
- 新增 Recommended Client Links，基于 `target` 参数生成 Happ、Streisand、V2rayNG、NekoBox、FoXray、Sing-box、Mihomo 等推荐链接。
- `sub/subController.go` 新增 `target` 解析和 `:subid/diagnose`、JSON diagnose、Clash diagnose。
- `sub/subscription_profile.go` 和 `sub/subscription_diagnostic.go` 新增目标 profile 和诊断统计。

结论：已保留并显著增强。

增强点：

- 新 UI 不只展示基础公开链接，还生成面向客户端的推荐链接。
- 后端新增 target-aware 输出和诊断端点，减少用户手动判断订阅格式。
- JSON 格式字段保留源码测试覆盖。

未完整保留：

- 新 UI Settings 中未提供旧订阅页二维码渲染；公开链接可复制/打开，但没有内置二维码。

### 4.10 Settings Telegram

旧版 V3.0.3：

- 支持 tgBotEnable、tgBotToken、tgBotProxy、tgBotAPIServer、tgBotChatId、tgRunTime、tgBotBackup、tgBotLoginNotify、tgCpu、tgLang。

新 UI 当前态：

- 保留全部字段，拆为 Bot Flags、Bot Connection、Runtime Rules。
- 仍通过 `/panel/setting/update` 保存。

结论：已保留并增强表单组织。

### 4.11 Settings LDAP

旧版 V3.0.3：

- 支持 ldapEnable、ldapHost、ldapPort、ldapUseTLS、ldapBindDN、ldapPassword、ldapBaseDN、ldapUserFilter、ldapUserAttr、ldapVlessField、ldapSyncCron、ldapFlagField、ldapTruthyValues、ldapInvertFlag、ldapInboundTags、ldapAutoCreate、ldapAutoDelete、ldapDefaultTotalGB、ldapDefaultExpiryDays、ldapDefaultLimitIP。
- `loadInboundTags` 加载入站 tag，并以多选控件辅助配置 `ldapInboundTags`。

新 UI 当前态：

- 保留全部 LDAP 字段。
- 拆为 LDAP Flags、Connection、User Mapping、Sync Defaults。
- `ldapInboundTags` 当前是字符串输入。

结论：字段保留并增强布局，但入站标签选择辅助未完整保留。

未完整保留：

- 旧版 LDAP 入站标签多选和动态加载 tag 没有迁移；新 UI 需要用户手动输入 CSV。

未保留原因：

- 当前 Settings 重组优先字段和保存路径，未把旧版派生选择器纳入公共组件。

### 4.12 备份恢复和面板运行时

旧版 V3.0.3：

- Dashboard 的 backup modal 支持导出数据库和导入数据库。
- Settings 支持 Restart Panel。
- 旧后端 `importDB` 导入后会重启 Xray。

新 UI 当前态：

- `SettingsView.vue` Backup tab 保留 Download、Import 和 Restart Panel。
- `api/server.ts` 继续使用 `/panel/api/server/getDb` 和 `/panel/api/server/importDB`。
- 当前 `web/controller/server.go` 对导入数据库新增大小、文件名和扩展名校验。

结论：已保留并增强。

增强点：

- 入口从 Dashboard 模态框迁移为 Settings Backup 专区，语义更清晰。
- 导入数据库有明确危险操作确认。
- 后端导入校验比旧版更严格。

### 4.13 Custom Geo 与 Geo 文件

旧版 V3.0.3：

- Dashboard version modal 中支持更新 geoip/geosite、更新全部 geofile。
- Custom Geo 支持 list、add、update、delete、download、update-all、aliases。

新 UI 当前态：

- Dashboard 中保留 geofile 更新：geoip.dat、geosite.dat、update all。
- Custom Geo 在 Dashboard 中独立表格化，支持 add、edit、delete、download、update resources。
- API SDK 有 `customGeo.ts` 封装 list/add/update/delete/download/updateAll。

结论：已保留并增强。

未完整保留：

- `aliases` 后端端点当前没有新 UI 显示入口；旧版主要在 Xray Routing/DNS 等高级表单中使用别名辅助，新 UI 高级 Xray 表单未迁移，所以该辅助链路也未迁移。

### 4.14 订阅后端输出逻辑

旧版 V3.0.3：

- `SUBController` 支持 URI 订阅、JSON 订阅、Clash 订阅。
- 支持 HTML 订阅页、订阅 header、加密输出、用户流量信息、Profile Title/Support/Profile URL/Announce/Routing。

新 UI 当前态：

- 保留上述输出逻辑。
- 新增 `target` 参数解析：URI 入口可以根据目标客户端输出 JSON 或 Clash。
- 新增 diagnose 端点，返回订阅格式下入站可输出和跳过原因。
- 新增或强化订阅 protocol capability 测试。

结论：已保留并增强。

增强点：

- 客户端推荐链接能直接落到对应目标格式。
- 诊断接口帮助定位“订阅为空/协议不支持/格式不匹配”等边缘问题。

### 4.15 旧 UI 回退

旧版 V3.0.3：

- `/panel/`、`/panel/inbounds`、`/panel/settings`、`/panel/xray` 直接渲染旧 HTML。

新 UI 当前态：

- `frontend/src/router/index.ts` 使用新 Vue Router 管理 `/`、`/logs`、`/cores`、`/xray`、`/inbounds`、`/settings`。
- `web/controller/xui.go` 将旧 UI 显式保留在 `/panel/legacy/`、`/panel/legacy/inbounds`、`/panel/legacy/settings`、`/panel/legacy/xray`。

结论：新增并符合阶段门禁。

增强点：

- 旧 UI 回退边界明确，降低新 UI parity 缺口风险。

### 4.16 Core Instances

旧版 V3.0.3：

- 没有 Core Instances 页面，也没有 `/panel/api/cores/*`。

新 UI 当前态：

- 新增 `web/controller/core.go`，提供 list/get/status/validate/start/stop/restart。
- 新增 `CoreInstancesView.vue` 和 `stores/core.ts`。
- 页面展示 capabilities，并在不支持生命周期时禁用按钮。
- 测试覆盖 unsupported lifecycle 返回错误。

结论：新增功能。

阶段风险说明：

- 该能力必须保持 Phase 10.1/10.2 门禁：默认 Xray 只读或能力受限展示，不能提前把旧 Xray 生命周期切入 CoreManager。

## 5. 已保留并增强的功能清单

| 功能 | 旧版实现 | 新 UI 实现 | 增强点 |
| --- | --- | --- | --- |
| 登录与 2FA 判断 | login.html + `/login` + `/getTwoFactorEnable` | `LoginView.vue` + `api/auth.ts` | 表单规则、loading、双语、标题本地化 |
| 日志查看 | Dashboard modal + `v-html` | 独立 Logs 页面 + `VirtualLogViewer` | 虚拟滚动、纯文本、复制、下载、自动跟随 |
| Settings 字段保存 | `AllSetting` + `/panel/setting/update` | `SettingsView.vue` + `PanelSettings` | 分区、公共表单组件、移动端样式、字段测试 |
| 订阅公开链接 | 旧 Settings/subpage | Public Links + Recommended Client Links | 推荐客户端 target 链接、复制/打开 |
| 订阅输出 | URI/JSON/Clash | 保留并新增 target/diagnose | 客户端格式自动选择、诊断统计 |
| Inbound 表单组织 | 大型旧模态框 | FormSection 分区 | 可读性、响应式、JSON 同步测试 |
| WireGuard 客户端 | 旧模型生成 | `inboundCompat.ts` 生成 | Vue 侧统一生成与分享链接 |
| Custom Geo | Dashboard 折叠区 | Dashboard 工作区表格 | 操作更集中 |
| 备份导入 | Dashboard modal | Settings Backup tab | 危险确认、后端大小/扩展名校验 |
| 旧 UI 回退 | 无显式 legacy 路径 | `/panel/legacy/` | 符合迁移门禁 |

## 6. 已保留但未明显增强的功能清单

| 功能 | 当前状态 | 说明 |
| --- | --- | --- |
| Xray Start/Restart/Stop | 保留 | 仍走旧 `/panel/api/server/*XrayService` |
| Xray 版本安装 | 保留 | 仍走旧 `/panel/api/server/installXray/:version` |
| Xray JSON 配置保存 | 保留 | 保留 JSON 读写，结构化编辑未迁移 |
| Inbounds 基础 CRUD | 保留 | 仍走旧 `/panel/api/inbounds/*` |
| 客户端基础 CRUD | 保留 | 仍走旧 addClient/updateClient/delClient |
| 入站导入 JSON | 保留 | 仍走旧 import 端点 |
| 面板设置保存 | 保留 | 仍走旧 setting/update |
| Telegram 设置 | 保留 | 字段完整，主要是布局增强 |
| LDAP 设置 | 保留 | 字段完整，但 tag 多选辅助未迁移 |

## 7. 未完整保留的功能和原因

| 功能 | 旧版位置 | 新 UI 当前状态 | 影响范围 | 原因 |
| --- | --- | --- | --- | --- |
| Xray 结构化高级编辑器 | `web/html/xray.html` + `settings/xray/*` + 多个 modal | 仅 JSON 编辑和摘要 | 高级运维用户 | 复杂度高，当前先保证 JSON 读写兼容，旧 UI 回退 |
| Routing 规则增删改排序 | `addRule/editRule/replaceRule/deleteRule` | 未提供图形化入口 | 路由规则维护 | 需单独迁移规则表单与排序 E2E |
| Outbounds 图形化 CRUD | `addOutbound/editOutbound/deleteOutbound` | 只可通过 JSON | 出站配置维护 | 需迁移 outbound modal |
| DNS/FakeDNS 图形化配置 | DNS/FakeDNS modal | 只可通过 JSON | DNS 高级配置 | 需迁移 DNS presets 和 FakeDNS 表单 |
| Reverse/Balancer 图形化配置 | reverse/balancer modal | 只可通过 JSON | 高级路由 | 需迁移关联规则同步逻辑 |
| Protocol Tools | `protocol_tools.js` 和 Xray 页 | 未完整入口 | 组合配置生成 | 属高级工具，尚未进入新 UI 范围 |
| 入站克隆 | `openCloneInbound` | 未提供 | 批量创建类似入站 | 可用旧 UI 回退；新 UI 需复制 payload 逻辑 |
| 入站二维码 | qrcode modal | 未提供 | 移动端扫码导入 | 分享链接可复制，但无二维码 |
| 全量分享/订阅导出 | `exportAllLinks/exportAllSubs/exportSubs` | 未提供完整入口 | 批量导出用户 | 新 UI 仅支持选中客户端导出链接 |
| 客户端批量新增 | `addBulkClient` modal | 未提供 | 批量用户导入 | 旧批量 modal 未迁移 |
| 从其他入站复制客户端 | `/copyClients` + modal | 后端存在，新 UI 未封装 | 跨入站迁移用户 | 需新 API 封装和选择器 |
| Reality/X25519/ML-DSA/ECH/VLESS 生成器 | inbound modal + server generator API | 未完整封装 | 高级协议配置 | 生成 API 仍在后端，前端未接入 |
| FinalMask 图形化编辑 | `stream_finalmask.html` | 仅 JSON | 高级混淆配置 | 结构复杂，尚未图形化 |
| External Proxy 图形化编辑 | `stream/external_proxy.html` | 仅保留/JSON | CDN/反代分享 | 新表单未覆盖 |
| CPU History 图表 | Dashboard CPU modal | 未提供 | 性能观察 | 当前 Dashboard 只显示即时状态 |
| Settings 安全告警 | `securityAlerts` | 未提供 | 默认危险配置提醒 | 表单重组未迁移派生告警 |
| LDAP 入站标签多选 | `loadInboundTags` + multiple select | 改为字符串输入 | LDAP 同步配置易用性 | 数据字段保留，选择辅助未迁移 |
| 订阅二维码 | 旧 subpage | Settings 不提供二维码 | 客户端扫码 | 链接可复制/打开，二维码未迁移 |

## 8. 新增功能清单

| 新增功能 | 位置 | 价值 |
| --- | --- | --- |
| Vue 3 / Vite / TypeScript 新工程 | `frontend/` | 可维护、可构建、可测试 |
| API SDK 分层 | `frontend/src/api/*` | 组件不直接散落旧 API URL |
| Pinia 状态管理 | `stores/server.ts`、`stores/core.ts` | 状态和 WebSocket 集中管理 |
| 新路由与布局 | `router/index.ts`、`MainLayout.vue` | 新 UI 独立导航 |
| VirtualLogViewer | `components/VirtualLogViewer.vue` | 高性能日志和 XSS 收口 |
| FormSection | `components/FormSection.vue` | 表单组织规范复用 |
| 响应式表单样式 | `app.css` | 移动端表单不横向溢出 |
| Recommended Client Links | `SettingsView.vue` | 减少订阅配置试错 |
| 订阅 target/profile | `sub/subscription_profile.go` | 订阅入口按客户端选择格式 |
| 订阅 diagnose | `sub/subscription_diagnostic.go` | 诊断订阅为空和格式不匹配 |
| Core Instances | `web/controller/core.go`、`CoreInstancesView.vue` | 为后续多内核阶段预埋受控只读/能力视图 |
| `/panel/legacy/` | `web/controller/xui.go` | 明确旧 UI 回退路径 |
| 数据库导入安全校验 | `web/controller/server.go` | 限制大小、文件名、扩展名 |

## 9. 风险与建议

高优先级建议：

1. 为 Xray 结构化高级编辑器建立单独 parity 计划。优先顺序建议为 Outbounds、Routing、DNS/FakeDNS、Balancers、Reverse、Protocol Tools。
2. 补 Inbounds 边缘操作：客户端批量新增、复制客户端、入站克隆、二维码、导出全部链接/订阅。
3. 接入后端已有高级生成器：X25519、ML-DSA-65、ML-KEM-768、ECH、VLESS encryption，并补 Reality Short IDs / target preset。
4. 恢复 Settings 安全告警和 LDAP 入站标签选择器，避免新 UI 在安全提醒和运维易用性上弱于旧 UI。
5. 为 CPU History 增加新 UI 可视化，或在报告/计划中明确不再作为新 UI parity 目标。

中优先级建议：

1. 为订阅二维码补轻量组件，复用 Public Links 和 Recommended Client Links。
2. 为 Advanced JSON 区增加“检测结构化表单未覆盖字段”的提示，避免用户点击 Apply 后误以为所有高级字段都已图形化管理。
3. 为未迁移旧 UI 能力在新 UI 中加入“Legacy fallback”快捷入口，降低用户迷路成本。

验证建议：

1. 当前报告只做静态功能逻辑对比，未运行真实浏览器 E2E。
2. 若进入修复阶段，应按缺口拆分测试先行：源码结构测试、API SDK 测试、浏览器多视口截图和关键流程 E2E。
3. Inbounds 修复必须验证新 UI 写入后旧 UI 可读可编辑。
4. Settings 修复必须验证保存后订阅输出与旧 UI 一致。

## 10. 结论

新 UI 对 V3.0.3 的日常核心功能已经覆盖较好，尤其是登录、日志、入站基础管理、客户端基础管理、设置保存、订阅输出、备份恢复和 Custom Geo。新 UI 的主要价值在于工程化、表单组织、日志安全、响应式和订阅推荐/诊断。

但不能宣称“已完全等价 V3.0.3”。当前仍存在明确未完整保留的功能，主要集中在旧版 Xray 结构化高级编辑器和 Inbounds 的批量/导出/二维码/高级生成器。由于旧 UI 已保留在 `/panel/legacy/`，这些缺口短期有回退路径；若目标是新 UI 完全替代旧 UI，则上述缺口应进入后续 parity 修复计划，并以测试先行方式逐项关闭。
