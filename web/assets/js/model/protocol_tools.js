(function (root, factory) {
    const api = factory();
    if (typeof module !== 'undefined' && module.exports) {
        module.exports = api;
    }
    if (root) {
        root.ProtocolToolGenerator = api.ProtocolToolGenerator;
        root.WarpMatrixBuilder = api.WarpMatrixBuilder;
        root.XrayOutboundTools = api.XrayOutboundTools;
    }
})(typeof window !== 'undefined' ? window : globalThis, function () {
    function deepClone(value) {
        if (value === null || value === undefined) {
            return value;
        }
        return JSON.parse(JSON.stringify(value));
    }

    function asInt(value, fallback) {
        const parsed = Number.parseInt(value, 10);
        return Number.isFinite(parsed) ? parsed : fallback;
    }

    function cleanString(value, fallback = '') {
        if (value === null || value === undefined) {
            return fallback;
        }
        const text = String(value).trim();
        return text.length > 0 ? text : fallback;
    }

    function jsonText(value) {
        return JSON.stringify(value, null, 2);
    }

    function hasValue(value) {
        return value !== undefined && value !== null && String(value).length > 0;
    }

    function addressWithPort(address, port) {
        if (!hasValue(address) && !hasValue(port)) {
            return null;
        }
        if (!hasValue(port)) {
            return String(address);
        }
        if (!hasValue(address)) {
            return `:${port}`;
        }
        return `${address}:${port}`;
    }

    function addressesFromServers(servers) {
        if (!Array.isArray(servers)) {
            return [];
        }
        return servers
            .map(server => addressWithPort(server && server.address, server && server.port))
            .filter(Boolean);
    }

    function addressesFromVnext(vnext) {
        if (!Array.isArray(vnext)) {
            return [];
        }
        return vnext
            .map(server => addressWithPort(server && server.address, server && server.port))
            .filter(Boolean);
    }

    const XrayOutboundTools = {
        findOutboundAddresses(outbound = {}) {
            const settings = outbound.settings || {};
            switch (outbound.protocol) {
            case 'vmess':
            case 'vless': {
                const vnextAddresses = addressesFromVnext(settings.vnext);
                if (vnextAddresses.length > 0) {
                    return vnextAddresses;
                }
                const directAddress = addressWithPort(settings.address, settings.port);
                return directAddress ? [directAddress] : [];
            }
            case 'http':
            case 'socks':
            case 'shadowsocks':
            case 'trojan': {
                const serverAddresses = addressesFromServers(settings.servers);
                if (serverAddresses.length > 0) {
                    return serverAddresses;
                }
                const directAddress = addressWithPort(settings.address, settings.port);
                return directAddress ? [directAddress] : [];
            }
            case 'dns':
            case 'hysteria': {
                const directAddress = addressWithPort(settings.address, settings.port);
                return directAddress ? [directAddress] : [];
            }
            case 'wireguard':
                return Array.isArray(settings.peers)
                    ? settings.peers.map(peer => peer && peer.endpoint).filter(Boolean)
                    : [];
            default:
                return [];
            }
        },
    };

    function makeURLParams(params) {
        const query = new URLSearchParams();
        Object.keys(params).forEach(key => {
            const value = params[key];
            if (value !== undefined && value !== null && String(value).length > 0) {
                query.set(key, String(value));
            }
        });
        return query.toString();
    }

    function buildVlessShareLink(input, network, security) {
        const server = cleanString(input.server, 'example.com');
        const port = asInt(input.port, 443);
        const uuid = cleanString(input.uuid, '11111111-1111-4111-8111-111111111111');
        const sni = cleanString(input.sni, server);
        const params = {
            type: network,
            security,
            encryption: 'none',
            flow: 'xtls-rprx-vision',
            sni,
            fp: cleanString(input.fingerprint, 'chrome'),
            pbk: cleanString(input.publicKey, 'PUBLIC_KEY'),
            sid: cleanString(input.shortId, ''),
            spx: cleanString(input.spiderX, '/'),
        };

        if (network === 'xhttp') {
            params.path = cleanString(input.path, '/xhttp');
            params.mode = cleanString(input.mode, 'auto');
        } else if (network === 'ws') {
            params.path = cleanString(input.path, '/');
            params.host = cleanString(input.host, sni);
        }

        return `vless://${uuid}@${server}:${port}?${makeURLParams(params)}#${encodeURIComponent(cleanString(input.remark, 'vless'))}`;
    }

    function buildVlessOutbound(input, network, security) {
        const server = cleanString(input.server, 'example.com');
        const port = asInt(input.port, 443);
        const uuid = cleanString(input.uuid, '11111111-1111-4111-8111-111111111111');
        const sni = cleanString(input.sni, server);
        const streamSettings = {
            network,
            security,
        };

        if (security === 'reality') {
            streamSettings.realitySettings = {
                serverName: sni,
                fingerprint: cleanString(input.fingerprint, 'chrome'),
                publicKey: cleanString(input.publicKey, 'PUBLIC_KEY'),
                shortId: cleanString(input.shortId, ''),
                spiderX: cleanString(input.spiderX, '/'),
            };
        } else if (security === 'tls') {
            streamSettings.tlsSettings = {
                serverName: sni,
                alpn: ['h2', 'http/1.1'],
            };
        }

        if (network === 'xhttp') {
            streamSettings.xhttpSettings = {
                path: cleanString(input.path, '/xhttp'),
                host: cleanString(input.host, sni),
                mode: cleanString(input.mode, 'auto'),
            };
        } else if (network === 'ws') {
            streamSettings.wsSettings = {
                path: cleanString(input.path, '/'),
                host: cleanString(input.host, sni),
            };
        }

        return {
            protocol: 'vless',
            tag: cleanString(input.tag, 'proxy'),
            settings: {
                vnext: [{
                    address: server,
                    port,
                    users: [{
                        id: uuid,
                        encryption: 'none',
                        flow: 'xtls-rprx-vision',
                    }],
                }],
            },
            streamSettings,
        };
    }

    function buildTrojanOutbound(input) {
        const server = cleanString(input.server, 'example.com');
        const sni = cleanString(input.sni, server);
        return {
            protocol: 'trojan',
            tag: cleanString(input.tag, 'proxy'),
            settings: {
                servers: [{
                    address: server,
                    port: asInt(input.port, 443),
                    password: cleanString(input.password, 'change-me'),
                }],
            },
            streamSettings: {
                network: 'tcp',
                security: 'tls',
                tlsSettings: {
                    serverName: sni,
                    alpn: ['h2', 'http/1.1'],
                },
            },
        };
    }

    function buildShadowsocks2022Outbound(input) {
        return {
            protocol: 'shadowsocks',
            tag: cleanString(input.tag, 'proxy'),
            settings: {
                servers: [{
                    address: cleanString(input.server, 'example.com'),
                    port: asInt(input.port, 443),
                    method: cleanString(input.method, '2022-blake3-aes-256-gcm'),
                    password: cleanString(input.password, 'SERVER_KEY:CLIENT_KEY'),
                }],
            },
            streamSettings: {
                network: 'tcp',
                security: 'none',
            },
        };
    }

    function buildHysteria2Outbound(input) {
        const server = cleanString(input.server, 'example.com');
        const sni = cleanString(input.sni, server);
        return {
            protocol: 'hysteria',
            tag: cleanString(input.tag, 'proxy'),
            settings: {
                version: 2,
                address: server,
                port: asInt(input.port, 443),
            },
            streamSettings: {
                network: 'hysteria',
                security: 'tls',
                tlsSettings: {
                    serverName: sni,
                    alpn: ['h3'],
                },
                hysteriaSettings: {
                    version: 2,
                    auth: cleanString(input.password, 'change-me'),
                },
            },
        };
    }

    function generateXrayCombo(input, outbound, shareLink, summary) {
        return {
            combo: input.combo,
            runtime: 'xray',
            saveToXray: true,
            summary,
            clientOutbound: jsonText(outbound),
            shareLink,
            notice: 'This output is compatible with Xray outbound JSON and can be used as a client-side config snippet.',
        };
    }

    function generateSingBoxTuic(input) {
        const server = cleanString(input.server, 'example.com');
        const outbound = {
            type: 'tuic',
            tag: cleanString(input.tag, 'tuic-out'),
            server,
            server_port: asInt(input.port, 443),
            uuid: cleanString(input.uuid, '11111111-1111-4111-8111-111111111111'),
            password: cleanString(input.password, 'change-me'),
            congestion_control: cleanString(input.congestionControl, 'cubic'),
            udp_relay_mode: 'native',
            zero_rtt_handshake: false,
            heartbeat: '10s',
            tls: {
                enabled: true,
                server_name: cleanString(input.sni, server),
                insecure: false,
            },
        };
        return {
            combo: input.combo,
            runtime: 'sing-box',
            saveToXray: false,
            summary: 'TUIC external sing-box outbound',
            singBoxOutbound: jsonText(outbound),
            notice: 'TUIC is not supported as an Xray inbound in this panel build; use this as an external sing-box config snippet.',
        };
    }

    function generateSingBoxAnyTLS(input) {
        const server = cleanString(input.server, 'example.com');
        const outbound = {
            type: 'anytls',
            tag: cleanString(input.tag, 'anytls-out'),
            server,
            server_port: asInt(input.port, 443),
            password: cleanString(input.password, 'change-me'),
            idle_session_check_interval: '30s',
            idle_session_timeout: '30s',
            min_idle_session: 5,
            tls: {
                enabled: true,
                server_name: cleanString(input.sni, server),
                insecure: false,
            },
        };
        return {
            combo: input.combo,
            runtime: 'sing-box',
            saveToXray: false,
            summary: 'AnyTLS external sing-box outbound',
            singBoxOutbound: jsonText(outbound),
            notice: 'AnyTLS is not supported as an Xray inbound in this panel build; use this as an external sing-box config snippet.',
        };
    }

    const ProtocolToolGenerator = {
        comboPresets: [
            { value: 'vless-reality-vision', label: 'VLESS Reality Vision', runtime: 'xray', saveToXray: true },
            { value: 'vless-xhttp-reality', label: 'VLESS XHTTP Reality Vision', runtime: 'xray', saveToXray: true },
            { value: 'vless-ws-tls', label: 'VLESS WS TLS', runtime: 'xray', saveToXray: true },
            { value: 'trojan-tcp-tls', label: 'Trojan TCP TLS', runtime: 'xray', saveToXray: true },
            { value: 'shadowsocks-2022', label: 'Shadowsocks 2022', runtime: 'xray', saveToXray: true },
            { value: 'hysteria2-tls', label: 'Hysteria2 TLS', runtime: 'xray', saveToXray: true },
            { value: 'tuic-singbox', label: 'TUIC sing-box', runtime: 'sing-box', saveToXray: false },
            { value: 'anytls-singbox', label: 'AnyTLS sing-box', runtime: 'sing-box', saveToXray: false },
        ],

        generateArgo(input = {}) {
            const mode = cleanString(input.mode, 'quick');
            const originUrl = cleanString(input.originUrl, 'http://localhost:2053');

            if (mode === 'fixed') {
                const token = cleanString(input.token, '<CLOUDFLARE_TUNNEL_TOKEN>');
                const tunnelName = cleanString(input.tunnelName, 'superxray');
                const command = `cloudflared tunnel run --token ${token}`;
                return {
                    mode,
                    originUrl,
                    command,
                    systemd: [
                        '[Unit]',
                        `Description=Cloudflare Tunnel for ${tunnelName}`,
                        'After=network-online.target',
                        'Wants=network-online.target',
                        '',
                        '[Service]',
                        'TimeoutStartSec=0',
                        'Type=simple',
                        `ExecStart=/usr/local/bin/cloudflared tunnel run --token ${token}`,
                        'Restart=on-failure',
                        'RestartSec=5s',
                        '',
                        '[Install]',
                        'WantedBy=multi-user.target',
                    ].join('\n'),
                    compose: [
                        'services:',
                        '  cloudflared:',
                        '    image: cloudflare/cloudflared:latest',
                        '    restart: unless-stopped',
                        `    # host command: cloudflared tunnel run --token ${token}`,
                        `    command: tunnel run --token ${token}`,
                    ].join('\n'),
                    externalProxy: {
                        dest: '<fixed-tunnel-host>',
                        port: 443,
                        forceTls: 'tls',
                        remark: tunnelName,
                    },
                    notice: 'Token is generated into this output only and is not submitted to the backend by Protocol Tools.',
                };
            }

            return {
                mode: 'quick',
                originUrl,
                command: `cloudflared tunnel --url ${originUrl}`,
                externalProxy: {
                    dest: '<trycloudflare-host>',
                    port: 443,
                    forceTls: 'tls',
                    remark: 'trycloudflare',
                },
                notice: 'Quick Tunnels are for testing and development. Use a fixed tunnel for production.',
            };
        },

        generateCombo(input = {}) {
            const combo = cleanString(input.combo, 'vless-reality-vision');
            const normalizedInput = { ...input, combo };
            switch (combo) {
            case 'vless-reality-vision': {
                const outbound = buildVlessOutbound(normalizedInput, 'tcp', 'reality');
                return generateXrayCombo(
                    normalizedInput,
                    outbound,
                    buildVlessShareLink(normalizedInput, 'tcp', 'reality'),
                    'VLESS over TCP + Reality + Vision',
                );
            }
            case 'vless-xhttp-reality': {
                const outbound = buildVlessOutbound(normalizedInput, 'xhttp', 'reality');
                return generateXrayCombo(
                    normalizedInput,
                    outbound,
                    buildVlessShareLink(normalizedInput, 'xhttp', 'reality'),
                    'VLESS over XHTTP + Reality + Vision',
                );
            }
            case 'vless-ws-tls': {
                const outbound = buildVlessOutbound(normalizedInput, 'ws', 'tls');
                outbound.settings.vnext[0].users[0].flow = '';
                return generateXrayCombo(
                    normalizedInput,
                    outbound,
                    buildVlessShareLink({ ...normalizedInput, publicKey: undefined, shortId: undefined }, 'ws', 'tls'),
                    'VLESS over WebSocket + TLS',
                );
            }
            case 'trojan-tcp-tls': {
                const outbound = buildTrojanOutbound(normalizedInput);
                const server = cleanString(normalizedInput.server, 'example.com');
                const port = asInt(normalizedInput.port, 443);
                const password = encodeURIComponent(cleanString(normalizedInput.password, 'change-me'));
                const sni = cleanString(normalizedInput.sni, server);
                return generateXrayCombo(
                    normalizedInput,
                    outbound,
                    `trojan://${password}@${server}:${port}?security=tls&type=tcp&sni=${encodeURIComponent(sni)}#trojan`,
                    'Trojan over TCP + TLS',
                );
            }
            case 'shadowsocks-2022': {
                const outbound = buildShadowsocks2022Outbound(normalizedInput);
                return generateXrayCombo(
                    normalizedInput,
                    outbound,
                    'ss://<base64(method:server-key:client-key)>@<server>:<port>#shadowsocks-2022',
                    'Shadowsocks 2022 outbound template',
                );
            }
            case 'hysteria2-tls': {
                const outbound = buildHysteria2Outbound(normalizedInput);
                const server = cleanString(normalizedInput.server, 'example.com');
                const port = asInt(normalizedInput.port, 443);
                const password = encodeURIComponent(cleanString(normalizedInput.password, 'change-me'));
                const sni = cleanString(normalizedInput.sni, server);
                return generateXrayCombo(
                    normalizedInput,
                    outbound,
                    `hysteria2://${password}@${server}:${port}?security=tls&sni=${encodeURIComponent(sni)}#hysteria2`,
                    'Hysteria2 over TLS',
                );
            }
            case 'tuic-singbox':
                return generateSingBoxTuic(normalizedInput);
            case 'anytls-singbox':
                return generateSingBoxAnyTLS(normalizedInput);
            default:
                return {
                    combo,
                    runtime: 'unknown',
                    saveToXray: false,
                    notice: `Unknown combo: ${combo}`,
                };
            }
        },
    };

    function isWarpTag(tag) {
        return tag === 'warp' || (typeof tag === 'string' && tag.startsWith('warp-'));
    }

    const WarpMatrixBuilder = {
        matrixOptions: [
            { tag: 'warp', label: 'WARP default', domainStrategy: 'ForceIP' },
            { tag: 'warp-ipv4', label: 'WARP IPv4', domainStrategy: 'ForceIPv4' },
            { tag: 'warp-ipv6', label: 'WARP IPv6', domainStrategy: 'ForceIPv6' },
            {
                tag: 'warp-openai',
                label: 'WARP OpenAI',
                domainStrategy: 'ForceIP',
                rule: { type: 'field', outboundTag: 'warp-openai', domain: ['geosite:openai'] },
            },
        ],

        buildOutbounds(baseSettings, selectedTags = ['warp']) {
            const base = deepClone(baseSettings) || {};
            return selectedTags
                .map(tag => this.matrixOptions.find(option => option.tag === tag))
                .filter(Boolean)
                .map(option => {
                    const settings = deepClone(base) || {};
                    settings.domainStrategy = option.domainStrategy;
                    return {
                        tag: option.tag,
                        protocol: 'wireguard',
                        settings,
                    };
                });
        },

        buildRules(selectedTags = ['warp']) {
            return selectedTags
                .map(tag => this.matrixOptions.find(option => option.tag === tag))
                .filter(option => option && option.rule)
                .map(option => deepClone(option.rule));
        },

        applyMatrix(templateSettings = {}, baseSettings, selectedTags = ['warp']) {
            const currentOutbounds = Array.isArray(templateSettings.outbounds) ? templateSettings.outbounds : [];
            const currentRouting = templateSettings.routing || {};
            const currentRules = Array.isArray(currentRouting.rules) ? currentRouting.rules : [];

            const outbounds = currentOutbounds.filter(outbound => !isWarpTag(outbound && outbound.tag));
            outbounds.push(...this.buildOutbounds(baseSettings, selectedTags));

            const routing = { ...deepClone(currentRouting), rules: currentRules.filter(rule => !isWarpTag(rule && rule.outboundTag)) };
            routing.rules.push(...this.buildRules(selectedTags));

            return {
                ...deepClone(templateSettings),
                outbounds,
                routing,
            };
        },

        isWarpTag,
    };

    return {
        ProtocolToolGenerator,
        WarpMatrixBuilder,
        XrayOutboundTools,
    };
});
