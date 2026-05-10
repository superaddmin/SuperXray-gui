# 新 UI 部署可行性深度分析与裁决报告

日期：2026-05-09

## 1. 结论先行

当前不能执行“正式生产替代部署”，也不能对外宣称“新 UI 已完整替代 V3.0.3”。

原因分两类：

- 功能 parity 层面：新 UI 仍缺 V3.0.3 的高级结构化 Xray 编辑、Inbounds 批量/复制/二维码/导出、高级协议生成器、Settings 安全告警、LDAP 标签多选和订阅二维码等能力。
- 发布门禁层面：本地基础构建和常规测试通过，但 release gate 失败，包含 `staticcheck`、`govulncheck`、`gosec` 阻断项；当前工作区也不是 clean 状态。

可执行的部署范围：

| 部署类型 | 当前裁决 | 条件 |
| --- | --- | --- |
| 本地验证部署 | 可以 | 用于开发自测、截图、E2E 补证。 |
| 受控 staging / 内测部署 | 可以，但必须标注风险 | 环境隔离、数据库备份、保留 `/panel/legacy/`、不承诺完整替代。 |
| 小流量生产灰度 | 暂不建议 | 至少修复 release gate 阻断项，并补最小 E2E/legacy fallback 验证。 |
| 正式生产替代部署 | 不可以 | 必须补齐 P0 parity 与 release gate。 |
| GitHub Release / 标签发布 | 不可以 | release gate 当前失败，且工作区未清洁。 |

最准确的状态定义是：

> 新 UI 当前“可构建、可测试、可进入受控灰度验证”，但“不可作为完整替代 V3.0.3 的正式生产发布”。

## 2. 本轮验证证据

### 2.1 已通过的基础部署证据

前端：

| 命令 | 结果 | 关键输出 |
| --- | --- | --- |
| `npm run test` | 通过 | 65 tests，65 pass，0 fail |
| `npm run typecheck` | 通过 | `vue-tsc -b --noEmit` exit 0 |
| `npm run lint` | 通过 | `eslint .` exit 0 |
| `npm run build` | 通过 | Vite production build exit 0，产物写入 `web/ui/` |

后端：

| 命令 | 结果 | 关键输出 |
| --- | --- | --- |
| `go test ./...` | 通过 | `sub`、`web`、`web/controller` 等包通过 |
| `go vet ./...` | 通过 | exit 0 |
| `go build -o bin/SuperXray.exe ./main.go` | 通过 | exit 0，生成 Windows 二进制 |

安全/源码抽查：

| 检查 | 结果 | 说明 |
| --- | --- | --- |
| `rg "v-html|innerHTML|dangerously|eval\\(|new Function|document\\.write" frontend/src` | 未命中 | 新 UI 源码未发现高风险 HTML 注入写法。 |
| `/panel/legacy/` 路由 | 存在 | `web/controller/xui.go` 保留 legacy index/inbounds/settings/xray。 |
| phase gate 复核 | 部分符合 | 新 UI 主要仍走 legacy API；但 E2E 和发布 gate 未完成。 |

### 2.2 未通过的发布门禁证据

执行：

```powershell
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --allow-dirty
```

结果：失败。

失败项：

| 阻断项 | 证据 | 影响 |
| --- | --- | --- |
| `staticcheck` | `sub/subService.go:1621:6: func getHostFromXFH is unused (U1000)` | 正式发布 gate 不通过。 |
| `govulncheck` | 当前 Go 为 `go1.26.2 windows/amd64`，报告 6 个 reachable Go 标准库漏洞，修复版本为 Go 1.26.3 | 公开生产发布存在安全门禁阻断。 |
| `gosec` | `sub/subService.go:855`，G115，`uint64 -> int` 转换 | 安全扫描阻断，需修复或带理由抑制。 |
| 工作区状态 | release gate 报告 `dirty worktree allowed by --allow-dirty`；`git status` 显示大量修改/新增/删除 | 不能执行正式 release/tag 发布。 |

这些不是 UI parity 缺口，而是“发布/部署质量 gate”缺口。即使产品接受功能不完整，也不能忽略这些 gate 去做正式发布。

## 3. “不能完整替代 V3.0.3”的细颗粒影响分析

### 3.1 Xray 结构化高级编辑器

| 维度 | 分析 |
| --- | --- |
| 旧版能力 | Routing、Outbounds、Reverse、Balancers、DNS、FakeDNS、Protocol Tools、规则增删改排序。 |
| 新 UI 状态 | 主要提供 JSON 编辑、摘要、保存、格式化、出站流量、出站测试。 |
| 用户影响 | 高级运维用户无法在新 UI 中完成对象级配置维护。 |
| 数据风险 | JSON 手工编辑可承载旧配置，但无法证明普通编辑路径不会覆盖未建模高级字段。 |
| 部署影响 | 阻断“完整替代部署”；不阻断“带 legacy fallback 的受控灰度”。 |
| 必须补证 | 旧 UI 创建复杂 Xray 配置后，新 UI 保存 JSON，再由旧 UI 读取和 Xray 重启验证。 |

裁决：完整替代 P0 阻断。

### 3.2 Inbounds 批量、复制、二维码、导出

| 维度 | 分析 |
| --- | --- |
| 旧版能力 | 入站克隆、二维码、全量分享链接、全量订阅导出、单入站订阅导出、批量客户端、新旧入站复制客户端。 |
| 新 UI 状态 | 保留基础 CRUD、单客户端管理、选中客户端导出链接、重置/删除耗尽客户端；未完整迁移上述批量/二维码能力。 |
| 用户影响 | 对运营型用户、批量开通用户、移动端扫码导入用户影响明显。 |
| 数据风险 | 后端 `copyClients` 仍存在但前端未封装，用户可能误以为能力被删除。 |
| 部署影响 | 阻断完整替代；受控灰度必须明确提示这些操作走 `/panel/legacy/inbounds`。 |
| 必须补证 | 新 UI 创建/编辑入站后旧 UI 可打开；未迁移操作在 legacy 路径可完成。 |

裁决：完整替代 P0 阻断，灰度需产品提示和回退入口。

### 3.3 高级协议生成器与复杂 stream 辅助

| 维度 | 分析 |
| --- | --- |
| 旧版能力 | X25519、ML-DSA-65、ML-KEM-768、ECH、VLESS encryption、Reality target preset、Short IDs、FinalMask、External Proxy。 |
| 新 UI 状态 | Transport 表单覆盖常用字段，Advanced JSON 可手工编辑，生成器未完整接入。 |
| 用户影响 | 高级协议配置效率下降，手工 JSON 出错概率上升。 |
| 数据风险 | `externalProxy`、`finalmask`、ECH 等未建模结构必须验证不被同步逻辑清空。 |
| 部署影响 | 阻断高级协议用户完整迁移。 |
| 必须补证 | 对 Reality/ECH/externalProxy/finalmask 入站执行“旧 UI 创建 -> 新 UI 修改普通字段 -> 旧 UI 再读 -> 订阅输出”链路。 |

裁决：完整替代 P0 阻断；基础灰度可接受但需限定用户群。

### 3.4 Settings 安全告警、LDAP tag 多选、订阅二维码

| 维度 | 分析 |
| --- | --- |
| 旧版能力 | 默认危险配置告警、LDAP 入站标签多选、订阅页二维码。 |
| 新 UI 状态 | 字段保存路径保留，辅助交互未完整保留。 |
| 用户影响 | 安全提示弱化、LDAP 配置易错、扫码订阅路径缺失。 |
| 数据风险 | 手写 `ldapInboundTags` CSV 易出现 tag 拼写或分隔错误。 |
| 部署影响 | 不一定阻断受控灰度，但阻断“体验不低于 V3.0.3”的正式替代。 |
| 必须补证 | Settings 全字段保存回归、LDAP 同步配置回归、订阅 URI/JSON/Clash 输出矩阵。 |

裁决：完整替代 P1/P0 边界项；如果目标客户使用 LDAP，应升为 P0。

### 3.5 CPU History 图表

| 维度 | 分析 |
| --- | --- |
| 旧版能力 | Dashboard CPU History modal 和 `/panel/api/server/cpuHistory/:bucket` 可视化。 |
| 新 UI 状态 | 展示即时 CPU 和 load，未迁移历史图。 |
| 用户影响 | 对性能观测用户有影响，但不直接影响配置写入。 |
| 部署影响 | 不阻断受控灰度；若 parity 定义包含所有 Dashboard 观察能力，则阻断完整替代。 |

裁决：需要产品明确是补迁移还是正式降级。

## 4. 部署风险矩阵

| 风险域 | 风险等级 | 当前状态 | 部署前最低要求 |
| --- | --- | --- | --- |
| 构建可用性 | 低 | 前端 build 和 Go build 已通过 | 保留当前构建产物，部署前重新跑一次。 |
| 单元/源码测试 | 中低 | 前端 65 tests 和 Go tests 通过 | 保持通过。 |
| Release gate | 高 | 失败：staticcheck、govulncheck、gosec | 正式部署前必须修复。 |
| 功能 parity | 高 | 未完整替代 V3.0.3 | 只能灰度，不能移除旧 UI。 |
| 数据兼容 | 高 | 静态分析认为大多走旧 API，但缺 E2E | 补新 UI 写入后旧 UI 可读测试。 |
| 高级字段保留 | 高 | Advanced JSON 兜底，但未充分验证不丢字段 | 补 externalProxy/finalmask/Reality/ECH 回归。 |
| 订阅兼容 | 中高 | 新增 target/diagnose，但缺逐格式对比 | 补 URI/JSON/Clash/Mihomo 输出矩阵。 |
| 安全表现 | 高 | 源码无 `v-html`，但 govulncheck/gosec 失败 | 修复 release gate，补 CSP/CSRF/XSS E2E。 |
| 回退能力 | 中 | `/panel/legacy/` 路由存在 | 浏览器验证默认 base path 和自定义 base path。 |
| 视觉响应式 | 中 | CSS/test 有覆盖，但缺截图证据 | 补桌面/移动端截图或 E2E 视觉记录。 |

## 5. 部署裁决分层

### 5.1 本地/开发部署

裁决：可以。

理由：

- 前端构建通过。
- Go 构建通过。
- 常规测试和 lint/typecheck 通过。

用途：

- 继续做浏览器 E2E。
- 采集截图和录屏。
- 验证 legacy fallback 和 base path。

### 5.2 Staging / 内测部署

裁决：可以，但必须受控。

前提：

- 部署环境隔离，不暴露为稳定正式版本。
- 明确标注“新 UI 灰度验证，不等于完整替代 V3.0.3”。
- 保留 `/panel/legacy/`。
- 部署前备份数据库。
- 高级用户和批量运营用户知道缺口和回退入口。

建议准入清单：

- 登录、Dashboard、Logs、Xray JSON、Inbounds 基础 CRUD、Settings 保存、备份下载/导入至少跑一轮浏览器 smoke。
- 验证 `/panel/legacy/`、`/panel/legacy/inbounds`、`/panel/legacy/settings`、`/panel/legacy/xray` 可访问。
- 验证无 base path 时和自定义 base path 时静态资源加载正常。

### 5.3 小流量生产灰度

裁决：当前暂不建议。

阻断原因：

- release gate 失败，尤其是 `govulncheck` 和 `gosec`。
- 缺少 E2E、视觉、多视口、legacy compatibility 证据。
- 未迁移能力对高级用户影响较大。

放行条件：

- 修复 `staticcheck` unused function。
- 升级或切换到修复 Go 标准库漏洞的 Go 版本，并重跑 `govulncheck`。
- 修复或合理抑制 `gosec` G115。
- 完成最小 E2E：登录、Inbounds 创建/编辑、Settings 保存、Xray JSON 保存、legacy fallback、订阅输出。

### 5.4 正式生产替代部署

裁决：不可以。

硬要求：

- release gate 全绿。
- 新 UI parity backlog 的 P0 项关闭。
- 完成“新 UI 写入后旧 UI 可读可编辑”的验证。
- 完成高级字段不丢失验证。
- 完成订阅输出一致性验证。
- 完成视觉/响应式截图证据。
- 明确旧 UI 移除时间表；在 Phase 9/10 前不得移除旧 UI。

## 6. 发布阻断项修复清单

P0：发布 gate 阻断

1. `staticcheck`：处理 `sub/subService.go:1621` 未使用的 `getHostFromXFH`。
2. `gosec`：修复 `sub/subService.go:855` 的 `uint64 -> int` 转换，先判断范围再转换。
3. `govulncheck`：当前 Go 为 `go1.26.2`，标准库漏洞修复版本为 `go1.26.3`，需要升级构建工具链或使用修复版本重跑。
4. clean worktree：提交或清理当前变更后再执行正式 release gate。

P0：灰度前验证阻断

1. 跑浏览器 smoke，覆盖新 UI 主路径。
2. 验证 `/panel/legacy/` 回退路径。
3. 验证带 base path 的静态资源和 legacy 路由。
4. 对真实配置或脱敏配置执行数据库备份和恢复演练。

P1：完整替代阻断

1. 迁移 Xray 结构化高级编辑器核心模块。
2. 补 Inbounds 批量、复制、二维码、导出。
3. 补高级协议生成器。
4. 恢复 Settings 安全告警、LDAP tag 多选、订阅二维码。

## 7. 部署执行建议

如果目标是“今天先看新 UI 是否能跑起来”：

- 可以执行本地或 staging 部署。
- 不要对外宣布替代旧版。
- 部署说明写明：高级能力请走 `/panel/legacy/`。

如果目标是“发布给真实生产用户作为默认 UI”：

- 当前不建议执行。
- 先修 release gate，再跑 E2E 和 legacy compatibility。

如果目标是“发布 3.0.4/正式 tag”：

- 当前不可以执行。
- `release_gate.py --allow-dirty` 都失败，正式发布更不能通过。

## 8. 最终裁决

最终裁决：

- 可以执行：本地验证部署、受控 staging / 内测部署。
- 暂不建议执行：小流量生产灰度。
- 不可以执行：正式生产替代部署、GitHub Release / 标签发布。

当前最优下一步不是继续讨论“能不能完整替代”，而是先关闭两条线：

1. 发布 gate 线：修复 `staticcheck`、`govulncheck`、`gosec`。
2. parity 证据线：补 E2E、legacy fallback、旧 UI 可读、高级字段不丢失和订阅输出一致性。
