# 开发者贡献指南

> **目标读者**：贡献者
> **相关文档**：[系统架构设计](architecture.md) | [核心模块解析](modules.md) | [部署指南](deployment.md)

---

## 1. 项目简介

### 1.1 项目背景

**SuperXray** 是 [X-UI](https://github.com/vaxilu/x-ui) 项目的增强分支，是一个基于 Web 的 Xray-core 代理服务器管理面板。项目使用 Go 语言开发，采用 Gin Web 框架，前端使用 Vue.js + Ant Design Vue。

### 1.2 贡献方式

欢迎通过以下方式贡献：

- 🐛 **提交 Bug**：[GitHub Issues](https://github.com/superaddmin/SuperXray-gui/issues)
- 💡 **功能建议**：[GitHub Issues](https://github.com/superaddmin/SuperXray-gui/issues)
- 🔧 **代码贡献**：提交 Pull Request
- 🌍 **翻译贡献**：添加或改进翻译文件
- 📖 **文档改进**：完善项目文档

---

## 2. 开发环境搭建

### 2.1 Go 环境配置

本项目需要 Go 1.26 或更高版本（参见 `go.mod`）。Ubuntu 默认仓库中的 Go 版本可能过旧（Ubuntu 22.04 仅提供 Go 1.18，Ubuntu 24.04 仅提供 Go 1.22），建议通过以下方式安装：

#### 方法一：使用官方二进制包安装（推荐）

```bash
# 下载 Go 1.26.2 Linux amd64 版本
wget https://go.dev/dl/go1.26.2.linux-amd64.tar.gz

# 解压到 /usr/local
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.26.2.linux-amd64.tar.gz

# 将 Go 添加到 PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

> **注意**：如果使用 ARM64（aarch64）架构，请将下载链接中的 `linux-amd64` 替换为 `linux-arm64`。

#### 方法二：使用 go install（需要预先安装旧版 Go）

```bash
# 如果系统已安装 Go 1.22+，可直接升级
go install golang.org/dl/go1.26.2@latest
go1.26.2 download
```

#### 方法三：使用 Homebrew（macOS）

```bash
brew install go
```

#### 验证

安装完成后，运行以下命令确认版本：

```bash
go version
# 应输出: go version go1.26.2 linux/amd64
```

### 2.2 克隆与构建

```bash
# 克隆仓库
git clone https://github.com/superaddmin/SuperXray-gui.git
cd SuperXray-gui

# 安装依赖
go mod download

# 编译（需要 CGO，因为使用 SQLite）
CGO_ENABLED=1 go build -ldflags "-w -s" -o x-ui main.go
```

### 2.3 本地运行与调试

```bash
# 1. 创建 x-ui 目录（存放数据库和日志）
mkdir -p x-ui

# 2. 复制环境变量文件，并按本地开发覆盖路径
cp .env.example .env
cat >.env <<'EOF'
XUI_DEBUG=true
XUI_LOG_LEVEL=debug
XUI_DB_FOLDER=x-ui
XUI_LOG_FOLDER=x-ui
XUI_BIN_FOLDER=x-ui
EOF

# 3. 以调试模式运行
XUI_DEBUG=true go run main.go
```

**调试模式特性**：
- Gin 使用 `DebugMode`（输出详细路由信息）
- HTML 模板从本地文件系统加载（支持热更新）
- 静态资源从本地文件系统加载

### 2.4 环境变量配置

在项目根目录创建 `.env` 文件：

```bash
# .env
XUI_DEBUG=true           # 本地开发启用调试模式；生产环境应为 false
XUI_LOG_LEVEL=debug      # 本地调试日志级别；生产环境建议 info
XUI_DB_FOLDER=x-ui       # 本地数据库目录；生产环境建议 /etc/x-ui
XUI_LOG_FOLDER=x-ui      # 本地日志目录；生产环境建议 /var/log/x-ui
XUI_BIN_FOLDER=x-ui      # 本地 Xray 二进制目录；生产环境建议 bin
```

**默认账号**：首次运行时自动创建 `admin` / `admin`。

---

## 3. 项目结构说明

### 3.1 目录结构总览

```
3x-ui/
├── main.go                    # 程序入口，CLI 命令解析
├── go.mod / go.sum            # Go 模块定义与依赖锁定
├── Dockerfile                 # 多阶段 Docker 构建
├── docker-compose.yml         # Docker Compose 编排
├── install.sh                 # 一键安装脚本（约 990 行）
├── update.sh                  # 更新脚本
├── .env.example               # 环境变量示例
│
├── config/                    # 配置管理
│   ├── config.go              # 配置加载（版本/日志/路径）
│   ├── version                # 版本号：2.9.4
│   └── name                   # 应用名：x-ui
│
├── database/                  # 数据库层
│   ├── db.go                  # SQLite 初始化、迁移、种子
│   └── model/
│       ├── model.go           # 数据模型定义
│       └── model_test.go      # 模型测试
│
├── logger/                    # 日志系统
│   └── logger.go              # 双后端日志（控制台+文件）
│
├── web/                       # Web 层（核心）
│   ├── web.go                 # HTTP 服务器主体（507 行）
│   ├── controller/            # 控制器层（9 个文件）
│   │   ├── base.go            # 基础控制器
│   │   ├── index.go           # 首页/登录/登出
│   │   ├── xui.go             # 面板页面路由
│   │   ├── api.go             # API 路由组入口
│   │   ├── inbound.go         # Inbound CRUD（494 行）
│   │   ├── setting.go         # 面板设置
│   │   ├── xray_setting.go    # Xray 配置管理
│   │   ├── server.go          # 服务器管理（364 行）
│   │   ├── websocket.go       # WebSocket 连接
│   │   ├── custom_geo.go      # 自定义 Geo 资源
│   │   └── util.go            # 工具函数
│   ├── service/               # 业务逻辑层（14 个文件）
│   │   ├── setting.go         # 设置服务（859 行）
│   │   ├── inbound.go         # Inbound 服务（2804 行）
│   │   ├── server.go          # 服务器监控（1329 行）
│   │   ├── tgbot.go           # Telegram Bot（3823 行）
│   │   ├── xray.go            # Xray 进程管理
│   │   ├── xray_setting.go    # Xray 配置模板
│   │   ├── user.go            # 用户认证
│   │   ├── outbound.go        # 出站流量
│   │   ├── panel.go           # 面板重启
│   │   ├── warp.go            # Cloudflare WARP
│   │   ├── nord.go            # NordVPN
│   │   ├── custom_geo.go      # 自定义 Geo
│   │   └── config.json        # Xray 默认配置模板
│   ├── job/                   # 后台定时任务（10 个 Job）
│   ├── websocket/             # WebSocket Hub
│   │   ├── hub.go             # 消息广播中心
│   │   └── notifier.go        # 广播通知函数
│   ├── middleware/            # 中间件
│   │   ├── domainValidator.go # 域名验证
│   │   └── redirect.go        # URL 重定向
│   ├── network/               # 网络层
│   │   ├── auto_https_listener.go  # HTTPS 自动重定向
│   │   └── auto_https_conn.go
│   ├── entity/                # Web 层实体
│   │   └── entity.go          # Msg, AllSetting
│   ├── session/               # 会话管理
│   │   └── session.go         # Cookie Store
│   ├── global/                # 全局变量
│   │   ├── global.go          # WebServer/SubServer 接口
│   │   └── hashStorage.go     # SHA-256 哈希存储
│   ├── locale/                # 国际化系统
│   │   └── locale.go          # go-i18n 集成
│   ├── html/                  # HTML 模板
│   │   ├── index.html         # 首页
│   │   ├── inbounds.html      # Inbounds 管理页
│   │   ├── login.html         # 登录页
│   │   ├── settings.html      # 设置页
│   │   ├── xray.html          # Xray 配置页
│   │   ├── component/         # Vue 组件模板
│   │   ├── form/              # 表单模板
│   │   ├── modals/            # 模态框模板
│   │   └── settings/          # 设置子页面
│   ├── assets/                # 静态资源
│   │   ├── js/                # JavaScript 文件
│   │   │   ├── model/         # 前端数据模型
│   │   │   ├── util/          # 前端工具
│   │   │   ├── websocket.js   # WebSocket 客户端
│   │   │   └── subscription.js # 订阅管理
│   │   ├── vue/               # Vue.js
│   │   ├── ant-design-vue/    # Ant Design Vue
│   │   ├── codemirror/        # 代码编辑器
│   │   └── ...                # 其他第三方库
│   └── translation/           # 翻译文件（13 种语言 TOML）
│
├── sub/                       # 订阅服务
│   ├── sub.go                 # 订阅服务器主体
│   ├── subController.go       # 订阅控制器
│   ├── subService.go          # Base64 订阅（1484 行）
│   ├── subJsonService.go      # JSON 订阅
│   ├── subClashService.go     # Clash 订阅
│   └── default.json           # JSON 订阅默认配置
│
├── util/                      # 工具包
│   ├── crypto/crypto.go       # bcrypt 密码哈希
│   ├── ldap/ldap.go           # LDAP 认证
│   ├── random/random.go       # 随机数生成
│   ├── json_util/json.go      # 自定义 JSON 类型
│   ├── reflect_util/reflect.go # 反射工具
│   ├── common/                # 通用工具
│   │   ├── err.go             # 错误处理
│   │   ├── format.go          # 格式化
│   │   └── multi_error.go     # 多错误合并
│   └── sys/                   # 系统相关
│       ├── psutil.go          # 进程工具
│       ├── sys_linux.go       # Linux 特定
│       ├── sys_darwin.go      # macOS 特定
│       └── sys_windows.go     # Windows 特定
│
├── xray/                      # Xray 集成包
├── media/                     # 截图与资源图片
├── windows_files/             # Windows 支持文件
├── .github/                   # CI/CD 配置
│   ├── workflows/
│   │   ├── release.yml        # 发布构建
│   │   ├── docker.yml         # Docker 推送
│   │   ├── codeql.yml         # 安全分析
│   │   └── cleanup_caches.yml # 缓存清理
│   ├── ISSUE_TEMPLATE/        # Issue 模板
│   ├── dependabot.yml         # 依赖自动更新
│   └── FUNDING.yml            # 资助配置
│
├── docs/                      # 技术文档
│   ├── architecture.md        # 系统架构设计
│   ├── deployment.md          # 部署指南
│   ├── modules.md             # 核心模块解析
│   ├── api.md                 # API 接口说明
│   └── development.md         # 本文档
│
└── plans/                     # 规划文档
    └── documentation-plan.md  # 文档体系规划
```

---

## 4. 开发规范

### 4.1 代码风格

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 规范
- 使用 `gofmt` 格式化代码
- 每个包添加包注释
- 导出函数添加文档注释

### 4.2 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 包名 | 小写，简短，无下划线 | `controller`, `service`, `model` |
| 文件名 | 小写，下划线分隔 | `check_client_ip_job.go` |
| 结构体 | 大驼峰 | `InboundController`, `XrayService` |
| 方法 | 大驼峰（导出）/ 小驼峰（私有） | `GetInbound()`, `addTraffic()` |
| 常量 | 大驼峰 / 全大写 | `Protocol`, `Hysteria` |
| 接口 | 大驼峰，常以 `-er` 结尾或以 `Service` 结尾 | `Tgbot`, `XrayService` |

### 4.3 错误处理

```go
// 推荐：检查错误并记录日志
inbound, err := a.inboundService.GetInbound(id)
if err != nil {
    jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.obtain"), err)
    return
}

// 推荐：使用 common.Combine 合并多个错误
return common.Combine(err1, err2)
```

### 4.4 日志规范

```go
// 使用 logger 包
logger.Info("Web server running HTTPS on", listener.Addr())
logger.Warning("start xray failed:", err)
logger.Error("restart xray failed:", err)
logger.Debug("Error stopping web server:", err)
```

---

## 5. 测试指南

### 5.1 运行测试

```bash
# 运行所有测试
go test ./...

# 运行指定包的测试
go test ./database/model/...
go test ./web/service/...
go test ./web/job/...

# 查看详细输出
go test -v ./...

# 运行指定测试函数
go test -run TestIsHysteria ./database/model/...
```

### 5.2 编写测试

现有测试文件分布：

| 文件 | 测试数 | 测试内容 |
|------|--------|---------|
| [`database/model/model_test.go`](../database/model/model_test.go) | 1 | `IsHysteria` 协议判断 |
| [`web/job/check_client_ip_job_test.go`](../web/job/check_client_ip_job_test.go) | 7 | IP 合并/过期/分区逻辑 |
| [`web/job/check_client_ip_job_integration_test.go`](../web/job/check_client_ip_job_integration_test.go) | 2 | IP 限制集成测试 |
| [`web/service/custom_geo_test.go`](../web/service/custom_geo_test.go) | 13 | Geo 资源验证/下载/修复 |
| [`web/service/xray_setting_test.go`](../web/service/xray_setting_test.go) | 1 | Xray 配置模板解包 |

**测试编写示例**：

```go
package model

import "testing"

func TestIsHysteria(t *testing.T) {
    tests := []struct {
        name     string
        protocol Protocol
        want     bool
    }{
        {"hysteria v1", Hysteria, true},
        {"hysteria v2", Hysteria2, true},
        {"vmess", VMESS, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := IsHysteria(tt.protocol); got != tt.want {
                t.Errorf("IsHysteria(%v) = %v, want %v", tt.protocol, got, tt.want)
            }
        })
    }
}
```

### 5.3 测试覆盖率

```bash
# 生成覆盖率报告
go test -cover ./...

# 生成详细覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

## 6. 国际化贡献

### 6.1 翻译文件格式

翻译文件使用 TOML 格式，位于 [`web/translation/`](../web/translation/) 目录：

```toml
# web/translation/translate.zh_CN.toml

[menu]
"menu.dashboard" = "仪表盘"
"menu.inbounds" = "入站列表"
"menu.settings" = "面板设置"
"menu.xray" = "Xray 配置"

[pages]
"pages.inbounds.toasts.obtain" = "获取入站信息"
"pages.inbounds.toasts.add" = "添加入站"
```

### 6.2 添加新语言

1. 复制 `translate.en_US.toml` 为新语言文件（如 `translate.fr_FR.toml`）
2. 翻译所有键值对
3. 在 [`web/locale/locale.go`](../web/locale/locale.go) 中注册新语言
4. 提交 Pull Request

### 6.3 更新翻译

1. 修改对应的 TOML 文件
2. 确保所有键与 `translate.en_US.toml` 保持一致
3. 提交 Pull Request

---

## 7. 提交规范

### 7.1 Commit Message 格式

```
<type>(<scope>): <subject>

<body>
```

**Type 类型**：

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | 修复 Bug |
| `docs` | 文档变更 |
| `style` | 代码格式（不影响功能） |
| `refactor` | 重构 |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 构建工具/依赖变更 |

**示例**：

```
feat(inbound): add support for Hysteria2 protocol
fix(telegram): resolve bot 409 conflict on restart
docs(api): update API documentation for inbound endpoints
```

### 7.2 PR 提交流程

1. Fork 仓库
2. 创建功能分支（`git checkout -b feat/my-feature`）
3. 提交变更（遵循 Commit Message 格式）
4. 推送到 Fork 仓库
5. 创建 Pull Request 到 `master` 分支
6. 等待 Code Review

### 7.3 Code Review 要求

- 代码风格符合项目规范
- 新功能需要包含测试
- 不引入新的 lint 警告
- 文档同步更新

---

## 8. 发布流程

### 8.1 版本号规范

项目使用 [语义化版本](https://semver.org/lang/zh-CN/)：

```
主版本号.次版本号.修订号
2.9.4
```

版本号存储在 [`config/version`](../config/version) 文件中。

### 8.2 CI/CD 流程

项目使用 GitHub Actions 进行持续集成/部署：

| 工作流 | 文件 | 触发条件 | 功能 |
|--------|------|---------|------|
| Release | [`release.yml`](../.github/workflows/release.yml) | Tag 推送 | 构建多平台二进制 + 发布 |
| Docker | [`docker.yml`](../.github/workflows/docker.yml) | Tag 推送 | 构建 Docker 镜像 + 推送 |
| CodeQL | [`codeql.yml`](../.github/workflows/codeql.yml) | PR/Push | 代码安全分析 |
| Cache Cleanup | [`cleanup_caches.yml`](../.github/workflows/cleanup_caches.yml) | PR 关闭 | 清理 CI 缓存 |

### 8.3 Docker 镜像构建

[`Dockerfile`](../Dockerfile) 使用多阶段构建：

```dockerfile
# Stage 1: Builder
FROM golang:1.26-alpine AS builder
# 编译 Go 二进制 + 下载 Xray + GeoIP 数据

# Stage 2: Final Image
FROM alpine
# 复制二进制 + 配置 fail2ban + 设置入口点
```

**构建命令**：

```bash
# 本地构建 Docker 镜像
docker build -t 3x-ui .

# 多架构构建
docker buildx build --platform linux/amd64,linux/arm64 -t 3x-ui .
```
