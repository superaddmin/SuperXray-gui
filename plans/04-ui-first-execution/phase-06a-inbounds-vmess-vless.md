# UI First Phase 6a - Inbounds VMess/VLESS 管理交付记录

## 目标

在不迁移后端 `model.Inbound`、不引入 CoreManager、不接入新内核的前提下，让新 Vue UI 先覆盖现有 Xray 入站管理的核心闭环：入站列表、详情、VMess/VLESS 入站新增与编辑、客户端新增与维护、基础分享链接生成。

## 实施范围

- 入站列表：
  - 读取旧 `/panel/api/inbounds/list`。
  - 展示协议、监听地址、端口、传输层、TLS/Reality 状态、启禁用状态、流量和客户端数量。
  - 支持协议、启禁用状态和关键词筛选。
- 入站操作：
  - 新增 VMess/VLESS 入站，写入旧 `settings`、`streamSettings`、`sniffing` 字符串字段。
  - 编辑入站基础字段和三段旧 JSON 字段。
  - 启禁用入站走旧 `/panel/api/inbounds/update/:id`。
  - 删除入站走旧 `/panel/api/inbounds/del/:id`。
- 客户端操作：
  - 从旧 `settings.clients` 解析 VMess/VLESS 客户端。
  - 新增客户端走旧 `/panel/api/inbounds/addClient`。
  - 编辑、启禁用客户端走旧 `/panel/api/inbounds/updateClient/:clientId`。
  - 删除客户端走旧 `/panel/api/inbounds/:id/delClient/:clientId`。
  - 单客户端流量重置走旧 `/panel/api/inbounds/:id/resetClientTraffic/:email`。
  - 当前入站全部客户端流量重置走旧 `/panel/api/inbounds/resetAllClientTraffics/:id`。
- 分享文本：
  - 对 VMess/VLESS 客户端生成基础分享链接。
  - 链接使用纯文本 textarea 展示和复制，不使用 HTML 注入。

## 安全与兼容约束

- 未改动 Go 后端模型、数据库结构和 Xray 运行时入口。
- 未引入 `proxy_inbounds`、`proxy_clients`、CoreManager、sing-box 或 Capability Schema。
- 新 UI 写入的入站仍由旧 UI 可识别的 `settings/streamSettings/sniffing` 字符串组成。
- 日志、JSON 和分享链接均以文本方式渲染，不使用 `v-html`。
- VMess/VLESS 客户端主键继续使用旧后端约定的 `id` 字段。
- 当前阶段不覆盖 Trojan、Shadowsocks、WireGuard、Hysteria、HTTP/Mixed/Tunnel/TUN 的专用表单；这些协议仍可通过旧 UI 或后续 Phase 6b 扩展。

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
rg "v-html" frontend/src web/ui -n
rg "proxy_inbounds|proxy_clients|CoreManager|sing-box|Capability Schema" frontend/src -n
npm run e2e -- --list
npm run e2e
```

## 回滚方式

- 恢复 `frontend/src/views/InboundsView.vue` 为 Phase 5 占位页。
- 移除 `frontend/src/utils/inboundCompat.ts` 中的 Phase 6a 兼容工具。
- 回退 `frontend/src/api/endpoints.ts`、`frontend/src/api/inbounds.ts`、`frontend/src/types/inbound.ts` 中新增的客户端操作接口。
- 保留旧 `/panel/inbounds` 页面作为完整入站管理入口；本阶段没有数据库迁移，因此无需数据回滚。
