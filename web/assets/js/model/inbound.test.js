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
            randomShadowsocksPassword() {
                return 'password';
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

    vm.runInNewContext(`${source}\nglobalThis.Inbound = Inbound; globalThis.Protocols = Protocols;`, sandbox);
    return {
        Inbound: sandbox.Inbound,
        Protocols: sandbox.Protocols,
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
