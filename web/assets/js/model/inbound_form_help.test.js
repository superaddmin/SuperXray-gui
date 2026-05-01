const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');

const InboundFormHelp = require('./inbound_form_help.js');

test('inbound form helper returns bilingual labels for core fields', () => {
    assert.equal(InboundFormHelp.getBilingualLabel('私钥'), '私钥 / Private Key');
    assert.equal(InboundFormHelp.getBilingualLabel('SNI'), 'SNI / 服务器名称');
    assert.equal(InboundFormHelp.getBilingualLabel('电子邮件'), '电子邮件 / Email');
});

test('inbound form helper normalizes noisy label text before lookup', () => {
    const help = InboundFormHelp.getHelpByLabel('  Target  ⟳ ');

    assert.equal(help.label.zh, '目标地址');
    assert.equal(help.label.en, 'Target');
    assert.match(help.description.zh, /Reality/);
    assert.match(help.description.en, /Reality/);
});

test('inbound form helper provides generic bilingual help for advanced fields', () => {
    const help = InboundFormHelp.getHelpByLabel('Route Mark');

    assert.equal(help.label.zh, '高级参数');
    assert.equal(help.label.en, 'Route Mark');
    assert.match(help.description.zh, /Xray 入站配置参数/);
    assert.match(help.description.en, /Xray inbound setting/);
});

test('inbound form helper blocks Reality creation when privateKey is empty', () => {
    const result = InboundFormHelp.validateInbound({
        stream: {
            security: 'reality',
            reality: {
                target: 'www.example.com:443',
                serverNames: 'www.example.com',
                shortIds: 'abcd1234',
                privateKey: '',
            },
        },
    });

    assert.equal(result.valid, false);
    assert.equal(result.fieldKey, 'privateKey');
    assert.match(result.message.zh, /Reality 私钥不能为空/);
    assert.match(result.message.en, /Reality privateKey is required/);
});

test('inbound form helper blocks TLS creation when certificate file fields are empty', () => {
    const result = InboundFormHelp.validateInbound({
        stream: {
            security: 'tls',
            tls: {
                certs: [
                    {
                        useFile: true,
                        certFile: '',
                        keyFile: '',
                    },
                ],
            },
        },
    });

    assert.equal(result.valid, false);
    assert.equal(result.fieldKey, 'tlsCertificate');
    assert.match(result.message.zh, /TLS 证书/);
    assert.match(result.message.en, /TLS certificate/);
});

test('inbound form helper blocks invalid Shadowsocks client email before saving', () => {
    const missing = InboundFormHelp.validateInbound({
        protocol: 'shadowsocks',
        settings: {
            shadowsockses: [{ email: '   ' }],
        },
    });

    assert.equal(missing.valid, false);
    assert.equal(missing.fieldKey, 'clientEmail');
    assert.match(missing.message.zh, /Shadowsocks 客户端 Email 不能为空/);
    assert.match(missing.message.en, /Shadowsocks client Email is required/);

    const unsafe = InboundFormHelp.validateInbound({
        protocol: 'shadowsocks',
        settings: {
            shadowsockses: [{ email: 'bad/user' }],
        },
    });

    assert.equal(unsafe.valid, false);
    assert.equal(unsafe.fieldKey, 'clientEmail');
    assert.match(unsafe.message.zh, /只能包含字母、数字、点、下划线、短横线和 @/);
    assert.match(unsafe.message.en, /letters, numbers, dots, underscores, hyphens, and @/);
});

test('inbound form helper accepts existing identifier-style Shadowsocks email values', () => {
    const result = InboundFormHelp.validateInbound({
        protocol: 'shadowsocks',
        settings: {
            shadowsockses: [{ email: 'user01' }],
        },
    });

    assert.equal(result.valid, true);
});

test('inbound form helper validates client email for other multi-user protocols', () => {
    const invalid = InboundFormHelp.validateInbound({
        protocol: 'vless',
        settings: {
            vlesses: [{ email: 'bad user' }],
        },
    });

    assert.equal(invalid.valid, false);
    assert.equal(invalid.fieldKey, 'clientEmail');
    assert.match(invalid.message.zh, /VLESS 客户端 Email/);
    assert.match(invalid.message.en, /VLESS client Email/);

    const valid = InboundFormHelp.validateInbound({
        protocol: 'trojan',
        settings: {
            trojans: [{ email: 'client-01@example.com' }],
        },
    });

    assert.equal(valid.valid, true);
});

test('add inbound client panels default to expanded so Flow is discoverable', () => {
    for (const file of [
        'web/html/form/protocol/vmess.html',
        'web/html/form/protocol/vless.html',
        'web/html/form/protocol/trojan.html',
        'web/html/form/protocol/shadowsocks.html',
        'web/html/form/protocol/hysteria.html',
    ]) {
        const source = fs.readFileSync(file, 'utf8');

        assert.match(source, /:default-active-key=['"]\['client'\]['"]/);
        assert.match(source, /<a-collapse-panel[^>]+key=['"]client['"]/);
    }
});

test('inbounds page loads inbound form helper before rendering inbound modal', () => {
    const page = fs.readFileSync('web/html/inbounds.html', 'utf8');

    const helperIndex = page.indexOf('assets/js/model/inbound_form_help.js');
    const modalIndex = page.indexOf('{{template "modals/inboundModal"}}');

    assert.notEqual(helperIndex, -1);
    assert.notEqual(modalIndex, -1);
    assert.ok(helperIndex < modalIndex);
});

test('inbound modal validates and enhances form through inbound form helper', () => {
    const modal = fs.readFileSync('web/html/modals/inbound_modal.html', 'utf8');

    assert.match(modal, /InboundFormHelp\.validateInbound\(inModal\.inbound\)/);
    assert.match(modal, /InboundFormHelp\.enhance\(/);
    assert.match(modal, /showInboundFieldHelp/);
});
