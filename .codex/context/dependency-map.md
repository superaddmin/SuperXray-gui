# SuperXray-gui 依赖与工具链地图

## 读取原则

版本以 `go.mod`、`go.sum`、根/前端 `package-lock.json` 和 `.github/workflows/*` 为准；不扫描 `node_modules/`、`bin/`、`tmp/`、`web/ui/assets/**`、`*.db`、`*.sqlite`。

## 核心依赖

| 组件 | 来源 | 用途 | 主要路径 |
| --- | --- | --- | --- |
| Go | `go.mod` | 服务入口和后端 | `main.go`, `web/`, `sub/`, `core/` |
| Gin | `go.mod` | HTTP 路由和中间件 | `web/web.go`, `web/controller/**`, `sub/sub.go` |
| GORM + SQLite | `go.mod` | ORM 和本地数据库 | `database/**`, `web/service/**` |
| Xray-core + gRPC | `go.mod` | Xray API 类型与通信 | `xray/**`, `web/service/xray.go` |
| Vue/Vite/TS | `frontend/package-lock.json` | 新 UI 和构建 | `frontend/src`, `web/ui` |

## 验证入口

- Go：`go test ./...`、`go vet ./...`、`go build -o bin/SuperXray.exe ./main.go`
- 前端：`cd frontend; npm run typecheck; npm run lint; npm run test; npm run build`
- `.codex`：`python .codex/skills/superxray-project-context/scripts/validate_codex_config.py`
- 安全/发布：`python scripts/secret_scan.py`、`python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only`