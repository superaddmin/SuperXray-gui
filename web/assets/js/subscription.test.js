const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');

test('shadowrocket subscription link uses standard padded base64', () => {
    const source = fs.readFileSync('web/assets/js/subscription.js', 'utf8');

    assert.match(source, /const base64Url = btoa\(rawUrl\);/);
    assert.doesNotMatch(source, /\.replace\(['"]\+['"], ['"]-['"]\)/);
    assert.doesNotMatch(source, /\.replace\(['"]\/['"], ['"]_['"]\)/);
    assert.doesNotMatch(source, /\.replace\(['"]=+['"], ['"]['"]\)/);

    const rawUrl = 'https://example.com/sub?x=~A?flag=shadowrocket';
    const base64Url = Buffer.from(rawUrl, 'binary').toString('base64');
    const shadowrocketUrl = `shadowrocket://add/sub/${base64Url}?remark=Subscription`;
    const encoded = shadowrocketUrl
        .replace('shadowrocket://add/sub/', '')
        .replace('?remark=Subscription', '');

    assert.equal(Buffer.from(encoded, 'base64').toString('binary'), rawUrl);
    assert.match(encoded, /[+/=]/);
});
