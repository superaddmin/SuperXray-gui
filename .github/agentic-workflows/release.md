# GitHub Agentic Workflow: Release

本文件是 SuperXray-gui 的智能发布流程说明书，供 GitHub Copilot、Codex、其他 AI Agent 和维护者共同使用。目标是让 Agent 能够在不猜测的情况下完成版本准备、验证、推送、发布监控和失败回滚。

## 入口文件

- `.github/workflows/release.yml`：Linux `amd64` / `arm64` 二进制包构建与 GitHub Release 上传。
- `.github/workflows/docker.yml`：GHCR 多架构镜像发布。
- `.github/workflows/test-arm64.yml`：ARM64 交叉编译与 QEMU 执行验证。
- `.codex/skills/superxray-release-cicd/scripts/release_gate.py`：本地和 CI 可复用的发布门禁。
- `.codex/skills/superxray-release-cicd/SKILL.md`：Codex 发布技能入口。

## Agent 执行契约

AI Agent 处理发布任务时必须遵守以下顺序：

1. 先读取本文件、`CHANGELOG.md`、`config/version`、相关 workflow 和当前 `git status`。
2. 先明确当前任务域：版本准备、发布修复、tag 发布、发布失败排障、回滚，不能混为一次大改。
3. 不创建临时发布脚本；已有重复校验必须进入 `release_gate.py` 或 GitHub Actions。
4. 不强推 tag，除非确认 GitHub Release 尚未公开成功，且使用 `--force-with-lease` 保护远端对象。
5. 不发布空 Release；Release Notes 必须来自 `CHANGELOG.md` 的对应版本段落。
6. 不把 Windows、legacy Linux 架构或额外制品加入发布，除非发布策略文件明确变更。
7. 完成后必须给出验证命令、Actions 链接、Release 链接、资产清单和剩余风险。

## 发布状态机

发布任务按以下状态推进：

```text
Draft metadata
  -> Local gate
  -> Push main
  -> Main CI green
  -> Create annotated tag
  -> Binary release green
  -> Docker release green
  -> Public verification
  -> Done
```

任何状态失败时，Agent 必须停在失败点，记录首个阻塞点和最小修复，不继续创建新 tag。

## 版本规则

- `config/version` 必须是 `X.Y.Z` 或 `X.Y.Z-prerelease`。
- Git tag 必须是 `vX.Y.Z` 或 `vX.Y.Z-prerelease`。
- tag 去掉前缀 `v` 后必须与 `config/version` 完全一致。
- `CHANGELOG.md` 必须包含 `## [X.Y.Z]` 对应段落。
- README、docs、安装示例和 Docker 示例不得残留旧版本号。

## 发布前门禁

在仓库根目录执行：

```powershell
git status --short --branch
go test ./...
go vet ./...
python .codex/skills/superxray-release-cicd/scripts/release_gate.py --tag vX.Y.Z --install-tools
```

如果工作区包含用户未提交改动，Agent 只能暂存和提交本次发布相关文件；不能回滚或覆盖无关改动。

## 标准发布步骤

1. 更新 `config/version`。
2. 更新 `CHANGELOG.md` 对应版本段落。
3. 同步 README、docs、安装示例和 Docker 示例中的版本号。
4. 运行发布前门禁。
5. 提交：

   ```bash
   git commit -m "chore: 发布 vX.Y.Z"
   ```

6. 推送 `main`：

   ```bash
   git push origin main
   ```

7. 等待 `main` 上的 `Release SuperXray`、`Multi-Arch Test` 和 CodeQL 通过。
8. 创建 annotated tag：

   ```bash
   git tag -a vX.Y.Z -m "release: vX.Y.Z"
   git push origin vX.Y.Z
   ```

9. 监控 tag 触发的 workflow。

## 发布结果校验

GitHub Release 必须包含：

```text
x-ui-linux-amd64.tar.gz
x-ui-linux-arm64.tar.gz
```

GHCR 镜像必须至少包含：

```text
ghcr.io/superaddmin/superxray-gui:X.Y.Z
ghcr.io/superaddmin/superxray-gui:vX.Y.Z
linux/amd64
linux/arm64
```

可使用 GitHub API 或 GHCR Registry API 查询结果；本机没有 Docker CLI 时，不要把缺少 Docker 当作发布失败。

## 失败处理

### tag 已推送但二进制 Release 未生成

1. 查询 GitHub Release 是否存在。
2. 查询远端 tag 对象。
3. 修复 `main` 上的首个失败点。
4. 推送 `main`。
5. 仅当 Release 未公开成功时，使用远端对象保护更新 tag：

   ```bash
   git tag -f -a vX.Y.Z -m "release: vX.Y.Z"
   git push --force-with-lease=refs/tags/vX.Y.Z:<old-tag-object> origin refs/tags/vX.Y.Z:refs/tags/vX.Y.Z
   ```

### Release 已公开但存在缺陷

不要改写 tag。发布新的 patch 版本，并在 `CHANGELOG.md` 说明修复内容。

### 部分资产上传

如果是草稿或尚未被用户消费的失败发布，可删除失败资产后 rerun workflow。稳定版公开后，优先走 patch 版本。

## Agent 输出模板

发布完成后，Agent 应输出：

- 提交 SHA 和 tag。
- GitHub Release 链接。
- 二进制资产名称和架构。
- Docker/GHCR workflow 结果。
- 本地验证命令和结果。
- 未处理的本地改动或风险。
