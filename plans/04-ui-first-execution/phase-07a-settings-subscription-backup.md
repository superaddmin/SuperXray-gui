# UI First Phase 7a - Settings / Subscription / Backup 基础迁移交付记录

## 目标

在 Phase 6 已完成主力入站和客户端闭环后，将旧 UI 中高风险但必要的设置、订阅与备份恢复入口迁移到新 Vue UI。所有操作继续复用旧后端 API、旧字段和旧数据库，不引入多内核数据模型。

## 实施范围

- Settings 页面：
  - 新增 Panel、Security、Subscription、Formats、Telegram、LDAP、Backup 七个设置分区。
  - 通过旧 `getAllSettings` 读取 `webListen`、`webDomain`、`webPort`、`webBasePath`、证书、session、流量单位、过期时间、时间格式等面板设置。
  - 通过旧 `updateSettings` 保存原有字段集合，不改变字段名和含义。
- 安全设置：
  - 支持 Two Factor 相关字段编辑。
  - 支持通过旧 `updateUser` 更新管理员用户名和密码。
- 订阅设置：
  - 支持 URI、JSON、Clash 订阅开关和标题、监听、域名、路径、更新间隔、证书等字段。
  - 支持 Announce、Routing Rules 等多行文本配置。
  - 支持从旧 `getDefaultSettings` 填充默认订阅配置。
- 格式片段：
  - 支持 `subJsonFragment`、`subJsonNoises`、`subJsonMux`、`subJsonRules` 的纯文本编辑。
  - 对非空 JSON 片段提供本地格式告警，但不阻塞保存，避免旧数据存在历史格式差异时影响其他设置。
- 通知与 LDAP：
  - 支持 Telegram bot、chat id、thread id、运行状态通知、流量通知等字段。
  - 支持 LDAP 登录开关、地址、端口、base DN、用户映射、TLS、管理员过滤器等字段。
- 备份恢复：
  - 支持通过旧 `getDb` 下载数据库备份。
  - 支持通过旧 `importDB` 上传数据库文件；上传前二次确认。
  - 支持通过旧 `restartPanel` 重启面板。

## 安全与兼容约束

- 未改动 Go 后端模型、数据库结构、Xray 生命周期和旧订阅生成逻辑。
- 未引入 `proxy_inbounds`、`proxy_clients`、CoreManager、sing-box 或 Capability Schema。
- 所有日志、配置片段和订阅片段均使用纯文本表单或 textarea，不使用 `v-html`。
- 数据库导入继续走旧后端校验和重启流程，前端只增加确认和文件上传入口。
- 旧 `/panel/settings` 保持可用，作为完整回退入口。

## 验收重点

- 新 UI Settings 页面在无后端运行时配置的本地开发模式下可安全渲染默认结构。
- 嵌入式环境中可通过旧 API 读取并保存现有 settings 字段。
- 订阅 URI/JSON/Clash 相关字段保存后，旧 UI 和旧订阅输出仍可读取。
- 数据库备份下载文件可用；数据库导入只在隔离测试实例验证。
- 重启面板操作必须有显式确认。

## 验证方式

```powershell
cd frontend
npm run format
npm run lint
npm run typecheck
npm run build
cd ..
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
rg "v-html" frontend/src web/ui -n
rg "proxy_inbounds|proxy_clients|CoreManager|sing-box|Capability Schema" frontend/src web/ui -n
rg "unsafe-inline|unsafe-eval" frontend/src web/ui -n
```

浏览器本地验收：

- 访问 `http://127.0.0.1:5173/settings`。
- 验证 Settings 页面渲染正常。
- 验证七个设置分区可见：Panel、Security、Subscription、Formats、Telegram、LDAP、Backup。
- 验证控制台无 warning/error。

说明：真实保存、管理员账号修改、数据库下载和数据库导入必须在隔离测试实例中执行；不要在生产实例上做首次导入验收。

## 回滚方式

- 将 `frontend/src/views/SettingsView.vue` 回退为 Phase 6 之前的占位或只读状态。
- 移除新增的 settings/server API 封装：`updateUser`、`restartPanel`、`getDb`、`importDB`。
- 保留旧 `/panel/settings` 页面作为完整设置、订阅和备份恢复入口；本阶段没有数据库迁移，因此无需数据回滚。
