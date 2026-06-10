# SuperXray-gui 多代理协作机制

## 默认流程

```text
任务进入
  -> superxray-ui-program-manager 判断阶段和范围
  -> routing.toml 按 priority 选择主责代理
  -> 主责代理实施或分析
  -> security/test/e2e/docs/release 按影响面审查
  -> 汇总验证、风险和回滚
```

## 首轮分流

- `.codex/**` 配置和技能：`superxray-ui-program-manager`，安全/测试/release 复核。
- `frontend/src/**`：`superxray-frontend-migrator`，安全与 E2E 复核。
- `web/ui/**`：视为生成产物，先确认来自 `cd frontend; npm run build`。
- `web/html/**`、`web/assets/**`：已退役旧 HTML UI 目录；如出现改动，只允许删除确认或防重新引入审查。
- `core/**` 与 `/panel/api/cores`：`superxray-core-runtime-architect`，必须确认 Phase 10 准入边界。
- `sub/**`、协议兼容工具、Gateway MVP：`superxray-subscription-protocol-specialist`。
- `.github/**`、Docker、安装脚本、版本资产：`superxray-devops-cicd-maintainer` 或 `superxray-release-gate`。

## 上下文传递

- 使用 `.codex/context/handoff-template.md`。
- 每次交接必须包含目标、阶段、路径、已读上下文、验证状态、风险与回滚。
- 接收代理只补读自己职责相关的文件，不重复扫描全仓库。
- 多个代理意见冲突时，由 `superxray-ui-program-manager` 根据阶段门禁裁决。
- 遵守 `.codex/governance.toml` 的上下文预算：默认最多读取 10 个文件，每个文件优先控制在 260 行内，长日志只传首个错误和复现命令。

## 状态机与退出条件

```text
triage -> owner -> review -> final_gate -> done
```

- 每个任务最多 2 次跨代理交接、1 轮审查返工。
- `superxray-ui-program-manager` 每个任务最多做 1 次阶段裁决；后续由主责代理推进。
- 同一阻塞点在 owner/reviewer 循环中第二次出现时，停止转交并输出 blocker、证据路径和下一步人工决策。
- 不允许通过“再交给另一个代理看看”规避失败测试、阶段门禁或安全阻断。

## 并行协作

可以并行：

- 只读文件扫描、依赖盘点、路由匹配和验证命令查找。
- 前端纯 UI 适配与后端只读 API 测试补充。
- Go 单元测试补充与文档同步。
- DevOps 元数据检查与 release notes 准备。

不应并行：

- 同一文件或同一 API 合约的双代理编辑。
- 数据模型变更与订阅输出变更，除非 database steward 先给出契约。
- CoreManager 生命周期变更与旧 Xray lifecycle 变更。
- 手工编辑 `web/ui` 与同时运行前端构建。

## 审查门禁

| 影响面 | 必须审查 |
| --- | --- |
| 用户输入、鉴权、下载、导入、外部请求、二进制执行 | `superxray-security-gate` |
| 数据模型、迁移、备份恢复、`.db` 文件 | `superxray-database-steward` |
| 协议、订阅、Xray config、Gateway Egress MVP | `superxray-subscription-protocol-specialist` |
| 用户可见 UI、新旧入口、截图/trace | `superxray-e2e-gate` |
| Docker、CI、安装脚本、ARM64 | `superxray-devops-cicd-maintainer` |
| 版本、CHANGELOG、发布资产、GHCR | `superxray-release-gate` |
| `.codex` 规则、路由、技能、代理 | `superxray-ui-program-manager` |

## 冲突裁决

裁决顺序：

1. 运行代码事实。
2. 测试和 E2E 结果。
3. `.codex/governance.toml`、`.codex/project.toml`、`plans/STATUS.md` 和 phase gates。
4. 架构文档和 README。
5. 历史 docs/superpowers 证据。
6. 代理假设。

如果任务越界，输出当前阶段可做的最小替代方案。

## 配置治理闭环

`.codex/**` 变更执行闭环：

1. `superxray-ui-program-manager` 判断变更是否属于治理、路由、agent、skill、context 或 workflow。
2. 读取 `.codex/context/codex-config-map.md` 与 `.codex/workflows/config-validation-and-efficiency.md`。
3. 若新增上下文或规则，更新 `.codex/configuration-update.md` 与相关 context map。
4. 若修改 agent，确认 4 个契约字段仍完整。
5. 运行 `validate_codex_config.py`、验证单测、`validate_skill_formats.py` 包装脚本、secret scan。
6. 交付说明记录效率指标、未运行命令原因、风险与回滚。
