# SuperXray-gui UI 先行、Xray 稳定迁移、再接多内核施工规划

> 目标：先完成新 UI 工程化和现有 Xray 功能等价迁移，确保 Xray 在新 UI 框架中稳定运行；再逐步引入 CoreManager、多内核实例、sing-box、Hysteria2 和 mihomo 订阅导出能力。

---

## 1. 方案定位

本方案是 [`../03-ui-design/multi-core-ui-design-plan.md`](../03-ui-design/multi-core-ui-design-plan.md) 与 [`../02-architecture/backend-multi-core-architecture-plan.md`](../02-architecture/backend-multi-core-architecture-plan.md) 的施工顺序修正版。

核心策略：

```text
先升级 UI 框架
  ↓
让现有 Xray 在新 UI 中稳定运行
  ↓
新旧 UI 并行灰度和可回退
  ↓
再启动多内核后端抽象
  ↓
最后接入 sing-box / Hysteria2 / mihomo
```

这样做的原因：

1. 当前后端、数据库、订阅、定时任务和 Xray 进程管理都强绑定单 Xray 实例。
2. 如果同时做 Vue3 迁移、CoreManager、数据模型迁移和 sing-box 接入，风险会叠加。
3. 先让新 UI 完整承载现有 Xray，可以把最大用户可见风险提前暴露并独立解决。
4. 新 UI 工程化完成后，后续多内核页面、Capability Schema、动态表单会更容易落地。

---

## 2. 关键前置假设（审查校正）

> ⚠️ **重要校正**：本方案初版假设前端已是 Vue 3 + Vite + TypeScript 工程化方案。经实际代码审计发现完全不是——以下是必须修正的前提认知。

### 当前前端真实状态

| 假设项            | 实际状态                                                        | 影响                                        |
| ----------------- | --------------------------------------------------------------- | ------------------------------------------- |
| Vue 3             | **Vue 2**（`new Vue()` 构造器、in-DOM 模板、`Vue.component()`） | 阶段 1 不是"迁移"而是**从零搭建**           |
| Vite + TypeScript | **无** `package.json`、无任何构建工具、纯 ES5 JS                | 前端工程需完整搭建 CI/CD 工具链             |
| Ant Design Vue 4  | Ant Design Vue UMD 加载（版本待确认）                           | 需验证 Vue 3 兼容版本升级路径               |
| 已有构建脚本      | 不存在                                                          | 新增 `npm run build` / `typecheck` / `lint` |
| TypeScript 类型   | 纯 JS，零 `.d.ts`                                               | API SDK 需全部手写类型定义                  |
| 可复用前端代码    | Vue 2 模板和 JS **完全无法复用到 Vue 3 SFC**                    | 所有页面和组件都是新写                      |

### 对后续阶段的影响

1. **阶段 1 从"迁移"变为"新建"**，工期需上修。
2. **阶段 0 的 parity checklist 不能直接从 Vue 2 源码映射**，需基于 UI 行为列出。
3. **旧 Vue 2 模板在前端构建中无法复用**，但后端 Go 模板（`web/html/`）中的 API 端点不变。
4. 好消息：**后端 API 和数据模型完全不受影响**，双 UI 并行策略依然成立。

---

## 3. 总体目标

第一大里程碑：

```text
新 UI 完整承载现有 Xray 使用流程，并保留旧 UI 回退。
```

第二大里程碑：

```text
默认 Xray 被包装为 default-xray 实例，但旧数据模型和旧 API 仍兼容。
```

第三大里程碑：

```text
新增 sing-box experimental 实例能力，先支持手写配置、校验、启动、停止、重启和日志。
```

非目标：

- UI 迁移阶段不迁移数据库。
- UI 迁移阶段不接入 sing-box。
- Xray 未在新 UI 稳定前，不做 CoreManager 深度改造。
- 不一次性删除旧 HTML 模板。
- 不让新 UI 写出旧 UI 无法识别的数据。

---

## 4. 成功标准

新 UI 阶段成功标准：

1. 新 UI 能完成旧 UI 的核心 Xray 工作流。
2. 旧 UI 可作为回退入口继续访问。
3. 新 UI 创建、修改的数据，旧 UI 仍能读取和编辑。
4. `go test ./...`、`go vet ./...`、`go build -o bin/SuperXray.exe ./main.go` 通过。
5. 前端 `build`、`typecheck`、`lint` 通过。
6. 关键 E2E 流程通过：登录、查看状态、添加入站、添加客户端、重启 Xray、查看日志、生成订阅。
7. 新 UI 不使用 `v-html` 渲染日志、配置、订阅内容。
8. 新 UI 不依赖 Vue runtime template 的 `unsafe-eval`。

多内核阶段成功标准：

1. 默认 Xray 实例不破坏旧功能。
2. CoreManager 能管理默认 Xray 生命周期。
3. 旧 Xray API 可内部转发到默认实例。
4. sing-box experimental 可独立启动、停止、校验配置和查看日志。
5. sing-box 接入不会影响现有 Xray 订阅和入站管理。

---

## 5. 施工总原则

1. 先 UI，后内核。
2. 先只读，后写入。
3. 先旧 API 兼容，后新 API 抽象。
4. 先 Xray 等价，后多内核增强。
5. 先保留旧 UI，后灰度切换。
6. 先手写配置接入新内核，后动态表单和 Neutral Model。
7. 每阶段都必须可构建、可测试、可回退。

---

## 6. 阶段 0：现状冻结与验收基线

周期：2 到 4 天

目标：

```text
冻结旧 UI 的 Xray 行为边界，形成后续迁移的验收清单。
```

施工内容：

1. 梳理旧 UI 页面：
   - Dashboard
   - Inbounds
   - Xray 配置
   - Settings
   - Logs
   - Subscription
   - Login / Logout

2. 梳理旧 UI 核心操作：
   - 登录
   - 查看 Xray 状态
   - 启动 Xray
   - 停止 Xray
   - 重启 Xray
   - 查看面板日志
   - 查看 Xray 日志
   - 查看 Xray 配置 JSON
   - 编辑 Xray 配置模板
   - 新增入站
   - 编辑入站
   - 删除入站
   - 新增客户端
   - 编辑客户端
   - 删除客户端
   - 重置流量
   - 获取二维码和分享链接
   - 生成订阅
   - 备份和导入数据库
   - 修改面板设置

3. 建立 `Xray UI parity checklist`：
   - 页面名称
   - 旧入口
   - 旧 API
   - 新 UI 对应页面
   - 是否只读
   - 是否写入
   - 测试用例
   - 回退方式

4. 建立最小 E2E 脚本：
   - 登录
   - 打开 Dashboard
   - 打开 Inbounds
   - 新增入站
   - 新增客户端
   - 重启 Xray
   - 打开日志
   - 访问订阅

交付物：

- Xray 功能等价清单。
- UI 迁移验收清单。
- 最小 E2E 测试脚本。
- 当前旧 UI 行为基线记录。

验证命令：

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

阶段验收：

- 旧 UI 核心流程全部可复现。
- 每个待迁移功能都有验收项。
- 后续任一阶段失败时，可以明确回退到旧 UI。

回滚方式：

- 本阶段不改核心代码。
- 如补 E2E 失败，不阻塞现有运行，只记录缺口。

---

## 7. 阶段 1：新前端工程骨架

周期：**6 到 10 天**（审查校正：原估 3-6 天，实际是从零搭建 Vue 3 工程，无既有前端工具链）

目标：

```text
从零搭建新 UI 工程，不迁移复杂业务，不改变后端业务逻辑。
```

> **说明**：当前前端是 Vue 2 + 无构建工具的纯 HTML/JS，**无任何前端代码可复用**。本阶段是在 `/frontend/` 下建立全新的 Vue 3 + Vite + TypeScript 工程，所有代码都是新写。

推荐技术栈：

```text
Vue 3.4+
Vite 5+
TypeScript 5+
Pinia
Vue Router 4
Ant Design Vue 4（确认与 Vue 3 兼容的版本）
Axios
CodeMirror 6
```

推荐目录：

```text
frontend/
  package.json
  vite.config.ts
  tsconfig.json
  tsconfig.node.json
  .eslintrc.cjs
  .prettierrc
  src/
    main.ts
    App.vue
    env.d.ts
    router/
    stores/
    api/
    layouts/
    views/
    components/
    types/
```

施工内容：

1. 新增 `frontend/` 工程——从 `npm create vite@latest` 开始。
2. 安装和配置依赖：Vue 3、Vue Router、Pinia、Axios、Ant Design Vue 4、CodeMirror 6、ESLint、Prettier、TypeScript。
3. 配置 Vite 相对路径构建（`base: ''`），兼容面板 `webBasePath`。
4. 建立基础路由：
   - `/`
   - `/dashboard`
   - `/logs`
   - `/xray`
   - `/inbounds`
   - `/settings`
5. 建立 `MainLayout`：
   - 左侧菜单
   - 顶部状态栏
   - 内容区
   - 移动端折叠菜单
6. 建立基础状态：
   - 用户登录状态
   - app config
   - theme
7. 建立空页面和路由守卫。
8. **新增路径感知 CSP 中间件（后端）**：
   - 当前 CSP 全局包含 `'unsafe-eval'`（旧 Vue 2 需要）
   - 新增中间件：根据请求路径 `/panel/ui/*` 返回更严格的 CSP（无需 `unsafe-eval`）
   - 旧 `/*` 路径保持现有 CSP 不变
   - 见阶段 9 的 CSP 目标，但基础设施在此先行落地

交付物：

- 可构建的新 UI 工程。
- 可编译通过的 TypeScript 配置。
- 空 Dashboard 页面。
- 基础布局和路由。
- 前端构建脚本（`npm run build`、`typecheck`、`lint`、`preview`）。
- 后端路径感知 CSP 中间件初版。

前端验证：

```powershell
cd frontend
npm install
npm run typecheck    # 新增
npm run lint         # 新增
npm run build
```

阶段验收：

- 新 UI 构建成功。
- `npm run typecheck` 通过。
- `npm run lint` 通过。
- 新 UI 不使用 `v-html`。
- 新 UI 不使用 Vue 运行时模板编译（无 `unsafe-eval` 依赖）。
- 新 UI 尚不影响旧 UI。
- CSP 中间件能区分新旧 UI 路径并返回不同策略。

回滚方式：

- 删除或停用新 UI 入口即可。
- 旧 UI 路由不变。

---

## 8. 阶段 2：Go 后端接入新 UI 静态资源

周期：2 到 4 天

目标：

```text
让 Go/Gin 可以稳定托管新 UI 构建产物，同时旧 UI 保持不变。
```

施工内容：

1. 新增新 UI 构建产物托管。
2. 建议初期入口：

```text
/panel/ui
```

3. 旧 UI 保持：

```text
/panel
```

4. 新 UI 注入运行时配置：

```js
window.__APP_CONFIG__ = {
  basePath: "...",
  csrfToken: "...",
  version: "...",
};
```

5. 新 UI 路由刷新支持。
6. 新 UI 静态资源缓存策略：
   - 带 hash 的构建资源可以长缓存。
   - 入口 HTML 不长缓存。

交付物：

- Go 端新 UI 静态资源接入。
- 新 UI 入口路由。
- runtime app config 注入。

验证命令：

```powershell
go test ./...
go build -o bin/SuperXray.exe ./main.go
```

阶段验收：

- `/panel` 旧 UI 可用。
- `/panel/ui` 新 UI 可用。
- 配置 `webBasePath` 时，新旧 UI 静态资源都正常加载。
- 新 UI 构建产物进入最终二进制或发布包。

回滚方式：

- 移除新 UI 路由或关闭新 UI 入口。
- 旧 UI 不受影响。

---

## 9. 阶段 3：API SDK 与类型层

周期：4 到 7 天

目标：

```text
新 UI 先稳定消费现有 API，组件中不散落硬编码 URL。
```

推荐结构：

```text
frontend/src/api/
  request.ts
  server.ts
  inbounds.ts
  xray.ts
  settings.ts
  subscription.ts

frontend/src/types/
  server.ts
  inbound.ts
  xray.ts
  settings.ts
  subscription.ts
```

施工内容：

1. 建立统一 `request.ts`：
   - basePath
   - CSRF token
   - session timeout
   - 401 跳转
   - 后端错误提示
   - 文件下载
   - FormData 上传

2. 封装现有 API：
   - `/panel/api/server/status`
   - `/panel/api/server/getXrayVersion`
   - `/panel/api/server/restartXrayService`
   - `/panel/api/server/stopXrayService`
   - `/panel/api/server/logs/:count`
   - `/panel/api/server/xraylogs/:count`
   - `/panel/api/server/getConfigJson`
   - `/panel/api/inbounds/list`
   - `/panel/api/inbounds/add`
   - `/panel/api/inbounds/update/:id`
   - `/panel/api/inbounds/del/:id`
   - `/panel/xray/`
   - `/panel/xray/update`
   - `/panel/setting/all`
   - `/panel/setting/update`

3. 按现有响应建立 TypeScript 类型。
4. 暂不引入：
   - `CoreInstance`
   - `ProxyInbound`
   - `Capability Schema`
   - `NeutralConfig`

交付物：

- API SDK。
- TypeScript 类型定义。
- 统一错误处理。

阶段验收：

- 新 UI 可调用现有 status API。
- 后端错误能统一展示。
- 登录过期能统一跳转。
- 组件内不直接拼接 API URL。

回滚方式：

- 本阶段不影响旧 UI。
- API SDK 失败只影响新 UI。

---

## 10. 阶段 4：只读 Dashboard 与日志中心

周期：5 到 8 天

目标：

```text
先迁移只读能力，验证新 UI 与现有后端的稳定性。
```

施工内容：

1. Dashboard 只读展示：
   - Xray 状态
   - Xray 版本
   - 面板状态
   - CPU
   - 内存
   - 今日流量
   - 入站数量
   - 用户数量

2. 日志中心：
   - 面板日志
   - Xray 日志
   - 行数选择
   - 关键字筛选
   - **虚拟滚动渲染**（高频 WebSocket 日志推送，避免 DOM 节点爆炸）
   - **自动滚动 / 锁定控制**（新日志到达时自动滚到底部，用户手动滚动后暂停自动滚动）
   - 复制
   - 下载
   - 日志纯文本渲染（禁止 `v-html`，使用 `textContent` 或 Vue 插值）

3. 配置查看：
   - 当前 Xray config JSON 只读预览
   - JSON 格式化
   - 复制
   - 下载

4. WebSocket：
   - 先接现有消息类型
   - 不新增多内核事件

安全要求：

- 日志使用纯文本渲染。
- 配置预览使用纯文本或 CodeMirror 只读。
- 禁止 `v-html`。

交付物：

- 新 Dashboard。
- 新 Log Center。
- 新 Config Preview。

阶段验收：

- 新旧 Dashboard 数据一致。
- 日志不会触发 DOM XSS。
- 配置预览不会执行脚本。
- 浏览器刷新、登录过期、后端错误均表现正常。

回滚方式：

- 新 UI 页面不可用时继续使用旧 UI。

---

## 11. 阶段 5：Xray 生命周期与配置管理迁移

周期：7 到 12 天

目标：

```text
新 UI 能完成现有 Xray 生命周期和配置管理闭环。
```

施工内容：

1. Xray 生命周期：
   - 启动
   - 停止
   - 重启
   - 状态刷新
   - 错误结果展示

2. Xray 版本管理：
   - 查看当前版本
   - 查看可安装版本
   - 安装指定版本
   - 安装前二次确认

3. Xray 配置模板：
   - 加载配置模板
   - 编辑 JSON
   - 格式化
   - 保存
   - 保存后提示是否重启

4. Xray 高级配置：
   - Outbounds
   - Routing
   - DNS
   - Reverse
   - Balancers
   - Fakedns

交付物：

- Xray 管理页面。
- 配置模板编辑器。
- 版本管理操作。

阶段验收：

- 新 UI 可完成 Xray 停止和重启。
- 配置保存后旧 UI 仍可读取。
- 配置错误能展示后端错误摘要。
- 危险操作有二次确认。
- 不引入新后端配置模型。

验证命令：

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

回滚方式：

- 继续使用旧 `/panel/xray` 页面。
- 新 UI 不改变底层数据格式。

---

## 12. 阶段 6：入站与客户端迁移

周期：**3 到 5 周**（审查校正：原估 2-4 周，协议表单复杂度被低估）

目标：

```text
完成 Xray 最核心业务页面迁移，并保持旧 UI 数据兼容。
```

施工顺序：

1. 入站列表：
   - 协议
   - 监听地址
   - 端口
   - 传输层
   - TLS
   - 启用状态
   - 流量
   - 客户端数量

2. 入站详情：
   - 基本信息
   - 流量
   - 客户端
   - 分享信息
   - 订阅信息

3. **入站新增和编辑（按协议拆分子阶段）**：

   > `Settings` 和 `StreamSettings` 字段是 JSON 字符串，不同协议的 `settings` 结构完全不同。
   > 现有表单（`web/html/modals/inbound_modal.html`）为每种协议有独立字段组。
   > 建议按协议优先级分批迁移：
   - **3a. VMess + VLESS**（最常用，优先级最高）
   - **3b. Trojan + Shadowsocks**
   - **3c. Hysteria2 + WireGuard**（协议差异大，表单结构不同）
   - **3d. StreamSettings 通用表单**（TLS、WebSocket、gRPC、HTTPupgrade 等传输层配置）

   继续使用现有 `model.Inbound`
   继续提交 `settings`
   继续提交 `streamSettings`
   不引入 `proxy_inbounds`

4. 客户端管理：
   - 新增客户端
   - 编辑客户端
   - 删除客户端
   - 启用和禁用
   - 到期时间
   - 流量限制
   - IP 限制
   - Sub ID

5. 批量操作：
   - 批量导入客户端
   - 复制客户端
   - 批量删除
   - 重置客户端流量

6. 分享和订阅：
   - 二维码
   - URI
   - Clash
   - JSON
   - WireGuard 配置
   - Hysteria2 链接

关键约束：

- 新 UI 必须继续使用现有 `/panel/api/inbounds/*`。
- 新 UI 不写出旧 UI 无法识别的数据。
- 旧 UI 必须能编辑新 UI 创建的入站和客户端。

交付物：

- 新 Inbounds 页面。
- 新 Inbound Form。
- 新 Client Form。
- 新 Share / QR / Subscription 视图。

阶段验收：

完整闭环必须通过：

```text
新增入站
  ↓
新增客户端
  ↓
重启 Xray
  ↓
复制分享链接
  ↓
访问订阅
  ↓
重置流量
  ↓
删除客户端
  ↓
删除入站
```

验证命令：

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
cd frontend
npm run build
```

回滚方式：

- 新 UI Inbounds 出现问题时，使用旧 `/panel/inbounds`。
- 不需要数据库回滚。

---

## 13. 阶段 7：设置、订阅、备份恢复迁移

周期：1 到 2 周

目标：

```text
补齐 Xray 用户日常使用闭环。
```

施工内容：

1. 面板设置：
   - 面板端口
   - base path
   - HTTPS
   - 语言
   - 主题

2. 安全设置：
   - 用户名
   - 密码
   - 2FA
   - 登录相关设置

3. 订阅设置：
   - sub path
   - sub json path
   - clash path
   - URI 设置
   - JSON 订阅增强
   - Clash 订阅增强

4. Telegram Bot：
   - token
   - chat ID
   - 通知设置
   - 运行状态

5. 备份恢复：
   - 下载数据库
   - 导入数据库
   - 导入前校验
   - 导入后重启提示

交付物：

- 新 Settings 页面。
- 新 Subscription Settings 页面。
- 新 Backup / Restore 页面。

阶段验收：

- 设置保存后旧 UI 可读取。
- 订阅输出与旧 UI 一致。
- 数据库导入有二次确认。
- Telegram Bot 重启路径不引发 409 冲突。

回滚方式：

- 继续使用旧 Settings 页面。
- 导入数据库失败必须不中断当前数据库。

---

## 14. 阶段 8：新 UI 灰度切换

周期：5 到 10 天

目标：

```text
让新 UI 成为默认入口，同时保留旧 UI 回退。
```

施工内容：

1. 新 UI 切到默认入口：

```text
/panel
```

2. 旧 UI 移到：

```text
/panel/legacy
```

3. 增加 UI 模式设置：

```text
new
legacy
```

4. 管理员可从新 UI 切回旧 UI。
5. 文档写明回退方式。
6. 旧 UI 至少保留 1 到 2 个版本周期。

阶段验收：

- 默认打开 `/panel` 进入新 UI。
- `/panel/legacy` 可回退。
- 新 UI 覆盖旧 UI 核心功能。
- E2E 全部通过。

回滚方式：

- 将默认入口切回旧 UI。
- 保留新 UI 作为 `/panel/ui`。

---

## 15. 阶段 9：安全收口

周期：3 到 7 天

目标：

```text
利用新 UI 工程化成果，收紧 CSP、CSRF 和日志渲染边界。
```

施工内容：

1. 新 UI 移除 inline script（已由 Vite 构建产物保证）。
2. 新 UI 不依赖 `unsafe-eval`（已由 Vue 3 SFC 保证）。
3. 新 UI 禁止 `v-html` 渲染外部内容。
4. 日志、配置、订阅预览均按文本渲染。
5. 状态变更 API 全部带 CSRF token。
6. 配置下载、日志下载、数据库下载必须登录。
7. 对上传和导入文件做大小限制和类型校验。

新 UI CSP 目标：

```text
script-src 'self' 'nonce-...'
style-src 'self' 'nonce-...'
style-src-attr 'none'
object-src 'none'
base-uri 'self'
frame-ancestors 'none'
```

阶段验收：

- 新 UI 不需要 `unsafe-eval`。
- 新 UI 不需要 `script-src 'unsafe-inline'`。
- 新 UI `style-src` 不需要 `unsafe-inline`，动态 `<style>` 通过 nonce bootstrap 验证。
- 日志 DOM XSS 回归测试通过。
- CSRF 回归测试通过。

回滚方式：

- CSP 中间件已分路径（阶段 1 已建设基础设施），收紧 CSP 只影响 `/panel/ui/*` 路径。
- 旧 UI `/*` 路径保持现有宽 CSP 不变，互不影响。

---

## 16. 阶段 10：多内核后端启动

周期：分多轮实施

前置条件：

1. 新 UI 已稳定承载 Xray。
2. 旧 UI 已降级为回退入口。
3. Xray E2E 全部通过。
4. 新旧 UI 无数据格式漂移。
5. 安全基线已完成。

阶段 10.1：默认 Xray 实例只读化

目标：

```text
先让系统认识 default-xray，但不改变旧 Xray 行为。
```

施工内容：

- 新增只读 `CoreInstance` 概念。
- 创建默认实例：

```text
name: default-xray
coreType: xray
source: legacy-inbound-table
mode: legacy
```

- 新 UI 显示默认 Xray 实例。
- 生命周期操作仍调用现有 XrayService。

验收：

- default-xray 可展示。
- Xray 原有启停重启不变。
- 旧 API 不变。

阶段 10.2：CoreManager 包装默认 Xray

目标：

```text
用 CoreManager 调度默认 Xray，但不迁移旧 Inbound 表。
```

施工内容：

- 新增 `core/` 基础结构。
- 新增 `CoreRegistry`。
- 新增 `CoreManager`。
- 新增 `XrayAdapter`。
- 旧 API 内部转发：

```text
restartXrayService -> CoreManager.Restart(default-xray)
stopXrayService    -> CoreManager.Stop(default-xray)
xraylogs           -> CoreManager.Logs(default-xray)
```

验收：

- 旧 UI 和新 UI 行为一致。
- Xray 定时任务仍正常。
- 订阅服务不受影响。

阶段 10.3：sing-box experimental

目标：

```text
先以实验模式接入 sing-box，不做动态表单和统一入站。
```

施工内容：

- sing-box 二进制路径配置。
- 手写 JSON 配置。
- `sing-box check -c config.json`。
- start / stop / restart。
- 日志读取。
- 独立工作目录。
- 独立配置目录。
- 独立日志目录。

暂不做：

- 不做 sing-box 入站图形化表单。
- 不做统一订阅。
- 不做 Neutral Model。
- 不做自动下载和升级。

验收：

- sing-box 可独立启动和停止。
- sing-box 启动失败不影响 Xray。
- sing-box 配置错误可展示校验输出。

阶段 10.4：Capability Schema 和动态表单

目标：

```text
在 sing-box experimental 稳定后，再做能力驱动 UI。
```

施工内容：

- Capability Schema version。
- 字段类型：
  - text
  - password
  - number
  - switch
  - select
  - multi-select
  - textarea
  - json
  - yaml
  - uuid
  - port
  - ip
  - domain
  - file-path
- 后端二次校验。
- 前端 DynamicForm。

验收：

- 可用动态表单创建最小 sing-box inbound。
- 后端能生成并校验配置。
- 表单 schema 有版本号。

阶段 10.5：统一订阅模型

目标：

```text
将旧 Xray 入站和新内核入站统一映射为 SubscriptionNode。
```

施工内容：

- `SubscriptionNode`。
- URI exporter。
- sing-box JSON exporter。
- mihomo YAML exporter。
- Clash YAML exporter。
- 旧 Xray inbound 映射器。
- sing-box inbound 映射器。

验收：

- 原始 Xray 订阅输出不变。
- 新增 sing-box/mihomo 导出不影响旧订阅。

---

## 17. 测试策略

### 后端测试（每阶段必执行）

```powershell
go test ./...
go vet ./...
go build -o bin/SuperXray.exe ./main.go
```

### 前端测试（从阶段 1 开始每阶段必执行）

```powershell
cd frontend
npm run typecheck
npm run lint
npm run build
```

### E2E 策略（增量式，每阶段补对应流程）

> **原则**：不在阶段末一次性补全部 E2E，而是每个阶段完成后补充该阶段涉及的核心流程。
> 框架使用 Playwright（已配置 MCP），阶段 0 建立基础 E2E 骨架。

| 阶段   | 需补充的 E2E 流程                                                        |
| ------ | ------------------------------------------------------------------------ |
| 阶段 0 | E2E 基础框架搭建、登录                                                   |
| 阶段 4 | Dashboard 加载、Xray 状态读取、面板日志读取、Xray 日志读取               |
| 阶段 5 | Xray 重启、Xray 版本查看、配置模板编辑保存                               |
| 阶段 6 | 新增入站、新增客户端、生成分享链接、重置流量、删除客户端、删除入站       |
| 阶段 7 | 访问订阅、修改设置、数据库备份                                           |
| 阶段 8 | 新 UI 默认入口加载、旧 UI 回退入口访问                                   |
| 阶段 9 | 日志 XSS、配置预览 XSS、CSRF token 缺失/错误、未登录下载、数据库导入校验 |

### 完整 E2E 流程清单（最终覆盖目标）

1. 登录。
2. Dashboard 加载。
3. Xray 状态读取。
4. Xray 重启。
5. 面板日志读取。
6. Xray 日志读取。
7. 新增入站。
8. 新增客户端。
9. 生成分享链接。
10. 访问订阅。
11. 重置流量。
12. 删除客户端。
13. 删除入站。
14. 修改设置。
15. 数据库备份。

### 安全测试（阶段 9 集中验证）

1. 日志 XSS。
2. 配置预览 XSS。
3. 订阅预览 XSS。
4. CSRF token 缺失。
5. CSRF token 错误。
6. 未登录下载配置。
7. 未登录下载日志。
8. 数据库导入大小限制。

---

## 18. 风险矩阵

| 风险                             | 等级   | 影响                                   | 缓解                                                    |
| -------------------------------- | ------ | -------------------------------------- | ------------------------------------------------------- |
| 新 UI 与旧 API 字段不一致        | 高     | 保存后旧 UI 无法读取                   | UI 迁移期只按旧 API 类型建模，不写新字段                |
| 入站表单迁移遗漏边缘协议         | 高     | 用户配置丢失或不可编辑                 | 阶段 0 建完整 parity checklist，阶段 6 按协议拆分子阶段 |
| **前端前提假设错误（审查校正）** | **高** | **Vue 2→Vue 3 从零搭建，工期严重低估** | **已校正阶段 1 工期为 6-10 天，前端全部新写**           |
| CSP 收紧影响旧 UI                | 中     | 页面脚本无法运行                       | 新旧 UI 分路径 CSP 策略，阶段 1 即落地基础设施          |
| base path 处理错误               | 中     | 反代路径下资源 404                     | Vite 使用相对路径，Go 注入 basePath                     |
| Telegram Bot 重启冲突            | 中     | Bot 409 冲突                           | 保持现有 StopBot 流程                                   |
| E2E 不足                         | 中     | 回归难发现                             | 每阶段增量补充对应流程的 E2E                            |
| sing-box 提前接入                | 高     | UI 和后端双线复杂化                    | 明确禁止 UI 迁移阶段接入新内核                          |
| CoreManager 提前深改             | 高     | Xray 稳定性下降                        | 必须等新 UI 稳定后再进入阶段 10                         |

---

## 19. 文件与模块影响范围

UI 迁移阶段主要新增或修改：

```text
frontend/
web/
web/controller/
web/middleware/
web/html/
web/assets/
```

UI 迁移阶段原则上不修改：

```text
database/model/model.go
xray/process.go
web/service/xray.go
web/service/inbound.go
sub/
```

多内核阶段才新增或修改：

```text
core/
database/model/core_instance.go
web/service/core_service.go
web/controller/core.go
core/xray/
core/singbox/
core/hysteria2/
```

---

## 20. 版本发布策略

> **审查校正**：原方案使用 `vNext` 命名，与当前 v3 版本号不易区分。
> 建议改为 `v4.0.0-alpha/beta/rc` 格式，与当前版本主线对齐。

建议发布节奏：

1. `v4.0.0-alpha.1`
   - 新 UI 空壳和只读 Dashboard。

2. `v4.0.0-alpha.2`
   - 新 UI 日志、配置查看、Xray 生命周期。

3. `v4.0.0-beta.1`
   - 新 UI 入站和客户端核心流程。

4. `v4.0.0-beta.2`
   - 新 UI 设置、订阅、备份恢复。

5. `v4.0.0-rc.1`
   - 新 UI 默认入口，旧 UI 保留 `/panel/legacy`。

6. `v4.0.0`
   - 新 UI 稳定版。

7. `v4.1.0-alpha`
   - CoreManager default-xray 只读实例。

8. `v4.2.0-alpha`
   - sing-box experimental。

---

## 21. 最终施工路线图

```text
阶段 0：冻结旧 UI 行为和验收清单
  ↓
阶段 1：新前端工程骨架
  ↓
阶段 2：Go 接入新 UI 静态资源
  ↓
阶段 3：API SDK 和类型层
  ↓
阶段 4：只读 Dashboard / 日志 / 配置预览
  ↓
阶段 5：Xray 生命周期和配置管理
  ↓
阶段 6：入站和客户端管理
  ↓
阶段 7：设置、订阅、备份恢复
  ↓
阶段 8：新 UI 灰度成为默认入口
  ↓
阶段 9：安全收口
  ↓
阶段 10：CoreManager / default-xray / sing-box experimental
```

当前状态（2026-05-03）：

- Phase 8 已完成本地隔离面板验收：`/panel/` 默认进入新 UI，`/panel/legacy/` 旧 UI 回退可用，`/panel/ui/` 兼容入口可用。
- Phase 9 已完成一轮安全收口：旧 UI HTML sink、settings/xray CSRF、未登录下载、非法 DB 导入、新 UI `style-src` nonce 收紧和 E2E 回归已落地。
- Phase 10.1-10.5 已完成准入门禁评估，但尚未满足实施条件；需先完成真实 Xray core E2E、mutation E2E 和旧 UI 兼容抽检。

---

## 22. 后端方案交叉验证与裁决矩阵

本节用于把本施工方案与 [`../02-architecture/backend-multi-core-architecture-plan.md`](../02-architecture/backend-multi-core-architecture-plan.md) 进行交叉验证，明确哪些后端架构内容可以提前准备，哪些必须延后到 Xray 新 UI 稳定之后。

### 22.1 交叉验证结论

```text
后端架构方案方向正确；
但其 CoreManager、CoreInstance、proxy_inbounds、Capability Schema、SubscriptionNode 等内容
必须整体后移到阶段 10 之后分批落地。
```

裁决原则：

1. UI 迁移阶段只消费旧 API，不改变旧数据模型。
2. `CoreManager` 可以设计，但不在阶段 0 到阶段 9 进入主链路。
3. `CoreInstance` 首次落地必须是只读 `default-xray`，不能立即带来多实例写入。
4. `proxy_inbounds` 和 `proxy_clients` 必须晚于 `default-xray` 稳定运行。
5. `Capability Schema` 必须晚于 sing-box experimental 手写配置模式。
6. 统一订阅模型必须晚于至少一个非 Xray 内核稳定运行。

### 22.2 后端方案逐项对照

| 后端方案项                     | 后端方案位置       | 本施工方案裁决     | 允许落地阶段             | 说明                                 |
| ------------------------------ | ------------------ | ------------------ | ------------------------ | ------------------------------------ |
| CoreManager / CoreRegistry     | 后端方案 3、6、7   | 接受，但延后       | 阶段 10.2                | 新 UI 稳定前不接入主启动链路         |
| XrayAdapter                    | 后端方案 4、15、22 | 接受，作为兼容包装 | 阶段 10.2                | 首期只包装旧 XrayService             |
| CoreInstance 表                | 后端方案 9、22     | 接受，但先只读     | 阶段 10.1                | 仅创建 `default-xray` 视图概念       |
| proxy_inbounds / proxy_clients | 后端方案 9、22     | 延后               | 阶段 10.5 之后           | UI 迁移期禁止迁旧 Inbound 表         |
| NeutralConfig                  | 后端方案 10        | 延后               | sing-box experimental 后 | 先手写配置，再抽象生成器             |
| ConfigBuilder                  | 后端方案 10、11    | 分批接受           | 阶段 10.3 起             | sing-box 先做 check 和手写 JSON      |
| ProcessRunner / Supervisor     | 后端方案 12        | 接受，谨慎接入     | 阶段 10.3 起             | 先用于 sing-box，不重写 Xray process |
| CoreLogLine / LogParser        | 后端方案 13        | 接受               | 阶段 10.3 起             | 阶段 4 日志中心仍先读旧日志 API      |
| `/panel/api/cores`             | 后端方案 14        | 接受，延后         | 阶段 10.1 起             | 阶段 0-9 不新增核心依赖              |
| 旧 API 转发                    | 后端方案 15        | 接受               | 阶段 10.2                | 必须保证旧 UI 和新 UI 行为一致       |
| Service Container              | 后端方案 17        | 谨慎接受           | CoreManager 稳定后       | 不作为 UI 迁移前置                   |
| SubscriptionNode               | 后端方案 18        | 延后               | 阶段 10.5                | 必须保证原始 Xray 订阅输出不变       |
| core_assets / 二进制管理       | 后端方案 19        | 延后               | sing-box MVP 后          | 首期不做自动下载和升级               |
| 多内核安全要求                 | 后端方案 20        | 提前吸收安全基线   | 阶段 1、9、10            | CSP/CSRF/日志 XSS 先收口             |

### 22.3 阶段门禁

进入阶段 10 前必须满足：

- 新 UI 已完成 Dashboard、日志、Xray 生命周期、入站、客户端、设置、订阅迁移。
- 旧 UI 回退入口可用。
- 新旧 UI 对旧 `model.Inbound` 数据无格式漂移。
- Xray 关键 E2E 全部通过。
- `go test ./...`、`go vet ./...`、`go build -o bin/SuperXray.exe ./main.go` 通过。
- 前端 `typecheck`、`lint`、`build` 通过。
- 日志 XSS 和 CSRF 回归测试通过。

禁止提前进入阶段 10 的信号：

- 新 UI 仍有未迁移的 Xray 核心写入流程。
- 新 UI 写入的数据旧 UI 无法编辑。
- 订阅输出与旧 UI 不一致。
- Xray 重启、日志、入站、客户端任一 E2E 不稳定。

---

## 23. 多角色代理与技能推进机制

本项目采用项目级 `.codex/agents/` 与 `.codex/skills/` 来承载多角色推进策略。多角色配置只描述项目阶段职责和门禁，不复制全局 MCP、sandbox、approval 或模型配置。

### 23.1 项目代理角色

| 角色          | 项目配置                                          | 主要职责                                     | 介入阶段         |
| ------------- | ------------------------------------------------- | -------------------------------------------- | ---------------- |
| UI 迁移总协调 | `.codex/agents/superxray-ui-program-manager.toml` | 阶段裁决、交叉验证、禁止事项检查             | 全阶段           |
| 前端迁移实现  | `.codex/agents/superxray-frontend-migrator.toml`  | Vue3/Vite/TS、页面迁移、旧 API SDK           | 阶段 1-8         |
| Go 集成实现   | `.codex/agents/superxray-go-integration.toml`     | Go embed、base path、CSP 分路径、旧 API 兼容 | 阶段 2、5、9、10 |
| 安全门禁      | `.codex/agents/superxray-security-gate.toml`      | CSP、CSRF、XSS、下载鉴权、导入安全           | 阶段 1、4、9、10 |
| E2E 门禁      | `.codex/agents/superxray-e2e-gate.toml`           | Playwright 流程、截图、追踪、阶段验收        | 阶段 0、4-9      |
| 发布门禁      | `.codex/agents/superxray-release-gate.toml`       | release gate、CHANGELOG、版本与产物          | beta/rc/release  |

### 23.2 项目技能

新增技能：

```text
.codex/skills/superxray-ui-first-migration/SKILL.md
```

触发场景：

- 执行或审查 UI 先行迁移任务。
- 调整 [`ui-first-xray-stable-multi-core-roadmap.md`](ui-first-xray-stable-multi-core-roadmap.md)。
- 新增 Vue3/Vite 前端工程。
- 修改 Xray 新旧 UI 兼容路径。
- 准备进入 CoreManager 或 sing-box 阶段。

技能职责：

1. 先判定当前任务所属阶段。
2. 校验是否触犯阶段禁止事项。
3. 读取施工计划和后端架构方案中的相关章节。
4. 按角色分派实现、审查和验证。
5. 输出阶段验收所需命令和回滚方式。

### 23.3 多角色推进流程

每个阶段按以下顺序推进：

```text
Program Manager 确认阶段和门禁
  ↓
Architect / Go Integration / Frontend Migrator 拆分实现边界
  ↓
TDD 或最小回归测试先补关键路径
  ↓
Worker 实施当前阶段最小改动
  ↓
Security Gate 审查安全边界
  ↓
E2E Gate 跑阶段用户路径
  ↓
Build Resolver 处理构建和类型错误
  ↓
Code Reviewer / Go Reviewer 做最终审查
  ↓
Release Gate 仅在发布候选阶段介入
```

### 23.4 角色调用边界

- UI 阶段的前端实现不得修改 `database/model/model.go`、`xray/process.go`、`web/service/xray.go`、`web/service/inbound.go` 的核心语义。
- Go 集成阶段只允许新增新 UI 托管、runtime config、路径感知 CSP、旧 API 兼容补强。
- 安全门禁可以阻断阶段验收，但不直接推动 UI 范围外重构。
- E2E 门禁以阶段关键路径为准，不要求一次性覆盖全部最终流程。
- Release 门禁仅在 alpha/beta/rc/release 准备时运行。

---

## 24. 结论

本施工方案的关键不是“更快接入多内核”，而是先降低工程风险：

```text
先把现有 Xray 用户体验迁到可维护的新 UI
  ↓
再把默认 Xray 包装成实例
  ↓
最后扩展新内核
```

只要严格遵守“不在 UI 迁移阶段迁数据库、不提前接 sing-box、不提前深改 CoreManager”的边界，该路线具有较高可行性，且每个阶段都可以独立验证和回滚。
