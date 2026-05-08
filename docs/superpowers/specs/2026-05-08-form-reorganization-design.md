# SuperXray Inbounds 与 Settings 表单重组设计

## 背景

前置 UI 审计已定位新 Vue UI 在表单组织、移动端适配、控件可访问名称、按钮换行、长 JSON 编辑区和设置页信息密度方面存在问题。用户确认采用方案 B：重新分区 Inbounds 与 Settings 表单，统一容器和移动端行为，但不重写业务函数。

本设计只面向 `frontend/` 新 Vue UI。旧 UI、旧 Xray API、旧数据库模型和旧 payload 语义保持不变。

## 已确认范围

- 重组 `frontend/src/views/InboundsView.vue` 的入站编辑弹窗表单结构。
- 重组 `frontend/src/views/SettingsView.vue` 的设置页各 tab 内表单结构。
- 在 `frontend/src/styles/app.css` 增加统一表单容器、分区、动作区、JSON 区和移动端样式。
- 新增一个轻量公共展示组件 `frontend/src/components/FormSection.vue`，只负责标题、说明、动作槽和内容槽，不接收业务对象，不处理提交、校验或转换。
- 增补前端测试，锁定关键字段、关键函数调用路径和响应式类名，防止重排时丢字段或改错流程。

## 非目标

- 不重写 `submitInbound()`、`saveSettings()`、`confirmSave()`、`resetForm()`、`applySubscriptionDefaults()`、`confirmUpdateCredentials()`、`handleDbFileChange()`、`confirmRestartPanel()` 等业务函数。
- 不引入 schema 表单引擎、FormKit、VeeValidate 或新的状态管理抽象。
- 不修改后端接口、API payload、`model.Inbound`、旧数据库结构或旧 UI 可读性。
- 不迁移 Xray 生命周期到 CoreManager，不加入 sing-box 路径。
- 不使用 `v-html` 渲染 JSON、日志、配置预览、订阅文本或用户输入。
- 不在本任务内修复全部 UI 审计问题；本任务只覆盖 Inbounds 和 Settings 表单相关问题。

## 设计原则

- 先稳定行为，再改善视觉：模板和样式可以重排，数据流和函数入口保持原样。
- 分区按用户任务组织，而不是按代码字段顺序堆叠。
- 桌面保留运维控制台的信息密度，移动端优先避免横向溢出和按钮截断。
- 所有输入控件保留或补齐稳定 label / aria 名称，方便测试和辅助技术识别。
- 危险动作和普通保存动作视觉上保持区分，不用主色弱化破坏性行为。

## 公共表单组织规范

### `FormSection` 组件

新增轻量公共组件 `FormSection.vue`，接口固定为：

```vue
<FormSection eyebrow="Network" title="Basic Inbound" description="...">
  <template #actions>
    <!-- Sync / Apply / Format / Copy 等局部动作 -->
  </template>
  <!-- 表单字段 -->
</FormSection>
```

组件职责：

- 输出统一的 section 外框、标题层级、说明文本和动作槽布局。
- 不持有业务状态，不调用 API，不解析 JSON，不执行校验。
- 默认 slot 直接承载现有 `AFormItem`、`AInput`、`ASelect`、`textarea` 等控件。

### 公共样式

在 `app.css` 中统一以下样式层：

- `.form-section`：表单分区容器，使用当前 Fintech Ops Console 的深色面板、边框和 8 到 16px 级圆角。
- `.form-section-header`：标题、说明、动作按钮的响应式布局。
- `.form-grid`：保持现有类名，升级为可复用响应式网格，桌面双列，窄屏单列。
- `.form-grid--three`：仅用于足够宽的短字段组，窄屏回退单列。
- `.form-actions`：局部动作按钮区，允许 wrap，移动端按钮宽度可用。
- `.form-json-stack`：JSON 编辑器垂直分组，确保 textarea 不产生页面横向滚动。
- `.responsive-modal-form`：用于入站弹窗内容，使弹窗宽度、内边距和滚动行为跟随视口。

现有 `.settings-feature-panel`、`.json-section`、`.json-section-title` 可以迁移到新样式语义，或作为兼容别名保留，避免一次性大面积改名造成风险。

## Inbounds 表单设计

入站新增/编辑仍使用现有 `AModal`、`inboundModalOpen`、`inboundModalTitle`、`savingInbound` 和 `@ok="submitInbound"`。弹窗内容改为以下分区：

### Basic Inbound

字段：

- Protocol
- Remark
- Listen
- Port
- Enable
- Traffic Limit GB
- Expiry Timestamp
- Traffic Reset

保留行为：

- 编辑模式下 Protocol 禁用。
- Port 保留 `1..65535` 限制。
- 流量、过期、重置字段继续绑定现有 `inboundEditor`。

### WireGuard Settings

仅在 `inboundEditor.protocol === 'wireguard'` 时显示。

字段和动作保持现有能力：

- MTU
- noKernelTun
- Server Private Key
- Server Public Key
- `syncWireguardEditorFromSettings`
- `applyWireguardEditorToSettings`
- `generateWireguardServerKeys`

### Transport Settings

仅在 `protocolSupportsStream(inboundEditor.protocol)` 时显示。

组织方式：

- 第一行放 Network、Security 等主开关字段。
- TCP、mKCP、WS、gRPC、HTTPUpgrade、XHTTP 相关字段按网络类型局部分组。
- TLS 和 Reality 字段按安全类型局部分组。
- Sockopt 使用独立子分区，只有 `streamEditor.sockoptEnabled` 为真时展开高级字段。
- Hysteria/Hysteria2 字段保留现有协议判断。

保留行为：

- `syncStreamEditorFromSettings`
- `applyStreamEditorToSettings`
- 所有 `v-if` 协议和安全类型规则。
- 所有现有 `streamEditor` 字段名和 `v-model`。

### Default Client

仅在 `inboundClientSectionVisible` 时显示。

字段：

- Email
- UUID / Password / Auth，按现有协议判断显示
- Security / Flow / Method，按现有协议判断显示
- Traffic Limit GB
- Expiry Timestamp
- IP Limit
- Reset Days
- Sub ID
- Enable
- Comment

保留行为：

- `syncInboundClientEditorFromSettings`
- `applyInboundClientEditorToSettings`
- `randomUuid`
- `generateClientCredential`
- `usesUuidClientId`
- `usesPasswordClientId`
- `usesAuthClientId`
- `inboundVlessFlowVisible`
- `buildClientPayloadFromEditor`
- `submitInbound()` 内对 `applyInboundClientEditorToSettings()` 的调用路径

### Advanced JSON

字段：

- Settings JSON
- Stream Settings JSON
- Sniffing JSON

保留行为：

- `formatInboundJson('settings')`
- `formatInboundJson('streamSettings')`
- `formatInboundJson('sniffing')`
- 所有 JSON 文本仍绑定 `inboundEditor.settings`、`inboundEditor.streamSettings`、`inboundEditor.sniffing`
- 现有 JSON normalization 和提交校验路径不改

## Settings 表单设计

设置页保留 `ATabs` 作为一级任务导航，不把所有设置堆到一个长页。每个 tab 内使用统一 `FormSection` 与 `.form-grid`。

### Panel

分区：

- Web Endpoint：Web Listen、Web Domain、Web Port、Base Path
- TLS Files：Certificate File、Key File
- Session and Display：Session Max Age、Page Size、Datepicker、Time Location
- Thresholds and Naming：Expire Diff、Traffic Diff、Remark Model

保留行为：

- 顶部 Refresh、Reset、Save 分别继续调用 `loadSettings`、`resetForm`、`confirmSave`
- `settingsChanged`、`settingsLoaded`、`saving`、`loading` 逻辑不改

### Security

分区：

- Two Factor：Two Factor 开关、Token、Setup URI、Generate Token、Disable Two Factor
- Credentials：Current Username、Current Password、New Username、New Password、Update

保留行为：

- `generateTwoFactorToken`
- `disableTwoFactor`
- `confirmUpdateCredentials`
- `updateCredentials`
- 凭证字段校验仍在现有函数中执行
- 保存 panel settings 后旧设置页仍读取相同 two-factor 字段

### Subscription

分区：

- Feature Flags：URI、JSON、Clash/Mihomo、Encrypted、Show Info、Routing
- Server Endpoint：Title、Updates、Listen、Port、Domain、Path
- Public URIs：URI、JSON Path、JSON URI、Clash Path、Clash URI
- Metadata and TLS：Support URL、Profile URL、Certificate File、Key File
- External Traffic：External Traffic Inform、External Traffic URI
- Public Links：只读链接输出、Copy Links、Open URI、Open JSON、Open Clash
- Recommended Client Links：保留推荐客户端卡片、Copy Recommended、Copy/Open 单项链接
- Announce and Routing：Announce、Routing Rules

保留行为：

- `applySubscriptionDefaults`
- `copySubscriptionLinks`
- `openPublicLink`
- `copyRecommendedLinks`
- `copyRecommendedLink`
- `recommendedSubscriptionLinks`
- `subscriptionPublicLinkText`

### Formats

分区：

- JSON Fragment
- JSON Noises
- JSON Mux
- JSON Rules

保留行为：

- `jsonWarnings` 警告继续显示
- 所有字段继续绑定 `settings.subJsonFragment`、`settings.subJsonNoises`、`settings.subJsonMux`、`settings.subJsonRules`

### Telegram

分区：

- Bot Flags：Enabled、Backup、Login Notify
- Bot Connection：Bot Token、Chat ID、Proxy、API Server
- Runtime Rules：Runtime、CPU Threshold、Language

保留所有 `settings.tg*` 字段。

### LDAP

分区：

- LDAP Flags：Enabled、TLS、Invert Flag、Auto Create、Auto Delete
- Connection：Host、Port、Bind DN、Password、Base DN
- User Mapping：User Filter、User Attr、VLESS Field、Flag Field、Truthy Values
- Sync Defaults：Sync Cron、Inbound Tags、Default Total GB、Default Expiry Days、Default Limit IP

保留所有 `settings.ldap*` 字段。

### Backup

分区：

- Database Backup / Restore：Download、Import、隐藏 file input、导入 warning
- Panel Runtime：Restart

保留行为：

- `downloadDb`
- `openDbFilePicker`
- `handleDbFileChange`
- `confirmRestartPanel`
- `restartPanel`
- 导入数据库继续走旧恢复路径，并保持重启 Xray 的提示

## 响应式设计

- 桌面：表单字段默认双列，短字段可三列；长 JSON、只读输出和说明横跨整行。
- 平板：减少 gap，动作区允许换行，避免按钮挤压标题。
- 手机：所有表单字段单列，section header 垂直排列，动作按钮换行并保持可触控高度。
- 弹窗：入站编辑弹窗使用 `max-width: calc(100vw - 24px)` 和内部滚动，避免 375px 宽度下横向溢出。
- JSON textarea：最小高度按场景区分，宽度固定为父容器 100%，使用等宽字体，长行允许横向滚动在 textarea 内部完成，不推动页面。
- Settings 推荐客户端卡片和公开链接按钮在窄屏改为单列，并避免英文按钮文本截断。

## 可访问性与状态反馈

- 每个 `AFormItem` 保留可见 label；仅当 Ant Design Vue 组件无法稳定暴露名称时补充 `aria-label`。
- 只读 textarea 和隐藏 file input 保留明确 `aria-label`。
- Switch、Checkbox、危险按钮和禁用按钮在深色背景下需要满足可读对比度。
- 局部动作按钮保留 loading、disabled 或危险状态，不改变原有条件。
- 表单分区标题不替代字段 label，避免屏幕阅读器丢失具体输入名称。

## 测试策略

实现阶段必须测试先行：

- 扩展 `frontend/tests/inbounds-view.test.ts`，确认 Inbounds 重组后仍包含 Default Client、JSON sync/apply、credential 生成、`submitInbound()` 调用 `applyInboundClientEditorToSettings()`、所有 JSON editor 与 format 动作。
- 新增 `frontend/tests/settings-view.test.ts`，用源码结构测试锁定 Settings 的全部字段、关键动作函数和各 tab 分区。
- 新增或扩展样式测试，确认公共表单类存在，移动端媒体查询覆盖 `.form-grid`、`.form-section-header`、`.form-actions`、`.responsive-modal-form`。
- 新增 `FormSection.vue` 组件源码测试，确认 props/slots 简单且不引用业务 API。

验证命令：

```powershell
cd frontend
npm run test
npm run typecheck
npm run lint
npm run build
```

浏览器验证：

- 启动 Vite dev server。
- 用 Playwright 或 in-app browser 检查 375、768、1280 宽度。
- 覆盖 Settings 各 tab、Inbounds 新增/编辑弹窗、Default Client、JSON 编辑区、订阅公开链接按钮。
- 截图保存到新的 `tmp/` 审计目录，作为 UI 修复证据。

## 风险与回滚

风险：

- 模板重排时漏掉字段或条件渲染。
- 公共组件引入后 slot 结构影响 Ant Design Vue 表单间距。
- 移动端样式修复影响其他页面沿用的 `.form-grid`。
- Settings tab 内容过多，重排后仍需浏览器实测才能确认没有按钮截断。

缓解：

- 先写源码结构测试，字段和函数路径全部锁定后再改模板。
- `FormSection` 保持展示型组件，不接业务状态。
- CSS 保留旧类兼容，新增类逐步接管，不一次性删除 `.settings-feature-panel` 和 `.json-section`。
- 实现后跑 typecheck、lint、build，并做多视口截图。

回滚：

- 回退 `FormSection.vue`、`InboundsView.vue`、`SettingsView.vue`、`app.css` 和新增测试即可恢复旧表单组织。
- 因为不改后端、数据库和 API payload，不需要数据迁移回滚。

## 完成标准

- Inbounds 和 Settings 表单按本设计完成分区。
- 现有业务函数、字段绑定、校验和提交路径没有被重写。
- 移动端 375px 不出现表单区域横向溢出或按钮截断。
- 所有关键字段保留稳定 label 或 aria 名称。
- 前端测试、typecheck、lint、build 通过。
- 生成新的截图证据，说明已覆盖 Inbounds 与 Settings 的主要表单场景。
