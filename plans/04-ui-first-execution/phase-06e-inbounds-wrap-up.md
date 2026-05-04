# UI First Phase 6e - Inbounds 收尾交付记录

## 目标

在 Phase 6a-6d 已覆盖六类主力协议后，完成入站管理的收尾体验：客户端批量选择、批量操作、全量分享导出，以及更完整的显式 mutation E2E 覆盖。所有能力继续保持旧 Xray API 与旧 JSON 数据结构兼容。

## 实施范围

- 批量选择：
  - 新 UI 客户端表格新增 row selection。
  - 选择状态会随当前入站刷新自动清理，避免操作过期客户端。
- 批量操作：
  - 支持 Reset Selected，逐个调用旧 `resetClientTraffic`。
  - 支持 Delete Selected，VMess/VLESS/Trojan/Shadowsocks/Hysteria2 逐个调用旧 `delClient`。
  - WireGuard 批量删除通过旧 inbound update 保存 `settings.peers`，不走 client API。
  - 批量删除禁止删除全部客户端/peer，避免旧后端 `no client remained` 或 WireGuard 空 peer。
- 分享导出：
  - 支持 Export Links，将当前入站所有可生成的分享链接复制并放入只读 textarea。
  - 不可生成分享链接的客户端会被跳过，仍保持纯文本渲染。
- E2E 覆盖：
  - 保留原 VLESS 写入基线。
  - 新增 Trojan、Shadowsocks、Hysteria2、WireGuard 的显式 mutation 写入基线。
  - 所有写入基线都必须设置 `SUPERXRAY_E2E_MUTATION=1` 才会执行。

## 安全与兼容约束

- 未改动 Go 后端模型、数据库结构和 Xray 生命周期入口。
- 未引入 `proxy_inbounds`、`proxy_clients`、CoreManager、sing-box 或 Capability Schema。
- WireGuard peer 继续写入旧 `settings.peers`。
- 批量操作不绕过旧 API 校验。
- 分享导出继续使用纯文本 textarea，不使用 `v-html`。

## 验收重点

- 选中客户端后可以批量重置非 WireGuard 客户端流量。
- 选中客户端后可以批量删除，但不能删除到空入站。
- WireGuard peer 批量删除后仍至少保留一个 peer。
- Export Links 能复制当前入站所有可用分享链接。
- mutation E2E 能在真实测试环境覆盖 VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard 创建与清理。

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

说明：默认 `.env` 占位时浏览器用例会跳过；真实写入验收需配置有效 `SUPERXRAY_E2E_*` 并显式设置 `SUPERXRAY_E2E_MUTATION=1`。

## 回滚方式

- 从 `frontend/src/views/InboundsView.vue` 移除 row selection、Reset Selected、Delete Selected、Export Links 和批量操作函数。
- 从 `tests/e2e/legacy-panel.spec.ts` 移除 Phase 6 多协议 mutation 写入基线。
- 保留旧 `/panel/inbounds` 页面作为完整入站管理回退入口；本阶段没有数据库迁移，因此无需数据回滚。
