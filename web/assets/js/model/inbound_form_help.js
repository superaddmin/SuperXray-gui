(function (root, factory) {
    if (typeof module === 'object' && module.exports) {
        module.exports = factory();
    } else {
        root.InboundFormHelp = factory();
    }
})(typeof globalThis !== 'undefined' ? globalThis : this, function () {
    const HELP_ITEMS = [
        {
            keys: ['启用', 'enable'],
            label: { zh: '启用', en: 'Enable' },
            description: {
                zh: '控制该入站是否参与 Xray 配置生成。关闭后会保留表单数据，但不会对外提供服务。',
                en: 'Controls whether this inbound is included in the generated Xray configuration. Disabled inbounds keep their data but do not serve traffic.',
            },
            tips: {
                zh: ['新建可先保持开启。排障或临时停用时再关闭。'],
                en: ['Keep it enabled for a new inbound. Turn it off only for troubleshooting or temporary suspension.'],
            },
        },
        {
            keys: ['备注', 'remark'],
            label: { zh: '备注', en: 'Remark' },
            description: {
                zh: '用于列表展示和管理识别，不会写入客户端连接地址。',
                en: 'Used for display and management in the inbound list. It is not part of the client connection URL.',
            },
            example: 'vless-reality-443',
        },
        {
            keys: ['协议', 'protocol'],
            label: { zh: '协议', en: 'Protocol' },
            description: {
                zh: '选择入站协议。常见公网代理入口可使用 VLESS、VMess、Trojan、Shadowsocks 或 Hysteria。',
                en: 'Select the inbound protocol. Common public proxy choices include VLESS, VMess, Trojan, Shadowsocks, and Hysteria.',
            },
        },
        {
            keys: ['监听', 'monitor', 'listen'],
            label: { zh: '监听地址', en: 'Listen Address' },
            description: {
                zh: '限制 Xray 监听的本机地址。留空通常表示监听所有地址；填写 127.0.0.1 表示仅本机访问。',
                en: 'Limits the local address Xray listens on. Empty usually means all addresses; 127.0.0.1 means local access only.',
            },
            example: '0.0.0.0 / 127.0.0.1',
        },
        {
            keys: ['端口', 'port'],
            label: { zh: '端口', en: 'Port' },
            description: {
                zh: '客户端连接该入站时使用的服务端口。必须是 1-65535，且不能被其他服务占用。',
                en: 'The server port clients connect to. It must be 1-65535 and must not be used by another service.',
            },
            example: '443',
        },
        {
            keys: ['总流量', 'total flow', 'total traffic'],
            label: { zh: '总流量', en: 'Total Traffic' },
            description: {
                zh: '该入站或客户端的可用总流量限制。0 表示不限制。',
                en: 'Total traffic quota for the inbound or client. 0 means unlimited.',
            },
        },
        {
            keys: ['流量重置', 'traffic reset', 'periodic traffic reset'],
            label: { zh: '流量重置', en: 'Traffic Reset' },
            description: {
                zh: '设置流量统计按小时、天、周或月自动重置。选择“从不”表示不自动清零。',
                en: 'Sets automatic traffic reset by hour, day, week, or month. "Never" disables automatic reset.',
            },
        },
        {
            keys: ['到期时间', 'expire date', 'expiry time'],
            label: { zh: '到期时间', en: 'Expiry Time' },
            description: {
                zh: '到达该时间后入站或客户端会过期。留空表示永不过期。',
                en: 'The inbound or client expires after this time. Leave it empty to never expire.',
            },
        },
        {
            keys: ['传输', 'transmission', 'network'],
            label: { zh: '传输方式', en: 'Transport' },
            description: {
                zh: '选择底层传输。VLESS Reality 和 Vision Flow 通常使用 TCP (RAW)。WebSocket、gRPC、XHTTP 等适合不同反代或客户端场景。',
                en: 'Select the transport layer. VLESS Reality and Vision Flow usually use TCP (RAW). WebSocket, gRPC, and XHTTP fit different reverse proxy or client scenarios.',
            },
        },
        {
            keys: ['安全', 'security'],
            label: { zh: '安全层', en: 'Security' },
            description: {
                zh: '选择无加密、Reality 或 TLS。Reality/TLS 会显示各自的证书、SNI、密钥等字段。',
                en: 'Choose none, Reality, or TLS. Reality/TLS reveal their certificate, SNI, key, and related fields.',
            },
        },
        {
            keys: ['客户', 'client', 'clients'],
            label: { zh: '客户', en: 'Client' },
            description: {
                zh: '配置可连接该入站的用户。VLESS 的 Flow 字段也在这里；如果选择 TLS/Reality + TCP，请展开此区域检查 Flow。',
                en: 'Configures users allowed to connect. VLESS Flow is also here; with TLS/Reality + TCP, expand this section and check Flow.',
            },
        },
        {
            keys: ['电子邮件', 'email'],
            label: { zh: '电子邮件', en: 'Email' },
            description: {
                zh: '客户端唯一标识，常用于流量统计、订阅和在线用户识别。建议只用字母、数字、点、下划线或短横线。',
                en: 'A unique client identifier used for traffic stats, subscription, and online-user tracking. Prefer letters, numbers, dots, underscores, or hyphens.',
            },
            example: 'user01',
        },
        {
            keys: ['id'],
            label: { zh: '用户 ID', en: 'Client ID' },
            description: {
                zh: 'VMess/VLESS 客户端 UUID。必须保持唯一，点击同步图标可以生成新的 UUID。',
                en: 'VMess/VLESS client UUID. It must be unique. Use the sync icon to generate a new UUID.',
            },
        },
        {
            keys: ['subscription'],
            label: { zh: '订阅标识', en: 'Subscription ID' },
            description: {
                zh: '用于订阅链接中的客户端标识。开启订阅功能并填写邮箱后才会显示。',
                en: 'Client identifier used by subscription links. It appears only when subscription is enabled and email is set.',
            },
        },
        {
            keys: ['评论', 'comment'],
            label: { zh: '评论', en: 'Comment' },
            description: {
                zh: '仅用于管理备注，可填写用途、客户名或到期说明。',
                en: 'Management-only note. You can record usage, customer name, or expiry notes here.',
            },
        },
        {
            keys: ['flow'],
            label: { zh: '流控', en: 'Flow' },
            description: {
                zh: 'VLESS 的 Vision 流控选项。仅在 VLESS + TCP + TLS/Reality 时显示。普通客户端不支持时请选择 none。',
                en: 'VLESS Vision flow control. It appears only for VLESS + TCP + TLS/Reality. Choose none if the client does not support it.',
            },
            example: 'xtls-rprx-vision',
        },
        {
            keys: ['authentication'],
            label: { zh: '认证方式', en: 'Authentication' },
            description: {
                zh: 'VLESS 非 TLS/Reality 模式下的附加认证选项。多数普通场景保持 None。',
                en: 'Additional authentication for VLESS without TLS/Reality. Most common setups should keep None.',
            },
        },
        {
            keys: ['decryption'],
            label: { zh: '解密', en: 'Decryption' },
            description: {
                zh: 'VLESS 加密/认证相关字段。未使用附加认证时保持 none。',
                en: 'VLESS encryption/authentication related field. Keep none when additional authentication is not used.',
            },
        },
        {
            keys: ['encryption'],
            label: { zh: '加密', en: 'Encryption' },
            description: {
                zh: 'VLESS 加密/认证相关字段。未使用附加认证时保持 none。',
                en: 'VLESS encryption/authentication related field. Keep none when additional authentication is not used.',
            },
        },
        {
            keys: ['fallbacks'],
            label: { zh: '回落', en: 'Fallbacks' },
            description: {
                zh: '当 VLESS/Trojan 入站无法完成协议握手时，把流量转发到指定目标。普通 Reality/TLS 入站可先不配置。',
                en: 'Forwards unmatched VLESS/Trojan traffic to another destination. Leave it empty for a normal Reality/TLS inbound unless you need fallback routing.',
            },
        },
        {
            keys: ['proxy protocol'],
            label: { zh: '代理协议', en: 'Proxy Protocol' },
            description: {
                zh: '用于接收上游负载均衡器传来的真实源 IP。只有上游明确发送 Proxy Protocol 时才开启。',
                en: 'Accepts the real source IP from an upstream load balancer. Enable it only when the upstream sends Proxy Protocol.',
            },
        },
        {
            keys: ['http 伪装', 'http camouflage', 'http header'],
            label: { zh: 'HTTP 伪装', en: 'HTTP Camouflage' },
            description: {
                zh: '为 TCP 传输添加 HTTP 头部伪装。只在明确需要 TCP HTTP Header 时开启。',
                en: 'Adds HTTP header camouflage to TCP transport. Enable it only when TCP HTTP Header is required.',
            },
        },
        {
            keys: ['sockopt'],
            label: { zh: 'Socket 选项', en: 'Sockopt' },
            description: {
                zh: '设置底层 socket 参数，如 TFO、MPTCP、拥塞控制等。不了解内核网络参数时建议保持默认。',
                en: 'Configures low-level socket options such as TFO, MPTCP, and congestion control. Keep defaults unless you know the kernel networking impact.',
            },
        },
        {
            keys: ['tcp masks'],
            label: { zh: 'TCP 掩码', en: 'TCP Masks' },
            description: {
                zh: '高级混淆/掩码参数。仅在服务端和客户端都明确支持时配置。',
                en: 'Advanced obfuscation/masking settings. Configure only when both server and client explicitly support them.',
            },
        },
        {
            keys: ['external proxy'],
            label: { zh: '外部代理', en: 'External Proxy' },
            description: {
                zh: '用于生成经过外部反向代理或 CDN 的客户端地址。它不改变 Xray 本身监听行为。',
                en: 'Used to generate client addresses through an external reverse proxy or CDN. It does not change how Xray listens locally.',
            },
        },
        {
            keys: ['sniffing'],
            label: { zh: '流量嗅探', en: 'Sniffing' },
            description: {
                zh: '从连接内容中识别域名或协议，用于路由分流。透明代理或复杂路由场景常用。',
                en: 'Detects domain/protocol information from traffic for routing decisions. Commonly used for transparent proxying or advanced routing.',
            },
        },
        {
            keys: ['show'],
            label: { zh: '显示握手', en: 'Show Handshake' },
            description: {
                zh: 'Reality 调试选项。生产环境通常关闭。',
                en: 'Reality debugging option. Usually keep it disabled in production.',
            },
        },
        {
            keys: ['xver', 'xver 0'],
            label: { zh: 'Proxy Protocol 版本', en: 'Xver' },
            description: {
                zh: 'Reality 回落时的 Proxy Protocol 版本。普通入站保持 0。',
                en: 'Proxy Protocol version for Reality fallback. Keep 0 for normal inbounds.',
            },
        },
        {
            keys: ['utls'],
            label: { zh: 'uTLS 指纹', en: 'uTLS Fingerprint' },
            description: {
                zh: '模拟客户端 TLS 指纹。Reality 常用 chrome。',
                en: 'Mimics a client TLS fingerprint. Reality commonly uses chrome.',
            },
        },
        {
            keys: ['target'],
            label: { zh: '目标地址', en: 'Target' },
            description: {
                zh: 'Reality 握手伪装目标，格式通常是 域名:443。建议使用真实可访问且支持 TLS 的站点。',
                en: 'Reality handshake camouflage target, usually domain:443. Use a real reachable TLS-enabled site.',
            },
            example: 'www.example.com:443',
        },
        {
            keys: ['sni'],
            label: { zh: '服务器名称', en: 'SNI' },
            description: {
                zh: 'TLS/Reality 握手中的 Server Name。Reality 模式下通常与 Target 的域名一致，可填写多个逗号分隔的名称。',
                en: 'Server Name used in TLS/Reality handshake. In Reality mode it usually matches the Target domain; multiple names may be comma-separated.',
            },
            example: 'www.example.com',
        },
        {
            keys: ['max time diff (ms)', 'max time diff'],
            label: { zh: '最大时间差', en: 'Max Time Diff' },
            description: {
                zh: 'Reality 客户端时间允许偏差，单位毫秒。0 表示不限制。',
                en: 'Allowed Reality client clock skew in milliseconds. 0 means unlimited.',
            },
        },
        {
            keys: ['min client ver'],
            label: { zh: '最低客户端版本', en: 'Min Client Version' },
            description: {
                zh: '限制 Reality 客户端最低版本。留空表示不限制。',
                en: 'Limits the minimum Reality client version. Empty means no limit.',
            },
        },
        {
            keys: ['max client ver'],
            label: { zh: '最高客户端版本', en: 'Max Client Version' },
            description: {
                zh: '限制 Reality 客户端最高版本。留空表示不限制。',
                en: 'Limits the maximum Reality client version. Empty means no limit.',
            },
        },
        {
            keys: ['short ids', 'shortids'],
            label: { zh: '短 ID', en: 'Short IDs' },
            description: {
                zh: 'Reality 客户端用于握手校验的短 ID。可点击同步图标随机生成；多个值用逗号分隔。',
                en: 'Reality short IDs used for handshake verification. Use the sync icon to randomize; separate multiple values with commas.',
            },
        },
        {
            keys: ['spiderx'],
            label: { zh: '爬虫路径', en: 'SpiderX' },
            description: {
                zh: 'Reality 客户端请求路径伪装参数。普通场景保持 /。',
                en: 'Reality client request path camouflage value. Keep / for common setups.',
            },
            example: '/',
        },
        {
            keys: ['公钥', 'public key'],
            label: { zh: '公钥', en: 'Public Key' },
            description: {
                zh: 'Reality 客户端需要使用的公钥。点击 Get New Cert 会和私钥一起生成。',
                en: 'The public key used by Reality clients. Get New Cert generates it together with the private key.',
            },
        },
        {
            keys: ['私钥', 'private key', 'privatekey'],
            label: { zh: '私钥', en: 'Private Key' },
            description: {
                zh: 'Reality 服务端必须填写的私钥。为空会导致 Xray 启动失败并报错 empty "privateKey"。请点击 Get New Cert 自动生成，或手动填入有效 X25519 私钥。',
                en: 'Required by the Reality server. If empty, Xray fails to start with empty "privateKey". Click Get New Cert or enter a valid X25519 private key manually.',
            },
            required: true,
        },
        {
            keys: ['mldsa65 seed'],
            label: { zh: 'ML-DSA-65 种子', en: 'mldsa65 Seed' },
            description: {
                zh: 'Reality 后量子相关可选参数。普通场景可留空。',
                en: 'Optional post-quantum Reality parameter. Leave empty for normal setups.',
            },
        },
        {
            keys: ['mldsa65 verify'],
            label: { zh: 'ML-DSA-65 校验', en: 'mldsa65 Verify' },
            description: {
                zh: 'Reality 后量子相关可选校验值，通常由 Get New Seed 生成。',
                en: 'Optional post-quantum Reality verification value, usually generated by Get New Seed.',
            },
        },
        {
            keys: ['cipher suites'],
            label: { zh: '加密套件', en: 'Cipher Suites' },
            description: {
                zh: 'TLS 可用加密套件。Auto 会使用 Xray 默认推荐配置。',
                en: 'Allowed TLS cipher suites. Auto uses the Xray recommended defaults.',
            },
        },
        {
            keys: ['min/max version'],
            label: { zh: '最低/最高 TLS 版本', en: 'Min/Max TLS Version' },
            description: {
                zh: '限制 TLS 协议版本。推荐最低 1.2，最高 1.3。',
                en: 'Limits TLS protocol versions. Recommended minimum is 1.2 and maximum is 1.3.',
            },
        },
        {
            keys: ['alpn'],
            label: { zh: '应用层协议', en: 'ALPN' },
            description: {
                zh: 'TLS 应用层协议协商值。常见值是 h2 和 http/1.1。',
                en: 'TLS application protocol negotiation values. Common values are h2 and http/1.1.',
            },
        },
        {
            keys: ['reject unknown sni'],
            label: { zh: '拒绝未知 SNI', en: 'Reject Unknown SNI' },
            description: {
                zh: '只接受证书匹配的 SNI。多域名或反代场景开启前需确认客户端 SNI 正确。',
                en: 'Accepts only SNI values matching certificates. Verify client SNI before enabling in multi-domain or reverse-proxy setups.',
            },
        },
        {
            keys: ['disable system root'],
            label: { zh: '禁用系统根证书', en: 'Disable System Root' },
            description: {
                zh: '禁用系统根 CA。普通 TLS 入站通常保持关闭。',
                en: 'Disables system root CAs. Usually keep it off for normal TLS inbounds.',
            },
        },
        {
            keys: ['session resumption'],
            label: { zh: '会话恢复', en: 'Session Resumption' },
            description: {
                zh: '允许 TLS 会话恢复，可减少重复握手成本。',
                en: 'Allows TLS session resumption to reduce repeated handshake overhead.',
            },
        },
        {
            keys: ['数字证书', 'certificate'],
            label: { zh: '数字证书', en: 'Certificate' },
            description: {
                zh: 'TLS 服务端证书。选择文件路径时填写证书文件和私钥文件路径；选择文件内容时粘贴 PEM 内容。',
                en: 'TLS server certificate. In file-path mode, fill certificate and private-key file paths; in content mode, paste PEM content.',
            },
            required: true,
        },
        {
            keys: ['one time loading'],
            label: { zh: '一次性加载', en: 'One Time Loading' },
            description: {
                zh: '启动时一次性加载证书内容。证书文件变化后通常需要重启。',
                en: 'Loads certificate content once at startup. Certificate file changes usually require a restart.',
            },
        },
        {
            keys: ['usage option'],
            label: { zh: '用途选项', en: 'Usage Option' },
            description: {
                zh: '证书用途。普通服务端证书使用 encipherment。',
                en: 'Certificate usage. Use encipherment for normal server certificates.',
            },
        },
        {
            keys: ['ech key'],
            label: { zh: 'ECH 密钥', en: 'ECH Key' },
            description: {
                zh: 'Encrypted Client Hello 相关密钥。不了解 ECH 时可留空。',
                en: 'Encrypted Client Hello key. Leave empty unless you intentionally use ECH.',
            },
        },
        {
            keys: ['ech config'],
            label: { zh: 'ECH 配置', en: 'ECH Config' },
            description: {
                zh: 'ECH 配置列表。通常由 Get New ECH Cert 生成。',
                en: 'ECH config list. Usually generated by Get New ECH Cert.',
            },
        },
        {
            keys: ['ech force query'],
            label: { zh: 'ECH 查询策略', en: 'ECH Force Query' },
            description: {
                zh: '控制 ECH 查询策略。普通场景保持 none。',
                en: 'Controls ECH query behavior. Keep none for common setups.',
            },
        },
        {
            keys: ['path'],
            label: { zh: '路径', en: 'Path' },
            description: {
                zh: 'HTTP/WebSocket/XHTTP 等传输使用的路径，通常以 / 开头。',
                en: 'Path used by HTTP/WebSocket/XHTTP transports. It usually starts with /.',
            },
            example: '/proxy',
        },
        {
            keys: ['host'],
            label: { zh: '主机名', en: 'Host' },
            description: {
                zh: 'HTTP Host 头或反代主机名。通常填写对外访问域名。',
                en: 'HTTP Host header or reverse-proxy hostname. Usually set it to the public domain.',
            },
        },
        {
            keys: ['dest'],
            label: { zh: '目标', en: 'Destination' },
            description: {
                zh: 'Fallback 转发目标，可为本机端口、域名或 IP:端口。',
                en: 'Fallback forwarding destination, such as a local port, domain, or IP:port.',
            },
            example: '127.0.0.1:8080',
        },
    ];

    const HELP_BY_KEY = new Map();

    function normalizeLabel(label) {
        return String(label || '')
            .replace(/[：:]/g, ' ')
            .replace(/[?？!！⟳↻↺]/g, ' ')
            .replace(/\s+/g, ' ')
            .trim()
            .toLowerCase();
    }

    function cleanLabelForDisplay(label) {
        return String(label || '')
            .replace(/[?？!！⟳↻↺]/g, '')
            .replace(/\s+/g, ' ')
            .trim();
    }

    function createGenericHelp(label) {
        const display = cleanLabelForDisplay(label);
        if (!display) return null;
        const hasChinese = /[\u3400-\u9fff]/.test(display);
        const zh = hasChinese ? display : '高级参数';
        const en = hasChinese ? 'Advanced Field' : display;
        return {
            keys: [display],
            label: { zh, en },
            description: {
                zh: `该字段是 Xray 入站配置参数“${display}”。请按照当前协议、传输方式和客户端兼容性填写；不确定时建议保持默认或留空，避免生成 Xray 不支持的配置。`,
                en: `This field is an Xray inbound setting named "${display}". Fill it according to the selected protocol, transport, and client compatibility. If unsure, keep the default or leave it empty to avoid unsupported Xray configuration.`,
            },
            tips: {
                zh: ['高级字段通常需要服务端和客户端同时支持。修改后建议先保存测试，再重启 Xray。'],
                en: ['Advanced fields usually require both server and client support. After changing them, save and test before restarting Xray.'],
            },
        };
    }

    function registerHelpItems() {
        HELP_ITEMS.forEach(item => {
            item.keys.forEach(key => HELP_BY_KEY.set(normalizeLabel(key), item));
        });
    }

    function getHelpByLabel(label) {
        const normalized = normalizeLabel(label);
        if (HELP_BY_KEY.has(normalized)) return HELP_BY_KEY.get(normalized);
        for (const [key, item] of HELP_BY_KEY.entries()) {
            if (normalized === key || normalized.includes(key)) return item;
        }
        return createGenericHelp(label);
    }

    function getBilingualLabel(label) {
        const help = getHelpByLabel(label);
        if (!help) return String(label || '').trim();
        if (!/[\u3400-\u9fff]/.test(String(label || ''))) {
            return `${help.label.en} / ${help.label.zh}`;
        }
        return `${help.label.zh} / ${help.label.en}`;
    }

    function isBlank(value) {
        return value == null || String(value).trim() === '';
    }

    function validateClientEmail(email, protocolName) {
        const value = String(email || '').trim();
        if (value === '') {
            return {
                valid: false,
                fieldKey: 'clientEmail',
                message: {
                    zh: `${protocolName} 客户端 Email 不能为空。请填写唯一的客户端标识。`,
                    en: `${protocolName} client Email is required. Enter a unique client identifier.`,
                },
            };
        }
        if (value.length > 128 || !/^[A-Za-z0-9._@-]+$/.test(value)) {
            return {
                valid: false,
                fieldKey: 'clientEmail',
                message: {
                    zh: `${protocolName} 客户端 Email 只能包含字母、数字、点、下划线、短横线和 @，且长度不能超过 128 个字符。`,
                    en: `${protocolName} client Email can contain only letters, numbers, dots, underscores, hyphens, and @, and must be 128 characters or fewer.`,
                },
            };
        }
        return { valid: true };
    }

    const CLIENT_COLLECTIONS = {
        vmess: { key: 'vmesses', label: 'VMess' },
        vless: { key: 'vlesses', label: 'VLESS' },
        trojan: { key: 'trojans', label: 'Trojan' },
        shadowsocks: { key: 'shadowsockses', label: 'Shadowsocks' },
        hysteria: { key: 'hysterias', label: 'Hysteria' },
    };

    function validateClientEmails(inbound) {
        const settings = inbound && inbound.settings ? inbound.settings : {};
        const protocol = inbound && inbound.protocol ? inbound.protocol : '';
        const collection = CLIENT_COLLECTIONS[protocol];
        if (!collection) return { valid: true };
        const clients = Array.isArray(settings[collection.key]) ? settings[collection.key] : [];
        for (const client of clients) {
            const result = validateClientEmail(client && client.email, collection.label);
            if (!result.valid) return result;
        }
        return { valid: true };
    }

    function validateInbound(inbound) {
        const stream = inbound && inbound.stream ? inbound.stream : {};

        const clientEmailValidation = validateClientEmails(inbound);
        if (!clientEmailValidation.valid) return clientEmailValidation;

        if (stream.security === 'reality' || stream.isReality === true) {
            const reality = stream.reality || {};
            if (isBlank(reality.privateKey)) {
                return {
                    valid: false,
                    fieldKey: 'privateKey',
                    message: {
                        zh: 'Reality 私钥不能为空。请点击 Get New Cert 自动生成公钥/私钥，或手动填写有效的 X25519 privateKey。',
                        en: 'Reality privateKey is required. Click Get New Cert to generate the public/private key pair, or enter a valid X25519 privateKey manually.',
                    },
                };
            }
            if (isBlank(reality.target)) {
                return {
                    valid: false,
                    fieldKey: 'target',
                    message: {
                        zh: 'Reality Target 不能为空。请填写真实可访问的 TLS 目标，例如 www.example.com:443。',
                        en: 'Reality Target is required. Enter a real reachable TLS target, for example www.example.com:443.',
                    },
                };
            }
            if (isBlank(reality.serverNames)) {
                return {
                    valid: false,
                    fieldKey: 'serverNames',
                    message: {
                        zh: 'Reality SNI 不能为空。请填写与 Target 匹配的域名。',
                        en: 'Reality SNI is required. Enter a domain that matches the Target.',
                    },
                };
            }
            if (isBlank(reality.shortIds)) {
                return {
                    valid: false,
                    fieldKey: 'shortIds',
                    message: {
                        zh: 'Reality Short IDs 不能为空。请点击 Short IDs 旁的同步图标生成。',
                        en: 'Reality Short IDs are required. Use the sync icon next to Short IDs to generate them.',
                    },
                };
            }
        }

        if (stream.security === 'tls' || stream.isTls === true) {
            const tls = stream.tls || {};
            const certs = Array.isArray(tls.certs) ? tls.certs : [];
            if (certs.length === 0) {
                return {
                    valid: false,
                    fieldKey: 'tlsCertificate',
                    message: {
                        zh: 'TLS 证书不能为空。请添加证书并填写公钥/私钥路径或 PEM 内容。',
                        en: 'TLS certificate is required. Add a certificate and fill the public/private key paths or PEM content.',
                    },
                };
            }
            for (const cert of certs) {
                if (cert && cert.useFile !== false) {
                    if (isBlank(cert.certFile) || isBlank(cert.keyFile)) {
                        return {
                            valid: false,
                            fieldKey: 'tlsCertificate',
                            message: {
                                zh: 'TLS 证书文件路径不完整。请填写公钥文件和私钥文件路径，或点击“从面板设置证书”。',
                                en: 'TLS certificate file paths are incomplete. Fill both public-key and private-key file paths, or click "Set certificate from panel".',
                            },
                        };
                    }
                } else if (isBlank(cert.cert) || isBlank(cert.key)) {
                    return {
                        valid: false,
                        fieldKey: 'tlsCertificate',
                        message: {
                            zh: 'TLS 证书内容不完整。请粘贴公钥 PEM 和私钥 PEM 内容。',
                            en: 'TLS certificate content is incomplete. Paste both public-key PEM and private-key PEM content.',
                        },
                    };
                }
            }
        }

        return { valid: true };
    }

    function escapeHtml(value) {
        return String(value || '')
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    }

    function renderHelpContent(help) {
        const zhTips = help.tips && help.tips.zh ? help.tips.zh : [];
        const enTips = help.tips && help.tips.en ? help.tips.en : [];
        const required = help.required
            ? '<div class="inbound-help-required">必填 Required</div>'
            : '';
        const example = help.example
            ? `<div class="inbound-help-example"><strong>示例 Example:</strong> <code>${escapeHtml(help.example)}</code></div>`
            : '';

        return `
            <div class="inbound-help-dialog-body">
                ${required}
                <p><strong>中文:</strong> ${escapeHtml(help.description.zh)}</p>
                <p><strong>English:</strong> ${escapeHtml(help.description.en)}</p>
                ${zhTips.length > 0 ? `<ul>${zhTips.map(t => `<li>${escapeHtml(t)}</li>`).join('')}</ul>` : ''}
                ${enTips.length > 0 ? `<ul>${enTips.map(t => `<li>${escapeHtml(t)}</li>`).join('')}</ul>` : ''}
                ${example}
            </div>`;
    }

    function enhanceLabels(rootElement) {
        rootElement.querySelectorAll('.ant-form-item-label > label').forEach(label => {
            if (label.dataset.inboundBilingual === '1') return;
            const help = getHelpByLabel(label.textContent);
            if (!help) return;
            const translation = document.createElement('span');
            translation.className = 'inbound-label-translation';
            translation.textContent = ` / ${help.label.en}`;
            if (normalizeLabel(label.textContent).includes(normalizeLabel(help.label.en))) {
                translation.textContent = ` / ${help.label.zh}`;
            }
            label.appendChild(translation);
            label.dataset.inboundBilingual = '1';
        });

        rootElement.querySelectorAll('.ant-collapse-header').forEach(header => {
            if (header.dataset.inboundBilingual === '1') return;
            const help = getHelpByLabel(header.textContent);
            if (!help) return;
            const text = document.createElement('span');
            text.className = 'inbound-collapse-translation';
            text.textContent = ` / ${help.label.en}`;
            header.appendChild(text);
            header.dataset.inboundBilingual = '1';
        });
    }

    function enhanceHelpButtons(rootElement, options) {
        rootElement.querySelectorAll('.ant-form-item').forEach(item => {
            if (item.dataset.inboundHelp === '1') return;
            const label = item.querySelector('.ant-form-item-label > label');
            const control = item.querySelector('.ant-form-item-control');
            if (!label || !control) return;
            if (!control.querySelector('input, textarea, .ant-select, .ant-switch, .ant-radio-group, .ant-input-number, .ant-calendar-picker')) {
                return;
            }
            const help = getHelpByLabel(label.textContent);
            if (!help) return;
            const button = document.createElement('button');
            button.type = 'button';
            button.className = 'inbound-field-help-button';
            button.setAttribute('aria-label', `${help.label.zh} / ${help.label.en} help`);
            button.innerHTML = '<span aria-hidden="true">!</span>';
            button.addEventListener('click', event => {
                event.preventDefault();
                event.stopPropagation();
                if (options && typeof options.onHelp === 'function') {
                    options.onHelp(help);
                }
            });
            control.classList.add('inbound-enhanced-control');
            control.appendChild(button);
            item.dataset.inboundHelp = '1';
        });
    }

    function enhance(rootElement, options = {}) {
        if (!rootElement || typeof rootElement.querySelectorAll !== 'function') return;
        enhanceLabels(rootElement);
        enhanceHelpButtons(rootElement, options);
    }

    registerHelpItems();

    return {
        HELP_ITEMS,
        enhance,
        escapeHtml,
        getBilingualLabel,
        getHelpByLabel,
        normalizeLabel,
        renderHelpContent,
        validateInbound,
    };
});
