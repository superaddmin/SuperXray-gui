# 3X-UI 技术文档体系规划方案

## 1. 文档目录结构

```
f:/3x-ui/
├── docs/
│   ├── architecture.md        # 系统架构设计文档
│   ├── deployment.md          # 环境搭建与部署指南
│   ├── modules.md             # 核心模块解析文档
│   ├── api.md                 # API 接口说明文档
│   └── development.md         # 开发者贡献指南
└── README.zh_CN.md            # 重构后的中文 README
```

## 2. 各文档详细章节大纲

---

### 2.1 `docs/architecture.md` — 系统架构设计

**目标读者**：开发者 / 架构师  
**预计篇幅**：约 400-500 行

```
# 系统架构设计

## 1. 项目概述
   - 1.1 项目定位与背景
   - 1.2 核心功能概览
   - 1.3 技术栈选型

## 2. 系统架构总览
   - 2.1 整体架构图（Mermaid）
   - 2.2 双服务器架构说明
   - 2.3 进程模型与通信方式

## 3. 分层架构设计
   - 3.1 入口层（main.go CLI）
   - 3.2 Web 层（Controller → Service → Database）
   - 3.3 订阅服务层（sub/）
   - 3.4 后台任务层（job/）
   - 3.5 基础设施层（config/database/logger/util）

## 4. 核心数据流
   - 4.1 请求处理流
   - 4.2 流量采集与推送流
   - 4.3 客户端 IP 监控流
   - 4.4 订阅请求流
   - 4.5 认证与安全流

## 5. 数据模型设计
   - 5.1 ER 图
   - 5.2 核心模型说明（User/Inbound/Client/Setting/CustomGeoResource）

## 6. 实时通信架构
   - 6.1 WebSocket Hub 设计
   - 6.2 消息类型与广播机制
   - 6.3 Worker Pool 并发模型

## 7. 安全架构
   - 7.1 认证机制（本地/LDAP/2FA）
   - 7.2 域名验证中间件
   - 7.3 fail2ban 集成
   - 7.4 路径遍历与 SSRF 防护

## 8. 国际化架构
   - 8.1 翻译文件组织
   - 8.2 go-i18n 集成方式
```

---

### 2.2 `docs/deployment.md` — 环境搭建与部署指南

**目标读者**：运维人员 / 开发者  
**预计篇幅**：约 300-400 行

```
# 环境搭建与部署指南

## 1. 系统要求
   - 1.1 硬件要求
   - 1.2 操作系统支持
   - 1.3 网络要求

## 2. Docker 部署（推荐）
   - 2.1 快速启动
   - 2.2 docker-compose.yml 详解
   - 2.3 环境变量配置
   - 2.4 数据持久化（卷挂载）
   - 2.5 SSL 证书配置
   - 2.6 fail2ban 配置

## 3. 一键脚本安装
   - 3.1 安装命令
   - 3.2 支持的操作系统
   - 3.3 自定义安装路径
   - 3.4 升级与卸载

## 4. 手动构建与部署
   - 4.1 从源码构建
   - 4.2 Go 环境准备
   - 4.3 编译步骤
   - 4.4 systemd 服务配置
   - 4.5 OpenRC 服务配置（Arch Linux）

## 5. 配置说明
   - 5.1 环境变量一览
   - 5.2 CLI 参数说明
   - 5.3 Web 面板设置
   - 5.4 订阅服务配置
   - 5.5 Telegram Bot 配置
   - 5.6 LDAP 配置

## 6. 反向代理配置
   - 6.1 Nginx 反向代理
   - 6.2 Caddy 反向代理
   - 6.3 Cloudflare CDN 配置

## 7. 常见问题排查
   - 7.1 端口冲突
   - 7.2 证书问题
   - 7.3 权限问题
   - 7.4 Xray 启动失败
```

---

### 2.3 `docs/modules.md` — 核心模块解析

**目标读者**：开发者  
**预计篇幅**：约 500-600 行

```
# 核心模块解析

## 1. 程序入口（main.go）
   - 1.1 CLI 命令解析
   - 1.2 启动流程
   - 1.3 信号处理

## 2. 配置管理（config/）
   - 2.1 配置加载机制
   - 2.2 环境变量映射
   - 2.3 版本管理

## 3. 数据库层（database/）
   - 3.1 初始化与迁移
   - 3.2 数据模型详解
   - 3.3 种子数据

## 4. Web 服务器（web/）
   - 4.1 服务器启动流程
   - 4.2 路由注册
   - 4.3 中间件链
   - 4.4 模板渲染

## 5. 控制器层（web/controller/）
   - 5.1 BaseController
   - 5.2 IndexController
   - 5.3 InboundController
   - 5.4 SettingController
   - 5.5 XraySettingController
   - 5.6 ServerController
   - 5.7 APIController
   - 5.8 WebSocketController
   - 5.9 CustomGeoController

## 6. 服务层（web/service/）
   - 6.1 SettingService
   - 6.2 InboundService
   - 6.3 XrayService
   - 6.4 ServerService
   - 6.5 UserService
   - 6.6 TgbotService
   - 6.7 OutboundService
   - 6.8 WarpService / NordService
   - 6.9 CustomGeoService

## 7. 后台任务（web/job/）
   - 7.1 Cron 调度系统
   - 7.2 各 Job 详解

## 8. WebSocket 实时通信（web/websocket/）
   - 8.1 Hub 架构
   - 8.2 消息广播
   - 8.3 并发控制

## 9. 订阅服务（sub/）
   - 9.1 订阅服务器架构
   - 9.2 Base64 订阅
   - 9.3 JSON 订阅
   - 9.4 Clash/Mihomo 订阅

## 10. 工具包（util/）
   - 10.1 crypto - 密码哈希
   - 10.2 ldap - LDAP 认证
   - 10.3 random - 随机数生成
   - 10.4 sys - 系统信息

## 11. 日志系统（logger/）
   - 11.1 双后端日志
   - 11.2 内存缓冲
```

---

### 2.4 `docs/api.md` — API 接口说明

**目标读者**：集成开发者  
**预计篇幅**：约 400-500 行

```
# API 接口说明

## 1. 概述
   - 1.1 认证方式
   - 1.2 请求/响应格式
   - 1.3 错误码定义

## 2. 认证接口
   - POST /login
   - POST /getTwoFactorEnable
   - GET /logout

## 3. 面板页面路由
   - GET /panel/
   - GET /panel/inbounds
   - GET /panel/settings
   - GET /panel/xray

## 4. 设置管理 API
   - POST /panel/setting/all
   - POST /panel/setting/defaultSettings
   - POST /panel/setting/update
   - POST /panel/setting/updateUser
   - POST /panel/setting/restartPanel
   - GET /panel/setting/getDefaultJsonConfig

## 5. Xray 配置 API
   - POST /panel/xray/
   - POST /panel/xray/update
   - GET /panel/xray/getDefaultJsonConfig
   - GET /panel/xray/getOutboundsTraffic
   - POST /panel/xray/resetOutboundsTraffic
   - POST /panel/xray/testOutbound
   - WARP 操作 API
   - NordVPN 操作 API

## 6. Inbound 管理 API
   - CRUD 操作
   - 客户端管理
   - 流量管理
   - IP 管理
   - 在线状态

## 7. 服务器管理 API
   - 状态查询
   - Xray 管理
   - 日志查询
   - 数据库导入导出
   - 证书/密钥生成

## 8. 自定义 Geo 资源 API
   - CRUD 操作
   - 下载/刷新

## 9. WebSocket 接口
   - 连接方式
   - 消息类型定义
   - 实时数据格式

## 10. 订阅服务 API
   - GET /sub/:subid
   - GET /json/:subid
   - GET /clash/:subid
```

---

### 2.5 `docs/development.md` — 开发者贡献指南

**目标读者**：贡献者  
**预计篇幅**：约 300-400 行

```
# 开发者贡献指南

## 1. 项目简介
   - 1.1 项目背景
   - 1.2 贡献方式

## 2. 开发环境搭建
   - 2.1 Go 环境配置
   - 2.2 克隆与构建
   - 2.3 本地运行与调试
   - 2.4 环境变量配置

## 3. 项目结构说明
   - 3.1 目录结构总览
   - 3.2 各目录职责

## 4. 开发规范
   - 4.1 代码风格
   - 4.2 命名规范
   - 4.3 错误处理
   - 4.4 日志规范

## 5. 测试指南
   - 5.1 运行测试
   - 5.2 编写测试
   - 5.3 测试覆盖率

## 6. 国际化贡献
   - 6.1 翻译文件格式
   - 6.2 添加新语言
   - 6.3 更新翻译

## 7. 提交规范
   - 7.1 Commit Message 格式
   - 7.2 PR 提交流程
   - 7.3 Code Review 要求

## 8. 发布流程
   - 8.1 版本号规范
   - 8.2 CI/CD 流程
   - 8.3 Docker 镜像构建
```

---

### 2.6 `README.zh_CN.md` — 重构后的中文 README

**目标读者**：所有用户  
**预计篇幅**：约 400-500 行

```
# 3X-UI 中文 README

## 项目 Logo + 徽章
## 项目简介
## 功能特性
## 快速开始
## 截图预览
## 安装方式
   - 一键脚本安装
   - Docker 安装
   - 手动构建
## 基本使用
   - 登录
   - 添加 Inbound
   - 客户端配置
   - 订阅链接
## 配置说明
   - 面板设置
   - Xray 配置
   - Telegram Bot
   - 订阅服务
## 目录结构
## 技术栈
## 本地开发
## 常见问题
## 致谢与支持
## 许可证
```

## 3. 文档间交叉引用关系

```
README.zh_CN.md ──→ docs/architecture.md（了解系统设计）
       │
       ├──→ docs/deployment.md（部署指南）
       │
       └──→ docs/development.md（参与贡献）

docs/architecture.md ──→ docs/modules.md（模块详情）
       │
       └──→ docs/api.md（接口详情）

docs/modules.md ──→ docs/api.md（对应API）
       │
       └──→ docs/architecture.md（架构背景）

docs/api.md ──→ docs/modules.md（实现细节）

docs/deployment.md ──→ docs/api.md（API验证）
       │
       └──→ docs/development.md（本地开发）

docs/development.md ──→ docs/modules.md（模块理解）
       │
       └──→ docs/architecture.md（架构理解）
```

## 4. 写作注意事项

1. **所有文档使用中文撰写**，技术术语保留英文原文
2. **代码引用**使用相对路径链接，如 `[main.go](../main.go)`
3. **Mermaid 图表**避免在方括号内使用双引号和圆括号
4. **API 文档**需包含完整的请求/响应示例
5. **架构文档**需包含清晰的 Mermaid 流程图
6. **文档头部**统一包含：标题、目标读者、最后更新时间
7. **交叉引用**使用相对路径链接，确保在 GitHub 上可点击
