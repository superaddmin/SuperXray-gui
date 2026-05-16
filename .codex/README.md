# SuperXray-gui `.codex` 项目配置

本目录是 SuperXray-gui 的项目级 AI 协作入口，负责把仓库真实架构、阶段门禁、代理角色、上下文交接和验证矩阵固化到本地配置中。

## 目录结构

```text
.codex/
├── README.md
├── project.toml
├── routing.toml
├── agents/
│   ├── README.md
│   ├── superxray-ui-program-manager.toml
│   ├── superxray-frontend-migrator.toml
│   ├── superxray-go-integration.toml
│   ├── superxray-backend-service-guardian.toml
│   ├── superxray-core-runtime-architect.toml
│   ├── superxray-database-steward.toml
│   ├── superxray-subscription-protocol-specialist.toml
│   ├── superxray-security-gate.toml
│   ├── superxray-test-strategist.toml
│   ├── superxray-e2e-gate.toml
│   ├── superxray-devops-cicd-maintainer.toml
│   ├── superxray-release-gate.toml
│   └── superxray-docs-i18n-maintainer.toml
├── context/
│   ├── project-map.md
│   └── handoff-template.md
├── prompts/
│   └── shared-system-prompt.md
├── workflows/
│   ├── multi-agent-workflow.md
│   └── verification-matrix.md
└── skills/
    ├── superxray-ui-first-migration/
    └── superxray-release-cicd/
```

## 使用顺序

1. 先读 `.codex/project.toml` 和 `.codex/context/project-map.md`。
2. 按 `.codex/routing.toml` 找主责代理。
3. 主责代理执行前读取对应 `required_context`。
4. 跨代理交接使用 `.codex/context/handoff-template.md`。
5. 完成前按 `.codex/workflows/verification-matrix.md` 选择验证命令。

## 配置边界

- 本目录不保存密钥、token、账号、数据库和运行状态。
- 本目录不复制全局 sandbox、approval、MCP 或模型配置。
- 项目级规则只记录 SuperXray-gui 的长期事实、阶段门禁和代理协作协议。
