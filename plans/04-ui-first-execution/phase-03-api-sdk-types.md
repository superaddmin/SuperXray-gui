# UI First Phase 3 - API SDK 与类型层交付记录

## 目标

让新 Vue 3 UI 通过集中 SDK 消费现有 Xray 后端 API，避免组件中散落硬编码 URL，并在进入只读 Dashboard、日志和配置预览前建立统一错误处理、登录过期跳转和 TypeScript 响应类型。

## 实施范围

- 新增 `frontend/src/api/endpoints.ts`，集中维护旧 API 路径。
- 新增 `frontend/src/api/request.ts`，统一处理：
  - `basePath` / `apiBasePath`
  - `X-CSRF-Token`
  - `X-Requested-With` AJAX 识别
  - 后端 `success/msg/obj` 响应解包
  - 后端错误提示
  - 401 与旧 API 404 登录过期跳转
  - 表单提交、JSON 提交、文件上传、文件下载
- 新增 SDK 模块：
  - `serverApi`
  - `inboundApi`
  - `xrayApi`
  - `settingsApi`
  - `subscriptionApi`
- 新增旧响应类型：
  - `types/api.ts`
  - `types/server.ts`
  - `types/inbound.ts`
  - `types/xray.ts`
  - `types/settings.ts`
  - `types/subscription.ts`
- 状态栏通过 `serverApi.getServerStatus()` 做最小只读调用，验证新 UI 可消费旧 status API。

## 边界约束

- 未引入 `CoreInstance`、`ProxyInbound`、`Capability Schema`、`NeutralConfig`。
- 未修改旧后端 API 语义。
- 未迁移 `model.Inbound`。
- 未新增任何新 UI 写入流程；SDK 中的写方法仅为后续阶段封装旧兼容端点，当前页面不调用。
- 组件中不直接拼接旧 API URL；旧 API 字符串只允许出现在 `frontend/src/api/endpoints.ts` 和 request 层的鉴权判断中。

## 验收方式

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
```

## 回滚方式

- 删除 `frontend/src/api/*` 新增 SDK 文件。
- 删除 `frontend/src/types/{api,server,inbound,xray,settings,subscription}.ts`。
- 将 `AppStatusBar.vue` 恢复为静态状态标签。
- 将 `DashboardView.vue` 恢复为 Phase 2 的静态占位内容。
