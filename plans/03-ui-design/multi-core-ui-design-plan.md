# SuperXray-gui 多内核 UI 设计方案

> 目标：将 SuperXray-gui 的前端从“Xray 单内核管理界面”升级为“多内核代理面板 UI”，支持 Xray-core、sing-box、Hysteria 2、mihomo 等内核的实例管理、协议配置、配置预览、状态监控、日志查看和订阅导出。

---

## 1. 设计背景

SuperXray-gui 当前 UI 主要围绕 Xray-core 构建：

- 页面文案和按钮大量使用 Xray 语义，例如“Restart Xray”“Xray Logs”“Xray Version”。
- 入站、用户、订阅、日志、配置查看都默认指向单一 Xray 实例。
- 前端使用 Go template + Vue 全局脚本 + Ant Design Vue 静态资源的方式组织页面。
- 多数页面是大模板文件，组件化、类型约束、状态管理和构建体系较弱。
- 当前界面适合单内核场景，但不适合未来动态增加多个内核、多个实例和多个协议能力。

多内核升级后，UI 需要从“协议配置页面”升级成“能力驱动的控制台”。

---

## 2. UI 总体目标

升级后的 UI 应支持：

```text
多内核 UI 控制台
= 内核实例管理
+ 动态协议表单
+ 配置预览和校验
+ 多实例运行状态
+ 统一日志中心
+ 统一订阅导出
+ 内核版本管理
+ 多内核 Dashboard
```

首期必须满足：

```text
1. 保留原 Xray 使用体验
2. 新增 sing-box 实例创建和管理
3. UI 不硬编码内核字段
4. 表单由后端 capability schema 驱动
5. 支持未来新增 Hysteria 2、mihomo、其他内核
```

---

## 3. UI 框架是否需要升级

### 3.1 结论

**需要升级，但不建议立即全量重写。**

建议采用两阶段路线：

```text
短期：
继续使用当前 UI 框架，新增“内核管理”页面，验证多内核后端能力。

中期：
迁移到 Vue 3 + Vite + TypeScript + Ant Design Vue 4，重构为模块化前端工程。
```

### 3.2 为什么需要升级

当前 UI 的主要问题：

```text
1. 页面模板过大，业务逻辑和视图混杂。
2. 缺少组件化动态表单体系。
3. 缺少 TypeScript 类型约束。
4. 缺少现代前端构建、lint、typecheck、单元测试。
5. CSP 安全优化困难，因为存在大量内联脚本。
6. 多内核能力差异会导致页面 if/else 急速膨胀。
7. 未来配置预览、diff、Monaco 编辑器、图表、WebSocket 状态管理都难以维护。
```

### 3.3 推荐技术栈

首选：

```text
Vue 3
Vite
TypeScript
Pinia
Vue Router
Ant Design Vue 4
Monaco Editor 或 CodeMirror
ECharts
Axios
```

备选：

```text
Vue 3
Vite
TypeScript
Pinia
Vue Router
Element Plus
Monaco Editor 或 CodeMirror
ECharts
Axios
```

推荐继续使用 **Ant Design Vue 4**，因为当前界面已经是 Ant Design Vue 风格，迁移成本最低。

---

## 4. UI 演进路线

### 4.1 第一阶段：不重写，新增多内核页面

目标：

```text
- 不破坏现有 Xray 页面
- 新增“内核管理”菜单
- 接入 CoreManager API
- 支持 Xray 默认实例展示
- 支持 sing-box 实例创建、校验、启动、停止、重启
```

适合时间：

```text
3～7 天
```

交付页面：

```text
内核管理
  - 实例列表
  - 新建实例
  - 版本管理
  - 配置预览
  - 日志查看
```

### 4.2 第二阶段：前端工程化

新增目录：

```text
frontend/
  package.json
  vite.config.ts
  tsconfig.json
  src/
    main.ts
    App.vue
    router/
    stores/
    api/
    components/
    layouts/
    views/
```

构建产物：

```text
frontend/dist
  ↓
Go embed
  ↓
web/assets 或 web/dist
```

### 4.3 第三阶段：模块化重构

逐步把旧页面拆成：

```text
Dashboard
Core Management
Inbound Management
Client Management
Routing / DNS
Subscription
Logs
Settings
```

### 4.4 第四阶段：动态能力驱动

最终目标：

```text
后端返回 capability schema
  ↓
前端动态生成协议表单
  ↓
提交 Neutral Model
  ↓
后端生成对应内核配置
```

---

## 5. 信息架构设计

推荐左侧菜单结构：

```text
仪表盘

内核管理
  - 内核实例
  - 新建实例
  - 版本管理
  - 配置模板

入站管理
  - 全部入站
  - 新建入站
  - 批量操作

用户管理
  - 用户列表
  - 流量限制
  - 到期管理
  - 在线用户

路由与 DNS
  - 路由规则
  - DNS 设置
  - 规则集
  - 出站策略组

订阅管理
  - 订阅用户
  - 导出格式
  - 客户端模板
  - 订阅预览

日志中心
  - 面板日志
  - 内核日志
  - 访问日志
  - 错误日志

系统设置
  - 面板安全
  - 证书管理
  - Telegram Bot
  - LDAP
  - 备份恢复
```

---

## 6. 多内核 Dashboard 设计

### 6.1 顶部概览卡片

```text
运行中实例数
总入站数
总用户数
今日流量
异常实例数
订阅访问数
```

### 6.2 内核状态卡片

每个内核实例显示：

```text
实例名称
内核类型
版本
运行状态
运行时长
CPU
内存
监听端口
入站数量
用户数量
最近错误
```

状态颜色：

```text
running       绿色
stopped       灰色
starting      蓝色
restarting    黄色
error         红色
unknown       紫色/灰色
```

### 6.3 快捷操作

```text
启动
停止
重启
配置校验
查看日志
查看配置
```

### 6.4 图表区域

建议使用 ECharts：

```text
全局流量曲线
实例 CPU 曲线
实例内存曲线
连接数曲线
在线用户趋势
```

---

## 7. 内核实例列表页面

### 7.1 表格字段

```text
名称
内核
版本
状态
自启动
监听端口
入站数
用户数
运行时长
最后更新时间
操作
```

### 7.2 操作按钮

```text
详情
启动
停止
重启
校验
配置
日志
升级
删除
```

### 7.3 筛选条件

```text
内核类型：全部 / Xray / sing-box / Hysteria2 / mihomo
状态：全部 / 运行中 / 已停止 / 异常
是否自启动
关键字搜索
端口搜索
```

### 7.4 批量操作

```text
批量启动
批量停止
批量重启
批量校验
批量启用自启动
批量禁用自启动
```

---

## 8. 新建内核实例向导

推荐使用 Step Form。

### Step 1：选择内核

```text
Xray-core
sing-box
Hysteria 2
mihomo
```

每个内核卡片展示：

```text
名称
说明
适合用途
支持协议
当前安装版本
是否已安装
```

示例：

```text
Xray-core
适合：VLESS/REALITY/XHTTP/Vision 等 Xray 生态
推荐：默认通用核心

sing-box
适合：多协议、DNS、路由、TUN、规则集
推荐：高级通用核心

Hysteria 2
适合：UDP/QUIC、高丢包网络、高速专用实例
推荐：专用高速核心

mihomo
适合：Clash 生态、订阅聚合、客户端规则
推荐：订阅和客户端导出增强
```

### Step 2：实例基础信息

字段：

```text
实例名称
是否启用
是否自启动
工作目录
配置文件路径
日志文件路径
```

### Step 3：选择用途

```text
普通代理服务
透明代理 / TUN
专用 Hysteria2 服务
客户端配置 / 订阅导出
高级自定义配置
```

### Step 4：创建入站

字段由 capability schema 动态生成。

基础字段：

```text
协议
监听地址
端口
传输层
TLS 模式
备注
```

### Step 5：用户配置

字段根据协议变化：

```text
VLESS：
- UUID
- Flow
- Email
- 到期时间
- 流量限制
- IP 限制

Trojan：
- Password
- Email
- 到期时间
- 流量限制

Shadowsocks：
- Method
- Password
- Email
- 流量限制

Hysteria2：
- Password
- Bandwidth
- Masquerade
```

### Step 6：路由与 DNS

如果内核支持：

```text
DNS 服务器
路由规则
规则集
出站策略
final outbound
auto detect interface
```

### Step 7：配置预览与校验

展示：

```text
生成配置
配置 diff
校验结果
启动前端口冲突检查
确认创建
```

---

## 9. 内核详情页面

### 9.1 页面结构

```text
基本信息
运行状态
入站列表
用户列表
路由/DNS
配置预览
日志
事件记录
```

### 9.2 基本信息

```text
实例 ID
实例名称
内核类型
版本
二进制路径
配置路径
日志路径
是否启用
是否自启动
创建时间
更新时间
```

### 9.3 运行状态

```text
状态
PID
运行时长
CPU
内存
连接数
流量
最近错误
```

### 9.4 操作区

```text
启动
停止
重启
Reload
校验配置
重新生成配置
下载配置
升级内核
删除实例
```

---

## 10. 入站管理 UI

### 10.1 入站列表

字段：

```text
名称
所属实例
内核
协议
监听地址
端口
传输层
TLS
用户数
启用状态
流量
操作
```

### 10.2 新建入站

推荐流程：

```text
选择实例
  ↓
后端返回该实例 core capability
  ↓
选择协议
  ↓
动态生成字段
  ↓
配置用户
  ↓
预览生成配置
  ↓
提交保存
```

### 10.3 动态字段示例

```json
{
  "name": "flow",
  "label": "Flow",
  "type": "select",
  "required": false,
  "options": [
    {
      "label": "None",
      "value": ""
    },
    {
      "label": "Vision",
      "value": "xtls-rprx-vision"
    }
  ],
  "visibleWhen": {
    "core": "xray",
    "protocol": "vless"
  }
}
```

### 10.4 表单字段类型

前端应支持：

```text
text
password
number
switch
select
multi-select
textarea
json
yaml
uuid
port
ip
domain
file-path
certificate-picker
```

---

## 11. 用户管理 UI

### 11.1 用户列表字段

```text
Email
所属入站
所属内核
协议
UUID / Password
启用状态
已用流量
总流量
到期时间
IP 限制
订阅 ID
操作
```

### 11.2 用户操作

```text
新增用户
编辑用户
启用/禁用
重置流量
延长到期时间
复制链接
查看二维码
删除用户
批量导入
批量导出
```

### 11.3 多协议用户字段处理

统一展示：

```text
凭据
```

展开后根据协议展示：

```text
VLESS/VMess：UUID
Trojan：Password
Shadowsocks：Method + Password
Hysteria2：Password
```

---

## 12. 路由与 DNS UI

### 12.1 为什么要单独页面

sing-box 和 mihomo 对路由、DNS、规则集支持更丰富。如果仍然放在普通入站表单里，页面会非常复杂。

### 12.2 页面模块

```text
DNS 服务器
DNS 规则
路由规则
规则集 rule_set
出站列表
策略组
final outbound
```

### 12.3 适配策略

```text
Xray：
显示基础 routing 配置。

sing-box：
显示 DNS + route + rule_set + outbound selector。

mihomo：
显示 proxy groups + rules + providers。

Hysteria2：
通常不显示复杂路由，仅显示 resolver 或高级配置。
```

---

## 13. 配置预览 UI

### 13.1 功能

必须支持：

```text
当前配置
新生成配置
Diff 对比
格式化
复制
下载
校验
回滚
```

### 13.2 编辑器

推荐：

```text
Monaco Editor
或 CodeMirror
```

支持格式：

```text
JSON
YAML
TOML
Plain text
```

### 13.3 校验结果展示

```text
校验成功
校验失败
错误行号
错误内容
原始输出
修复建议
```

---

## 14. 日志中心 UI

### 14.1 日志分类

```text
面板日志
Xray 日志
sing-box 日志
Hysteria2 日志
mihomo 日志
访问日志
错误日志
```

### 14.2 筛选条件

```text
实例
内核类型
日志级别
时间范围
关键字
用户
协议
IP
```

### 14.3 安全要求

日志区域不要使用 `v-html`。

推荐：

```html
<pre>{{ logLine.message }}</pre>
```

或：

```html
<span>{{ logLine.message }}</span>
```

如果需要高亮，使用结构化 token 渲染，不直接插入 HTML。

### 14.4 实时日志

WebSocket 推送：

```text
/panel/ws/logs?instanceId=xxx
```

支持：

```text
暂停
继续
清空
自动滚动
下载
复制
```

---

## 15. 订阅管理 UI

### 15.1 订阅用户列表

字段：

```text
用户
订阅 ID
绑定入站
绑定内核
导出格式
访问次数
最后访问时间
启用状态
操作
```

### 15.2 导出格式

```text
原始 URI
Xray JSON
sing-box JSON
mihomo YAML
Clash YAML
Shadowrocket URI
NekoBox / sing-box compatible
```

### 15.3 订阅预览

支持：

```text
复制订阅链接
二维码
预览导出内容
下载配置
测试生成
```

### 15.4 客户端模板

```text
sing-box Android
sing-box macOS
mihomo
Clash Verge
Shadowrocket
NekoBox
v2rayN
```

---

## 16. 内核版本管理 UI

### 16.1 页面功能

```text
查看已安装内核
查看最新版本
下载新版本
校验 SHA256
安装
回滚
删除旧版本
```

### 16.2 表格字段

```text
内核
当前版本
最新版本
OS
Arch
安装路径
校验状态
更新时间
操作
```

### 16.3 安全提示

安装前提示：

```text
请确认下载来源可信。
安装前会校验 SHA256。
运行中的实例不会被直接覆盖。
升级后可选择重启实例。
```

---

## 17. 多内核状态 WebSocket

### 17.1 事件类型

```text
core.status
core.started
core.stopped
core.restarted
core.error
core.log
core.config.validated
core.version.updated
traffic.updated
client.online
client.offline
```

### 17.2 前端状态管理

Pinia store：

```text
stores/
  coreStore.ts
  inboundStore.ts
  clientStore.ts
  logStore.ts
  subscriptionStore.ts
  settingsStore.ts
```

### 17.3 coreStore 示例

```ts
interface CoreInstance {
  id: number;
  name: string;
  coreType: "xray" | "sing-box" | "hysteria2" | "mihomo";
  version: string;
  status:
    | "running"
    | "stopped"
    | "starting"
    | "stopping"
    | "restarting"
    | "error"
    | "unknown";
  autoStart: boolean;
  ports: number[];
  uptime: number;
  cpu: number;
  memory: number;
}
```

---

## 18. API SDK 设计

前端不要在组件里直接写 axios URL，建议封装 API SDK。

```text
src/api/
  request.ts
  core.ts
  inbound.ts
  client.ts
  subscription.ts
  logs.ts
  settings.ts
```

示例：

```ts
export function listCoreInstances() {
  return request.get<CoreInstance[]>("/panel/api/cores/instances");
}

export function startCoreInstance(id: number) {
  return request.post(`/panel/api/cores/instances/${id}/start`);
}

export function validateCoreConfig(id: number) {
  return request.post<ValidationResult>(
    `/panel/api/cores/instances/${id}/validate`,
  );
}
```

---

## 19. 动态表单引擎

### 19.1 目标

通过后端 capability schema 动态生成表单，避免 UI 硬编码内核协议字段。

### 19.2 组件结构

```text
DynamicForm.vue
DynamicField.vue
FieldText.vue
FieldNumber.vue
FieldSelect.vue
FieldSwitch.vue
FieldJson.vue
FieldYaml.vue
FieldCertificate.vue
```

### 19.3 schema 示例

```json
{
  "fields": [
    {
      "name": "port",
      "label": "端口",
      "type": "port",
      "required": true,
      "default": 443
    },
    {
      "name": "tlsMode",
      "label": "TLS 模式",
      "type": "select",
      "options": [
        { "label": "None", "value": "none" },
        { "label": "TLS", "value": "tls" },
        { "label": "REALITY", "value": "reality" }
      ]
    }
  ]
}
```

### 19.4 前端校验

```text
必填
端口范围
IP 格式
域名格式
UUID 格式
JSON 格式
YAML 格式
文件路径格式
字段依赖关系
```

后端仍必须再次校验，不可信任前端校验。

---

## 20. 权限与安全 UI

如果未来增加多用户管理，UI 应支持权限：

```text
管理员
运维
只读
订阅用户
```

权限矩阵：

```text
查看 Dashboard
查看日志
新增实例
修改配置
启动/停止实例
下载数据库
修改安全设置
查看订阅链接
```

即使当前不马上实现，也建议 UI 结构预留。

---

## 21. 响应式设计

### 21.1 桌面端

```text
左侧菜单
顶部状态栏
主内容表格
右侧抽屉详情
```

### 21.2 移动端

```text
底部导航或折叠菜单
卡片式实例列表
关键操作按钮固定底部
日志页面支持横向滚动
配置预览只读优先
```

### 21.3 移动端重点

```text
按钮最小高度 44px
表格改卡片
抽屉代替复杂弹窗
长配置不默认展开
日志自动换行可切换
```

---

## 22. 视觉设计原则

### 22.1 颜色语义

```text
绿色：运行中 / 成功
蓝色：信息 / 启动中
黄色：警告 / 重启中
红色：错误 / 停止失败
灰色：停止 / 禁用
紫色：未知 / 高级
```

### 22.2 标签样式

内核标签：

```text
Xray-core
sing-box
Hysteria2
mihomo
```

协议标签：

```text
VLESS
VMess
Trojan
Shadowsocks
Hysteria2
TUIC
```

TLS 标签：

```text
TLS
REALITY
ACME
None
```

---

## 23. 交互设计原则

### 23.1 危险操作二次确认

需要确认：

```text
删除实例
停止运行中实例
覆盖配置
导入数据库
删除用户
升级内核
回滚配置
```

### 23.2 配置修改流程

推荐：

```text
编辑
  ↓
生成预览
  ↓
校验配置
  ↓
保存
  ↓
提示是否重启实例
```

不要保存后立即重启，除非用户开启自动应用。

### 23.3 错误提示

错误提示应包含：

```text
错误原因
影响范围
建议操作
原始输出展开项
```

---

## 24. 前端项目结构

推荐结构：

```text
frontend/
  src/
    main.ts
    App.vue

    router/
      index.ts

    layouts/
      MainLayout.vue
      AuthLayout.vue

    views/
      dashboard/
        DashboardView.vue

      cores/
        CoreListView.vue
        CoreCreateView.vue
        CoreDetailView.vue
        CoreVersionView.vue

      inbounds/
        InboundListView.vue
        InboundCreateView.vue
        InboundDetailView.vue

      clients/
        ClientListView.vue
        ClientCreateView.vue

      routing/
        RoutingView.vue
        DNSView.vue

      subscriptions/
        SubscriptionListView.vue
        SubscriptionPreviewView.vue

      logs/
        LogCenterView.vue

      settings/
        SettingsView.vue
        SecuritySettingsView.vue
        CertificateSettingsView.vue

    components/
      core/
        CoreStatusBadge.vue
        CoreTypeTag.vue
        CoreActionButtons.vue
        CoreInstanceCard.vue

      form/
        DynamicForm.vue
        DynamicField.vue

      config/
        ConfigPreview.vue
        ConfigDiff.vue
        ConfigValidationResult.vue

      logs/
        LogViewer.vue
        LogFilter.vue

      common/
        PageHeader.vue
        ConfirmAction.vue
        StatusTag.vue

    stores/
      coreStore.ts
      inboundStore.ts
      clientStore.ts
      logStore.ts
      settingsStore.ts

    api/
      request.ts
      core.ts
      inbound.ts
      client.ts
      logs.ts
      subscription.ts
      settings.ts

    types/
      core.ts
      inbound.ts
      client.ts
      capability.ts
      subscription.ts
```

---

## 25. 与 Go 后端集成

### 25.1 构建流程

```text
cd frontend
npm install
npm run build
```

输出：

```text
frontend/dist
```

Go 端：

```go
//go:embed dist/*
var frontendFS embed.FS
```

### 25.2 base path 支持

因为面板可能配置 `webBasePath`，前端构建必须支持动态 base。

方案：

```text
1. Vite build 使用相对路径
2. Go template 注入 window.__APP_CONFIG__
3. axios baseURL 从 window.__APP_CONFIG__.basePath 读取
```

示例：

```html
<script>
  window.__APP_CONFIG__ = {
    basePath: "{{ .base_path }}",
    version: "{{ .cur_ver }}",
  };
</script>
```

---

## 26. CSP 与安全升级

现代前端工程化后，应逐步实现：

```text
1. 移除大部分 inline script
2. 移除 unsafe-eval
3. 使用 nonce 或 hash
4. 日志不使用 v-html
5. 配置预览只做文本展示
6. 用户输入统一转义
```

安全目标：

```text
script-src 'self'
object-src 'none'
base-uri 'self'
frame-ancestors 'none'
```

如果确实需要 inline script，只保留注入配置的极小脚本，并使用 nonce。

---

## 27. UI 迁移策略

### 27.1 不建议一次性重写

风险：

```text
1. 旧功能容易断。
2. 多语言文案容易丢失。
3. 用户操作习惯变化过大。
4. 开发周期不可控。
```

### 27.2 推荐并行迁移

```text
旧 UI：
继续承载原 Xray 页面。

新 UI：
先承载多内核页面。

成熟后：
逐步迁移 Dashboard、入站、用户、订阅、设置。
```

### 27.3 路由策略

```text
旧页面：
/panel

新页面：
/panel/v2
或直接在同一 Layout 中新增新路由
```

建议初期使用：

```text
/panel/cores
```

不要引入 `/v2`，避免用户感知割裂。

---

## 28. MVP 页面清单

第一版多内核 UI 最小可用页面：

```text
1. 内核实例列表
2. 新建 sing-box 实例
3. 内核详情
4. 配置预览
5. 配置校验
6. 启动/停止/重启
7. 日志查看
```

不建议 MVP 就做：

```text
1. 完整路由规则编辑器
2. 完整 TUN 图形化配置
3. mihomo 全量规则组管理
4. 多租户权限
5. 复杂可视化拓扑
```

---

## 29. 开发排期建议

### 阶段 1：现有 UI 扩展，3～7 天

```text
- 新增内核管理菜单
- 实例列表页面
- 实例操作按钮
- 配置预览弹窗
- 日志弹窗
```

### 阶段 2：动态表单 MVP，5～10 天

```text
- capability API 接入
- DynamicForm
- DynamicField
- sing-box 基础入站表单
- 表单校验
```

### 阶段 3：前端工程化，1～2 周

```text
- frontend 目录
- Vite
- TypeScript
- Pinia
- Router
- API SDK
- Ant Design Vue 4
```

### 阶段 4：核心页面迁移，2～4 周

```text
- Dashboard
- Core Management
- Inbound Management
- Client Management
- Log Center
- Subscription Management
```

### 阶段 5：高级能力，2～4 周

```text
- Monaco/CodeMirror 配置编辑器
- Diff 预览
- ECharts 监控
- WebSocket 实时日志
- 订阅模板管理
- 路由/DNS 高级编辑器
```

---

## 30. 验收标准

### 30.1 功能验收

```text
1. 能展示默认 Xray 实例。
2. 能创建 sing-box 实例。
3. 能配置 sing-box 基础入站。
4. 能校验 sing-box 配置。
5. 能启动、停止、重启内核实例。
6. 能查看对应实例日志。
7. 能导出基础订阅。
8. 不影响原 Xray 使用流程。
```

### 30.2 UI 验收

```text
1. 动态表单由 capability schema 驱动。
2. 页面不硬编码 sing-box 专属字段。
3. 日志不使用 v-html。
4. 配置预览支持 JSON/YAML。
5. 移动端可以完成核心操作。
6. 错误提示清晰。
```

### 30.3 安全验收

```text
1. 所有状态变更 API 经过 CSRF。
2. 配置编辑不执行任意脚本。
3. 日志内容不会触发 XSS。
4. 危险操作有二次确认。
5. 配置下载和日志下载需要登录。
```

---

## 31. 最终建议

UI 升级应遵循以下原则：

```text
先加能力，不急着美化。
先动态表单，不写死协议。
先接入 sing-box，不一次接入所有内核。
先保留旧 Xray，不破坏现有用户。
先工程化，再全面重构。
```

推荐最终 UI 技术路线：

```text
Vue 3 + Vite + TypeScript + Pinia + Ant Design Vue 4
```

推荐最终 UI 架构：

```text
Capability Schema
  ↓
Dynamic Form
  ↓
Neutral Model
  ↓
Config Preview
  ↓
Backend Config Builder
  ↓
CoreManager
```

这样 SuperXray-gui 的 UI 才能真正支撑未来持续增加多内核，而不是每接入一个内核就重写一次页面。
