# 仓库资源清理与完整性核对记录（2026-10-22）

本轮基于提交 `dba9c33935653955f716c8ff8db7d1e3413125aa`，产品版本 `3.4.3`，Go `1.27.2`，Node `22.23.2` / npm `10.9.8`。目标是复核 2026-10-15 清理后的工作区，识别新增的垃圾、临时、构建、缓存、日志、重复、过期和无引用文件，并在删除后重新验证安装、构建、测试与运行入口。

## 范围与判定方法

遍历仓库全部可见、隐藏和 ignored 内容（不进入 `.git` 内部对象），按以下证据判定：

- 版本控制：`git ls-files`、`git status --porcelain`、`git status --ignored`。
- 构建与依赖：`go.mod`、`go.sum`、两份 `package-lock.json`、`frontend/vite.config.ts`、Docker 与 CI 配置。
- 代码引用：Go import、Go embed、前端静态/动态 import、Vue Router、测试入口和运行时路径。
- 文档引用：Markdown 本地链接、README 截图、`docs/assets/` 和历史审计记录。
- 文件属性：名称、扩展名、大小、修改时间、内容哈希、空文件和空目录。

不以“文本未命中”作为唯一删除依据；`web/ui/`、`frontend/public/` 等构建复制副本按 embed 与构建契约保留。

## 分类检查结果

| 类别 | 检查结果 | 处理 |
|---|---|---|
| 临时文件 | 无 `tmp/`、`*.tmp`、`*.temp`、`*.swp`、`*.orig`、`*.rej`；无空目录 | 无需删除 |
| 构建产物 | 旧部署目录中的 `x-ui-linux-amd64.tar.gz`、`Xray-windows-64.zip` 及其解压副本 | 删除，见下表 |
| 构建产物（受控） | `web/ui/` 62 个文件是 `web/ui.go` 的 `//go:embed ui` 输入 | 保留 |
| 缓存文件 | 工作区无常驻缓存；本轮验证生成的 `node_modules`、`.vite`、`*.tsbuildinfo` | 验证完成后移除 |
| 日志文件 | `web/job/log/log/3xui.log` 为本轮 Go 测试生成；`.local` 下 3 份旧部署客户端日志 | 前者删除；后者作为交付证据保留 |
| 重复文件 | 哈希比对仅发现 3 组：`frontend/public` 与 `web/ui` 的 favicon、QRious、OpenAPI JSON | 保留，属构建/embed 复制要求 |
| 过期文件 | 历史审计、上游 TSV、规划文档均有索引或代码引用；无失效运行配置 | 保留 |
| 无引用冗余 | `frontend/src` 全部从 `main.ts` 可达；`web/ui` 资源无孤立 chunk；`media/` 17 个文件与 `docs/assets/` 2 张图均被引用；文档本地链接全部有效 | 无需删除 |
| 依赖文件 | `go.sum`、两份 lock v3 与实际依赖一致；安装目录可重建 | 保留锁文件，移除安装目录 |

## 已删除文件

| 路径 | 文件数 | 字节 | 判定依据 | 删除风险 |
|---|---:|---:|---|---|
| `.local/ops/deployment-192.220.24.138-20261015/artifact/x-ui-linux-amd64.tar.gz` | 1 | 69,960,982 | 已完成旧主机部署的构建归档，可由源码重建 | 低；交付信息仍保存在同目录 `delivery.json` |
| `.local/ops/deployment-192.220.24.138-20261015/Xray-windows-64.zip` | 1 | 20,913,304 | Xray 官方发行包，可重新下载 | 低 |
| `.local/ops/deployment-192.220.24.138-20261015/xray-client/` | 9 | 66,334,634 | 上一条 zip 的解压副本，内容重复 | 低 |

合计 **11 个文件、157,208,920 字节（约 149.93 MiB）**。删除前确认无进程占用，且路径均在 Git 忽略范围内；未删除任何受版本控制的文件。

## 验证生成物清理

以下文件由本轮安装、测试和构建产生，属于可重建内容，验证完成后已移除，避免把依赖缓存和测试日志留在工作区：

| 路径 | 文件数 | 字节 |
|---|---:|---:|
| `frontend/node_modules/` | 26,176 | 205,148,396 |
| `node_modules/`（根目录 E2E 依赖） | 630 | 13,927,758 |
| `playwright-report/` | 1 | 526,152 |
| `web/job/log/log/3xui.log` | 1 | 1,212 |
| `frontend/tsconfig.app.tsbuildinfo` | 1 | 1,440 |
| `frontend/tsconfig.node.tsbuildinfo` | 1 | 47 |

合计 **26,810 个文件、219,605,005 字节（约 209.43 MiB）**。删除后重新执行 `npm ci` 即可恢复，锁文件未改动。

## 保留文件与依据

| 路径或类别 | 保留理由 |
|---|---|
| `web/ui/`（62 个文件） | Go 编译期 embed 输入；本轮重新构建后文件名与内容哈希未变化 |
| `frontend/public/` | 前端静态输入；favicon、QRious、OpenAPI JSON 与 `web/ui` 副本一致是构建要求 |
| `bin/xray-windows-amd64.exe` | 真实核心测试依赖，环境变量 `SUPERXRAY_TEST_XRAY_BINARY` 指向该文件 |
| `.local/ops/` 的 `delivery.json`、`validation.json`、订阅/凭据记录、脚本与备份 | 私有运维交付和历史证据，不属于源码或可删除缓存 |
| `docs/private-ops/`（2 个文件） | 被忽略的私有历史记录，不公开、不重写 |
| `x-ui/x-ui.db` | 旧本地数据库（1 个用户、1 条设置、0 个入站）；经确认保留原样 |
| `.env`、`.vscode/` | 本地配置与编辑器设置，均被忽略 |
| `media/` 17 个文件、`docs/assets/` 2 张图 | 全部被 README 或审计文档引用 |
| `docs/` 合并后的历史审计、四份上游 TSV、`plans/` | 可复核的决策与来源记录，有索引或代码依据 |
| `go.sum`、根与前端 `package-lock.json` | 依赖锁定文件，安装与构建必需 |

## 需继续人工确认的项目

| 路径 | 现状 | 说明 |
|---|---|---|
| `.local/ops/deployment-192.220.24.138-20261015/client-stderr.log` | 0 字节，保留 | 空日志无内容；本轮按保留日志记录的选项未删除，可后续单独清理 |
| `.local/ops/passwall-config-before-quic.bak` | 8,197 字节，保留 | 修改前的回滚备份，按运维备份策略保留 |
| `.local/ops/deployment-192.220.24.138-20261015/client-*.log`、`client-process.txt` | 约 1 KB，保留 | 旧部署验证输出，作为交付证据 |
| `bin/xray-windows-amd64.exe` | 38,781,440 字节，保留 | 若移除，真实核心相关测试将跳过或失败 |
| `x-ui/x-ui.db` | 65,536 字节，保留 | 不在当前运行路径；确认保留以避免丢失旧账户/设置 |

## 忽略规则更新

`.gitignore` 新增 `web/job/log/`，显式忽略测试生成的日志目录。其余本轮涉及的路径已被现有规则覆盖：`.local/`、`bin/`、`node_modules/`、`*.tsbuildinfo`、`*.log`、`*.tar.gz`、`*.db`、`docs/private-ops/`。`.dockerignore` 已排除 `.local/`、`bin/`、`node_modules/`、日志和私有运维目录，无需改动。

## 文档合并与代码对齐

docs 目录没有字节级重复文件，冗余主要来自主题重叠。本轮按确认的范围合并：

| 处理 | 文件 | 说明 |
|---|---|---|
| 合并 | `repository-audit-2026-10-14.md`、`repository-audit-2026-10-15.md` → `repository-audit-history.md` | 只保留时间线、关键数字、隔离目录和验证范围；旧两份删除 |
| 合并 | `passwall-openwrt-ai-routing-playbook.md` → `operations-ai-routing-overview.md` | 代理路径类型、历史案例、验证命令、常见误判与回滚并入总览；原文件删除 |
| 保留 | 其余当前实现、OpenAPI、上游记录和 UI 审计 | 职责不同，不按标题相似度删除 |

同时修正 3 处与代码不一致的陈述：

| 文档 | 原内容 | 修正 |
|---|---|---|
| `deployment.md` | 安装完成输出使用 `Username/Password/Port/Access URL` 英文标签 | 改为 `install.sh` 实际输出的中文标签 |
| `development.md` | 声称源码树 `bin/` 不提供运行核心 | 改为“`bin/` 被 gitignore，克隆不含核心；当前工作区保留本地测试核心” |
| `SUPERXRAY_GUI_VPN_LAYER_HANDOFF_CN.md` | 拓扑、`1080/8081`、HTTP 入口与当前 MVP 混用 | 明确当前仅生成 5 个 SOCKS 端口（`11801/11802/11803/11901/11981`）及对应 tag；`8081` 未生成 |

`docs/README.md` 索引和 `.codex/routing.toml` 的文档路由已同步。四份上游 TSV 是固定快照，仍保留旧文档名，文中已注明不代表文件当前存在。

## 本轮验证

| 命令或检查 | 结果 |
|---|---|
| `go mod verify` | 通过，所有模块校验成功 |
| `go mod tidy -diff` | 无差异 |
| `go vet`（32 个项目包，排除 `node_modules`） | 通过 |
| `go build -o <临时目录>/SuperXray.exe ./main.go`、`SuperXray.exe -v` | 通过，输出 `3.4.3` |
| `$env:SUPERXRAY_TEST_XRAY_BINARY=...\bin\xray-windows-amd64.exe; go test <32 个包>` | 全部通过 |
| 根目录 `npm ci` | 通过，安装 4 个 E2E 依赖 |
| `frontend/ npm ci` | 通过，安装 242 个包 |
| `npm run typecheck` | 通过 |
| `npm run test` | 144/144 通过，0 失败、0 跳过 |
| `npm run lint` | 退出 0：0 errors、176 条既有 Vue 样式 warnings |
| `npm run build` | 通过；重新生成 OpenAPI JSON 并构建，`web/ui` chunk 名称与提交版本一致，`git status` 无源码差异 |
| `npx playwright test --list` | 发现 19 个用例；未执行（需要运行中的面板与 `.env` 环境） |
| 前端无引用扫描 | 无孤立 `.ts`/`.vue` 文件 |
| `web/ui` 孤立资源扫描 | 无未引用 chunk、字体或 CSS |
| 文档本地链接检查（合并前后） | 全部有效；删除的 3 个文件无残留 Markdown 链接 |
| 文档-代码事实核对（Go 侧：架构/模块/API/可靠性/后端契约） | 未发现不一致 |
| 文档-代码事实核对（前端/脚本侧：部署/开发/入站/Gateway/UI） | 发现 3 处不一致，均已修正 |
| `python scripts/secret_scan.py` | 通过，未发现高风险密钥 |
| `git diff --check` | 通过，无空白错误 |

## 当前规模与边界

清理、验证生成物移除和文档合并后，工作区（不含 `.git`）为 **557 个文件、49,398,724 字节（约 47.11 MiB）**，其中 `.local/` 仅保留 22 个交付与备份文件；本轮开始前为 569 个文件、约 197.05 MiB。删除的 11 个旧部署文件均可通过源码构建或官方发行包重新获取；文档合并净减少 2 个文件（删除 3、增加 1）。

本轮未执行 Docker 多架构构建、未运行浏览器 E2E、未重新做视觉/无障碍验收；真实核心测试使用本地 `bin/xray-windows-amd64.exe`。上述未执行项不能由本轮的静态检查和单元测试替代。`docs/private-ops/` 与 `.local/` 私有内容仅按路径、结构和用途核对，未公开或改写其中凭据。

`release_gate.py --ci --metadata-only` 当前会把历史集成文档和上游 TSV 中的旧版本号判为 stale；这些内容是固定历史快照，本轮未改写，因此该门禁结果不作为本轮通过项。该误报与本次文档合并无关。
