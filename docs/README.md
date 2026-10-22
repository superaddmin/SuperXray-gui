# 文档索引

核对日期：2026-10-23。当前代码版本为 `3.6.0`（`config/version`），工作区分支为 `integration/3x-ui-v3.10.2`，默认运行时为 Xray `26.10.21`（配置兼容基线 `26.10.10`），Go 模块声明 `1.27.2`。本文记录当前工作区实现，远端发布和实际部署环境需另行验证。

## 当前实现与使用

| 文档 | 内容与事实入口 |
|---|---|
| [系统架构](architecture.md) | Go/Web/Sub/Vue 分层、数据流、Core 边界、后台任务 |
| [核心模块](modules.md) | 目录职责、服务与配置；源码 `web/`、`core/`、`database/`、`sub/` |
| [API 说明](api.md) | 旧面板 API、版本化只读 API、鉴权/CSRF、WebSocket、订阅 |
| [面板 OpenAPI](openapi/panel-api.yaml) | 面板源契约；`tools/openapiexport` 生成前端 JSON |
| [版本化 OpenAPI](openapi/api-v1.yaml) | `/api/v1/health/live`、`health/ready`、`metrics`，Go 契约测试守护 |
| [开发指南](development.md) | 工具链、构建、验证、Go embed、目录拓扑、发布流程 |
| [部署指南](deployment.md) | Linux 安装、Docker、反向代理、备份和恢复；以安装脚本/工作流为准 |
| [入站创建指南](inbound-creation-guide.md) | 现有 Vue 表单、协议、客户端和导出 |
| [运行可靠性](runtime-reliability.md) | HTTP 超时、调度、Xray 进程与运行边界 |
| [后端重基线契约](backend-internal-rebaseline-contract.md) | 已实现的渐进边界与未落地的目标架构 |

## 运维模板与外部环境

| 文档 | 使用范围 |
|---|---|
| [部署与路由总览](operations-ai-routing-overview.md) | 服务器、OpenWrt/Passwall 与 AI 出口的职责分离、三视角验证和常见误判；已合并原 Passwall 手册 |
| [AI 分流与住宅出口](ai-routing-and-residential-egress.md) | Xray 模板示例与客户端/网关/服务端取证 |
| [Gateway MVP 交接](SUPERXRAY_GUI_VPN_LAYER_HANDOFF_CN.md) | SOCKS 入站与 CSV；外部 Gateway 的集成约定 |
| [Gateway 超时修复](gateway-egress-timeout-remediation-runbook.md) | 分层排障、验证脚本的能力和限制 |
| [入站/订阅审计模板](server-inbound-subscription-audit.template.md) | 脱敏模板；必须填写现场证据，不能当已验收结果 |

示例中的主机、端口、网段和凭据需替换为本环境值。保存配置不等于运行时生效；必须重新验证实际路由、出口、端口和目标服务。`docs/private-ops/` 是被忽略的本地私有历史记录，不属于公开源码文档或默认配置。

## 历史审计与维护政策

| 文档 | 性质 |
|---|---|
| [Legacy → Vue 对账](legacy-vue-ui-parity-audit.md) | 保留历史迁移证据，并对照当前 Vue 实现 |
| [UI/UX 审计](ui-ux-audit.md) | 历史截图/视觉结论与当前源码对照；本次未重新视觉验收 |
| [上游同步政策](upstream-sync-policy.md) | 同步范围、兼容与验证规则 |
| [上游 v1.11.4 集成记录](upstream-alireza0-v1.11.4-integration.md) | 历史集成记录，不能推定未来上游已同步 |
| [上游 3x-ui v3.10.2 集成记录](upstream-3x-ui-v3.10.2-integration.md) | 本地定制保护、选择性移植、Xray 迁移及已执行验证 |
| [2026-10-22 清理与完整性核对](repository-audit-2026-10-22.md) | 当前工作区的删除、保留、验证与边界 |
| [2026-10-14 至 10-15 清理历史](repository-audit-history.md) | 已合并前两轮清理记录；只保留时间线、关键数字和隔离位置 |

3x-ui 比较证据包含四份固定基线清单：[本地提交](upstream-3x-ui-v3.10.2-local-commits.tsv)、[上游提交](upstream-3x-ui-v3.10.2-upstream-commits.tsv)、[两侧文件差异](upstream-3x-ui-v3.10.2-file-manifest.tsv)、[直接树差异](upstream-3x-ui-v3.10.2-tree-diff.tsv)。这些是完整盘点结果，`inventory_only` 不表示每条上游变更已审查或合并。

[桌面截图](assets/xray-mvp-desktop.png) 与 [移动截图](assets/xray-mvp-mobile.png) 是 2026-05-16 的历史审计附件，仍被文档引用，因此保留。当前产品截图另见仓库 `media/`。

## 维护规则

先核对源码、测试和当前配置，再更新说明；不要将路线图写成已实现功能。阶段规划在 [`plans/`](../plans/README.md)，不存在的历史路径以代码文本标识，不能作为有效源码链接。

面板路径均相对于 `webBasePath`：源码默认 `/super/`，默认登录页是 `/super/panel/login`。订阅是独立服务，路径由订阅设置决定。旧 HTML UI、旧订阅 HTML 页及 `/panel/legacy*` 已退役。

修改面板 OpenAPI 后在 `frontend/` 运行 `npm run build`，同步 `frontend/public/openapi.json` 和 `web/ui/openapi.json`；`web/ui` 是 Go 编译时 embed 输入，必须保留。验证命令与环境要求见 [开发指南](development.md)。
