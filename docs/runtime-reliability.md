# 请求指标、Geo 更新与流量统计可靠性

> 源码核对日期：2026-10-14。事实来源：`web/web.go`、`sub/sub.go`、`web/middleware/request_log.go`、`web/service/metrics.go`、`web/service/geofile.go`、`web/service/inbound.go`、`web/service/outbound.go`、`web/job/xray_traffic_job.go` 及其回归测试。

本文记录请求观测、HTTP 超时、定时任务、Geo 更新与流量统计的当前行为和验证方式。

## 请求指标

公开 `GET <webBasePath>api/v1/metrics` 返回 `APIResponse.data` 下的进程内请求快照；live/ready 也带同结构指标。请求计数覆盖 Web 面板服务，不覆盖独立 Sub Server；面板重启后计数清零。指标请求生成响应时仍属于 in-flight，完成后才累计到总数。

- 未匹配路由统一记录为 `<unmatched>`，请求日志和指标不再保留未知 URL 路径。
- 路由维度最多 256 项，状态维度最多 64 项，均包含 `<overflow>` 桶。超过上限的新维度汇总到该桶，已存在维度继续计数。
- method 和 route 标签分别限制为 32 和 512 字节，超长标签使用固定占位符。

验证：连续请求两个不同的不存在路径，应均返回 404；指标中 `GET <unmatched>` 增长，两个原始路径不应出现。不要将路由总数固定为某个值，正常业务请求也会增加已有桶的计数。

## 默认 Geo 文件更新

更新入口仅接受内置六个文件名或全部更新，并串行执行。下载先写入同目录唯一临时文件，限制为 128 MiB，完成同步和关闭后验证 protobuf 内容。整个批次下载、验证成功才替换文件；任何下载失败均保留原有批次。全部返回 304 时不重启核心。

替换前创建恢复副本。替换后先用数据库生成的实际 Xray 配置执行 `xray run -test -config`，通过才重启；预检失败恢复原文件且不重启。同步重启失败时恢复原文件，再尝试启动旧配置。恢复失败会保留 backup 文件，并在错误中指出恢复位置。备份在重启成功后才清理。

这不是断电或进程强制退出情况下的持久化批次事务。默认 Geo 更新锁也不覆盖 Xray 安装或其他资源更新。Custom Geo 更新流程不属于本次变更。下载仍为同步请求，前端等待超时不代表服务器已经停止更新；应先检查后台结果再重试。

## 流量统计

入站及客户端更新使用同一个 SQLite/GORM 事务，出站使用独立事务。查询、保存和 commit 错误向调用方传播；业务失败时整个对应事务回滚。事务中已发生的 Xray 副作用会保留重新同步运行态的标志。

后台任务按入站 tag、客户端 email 和出站 tag 合并未落库增量。仅在对应事务成功后清除增量，另一侧失败不会使成功侧重复计数。核心停止或采集失败时仍重试已收集的数据；设置重启标志前重新检查核心运行状态。

**待重试增量仅保存在面板进程内存中。** 面板进程退出时可能丢失尚未落库的数据；本次修复不提供持久化 exactly-once 计费保证。

## HTTP 超时与定时任务

Web Server 与独立 Sub Server 的 `newHTTPServer` 都使用以下限制；反向代理或客户端可另设更短超时。

| 配置 | 当前值 |
| --- | --- |
| ReadHeaderTimeout | 5 秒 |
| ReadTimeout | 30 秒 |
| WriteTimeout | 60 秒 |
| IdleTimeout | 120 秒 |
| MaxHeaderBytes | 512 KiB |

Web cron 使用配置时区、秒字段解析和 `cron.SkipIfStillRunning`。同一个调度项上次仍在执行时跳过本次触发，不排队补跑；不同任务仍可同时运行。`XrayTrafficJob` 自身还有互斥锁，Geo 默认更新另有批次锁。

启动初始化先尝试重启 Xray，然后注册每秒运行检查与每 30 秒重启标志检查。流量采集任务在延迟 5 秒后注册 `@every 10s`，因此不是承诺启动第 5 秒已完成第一次采集。客户端 IP 检查每 10 秒、日志清理每日、流量自动重置按小时/日/周/月调度；服务器状态由控制器每 2 秒刷新。

LDAP 同步仅在启用 LDAP 时注册，缺失计划默认 `@every 1m`。TG 统计通知仅在启用 Bot 时注册；非法或空计划按实现记录警告/回退，CPU 告警还要求阈值大于 0。关闭这些功能不会注册对应功能任务。调度注册失败会记录警告；HTTP 成功不能证明定时任务或外部通知已经成功。

## 本地验证

```powershell
go test ./web/middleware ./web/service ./web/job ./web/controller
go test -race ./web/middleware ./web/service ./web/job
$pkgs = @(go list ./... | Where-Object { $_ -notmatch '/node_modules/' })
go test $pkgs
go vet $pkgs
python scripts/secret_scan.py
git diff --check
```

回归测试覆盖指标上限、Geo 下载失败和回滚、预检及重启失败、事务保存和 commit 失败、入站和出站独立重试、停止状态与无效增量。Linux 交付需要启用 CGO 并使用对应架构编译器，沿用项目 release workflow 的构建方式。

## 上线与回滚

1. 核对当前二进制哈希；备份二进制、配置、systemd 单元及 SQLite 数据库。运行中的数据库使用 SQLite backup API 创建一致性快照，并验证 `PRAGMA quick_check`。备份目录使用 `0700`，数据库及敏感配置使用 `0600`。
2. 上传到同目录 staging 文件，核对 SHA-256，确认版本、动态库兼容性和当前 Xray 配置检查通过，再原子替换二进制并重启面板。
3. 验证 ready、Xray 进程、登录页、实际 Host 下的订阅输出、404 指标归并、数据库完整性和后续流量写入；检查最近日志。
4. 启动或健康检查失败时恢复旧二进制并重启。仅回滚程序时不要恢复旧数据库，以免覆盖升级后的流量和配置。

公网面板和订阅应使用有效证书的 HTTPS，或按实际访问需求限制到 SSH 隧道。变更监听地址前必须确认客户端入口。住宅出口需要真实上游，并从实际业务路径验证最终出口；回环 SOCKS 转发到 `freedom` 仍然使用本机出口，不能视为住宅代理。HTTPS 域名/证书和真实代理参数未确定时，先记录待配置项，避免编造上游或中断已有客户端。
