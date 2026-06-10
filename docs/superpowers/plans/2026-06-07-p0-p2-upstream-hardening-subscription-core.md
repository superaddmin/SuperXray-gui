# P0-P2 上游硬化、订阅增强与 CoreManager 门禁 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不破坏既有 Xray 生命周期、旧 UI 回退和 legacy 数据契约的前提下，落地 P0-P2 的低风险上游同步治理、订阅/协议增强、CoreManager 边界与发布门禁硬化。

**Architecture:** 本轮只做最小可验证增量：P0 先补齐版本漂移门禁、settings 空值 fallback 与上游同步策略文档；P1 在现有订阅服务内增强 Clash/Mihomo routing 与 diagnose 元数据；P2 通过测试和 release gate 固化 CoreManager/sing-box 边界，不迁移旧模型、不接管 legacy Xray 生命周期、不新增生产 `egress_*` 数据路径。

**Tech Stack:** Go 1.26.4、Gin、GORM、goccy/go-yaml、Python 3 release gate、Vue/Vite 现有验证链路。

---

## 变更文件结构

- Create: `docs/superpowers/plans/2026-06-07-p0-p2-upstream-hardening-subscription-core.md`
  - 本计划与回滚/部署前置条件。
- Create: `docs/upstream-sync-policy.md`
  - 上游 3x-ui 同步雷达、选择性移植准则、P0-P2 后续策略。
- Modify: `.codex/project.toml`
  - 将项目治理里的 Go 版本同步为 `1.26.4`，与 `go.mod` 一致。
- Modify: `.codex/skills/superxray-release-cicd/scripts/release_gate.py`
  - metadata gate 增加 `go.mod` 与 `.codex/project.toml` Go 版本一致性检查。
- Modify/Test: `.codex/skills/superxray-release-cicd/tests/test_release_gate.py`
  - TDD 覆盖 Go 版本漂移失败和一致时通过。
- Modify/Test: `web/service/setting.go`、`web/service/setting_test.go`
  - TDD 覆盖 DB 中必需配置为空时回落默认值，避免空端口/空路径导致运行时错误。
- Modify/Test: `sub/subClashService.go`、`sub/subscription_output_matrix_test.go`
  - TDD 覆盖 Clash/Mihomo 使用行分隔 routing rules，并保留默认 `MATCH,PROXY`。
- Modify/Test: `sub/subscription_diagnostic.go`、`sub/subscription_diagnostic_test.go`
  - TDD 覆盖 diagnose 返回支持格式与协议，不泄漏 token/UUID/sub URL。
- Modify/Test: `web/service/core_service_test.go`
  - TDD 覆盖 default-xray 的 Validate/Start/Stop/Restart 均拒绝 CoreManager 生命周期接管。

---

## Task 1: P0 发布门禁与项目 Go 版本一致性

**Files:**
- Modify: `.codex/project.toml`
- Modify: `.codex/skills/superxray-release-cicd/scripts/release_gate.py`
- Test: `.codex/skills/superxray-release-cicd/tests/test_release_gate.py`

- [ ] **Step 1: Write the failing tests**

新增两个测试：

```python
def test_check_project_go_version_metadata_fails_on_drift(self):
    root = self.make_root()
    self.write(root / "go.mod", "module example.test/app\n\ngo 1.26.4\n")
    self.write(root / ".codex/project.toml", '[stack.backend]\nversion = "1.26.3"\n')
    gate = self.gate(root)
    with self.assertRaisesRegex(RuntimeError, "Go version drift"):
        gate.check_project_go_version_metadata()

def test_check_project_go_version_metadata_passes_when_versions_match(self):
    root = self.make_root()
    self.write(root / "go.mod", "module example.test/app\n\ngo 1.26.4\n")
    self.write(root / ".codex/project.toml", '[stack.backend]\nversion = "1.26.4"\n')
    gate = self.gate(root)
    gate.check_project_go_version_metadata()
```

- [ ] **Step 2: Run RED**

Run:

```powershell
python -m unittest discover -s .codex/skills/superxray-release-cicd/tests -p test_release_gate.py
```

Expected: FAIL，因为 `Gate.check_project_go_version_metadata` 尚不存在。

- [ ] **Step 3: Implement minimal release gate**

在 metadata checks 中加入 `check_project_go_version_metadata`，解析 `go.mod` 的 `go` 指令和 `.codex/project.toml` 的 `[stack.backend].version`，不一致时报错。

- [ ] **Step 4: Update project metadata**

把 `.codex/project.toml` 的 `[stack.backend].version` 从 `1.26.3` 改为 `1.26.4`。

- [ ] **Step 5: Run GREEN**

Run:

```powershell
python -m unittest discover -s .codex/skills/superxray-release-cicd/tests -p test_release_gate.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```

Expected: PASS。

---

## Task 2: P0 settings 空值 fallback

**Files:**
- Modify: `web/service/setting.go`
- Test: `web/service/setting_test.go`

- [ ] **Step 1: Write the failing test**

新增测试：DB 中 `webPort` 保存为空字符串时，`GetPort()` 返回默认 `2053`，避免启动配置因空值解析失败。

- [ ] **Step 2: Run RED**

Run:

```powershell
go test ./web/service -run TestSettingServiceFallsBackToDefaultForEmptyRequiredSetting -count=1
```

Expected: FAIL，当前 `strconv.Atoi("")` 返回错误。

- [ ] **Step 3: Implement minimal fallback**

在 `getString` 读到已存在 setting 但值为空时，如果 `defaultValueMap[key]` 是非空值，则返回默认值；默认就是空值的字段继续返回空，避免覆盖用户可留空配置。

- [ ] **Step 4: Run GREEN**

Run:

```powershell
go test ./web/service -run TestSettingServiceFallsBackToDefaultForEmptyRequiredSetting -count=1
```

Expected: PASS。

---

## Task 3: P1 Clash/Mihomo routing rules

**Files:**
- Modify: `sub/subClashService.go`
- Test: `sub/subscription_output_matrix_test.go`

- [ ] **Step 1: Write the failing test**

新增测试：当订阅设置中包含行分隔 routing rules 时，Clash YAML 的 `rules` 使用这些规则，并保留兜底 `MATCH,PROXY`。

- [ ] **Step 2: Run RED**

Run:

```powershell
go test ./sub -run TestSubClashBuildRulesUsesConfiguredRoutingRules -count=1
```

Expected: FAIL，因为 `buildClashRules` 或 routing 注入尚不存在。

- [ ] **Step 3: Implement minimal routing**

在 `SubClashService` 增加 `routingRules string` 与 `WithRoutingRules`，解析非空、非注释、包含逗号的规则；若没有 `MATCH,` 规则则追加 `MATCH,PROXY`。默认空配置保持原行为。

- [ ] **Step 4: Wire controller**

在 `NewSUBController` 中使用 `NewSubClashService(sub).WithRoutingRules(subRoutingRules)`。

- [ ] **Step 5: Run GREEN**

Run:

```powershell
go test ./sub -run TestSubClashBuildRulesUsesConfiguredRoutingRules -count=1
go test ./sub -count=1
```

Expected: PASS。

---

## Task 4: P1 订阅 diagnose 增强

**Files:**
- Modify: `sub/subscription_diagnostic.go`
- Test: `sub/subscription_diagnostic_test.go`

- [ ] **Step 1: Write the failing test**

新增测试：diagnose JSON 返回 `supportedFormats` 和 `supportedProtocols`，仅包含格式/协议能力元数据，不返回 client UUID、订阅完整 URL 或 token。

- [ ] **Step 2: Run RED**

Run:

```powershell
go test ./sub -run TestDiagnoseSubscriptionInboundsReportsSupportedFormatsAndProtocols -count=1
```

Expected: FAIL，因为字段尚不存在。

- [ ] **Step 3: Implement minimal fields**

在 `SubscriptionDiagnostic` 增加：

```go
SupportedFormats   []string `json:"supportedFormats"`
SupportedProtocols []string `json:"supportedProtocols"`
```

并从现有 `subscriptionClientProtocols()` 与 `subscriptionPeerProtocols()` 派生稳定去重列表。

- [ ] **Step 4: Run GREEN**

Run:

```powershell
go test ./sub -run TestDiagnoseSubscriptionInboundsReportsSupportedFormatsAndProtocols -count=1
go test ./sub -count=1
```

Expected: PASS。

---

## Task 5: P2 CoreManager lifecycle 边界回归

**Files:**
- Modify: `web/service/core_service_test.go`

- [ ] **Step 1: Write the failing/expanded boundary test**

扩展 default-xray 测试：Validate/Start/Stop/Restart 均必须返回 `core.ErrLifecycleUnsupported`。

- [ ] **Step 2: Run boundary test**

Run:

```powershell
go test ./web/service -run TestDefaultXrayAdapterRejectsCoreManagerLifecycle -count=1
```

Expected: PASS（若失败则修复 adapter），作为 P2 防回归门禁。

---

## Task 6: Documentation, local verification and review

**Files:**
- Create: `docs/upstream-sync-policy.md`
- No production code unless tests are already green.

- [ ] **Step 1: Add upstream sync policy**

记录上游 3x-ui `3.2.8` 与本项目 `3.3.2` 的选择性同步策略、禁止直接覆盖的本地能力、后续雷达清单。

- [ ] **Step 2: Run focused verification**

Run:

```powershell
go test ./sub ./web/service ./web/controller ./core/... -count=1
python -m unittest discover -s .codex/skills/superxray-release-cicd/tests -p test_release_gate.py
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --ci --metadata-only
```

- [ ] **Step 3: Run full local verification**

Run:

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
cd frontend; npm run typecheck; npm run lint; npm run test; npm run build
cd ..
python scripts/secret_scan.py
rg "v-html|innerHTML|insertAdjacentHTML" web/html frontend/src -n
rg "proxy_inbounds|proxy_clients" core web/controller web/middleware web/service database/model frontend/src web/ui -n
```

- [ ] **Step 4: Multi-role review**

调用多角色代理做交叉审核：Go reviewer、Security reviewer、Subscription/Core domain reviewer、Release/DevOps reviewer。对审查意见逐条判断并只接受有证据的修改。

---

## 生产部署前置条件

本地通过不等于生产已稳定。生产更新必须先获得以下信息后才能执行：

1. 服务器地址、SSH 端口、登录方式和允许使用的账号。
2. 部署方式：systemd 二进制、Docker/GHCR、GitHub Release、还是现有 `x-ui.sh/update.sh`。
3. 服务名、安装目录、数据库路径、备份目录、日志目录。
4. 面板 base path、端口、反向代理/TLS 方式。
5. 维护窗口、是否允许重启 Xray/面板、是否允许 mutation E2E。
6. 回滚版本、二进制/容器 tag、数据库备份和恢复步骤。

生产执行顺序：

```powershell
# 本地
go build -o bin/SuperXray.exe ./main.go
python scripts/secret_scan.py

# 服务器（示例，占位符，不含真实凭证）
ssh <USER>@<SERVER>
sudo systemctl stop <SERVICE>
sudo cp <DB_PATH> <BACKUP_DIR>/<timestamp>.db
sudo install -m 0755 <NEW_BINARY> <INSTALL_DIR>/SuperXray
sudo systemctl start <SERVICE>
sudo systemctl status <SERVICE> --no-pager
curl -fsS http://127.0.0.1:<PORT>/<BASE_PATH>/panel/
```

未获得上述信息前，不在生产服务器执行任何更新命令。
