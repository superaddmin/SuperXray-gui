# alireza0/x-ui v1.11.4 集成记录

## 1. 目标与结论

本次集成完整抓取 `alireza0/x-ui v1.11.4` Git 对象，再按功能移植到 SuperXray。两个仓库没有共同祖先，因此不使用 `--allow-unrelated-histories` 直接合并文件树，避免上游旧 HTML UI、数据模型和发布元数据覆盖本地实现。

本地保护边界：

- Vue 源码继续位于 `frontend/src`，`web/ui` 只由 Vite 构建生成。
- 活跃入站写模型继续使用 `database/model.Inbound`。
- 保留订阅、Gateway Egress MVP、WARP Matrix 和面板代理能力。
- CoreManager 不接管 legacy Xray 的启动、停止和重启。
- 不重新引入 `web/html`、`web/assets` 或 `/panel/legacy*`。
- 保留 SuperXray 版本 `3.4.3`，上游 tag 只作为供应方基线。
- 状态变更 API 继续经过登录鉴权与 Session CSRF 中间件。

## 2. 固定对象与回滚点

```text
upstream v1.11.3: 419fce7c564b5e66bb5285cbd01814ec40f5f869
upstream v1.11.4: 73b050c95e3fb25128baedc8668d72a45a7b5b6f
local vendor tag: vendor/alireza0-xui/v1.11.4
integration branch: integration/upstream-alireza0-v1.11.4
integration worktree: F:\SuperXray-integrate-alireza0-v1.11.4
pre-integration ref: backup/pre-alireza0-v1.11.4-20260810-193827
pre-integration commit: 807f6280e32b2dba88cbcf910c40a40fe3e1fb9f
bundle: F:\SuperXray-backups\SuperXray-main-20260810-193827.bundle
bundle SHA256: 0E9E82BF86F14A22A0FA133E836FB3347CE8A5E6E840455BCEEE17BA7C2DA8AD
```

回滚引用和 bundle 在合并后继续保留。恢复前先停止写入并备份运行数据库；代码回滚优先使用 `git revert`，不改写已经推送的 `main` 历史。

## 3. 状态与远程检查

在 PowerShell 中执行：

```powershell
git status --short --branch
git remote -v
git worktree list --porcelain
git log --graph --decorate --oneline --all --max-count=30
git fsck --full
```

配置只读供应方远程：

```powershell
git remote add alireza0-xui https://github.com/alireza0/x-ui.git
git remote set-url --push alireza0-xui DISABLED
git fetch alireza0-xui refs/tags/v1.11.4:refs/tags/vendor/alireza0-xui/v1.11.4
git config --get-regexp '^remote\.alireza0-xui\.'
```

核对 tag 和 GitHub ref 后固定提交：

```powershell
git rev-parse vendor/alireza0-xui/v1.11.4^{commit}
git show --no-patch --format=fuller vendor/alireza0-xui/v1.11.4
git log --oneline 419fce7c564b5e66bb5285cbd01814ec40f5f869..73b050c95e3fb25128baedc8668d72a45a7b5b6f
```

## 4. 分支策略与实施

为保持 `main` 可用，使用独立分支和 worktree：

```powershell
git branch backup/pre-alireza0-v1.11.4-20260810-193827 main
git bundle create F:\SuperXray-backups\SuperXray-main-20260810-193827.bundle --all
Get-FileHash F:\SuperXray-backups\SuperXray-main-20260810-193827.bundle -Algorithm SHA256
git worktree add F:\SuperXray-integrate-alireza0-v1.11.4 -b integration/upstream-alireza0-v1.11.4 main
```

功能组保持独立提交，便于逐组回滚：

```text
b1865e834 chore: 同步 x-ui v1.11.4 Go 依赖更新
b32798947 feat: 安全移植 x-ui v1.11.4 自签证书能力
7d998f747 feat: 移植 x-ui v1.11.4 Hysteria 与 TUN 配置
74ae3cf40 feat: 移植 x-ui v1.11.4 XHTTP 与 XMUX 配置
```

## 5. 上游提交覆盖矩阵

| 上游提交                         | 处理     | 本地保护措施                                                                           |
| -------------------------------- | -------- | -------------------------------------------------------------------------------------- |
| `5eb6b5215` Hysteria 默认值      | 移植     | 保留 Salamander、UDP Hop、`finalmask` 和未知字段                                       |
| `9581110da` TUN 配置             | 移植     | 同时读取当前 schema 与 `MTU/Gateway/DNS` 旧别名，校验 MTU/CIDR/DNS/接口名              |
| `e99ff1089` outbound XHTTP extra | 等价覆盖 | 本地 outbound 已有原始 `Stream Settings JSON`，未知 `extra` 可无损读写                 |
| `deb9386d2` 删除 WARP modal      | 不采用   | 上游改动针对旧 HTML routing modal；本地 WARP Matrix 是受测试保护的专属功能             |
| `11f4bbe29` 全局 JSON 收缩       | 不采用   | 保留显式默认值和 `tcpFastOpen` 旧配置兼容；XHTTP 仅对目标字段受控省略                  |
| `93990f659` 隐藏客户端 JSON      | 等价覆盖 | 本地客户端使用独立表单/API，并保留高级 JSON 兼容入口，避免多客户端字段被快照覆盖       |
| `2111bf0c5` XHTTP/XMUX           | 移植     | 已知字段迁入 `xhttpSettings.extra`，保留未知根字段、未知 `extra` 与自定义 headers      |
| `767abf52e` 自签名证书           | 安全移植 | ECDSA P-256、PKCS#8、一年期服务器证书；SAN 数量/长度/DNS/IP 校验；登录与 CSRF 保持生效 |
| `efc083b5f` Go 依赖              | 移植     | 使用本地 module path 和发布工具链，不替换本地工程布局                                  |
| `73b050c95` 版本号               | 不采用   | 产品版本继续由 `config/version=3.4.3` 管理                                             |

## 6. 冲突排查与保留方案

合并前比较共同祖先两侧变化：

```powershell
git merge-base main integration/upstream-alireza0-v1.11.4
git rev-list --left-right --count main...integration/upstream-alireza0-v1.11.4
git diff --name-status main...integration/upstream-alireza0-v1.11.4
git diff --check main...integration/upstream-alireza0-v1.11.4
```

若出现冲突，先列出冲突类型，不批量选择 `ours` 或 `theirs`：

```powershell
git diff --name-only --diff-filter=U
git ls-files -u
git diff --cc
```

按所有权处理：源码逐段合并；`web/ui` 不手工解冲突，而是在源文件合并并通过测试后执行 `npm run build` 重新生成；`go.mod/go.sum` 使用 `go mod tidy` 和 `go mod verify` 校验；OpenAPI 以 `docs/openapi/panel-api.yaml` 为事实源重新生成。

## 7. 合并与推送

先确认 `origin/main` 没有未知新提交，并确保两个 worktree 均无未提交改动：

```powershell
git fetch origin --prune
git status --short --branch
git log --oneline main..origin/main
git log --oneline origin/main..main
```

在主工作树执行非快进合并，保留集成边界：

```powershell
git merge --no-ff integration/upstream-alireza0-v1.11.4 -m "merge: 集成 alireza0 x-ui v1.11.4"
git status --short --branch
git log --graph --decorate --oneline --max-count=20
git push origin main
```

推送后核对本地与远端对象一致：

```powershell
git fetch origin main
$localMain = git rev-parse main
$remoteMain = git rev-parse origin/main
if ($localMain -ne $remoteMain) { throw "origin/main 与本地 main 不一致" }
git ls-remote origin refs/heads/main
```

## 8. 最终验证

```powershell
go mod verify
go test ./... -count=1
go vet ./...
go build -o bin/SuperXray.exe ./main.go

Push-Location frontend
npm run typecheck
npm run lint
npm run test
npm run build
npm audit --json
Pop-Location

python scripts/secret_scan.py
python .codex/skills/superxray-project-context/tests/test_validate_codex_config.py
python .codex/skills/superxray-project-context/scripts/validate_codex_config.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only

rg "v-html|innerHTML|insertAdjacentHTML" frontend/src web/ui sub -n
rg "unsafe-inline|unsafe-eval" frontend/src web/ui -n
rg "proxy_inbounds|proxy_clients" core web/controller web/middleware web/service database/model frontend/src web/ui -n
rg "/panel/legacy|web/html|web/assets" web frontend -n
```

`rg` 在“未命中”时返回退出码 1，该结果对上述禁止项扫描表示通过。真实 Xray 生命周期和浏览器 E2E 仍按部署环境的 `SUPERXRAY_E2E_*` 条件执行。
