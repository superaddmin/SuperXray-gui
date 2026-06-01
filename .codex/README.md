# SuperXray-gui `.codex` 项目配置

本目录是 SuperXray-gui 的项目级 AI 协作入口，负责把仓库真实架构、阶段门禁、代理角色、上下文交接、技能和验证矩阵固化到本地配置中。

## 当前项目画像

- 后端：Go 1.26.3、Gin、GORM、SQLite、robfig/cron、gorilla/websocket、go-i18n、Xray-core gRPC/API。
- 前端：Vue 3.5、Vite 8、TypeScript 6、Pinia、Ant Design Vue 4、Axios、Vue Router。
- 旧 UI：`web/html` 与 `web/assets` 仍是 `/panel/legacy/` 回退边界。
- 新 UI：`frontend/src` 构建到 `web/ui`，由 Go embed 托管，默认入口为 `/panel/`。
- 订阅：`sub/` 输出 URI/Base64、Xray JSON、Clash/Mihomo、WireGuard 配置和 diagnose。
- 多核心：`core/` 已有最小 CoreManager、`default-xray` 只读观察和 `experimental-sing-box` 外部适配器；仍禁止接管旧 Xray 生命周期。
- 发布：GitHub Actions 发布 Linux `amd64` / `arm64` 二进制资产，并可发布 GHCR `linux/amd64,linux/arm64` 镜像。

## 目录结构

```text
.codex/
├── README.md
├── governance.toml
├── project.toml
├── routing.toml
├── agents/
├── context/
│   ├── project-map.md
│   └── handoff-template.md
├── prompts/
│   └── shared-system-prompt.md
├── workflows/
│   ├── multi-agent-workflow.md
│   └── verification-matrix.md
└── skills/
    ├── superxray-project-context/
    ├── superxray-ui-first-migration/
    └── superxray-release-cicd/
```

## 使用顺序

1. 先读 `.codex/governance.toml`、`.codex/project.toml`、`.codex/routing.toml` 和 `.codex/context/project-map.md`。
2. 需要快速建立项目画像时使用 `.codex/skills/superxray-project-context/SKILL.md`。
3. 涉及 UI-first 阶段、Xray parity、legacy fallback、CoreManager 门禁时使用 `superxray-ui-first-migration` 技能。
4. 涉及版本、CHANGELOG、GitHub Release、GHCR、release workflow 时使用 `superxray-release-cicd` 技能。
5. 按 `.codex/routing.toml` 找优先级最高的主责代理。
6. 主责代理执行前读取对应 `required_context`，交接时使用 `.codex/context/handoff-template.md`。
7. 完成前按 `.codex/workflows/verification-matrix.md` 选择最小相关验证命令。

## 配置边界

- 本目录只保存项目级 AI 协作规则、技能和静态上下文。
- 本目录不保存密钥、token、账号、数据库、运行日志、审计原始材料或全局 Codex 状态。
- 本目录不复制全局 sandbox、approval、MCP、模型、账号或认证配置。
- 历史 `docs/superpowers/**` 只能作为证据材料，不得覆盖 `.codex/governance.toml`、`plans/STATUS.md`、当前阶段门禁或源码事实。
- 真实服务器审计、订阅 URL、代理账号、面板路径、UUID 和数据库内容不得落入仓库；提交前运行 `python scripts/secret_scan.py`。

## 当前硬门禁

- Phase 10.2 前不得让 CoreManager 接管旧 Xray 启停重启。
- 未经明确架构门禁，不得把活跃写路径从 `database/model.Inbound` 迁移到 `proxy_inbounds` / `proxy_clients`。
- Gateway Egress MVP 只允许生成 Xray-compatible SOCKS5 inbound 与 CSV manifest；不得直接落地生产 `egress_*` 数据库/API。
- 不删除 `/panel/legacy/`、`web/html` 或 `web/assets`，直到 legacy 退场门禁通过。
- 日志、配置、订阅、外部内容和导入预览必须纯文本渲染，不使用 `v-html`、`innerHTML` 或 `insertAdjacentHTML`。

## 编码约定

`.codex` 下文本文件为 UTF-8 无 BOM、LF。修改含中文文件前检查编码与换行，写回后执行回读与字节校验。
