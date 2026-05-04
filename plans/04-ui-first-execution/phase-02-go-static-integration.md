# UI First Phase 2 - Go 静态资源接入记录

## 目标

将 Phase 1 的 Vue 3/Vite 新前端构建产物接入 Go 后端，并通过 `/panel/ui/` 提供受登录保护的新 UI 入口。旧版 `/panel/` 继续作为主入口与回退入口，不改变现有 Xray API、数据模型和生命周期。

## 实施范围

- Vite 生产构建输出目录调整为 `web/ui`，用于 Go `go:embed`。
- Go 后端新增 `/panel/ui` 到 `/panel/ui/` 的跳转。
- Go 后端新增 `/panel/ui/` SPA 入口、静态文件服务和历史路由回退。
- 新 UI HTML 响应注入 `window.__SUPERXRAY_UI_CONFIG__`，包含 `basePath`、`apiBasePath`、`uiBasePath`、`cspNonce`、版本号。
- 新 UI 静态资源使用缓存策略：入口 HTML `no-store`，哈希资源 `immutable`。
- 新 UI 入口沿用现有登录态检查；未登录访问跳回旧登录入口。

## 安全约束

- 新 UI 继续使用严格 CSP 路径分流，不依赖 `unsafe-inline` 或 `unsafe-eval`。
- 运行时配置脚本使用后端生成的 CSP nonce。
- 静态文件路径做清理，拒绝目录穿越和 Windows 反斜杠路径。
- 不启用新写路径，不迁移 `model.Inbound`，不引入多内核生命周期管理。

## 验证项

```powershell
cd frontend
npm run typecheck
npm run lint
npm run build
cd ..
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

## 回滚方式

- 删除 `/panel/ui` 路由注册与 `web/ui.go`。
- 将 `frontend/vite.config.ts` 的 `build.outDir` 恢复为 `dist`。
- 删除生成的 `web/ui` 构建产物。
