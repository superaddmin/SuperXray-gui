# CHANGELOG

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与语义化版本号。

## [2.9.6] - 2026-04-29

### Fixed

- 修复 GitHub Release tag 构建时 release notes 无法匹配带日期 CHANGELOG 标题的问题。
- 修复 Docker tag 构建中 `DockerInit.sh` 未设置执行权限导致 buildx 退出 126 的问题。

### Changed

- 将项目版本号更新为 `2.9.6`，并同步文档中的版本引用。

## [2.9.5] - 2026-04-29

### Fixed

- 修复 Docker tag 构建失败：移除默认发布路径对 Docker Hub 凭据的依赖，仅发布小写 GHCR 镜像。
- 修复面板侧栏在非根路径部署时因 `themeSwitcher` 与 logo 路径导致的渲染中断。

### Changed

- 将项目版本号更新为 `2.9.5`，并同步文档中的版本引用。

## [2.9.4] - 2026-04-29

### Added

- 增加 Ubuntu 服务器部署说明，覆盖 amd64 与 arm64 架构、Release 包安装、源码构建、证书配置和常见故障排查。
- 增加开发文档、模块说明和多语言 README/翻译配置校验说明。

### Changed

- 将服务器安装脚本和面板管理脚本的高频交互提示中文化，覆盖首次安装、证书配置、服务启停、Fail2ban/IP 限制和常见状态提示。
- 将 `x-ui setting` 命令的安装期成功/失败输出中文化。
- 将项目版本号更新为 `2.9.4`，并同步文档中的版本引用。

### Fixed

- 修复 Release 资源命名与安装脚本下载路径不一致导致的服务器安装失败风险。
- 修复路径根封装、错误处理、配置 JSON 拼接和前端日志显示相关安全问题。
- 修复 ARM64 构建与验证流程中的 CGO 编译环境约束。

### Security

- 强化 Web 服务安全基线，包括安全响应头、可信代理、Host 校验、限流和敏感配置处理。
- 增加 DOM XSS 回归验证和 Go race 测试验证路径。
