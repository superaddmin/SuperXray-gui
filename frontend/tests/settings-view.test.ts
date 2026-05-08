import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync('frontend/src/views/SettingsView.vue', 'utf8');

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

test('settings view uses form sections for each settings workflow', () => {
  assert.match(source, /import FormSection from '@\/components\/FormSection\.vue';/);

  for (const title of [
    'Web Endpoint',
    'TLS Files',
    'Session and Display',
    'Thresholds and Naming',
    'Two Factor',
    'Credentials',
    'Feature Flags',
    'Server Endpoint',
    'Public URIs',
    'Metadata and TLS',
    'External Traffic',
    'Public Links',
    'Recommended Client Links',
    'Announce and Routing',
    'JSON Formats',
    'Bot Flags',
    'Bot Connection',
    'Runtime Rules',
    'LDAP Flags',
    'Connection',
    'User Mapping',
    'Sync Defaults',
    'Database Backup / Restore',
    'Panel Runtime',
  ]) {
    assert.match(source, new RegExp(`title="${escapeRegExp(title)}"`));
  }
});

test('settings view keeps all critical action flows', () => {
  for (const action of [
    '@click="loadSettings"',
    '@click="resetForm"',
    '@click="confirmSave"',
    '@click="generateTwoFactorToken"',
    '@click="disableTwoFactor"',
    '@click="confirmUpdateCredentials"',
    '@click="applySubscriptionDefaults"',
    '@click="copySubscriptionLinks"',
    '@click="copyRecommendedLinks"',
    '@click="downloadDb"',
    '@click="openDbFilePicker"',
    '@change="handleDbFileChange"',
    '@click="confirmRestartPanel"',
  ]) {
    assert.match(source, new RegExp(escapeRegExp(action)));
  }
});

test('settings view keeps representative field bindings from every tab', () => {
  for (const binding of [
    'settings.webListen',
    'settings.webDomain',
    'settings.webPort',
    'settings.webBasePath',
    'settings.webCertFile',
    'settings.webKeyFile',
    'settings.sessionMaxAge',
    'settings.pageSize',
    'settings.expireDiff',
    'settings.trafficDiff',
    'settings.remarkModel',
    'settings.datepicker',
    'settings.timeLocation',
    'settings.twoFactorEnable',
    'settings.twoFactorToken',
    'credentials.oldUsername',
    'credentials.oldPassword',
    'credentials.newUsername',
    'credentials.newPassword',
    'settings.subEnable',
    'settings.subJsonEnable',
    'settings.subClashEnable',
    'settings.subEncrypt',
    'settings.subShowInfo',
    'settings.subEnableRouting',
    'settings.subTitle',
    'settings.subUpdates',
    'settings.subListen',
    'settings.subPort',
    'settings.subDomain',
    'settings.subPath',
    'settings.subURI',
    'settings.subJsonPath',
    'settings.subJsonURI',
    'settings.subClashPath',
    'settings.subClashURI',
    'settings.subSupportUrl',
    'settings.subProfileUrl',
    'settings.subCertFile',
    'settings.subKeyFile',
    'settings.externalTrafficInformEnable',
    'settings.externalTrafficInformURI',
    'settings.subAnnounce',
    'settings.subRoutingRules',
    'settings.subJsonFragment',
    'settings.subJsonNoises',
    'settings.subJsonMux',
    'settings.subJsonRules',
    'settings.tgBotEnable',
    'settings.tgBotBackup',
    'settings.tgBotLoginNotify',
    'settings.tgBotToken',
    'settings.tgBotChatId',
    'settings.tgBotProxy',
    'settings.tgBotAPIServer',
    'settings.tgRunTime',
    'settings.tgCpu',
    'settings.tgLang',
    'settings.ldapEnable',
    'settings.ldapUseTLS',
    'settings.ldapInvertFlag',
    'settings.ldapAutoCreate',
    'settings.ldapAutoDelete',
    'settings.ldapHost',
    'settings.ldapPort',
    'settings.ldapBindDN',
    'settings.ldapPassword',
    'settings.ldapBaseDN',
    'settings.ldapUserFilter',
    'settings.ldapUserAttr',
    'settings.ldapVlessField',
    'settings.ldapSyncCron',
    'settings.ldapFlagField',
    'settings.ldapTruthyValues',
    'settings.ldapInboundTags',
    'settings.ldapDefaultTotalGB',
    'settings.ldapDefaultExpiryDays',
    'settings.ldapDefaultLimitIP',
  ]) {
    assert.match(source, new RegExp(escapeRegExp(binding)));
  }
});
