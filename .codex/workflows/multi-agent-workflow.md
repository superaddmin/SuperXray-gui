# SuperXray-gui 多代理协作机制

## 默认流程

```text
任务进入
  -> superxray-ui-program-manager 判断阶段和范围
  -> routing.toml 选择主责代理
  -> 主责代理实施或分析
  -> security/test/e2e/docs/release 按影响面审查
  -> 汇总验证、风险和回滚
```

## 上下文传递

- 使用 `.codex/context/handoff-template.md`。
- 每次交接必须包含目标、阶段、路径、已读上下文、验证状态、风险与回滚。
- 接收代理只补读自己职责相关的文件，不重复扫描全仓库。
- 多个代理意见冲突时，由 `superxray-ui-program-manager` 根据阶段门禁裁决。

## 并行协作

可以并行：

- 前端纯 UI 适配与后端只读 API 测试补充。
- Go 单元测试补充与文档同步。
- DevOps 元数据检查与 release notes 准备。

不应并行：

- 同一文件或同一 API 合约的双代理编辑。
- 数据模型变更与订阅输出变更，除非 database steward 先给出契约。
- CoreManager 生命周期变更与旧 Xray lifecycle 变更。

## 审查门禁

| 影响面 | 必须审查 |
| --- | --- |
| 用户输入、鉴权、下载、导入、外部请求、二进制执行 | `superxray-security-gate` |
| 数据模型、迁移、备份恢复 | `superxray-database-steward` |
| 协议、订阅、Xray config | `superxray-subscription-protocol-specialist` |
| 用户可见 UI | `superxray-e2e-gate` |
| Docker、CI、安装脚本 | `superxray-devops-cicd-maintainer` |
| 版本、CHANGELOG、发布资产 | `superxray-release-gate` |

## 冲突裁决

裁决顺序：

1. 运行代码事实。
2. 测试和 E2E 结果。
3. `plans/STATUS.md` 和 phase gates。
4. 架构文档和 README。
5. 代理假设。

如果任务越界，输出当前阶段可做的最小替代方案。
