# UI First Phase 0 - E2E Baseline

> 目标：建立旧 UI 可复现的最小端到端验收脚本，后续新 UI 每迁移一块能力，都要与这套基线对齐。

## 测试入口

- 配置文件：`playwright.config.ts`
- 测试脚本：`tests/e2e/legacy-panel.spec.ts`
- 执行命令：`npm run e2e`
- 默认浏览器：Chromium
- 产物目录：`playwright-report/`、`test-results/`，已加入 `.gitignore`

## 环境变量

必须提供：

```powershell
$env:SUPERXRAY_E2E_BASE_URL = "http://127.0.0.1:2053/<webBasePath>/"
$env:SUPERXRAY_E2E_USERNAME = "<username>"
$env:SUPERXRAY_E2E_PASSWORD = "<password>"
```

可选：

```powershell
$env:SUPERXRAY_E2E_TOTP = "<2fa-code>"
$env:SUPERXRAY_E2E_MUTATION = "1"
$env:SUPERXRAY_E2E_RESTART = "1"
$env:SUPERXRAY_E2E_IMPORT_DB = "1"
$env:SUPERXRAY_E2E_SUB_URL = "http://127.0.0.1:2096/sub/<subId>"
```

说明：

- `SUPERXRAY_E2E_BASE_URL` 必须包含真实 `webBasePath`，并建议以 `/` 结尾。
- 默认只运行登录、页面导航、状态、设置、入站列表、日志读取等低风险基线。
- `SUPERXRAY_E2E_MUTATION=1` 会运行入站/客户端写入与新旧 UI 兼容矩阵，测试数据会在测试结束时删除或恢复。
- `SUPERXRAY_E2E_RESTART=1` 会触发旧 API 的 Xray 重启路径，仅在测试环境执行。
- `SUPERXRAY_E2E_IMPORT_DB=1` 会下载当前数据库备份并通过 `importDB` 回灌，仅在真实 Xray core 隔离环境执行。
- `SUPERXRAY_E2E_SUB_URL` 用于验证已知订阅 URL；不自动猜测订阅端口和 subId。

## 本地运行

```powershell
npm install
npx playwright install chromium
npm run e2e
```

本机隔离运行环境已在 2026-05-04 修复：

- `.env` 已配置为 `http://127.0.0.1:2073/phase7a/`、`phase7a-admin` 测试账号。
- `tmp/phase7a-runtime/xray-bin/` 已补齐 `xray-windows-amd64.exe`、`geoip.dat`、`geosite.dat`。
- 当前隔离 Xray core：`Xray 26.3.27`。
- 已开启 `SUPERXRAY_E2E_MUTATION=1`、`SUPERXRAY_E2E_RESTART=1`、`SUPERXRAY_E2E_IMPORT_DB=1` 和 `SUPERXRAY_E2E_SUB_URL`。
- `npm run e2e` 已达到 13 passed / 0 skipped / 0 failed。

启动隔离面板：

```powershell
$runtime = Resolve-Path 'tmp/phase7a-runtime'
$env:XUI_DB_FOLDER = (Resolve-Path 'tmp/phase7a-runtime/db').Path
$env:XUI_BIN_FOLDER = (Resolve-Path 'tmp/phase7a-runtime/xray-bin').Path
$env:XUI_LOG_FOLDER = (Resolve-Path 'tmp/phase7a-runtime/log').Path
$p = Start-Process -FilePath (Resolve-Path 'bin/SuperXray.exe').Path -ArgumentList 'run' -WorkingDirectory (Resolve-Path '.').Path -RedirectStandardOutput (Join-Path $runtime 'server.e2e-env.out.log') -RedirectStandardError (Join-Path $runtime 'server.e2e-env.err.log') -WindowStyle Hidden -PassThru
[System.IO.File]::WriteAllText((Join-Path $runtime 'server.e2e-env.pid'), [string]$p.Id, [System.Text.UTF8Encoding]::new($false))
```

停止隔离面板：

```powershell
$pidValue = [int](Get-Content -Encoding UTF8 'tmp/phase7a-runtime/server.e2e-env.pid')
Stop-Process -Id $pidValue -ErrorAction SilentlyContinue
```

如需观察浏览器：

```powershell
npm run e2e:headed
```

## 当前覆盖

| 流程                    | 默认执行 | 风险 | 说明                                                                                                               |
| ----------------------- | -------: | ---- | ------------------------------------------------------------------------------------------------------------------ |
| 登录                    |       是 | 低   | 支持普通登录；开启 2FA 时需提供 `SUPERXRAY_E2E_TOTP`                                                               |
| Dashboard 状态读取      |       是 | 低   | 校验 `/panel/api/server/status`                                                                                    |
| Inbounds 页面与列表读取 |       是 | 低   | 校验 `/panel/inbounds` 与 `/panel/api/inbounds/list`                                                               |
| Xray 配置页面读取       |       是 | 中   | 校验 `/panel/xray` 与 `POST /panel/xray/`，不保存配置                                                              |
| Settings 页面读取       |       是 | 低   | 校验 `/panel/settings` 与 `POST /panel/setting/all`                                                                |
| 面板日志/Xray 日志读取  |       是 | 低   | 校验日志 API，不渲染 HTML                                                                                          |
| 新增禁用入站/客户端     |       否 | 中   | 需 `SUPERXRAY_E2E_MUTATION=1`，测试后删除                                                                          |
| 新 UI 创建后旧 UI 读取  |       否 | 中   | 需 `SUPERXRAY_E2E_MUTATION=1`，六类协议禁用入站均校验 Legacy UI 与旧 API 可读                                      |
| 新 UI 设置后旧 UI 读取  |       否 | 中   | 需 `SUPERXRAY_E2E_MUTATION=1`，保存订阅标题/公告后校验 Legacy Settings 可读                                        |
| 新 UI 编辑后旧 UI 读取  |       否 | 中   | 需 `SUPERXRAY_E2E_MUTATION=1`，校验入站、VMess/VLESS 外的主力客户端、WireGuard peer 编辑后 Legacy UI 与旧 API 可读 |
| 重启 Xray               |       否 | 高   | 需 `SUPERXRAY_E2E_RESTART=1`                                                                                       |
| DB 备份回灌导入         |       否 | 高   | 需 `SUPERXRAY_E2E_IMPORT_DB=1`，仅隔离环境                                                                         |
| 访问订阅 URL            |       否 | 低   | 需显式提供 `SUPERXRAY_E2E_SUB_URL`                                                                                 |

## 阶段门禁

Phase 0 的 E2E 只证明旧行为可复现，不要求新 UI 存在。进入后续阶段时：

- Phase 1/2：新增前端壳和 Go 静态集成后，必须新增新 UI 登录和回退入口测试。
- Phase 4：Dashboard、日志、配置只读视图必须复用本基线的 API 断言。
- Phase 5：Xray 重启和配置保存必须在旧 UI 可回退前提下单独开关执行。
- Phase 6：入站和客户端写入必须保持旧 UI 可读写，并复用 mutation 基线做对照。
- Phase 8：灰度开关必须同时验证新 UI 默认入口与 `/panel/legacy`。
- Phase 9：安全收口必须加入日志 XSS、CSRF、未登录下载、CSP 检查。

## 回滚

本阶段新增文件不参与运行时代码路径。若 E2E 配置影响本地开发，可删除：

```text
package.json
playwright.config.ts
tests/e2e/
plans/04-ui-first-execution/phase-00-*.md
```

不会影响 Go 构建、旧 UI 路由、数据库或 Xray 生命周期。
