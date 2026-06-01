# SuperXray-gui 验证矩阵

按变更影响面选择最小相关命令；发布前再扩大到全量。

| 变更类型 | 最小验证 | 扩展验证 |
| --- | --- | --- |
| Go controller/service/job | `go test ./web/service ./web/controller ./web/job` | `go test ./...`, `go vet ./...` |
| 数据库模型/迁移/导入 | `go test ./database ./database/model` | `go test ./...`, 备份/导入隔离验证 |
| Xray 进程/API | `go test ./xray ./web/service` | 真实 Xray core 隔离 E2E |
| CoreManager/sing-box | `go test ./core/... ./web/service ./web/controller` | 禁止写路径搜索、CoreInstances E2E |
| 订阅输出 | `go test ./sub ./web/service` | 订阅 URL/JSON/Clash/Mihomo/WireGuard 矩阵抽检 |
| Gateway Egress MVP | `cd frontend; npm run test` | Xray config 与 CSV manifest 手工/浏览器抽检 |
| Vue 新 UI | `cd frontend; npm run typecheck`, `cd frontend; npm run lint` | `cd frontend; npm run test`, `cd frontend; npm run build`, E2E |
| `web/ui` 构建产物 | `cd frontend; npm run build` | `go build -o bin/SuperXray.exe ./main.go`, 浏览器冒烟 |
| Legacy UI 或 Go embed | `go test ./web`, `go build -o bin/SuperXray.exe ./main.go` | 浏览器检查 `/panel/`, `/panel/ui/`, `/panel/legacy/` |
| 安全中间件/下载/导入 | `go test ./web/middleware ./web/controller ./web/service` | XSS/CSRF 搜索、E2E 安全路径 |
| Playwright 旅程 | `npm run e2e` | headed/UI 模式、截图/trace 分析 |
| Docker/CI/脚本 | release gate metadata check | Docker build、CI dry-run、shellcheck 如可用 |
| 发布 | `release_gate.py --install-tools` | Go 全量、frontend 全量、E2E、CodeQL/CI |
| 文档/i18n | `python scripts/secret_scan.py`、文档链接和 key 对齐检查 | frontend i18n tests、release metadata check |
| Codex 治理配置 | skill validate、secret scan、governance/routing 手工一致性检查 | release metadata check、相关 agent workflow 演练 |

## 常用搜索

```powershell
rg "v-html|innerHTML|insertAdjacentHTML" web/html frontend/src -n
rg "unsafe-inline|unsafe-eval" frontend/src web/ui -n
rg "proxy_inbounds|proxy_clients" core web/controller web/middleware web/service database/model frontend/src web/ui -n
rg "egress_|Gateway Egress|gatewayEgressMvp" frontend/src docs plans -n
python scripts/secret_scan.py
```

## E2E 环境变量

```powershell
$env:SUPERXRAY_E2E_BASE_URL = "http://127.0.0.1:<port>/<webBasePath>/panel/"
$env:SUPERXRAY_E2E_USERNAME = "<username>"
$env:SUPERXRAY_E2E_PASSWORD = "<password>"
$env:SUPERXRAY_E2E_RESTART = "1"
$env:SUPERXRAY_E2E_MUTATION = "1"
$env:SUPERXRAY_E2E_IMPORT_DB = "1"
$env:SUPERXRAY_E2E_SUB_URL = "<redacted-subscription-url>"
```

真实凭证、面板路径和订阅 URL 不得写入仓库；交接记录只保留占位符。

## 结果记录

交付说明必须写明：

- 已运行命令。
- 通过/失败结果。
- 未运行原因。
- 失败时的首个错误和下一步动作。
