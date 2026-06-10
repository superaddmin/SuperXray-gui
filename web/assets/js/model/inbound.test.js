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
            randomSecret(length) {
                return `secret:${length}`;
            },
            randomShortIds() {
                return ['0123456789abcdef'];
            },
            randomShadowsocksPassword(method) {
                return `password:${method || 'default'}`;
            },
            normalizeShadowsocksMethod(method = '') {
                return String(method || '').trim().toLowerCase().replace(/_/g, '-');
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

test('Trojan and Hysteria default credentials use the unified strong secret generator', () => {
    const { Inbound } = loadInboundModel();

    const trojan = new Inbound.TrojanSettings.Trojan();
    const hysteria = new Inbound.HysteriaSettings.Hysteria();

    assert.equal(trojan.password, 'secret:32');
    assert.equal(hysteria.auth, 'secret:32');
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

test('legacy Shadowsocks settings canonicalize uppercase cipher aliases before save', () => {
    const { Inbound, SSMethods } = loadInboundModel();

    const settings = Inbound.ShadowsocksSettings.fromJson({
        method: 'CHACHA20_POLY1305',
        password: 'stale-server-password',
        network: 'tcp,udp',
        clients: [
            {
                method: 'CHACHA20_POLY1305',
                email: 'legacy@example',
                password: 'client-password',
                enable: true,
            },
        ],
    });

    const json = settings.toJson();

    assert.equal(settings.method, SSMethods.CHACHA20_POLY1305);
    assert.equal(json.method, SSMethods.CHACHA20_POLY1305);
    assert.equal(json.password, undefined);
    assert.equal(json.clients[0].method, SSMethods.CHACHA20_POLY1305);
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

test('protocol edit client lists let operators edit client email', () => {
    const protocolForms = [
        'web/html/form/protocol/vmess.html',
        'web/html/form/protocol/vless.html',
        'web/html/form/protocol/trojan.html',
        'web/html/form/protocol/shadowsocks.html',
        'web/html/form/protocol/hysteria.html',
    ];

    for (const file of protocolForms) {
        const form = fs.readFileSync(file, 'utf8');

        assert.match(form, /<a-input\s+v-model\.trim="client\.email"/, `${file} should expose an editable client email input`);
        assert.doesNotMatch(form, /<td>\s*\[\[\s*client\.email\s*\]\]\s*<\/td>/, `${file} should not render client email as read-only text`);
    }
});

test('protocol edit client lists let operators edit password credentials', () => {
    const cases = [
        { file: 'web/html/form/protocol/trojan.html', field: 'password' },
        { file: 'web/html/form/protocol/shadowsocks.html', field: 'password' },
        { file: 'web/html/form/protocol/hysteria.html', field: 'auth' },
    ];

    for (const item of cases) {
        const form = fs.readFileSync(item.file, 'utf8');

        assert.match(form, new RegExp(`<a-input\\s+v-model\\.trim="client\\.${item.field}"`), `${item.file} should expose editable client ${item.field}`);
        assert.doesNotMatch(form, new RegExp(`<td>\\s*\\[\\[\\s*client\\.${item.field}\\s*\\]\\]\\s*<\\/td>`), `${item.file} should not render client ${item.field} as read-only text`);
    }
});

test('protocol settings include edited client email in save payload', () => {
    const { Inbound, Protocols } = loadInboundModel();
    const cases = [
        { protocol: Protocols.VMESS, collection: 'vmesses' },
        { protocol: Protocols.VLESS, collection: 'vlesses' },
        { protocol: Protocols.TROJAN, collection: 'trojans' },
        { protocol: Protocols.SHADOWSOCKS, collection: 'shadowsockses' },
        { protocol: Protocols.HYSTERIA, collection: 'hysterias' },
    ];

    for (const item of cases) {
        const settings = Inbound.Settings.getSettings(item.protocol);

        settings[item.collection][0].email = 'operator@example.com';

        const payload = JSON.parse(settings.toString());

        assert.equal(payload.clients[0].email, 'operator@example.com', `${item.protocol} should serialize edited client email`);
    }
});

test('new TCP finalmask fragment defaults to a non-zero length range', () => {
    const { Inbound } = loadInboundModel();

    const inbound = new Inbound();
    inbound.stream.addTcpMask('fragment');

    assert.equal(inbound.stream.finalmask.tcp[0].settings.length, '100-200');
});

test('Hysteria defaults to h3 without uTLS fingerprint', () => {
    const { Inbound, Protocols } = loadInboundModel();

    const inbound = new Inbound();
    inbound.protocol = Protocols.HYSTERIA;

    assert.deepEqual(Array.from(inbound.stream.tls.alpn), ['h3']);
    assert.equal(inbound.stream.tls.settings.fingerprint, '');
});

test('Hysteria share link exports UDP hop ports as mport', () => {
    const { Inbound, Protocols } = loadInboundModel();

    const inbound = new Inbound();
    inbound.protocol = Protocols.HYSTERIA;
    inbound.settings.version = 2;
    inbound.stream.finalmask.enableQuicParams = true;
    inbound.stream.finalmask.quicParams.udpHop = { ports: '40000-45000', interval: '5-10' };

    const link = inbound.genHysteriaLink('203.0.113.20', 443, 'hy2-hop', 'hy2-auth');
    const url = new URL(link);

    assert.equal(url.searchParams.get('alpn'), 'h3');
    assert.equal(url.searchParams.get('mport'), '40000-45000');
    assert.equal(url.searchParams.has('fp'), false);
});

test('Hysteria share link URI-encodes auth userinfo', () => {
    const { Inbound, Protocols } = loadInboundModel();

    const inbound = new Inbound();
    inbound.protocol = Protocols.HYSTERIA;
    inbound.settings.version = 2;

    const link = inbound.genHysteriaLink(
        '203.0.113.20',
        443,
        'hy2-auth',
        'hy2/auth=with padding',
    );

    assert.match(link, /^hysteria2:\/\/hy2%2Fauth%3Dwith%20padding@203\.0\.113\.20:443/);
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
