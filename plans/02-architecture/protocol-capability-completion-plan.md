# 协议周边能力补齐执行计划

> 执行顺序：P0 -> P1 -> P2 -> P3。每一阶段都先补回归测试，再补实现，最后更新文档或矩阵。

## 目标

补齐现有协议的校验、订阅、兼容、安全和工具链能力，让面板支持范围从“能配置”推进到“能稳定创建、订阅导出、被测试覆盖和被文档解释”。

## P0：主力协议正确性与安全校验

范围：

- `vmess` / `vless`：客户端 UUID 必须有效。
- `vless`：`flow` 只能在 TCP + TLS/Reality 组合下使用。
- `trojan`：客户端 password 必填。
- `shadowsocks`：method 必填；legacy password 必填；2022 server/client key 必须是匹配长度的 base64。
- `hysteria` / `hysteria2`：客户端 auth 必填。

验证：

- 新增 `web/service/protocol_validation_test.go`。
- 运行 `go test ./web/service`。

## P1：WireGuard 订阅与导出周边能力

范围：

- WireGuard peer 增加订阅元数据：`email`、`enable`、`subId`。
- 订阅发现支持 `settings.peers[*].subId`。
- 普通订阅输出 WireGuard `.conf`。
- JSON 订阅输出 Xray WireGuard outbound。
- Clash 订阅输出 Mihomo WireGuard proxy。
- 前端 WireGuard peer 表单展示并保存订阅元数据。

验证：

- 新增 `sub/wireguard_subscription_test.go`。
- 更新 `web/assets/js/model/inbound.test.js`。
- 运行 `go test ./sub` 与 `node --test web/assets/js/model/inbound.test.js`。

## P2：兼容性、安全与性能小闭环

范围：

- 补齐 Go 协议常量中前端已支持的 `tun`。
- 订阅生成避免 WireGuard 缺失 peer 或禁用 peer 时产生空配置。
- WireGuard key、allowedIPs、endpoint 输出做最小安全校验，避免无效配置流入订阅。

验证：

- WireGuard 测试覆盖禁用 peer、缺 key、缺 allowedIPs 的跳过行为。
- 运行 `go test ./sub ./web/service`。

## P3：协议矩阵与文档同步

范围：

- 新增协议能力矩阵测试，固定入站协议、订阅协议、JSON/Clash 覆盖范围。
- 更新 `docs/inbound-creation-guide.md`，补充 WireGuard 订阅元数据和协议校验说明。

验证：

- 新增 `sub/protocol_capability_test.go`。
- 运行 `go test ./sub`。
- 运行 `git diff --check`。
