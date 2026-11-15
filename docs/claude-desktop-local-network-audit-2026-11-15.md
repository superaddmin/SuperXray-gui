# 本机 Claude Desktop 网络信息与真实 IP 审计

日期：2026-11-15，Asia/Shanghai。前序仓库分析见[订阅来源隐私审计](claude-subscription-origin-privacy-audit-2026-11-15.md)。本报告针对本机已经安装并运行的 Windows 客户端，不能由前序订阅源码分析替代。

用户已确认也使用 **Code、Cowork 或 SSH 远程功能**，因此相应地址读取与代理继承路径属于实际关注范围；这个回答不证明三种功能当前都在运行，也没有确认使用语音。

## 结论与证据等级

**客户端有在特定功能下读取本机 IP、IPv6、MAC 的真实业务代码；本次尚未证明终端原始公网 IP 已上传给 Anthropic。** 约 30 秒、10 次 TCP 采样中，所见 Claude 连接全部使用 FlClash 的 fake-IP，没有观察到绕过 FlClash 的公网 TCP 连接。这不是全部协议、全部时间或发送载荷的无泄漏证明。

要分别判断三个问题：

1. **能否读取：能。** 本机有物理网卡全球 IPv6 地址；枚举网卡可以取得这样的地址，而不需要访问“查询公网 IP”网站。物理网卡的私有 IPv4 地址则不等于路由器 NAT 后的公网 IPv4。
2. **本次是否读取：未对全部相关函数作运行跟踪。** SSH 监测、预览安全检查、Cowork/VM 诊断等调用有各自触发条件，不能把安装包中存在代码写成当前会话已经执行。
3. **是否外发真实地址：尚未证实。** 已追踪的网卡监测主要在内存比较，IPv6 遥测只发送能力分类；另外发现语音 SDP 的发送路径，需要单独核对实际候选地址，不能仅据常规遥测字段推断整个应用没有地址外发。

**即使真实 IP 未暴露，地区线索仍可能存在。** 本机 Windows 时区为 `China Standard Time`（UTC+8），区域/UI 区域为 `zh-CN`，Claude 配置语言为 `zh-Hans`；客户端 Sentry 上下文构造包含 locale/timezone。静态实现支持在相关错误报告产生时附带这些字段，但本次没有验证具体事件是否成功发出。中文语言或 UTC+8 也不能单独证明人在中国。

## 检查对象与方法

| 项目 | 本次证据 |
| --- | --- |
| 安装类型 | Microsoft Store，包 `Claude_5.11492.5.0_x64__pzs8sxrjxfjjc` |
| 可执行文件版本 | `5.11492.5`；Authenticode 状态 `Valid`，签名者 `Anthropic, PBC` |
| 主安装文件 | `%ProgramFiles%\WindowsApps\Claude_5.11492.5.0_x64__pzs8sxrjxfjjc\app\resources\app.asar` |
| app.asar | 88,812,777 字节；SHA256 `80c8c9716db54cedc93f64f4e4829a3532937e8bb5a4fb55049b5238c54ed9da`，读取前后相同 |
| ion-dist.asar | 208,001,845 字节；SHA256 `1d55d633d14aede748af3cb22c244f4ddc2f795d9b1153b175b993ae5d46aa1b`，读取前后相同；补扫 4,792 个 JS 中的网络信息及 WebRTC/语音实现 |
| 代理软件 | FlClash `0.9.3+2026110401`，FlClashCore 与 TUN 网卡均运行 |
| 操作方式 | 只读进程/连接/路由/选定配置与日志；ASAR 在内存中解析，未解包或执行其中代码 |
| 内容边界 | 未读取聊天、Cookie、认证文件、Local Storage、IndexedDB 或用户会话数据库；未输出真实公网 IPv6、节点凭据或订阅 URL |
| 未执行的验证 | 未触发语音/SSH/Cowork 会话，未发送测试聊天，未进行 TLS 中间人解密、全量抓包或代理中断测试 |

可执行文件签名有效不等于对每个外部资源做了完整性认证；ASAR 摘要用于固定本次检查对象，不用于宣称所有内容无风险。

以下静态定位使用 ASAR 内文件路径及**文件内从 0 开始的字节偏移**，不是源码行号；压缩代码中的函数名只对应本版本。

## 客户端具体读取什么，以及流向哪里

| 信息/行为 | 已确认的代码路径 | 本次判断 |
| --- | --- | --- |
| 接口 IPv4、IPv6、MAC | `app.asar/.vite/build/index.chunk-ChDCCO7W.js`，偏移 784801、785220：`networkInterfaces()`，`${family}\|${address}\|${mac}` 放入内存 Map，每 3 秒比较；没有 SSH 远程控制器时 `zD.size===0` 提前返回 | 有条件地读取完整接口信息。已追踪的日志只有固定变化事件，不打印地址；未发现此 Map 直接进入常规遥测。不能证明本次实际启动该监测。 |
| 网络身份变化 | 同文件，偏移 465287：使用 IPv4 地址及 IPv6 前 64 位作内存比较 | 用于变化判定和重连；未发现该地址/前缀被直接加入所追踪的发送字段。 |
| 本机地址安全检查 | `index.chunk-DL8fZzVo.js`，偏移 3691567：枚举接口并缓存地址集合，阻止预览访问主机、私网和元数据地址 | 这里读取地址是访问限制的一部分，不是公网 IP 上报器。 |
| IPv6 能力 | 同文件，偏移 4831639、4867908：分类 `global / ula_only / none / unknown`；`lam_vm_startup_completed` 事件含 `vm_ipv6` | VM 功能相应分支会发送 `off`、`on_global` 等分类，不含具体 IPv6 地址；不能从 `on_global` 推出中国地区。 |
| 主机名 | `index.chunk-D1otqB6N.js`，偏移 149647、74806：`machineName: os.hostname()`，随后 `api.registerEnvironment(...)` | 特定远程运行器注册有发送电脑名的实现。不能把 Sentry 默认不带 server name 外推为整个应用不发送主机名。 |
| 系统与设备环境 | `index.chunk-DL8fZzVo.js`，偏移 3477471、3482000 起：构造事件并 POST `/api/event_logging/v2/batch` | 公共字段含平台、架构、系统版本、CPU、内存、安装/会话标识、账号/组织信息。未在该公共构造器找到原始接口 IP/MAC。 |
| 时区、语言及错误上下文 | `index.pre.js`，偏移 169074、170345、251263、1047408：Sentry 上下文和初始化 | 存在 locale/timezone、OS、CPU、内存等上下文；`sendDefaultPii:false`、`includeServerName:false` 不是“零设备数据”或“接收方看不到连接 IP”。 |
| 网络诊断 | `index.chunk-Cm4XzlH2.js`，偏移 26018：统计外部 IPv4 接口数；DL8fZzVo 文件偏移 4819547 的 VPN/接口专项诊断有 darwin 门槛 | 前者是数量；后者在 Windows 返回空值，不能把 macOS 的 VPN 诊断全部算到这台 Windows 上。 |
| DNS 与代理状态 | 预览安全检查使用 Node DNS `resolve4/resolve6/lookup`；SSH 使用 DNS；Electron 代码调用 `resolveProxy()` | DNS 不天然服从 HTTP 代理。实际是否进入 FlClash 取决于系统路由及 DNS 接管，不可从 HTTP 代理设置单独判断。 |
| 网络质量与在线状态 | `ion-dist.asar/assets/v1/shared-0-heujONeF.js`，字节 341733：读取 `navigator.connection`，组装 `network_effective_type / network_downlink_mbps / network_rtt_ms / network_save_data`；另有连接变更/在线状态处理 | 网络类型、估计带宽、延迟和省流量偏好不是 IP 地址。具体事件是否触发仍需运行证据。 |

没有在已追踪路径中找到调用 `ifconfig.me` 获取真实公网 IP 的证据；其字符串位于域名字典，字符串出现不是请求发生的证据。Statsig 命中也主要是兼容结构和字典，不能据此宣称该安装版本正在向 Statsig 上报。

SSH 条件进一步核实：创建或尝试远程连接、产生首个远程控制器即可启动每 3 秒监听，不要求 SSH 已连接成功；控制器集合清空后仍保留定时器，但提前返回而不再枚举接口（ChDCCO7W 文件字节 784564、786203、785220）。

时区/语言并非只在依赖中出现：`index.pre.js` 字节 170144 的函数使用 `Intl.DateTimeFormat().resolvedOptions()` 返回 locale/timeZone；默认 Context integration 的 culture 分支将其放入 `contexts.culture.locale/timezone`。这是代码中的事件上下文构造，未捕获本机实际发送载荷。

ion 的针对性搜索没有找到业务枚举 SSID/BSSID、默认网关或 DNS 服务器的实现，相关字符串命中为语法词典等；也未命中常见外部公网 IP 查询服务。该有限搜索不能证明原生模块和未来加载的代码绝无这些能力。

### 遥测目的地与隐私开关的边界

桌面包存在以下实际发送实现：

- `https://claude.ai/api/event_logging/v2/batch`：通用事件批量发送。
- `o1158394.ingest.us.sentry.io`：本版本桌面 Sentry 接收域。
- 远程运行器注册与语音信令：各自独立的业务通道，不等同于 Sentry 错误报告。

这些是安装包代码证据，不是本次抓获的请求清单。上一份报告中的 Claude Code Datadog 端点与开关不能直接替代这个 Desktop 版本的核查结果。

Sentry 默认 PII 关闭和 server name 关闭，只限制对应 SDK 的默认采集；已看到事件/breadcrumb/崩溃 scope 的脱敏处理，但不能据此证明所有自定义字段、异常文本或附件都绝不会包含地址。接收方仍能看到到达它的网络连接 IP，完整代理时通常是代理出口。

### 语音 WebRTC / SDP 是独立验证项

`ion-dist.asar` 中的 `assets/v1/ccc68ce7f-JxoeNNHB.js` 有实际语音实现：

1. 字节 121622 创建 `RTCPeerConnection({iceServers:[]})`，没有在此配置 STUN/TURN。
2. 字节 123998 读取 `localDescription.sdp`，经 prepared 回调到字节 149236 的 `signaling.sendMediaOffer(...)`。
3. 字节 222564 在 WebSocket 打开、会话初始化且选择 WebRTC 媒体时发送 `media_offer` 的 SDP；字节 164740 的序列化没有额外候选地址过滤。
4. 字节 134907 的统计提取保留 candidateType/protocol/RTT 等，不取 address。

`assets/v1/shared-100-BlScigbm.js` 字节 47029 表明：RTCPeerConnection 可用且功能开关 `__gb__3570531266` 启用时选择 `webrtc_media`，否则为 `socket_audio`，另有失败回退。信令路径位于 `/api/ws/voice/webrtc/organizations/.../chat_conversations/...`。本次没有读取这些会话标识或触发该功能。

**发送 SDP 不等于已证明发送真实公网 IP。** SDP 内实际候选可能受 Chromium 的接口选择、mDNS 地址隐藏、代理策略、媒体模式和收集时机影响；没有配置外部 STUN/TURN，也不意味着不进行 ICE 连通性检查、不发送媒体包，或所有 host 候选都不包含可路由 IPv6。需要观察真实语音会话的候选和发送内容才能下结论。本次没有启动语音或读取音频。未找到远程媒体分支另外逐条发送本地 icecandidate 的实现；生成并发送 SDP 的路径仍需独立评估。

桌面 JS 共找到五处 `setWebRTCIPHandlingPolicy("disable_non_proxied_udp")`，分别属于 widget、受条件控制的 LaunchPreview/SSH pane、两个 Cowork artifact 视图及文件/PDF 预览。未找到主聊天/语音 WebContents 的同类调用，也未找到对应全局强制开关。部分页面 CSP 的 `webrtc 'block'` 同样有作用范围，不能把这些局部保护当作主语音保证。静态未找到也不替代运行时策略检查。

普通分析事件另外存在 `piiScrub` 插件，删除顶层 `ip / ipAddress / ip_address` 等字段（`shared-67-rE4eQxQz.js` 字节 138731、182237）。这不等于过滤任意嵌套载荷，也不约束上述语音 SDP。

## 本机运行时与 FlClash 证据

### TCP 采样

北京时间 **13:57:32–13:58:02**，间隔约 3 秒，共 10 次，按当时所有 `claude` 进程 PID 查询 TCP 表：

- 网络服务角色为 Electron `network.mojom.NetworkService`。
- 累计观察到 9 个不同 fake-IP 目的地址，端口均为 443，本地源为 TUN 地址 `198.18.0.1`；地址包含 `198.18.0.55` 等。
- 观察到的 TCP 状态为 Established 或 CloseWait；**非 fake-IP、非回环的 Claude TCP 观察数为 0**。
- 另观察到 Claude 网络进程有一个 UDP 绑定端口；Windows UDP 端点表不提供该端口的逐包目的地，因此不能断言没有 UDP 外发。
- 后续子进程树快照仍为 Claude 主进程及 Claude 子进程，没有在该树中看到单独的 SSH/VM 可执行程序；这不是历史或将来进程的保证。

按用户补充进一步检查到 **`CoworkVMService` 正在运行**，启动类型为 Auto，进程为 `cowork-svc.exe`。它是独立服务，不在桌面主进程的子树中。另一次针对该服务和当时 vmwp/vmmem 的端点快照没有返回 TCP 远端连接或 UDP 绑定；不能据此证明 guest 没有联网，因为虚拟机、HNS/NAT 或宿主转发的流量不一定归属于这些 PID，也未确认现存所有虚拟机属于 Claude。

FlClashCore 的同时期外连包括一个远端节点的 443，以及 `223.6.6.6:443`、`1.12.12.12:443`；后两者与当前国内 DoH 配置相符。没有控制器连接记录，无法把每条 core 外连逐一归属到 Claude 或某个遥测域名，不能据此给出最终出口国家或逐请求路径证明。

### 实际代理配置

只读来源：`%APPDATA%\com.follow\clash\config.yaml`，最后写入时间 13:42:44。以下只摘录非敏感字段。

| 项目 | 当前文件/系统证据 | 解释 |
| --- | --- | --- |
| 流量规则 | `mode: rule`；规则只有 `MATCH,PROXY` | 文件中没有 Claude/遥测专属直连规则。 |
| PROXY 组 | select，唯一远端节点及 `DIRECT` | 当前选择未持久化；不能仅据远端节点排第一就断言当前选择。外连快照支持核心正在使用远端节点，但不替代选择值和逐连接记录。 |
| TUN | enable、auto-route、auto-detect-interface 开启；`dns-hijack: [any:53]` | 系统确有 `FlClash / Meta Tunnel` 网卡 Up，支持已经接管所观察流量。 |
| 系统 HTTP 代理 | Windows `ProxyEnable=0`，保留地址 `127.0.0.1:7890`；WinHTTP 为 direct | 当前主要依赖 TUN。系统 HTTP 代理关闭不等于已经直连泄漏。 |
| IPv6 | 内核顶层 `ipv6:true`，DNS `ipv6:false`，GUI 偏好另有 `vpnProps.ipv6:false` | 不是同一开关。现场 WLAN 有全球 IPv6，TUN 有覆盖全球单播地址范围的路由；不能说设备已无 IPv6。 |
| DNS | fake-IP；nameserver 为 `https://doh.pub`、`https://dns.alidns.com`；`respect-rules:false` | DNS 路径与目标业务的 `MATCH,PROXY` 不能混为一谈。 |
| Bootstrap | 包含 114.114.114.114、223.5.5.5、8.8.8.8、1.0.0.1 | 存在额外解析路径；本次没有给每一条查询做归属。 |
| 可观测性 | `log-level:error`、`find-process-mode:off`、GUI openLogs=false、sniffer关闭 | 缺少可直接恢复每条连接域名/规则/代理链的现存记录。 |
| 控制器与缓存 | controller 地址空；store-selected/store-fake-ip 均 false | 未发现可只读查询当前选择和连接的已配置接口；11月1日旧 cache.db 不代表当前映射，本次未继续读取其内容。 |
| MITM | `mitm.enable:false` | 当前没有借此读取 HTTPS 遥测正文。 |

**国内 DNS 首先影响 DNS 服务的可见性。** 它可能看到对应查询及连接源地址；这不构成“Claude 已经获得这些查询来源”的证明。系统有全球 IPv6 也只是具备潜在暴露条件，不能当作泄漏已经发生。

尝试只读查看 FlClash 界面时，工具提示其 Windows 完整性级别更高，可访问树只有标题；窗口截图两次超时，因此没有取得 UI 当前选择值，也未切换节点或更改设置。

### Code / Cowork / SSH 的额外网络边界

用户已确认涉及这些功能。下表偏移均在 `app.asar::.vite/build/index.chunk-DL8fZzVo.js` 内，属于实现证据；真实会话环境、PAC 配置和远端出口未作运行跟踪。

| 功能 | 实现与定位 | 对本机判定的影响 |
| --- | --- | --- |
| Code 代理解析 | 字节 1534376、1534626：优先固定代理，否则 Electron `resolveProxy()`；结果可生成 HTTP_PROXY、HTTPS_PROXY 和仅本地地址的 NO_PROXY | 不可只看系统 HTTP 代理开关。 |
| Code 既有配置 | 字节 1537301：此前缓存过默认 API 的 HTTP 代理时，即使本次解析 DIRECT 仍保留旧代理；字节 6043085、1541934、6019765：继承进程/用户/会话环境，只补缺失变量，受管设置可进一步覆盖 | DIRECT 不等于清空环境代理；本次没有读取完整实际 Code 会话环境。CLI 启动器确实接收整理后的 env，见 CjUF9KtU 文件字节 2285220、2285379。 |
| Cowork guest 代理 | 字节 4864700、4815970：显式固定代理/PAC 经专用路径构造 guest 参数，可通过 host alias 访问主机 loopback；没有在此套用 Code 的系统代理整理函数 | 桌面、Code 与 guest 不共享一条可以直接假定的代理配置链。 |
| PAC 下载及失败 | 字节 4816606 的函数：专用 session 使用 `setProxy({mode:"direct"})` 下载；失败、超时、空或过大等情况返回空 PAC，日志为 `guest connects directly`，不会因此阻止 VM 启动 | 存在代理层降级行为；只有实际使用该 PAC 路径才相关。它是否绕过 TUN 取决于宿主/guest 网络，不能据这句日志直接判定本机已漏。 |
| 服务与 guest IPv6 | 字节 3470312、3461286：通过 `\\.\pipe\cowork-vm-service` 传入 guestProxy/ipv6；字节 4832007：仅 `workspace.vmIpv6Enabled===true` 时设置预置 guest IPv6 | 不是把物理网卡全球 IPv6 原样赋给 guest；本次未确认实际开关与 guest 路由。 |

实际 Windows NAT/转发实现不在已检查 JS 中。部分公共调用及遥测使用 `gvisor` 字符串，但 Windows 服务适配器没有把该参数作为网络模式字段发送；不能用这个字符串证明实际网络实现或 TUN 接管方式。SSH 远端执行的请求还使用远端自己的网络，不能从本机快照验证其出口。

还有独立于遥测的**任务内容路径**：如果 Code/Cowork 的网络诊断工具输出、读取的配置或用户消息包含真实地址，并被加入模型请求，上游可以在请求正文中收到这些信息；代理不会自动改写正文。这里只说明条件，不表示本次发生过；本次没有读取用户会话，也没有让 Claude 执行网络诊断命令。

### 客户端日志与配置

- `%APPDATA%\Claude\config.json` 记录版本 `5.11492.5`、`locale=zh-Hans`。
- 所检查 `settings.json` 的 env 没有显式 HTTP_PROXY/HTTPS_PROXY/NO_PROXY；不能据此推断进程环境或 TUN 状态。
- `%APPDATA%\Claude\logs` 为空；Store LocalCache 下 main、claude.ai-web、mcp、ssh、cowork_vm_node 五个日志文件均为 0 字节。**日志为空不代表没有联网。**
- 存在近期更新的本地 Sentry 状态文件；本次只看元数据，不读取事件内容，不能从文件存在推断成功上传、发送了哪些字段或携带真实 IP。
- Windows DNS 缓存没有返回本次限定 Claude/Anthropic/Sentry/Statsig/Datadog 条件下的记录，不能由此恢复 fake-IP 域名映射。

## 仍需什么证据才能确定泄漏

| 问题 | 本次结论 | 更强判定所需证据 |
| --- | --- | --- |
| 是否绕过代理把终端 IP 作为网络来源暴露 | 有限 TCP 窗口未观察到 | 覆盖实际功能的 IPv4/IPv6、TCP/UDP 连接归属及 TUN 后的代理链，包含短连接和异常/中断情形。 |
| 是否在代理内的事件载荷中上传本机地址 | 已追踪常规字段未发现完整 IP/MAC，不能排除所有路径 | 触发条件与地址到发送载荷的数据流，必要时经脱敏的发送前事件或候选地址检查；不能只看出口 IP。 |
| 语音是否发送可识别真实 IP 的 SDP | 已发现 SDP 发送实现，未触发/观察实际内容 | 指定版本下的实际语音候选、主视图策略和发送 SDP；不能仅凭是否配置 STUN 推断。 |
| 能否推断地区 | 有时区、语言、账号等潜在线索 | 本次没有平台侧处理或风控结果，不能断言推断方式、准确率或已经判为中国。 |

若后续加深验证，应优先限定用户实际使用的功能，再增加只记录域名/进程/规则/代理链的可观测性，并专门检查语音 SDP；不要把关闭某个遥测开关、DNS IPv6=false 或一次 IP 查询当成全应用无泄漏证明。本次没有修改客户端、代理、系统网络、权限或隐私设置。

本报告是某版本、某配置、某时段的观察结果；已盘点但没有深入逆向 `chrome-native-host.exe`、SSH broker/proxy/askpass、`cowork-svc.exe`、`claude-native-binding.node`、ConPTY、MSAL 和 GitHub MCP 等原生辅助文件内部实现。静态归档也不能完全代表运行时下载的网页代码或未来版本。没有将任何完整节点/用户地址写入报告，也没有提交或发布本机诊断材料。
