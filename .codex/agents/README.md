# SuperXray-gui 项目级代理目录

本目录定义 SuperXray-gui 的项目专属 AI 代理角色。所有角色默认使用中文沟通，遵循 UI-first、Xray 稳定迁移、再接多内核的路线，并以仓库内真实代码、`plans/` 阶段文档和 `.codex/skills/` 项目技能为准。

## 项目事实

- 后端主栈：Go 1.26.3、Gin、GORM、SQLite、robfig/cron、gorilla/websocket、go-i18n、Xray-core gRPC。
- 前端主栈：Vue 3、Vite、TypeScript、Pinia、Ant Design Vue 4、Axios。
- 旧 UI：`web/html` 与 `web/assets` 仍是受控回退边界，不得在新 UI 完全等价前删除。
- 新 UI：`frontend/` 构建输出到 `web/ui`，由 Go 通过嵌入式静态资源托管。
- 当前主线：Phase 9 安全收口与 Phase 10 最小多核心入口并行；`default-xray` 仍不得由 CoreManager 接管生命周期。

## 代理角色总览

| 代理 | 主责 | 典型路径 |
| --- | --- | --- |
| `superxray-ui-program-manager` | 阶段裁决、范围控制、角色编排 | `plans/**`, `.codex/**` |
| `superxray-frontend-migrator` | Vue 3/Vite 新 UI、旧 API SDK、UI parity | `frontend/**`, `web/ui/**` |
| `superxray-go-integration` | Go 静态托管、路由、runtime config、旧 API 兼容 | `web/web.go`, `web/controller/**`, `web/middleware/**` |
| `superxray-backend-service-guardian` | Gin service/controller/job 业务逻辑 | `web/service/**`, `web/job/**`, `xray/**` |
| `superxray-core-runtime-architect` | CoreManager、default-xray、experimental sing-box | `core/**`, `web/service/core_service.go`, `web/controller/core.go` |
| `superxray-database-steward` | SQLite/GORM 模型、迁移、备份/导入安全 | `database/**`, `web/service/server.go` |
| `superxray-subscription-protocol-specialist` | Xray 协议、订阅输出、Gateway egress MVP | `sub/**`, `web/service/protocol_validation.go`, `frontend/src/utils/*Compat*` |
| `superxray-security-gate` | CSP/CSRF/XSS、下载鉴权、导入安全、二进制执行安全 | `web/middleware/**`, `web/controller/**`, `frontend/src/**` |
| `superxray-test-strategist` | 单元/集成/类型测试策略与覆盖缺口 | `*_test.go`, `frontend/tests/**`, `web/assets/**/*.test.js` |
| `superxray-e2e-gate` | Playwright 旅程、截图、trace、阶段验收 | `tests/e2e/**`, `playwright.config.ts` |
| `superxray-devops-cicd-maintainer` | Docker、安装脚本、GitHub Actions、CI 可复现性 | `.github/**`, `Dockerfile`, `install.sh`, `update.sh` |
| `superxray-release-gate` | 版本、CHANGELOG、Release 资产和发布门禁 | `CHANGELOG.md`, `config/version`, `.github/workflows/release.yml` |
| `superxray-docs-i18n-maintainer` | docs/plans/README/i18n 文案一致性 | `docs/**`, `plans/**`, `README*.md`, `web/translation/**` |

## 协作顺序

1. `superxray-ui-program-manager` 先判断阶段、允许路径、禁止事项和主责代理。
2. 主责代理读取 `.codex/context/project-map.md` 与 `.codex/routing.toml`，只处理自己负责的路径。
3. 涉及安全、测试、发布的任务必须分别交给 `superxray-security-gate`、`superxray-test-strategist`/`superxray-e2e-gate`、`superxray-release-gate` 做门禁。
4. 每次交接必须使用 `.codex/context/handoff-template.md`，写清目标、触碰路径、验证命令、风险、回滚。
5. 冲突裁决顺序：运行代码事实 > 测试结果 > `plans/STATUS.md` > 阶段计划 > README/文档 > 代理假设。

## 硬边界

- 不提交密钥、token、证书、数据库和运行状态文件。
- 不无关重构、不批量格式化、不替换既有技术栈。
- Phase 10.2 前不得让 CoreManager 接管旧 Xray 启停重启。
- 不迁移旧 `model.Inbound` 到 `proxy_inbounds` / `proxy_clients` 活跃写路径。
- 新 UI 写入必须保持 Legacy UI 与旧 API 可读。
- 日志、配置、订阅、导入预览不得使用 `v-html`、`innerHTML` 或 `insertAdjacentHTML`。
