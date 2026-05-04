# UI First Phase 5 - Xray 生命周期与配置管理交付记录

## 目标

让新 UI 能通过现有旧后端接口完成 Xray 生命周期控制、版本管理和配置模板编辑保存，同时保持旧 `/panel/xray` 页面可继续读取同一份配置。

## 实施范围

- Xray 生命周期：
  - 展示现有 status API 中的 Xray 状态、版本和错误信息。
  - 通过旧 `restartXrayService` 路径执行 Start / Restart。
  - 通过旧 `stopXrayService` 路径执行 Stop。
  - 危险操作使用二次确认。
- Xray 版本管理：
  - 通过旧 `getXrayVersion` 读取可安装版本。
  - 通过旧 `installXray/:version` 安装所选版本。
  - 安装前二次确认，明确旧后端会停止、替换并重启 Xray。
- Xray 配置模板：
  - 通过旧 `/panel/xray/` 加载 `xraySetting` 和 `outboundTestUrl`。
  - 使用 JSON 文本编辑器编辑完整模板，覆盖 Outbounds / Routing / DNS / Reverse / Balancers / FakeDNS 等高级配置。
  - 支持 JSON 格式化、复制、下载。
  - 保存前校验 JSON，保存走旧 `/panel/xray/update`，字段仍为 `xraySetting` 和 `outboundTestUrl`。
  - 保存后提示是否重启 Xray。

## 安全与兼容约束

- 未引入 CoreManager、sing-box、Capability Schema 或新后端配置模型。
- 未迁移 `model.Inbound`。
- 配置内容使用纯文本 textarea 渲染，不使用 `v-html`。
- 版本安装、停止、重启和保存后重启均有确认。
- 保存的数据格式仍为旧 UI 使用的 Xray JSON 模板字符串。

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

- 恢复 `frontend/src/views/XrayView.vue` 为 Phase 4 的只读配置预览。
- 保留旧 `/panel/xray` 页面作为生命周期和配置管理入口。
- 不需要迁移或回滚数据库结构，因为本阶段没有新增后端模型。
