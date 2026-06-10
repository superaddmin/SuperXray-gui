# .codex 配置体系重构说明

> 本文件是 `.codex` 配置体系的历史变更记录，不是最高优先级运行指令。当前执行以 `.codex/governance.toml`、`.codex/project.toml`、`.codex/routing.toml` 和 context maps 为准。

本次采用“方案 B：保留现有 agent/skill 名称，增量强化配置体系”。目标是在不破坏现有 SuperXray-gui 项目级 Codex 入口的前提下，补齐架构索引、代理协作契约、技能验证脚本和效率评估闭环。

## 1. 技术架构分析结果

### 目录结构与模块划分

| 路径 | 职责 | 主责代理 |
| --- | --- | --- |
| `main.go` | CLI、环境变量、数据库初始化、Web/Sub 服务生命周期 | `superxray-backend-service-guardian` |
| `config/` | 版本、应用名、路径与运行配置 | `superxray-backend-service-guardian` |
| `database/` | SQLite/GORM 初始化、模型、seeders、旧模型兼容 | `superxray-database-steward` |
| `web/` | Gin 面板服务、controller/service/middleware/job/websocket、新 UI embed | `superxray-go-integration` / `superxray-backend-service-guardian` |
| `frontend/` | Vue 3/Vite/TypeScript 新 UI 源码与测试 | `superxray-frontend-migrator` |
| `web/ui/` | Vite 构建产物，Go embed 静态资源 | `superxray-frontend-migrator` |
| `web/html/`, `web/assets/` | 已退役 Legacy UI 与旧静态资源；不得重新挂载 | `superxray-go-integration` |
| `sub/` | 订阅服务、协议输出、diagnose、Clash/Mihomo/WireGuard | `superxray-subscription-protocol-specialist` |
| `xray/` | Legacy Xray 进程/API/traffic 集成 | `superxray-backend-service-guardian` |
| `core/` | CoreManager、default-xray 只读视图、experimental sing-box 适配器 | `superxray-core-runtime-architect` |
| `.github/`, `Dockerfile`, `install.sh` | CI/CD、容器、安装更新、发布资产 | `superxray-devops-cicd-maintainer` / `superxray-release-gate` |
| `.codex/` | 项目级 AI 协作配置、代理、技能、上下文与验证 | `superxray-ui-program-manager` |

### 核心依赖与技术栈

- 后端：Go 1.26.4、Gin 1.12、GORM 1.31、SQLite、Xray-core gRPC/API、robfig/cron、gorilla/websocket、go-i18n、telego、LDAP、TOTP。
- 前端：Vue 3.5、Vite 8、TypeScript 6、Pinia 3、Ant Design Vue 4、Axios 1.16、Vue Router 4。
- 数据库：SQLite + GORM；活跃写模型仍为 `database/model.Inbound`，客户端嵌入 `Inbound.Settings` JSON。
- 构建与验证：`go test` / `go vet` / `go build`、`npm run typecheck/lint/test/build`、Playwright、release gate、secret scan。
- 部署发布：GitHub Actions 发布 Linux `amd64` / `arm64` 二进制资产，可选 GHCR 多架构镜像。

### 业务功能映射

- 面板与安全：登录、session、CSRF、CSP、设置、备份恢复、日志下载。
- Xray 运维：运行状态、启动/停止/重启、版本安装、配置模板、流量统计。
- 入站管理：VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard、客户端管理、批量操作。
- 订阅输出：URI/Base64、Xray JSON、Clash/Mihomo、WireGuard、diagnose。
- Gateway Egress MVP：生成 Xray-compatible SOCKS5 inbound 与 Gateway CSV manifest；不得扩展为生产 `egress_*` 数据库/API。
- 多核心入口：`default-xray` 只读观察与 `experimental-sing-box` 外部适配器；Phase 10.2 前不得接管 legacy Xray 生命周期。

## 2. 配置变更总览

### 新增上下文索引

- `.codex/context/dependency-map.md`：依赖、工具链、验证入口索引。
- `.codex/context/business-flow-map.md`：业务链路到源码路径、代理与验证命令的映射。
- `.codex/context/codex-config-map.md`：`.codex` 文件职责、必跑验证与 agent 契约字段说明。
- `.codex/workflows/config-validation-and-efficiency.md`：配置有效性验证、技能格式验证、效率指标与迭代机制。

### 代理角色体系重构

保留 13 个现有代理名称，在每个 `.codex/agents/*.toml` 中新增统一协作契约：

- `knowledge_inputs`：角色专属优先读取索引与关键源码。
- `handoff_outputs`：固定交接输出模板与项目地图。
- `collaboration_rules`：路由确认、跨职责交接、上下文读取与验证输出规则。
- `efficiency_metrics`：`first_route_accuracy`、`context_files_read_count`、`verification_commands_executed`、`handoff_blocker_clarity`。

### 专业技能库升级

- `superxray-project-context` 增加 deterministic 配置验证脚本：`.codex/skills/superxray-project-context/scripts/validate_codex_config.py`。
- 增加单测：`.codex/skills/superxray-project-context/tests/test_validate_codex_config.py`。
- `.codex/project.toml` 新增 `[stack.ai_config]` 与 `[stack.testing].codex`。
- `.codex/governance.toml` 升级到 `version = 3` 并新增 `[codex_validation]`。
- `.codex/routing.toml` 将 `.codex/configuration-update.md` 纳入治理路由，并把验证脚本/测试加入 `codex-governance` 与 `project-skills` 验证命令。

## 3. 配置验证方案

必跑命令：

```powershell
python .codex/skills/superxray-project-context/tests/test_validate_codex_config.py
python .codex/skills/superxray-project-context/scripts/validate_codex_config.py
python .codex/skills/superxray-project-context/scripts/validate_skill_formats.py .codex/skills/superxray-project-context .codex/skills/superxray-ui-first-migration .codex/skills/superxray-release-cicd
python scripts/secret_scan.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```

编码检查：

```powershell
python -c "from pathlib import Path; bad=[]; [bad.append(str(p)) for p in Path('.codex').rglob('*') if p.is_file() and p.suffix.lower() in {'.md','.toml','.yaml','.yml','.py'} and (p.read_bytes().startswith(b'\xef\xbb\xbf') or b'\r\n' in p.read_bytes() or b'\r' in p.read_bytes())]; print('OK' if not bad else '\n'.join(bad))"
```

## 4. 效率评估与迭代机制

每次 `.codex` 或项目栈演进后记录并复核：

- `first_route_accuracy`：首次路由是否选中正确主责代理。
- `context_files_read_count`：是否控制在治理预算内，避免全仓库无差别读取。
- `verification_commands_executed`：是否执行最小相关验证命令。
- `handoff_blocker_clarity`：交接是否包含首个阻塞点、复现命令、风险和回滚。
- `forbidden_scope_hits`：是否触碰 Phase 10.2 前禁止项、旧模型迁移、legacy UI 退场等硬边界。
- `secret_scan_findings`：配置、文档、脚本是否引入敏感信息。

技能更新策略：技术栈、业务路由、阶段门禁或发布策略变化时，同步更新 `.codex/context/*.md`、`.codex/project.toml`、`.codex/governance.toml`、`.codex/routing.toml`、相关 agent TOML 和技能引用，并重新运行配置验证。

## 5. 应用指南

1. 任务进入后先读 `.codex/governance.toml`、`.codex/project.toml`、`.codex/routing.toml` 和 `.codex/context/project-map.md`。
2. 按 `.codex/routing.toml` 选择最高优先级路由和主责代理。
3. 主责代理只读取 `required_context` 与 `knowledge_inputs` 中和当前任务直接相关的内容。
4. 跨职责边界时使用 `.codex/context/handoff-template.md`，不要复制大段日志或敏感信息。
5. 完成前运行 `.codex/workflows/verification-matrix.md` 中的最小相关命令；`.codex` 变更必须运行配置验证脚本与技能验证。
6. 输出交付说明时写明：变更点、已运行验证、未运行原因、风险与回滚方式。

## 6. 回滚方式

若配置验证失败且无法立即修复，按 Git 回滚本次 `.codex` 与 `.gitignore` 变更：

```powershell
git restore --staged .codex .gitignore
git restore .codex .gitignore
```

若只需回滚新增文件，删除对应新增文件后重新运行 `validate_codex_config.py`，确认路由、技能和上下文索引没有悬空引用。
