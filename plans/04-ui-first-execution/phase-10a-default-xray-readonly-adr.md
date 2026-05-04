# UI First Phase 10.1 - default-xray 只读实例 ADR

## 状态

已接受为 Phase 10.1 前置设计。

2026-05-04 风险接受/强制进入更新：原 ADR 仅允许 `default-xray` 只读实例；因上线部署需要，已在 [`phase-10-entry-gate-assessment.md`](phase-10-entry-gate-assessment.md) 中记录风险接受，允许最小 active CoreManager/sing-box 后端入口先行实施。该变更只放宽 `CoreManager` 和 `experimental-sing-box` API 层，不放宽旧 `model.Inbound` 迁移、`proxy_inbounds/proxy_clients` 写入、Capability Schema 写入表单或 Xray 旧生命周期接管。

## 背景

Phase 0-9 的主线目标是让新 Vue UI 稳定承载现有 Xray 工作流，并保留旧 UI 回退。当前多内核后端方案已经设计 CoreManager、CoreInstance、Capability Schema 和未来多内核数据模型，但这些能力一旦提前进入主链路，会叠加 UI 迁移、安全收口和生命周期改造风险。

Phase 10.1 的目标不是接入新内核，也不是改变 Xray 启停方式，而是让系统开始拥有一个可观察、只读、可回退的默认实例概念：

```text
default-xray
  -> 现有 XrayService
  -> 旧 model.Inbound
  -> 旧订阅输出
  -> 旧 API 行为
```

## 决策

Phase 10.1 首轮的 `default-xray` 仍采用“虚拟只读实例”，不创建数据库表，不新增写入模型，不让 CoreManager 接管 Xray 生命周期。

`default-xray` 的语义如下：

| 字段              | 决策值或来源                         | 说明                         |
| ----------------- | ------------------------------------ | ---------------------------- |
| `id`              | `default-xray`                       | 稳定字符串标识，不使用 DB id |
| `name`            | `default-xray`                       | 面向 API 和内部引用          |
| `displayName`     | `Default Xray`                       | 面向 UI 展示                 |
| `coreType`        | `xray`                               | 当前唯一真实核心             |
| `source`          | `legacy-inbound-table`               | 数据仍来自旧 `model.Inbound` |
| `mode`            | `legacy`                             | 表示未迁移到多内核模型       |
| `lifecycleOwner`  | `legacy-xray-service`                | 启停重启仍走旧服务           |
| `status`          | `/panel/api/server/status` 的 `xray` | 不新增状态采样器             |
| `subscription`    | 旧订阅服务                           | 输出语义保持不变             |
| `writeSupported`  | `false`                              | Phase 10.1 不提供写入        |
| `managerAttached` | `false`                              | Phase 10.2 前不接管生命周期  |

## 原常规门禁下允许的最小 API 形态

Phase 10.1 代码实施时，可以新增只读接口，但必须满足以下限制：

```text
GET /panel/api/cores/instances
GET /panel/api/cores/instances/default-xray
```

限制：

- 仅返回 `default-xray` 虚拟实例和现有 Xray 状态摘要。
- 继续使用 `/panel/api` 的登录鉴权。
- 不新增 `POST`、`PUT`、`PATCH`、`DELETE`。
- 不读取或写入 `core_instances`、`proxy_inbounds`、`proxy_clients`。
- 不改变 `/panel/api/server/restartXrayService`、`stopXrayService`、`xraylogs` 等旧端点。

建议响应契约：

```json
{
  "id": "default-xray",
  "name": "default-xray",
  "displayName": "Default Xray",
  "coreType": "xray",
  "source": "legacy-inbound-table",
  "mode": "legacy",
  "lifecycleOwner": "legacy-xray-service",
  "status": {
    "state": "running",
    "version": "v25.x",
    "errorMsg": ""
  },
  "capabilities": {
    "read": true,
    "write": false,
    "lifecycleViaCoreManager": false
  }
}
```

## UI 展示约束

Phase 10.1 只允许展示 `default-xray` 的只读状态，不允许新增多内核创建向导或 sing-box 主导航。

允许：

- 在状态栏、Dashboard 或未来 Core Overview 中显示 `default-xray`。
- 标明 `legacy`、`xray`、`read-only`。
- 展示现有 Xray 状态、版本和错误摘要。

禁止：

- 新增 sing-box、Hysteria2、mihomo 入口。
- 新增实例启动、停止、重启按钮并指向 CoreManager。
- 新增 Capability Schema 动态表单。
- 新增多内核入站写入页。

## 强制进入后的后端实现边界

原常规门禁下，首轮实现应优先放在既有 Web API 层，避免引入完整 CoreManager 目录结构。风险接受后，本阶段允许新增最小 CoreManager 目录结构，但必须保持以下边界：

允许新增：

- `core/`：内存 CoreManager、实例契约和最小适配器接口。
- `core/singbox/`：外部 sing-box binary 的实验生命周期适配器。
- `web/controller/core.go`：`/panel/api/cores/*` controller。
- `web/service/core_service.go`：`default-xray` 与 `experimental-sing-box` 聚合服务。
- `web/controller/core_test.go` 或同等测试：鉴权和响应契约。

暂不新增：

- `database/model/core_instance.go`。
- `proxy_inbounds`、`proxy_clients` 数据表。
- sing-box 生产配置生成器。
- Capability Schema 写入表单。
- 旧 Xray 生命周期接管。

## 验收标准

进入 Phase 10.1 代码实施前：

- Phase 9 安全收口通过。
- 真实 Xray core 隔离实例可运行。
- `SUPERXRAY_E2E_RESTART=1` 非跳过通过。
- `SUPERXRAY_E2E_MUTATION=1` 非跳过通过。
- `SUPERXRAY_E2E_IMPORT_DB=1` 非跳过通过。
- `SUPERXRAY_E2E_SUB_URL` 非跳过通过或有明确替代验收记录。
- 旧 UI 可读取新 UI 写入的主力协议入站、客户端和设置。

原常规门禁下的 Phase 10.1 代码实施后：

- `GET /panel/api/cores/instances` 未登录返回 404。
- 登录后只返回一个 `default-xray`。
- 返回状态与 `/panel/api/server/status` 的 Xray 摘要一致。
- 旧 `/panel/api/server/*` 生命周期端点行为不变。
- 禁止项扫描无 active 命中：

```powershell
rg 'proxy_inbounds|proxy_clients|CoreManager|sing-box|Capability Schema' frontend\src web\ui web\controller web\middleware web\service database\model -n
```

风险接受/强制进入后，扫描口径调整为：

- 继续禁止 `proxy_inbounds`、`proxy_clients` active 命中。
- `CoreManager` 和 `sing-box` 只允许出现在 `core/`、`web/service/core_service.go`、`web/controller/core.go`、`web/controller/api.go` 和 Phase 10 文档。
- `Capability Schema` 仍不得进入 active 写入表单。

## 被拒绝方案

| 方案                              | 拒绝原因                                      |
| --------------------------------- | --------------------------------------------- |
| 立即创建 `core_instances` 表      | 会提前打开数据迁移面，不利于回滚              |
| 立即让 CoreManager 接管 Xray 启停 | 生命周期风险高，必须等 10.2 且旧 API 对照完备 |
| 直接新增 sing-box UI              | 违反 Phase 10.3 顺序，且会影响 Xray 稳定验收  |
| 用 Capability Schema 改写表单     | 过早抽象，会干扰旧 `model.Inbound` 兼容       |

## 回滚方式

- ADR 本身不影响运行时。
- 如后续只读 API 实施后出现问题，可移除 `/panel/api/cores/*` 路由，新旧 Xray API 不受影响。
- 不涉及数据库迁移，因此不需要数据库回滚。
