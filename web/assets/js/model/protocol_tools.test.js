const assert = require('node:assert/strict');
const test = require('node:test');

const {
    ProtocolToolGenerator,
    WarpMatrixBuilder,
} = require('./protocol_tools.js');

test('generates quick tunnel command', () => {
    const result = ProtocolToolGenerator.generateArgo({
        mode: 'quick',
        originUrl: 'http://localhost:2053',
    });

    assert.equal(result.mode, 'quick');
    assert.match(result.command, /cloudflared tunnel --url http:\/\/localhost:2053/);
    assert.equal(result.externalProxy.dest, '<trycloudflare-host>');
    assert.equal(result.externalProxy.port, 443);
});

test('generates fixed tunnel command without persisting token', () => {
    const result = ProtocolToolGenerator.generateArgo({
        mode: 'fixed',
        originUrl: 'http://127.0.0.1:2053',
        token: 'eyJ.test-token',
        tunnelName: 'superxray',
    });

    assert.equal(result.mode, 'fixed');
    assert.match(result.command, /cloudflared tunnel run --token eyJ\.test-token/);
    assert.match(result.systemd, /ExecStart=\/usr\/local\/bin\/cloudflared tunnel run --token eyJ\.test-token/);
    assert.match(result.compose, /cloudflared tunnel run --token eyJ\.test-token/);
    assert.equal(result.externalProxy.dest, '<fixed-tunnel-host>');
});

test('marks tuic and anytls as external only', () => {
    for (const combo of ['tuic-singbox', 'anytls-singbox']) {
        const result = ProtocolToolGenerator.generateCombo({
            combo,
            server: 'example.com',
            port: 443,
            uuid: '11111111-1111-4111-8111-111111111111',
            password: 'secret',
            sni: 'example.com',
        });

        assert.equal(result.saveToXray, false);
        assert.equal(result.runtime, 'sing-box');
        assert.match(result.notice, /not supported as an Xray inbound/);
        assert.doesNotThrow(() => JSON.parse(result.singBoxOutbound));
    }
});

test('generates xray vless reality vision combo', () => {
    const result = ProtocolToolGenerator.generateCombo({
        combo: 'vless-reality-vision',
        server: 'example.com',
        port: 443,
        uuid: '11111111-1111-4111-8111-111111111111',
        publicKey: 'pub',
        shortId: 'abcd',
        sni: 'www.microsoft.com',
    });

    assert.equal(result.saveToXray, true);
    const outbound = JSON.parse(result.clientOutbound);
    assert.equal(outbound.protocol, 'vless');
    assert.equal(outbound.streamSettings.security, 'reality');
    assert.equal(outbound.settings.vnext[0].users[0].flow, 'xtls-rprx-vision');
    assert.match(result.shareLink, /^vless:\/\/11111111-1111-4111-8111-111111111111@example\.com:443\?/);
});

test('generates xray vless xhttp reality combo', () => {
    const result = ProtocolToolGenerator.generateCombo({
        combo: 'vless-xhttp-reality',
        server: 'example.com',
        port: 443,
        uuid: '11111111-1111-4111-8111-111111111111',
        publicKey: 'pub',
        shortId: 'abcd',
        sni: 'www.microsoft.com',
        path: '/xhttp',
    });

    assert.equal(result.saveToXray, true);
    const outbound = JSON.parse(result.clientOutbound);
    assert.equal(outbound.streamSettings.network, 'xhttp');
    assert.equal(outbound.streamSettings.xhttpSettings.path, '/xhttp');
    assert.match(result.shareLink, /type=xhttp/);
});

test('builds warp matrix and preserves non-warp config', () => {
    const base = {
        mtu: 1420,
        secretKey: 'private',
        address: ['172.16.0.2/32', '2606:4700:110:abcd::1/128'],
        reserved: [1, 2, 3],
        peers: [{ publicKey: 'peer', endpoint: '162.159.192.1:2408' }],
        noKernelTun: false,
    };
    const existing = {
        outbounds: [
            { tag: 'direct', protocol: 'freedom' },
            { tag: 'warp-old', protocol: 'wireguard' },
        ],
        routing: {
            rules: [
                { outboundTag: 'direct', domain: ['geosite:cn'] },
                { outboundTag: 'warp-old', domain: ['geosite:openai'] },
            ],
        },
    };

    const result = WarpMatrixBuilder.applyMatrix(existing, base, ['warp', 'warp-ipv4', 'warp-openai']);

    assert.deepEqual(result.outbounds.map(o => o.tag), ['direct', 'warp', 'warp-ipv4', 'warp-openai']);
    assert.equal(result.outbounds[2].settings.domainStrategy, 'ForceIPv4');
    assert.equal(result.routing.rules.length, 2);
    assert.equal(result.routing.rules[0].outboundTag, 'direct');
    assert.equal(result.routing.rules[1].outboundTag, 'warp-openai');
    assert.deepEqual(result.routing.rules[1].domain, ['geosite:openai']);
});
