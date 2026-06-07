# 上游 3x-ui 同步策略与落地雷达

> 更新日期：2026-06-07  
> 上游仓库：`MHSanaei/3x-ui`  
> 当前上游基线：`upstream/main` = `483952cfa`，版本 `3.2.8`  
> 当前项目版本：`3.0.19`

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

## 3. 上游雷达清单

当前 `upstream/main` 最近重点包括：

- `fix(finalmask): validate fragment mask length so empty/zero-min can't crash xray`
- `fix(sub): restore standard base64 for Shadowrocket sub link`
- `feat(nodes): multi-hop node attribution for chained sub-nodes`
- `fix(panel): normalize XHTTP/sockopt/Reality wire output and validate REALITY target`
- `fix(iplimit): skip stale access-log emails after client rename/delete`
- `fix(sub): don't project public inbounds through a fallback master`
- `fix(finalmask): treat sudoku customTables as array of tables`
- `feat(x-ui.sh): add migrateDB command for SQLite .db <-> .dump`
- `fix(outbound): import ech and pcs from TLS share links`
- `fix(external-proxy): relabel "Host" as "Address", add per-entry ECH`

## 4. 优先级策略

### P0：立即可落地

- 安全/崩溃修复：finalmask、settings fallback、订阅 header/编码/空值防御。
- 依赖与元数据漂移门禁：`go.mod`、`.codex/project.toml`、release gate。
- 订阅 diagnose：只暴露聚合能力和跳过原因，不暴露 client UUID、真实 sub URL、token 或代理凭据。

### P1：订阅与协议增强

- Clash/Mihomo routing rules。
- XHTTP/sockopt/Reality/ECH 等上游协议输出修复需先补本项目旧 UI/新 UI/订阅矩阵测试后再移植。
- Shadowrocket base64 行为需单独评估当前客户端兼容性，不能破坏已有 Generic URI 输出。

### P2：CoreManager 与供应链门禁

- default-xray 继续保持 read-only 与 legacy lifecycle owner。
- experimental sing-box 只能在显式 binary/config 存在时作为实验入口使用。
- 二进制下载、迁移脚本和 DB dump/import 工具必须先经过 allowlist、SHA256、临时目录和运行中不覆盖策略设计，再进入生产脚本。

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
