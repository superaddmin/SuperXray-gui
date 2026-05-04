# UI First Phase 1 - Frontend Shell Implementation Record

## 阶段定位

- 当前阶段：Phase 1，新 Vue 3/Vite 前端工程骨架。
- 施工目标：新建可构建的新 UI 壳，不迁移复杂业务，不改变旧 API 语义。
- 强制边界：旧 `/panel`、旧 API、`model.Inbound`、Xray 生命周期、CoreManager、多内核写路径均保持不变。

## 已交付

| 交付物            | 路径                                                    | 说明                                                          |
| ----------------- | ------------------------------------------------------- | ------------------------------------------------------------- |
| Vite 前端工程     | `frontend/`                                             | Vue 3、TypeScript、Vite 8                                     |
| 基础路由          | `frontend/src/router/index.ts`                          | `/`、`/dashboard`、`/logs`、`/xray`、`/inbounds`、`/settings` |
| 主布局            | `frontend/src/layouts/MainLayout.vue`                   | 侧边栏、顶部栏、移动端折叠                                    |
| 状态管理          | `frontend/src/stores/`                                  | app/runtime config/theme/session 占位                         |
| API 基础层        | `frontend/src/api/http.ts`                              | Axios、cookie、后续 CSRF token 注入点                         |
| CSP 预留          | `frontend/src/App.vue`、`frontend/src/types/runtime.ts` | 支持 `cspNonce` 传入 Ant Design Vue                           |
| 严格 CSP 基础设施 | `web/middleware/security.go`                            | `/panel/ui` 路径不包含 `unsafe-eval`/`unsafe-inline`          |
| CSP 单测          | `web/middleware/security_test.go`                       | 覆盖旧 UI 与新 UI 路径差异                                    |

## 前端依赖

```text
Vue 3.5
Vite 8
TypeScript 6
Vue Router 4
Pinia 3
Ant Design Vue 4
Axios 1
CodeMirror 6
ESLint 9
Prettier 3
```

## 验证结果

```powershell
cd frontend
npm run typecheck
npm run lint
npm run build
```

全部通过。

```powershell
go test ./...
go vet ./...
go build -o bin\SuperXray.exe .\main.go
```

全部通过。

## 当前限制

- 新 UI 尚未由 Go 托管；这是 Phase 2 的范围。
- 新 UI 页面仍为空壳，不读取真实业务 API；只保留路由与布局。
- `frontend/node_modules` 存在时，`go test ./...` 会扫描到部分 npm 依赖内自带的 Go 示例包；本次验证已通过，但后续 CI 如需收敛，应在 Go 测试命令或构建环境中排除 `frontend/node_modules`。
- `npm install` 在 Node 23 下会出现少量传递依赖 engine warning；Node 20/22/24 LTS 路径更干净。

## 回滚方式

删除或停用以下内容即可回到 Phase 0 状态：

```text
frontend/
plans/04-ui-first-execution/phase-01-frontend-shell.md
web/middleware/security.go 中的新 UI CSP 分支
web/middleware/security_test.go 中的新 UI CSP 单测
```

旧 UI 路由、旧 API、数据库和 Xray 运行路径不受影响。

## Phase 2 入口

下一阶段应接入 Go 静态托管，建议入口保持：

```text
/panel/ui
```

并保留旧入口：

```text
/panel
```

Phase 2 必须注入 `window.__SUPERXRAY_UI_CONFIG__`，至少包含：

```text
basePath
uiBasePath
apiBasePath
csrfToken
cspNonce
version
```
