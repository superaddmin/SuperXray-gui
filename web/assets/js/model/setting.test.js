const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

function loadSettingModel() {
    const source = fs.readFileSync('web/assets/js/model/setting.js', 'utf8');
    const sandbox = {
        ObjectUtil: {
            cloneProps(target, sourceData) {
                for (const key of Object.keys(sourceData || {})) {
                    if (Object.prototype.hasOwnProperty.call(target, key)) {
                        target[key] = sourceData[key];
                    }
                }
            },
            equals(left, right) {
                return JSON.stringify(left) === JSON.stringify(right);
            },
        },
    };

    vm.runInNewContext(`${source}\nglobalThis.AllSetting = AllSetting;`, sandbox);
    return sandbox.AllSetting;
}

test('legacy settings model preserves panel outbound proxy setting', () => {
    const AllSetting = loadSettingModel();
    const panelProxy = 'socks5://user:pass@127.0.0.1:1080';

    const setting = new AllSetting({ panelProxy });

    assert.equal(setting.panelProxy, panelProxy);
    assert.equal(Object.prototype.hasOwnProperty.call(setting, 'panelProxy'), true);
});
