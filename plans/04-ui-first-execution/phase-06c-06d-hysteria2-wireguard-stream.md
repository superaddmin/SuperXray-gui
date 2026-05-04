# UI First Phase 6c/6d - Hysteria2/WireGuard 与 StreamSettings 通用表单交付记录

## 目标

在 Phase 6a/6b 已覆盖 VMess、VLESS、Trojan、Shadowsocks 的基础上，继续保持旧 `model.Inbound`、旧 `/panel/api/inbounds/*` API 和旧 `settings/streamSettings/sniffing` JSON 字段不变，补齐 Hysteria2、WireGuard 以及常用 StreamSettings 表单化能力。

## 实施范围

- Hysteria2：
  - 新 UI 新建协议新增 Hysteria2。
  - 为保持旧 UI 可读，新建时仍写入旧协议值 `hysteria`，并通过 `settings.version = 2` 表达 Hysteria2。
  - 客户端主键使用旧后端约定的 `auth`。
  - 客户端新增、编辑、启禁用、删除继续走旧客户端 API。
  - 分享链接新增基础 `hysteria2://` 文本导出。
- WireGuard：
  - 新 UI 新建协议新增 WireGuard。
  - 设置写入旧 UI/订阅代码可读的 `settings.secretKey`、`settings.pubKey`、`settings.peers`、`mtu`、`noKernelTun`。
  - WireGuard peer 不走旧客户端 API，而是通过旧 `/panel/api/inbounds/update/:id` 保存整段 `settings.peers`。
  - peer 主键使用 `publicKey`，支持新增、编辑、启禁用、删除和基础 `wireguard://` 分享链接。
  - 前端复用旧 UI 的 X25519/WireGuard keypair 生成算法，确保密钥格式兼容。
- StreamSettings 通用表单：
  - 新增 TCP、WebSocket、gRPC、HTTPUpgrade 常用字段。
  - 新增 TLS 常用字段：SNI、ALPN、fingerprint、证书文件路径、密钥文件路径。
  - 新增 Reality 常用字段：target、serverNames、privateKey、shortIds。
  - Hysteria2 固定使用 `network: hysteria` 与 `security: tls`。
  - 高级字段仍保留 JSON 编辑器兜底，表单提供 Sync JSON 与 Apply 双向辅助。

## 安全与兼容约束

- 未改动 Go 后端模型、数据库结构和 Xray 生命周期入口。
- 未引入 `proxy_inbounds`、`proxy_clients`、CoreManager、sing-box 或 Capability Schema。
- Hysteria2 不写新表，不引入新协议模型；仍用旧 UI 兼容的 `hysteria + version: 2`。
- WireGuard peer 仍保存在旧 `settings.peers`，不伪装成 `settings.clients`。
- Hysteria2 保存前要求 TLS 证书文件路径或内联证书内容完整，避免后端校验失败。
- 所有配置、凭据和分享文本均按文本渲染，不使用 `v-html`。

## 验收重点

- Hysteria2：
  - 可以创建 `settings.version = 2` 的 Hysteria2 入站。
  - 可以添加、编辑、启禁用、删除 `auth` 客户端。
  - TLS 证书缺失时新 UI 会阻止保存并给出明确提示。
  - 可以生成基础 `hysteria2://` 分享链接。
- WireGuard：
  - 可以创建带 server keypair 和默认 peer 的 WireGuard 入站。
  - 可以生成 server keypair、peer keypair 和 preshared key。
  - 可以通过整段 inbound update 保存 peer 新增、编辑、启禁用、删除。
  - 可以生成基础 `wireguard://` 分享链接。
- StreamSettings：
  - TCP/WS/gRPC/HTTPUpgrade 表单写入旧 `streamSettings` JSON。
  - TLS/Reality 表单写入旧 UI 可读字段。
  - JSON 编辑器仍可作为高级配置回退入口。

## 验证方式

```powershell
cd frontend
npm run typecheck
npm run lint
npm run format
npm run build
cd ..
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
rg "v-html" frontend/src web/ui -n
rg "proxy_inbounds|proxy_clients|CoreManager|sing-box|Capability Schema" frontend/src web/ui -n
rg "unsafe-inline|unsafe-eval" frontend/src web/ui -n
npm run e2e -- --list
npm run e2e
```

说明：真实写入 E2E 仍需要有效 `SUPERXRAY_E2E_*` 环境；占位 `.env` 会跳过浏览器用例。

## 回滚方式

- 从 `frontend/src/views/InboundsView.vue` 移除 Hysteria2/WireGuard 协议选项、客户端表单分支、WireGuard peer 整段保存逻辑和 StreamSettings 表单。
- 从 `frontend/src/utils/inboundCompat.ts` 移除 Hysteria2/WireGuard 默认设置、分享链接和 WireGuard keypair 工具。
- 恢复 `frontend/src/types/inbound.ts` 中新增的 Hysteria2/WireGuard 字段。
- 保留旧 `/panel/inbounds` 页面作为完整入站管理回退入口；本阶段没有数据库迁移，因此无需数据回滚。
