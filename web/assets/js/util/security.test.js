const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

function loadHtmlUtil() {
    const source = fs.readFileSync('web/assets/js/util/index.js', 'utf8');
    const sandbox = {
        Blob,
        Intl,
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

    vm.runInNewContext(`${source}\nglobalThis.HtmlUtil = HtmlUtil;`, sandbox);
    return sandbox.HtmlUtil;
}

test('HtmlUtil.escape encodes markup before log HTML rendering', () => {
    const HtmlUtil = loadHtmlUtil();

    assert.equal(
        HtmlUtil.escape('<img src=x onerror=alert("x")>&\''),
        '&lt;img src=x onerror=alert(&quot;x&quot;)&gt;&amp;&#39;'
    );
});
