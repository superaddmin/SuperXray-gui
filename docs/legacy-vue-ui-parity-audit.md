# Legacy -> Vue UI 功能对账表

审计日期：2026-05-13
审计范围：`web/html/*.html` 旧版后台页面、`web/html/modals/*` 旧版弹窗、`frontend/src/views/*` 新版 Vue 视图、`frontend/src/router/index.ts` 新版路由。
审计方式：静态代码对账，不依赖运行时截图；重点核对页面入口、按钮、二级/三级弹窗流程、订阅与导出链路、Xray 配置工具链。

## 审计结论

当前 Vue UI 迁移状态不是“按钮随机消失”，而是更接近下面这个模式：

```text
一级页面骨架已迁移
  -> 常用基础操作大多已迁移
  -> 部分入口被藏深或改名
  -> 多个二级/三级弹窗流程未完整迁移
  -> Xray 可视化编辑能力迁移不足最明显
```

本次对账将状态分为四类：

- `已迁移`：功能和入口都在，语义基本一致
- `入口变化`：功能还在，但入口更深、改名或路径变化
- `功能退化`：有替代，但能力明显变弱
- `缺失`：新 Vue UI 中没有等价入口或流程

## 路由层总览

新版 Vue 路由当前只有：

- `dashboard`
- `logs`
- `cores`
- `xray`
- `inbounds`
- `settings`

证据：[router/index.ts](../frontend/src/router/index.ts:7)

这意味着旧版依赖子页、弹窗、工具页承载的能力，如果没有显式迁入以上六个页面，就天然处于高风险状态。

## 一、Dashboard / 首页

### 对账表

| Legacy 页面/入口 | Legacy 能力 | Vue 页面/入口 | 状态 | 说明 | 证据 |
|---|---|---|---|---|---|
| `index.html` 首页动作区 | 刷新状态 | `DashboardView` 顶部 `Refresh` | 已迁移 | 主入口存在 | [index.html](../web/html/index.html:171), [DashboardView.vue](../frontend/src/views/DashboardView.vue:8) |
| `index.html` 首页动作区 | Xray 重启 / 停止 | `XrayView` 生命周期区 | 入口变化 | 从首页移到 `Xray` 页 | [index.html](../web/html/index.html:171), [XrayView.vue](../frontend/src/views/XrayView.vue:50) |
| `index.html` 首页动作区 | 打开 Xray 日志 / 面板日志 | `LogsView` | 入口变化 | 独立日志页替代首页快捷入口 | [index.html](../web/html/index.html:179), [index.html](../web/html/index.html:387), [LogsView.vue](../frontend/src/views/LogsView.vue:5) |
| `index.html` 首页动作区 | 打开配置 `config.json` | 无直接等价快捷入口 | 功能退化 | 新版只有 `Xray Template Editor`，缺首页快捷查看 | [index.html](../web/html/index.html:242), [XrayView.vue](../frontend/src/views/XrayView.vue:85) |
| `index.html` 首页动作区 | 备份弹窗快捷入口 | `Settings -> Backup` | 入口变化 | 功能仍在，但首页快捷入口消失 | [index.html](../web/html/index.html:246), [SettingsView.vue](../frontend/src/views/SettingsView.vue:544) |
| `index.html` 版本弹窗 | 切换 Xray 版本 | `XrayView` 版本区 | 入口变化 | 从首页弹窗迁到专页 | [index.html](../web/html/index.html:303), [XrayView.vue](../frontend/src/views/XrayView.vue:61) |
| `index.html` Geofiles 区 | 更新 geosite/geoip、自定义 Geo 资源 | `DashboardView` Geo Maintenance | 已迁移 | 这条线迁得比较完整 | [index.html](../web/html/index.html:317), [DashboardView.vue](../frontend/src/views/DashboardView.vue:41) |
| `index.html` CPU History | CPU 历史弹窗 | 无 | 缺失 | 新仪表盘无 CPU 历史查看 | [index.html](../web/html/index.html:89), [index.html](../web/html/index.html:513) |
| `index.html` IP 可见性切换 | 隐藏/显示公网 IP | 无 | 缺失 | 新版直接展示 Public IP，无隐私切换 | [index.html](../web/html/index.html:214), [DashboardView.vue](../frontend/src/views/DashboardView.vue:42) |

### 结论

- `Dashboard` 主体数据和 Geo 维护迁移较完整。
- 首页的“运维快捷入口”大量被拆散到 `Xray / Logs / Settings`，属于可接受的入口重组。
- 真正缺的是 `CPU History` 和 `IP 可见性切换` 这类首页辅助能力。

## 二、Inbounds / 入站管理

### 对账表

| Legacy 页面/入口 | Legacy 能力 | Vue 页面/入口 | 状态 | 说明 | 证据 |
|---|---|---|---|---|---|
| `inbounds.html` 主工具栏 | 添加入站 | `InboundsView` 顶部 `New Inbound` | 已迁移 | 基础入口存在 | [inbounds.html](../web/html/inbounds.html:130), [InboundsView.vue](../frontend/src/views/InboundsView.vue:29) |
| `inbounds.html` 通用操作 | 导入入站 JSON | `InboundsView` 顶部 `Import JSON` | 已迁移 | 基础入口存在 | [inbounds.html](../web/html/inbounds.html:142), [InboundsView.vue](../frontend/src/views/InboundsView.vue:16) |
| `inbounds.html` 通用操作 | 重置全部入站流量 | `InboundsView` 顶部 `Reset All Traffic` | 已迁移 | 基础入口存在 | [inbounds.html](../web/html/inbounds.html:155), [InboundsView.vue](../frontend/src/views/InboundsView.vue:20) |
| `inbounds.html` 行级菜单 | 编辑 / 删除 | 表格行按钮 `Edit / Delete` | 已迁移 | 基础 CRUD 在 | [inbounds.html](../web/html/inbounds.html:261), [InboundsView.vue](../frontend/src/views/InboundsView.vue:144) |
| `inbounds.html` 行级菜单 | `showInfo` 客户端详情 | `Details` 抽屉 | 功能退化 | 有详情抽屉，但不是旧版客户端深度信息弹窗 | [inbounds.html](../web/html/inbounds.html:265), [InboundsView.vue](../frontend/src/views/InboundsView.vue:140) |
| `inbounds.html` 行级菜单 | `qrcode` 二维码弹窗 | 无 | 缺失 | 新版没有二维码入口 | [inbounds.html](../web/html/inbounds.html:269), [qrcode_modal.html](../web/html/modals/qrcode_modal.html:2) |
| `inbounds.html` 行级菜单 | `addBulkClient` 批量添加客户端 | 无 | 缺失 | 新版只有单个 `Add Client` | [inbounds.html](../web/html/inbounds.html:279), [InboundsView.vue](../frontend/src/views/InboundsView.vue:293) |
| `inbounds.html` 行级菜单 | `copyClients` 从其他入站复制客户端 | 无 | 缺失 | 新版无对应入口 | [inbounds.html](../web/html/inbounds.html:283), [inbounds.html](../web/html/inbounds.html:903) |
| `inbounds.html` 行级菜单 | `clone` 克隆入站 | 无 | 缺失 | 新版没有克隆动作 | [inbounds.html](../web/html/inbounds.html:312), [inbounds.html](../web/html/inbounds.html:1480) |
| `inbounds.html` 行级菜单 | `clipboard` 导出当前入站原始数据 | 无 | 缺失 | 新版没有“导出/复制当前入站 JSON”按钮 | [inbounds.html](../web/html/inbounds.html:304), [inbounds.html](../web/html/inbounds.html:2082) |
| `inbounds.html` 行级菜单 | `resetTraffic` 当前入站流量重置 | 无 | 缺失 | 新版仅有全局重置，没有单入站重置 | [inbounds.html](../web/html/inbounds.html:308), [InboundsView.vue](../frontend/src/views/InboundsView.vue:20) |
| `inbounds.html` 通用操作 | 导出全部分享链接 | 无 | 缺失 | 新版只有抽屉内选中客户端导出 | [inbounds.html](../web/html/inbounds.html:146), [inbounds.html](../web/html/inbounds.html:1422), [InboundsView.vue](../frontend/src/views/InboundsView.vue:253) |
| `inbounds.html` 通用操作 | 导出全部订阅链接 | 无 | 缺失 | 新版没有全局订阅导出 | [inbounds.html](../web/html/inbounds.html:150), [inbounds.html](../web/html/inbounds.html:1425) |
| `inbounds.html` 行级菜单 | 导出当前入站分享链接 | 抽屉内 `Export Links` | 入口变化 + 功能退化 | 按钮还在，但藏到 `Details -> Clients`，且语义更像 share links | [inbounds.html](../web/html/inbounds.html:291), [InboundsView.vue](../frontend/src/views/InboundsView.vue:258) |
| `inbounds.html` 行级菜单 | 导出当前入站订阅链接 | 无 | 缺失 | 新版没有等价的 `subs` 导出动作 | [inbounds.html](../web/html/inbounds.html:295), [inbounds.html](../web/html/inbounds.html:1462) |
| `inbound_info_modal` | 客户端详情内展示订阅 URL / JSON URL | 无 | 缺失 | 新版详情抽屉不展示这条信息链 | [inbound_info_modal.html](../web/html/modals/inbound_info_modal.html:321), [inbound_info_modal.html](../web/html/modals/inbound_info_modal.html:654) |

### 根因判断

`Export Links` 没“显示”的根因不是条件判断，而是入口迁深：

- 先点 `Details`
- 再进入抽屉内 `Clients`
- 再选客户端
- 才能点击 `Export Links`

证据：

- 行级入口是 `Details` [InboundsView.vue](../frontend/src/views/InboundsView.vue:140)
- `Export Links` 在抽屉内部 [InboundsView.vue](../frontend/src/views/InboundsView.vue:253)
- 数据源是抽屉里的 `selectedClientRows` [InboundsView.vue](../frontend/src/views/InboundsView.vue:1473)

### 结论

`Inbounds` 是本次迁移里缺口最密集的一页之一：

- 基础 CRUD 在
- 单客户端编辑/删除/重置在
- 二级和三级流程缺很多
- “订阅 / 二维码 / 批量 / 克隆 / 跨入站复制 / 原始导出”几乎都没迁全

## 三、Xray / 模板与出站工具

### 对账表

| Legacy 页面/入口 | Legacy 能力 | Vue 页面/入口 | 状态 | 说明 | 证据 |
|---|---|---|---|---|---|
| `xray.html` 顶部 | 启动 / 停止 / 重启 | `XrayView` 生命周期区 | 已迁移 | 基础运行控制在 | [xray.html](../web/html/xray.html:77), [XrayView.vue](../frontend/src/views/XrayView.vue:50) |
| `xray.html` 版本工具 | 安装 Xray 版本 | `XrayView` 版本区 | 已迁移 | 版本切换在 | [index.html](../web/html/index.html:1186), [XrayView.vue](../frontend/src/views/XrayView.vue:61) |
| `xray.html` 高级模板 | 原始 JSON 编辑 | `XrayView` Template Editor | 已迁移 | 有原始 JSON 编辑 | [xray.html](../web/html/xray.html:129), [XrayView.vue](../frontend/src/views/XrayView.vue:85) |
| `xray.html` Outbounds 页 | 出站流量统计 | `XrayView` Outbound Tools 表格 | 已迁移 | 统计和重置还在 | [outbounds.html](../web/html/settings/xray/outbounds.html:18), [XrayView.vue](../frontend/src/views/XrayView.vue:166) |
| `xray.html` Outbounds 页 | `addOutbound` 添加出站 | 无 | 缺失 | 新版无出站新增按钮 | [xray.html](../web/html/xray.html:604), [outbounds.html](../web/html/settings/xray/outbounds.html:6) |
| `xray.html` Outbounds 页 | `editOutbound` 编辑出站 | 无 | 缺失 | 新版无出站编辑入口 | [xray.html](../web/html/xray.html:620), [outbounds.html](../web/html/settings/xray/outbounds.html:49) |
| `xray.html` Outbounds 页 | `deleteOutbound` 删除出站 | 无 | 缺失 | 新版无删除入口 | [outbounds.html](../web/html/settings/xray/outbounds.html:59), [XrayView.vue](../frontend/src/views/XrayView.vue:175) |
| `xray.html` Outbounds 页 | `setFirstOutbound` 调整主出站顺序 | 无 | 缺失 | 新版无排序入口 | [outbounds.html](../web/html/settings/xray/outbounds.html:45) |
| `xray.html` Providers | Warp / Nord 数据拉取 | `XrayView` Providers 区 | 功能退化 | 仅保留数据读取，缺真正编辑/应用流程 | [xray.html](../web/html/xray.html:1173), [XrayView.vue](../frontend/src/views/XrayView.vue:189) |
| `xray.html` Warp Modal | `WARP Matrix`、应用路由矩阵 | 无新 UI 等价入口 | 缺失 | Vue 页只有 provider data，没有矩阵应用 UI | [warp_modal.html](../web/html/modals/warp_modal.html:91), [warp_modal.html](../web/html/modals/warp_modal.html:255) |
| `xray.html` Routing 页 | 规则可视化增删改 | 无 | 缺失 | 新版只有 section summary，没有规则编辑器 | [xray.html](../web/html/xray.html:84), [xray.html](../web/html/xray.html:1039), [XrayView.vue](../frontend/src/views/XrayView.vue:357) |
| `xray.html` Reverse 页 | 反向代理配置增删改 | 无 | 缺失 | 路由不存在等价页 | [xray.html](../web/html/xray.html:98), [xray.html](../web/html/xray.html:709) |
| `xray.html` Balancer 页 | balancer 增删改 | 无 | 缺失 | 路由不存在等价页 | [xray.html](../web/html/xray.html:106), [xray.html](../web/html/xray.html:820) |
| `xray.html` DNS 页 | DNS server / FakeDNS 可视化编辑 | 无 | 缺失 | 新版只有高级 JSON 统计，没有 DNS/FakeDNS 工具 | [xray.html](../web/html/xray.html:114), [xray.html](../web/html/xray.html:979), [xray.html](../web/html/xray.html:1009) |
| `xray.html` Protocol Tools 页 | 协议工具页 | 无 | 缺失 | 新版没有 `Protocol Tools` 对应页签 | [xray.html](../web/html/xray.html:122) |

### 结论

`XrayView` 当前更像“运行时 + 原始模板编辑器 + 出站统计面板”，而不是旧 UI 那种完整的 Xray 配置工作台。
如果按功能风险排序，这一页的缺口严重程度不低于 `Inbounds`。

## 四、Settings / 面板设置

### 对账表

| Legacy 页面/入口 | Legacy 能力 | Vue 页面/入口 | 状态 | 说明 | 证据 |
|---|---|---|---|---|---|
| `settings.html` | 保存 / 重启面板 | `SettingsView` 顶部按钮 | 已迁移 | 主入口在 | [settings.html](../web/html/settings.html:32), [SettingsView.vue](../frontend/src/views/SettingsView.vue:8) |
| `settings.html` 面板设置 | 面板通用字段 | `Panel` tab | 已迁移 | 基础项在 | [settings.html](../web/html/settings.html:54), [SettingsView.vue](../frontend/src/views/SettingsView.vue:21) |
| `settings.html` 安全设置 | 修改账号密码 | `Security -> Credentials` | 已迁移 | 功能在 | [settings.html](../web/html/settings.html:61), [SettingsView.vue](../frontend/src/views/SettingsView.vue:102) |
| `settings.html` Telegram | TG Bot 设置 | `Telegram` tab | 已迁移 | 功能在 | [settings.html](../web/html/settings.html:68), [SettingsView.vue](../frontend/src/views/SettingsView.vue:393) |
| `settings.html` 订阅设置 | 订阅开关、URI、JSON、Clash | `Subscription` tab | 已迁移 | 主配置项迁得比较完整 | [settings.html](../web/html/settings.html:75), [SettingsView.vue](../frontend/src/views/SettingsView.vue:189) |
| `settings.html` 订阅格式 | JSON fragment / mux / rules | `Formats` tab | 已迁移 | 功能在 | [settings.html](../web/html/settings.html:82), [SettingsView.vue](../frontend/src/views/SettingsView.vue:374) |
| `settings.html` 2FA 弹窗 | 图形化 QR 配置与校验 | 仅 `Setup URI` 文本 | 功能退化 | 新版能生成 token，但缺二维码交互体验 | [settings.html](../web/html/settings.html:353), [two_factor_modal.html](../web/html/modals/two_factor_modal.html:10), [SettingsView.vue](../frontend/src/views/SettingsView.vue:124) |
| 订阅子页 | 订阅页面、QR、客户端一键导入菜单 | `Subscription` tab 里的公开链接与推荐链接 | 功能退化 | 有 URI 文本，但缺独立订阅子页和 QR/导入链路 | [subpage.html](../web/html/settings/panel/subscription/subpage.html:273), [SettingsView.vue](../frontend/src/views/SettingsView.vue:298), [SettingsView.vue](../frontend/src/views/SettingsView.vue:326) |
| 订阅子页 | JSON / Clash 订阅二维码 | 无 | 缺失 | 新版只复制链接，不生成订阅二维码 | [subscription.js](../web/assets/js/subscription.js:105) |
| 订阅子页 | Android/iOS 客户端导入深链菜单 | 无 | 缺失 | 新版只生成 target 参数链接，不提供旧版平台菜单 | [subpage.html](../web/html/settings/panel/subscription/subpage.html:225), [subpage.html](../web/html/settings/panel/subscription/subpage.html:248) |
| 备份页 | 数据库导出 / 导入 | `Backup` tab | 已迁移 | 主流程在 | [index.html](../web/html/index.html:493), [SettingsView.vue](../frontend/src/views/SettingsView.vue:544) |

### 结论

`SettingsView` 是迁移最完整的一页之一。
真正的问题不在“主配置项”，而在“订阅展示与交付体验”：

- 公开 URI 在
- 推荐链接在
- 但旧的订阅子页、二维码、平台导入菜单没有被带过来

## 五、Logs / 日志

### 对账表

| Legacy 页面/入口 | Legacy 能力 | Vue 页面/入口 | 状态 | 说明 | 证据 |
|---|---|---|---|---|---|
| 首页日志弹窗 | 面板日志查看、下载 | `LogsView` | 已迁移 | 功能聚合到独立日志页 | [index.html](../web/html/index.html:387), [LogsView.vue](../frontend/src/views/LogsView.vue:5) |
| 首页 Xray 日志弹窗 | Xray 访问日志过滤、下载 | `LogsView` | 已迁移 | 独立日志页承接 | [index.html](../web/html/index.html:434), [LogsView.vue](../frontend/src/views/LogsView.vue:116) |

### 结论

`Logs` 这条线整体没有明显缺口，更多是入口从首页弹窗改成了专页。

## 六、优先级修复队列

### P0：直接影响核心配置闭环

1. `Xray` 可视化编辑缺口
   包括：`Outbounds CRUD`、`Routing`、`DNS/FakeDNS`、`Balancer`、`Reverse`、`Protocol Tools`、`WARP Matrix`

2. `Inbounds` 深层操作缺口
   包括：`批量添加客户端`、`跨入站复制客户端`、`克隆入站`、`二维码`、`当前入站订阅导出`

### P1：高频使用但可暂时绕过

1. `Export Links` 入口提升
   目前用户容易误以为功能消失，应从抽屉深层挪回更明显位置

2. 订阅交付体验补齐
   包括：订阅子页、QR、平台导入菜单、每客户端订阅详情

3. 单入站流量重置、原始 JSON 导出恢复

### P2：辅助与体验型能力

1. Dashboard `CPU History`
2. Dashboard `IP 可见性切换`
3. 首页快捷入口恢复或替代

## 七、建议补齐顺序

推荐我们按下面顺序逐项补：

1. `Inbounds`
   因为它最直接影响“按钮不见了”的主诉，也最贴近日常操作。

2. `Xray`
   因为这里是结构性缺口，旧 UI 工作台能力掉得最多。

3. `Subscription delivery`
   把 `Settings + 订阅子页 + 二维码 + 导入菜单` 串成完整链路。

4. `Dashboard`
   最后补首页级快捷能力和辅助工具。

## 八、下一步建议

下一轮建议直接从 `Inbounds` 开始做 P0/P1 修复拆单：

1. 恢复更明显的 `导出分享链接 / 导出订阅链接` 入口
2. 增加 `二维码` 和 `客户端详情订阅链接`
3. 增加 `批量添加客户端`
4. 增加 `跨入站复制客户端`
5. 增加 `克隆入站`

这样能最快把“功能像没了”的主观体验拉回来。
