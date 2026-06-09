# Codex Retrospective Governance And Docs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 基于本轮任务对话的实战复盘，增强 `.codex` 治理/上下文/工作流，并补齐 docs 下的运行态排障、OpenWrt/Passwall/AI 分流避坑指南与复盘文档。

**Architecture:** 先沉淀一份对话复盘与运行态网络排障地图，再把这些经验约束回写到 `.codex/governance.toml`、`.codex/project.toml`、`.codex/routing.toml` 与新 workflow/context 文档中，最后把 docs 中的开发规范、部署说明与 AI 路由文档补成可执行的实战手册。

**Tech Stack:** TOML, Markdown, Python config validation, secret scan

---

### Task 1: 写入复盘与运行态知识地图

**Files:**
- Create: `.codex/context/conversation-retrospective-map.md`
- Create: `.codex/context/runtime-network-debug-map.md`
- Create: `docs/superpowers/retrospectives/2026-06-10-network-routing-retrospective.md`

- [ ] **Step 1: 归纳本轮对话暴露的陷阱与认知偏差**
- [ ] **Step 2: 形成运行态网络排障证据优先级地图**
- [ ] **Step 3: 写入项目级复盘文档，包含证据链、误判模式、修正策略**

### Task 2: 强化 `.codex` 治理与工作流

**Files:**
- Modify: `.codex/project.toml`
- Modify: `.codex/governance.toml`
- Modify: `.codex/routing.toml`
- Create: `.codex/workflows/network-routing-debug-checklist.md`
- Modify: `.codex/context/project-map.md`
- Modify: `.codex/context/codex-config-map.md`
- Modify: `.codex/workflows/config-validation-and-efficiency.md`

- [ ] **Step 1: 将新增 context/workflow 文件纳入 source_of_truth 与 first-read**
- [ ] **Step 2: 在 governance 中加入任务域切换、运行态证据优先级、双视角验证、网络排障禁忌**
- [ ] **Step 3: 在 routing/project-map 中明确 OpenWrt/部署/网络审计类任务的上下文入口和验证要求**
- [ ] **Step 4: 在 codex-config-map / config-validation docs 中加入新文件职责与必跑验证**

### Task 3: 完善 docs 开发规范与实战手册

**Files:**
- Modify: `docs/development.md`
- Modify: `docs/deployment.md`
- Modify: `docs/ai-routing-and-residential-egress.md`
- Create: `docs/passwall-openwrt-ai-routing-playbook.md`

- [ ] **Step 1: 在 development.md 加入运行态系统任务的证据优先级与回滚要求**
- [ ] **Step 2: 在 deployment.md 加入线上更新/配置变更的验证与回滚模板**
- [ ] **Step 3: 在 ai-routing-and-residential-egress.md 补齐主 WiFi / USA WiFi 分工、透明代理/SOCKS/DNS 差异**
- [ ] **Step 4: 新建 OpenWrt/Passwall/AI 路由实战手册，沉淀常见误判与排障路径**

### Task 4: 验证并汇总变更

**Files:**
- Read: `.codex/**`
- Read: `docs/**`

- [ ] **Step 1: 运行 `.codex` 配置验证脚本**

```powershell
python .codex/skills/superxray-project-context/scripts/validate_codex_config.py
python .codex/skills/superxray-project-context/tests/test_validate_codex_config.py
```

- [ ] **Step 2: 运行项目技能 quick validate 与 secret scan**

```powershell
python C:/Users/www/.codex/skills/.system/skill-creator/scripts/quick_validate.py .codex/skills/superxray-project-context
python C:/Users/www/.codex/skills/.system/skill-creator/scripts/quick_validate.py .codex/skills/superxray-ui-first-migration
python C:/Users/www/.codex/skills/.system/skill-creator/scripts/quick_validate.py .codex/skills/superxray-release-cicd
python scripts/secret_scan.py
```

- [ ] **Step 3: 输出最终变更报告，说明哪些经验已沉淀进 `.codex` 与 docs**
