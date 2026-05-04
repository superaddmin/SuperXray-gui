# Phase 10 Legacy UI 退场与功能残缺对比报告

日期：2026-05-04

## 结论

当前仓库已经完成“新 UI 默认入口”切换：`/panel/`、`/panel/dashboard`、`/panel/inbounds`、`/panel/xray`、`/panel/settings`、`/panel/logs` 均由 Vue 3/Vite 新 UI 承载；`/panel/ui/` 作为兼容入口保留。

但旧 UI 尚不满足“全量删除”条件。`web/html`、`web/assets`、`/panel/legacy` 仍承担旧 UI 回退、兼容矩阵验收、登录页、订阅公开页和若干 Xray 高级管理能力。按照 UI-first 迁移门禁，旧 UI 必须等新 UI 通过完整 Xray 等价迁移与生产灰度后再退场。

本轮可安全清理的旧框架冗余为旧 vendor sourcemap 调试产物；运行期仍被旧回退入口使用的 Vue 2、Ant Design Vue、CodeMirror、Moment、Axios 等资产继续保留。

## 已完成清理

已删除不影响运行时的旧 UI sourcemap 文件，并同步移除 minified vendor 文件尾部的 `sourceMappingURL` 注释，避免浏览器继续请求不存在的调试映射。

| 文件                                        | 处理                |
| ------------------------------------------- | ------------------- |
| `web/assets/ant-design-vue/antd.min.js.map` | 删除                |
| `web/assets/axios/axios.min.js.map`         | 删除                |
| `web/assets/moment/moment.min.js.map`       | 删除                |
| `web/assets/otpauth/otpauth.umd.min.js.map` | 删除                |
| `web/assets/ant-design-vue/antd.min.js`     | 移除 sourcemap 尾注 |
| `web/assets/axios/axios.min.js`             | 移除 sourcemap 尾注 |
| `web/assets/moment/moment.min.js`           | 移除 sourcemap 尾注 |
| `web/assets/otpauth/otpauth.umd.min.js`     | 移除 sourcemap 尾注 |

## 2026-05-04 门禁补齐进展

本轮继续按“先补新 UI 替代，再删除旧 UI”的门禁推进，已新增或补齐以下新 UI 能力：

| 门禁项                | 当前进展                                                                                                                                              | 验收证据                                                                  |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| 新 UI 登录页          | `/panel/login` 已由 Vue 3/Vite 新 UI 承载，未登录访问新 UI 路由会跳转到新登录页，登录后回到 `/panel/`                                                 | `can log in through the new Vue UI login page` E2E 通过                   |
| Custom Geo 与 Geofile | Dashboard 新增 Geo Maintenance、geoip/geosite 更新、Custom Geo 资源列表与新增/编辑/下载/删除/更新入口                                                 | `can open new UI geo maintenance controls` E2E 通过                       |
| Xray 出站工具         | Xray 页新增 Outbound Tools，覆盖出站流量刷新、出站测试、流量重置、Warp/Nord 触发入口                                                                  | `can open new UI Xray outbound tools` E2E 通过                            |
| 入站导入与批量工具    | Inbounds 页新增 Import JSON、Reset All Traffic、Delete Depleted 入口，并保留批量导出/重置/删除                                                        | `can open new UI inbound import and batch controls` E2E 通过              |
| 在线/IP 管理          | Inbounds 页新增 Online Clients、Refresh Activity；详情抽屉新增 Online / IP Management、View IPs、Clear IPs，并复用旧在线/最后在线/IP API              | `can open new UI online and IP management controls` E2E 通过              |
| 2FA 设置替代          | Settings Security 页新增 Two Factor Setup、Generate Token、Disable Two Factor 和 `otpauth://` setup URI；仍写旧 `twoFactorEnable/twoFactorToken` 字段 | `can open new UI two-factor setup and subscription public links` E2E 通过 |
| 订阅公开链接替代      | Settings Subscription 页新增 Subscription Public Links、Copy Links、Open URI/JSON/Clash；仍读取旧 `subURI/subJsonURI/subClashURI` 字段                | `can open new UI two-factor setup and subscription public links` E2E 通过 |
| 订阅输出矩阵          | Go 层补齐 VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard 的订阅链接/配置、JSON outbound、Clash/mihomo proxy 基础输出矩阵                     | `go test ./sub` 通过                                                      |

这些补齐项降低了旧 UI 退场缺口，但不等于可以立即删除 `/panel/legacy`、`web/html` 或旧 `web/assets`。旧 UI 仍承担灰度回退、兼容矩阵和订阅服务端模板职责。

## 暂不能删除的旧 UI 边界

| 边界                                                | 当前作用                                               | 删除风险                                                   |
| --------------------------------------------------- | ------------------------------------------------------ | ---------------------------------------------------------- |
| `web/html/login.html`                               | Legacy 登录页回退模板                                  | 新 UI 登录页已覆盖主入口，但生产灰度前删除会失去登录回退面 |
| `web/html/index.html`                               | Legacy Dashboard 与 Custom Geo、系统工具入口           | 删除会破坏 `/panel/legacy/` 回退和 Custom Geo UI           |
| `web/html/inbounds.html`                            | Legacy 入站管理和兼容读取验收                          | 删除会破坏新旧 UI 数据兼容矩阵                             |
| `web/html/xray.html`                                | Legacy Xray 配置、出站测试、Warp/Nord 工具             | 删除会丢失新 UI 尚未覆盖的高级能力                         |
| `web/html/settings.html`                            | Legacy 设置页与 2FA/订阅配置回退                       | 删除会减少设置故障回退面                                   |
| `web/html/settings/panel/subscription/subpage.html` | 订阅公开页服务端模板                                   | 删除会影响订阅 URL 输出                                    |
| `web/assets/**`                                     | Legacy Vue 2、Ant Design Vue、CodeMirror、工具库和样式 | 删除会破坏旧模板渲染                                       |
| `/panel/legacy` 路由                                | Phase 8/9/10 回退入口                                  | 删除会违反当前 UI-first 回退门禁                           |
| `tests/e2e/legacy-panel.spec.ts`                    | 新 UI 写入、旧 UI 可读的兼容验收                       | 删除会失去等价迁移证据                                     |

## 旧 UI 与新 UI 功能覆盖矩阵

| 功能域            | 旧 UI 能力                                                              | 新 UI 当前状态                                                                        | 残缺判断                             |
| ----------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------- | ------------------------------------ |
| 登录与会话        | 登录、登出、2FA 检查                                                    | 新 UI 已提供 `/panel/login` 登录页并复用旧登录 API/会话                               | 主入口等价，旧登录页仍保留回退       |
| Dashboard         | 状态、流量、Xray 版本、日志入口、Custom Geo、Geofile 更新、DB 下载/导入 | 覆盖状态、版本、日志、DB 下载/导入、Xray 生命周期、Custom Geo 与 Geofile 更新         | 基础能力等价，仍需生产灰度回归       |
| 日志              | 面板日志、Xray 日志                                                     | 已用纯文本/虚拟列表渲染                                                               | 基础能力等价                         |
| Xray 配置         | JSON 配置编辑、保存、运行结果、默认配置、出站流量、出站测试、Warp/Nord  | 覆盖配置读取/保存、运行结果、启动/停止/重启/安装、出站流量、出站测试、Warp/Nord 入口  | 默认配置与结构化高级编辑仍需核对     |
| Xray 结构化编辑   | DNS、FakeDNS、路由规则、Balancer、Reverse、Outbound 弹窗                | 新 UI 主要提供 JSON 编辑                                                              | 结构化编辑器缺失                     |
| 入站列表          | 列表、新增、编辑、删除、启停字段、分享                                  | 覆盖主流程和分享文本                                                                  | 基础能力等价                         |
| 主力协议表单      | VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard                 | 已覆盖并通过兼容抽检                                                                  | 主力协议等价                         |
| 其他协议/特殊入站 | HTTP、SOCKS、Dokodemo、透明代理类扩展                                   | 新 UI 未形成完整高级表单矩阵                                                          | 仍需补齐或明确不支持策略             |
| 客户端/Peer       | 新增、编辑、删除、重置单个/全部流量、二维码/分享                        | 覆盖主力协议客户端/peer 编辑、流量重置、分享链接复制和在线/IP 管理入口                | 基础能力等价，仍需真实在线数据回归   |
| 入站导入          | 原始 JSON 导入                                                          | 新 UI 已暴露 Import JSON 入口并复用旧导入 API                                         | 需补真实导入 mutation 回归           |
| 在线状态          | onlines、lastOnline、client IP 查看/清理                                | 新 UI 已接入 Online Clients、lastOnline、client IP 查看和清理入口                     | 需补真实在线客户端回归               |
| 设置              | 面板设置、订阅设置、账号密码、重启面板、备份恢复、2FA                   | 覆盖主要设置、订阅、账号、备份恢复、2FA token 生成/清除                               | 仍需更多设置项兼容抽检               |
| 订阅公开页        | 服务端模板输出订阅页面                                                  | 新 UI 提供公开链接复制/打开入口；后端订阅输出矩阵已有 Go 回归，服务端模板仍承载公开页 | 不能删除服务端订阅模板               |
| 安全策略          | Legacy CSP 仍需兼容 Vue 2 内联模板                                      | 新 UI CSP 已不使用 `unsafe-inline` / `unsafe-eval`                                    | Legacy 退场前不能统一收紧到新 UI CSP |

## 后续全量切换门禁

1. 新 UI 登录页已完成；生产灰度前继续保留 `login.html` 回退。
2. Custom Geo、Geofile 更新、出站流量、出站测试、Warp/Nord 已补入口；Xray 默认配置和结构化高级编辑仍需核对。
3. 补齐 Xray DNS、FakeDNS、路由规则、Balancer、Reverse、Outbound 的结构化编辑能力，或形成明确的“JSON 高级模式替代”验收标准。
4. 入站 JSON 导入、批量重置/删除、删除耗尽客户端、分享复制、在线/最后在线/IP 管理已补入口；仍需真实导入 mutation 和真实在线客户端回归。
5. 2FA 设置/删除和订阅公开链接入口已补齐；订阅实际输出仍需保持旧服务端路径，基础输出矩阵已补 Go 回归，CI/第二环境仍需复刻。
6. 将 E2E 从“新 UI 写入、旧 UI 可读”升级为“新 UI 独立完成全流程，旧 UI 仅灰度回滚抽检”。
7. 完成至少一轮真实环境非跳过 E2E、响应式截图、控制台零错误、Go 构建和前端构建验证。
8. 生产灰度至少保留一个版本周期后，再删除 `/panel/legacy`、`web/html` 旧模板、`web/assets` 旧 vendor，并同步移除 Legacy CSP 宽策略。

## 当前建议

维持“新 UI 默认、旧 UI 受控回退”的状态继续上线准备。旧 UI 框架冗余清理应限定在 sourcemap、未引用调试产物和确认不被模板引用的静态文件；功能性模板、旧 vendor 和兼容测试暂不删除。

## 本轮验证

```powershell
cd frontend
npm run format
npm run lint
npm run typecheck
npm run build
npm exec --yes knip -- --reporter compact
npm exec --yes depcheck -- --json
cd ..
go test ./web/...
go test ./...
go vet ./...
go build -o bin\SuperXray.exe .\main.go
rg 'v-html|innerHTML|insertAdjacentHTML' frontend\src web\html -n
rg 'v-html' frontend\src web\html web\ui -n
rg 'unsafe-inline|unsafe-eval' frontend\src web\ui -n
npm run e2e
```

验证结果：

| 命令                                                                  | 结果                                                 |
| --------------------------------------------------------------------- | ---------------------------------------------------- |
| `npm run format`                                                      | 通过                                                 |
| `npm run lint`                                                        | 通过                                                 |
| `npm run typecheck`                                                   | 通过                                                 |
| `npm run build`                                                       | 通过，已重新生成 `web/ui`                            |
| `npm exec --yes knip -- --reporter compact`                           | 通过，无未使用文件/导出/依赖输出                     |
| `npm exec --yes depcheck -- --json`                                   | 通过，`dependencies` 与 `devDependencies` 均为空数组 |
| `go test ./web/...`                                                   | 通过                                                 |
| `go test ./...`                                                       | 通过                                                 |
| `go vet ./...`                                                        | 通过                                                 |
| `go build -o bin\SuperXray.exe .\main.go`                             | 通过                                                 |
| `rg 'v-html\|innerHTML\|insertAdjacentHTML' frontend\src web\html -n` | 通过，无匹配                                         |
| `rg 'v-html' frontend\src web\html web\ui -n`                         | 通过，无匹配                                         |
| `rg 'unsafe-inline\|unsafe-eval' frontend\src web\ui -n`              | 通过，无匹配                                         |
| `npm run e2e`                                                         | 通过，19 passed / 0 skipped / 0 failed               |
