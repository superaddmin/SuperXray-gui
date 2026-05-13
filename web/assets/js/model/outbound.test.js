const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

function loadOutboundModel() {
    const source = fs.readFileSync('web/assets/js/model/outbound.js', 'utf8');
    const sandbox = {
        Base64: { decode: value => Buffer.from(value, 'base64').toString('utf8') },
        console,
        ObjectUtil: {
            isArrEmpty(value) {
                return !Array.isArray(value) || value.length === 0;
            },
            isEmpty(value) {
                return value == null || value === '' || (Array.isArray(value) && value.length === 0);
            },
        },
        RandomUtil: {
            randomLowerAndNum(length) {
                return 'a'.repeat(length);
            },
            randomSeq(length) {
                return 'b'.repeat(length);
            },
        },
        Wireguard: {
            generateKeypair() {
                return { privateKey: 'private-key', publicKey: 'public-key' };
            },
        },
        URL,
        URLSearchParams,
    };

    vm.runInNewContext(`${source}\nglobalThis.Outbound = Outbound; globalThis.Protocols = Protocols;`, sandbox);
    return {
        Outbound: sandbox.Outbound,
        Protocols: sandbox.Protocols,
    };
}

test('socks5 proxy URI imports as socks outbound with credentials', () => {
    const { Outbound, Protocols } = loadOutboundModel();

    const outbound = Outbound.fromLink('socks5://user:pass@192.168.1.1:1080');

    assert.ok(outbound);
    assert.equal(outbound.protocol, Protocols.Socks);
    assert.equal(outbound.settings.address, '192.168.1.1');
    assert.equal(outbound.settings.port, 1080);
    assert.equal(outbound.settings.user, 'user');
    assert.equal(outbound.settings.pass, 'pass');
});

test('http proxy URI imports as http outbound without credentials', () => {
    const { Outbound, Protocols } = loadOutboundModel();

    const outbound = Outbound.fromLink('http://192.168.1.1:8080');

    assert.ok(outbound);
    assert.equal(outbound.protocol, Protocols.HTTP);
    assert.equal(outbound.settings.address, '192.168.1.1');
    assert.equal(outbound.settings.port, 8080);
    assert.equal(outbound.settings.user, '');
    assert.equal(outbound.settings.pass, '');
});

test('https proxy URI imports as http outbound with credentials and tls stream', () => {
    const { Outbound, Protocols } = loadOutboundModel();

    const outbound = Outbound.fromLink('https://user:pass@proxy.example.com:443');

    assert.ok(outbound);
    assert.equal(outbound.protocol, Protocols.HTTP);
    assert.equal(outbound.settings.address, 'proxy.example.com');
    assert.equal(outbound.settings.port, 443);
    assert.equal(outbound.settings.user, 'user');
    assert.equal(outbound.settings.pass, 'pass');
    assert.equal(outbound.stream.network, 'tcp');
    assert.equal(outbound.stream.security, 'tls');
});

test('socks5h proxy URI imports as socks outbound', () => {
    const { Outbound, Protocols } = loadOutboundModel();

    const outbound = Outbound.fromLink('socks5h://user:pass@proxy.example.com:1080');

    assert.ok(outbound);
    assert.equal(outbound.protocol, Protocols.Socks);
    assert.equal(outbound.settings.address, 'proxy.example.com');
    assert.equal(outbound.settings.port, 1080);
    assert.equal(outbound.settings.user, 'user');
    assert.equal(outbound.settings.pass, 'pass');
});

test('proxy URI imports IPv6 host', () => {
    const { Outbound, Protocols } = loadOutboundModel();

    const outbound = Outbound.fromLink('socks5://user:pass@[2001:db8::1]:1080');

    assert.ok(outbound);
    assert.equal(outbound.protocol, Protocols.Socks);
    assert.equal(outbound.settings.address, '[2001:db8::1]');
    assert.equal(outbound.settings.port, 1080);
});

test('proxy URI decodes encoded credentials', () => {
    const { Outbound, Protocols } = loadOutboundModel();

    const outbound = Outbound.fromLink('http://user%40mail:p%40ss%3Aword@example.com:8080');

    assert.ok(outbound);
    assert.equal(outbound.protocol, Protocols.HTTP);
    assert.equal(outbound.settings.user, 'user@mail');
    assert.equal(outbound.settings.pass, 'p@ss:word');
});

test('proxy URI import rejects missing or invalid ports', () => {
    const { Outbound } = loadOutboundModel();

    assert.equal(Outbound.fromLink('socks5://127.0.0.1'), null);
    assert.equal(Outbound.fromLink('http://127.0.0.1:70000'), null);
});
