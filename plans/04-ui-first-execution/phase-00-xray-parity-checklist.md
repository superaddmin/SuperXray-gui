# UI First Phase 0 - Xray Parity Checklist

> 目标：冻结旧 UI 的 Xray 行为边界，作为 Vue 3 新 UI 迁移的验收基线。

## 阶段边界

- 当前阶段：Phase 0，现状冻结与验收基线。
- 允许改动：文档、测试脚本、E2E 基线、项目级代理/技能说明。
- 禁止改动：`model.Inbound` 数据模型、Xray 生命周期、CoreManager 写路径、sing-box UI 或生命周期、旧 `/panel` 行为。
- 回退方式：删除本阶段新增的文档与 E2E 文件即可，运行时代码无行为变化。

## 旧入口盘点

| 能力域             | 旧页面/入口              | 主要文件                                                                       | 主要 API/路由                                                                                            | 读写属性          | 新 UI 暂定位置              |
| ------------------ | ------------------------ | ------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- | ----------------- | --------------------------- |
| 登录/登出          | `/`, `/login`, `/logout` | `web/html/login.html`, `web/controller/index.go`                               | `POST /login`, `POST /getTwoFactorEnable`, `GET /logout`                                                 | 写 session        | `/ui/login` 或复用旧入口    |
| Dashboard          | `/panel/`                | `web/html/index.html`, `web/controller/server.go`                              | `/panel/api/server/status`, `/cpuHistory/:bucket`, `/getXrayVersion`, `/getConfigJson`                   | 只读为主          | `/ui/dashboard`             |
| Xray 控制          | `/panel/`                | `web/html/index.html`, `web/controller/server.go`                              | `POST /stopXrayService`, `POST /restartXrayService`, `POST /installXray/:version`, `POST /updateGeofile` | 写入/危险操作     | `/ui/dashboard`             |
| 面板日志           | `/panel/` 日志弹窗       | `web/html/index.html`, `web/controller/server.go`                              | `POST /logs/:count`                                                                                      | 只读              | `/ui/logs/panel`            |
| Xray 日志          | `/panel/` Xray 日志弹窗  | `web/html/index.html`, `web/controller/server.go`                              | `POST /xraylogs/:count`                                                                                  | 只读              | `/ui/logs/xray`             |
| 入站管理           | `/panel/inbounds`        | `web/html/inbounds.html`, `web/controller/inbound.go`                          | `/panel/api/inbounds/list`, `/add`, `/update/:id`, `/del/:id`, `/import`                                 | 读写              | `/ui/inbounds`              |
| 客户端管理         | `/panel/inbounds`        | `web/html/modals/client_modal.html`, `web/controller/inbound.go`               | `/addClient`, `/updateClient/:clientId`, `/:id/delClient/:clientId`, `/:id/resetClientTraffic/:email`    | 读写              | `/ui/inbounds/:id/clients`  |
| 在线/最后在线      | `/panel/inbounds`        | `web/html/inbounds.html`, `web/controller/inbound.go`                          | `POST /onlines`, `POST /lastOnline`, `POST /clientIps/:email`                                            | 只读/清理写入     | `/ui/inbounds/activity`     |
| 分享/二维码        | `/panel/inbounds` 弹窗   | `web/html/modals/qrcode_modal.html`, `web/html/modals/inbound_info_modal.html` | 依赖入站列表、默认设置、订阅 URL                                                                         | 只读              | `/ui/inbounds/share`        |
| Xray 配置模板      | `/panel/xray`            | `web/html/xray.html`, `web/controller/xray_setting.go`                         | `POST /panel/xray/`, `POST /panel/xray/update`, `GET /getDefaultJsonConfig`                              | 读写              | `/ui/xray/config`           |
| Xray 出站/路由工具 | `/panel/xray`            | `web/html/settings/xray/*`, `web/html/modals/xray_*`                           | `POST /testOutbound`, `POST /resetOutboundsTraffic`, `GET /getOutboundsTraffic`                          | 读写              | `/ui/xray/outbounds`        |
| Warp/Nord 工具     | `/panel/xray`            | `web/html/modals/warp_modal.html`, `web/html/modals/nord_modal.html`           | `POST /panel/xray/warp/:action`, `POST /panel/xray/nord/:action`                                         | 写入外部凭据/配置 | `/ui/xray/tools`            |
| 设置               | `/panel/settings`        | `web/html/settings.html`, `web/controller/setting.go`                          | `POST /panel/setting/all`, `/update`, `/updateUser`, `/restartPanel`                                     | 读写              | `/ui/settings`              |
| 订阅配置           | `/panel/settings`        | `web/html/settings/panel/subscription/*`                                       | 设置 API 读写，Sub Server 输出 `/sub/:subid`, `/json/:subid`, `/clash/:subid`                            | 读写配置/只读输出 | `/ui/settings/subscription` |
| 自定义 Geo         | `/panel/`                | `web/html/index.html`, `web/controller/custom_geo.go`                          | `/panel/api/custom-geo/list`, `/add`, `/update/:id`, `/delete/:id`, `/download/:id`, `/update-all`       | 读写/下载         | `/ui/resources/geo`         |
| 备份/导入 DB       | `/panel/`                | `web/html/index.html`, `web/controller/server.go`                              | `GET /getDb`, `POST /importDB`                                                                           | 下载/危险写入     | `/ui/maintenance/backup`    |
| 实时状态           | 全局                     | `web/websocket/*`, `web/controller/websocket.go`                               | `GET /ws`                                                                                                | 只读推送          | 新 UI 状态通道              |

## Xray 等价验收清单

| 优先级 | 操作                | 旧入口                  | 旧 API                                                          | 新 UI 验收要求                                        | E2E 基线                        | 回退方式               |
| ------ | ------------------- | ----------------------- | --------------------------------------------------------------- | ----------------------------------------------------- | ------------------------------- | ---------------------- |
| P0     | 登录并进入面板      | `/` -> `/panel/`        | `POST /login`                                                   | 新 UI 登录成功后能读取旧 session API                  | `legacy-panel.spec.ts` 默认执行 | 回旧 `/panel/`         |
| P0     | 查看 Dashboard 状态 | `/panel/`               | `GET /panel/api/server/status`                                  | 状态字段、Xray 状态、流量展示与旧 UI 一致             | 默认执行                        | 回旧 `/panel/`         |
| P0     | 查看 Inbounds 列表  | `/panel/inbounds`       | `GET /panel/api/inbounds/list`                                  | 协议、端口、客户端数、流量、启用状态不丢失            | 默认执行                        | 回旧 `/panel/inbounds` |
| P0     | 查看 Xray 配置页    | `/panel/xray`           | `POST /panel/xray/`                                             | 模板 JSON 可加载，错误配置不白屏                      | 默认执行                        | 回旧 `/panel/xray`     |
| P0     | 查看 Settings       | `/panel/settings`       | `POST /panel/setting/all`                                       | 设置项读回完整，敏感项不额外暴露                      | 默认执行                        | 回旧 `/panel/settings` |
| P0     | 查看面板日志        | `/panel/` 日志弹窗      | `POST /panel/api/server/logs/:count`                            | 纯文本渲染，不使用 `v-html`                           | 默认执行 API 基线               | 回旧 `/panel/`         |
| P0     | 查看 Xray 日志      | `/panel/` Xray 日志弹窗 | `POST /panel/api/server/xraylogs/:count`                        | 表格/纯文本渲染，不执行日志内容                       | 默认执行 API 基线               | 回旧 `/panel/`         |
| P1     | 新增入站            | `/panel/inbounds`       | `POST /panel/api/inbounds/add`                                  | 写入字段与 `model.Inbound` 兼容，旧 UI 可继续编辑     | `SUPERXRAY_E2E_MUTATION=1`      | 删除测试入站，回旧 UI  |
| P1     | 新增客户端          | `/panel/inbounds`       | `POST /panel/api/inbounds/addClient`                            | `settings.clients` 结构与旧 UI 一致                   | `SUPERXRAY_E2E_MUTATION=1`      | 删除测试入站，回旧 UI  |
| P1     | 删除入站/客户端     | `/panel/inbounds`       | `POST /del/:id`, `POST /:id/delClient/:clientId`                | 删除后列表、Xray 重启标记与旧行为一致                 | Mutation 清理路径覆盖删除入站   | 回 DB 备份/旧 UI       |
| P1     | 重置流量            | `/panel/inbounds`       | `POST /resetAllTraffics`, `POST /:id/resetClientTraffic/:email` | 只重置目标统计，不破坏客户端配置                      | 后续补充                        | 回旧 UI                |
| P1     | 重启 Xray           | `/panel/`               | `POST /panel/api/server/restartXrayService`                     | 只走旧 `XrayService` 路径，阶段 10 前不进 CoreManager | `SUPERXRAY_E2E_RESTART=1`       | 回旧 UI/服务命令       |
| P1     | 查看订阅            | Sub Server              | `/sub/:subid`, `/json/:subid`, `/clash/:subid`                  | 新 UI 生成 URL 与旧 UI 相同                           | `SUPERXRAY_E2E_SUB_URL`         | 回旧二维码/分享弹窗    |
| P2     | 修改 Xray 配置模板  | `/panel/xray`           | `POST /panel/xray/update`                                       | 保存后旧 UI 可读；失败不写坏 DB                       | 后续 Phase 5                    | 回旧 UI/备份恢复       |
| P2     | 修改设置            | `/panel/settings`       | `POST /panel/setting/update`                                    | 设置保存后旧 UI 与 sub server 行为一致                | 后续 Phase 7                    | 回旧 UI/备份恢复       |
| P2     | DB 导入             | `/panel/`               | `POST /panel/api/server/importDB`                               | 大小限制、失败不破坏当前 DB、导入后服务状态明确       | 后续 Phase 7/9                  | 回 DB 备份             |

## 当前阻断与后续要求

- 新 UI 建成前，所有写入必须继续调用旧 API，不能引入 `proxy_inbounds`、`proxy_clients` 或中性配置写模型。
- 新 UI 的日志、配置预览、订阅预览必须默认纯文本渲染。
- Phase 1 新前端工程必须生成稳定定位器，补齐 UI 级 E2E；Phase 0 当前先用旧 API 和旧页面作为验收基线。
- Phase 10 前，Xray 重启、停止、安装仍以 `web/service` 与 `xray/` 现有路径为准。
