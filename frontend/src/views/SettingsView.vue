<template>
  <section class="page-stack">
    <PageHeader eyebrow="Panel" title="Settings">
      <ASpace wrap>
        <AButton :loading="loading" @click="loadSettings">
          <template #icon><ReloadOutlined /></template>
          Refresh
        </AButton>
        <AButton :disabled="!settingsChanged" @click="resetForm">
          <template #icon><UndoOutlined /></template>
          Reset
        </AButton>
        <AButton :loading="saving" :disabled="!settingsLoaded" type="primary" @click="confirmSave">
          <template #icon><SaveOutlined /></template>
          Save
        </AButton>
      </ASpace>
    </PageHeader>

    <AAlert v-if="error" banner type="warning" :message="error" />

    <ACard class="work-panel" :bordered="false">
      <ATabs v-model:active-key="activeTab">
        <ATabPane key="panel" tab="Panel">
          <AForm layout="vertical">
            <div class="form-grid">
              <AFormItem label="Web Listen">
                <AInput
                  v-model:value="settings.webListen"
                  aria-label="Web listen address"
                  placeholder="0.0.0.0"
                />
              </AFormItem>
              <AFormItem label="Web Domain">
                <AInput v-model:value="settings.webDomain" aria-label="Web domain" />
              </AFormItem>
              <AFormItem label="Web Port">
                <AInputNumber
                  v-model:value="settings.webPort"
                  aria-label="Web port"
                  class="full-width"
                  :min="1"
                  :max="65535"
                />
              </AFormItem>
              <AFormItem label="Base Path">
                <AInput v-model:value="settings.webBasePath" aria-label="Base path" />
              </AFormItem>
              <AFormItem label="Certificate File">
                <AInput v-model:value="settings.webCertFile" aria-label="Certificate file" />
              </AFormItem>
              <AFormItem label="Key File">
                <AInput v-model:value="settings.webKeyFile" aria-label="Key file" />
              </AFormItem>
              <AFormItem label="Session Max Age">
                <AInputNumber
                  v-model:value="settings.sessionMaxAge"
                  aria-label="Session max age"
                  class="full-width"
                  :min="1"
                />
              </AFormItem>
              <AFormItem label="Page Size">
                <AInputNumber
                  v-model:value="settings.pageSize"
                  aria-label="Page size"
                  class="full-width"
                  :min="1"
                />
              </AFormItem>
              <AFormItem label="Expire Diff">
                <AInputNumber
                  v-model:value="settings.expireDiff"
                  aria-label="Expire diff"
                  class="full-width"
                  :min="0"
                />
              </AFormItem>
              <AFormItem label="Traffic Diff">
                <AInputNumber
                  v-model:value="settings.trafficDiff"
                  aria-label="Traffic diff"
                  class="full-width"
                  :min="0"
                />
              </AFormItem>
              <AFormItem label="Remark Model">
                <AInput v-model:value="settings.remarkModel" aria-label="Remark model" />
              </AFormItem>
              <AFormItem label="Datepicker">
                <ASelect
                  v-model:value="settings.datepicker"
                  aria-label="Datepicker mode"
                  :options="datepickerOptions"
                />
              </AFormItem>
              <AFormItem label="Time Location">
                <AInput v-model:value="settings.timeLocation" aria-label="Time location" />
              </AFormItem>
            </div>
          </AForm>
        </ATabPane>

        <ATabPane key="security" tab="Security">
          <AForm layout="vertical">
            <div class="form-grid">
              <AFormItem label="Two Factor">
                <ASwitch v-model:checked="settings.twoFactorEnable" />
              </AFormItem>
            </div>
          </AForm>

          <div class="settings-feature-panel">
            <div class="panel-header compact-panel-header">
              <div>
                <p class="page-eyebrow">Security</p>
                <h2>Two Factor Setup</h2>
              </div>
              <ASpace wrap>
                <AButton @click="generateTwoFactorToken">
                  <template #icon><KeyOutlined /></template>
                  Generate Token
                </AButton>
                <AButton danger @click="disableTwoFactor">Disable Two Factor</AButton>
              </ASpace>
            </div>
            <div class="settings-token-grid">
              <AFormItem label="Two Factor Token">
                <AInput v-model:value="settings.twoFactorToken" type="password" />
              </AFormItem>
              <AFormItem label="Setup URI">
                <textarea
                  aria-label="Two-factor setup URI"
                  class="json-editor compact-json-editor"
                  readonly
                  :value="twoFactorSetupUri"
                />
              </AFormItem>
            </div>
            <p class="muted-text">
              Generate or disable the token here, then save panel settings so the legacy settings
              page reads the same two-factor fields.
            </p>
          </div>

          <ADivider />

          <div class="panel-header">
            <div>
              <p class="page-eyebrow">Credentials</p>
              <h2>Update Login</h2>
            </div>
            <AButton :loading="updatingCredentials" danger @click="confirmUpdateCredentials">
              Update
            </AButton>
          </div>
          <AForm layout="vertical">
            <div class="form-grid">
              <AFormItem label="Current Username">
                <AInput v-model:value="credentials.oldUsername" autocomplete="username" />
              </AFormItem>
              <AFormItem label="Current Password">
                <AInput
                  v-model:value="credentials.oldPassword"
                  autocomplete="current-password"
                  type="password"
                />
              </AFormItem>
              <AFormItem label="New Username">
                <AInput v-model:value="credentials.newUsername" autocomplete="username" />
              </AFormItem>
              <AFormItem label="New Password">
                <AInput
                  v-model:value="credentials.newPassword"
                  autocomplete="new-password"
                  type="password"
                />
              </AFormItem>
            </div>
          </AForm>
        </ATabPane>

        <ATabPane key="subscription" tab="Subscription">
          <div class="panel-header">
            <div>
              <p class="page-eyebrow">Subscription</p>
              <h2>Endpoints</h2>
            </div>
            <AButton :loading="loadingDefaults" @click="applySubscriptionDefaults">
              Fill Defaults
            </AButton>
          </div>
          <AForm layout="vertical">
            <div class="settings-switch-row">
              <ACheckbox v-model:checked="settings.subEnable">URI</ACheckbox>
              <ACheckbox v-model:checked="settings.subJsonEnable">JSON</ACheckbox>
              <ACheckbox v-model:checked="settings.subClashEnable">Clash/Mihomo</ACheckbox>
              <ACheckbox v-model:checked="settings.subEncrypt">Encrypted</ACheckbox>
              <ACheckbox v-model:checked="settings.subShowInfo">Show Info</ACheckbox>
              <ACheckbox v-model:checked="settings.subEnableRouting">Routing</ACheckbox>
            </div>

            <div class="form-grid">
              <AFormItem label="Title">
                <AInput v-model:value="settings.subTitle" />
              </AFormItem>
              <AFormItem label="Updates">
                <AInputNumber v-model:value="settings.subUpdates" class="full-width" :min="1" />
              </AFormItem>
              <AFormItem label="Listen">
                <AInput v-model:value="settings.subListen" />
              </AFormItem>
              <AFormItem label="Port">
                <AInputNumber
                  v-model:value="settings.subPort"
                  class="full-width"
                  :min="1"
                  :max="65535"
                />
              </AFormItem>
              <AFormItem label="Domain">
                <AInput v-model:value="settings.subDomain" />
              </AFormItem>
              <AFormItem label="Path">
                <AInput v-model:value="settings.subPath" />
              </AFormItem>
              <AFormItem label="URI">
                <AInput v-model:value="settings.subURI" />
              </AFormItem>
              <AFormItem label="JSON Path">
                <AInput v-model:value="settings.subJsonPath" />
              </AFormItem>
              <AFormItem label="JSON URI">
                <AInput v-model:value="settings.subJsonURI" />
              </AFormItem>
              <AFormItem label="Clash Path">
                <AInput v-model:value="settings.subClashPath" />
              </AFormItem>
              <AFormItem label="Clash URI">
                <AInput v-model:value="settings.subClashURI" />
              </AFormItem>
              <AFormItem label="Support URL">
                <AInput v-model:value="settings.subSupportUrl" />
              </AFormItem>
              <AFormItem label="Profile URL">
                <AInput v-model:value="settings.subProfileUrl" />
              </AFormItem>
              <AFormItem label="Certificate File">
                <AInput v-model:value="settings.subCertFile" />
              </AFormItem>
              <AFormItem label="Key File">
                <AInput v-model:value="settings.subKeyFile" />
              </AFormItem>
              <AFormItem label="External Traffic Inform">
                <ASwitch v-model:checked="settings.externalTrafficInformEnable" />
              </AFormItem>
              <AFormItem label="External Traffic URI">
                <AInput v-model:value="settings.externalTrafficInformURI" />
              </AFormItem>
            </div>

            <div class="settings-feature-panel">
              <div class="panel-header compact-panel-header">
                <div>
                  <p class="page-eyebrow">Public Access</p>
                  <h2>Subscription Public Links</h2>
                </div>
                <AButton @click="copySubscriptionLinks">
                  <template #icon><CopyOutlined /></template>
                  Copy Links
                </AButton>
              </div>
              <textarea
                aria-label="Subscription public links"
                class="json-editor compact-json-editor"
                readonly
                :value="subscriptionPublicLinkText"
              />
              <ASpace class="public-link-actions" wrap>
                <AButton :disabled="!settings.subURI" @click="openPublicLink(settings.subURI)">
                  Open URI
                </AButton>
                <AButton
                  :disabled="!settings.subJsonURI"
                  @click="openPublicLink(settings.subJsonURI)"
                >
                  Open JSON
                </AButton>
                <AButton
                  :disabled="!settings.subClashURI"
                  @click="openPublicLink(settings.subClashURI)"
                >
                  Open Clash
                </AButton>
              </ASpace>
            </div>

            <AFormItem label="Announce">
              <textarea v-model="settings.subAnnounce" class="json-editor compact-json-editor" />
            </AFormItem>
            <AFormItem label="Routing Rules">
              <textarea
                v-model="settings.subRoutingRules"
                class="json-editor compact-json-editor"
              />
            </AFormItem>
          </AForm>
        </ATabPane>

        <ATabPane key="formats" tab="Formats">
          <AAlert
            v-if="jsonWarnings.length"
            class="mb-12"
            type="warning"
            :message="jsonWarnings.join(' | ')"
          />
          <AForm layout="vertical">
            <AFormItem label="JSON Fragment">
              <textarea v-model="settings.subJsonFragment" class="json-editor modal-json-editor" />
            </AFormItem>
            <AFormItem label="JSON Noises">
              <textarea v-model="settings.subJsonNoises" class="json-editor modal-json-editor" />
            </AFormItem>
            <AFormItem label="JSON Mux">
              <textarea v-model="settings.subJsonMux" class="json-editor modal-json-editor" />
            </AFormItem>
            <AFormItem label="JSON Rules">
              <textarea v-model="settings.subJsonRules" class="json-editor modal-json-editor" />
            </AFormItem>
          </AForm>
        </ATabPane>

        <ATabPane key="telegram" tab="Telegram">
          <AForm layout="vertical">
            <div class="settings-switch-row">
              <ACheckbox v-model:checked="settings.tgBotEnable">Enabled</ACheckbox>
              <ACheckbox v-model:checked="settings.tgBotBackup">Backup</ACheckbox>
              <ACheckbox v-model:checked="settings.tgBotLoginNotify">Login Notify</ACheckbox>
            </div>
            <div class="form-grid">
              <AFormItem label="Bot Token">
                <AInput v-model:value="settings.tgBotToken" type="password" />
              </AFormItem>
              <AFormItem label="Chat ID">
                <AInput v-model:value="settings.tgBotChatId" />
              </AFormItem>
              <AFormItem label="Proxy">
                <AInput v-model:value="settings.tgBotProxy" />
              </AFormItem>
              <AFormItem label="API Server">
                <AInput v-model:value="settings.tgBotAPIServer" />
              </AFormItem>
              <AFormItem label="Runtime">
                <AInput v-model:value="settings.tgRunTime" />
              </AFormItem>
              <AFormItem label="CPU Threshold">
                <AInputNumber v-model:value="settings.tgCpu" class="full-width" :min="0" />
              </AFormItem>
              <AFormItem label="Language">
                <AInput v-model:value="settings.tgLang" />
              </AFormItem>
            </div>
          </AForm>
        </ATabPane>

        <ATabPane key="ldap" tab="LDAP">
          <AForm layout="vertical">
            <div class="settings-switch-row">
              <ACheckbox v-model:checked="settings.ldapEnable">Enabled</ACheckbox>
              <ACheckbox v-model:checked="settings.ldapUseTLS">TLS</ACheckbox>
              <ACheckbox v-model:checked="settings.ldapInvertFlag">Invert Flag</ACheckbox>
              <ACheckbox v-model:checked="settings.ldapAutoCreate">Auto Create</ACheckbox>
              <ACheckbox v-model:checked="settings.ldapAutoDelete">Auto Delete</ACheckbox>
            </div>
            <div class="form-grid">
              <AFormItem label="Host">
                <AInput v-model:value="settings.ldapHost" />
              </AFormItem>
              <AFormItem label="Port">
                <AInputNumber
                  v-model:value="settings.ldapPort"
                  class="full-width"
                  :min="1"
                  :max="65535"
                />
              </AFormItem>
              <AFormItem label="Bind DN">
                <AInput v-model:value="settings.ldapBindDN" />
              </AFormItem>
              <AFormItem label="Password">
                <AInput v-model:value="settings.ldapPassword" type="password" />
              </AFormItem>
              <AFormItem label="Base DN">
                <AInput v-model:value="settings.ldapBaseDN" />
              </AFormItem>
              <AFormItem label="User Filter">
                <AInput v-model:value="settings.ldapUserFilter" />
              </AFormItem>
              <AFormItem label="User Attr">
                <AInput v-model:value="settings.ldapUserAttr" />
              </AFormItem>
              <AFormItem label="VLESS Field">
                <AInput v-model:value="settings.ldapVlessField" />
              </AFormItem>
              <AFormItem label="Sync Cron">
                <AInput v-model:value="settings.ldapSyncCron" />
              </AFormItem>
              <AFormItem label="Flag Field">
                <AInput v-model:value="settings.ldapFlagField" />
              </AFormItem>
              <AFormItem label="Truthy Values">
                <AInput v-model:value="settings.ldapTruthyValues" />
              </AFormItem>
              <AFormItem label="Inbound Tags">
                <AInput v-model:value="settings.ldapInboundTags" />
              </AFormItem>
              <AFormItem label="Default Total GB">
                <AInputNumber
                  v-model:value="settings.ldapDefaultTotalGB"
                  class="full-width"
                  :min="0"
                />
              </AFormItem>
              <AFormItem label="Default Expiry Days">
                <AInputNumber
                  v-model:value="settings.ldapDefaultExpiryDays"
                  class="full-width"
                  :min="0"
                />
              </AFormItem>
              <AFormItem label="Default Limit IP">
                <AInputNumber
                  v-model:value="settings.ldapDefaultLimitIP"
                  class="full-width"
                  :min="0"
                />
              </AFormItem>
            </div>
          </AForm>
        </ATabPane>

        <ATabPane key="backup" tab="Backup">
          <div class="panel-header">
            <div>
              <p class="page-eyebrow">Database</p>
              <h2>Backup / Restore</h2>
            </div>
            <ASpace wrap>
              <AButton :loading="downloadingDb" @click="downloadDb">
                <template #icon><DownloadOutlined /></template>
                Download
              </AButton>
              <AButton :loading="importingDb" danger @click="openDbFilePicker">
                <template #icon><ImportOutlined /></template>
                Import
              </AButton>
            </ASpace>
          </div>
          <AAlert
            type="warning"
            message="Database import uses the existing legacy restore path and restarts Xray after import."
          />
          <!-- eslint-disable vue/html-self-closing -->
          <input
            ref="dbFileInput"
            aria-label="Import database file"
            accept=".db,.sqlite,.sqlite3"
            class="visually-hidden"
            type="file"
            @change="handleDbFileChange"
          />
          <!-- eslint-enable vue/html-self-closing -->

          <ADivider />

          <div class="panel-header">
            <div>
              <p class="page-eyebrow">Panel Runtime</p>
              <h2>Restart Panel</h2>
            </div>
            <AButton :loading="restartingPanel" danger @click="confirmRestartPanel">
              Restart
            </AButton>
          </div>
        </ATabPane>
      </ATabs>
    </ACard>
  </section>
</template>

<script setup lang="ts">
import {
  CopyOutlined,
  DownloadOutlined,
  ImportOutlined,
  KeyOutlined,
  ReloadOutlined,
  SaveOutlined,
  UndoOutlined,
} from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Checkbox as ACheckbox,
  Divider as ADivider,
  Form as AForm,
  FormItem as AFormItem,
  Input as AInput,
  InputNumber as AInputNumber,
  Modal,
  Select as ASelect,
  Space as ASpace,
  Switch as ASwitch,
  TabPane as ATabPane,
  Tabs as ATabs,
  message,
} from 'ant-design-vue';
import { computed, onMounted, ref } from 'vue';

import { downloadDatabase, importDatabase } from '@/api/server';
import {
  getAllSettings,
  getDefaultSettings,
  restartPanel,
  updateSettings,
  updateUserCredentials,
} from '@/api/settings';
import PageHeader from '@/components/PageHeader.vue';
import type { PanelSettings, UserCredentialsUpdateForm } from '@/types/settings';
import { hasInjectedRuntimeConfig } from '@/types/runtime';
import { copyText, downloadBlob } from '@/utils/textExport';

type SettingsTab =
  | 'panel'
  | 'security'
  | 'subscription'
  | 'formats'
  | 'telegram'
  | 'ldap'
  | 'backup';

const activeTab = ref<SettingsTab>('panel');
const settings = ref<PanelSettings>(createEmptySettings());
const loadedSettings = ref<PanelSettings | null>(null);
const credentials = ref<UserCredentialsUpdateForm>({
  oldUsername: '',
  oldPassword: '',
  newUsername: '',
  newPassword: '',
});
const dbFileInput = ref<HTMLInputElement | null>(null);
const loading = ref(false);
const saving = ref(false);
const downloadingDb = ref(false);
const importingDb = ref(false);
const restartingPanel = ref(false);
const updatingCredentials = ref(false);
const loadingDefaults = ref(false);
const error = ref('');

const settingsLoaded = computed(() => Boolean(loadedSettings.value));
const settingsChanged = computed(
  () =>
    Boolean(loadedSettings.value) &&
    JSON.stringify(settings.value) !== JSON.stringify(loadedSettings.value),
);
const jsonWarnings = computed(() =>
  [
    jsonFieldWarning('JSON Fragment', settings.value.subJsonFragment),
    jsonFieldWarning('JSON Noises', settings.value.subJsonNoises),
    jsonFieldWarning('JSON Mux', settings.value.subJsonMux),
    jsonFieldWarning('JSON Rules', settings.value.subJsonRules),
  ].filter(Boolean),
);
const twoFactorSetupUri = computed(() => {
  const secret = settings.value.twoFactorToken.trim();
  if (!secret) {
    return 'Generate a token to create a two-factor setup URI.';
  }
  const issuer = 'SuperXray';
  const account = 'panel';
  return `otpauth://totp/${encodeURIComponent(`${issuer}:${account}`)}?secret=${encodeURIComponent(secret)}&issuer=${encodeURIComponent(issuer)}`;
});
const subscriptionPublicLinks = computed(() =>
  [
    { label: 'URI', value: settings.value.subURI.trim() },
    { label: 'JSON', value: settings.value.subJsonURI.trim() },
    { label: 'Clash', value: settings.value.subClashURI.trim() },
  ].filter((link) => link.value.length > 0),
);
const subscriptionPublicLinkText = computed(() => {
  if (subscriptionPublicLinks.value.length === 0) {
    return 'No subscription public links configured. Use Fill Defaults or enter URI values first.';
  }
  return subscriptionPublicLinks.value.map((link) => `${link.label}: ${link.value}`).join('\n');
});

const datepickerOptions = [
  { label: 'Gregorian', value: 'gregorian' },
  { label: 'Jalali', value: 'jalali' },
];

async function loadSettings() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    const payload = await getAllSettings({ notifyOnError: false });
    settings.value = cloneSettings(payload);
    loadedSettings.value = cloneSettings(payload);
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load settings';
  } finally {
    loading.value = false;
  }
}

function resetForm() {
  if (!loadedSettings.value) {
    return;
  }
  settings.value = cloneSettings(loadedSettings.value);
  error.value = '';
}

function confirmSave() {
  Modal.confirm({
    title: 'Save panel settings?',
    content:
      'Settings are saved through the existing legacy endpoint and remain editable in old UI.',
    okText: 'Save',
    onOk: saveSettings,
  });
}

async function saveSettings() {
  saving.value = true;
  error.value = '';
  try {
    await updateSettings(settings.value, { notifyOnError: false });
    loadedSettings.value = cloneSettings(settings.value);
    void message.success('Settings saved');
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to save settings';
  } finally {
    saving.value = false;
  }
}

function confirmUpdateCredentials() {
  if (
    !credentials.value.oldUsername ||
    !credentials.value.oldPassword ||
    !credentials.value.newUsername ||
    !credentials.value.newPassword
  ) {
    error.value = 'All credential fields are required';
    return;
  }

  Modal.confirm({
    title: 'Update login credentials?',
    content: 'You may need to sign in again after this change.',
    okButtonProps: { danger: true },
    okText: 'Update',
    onOk: updateCredentials,
  });
}

function generateTwoFactorToken() {
  settings.value.twoFactorEnable = true;
  settings.value.twoFactorToken = randomBase32Secret();
  void message.success('Two-factor token generated; save settings to apply it');
}

function disableTwoFactor() {
  settings.value.twoFactorEnable = false;
  settings.value.twoFactorToken = '';
  void message.success('Two-factor settings cleared; save settings to apply it');
}

async function copySubscriptionLinks() {
  if (subscriptionPublicLinks.value.length === 0) {
    void message.warning('No subscription public links to copy');
    return;
  }
  await copyText(subscriptionPublicLinkText.value);
  void message.success('Subscription links copied');
}

function openPublicLink(value: string) {
  const link = value.trim();
  if (!link) {
    void message.warning('Subscription link is empty');
    return;
  }
  window.open(link, '_blank', 'noopener,noreferrer');
}

async function updateCredentials() {
  updatingCredentials.value = true;
  error.value = '';
  try {
    await updateUserCredentials(credentials.value, { notifyOnError: false });
    credentials.value = {
      oldUsername: '',
      oldPassword: '',
      newUsername: '',
      newPassword: '',
    };
    void message.success('Credentials updated');
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to update credentials';
  } finally {
    updatingCredentials.value = false;
  }
}

async function downloadDb() {
  downloadingDb.value = true;
  error.value = '';
  try {
    const blob = await downloadDatabase({ notifyOnError: false });
    downloadBlob(`superxray-${new Date().toISOString().slice(0, 10)}.db`, blob);
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to download database';
  } finally {
    downloadingDb.value = false;
  }
}

function openDbFilePicker() {
  dbFileInput.value?.click();
}

function handleDbFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file) {
    return;
  }

  Modal.confirm({
    title: `Import ${file.name}?`,
    content: 'The legacy import path validates the SQLite database and restarts Xray after import.',
    okButtonProps: { danger: true },
    okText: 'Import',
    onOk: () => runImportDb(file),
  });
}

async function runImportDb(file: File) {
  importingDb.value = true;
  error.value = '';
  try {
    await importDatabase(file, { notifyOnError: false });
    void message.success('Database imported');
    await loadSettings();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to import database';
  } finally {
    importingDb.value = false;
  }
}

function confirmRestartPanel() {
  Modal.confirm({
    title: 'Restart panel?',
    content: 'The panel process will restart through the existing legacy path.',
    okButtonProps: { danger: true },
    okText: 'Restart',
    onOk: runRestartPanel,
  });
}

async function runRestartPanel() {
  restartingPanel.value = true;
  error.value = '';
  try {
    await restartPanel({ notifyOnError: false });
    void message.success('Panel restart command sent');
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to restart panel';
  } finally {
    restartingPanel.value = false;
  }
}

async function applySubscriptionDefaults() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingDefaults.value = true;
  error.value = '';
  try {
    const defaults = await getDefaultSettings({ notifyOnError: false });
    settings.value.subTitle = defaults.subTitle || settings.value.subTitle;
    settings.value.subURI = defaults.subURI || settings.value.subURI;
    settings.value.subJsonURI = defaults.subJsonURI || settings.value.subJsonURI;
    settings.value.subClashURI = defaults.subClashURI || settings.value.subClashURI;
    void message.success('Default subscription URIs applied');
  } catch (caught) {
    error.value =
      caught instanceof Error ? caught.message : 'Failed to generate subscription defaults';
  } finally {
    loadingDefaults.value = false;
  }
}

function cloneSettings(value: PanelSettings): PanelSettings {
  return JSON.parse(JSON.stringify(value)) as PanelSettings;
}

function jsonFieldWarning(label: string, value: string): string {
  if (!value.trim()) {
    return '';
  }
  try {
    JSON.parse(value);
    return '';
  } catch {
    return `${label} is not valid JSON`;
  }
}

function randomBase32Secret(length = 32): string {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  const bytes = new Uint8Array(length);
  window.crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => alphabet[byte % alphabet.length]).join('');
}

function createEmptySettings(): PanelSettings {
  return {
    webListen: '',
    webDomain: '',
    webPort: 2053,
    webCertFile: '',
    webKeyFile: '',
    webBasePath: '/',
    sessionMaxAge: 360,
    pageSize: 25,
    expireDiff: 0,
    trafficDiff: 0,
    remarkModel: '-ieo',
    datepicker: 'gregorian',
    tgBotEnable: false,
    tgBotToken: '',
    tgBotProxy: '',
    tgBotAPIServer: '',
    tgBotChatId: '',
    tgRunTime: '@daily',
    tgBotBackup: false,
    tgBotLoginNotify: true,
    tgCpu: 80,
    tgLang: 'en-US',
    timeLocation: 'Local',
    twoFactorEnable: false,
    twoFactorToken: '',
    subEnable: true,
    subJsonEnable: false,
    subTitle: '',
    subSupportUrl: '',
    subProfileUrl: '',
    subAnnounce: '',
    subEnableRouting: true,
    subRoutingRules: '',
    subListen: '',
    subPort: 2096,
    subPath: '/sub/',
    subDomain: '',
    subCertFile: '',
    subKeyFile: '',
    subUpdates: 12,
    externalTrafficInformEnable: false,
    externalTrafficInformURI: '',
    subEncrypt: true,
    subShowInfo: true,
    subURI: '',
    subJsonPath: '/json/',
    subJsonURI: '',
    subClashEnable: true,
    subClashPath: '/clash/',
    subClashURI: '',
    subJsonFragment: '',
    subJsonNoises: '',
    subJsonMux: '',
    subJsonRules: '',
    ldapEnable: false,
    ldapHost: '',
    ldapPort: 389,
    ldapUseTLS: false,
    ldapBindDN: '',
    ldapPassword: '',
    ldapBaseDN: '',
    ldapUserFilter: '(objectClass=person)',
    ldapUserAttr: 'mail',
    ldapVlessField: 'vless_enabled',
    ldapSyncCron: '@every 1m',
    ldapFlagField: '',
    ldapTruthyValues: 'true,1,yes,on',
    ldapInvertFlag: false,
    ldapInboundTags: '',
    ldapAutoCreate: false,
    ldapAutoDelete: false,
    ldapDefaultTotalGB: 0,
    ldapDefaultExpiryDays: 0,
    ldapDefaultLimitIP: 0,
  };
}

onMounted(() => {
  void loadSettings();
});
</script>
