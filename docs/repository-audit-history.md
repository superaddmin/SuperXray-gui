# 仓库清理历史记录（2026-10-14 至 2026-10-15）

> **历史性质**：本文合并了原 `repository-audit-2026-10-14.md` 与 `repository-audit-2026-10-15.md` 的独有结论，只保留时间线、关键数字、隔离位置和验证范围。当前工作区状态以 [2026-10-22 清理与完整性核对](repository-audit-2026-10-22.md) 为准，文中数字不得当作当前文件清单。

## 时间线

| 日期 | 基线 | 处理 | 结果 |
|---|---|---|---|
| 2026-10-14 | `main` / `d4a8e8913` / 产品 `3.4.3` | 首次全量清理，剔除依赖、构建产物和缓存 | 从 26,673 个文件、337,098,121 字节降至 497 个文件、8,871,489 字节 |
| 2026-10-15 | `integration/3x-ui-v3.10.2` / `d87b7c157` / Go `1.27.2` | 上游 v3.10.2 适配后的第二轮清理 | 最终工作区 565 个文件（不含 `.git`） |
| 2026-10-22 | `integration/3x-ui-v3.10.2` / `dba9c3393` / 产品 `3.4.3` | 删除旧部署可重建产物、合并文档并复验 | 557 个文件、49,398,724 字节，详见当前审计 |

## 2026-10-14 轮

- 永久删除被工具策略拒绝，确认的冗余产物改为移出工作区到 `F:/SuperXray-gui-cleanup-20261015/`，属于可恢复隔离，未释放磁盘空间。
- 隔离内容共 26,177 个文件、328,291,744 字节（约 313.08 MiB）：`frontend/node_modules/`、`bin/x-ui-linux-amd64-hotfix`、`build/x-ui-linux-amd64`、日志标记、`.playwright-mcp/`、TypeScript 缓存、Python 字节码和测试日志。
- `build/` 中 9 份私有运维脚本、配置和备份移到 `.local/ops/`，移动前后 SHA-256 一致；`.gitignore` 与 `.dockerignore` 同步忽略 `.local/`。
- 保留 `web/ui/`、`frontend/public/`、`media/`、`docs/assets/`、`.env`、`.vscode/`、`x-ui/x-ui.db`、`docs/private-ops/` 和 `.local/ops/`。
- 验证通过：`go test`、`go vet`、`go mod tidy -diff`、`go build`（输出 `3.4.3`）、前端 `typecheck`/`lint`/`build`、125 项前端测试、`secret_scan.py`、`release_gate.py --ci --metadata-only`、`git diff --check` 和本地链接检查。
- 当时未解决：13 个既有前端文件不符合 Prettier；本机 staticcheck 与 Go 1.27 export data 不兼容；未运行 E2E、Docker、Linux 打包和外部环境验证。

## 2026-10-15 轮

- 清理对象包括上游适配、Docker 和测试修复，未回退业务改动，也未提交或推送。
- 隔离目录为 `F:/SuperXray-gui-cleanup-20261015-v3102/`，分三批共 26,852 个文件、1,164,882,704 字节（约 1110.92 MiB）；原路径、文件数和字节数记录在隔离根目录的 `manifest.json`、`post-validation/manifest.json` 和 `final-generated/manifest.json`。
- 清理前较大目录：`tmp/` 约 787.99 MiB、前端目录（含依赖）约 196.62 MiB、`bin/` 约 150.41 MiB、`.local/` 约 149.95 MiB、根 `node_modules/` 约 13.28 MiB；`.git/` 保留。
- 保留 `x-ui/x-ui.db`、`.local/ops/`、`docs/private-ops/`、`bin/xray-windows-amd64.exe`、`web/ui/`、`frontend/public/`、`media/`、`docs/assets/` 和四份上游 TSV。
- 验证通过：`go mod tidy -diff`、`go mod verify`、OpenAPI 契约测试、前端 `format`/`typecheck`、144 项前端测试、`npm run build`、Go 全量测试与 `go vet`、`release_gate.py --ci --metadata-only`、`secret_scan.py`、`git diff --check`、文档链接/JSON/YAML/编码检查。
- 未重复运行 Docker 双架构构建、QEMU arm64 启动和浏览器 E2E；这些属于此前集成轮的历史验证。

## 与当前文档的关系

- 当前审计见 [repository-audit-2026-10-22.md](repository-audit-2026-10-22.md)。
- 上游差异与固定基线证据见 [v3.10.2 集成记录](upstream-3x-ui-v3.10.2-integration.md)、[上游同步政策](upstream-sync-policy.md) 和四份 `upstream-3x-ui-v3.10.2-*.tsv`。TSV 是固定快照，可能仍列出后来合并或删除的历史文档名，不代表这些文件当前存在。
- 原 `passwall-openwrt-ai-routing-playbook.md` 已并入 [服务器部署 + OpenWrt 路由 + AI 出口治理统一总览](operations-ai-routing-overview.md)。
