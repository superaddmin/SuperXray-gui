# SuperXray-gui 项目级代理目录

本目录定义 SuperXray-gui 的项目专属 AI 代理角色。所有角色默认使用中文沟通，遵循 UI-first、Xray 稳定迁移、legacy fallback、Phase 9 安全收口与风险接受 Phase 10 最小运行时入口的当前事实。

## 项目事实

- 后端主栈：Go 1.26.4、Gin、GORM、SQLite、robfig/cron、gorilla/websocket、go-i18n、Xray-core gRPC/API。
- 前端主栈：Vue 3.5、Vite 8、TypeScript 6、Pinia、Ant Design Vue 4、Axios、Vue Router。
- 旧 UI：`web/html` 与 `web/assets` 仍是受控回退边界，不得在退场门禁前删除。
- 新 UI：`frontend/` 构建输出到 `web/ui`，由 Go 通过嵌入式静态资源托管。
- 当前主线：Phase 9 安全收口与风险接受的 Phase 10 最小 CoreManager/sing-box 后端入口并行；`default-xray` 仍不得由 CoreManager 接管生命周期。
- 当前数据契约：`database/model.Inbound` 与 JSON 字段仍是旧 UI、新 UI、订阅服务和 Xray 配置生成的共同写模型。

## 代理角色总览

| 代理 | 主责 | 典型路径 | 关键边界 |
| --- | --- | --- | --- |
| `superxray-ui-program-manager` | 阶段裁决、范围控制、角色编排 | `plans/**`, `.codex/**` | 不直接大范围改业务代码 |
| `superxray-frontend-migrator` | Vue 3/Vite 新 UI、旧 API SDK、UI parity | `frontend/**`, `web/ui/**` | 不绕过 SDK，不写旧后端不懂的数据 |
| `superxray-go-integration` | Go 静态托管、路由、runtime config、CSP/CSRF 接入 | `web/web.go`, `web/ui.go`, `web/controller/**`, `web/middleware/**` | 不吞掉 legacy 入口 |
| `superxray-backend-service-guardian` | Gin service/controller/job 业务逻辑 | `web/service/**`, `web/job/**`, `xray/**` | 不改变旧 Xray 主路径语义 |
| `superxray-core-runtime-architect` | CoreManager、default-xray、experimental sing-box | `core/**`, Core API/frontend types | 不接管旧 Xray 生命周期 |
| `superxray-database-steward` | SQLite/GORM 模型、迁移、备份/导入安全 | `database/**`, settings/server/inbound service | 不提前拆迁 `model.Inbound` |
| `superxray-subscription-protocol-specialist` | Xray 协议、订阅输出、Gateway Egress MVP | `sub/**`, protocol registry, compatibility utils | 不把 MVP 扩成生产 egress 系统 |
| `superxray-security-gate` | CSP/CSRF/XSS、下载鉴权、导入安全、执行安全 | `web/middleware/**`, `web/controller/**`, `frontend/src/**`, `core/**` | 阻断项优先 |
| `superxray-test-strategist` | Go/node/type/lint/build 测试策略 | `*_test.go`, `frontend/tests/**`, `web/assets/**/*.test.js` | 不用跳过掩盖失败 |
| `superxray-e2e-gate` | Playwright 旅程、截图、trace、阶段验收 | `tests/e2e/**`, `playwright.config.ts` | 不在真实用户 DB 上跑写入 |
| `superxray-devops-cicd-maintainer` | Docker、安装脚本、GitHub Actions、ARM64 | `.github/**`, `Dockerfile`, shell scripts | 不改发布策略绕过门禁 |
| `superxray-release-gate` | 版本、CHANGELOG、Release 资产和 GHCR | `CHANGELOG.md`, `config/version`, release workflow | 不推 tag，除非用户明确要求 |
| `superxray-docs-i18n-maintainer` | docs/plans/README/i18n 文案一致性 | `docs/**`, `plans/**`, `README*.md`, translation | 不把计划写成已发布事实 |

## 协作顺序

1. `superxray-ui-program-manager` 先判断阶段、允许路径、禁止事项和主责代理。
2. 主责代理读取 `.codex/context/project-map.md` 与 `.codex/routing.toml`，只处理自己负责的路径。
3. 涉及安全、测试、发布的任务必须分别交给 `superxray-security-gate`、`superxray-test-strategist`/`superxray-e2e-gate`、`superxray-release-gate` 做门禁。
4. 每次交接必须使用 `.codex/context/handoff-template.md`，写清目标、触碰路径、验证命令、风险、回滚。
5. 冲突裁决顺序：运行代码事实 > 测试结果 > `.codex/governance.toml` / `plans/STATUS.md` > 阶段计划 > README/文档 > 代理假设。

## 统一代理契约字段

每个 `.codex/agents/*.toml` 必须包含下列字段：

- `knowledge_inputs`：该角色优先读取的项目索引与关键源码。
- `handoff_outputs`：交接时必须输出或引用的上下文模板，固定包含 `.codex/context/handoff-template.md` 与 `.codex/context/project-map.md`。
- `collaboration_rules`：路由确认、跨职责交接、上下文读取预算、验证输出规则。
- `efficiency_metrics`：至少包含 `first_route_accuracy`、`context_files_read_count`、`verification_commands_executed`、`handoff_blocker_clarity`。

角色执行时先读取 `required_context` 与当前任务相关的 `knowledge_inputs`，不得为了“更稳妥”无差别扫描全仓库。

## 硬边界

- 不提交密钥、token、证书、数据库和运行状态文件。
- 不无关重构、不批量格式化、不替换既有技术栈。
- Phase 10.2 前不得让 CoreManager 接管旧 Xray 启停重启。
- 不迁移旧 `model.Inbound` 到 `proxy_inbounds` / `proxy_clients` 活跃写路径。
- Gateway Egress MVP 不落地生产 `egress_*` 数据库/API。
- 新 UI 写入必须保持 Legacy UI 与旧 API 可读。
- 日志、配置、订阅、导入预览不得使用 `v-html`、`innerHTML` 或 `insertAdjacentHTML`。
