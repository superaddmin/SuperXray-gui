# UI First Phase 6b - Inbounds Trojan/Shadowsocks 管理交付记录

## 目标

在 Phase 6a 的 VMess/VLESS 入站管理基础上，继续沿用旧 `model.Inbound`、旧 `/panel/api/inbounds/*` API 和旧 `settings/streamSettings/sniffing` JSON 字段，补齐 Trojan 与 Shadowsocks 的新 UI 入站和客户端操作闭环。

## 实施范围

- 入站协议：
  - 新建入站协议选项新增 Trojan、Shadowsocks。
  - Trojan 默认写入旧 UI 可读的 `{ clients: [], fallbacks: [] }`。
  - Shadowsocks 默认写入旧 UI 可读的 legacy 方法 `chacha20-ietf-poly1305`、`network: tcp,udp`、`clients: []`、`ivCheck: false`。
- 客户端主键兼容：
  - VMess/VLESS 继续使用 `id`。
  - Trojan 使用旧后端约定的 `password`。
  - Shadowsocks 使用旧后端约定的 `email`。
- 客户端表单：
  - VMess/VLESS 保留 UUID。
  - Trojan/Shadowsocks 切换为 Password。
  - Shadowsocks Method 从入站 `settings.method` 派生，避免写出与旧后端校验冲突的客户端方法。
  - Shadowsocks 2022 AES 方法生成标准 Base64 密钥；legacy 方法生成 URL-safe 随机密码。
  - Shadowsocks `2022-blake3-chacha20-poly1305` 作为单用户方法，不开放客户端新增。
- 分享文本：
  - 新增 Trojan 基础分享链接。
  - 新增 Shadowsocks SIP002 基础分享链接。
  - 分享链接继续通过只读 textarea 展示和复制，不使用 HTML 注入。

## 安全与兼容约束

- 未改动 Go 后端模型、数据库结构和 Xray 生命周期入口。
- 未引入 `proxy_inbounds`、`proxy_clients`、CoreManager、sing-box 或 Capability Schema。
- 新 UI 写入的 Trojan/Shadowsocks 客户端仍由旧 UI 和旧 API 可解析。
- Shadowsocks 2022 继续遵守后端校验：客户端不得带 `method`，2022 AES 密钥必须是标准 Base64 且长度正确。
- Shadowsocks legacy 客户端继续带与入站一致的 `method`，避免保存时被后端拒绝。
- 所有凭据、JSON 和分享文本均按文本渲染，不使用 `v-html`。

## 验收重点

- Trojan：
  - 可以创建入站。
  - 可以添加、编辑、启禁用、删除客户端。
  - 客户端更新和删除使用 `password` 作为旧 API `clientId`。
  - 可以生成基础 `trojan://` 分享链接。
- Shadowsocks：
  - 可以创建 legacy multi-user 入站。
  - 可以添加、编辑、启禁用、删除客户端。
  - 客户端更新和删除使用 `email` 作为旧 API `clientId`。
  - legacy 客户端保存时带入站同名 `method`。
  - 2022 AES 客户端密码生成标准 Base64 密钥，并在保存时不写客户端 `method`。
  - 单用户 2022 chacha 方法不允许新增客户端。
  - 可以生成基础 `ss://` 分享链接。

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
npm run e2e -- --list
npm run e2e
```

说明：真实写入 E2E 仍需要有效 `SUPERXRAY_E2E_*` 环境；占位 `.env` 会跳过浏览器用例。

## 回滚方式

- 从 `frontend/src/views/InboundsView.vue` 移除 Trojan/Shadowsocks 协议选项和客户端表单分支。
- 从 `frontend/src/utils/inboundCompat.ts` 移除 Trojan/Shadowsocks 默认设置、客户端主键分流、Shadowsocks 方法/密钥工具和分享链接生成。
- 恢复 `frontend/src/types/inbound.ts` 中 `XrayEditableInboundProtocol` 为 VMess/VLESS。
- 保留旧 `/panel/inbounds` 页面作为完整入站管理回退入口；本阶段没有数据库迁移，因此无需数据回滚。
