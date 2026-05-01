const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

function loadInboundModel() {
    const source = fs.readFileSync('web/assets/js/model/inbound.js', 'utf8');
    const sandbox = {
        Base64: { encode: value => Buffer.from(value).toString('base64') },
        console,
        moment: value => ({ valueOf: () => value }),
        NumberFormatter: {
            toFixed(num, n) {
                const scale = Math.pow(10, n);
                return Math.floor(num * scale) / scale;
            },
        },
        ObjectUtil: {
            clone(value) {
                return JSON.parse(JSON.stringify(value));
            },
            isEmpty(value) {
                return value == null || value === '' || (Array.isArray(value) && value.length === 0);
            },
        },
        RandomUtil: {
            randomInteger() {
                return 12345;
            },
            randomLowerAndNum(length) {
                return 'a'.repeat(length);
            },
            randomSeq(length) {
                return 'b'.repeat(length);
            },
            randomShortIds() {
                return ['0123456789abcdef'];
            },
            randomShadowsocksPassword(method) {
                return `password:${method || 'default'}`;
            },
            randomUUID() {
                return '00000000-0000-4000-8000-000000000000';
            },
        },
        SizeFormatter: { ONE_GB: 1024 * 1024 * 1024 },
        URI: {
            encode: encodeURIComponent,
        },
        Wireguard: {
            generateKeypair() {
                return { privateKey: 'private-key', publicKey: 'public-key' };
            },
        },
        URL,
        URLSearchParams,
    };

    vm.runInNewContext(`${source}\nglobalThis.Inbound = Inbound; globalThis.Protocols = Protocols; globalThis.SSMethods = SSMethods;`, sandbox);
    return {
        Inbound: sandbox.Inbound,
        Protocols: sandbox.Protocols,
        SSMethods: sandbox.SSMethods,
    };
}

test('default client settings serialize Telegram ID as a number', () => {
    const { Inbound, Protocols } = loadInboundModel();

    const settings = Inbound.Settings.getSettings(Protocols.VLESS).toJson();

    assert.equal(settings.clients[0].tgId, 0);
    assert.equal(typeof settings.clients[0].tgId, 'number');
});

test('blank Telegram ID input serializes as zero', () => {
    const { Inbound } = loadInboundModel();
    const client = new Inbound.VLESSSettings.VLESS();

    client.tgId = '';

    assert.equal(client.toJson().tgId, 0);
    assert.equal(typeof client.toJson().tgId, 'number');
});

test('default inbound settings serialize clients as an array', () => {
    const { Inbound } = loadInboundModel();

    const inbound = new Inbound();
    const settings = JSON.parse(inbound.settings.toString());

    assert.ok(Array.isArray(settings.clients));
    assert.equal(typeof settings.clients, 'object');
    assert.notEqual(typeof settings.clients, 'string');
});

test('new Shadowsocks clients generate passwords for the selected method', () => {
    const { Inbound, SSMethods } = loadInboundModel();

    const client = new Inbound.ShadowsocksSettings.Shadowsocks(SSMethods.CHACHA20_IETF_POLY1305);

    assert.equal(client.password, `password:${SSMethods.CHACHA20_IETF_POLY1305}`);
});

test('default Shadowsocks 2022 settings generate server and client keys for the selected method', () => {
    const { Inbound, Protocols, SSMethods } = loadInboundModel();

    const settings = Inbound.Settings.getSettings(Protocols.SHADOWSOCKS);

    assert.equal(settings.method, SSMethods.BLAKE3_AES_256_GCM);
    assert.equal(settings.password, `password:${SSMethods.BLAKE3_AES_256_GCM}`);
    assert.equal(settings.shadowsockses[0].password, `password:${SSMethods.BLAKE3_AES_256_GCM}`);
});

test('legacy Shadowsocks settings serialize clients with the inbound method', () => {
    const { Inbound, SSMethods } = loadInboundModel();

    const settings = Inbound.ShadowsocksSettings.fromJson({
        method: SSMethods.CHACHA20_IETF_POLY1305,
        password: 'stale-server-password',
        network: 'tcp,udp',
        clients: [
            {
                email: 'legacy@example',
                password: 'client-password',
                enable: true,
            },
        ],
    });

    const json = settings.toJson();

    assert.equal(json.password, undefined);
    assert.equal(json.clients[0].method, SSMethods.CHACHA20_IETF_POLY1305);
    assert.equal(json.clients[0].password, 'client-password');
});

test('single-user Shadowsocks 2022 chacha settings drop stale clients when serialized', () => {
    const { Inbound, SSMethods } = loadInboundModel();

    const settings = Inbound.ShadowsocksSettings.fromJson({
        method: SSMethods.BLAKE3_CHACHA20_POLY1305,
        password: 'server-key',
        network: 'tcp,udp',
        clients: [
            {
                method: SSMethods.CHACHA20_IETF_POLY1305,
                email: 'stale@example',
                password: 'stale-client-key',
                enable: true,
            },
        ],
    });

    const json = settings.toJson();

    assert.equal(json.password, 'server-key');
    assert.equal(json.clients.length, 0);
});

test('Shadowsocks helpers tolerate missing method in stored settings', () => {
    const { Inbound, Protocols } = loadInboundModel();

    const settings = Inbound.ShadowsocksSettings.fromJson({ clients: [] });
    const inbound = new Inbound(12345, '', Protocols.SHADOWSOCKS, settings);

    assert.equal(inbound.isSS2022, false);
    assert.equal(inbound.isSSMultiUser, true);
});

test('bulk Shadowsocks client creation uses the inbound method for generated passwords', () => {
    const modal = fs.readFileSync('web/html/modals/client_bulk_modal.html', 'utf8');

    assert.match(
        modal,
        /RandomUtil\.randomShadowsocksPassword\(clientsBulkModal\.inbound\.settings\.method\)/
    );
    assert.match(modal, /clientsBulkModal\.inbound\.isSS2022\s+\?\s+''\s+:\s+clientsBulkModal\.inbound\.settings\.method/);
    assert.doesNotMatch(modal, /shadowsockses\[0\]\.method/);
});

test('single Shadowsocks client creation uses the inbound method for generated passwords', () => {
    const modal = fs.readFileSync('web/html/modals/client_modal.html', 'utf8');

    assert.match(modal, /RandomUtil\.randomShadowsocksPassword\(inbound\.settings\.method\)/);
    assert.match(modal, /inbound\.isSS2022\s+\?\s+''\s+:\s+inbound\.settings\.method/);
    assert.doesNotMatch(modal, /clients\[0\]\.method/);
});

test('WireGuard peers preserve subscription metadata', () => {
    const { Inbound } = loadInboundModel();

    const peer = new Inbound.WireguardSettings.Peer(
        'peer-private',
        'peer-public',
        'peer-psk',
        ['10.0.0.2'],
        25,
        'wg@example',
        false,
        'sub-123'
    );
    const json = peer.toJson();

    assert.equal(json.email, 'wg@example');
    assert.equal(json.enable, false);
    assert.equal(json.subId, 'sub-123');

    const restored = Inbound.WireguardSettings.Peer.fromJson(json);
    assert.equal(restored.email, 'wg@example');
    assert.equal(restored.enable, false);
    assert.equal(restored.subId, 'sub-123');
});

test('add inbound form shows connection choices before protocol client fields', () => {
    const form = fs.readFileSync('web/html/form/inbound.html', 'utf8');

    const streamNetworkIndex = form.indexOf('{{template "form/streamNetwork"}}');
    const tlsSecurityIndex = form.indexOf('{{template "form/tlsSecurity"}}');
    const vlessClientIndex = form.indexOf('{{template "form/vless"}}');

    assert.notEqual(streamNetworkIndex, -1);
    assert.notEqual(tlsSecurityIndex, -1);
    assert.notEqual(vlessClientIndex, -1);
    assert.ok(streamNetworkIndex < vlessClientIndex);
    assert.ok(tlsSecurityIndex < vlessClientIndex);
});

test('VLESS protocol advanced fields are hidden when TLS or Reality is selected', () => {
    const vlessForm = fs.readFileSync('web/html/form/protocol/vless.html', 'utf8');

    assert.match(vlessForm, /!inbound\.stream\.isTls\s+&&\s+!inbound\.stream\.isReality/);
    assert.doesNotMatch(vlessForm, /inbound\.stream\.isTLS/);
});

test('inbounds page serializes client payloads with JSON.stringify', () => {
    const page = fs.readFileSync('web/html/inbounds.html', 'utf8');

    assert.doesNotMatch(page, /settings:\s*['"`]\{"clients":\s*\[/);
    assert.doesNotMatch(page, /clients\.toString\(\)/);
    assert.match(page, /JSON\.stringify\(\{\s*clients:/);
});

test('inbounds page encodes path parameters that can contain special characters', () => {
    const page = fs.readFileSync('web/html/inbounds.html', 'utf8');

    assert.match(page, /encodeURIComponent\(clientId\)/);
    assert.match(page, /encodeURIComponent\(client\.email\)/);
});

test('inbounds page does not leak high-risk locals into global scope', () => {
    const page = fs.readFileSync('web/html/inbounds.html', 'utf8');
    const highRiskNames = [
        'to_inbound',
        'clients',
        'clientStats',
        'now',
        'dbInbound',
        'clientId',
        'newDbInbound',
        'rootInbound',
        'newInbound',
        'inbound',
        'remainedSeconds',
        'resetSeconds',
    ];

    for (const name of highRiskNames) {
        const bareAssignment = new RegExp(`(^|[^\\w.$:-])${name}\\s*=(?!>)`, 'm');
        const offenders = page
            .split(/\r?\n/)
            .map((line, index) => ({ line, index: index + 1 }))
            .filter(({ line }) => bareAssignment.test(line))
            .filter(({ line }) => !new RegExp(`\\b(?:const|let|var)\\s+[^;]*\\b${name}\\b`).test(line));

        assert.deepEqual(offenders, [], `${name} should be declared before assignment`);
    }
});
