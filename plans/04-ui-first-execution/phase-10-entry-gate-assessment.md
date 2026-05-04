# Phase 10.1-10.5 多内核准入门禁评估

## 结论

当前已进入 Phase 9 安全收口推进段，新 UI `style-src` 已通过 nonce bootstrap 收紧，本机隔离真实 Xray core E2E 已完成非跳过验收。按原门禁，CI/第二环境复刻尚未补齐，仍 **不满足 Phase 10.1-10.5 的常规代码实施准入条件**。

2026-05-04 风险接受/强制进入更新：因项目上线部署需要，产品和技术侧接受上述残余风险，强制进入 active CoreManager/sing-box 最小实施段。该决策不等同于 Phase 9 全部闭环，也不允许扩大为完整多内核数据迁移；本阶段只允许新增可回滚的 CoreManager API 层和 `experimental-sing-box` 外部进程适配入口。

2026-05-04 更新：Phase 10.1 `default-xray` 只读实例 ADR 已落地，见 [`phase-10a-default-xray-readonly-adr.md`](phase-10a-default-xray-readonly-adr.md)。该 ADR 只裁定虚拟只读实例语义，不代表 CoreManager 或多内核代码已准入。

2026-05-04 E2E 环境更新：本地 `.env` 已替换占位值，隔离 `xray-bin` 已补齐 `Xray 26.3.27`，`npm run e2e` 在 mutation、restart、DB import、subscription 开关全开时 19 passed / 0 skipped / 0 failed。

## 当前已满足条件

- 新 Vue UI 已成为本地隔离环境默认 `/panel/` 入口。
- `/panel/legacy/` 回退入口可用。
- `/panel/ui/` 兼容入口可用。
- Phase 7a Settings/Subscription/Backup 基础流可通过旧 API 保存设置并下载 SQLite 备份。
- 新 UI 日志和配置渲染路径未发现 `v-html`。
- 旧 UI 模板未发现 `v-html`、`innerHTML`、`insertAdjacentHTML`。
- `/panel/setting/*` 与 `/panel/xray/*` POST 已接入 CSRF 校验。
- 未登录数据库下载和非法数据库导入已有 E2E 回归。
- 新 UI `script-src` 不包含 `unsafe-inline` 和 `unsafe-eval`。
- 新 UI `style-src` 不包含 `unsafe-inline`，Ant Design Vue 动态 `<style>` 已通过 bootstrap 自动附加 nonce。
- 新 UI `style-src-attr` 已收紧为 `'none'`，本地浏览器控制台无 CSP error/warning。
- `go test ./...`、`go vet ./...`、`go build -o bin/SuperXray.exe ./main.go` 通过。
- 前端 `typecheck`、`lint`、`format`、`build` 通过。
- Phase 10.1 `default-xray` 虚拟只读实例 ADR 已定义。
- 本机隔离真实 Xray core E2E 已通过：19 passed / 0 skipped / 0 failed。
- `SUPERXRAY_E2E_RESTART=1`、`SUPERXRAY_E2E_MUTATION=1`、`SUPERXRAY_E2E_IMPORT_DB=1` 与 `SUPERXRAY_E2E_SUB_URL` 均已非跳过运行。
- 新旧 UI 兼容自动抽检已覆盖：新 Vue UI 创建 VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard 六类禁用入站后，Legacy UI 与旧入站列表 API 可读取。
- Settings 兼容自动抽检已覆盖：新 Vue UI 保存订阅标题和公告后，Legacy Settings UI 可读取。
- 编辑兼容自动抽检已覆盖：新 Vue UI 编辑 VLESS 入站、VLESS 客户端和 WireGuard peer 后，Legacy UI 可读取入站行和 VLESS 客户端展开行，旧入站列表 API 可读取关键字段。
- 扩展客户端编辑兼容自动抽检已覆盖：新 Vue UI 编辑 Trojan、Shadowsocks、Hysteria2 客户端后，Legacy UI 展开行与旧入站列表 API 可读取关键字段。
- 在线/IP 管理入口已覆盖：新 UI Inbounds 页展示 Online Clients、Refresh Activity，并在详情抽屉内提供 Online / IP Management、View IPs、Clear IPs 控制；E2E 会创建临时禁用 VLESS 入站后验收并清理。
- 订阅输出矩阵已补 Go 层回归：VMess、VLESS、Trojan、Shadowsocks、Hysteria2 覆盖订阅链接、JSON outbound、Clash/mihomo proxy；WireGuard 覆盖配置文本、JSON outbound、Clash/mihomo proxy。
- 2026-05-04 已按“Legacy 隔离 + 新 UI 清理”执行 UI 框架迁移收口：新 UI 未使用文件、未使用导出和未使用依赖已通过 `knip`、`depcheck` 清零，CodeMirror 依赖已移除，`web/html` 与 `/panel/legacy` 仅作为回退和兼容验收边界保留。

## 已风险接受的门禁缺口

- Phase 9 安全收口尚未完全完成，但本次强制进入接受残余风险：
  - CSRF 错误 token、更多下载鉴权和上传/导入边界仍可继续扩展。
  - Legacy UI 仍需要宽 CSP 才能作为回退入口运行。
- CI 或其他测试机尚未复刻本机隔离 E2E 环境，本次强制进入接受残余风险：
  - 需要有效 `.env`。
  - 需要隔离 DB/bin/log 目录。
  - 需要真实 `xray-windows-amd64.exe`、`geoip.dat`、`geosite.dat`。
- 旧 UI 对新 UI 创建数据已完成六类主力协议自动抽检，Legacy Settings 对新 UI 保存订阅标题/公告也已完成自动抽检；入站基础字段、VLESS/Trojan/Shadowsocks/Hysteria2 客户端和 WireGuard peer 编辑后旧 UI/API 可读已完成自动抽检。订阅输出矩阵已补 Go 层回归；CI/第二环境复刻仍需补齐。

## 强制进入的实施边界

本次强制进入只允许实施以下最小 active 后端能力：

- `/panel/api/cores/*` 后端接口。
- CoreManager 内存注册表和实例聚合服务。
- `default-xray` 虚拟实例，只观察现有 Xray 状态，不接管旧 Xray 生命周期。
- `experimental-sing-box` 实验实例，只在本机存在 sing-box binary 和显式配置文件时允许 `validate/start/stop/restart`。
- sing-box 启停使用外部二进制和配置文件，不生成生产配置、不写入旧 Xray 入站表。

仍需遵守：

- 旧 Xray 的 `/panel/api/server/*` 生命周期端点保持原行为。
- 不迁移旧 `model.Inbound`。
- 不创建 `proxy_inbounds`、`proxy_clients` 写入路径。
- 不修改旧 Xray 订阅输出语义。
- 不新增 Capability Schema 驱动写入表单。
- 新 UI 和旧 UI 的既有 Xray 等价路径必须继续可用。

## 强制进入已实施入口

后端已新增最小 active CoreManager/sing-box 入口：

- `core/`：内存 `CoreManager`、实例契约、状态契约和生命周期结果。
- `core/singbox/`：外部 sing-box binary 实验适配器。
- `web/service/core_service.go`：注册 `default-xray` 和 `experimental-sing-box`。
- `web/controller/core.go`：挂载 `/panel/api/cores/*`。

API 契约：

```text
GET  /panel/api/cores/instances
GET  /panel/api/cores/instances/:id
GET  /panel/api/cores/instances/:id/status
POST /panel/api/cores/instances/:id/validate
POST /panel/api/cores/instances/:id/start
POST /panel/api/cores/instances/:id/stop
POST /panel/api/cores/instances/:id/restart
```

实例语义：

- `default-xray`：只读观察现有 Xray 状态，`start/stop/restart/validate` 经 CoreManager 调用时返回不支持。
- `experimental-sing-box`：默认从 `XUI_BIN_FOLDER` 查找 `sing-box.exe` 或 `sing-box`，默认配置为 `sing-box-config.json`；缺 binary 时状态为 `not-installed`，缺 config 时状态为 `not-configured`。

可选环境变量：

```powershell
$env:SUPERXRAY_SING_BOX_BINARY = 'F:\SuperXray-gui\bin\sing-box.exe'
$env:SUPERXRAY_SING_BOX_CONFIG = 'F:\SuperXray-gui\bin\sing-box-config.json'
$env:SUPERXRAY_SING_BOX_LOG_FOLDER = 'F:\SuperXray-gui\log'
```

安全边界：

- 所有接口挂在既有 `/panel/api` 下，继续复用登录鉴权和 CSRF 中间件。
- sing-box 启动使用 `exec.Command` 参数数组，不经过 shell 拼接。
- 当前不生成 sing-box 配置、不写入数据库、不改变旧订阅输出。

## Phase 10.1-10.5 仍禁止事项

即使已风险接受强制进入，以下事项仍禁止：

- 创建 active `proxy_inbounds` 或 `proxy_clients` 写入路径。
- 迁移旧 `model.Inbound`。
- 用 CoreManager 接管现有 Xray 启停重启。
- 引入 Capability Schema 驱动的新写入表单。
- 修改旧 Xray 订阅输出语义。
- 在 UI 中把 sing-box 作为默认生产核心暴露给普通用户。

## UI 框架迁移收口记录

2026-05-04 按方案 A 执行第一轮新 UI 框架收口：

- 清理范围：`frontend/src`、`frontend/package.json`、`frontend/package-lock.json`、`web/ui` 构建产物。
- 删除项：未使用的 API barrel、subscription wrapper、session store、subscription type 和旧 `env.d.ts`。
- 收窄项：当前视图未调用的旧 API wrapper 和仅模块内部使用的工具类型/函数不再对外导出。
- 依赖项：移除 `@codemirror/lang-json` 与 `codemirror`，生产构建不再包含 CodeMirror 相关 chunk。
- 交互项：移除无效主题切换状态，保留与当前深色视觉体系一致的主题指示器。
- 可访问性项：主导航、筛选控件、配置编辑器和默认 Settings 表单已补充显式可读名称。
- 边界项：`/panel/legacy`、`web/html` 和 `web/assets` 不视为死代码，继续作为 Phase 10 回退入口和新旧 UI 兼容验收基线。
- 取证报告：见 [`.reports/dead-code-analysis.md`](../../.reports/dead-code-analysis.md)。

本轮验证：

```powershell
cd frontend
npm run format
npm run lint
npm run typecheck
npm run build
npm exec --yes knip -- --reporter compact
npm exec --yes depcheck -- --json
npm run e2e
```

本机新二进制截图式响应式检查已通过：Dashboard、Inbounds、Settings、Xray、Logs 在 1440x960 与 390x844 下均无横向溢出、无控制台错误、无可聚焦控件可读名称缺口。

## 下一步动作

1. 已落地最小 active CoreManager/sing-box 后端入口：
   - CoreManager 注册 `default-xray` 和 `experimental-sing-box`。
   - API 使用现有 `/panel/api` 登录鉴权和 CSRF 中间件。
   - sing-box 缺 binary 或缺 config 时返回明确不可启动错误。
2. 完成 Phase 9 剩余安全收口：
   - 补 CSRF 错误 token 回归。
   - 扩展更多下载鉴权回归。
   - 在 CI 或第二台测试机复刻本机隔离 Xray core E2E。
3. 在真实 Xray core 隔离实例运行：
   - 默认 E2E。
   - `SUPERXRAY_E2E_RESTART=1`。
   - `SUPERXRAY_E2E_MUTATION=1`。
   - `SUPERXRAY_E2E_IMPORT_DB=1`。
   - `SUPERXRAY_E2E_SUB_URL`。
4. 扩展旧 UI 兼容抽检：
   - 已完成：`/panel/legacy/inbounds` 可读取新 UI 创建的六类主力协议禁用入站。
   - 已完成：`/panel/legacy/settings` 可读取新 UI 保存的订阅标题和公告。
   - 已完成：Legacy UI 可读取新 UI 编辑后的 VLESS 入站行和 VLESS 客户端展开行，旧入站列表 API 可读取 VLESS 入站/客户端和 WireGuard peer 的编辑修改。
   - 已完成：Legacy UI 展开行和旧入站列表 API 可读取新 UI 对 Trojan、Shadowsocks、Hysteria2 客户端的编辑修改。
   - 待扩展：更多设置项，以及真实订阅 URL 在第二环境/CI 的非跳过复刻。
5. 为强制进入补充回滚记录：
   - 移除 `/panel/api/cores/*` 路由即可关闭新入口。
   - 如 sing-box 已启动，先调用 stop 或终止对应进程。
   - 因本阶段不做数据库迁移，不需要数据库回滚。
