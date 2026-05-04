<template>
  <section class="page-stack">
    <PageHeader eyebrow="Runtime" title="Xray">
      <ASpace wrap>
        <AButton :loading="refreshing" @click="refreshRuntime">
          <template #icon><ReloadOutlined /></template>
          Refresh
        </AButton>
        <AButton :disabled="!configText" @click="copyConfig">
          <template #icon><CopyOutlined /></template>
          Copy
        </AButton>
        <AButton :disabled="!configText" @click="downloadConfig">
          <template #icon><DownloadOutlined /></template>
          Download
        </AButton>
      </ASpace>
    </PageHeader>

    <AAlert v-if="error" banner type="warning" :message="error" />

    <div class="status-grid">
      <StatusTile label="Xray State" :value="xrayStateLabel" :hint="xrayErrorHint" />
      <StatusTile label="Current Version" :value="currentVersion" hint="Existing Xray process" />
      <StatusTile label="Template" :value="templateState" :hint="configChangedHint" />
      <StatusTile
        label="Outbound Test"
        :value="outboundTestUrl || '-'"
        hint="Saved with legacy template"
      />
    </div>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Lifecycle</p>
          <h2>Xray Runtime Control</h2>
        </div>
        <ASpace wrap>
          <AButton :loading="lifecycleBusy" type="primary" @click="confirmStartOrRestart">
            <template #icon><PoweroffOutlined /></template>
            {{ startRestartLabel }}
          </AButton>
          <AButton :loading="lifecycleBusy" danger @click="confirmStop">
            <template #icon><PauseCircleOutlined /></template>
            Stop
          </AButton>
        </ASpace>
      </div>
      <pre class="code-preview compact-preview">{{ runtimeResult }}</pre>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Version</p>
          <h2>Xray Version Management</h2>
        </div>
        <ASpace wrap>
          <AButton :loading="loadingVersions" @click="loadVersions">
            <template #icon><ReloadOutlined /></template>
            Versions
          </AButton>
          <AButton
            :disabled="!selectedVersion"
            :loading="installingVersion"
            danger
            @click="confirmInstallVersion"
          >
            Install
          </AButton>
        </ASpace>
      </div>
      <ASelect
        v-model:value="selectedVersion"
        aria-label="Select Xray version"
        class="version-select"
        :loading="loadingVersions"
        :options="versionOptions"
        placeholder="Select Xray version"
        show-search
      />
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Configuration</p>
          <h2>Xray Template Editor</h2>
        </div>
        <ASpace wrap>
          <AButton :disabled="!configText" @click="formatConfig">
            <template #icon><AlignLeftOutlined /></template>
            Format
          </AButton>
          <AButton
            :disabled="!configChanged"
            :loading="savingConfig"
            type="primary"
            @click="confirmSaveConfig"
          >
            Save
          </AButton>
        </ASpace>
      </div>

      <AInput
        v-model:value="outboundTestUrl"
        aria-label="Outbound test URL"
        class="mb-12"
        placeholder="Outbound test URL"
      />
      <textarea
        v-model="configText"
        aria-label="Xray JSON template"
        class="json-editor"
        spellcheck="false"
      />
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Outbound</p>
          <h2>Outbound Tools</h2>
        </div>
        <ASpace wrap>
          <AButton :loading="loadingOutboundsTraffic" @click="loadOutboundsTraffic">
            <template #icon><ReloadOutlined /></template>
            Refresh Traffic
          </AButton>
          <AButton
            :disabled="outboundsFromConfig.length === 0"
            :loading="testingOutbound"
            type="primary"
            @click="runFirstOutboundTest"
          >
            Test First Outbound
          </AButton>
          <AButton :loading="resettingOutboundTraffic" danger @click="confirmResetAllTraffic">
            Reset All Traffic
          </AButton>
        </ASpace>
      </div>

      <ATable
        :columns="outboundTrafficColumns"
        :data-source="outboundTrafficRows"
        :loading="loadingOutboundsTraffic"
        :pagination="false"
        row-key="tag"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <AButton
              size="small"
              type="link"
              :loading="resettingOutboundTag === record.tag"
              @click="confirmResetOutboundTraffic(record.tag)"
            >
              Reset
            </AButton>
          </template>
        </template>
      </ATable>

      <pre v-if="outboundToolResult" class="code-preview compact-preview mt-16">{{
        outboundToolResult
      }}</pre>

      <div class="panel-header mt-16">
        <div>
          <p class="page-eyebrow">Providers</p>
          <h2>Warp / Nord</h2>
        </div>
        <ASpace wrap>
          <AButton :loading="providerAction === 'warp-data'" @click="runProviderData('warp')">
            Warp Data
          </AButton>
          <AButton
            :loading="providerAction === 'warp-config'"
            @click="runProviderData('warp-config')"
          >
            Warp Config
          </AButton>
          <AButton :loading="providerAction === 'nord-data'" @click="runProviderData('nord')">
            Nord Data
          </AButton>
          <AButton
            :loading="providerAction === 'nord-countries'"
            @click="runProviderData('nord-countries')"
          >
            Nord Countries
          </AButton>
        </ASpace>
      </div>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <ATable
        :columns="sectionColumns"
        :data-source="advancedRows"
        :pagination="false"
        row-key="key"
        size="middle"
      />
    </ACard>
  </section>
</template>

<script setup lang="ts">
import {
  AlignLeftOutlined,
  CopyOutlined,
  DownloadOutlined,
  PauseCircleOutlined,
  PoweroffOutlined,
  ReloadOutlined,
} from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Input as AInput,
  Modal,
  Select as ASelect,
  Space as ASpace,
  Table as ATable,
  message,
} from 'ant-design-vue';
import { computed, onMounted, ref } from 'vue';

import {
  getXrayVersions,
  installXrayVersion,
  startXrayService,
  stopXrayService,
} from '@/api/server';
import {
  getOutboundsTraffic,
  getXrayResult,
  getXraySetting,
  resetOutboundsTraffic,
  runNordAction,
  runWarpAction,
  testOutbound,
  updateXraySetting,
} from '@/api/xray';
import PageHeader from '@/components/PageHeader.vue';
import StatusTile from '@/components/StatusTile.vue';
import { useServerStore } from '@/stores/server';
import type { JsonObject, JsonValue } from '@/types/api';
import { hasInjectedRuntimeConfig } from '@/types/runtime';
import type { OutboundTraffic } from '@/types/xray';
import { formatBytes, formatCount } from '@/utils/format';
import { copyText, downloadText } from '@/utils/textExport';

const serverStore = useServerStore();
const error = ref('');
const configText = ref('');
const originalConfigText = ref('');
const outboundTestUrl = ref('');
const lifecycleBusy = ref(false);
const loadingConfig = ref(false);
const savingConfig = ref(false);
const loadingVersions = ref(false);
const installingVersion = ref(false);
const availableVersions = ref<string[]>([]);
const selectedVersion = ref<string>();
const xrayResult = ref('');
const outboundsTraffic = ref<OutboundTraffic[]>([]);
const loadingOutboundsTraffic = ref(false);
const testingOutbound = ref(false);
const resettingOutboundTraffic = ref(false);
const resettingOutboundTag = ref('');
const outboundToolResult = ref('');
const providerAction = ref('');

const status = computed(() => serverStore.status);
const refreshing = computed(
  () => serverStore.loadingStatus || loadingConfig.value || loadingOutboundsTraffic.value,
);
const currentVersion = computed(() => status.value?.xray.version || '-');
const xrayStateLabel = computed(() => {
  const state = status.value?.xray.state;
  return state ? state.charAt(0).toUpperCase() + state.slice(1) : '-';
});
const xrayErrorHint = computed(() => status.value?.xray.errorMsg || 'Existing Xray service');
const startRestartLabel = computed(() =>
  status.value?.xray.state === 'stop' ? 'Start' : 'Restart',
);
const configChanged = computed(
  () =>
    configText.value !== originalConfigText.value ||
    outboundTestUrl.value !== loadedOutboundTestUrl.value,
);
const configChangedHint = computed(() =>
  configChanged.value ? 'Unsaved changes' : 'Legacy-compatible',
);
const templateState = computed(() => (parsedConfig.value ? 'Valid JSON' : 'Invalid JSON'));
const runtimeResult = computed(() => {
  const lines = [
    `State: ${status.value?.xray.state || '-'}`,
    `Version: ${status.value?.xray.version || '-'}`,
    status.value?.xray.errorMsg ? `Status error: ${status.value.xray.errorMsg}` : '',
    xrayResult.value ? `Result: ${xrayResult.value}` : '',
  ].filter(Boolean);
  return lines.join('\n') || 'No runtime result.';
});
const versionOptions = computed(() =>
  availableVersions.value.map((version) => ({
    label: version,
    value: version,
  })),
);
const parsedConfig = computed<JsonValue | null>(() => parseJson(configText.value));
const outboundsFromConfig = computed<JsonObject[]>(() => {
  const config = parsedConfig.value;
  if (!isJsonObject(config) || !Array.isArray(config.outbounds)) {
    return [];
  }
  return config.outbounds.filter(isJsonObject);
});
const advancedRows = computed(() => [
  {
    key: 'outbounds',
    section: 'Outbounds',
    value: sectionArrayCount(parsedConfig.value, 'outbounds'),
    detail: 'Legacy outbound array in Xray template',
  },
  {
    key: 'routing',
    section: 'Routing',
    value: nestedArrayCount(parsedConfig.value, 'routing', 'rules'),
    detail: `${nestedArrayCount(parsedConfig.value, 'routing', 'balancers')} balancers`,
  },
  {
    key: 'dns',
    section: 'DNS',
    value: objectState(parsedConfig.value, 'dns'),
    detail: 'Raw DNS object remains editable in JSON',
  },
  {
    key: 'reverse',
    section: 'Reverse',
    value: objectState(parsedConfig.value, 'reverse'),
    detail: 'Reverse bridge and portal settings',
  },
  {
    key: 'fakedns',
    section: 'FakeDNS',
    value: sectionArrayCount(parsedConfig.value, 'fakedns'),
    detail: 'Top-level fakedns array when present',
  },
]);
const sectionColumns = [
  { title: 'Section', dataIndex: 'section', key: 'section' },
  { title: 'State', dataIndex: 'value', key: 'value' },
  { title: 'Detail', dataIndex: 'detail', key: 'detail' },
];
const outboundTrafficColumns = [
  { title: 'Tag', dataIndex: 'tag', key: 'tag' },
  { title: 'Up', dataIndex: 'upText', key: 'upText' },
  { title: 'Down', dataIndex: 'downText', key: 'downText' },
  { title: 'Total', dataIndex: 'totalText', key: 'totalText' },
  { title: 'Actions', key: 'action', width: 120 },
];
const outboundTrafficRows = computed(() => {
  const trafficRows = outboundsTraffic.value.map((traffic) => ({
    ...traffic,
    downText: formatBytes(traffic.down),
    totalText: formatBytes(traffic.total),
    upText: formatBytes(traffic.up),
  }));
  if (trafficRows.length > 0) {
    return trafficRows;
  }
  return outboundsFromConfig.value.map((outbound, index) => {
    const tag = typeof outbound.tag === 'string' ? outbound.tag : `outbound-${index + 1}`;
    return {
      down: 0,
      downText: '-',
      id: index,
      tag,
      total: 0,
      totalText: '-',
      up: 0,
      upText: '-',
    };
  });
});

const loadedOutboundTestUrl = ref('');

async function refreshRuntime() {
  await Promise.all([serverStore.refreshStatus(), refreshXrayResult(), loadOutboundsTraffic()]);
}

async function refreshXrayResult() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  try {
    xrayResult.value = await getXrayResult({ notifyOnError: false });
  } catch {
    xrayResult.value = '';
  }
}

async function loadOutboundsTraffic() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingOutboundsTraffic.value = true;
  try {
    outboundsTraffic.value = await getOutboundsTraffic({ notifyOnError: false });
  } catch {
    outboundsTraffic.value = [];
  } finally {
    loadingOutboundsTraffic.value = false;
  }
}

async function refreshConfig() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingConfig.value = true;
  error.value = '';
  try {
    const payload = await getXraySetting({ notifyOnError: false });
    const formatted = JSON.stringify(payload.xraySetting, null, 2);
    configText.value = formatted;
    originalConfigText.value = formatted;
    outboundTestUrl.value = payload.outboundTestUrl || '';
    loadedOutboundTestUrl.value = payload.outboundTestUrl || '';
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load Xray settings';
  } finally {
    loadingConfig.value = false;
  }
}

async function copyConfig() {
  await copyText(configText.value);
  void message.success('Copied');
}

function downloadConfig() {
  downloadText('xray-config.json', configText.value, 'application/json;charset=utf-8');
}

function formatConfig() {
  const parsed = parseJson(configText.value);
  if (!parsed) {
    error.value = 'Xray template is not valid JSON';
    return;
  }
  configText.value = JSON.stringify(parsed, null, 2);
  error.value = '';
}

function confirmStartOrRestart() {
  Modal.confirm({
    title: `${startRestartLabel.value} Xray?`,
    content: 'This uses the existing legacy Xray control path.',
    okText: startRestartLabel.value,
    onOk: runStartOrRestart,
  });
}

async function runStartOrRestart() {
  lifecycleBusy.value = true;
  error.value = '';
  try {
    await startXrayService({ notifyOnError: false });
    void message.success(`${startRestartLabel.value} command sent`);
    await refreshRuntime();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to control Xray';
  } finally {
    lifecycleBusy.value = false;
  }
}

function confirmStop() {
  Modal.confirm({
    title: 'Stop Xray?',
    content: 'Active proxy traffic may be interrupted.',
    okButtonProps: { danger: true },
    okText: 'Stop',
    onOk: runStop,
  });
}

async function runStop() {
  lifecycleBusy.value = true;
  error.value = '';
  try {
    await stopXrayService({ notifyOnError: false });
    void message.success('Stop command sent');
    await refreshRuntime();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to stop Xray';
  } finally {
    lifecycleBusy.value = false;
  }
}

async function loadVersions() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingVersions.value = true;
  error.value = '';
  try {
    availableVersions.value = await getXrayVersions({ notifyOnError: false });
    selectedVersion.value ||= availableVersions.value[0];
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load Xray versions';
  } finally {
    loadingVersions.value = false;
  }
}

function confirmInstallVersion() {
  if (!selectedVersion.value) {
    return;
  }

  Modal.confirm({
    title: `Install Xray ${selectedVersion.value}?`,
    content: 'The legacy backend will stop Xray, replace the binary, and restart Xray.',
    okButtonProps: { danger: true },
    okText: 'Install',
    onOk: installSelectedVersion,
  });
}

async function installSelectedVersion() {
  if (!selectedVersion.value) {
    return;
  }

  installingVersion.value = true;
  error.value = '';
  try {
    await installXrayVersion(selectedVersion.value, { notifyOnError: false });
    void message.success('Install command completed');
    await refreshRuntime();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to install Xray version';
  } finally {
    installingVersion.value = false;
  }
}

function confirmSaveConfig() {
  const parsed = parseJson(configText.value);
  if (!parsed) {
    error.value = 'Xray template is not valid JSON';
    return;
  }

  Modal.confirm({
    title: 'Save Xray configuration?',
    content:
      'The template will be saved through the existing legacy endpoint so the old UI can still read it.',
    okText: 'Save',
    onOk: () => saveConfig(parsed),
  });
}

async function saveConfig(parsed: JsonValue) {
  savingConfig.value = true;
  error.value = '';
  const formatted = JSON.stringify(parsed, null, 2);
  try {
    await updateXraySetting(
      {
        xraySetting: formatted,
        outboundTestUrl: outboundTestUrl.value,
      },
      { notifyOnError: false },
    );
    configText.value = formatted;
    originalConfigText.value = formatted;
    loadedOutboundTestUrl.value = outboundTestUrl.value;
    void message.success('Xray configuration saved');
    confirmRestartAfterSave();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to save Xray settings';
  } finally {
    savingConfig.value = false;
  }
}

async function runFirstOutboundTest() {
  const outbound = outboundsFromConfig.value[0];
  if (!outbound) {
    return;
  }

  testingOutbound.value = true;
  outboundToolResult.value = '';
  try {
    const result = await testOutbound(
      JSON.stringify(outbound),
      JSON.stringify(outboundsFromConfig.value),
      { notifyOnError: false },
    );
    outboundToolResult.value = result.success
      ? `Outbound test succeeded in ${result.delay} ms with status ${result.statusCode || '-'}`
      : `Outbound test failed: ${result.error || 'unknown error'}`;
  } catch (caught) {
    outboundToolResult.value = caught instanceof Error ? caught.message : 'Outbound test failed';
  } finally {
    testingOutbound.value = false;
  }
}

function confirmResetAllTraffic() {
  Modal.confirm({
    title: 'Reset all outbound traffic?',
    content: 'This resets stored traffic counters for all outbound tags.',
    okButtonProps: { danger: true },
    okText: 'Reset',
    onOk: () => resetTraffic('-alltags-'),
  });
}

function confirmResetOutboundTraffic(tag: string) {
  Modal.confirm({
    title: `Reset ${tag} traffic?`,
    content: 'This resets stored traffic counters for the selected outbound tag.',
    okButtonProps: { danger: true },
    okText: 'Reset',
    onOk: () => resetTraffic(tag),
  });
}

async function resetTraffic(tag: string) {
  resettingOutboundTraffic.value = tag === '-alltags-';
  resettingOutboundTag.value = tag === '-alltags-' ? '' : tag;
  try {
    await resetOutboundsTraffic(tag, { notifyOnError: false });
    void message.success('Outbound traffic reset');
    await loadOutboundsTraffic();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to reset outbound traffic';
  } finally {
    resettingOutboundTraffic.value = false;
    resettingOutboundTag.value = '';
  }
}

async function runProviderData(target: 'nord' | 'nord-countries' | 'warp' | 'warp-config') {
  providerAction.value = `${target}-data`;
  outboundToolResult.value = '';
  try {
    const result =
      target === 'warp'
        ? await runWarpAction('data', {}, { notifyOnError: false })
        : target === 'warp-config'
          ? await runWarpAction('config', {}, { notifyOnError: false })
          : target === 'nord-countries'
            ? await runNordAction('countries', {}, { notifyOnError: false })
            : await runNordAction('data', {}, { notifyOnError: false });
    outboundToolResult.value = result || 'No provider data returned.';
  } catch (caught) {
    outboundToolResult.value = caught instanceof Error ? caught.message : 'Provider action failed';
  } finally {
    providerAction.value = '';
  }
}

function confirmRestartAfterSave() {
  Modal.confirm({
    title: 'Restart Xray now?',
    content: 'Restart applies the saved template through the existing Xray service path.',
    okText: 'Restart',
    onOk: runStartOrRestart,
  });
}

function parseJson(value: string): JsonValue | null {
  try {
    return JSON.parse(value) as JsonValue;
  } catch {
    return null;
  }
}

function isJsonObject(value: JsonValue | null | undefined): value is JsonObject {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value));
}

function sectionArrayCount(config: JsonValue | null, key: string): string {
  if (!isJsonObject(config)) {
    return '-';
  }
  const section = config[key];
  return Array.isArray(section) ? formatCount(section.length) : objectState(config, key);
}

function nestedArrayCount(config: JsonValue | null, sectionKey: string, childKey: string): string {
  if (!isJsonObject(config)) {
    return '-';
  }
  const section = config[sectionKey];
  if (!isJsonObject(section)) {
    return '-';
  }
  const child = section[childKey];
  return Array.isArray(child) ? formatCount(child.length) : '-';
}

function objectState(config: JsonValue | null, key: string): string {
  if (!isJsonObject(config)) {
    return '-';
  }
  return config[key] ? 'Present' : 'Missing';
}

onMounted(() => {
  serverStore.connectRealtime();
  void refreshRuntime();
  void refreshConfig();
});
</script>
