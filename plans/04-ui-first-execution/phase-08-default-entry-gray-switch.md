# UI First Phase 8 - 新 UI 默认入口与 Legacy 回退交付记录

## 目标

在 Phase 7a 本地设置/订阅/备份基础迁移通过后，将新 Vue UI 提升为面板默认入口，同时保留旧 UI 作为 `/panel/legacy/` 回退入口。Phase 8 仍然只服务现有 Xray，不进入 CoreManager、sing-box 或新数据模型。

## 实施范围

- 默认新 UI 入口：
  - `/panel` 重定向到 `/panel/`。
  - `/panel/`、`/panel/dashboard`、`/panel/logs`、`/panel/xray`、`/panel/inbounds`、`/panel/settings` 均返回新 Vue SPA。
  - `/panel/assets/*` 托管新 UI 构建产物。
- 兼容新 UI 入口：
  - `/panel/ui/` 继续可用，用于灰度期兼容旧链接和回滚。
  - `/panel/ui/*` 仍支持前端路由刷新。
- 旧 UI 回退入口：
  - `/panel/legacy/` 进入旧 Dashboard。
  - `/panel/legacy/inbounds`、`/panel/legacy/xray`、`/panel/legacy/settings` 保持旧页面可访问。
  - 旧 UI 侧栏链接改为基于当前 legacy panel path，避免从回退入口跳回默认新 UI。
- CSP 分路径策略：
  - 新 UI `script-src` 不包含 `unsafe-inline` 和 `unsafe-eval`，继续使用 nonce 注入运行时配置。
  - 新 UI 当前因 Ant Design Vue 运行时样式仍保留 `style-src 'unsafe-inline'`，不得与 style nonce 混用。
  - Legacy UI 因 Vue 2 in-DOM 模板和历史内联脚本，继续保留 `script-src 'unsafe-inline' 'unsafe-eval'`。

## 本地运行环境

隔离环境：

```text
base URL: http://127.0.0.1:2073/phase7a/
new UI:   http://127.0.0.1:2073/phase7a/panel/
legacy:   http://127.0.0.1:2073/phase7a/panel/legacy/
compat:   http://127.0.0.1:2073/phase7a/panel/ui/
runtime:  tmp/phase7a-runtime/
```

说明：该隔离运行目录未放入真实 `xray-windows-amd64.exe`，因此 Xray core 启动会出现预期报错；本阶段验收只采信面板入口、设置保存、备份下载、CSP 和旧 UI 回退能力。

## 验收结果

- `/panel/` 默认进入新 UI Dashboard。
- `/panel/ui/` 兼容入口仍可打开新 UI。
- `/panel/legacy/` 可打开旧 UI Dashboard。
- 新 UI 浏览器控制台无 error/warning。
- Legacy UI 浏览器控制台无 error/warning。
- Phase 7a 设置保存和数据库备份下载 E2E 通过。
- 新 UI 和 Legacy UI 截图已保存：
  - `superxray-phase8-default-ui.png`
  - `superxray-phase8-legacy-ui.png`

## 验证命令

```powershell
cd frontend
npm run typecheck
npm run lint
npm run format
npm run build
cd ..
go test ./web/middleware ./web ./web/controller
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
$env:SUPERXRAY_E2E_BASE_URL = 'http://127.0.0.1:2073/phase7a/'
$env:SUPERXRAY_E2E_USERNAME = 'phase7a-admin'
$env:SUPERXRAY_E2E_PASSWORD = 'phase7a-local-pass'
npm run e2e
```

E2E 结果：

```text
7 tests: 3 passed, 4 skipped
```

跳过项原因：

- mutation E2E 未设置 `SUPERXRAY_E2E_MUTATION=1`。
- Xray restart E2E 未设置 `SUPERXRAY_E2E_RESTART=1`，且隔离环境无真实 Xray core。
- subscription URL E2E 未设置 `SUPERXRAY_E2E_SUB_URL`。

## 风险与待收口项

- 新 UI `style-src 'unsafe-inline'` 是 Ant Design Vue 运行时样式兼容项，Phase 9 必须继续评估是否可通过预编译、hash 或组件替换收紧。
- Legacy UI 仍保留 `script-src 'unsafe-inline' 'unsafe-eval'`，仅作为回退入口接受；默认新 UI 不继承该策略。
- 隔离环境未验证真实 Xray core 启停、订阅输出和数据库导入成功路径。
- Phase 8 通过不等于 Phase 10 准入通过；仍需完成 Phase 9 安全收口和真实 Xray E2E。

## 回滚方式

- 将 `/panel/` 默认入口重新指向旧 UI。
- 保留新 UI 在 `/panel/ui/` 作为兼容入口。
- 保留 `/panel/legacy/` 直到至少 1 到 2 个版本周期。
- 本阶段不改数据库结构、不迁移 `model.Inbound`，无需数据回滚。
