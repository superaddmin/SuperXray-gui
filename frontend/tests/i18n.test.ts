import assert from 'node:assert/strict';
import test from 'node:test';

import {
  DEFAULT_LOCALE,
  getLanguageButtonLabel,
  getNextLocale,
  getRouteTitle,
  getTranslationAuditReport,
  translate,
  translateDomText,
} from '../src/i18n/messages.ts';

test('defaults to Chinese and can toggle between Chinese and English', () => {
  assert.equal(DEFAULT_LOCALE, 'zh-CN');
  assert.equal(getLanguageButtonLabel('zh-CN'), '中文');
  assert.equal(getLanguageButtonLabel('en-US'), 'English');
  assert.equal(getNextLocale('zh-CN'), 'en-US');
  assert.equal(getNextLocale('en-US'), 'zh-CN');
});

test('translates primary navigation and route titles', () => {
  assert.equal(translate('nav.dashboard', 'zh-CN'), '仪表盘');
  assert.equal(translate('nav.dashboard', 'en-US'), 'Dashboard');
  assert.equal(getRouteTitle('cores', 'zh-CN'), '内核实例');
  assert.equal(getRouteTitle('cores', 'en-US'), 'Core Instances');
});

test('translates xray workspace and gateway mvp actions without dom fallback', () => {
  assert.equal(translate('action.moreActions', 'zh-CN'), '更多操作');
  assert.equal(translate('status.closeNav', 'zh-CN'), '关闭导航');
  assert.equal(translate('xray.workspace.gateway', 'zh-CN'), 'Gateway 出口');
  assert.equal(translate('xray.gateway.generateConfig', 'zh-CN'), '生成 Xray 配置');
  assert.equal(translate('xray.gateway.copyManifest', 'zh-CN'), '复制登记清单');
  assert.equal(translate('xray.gateway.downloadManifest', 'zh-CN'), '下载登记清单');
  assert.equal(translate('xray.gateway.listenHost', 'zh-CN'), 'Xray 监听主机');
  assert.equal(translate('xray.gateway.manifestHost', 'zh-CN'), 'Gateway 登记主机');
  assert.equal(translate('xray.gateway.previewProfiles', 'zh-CN'), '配置档');
  assert.equal(translate('xray.gateway.strategyLabel', 'zh-CN'), '网络策略标签');
  assert.equal(translate('xray.status.currentVersion', 'zh-CN'), '当前版本');
  assert.equal(translate('xray.status.invalidJson', 'zh-CN'), 'JSON 无效');
  assert.equal(translate('xray.status.legacyCompatible', 'zh-CN'), '旧版兼容');
  assert.equal(translate('xray.gateway.validStrategyRequired', 'zh-CN'), '请选择有效网络策略后再导出登记清单。');
});

test('translates dashboard generated labels and dynamic hints', () => {
  assert.equal(translateDomText('Overview', 'zh-CN'), '概览');
  assert.equal(translateDomText('TYPE', 'zh-CN'), '类型');
  assert.equal(translateDomText('No data', 'zh-CN'), '暂无数据');
  assert.equal(translateDomText('0 enabled', 'zh-CN'), '0 已启用');
  assert.equal(translateDomText('Update geoip.dat', 'zh-CN'), '更新 geoip.dat');
  assert.equal(translateDomText('版本 -', 'en-US'), 'Version -');
  assert.equal(translateDomText('swap', 'zh-CN'), 'swap');
  assert.equal(translateDomText('登录', 'en-US'), 'Sign in');
});

test('translates inbound drawer and rule editor terms reviewed for bilingual UI', () => {
  const reviewedTerms = [
    ['New Inbound', '新建入站', 'New Inbound'],
    [
      'Manage Xray listeners, clients, live activity, traffic counters, and sharing tools.',
      '管理 Xray 监听、客户端、在线状态、流量计数和分享工具。',
      'Manage Xray listeners, clients, live activity, traffic counters, and sharing tools.',
    ],
    ['Import JSON', '导入 JSON', 'Import JSON'],
    ['Total', '总数', 'Total'],
    ['Legacy Xray inbounds', '旧版 Xray 入站', 'Legacy Xray inbounds'],
    ['Active listeners', '活动监听', 'Active listeners'],
    ['Live activity from Xray', '来自 Xray 的在线活动', 'Live activity from Xray'],
    ['Configured users', '已配置用户', 'Configured users'],
    ['All protocols', '全部协议', 'All protocols'],
    ['All states', '全部状态', 'All states'],
    ['Search inbounds', '搜索入站', 'Search inbounds'],
    ['Inbound', '入站', 'Inbound'],
    ['Address', '地址', 'Address'],
    ['Edit Inbound', '编辑入站', 'Edit Inbound'],
    ['Protocol', '协议', 'Protocol'],
    ['Remark', '备注', 'Remark'],
    ['Listen', '监听地址', 'Listen'],
    ['Traffic Limit GB', '流量限制 GB', 'Traffic Limit GB'],
    ['Expiry Timestamp', '到期时间戳', 'Expiry Timestamp'],
    ['Traffic Reset', '流量重置', 'Traffic Reset'],
    ['Enable', '启用', 'Enable'],
    ['Close', '关闭', 'Close'],
    ['Increase Value', '增加数值', 'Increase Value'],
    ['Decrease Value', '减少数值', 'Decrease Value'],
    ['Online Clients', '在线客户端', 'Online Clients'],
    ['Inbound counters', '入站流量计数', 'Inbound counters'],
    ['Transport', '传输', 'Transport'],
    ['Network', '网络', 'Network'],
    ['TCP Proxy Protocol', 'TCP 代理协议', 'TCP Proxy Protocol'],
    ['TCP Header', 'TCP 头部', 'TCP Header'],
    ['Sockopt Enabled', '启用 Sockopt', 'Sockopt Enabled'],
    ['Sync JSON', '同步 JSON', 'Sync JSON'],
    ['Settings JSON', '设置 JSON', 'Settings JSON'],
    ['Stream Settings JSON', '传输设置 JSON', 'Stream Settings JSON'],
    ['Sniffing JSON', '嗅探 JSON', 'Sniffing JSON'],
    ['Add Client', '添加客户端', 'Add Client'],
    ['Share Link', '分享链接', 'Share Link'],
    ['Online', '在线', 'Online'],
    ['Offline', '离线', 'Offline'],
    ['never', '从不', 'never'],
  ] as const;

  for (const [source, zhCN, enUS] of reviewedTerms) {
    assert.equal(translateDomText(source, 'zh-CN'), zhCN);
    assert.equal(translateDomText(zhCN, 'en-US'), enUS);
  }

  assert.equal(translateDomText('Inbound 12', 'zh-CN'), '入站 12');
  assert.equal(translateDomText('Disable inbound demo', 'zh-CN'), '禁用入站 demo');
  assert.equal(translateDomText('禁用入站 demo', 'en-US'), 'Disable inbound demo');
  assert.equal(translateDomText('Enable inbound demo', 'zh-CN'), '启用入站 demo');
});

test('translates inbound subpage actions, beginner hints, and advanced transport fields', () => {
  const reviewedTerms = [
    ['Activity', '活动', 'Activity'],
    ['Online / IP Management', '在线 / IP 管理', 'Online / IP Management'],
    ['View IPs', '查看 IP', 'View IPs'],
    ['Clear IPs', '清空 IP', 'Clear IPs'],
    ['Users', '用户', 'Users'],
    ['Export Links', '导出链接', 'Export Links'],
    ['Reset Selected', '重置已选', 'Reset Selected'],
    ['Delete Selected', '删除已选', 'Delete Selected'],
    ['Reset All', '全部重置', 'Reset All'],
    ['Delete Depleted', '删除已耗尽', 'Delete Depleted'],
    ['Export', '导出', 'Export'],
    ['Expires', '过期时间', 'Expires'],
    ['Last Online', '最后在线', 'Last Online'],
    ['IP Records', 'IP 记录', 'IP Records'],
    ['Limit', '限制', 'Limit'],
    ['Usage', '用量', 'Usage'],
    ['Expiry', '到期时间', 'Expiry'],
    ['Server Private Key', '服务端私钥', 'Server Private Key'],
    ['Server Public Key', '服务端公钥', 'Server Public Key'],
    ['noKernelTun', '禁用内核 TUN', 'noKernelTun'],
    ['WS Path', 'WS 路径', 'WS Path'],
    ['WS Host', 'WS 主机', 'WS Host'],
    ['WS Heartbeat Period', 'WS 心跳周期', 'WS Heartbeat Period'],
    ['gRPC Service', 'gRPC 服务', 'gRPC Service'],
    ['gRPC Authority', 'gRPC Authority 头', 'gRPC Authority'],
    ['gRPC Multi Mode', 'gRPC 多路模式', 'gRPC Multi Mode'],
    ['HTTPUpgrade Path', 'HTTPUpgrade 路径', 'HTTPUpgrade Path'],
    ['HTTPUpgrade Host', 'HTTPUpgrade 主机', 'HTTPUpgrade Host'],
    ['XHTTP Path', 'XHTTP 路径', 'XHTTP Path'],
    ['XHTTP Host', 'XHTTP 主机', 'XHTTP Host'],
    ['XHTTP Mode', 'XHTTP 模式', 'XHTTP Mode'],
    ['No SSE Header', '无 SSE 头', 'No SSE Header'],
    ['Max Buffered Posts', '最大缓冲 POST 数', 'Max Buffered Posts'],
    ['Each Post Bytes', '单次 POST 字节', 'Each Post Bytes'],
    ['Stream Up Server Secs', '上行流服务器秒数', 'Stream Up Server Secs'],
    ['Padding Bytes', '填充字节', 'Padding Bytes'],
    ['Reject Unknown SNI', '拒绝未知 SNI', 'Reject Unknown SNI'],
    ['Disable System Root', '禁用系统根证书', 'Disable System Root'],
    ['Session Resumption', '会话恢复', 'Session Resumption'],
    ['ECH Server Keys', 'ECH 服务端密钥', 'ECH Server Keys'],
    ['ECH Config List', 'ECH 配置列表', 'ECH Config List'],
    ['Max Time Diff', '最大时间差', 'Max Time Diff'],
    ['ML-DSA-65 Seed', 'ML-DSA-65 种子', 'ML-DSA-65 Seed'],
    ['ML-DSA-65 Verify', 'ML-DSA-65 校验', 'ML-DSA-65 Verify'],
    ['TCP Fast Open', 'TCP 快速打开', 'TCP Fast Open'],
    ['Multipath TCP', '多路径 TCP', 'Multipath TCP'],
    ['Penetrate', '穿透', 'Penetrate'],
    ['V6 Only', '仅 IPv6', 'V6 Only'],
    ['Domain Strategy', '域名策略', 'Domain Strategy'],
    ['TCP Congestion', 'TCP 拥塞控制', 'TCP Congestion'],
    ['Dialer Proxy', '拨号代理', 'Dialer Proxy'],
    ['Interface Name', '网卡名称', 'Interface Name'],
    ['Trusted X-Forwarded-For', '可信 X-Forwarded-For', 'Trusted X-Forwarded-For'],
    ['Private Key', '私钥', 'Private Key'],
    ['Public Key', '公钥', 'Public Key'],
    ['Pre Shared Key', '预共享密钥', 'Pre Shared Key'],
    ['Allowed IPs', '允许的 IP', 'Allowed IPs'],
    ['Keep Alive', '保活间隔', 'Keep Alive'],
    ['IP Limit', 'IP 限制', 'IP Limit'],
    ['Reset Days', '重置天数', 'Reset Days'],
    ['Sub ID', '订阅 ID', 'Sub ID'],
    ['Comment', '注释', 'Comment'],
    [
      'Paste a legacy inbound JSON object. It will be imported through the existing Xray API so old UI remains readable.',
      '粘贴旧版入站 JSON 对象。系统会通过现有 Xray API 导入，确保旧 UI 仍可读取。',
      'Paste a legacy inbound JSON object. It will be imported through the existing Xray API so old UI remains readable.',
    ],
  ] as const;

  for (const [source, zhCN, enUS] of reviewedTerms) {
    assert.equal(translateDomText(source, 'zh-CN'), zhCN);
    assert.equal(translateDomText(zhCN, 'en-US'), enUS);
  }
});

test('translates child page form attributes with consistent locale state', () => {
  const attributeTerms = [
    ['Web listen address', 'Web 监听地址', 'Web Listen'],
    ['Web domain', 'Web 域名', 'Web Domain'],
    ['Web port', 'Web 端口', 'Web Port'],
    ['Base path', '基础路径', 'Base Path'],
    ['Certificate file', '证书文件', 'Certificate File'],
    ['Key file', '密钥文件', 'Key File'],
    ['Session max age', '会话最长时间', 'Session Max Age'],
    ['Page size', '分页大小', 'Page Size'],
    ['Expire diff', '过期差值', 'Expire Diff'],
    ['Traffic diff', '流量差值', 'Traffic Diff'],
    ['Remark model', '备注模型', 'Remark Model'],
    ['Datepicker mode', '日期选择器', 'Datepicker'],
    ['Time location', '时区位置', 'Time Location'],
  ] as const;

  for (const [source, zhCN, enUS] of attributeTerms) {
    assert.equal(translateDomText(source, 'zh-CN'), zhCN);
    assert.equal(translateDomText(zhCN, 'en-US'), enUS);
  }
});

test('audits reviewed translation entries for complete Chinese and English pairs', () => {
  const report = getTranslationAuditReport();
  assert.equal(report.missing.length, 0);
  assert.equal(report.duplicates.length, 0);
  assert.ok(report.messageEntries >= 100);
  assert.ok(report.reviewedEntries >= 80);
});
