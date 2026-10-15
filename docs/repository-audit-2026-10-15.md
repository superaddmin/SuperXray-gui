# 仓库资源清理与文档核对记录（2026-10-15）

本轮对应 `integration/3x-ui-v3.10.2`，HEAD 为 `d87b7c157f3c764156f0d1346cb2302517daff66`，产品版本 `3.4.3`，Go `1.27.2`，目标 Xray `26.10.10`。核对对象包括此前尚未提交的上游适配及 Docker/测试修复；未回退这些改动，未提交、推送或发布。

## 范围与判定方法

遍历仓库全部可见、隐藏和 ignored 目录，检查文件数量、体积、链接和工作区进程；不进入仓库外部链接，也不清理 `.git` 内部对象。源码和资源按实际 import、动态 import、Go embed、运行路径、构建/发布脚本及文档引用判断用途；依赖按模块工具、锁文件、直接引用和 peer 契约核对。不能仅因没有直接文本命中就删除资源。

清理前较大的目录为：`tmp/` 约 787.99 MiB、前端目录（含安装依赖）约 196.62 MiB、`bin/` 约 150.41 MiB、`.local/` 约 149.95 MiB、根 `node_modules/` 约 13.28 MiB；`.git/` 约 299.42 MiB，保留。`.local/ops/` 是私有运维资产和备份，不按体积当作垃圾。没有发现需要跨越的 reparse point，搬迁前未发现从工作区启动的可执行进程。

清理前公开 `docs/` 共 29 个文件：21 份 Markdown、2 份 OpenAPI YAML、4 份固定基线 TSV、2 张历史截图，另有 2 份 ignored 私有记录。公开 Markdown/YAML 全文核对；TSV 与固定 Git refs 重新比较；图片核对引用和历史性质。新增本记录后公开文档为 30 个文件（22 份 Markdown）。私有记录只核对编码、结构和历史性质，不公开内容，不把旧现场记录重写成当前生产验收。

## 可恢复清理结果

永久删除操作在执行前被自动审批拒绝，返回 `blocked by policy`，没有提供进一步原因。未尝试绕过该限制；改为将确认可重建的产物移到仓库外独立目录：

```text
F:/SuperXray-gui-cleanup-20261015-v3102/
```

**这是移出工作区的可恢复隔离，没有释放磁盘空间。** 原路径、文件数和字节数分别记录在隔离根目录的 `manifest.json`、`post-validation/manifest.json` 与收尾批次 `final-generated/manifest.json`。每次移动前检查绝对路径边界、reparse point、Git 跟踪状态和目标不存在，保持相对目录，不覆盖其他隔离批次。

| 原路径 | 文件数 | 字节 | 处理依据 |
|---|---:|---:|---|
| `tmp/` | 34 | 826272044 | 集成检查生成的 Linux/arm64 二进制、工具和中间产物；历史哈希及命令保留在集成记录 |
| `playwright-report/` | 1 | 562584 | 已完成的隔离 E2E 浏览器报告，不作为源码提交 |
| `test-results/` | 1 | 45 | Playwright 上次运行状态 |
| `.codex/skills/superxray-project-context/scripts/__pycache__/` | 1 | 32771 | Python 可重建字节码 |
| `frontend/node_modules/.vite-temp/` | 0 | 0 | 空 Vite 临时目录 |
| 两份 `frontend/tsconfig.*.tsbuildinfo`（首次清理） | 2 | 1487 | TypeScript 增量缓存 |
| `web/job/log/`（首次清理） | 1 | 1002 | 测试生成日志 |
| 根 `node_modules/` | 630 | 13927758 | 根锁文件可重建的 E2E 依赖安装目录 |
| `frontend/node_modules/` | 26176 | 205148396 | 完成前端验证后移走；包声明和锁文件保持不变 |
| 两份 `frontend/tsconfig.*.tsbuildinfo`（验证后） | 2 | 1487 | 本轮 typecheck/build 再生缓存 |
| `bin/SuperXray.exe` | 1 | 118932431 | 已验证的本地面板构建产物；新文档嵌入版编译到仓库外 |

上述两批合计 **26849 个文件、1164880005 字节**。最终契约修正后再次构建和测试，另隔离两份 TypeScript 缓存及一份测试日志，共 3 个文件、2699 字节，记录在 `final-generated/manifest.json`。三批总计 **26852 个文件、1164882704 字节（约 1110.92 MiB）**；再次生成的文件单列，不计入首次存量。

逐项重新读取三个 manifest 并核对隔离目标的文件数量和字节数，全部一致。最终工作区排除 `.git` 共 565 个文件，包含保留的运行核心、私有运维资产和本地数据库；没有读取失败、reparse point、空目录，常见临时/缓存候选扫描未发现剩余的公开工作区产物。私有资产中的备份文件按保留策略留在原位。

## 保留的有效资产与依赖

| 路径或类别 | 保留理由 |
|---|---|
| `.git/`、`.vscode/`、`.env` | Git 历史和用户配置；不读取或公开敏感配置值 |
| `x-ui/x-ui.db`、`.local/ops/`、`docs/private-ops/` | 用户数据、备份和私有历史，不属于可删除缓存 |
| `bin/xray-windows-amd64.exe` | 已校验的运行核心及真实配置测试依赖，不能由面板 Go build 重建 |
| `web/ui/` | `web/ui.go` 的 `//go:embed ui` 输入；生产编译需要，保留并通过 Vite 重建 |
| `frontend/public/` | 前端静态输入；OpenAPI JSON 从 YAML 生成，复制到 `web/ui` |
| `media/` 的 17 个文件、`docs/assets/` 的 2 张图片 | 全部有引用，无可删重复图片；文档截图保留原日期 |
| favicon、QRious、OpenAPI 的两份副本 | 前端输入与 Go embed 交付副本，内容相同是构建要求 |
| 翻译 TOML、`tools/openapiexport`、部署及验证脚本 | embed、运行时或构建调用有效，不能按普通未引用文件处理 |
| 四份上游 TSV、历史审计和规划 | 可复核的来源与决策记录；用日期和性质区分现行说明，不删除历史证据 |

依赖审计没有发现可证明未使用的依赖声明：`go mod tidy -diff` 无差异，`go mod verify` 通过；根和前端 `npm ls --depth=0` 均无 missing/extraneous，两份 lock v3 的根依赖与各自 `package.json` 一致。根 2 个开发依赖、前端 8 个运行依赖及 13 个开发依赖均有引用或工具链用途，`vue-eslint-parser` 还承担 `eslint-plugin-vue` 的 peer 契约。

前端 TS/Vue/CSS/SVG 从 `main.ts` 的静态/动态导入可达；Go 应用包有运行入口，导出工具由前端 build 调用。`web/ui` 构建产物未发现孤立资源。旧 `web/html`、`web/assets` 已不存在，保留的 legacy 命名测试实际验证 Vue 或退役边界。未为清理而升级依赖、删除功能或重写业务逻辑。

## 逐文件核对矩阵

| 文档 | 代码依据与本轮处理 |
|---|---|
| [README.md](README.md) | 更新分支/版本/核心、当前审计和上游清单入口，区分现行、历史和外部运维资料 |
| [architecture.md](architecture.md) | Main、Web、Sub、数据库维护锁及 Xray 服务；修正 `service.InitDB`、统一事务和生命周期边界 |
| [modules.md](modules.md) | Config/Repository/Controller/Service；补候选导入、兼容层、可信代理、核心安装预检 |
| [api.md](api.md) | Controller/Entity/Middleware/Sub 全量路由；修 TOTP、设置更新、流量覆盖、Xray 包装格式、导入恢复 |
| [openapi/panel-api.yaml](openapi/panel-api.yaml) | 面板 74 操作契约；补 `twoFactorCode`、ECH 必填 SNI、流量覆盖/缺省归零、核心和导入失败语义，以及数据库下载的 HTTP 200 JSON 错误响应 |
| [openapi/api-v1.yaml](openapi/api-v1.yaml) | Health/Metrics/Middleware；明确可信代理和进程内指标生命周期 |
| [runtime-reliability.md](runtime-reliability.md) | Job/Database/Xray process；修预检、原子替换、失败回滚和 500ms 观察限制 |
| [backend-internal-rebaseline-contract.md](backend-internal-rebaseline-contract.md) | 当前事务及候选迁移已实现；未来 `internal/` 和 down runner 保持目标性质 |
| [development.md](development.md) | Go/npm/Vite/Docker/CI、测试源码；更新工具链、双锁文件、真实核心环境变量和重建步骤 |
| [deployment.md](deployment.md) | install/update/x-ui、DockerInit、systemd、设置服务；修备份、环境配置保留、代理和容器构建说明 |
| [inbound-creation-guide.md](inbound-creation-guide.md) | InboundsView/inboundCompat/协议工具/订阅；修 XHTTP/finalmask/TLS/Reality 与保存校验 |
| [ai-routing-and-residential-egress.md](ai-routing-and-residential-egress.md) | Xray 模板保存链及 Gateway 工具；补候选核心预检和现场验证边界 |
| [SUPERXRAY_GUI_VPN_LAYER_HANDOFF_CN.md](SUPERXRAY_GUI_VPN_LAYER_HANDOFF_CN.md) | 当前 SOCKS/CSV MVP 与外部治理设想分开；修核心升级及模板预检说明 |
| [gateway-egress-timeout-remediation-runbook.md](gateway-egress-timeout-remediation-runbook.md) | Gateway 工具、脚本实际参数、核心校验；区分配置解析、启动与真实出口连通性 |
| [passwall-openwrt-ai-routing-playbook.md](passwall-openwrt-ai-routing-playbook.md) | 保留历史现场证据，说明 GET trace 与仓库无法验证的 OpenWrt 当前状态 |
| [operations-ai-routing-overview.md](operations-ai-routing-overview.md) | 部署/MVP/出口验证脚本；更新预检边界，不把外部 Gateway 当仓库内建能力 |
| [server-inbound-subscription-audit.template.md](server-inbound-subscription-audit.template.md) | 实际订阅/诊断/迁移路径；补升级前后校验及明确未验证项 |
| [legacy-vue-ui-parity-audit.md](legacy-vue-ui-parity-audit.md) | Router/Views/SDK/后端；补 TOTP、配额、协议迁移及当前功能缺口 |
| [ui-ux-audit.md](ui-ux-audit.md) | 当前 Vue 源码与测试对照；保持 2026-05-16 截图和视觉结论的历史边界 |
| [upstream-sync-policy.md](upstream-sync-policy.md) | 固定 refs、当前代码与测试；更新 3x-ui 基线及已移植/未采用边界 |
| [upstream-alireza0-v1.11.4-integration.md](upstream-alireza0-v1.11.4-integration.md) | 重核 tag/backup ref/push 配置；历史分支、worktree、bundle 不存在，不再宣称可用 |
| [upstream-3x-ui-v3.10.2-integration.md](upstream-3x-ui-v3.10.2-integration.md) | 保留上轮验证和哈希，更新产物已隔离的位置与证据性质 |
| [repository-audit-2026-10-14.md](repository-audit-2026-10-14.md) | 加历史标识和新记录链接，保留当时工具链/数量/未完成项，不冒充当前状态 |
| 四份 `upstream-3x-ui-v3.10.2-*.tsv` | 131 本地提交、973 上游提交、1184/1978 两侧文件差异、2522 直接树差异逐字段复核一致 |
| 两张 `assets/xray-mvp-*.png` | 保留有引用的历史审计附件；本轮没有重新截图或视觉验收 |
| `private-ops/` 两份记录 | 私有历史仅做脱敏结构/编码检查，保留原样，未重新访问服务器验证 |

TSV 使用不可变 refs：共同基线 `b5ef412b8db999a6b981e1550762551d7b06540f`、本地 `d87b7c157f3c764156f0d1346cb2302517daff66`、上游 `608ef4e4010bf83e2e78328e08f77efc051ad4d7`。提交 ID/日期/标题、文件状态/旧路径/新路径均重新计算比对，所有 `review_state` 仍为 `inventory_only`，不把盘点等同于 973 条提交全部语义审查或合并。

## 本轮验证

验证在 Windows PowerShell、Go `1.27.2`、Node `22.23.2` / npm `10.9.8` 下执行。Go 真实核心测试使用保留的 `bin/xray-windows-amd64.exe`，不连接或改写本地业务数据库。

| 命令或检查 | 结果 |
|---|---|
| `go mod tidy -diff`、`go mod verify` | 通过，无依赖元数据差异 |
| 根及前端 `npm ls --depth=0`、锁文件声明对比 | 通过，无缺失/多余包 |
| `go test ./web/controller ./tools/openapiexport -run OpenAPI -count=1` | 通过，覆盖 Panel 74 操作及 V1 3 操作的路由和安全契约 |
| 前端 `npm run format`、`npm run typecheck` | 通过 |
| 前端 `npm run lint` | 退出 0：0 errors、176 条既有 Vue 样式 warnings；未新增批量源码格式改动 |
| 前端 `npm run test` | 144/144 通过，0 失败、0 跳过 |
| 前端 `npm run build` | 通过；按 YAML 最终版本重新导出 OpenAPI JSON，执行 vue-tsc 和 Vite，保留 `web/ui` |
| `$env:SUPERXRAY_TEST_XRAY_BINARY='F:/SuperXray-gui/bin/xray-windows-amd64.exe'; go test ./...`、`go vet ./...` | 全量通过；清理 npm 安装目录后不再扫描第三方包内 Go 示例 |
| `go build -o <隔离目录>/verification/SuperXray.exe ./main.go`、二进制 `-v` | 通过，输出 `3.4.3`；最终 OpenAPI 生成后重新编译，产物未放回仓库 |
| `release_gate.py --ci --metadata-only` | 通过；首次检查与 OpenAPI 导出同时运行曾读到旧文件，生成完成后顺序执行通过 |
| `python scripts/secret_scan.py`、`git diff --check` | 通过 |
| Markdown 本地链接、JSON 示例、编码与生成物 | 161 个本地文件链接有效；30 个 JSON 示例可解析；28 份公开文本严格 UTF-8 无 BOM、无替换字符；Markdown/YAML 保持 LF，TSV 保留原换行（其中两份 CRLF） |
| 独立文档/契约/清理审查 | 发现的部署备份、环境配置覆盖、数据库下载响应及认证/指标文案问题均已修正并复核，无剩余阻断项 |

两份生成 OpenAPI JSON 字节相同，SHA-256 为 `153df0be62f4dbfe3de31b8366f8affb64c53428217787c3bf374887a3b8571b`，所有内部 `$ref` 可解析。最终 Windows 验证二进制为 118936527 字节，SHA-256 为 `782649eba2e24dad002c0b5a7f29b9a9db975a347823df6853bd3841ce8f9f9c`，存于隔离目录的 `verification/`，不计入工作区搬迁数量。

上次集成任务的 Linux race、双架构 Docker 构建、QEMU arm64 启动、19/19 浏览器 E2E 和安全扫描是已完成的历史验证，具体命令、摘要及限制见集成记录。本轮文档及生成契约更新未重复运行全部 Docker/E2E；本机已有镜像未因文档更新重建，不把旧镜像声称为包含本轮 OpenAPI 文档。

## 当前边界与恢复方式

审查发现的现有实现边界已如实写入对应文档：WireGuard 共享客户端表单显示 Comment，但 payload 删除 `client.comment`，不支持 per-peer 备注持久化；Xray 模板和 `outboundTestUrl` 是分开保存，后者错误未向控制器调用方返回，不能承诺两项原子更新。这些不是文件清理引入的变化，本轮未扩大业务修改范围。

文档核对以本地代码、契约与测试为依据；历史截图、私有运维记录和服务器/OpenWrt/外部 Gateway 的实时状态不具备本轮现场证据，已明确日期或使用范围。没有将这些未验证内容纳入“100% 一致”的绝对保证。

重新开发时，在根目录和 `frontend/` 分别执行 `npm ci` 安装各自依赖；构建前端运行 `npm run build`，面板可用 `go build -o bin/SuperXray.exe ./main.go` 重建。Go/npm 全局缓存、Docker 镜像/构建缓存和 Playwright 全局浏览器未清理。若恢复隔离文件，应按对应 manifest 选择原相对路径，先确认目标不存在，避免整体覆盖新工作区；私有用户数据一直保留原位。文档回滚需同步 YAML、生成 JSON 及前端构建，禁止用整个工作区回退覆盖此前业务修改。
