# 上游 3x-ui 同步策略与落地雷达

> 更新日期：2026-06-09
> 上游仓库：`MHSanaei/3x-ui`
> 当前上游基线：`v3.3.0` / `upstream/main` = `f8e89cc84`（tag object `02edec359`）
> 当前项目版本：`3.3.0`

## 1. 同步原则

SuperXray-gui 已在上游 3x-ui 基础上加入 Vue 3 新 UI、legacy UI 回退、Phase 9 安全收口、Gateway Egress MVP 与风险接受的最小 CoreManager/sing-box 后端入口。后续同步上游时采用“基线记录 + 选择性移植 + 本地门禁验证”的策略，不直接用上游文件覆盖本项目。

必须保留的本地边界：

- 不删除 `/panel/legacy/`、`web/html`、`web/assets`。
- 不让 CoreManager 接管 legacy Xray 的 start/stop/restart。
- 不把活跃写模型从 `database/model.Inbound` 迁移到 `proxy_inbounds` / `proxy_clients`。
- 不新增生产 `egress_*` 数据库/API。
- 不把 experimental sing-box 提升为生产默认。
- 不使用 `v-html`、`innerHTML`、`insertAdjacentHTML` 渲染日志、配置、订阅或外部内容。

## 2. 已同步内容

本轮已完成以下上游同步/衍生落地：

1. 记录 `MHSanaei/3x-ui upstream/main` 的合并基线。
2. 移植 finalmask fragment 空/零长度防崩溃修复。
3. 同步 Go 版本到 `1.26.4`，并更新 Xray-core、gRPC、fasthttp、gopsutil、telego、validator、`x/crypto` 等依赖。
4. 补齐 release gate：`go.mod` 的 `go` 指令必须与 `.codex/project.toml` 的 `[stack.backend].version` 一致。
5. 补齐 settings 空值 fallback，避免非空默认配置被空字符串持久化后导致启动或读取失败。
6. 增强 Clash/Mihomo 订阅 routing rules，支持复用订阅设置中的行分隔规则并保留 `MATCH,PROXY` 兜底。
7. 增强订阅 diagnose 输出，返回支持格式和协议能力元数据。
8. 扩展 default-xray CoreManager lifecycle 拒绝回归测试，确保 `Validate/Start/Stop/Restart` 均不接管 legacy 生命周期。
9. 针对上游 `v3.3.0` 的 WARP 出站修复，新增 `panelProxy` 设置、HTTP/HTTPS/SOCKS5 出站客户端构造器，并让 WARP API 请求使用面板代理。
10. 在 Vue Settings 页面暴露 `Panel Outbound Proxy`，保持新 UI 与 legacy setting form 的 `panelProxy` 字段一致。
11. 增强 release metadata gate，加入 `docs/openapi/panel-api.yaml` -> `frontend/public/openapi.json` 生成物陈旧检查，避免 OpenAPI 文档和 Vite public 输入漂移。
12. 对上游订阅相关修复执行本地回归确认：Clash rules、VLESS encryption、public/fallback、Shadowrocket、routing/IPLimit 相关聚焦测试通过。

## 3. 上游雷达清单

当前 `v3.3.0` 最近重点包括：

- `a32c6803d`：WARP API 请求走 `panelProxy`，解决受限网络下 Cloudflare WARP 注册/授权请求不可达问题。
- `d9ccf157c`：新增手动/自动 WARP IP rotation；本项目只先落地低风险代理出站，不直接引入自动轮换 job。
- `1ca5924a` / `f8e89cc84`：MTProto/mtg sidecar、日志可见性与孤儿进程清理；本项目暂缓，避免引入新的生产 sidecar 生命周期。
- `0daedd3d`：subscription-based outbounds auto-update；本项目暂缓，避免引入未经 Phase gate 的外部订阅自动写配置路径。
- `abf6b879`：custom subscription pages；本项目暂缓，需先完成模板/XSS/外部内容渲染审查。
- `e6c1ce9a`、`1c74b995`、`9f31d7d0`、`be8bd4e22`：node tree、多跳归因、access.log 同步与流量 reset 传播；本项目暂缓，避免越过当前 legacy model 与 CoreManager 边界。
- `98ba8803`、`46684dd1`、`2b4e199a`、`e409bc30` 等订阅/Clash/IP limit 修复：本项目优先通过本地订阅矩阵测试确认，必要时再按 TDD 小步移植。
- `a014c017`、`83799d71`、`e56f6c63`：OpenAPI schema/examples/server base path 优化；本项目保持 Vue 自有 OpenAPI 方案，并新增生成物 stale gate。

## 4. 优先级策略

### P0：立即可落地

- 安全/崩溃修复：finalmask、settings fallback、订阅 header/编码/空值防御。
- 依赖与元数据漂移门禁：`go.mod`、`.codex/project.toml`、release gate。
- 订阅 diagnose：只暴露聚合能力和跳过原因，不暴露 client UUID、真实 sub URL、token 或代理凭据。
- WARP API 面板代理：只允许管理员配置的 `panelProxy` 影响面板自身出站请求；无效代理必须回退直连并记录 warning。
- OpenAPI stale gate：生成物不一致时 metadata-only gate 必须失败。

### P1：订阅与协议增强

- Clash/Mihomo routing rules。
- XHTTP/sockopt/Reality/ECH 等上游协议输出修复需先补本项目旧 UI/新 UI/订阅矩阵测试后再移植。
- Shadowrocket base64 行为需单独评估当前客户端兼容性，不能破坏已有 Generic URI 输出。
- Settings UI 可以暴露与 legacy 设置兼容的低风险字段，但必须通过 `frontend/tests/settings-view.test.ts` 覆盖字段绑定。

### P2：CoreManager 与供应链门禁

- default-xray 继续保持 read-only 与 legacy lifecycle owner。
- experimental sing-box 只能在显式 binary/config 存在时作为实验入口使用。
- 二进制下载、迁移脚本和 DB dump/import 工具必须先经过 allowlist、SHA256、临时目录和运行中不覆盖策略设计，再进入生产脚本。
- MTProto/mtg sidecar、OutboundSubscription 自动刷新、node tree/access.log 跨节点同步暂缓；进入生产前必须单独完成生命周期、权限、XSS/SSRF、回滚和旧模型兼容评审。

## 5. 标准执行流程

```powershell
git fetch upstream main --prune
git log --oneline HEAD..upstream/main
git log --oneline upstream/main..HEAD
```

如上游有新提交：

1. 先分类为安全修复、协议输出、UI/i18n、脚本/发布、数据库/迁移、节点同步。
2. 对每个候选提交写最小回归测试，先确认 RED。
3. 只移植必要代码，不覆盖本地 Phase 9/10 边界。
4. 跑聚焦验证：

```powershell
go test ./sub ./web/service ./web/controller ./core/... -count=1
python -m unittest discover -s .codex/skills/superxray-release-cicd/tests -p test_release_gate.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```

5. 发布前跑全量验证和 secret scan。

## 6. 回滚策略

- 单个同步点必须保持小提交；如验证失败，用 `git revert <commit>` 回滚该同步点。
- 生产部署前必须备份 SQLite DB、旧二进制/容器 tag、配置目录和 systemd unit。
- 若订阅输出异常，优先回滚订阅服务相关提交，不回滚无关 UI 或 CoreManager 代码。
- 若 CoreManager 相关 API 异常，必须确认 legacy Xray lifecycle 不受影响；禁止通过 CoreManager 强行重启 default-xray。
