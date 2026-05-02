const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

function loadHtmlUtil() {
    const source = fs.readFileSync('web/assets/js/util/index.js', 'utf8');
    const sandbox = {
        Blob,
        Intl,
        SSMethods: {
            CHACHA20_POLY1305: 'chacha20-poly1305',
            CHACHA20_IETF_POLY1305: 'chacha20-ietf-poly1305',
            BLAKE3_AES_128_GCM: '2022-blake3-aes-128-gcm',
            BLAKE3_AES_256_GCM: '2022-blake3-aes-256-gcm',
        },
        URL,
        Vue: { prototype: { $message: { error() {}, success() {} } } },
        console,
        document: {
            cookie: '',
            documentElement: { scrollTop: 0 },
        },
        window: {
            addEventListener() {},
            atob: (value) => Buffer.from(value, 'base64').toString('binary'),
            btoa: (value) => Buffer.from(value, 'binary').toString('base64'),
            crypto: {
                getRandomValues(values) {
                    values.fill(1);
                    return values;
                },
                randomUUID() {
                    return '00000000-0000-4000-8000-000000000000';
                },
            },
            document: {
                createElement() {
                    return { click() {}, remove() {}, style: {} };
                },
                documentElement: { scrollTop: 0 },
            },
            innerWidth: 1024,
            location: { hostname: 'localhost', port: '', protocol: 'https:', reload() {} },
            navigator: { language: 'en-US' },
            pageYOffset: 0,
            removeEventListener() {},
        },
    };

    vm.runInNewContext(`${source}\nglobalThis.HtmlUtil = HtmlUtil; globalThis.RandomUtil = RandomUtil;`, sandbox);
    return {
        HtmlUtil: sandbox.HtmlUtil,
        RandomUtil: sandbox.RandomUtil,
        SSMethods: sandbox.SSMethods,
    };
}

test('HtmlUtil.escape encodes markup before log HTML rendering', () => {
    const { HtmlUtil } = loadHtmlUtil();

    assert.equal(
        HtmlUtil.escape('<img src=x onerror=alert("x")>&\''),
        '&lt;img src=x onerror=alert(&quot;x&quot;)&gt;&amp;&#39;'
    );
});

test('randomSecret generates a 32-byte URL-safe credential by default', () => {
    const { RandomUtil } = loadHtmlUtil();

    const password = RandomUtil.randomSecret();

    assert.equal(password, 'AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE');
    assert.equal(password.length, 43);
    assert.doesNotMatch(password, /[+/=]/);
});

test('legacy Shadowsocks AEAD methods generate 32-byte URL-safe client passwords', () => {
    const { RandomUtil, SSMethods } = loadHtmlUtil();

    const password = RandomUtil.randomShadowsocksPassword(SSMethods.CHACHA20_IETF_POLY1305);

    assert.equal(password, 'AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE');
    assert.equal(password.length, 43);
    assert.doesNotMatch(password, /[+/=]/);
});

test('Shadowsocks method aliases are canonicalized before credential rules apply', () => {
    const { RandomUtil, SSMethods } = loadHtmlUtil();

    assert.equal(RandomUtil.normalizeShadowsocksMethod('CHACHA20_POLY1305'), SSMethods.CHACHA20_POLY1305);

    const password = RandomUtil.randomShadowsocksPassword('CHACHA20_POLY1305');

    assert.equal(password, 'AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE');
    assert.equal(password.length, 43);
    assert.doesNotMatch(password, /[+/=]/);
});
