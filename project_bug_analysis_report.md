# SuperXray-gui 潜在 Bug 分析报告

生成日期：2026-05-07  
工作区：`f:\SuperXray-gui`  
审查方式：静态分析工具 + 人工代码审查  
报告性质：潜在问题清单，不代表所有条目均已被动态复现；优先级按可利用性、影响面、触发概率和修复成本综合排序。

## 1. 执行摘要

本次审查覆盖 Go 后端、订阅服务、面板 API、后台任务、Xray/Core 集成、Telegram 集成、Custom Geo、WebSocket、新 Vue 3 前端、旧版 HTML/Vue 2 页面、构建与验证配置。

静态工具整体结果较好：`go vet`、`go test`、`staticcheck`、`govulncheck`、`gosec`、前端 typecheck/lint/build/format、npm audit 均未直接报出漏洞或类型错误。但人工审查发现若干静态工具不易捕获的问题，集中在订阅生成、登录安全、请求头信任、后台任务 panic、前端随机数、资源上限和验证边界。

最高优先级建议先修复：

1. 登录失败路径泄露明文密码到日志和 Telegram。
2. 订阅服务把请求态写入共享 `SubService` 字段，存在并发串号/数据竞争风险。
3. 订阅链接和 Profile 头直接信任 `Host` / `X-Forwarded-*`，存在链接污染风险。
4. 订阅生成大量 unchecked type assertion 和索引访问，畸形配置可触发 panic。

## 2. 审查范围与方法

### 2.1 静态工具

| 类别 | 命令/工具 | 结果 |
| --- | --- | --- |
| Go 基础验证 | `go vet ./...` | 通过 |
| Go 单元测试 | `go test ./...` | 通过 |
| Go 静态检查 | `staticcheck -f=json ./...` | 通过，无有效 issue |
| Go 漏洞库 | `govulncheck ./...` | `No vulnerabilities found` |
| Go 安全扫描 | `gosec -fmt=json -exclude-dir=web/assets -exclude-dir=frontend/node_modules ./...` | 0 issues |
| Go Race 重点路径 | `go test -race ./core/... ./web/service ./web/job ./web/websocket ./xray` | 通过 |
| Go Race 订阅包 | `go test -race ./sub` | 通过，但现有测试未覆盖并发请求态 |
| Go 构建 | `go build -o $env:TEMP\SuperXray-review.exe ./main.go` | 通过 |
| 前端类型 | `cd frontend; npm run typecheck` | 通过 |
| 前端 lint | `cd frontend; npm run lint` | 通过 |
| 前端构建 | `cd frontend; npm run build` | 通过 |
| 前端格式 | `cd frontend; npm run format` | 通过 |
| 依赖审计 | 根目录与 `frontend` 执行 `npm audit --audit-level=moderate` | 0 vulnerabilities |
| E2E | `npm run e2e` | 失败，统一为 `net::ERR_CONNECTION_REFUSED`，本地 `127.0.0.1:2073` 服务未启动 |

### 2.2 人工审查重点

- 登录、会话、CSRF、CSP、Cookie、WebSocket。
- 订阅输出：plain/json/clash、Host 解析、外部代理、TLS/Reality/Hysteria/WireGuard。
- Custom Geo 下载、SSRF、路径边界、文件写入。
- LDAP、流量统计、Telegram、Xray lifecycle、CoreManager。
- Vue 3 前端 API SDK、运行时配置、随机值生成、设置表单。
- 工具链、忽略规则、验证范围污染。

## 3. 问题分类统计

| 分类 | 数量 | 最高风险 |
| --- | ---: | --- |
| 安全与隐私 | 7 | 高 |
| 并发/异步 | 2 | 高 |
| 边界条件/异常处理 | 5 | 高 |
| 性能/资源耗尽 | 3 | 中 |
| 配置/验证治理 | 3 | 中 |
| 第三方集成 | 2 | 高 |
| 工程维护/CI | 2 | 低 |

## 4. 优先级排序

| 优先级 | 问题编号 | 建议时限 |
| --- | --- | --- |
| P0 | BUG-001, BUG-002, BUG-003, BUG-004 | 立即修复 |
| P1 | BUG-005, BUG-006, BUG-007, BUG-008, BUG-009, BUG-010 | 下一个安全/稳定性迭代 |
| P2 | BUG-011, BUG-012, BUG-013, BUG-014, BUG-015 | 常规迭代 |
| P3 | BUG-016, BUG-017, BUG-018 | 工程治理窗口 |

## 5. 详细问题清单

### BUG-001 登录失败泄露明文密码

- 位置：
  - `web/controller/index.go:78`
  - `web/controller/index.go:81`
  - `web/controller/index.go:83`
  - `web/controller/index.go:90`
  - `web/service/tgbot.go:2938`
- 类型：安全与隐私 / 第三方通知集成
- 风险等级：高
- 证据：
  - `safePass := template.HTMLEscapeString(form.Password)`
  - `logger.Warningf("wrong username: \"%s\", password: \"%s\", IP: \"%s\"", safeUser, safePass, getRemoteIp(c))`
  - `notifyPass := safePass`
  - `a.tgbot.UserLoginNotify(safeUser, notifyPass, getRemoteIp(c), timeStr, 0)`
- 触发条件：任意失败登录，包括用户误输真实密码、撞库、2FA 失败前的密码校验失败。
- 影响：明文密码进入本地日志和 Telegram 管理员通知；如果日志被采集、备份、转发或 Telegram 账号泄露，会扩大凭证泄露范围。
- 修复建议：
  - 失败登录永远不要记录密码原文或转发到 Telegram。
  - 只记录用户名、来源 IP、失败原因枚举和时间。
  - 对 Telegram 通知统一传入固定脱敏值，例如 `***`。
  - 增加回归测试，断言失败登录日志/通知不包含提交的密码。

### BUG-002 订阅服务共享请求态导致并发串号

- 位置：
  - `sub/subService.go:27`
  - `sub/subService.go:30`
  - `sub/subService.go:45`
  - `sub/subService.go:59`
  - `sub/subService.go:560-563`
  - `sub/subService.go:1502-1518`
  - `sub/subController.go:59-77`
- 类型：并发 / 数据竞争 / 订阅业务逻辑
- 风险等级：高
- 证据：
  - `SubService` 持有 `address`、`datepicker` 字段。
  - 每个请求在 `GetSubs` 中执行 `s.address = host` 和 `s.datepicker = ...`。
  - 后续链接生成通过 `resolveInboundAddress` 读取 `s.address`。
  - `SUBController` 初始化时创建一个共享 `SubService`，所有订阅请求复用同一实例。
- 触发条件：多个客户端同时请求订阅，且请求 Host 不同，或一个请求正在生成链接时另一个请求覆盖 `s.address`。
- 影响：订阅链接中的服务器地址可能被其他请求串改；理论上还会触发 Go 数据竞争，现有 `go test -race ./sub` 没有并发请求用例，无法覆盖该路径。
- 修复建议：
  - 移除 `SubService.address` 和 `SubService.datepicker` 的请求态字段。
  - 将 `host`、`datepicker` 作为参数显式传入 `getLink`、`resolveInboundAddress`、`BuildPageData`。
  - 增加并发单测：并发请求不同 Host，断言生成链接不串号，并用 `-race` 验证。

### BUG-003 订阅 URL 信任 Host / X-Forwarded-Host 导致链接污染

- 位置：
  - `sub/subService.go:1381-1420`
  - `sub/subService.go:1425-1439`
  - `sub/subService.go:1447-1451`
  - `sub/subController.go:164-168`
  - `sub/subController.go:186-190`
  - `sub/subController.go:203-207`
  - `web/controller/util.go:84-95`
- 类型：安全 / Host Header Injection / 反向代理边界
- 风险等级：高
- 证据：
  - `ResolveRequest` 优先读取 `X-Forwarded-Host`，再读 `X-Real-IP` 和 `Request.Host`。
  - `BuildURLs` 在缺少显式 `subURI/subJsonURI/subClashURI` 时使用请求派生的 `hostWithPort`。
  - `Profile-Web-Page-Url` 也在缺少配置时由 `scheme://hostWithPort + RequestURI` 生成。
- 触发条件：攻击者直接访问订阅服务或穿透未清洗请求头的反向代理，并发送恶意 `Host` / `X-Forwarded-Host`。
- 影响：订阅 HTML 页面、订阅 URL、Profile 头可能指向攻击者域名，造成链接污染、钓鱼、客户端 profile 元数据污染；同时 `web/controller/util.go` 中旧页面 title/description 的 host 展示也可被污染。
- 修复建议：
  - 默认使用配置中的 canonical subscription origin；未配置时仅接受 `Request.Host` 且做严格 Host 校验。
  - 只有当请求来源是可信反向代理 IP 时才读取 `X-Forwarded-*`。
  - 对 Host 做 allowlist 或标准化校验，拒绝空白、控制字符、逗号、多 Host、非法端口。
  - 增加 Host header injection 回归测试。

### BUG-004 订阅生成存在大量 unchecked type assertion 和索引访问

- 位置：
  - `sub/subService.go:245-248`
  - `sub/subService.go:266-270`
  - `sub/subService.go:329-333`
  - `sub/subService.go:584-625`
  - `sub/subService.go:697-704`
  - `sub/subService.go:720-727`
  - `sub/subService.go:742-756`
  - `sub/subJsonService.go:216-220`
  - `sub/subJsonService.go:269-271`
  - `sub/subJsonService.go:323-331`
  - `sub/subJsonService.go:470`
- 类型：边界条件 / panic / 输入数据验证
- 风险等级：高
- 证据：
  - `findClientIndex` 找不到客户端时返回 `-1`，但 VMess/VLESS/Trojan 路径直接访问 `clients[clientIndex]`。
  - 多处直接执行 `stream["network"].(string)`、`settings["path"].(string)`、`a.(string)`、`requestPath[0].(string)`。
  - Reality `serverNames` / `shortIds` 如果类型不符或数组为空，存在 panic 风险。
  - JSON 订阅中 `streamSettings["tlsSettings"].(map[string]any)`、`stream["hysteriaSettings"].(map[string]any)` 直接强转。
- 触发条件：数据库中存在旧版、手工导入、API 写入、迁移异常或损坏的 `settings` / `streamSettings`；或者外部代理配置字段类型不符合预期。
- 影响：订阅请求可触发 500，严重时如果缺少顶层 recover 会影响服务可用性；公开订阅入口会放大可用性风险。
- 修复建议：
  - 引入集中 safe getter，例如 `getString`, `getMap`, `getStringSlice`, `firstString`。
  - 找不到客户端或字段缺失时跳过该节点并记录结构化 warning，而不是 panic。
  - 对导入、编辑和保存路径做 schema 校验，保证订阅层只处理规范数据。
  - 增加畸形 `streamSettings` / 空数组 / 缺客户端回归测试。

### BUG-005 JSON 订阅 externalProxy 会串联修改同一份 stream map

- 位置：
  - `sub/subJsonService.go:213-230`
  - `sub/subJsonService.go:232-252`
  - 对比实现：`sub/subClashService.go:155-158`
- 类型：复杂业务逻辑 / 数据别名 / 订阅输出错误
- 风险等级：中
- 证据：
  - `newStream := stream` 只是复制 map 引用。
  - 循环中按 `forceTls` 修改 `newStream["security"]`、`tlsSettings`，会污染后续 external proxy。
  - Clash 实现使用 `workingStream := cloneMap(stream)`，说明这里需要每个 proxy 独立副本。
- 触发条件：一个入站配置多个 `externalProxy`，且 `forceTls` 混用 `tls`、`none`、`same`。
- 影响：JSON 订阅的外部代理节点可能继承上一个节点的 TLS/Reality 设置，导致客户端连接失败或安全参数错误。
- 修复建议：
  - 与 Clash 路径一致，在 JSON 订阅中为每个 external proxy 使用 `cloneMap(stream)`。
  - 避免直接修改 `inbound.Listen` / `inbound.Port`，改用局部 `workingInbound := *inbound`。
  - 增加多 external proxy 混合 `forceTls` 的矩阵测试。

### BUG-006 LDAP 同步 Job 读取设置失败会 panic，Cron 未显式 Recover

- 位置：
  - `web/job/ldap_sync_job.go:26-45`
  - `web/job/ldap_sync_job.go:72-83`
  - `web/job/ldap_sync_job.go:94`
  - `web/job/ldap_sync_job.go:112-115`
  - `web/job/ldap_sync_job.go:161`
  - `web/web.go:78-83`
- 类型：后台任务 / 异常处理 / 可用性
- 风险等级：中
- 证据：
  - `mustGetString`、`mustGetInt`、`mustGetBool` 在设置读取错误时直接 `panic(err)`。
  - Cron 初始化只配置了 `cron.SkipIfStillRunning`，未使用 `cron.Recover`。
- 触发条件：LDAP 已启用但设置缺失、数据库短暂异常、配置值类型错误或读取失败。
- 影响：后台任务 panic 可能中断 cron goroutine，甚至在未 recover 的情况下导致进程崩溃。
- 修复建议：
  - 将 `mustGet*` 改为返回 `(value, error)`，`Run` 中记录错误并安全退出本轮。
  - Cron chain 增加 `cron.Recover(cron.DefaultLogger)`。
  - 增加 LDAP 设置读取失败的回归测试，验证不会 panic。

### BUG-007 Custom Geo 下载缺少响应体大小上限

- 位置：
  - `web/service/custom_geo.go:370`
  - `web/service/custom_geo.go:399-416`
  - 对比实现：`web/service/http_client.go:11-14`、`web/service/http_client.go:33-40`
  - 对比实现：`web/service/server.go:654-665`、`web/service/server.go:1143-1184`
- 类型：资源耗尽 / 文件下载 / DoS
- 风险等级：中
- 证据：
  - Custom Geo 使用 `http.Client{Timeout: 10 * time.Minute}`。
  - 下载落盘时直接 `io.Copy(out, resp.Body)`，只检查最小尺寸 `minDatBytes`，没有最大 `Content-Length` 或 `LimitReader`。
  - 其他 geofile/Xray 下载路径已有 `validateContentLength` 和 `copyLimited`。
- 触发条件：管理员配置恶意或错误 URL，远端返回超大文件、无限流或压缩炸弹式响应。
- 影响：磁盘占用、网络带宽和长连接资源被消耗，可能影响面板和 Xray 运行。
- 修复建议：
  - 复用 `maxDownloadFileBytes`、`validateContentLength`、`copyLimited`。
  - 对未知 Content-Length 也用 `io.LimitReader(limit+1)`。
  - 增加超大响应和无 Content-Length 的下载测试。

### BUG-008 新 Vue UI 使用 Math.random 生成订阅密钥

- 位置：
  - `frontend/src/views/InboundsView.vue:1414`
  - `frontend/src/views/InboundsView.vue:1558`
  - `frontend/src/views/InboundsView.vue:2159`
  - `frontend/src/views/InboundsView.vue:2173`
  - `frontend/src/views/InboundsView.vue:2581-2598`
  - 对比旧 UI：`web/assets/js/util/index.js:104-147`
- 类型：安全 / 随机数 / 订阅 bearer secret
- 风险等级：中
- 证据：
  - `randomToken` 使用 `Math.floor(Math.random() * alphabet.length)`。
  - `subId` 是订阅访问口令，前端多处用 `randomToken(16)` 生成。
  - 旧 UI `RandomUtil` 已使用 `window.crypto.getRandomValues` / `crypto.randomUUID`。
- 触发条件：新 UI 创建或编辑客户端/peer 时自动生成 `subId`。
- 影响：`Math.random` 不是密码学安全随机源，订阅 ID 的不可预测性低于预期；如果订阅 URL 暴露，攻击者更容易做统计或预测攻击。
- 修复建议：
  - 使用 `crypto.getRandomValues` 生成 token。
  - `randomUuid` 的 fallback 也应使用 `crypto.getRandomValues`，或在缺少 Web Crypto 时阻止生成安全凭据。
  - 增加单元测试或 lint rule 禁止安全凭据路径使用 `Math.random`。

### BUG-009 默认 admin/admin 与登录无限速组合风险

- 位置：
  - `database/db.go:27-28`
  - `database/db.go:51-70`
  - `web/controller/index.go:59-91`
- 类型：认证安全 / 默认弱口令 / 暴力破解
- 风险等级：中
- 证据：
  - 空库初始化创建 `admin` / `admin`。
  - 登录接口只校验空用户名/密码，没有 IP/用户名维度的速率限制、失败锁定或退避。
- 触发条件：源码直跑、Docker/安装脚本未随机化或用户未修改默认密码；面板暴露到公网。
- 影响：默认口令扫描或在线暴力破解成功概率上升。
- 修复建议：
  - 首次启动生成随机一次性初始凭据，或强制初始化向导。
  - 登录增加 IP + 用户名双维度限速、失败退避、短时锁定。
  - 默认账号首次登录后强制修改密码。

### BUG-010 remarkModel 缺少后端验证，可导致订阅 panic

- 位置：
  - `frontend/src/views/SettingsView.vue:87-88`
  - `web/entity/entity.go:113-186`
  - `web/service/setting.go:720-738`
  - `sub/subService.go:875-877`
- 类型：配置验证 / 边界条件
- 风险等级：中
- 证据：
  - 新 UI 直接用普通输入框编辑 `settings.remarkModel`。
  - `AllSetting.CheckValid` 未校验 `RemarkModel`。
  - `genRemark` 直接访问 `s.remarkModel[0]` 和 `s.remarkModel[1:]`。
- 触发条件：管理员把 `remarkModel` 保存为空字符串，或通过 API/导入写入非法格式。
- 影响：任意订阅生成路径调用 `genRemark` 时 panic，订阅不可用。
- 修复建议：
  - 后端校验 `remarkModel` 非空且格式合法，例如第 1 位分隔符，后续字符只允许 `i/e/o` 的合理组合。
  - UI 保持与旧版选择器一致，避免纯文本误填。
  - `genRemark` 对非法值回退到 `-ieo` 并记录 warning。

### BUG-011 反向代理 HTTPS 场景下会话 Cookie Secure 推断不完整

- 位置：
  - `web/web.go:228-244`
  - `web/session/session.go:103-109`
  - `web/middleware/security.go:177-185`
- 类型：会话安全 / 部署配置
- 风险等级：中
- 证据：
  - 会话创建时仅当面板本身配置 cert/key 才设置 `sessionOptions.Secure = true`。
  - 但 logout 清理和 CSRF scheme 判断会读取 `X-Forwarded-Proto`。
- 触发条件：常见部署方式是 TLS 终止在 Nginx/CDN，Go 面板只跑 HTTP，`certFile/keyFile` 为空。
- 影响：外部 HTTPS 部署下，浏览器收到的 session cookie 可能缺少 `Secure`，如果同域 HTTP 可达则有泄露风险。
- 修复建议：
  - 增加显式配置项，例如 `sessionCookieSecure` / `trustedProxy`.
  - 只有在可信反代边界内才根据 `X-Forwarded-Proto` 设置 Secure。
  - 文档说明反代部署推荐配置，并补充集成测试。

### BUG-012 WebSocket Origin 校验为手写字符串解析且空 Origin 放行

- 位置：
  - `web/controller/websocket.go:37-67`
  - `web/controller/websocket.go:84-93`
- 类型：WebSocket 安全 / Origin 校验
- 风险等级：中
- 证据：
  - `CheckOrigin` 手动 trim `http://` / `https://` 并用 `strings.Index(..., ":")` 去端口。
  - 空 `Origin` 直接放行。
  - 复杂 Host、IPv6、大小写、尾点、代理场景没有统一解析。
- 触发条件：已登录用户浏览器访问恶意站点尝试跨站 WebSocket，或非浏览器客户端构造无 Origin 请求。
- 影响：当前仍有 session auth，风险受限；但 Origin 校验边界弱，后续如果 WebSocket 增加敏感写操作，风险会上升。
- 修复建议：
  - 使用 `url.Parse` 和 `net.SplitHostPort` 标准化 Origin/Host。
  - 与 CSRF/可信代理策略共用 same-origin 判断。
  - 对浏览器场景拒绝跨站和非法 Origin；无 Origin 仅允许明确的非浏览器客户端策略。

### BUG-013 logout 使用 GET 执行状态变更

- 位置：
  - `web/controller/index.go:41-45`
  - `web/controller/index.go:109-119`
- 类型：CSRF / HTTP 语义
- 风险等级：低
- 证据：
  - `g.GET("/logout", a.logout)`。
  - handler 清理 session 并保存 cookie。
- 触发条件：已登录用户访问第三方页面，页面加载 `<img src=".../logout">` 或诱导点击链接。
- 影响：攻击者可触发用户退出登录，通常是低危可用性/体验问题。
- 修复建议：
  - 改为 POST logout，并使用 CSRF token。
  - 保留 GET 时也应确认 SameSite 与 Origin/Referer 策略，或只用于展示确认页。

### BUG-014 订阅生成存在重复解析和线性查找，客户端多时性能退化

- 位置：
  - `sub/subService.go:78-105`
  - `sub/subService.go:168-174`
  - `sub/subService.go:245-248`
  - `sub/subService.go:266-270`
  - `sub/subService.go:329-333`
- 类型：性能瓶颈 / 复杂度
- 风险等级：中
- 证据：
  - `GetSubs` 外层已解析 clients 并遍历，但 `genVmessLink` / `genVlessLink` / `genTrojanLink` 再次 `GetClients` 并 `findClientIndex`。
  - `getClientTraffics` 对每个 client 都线性扫描 `ClientStats`。
- 触发条件：单个入站大量客户端，或订阅 ID 被频繁请求。
- 影响：重复 JSON 解析和 O(N²) 查找会增加 CPU 延迟，订阅入口更容易被流量放大。
- 修复建议：
  - 在 `GetSubs` 中构建 `email -> client` 和 `email -> traffic` map。
  - `getLink` 直接接收 `model.Client` 与 traffic map，不再重复解析。
  - 增加大客户端数 benchmark。

### BUG-015 出站测试会完整 drain 响应体，缺少最大读取上限

- 位置：
  - `web/service/outbound.go:341-364`
  - `web/service/outbound.go:368-388`
  - `web/service/outbound.go:382-396`
- 类型：资源耗尽 / 性能
- 风险等级：低
- 证据：
  - `testConnection` 对 warmup 和正式请求都执行 `io.Copy(io.Discard, resp.Body)`。
  - 虽有 10 秒 timeout，但没有响应体大小上限。
- 触发条件：管理员配置的 outbound test URL 返回大响应或持续流。
- 影响：测试出站时占用带宽和 goroutine 时间，影响面板响应；风险低于公开接口，因为 URL 来自设置且接口需登录。
- 修复建议：
  - 使用 `io.Copy(io.Discard, io.LimitReader(resp.Body, maxAPIResponseBytes+1))`。
  - 对超过上限的响应仍可返回状态码和延迟，不必完整 drain。

### BUG-016 手动 Telegram 备份路由使用零值 Tgbot 服务，依赖隐藏全局状态

- 位置：
  - `web/controller/api.go:14-25`
  - `web/controller/api.go:61-67`
  - `web/web.go:300`
  - `web/service/tgbot.go:2828-2833`
- 类型：第三方集成 / 依赖注入 / 可维护性
- 风险等级：低
- 证据：
  - `APIController` 有 `Tgbot service.Tgbot` 字段，但 `NewAPIController` 未注入 `s.tgbotService`。
  - `/panel/api/backuptotgbot` 调用的是零值 `a.Tgbot.SendBackupToAdmins()`。
  - 当前 `SendBackupToAdmins` 依赖全局 `isRunning`、`adminIds`、`bot`，所以可能仍可工作，但依赖关系不清晰。
- 触发条件：后续 `SendBackupToAdmins` 或其调用链改为使用 `Tgbot` 实例内服务字段。
- 影响：未来重构容易引入 nil/零值行为差异；当前手动备份路径难以单测和替换。
- 修复建议：
  - `NewAPIController` 接收并保存 `service.Tgbot` 或接口。
  - 为 `/backuptotgbot` 增加单元测试，验证调用已启动的 bot 服务实例。

### BUG-017 `.gitignore` 忽略 docs 目录，新增文档容易丢失

- 位置：
  - `.gitignore:62-63`
- 类型：工程治理 / 文档可维护性
- 风险等级：低
- 证据：
  - `.gitignore` 写有 `# Ignore docs directory` 和 `docs/`。
  - 仓库已有被跟踪的 `docs` 文件，但新增文档默认会被忽略。
- 触发条件：后续新增 ADR、审查报告、使用说明或迁移文档放入 `docs/`。
- 影响：文档更新可能不会进入提交，导致评审材料或整改记录缺失。
- 修复建议：
  - 如果 `docs/` 是正式文档目录，移除该 ignore。
  - 如果只是要忽略生成文档，改为忽略具体输出目录，例如 `docs/generated/`。

### BUG-018 Go 全仓命令会扫描 frontend/node_modules

- 位置：
  - `frontend/node_modules/flatted/golang/pkg/flatted`
  - `.gitignore:33`
- 类型：CI/验证范围污染
- 风险等级：低
- 证据：
  - `go list ./...` 输出包含 `github.com/superaddmin/SuperXray-gui/v2/frontend/node_modules/flatted/golang/pkg/flatted`。
  - Go 的 `./...` 不会天然忽略 `node_modules` 中的 Go 包。
- 触发条件：安装前端依赖后执行 `go test ./...`、`go vet ./...`、`staticcheck ./...`。
- 影响：未来任意前端依赖携带不兼容 Go 包时，后端 CI/本地验证可能被无关包污染而失败。
- 修复建议：
  - CI 使用显式 Go 包列表并过滤 `node_modules`。
  - 或将前端依赖置于 Go module 之外，或采用脚本生成后端包列表。
  - 在 README/CI 中记录推荐命令，避免开发者误用。

## 6. 入站协议、订阅兼容与可运行状态复核

本节基于 `database/model/model.go`、`web/service/protocol_validation.go`、`sub/protocol_capability.go`、`sub/subService.go`、`sub/subJsonService.go`、`sub/subClashService.go`、`frontend/src/schemas/protocolRegistry.ts` 与 `frontend/src/utils/inboundCompat.ts` 交叉复核。结论是：后端入站模型已覆盖 11 类协议；订阅分发仅覆盖可作为客户端节点导入的代理/隧道协议；透明代理、本地代理和流量转发类入站不进入订阅输出。

### 6.1 入站协议可运行矩阵

| 协议 | 后端模型 | 保存校验 | 新 UI 编辑 | 订阅输出 | 可运行状态 | 主要风险/限制 |
| --- | --- | --- | --- | --- | --- | --- |
| VMess | 支持 | UUID 校验 | 支持 | Base64/JSON/Clash | 主路径完整，可运行 | 畸形 `streamSettings` 仍可能触发订阅层 unchecked assertion |
| VLESS | 支持 | UUID 与 flow 校验，校验 TLS/Reality 约束 | 支持 | Base64/JSON/Clash | 主路径完整，可运行 | Reality/XTLS flow 与客户端支持度强相关 |
| Trojan | 支持 | password 必填 | 支持 | Base64/JSON/Clash | 主路径完整，可运行 | TLS/SNI/ALPN 配置错误会导致客户端连接失败 |
| Shadowsocks | 支持 | method/password/client 约束 | 支持 | Base64/JSON/Clash | 主路径完整，可运行 | 加密方法需与客户端兼容 |
| Hysteria | 支持 | auth 必填且要求 TLS | 支持 | Base64/JSON/Clash | 主路径完整，可运行 | Hysteria v1 客户端生态差异较大 |
| Hysteria2 | 支持 | auth 必填且要求 TLS | 支持 | Base64/JSON/Clash | 主路径完整，可运行 | 依赖客户端 Hysteria2 支持 |
| WireGuard | 支持 | 订阅侧按 peer 输出，保存侧相对轻校验 | 支持 | WireGuard 配置/JSON/Clash | 可运行 | 与普通代理客户端导入方式不同，peer 字段完整性更关键 |
| Tunnel | 支持 | 轻校验，主要依赖 Xray 运行时 | 支持 | 不支持 | 可保存，运行依赖配置完整性 | 不生成订阅节点；需补 schema 校验和运行态测试 |
| HTTP | 支持 | 轻校验，主要依赖 Xray 运行时 | 支持 | 不支持 | 可保存，运行依赖配置完整性 | 不生成订阅节点；账号/透明代理字段需进一步校验 |
| Mixed | 支持 | 轻校验，主要依赖 Xray 运行时 | 支持 | 不支持 | 可保存，运行依赖配置完整性 | 不生成订阅节点；HTTP/SOCKS 组合行为需实测覆盖 |
| Tun | 支持 | 轻校验，主要依赖 Xray 运行时和系统权限 | 支持 | 不支持 | 可保存，运行依赖系统路由/权限 | 透明代理类配置复杂，Windows/Linux 行为差异需单独验证 |

### 6.2 订阅格式与客户端兼容矩阵

| 订阅格式 | 路径 | 适用客户端 | 当前协议覆盖 | 关键限制 |
| --- | --- | --- | --- | --- |
| Base64 / Plain URI | `/sub/<subid>` | V2rayN、Shadowrocket、支持标准分享链接的客户端 | VMess、VLESS、Trojan、Shadowsocks、Hysteria、Hysteria2；WireGuard 输出独立配置文本 | 输出质量依赖各客户端对 URI 参数、Reality、Hysteria 的解析能力 |
| Xray JSON | `/json/<subid>` | Xray 客户端、支持 Xray outbound JSON 的工具 | VMess、VLESS、Trojan、Shadowsocks、Hysteria、Hysteria2、WireGuard | `externalProxy` 当前存在 map alias 风险，见 BUG-005 |
| Clash/Mihomo YAML | `/clash/<subid>` | Clash、Mihomo 及兼容客户端 | VMess、VLESS、Trojan、Shadowsocks、Hysteria、Hysteria2、WireGuard | 普通代理协议当前仅生成 TCP、WebSocket、gRPC；mKCP、HTTPUpgrade、XHTTP 不在 Clash 输出路径中 |

### 6.3 协议复核新增结论

- 后端可保存的协议不等于可订阅协议。`Tunnel`、`HTTP`、`Mixed`、`Tun` 是入站规则能力，不是客户端订阅节点能力。
- 严格保存校验主要覆盖 VMess、VLESS、Trojan、Shadowsocks、Hysteria/Hysteria2；Tunnel/HTTP/Mixed/Tun/WireGuard 的部分字段仍更依赖 Xray 运行时校验。
- 新 Vue UI 的协议注册表已经覆盖旧 UI 主要入站协议，但 Tun、Tunnel、Mixed、HTTP 的结构化字段仍应继续补充更细 schema，以避免“能保存但运行时失败”的配置。
- Clash/Mihomo 订阅支持协议较广，但传输层覆盖较窄。对于 mKCP、HTTPUpgrade、XHTTP 等传输，应优先通过 Xray JSON 或目标客户端手工配置验证。
- 当前项目已有订阅输出矩阵测试覆盖 VMess/VLESS/Trojan/Shadowsocks/Hysteria2/WireGuard，但还应补充 Hysteria v1、畸形 stream、空 clients、缺 peer、复杂 externalProxy 和非 Clash 传输降级场景。

## 7. 已确认的良好点

- `web/web.go:67-75` 与 `sub/sub.go:58-65` 均配置了 HTTP server timeout 和 `MaxHeaderBytes`。
- `web/middleware/security.go` 已有 CSP、CSRF、安全响应头；新 UI CSP 不含 `unsafe-inline` / `unsafe-eval`。
- `web/controller/server.go:319` 及相关路径已对 DB import 做请求体大小限制，服务层也做 SQLite 签名、完整性、权限和回滚处理。
- `web/service/server.go` 中 Xray/geofile 官方下载路径已有 `validateContentLength` 与 `copyLimited`。
- `xray/process.go` 中配置和 crash log 文件权限已有收敛。
- `web/job/ldap_sync_job.go` 的客户端 JSON 已使用 `json.Marshal`，未发现旧式字符串拼接 JSON。
- `gosec`、`govulncheck`、`npm audit` 当前未发现直接依赖漏洞。

## 8. 整改路线建议

### P0：安全和可用性止血

1. 修复 BUG-001：删除密码日志/通知。
2. 修复 BUG-002：移除订阅请求态共享字段，补并发 race 测试。
3. 修复 BUG-003：建立 canonical origin / trusted proxy 策略。
4. 修复 BUG-004：订阅层 safe getter + 畸形配置测试。

### P1：稳定性和边界补强

1. 修复 BUG-005：JSON externalProxy 使用 clone。
2. 修复 BUG-006：LDAP Job 去 panic，Cron 加 recover。
3. 修复 BUG-007：Custom Geo 下载加最大响应体限制。
4. 修复 BUG-008：新 UI 安全随机数统一走 Web Crypto。
5. 修复 BUG-009：首次初始化凭据随机化 + 登录限速。
6. 修复 BUG-010：`remarkModel` 前后端双层校验。

### P2：防御纵深和性能治理

1. 强化会话 Secure cookie 反代配置。
2. 标准化 WebSocket Origin 校验。
3. logout 改 POST + CSRF。
4. 优化订阅生成复杂度。
5. 限制 outbound test response drain。

### P3：工程治理

1. 调整 `.gitignore` 的 `docs/` 策略。
2. 修正 Go 全仓验证命令，避免 `node_modules` 污染。
3. 为报告中的 P0/P1 问题分别建立跟踪 issue，避免一次 PR 混改。

## 9. 验证限制与残余风险

- E2E 失败原因是本地目标服务未启动：`net::ERR_CONNECTION_REFUSED`，未形成 UI 业务断言结果。因此 UI 真实交互路径仍需在启动面板后复跑。
- `go test -race ./sub` 通过，但缺少并发订阅请求测试，不能证明 BUG-002 不存在。
- 本次没有对 shell 安装脚本做完整安全审计，仅做了关键字和入口级扫描。
- Host/Forwarded Header 风险与真实反向代理配置强相关，需要结合部署拓扑复核。
- 本报告聚焦潜在 bug 和安全风险，未尝试直接修复代码，避免把审查任务混入大规模行为变更。
