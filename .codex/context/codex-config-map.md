# SuperXray-gui `.codex` 配置地图

| 文件 | 职责 | 必跑验证 |
| --- | --- | --- |
| `.codex/project.toml` | 项目事实、技术栈、业务域、代理/技能索引 | `validate_codex_config.py` |
| `.codex/governance.toml` | 阶段门禁、分层上下文预算、安全边界 | `validate_codex_config.py`, `secret_scan.py` |
| `.codex/routing.toml` | 路由、主责代理、审查代理、验证命令 | `validate_codex_config.py` |
| `.codex/context/conversation-retrospective-map.md` | 对话复盘、认知盲区、误判模式与修正策略 | `validate_codex_config.py` |
| `.codex/context/runtime-network-debug-map.md` | 网络/代理/分流问题的三视角证据地图 | `validate_codex_config.py` |
| `.codex/workflows/network-routing-debug-checklist.md` | 网络/路由类任务的标准排障顺序和回滚要求 | `validate_codex_config.py` |
| `.codex/agents/*.toml` | 代理角色、职责、输入上下文、交接输出、指标 | `validate_codex_config.py` |
| `.codex/skills/superxray-project-context/scripts/validate_skill_formats.py` | 可移植技能格式验证包装；通过 `CODEX_SKILL_VALIDATOR` 覆盖全局 quick validator 路径 | `validate_codex_config.py` |

每个 agent TOML 必须包含 `knowledge_inputs`、`handoff_outputs`、`collaboration_rules`、`efficiency_metrics`。

```powershell
python .codex/skills/superxray-project-context/scripts/validate_codex_config.py
python .codex/skills/superxray-project-context/tests/test_validate_codex_config.py
python .codex/skills/superxray-project-context/scripts/validate_skill_formats.py .codex/skills/superxray-project-context .codex/skills/superxray-ui-first-migration .codex/skills/superxray-release-cicd
python scripts/secret_scan.py
```
