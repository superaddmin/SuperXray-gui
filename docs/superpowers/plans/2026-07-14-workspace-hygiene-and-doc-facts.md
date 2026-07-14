# Workspace Hygiene And Documentation Facts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or equivalent focused delegation to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 执行既有清理设计第 1–5 项，在保留运行数据库、部署私钥和可发布资源的前提下，修复秘密扫描缺口、删除可重建产物、治理本地日志并收敛重复项目事实。

**Architecture:** 将工作拆分为安全扫描、存储清理、运行日志和文档事实四个互不重叠的责任域。项目事实继续以 `.codex/project.toml`、项目上下文和现行源码为准；工具入口文档仅保留简短指针与必要用法。

**Tech Stack:** Python `unittest`、Git ignore 规则、PowerShell、Go/Vue 项目文档。

## Global Constraints

- 保留 `docs/super.pem`；不人工查看、不输出、不改写、不提交其内容，仅允许秘密扫描器做模式检测。
- 保留 `x-ui/x-ui.db` 以及任何 SQLite sidecar；本地没有运行中的 x-ui/Xray 进程后才处理日志。
- 保留 `web/ui`、`media`、`docs/assets`、发布脚本和历史治理文档。
- 删除操作只允许落在 `<repo-root>` 工作区内已盘点的精确路径。
- 所有修改文件保持原有 UTF-8 无 BOM、LF。
- 不升级依赖，不改业务 API，不重新生成 `web/ui`。

---

### Task 1: 密钥忽略与秘密扫描回归

**Files:**
- Modify: `.gitignore`
- Modify: `scripts/secret_scan.py`
- Modify: `tests/test_secret_scan.py`

- [x] 先添加 `.pem` 私钥可被扫描器识别的失败测试并确认失败。
- [x] 将私钥/证书扩展加入仓库忽略规则，并让文本 PEM/KEY 文件进入扫描范围。
- [x] 运行定向单元测试确认通过；默认扫描应跳过已忽略的本地密钥，`--all` 应报告该本地私钥路径。

### Task 2: 高置信度可重建产物清理

**Paths:**
- Delete: `build/`
- Delete: `.codex/skills/superxray-release-cicd/scripts/__pycache__/`
- Delete: `.codex/skills/superxray-release-cicd/tests/__pycache__/`
- Delete: `frontend/tsconfig.app.tsbuildinfo`
- Delete: `frontend/tsconfig.node.tsbuildinfo`
- Delete: `web/job/log/log/3xui.log`

- [x] 记录删除前文件数、大小和绝对路径。
- [x] 用 PowerShell `Remove-Item -LiteralPath` 删除精确目标。
- [x] 逐项确认目标已消失。

### Task 3: 可重装依赖缓存清理

**Paths:**
- Delete: `node_modules/`
- Delete: `frontend/node_modules/`

- [x] 验证两个绝对路径都位于工作区根目录下。
- [x] 删除两套 `node_modules`，保留 `package.json` 与 lockfile。
- [x] 记录释放空间，并记录恢复命令：根目录与 `frontend/` 分别执行 `npm ci`。

### Task 4: 本地 x-ui 日志保留治理

**Paths:**
- Keep: `x-ui/x-ui.db`
- Clean: `x-ui/x-ui/*.log`

- [x] 确认本机没有 x-ui/Xray 进程。
- [x] 采用保守规则：保留数据库，删除 30 天前的本地日志与零字节历史日志。
- [x] 确认数据库大小和哈希未变化。

### Task 5: 重复项目事实收敛

**Files:**
- Modify: `.github/copilot-instructions.md`
- Modify: `CLAUDE.md`
- Modify: `README.zh_CN.md`
- Modify: `frontend/README.md`
- Modify: `CONTRIBUTING.md`

- [x] 以 `.codex/project.toml`、`go.mod`、`package.json` 和现行源码为事实源。
- [x] 统一 Go 版本为 `1.26.4`，删除已退役 `web/html`/`web/assets` 和不存在 `.vscode/tasks.json` 的陈述。
- [x] 将前端说明更新为现行 Vue UI、`frontend/src -> web/ui` 构建边界和当前验证入口。
- [x] 扩充贡献指南，覆盖前后端最小验证、文档同步和秘密扫描，不复制整段架构事实。

### Task 6: 验证与交付

- [x] 运行 Python 单元测试、默认秘密扫描和 `--all` 预期发现检查。
- [x] 校验文档版本、退役路径和不存在入口不再漂移。
- [x] 校验目标缓存已删除、数据库及密钥仍保留、Git diff 仅包含计划内文本变更。
- [x] 输出删除清单、释放空间、验证结果、恢复命令与保留项。
