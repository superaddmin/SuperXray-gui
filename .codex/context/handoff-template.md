# 多代理上下文交接模板

每次从一个代理交给另一个代理时，使用下列结构。交接内容应简洁，优先给证据路径和命令，不粘贴大段日志。

```markdown
## Handoff

任务目标：
- 待填写

当前阶段判断：
- Phase：
- 允许范围：
- 禁止事项：

已读上下文：
- 待填写

已修改/计划修改路径：
- 待填写

关键事实：
- 待填写

待处理问题：
- 待填写

验证命令与结果：
- `命令`：未运行/通过/失败
- `.codex` 变更需包含 `validate_codex_config.py` 与技能验证结果

风险与回滚：
- 风险：
- 回滚：

交给：
- 主责代理：
- 需要审查：
- 需要最终门禁：
```

## 交接规则

- 只传递与下一步相关的上下文。
- 失败日志只传首个错误、直接阻塞点、复现命令和必要文件路径。
- 不传递密钥、cookie、token、证书、数据库内容或用户隐私。
- 不传递真实订阅 URL、subId、客户端 UUID、代理账号密码、面板隐藏路径或生产数据库片段。
- 若已达到 `.codex/governance.toml` 的最大交接深度，必须输出 blocker 而不是继续转交。
- 跨前端/后端/数据库/订阅边界时，必须写清旧 UI、新 UI、旧 API、订阅输出是否受影响。
- 交接后接收代理必须先确认路径归属，再执行改动。

## 配置交接附加规则

涉及 `.codex`、agent、skill、routing、governance、context map 的交接必须额外说明：

- 是否更新 `.codex/configuration-update.md`。
- 是否同步 `dependency-map.md`、`business-flow-map.md`、`codex-config-map.md`。
- 是否新增或修改 agent 契约字段。
- 是否运行 `validate_codex_config.py`、验证单测与 `quick_validate.py`。
- 效率指标中是否出现路由误判、上下文过量读取或验证缺口。
