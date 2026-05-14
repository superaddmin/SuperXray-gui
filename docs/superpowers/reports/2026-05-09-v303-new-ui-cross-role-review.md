# V3.0.3 与新 UI 功能逻辑对比报告多角色交叉验证评审

日期：2026-05-09

> **状态更新（2026-05-15）**：本评审是对 2026-05-09 静态报告的交叉验证，部分风险项已经在后续版本收敛。`v3.0.10` 补齐 Inbounds 全量/单入站分享与订阅导出、二维码、JSON、Clone、Reset 等入口；`v3.0.11` 修复订阅开关启用但公开 URI 未保存时出现 `No subscription links are available for this inbound` 的问题。文内关于这些入口“未迁移”的表述仅保留为历史审计背景。

## 1. 评审对象与核实边界

本评审基于 `docs/superpowers/reports/2026-05-09-v303-new-ui-functional-comparison.md` 的结论进行二次交叉验证。

参与视角：

- 产品经理：业务闭环、需求符合度、用户价值和发布边界。
- 开发工程师：技术实现可行性、旧 API 兼容、数据流转和阶段门禁。
- 业务分析师：端到端流程连贯性、角色路径、异常与回退机制。
- QA 测试专家：场景覆盖、回归风险、可测试性和潜在缺陷。

核实方法：

- 静态复核原报告结论。
- 抽查新 UI 当前态：`frontend/src/views/InboundsView.vue`、`frontend/src/views/SettingsView.vue`、`frontend/src/views/XrayView.vue`、`frontend/src/api/endpoints.ts`、`frontend/src/api/inbounds.ts`。
- 抽查旧版 `v3.0.3`：`web/html/inbounds.html`、`web/html/index.html`、`web/html/settings.html`、`web/controller/inbound.go`、`web/controller/server.go`、订阅二维码页面与旧入站模态框。
- 复核 UI-first phase gate：Phase 6/7 要求新写入保持 legacy-compatible，Phase 8 要求 `/panel/legacy` 可用，Phase 10 前禁止迁移 `model.Inbound`、新增 sing-box 写路径或移除旧 UI。

边界声明：

- 本报告是“交叉验证评审”，不是修复实施报告。
- 本轮未运行真实浏览器 E2E、视觉截图、多视口测试或订阅输出逐字节对比。
- 因此最终结论只能确认“原报告的静态逻辑判断基本成立”，不能声明“新 UI 已完全等价 V3.0.3”。

## 2. 关键差异点复核结论

| 差异点 | 原报告结论 | 交叉验证裁决 | 证据与说明 |
| --- | --- | --- | --- |
| 日常核心闭环 | 大部分已保留 | 确认 | 登录、Logs、Settings 保存、Inbounds 基础 CRUD、客户端基础管理、备份恢复、Custom Geo 仍走旧 API 或等价封装。 |
| Xray 结构化高级编辑器 | 未完整保留 | 确认，且影响应标为高 | 新 UI `XrayView.vue` 主要提供 JSON 编辑、摘要、出站流量和生命周期；旧版存在 Routing/Outbounds/DNS/FakeDNS/Reverse/Balancer 等结构化 CRUD。 |
| Inbounds 批量/复制/二维码/导出 | 未完整保留 | 确认，且不应视为小缺口 | 旧版 `inbounds.html` 有 `addBulkClient`、`copyClients`、`qrcode`、`exportAllLinks`、`exportAllSubs`、`exportSubs`；新 `api/inbounds.ts` 未封装 `copyClients`，新 UI 未出现二维码和全量导出入口。 |
| 高级协议生成器 | 未完整保留 | 确认 | 后端仍有 `getNewX25519Cert`、`getNewEchCert`、VLESS encryption 等生成器；新 UI 当前没有完整接入这些按钮和辅助流程。 |
| Settings 字段和保存路径 | 已保留并增强布局 | 基本确认 | 字段大多保留，保存仍走 `/panel/setting/update`；但安全告警、LDAP tag 多选、旧 2FA modal 流程未完全复刻。 |
| 订阅输出与推荐链接 | 已保留并增强 | 确认，但需 E2E/接口回归补证 | 新增 target-aware 推荐链接和 diagnose 属增强；仍缺订阅二维码。 |
| 旧 UI 回退 | 已显式保留 | 确认，但只能作为迁移保障 | `web/controller/xui.go` 提供 `/panel/legacy/`、`/panel/legacy/inbounds`、`/panel/legacy/settings`、`/panel/legacy/xray`。这不能替代新 UI parity 完成标准。 |
| Core Instances | 新增功能 | 确认，但必须受 phase gate 约束 | 该能力不能提前改变旧 Xray 生命周期语义，当前评审只认可其作为受控新增入口。 |

## 3. 产品经理评审

### 3.1 产品经理质疑

1. “日常核心闭环已覆盖”是否等于“用户可迁移到新 UI”？
   - 不能直接等同。对普通用户，登录、查看状态、日志、基础入站、客户端、Settings 保存已经形成闭环。
   - 对高级运维用户，Xray 结构化编辑器、Reality/ECH/X25519 生成器、批量客户端、二维码和导出订阅属于高频效率能力，缺失会阻断完整迁移。

2. “旧 UI 回退”是否可作为产品完成标准？
   - 不能。旧 UI 回退是灰度和容灾能力，只能降低迁移风险。
   - 若产品目标是“新 UI 替代旧 UI”，回退入口不能抵消新 UI 功能缺口。

3. Settings 的“增强”是否全面成立？
   - 布局和字段组织增强成立。
   - 但安全告警、LDAP 入站标签选择器、2FA 旧弹窗校验流程未完整保留，因此不能笼统称 Settings 全面增强。

4. Inbounds 新表单是否满足“保留全部原有功能逻辑”的用户要求？
   - 基础 CRUD 和 JSON 同步路径满足当前阶段目标。
   - 但批量导入、复制客户端、二维码、导出全部链接/订阅等功能没有新 UI 等价入口，不满足“全部原有功能逻辑”这一最高标准。

### 3.2 产品经理确认意见

- 原报告将功能划分为“已保留并增强、已保留但未增强、未完整保留、新增功能”是合理的。
- 原报告没有误导性宣称“完全等价 V3.0.3”，符合产品风险表达。
- 新 UI 的主要产品价值确认存在：更清晰的表单组织、日志安全展示、移动端适配、订阅推荐链接和诊断能力。

### 3.3 产品经理结论

原报告的产品结论基本成立，但应将“新 UI 可独立替代旧 UI”的状态判定为未达成。当前可作为“新 UI 主路径灰度 + 旧 UI 高级能力回退”的阶段成果，不适合宣称 V3.0.3 parity 完成。

## 4. 开发工程师评审

### 4.1 开发工程师质疑

1. Advanced JSON 是否真的能保证高级字段不丢失？
   - JSON 编辑本身能承载旧结构，但结构化表单的 Apply/同步逻辑只覆盖部分字段。
   - 对 `externalProxy`、`finalmask`、Reality 复杂字段、ECH 等高级结构，需要专门回归验证“编辑普通字段后高级字段仍保留”。

2. Inbounds API SDK 是否完整覆盖旧后端能力？
   - 不完整。`frontend/src/api/endpoints.ts` 和 `frontend/src/api/inbounds.ts` 覆盖了基础 CRUD、client CRUD、reset、online、lastOnline、clientIps。
   - 未封装旧后端仍存在的 `/:id/copyClients`，也未覆盖 `updateClientTraffic/:email`、`delClientByEmail/:email` 等边缘端点。

3. Xray 页面是否只是“JSON 编辑器 + 摘要”？
   - 基本是。新 UI 有 JSON 保存、格式化、复制下载、出站流量、出站测试、Warp/Nord action。
   - 但旧结构化配置编辑器的对象级 CRUD 未迁移，这不是技术不可行，而是迁移范围未覆盖。

4. Core Instances 是否可能破坏 phase gate？
   - 只要保持能力受限、只读/受控生命周期，不改变旧 Xray 默认启动路径，就可接受。
   - 如果后续让 CoreManager 提前接管旧 Xray 写路径，会违反 Phase 10 前门禁。

5. Settings 2FA 流程是否与旧版严格一致？
   - 字段和保存路径基本一致。
   - 旧版 modal 校验交互未完全复刻，新 UI 通过生成 token、展示 setup URI、保存设置完成；这属于流程变化，应补测试和产品确认。

### 4.2 开发工程师确认意见

- 新 UI 大部分写操作仍走旧 API，符合 UI-first 阶段“不改旧数据模型”的要求。
- `/panel/legacy/` 路由存在，符合回退门禁。
- 新 UI 未发现依赖 `v-html` 的日志渲染路径，日志纯文本展示方向正确。
- 订阅 target/diagnose 属后端增强，但应补接口级兼容测试证明 legacy 输出未回归。

### 4.3 开发工程师结论

原报告对技术现状的判断可信。需要补充的工程风险是：Advanced JSON 只能证明“可手工承载旧结构”，不能自动证明“所有表单编辑路径都不会覆盖未建模字段”。后续实现必须以字段保留回归测试和旧 UI 可读性测试作为准入。

## 5. 业务分析师评审

### 5.1 业务分析师质疑

1. 操作入口迁移是否会造成流程割裂？
   - 是。旧 Dashboard 中的 Xray 生命周期、版本安装、备份导入被拆到 Xray 和 Settings Backup。
   - 拆分更符合信息架构，但老用户需要明确导航和迁移提示，否则会误判功能消失。

2. 异常处理是否覆盖导入、保存、重启等危险操作？
   - 新 UI 对重置流量、删除耗尽客户端、备份导入、重启面板、Xray 安装/重启等已有确认弹窗或错误状态。
   - 但缺少 E2E 证据验证失败路径，例如导入非法 DB、保存无效 JSON、订阅路径为空、旧 API 返回错误时的用户提示。

3. 订阅推荐链接是否会改变用户理解模型？
   - 会。新增 target-aware 推荐链接降低配置成本，但也增加“同一订阅入口按客户端输出不同格式”的认知变化。
   - 需要在 QA 中覆盖 target 参数、旧 URL 无 target 参数、JSON/Clash/URI 三类输出一致性。

4. LDAP tag 输入从多选变成字符串是否影响业务流程？
   - 会影响。字段仍保留，但选择器缺失会提升配置错误概率。
   - 对 LDAP 自动同步场景，这是“易用性下降 + 错配风险上升”，不宜只归类为轻微 UI 缺口。

### 5.2 业务分析师确认意见

- 原报告对“已迁移入口”和“未迁移图形化入口”的区分清楚。
- 新 UI 表单分区改善了 Settings 和 Inbounds 的阅读路径，符合表单重组目标。
- 旧 UI 回退能维持短期业务连续性，但需要在新 UI 中暴露清晰入口，避免用户无法发现回退路径。

### 5.3 业务分析师结论

流程闭环在“普通管理路径”上成立，在“高级运维路径”和“批量操作路径”上不完整。建议将业务流程按普通用户、高级协议配置用户、批量客户运营用户、LDAP 管理员四类角色分别建立迁移验收清单。

## 6. QA 测试专家评审

### 6.1 QA 质疑

1. 当前测试是否足以支撑“完整保留”？
   - 不足。现有测试主要覆盖组件结构、字段绑定、JSON sync/apply 和部分提交路径。
   - 缺少真实浏览器 E2E、多视口截图、旧 UI 读写兼容、订阅输出对比、导入失败路径和高级协议字段不丢失回归。

2. 哪些缺口最容易形成线上缺陷？
   - Inbounds 普通字段编辑后覆盖 `streamSettings` 中未图形化字段。
   - Settings 保存后丢失或误写订阅/LDAP/TG 边缘字段。
   - Xray JSON 保存格式正确但业务语义破坏，导致 Xray 重启失败。
   - 订阅 target-aware 输出对部分客户端格式不兼容。
   - 旧 UI 回退路径在 base path、登录态或静态资源路径下不可用。

3. 哪些场景必须补回归？
   - 新 UI 创建/编辑入站后，旧 UI 能打开并再次保存。
   - 旧 UI 创建含 Reality、ECH、externalProxy、finalmask 的入站后，新 UI 编辑基础字段不丢高级字段。
   - `copyClients`、批量客户端、二维码、导出订阅这些未迁移项在新 UI 标注回退或补齐入口。
   - Settings 保存所有 tab 后，订阅 URI/JSON/Clash 输出与保存前预期一致。
   - 备份导入合法、非法扩展名、超大文件、后端错误四类路径。
   - `/panel/legacy/` 在带 base path 的部署下可访问。

4. 视觉与响应式是否已证明？
   - 不能。报告描述了响应式表单样式和统一组件，但本轮未提供截图或录屏证据。
   - 需要至少覆盖桌面、平板、移动端视口下 Inbounds 编辑、客户端 Drawer、Settings 多 tab、Xray JSON 编辑、Logs 长列表。

### 6.2 QA 确认意见

- 原报告的“验证建议”是合理的，明确承认当前只是静态功能逻辑对比。
- 未完整保留功能清单足够具体，可以直接转化为测试矩阵。
- 对旧 UI 回退的存在性判断有代码证据，但仍需浏览器验证。

### 6.3 QA 结论

当前质量门禁不应放行“完全替代旧 UI”。可以放行“新 UI 主流程灰度验证”，条件是补充 E2E、视觉截图、legacy compatibility 测试和高级字段保留测试。未迁移功能需要在测试计划中显式标为“legacy fallback 覆盖”或“新 UI 待补齐”。

## 7. 交叉质疑汇总

| 议题 | 产品经理观点 | 开发工程师观点 | 业务分析师观点 | QA 观点 | 综合裁决 |
| --- | --- | --- | --- | --- | --- |
| 新 UI 是否可替代旧 UI | 不能，缺高级能力 | 不能，API SDK 未覆盖旧能力全集 | 不能，高级流程断点明显 | 不能，缺 E2E 和兼容证据 | 未达成 V3.0.3 完全 parity |
| 旧 UI 回退是否足够 | 可灰度，不可算完成 | 符合 phase gate | 可保障业务连续性 | 需验证 base path 和登录态 | 作为短期保障成立 |
| Settings 是否增强 | 布局增强，安全提示退步 | 字段保存基本保留 | LDAP 字符串输入带来错配风险 | 需全字段保存回归 | 部分增强，不能无条件称全面增强 |
| Inbounds 是否满足全部逻辑 | 主路径满足，高级批量缺失 | 旧端点未完全封装 | 批量运营路径不闭合 | 缺二维码/复制/批量 E2E | 主路径通过，边缘路径不完整 |
| Advanced JSON 是否足够 | 对高级用户门槛高 | 可承载但需防覆盖 | 流程上不等价结构化编辑器 | 必须做不丢字段回归 | 只能作为兼容兜底 |
| 订阅增强是否成立 | 推荐链接有产品价值 | 需兼容测试 | target 逻辑改变用户路径 | 需多格式输出矩阵 | 增强成立，需补测试证据 |
| Core Instances 是否可接受 | 可作为未来能力入口 | 必须遵守 Phase 10 门禁 | 需避免与 Xray 控制混淆 | 需 unsupported lifecycle 测试 | 新增可保留，但不得改变旧默认行为 |

## 8. 最终核实结论

1. 原功能逻辑对比报告的总体结论成立：新 UI 已覆盖并增强 V3.0.3 的日常核心管理能力，但不能宣称完全等价旧版。
2. 原报告列出的主要未完整保留项经复核成立，尤其是 Xray 结构化高级编辑器、Inbounds 批量/复制/二维码/导出、高级协议生成器、Settings 安全告警、LDAP 标签多选和订阅二维码。
3. “已保留并增强”的结论需要分层表述：日志安全、订阅推荐/诊断、表单组织、备份导入校验属于明确增强；Settings 和 Inbounds 只能说“主路径增强”，不能覆盖全部旧功能。
4. `/panel/legacy/` 回退符合 UI-first 阶段门禁，但它是迁移保障，不是 parity 完成证明。
5. Advanced JSON 是重要兼容兜底，但不是旧版结构化编辑器的等价替代；必须通过“不丢高级字段”的回归测试才能作为稳定迁移路径。
6. 当前最合适的产品状态定义是：“新 UI 可进入主路径灰度验证，高级/批量/边缘能力仍依赖 legacy fallback 或后续 parity 任务。”

## 9. 后续优先级建议

P0：进入下一阶段前必须补证

- 新 UI 写入后旧 UI 可读可编辑：Inbounds、Settings、Xray JSON。
- 含 Reality、ECH、externalProxy、finalmask 的入站高级字段不丢失。
- 订阅 URI/JSON/Clash 与 target-aware 输出矩阵回归。
- `/panel/legacy/` 在默认路径和自定义 base path 下可访问。

P1：优先补齐新 UI parity

- Inbounds：二维码、全量链接导出、全量订阅导出、单入站订阅导出、客户端批量新增、复制客户端、入站克隆。
- Xray：Outbounds、Routing、DNS/FakeDNS、Balancers、Reverse 的结构化编辑。
- Settings：安全告警、LDAP 入站标签选择器、订阅二维码。

P2：增强和体验完善

- Reality/ECH/X25519/ML-DSA/ML-KEM/VLESS encryption 生成器。
- Advanced JSON 未覆盖字段提示。
- 新 UI 中提供明确 legacy fallback 快捷入口。
- CPU History 图表迁移或正式降级说明。

## 10. 评审裁决

综合四个角色意见，本次交叉验证裁决为：

- 原报告内容可信，关键差异点定位基本准确。
- 新 UI 当前态可支撑主流程灰度，但不满足“完整替代 V3.0.3”的发布标准。
- 后续若继续推进 UI 替代旧版，必须把未完整保留项转化为可执行 parity backlog，并用测试先行方式逐项关闭。
