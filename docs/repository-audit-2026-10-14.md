# 仓库清理与文档核对记录

开始日期：2026-10-14；收尾日期：2026-10-15。代码基线：`main`，`d4a8e8913`，产品版本 `3.4.3`。结果为本地工作区变更，未提交、推送或发布。

## 范围与结论

排除 `.git` 内部对象后，对全部可见、隐藏和被忽略目录做文件元数据遍历：原有 26,673 个文件、1,668 个目录、337,098,121 字节，其中 481 个文件受版本控制；没有符号链接、空目录或读取失败。已跟踪资源做 SHA-256 查重，前端 52 个源码文件检查导入可达性，Go/npm 依赖按实际导入、配置、脚本和 peer 契约核对。

原有公开 `docs/` 为 18 份 Markdown、2 份 OpenAPI YAML、2 张图片，均已逐项核对。另有 2 份 ignored 私有历史记录，仅进行脱敏元数据和仓库路径核对，保留原样。新增本记录与 [文档索引](README.md)。

文档已按当前源码修正路由、CSRF、数据字段、默认配置、入站操作、构建/部署和历史边界。历史审计、外部系统设计及现场案例明确标注性质，不把旧测试或现场数据写成当前验收。静态契约和已运行测试能证明已检查范围，不能证明全部部署环境、外部核心和第三方服务的“100% 完全一致”。

## 实际清理方式

永久删除命令被工具执行策略拒绝，返回 `blocked by policy`；未永久删除任何文件。确认的冗余产物已移出仓库至：

```text
F:/SuperXray-gui-cleanup-20261015/
```

这是可恢复隔离，**没有释放磁盘空间**。原有冗余产物合计 26,177 个文件、328,291,744 字节（约 313.08 MiB）：

| 原路径 | 文件数 | 字节 | 判定依据 |
|---|---:|---:|---|
| `frontend/node_modules/` | 26,167 | 204,662,964 | 锁文件可通过 `npm ci` 重建，全部包有使用证据 |
| `bin/x-ui-linux-amd64-hotfix` | 1 | 56,024,440 | ignored 本地面板构建，未被仓库引用或进程使用 |
| `build/x-ui-linux-amd64` | 1 | 67,568,808 | ignored 本地面板构建，非运行核心或源码 |
| `build/logmarker.txt`、`build/logmarker2.txt` | 2 | 6 | 仅数字日志标记 |
| `.playwright-mcp/` | 2 | 172 | 旧浏览器工具日志 |
| `frontend/tsconfig.app.tsbuildinfo`、`tsconfig.node.tsbuildinfo` | 2 | 1,457 | TypeScript 增量缓存 |
| `.codex/skills/superxray-project-context/scripts/__pycache__/` | 1 | 32,771 | Python 字节码缓存 |
| `web/job/log/` | 1 | 1,126 | 测试初始化生成的 `log/log/3xui.log`，在测试完成后移走 |

本轮验证生成的两份 Windows 二进制也放在同一隔离目录：225,146,616 字节。隔离目录最终有 26,179 个文件、553,438,360 字节，另收纳空的 `bin/`、`build/` 目录。搬迁后仓库在新增本记录前为 497 个文件、94 个目录、8,871,489 字节，没有空目录；最终文件数因本记录增加 1。

所有移动目标先确认绝对路径、版本控制状态、用途和引用；没有移动 `.git`、全局 Codex 状态、活跃数据库或用户配置。

## 保留与目录整顿

`build/` 混有 9 份私有运维脚本、配置和备份，共 21,543 字节。它们没有仓库引用或 `build/` 相对路径依赖，已移到 `.local/ops/`，移动前后 SHA-256 一致。`.gitignore` 和 `.dockerignore` 同时忽略 `.local/`，防止私有资产进入提交或 Docker `COPY . .`。Docker 还排除 `build/`、Python 缓存和 TypeScript 增量缓存；容器内仍按既有入口重建 `build/x-ui` 与 `build/bin`。

保留以下有效资产：

| 路径 | 原因 |
|---|---|
| `web/ui/` | 受版本控制的 Go embed 编译输入，不是可删缓存 |
| `frontend/public/` | 前端静态输入，含生成的 OpenAPI JSON、favicon 和 QRious |
| `media/`、`docs/assets/`、前端 logo | 有真实引用；两张文档图片是历史审计附件 |
| `.env`、`.vscode/` | 用户本地配置；不读取或公开敏感值 |
| `x-ui/x-ui.db` | 本地 SQLite 数据库，未改写或移动 |
| `docs/private-ops/` | 私有历史现场记录，未公开或改写 |
| `.local/ops/` | 本轮整顿后保留的私有运维资产 |

已跟踪文件中只有 favicon、OpenAPI JSON、QRious 三组相同哈希，均为 `frontend/public` 源输入与 `web/ui` 交付副本。没有删除有效重复副本。根 npm 的 2 个开发依赖、前端 8 个运行依赖和 13 个开发依赖都有使用或工具链证据；`npm ls --all --json` 没有 problems，未删除依赖声明。

Go `tidy -diff` 仅发现 protobuf 被源码直接引用却标为间接依赖；已移除对应 `// indirect`，版本和 `go.sum` 没有变化。未修改业务逻辑、升级依赖或批量格式化源码。

## 逐文件文档核对

| 文档 | 核对依据与处理 |
|---|---|
| [architecture.md](architecture.md) | Main/Web/Sub/Repository/Core/任务源码；修默认路径、启动顺序、当前 SDK、V1、订阅和条件调度 |
| [development.md](development.md) | Go/npm/Vite/Config/CI；修 Node 要求、OpenAPI 生成、两个锁文件、Windows 路径与编译时 embed |
| [deployment.md](deployment.md) | install/管理脚本、Docker、service、设置默认值；修安装四分支、默认路径、格式开关、构建与停服备份 |
| [modules.md](modules.md) | Config/Main/Repository/Controller/前端路由；修模块落地状态、资源 hash、SDK、鉴权与启动顺序 |
| [api.md](api.md) | 所有 Web Controller、Session、Middleware、Sub；修 CSRF、Cookie、前缀、字符串客户端 ID、公开 V1 和无 HTML 订阅 |
| [panel-api.yaml](openapi/panel-api.yaml) | `/panel/api/*` 全量路由和实体；修 Core ID、嵌入客户端 ID、协议枚举、全局 `-1`、Geo/证书/CSRF/模型字段，覆盖 75 个设置字段 |
| [api-v1.yaml](openapi/api-v1.yaml) | V1 Controller、Health、Metrics、APIResponse；修默认前缀、公开访问、限流、HTTP 200 和指标容量语义 |
| [inbound-creation-guide.md](inbound-creation-guide.md) | Vue 入站/客户端表单、协议工具、模型与校验；移除旧 UI 控件步骤，按实际字段和导出方式重写 |
| [runtime-reliability.md](runtime-reliability.md) | HTTP/Cron/Geo/流量事务；补超时、调度与条件任务，保留内存重试和回滚限制 |
| [backend-internal-rebaseline-contract.md](backend-internal-rebaseline-contract.md) | Repository、baseline 模型、APIResponse；区分 `internal/` 目标与当前根级目录，修已落地边界 |
| [legacy-vue-ui-parity-audit.md](legacy-vue-ui-parity-audit.md) | 当前 Vue 路由、Views、SDK、Go 路由；保留历史记录，补 API Docs、真实表单和退役范围 |
| [ui-ux-audit.md](ui-ux-audit.md) | Router、Layout、样式与已有测试；标记历史视觉结论，替换失效源码链接，补当前对照 |
| [ai-routing-and-residential-egress.md](ai-routing-and-residential-egress.md) | Gateway/Xray 工具与保存流程；修实际 API、生成/规则顺序、占位出口、CSV 和订阅语义 |
| [SUPERXRAY_GUI_VPN_LAYER_HANDOFF_CN.md](SUPERXRAY_GUI_VPN_LAYER_HANDOFF_CN.md) | MVP/Core/模板源码；区分当前 5 口 SOCKS/CSV 和外部治理目标、历史现场与截图 |
| [server-inbound-subscription-audit.template.md](server-inbound-subscription-audit.template.md) | Sub/诊断/Gateway；补版本、路径、GET、证据视角与未验证项 |
| [passwall-openwrt-ai-routing-playbook.md](passwall-openwrt-ai-routing-playbook.md) | 现场与仓库职责核对；保留用户原有“实测现状”变更，补历史、fw4/nft、GET trace 和推断限制 |
| [operations-ai-routing-overview.md](operations-ai-routing-overview.md) | 部署/MVP/验证脚本；移除硬编码公网地址，区分显式 SOCKS 与 WiFi 透明代理，修探测命令 |
| [gateway-egress-timeout-remediation-runbook.md](gateway-egress-timeout-remediation-runbook.md) | MVP 和真实验证脚本；修先替换占位再保存，说明脚本与外部 job 接口范围 |
| [upstream-sync-policy.md](upstream-sync-policy.md) | 当前源码、依赖、CI、本地 Git refs；区分供应方历史基线和当前状态，不假设远端已同步 |
| [upstream-alireza0-v1.11.4-integration.md](upstream-alireza0-v1.11.4-integration.md) | 历史 tag/ref、现存源码；标记失效 bundle/worktree 路径和历史验证性质 |
| `assets/xray-mvp-desktop.png`、`assets/xray-mvp-mobile.png` | 仍被交接/UI 审计引用；保留图片并注明历史日期 |
| `private-ops/` 两份记录 | 严格 UTF-8/历史性质/仓库路径只读核对；不公开内容，不声称重新验证服务器 |

面板 YAML 覆盖 74 个操作（25 GET、49 POST），与实际路由比较遗漏 0、陈旧 0；范围外 20 个认证/兼容/CSRF 操作由 API 文档逐项说明。Web Engine 合计 112 个注册操作，其中 UI/静态 13、V1 3、WebSocket 1、DevTools 1；独立订阅服务另计。DevTools 路由不带 `webBasePath`，业务路由带配置前缀。

`tools/openapiexport` 已生成最终 JSON，两份 `frontend/public/openapi.json` / `web/ui/openapi.json` 与 YAML 语义及 SHA-256 对齐，包含最终 `EmbeddedClientIDPath` 修正。前端已从源码重建，bundle 未产生额外差异；最终 YAML 参数修正另走导出入口同步 JSON。

## 验证结果

验证环境：Windows PowerShell 7.6.6、Go 1.27.0、Node.js 22.23.2、npm 10.9.8、MinGW GCC。项目声明 Go 1.26.5，未修改工具链声明。

| 检查 | 结果 |
|---|---|
| `go test ./...` | 清理前后通过；移走 node_modules 后不再扫描 npm 包内 Go fixture |
| `go vet ./...`、`go mod tidy -diff` | 通过，依赖元数据无剩余差异 |
| `go build ... ./main.go` 与二进制 `-v` | 通过，输出 `3.4.3`；验证二进制在仓库外隔离目录 |
| Panel/V1 路由、响应、鉴权契约测试 | 通过；现有 Controller/OpenAPI 导出测试覆盖此次契约修正 |
| 前端 `npm run typecheck` | 通过 |
| 前端 `npm run lint` | 退出 0，无错误；XrayView 有既存格式警告 |
| 前端 `npm run test` 与清理后 Node 直跑 | 125 项通过，0 失败、0 跳过 |
| 前端 `npm run build` | 通过，运行 OpenAPI 导出、vue-tsc、Vite；仅两个 JSON 有实际生成差异 |
| `python scripts/secret_scan.py` | 通过；只扫描 tracked/非 ignored 文件，不把私有资料纳入公开文档 |
| `release_gate.py --ci --metadata-only` | 最终通过，包含版本、Go 元数据和生成 OpenAPI 检查 |
| `git diff --check` | 通过 |
| 本地 Markdown 链接、JSON 示例、YAML 解析 | 无失效目标；原有 30 个 JSON 示例均可解析，两个 YAML 可解析 |
| 编码/BOM/换行/乱码检查 | 原有公开文本保持 UTF-8 无 BOM、LF；无乱码标记 |
| 独立 Go/代码/文档复审 | 已反馈问题全部修正，没有未解决的 CRITICAL/HIGH |

已发现且未扩展修复的既存问题：

- `npm run format` 退出 1：13 个原有前端文件不符合 Prettier。此次未改这些源码，避免无关批量格式化。
- 本机 staticcheck 版本不兼容 Go 1.27 export data（最大支持版本 2、实际 4），未获得有效扫描结果。
- 未运行真实环境 Playwright E2E、视觉/无障碍验收、Linux amd64/arm64 打包、Docker 部署、服务器/OpenWrt/外部 Gateway/供应商探测或远端上游 fetch。文档不将其描述为本轮通过。

## 重建与回滚

前端完整构建需要在 `frontend/` 重新执行 `npm ci`；根目录的 E2E 依赖单独执行根 `npm ci`。Go 工具链缓存和 npm 全局缓存未处理。面板构建用 `go build -o bin/SuperXray.exe ./main.go` 重建本地输出，运行核心需按平台另外准备。

需要恢复隔离产物时，仅将指定子目录或文件移回原路径，并先确认没有新文件与之冲突；不要整体覆盖当前仓库。私有资产可从 `.local/ops/` 按需恢复到旧位置，但不得加入 Git 或镜像上下文。文档/契约回滚时同步恢复 YAML 与两份生成 JSON，并再次运行契约和元数据检查；保留用户原有 Passwall 变更。
