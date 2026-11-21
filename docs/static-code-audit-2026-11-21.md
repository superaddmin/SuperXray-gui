# SuperXray 全项目静态代码审核与回归记录

审核日期为 2026-11-21，基线为 `main` 的 `dc24d7448`。下表的位置以修改前的提交和函数名为准，便于在文件移动或代码删除后定位。可用 `git show dc24d7448:路径` 查看原文。

按共同根因归并，已修复 38 组问题，高风险 5 组、中风险 21 组、低风险 12 组。影响范围最大的是 WebSocket 广播与来源校验，以及安装、升级和网关校验的失败处理。

## 范围与方法

| 对象 | 覆盖与检查方式 |
|---|---|
| Git 工作树 | 基线清单有 564 个跟踪文件。遍历根目录及 `.codex`、`.github`、`config`、`core`、`database`、`docs`、`frontend`、`logger`、`media`、`plans`、`scripts`、`sub`、`tests`、`tools`、`util`、`web`、`xray`。已有未跟踪的 `AGENTS.md` 是工作规范。 |
| Go | 216 个文件，含 102 个测试文件，覆盖 32 个项目包。按 go.mod 使用 Go 1.27.4。运行编译、vet、Staticcheck、带测试调用图分析、普通及竞态测试，人工复核进程、套接字、关闭顺序与平台专属入口。 |
| Vue、TS、JS | 58 个 TS、17 个 Vue，含前端测试、配置、Playwright 入口和浏览器测试。检查组件模板、路由、store、API、异步顺序和资源释放，使用 vue-tsc、ESLint、Prettier、Node 测试、Vite 构建和浏览器。 |
| Python 与配置 | 检查 10 个跟踪 Python 文件及其测试。解析 29 个 TOML、11 个 JSON、17 份有效 YAML、6 个 SVG、2 个 HTML，并检查 JSON/YAML 重复键。另有一份被误命名为 YAML 的 PR 模板，见下表。 |
| 系统和 CI | 审核 6 个跟踪 Shell 脚本、OpenRC、3 份 systemd unit、Dockerfile、Compose、5 个 GitHub Actions 工作流及发行检查入口。对安装、启动、更新和网关校验追踪返回码与失败路径。 |
| 本地私有运维文件 | 在 `.local/ops` 对 53 个 Shell 脚本、28 个 Python 脚本和 1 个 JS 文件分别执行语法检查，并解析 40 个 JSON。只离线检查，未执行私有运维命令。另有本地 service 配置。 |
| 生成物与依赖 | `web/ui` 的 66 个基线文件通过前端构建、Go embed 和浏览器消费路径验证。第三方压缩 JS、字体、图片、已安装 npm 依赖和预编译核心按来源及集成入口核对。未逐行审计第三方实现。 |

高风险表示可能中断服务、越过应有的来源判断或使交付验证失效。中风险表示特定有效输入、异常路径或并发顺序会产生错误结果。低风险包括可证明的死代码、局部性能问题和维护缺陷。工具告警经调用方、框架绑定或测试复核后才作为删除依据。

## 后端与运行时

| 风险 | 类型 | 基线位置 | 问题与修复结果 |
|---|---|---|---|
| 高 | 并发死锁与竞态 | `web/websocket/hub.go:135,187,225` | 广播遇到慢客户端时，工作线程可能等待主循环消费注销队列，而主循环正在等广播结束。按主题发送也可能与关闭发送队列并发。现在成员变更、非阻塞入队和断开共用同一把锁，覆盖了慢客户端和并发关闭。 |
| 中 | 连接生命周期 | `web/websocket/hub.go:84,355` | 独立注册、注销队列不能保证先后顺序，停机时仍可能留下客户端。注册和关闭现在同步修改状态，停机后注册会关闭客户端。 |
| 高 | 来源校验 | `web/controller/websocket.go:37` | 来源校验手工切分主机，遗漏端口、协议和 IPv6 边界。现在按请求的可信协议、主机与有效端口比较浏览器来源，覆盖默认端口和代理场景。 |
| 中 | 日志隐私 | `web/controller/websocket.go:157` | 调试日志写入客户端消息原文。现仅记录消息长度。 |
| 中 | 读取错误与并发 | `web/network/auto_https_conn.go:36,67` | 探测首包时丢失 EOF 或超时，多读者可能竞争首包缓存。现在保留已读字节及其读取错误，串行保护首包状态，零长度读取直接返回。 |
| 中 | TCP 分段与缓冲边界 | `web/network/auto_https_conn.go:36` | TCP 分段或请求头超过 2 KiB 时，明文请求可能无法获得 307 跳转。现在读取到完整请求头，最多 1 MiB，TLS 首包正常透传。net.Pipe 测试覆盖分段、4 KiB 首部和超过上界的数据回放。 |
| 中 | 进程生命周期／死字段 | `core/singbox/adapter.go:195,225` | Stop 返回后旧进程仍在退出，旧 Wait 结果可能覆盖新实例状态。现在以每次启动独立的退出通知协调 Stop、Start 和调用方取消，同时移除只写的 lastExit 字段。 |
| 低 | 死导入与不可达分支 | `main.go:12,69,78,109,118` | 移除无用途的 unsafe 导入及 4 个 log.Fatalf 后永远不能执行的 return。平台工具中的 linkname 导入有实际作用。 |
| 低 | 热路径性能 | `xray/log_writer.go:35,57` | 持锁写日志的热路径反复编译两条固定正则。改为包级编译复用。 |
| 中 | 测试时序 | `xray/process_lifecycle_test.go`，启动失败回滚用例 | race 版测试子进程的启动耗时偶尔超过 500 ms，导致误判为启动成功。让测试向私有启动函数传入观察窗口，生产维持 500 ms，退出用例设 15 s 上限并在子进程退出后立即返回。 |

## 前端、接口和浏览器测试

下表中 `src/` 与 `tests/*.test.ts` 相对于 `frontend/`。

| 风险 | 类型 | 基线位置 | 问题与修复结果 |
|---|---|---|---|
| 中 | 预设数据错误 | `src/utils/xrayCompat.ts:229` | AdGuard 家庭保护预设重复使用普通 DNS。核对服务商公布的地址后更正两条 IPv4 和两条 IPv6，测试确认写入实际 DNS 配置。 |
| 中 | 数据兼容与部分更新 | `src/utils/xrayProtocolTools.ts:300`、`src/views/XrayView.vue:2134` | WARP 读取错误的配置层级，数据缺失时会产生异常或只更新部分状态。现在兼容嵌套与历史扁平返回，验证 peer、地址和所需字段后一次更新。 |
| 中 | 异步响应竞态 | `src/views/LogsView.vue:141,155,164` | Xray 日志关键字需手动刷新，较早请求也可能覆盖最新结果或错误状态。加入防抖查询和请求序号，浏览器回归显式控制两次响应的先后。 |
| 低 | 输入边界 | `src/utils/inboundCompat.ts:2073` | IPv4 段之后仍带 IPv6 压缩符的非法地址被接受。现要求点分段位于地址末尾，测试覆盖合法和非法组合。 |
| 低 | 错误处理与资源释放 | `src/utils/textExport.ts:13` | 浏览器拒绝复制时仍报告成功，抛异常时遗留包含复制内容的文本框。现在检查返回值并在 finally 内清理，测试覆盖成功、拒绝与异常。 |
| 低 | 类型收窄与缺值 | `tests/e2e/legacy-panel.spec.ts:52,1347`（仓库根） | 响应助手类型排除了受支持的浏览器响应对象，可空 ID 也可能写入表单。收窄到实际使用的公共方法，写入前先验证 ID。 |
| 低 | 死代码 | `src/api/cores.ts:12`、`src/api/endpoints.ts:10`、`src/stores/server.ts:17`、`src/api/request.ts:56` | 删除无调用的前端 API 包装和路径构造器、只写与重复的 WebSocket 状态、行为相同的错误类型分支。后端路由继续提供服务。 |
| 低 | 代码格式 | `src/components/AppStatusBar.vue`、`src/views/XrayView.vue` | 修复既有 Prettier 不一致处，项目格式检查通过。 |

## 运维脚本、安装与 CI

| 风险 | 类型 | 基线位置 | 问题与处理 |
|---|---|---|---|
| 中 | 退出码覆盖 | `install.sh:116`、`update.sh:90`、`x-ui.sh:23` | awk 在 END 阶段覆盖匹配退出码，已占用端口仍被报告为空闲。改为保留匹配状态，覆盖空闲和占用端口。 |
| 高 | 误报校验成功 | `scripts/gateway-egress-validate.sh:332` | Python 回退分支在子 shell 内累加失败数，主脚本可能返回成功。调整数据传递和失败传播，覆盖配置异常及解析失败。 |
| 中 | 输入解析与访问范围 | 同脚本 `142,254,401` | IPv6 环回未被正确识别，TSV 空监听字段会发生列移位，非环回监听缺少 CIDR 时仍能通过。修正地址判断和字段解析，对缺失访问范围返回失败。 |
| 中 | 服务重载 | `x-ui.rc:15`、`x-ui.sh:430` | OpenRC reload 的参数与进程标识使用错误，Alpine 的 CLI 重载入口不可用。修正 init 脚本和 CLI 服务分支。 |
| 中 | 入口传参与架构 | `DockerEntrypoint.sh:7`、`DockerInit.sh:29` | 容器入口忽略传入命令，不支持的架构会落入错误默认值。入口现转交参数并保留默认启动，未知架构显式失败。 |
| 高 | 架构失败路径 | `install.sh:41`、`update.sh:69` | 架构探测失败可能删除当前脚本后继续安装。移除该删除操作并将架构错误返回到调用入口。 |
| 高 | 安装切换与回滚 | `install.sh:778,833,887`、`update.sh:704,765,805` | 下载或解压出错时可能先停止旧服务并丢失旧二进制。自定义安装目录末级名称还会导致归档路径错配。修复及失败回滚用例见验证记录。 |
| 低 | 死代码 | `install.sh:54,182,586`、`update.sh:82,157,555`、`x-ui.sh:40,46,624,1413` | 对照函数调用和脚本外部入口，移除 7 个无调用 helper、未使用 SSL 参数及无效局部赋值。 |
| 低 | 不可达条件与配置格式 | `.github/workflows/codeql.yml:15`、`.github/pull_request_template.yml:1` | 去掉语言矩阵无法选中的 Swift runner 分支。PR 模板内容为 Markdown，保留原文并改为 GitHub 识别的 .md 扩展名。 |

## Python 工具和本地脚本

下表中的 `render_subscription_delivery.py` 和 `validate_deployment_outputs.py` 位于 `.codex/skills/superxray-server-deployment/scripts/`。

| 风险 | 类型 | 基线位置 | 问题与修复结果 |
|---|---|---|---|
| 中 | 文件路径枚举 | `scripts/secret_scan.py:124,138` | Git 默认会给部分文件名加引号并转义，按行拼回路径会漏查中文文件。使用 NUL 分隔的 Git 原始路径，检查退出码，新增中文文件名测试。 |
| 中 | 凭据漏扫 | `scripts/secret_scan.py:188,231` | 按整行判断示例 URI，同行真实匹配可能一并跳过。现在针对每个匹配分别核对主机或示例值，加入同一行混合输入的回归。 |
| 低 | 遍历与文件边界 | `scripts/secret_scan.py:143,167` | 全目录扫描先枚举所有依赖目录，还可能把目录当成文件或读取符号链接目标。现在遍历时剪枝，读取前检查文件类型。原 os 导入现用于文件系统解码及遍历。 |
| 低 | 死导入 | `.codex/skills/superxray-project-context/scripts/validate_codex_config.py:6` | 去掉未使用的 sys 导入。 |
| 中 | URL 构造 | `render_subscription_delivery.py:34,59,92` | 面板路径使用字符串后缀截断，会截错带有 panel 后缀的目录，又无法处理不带末尾斜杠的标准路径。组成订阅地址时还遗漏裸 IPv6 的方括号。改为完整路径段判断，并按 URI authority 格式生成 IPv6 链接。 |
| 中 | 清单结构与无用字段 | `render_subscription_delivery.py:26,46,70,84,98` | 有完整 URL 的清单仍被强制要求提供 subId，形状错误或超大时间戳可能抛出未处理异常，非法端口可被默认值掩盖。统一直接链接和按 ID 回退的路径，校验容器、端口和时间范围，删除 4 个未进入交付文本的字段。 |
| 中 | 日志隐私 | `render_subscription_delivery.py:177` | 非 HTTPS 警告写出完整订阅 URL。现在只记录字段名，测试检查 stderr 不包含主机。 |
| 中 | 异常与数据泄露 | 两个交付脚本的日期、端口、URL 解析及 CLI 错误分支 | 原生异常会将异常输入回显到 stderr，大数毫秒计算还可能溢出。改为仅报告字段和原因，增加超大日期、IPv6、端口及错误输出脱敏回归。 |
| 中 | 类型边界／死返回值 | `validate_deployment_outputs.py:26,38,84,119` | 在检查类型前对 subscriptions 调用 get，非整数、非有限节点计数可通过，成功摘要输出受保护主机。先校验对象类型，拒绝异常计数，成功摘要仅保留状态，同时移除调用方总会丢弃的返回值。 |
| 中 | 脱敏校验 | `validate_deployment_outputs.py:76,98,110` | 脱敏校验只认占位符前缀，server.host 和 SOCKS URI 附带内容也可能越过校验。现检查完整值及 URI 组成部分，保留显式私有模式和历史占位符格式。 |
| 低 | 死导入 | `.local/ops` 的两份 `gen_secrets.py:5`、两份 `verify_public.py:3`、盘点重试脚本第 4 行、根层凭据查找脚本第 2 行 | 删除 6 个未使用的标准库导入，逐一核查动态调用与原始编码，Ruff 复扫通过。文件按原位置保留在 Git 忽略目录。 |

## 保留项与覆盖限制

- Go 的调用图还列出 7 个静态候选。HostProc 有 Linux 使用方。logger.Notice、logger.Noticef、reflect_util.GetFieldValues、global.GetSubServer、service.RemoveIndex、websocket.BroadcastInvalidate 均为导出 API，没有证据能排除仓库外调用，按兼容接口保留。未发现能够确证已经废弃并可安全删除的大段注释代码。
- 上一轮 ESLint 的 176 条模板排版告警已清零。XrayView 的 162 条通过调整模板布局修复。InboundsView 的 14 条涉及顶层逻辑表达式续行，Prettier 负责这类 `v-if` 表达式内部的缩进，ESLint 继续检查周围模板。现在 `npm run lint` 执行 `--max-warnings=0`，`npm run format` 和 lint 同时通过。修复前后 Vue 渲染函数一致，重新构建的 66 个文件逐个 SHA-256 相同。
- 额外开启 noUncheckedIndexedAccess 后有 42 个索引诊断。messages.ts 有 1 处、SettingsView.vue 有 2 处为必选正则捕获，core.ts 有 1 处非空数组前置条件，inboundCompat.ts 有 34 处固定长度及范围限制，xrayCompat.ts 有 4 处受下标或 findIndex 判断保护。若将此选项升为项目标准，可分别用定长元组、已缩窄的局部变量表达这些条件，再把开关加入 CI。
- 实际生产部署、云端发布、全部 Linux 发行版的服务管理器、第三方 WARP 账户注册和私有运维脚本的远程副作用，均不在本轮运行验证内。离线样例、真实本地核心和受控测试覆盖了相关代码路径中的一部分。

## 修复后验证

| 验证入口 | 实际结果 |
|---|---|
| Go 项目包 | 按 CONTRIBUTING.md 从 `go list ./...` 滤掉 node_modules。对 32 个包运行 `go test -count=1`、`go test -race -p 2 -count=1`、`go vet` 和 Staticcheck v0.8.2，通过。真实核心用例使用工作区的 Windows Xray 二进制。Linux/Darwin 的 util/sys 交叉 vet 通过。 |
| 并发与启动回归 | sing-box、WebSocket 和自动 HTTPS 的 `go test -race -count=3` 通过。Xray 启动失败、配置回滚两组测试以 `-race -count=10` 通过。新增分段和超限场景后再次运行网络包竞态测试、vet 和 Staticcheck，通过。 |
| 前端 | 在 frontend 运行 `npm run typecheck`、`npm test`、`npm run lint`、`npm run format`、`npm run build`。161/161 项测试通过，ESLint 错误为 0，Prettier 通过。Vite 重新生成 Go 嵌入的 web/ui，隔离面板的 Go 构建成功。 |
| 浏览器与真实服务 | 新建临时 SQLite、随机本地测试账号、独立 Xray 配置和订阅，面板及订阅仅监听环回地址。启用 mutation、restart、import 三项开关，订阅端点返回 HTTP 200 和非空内容。Chromium 的 22/22 项通过，含登录、日志乱序、协议配置、启停客户端、Xray 重启和备份导入。首轮暴露了新增测试定位符错误与夹具占用 SQLite，修正定位并关闭夹具连接后重新运行全部用例。 |
| Python | `python -m unittest discover -s tests -p 'test_*.py'` 通过 16 项。项目治理、发行门禁、交付工具的测试分别通过 9、20、7 项，合计 52 项。Ruff 的 F、E9、B、ASYNC、SIM、RET 检查通过，项目治理与 4 个技能格式验证通过。 |
| 安装和系统脚本 | Ubuntu 24.04 上 `bash tests/ops-scripts.sh` 通过，覆盖带空格的自定义安装目录、正常切换、损坏包、缺少二进制、切换失败恢复和自定义文件保留。同时验证 OpenRC/CLI、容器入口、ss/netstat、jq/Python 两种网关校验路径。8 份脚本语法和 ShellCheck error 级检查通过，3 个 systemd 模板的启动路径有断言。 |
| CI、配置、凭据 | 5 个 Actions 的 actionlint、`docker compose config`、`release_gate.py --ci --metadata-only`、`python scripts/secret_scan.py`、`git diff --check` 通过。本地编辑器 JSON、运行配置 JSON 和私有 systemd 配置也完成结构检查。 |

安装与更新均在目标文件系统中解包，检查必备二进制后才切换目录。切换前保存原目录、unit 和管理入口，已存在的自定义运行文件会合并到新目录。Arch 的默认服务路径已与脚本对齐，自定义目录会写入对应的 systemd/OpenRC 入口。两条安装路径均用自定义数据文件验证了成功迁移与失败恢复。两份包含空格路径的生成 unit 通过了 systemd-analyze verify。

## ShellCheck 后续修复与复核

ShellCheck 0.9.0 对同一组 8 份脚本按 `--severity=style` 复扫，error、warning、info 和 style 均为 0。修复涵盖参数引用、ACME 多参数数组、命令退出码、文件名展开和服务管理操作。框架消费的 OpenRC 配置和隔离测试替身在对应位置标明了调用关系，发行版的 os-release 按外部运行时输入处理。

复核时还修复了管理菜单重置账户时的双因素开关，以及脚本下载失败后仍可能执行部分内容的问题。隔离测试覆盖保留和禁用双因素的两种选择，以及下载中断时的退出行为。可从仓库根目录按 CONTRIBUTING.md 的 Shell and OpenRC 一节复查。

下表保留清理前 74 条 warning、124 条 info、73 条 style 的位置，便于与原审核记录核对。行号对应当时工作树。

| 规则 | 处理建议 | 文件与行号 |
|---|---|---|
| SC1007 | 核对等号周围空格是否有意给单条命令设置空变量。 | `tests/ops-scripts.sh:115` |
| SC1090 | 动态 source 路径按安装约定验证。 | `tests/ops-scripts.sh:10,32,75` |
| SC1091 | 为可选系统文件或测试夹具指定检查时可访问的路径。 | `install.sh:24,27`、`tests/ops-scripts.sh:95`、`update.sh:40,43`、`x-ui.sh:52,55` |
| SC2009 | 进程筛选可用 pgrep 的精确匹配。 | `x-ui.sh:724` |
| SC2016 | 确认单引号内的变量是模板文本或应立即展开。 | `scripts/gateway-egress-validate.sh:631`、`tests/ops-scripts.sh:79` |
| SC2034 | 确认 OpenRC、回调或被 source 的调用方是否读取该变量。 | `tests/ops-scripts.sh:34,35,72,96`、`x-ui.rc:4,6,7,8` |
| SC2035 | 文件名通配符使用显式相对路径，避免首字符被当成选项。 | `x-ui.sh:1585,1630` |
| SC2086 | 逐项区分单一参数与有意的分词，用双引号或数组表达。 | `install.sh:123,265,270,277,278,288,297,298,406,409,447,448,449,463,473,475,476,479,481,482`、`update.sh:89,232,237,244,245,255,264,265,371,374,412,413,414,428,438,439,440,443,444,445`、`x-ui.sh:34,211,217,351,831,832,834,835,846,847,902,903,905,906,917,918,1062,1079,1271,1276,1283,1284,1294,1303,1304,1436,1439,1476,1477,1478,1492,1502,1503,1508,1509,1585,1596,1599,1630,1631,1632,1648,1649,1650,1753,1900,1905,2108,2109` |
| SC2115 | 删除路径变量使用非空断言并检查目标目录。 | `x-ui.sh:217` |
| SC2119 | 核对省略函数参数是否有默认值。 | `x-ui.sh:2286,2295,2301` |
| SC2120 | 核对可选参数及所有调用方。 | `x-ui.sh:135,229,283` |
| SC2126 | 计数可直接使用 grep 的计数选项。 | `x-ui.sh:724` |
| SC2154 | 检查调用环境或初始化点，必要时显式设默认值。 | `tests/ops-scripts.sh:76,100,102` |
| SC2155 | 命令返回值参与判断时拆开声明和赋值。 | `install.sh:333,334,374,648,649,650,652,663,664,665,674,675,676,683,713,735,736`、`update.sh:298,299,339,623,624,625,638,639,640,650,685`、`x-ui.sh:297,305,306,307,308,314,1054,1071,1088,1108,1158,1159,1162,1337,1338,1408,1534,1535,2126,2127,2128,2135,2136,2137,2138,2139` |
| SC2164 | cd 失败时明确返回或退出。 | `x-ui.rc:13` |
| SC2181 | 在 if 条件中直接检查命令退出状态。 | `install.sh:169,192,273,316,341,407,471,572,901,918,929,959,974,985,994,1003,1026`、`update.sh:136,159,240,281,306,372,436,534,833,836,872,874,887,898,907,916,938`、`x-ui.sh:96,110,121,129,138,197,231,285,298,328,359,370,415,459,476,716,734,971,1025,1141,1183,1279,1343,1377,1437,1499,1546,1551,1576,1586,1600,1634,1643,1744,1766` |
| SC2236 | 使用直接的非空或空值判断。 | `install.sh:890`、`update.sh:824,827` |
| SC2317 | 核对 trap、外部回调、测试替身和间接调用后再判断可达性。 | `tests/ops-scripts.sh:49,50,81,82,97,98,100` |
