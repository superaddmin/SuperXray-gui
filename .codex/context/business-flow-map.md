# SuperXray-gui 业务流程地图

用于把任务映射到业务链路、源码路径、主责代理和验证命令；涉及写入、重启、导入、下载、外部请求或二进制执行时追加安全门禁。

## 关键链路

- 登录与 Session：`web/controller/index.go`、`web/session/session.go`、`web/middleware/security.go`、`web/ui.go`、`frontend/src/views/LoginView.vue`。
- 新 UI 与旧 API 兼容：`web/ui.go`、`web/web.go`、`web/controller/xui.go`、`frontend/src/router/index.ts`。旧 HTML UI 的 `web/html/**`、`web/assets/**` 已退役。
- Xray 生命周期：`frontend/src/views/XrayView.vue`、`web/controller/xray_setting.go`、`web/service/xray.go`、`xray/**`。
- Inbounds 与 Clients：`frontend/src/api/inbounds.ts`、`web/controller/inbound.go`、`web/service/inbound.go`、`database/model/model.go`、`sub/**`。
- 订阅：`sub/sub.go`、`sub/subController.go`、`sub/subService.go`、`sub/subJsonService.go`、`sub/subClashService.go`。
- Core：`core/types.go`、`core/manager.go`、`web/service/core_service.go`、`web/controller/core.go`、`frontend/src/types/core.ts`。

## 硬约束

Phase 10.2 前不得通过 CoreManager 接管 legacy Xray 生命周期；活跃写模型仍是 `database/model.Inbound`；不得重新引入 `/panel/legacy/`、`web/html`、`web/assets`。