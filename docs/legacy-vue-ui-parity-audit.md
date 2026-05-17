# Legacy -> Vue UI 功能对账表

审计日期：2026-05-18

审计范围：

- Legacy UI：`web/html/*.html`、`web/html/modals/*`、`web/html/settings/**`、`web/assets/js/**`
- 新 Vue UI：`frontend/src/router/index.ts`、`frontend/src/views/*`、`frontend/src/api/*`、`frontend/src/utils/*`、`frontend/src/schemas/*`
- 后端入口：`web/ui.go`、`web/controller/*`

审计方式：静态代码对账。本文记录当前源码事实，不以历史计划或旧审计结论为准。

---

## 1. 总体结论

当前 Vue UI 已经从“只读骨架”推进到“默认入口 + 核心 Xray/Inbounds/Settings 工作流可用 + legacy fallback 保留”的状态。

```text
/panel/ 默认新 Vue UI
  -> Dashboard / Logs / Cores / Xray / Inbounds / Settings
  -> 继续使用 legacy API 和 model.Inbound
  -> /panel/legacy/* 保留回退
  -> /panel/ui/* 保留新 UI 兼容入口
```

与 2026-05-13 的旧审计相比，当前最重要变化：

- Inbounds 已补齐行级订阅导出、分享链接导出、QR、批量添加客户端、跨入站复制客户端、克隆入站、单入站/客户端流量重置等能力。
- Xray 已补齐结构化 Outbounds、Routing、DNS、FakeDNS、Balancer、Reverse、Residential IP Pool、AI Routing、Gateway Egress MVP、Protocol Tools、WARP Matrix、DNS/Runtime Policy、Observatory。
- Settings 主配置、订阅配置、格式配置、Telegram、备份恢复继续可用。
- Logs 独立页面承接面板日志与 Xray 日志。
- Core Instances 页面展示 `default-xray` 和 `experimental-sing-box`。

仍需注意：

- 新 UI 是新的工作台，不再逐项复刻 legacy 的所有弹窗位置；部分入口已重组。
- Legacy UI 仍是兼容和回滚边界，不能删除。
- 新 UI 写入必须继续保持 legacy UI 可读。

---

## 2. 路由层对账

### 2.1 Go 托管路由

| 路由 | 状态 | 说明 |
|---|---|---|
| `/panel/` | 已迁移 | 新 UI 默认入口 |
| `/panel/login` | 已迁移 | 新 UI 登录 |
| `/panel/dashboard` | 已迁移 | 跳转到 Dashboard |
| `/panel/logs` | 已迁移 | 日志 |
| `/panel/cores` | 已迁移 | Core 实例 |
| `/panel/xray` | 已迁移 | Xray 配置工作台 |
| `/panel/inbounds` | 已迁移 | Inbounds 管理 |
| `/panel/settings` | 已迁移 | 设置 |
| `/panel/ui/*` | 兼容入口 | 仍指向新 UI |
| `/panel/legacy/*` | 保留 | legacy fallback |

### 2.2 Vue Router

`frontend/src/router/index.ts` 当前页面：

- `login`
- `dashboard`
- `logs`
- `cores`
- `xray`
- `inbounds`
- `settings`
- `not-found`

新 UI 没有把 legacy 每个二级弹窗拆成独立路由，而是在页面内用 Modal、Drawer、Table action 和结构化表单承接。

---

## 3. Dashboard

| Legacy 能力 | 新 UI 状态 | 当前入口 / 说明 |
|---|---|---|
| 查看服务器状态 | 已迁移 | Dashboard 读取 `/panel/api/server/status` |
| 刷新状态 | 已迁移 | Dashboard 顶部刷新 |
| Geo 文件维护 | 已迁移 | Dashboard Geo Maintenance |
| Xray 停止/重启 | 入口变化 | 移至 Xray 页面生命周期区 |
| 面板日志 / Xray 日志 | 入口变化 | 移至 Logs 页面 |
| 数据库备份导入 | 入口变化 | 移至 Settings Backup |
| Xray 版本管理 | 入口变化 | 移至 Xray 页面 |
| CPU History | 部分迁移 | 后端 API 存在；新 UI 是否展示以 Dashboard 当前实现为准 |
| 公网 IP 显示/隐藏 | 需继续关注 | 隐私切换体验与 legacy 不完全等价 |

结论：Dashboard 已承担核心状态面板职责；运维快捷入口被拆分到更明确的专页。

---

## 4. Inbounds

### 4.1 列表与 CRUD

| Legacy 能力 | 新 UI 状态 | 当前实现 |
|---|---|---|
| 添加入站 | 已迁移 | `New Inbound` |
| 编辑入站 | 已迁移 | 行级 `Edit` |
| 删除入站 | 已迁移 | 行级 `Delete` |
| 导入 Inbound JSON | 已迁移 | 顶部 `Import JSON` |
| 克隆入站 | 已迁移 | 行级/菜单克隆，生成新端口和空客户端列表 |
| 重置全部入站流量 | 已迁移 | 顶部菜单 `Reset All Traffic` |
| 重置单入站流量 | 已迁移 | 详情/行级操作 |
| 重置全部客户端流量 | 已迁移 | 顶部菜单 `Reset All Clients` |

### 4.2 客户端管理

| Legacy 能力 | 新 UI 状态 | 当前实现 |
|---|---|---|
| 查看详情 | 已迁移 | `Details` 抽屉 |
| 新增客户端 | 已迁移 | 详情抽屉内添加 |
| 编辑客户端 | 已迁移 | 客户端行级编辑 |
| 删除客户端 | 已迁移 | 客户端行级删除 |
| 重置客户端流量 | 已迁移 | 单个和批量选择均支持 |
| 重置当前入站所有客户端流量 | 已迁移 | 详情操作 |
| 批量添加客户端 | 已迁移 | `Bulk Add` modal，数量限制 1..500 |
| 跨入站复制客户端 | 已迁移 | `copyClients` modal，调用 `/panel/api/inbounds/:id/copyClients` |
| 按 email 删除客户端 | API 存在 | 新 UI 可按当前操作路径调用相关兼容层 |

### 4.3 分享、订阅与 QR

| Legacy 能力 | 新 UI 状态 | 当前实现 |
|---|---|---|
| 当前入站分享链接导出 | 已迁移 | 行级和详情内 `Export Share Links` |
| 当前入站订阅链接导出 | 已迁移 | 行级和详情内 `Export Subscription Links` |
| 全部分享链接导出 | 已迁移 | 顶部菜单 `Export All` |
| 全部订阅链接导出 | 已迁移 | 顶部菜单 `Export All Subscriptions` |
| 客户端分享链接 | 已迁移 | 客户端 Access / Export 区 |
| 客户端订阅链接 | 已迁移 | `buildClientSubscriptionLinks` 使用 `subURI`、可选 JSON/Clash URI |
| 二维码 | 已迁移 | 动态加载 `assets/qrcode/qrious2.min.js`，渲染 canvas |
| Gateway Proxy URI | 新增能力 | 生成本机 HTTP/SOCKS5 代理 URI，供 Super-Code-Gateway 使用 |

结论：Inbounds 当前已覆盖 legacy 高频操作，并新增 Gateway Proxy URI。入口位置与 legacy 不完全相同，但不再是“导出/二维码/批量/复制缺失”的状态。

---

## 5. Xray

### 5.1 生命周期与版本

| Legacy 能力 | 新 UI 状态 | 当前实现 |
|---|---|---|
| 停止 Xray | 已迁移 | `/panel/api/server/stopXrayService` |
| 启动/重启 Xray | 已迁移 | `/panel/api/server/restartXrayService` |
| 查看 Xray result | 已迁移 | `/panel/xray/getXrayResult` |
| 查看版本列表 | 已迁移 | `/panel/api/server/getXrayVersion` |
| 安装指定版本 | 已迁移 | `/panel/api/server/installXray/:version` |
| 查看/保存模板 JSON | 已迁移 | `/panel/xray/`、`/panel/xray/update` |

说明：新 UI 的“启动”语义仍复用 legacy restart endpoint；这不是 CoreManager 生命周期。

### 5.2 结构化配置

| Legacy 能力 | 新 UI 状态 | 当前实现 |
|---|---|---|
| Outbounds 流量统计 | 已迁移 | Outbound Tools |
| Outbound 测试 | 已迁移 | `testOutbound`，测试 URL 服务端保存 |
| Outbound 流量重置 | 已迁移 | 单 tag 与全部 |
| Outbound CRUD | 已迁移 | Structured Outbounds |
| Routing rule CRUD / 排序 | 已迁移 | Routing Rules |
| DNS server 编辑 | 已迁移 | DNS Servers |
| FakeDNS 编辑 | 已迁移 | FakeDNS Pools |
| Balancer CRUD | 已迁移 | Balancers |
| Reverse CRUD | 已迁移 | Reverse |
| DNS Presets | 已迁移/新增 | DNS Presets |
| DNS Policy | 已迁移/新增 | DNS Policy |
| Runtime Policy | 已迁移/新增 | Runtime Policy |
| Observatory / Burst Observatory | 已迁移 | Observatory |
| Residential IP Pool | 已迁移/新增 | socks outbound 模板级编辑 |
| Apply AI Routing | 已迁移/新增 | 生成 `ai-residential` balancer 和 AI 域名规则 |
| WARP Matrix | 已迁移/新增 | 加载 WARP data/config 后应用矩阵 |
| Protocol Tools | 已迁移/新增 | Argo、Xray combo、external-only 输出 |
| Gateway Egress MVP | 新增能力 | 生成 Gateway-facing SOCKS5 inbounds、placeholder outbounds 和 CSV manifest |

结论：Xray 页面已经从“模板编辑器 + 出站统计”升级为主要配置工作台。仍需注意所有结构化能力都是模板 JSON 级写入，保存后需要用户显式重启 Xray。

---

## 6. Settings

| Legacy 能力 | 新 UI 状态 | 当前入口 |
|---|---|---|
| 面板基础设置 | 已迁移 | Settings Panel |
| 修改账号密码 | 已迁移 | Security / Credentials，字段为 old/new username/password |
| 2FA | 已迁移 | Security 区域，体验与 legacy QR 弹窗不完全一致 |
| 订阅开关、路径、URI | 已迁移 | Subscription |
| JSON / Clash 订阅格式设置 | 已迁移 | Formats |
| Telegram Bot | 已迁移 | Telegram |
| LDAP 设置 | 已迁移 | Settings 对应区域 |
| 数据库下载 | 已迁移 | Backup |
| 数据库导入 | 已迁移 | Backup，后端校验文件大小、文件名和扩展名 |
| 面板重启 | 已迁移 | Settings 顶部/操作区 |

结论：Settings 是迁移较完整的页面。剩余差异主要是交互形态，而不是缺少后端能力。

---

## 7. Logs

| Legacy 能力 | 新 UI 状态 | 当前实现 |
|---|---|---|
| 面板日志查看 | 已迁移 | LogsView，`POST /panel/api/server/logs/:count` |
| 面板日志 level/syslog 过滤 | 已迁移 | POST form 字段 |
| Xray 日志查看 | 已迁移 | LogsView，`POST /panel/api/server/xraylogs/:count` |
| Xray 日志 filter/direct/blocked/proxy 过滤 | 已迁移 | POST form 字段 |
| 下载/复制日志 | 已迁移 | LogsView 工具 |
| 安全文本渲染 | 已迁移 | 禁止 `v-html` |

结论：Logs 已从 legacy 首页弹窗迁移成独立工作页。

---

## 8. Core Instances

新 UI 新增 `CoreInstancesView`，legacy UI 无等价页面。

| 实例 | 状态 |
|---|---|
| `default-xray` | 只读 legacy Xray 视图，生命周期不走 CoreManager |
| `experimental-sing-box` | 外部二进制实验实例，支持 validate/start/stop/restart |

注意：该页面不表示多内核生产化已完成；当前仍禁止把 legacy Xray 生命周期迁到 CoreManager。

---

## 9. 订阅服务与导出

新 UI 不直接实现订阅服务器；它通过设置和 Inbounds 导出入口消费现有订阅服务。

当前后端订阅服务支持：

- `/sub/:subid`
- `/json/:subid`，需 `subJsonEnable=true`
- `/clash/:subid`，需 `subClashEnable=true`
- 每种格式的 `:subid/diagnose`
- `/sub/:subid?target=xray|mihomo|stash|...` target-aware 输出
- HTML 订阅页：`Accept: text/html`、`?html=1` 或 `?view=html`

新 UI 的 `mergeSubscriptionEndpointDefaults` 会在公开 URI 缺失时读取默认设置，避免新装环境导出 0 条订阅链接。

---

## 10. 当前缺口与风险

### P0：必须持续守住

| 风险 | 当前要求 |
|---|---|
| 新 UI 写入破坏 legacy UI | 所有 Inbound/Xray/Settings 写入必须保持旧 API 和旧数据模型兼容 |
| CoreManager 越界 | `default-xray` 生命周期不能通过 CoreManager |
| legacy fallback 被误删 | `/panel/legacy/*` 必须可访问 |
| 日志/配置 XSS | 不使用 `v-html`、`innerHTML`、`insertAdjacentHTML` 渲染外部内容 |

### P1：建议继续体验补齐

| 项目 | 说明 |
|---|---|
| Dashboard 辅助能力 | CPU History 展示、IP 隐私切换等体验可继续对齐 legacy |
| 2FA 体验 | 新 UI 可进一步补齐 QR 交互 |
| 订阅交付体验 | 可继续增强平台导入菜单、二维码和诊断入口可见性 |
| Xray 模板变更验证 | 保存前/保存后更明显地区分“已修改模板”和“已重启生效” |

### P2：后续阶段事项

| 项目 | 前置条件 |
|---|---|
| CoreManager 包装 default-xray 生命周期 | 新旧 UI Xray E2E 稳定、legacy 行为一致 |
| sing-box 生产化 | 实验实例稳定，能力 schema 和后端校验完善 |
| 统一订阅模型 | legacy Xray 订阅输出保持不变 |

---

## 11. 推荐回归清单

每次修改新 UI 或 legacy 兼容 API 后，至少覆盖：

1. 登录 `/panel/login`。
2. Dashboard 加载状态。
3. Logs 读取面板日志和 Xray 日志。
4. Inbounds 新增、编辑、克隆、导出分享链接、导出订阅链接、QR、批量添加客户端、复制客户端、重置流量、删除。
5. Xray 加载模板、结构化编辑、保存、重启、查看 result。
6. Settings 保存、更新用户、数据库下载、数据库导入校验失败路径。
7. `/panel/legacy/`、`/panel/legacy/inbounds`、`/panel/legacy/settings`、`/panel/legacy/xray` 可访问。
8. `/panel/api/*` 未登录返回 404。
9. 缺失或错误 CSRF token 的 POST 返回 403。
10. 新 UI 日志/配置预览不执行 HTML。

---

## 12. 结论

当前迁移状态已经不适合继续引用“新 UI 大量按钮缺失”的旧结论。更准确的判断是：

```text
核心日常工作流已迁移
  -> 新 UI 已成为默认入口
  -> legacy UI 保留回退
  -> Xray 和 Inbounds 页面已具备大量结构化增强能力
  -> 后续重点从“补缺”转向“回归稳定、体验一致和阶段门禁”
```
