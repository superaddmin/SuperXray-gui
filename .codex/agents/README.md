# SuperXray-gui Project Agents

这些项目级代理配置用于推进 `UI 先行、Xray 稳定迁移、再接多内核` 施工路线。

原则：

- 只放项目专用角色职责和阶段门禁。
- 不复制全局 MCP、sandbox、approval、模型或密钥配置。
- 具体执行以 `plans/01-strategy/ui-first-xray-stable-multi-core-roadmap.md` 为准。
- UI 迁移阶段禁止迁移数据库、禁止提前接入 sing-box、禁止提前深改 CoreManager。

推荐角色链路：

```text
superxray-ui-program-manager
  -> superxray-frontend-migrator
  -> superxray-go-integration
  -> superxray-security-gate
  -> superxray-e2e-gate
  -> superxray-release-gate
```

Phase 0 已落地基线：

| 产物 | 负责人 | 用途 |
|---|---|---|
| `plans/04-ui-first-execution/phase-00-xray-parity-checklist.md` | `superxray-ui-program-manager` | 裁决旧 UI/Xray 等价范围、确认阶段门禁 |
| `plans/04-ui-first-execution/phase-00-e2e-baseline.md` | `superxray-e2e-gate` | 维护旧 UI 可复现流程和测试运行说明 |
| `tests/e2e/legacy-panel.spec.ts` | `superxray-e2e-gate` | 作为新 UI 各阶段迁移的旧行为对照测试 |
| `playwright.config.ts` | `superxray-e2e-gate` | 固化 E2E 产物、浏览器和运行策略 |

阶段推进规则：

- Phase 1 开始前，`superxray-ui-program-manager` 必须确认 Phase 0 清单无 P0 缺口。
- Phase 1/2 只允许 `superxray-frontend-migrator` 与 `superxray-go-integration` 建新 UI 壳和静态入口，不改旧 API 语义。
- Phase 4 以后每迁移一个只读页面，`superxray-e2e-gate` 必须补新 UI 对照测试。
- Phase 5/6 开始写入前，`superxray-security-gate` 必须确认日志/配置预览无 `v-html`，写 API 仍满足 CSRF。
- Phase 10 前，任何代理都不得把 Xray 生命周期切到 CoreManager，也不得新增 sing-box 写路径。
