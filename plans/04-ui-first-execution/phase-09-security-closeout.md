# UI First Phase 9 - 安全收口推进记录

## 目标

在 Phase 8 默认入口灰度通过后，继续收紧新旧 UI 的 XSS、CSRF、下载鉴权和数据库导入边界。Phase 9 仍然只服务现有 Xray 等价迁移，不进入 CoreManager、sing-box 或新数据模型。

## 本轮完成项

- 旧 UI HTML sink 收口：
  - 旧 Dashboard 日志区域保持 Vue 文本插值渲染。
  - WireGuard 配置预览从 `v-html` 改为 `<pre>` 文本渲染。
  - 入站帮助弹窗从 `innerHTML` 改为 Vue VNode 文本节点渲染。
  - `web/html` 与 `frontend/src` 未发现 `v-html`、`innerHTML`、`insertAdjacentHTML`。
- CSRF 收口：
  - `/panel/api/*` 继续使用 CSRF 中间件。
  - `/panel/setting/*` POST 已接入 CSRF 中间件。
  - `/panel/xray/*` POST 已接入 CSRF 中间件。
  - 单独伪造 `X-Requested-With: XMLHttpRequest` 不再被视为可信。
  - 登录成功后生成 session 级 CSRF token，并注入新 UI runtime config 与 Legacy UI 全局 axios header。
  - 受保护的状态变更请求必须携带有效 `X-CSRF-Token`；错误 token、缺失 token、跨 Origin 或 scheme mismatch 均拒绝。
- 下载与导入安全：
  - 未登录访问 `/panel/api/server/getDb` 继续返回 404，隐藏 API 存在性。
  - 数据库导入在控制器层新增文件名、扩展名、最小大小和最大大小校验。
  - 仅允许 `.db`、`.sqlite`、`.sqlite3` 数据库文件名。
  - 无效导入不再由控制器额外触发 Xray 重启。
  - 服务层仍保留 SQLite magic header 与 integrity check。
- E2E 回归：
  - 新增 Phase 9 用例：CSRF 缺失、CSRF 错误 token、未登录下载、非法 DB 导入。
  - 新增显式门禁用例：`SUPERXRAY_E2E_IMPORT_DB=1` 时下载当前备份并通过 `importDB` 回灌，验证数据库导入成功路径。
  - 新 UI CSP 回归新增 `style-src` nonce 与 `style-src-attr 'none'` 断言。
  - 新增新旧 UI 兼容矩阵用例：新 Vue UI 创建 VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard 六类禁用入站后，Legacy UI 与旧 `/panel/api/inbounds/list` 可读取，测试结束清理。
  - 新增 Settings 兼容用例：新 Vue UI 保存订阅标题和公告后，Legacy Settings UI 可在 Subscription/Information 面板读取，测试结束恢复原值。
  - 新增编辑兼容矩阵用例：旧 API 造隔离数据后，通过新 Vue UI 编辑 VLESS 入站、VLESS 客户端和 WireGuard peer，再验证 Legacy UI 入站行、VLESS 客户端展开行与旧入站列表 API 均可读取编辑后的字段。
  - 新增扩展客户端编辑兼容用例：通过新 Vue UI 编辑 Trojan、Shadowsocks、Hysteria2 客户端后，Legacy UI 展开行与旧入站列表 API 均可读取编辑后的邮箱、凭据、备注、subId 和启用状态。
  - 本地隔离真实 Xray core E2E：13 个用例，13 passed，0 skipped。

## CSP 当前状态

- 新 UI：
  - `script-src` 不包含 `unsafe-inline`。
  - `script-src` 不包含 `unsafe-eval`。
  - runtime config 继续通过 nonce 注入。
  - `style-src` 不包含 `unsafe-inline`，改为 `style-src 'self' 'nonce-...'`。
  - 运行时 bootstrap 会在 Ant Design Vue CSS-in-JS 创建 `<style>` 前自动附加 CSP nonce。
  - `style-src-attr` 已收紧为 `'none'`。
- Legacy UI：
  - 作为回退入口继续保留 `script-src 'unsafe-inline' 'unsafe-eval'`。
  - 旧 UI 宽 CSP 不传递给新 UI 默认入口。
  - Legacy UI 的状态变更请求已通过页面注入的 CSRF token 进入同一校验路径。

## 验证命令

```powershell
node --test web\assets\js\model\inbound_form_help.test.js
go test ./web ./web/controller ./web/middleware
go test ./...
go vet ./...
cd frontend
npm run typecheck
npm run lint
npm run format
npm run build
cd ..
go build -o bin/SuperXray.exe ./main.go
rg 'v-html|innerHTML|insertAdjacentHTML' web\html frontend\src -n
rg 'v-html' web\html frontend\src web\ui -n
rg 'unsafe-inline|unsafe-eval' frontend\src web\ui -n
rg 'proxy_inbounds|proxy_clients|CoreManager|sing-box|Capability Schema' frontend\src web\ui web\controller web\middleware web\service database\model -n
$env:SUPERXRAY_E2E_BASE_URL = 'http://127.0.0.1:2073/phase7a/'
$env:SUPERXRAY_E2E_USERNAME = 'phase7a-admin'
$env:SUPERXRAY_E2E_PASSWORD = 'phase7a-local-pass'
$env:SUPERXRAY_E2E_MUTATION = '1'
$env:SUPERXRAY_E2E_RESTART = '1'
$env:SUPERXRAY_E2E_IMPORT_DB = '1'
$env:SUPERXRAY_E2E_SUB_URL = 'http://127.0.0.1:2096/sub/'
npm run e2e
```

## 本地验收结果

- 2026-05-04 本轮补充验证：
  - E2E 环境根因已解决：`.env` 占位值已替换为本地隔离实例，`tmp/phase7a-runtime/xray-bin/` 已补齐 Xray core。
  - 当前隔离 Xray core：`Xray 26.3.27`，可正常启动和重启。
  - `go test ./web/middleware ./web/controller ./web` 通过。
  - `go test ./...`、`go vet ./...`、`go build -o bin/SuperXray.exe ./main.go` 通过。
  - `frontend` 下 `npm run typecheck`、`npm run lint`、`npm run format`、`npm run build` 通过，并重新生成 `web/ui`。
  - `npm run e2e` 在本地隔离真实 Xray core 环境通过：13 passed / 0 skipped / 0 failed。
  - 已覆盖 `SUPERXRAY_E2E_MUTATION=1`、`SUPERXRAY_E2E_RESTART=1`、`SUPERXRAY_E2E_IMPORT_DB=1` 和 `SUPERXRAY_E2E_SUB_URL` 非跳过路径。
  - 新旧 UI 兼容自动抽检已覆盖：新 Vue UI 创建六类主力协议禁用入站后，Legacy UI 和旧入站列表 API 均可读取同一 `remark` 与 `protocol`。
  - Settings 兼容自动抽检已覆盖：新 Vue UI 保存订阅标题和公告后，Legacy Settings UI 可读取实际表单值。
  - 编辑兼容自动抽检已覆盖：新 Vue UI 编辑 VLESS 入站、VLESS 客户端和 WireGuard peer 后，Legacy UI 可读取入站行和 VLESS 客户端展开行，旧入站列表 API 可读取编辑后的 `remark`、`listen`、客户端邮箱/备注/subId/启用状态、peer allowedIPs/keepAlive/subId/启用状态。
  - 扩展客户端编辑兼容自动抽检已覆盖：新 Vue UI 编辑 Trojan、Shadowsocks、Hysteria2 客户端后，Legacy UI 展开行和旧入站列表 API 可读取编辑后的邮箱、凭据、备注、subId 和启用状态。
  - Phase 9 E2E 新增下载/读取鉴权边界：未登录访问 `getDb`、`getConfigJson`、`logs`、`xraylogs`、`custom-geo/download` 均返回 404。
  - Phase 9 E2E 新增导入边界：缺少 `db` 表单字段、无效扩展名、过小 `.db`、伪 SQLite magic/corrupt `.db` 均返回失败响应。
  - Phase 9 E2E 导入成功路径：下载当前 SQLite 备份后通过 `importDB` 回灌成功，并触发 Xray 重启成功。
  - `web/html` 与 `frontend/src` 未发现 `v-html`、`innerHTML`、`insertAdjacentHTML`。
  - `frontend/src` 与 `web/ui` 未发现 `unsafe-inline` 或 `unsafe-eval`。
  - Phase 10 前禁止项 `proxy_inbounds`、`proxy_clients`、`CoreManager`、`sing-box`、`Capability Schema` 未出现在 active 代码路径。
- 新 UI：`http://127.0.0.1:2073/phase7a/panel/` 正常，浏览器控制台无 error/warning。
- 新 UI 浏览器检查：21 个 Ant Design Vue 动态 `<style>` 节点全部带 nonce。
- Legacy UI：`http://127.0.0.1:2073/phase7a/panel/legacy/` 正常，浏览器控制台无 error/warning。
- 2026-05-03 隔离面板 E2E：8 个用例，4 passed，4 skipped；该旧结果已被 2026-05-04 的 13 passed 非跳过结果替代。

## 剩余风险

- Legacy UI 仍依赖宽 CSP；短期只作为回退入口接受，长期应逐步下线或继续模板改造。
- 已打开页面的旧会话如果在升级前没有刷新页面，可能缺少新注入的 CSRF token；刷新页面或重新登录即可恢复。
- 本机隔离 Xray core E2E 已通过；CI 或其他测试机仍需复刻 `.env`、隔离 DB/bin/log 目录和 Xray core 二进制。
- 旧 UI 对新 UI 创建数据已完成六类主力协议自动抽检；Legacy Settings 对新 UI 保存订阅标题/公告也已完成自动抽检；入站基础字段、VLESS/Trojan/Shadowsocks/Hysteria2 客户端和 WireGuard peer 编辑后旧 UI/API 可读已完成自动抽检。订阅输出矩阵仍需在第二环境继续扩展。

## 回滚方式

- 如 `/panel/setting/*` 或 `/panel/xray/*` 因 CSRF 误拦截，可临时回退对应 controller 的 CSRF 中间件挂载，但必须保留 `/panel/api/*` 的 CSRF。
- 如新 UI 动态样式 nonce bootstrap 导致第三方样式异常，可临时回退新 UI `style-src` 到 `unsafe-inline`，但必须保留 `script-src` nonce 且记录浏览器控制台证据。
- 如旧 UI 文本渲染影响 WireGuard 配置展示，可回退到 `<pre>` 结构调整，不应恢复 `v-html`。
- 如数据库导入校验误拦截合法文件，可扩展允许扩展名或错误提示，不应移除 SQLite magic/integrity 校验。
