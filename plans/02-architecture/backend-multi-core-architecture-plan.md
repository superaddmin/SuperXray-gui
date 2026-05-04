# SuperXray-gui 后端架构实施方案

> 目标：将 SuperXray-gui 从“Xray 单内核面板”升级为“多内核代理面板”，首期保留 Xray-core，新增 sing-box，并为后续接入 Hysteria 2、mihomo、其他内核预留架构扩展能力。

---

## 1. 改造背景

当前 SuperXray-gui 的后端设计以 Xray-core 为中心：

- Web Server 启动后直接调用 Xray 相关服务。
- 定时任务围绕 Xray 运行状态、流量统计、日志解析展开。
- API 命名、页面操作、服务层命名多处包含 `Xray`。
- 配置生成、进程管理、日志解析、订阅导出都与 Xray 模型强绑定。

这种架构在单内核阶段简单直接，但如果未来要增加 sing-box、Hysteria 2、mihomo 等内核，会出现以下问题：

1. 每增加一个内核都需要重复写控制器、服务、日志、配置、订阅逻辑。
2. 前端表单会不断堆叠 if/else，难以维护。
3. 不同内核的协议能力不同，不能用单一 Xray 字段模型表达。
4. 进程生命周期、配置校验、版本管理、二进制下载缺少统一抽象。
5. 旧功能和新内核耦合后，容易破坏原有 Xray 面板能力。

因此，后端应先做“内核抽象层”，再逐步接入新内核。

---

## 2. 总体目标

升级后的系统应支持：

```text
多内核代理面板
= 内核注册
+ 内核实例管理
+ 多协议能力模型
+ 配置生成器
+ 生命周期管理
+ 统一日志
+ 统一状态监控
+ 统一订阅导出
+ 版本下载与校验
```

首期目标：

```text
阶段 1：保留 Xray-core，封装为默认内核实例
阶段 2：新增 sing-box 作为第二核心
阶段 3：统一订阅导出模型
阶段 4：接入 Hysteria 2 专用实例
阶段 5：mihomo 作为客户端配置/订阅导出增强
```

---

## 3. 后端架构蓝图

推荐的新架构：

```text
web/controller
    ↓
web/service
    ↓
core/CoreManager
    ↓
core/CoreRegistry
    ↓
core/xray/XrayAdapter
core/singbox/SingBoxAdapter
core/hysteria2/Hysteria2Adapter
    ↓
process manager / config builder / log parser
```

核心思想：

- Controller 不直接操作 Xray。
- Service 不直接硬编码 Xray 逻辑。
- 所有内核通过统一 Core 接口被 CoreManager 调度。
- 每个内核只需要实现自己的 Adapter、ConfigBuilder、Validator、LogParser。
- UI 和 API 通过 capability schema 动态了解内核支持什么功能。

---

## 4. 推荐目录结构

新增目录：

```text
core/
  types.go
  manager.go
  registry.go
  capability.go
  errors.go
  event.go

core/process/
  runner.go
  supervisor.go
  pid.go
  signal.go

core/config/
  neutral_model.go
  renderer.go
  validator.go

core/xray/
  adapter.go
  config_builder.go
  validator.go
  log_parser.go
  binary.go
  process.go

core/singbox/
  adapter.go
  config_builder.go
  validator.go
  log_parser.go
  binary.go
  process.go

core/hysteria2/
  adapter.go
  config_builder.go
  validator.go
  log_parser.go
  binary.go
  process.go

web/service/core_service.go
web/controller/core.go

database/model/core_instance.go
database/model/proxy_inbound.go
database/model/proxy_client.go
database/model/core_asset.go
```

旧 Xray 代码不需要立即删除，先逐步迁移：

```text
xray/
web/service/xray.go
web/service/inbound.go
```

这些旧模块先被 `core/xray` 适配器调用，后续再慢慢内聚。

---

## 5. Core 接口设计

### 5.1 核心接口

```go
type Core interface {
    Name() string
    Type() CoreType
    Version() string

    BinaryPath() string
    ConfigPath(instanceID int64) string
    LogPath(instanceID int64) string

    GenerateConfig(instance CoreInstance) ([]byte, error)
    ValidateConfig(configPath string) error

    Start(instanceID int64) error
    Stop(instanceID int64) error
    Restart(instanceID int64) error
    Reload(instanceID int64) error
    IsRunning(instanceID int64) bool

    GetStatus(instanceID int64) (*CoreStatus, error)
    GetLogs(instanceID int64, query LogQuery) ([]CoreLogLine, error)
}
```

### 5.2 内核类型

```go
type CoreType string

const (
    CoreTypeXray      CoreType = "xray"
    CoreTypeSingBox   CoreType = "sing-box"
    CoreTypeHysteria2 CoreType = "hysteria2"
    CoreTypeMihomo    CoreType = "mihomo"
)
```

### 5.3 状态模型

```go
type CoreStatus struct {
    InstanceID int64     `json:"instanceId"`
    CoreType   CoreType  `json:"coreType"`
    State      string    `json:"state"`
    PID        int       `json:"pid"`
    Version    string    `json:"version"`
    Uptime     int64     `json:"uptime"`
    CPU        float64   `json:"cpu"`
    Memory     uint64    `json:"memory"`
    Error      string    `json:"error,omitempty"`
    UpdatedAt  time.Time `json:"updatedAt"`
}
```

状态枚举：

```text
running
stopped
starting
stopping
restarting
error
unknown
```

---

## 6. CoreRegistry 设计

CoreRegistry 负责注册和获取内核实现。

```go
type CoreRegistry struct {
    cores map[CoreType]Core
}

func NewCoreRegistry() *CoreRegistry {
    return &CoreRegistry{
        cores: map[CoreType]Core{},
    }
}

func (r *CoreRegistry) Register(core Core) {
    r.cores[core.Type()] = core
}

func (r *CoreRegistry) Get(coreType CoreType) (Core, bool) {
    core, ok := r.cores[coreType]
    return core, ok
}

func (r *CoreRegistry) List() []CoreType {
    types := make([]CoreType, 0, len(r.cores))
    for t := range r.cores {
        types = append(types, t)
    }
    return types
}
```

初始化时：

```go
registry := core.NewCoreRegistry()
registry.Register(xray.NewAdapter(...))
registry.Register(singbox.NewAdapter(...))
registry.Register(hysteria2.NewAdapter(...))
```

---

## 7. CoreManager 设计

CoreManager 是统一调度中心。

```go
type CoreManager struct {
    registry CoreRegistry
    repo     CoreInstanceRepository
    events   CoreEventBus
}

func (m *CoreManager) Start(id int64) error
func (m *CoreManager) Stop(id int64) error
func (m *CoreManager) Restart(id int64) error
func (m *CoreManager) Reload(id int64) error
func (m *CoreManager) Validate(id int64) error
func (m *CoreManager) Status(id int64) (*CoreStatus, error)
func (m *CoreManager) Logs(id int64, query LogQuery) ([]CoreLogLine, error)
```

启动流程：

```text
1. 根据 instanceID 读取 core_instances
2. 根据 core_type 从 registry 找到对应 Core
3. 生成配置
4. 写入配置文件
5. 执行配置校验
6. 检查端口冲突
7. 启动进程
8. 更新实例状态
9. 发送 CoreEvent
```

停止流程：

```text
1. 查询实例
2. 找到内核 Adapter
3. 发送优雅停止信号
4. 等待进程退出
5. 超时后强制 kill
6. 更新状态
7. 发送 CoreEvent
```

---

## 8. 能力模型 Capability

### 8.1 为什么需要能力模型

不同内核支持的协议和配置项不同：

- Xray 支持 VLESS、VMess、Trojan、REALITY、Vision、XHTTP 等。
- sing-box 支持 VLESS、VMess、Trojan、Shadowsocks、Hysteria2、TUIC、TUN、route、rule_set 等。
- Hysteria 2 更适合作为单独高速 QUIC 实例。
- mihomo 更适合作为客户端配置和订阅导出。

如果 UI 写死字段，未来每增加内核都会重写表单。

因此后端需要提供 capability API，让前端动态渲染表单。

### 8.2 Capability 结构

```go
type CoreCapability struct {
    CoreType          CoreType              `json:"coreType"`
    DisplayName       string                `json:"displayName"`
    Inbounds          []ProtocolCapability  `json:"inbounds"`
    Outbounds         []ProtocolCapability  `json:"outbounds"`
    SupportsDNS       bool                  `json:"supportsDNS"`
    SupportsRouting   bool                  `json:"supportsRouting"`
    SupportsTUN       bool                  `json:"supportsTUN"`
    SupportsStats     bool                  `json:"supportsStats"`
    SupportsUsers     bool                  `json:"supportsUsers"`
    SupportsAPI       bool                  `json:"supportsAPI"`
    SupportsHotReload bool                  `json:"supportsHotReload"`
}
```

协议能力：

```go
type ProtocolCapability struct {
    Protocol      string          `json:"protocol"`
    DisplayName   string          `json:"displayName"`
    Transports    []string        `json:"transports"`
    TLSModes      []string        `json:"tlsModes"`
    UserFields    []FieldSchema   `json:"userFields"`
    SettingFields []FieldSchema   `json:"settingFields"`
}
```

字段 schema：

```go
type FieldSchema struct {
    Name        string        `json:"name"`
    Label       string        `json:"label"`
    Type        string        `json:"type"`
    Required    bool          `json:"required"`
    Default     any           `json:"default,omitempty"`
    Options     []FieldOption `json:"options,omitempty"`
    Help        string        `json:"help,omitempty"`
    VisibleWhen map[string]any `json:"visibleWhen,omitempty"`
}
```

示例：

```json
{
  "coreType": "sing-box",
  "displayName": "sing-box",
  "supportsDNS": true,
  "supportsRouting": true,
  "supportsTUN": true,
  "inbounds": [
    {
      "protocol": "vless",
      "displayName": "VLESS",
      "transports": ["tcp", "ws", "grpc"],
      "tlsModes": ["none", "tls", "reality"],
      "userFields": [
        {
          "name": "uuid",
          "label": "UUID",
          "type": "uuid",
          "required": true
        }
      ]
    }
  ]
}
```

---

## 9. 数据库设计

### 9.1 core_instances

记录每个内核实例。

```go
type CoreInstance struct {
    Id          int64     `gorm:"primaryKey"`
    Name        string    `gorm:"uniqueIndex"`
    CoreType    string    `gorm:"index"`
    Version     string
    Enabled     bool
    AutoStart   bool
    Status      string
    BinaryPath  string
    ConfigPath  string
    LogPath     string
    ListenPorts string    `gorm:"type:text"` // JSON array
    Extra       string    `gorm:"type:text"` // JSON object
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 9.2 proxy_inbounds

统一入站模型。

```go
type ProxyInbound struct {
    Id             int64     `gorm:"primaryKey"`
    CoreInstanceId int64    `gorm:"index"`
    Remark         string
    Protocol       string   `gorm:"index"`
    Listen         string
    Port           int      `gorm:"index"`
    Transport      string
    TLSMode        string
    Enabled        bool
    Settings       string   `gorm:"type:text"` // JSON
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### 9.3 proxy_clients

统一用户模型。

```go
type ProxyClient struct {
    Id         int64     `gorm:"primaryKey"`
    InboundId  int64    `gorm:"index"`
    Email      string   `gorm:"index"`
    UUID       string
    Password   string
    Flow       string
    LimitIP    int
    TotalGB    int64
    ExpiryTime int64
    Enabled    bool
    Extra      string   `gorm:"type:text"` // JSON
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

### 9.4 core_assets

记录内核二进制。

```go
type CoreAsset struct {
    Id          int64     `gorm:"primaryKey"`
    CoreType    string    `gorm:"index"`
    Version     string    `gorm:"index"`
    Arch        string
    OS          string
    URL         string
    SHA256      string
    BinaryPath  string
    Installed   bool
    Verified    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 9.5 core_events

记录生命周期事件。

```go
type CoreEvent struct {
    Id             int64     `gorm:"primaryKey"`
    CoreInstanceId int64    `gorm:"index"`
    EventType      string
    Message        string
    Level          string
    CreatedAt      time.Time
}
```

---

## 10. 配置生成器设计

### 10.1 Neutral Model

UI 和业务层不直接生成 Xray/sing-box 配置，而是生成统一中间模型。

```go
type NeutralConfig struct {
    Instance CoreInstance
    Inbounds []NeutralInbound
    Outbounds []NeutralOutbound
    DNS      *NeutralDNS
    Routing  *NeutralRouting
}
```

入站模型：

```go
type NeutralInbound struct {
    Protocol  string
    Listen    string
    Port      int
    Transport string
    TLS       *NeutralTLS
    Users     []NeutralUser
    Settings  map[string]any
}
```

用户模型：

```go
type NeutralUser struct {
    Email      string
    UUID       string
    Password   string
    Flow       string
    LimitIP    int
    TotalGB    int64
    ExpiryTime int64
    Extra      map[string]any
}
```

### 10.2 ConfigBuilder 接口

```go
type ConfigBuilder interface {
    Build(config NeutralConfig) ([]byte, error)
    ValidateModel(config NeutralConfig) error
}
```

Xray:

```go
type XrayConfigBuilder struct{}

func (b *XrayConfigBuilder) Build(config NeutralConfig) ([]byte, error) {
    // NeutralConfig -> Xray JSON
}
```

sing-box:

```go
type SingBoxConfigBuilder struct{}

func (b *SingBoxConfigBuilder) Build(config NeutralConfig) ([]byte, error) {
    // NeutralConfig -> sing-box JSON
}
```

Hysteria2:

```go
type Hysteria2ConfigBuilder struct{}

func (b *Hysteria2ConfigBuilder) Build(config NeutralConfig) ([]byte, error) {
    // NeutralConfig -> Hysteria2 YAML
}
```

---

## 11. 配置校验

不同内核使用不同校验命令。

### 11.1 Xray

```text
xray -test -config config.json
```

或根据当前 Xray-core 实际命令封装。

### 11.2 sing-box

```text
sing-box check -c config.json
```

### 11.3 Hysteria 2

```text
hysteria server --config config.yaml --check
```

如果内核不支持纯 check 命令，则采用：

```text
1. 临时配置文件
2. 启动前 dry-run
3. 捕获 stderr
4. 超时退出
```

统一返回：

```go
type ConfigValidationResult struct {
    Valid   bool     `json:"valid"`
    Errors  []string `json:"errors"`
    Warnings []string `json:"warnings"`
    RawOutput string `json:"rawOutput"`
}
```

---

## 12. 进程管理设计

### 12.1 ProcessRunner

```go
type ProcessRunner interface {
    Start(cmd ProcessCommand) (*ProcessHandle, error)
    Stop(handle ProcessHandle) error
    Kill(handle ProcessHandle) error
    IsRunning(handle ProcessHandle) bool
}
```

命令结构：

```go
type ProcessCommand struct {
    Binary string
    Args   []string
    Env    []string
    WorkDir string
    Stdout io.Writer
    Stderr io.Writer
}
```

### 12.2 Supervisor

Supervisor 负责：

```text
- 启动进程
- 记录 PID
- 监听进程退出
- 异常退出状态更新
- 自动重启策略
- 优雅停止和超时 kill
```

自动重启策略：

```go
type RestartPolicy struct {
    Enabled bool
    MaxRetries int
    WindowSeconds int
    BackoffSeconds int
}
```

建议默认：

```text
Xray：允许自动重启
sing-box：允许自动重启
Hysteria2：允许自动重启
mihomo：视部署模式决定
```

---

## 13. 日志系统

### 13.1 统一日志行

```go
type CoreLogLine struct {
    Time      time.Time `json:"time"`
    CoreType  string    `json:"coreType"`
    InstanceID int64    `json:"instanceId"`
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    Raw       string    `json:"raw"`
}
```

### 13.2 LogParser 接口

```go
type LogParser interface {
    Parse(line string) CoreLogLine
}
```

Xray、sing-box、Hysteria2 分别实现自己的 parser。

### 13.3 安全要求

日志展示必须：

```text
- 后端不要返回未脱敏 token、私钥、订阅密钥
- 前端不要使用 v-html 渲染日志
- 日志下载需要鉴权
- 日志查询需要分页和数量限制
```

---

## 14. API 设计

新增 API 前缀：

```text
/panel/api/cores
```

### 14.1 内核类型和能力

```text
GET /panel/api/cores/types
GET /panel/api/cores/capabilities
GET /panel/api/cores/capabilities/:coreType
```

### 14.2 内核实例

```text
GET    /panel/api/cores/instances
POST   /panel/api/cores/instances
GET    /panel/api/cores/instances/:id
PUT    /panel/api/cores/instances/:id
DELETE /panel/api/cores/instances/:id
```

### 14.3 生命周期操作

```text
POST /panel/api/cores/instances/:id/start
POST /panel/api/cores/instances/:id/stop
POST /panel/api/cores/instances/:id/restart
POST /panel/api/cores/instances/:id/reload
POST /panel/api/cores/instances/:id/validate
```

### 14.4 配置

```text
GET  /panel/api/cores/instances/:id/config
POST /panel/api/cores/instances/:id/config/render
POST /panel/api/cores/instances/:id/config/apply
```

### 14.5 日志和状态

```text
GET /panel/api/cores/instances/:id/status
GET /panel/api/cores/instances/:id/logs
```

### 14.6 版本管理

```text
GET  /panel/api/cores/assets
GET  /panel/api/cores/assets/:coreType/versions
POST /panel/api/cores/assets/:coreType/install
POST /panel/api/cores/assets/:coreType/verify
```

---

## 15. 旧 API 兼容方案

为了避免破坏现有用户，旧 API 不要立即删除。

旧 API：

```text
/panel/api/server/restartXrayService
/panel/api/server/stopXrayService
/panel/api/server/installXray/:version
/panel/api/server/xraylogs/:count
```

内部转发：

```text
restartXrayService
  ↓
CoreManager.Restart(defaultXrayInstanceID)
```

旧 `InboundService` 也暂时保留，先对应默认 Xray 实例。

迁移完成后，再逐步新增：

```text
/panel/api/inbounds?coreInstanceId=xxx
```

---

## 16. 启动流程改造

当前启动流程应改为：

```text
1. InitDB
2. InitServiceContainer
3. InitCoreRegistry
4. InitCoreManager
5. Load auto_start core instances
6. Validate configs
7. Start enabled instances
8. Start Web Server
9. Start Sub Server
10. Start jobs
```

如果要保持原行为，也可以先启动 Web Server，再启动 CoreManager，但推荐让 CoreManager 成为应用初始化的一部分。

伪代码：

```go
func runWebServer() {
    database.InitDB(config.GetDBPath())

    container := service.NewContainer()
    registry := core.NewRegistry()
    registry.Register(xray.NewAdapter(container))
    registry.Register(singbox.NewAdapter(container))

    coreManager := core.NewManager(registry, container.CoreInstanceRepo)
    container.SetCoreManager(coreManager)

    coreManager.StartAutoStartInstances()

    server := web.NewServer(container)
    server.Start()
}
```

---

## 17. Service Container

建议新增服务容器，减少全局变量。

```go
type Container struct {
    DB *gorm.DB

    SettingService SettingService
    UserService    UserService
    InboundService InboundService

    CoreManager *core.CoreManager
    CoreRegistry *core.CoreRegistry

    EventBus EventBus
    Logger   Logger
}
```

Controller 初始化时注入：

```go
func NewCoreController(g *gin.RouterGroup, container *service.Container) *CoreController {
    return &CoreController{
        coreService: container.CoreService,
    }
}
```

---

## 18. 订阅系统改造

### 18.1 统一订阅节点

```go
type SubscriptionNode struct {
    Name      string
    CoreType  string
    Protocol  string
    Server    string
    Port      int
    UUID      string
    Password  string
    Transport string
    TLS       map[string]any
    Reality   map[string]any
    Extra     map[string]any
}
```

### 18.2 导出格式

支持：

```text
原始 URI：
- vless://
- vmess://
- trojan://
- ss://
- hysteria2://

客户端配置：
- sing-box JSON
- Clash/mihomo YAML
- Xray JSON
```

### 18.3 订阅生成流程

```text
proxy_inbounds + proxy_clients
  ↓
SubscriptionNode[]
  ↓
Exporter
  ↓
URI / YAML / JSON
```

Exporter 接口：

```go
type SubscriptionExporter interface {
    Name() string
    Export(nodes []SubscriptionNode, options ExportOptions) ([]byte, error)
}
```

---

## 19. 版本和二进制管理

### 19.1 下载流程

```text
1. 获取 release 信息
2. 选择 OS/Arch 对应 artifact
3. 下载到临时目录
4. 校验 SHA256
5. 解压
6. 写入 binary path
7. chmod 0755
8. 记录 core_assets
9. 执行 version 命令验证
```

### 19.2 禁止行为

```text
- 不要直接覆盖运行中的二进制
- 不要未校验 hash 就安装
- 不要让用户传任意 URL 下载并执行
- 不要让 web 输入直接拼接 shell 命令
```

---

## 20. 安全要求

多内核后必须增加以下安全措施：

```text
1. 所有状态变更 API 使用 POST/PUT/PATCH/DELETE
2. CSRF 使用 token，不单独信任 X-Requested-With
3. 配置文件权限 0600
4. 二进制目录权限 0750/0755
5. 日志脱敏
6. 下载文件大小限制
7. 解压路径 traversal 防护
8. 配置渲染后必须校验
9. 启动前必须检查端口冲突
10. 默认内核实例不允许空密码或默认 admin/admin
```

---

## 21. 测试方案

### 21.1 单元测试

```text
core registry
core manager
config builder
capability schema
port conflict
version parser
log parser
subscription exporter
```

### 21.2 集成测试

```text
Xray minimal config build
sing-box minimal config build
Hysteria2 minimal config build
config check command
start/stop/restart lifecycle
database migration
```

### 21.3 安全测试

```text
gosec
govulncheck
go test -race
路径穿越测试
命令注入测试
日志 XSS 回归测试
上传大小限制测试
```

### 21.4 CI 建议

```text
go test ./...
go test -race ./...
govulncheck ./...
gosec ./...
frontend typecheck
frontend lint
frontend build
docker build
```

---

## 22. 迁移方案

### 22.1 第一次迁移

新增默认 Xray 实例：

```text
core_instances:
  name: default-xray
  core_type: xray
  enabled: true
  auto_start: true
```

旧 inbound 数据暂不移动，仍由 XrayAdapter 读取旧表。

### 22.2 第二次迁移

新增 `proxy_inbounds` 和 `proxy_clients`，但不强制迁移旧数据。

新增转换工具：

```text
旧 Xray inbound
  ↓
proxy_inbounds + proxy_clients
```

用户可选择：

```text
保持旧模式
迁移到多内核模式
```

### 22.3 第三次迁移

默认使用统一模型生成 Xray 配置，旧 Xray inbound 表只保留兼容读取。

---

## 23. 实施排期

### 第 0 阶段：安全基线，1～3 天

```text
- 日志 v-html 改纯文本或严格转义
- CSRF token 化
- 默认密码和默认路径强制初始化
- Docker 生产模式安全加固
```

### 第 1 阶段：Core 抽象，3～5 天

```text
- Core 接口
- CoreRegistry
- CoreManager
- CoreInstance 表
- XrayAdapter
- 默认 Xray 实例
- 旧 API 转发到 CoreManager
```

### 第 2 阶段：sing-box MVP，7～14 天

```text
- sing-box 二进制管理
- sing-box ConfigBuilder
- sing-box check
- sing-box start/stop/restart
- sing-box logs
- sing-box 基础 inbound：
  - vless
  - trojan
  - shadowsocks
  - hysteria2
```

### 第 3 阶段：统一订阅，7～10 天

```text
- SubscriptionNode
- URI exporter
- sing-box JSON exporter
- mihomo YAML exporter
- 订阅绑定多内核实例
```

### 第 4 阶段：Hysteria2，7～10 天

```text
- Hysteria2Adapter
- Hysteria2 ConfigBuilder
- ACME/TLS
- bandwidth
- masquerade
- 日志解析
```

### 第 5 阶段：高级能力，2～4 周

```text
- 多实例监控
- 配置 diff
- 一键回滚
- 内核版本自动检测
- Release hash 校验
- 多内核 dashboard
```

---

## 24. 关键落地原则

1. **先抽象，再接入新内核。**
2. **不要把 sing-box 写成第二套 XrayService。**
3. **不要让 UI 硬编码每个内核字段。**
4. **不要破坏旧 Xray 用户。**
5. **所有内核都必须经过统一生命周期管理。**
6. **配置生成和配置校验必须分离。**
7. **订阅导出必须基于统一节点模型。**
8. **多内核不是多个二进制，而是一套可扩展平台能力。**

---

## 25. 最终推荐架构结论

推荐后端演进路线：

```text
当前：
XrayService + InboundService + ServerController

目标：
CoreManager
  ├── XrayAdapter
  ├── SingBoxAdapter
  ├── Hysteria2Adapter
  └── FutureAdapter

NeutralConfig
  ├── XrayConfigBuilder
  ├── SingBoxConfigBuilder
  └── Hysteria2ConfigBuilder

SubscriptionNode
  ├── URIExporter
  ├── SingBoxExporter
  ├── MihomoExporter
  └── XrayExporter
```

首期最小可用目标：

```text
1. 默认 Xray 实例不破坏旧功能
2. 能新增 sing-box 实例
3. 能启动/停止/重启 sing-box
4. 能生成并校验 sing-box 配置
5. 能在订阅里导出 sing-box/mihomo 客户端配置
```

完成以上改造后，SuperXray-gui 才真正具备从“Xray 面板”升级为“多内核代理面板”的底层能力。
