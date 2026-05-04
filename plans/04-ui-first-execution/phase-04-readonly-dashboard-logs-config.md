# UI First Phase 4 - 只读 Dashboard / 日志 / 配置预览交付记录

## 目标

在不改变旧后端 API、不新增写路径的前提下，让新 UI 承载第一批只读能力：Dashboard 状态、日志中心和 Xray 配置 JSON 预览。

## 实施范围

- Dashboard：
  - 通过 `serverApi.getServerStatus()` 展示 Xray 状态、版本、CPU、内存、流量、连接、运行时信息。
  - 通过 `inboundApi.listInbounds()` 统计入站数量和客户端数量。
  - 接入现有 WebSocket `status`、`xray_state`、`invalidate` 消息作为只读刷新信号。
- Logs：
  - 通过旧 API 读取面板日志和 Xray 访问日志。
  - 支持行数选择、级别筛选、关键字筛选、Direct/Blocked/Proxy 过滤。
  - 使用固定行高虚拟滚动组件渲染日志，避免大量 DOM 节点。
  - 支持自动跟随 / 手动锁定、复制和本地下载。
- Xray Config：
  - 通过旧 API 读取当前 Xray config JSON。
  - 使用只读 `<pre>` 文本预览。
  - 支持复制和本地下载。

## 安全约束

- 未使用 `v-html`。
- 日志行使用 Vue 文本插值渲染。
- 配置预览使用纯文本节点渲染，不执行 JSON 内容。
- 不新增 WebSocket 消息类型。
- 不触发 Xray 启停、重启、更新或配置保存。
- 不迁移 `model.Inbound`，不引入 CoreManager / sing-box / Capability Schema。

## 验证方式

```powershell
cd frontend
npm run typecheck
npm run lint
npm run build
cd ..
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
npm run e2e -- --list
npm run e2e
```

## 回滚方式

- 恢复 `DashboardView.vue`、`LogsView.vue`、`XrayView.vue` 为 Phase 3 占位页。
- 删除 `VirtualLogViewer.vue`、`frontend/src/api/websocket.ts` 与文本导出工具。
- 继续使用旧 `/panel/` 页面承载 Dashboard、日志和配置查看。
