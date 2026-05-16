export type AppLocale = 'zh-CN' | 'en-US';

export const DEFAULT_LOCALE: AppLocale = 'zh-CN';
export const LOCALE_STORAGE_KEY = 'superxray.locale';

const messages = {
  'action.add': { 'zh-CN': '添加', 'en-US': 'Add' },
  'action.addResource': { 'zh-CN': '添加资源', 'en-US': 'Add Resource' },
  'action.apply': { 'zh-CN': '应用', 'en-US': 'Apply' },
  'action.copy': { 'zh-CN': '复制', 'en-US': 'Copy' },
  'action.copyLinks': { 'zh-CN': '复制链接', 'en-US': 'Copy Links' },
  'action.delete': { 'zh-CN': '删除', 'en-US': 'Delete' },
  'action.disableTwoFactor': { 'zh-CN': '禁用双因素', 'en-US': 'Disable Two Factor' },
  'action.download': { 'zh-CN': '下载', 'en-US': 'Download' },
  'action.edit': { 'zh-CN': '编辑', 'en-US': 'Edit' },
  'action.fillDefaults': { 'zh-CN': '填入默认值', 'en-US': 'Fill Defaults' },
  'action.format': { 'zh-CN': '格式化', 'en-US': 'Format' },
  'action.generateToken': { 'zh-CN': '生成令牌', 'en-US': 'Generate Token' },
  'action.import': { 'zh-CN': '导入', 'en-US': 'Import' },
  'action.install': { 'zh-CN': '安装', 'en-US': 'Install' },
  'action.moreActions': { 'zh-CN': '更多操作', 'en-US': 'More actions' },
  'action.openClash': { 'zh-CN': '打开 Clash', 'en-US': 'Open Clash' },
  'action.openJson': { 'zh-CN': '打开 JSON', 'en-US': 'Open JSON' },
  'action.openUri': { 'zh-CN': '打开 URI', 'en-US': 'Open URI' },
  'action.refresh': { 'zh-CN': '刷新', 'en-US': 'Refresh' },
  'action.refreshActivity': { 'zh-CN': '刷新活动', 'en-US': 'Refresh Activity' },
  'action.refreshTraffic': { 'zh-CN': '刷新流量', 'en-US': 'Refresh Traffic' },
  'action.reset': { 'zh-CN': '重置', 'en-US': 'Reset' },
  'action.resetAllTraffic': { 'zh-CN': '重置全部流量', 'en-US': 'Reset All Traffic' },
  'action.restart': { 'zh-CN': '重启', 'en-US': 'Restart' },
  'action.save': { 'zh-CN': '保存', 'en-US': 'Save' },
  'action.signIn': { 'zh-CN': '登录', 'en-US': 'Sign in' },
  'action.start': { 'zh-CN': '启动', 'en-US': 'Start' },
  'action.status': { 'zh-CN': '状态', 'en-US': 'Status' },
  'action.stop': { 'zh-CN': '停止', 'en-US': 'Stop' },
  'action.syncJson': { 'zh-CN': '同步 JSON', 'en-US': 'Sync JSON' },
  'action.testFirstOutbound': { 'zh-CN': '测试第一个出站', 'en-US': 'Test First Outbound' },
  'action.update': { 'zh-CN': '更新', 'en-US': 'Update' },
  'action.updateAll': { 'zh-CN': '全部更新', 'en-US': 'Update all' },
  'action.updateLogin': { 'zh-CN': '更新登录信息', 'en-US': 'Update Login' },
  'action.updateResources': { 'zh-CN': '更新资源', 'en-US': 'Update Resources' },
  'action.validate': { 'zh-CN': '校验', 'en-US': 'Validate' },
  'action.versions': { 'zh-CN': '版本', 'en-US': 'Versions' },
  'common.backup': { 'zh-CN': '备份', 'en-US': 'Backup' },
  'common.blocked': { 'zh-CN': '已拦截', 'en-US': 'Blocked' },
  'common.clients': { 'zh-CN': '客户端', 'en-US': 'Clients' },
  'common.configuration': { 'zh-CN': '配置', 'en-US': 'Configuration' },
  'common.connections': { 'zh-CN': '连接数', 'en-US': 'Connections' },
  'common.cpu': { 'zh-CN': 'CPU', 'en-US': 'CPU' },
  'common.credentials': { 'zh-CN': '凭据', 'en-US': 'Credentials' },
  'common.actions': { 'zh-CN': '操作', 'en-US': 'Actions' },
  'common.database': { 'zh-CN': '数据库', 'en-US': 'Database' },
  'common.detail': { 'zh-CN': '详情', 'en-US': 'Detail' },
  'common.direct': { 'zh-CN': '直连', 'en-US': 'Direct' },
  'common.disk': { 'zh-CN': '磁盘', 'en-US': 'Disk' },
  'common.enabled': { 'zh-CN': '已启用', 'en-US': 'Enabled' },
  'common.encrypted': { 'zh-CN': '已加密', 'en-US': 'Encrypted' },
  'common.experimental': { 'zh-CN': '实验性', 'en-US': 'Experimental' },
  'common.externalCode': { 'zh-CN': '外部代码', 'en-US': 'External Code' },
  'common.filter': { 'zh-CN': '筛选', 'en-US': 'Filter' },
  'common.formats': { 'zh-CN': '格式', 'en-US': 'Formats' },
  'common.instances': { 'zh-CN': '实例', 'en-US': 'Instances' },
  'common.language': { 'zh-CN': '语言', 'en-US': 'Language' },
  'common.lastUpdated': { 'zh-CN': '最后更新', 'en-US': 'Last Updated' },
  'common.lifecycle': { 'zh-CN': '生命周期', 'en-US': 'Lifecycle' },
  'common.loadAverage': { 'zh-CN': '平均负载', 'en-US': 'Load Average' },
  'common.memory': { 'zh-CN': '内存', 'en-US': 'Memory' },
  'common.metric': { 'zh-CN': '指标', 'en-US': 'Metric' },
  'common.noData': { 'zh-CN': '暂无数据', 'en-US': 'No data' },
  'common.observability': { 'zh-CN': '可观测性', 'en-US': 'Observability' },
  'common.outbound': { 'zh-CN': '出站', 'en-US': 'Outbound' },
  'common.overview': { 'zh-CN': '概览', 'en-US': 'Overview' },
  'common.panel': { 'zh-CN': '面板', 'en-US': 'Panel' },
  'common.panelRuntime': { 'zh-CN': '面板运行时', 'en-US': 'Panel Runtime' },
  'common.panelUptime': { 'zh-CN': '面板运行时长', 'en-US': 'Panel Uptime' },
  'common.providers': { 'zh-CN': '提供商', 'en-US': 'Providers' },
  'common.proxy': { 'zh-CN': '代理', 'en-US': 'Proxy' },
  'common.publicAccess': { 'zh-CN': '公开访问', 'en-US': 'Public Access' },
  'common.publicIp': { 'zh-CN': '公网 IP', 'en-US': 'Public IP' },
  'common.read': { 'zh-CN': '读取', 'en-US': 'Read' },
  'common.readOnly': { 'zh-CN': '只读', 'en-US': 'Read only' },
  'common.restore': { 'zh-CN': '恢复', 'en-US': 'Restore' },
  'common.routing': { 'zh-CN': '路由', 'en-US': 'Routing' },
  'common.running': { 'zh-CN': '运行中', 'en-US': 'Running' },
  'common.runtime': { 'zh-CN': '运行时', 'en-US': 'Runtime' },
  'common.security': { 'zh-CN': '安全', 'en-US': 'Security' },
  'common.selectedCore': { 'zh-CN': '已选内核', 'en-US': 'Selected Core' },
  'common.showInfo': { 'zh-CN': '显示信息', 'en-US': 'Show Info' },
  'common.subscription': { 'zh-CN': '订阅', 'en-US': 'Subscription' },
  'common.swap': { 'zh-CN': '交换分区', 'en-US': 'Swap' },
  'common.syslog': { 'zh-CN': '系统日志', 'en-US': 'Syslog' },
  'common.template': { 'zh-CN': '模板', 'en-US': 'Template' },
  'common.traffic': { 'zh-CN': '流量', 'en-US': 'Traffic' },
  'common.twoFactor': { 'zh-CN': '双因素', 'en-US': 'Two Factor' },
  'common.version': { 'zh-CN': '版本', 'en-US': 'Version' },
  'common.value': { 'zh-CN': '值', 'en-US': 'Value' },
  'common.write': { 'zh-CN': '写入', 'en-US': 'Write' },
  'common.xrayState': { 'zh-CN': 'Xray 状态', 'en-US': 'Xray State' },
  'dashboard.customGeo': { 'zh-CN': '自定义 Geo', 'en-US': 'Custom Geo' },
  'dashboard.description': {
    'zh-CN': '专为ChatGPT Claude Gemini设计的网络防封控管理工具。',
    'en-US': 'Live Xray health, traffic, clients, and geo maintenance.',
  },
  'dashboard.geoMaintenance': { 'zh-CN': 'Geo 维护', 'en-US': 'Geo Maintenance' },
  'dashboard.geoText': {
    'zh-CN': '无需离开新 UI 即可管理外部 geoip/geosite 资源。',
    'en-US': 'Manage external geoip/geosite resources without leaving the new UI.',
  },
  'field.alias': { 'zh-CN': '别名', 'en-US': 'Alias' },
  'field.announce': { 'zh-CN': '公告', 'en-US': 'Announce' },
  'field.apiServer': { 'zh-CN': 'API 服务器', 'en-US': 'API Server' },
  'field.autoCreate': { 'zh-CN': '自动创建', 'en-US': 'Auto Create' },
  'field.autoDelete': { 'zh-CN': '自动删除', 'en-US': 'Auto Delete' },
  'field.baseDn': { 'zh-CN': 'Base DN', 'en-US': 'Base DN' },
  'field.basePath': { 'zh-CN': '基础路径', 'en-US': 'Base Path' },
  'field.bindDn': { 'zh-CN': 'Bind DN', 'en-US': 'Bind DN' },
  'field.botToken': { 'zh-CN': 'Bot 令牌', 'en-US': 'Bot Token' },
  'field.certificateFile': { 'zh-CN': '证书文件', 'en-US': 'Certificate File' },
  'field.chatId': { 'zh-CN': 'Chat ID', 'en-US': 'Chat ID' },
  'field.clashPath': { 'zh-CN': 'Clash 路径', 'en-US': 'Clash Path' },
  'field.clashUri': { 'zh-CN': 'Clash URI', 'en-US': 'Clash URI' },
  'field.cpuThreshold': { 'zh-CN': 'CPU 阈值', 'en-US': 'CPU Threshold' },
  'field.currentPassword': { 'zh-CN': '当前密码', 'en-US': 'Current Password' },
  'field.currentUsername': { 'zh-CN': '当前用户名', 'en-US': 'Current Username' },
  'field.datepicker': { 'zh-CN': '日期选择器', 'en-US': 'Datepicker' },
  'field.defaultExpiryDays': { 'zh-CN': '默认过期天数', 'en-US': 'Default Expiry Days' },
  'field.defaultLimitIp': { 'zh-CN': '默认 IP 限制', 'en-US': 'Default Limit IP' },
  'field.defaultTotalGb': { 'zh-CN': '默认总流量 GB', 'en-US': 'Default Total GB' },
  'field.domain': { 'zh-CN': '域名', 'en-US': 'Domain' },
  'field.expireDiff': { 'zh-CN': '过期差值', 'en-US': 'Expire Diff' },
  'field.externalTrafficInform': {
    'zh-CN': '外部流量通知',
    'en-US': 'External Traffic Inform',
  },
  'field.externalTrafficUri': { 'zh-CN': '外部流量 URI', 'en-US': 'External Traffic URI' },
  'field.flagField': { 'zh-CN': '标记字段', 'en-US': 'Flag Field' },
  'field.host': { 'zh-CN': '主机', 'en-US': 'Host' },
  'field.inboundTags': { 'zh-CN': '入站标签', 'en-US': 'Inbound Tags' },
  'field.invertFlag': { 'zh-CN': '反转标记', 'en-US': 'Invert Flag' },
  'field.jsonFragment': { 'zh-CN': 'JSON 片段', 'en-US': 'JSON Fragment' },
  'field.jsonMux': { 'zh-CN': 'JSON Mux', 'en-US': 'JSON Mux' },
  'field.jsonNoises': { 'zh-CN': 'JSON 噪声', 'en-US': 'JSON Noises' },
  'field.jsonPath': { 'zh-CN': 'JSON 路径', 'en-US': 'JSON Path' },
  'field.jsonRules': { 'zh-CN': 'JSON 规则', 'en-US': 'JSON Rules' },
  'field.jsonUri': { 'zh-CN': 'JSON URI', 'en-US': 'JSON URI' },
  'field.keyFile': { 'zh-CN': '密钥文件', 'en-US': 'Key File' },
  'field.listen': { 'zh-CN': '监听', 'en-US': 'Listen' },
  'field.loginNotify': { 'zh-CN': '登录通知', 'en-US': 'Login Notify' },
  'field.newPassword': { 'zh-CN': '新密码', 'en-US': 'New Password' },
  'field.newUsername': { 'zh-CN': '新用户名', 'en-US': 'New Username' },
  'field.pageSize': { 'zh-CN': '分页大小', 'en-US': 'Page Size' },
  'field.password': { 'zh-CN': '密码', 'en-US': 'Password' },
  'field.port': { 'zh-CN': '端口', 'en-US': 'Port' },
  'field.profileUrl': { 'zh-CN': '配置 URL', 'en-US': 'Profile URL' },
  'field.proxy': { 'zh-CN': '代理', 'en-US': 'Proxy' },
  'field.remarkModel': { 'zh-CN': '备注模型', 'en-US': 'Remark Model' },
  'field.routingRules': { 'zh-CN': '路由规则', 'en-US': 'Routing Rules' },
  'field.runtime': { 'zh-CN': '运行时间', 'en-US': 'Runtime' },
  'field.sessionMaxAge': { 'zh-CN': '会话最长时间', 'en-US': 'Session Max Age' },
  'field.setupUri': { 'zh-CN': '设置 URI', 'en-US': 'Setup URI' },
  'field.subPath': { 'zh-CN': '订阅路径', 'en-US': 'Path' },
  'field.supportUrl': { 'zh-CN': '支持 URL', 'en-US': 'Support URL' },
  'field.syncCron': { 'zh-CN': '同步 Cron', 'en-US': 'Sync Cron' },
  'field.timeLocation': { 'zh-CN': '时区位置', 'en-US': 'Time Location' },
  'field.title': { 'zh-CN': '标题', 'en-US': 'Title' },
  'field.trafficDiff': { 'zh-CN': '流量差值', 'en-US': 'Traffic Diff' },
  'field.truthyValues': { 'zh-CN': '真值列表', 'en-US': 'Truthy Values' },
  'field.twoFactorCode': { 'zh-CN': '双因素验证码', 'en-US': 'Two-factor code' },
  'field.twoFactorToken': { 'zh-CN': '双因素令牌', 'en-US': 'Two Factor Token' },
  'field.type': { 'zh-CN': '类型', 'en-US': 'Type' },
  'field.updates': { 'zh-CN': '更新周期', 'en-US': 'Updates' },
  'field.url': { 'zh-CN': 'URL', 'en-US': 'URL' },
  'field.uri': { 'zh-CN': 'URI', 'en-US': 'URI' },
  'field.userAttr': { 'zh-CN': '用户属性', 'en-US': 'User Attr' },
  'field.userFilter': { 'zh-CN': '用户过滤器', 'en-US': 'User Filter' },
  'field.username': { 'zh-CN': '用户名', 'en-US': 'Username' },
  'field.vlessField': { 'zh-CN': 'VLESS 字段', 'en-US': 'VLESS Field' },
  'field.webDomain': { 'zh-CN': 'Web 域名', 'en-US': 'Web Domain' },
  'field.webListen': { 'zh-CN': 'Web 监听地址', 'en-US': 'Web Listen' },
  'field.webPort': { 'zh-CN': 'Web 端口', 'en-US': 'Web Port' },
  'inbounds.addInbound': { 'zh-CN': '新建入站', 'en-US': 'Add Inbound' },
  'inbounds.clientManager': { 'zh-CN': '客户端管理', 'en-US': 'Client Manager' },
  'inbounds.description': {
    'zh-CN': '用旧 API 管理 Xray 入站、客户端、分享链接和订阅片段。',
    'en-US':
      'Manage Xray inbounds, clients, sharing links, and subscription snippets through legacy APIs.',
  },
  'inbounds.importJson': { 'zh-CN': '导入 JSON', 'en-US': 'Import JSON' },
  'inbounds.streamSettingsForm': { 'zh-CN': '传输设置表单', 'en-US': 'Stream Settings Form' },
  'inbounds.wireguardSettings': { 'zh-CN': 'WireGuard 设置', 'en-US': 'WireGuard Settings' },
  'language.button.en': { 'zh-CN': 'English', 'en-US': 'English' },
  'language.button.zh': { 'zh-CN': '中文', 'en-US': '中文' },
  'language.toggleToEnglish': { 'zh-CN': '切换到 English', 'en-US': 'Switch to English' },
  'language.toggleToChinese': { 'zh-CN': '切换到中文', 'en-US': 'Switch to Chinese' },
  'login.description': {
    'zh-CN': '登录访问 SuperXray 运维控制台',
    'en-US': 'Sign in to access the SuperXray operations console',
  },
  'login.passwordPlaceholder': { 'zh-CN': '输入密码', 'en-US': 'Enter your password' },
  'login.passwordRequired': { 'zh-CN': '请输入密码', 'en-US': 'Password is required' },
  'login.signInFailed': { 'zh-CN': '登录失败', 'en-US': 'Sign in failed' },
  'login.title': { 'zh-CN': '安心掌控 Xray', 'en-US': 'Control Xray with Confidence' },
  'login.twoFactorRequired': {
    'zh-CN': '请输入双因素验证码',
    'en-US': 'Two-factor code is required',
  },
  'login.usernameRequired': { 'zh-CN': '请输入用户名', 'en-US': 'Username is required' },
  'logs.auto': { 'zh-CN': '自动', 'en-US': 'Auto' },
  'logs.description': {
    'zh-CN': '检查面板和 Xray 日志，支持筛选、导出与自动跟随。',
    'en-US': 'Inspect panel and Xray logs with filtering, export, and auto-follow controls.',
  },
  'logs.locked': { 'zh-CN': '锁定', 'en-US': 'Locked' },
  'logs.source': { 'zh-CN': '日志来源', 'en-US': 'Log source' },
  'nav.cores': { 'zh-CN': '内核', 'en-US': 'Cores' },
  'nav.dashboard': { 'zh-CN': '仪表盘', 'en-US': 'Dashboard' },
  'nav.inbounds': { 'zh-CN': '入站', 'en-US': 'Inbounds' },
  'nav.logs': { 'zh-CN': '日志', 'en-US': 'Logs' },
  'nav.settings': { 'zh-CN': '设置', 'en-US': 'Settings' },
  'nav.xray': { 'zh-CN': 'Xray', 'en-US': 'Xray' },
  'notFound.description': {
    'zh-CN': '请求的面板路由不存在。',
    'en-US': 'The requested panel route was not found.',
  },
  'route.cores': { 'zh-CN': '内核实例', 'en-US': 'Core Instances' },
  'route.dashboard': { 'zh-CN': '仪表盘', 'en-US': 'Dashboard' },
  'route.inbounds': { 'zh-CN': '入站', 'en-US': 'Inbounds' },
  'route.login': { 'zh-CN': '登录', 'en-US': 'Login' },
  'route.logs': { 'zh-CN': '日志', 'en-US': 'Logs' },
  'route.not-found': { 'zh-CN': '未找到', 'en-US': 'Not Found' },
  'route.panel': { 'zh-CN': '面板', 'en-US': 'Panel' },
  'route.settings': { 'zh-CN': '设置', 'en-US': 'Settings' },
  'route.xray': { 'zh-CN': 'Xray', 'en-US': 'Xray' },
  'settings.backupRestore': { 'zh-CN': '备份 / 恢复', 'en-US': 'Backup / Restore' },
  'settings.description': {
    'zh-CN': '管理面板安全、订阅端点、备份和旧 UI 兼容设置。',
    'en-US':
      'Manage panel security, subscription endpoints, backups, and legacy-compatible settings.',
  },
  'settings.endpoints': { 'zh-CN': '端点', 'en-US': 'Endpoints' },
  'settings.publicLinks': { 'zh-CN': '订阅公开链接', 'en-US': 'Subscription Public Links' },
  'settings.restartPanel': { 'zh-CN': '重启面板', 'en-US': 'Restart Panel' },
  'settings.twoFactorSetup': { 'zh-CN': '双因素设置', 'en-US': 'Two Factor Setup' },
  'status.darkTheme': { 'zh-CN': '深色主题', 'en-US': 'Dark theme' },
  'status.expandNav': { 'zh-CN': '展开导航', 'en-US': 'Expand navigation' },
  'status.collapseNav': { 'zh-CN': '收起导航', 'en-US': 'Collapse navigation' },
  'status.closeNav': { 'zh-CN': '关闭导航', 'en-US': 'Close navigation' },
  'status.phase10': { 'zh-CN': '阶段 10', 'en-US': 'Phase 10' },
  'xray.configEditor': { 'zh-CN': 'Xray 模板编辑器', 'en-US': 'Xray Template Editor' },
  'xray.description': {
    'zh-CN': '控制旧版 Xray 运行时、模板、版本、出站与提供商工具。',
    'en-US': 'Control the legacy Xray runtime, template, versions, outbounds, and provider tools.',
  },
  'xray.gateway.copyManifest': { 'zh-CN': '复制登记清单', 'en-US': 'Copy Manifest' },
  'xray.gateway.description': {
    'zh-CN':
      '生成面向 Gateway 容器登记的 Xray 兼容 SOCKS5 入站与 CSV 清单，不新增后端出口模型。',
    'en-US':
      'Generate Xray-compatible Gateway-facing SOCKS5 inbounds and a manifest without adding backend egress models.',
  },
  'xray.gateway.downloadManifest': { 'zh-CN': '下载登记清单', 'en-US': 'Download Manifest' },
  'xray.gateway.eyebrow': { 'zh-CN': 'Gateway', 'en-US': 'Gateway' },
  'xray.gateway.generateConfig': { 'zh-CN': '生成 Xray 配置', 'en-US': 'Generate Xray Config' },
  'xray.gateway.listenHost': { 'zh-CN': 'Xray 监听主机', 'en-US': 'Xray Listen Host' },
  'xray.gateway.manifestHost': { 'zh-CN': 'Gateway 登记主机', 'en-US': 'Gateway Manifest Host' },
  'xray.gateway.previewListenHost': { 'zh-CN': 'Xray 监听主机', 'en-US': 'Xray listen host' },
  'xray.gateway.previewManifestHost': {
    'zh-CN': 'Gateway 登记主机',
    'en-US': 'Gateway manifest host',
  },
  'xray.gateway.previewPorts': { 'zh-CN': '端口', 'en-US': 'ports' },
  'xray.gateway.previewProfiles': { 'zh-CN': '配置档', 'en-US': 'profiles' },
  'xray.gateway.strategyLabel': { 'zh-CN': '网络策略标签', 'en-US': 'Network Strategy Label' },
  'xray.gateway.title': { 'zh-CN': 'Gateway 出口 MVP', 'en-US': 'Gateway Egress MVP' },
  'xray.gateway.validStrategyRequired': {
    'zh-CN': '请选择有效网络策略后再导出登记清单。',
    'en-US': 'Select a valid network strategy before exporting manifest.',
  },
  'xray.outboundTools': { 'zh-CN': '出站工具', 'en-US': 'Outbound Tools' },
  'xray.runtimeControl': { 'zh-CN': 'Xray 运行控制', 'en-US': 'Xray Runtime Control' },
  'xray.status.currentVersion': { 'zh-CN': '当前版本', 'en-US': 'Current Version' },
  'xray.status.existingProcess': { 'zh-CN': '现有 Xray 进程', 'en-US': 'Existing Xray process' },
  'xray.status.existingService': { 'zh-CN': '现有 Xray 服务', 'en-US': 'Existing Xray service' },
  'xray.status.invalidJson': { 'zh-CN': 'JSON 无效', 'en-US': 'Invalid JSON' },
  'xray.status.legacyCompatible': { 'zh-CN': '旧版兼容', 'en-US': 'Legacy-compatible' },
  'xray.status.outboundTest': { 'zh-CN': '出站测试', 'en-US': 'Outbound Test' },
  'xray.status.savedLegacyTemplate': {
    'zh-CN': '随旧版模板保存',
    'en-US': 'Saved with legacy template',
  },
  'xray.status.unsavedChanges': { 'zh-CN': '存在未保存更改', 'en-US': 'Unsaved changes' },
  'xray.status.validJson': { 'zh-CN': 'JSON 有效', 'en-US': 'Valid JSON' },
  'xray.versionManagement': { 'zh-CN': 'Xray 版本管理', 'en-US': 'Xray Version Management' },
  'xray.workspace.dns': { 'zh-CN': 'DNS 策略', 'en-US': 'DNS Policy' },
  'xray.workspace.eyebrow': { 'zh-CN': '工作区', 'en-US': 'Workspace' },
  'xray.workspace.gateway': { 'zh-CN': 'Gateway 出口', 'en-US': 'Gateway Egress' },
  'xray.workspace.outbound': { 'zh-CN': '出站工具', 'en-US': 'Outbound Tools' },
  'xray.workspace.runtime': { 'zh-CN': '运行控制', 'en-US': 'Runtime Control' },
  'xray.workspace.structured': { 'zh-CN': '结构化配置', 'en-US': 'Structured Config' },
  'xray.workspace.template': { 'zh-CN': '模板编辑', 'en-US': 'Template Editor' },
  'xray.workspace.title': { 'zh-CN': 'Xray 工作区', 'en-US': 'Xray Workspace' },
  'xray.workspace.tools': { 'zh-CN': '协议工具', 'en-US': 'Protocol Tools' },
} as const satisfies Record<string, Record<AppLocale, string>>;

export type MessageKey = keyof typeof messages;

export const reviewedDomTranslations = [
  {
    source: 'New Inbound',
    zhCN: '新建入站',
    enUS: 'New Inbound',
    context: 'Inbound modal title',
  },
  {
    source: 'Manage Xray listeners, clients, live activity, traffic counters, and sharing tools.',
    zhCN: '管理 Xray 监听、客户端、在线状态、流量计数和分享工具。',
    enUS: 'Manage Xray listeners, clients, live activity, traffic counters, and sharing tools.',
    context: 'Inbounds page description',
  },
  { source: 'Import JSON', zhCN: '导入 JSON', enUS: 'Import JSON', context: 'Inbounds action' },
  { source: 'Total', zhCN: '总数', enUS: 'Total', context: 'Inbounds summary' },
  {
    source: 'Legacy Xray inbounds',
    zhCN: '旧版 Xray 入站',
    enUS: 'Legacy Xray inbounds',
    context: 'Inbounds summary',
  },
  {
    source: 'Active listeners',
    zhCN: '活动监听',
    enUS: 'Active listeners',
    context: 'Inbounds summary',
  },
  {
    source: 'Live activity from Xray',
    zhCN: '来自 Xray 的在线活动',
    enUS: 'Live activity from Xray',
    context: 'Inbounds summary',
  },
  {
    source: 'Configured users',
    zhCN: '已配置用户',
    enUS: 'Configured users',
    context: 'Inbounds summary',
  },
  { source: 'All protocols', zhCN: '全部协议', enUS: 'All protocols', context: 'Inbounds filter' },
  { source: 'All states', zhCN: '全部状态', enUS: 'All states', context: 'Inbounds filter' },
  {
    source: 'Search inbounds',
    zhCN: '搜索入站',
    enUS: 'Search inbounds',
    context: 'Inbounds filter',
  },
  { source: 'Inbound', zhCN: '入站', enUS: 'Inbound', context: 'Inbounds table' },
  { source: 'Address', zhCN: '地址', enUS: 'Address', context: 'Inbounds table' },
  {
    source: 'Edit Inbound',
    zhCN: '编辑入站',
    enUS: 'Edit Inbound',
    context: 'Inbound modal title',
  },
  { source: 'Inbound Details', zhCN: '入站详情', enUS: 'Inbound Details', context: 'Drawer title' },
  { source: 'Protocol', zhCN: '协议', enUS: 'Protocol', context: 'Inbound form' },
  { source: 'Remark', zhCN: '备注', enUS: 'Remark', context: 'Inbound form' },
  { source: 'Listen', zhCN: '监听地址', enUS: 'Listen', context: 'Inbound form' },
  { source: 'Port', zhCN: '端口', enUS: 'Port', context: 'Inbound form' },
  {
    source: 'Traffic Limit GB',
    zhCN: '流量限制 GB',
    enUS: 'Traffic Limit GB',
    context: 'Inbound form',
  },
  {
    source: 'Expiry Timestamp',
    zhCN: '到期时间戳',
    enUS: 'Expiry Timestamp',
    context: 'Inbound form',
  },
  { source: 'Traffic Reset', zhCN: '流量重置', enUS: 'Traffic Reset', context: 'Inbound form' },
  { source: 'Enable', zhCN: '启用', enUS: 'Enable', context: 'Inbound form' },
  { source: 'Close', zhCN: '关闭', enUS: 'Close', context: 'Ant modal action' },
  {
    source: 'Increase Value',
    zhCN: '增加数值',
    enUS: 'Increase Value',
    context: 'Ant input number action',
  },
  {
    source: 'Decrease Value',
    zhCN: '减少数值',
    enUS: 'Decrease Value',
    context: 'Ant input number action',
  },
  {
    source: 'Online Clients',
    zhCN: '在线客户端',
    enUS: 'Online Clients',
    context: 'Inbounds summary',
  },
  {
    source: 'Inbound counters',
    zhCN: '入站流量计数',
    enUS: 'Inbound counters',
    context: 'Inbounds summary',
  },
  { source: 'Transport', zhCN: '传输', enUS: 'Transport', context: 'Inbound detail' },
  { source: 'Disabled', zhCN: '已禁用', enUS: 'Disabled', context: 'Status label' },
  { source: 'Online', zhCN: '在线', enUS: 'Online', context: 'Client status' },
  { source: 'Offline', zhCN: '离线', enUS: 'Offline', context: 'Client status' },
  { source: 'Network', zhCN: '网络', enUS: 'Network', context: 'Stream settings' },
  {
    source: 'TCP Proxy Protocol',
    zhCN: 'TCP 代理协议',
    enUS: 'TCP Proxy Protocol',
    context: 'Stream settings',
  },
  { source: 'TCP Header', zhCN: 'TCP 头部', enUS: 'TCP Header', context: 'Stream settings' },
  { source: 'Sync JSON', zhCN: '同步 JSON', enUS: 'Sync JSON', context: 'JSON sync action' },
  {
    source: 'WS Proxy Protocol',
    zhCN: 'WS 代理协议',
    enUS: 'WS Proxy Protocol',
    context: 'Stream settings',
  },
  {
    source: 'HTTPUpgrade Proxy Protocol',
    zhCN: 'HTTPUpgrade 代理协议',
    enUS: 'HTTPUpgrade Proxy Protocol',
    context: 'Stream settings',
  },
  { source: 'TLS SNI', zhCN: 'TLS SNI', enUS: 'TLS SNI', context: 'TLS settings' },
  {
    source: 'TLS Min Version',
    zhCN: 'TLS 最低版本',
    enUS: 'TLS Min Version',
    context: 'TLS settings',
  },
  {
    source: 'TLS Max Version',
    zhCN: 'TLS 最高版本',
    enUS: 'TLS Max Version',
    context: 'TLS settings',
  },
  { source: 'TLS ALPN', zhCN: 'TLS ALPN', enUS: 'TLS ALPN', context: 'TLS settings' },
  { source: 'TLS Fingerprint', zhCN: 'TLS 指纹', enUS: 'TLS Fingerprint', context: 'TLS settings' },
  {
    source: 'Reality Show',
    zhCN: 'Reality 显示调试',
    enUS: 'Reality Show',
    context: 'Reality settings',
  },
  {
    source: 'Reality Xver',
    zhCN: 'Reality Xver',
    enUS: 'Reality Xver',
    context: 'Reality settings',
  },
  {
    source: 'Reality Target',
    zhCN: 'Reality 目标',
    enUS: 'Reality Target',
    context: 'Reality settings',
  },
  {
    source: 'Reality Server Names',
    zhCN: 'Reality 服务器名称',
    enUS: 'Reality Server Names',
    context: 'Reality settings',
  },
  {
    source: 'Reality Private Key',
    zhCN: 'Reality 私钥',
    enUS: 'Reality Private Key',
    context: 'Reality settings',
  },
  {
    source: 'Reality Short IDs',
    zhCN: 'Reality Short IDs',
    enUS: 'Reality Short IDs',
    context: 'Reality settings',
  },
  {
    source: 'Reality Public Key',
    zhCN: 'Reality 公钥',
    enUS: 'Reality Public Key',
    context: 'Reality settings',
  },
  {
    source: 'Reality SpiderX',
    zhCN: 'Reality SpiderX',
    enUS: 'Reality SpiderX',
    context: 'Reality settings',
  },
  {
    source: 'Min Client Version',
    zhCN: '最低客户端版本',
    enUS: 'Min Client Version',
    context: 'Reality settings',
  },
  {
    source: 'Max Client Version',
    zhCN: '最高客户端版本',
    enUS: 'Max Client Version',
    context: 'Reality settings',
  },
  {
    source: 'Hysteria2 Auth',
    zhCN: 'Hysteria2 认证',
    enUS: 'Hysteria2 Auth',
    context: 'Hysteria2 settings',
  },
  {
    source: 'UDP Idle Timeout',
    zhCN: 'UDP 空闲超时',
    enUS: 'UDP Idle Timeout',
    context: 'Hysteria2 settings',
  },
  {
    source: 'Sockopt Enabled',
    zhCN: '启用 Sockopt',
    enUS: 'Sockopt Enabled',
    context: 'Sockopt settings',
  },
  {
    source: 'Sockopt Proxy Protocol',
    zhCN: 'Sockopt 代理协议',
    enUS: 'Sockopt Proxy Protocol',
    context: 'Sockopt settings',
  },
  {
    source: 'Settings JSON',
    zhCN: '设置 JSON',
    enUS: 'Settings JSON',
    context: 'Advanced JSON editor',
  },
  {
    source: 'Stream Settings JSON',
    zhCN: '传输设置 JSON',
    enUS: 'Stream Settings JSON',
    context: 'Advanced JSON editor',
  },
  {
    source: 'Sniffing JSON',
    zhCN: '嗅探 JSON',
    enUS: 'Sniffing JSON',
    context: 'Advanced JSON editor',
  },
  { source: 'Activity', zhCN: '活动', enUS: 'Activity', context: 'Inbound drawer' },
  {
    source: 'Online / IP Management',
    zhCN: '在线 / IP 管理',
    enUS: 'Online / IP Management',
    context: 'Inbound drawer',
  },
  { source: 'View IPs', zhCN: '查看 IP', enUS: 'View IPs', context: 'Client IP action' },
  { source: 'Clear IPs', zhCN: '清空 IP', enUS: 'Clear IPs', context: 'Client IP action' },
  { source: 'Users', zhCN: '用户', enUS: 'Users', context: 'Inbound drawer' },
  { source: 'Export Links', zhCN: '导出链接', enUS: 'Export Links', context: 'Client action' },
  { source: 'Reset Selected', zhCN: '重置已选', enUS: 'Reset Selected', context: 'Client action' },
  {
    source: 'Delete Selected',
    zhCN: '删除已选',
    enUS: 'Delete Selected',
    context: 'Client action',
  },
  { source: 'Reset All', zhCN: '全部重置', enUS: 'Reset All', context: 'Client action' },
  {
    source: 'Delete Depleted',
    zhCN: '删除已耗尽',
    enUS: 'Delete Depleted',
    context: 'Client action',
  },
  { source: 'Export', zhCN: '导出', enUS: 'Export', context: 'Sharing preview' },
  { source: 'Expires', zhCN: '过期时间', enUS: 'Expires', context: 'Inbound summary' },
  { source: 'Last Online', zhCN: '最后在线', enUS: 'Last Online', context: 'Activity table' },
  { source: 'IP Records', zhCN: 'IP 记录', enUS: 'IP Records', context: 'Activity table' },
  { source: 'Limit', zhCN: '限制', enUS: 'Limit', context: 'Client table' },
  { source: 'Usage', zhCN: '用量', enUS: 'Usage', context: 'Client table' },
  { source: 'Expiry', zhCN: '到期时间', enUS: 'Expiry', context: 'Client table' },
  {
    source: 'Server Private Key',
    zhCN: '服务端私钥',
    enUS: 'Server Private Key',
    context: 'WireGuard settings',
  },
  {
    source: 'Server Public Key',
    zhCN: '服务端公钥',
    enUS: 'Server Public Key',
    context: 'WireGuard settings',
  },
  {
    source: 'noKernelTun',
    zhCN: '禁用内核 TUN',
    enUS: 'noKernelTun',
    context: 'WireGuard settings',
  },
  { source: 'WS Path', zhCN: 'WS 路径', enUS: 'WS Path', context: 'Stream settings' },
  { source: 'WS Host', zhCN: 'WS 主机', enUS: 'WS Host', context: 'Stream settings' },
  {
    source: 'WS Heartbeat Period',
    zhCN: 'WS 心跳周期',
    enUS: 'WS Heartbeat Period',
    context: 'Stream settings',
  },
  {
    source: 'gRPC Service',
    zhCN: 'gRPC 服务',
    enUS: 'gRPC Service',
    context: 'Stream settings',
  },
  {
    source: 'gRPC Authority',
    zhCN: 'gRPC Authority 头',
    enUS: 'gRPC Authority',
    context: 'Stream settings',
  },
  {
    source: 'gRPC Multi Mode',
    zhCN: 'gRPC 多路模式',
    enUS: 'gRPC Multi Mode',
    context: 'Stream settings',
  },
  {
    source: 'HTTPUpgrade Path',
    zhCN: 'HTTPUpgrade 路径',
    enUS: 'HTTPUpgrade Path',
    context: 'Stream settings',
  },
  {
    source: 'HTTPUpgrade Host',
    zhCN: 'HTTPUpgrade 主机',
    enUS: 'HTTPUpgrade Host',
    context: 'Stream settings',
  },
  {
    source: 'XHTTP Path',
    zhCN: 'XHTTP 路径',
    enUS: 'XHTTP Path',
    context: 'Stream settings',
  },
  {
    source: 'XHTTP Host',
    zhCN: 'XHTTP 主机',
    enUS: 'XHTTP Host',
    context: 'Stream settings',
  },
  {
    source: 'XHTTP Mode',
    zhCN: 'XHTTP 模式',
    enUS: 'XHTTP Mode',
    context: 'Stream settings',
  },
  {
    source: 'No SSE Header',
    zhCN: '无 SSE 头',
    enUS: 'No SSE Header',
    context: 'Stream settings',
  },
  {
    source: 'Max Buffered Posts',
    zhCN: '最大缓冲 POST 数',
    enUS: 'Max Buffered Posts',
    context: 'Stream settings',
  },
  {
    source: 'Each Post Bytes',
    zhCN: '单次 POST 字节',
    enUS: 'Each Post Bytes',
    context: 'Stream settings',
  },
  {
    source: 'Stream Up Server Secs',
    zhCN: '上行流服务器秒数',
    enUS: 'Stream Up Server Secs',
    context: 'Stream settings',
  },
  {
    source: 'Padding Bytes',
    zhCN: '填充字节',
    enUS: 'Padding Bytes',
    context: 'Stream settings',
  },
  {
    source: 'Reject Unknown SNI',
    zhCN: '拒绝未知 SNI',
    enUS: 'Reject Unknown SNI',
    context: 'TLS settings',
  },
  {
    source: 'Disable System Root',
    zhCN: '禁用系统根证书',
    enUS: 'Disable System Root',
    context: 'TLS settings',
  },
  {
    source: 'Session Resumption',
    zhCN: '会话恢复',
    enUS: 'Session Resumption',
    context: 'TLS settings',
  },
  {
    source: 'ECH Server Keys',
    zhCN: 'ECH 服务端密钥',
    enUS: 'ECH Server Keys',
    context: 'TLS settings',
  },
  {
    source: 'ECH Config List',
    zhCN: 'ECH 配置列表',
    enUS: 'ECH Config List',
    context: 'TLS settings',
  },
  {
    source: 'Max Time Diff',
    zhCN: '最大时间差',
    enUS: 'Max Time Diff',
    context: 'Reality settings',
  },
  {
    source: 'ML-DSA-65 Seed',
    zhCN: 'ML-DSA-65 种子',
    enUS: 'ML-DSA-65 Seed',
    context: 'Reality settings',
  },
  {
    source: 'ML-DSA-65 Verify',
    zhCN: 'ML-DSA-65 校验',
    enUS: 'ML-DSA-65 Verify',
    context: 'Reality settings',
  },
  {
    source: 'TCP Fast Open',
    zhCN: 'TCP 快速打开',
    enUS: 'TCP Fast Open',
    context: 'Sockopt settings',
  },
  {
    source: 'Multipath TCP',
    zhCN: '多路径 TCP',
    enUS: 'Multipath TCP',
    context: 'Sockopt settings',
  },
  { source: 'Penetrate', zhCN: '穿透', enUS: 'Penetrate', context: 'Sockopt settings' },
  { source: 'V6 Only', zhCN: '仅 IPv6', enUS: 'V6 Only', context: 'Sockopt settings' },
  {
    source: 'Domain Strategy',
    zhCN: '域名策略',
    enUS: 'Domain Strategy',
    context: 'Sockopt settings',
  },
  {
    source: 'TCP Congestion',
    zhCN: 'TCP 拥塞控制',
    enUS: 'TCP Congestion',
    context: 'Sockopt settings',
  },
  {
    source: 'Dialer Proxy',
    zhCN: '拨号代理',
    enUS: 'Dialer Proxy',
    context: 'Sockopt settings',
  },
  {
    source: 'Interface Name',
    zhCN: '网卡名称',
    enUS: 'Interface Name',
    context: 'Sockopt settings',
  },
  {
    source: 'Trusted X-Forwarded-For',
    zhCN: '可信 X-Forwarded-For',
    enUS: 'Trusted X-Forwarded-For',
    context: 'Sockopt settings',
  },
  { source: 'Add Client', zhCN: '添加客户端', enUS: 'Add Client', context: 'Client modal' },
  { source: 'Edit Client', zhCN: '编辑客户端', enUS: 'Edit Client', context: 'Client modal' },
  { source: 'Client', zhCN: '客户端', enUS: 'Client', context: 'Client table' },
  { source: 'Share Link', zhCN: '分享链接', enUS: 'Share Link', context: 'Client sharing' },
  { source: 'Share', zhCN: '分享', enUS: 'Share', context: 'Client action' },
  { source: 'Private Key', zhCN: '私钥', enUS: 'Private Key', context: 'Client modal' },
  { source: 'Public Key', zhCN: '公钥', enUS: 'Public Key', context: 'Client modal' },
  {
    source: 'Pre Shared Key',
    zhCN: '预共享密钥',
    enUS: 'Pre Shared Key',
    context: 'Client modal',
  },
  { source: 'Allowed IPs', zhCN: '允许的 IP', enUS: 'Allowed IPs', context: 'Client modal' },
  { source: 'Keep Alive', zhCN: '保活间隔', enUS: 'Keep Alive', context: 'Client modal' },
  { source: 'IP Limit', zhCN: 'IP 限制', enUS: 'IP Limit', context: 'Client modal' },
  { source: 'Reset Days', zhCN: '重置天数', enUS: 'Reset Days', context: 'Client modal' },
  { source: 'Sub ID', zhCN: '订阅 ID', enUS: 'Sub ID', context: 'Client modal' },
  { source: 'Comment', zhCN: '注释', enUS: 'Comment', context: 'Client modal' },
  {
    source: 'Client IP Records',
    zhCN: '客户端 IP 记录',
    enUS: 'Client IP Records',
    context: 'Client IP modal',
  },
  {
    source: 'Client IP records',
    zhCN: '客户端 IP 记录',
    enUS: 'Client IP records',
    context: 'Client IP aria label',
  },
  {
    source: 'Import Inbound JSON',
    zhCN: '导入入站 JSON',
    enUS: 'Import Inbound JSON',
    context: 'Import modal',
  },
  {
    source:
      'Paste a legacy inbound JSON object. It will be imported through the existing Xray API so old UI remains readable.',
    zhCN: '粘贴旧版入站 JSON 对象。系统会通过现有 Xray API 导入，确保旧 UI 仍可读取。',
    enUS: 'Paste a legacy inbound JSON object. It will be imported through the existing Xray API so old UI remains readable.',
    context: 'Import modal',
  },
  {
    source: 'Inbound import JSON',
    zhCN: '入站导入 JSON',
    enUS: 'Inbound import JSON',
    context: 'Import textarea',
  },
  { source: 'OK', zhCN: '确定', enUS: 'OK', context: 'Modal action' },
  { source: 'Cancel', zhCN: '取消', enUS: 'Cancel', context: 'Modal action' },
  { source: 'Generate', zhCN: '生成', enUS: 'Generate', context: 'Credential action' },
  { source: 'never', zhCN: '从不', enUS: 'never', context: 'Traffic reset default' },
  {
    source: 'Inbound setup guide',
    zhCN: '入站填写指南',
    enUS: 'Inbound setup guide',
    context: 'Beginner help',
  },
  {
    source:
      'Start with protocol, remark, listen address, and port. Keep 0.0.0.0 to accept connections on all interfaces; set traffic and expiry to 0 for no limit. Use the transport form for common TCP/WS/gRPC/TLS/Reality options, then click Sync JSON only when you need to inspect or manually adjust the raw legacy JSON.',
    zhCN: '新手建议先填写协议、备注、监听地址和端口。监听地址保持 0.0.0.0 表示接受所有网卡连接；流量限制和到期时间填 0 表示不限制。常见 TCP/WS/gRPC/TLS/Reality 配置优先使用传输设置表单；只有需要检查或手动调整旧版 JSON 时再点击“同步 JSON”。',
    enUS: 'Start with protocol, remark, listen address, and port. Keep 0.0.0.0 to accept connections on all interfaces; set traffic and expiry to 0 for no limit. Use the transport form for common TCP/WS/gRPC/TLS/Reality options, then click Sync JSON only when you need to inspect or manually adjust the raw legacy JSON.',
    context: 'Beginner help',
  },
] as const;

type ReviewedDomTranslation = (typeof reviewedDomTranslations)[number];

const reviewedSourceTextToEntry = Object.fromEntries(
  reviewedDomTranslations.map((entry) => [entry.source, entry]),
) as Record<string, ReviewedDomTranslation>;

const reviewedChineseTextToEntry = Object.fromEntries(
  reviewedDomTranslations.map((entry) => [entry.zhCN, entry]),
) as Record<string, ReviewedDomTranslation>;

const englishTextToKey: Record<string, MessageKey> = {
  'Add Resource': 'action.addResource',
  Add: 'action.add',
  Apply: 'action.apply',
  Auto: 'logs.auto',
  Backup: 'common.backup',
  'Backup / Restore': 'settings.backupRestore',
  Blocked: 'common.blocked',
  Clients: 'common.clients',
  Configuration: 'common.configuration',
  Connections: 'common.connections',
  Copy: 'action.copy',
  'Copy Links': 'action.copyLinks',
  CPU: 'common.cpu',
  Credentials: 'common.credentials',
  Actions: 'common.actions',
  Alias: 'field.alias',
  Dashboard: 'nav.dashboard',
  Database: 'common.database',
  Delete: 'action.delete',
  Detail: 'common.detail',
  Direct: 'common.direct',
  Disk: 'common.disk',
  Download: 'action.download',
  Edit: 'action.edit',
  Enabled: 'common.enabled',
  Encrypted: 'common.encrypted',
  Endpoints: 'settings.endpoints',
  Experimental: 'common.experimental',
  'External Code': 'common.externalCode',
  'Fill Defaults': 'action.fillDefaults',
  Filter: 'common.filter',
  Format: 'action.format',
  Formats: 'common.formats',
  'Generate Token': 'action.generateToken',
  Import: 'action.import',
  Inbounds: 'nav.inbounds',
  Install: 'action.install',
  Instances: 'common.instances',
  'Last Updated': 'common.lastUpdated',
  Language: 'common.language',
  Lifecycle: 'common.lifecycle',
  'Load Average': 'common.loadAverage',
  Locked: 'logs.locked',
  Logs: 'nav.logs',
  Memory: 'common.memory',
  Metric: 'common.metric',
  'No data': 'common.noData',
  Observability: 'common.observability',
  Outbound: 'common.outbound',
  Overview: 'common.overview',
  Panel: 'common.panel',
  'Panel Runtime': 'common.panelRuntime',
  'Panel Uptime': 'common.panelUptime',
  Providers: 'common.providers',
  Proxy: 'common.proxy',
  'Public Access': 'common.publicAccess',
  'Public IP': 'common.publicIp',
  Read: 'common.read',
  'Read only': 'common.readOnly',
  Refresh: 'action.refresh',
  'Refresh Activity': 'action.refreshActivity',
  'Refresh Traffic': 'action.refreshTraffic',
  Reset: 'action.reset',
  'Reset All Traffic': 'action.resetAllTraffic',
  Restart: 'action.restart',
  Restore: 'common.restore',
  Routing: 'common.routing',
  Running: 'common.running',
  Runtime: 'common.runtime',
  Save: 'action.save',
  Security: 'common.security',
  'Selected Core': 'common.selectedCore',
  Settings: 'nav.settings',
  'Sign in': 'action.signIn',
  'Show Info': 'common.showInfo',
  Start: 'action.start',
  Status: 'action.status',
  Stop: 'action.stop',
  Subscription: 'common.subscription',
  Swap: 'common.swap',
  Syslog: 'common.syslog',
  Template: 'common.template',
  Traffic: 'common.traffic',
  Type: 'field.type',
  'Two Factor': 'common.twoFactor',
  Update: 'action.update',
  'Update all': 'action.updateAll',
  'Update Login': 'action.updateLogin',
  'Update Resources': 'action.updateResources',
  Validate: 'action.validate',
  Value: 'common.value',
  Version: 'common.version',
  Versions: 'action.versions',
  Write: 'common.write',
  Xray: 'nav.xray',
  'Xray State': 'common.xrayState',
};

const phraseTextToKey: Record<string, MessageKey> = {
  'Add Resource': 'action.addResource',
  'Control Xray with Confidence': 'login.title',
  'Core Instances': 'route.cores',
  'Custom Geo': 'dashboard.customGeo',
  'Default Xray and experimental core adapters under the current migration gate.': 'route.cores',
  'Disable Two Factor': 'action.disableTwoFactor',
  'Enter your password': 'login.passwordPlaceholder',
  'Geo Maintenance': 'dashboard.geoMaintenance',
  'Inspect panel and Xray logs with filtering, export, and auto-follow controls.':
    'logs.description',
  'Live Xray health, traffic, clients, and geo maintenance.': 'dashboard.description',
  'Manage external geoip/geosite resources without leaving the new UI.': 'dashboard.geoText',
  'Manage panel security, subscription endpoints, backups, and legacy-compatible settings.':
    'settings.description',
  'Sign in to access the SuperXray operations console': 'login.description',
  'Stream Settings Form': 'inbounds.streamSettingsForm',
  'Subscription Public Links': 'settings.publicLinks',
  'The requested panel route was not found.': 'notFound.description',
  'Two Factor Setup': 'settings.twoFactorSetup',
  'WireGuard Settings': 'inbounds.wireguardSettings',
  'Xray Runtime Control': 'xray.runtimeControl',
  'Xray Template Editor': 'xray.configEditor',
  'Xray Version Management': 'xray.versionManagement',
  'Outbound Tools': 'xray.outboundTools',
  'Control the legacy Xray runtime, template, versions, outbounds, and provider tools.':
    'xray.description',
};

const attributeTextToKey: Record<string, MessageKey> = {
  'Base path': 'field.basePath',
  'Certificate file': 'field.certificateFile',
  'Collapse navigation': 'status.collapseNav',
  'Dark theme': 'status.darkTheme',
  'Datepicker mode': 'field.datepicker',
  'Expand navigation': 'status.expandNav',
  'Expire diff': 'field.expireDiff',
  'Filter logs': 'common.filter',
  'Import database file': 'action.import',
  'Include syslog': 'common.syslog',
  'Key file': 'field.keyFile',
  'Log line count': 'logs.source',
  'Log source': 'logs.source',
  'Outbound test URL': 'field.url',
  'Page size': 'field.pageSize',
  'Panel log level': 'logs.source',
  Password: 'field.password',
  'Remark model': 'field.remarkModel',
  'Select Xray version': 'common.version',
  'Session max age': 'field.sessionMaxAge',
  'Show blocked logs': 'common.blocked',
  'Show direct logs': 'common.direct',
  'Show proxy logs': 'common.proxy',
  'Subscription public links': 'settings.publicLinks',
  'Time location': 'field.timeLocation',
  'Toggle log auto follow': 'logs.auto',
  'Traffic diff': 'field.trafficDiff',
  'Two-factor setup URI': 'field.setupUri',
  Username: 'field.username',
  'Web domain': 'field.webDomain',
  'Web listen address': 'field.webListen',
  'Web port': 'field.webPort',
  'Xray JSON template': 'xray.configEditor',
};

const messageEnglishTextToKey = Object.fromEntries(
  Object.entries(messages).map(([key, value]) => [value['en-US'], key as MessageKey]),
) as Record<string, MessageKey>;

const textToKey: Record<string, MessageKey> = {
  ...messageEnglishTextToKey,
  ...englishTextToKey,
  ...phraseTextToKey,
  ...attributeTextToKey,
};

const chineseTextToKey: Record<string, MessageKey> = {
  ...(Object.fromEntries(
    Object.entries(messages).map(([key, value]) => [value['zh-CN'], key as MessageKey]),
  ) as Record<string, MessageKey>),
  登录: 'action.signIn',
};

const lowerEnglishTextToKey = Object.fromEntries(
  Object.entries(textToKey).map(([text, key]) => [text.toLocaleLowerCase('en-US'), key]),
) as Record<string, MessageKey>;

export function normalizeLocale(value: unknown): AppLocale {
  return value === 'en-US' || value === 'zh-CN' ? value : DEFAULT_LOCALE;
}

export function getStoredLocale(): AppLocale {
  if (typeof window === 'undefined') {
    return DEFAULT_LOCALE;
  }
  return normalizeLocale(window.localStorage.getItem(LOCALE_STORAGE_KEY));
}

export function translate(key: MessageKey, locale: AppLocale): string {
  return messages[key][locale];
}

export function getNextLocale(locale: AppLocale): AppLocale {
  return locale === 'zh-CN' ? 'en-US' : 'zh-CN';
}

export function getLanguageButtonLabel(locale: AppLocale): string {
  return locale === 'zh-CN'
    ? translate('language.button.zh', locale)
    : translate('language.button.en', locale);
}

export function getLanguageToggleAriaLabel(locale: AppLocale): string {
  return locale === 'zh-CN'
    ? translate('language.toggleToEnglish', locale)
    : translate('language.toggleToChinese', locale);
}

export function getRouteTitle(routeName: unknown, locale: AppLocale): string {
  const suffix = typeof routeName === 'string' && routeName ? routeName : 'panel';
  const key = `route.${suffix}` as MessageKey;
  return key in messages ? translate(key, locale) : translate('route.panel', locale);
}

export function formatDocumentTitle(routeName: unknown, locale: AppLocale): string {
  return `${getRouteTitle(routeName, locale)} - SuperXray`;
}

export function translateDomText(source: string, locale: AppLocale): string {
  const reviewedEntry = reviewedSourceTextToEntry[source] || reviewedChineseTextToEntry[source];
  if (reviewedEntry) {
    return locale === 'zh-CN' ? reviewedEntry.zhCN : reviewedEntry.enUS;
  }

  const dynamicText = translateDynamicText(source, locale);
  if (dynamicText) {
    return dynamicText;
  }

  const key =
    textToKey[source] ||
    (isUppercaseLabel(source)
      ? lowerEnglishTextToKey[source.toLocaleLowerCase('en-US')]
      : undefined) ||
    chineseTextToKey[source];
  return key ? translate(key, locale) : source;
}

export function getTranslationAuditReport() {
  const missing: string[] = [];
  const seenSources = new Set<string>();
  const duplicates: string[] = [];
  const auditEntries: ReadonlyArray<{
    context: string;
    enUS: string;
    source: string;
    zhCN: string;
  }> = reviewedDomTranslations;

  for (const entry of auditEntries) {
    if (!entry.source.trim() || !entry.zhCN.trim() || !entry.enUS.trim()) {
      missing.push(entry.source || entry.context);
    }
    if (seenSources.has(entry.source)) {
      duplicates.push(entry.source);
    }
    seenSources.add(entry.source);
  }

  return {
    duplicates,
    missing,
    messageEntries: Object.keys(messages).length,
    reviewedEntries: reviewedDomTranslations.length,
  };
}

function isUppercaseLabel(source: string): boolean {
  return /[A-Z]/.test(source) && source === source.toLocaleUpperCase('en-US');
}

function translateDynamicText(source: string, locale: AppLocale): string | undefined {
  const inboundTitleMatch = source.match(/^(?:Inbound|入站)\s+(.+)$/i);
  if (inboundTitleMatch) {
    return locale === 'zh-CN' ? `入站 ${inboundTitleMatch[1]}` : `Inbound ${inboundTitleMatch[1]}`;
  }

  const inboundToggleMatch = source.match(
    /^(Enable|Disable|启用|禁用)\s*(?:inbound|入站)\s+(.+)$/i,
  );
  if (inboundToggleMatch) {
    const verb = inboundToggleMatch[1].toLocaleLowerCase('en-US');
    const target = inboundToggleMatch[2];
    if (locale === 'zh-CN') {
      return `${verb === 'disable' || verb === '禁用' ? '禁用' : '启用'}入站 ${target}`;
    }
    return `${verb === 'disable' || verb === '禁用' ? 'Disable' : 'Enable'} inbound ${target}`;
  }

  const enabledMatch = source.match(/^(\d+)\s+(enabled|已启用)$/i);
  if (enabledMatch) {
    return locale === 'zh-CN' ? `${enabledMatch[1]} 已启用` : `${enabledMatch[1]} enabled`;
  }

  const updateFileMatch = source.match(/^(?:Update|更新)\s+(.+\.(?:dat|json|db|sqlite))$/i);
  if (updateFileMatch) {
    return locale === 'zh-CN' ? `更新 ${updateFileMatch[1]}` : `Update ${updateFileMatch[1]}`;
  }

  const versionMatch = source.match(/^(?:Version|版本)\s+(.+)$/i);
  if (versionMatch) {
    return locale === 'zh-CN' ? `版本 ${versionMatch[1]}` : `Version ${versionMatch[1]}`;
  }

  const cpuSpeedMatch = source.match(/^(?:CPU speed|CPU 速度)\s+(.+)$/i);
  if (cpuSpeedMatch) {
    return locale === 'zh-CN' ? `CPU 速度 ${cpuSpeedMatch[1]}` : `CPU speed ${cpuSpeedMatch[1]}`;
  }

  return undefined;
}
