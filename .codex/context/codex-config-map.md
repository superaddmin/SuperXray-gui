# SuperXray-gui `.codex` 配置地图

| 文件 | 职责 | 必跑验证 |
| --- | --- | --- |
| `.codex/project.toml` | 项目事实、技术栈、业务域、代理/技能索引 | `validate_codex_config.py` |
| `.codex/governance.toml` | 阶段门禁、上下文预算、安全边界 | `validate_codex_config.py`, `secret_scan.py` |
| `.codex/routing.toml` | 路由、主责代理、审查代理、验证命令 | `validate_codex_config.py` |
| `.codex/agents/*.toml` | 代理角色、职责、输入上下文、交接输出、指标 | `validate_codex_config.py` |

每个 agent TOML 必须包含 `knowledge_inputs`、`handoff_outputs`、`collaboration_rules`、`efficiency_metrics`。

```powershell
python .codex/skills/superxray-project-context/scripts/validate_codex_config.py
python .codex/skills/superxray-project-context/tests/test_validate_codex_config.py
python scripts/secret_scan.py
```