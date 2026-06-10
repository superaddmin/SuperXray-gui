# `.codex` 配置验证与效率评估

## 配置有效性验证

```powershell
python .codex/skills/superxray-project-context/scripts/validate_codex_config.py
```

检查范围：

- 必备 `.codex` 文件与技能目录是否存在。
- `.md`、`.toml`、`.yaml`、`.yml`、`.py` 是否为 UTF-8 无 BOM + LF。
- 13 个 agent TOML 是否包含 `knowledge_inputs`、`handoff_outputs`、`collaboration_rules`、`efficiency_metrics`。
- `.codex/project.toml` 是否索引 `codex_config`、`dependency_context`、`business_flow_context`、`[stack.ai_config]`、`[stack.testing].codex` 和 `project_context_validator`。
- `.codex/governance.toml` 是否为 `version = 3`，是否包含 `[codex_validation]` 与 `bootstrap_read` / `extended_read` / `on_demand_read` 分层上下文索引。
- `.codex/routing.toml` 是否存在未知 agent 引用、非便携绝对验证路径、缺失验证路径，并将验证脚本纳入 `codex-governance` / `project-skills` 路由。
- 技能 `SKILL.md` frontmatter 与 `agents/openai.yaml` 基本字段是否存在。
- 新增运行态知识文件与网络调试 checklist 是否存在并被索引；敏感 artifact glob 是否被 `.gitignore` 覆盖。

## 验证器单测

```powershell
python .codex/skills/superxray-project-context/tests/test_validate_codex_config.py
```

覆盖场景：

- 有效最小 `.codex` 树无 error。
- route 引用未知 agent 报 `route_unknown_agent`。
- UTF-8 BOM 报 `text_encoding_policy`。
- 嵌套 `interface.display_name` / `short_description` / `default_prompt` 的 `agents/openai.yaml` 被接受。
- 分层上下文读取替代旧 `first_read` 时仍覆盖必要 context map。
- agent 声明上下文超过预算时给出 warning。
- verification 命令中的本机绝对路径给出 warning。
- 敏感 artifact glob 未被 `.gitignore` 覆盖时给出 warning。

## 技能格式验证

```powershell
python .codex/skills/superxray-project-context/scripts/validate_skill_formats.py .codex/skills/superxray-project-context .codex/skills/superxray-ui-first-migration .codex/skills/superxray-release-cicd
```

该包装脚本默认探测当前用户目录下的全局 `quick_validate.py`，也可用 `$env:CODEX_SKILL_VALIDATOR` 指定路径。未找到全局验证器时输出 `SKIP` 并返回 0；结构性阻断仍由 `validate_codex_config.py` 覆盖。

## 效率指标

| 指标 | 含义 | 期望 |
| --- | --- | --- |
| `first_route_accuracy` | 首次路由是否选中正确主责代理 | 错误时更新 routing 或 agent 描述 |
| `context_files_read_count` | 单轮读取上下文数量 | 遵守 `.codex/governance.toml` 预算 |
| `verification_commands_executed` | 实际执行验证命令数 | 覆盖最小相关验证 |
| `handoff_blocker_clarity` | 交接是否包含首个阻塞点、复现命令、风险和回滚 | 阻塞清晰可接手 |
| `forbidden_scope_hits` | 是否触碰阶段硬边界 | 出现即停止并交 program manager 裁决 |
| `secret_scan_findings` | secret scan 发现数量 | 提交前必须为 0 或明确误报依据 |
| `task_domain_switches_explicit` | 跨域任务是否显式重述当前主问题与暂挂项 | 避免发布/部署/网络/文档混线 |
| `runtime_evidence_chain_complete` | 是否至少有双视角运行态证据支撑网络结论 | 网络结论必须可复现、可回滚 |

## 迭代机制

- 技术栈版本变化：同步 `.codex/project.toml`、`dependency-map.md`、`project-map.md`、`current-stack.md`。
- 业务流程变化：同步 `business-flow-map.md`、对应 agent `knowledge_inputs`、验证矩阵。
- 阶段门禁变化：同步 `governance.toml`、`routing.toml`、UI-first migration skill 和 phase gates。
- 技能或 agent 改动：运行验证器单测、配置验证和 `validate_skill_formats.py` 包装脚本。
- 发布/安全相关配置变化：追加 `python scripts/secret_scan.py` 与 release metadata gate。
- 运行态排障/网络治理经验变化：同步 `conversation-retrospective-map.md`、`runtime-network-debug-map.md`、`network-routing-debug-checklist.md` 与相关 docs 运行手册。
