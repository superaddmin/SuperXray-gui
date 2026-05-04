# SuperXray-gui 技术计划文档目录

最后更新：2026-05-04

## 目录定位

`plans/` 只保留“计划、路线图、阶段记录、状态追踪”类文档，不承载用户手册和代码级长期说明。长期产品/开发文档应沉淀到 `docs/`，阶段性施工和管理对齐保留在 `plans/`。

## 新目录架构

```text
plans/
├── README.md
├── STATUS.md
├── 00-governance/
│   └── documentation-system-plan.md
├── 01-strategy/
│   └── ui-first-xray-stable-multi-core-roadmap.md
├── 02-architecture/
│   ├── backend-multi-core-architecture-plan.md
│   └── protocol-capability-completion-plan.md
├── 03-ui-design/
│   └── multi-core-ui-design-plan.md
└── 04-ui-first-execution/
    ├── phase-00-e2e-baseline.md
    ├── phase-00-xray-parity-checklist.md
    ├── phase-01-frontend-shell.md
    ├── phase-02-go-static-integration.md
    ├── phase-03-api-sdk-types.md
    ├── phase-04-readonly-dashboard-logs-config.md
    ├── phase-05-xray-lifecycle-config.md
    ├── phase-06a-inbounds-vmess-vless.md
    ├── phase-06b-inbounds-trojan-shadowsocks.md
    ├── phase-06c-06d-hysteria2-wireguard-stream.md
    ├── phase-06e-inbounds-wrap-up.md
    ├── phase-07a-settings-subscription-backup.md
    ├── phase-08-default-entry-gray-switch.md
    ├── phase-09-security-closeout.md
    ├── phase-10-entry-gate-assessment.md
    └── phase-10a-default-xray-readonly-adr.md
```

## 分类原则

| 分类                    | 用途                                     | 主要读者                      | 更新频率                   |
| ----------------------- | ---------------------------------------- | ----------------------------- | -------------------------- |
| `00-governance`         | 文档体系、协作规范、长期知识库规划       | 技术负责人、项目经理、维护者  | 低频，架构或流程变化时更新 |
| `01-strategy`           | 总体施工路线、阶段门禁、优先级裁决       | 管理层、Tech Lead、全体开发者 | 每个大阶段结束后更新       |
| `02-architecture`       | 后端抽象、协议能力、数据模型和未来演进   | 后端开发、架构师、安全审查    | 架构实施前后更新           |
| `03-ui-design`          | 新 UI 信息架构、交互设计、多内核 UI 蓝图 | 前端开发、产品、设计审查      | UI 阶段切换时更新          |
| `04-ui-first-execution` | UI-first 每阶段实施记录、验收和回滚      | 实施工程师、测试、发布负责人  | 每个阶段或子阶段完成后更新 |

## 推荐阅读路径

### 管理层汇报

1. [STATUS.md](STATUS.md)
2. [01-strategy/ui-first-xray-stable-multi-core-roadmap.md](01-strategy/ui-first-xray-stable-multi-core-roadmap.md)
3. [04-ui-first-execution/phase-08-default-entry-gray-switch.md](04-ui-first-execution/phase-08-default-entry-gray-switch.md)
4. [04-ui-first-execution/phase-10-entry-gate-assessment.md](04-ui-first-execution/phase-10-entry-gate-assessment.md)
5. [04-ui-first-execution/phase-10a-default-xray-readonly-adr.md](04-ui-first-execution/phase-10a-default-xray-readonly-adr.md)

### 新成员理解项目路线

1. [00-governance/documentation-system-plan.md](00-governance/documentation-system-plan.md)
2. [01-strategy/ui-first-xray-stable-multi-core-roadmap.md](01-strategy/ui-first-xray-stable-multi-core-roadmap.md)
3. [03-ui-design/multi-core-ui-design-plan.md](03-ui-design/multi-core-ui-design-plan.md)
4. [02-architecture/backend-multi-core-architecture-plan.md](02-architecture/backend-multi-core-architecture-plan.md)

### 当前开发执行

1. [04-ui-first-execution/phase-00-xray-parity-checklist.md](04-ui-first-execution/phase-00-xray-parity-checklist.md)
2. [04-ui-first-execution/phase-00-e2e-baseline.md](04-ui-first-execution/phase-00-e2e-baseline.md)
3. [04-ui-first-execution/phase-06a-inbounds-vmess-vless.md](04-ui-first-execution/phase-06a-inbounds-vmess-vless.md)
4. [04-ui-first-execution/phase-06b-inbounds-trojan-shadowsocks.md](04-ui-first-execution/phase-06b-inbounds-trojan-shadowsocks.md)
5. [04-ui-first-execution/phase-06c-06d-hysteria2-wireguard-stream.md](04-ui-first-execution/phase-06c-06d-hysteria2-wireguard-stream.md)
6. [04-ui-first-execution/phase-06e-inbounds-wrap-up.md](04-ui-first-execution/phase-06e-inbounds-wrap-up.md)
7. [04-ui-first-execution/phase-07a-settings-subscription-backup.md](04-ui-first-execution/phase-07a-settings-subscription-backup.md)
8. [04-ui-first-execution/phase-08-default-entry-gray-switch.md](04-ui-first-execution/phase-08-default-entry-gray-switch.md)
9. [04-ui-first-execution/phase-09-security-closeout.md](04-ui-first-execution/phase-09-security-closeout.md)
10. [04-ui-first-execution/phase-10-entry-gate-assessment.md](04-ui-first-execution/phase-10-entry-gate-assessment.md)
11. [04-ui-first-execution/phase-10a-default-xray-readonly-adr.md](04-ui-first-execution/phase-10a-default-xray-readonly-adr.md)
12. [STATUS.md](STATUS.md)

## 文档迁移映射

| 原路径                                                                     | 新路径                                                                   | 说明                    |
| -------------------------------------------------------------------------- | ------------------------------------------------------------------------ | ----------------------- |
| `plans/documentation-plan.md`                                              | `plans/00-governance/documentation-system-plan.md`                       | 技术文档体系规划        |
| `plans/SuperXray-gui_ui_first_xray_stable_multi_core_construction_plan.md` | `plans/01-strategy/ui-first-xray-stable-multi-core-roadmap.md`           | UI-first 主路线图       |
| `plans/SuperXray-gui_backend_architecture_plan.md`                         | `plans/02-architecture/backend-multi-core-architecture-plan.md`          | 多内核后端架构方案      |
| `plans/protocol-capability-completion-plan.md`                             | `plans/02-architecture/protocol-capability-completion-plan.md`           | 协议能力补齐计划        |
| `plans/SuperXray-gui_multi_core_ui_design_plan.md`                         | `plans/03-ui-design/multi-core-ui-design-plan.md`                        | 多内核 UI 设计方案      |
| `plans/ui-first-phase0_e2e_baseline.md`                                    | `plans/04-ui-first-execution/phase-00-e2e-baseline.md`                   | Phase 0 E2E 基线        |
| `plans/ui-first-phase0_xray_parity_checklist.md`                           | `plans/04-ui-first-execution/phase-00-xray-parity-checklist.md`          | Phase 0 Xray 等价清单   |
| `plans/ui-first-phase1_frontend_shell.md`                                  | `plans/04-ui-first-execution/phase-01-frontend-shell.md`                 | Phase 1 前端壳          |
| `plans/ui-first-phase2_go_static_integration.md`                           | `plans/04-ui-first-execution/phase-02-go-static-integration.md`          | Phase 2 Go 静态资源接入 |
| `plans/ui-first-phase3_api_sdk_types.md`                                   | `plans/04-ui-first-execution/phase-03-api-sdk-types.md`                  | Phase 3 SDK/类型层      |
| `plans/ui-first-phase4_readonly_dashboard_logs_config.md`                  | `plans/04-ui-first-execution/phase-04-readonly-dashboard-logs-config.md` | Phase 4 只读页面        |
| `plans/ui-first-phase5_xray_lifecycle_config.md`                           | `plans/04-ui-first-execution/phase-05-xray-lifecycle-config.md`          | Phase 5 Xray 管理       |
| `plans/ui-first-phase6a_inbounds_vmess_vless.md`                           | `plans/04-ui-first-execution/phase-06a-inbounds-vmess-vless.md`          | Phase 6a 入站/客户端    |
| 新增                                                                       | `plans/04-ui-first-execution/phase-07a-settings-subscription-backup.md`  | Phase 7a 设置/订阅/备份 |
| 新增                                                                       | `plans/04-ui-first-execution/phase-08-default-entry-gray-switch.md`      | Phase 8 默认入口灰度    |
| 新增                                                                       | `plans/04-ui-first-execution/phase-09-security-closeout.md`              | Phase 9 安全收口        |
| 新增                                                                       | `plans/04-ui-first-execution/phase-10-entry-gate-assessment.md`          | Phase 10 准入门禁评估   |
| 新增                                                                       | `plans/04-ui-first-execution/phase-10a-default-xray-readonly-adr.md`     | Phase 10.1 只读 ADR     |

## 维护规则

1. 新阶段记录统一放入 `04-ui-first-execution/`，命名为 `phase-XX-topic.md`。
2. 总路线、阶段门禁、风险矩阵统一更新 `01-strategy/ui-first-xray-stable-multi-core-roadmap.md`。
3. 后端多内核设计只在 Phase 10 之后进入实施状态；Phase 0-9 只能作为审查参考。
4. 管理层进度统一更新 [STATUS.md](STATUS.md)，不要把状态散落在多个阶段记录里。
5. 如需新增长期产品说明或使用手册，应写入 `docs/`，并在本目录只保留计划入口链接。
