# Legacy -> Vue UI 功能对账表

历史审计日期：2026-05-18；当前源码复核日期：2026-10-15。

> **当前边界**：Legacy HTML UI 已退役。`web/html`、`web/assets` 已移除，`/panel/legacy/*` 不再注册（默认 `NoRoute` 返回 `404 Not Found`），`TestXUIControllerDoesNotRegisterLegacyPanelRoutes` 守护此边界。本文保留历史功能清单作为对账背景，当前状态只依据现存源码；不代表重新运行了旧版本或本轮完成了部署环境 E2E。

审计范围：

- Legacy UI：历史审计所用的 `web/html`、`web/assets` 路径；当前已删除，本轮没有把这些历史路径当作现存证据
- 新 Vue UI：`frontend/src/router/index.ts`、`frontend/src/views/*`、`frontend/src/api/*`、`frontend/src/utils/*`、`frontend/src/schemas/*`
- 后端入口：`web/ui.go`、`web/controller/*`

复核方式：将历史清单与当前 Vue Router、页面、SDK、兼容工具、Go 路由及回归测试逐项比对。部署可用性和外部 Xray/订阅客户端行为仍需目标环境回归。

此前 v3.10.2 集成后的隔离 Docker E2E 19/19 通过，详细环境与用例范围见 [集成验证记录](upstream-3x-ui-v3.10.2-integration.md)。本次文档核对不重写该记录，也不将功能测试等同于视觉/无障碍或全部生产客户端验收。以下面板路径均相对于配置的 `webBasePath`。

---

## 1. 总体结论

当前 Vue UI 已经从“只读骨架”推进到“默认入口 + 核心 Xray/Inbounds/Settings 工作流可用 + legacy HTML UI 已退役”的状态。

```text
/panel/ 默认新 Vue UI
  -> Dashboard / Logs / Cores / Xray / Inbounds / Settings / API Docs
  -> 继续使用 legacy API 和 model.Inbound
  -> /panel/ui/* 保留新 UI 兼容入口
  -> /panel/legacy/* 已退役，路由不再注册
```

与 2026-05-13 的旧审计相比，当前最重要变化：

- Inbounds 已补齐行级订阅导出、分享链接导出、QR、批量添加客户端、跨入站复制客户端、克隆入站、单入站/客户端流量重置等能力。
- Xray 已补齐结构化 Outbounds、Routing、DNS、FakeDNS、Balancer、Reverse、Residential IP Pool、AI Routing、Gateway Egress MVP、Protocol Tools、WARP Matrix、DNS/Runtime Policy、Observatory。
- Settings 主配置、订阅配置、格式配置、Telegram、备份恢复继续可用。
- TOTP 启用、禁用与替换加入验证码确认；入站和两类客户端编辑器保留未修改配额的原始字节数。
- Xray v26.10.10 的 TLS/Reality、XHTTP、UDP Hop、xDNS 和 WireGuard 兼容由共享转换、服务端校验及订阅能力判断共同保护，不表示所有组合都可运行。
- Logs 独立页面承接面板日志与 Xray 日志。
- Core Instances 页面展示 `default-xray` 和 `experimental-sing-box`。

仍需注意：

- 新 UI 是新的工作台，不再逐项复刻 legacy 的所有弹窗位置；部分入口已重组。
- Legacy HTML UI 已退役，`web/html`、`web/assets` 和 `/panel/legacy/*` 路由不再保留。
- 新 UI 写入必须继续保持旧 API 和 `model.Inbound` 兼容数据可读。

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
| `/panel/docs` | 新增页面 | API Docs，读取已登录 `/panel/api/openapi.json` |
| `/panel/ui/*` | 兼容入口 | 仍指向新 UI |
| `/panel/legacy/*` | 已退役 | 路由不再注册，返回 `404 Not Found` |

### 2.2 Vue Router

`frontend/src/router/index.ts` 当前页面：

- `login`
- `dashboard`
- `logs`
- `cores`
- `xray`
- `inbounds`
- `settings`
- `docs`
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
| CPU History | 后端能力保留 | 后端 `/cpuHistory/:bucket` 存在；当前 Dashboard 展示实时 CPU，未消费该历史 API |
| 公网 IP 显示/隐藏 | 展示已实现，切换缺失 | 当前 Dashboard 状态明细显示 IPv4/IPv6，未实现独立隐藏开关 |

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
| 入站配额 | 兼容保护 | 未修改额度时保留原始字节数；显式修改按 GiB 转回字节，显式设零为无限，避免只修改备注把小配额变为零 |

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
| 按 email 删除客户端 | API 保留 | `/panel/api/inbounds/:id/delClientByEmail/:email` 存在；当前 SDK 删除使用 client ID 路径 |
| 客户端配额 | 兼容保护 | 客户端编辑器与入站内默认客户端编辑器共用 `trafficQuota.ts`；未修改时保留原始额度 |

WireGuard peer 不使用普通客户端的配额/IP 限制/过期字段。共享编辑器虽显示 Comment，当前 WireGuard payload 构造会移除 `comment`，不能据此宣称支持 per-peer 备注保存。

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

### 4.4 传输、安全与旧配置

| 配置范围 | 当前编辑/兼容边界 |
|---|---|
| TLS | 表单支持证书文件、内联证书/私钥、自签证书生成、SNI、ALPN、TLS 版本和 ECH；旧 pin 数组由后端转为字符串。目标 Xray 已删除的 `allowInsecure: true` 返回明确错误，需管理员配置可信证书或 pin |
| Reality | 表单保留 target、serverNames、密钥、shortIds、版本与 ML-DSA-65 字段；读取兼容旧 `dest`，后端写为 `target`；非空 TCP finalmask 与 Reality 组合拒绝保存/迁移 |
| XHTTP / XMUX | 兼容根字段与 `extra`，旧 `sessionPlacement/sessionKey` 转为 `sessionIDPlacement/sessionIDKey`，保留 table/length 和未知扩展；服务端运行配置剔除客户端专属投影，持久化与订阅保留 |
| Hysteria / UDP Hop / xDNS | 缺 Hysteria version 补 `2`；旧 UDP Hop 与 xDNS 结构受控转换，冲突、缺失字段和不支持值返回错误；Salamander 密码至少 4 个 UTF-8 字节 |
| WireGuard / TUN | 保留本地表单和导出；WireGuard 旧 workers/DNS/策略由兼容层处理，TUN 继续支持已移植的旧别名。平台、设备权限和目标客户端能力仍须单独验证 |

源码入口为 `frontend/src/utils/inboundCompat.ts`、`frontend/src/utils/realitySettings.ts`、`util/xraycompat/` 与 `sub/`。未知字段保留不代表目标核心接受；缺失 TLS/Reality 材料不会由迁移自动生成替换。完整转换和人工处理边界见 [v3.10.2 迁移规则](upstream-3x-ui-v3.10.2-integration.md)。

---

## 5. Xray

### 5.1 生命周期与版本

| Legacy 能力 | 新 UI 状态 | 当前实现 |
|---|---|---|
| 停止 Xray | 已迁移 | `/panel/api/server/stopXrayService` |
| 启动/重启 Xray | 已迁移 | `/panel/api/server/restartXrayService` |
| 查看 Xray result | 已迁移 | `/panel/api/xray/getXrayResult` |
| 查看版本列表 | 已迁移 | `/panel/api/server/getXrayVersion` |
| 安装指定版本 | 已迁移并加强校验 | `/panel/api/server/installXray/:version`；仅接受 v26.10.10 及以上稳定版本，核对 SHA-256、候选核心/配置，失败尝试恢复旧核心 |
| 查看/保存模板 JSON | 已迁移 | `/panel/api/xray/`、`/panel/api/xray/update`，兼容 `/panel/xray/*` 仍保留且 POST 同样需要 CSRF |

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

结论：Xray 页面已经从“模板编辑器 + 出站统计”升级为主要配置工作台。所有结构化能力都是模板 JSON 级编辑；保存前服务端归一兼容字段，将模板与已启用入站合成为完整候选配置并调用目标核心预检，校验失败不写入模板。保存成功后页面询问是否重启，需要用户确认才重启；核心预检不验证端口可绑定或外部网络可达。

---

## 6. Settings

| Legacy 能力 | 新 UI 状态 | 当前入口 |
|---|---|---|
| 面板基础设置 | 已迁移 | Settings Panel |
| 修改账号密码 | 已迁移 | Security / Credentials，字段为 old/new username/password |
| 2FA | 已迁移并加强确认 | Security 区域提供开关、新密钥、Setup URI 与验证码；首次启用校验新认证器，禁用/替换校验旧认证器，空密钥保留旧值、禁用成功清除旧值；不回显已保存密钥，当前无 QR 弹窗 |
| 订阅开关、路径、URI | 已迁移 | Subscription |
| JSON / Clash 订阅格式设置 | 已迁移 | Formats |
| Telegram Bot | 已迁移 | Telegram |
| LDAP 设置 | 已迁移 | Settings 对应区域 |
| 数据库下载 | 已迁移 | Backup |
| 数据库导入 | 已迁移并加强恢复 | Backup；后端校验上传和 SQLite，在独立候选库完成事务迁移与完整 Xray 配置预检后才切换；失败保留或恢复原库，恢复失败返回备份路径 |
| 面板重启 | 已迁移 | Settings 顶部/操作区 |

结论：Settings 是迁移较完整的页面。剩余差异主要是交互形态，而不是缺少后端能力。

TOTP 确认字段 `twoFactorCode` 只在请求中使用，不持久化；验证失败使整次设置更新失败。当前确认验证只接受当前时间窗口，需保持服务器与认证器时间同步。

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
- WireGuard Peer 的配置文本、HTTP/Mixed 的代理账户输出

当前 `sub/subController.go` 不包含 HTML 订阅渲染，`Accept: text/html`、`?html=1`、`?view=html` 不切换格式。源码默认订阅开启、JSON 关闭、Clash 开启；入口独立于面板 `webBasePath`。

新 UI 的 `mergeSubscriptionEndpointDefaults` 会在公开 URI 缺失时读取默认设置，避免新装环境导出 0 条订阅链接。

---

## 10. 当前缺口与风险

### P0：必须持续守住

| 风险 | 当前要求 |
|---|---|
| 新 UI 写入破坏旧 API 兼容 | 所有 Inbound/Xray/Settings 写入必须保持旧 API 和旧数据模型兼容 |
| CoreManager 越界 | `default-xray` 生命周期不能通过 CoreManager |
| Legacy UI 被误恢复 | `/panel/legacy/*` 必须保持退役，不得重新注册路由或恢复 `web/html`、`web/assets` |
| 日志/配置 XSS | 不使用 `v-html`、`innerHTML`、`insertAdjacentHTML` 渲染外部内容 |

### P1：建议继续体验补齐

| 项目 | 说明 |
|---|---|
| Dashboard 辅助能力 | CPU History 展示、IP 隐私切换等体验可继续对齐 legacy |
| 2FA 体验 | 新 UI 可进一步补齐 QR 交互 |
| 订阅交付体验 | 可继续增强平台导入菜单、二维码和诊断入口可见性 |
| WireGuard peer 备注 | 共享表单有 Comment，但 WireGuard payload 移除该字段；后续需明确支持契约并对齐控件与保存行为 |
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

1. 登录 `/panel/login`，启用 TOTP 时覆盖验证码路径。
2. Dashboard 加载状态。
3. Logs 读取面板日志和 Xray 日志。
4. Inbounds 新增、编辑、克隆、导出分享链接、导出订阅链接、QR、批量添加客户端、复制客户端、重置流量、删除；检查小配额未修改时字节数不变、显式设零可生效。
5. Xray 加载旧 TLS/Reality/XHTTP/UDP Hop 配置、结构化编辑、保存、重启、查看 result；冲突或 schema 错误不覆盖原模板和可用核心。
6. Settings 保存、更新用户、TOTP 启用/禁用/替换验证与整次更新失败路径、数据库下载、候选迁移/预检失败保留原库及重复导入。
7. `/panel/legacy/`、`/panel/legacy/inbounds`、`/panel/legacy/settings`、`/panel/legacy/xray` 返回 `404 Not Found`，确认 Legacy UI 保持退役（由 `TestXUIControllerDoesNotRegisterLegacyPanelRoutes` 守护）。
8. `/panel/api/*` 未登录返回 404。
9. 缺失或错误 CSRF token 的 POST 返回 403。
10. 新 UI 日志/配置预览不执行 HTML。

---

## 12. 结论

当前迁移状态已经不适合继续引用“新 UI 大量按钮缺失”的旧结论。更准确的判断是：

```text
核心日常工作流已迁移
  -> 新 UI 已成为默认入口
  -> legacy HTML UI 已正式退役，路由不再注册
  -> Xray 和 Inbounds 页面已具备大量结构化增强能力
  -> 后续重点从“补缺”转向“回归稳定、体验一致和阶段门禁”
```
