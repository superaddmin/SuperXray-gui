# SuperXray-gui 静态分析报告

- 生成时间：04/28/2026 23:12:55
- 仓库：`f:\SuperXray-gui`
- 扫描范围：Go 后端、Gin 路由、嵌入式 Vue/HTML/JS、Dockerfile、docker-compose.yml；排除 web/assets 下第三方压缩库和 sourcemap。
- 排除说明：忽略注释、明显第三方压缩资源、sourcemap 与自动生成代码；`gosec` 扫描时排除了 `web/assets`。
- 结果产物：[`static_analysis_findings.json`](./static_analysis_findings.json) 保存原始工具明细，本文档保存人工复核后的跟踪清单。

## 执行摘要

本次共确认 14 条重点问题：致命 1 条、重要 10 条、轻微 3 条。最优先处理项是 `web/html/index.html` 的日志 `v-html` DOM XSS；随后建议集中处理 CSRF/状态变更 GET、HTTP server/client 超时、上传大小限制、Telegram bot 共享状态竞态和 JSON 手工拼接问题。

## 工具执行结果

| 工具 | 命令 | 结果 |
|---|---|---|
| go vet | `go vet ./...` | 通过 |
| staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest -f=json ./...` | 通过，无诊断 |
| gosec | `go run github.com/securego/gosec/v2/cmd/gosec@latest -fmt=json -exclude-dir=web/assets ./...` | 原始命中 155 条：HIGH 7 / MEDIUM 42 / LOW 106 |
| govulncheck | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` | No vulnerabilities found |
| ESLint | `npx --yes eslint@8.57.1 ... web/assets/js` | 4 条 warning，无 error |
| dockerfilelint | `npx --yes dockerfilelint Dockerfile` | 1 条 clarity issue |
| yaml-lint | `npx --yes yaml-lint docker-compose.yml` | 通过 |
| go test | `go test ./...; $env:CGO_ENABLED="1"; go test ./...` | 默认 CGO_ENABLED=0 时 sqlite 集成测试失败；CGO_ENABLED=1 时本机缺少 gcc，runtime/cgo 构建失败 |

## 已确认问题清单

| ID | 严重级别 | 类别 | 文件路径 | 行号 | 问题描述 | 修复建议 |
|---|---|---|---|---|---|---|
| CONF-001 | 致命 | XSS | `web/html/index.html` | 440,475,848-855,916-923 | 日志内容经字符串拼接后通过 v-html 渲染，未对 message、Email、Inbound、Outbound 等字段做 HTML 转义。攻击者若能写入 Xray/访问日志，管理员查看日志时可能触发 DOM XSS。 | 将日志作为文本渲染，或对所有动态字段做 HTML 转义/DOMPurify 后再渲染；优先用 textContent/插值和 CSS 类替代 v-html。 |
| CONF-002 | 重要 | CSRF/HTTP语义 | `web/controller/api.go; web/controller/server.go` | 54,135-138 | Cookie 会话保护的 API 未见 CSRF token/Origin 校验，且 /panel/api/backuptotgbot 使用 GET 触发发送备份的副作用。 | 为状态变更接口添加 CSRF token 或 Origin/Referer 校验；把有副作用的 GET 改为 POST，并保持 SameSite=Lax/Strict。 |
| CONF-003 | 重要 | DoS | `web/web.go; sub/sub.go` | 445-447,361-363 | 两个 http.Server 只设置 Handler，缺少 ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout/MaxHeaderBytes，存在慢连接资源耗尽风险。 | 按业务场景设置超时与头大小限制，例如 ReadHeaderTimeout 5s、ReadTimeout 30s、WriteTimeout 60s、IdleTimeout 120s。 |
| CONF-004 | 重要 | 会话安全 | `web/web.go; web/session/session.go` | 209-213,65-70 | Session cookie 设置了 HttpOnly/SameSite，但未在 HTTPS 部署时设置 Secure；仓库内也未见 CSP、X-Frame-Options/frame-ancestors、nosniff 等安全响应头。 | 根据是否启用 TLS 条件化设置 Secure；增加集中安全头中间件，至少覆盖 CSP、X-Content-Type-Options、frame-ancestors/X-Frame-Options、Referrer-Policy。 |
| CONF-005 | 重要 | 并发竞态/逻辑错误 | `web/service/tgbot.go` | 43-95,476-493,499-504,1755-1795,3797-3800 | Telegram bot 使用全局 userStates map 和 package-level client_* 状态，并在多个 goroutine 中读写，可能触发 concurrent map read/write panic，也可能在多个聊天/管理员操作时互相覆盖状态。 | 把会话状态封装到 Tgbot 实例内，按 chatId 建模并用 mutex/sync.Map 保护；全局 client_* 改成每个 chatId 的状态结构。 |
| CONF-006 | 重要 | 资源耗尽 | `web/service/server.go; web/service/warp.go` | 526,625,638,1094-1099,1138; 56,86,142 | 多处外部 HTTP 请求使用 http.Get 或空 http.Client，缺少超时；下载 Xray/geofile/WARP 响应未限制体积，可能导致请求长期挂起或磁盘/内存消耗。 | 统一使用带 Timeout 的 http.Client；对响应体使用 io.LimitReader 或 io.CopyN 上限；下载前校验状态码、Content-Length 和目标域。 |
| CONF-007 | 重要 | 上传/DoS | `web/controller/server.go; web/service/server.go` | 285,295,906-953 | 数据库导入使用 FormFile/ImportDB，未对请求体和上传文件大小设置上限；大文件会被写入临时 DB 并触发完整性校验。 | 在路由层使用 http.MaxBytesReader 或 Gin 中间件限制 body，检查 multipart FileHeader.Size，并设置合理 DB 文件大小上限。 |
| CONF-008 | 重要 | JSON注入/配置损坏 | `web/job/ldap_sync_job.go` | 322-339 | LDAP 同步手工拼接 JSON，把 c.ID/c.Password/c.Email 直接写入字符串；包含引号、反斜杠或控制字符的 LDAP 值会破坏 JSON，严重时可注入额外字段。 | 用 json.Marshal 序列化结构体或 map，禁止手工拼接 JSON。 |
| CONF-009 | 重要 | JSON注入/配置损坏 | `web/service/warp.go` | 74,112-117,134 | WARP 请求和持久化数据用 fmt.Sprintf 拼接 JSON，publicKey、license、hostname、secretKey 等值未转义；特殊字符会产生非法 JSON 或字段注入。 | 定义结构体/map 后 json.Marshal；同时处理 SetWarp 返回错误，避免静默失败。 |
| CONF-010 | 重要 | 文件权限 | `xray/process.go; web/service/server.go` | 261,321,683 | Xray 配置、崩溃日志和解压文件使用 fs.ModePerm/os.ModePerm，可能生成 0777 权限；Xray 配置通常包含客户端凭据。 | 配置/崩溃日志使用 0600 或 0640；目录使用 0750；仅可执行二进制在需要时设置 0755。 |
| CONF-011 | 重要 | 错误处理/空指针 | `web/job/check_client_ip_job.go` | 152-154 | 忽略 GetAccessLogPath/os.Open 错误后立即 defer file.Close；如果 os.Open 失败，file 为 nil，可能 panic。 | 检查错误后再 defer Close；失败时记录并返回或跳过本轮任务。 |
| CONF-012 | 轻微 | 弱哈希 | `web/global/hashStorage.go` | 4,38 | HashStorage 使用 MD5 作为查询内容键。若该键被当作不可碰撞标识使用，存在碰撞/覆盖风险。 | 改用 SHA-256/HMAC-SHA-256 或随机 token；若仅非安全缓存键，添加注释并限制作用域。 |
| CONF-013 | 轻微 | 前端健壮性 | `web/assets/js/util/index.js` | 192,211,279,281 | 直接调用 hasOwnProperty，若对象覆盖该属性会导致过滤/复制逻辑异常。 | 使用 Object.prototype.hasOwnProperty.call(...)。 |
| CONF-014 | 轻微 | 供应链/容器 | `Dockerfile` | 24 | 最终镜像 FROM alpine 未固定 tag/digest，构建结果会随上游 latest 漂移。 | 固定 Alpine 版本或 digest，并定期更新。 |

## 原始工具告警汇总

- 原始工具告警总数：160 条。所有逐条明细见 [`static_analysis_findings.json`](./static_analysis_findings.json) 的 `rawToolFindings`。
- `gosec` 告警：155 条；按人工映射严重级别为重要 49 条、轻微 106 条。
- `ESLint` 告警：4 条，均为 `no-prototype-builtins`。
- `dockerfilelint` 告警：1 条，最终镜像基础镜像未固定 tag。

| 规则/工具 | 数量 | 说明 |
|---|---:|---|
| gosec, G104 | 103 | 原始扫描归档项，需结合上下文复核 |
| gosec, G304 | 14 | 原始扫描归档项，需结合上下文复核 |
| gosec, G204 | 8 | 原始扫描归档项，需结合上下文复核 |
| gosec, G115 | 7 | 原始扫描归档项，需结合上下文复核 |
| gosec, G302 | 6 | 原始扫描归档项，需结合上下文复核 |
| eslint, no-prototype-builtins | 4 | 原始扫描归档项，需结合上下文复核 |
| gosec, G301 | 4 | 原始扫描归档项，需结合上下文复核 |
| gosec, G103 | 3 | 原始扫描归档项，需结合上下文复核 |
| gosec, G117 | 3 | 原始扫描归档项，需结合上下文复核 |
| gosec, G112 | 2 | 原始扫描归档项，需结合上下文复核 |
| dockerfilelint, Base Image Missing Tag | 1 | 原始扫描归档项，需结合上下文复核 |
| gosec, G102 | 1 | 原始扫描归档项，需结合上下文复核 |
| gosec, G107 | 1 | 原始扫描归档项，需结合上下文复核 |
| gosec, G306 | 1 | 原始扫描归档项，需结合上下文复核 |
| gosec, G401 | 1 | 原始扫描归档项，需结合上下文复核 |
| gosec, G501 | 1 | 原始扫描归档项，需结合上下文复核 |

## 验证限制

- `go vet`、`staticcheck`、`govulncheck`、`yaml-lint` 通过或无漏洞命中。
- `go test ./...` 未能完成：默认 `CGO_ENABLED=0` 时 `go-sqlite3` 使用 stub 导致 SQLite 相关测试失败；切换 `$env:CGO_ENABLED="1"` 后本机缺少 `gcc`，`runtime/cgo` 构建失败。
- 本报告是静态分析与人工复核结果，不包含运行时渗透测试、真实部署安全头验证或并发 race detector 覆盖。

## 建议修复顺序

1. 先修复 `CONF-001` 日志 DOM XSS，并增加前端回归验证。
2. 处理 `CONF-002`、`CONF-003`、`CONF-004`、`CONF-007`，形成 Web 服务安全基线。
3. 修复 `CONF-005` 并在具备 CGO/gcc 环境后运行 `go test -race ./...`。
4. 将 `CONF-008`、`CONF-009` 的 JSON 拼接替换为 `json.Marshal`，避免配置损坏和注入。
5. 处理文件权限、错误处理、弱哈希、ESLint 与 Dockerfile 供应链项。
