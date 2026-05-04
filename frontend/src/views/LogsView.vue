<template>
  <section class="page-stack">
    <PageHeader eyebrow="Observability" title="Logs">
      <ASpace wrap>
        <ASwitch
          v-model:checked="autoFollow"
          aria-label="Toggle log auto follow"
          checked-children="Auto"
          un-checked-children="Locked"
        />
        <AButton :disabled="!logText" @click="copyLogs">
          <template #icon><CopyOutlined /></template>
          Copy
        </AButton>
        <AButton :disabled="!logText" @click="downloadLogs">
          <template #icon><DownloadOutlined /></template>
          Download
        </AButton>
        <AButton :loading="loading" type="primary" @click="refreshLogs">
          <template #icon><ReloadOutlined /></template>
          Refresh
        </AButton>
      </ASpace>
    </PageHeader>

    <AAlert v-if="error" banner type="warning" :message="error" />

    <ACard class="work-panel" :bordered="false">
      <div class="toolbar-grid">
        <ASegmented v-model:value="logSource" aria-label="Log source" :options="sourceOptions" />
        <ASelect v-model:value="lineCount" aria-label="Log line count" :options="lineOptions" />
        <ASelect
          v-if="logSource === 'panel'"
          v-model:value="panelLevel"
          aria-label="Panel log level"
          :options="levelOptions"
        />
        <AInput v-model:value="keyword" allow-clear aria-label="Filter logs" placeholder="Filter" />
        <ACheckbox
          v-if="logSource === 'panel'"
          v-model:checked="syslog"
          aria-label="Include syslog"
        >
          Syslog
        </ACheckbox>
        <ACheckbox
          v-if="logSource === 'xray'"
          v-model:checked="showDirect"
          aria-label="Show direct logs"
        >
          Direct
        </ACheckbox>
        <ACheckbox
          v-if="logSource === 'xray'"
          v-model:checked="showBlocked"
          aria-label="Show blocked logs"
        >
          Blocked
        </ACheckbox>
        <ACheckbox
          v-if="logSource === 'xray'"
          v-model:checked="showProxy"
          aria-label="Show proxy logs"
        >
          Proxy
        </ACheckbox>
      </div>

      <VirtualLogViewer v-model:auto-follow="autoFollow" :lines="visibleLines" />
    </ACard>
  </section>
</template>

<script setup lang="ts">
import { CopyOutlined, DownloadOutlined, ReloadOutlined } from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Checkbox as ACheckbox,
  Input as AInput,
  Segmented as ASegmented,
  Select as ASelect,
  Space as ASpace,
  Switch as ASwitch,
  message,
} from 'ant-design-vue';
import { computed, onMounted, ref, watch } from 'vue';

import { getPanelLogs, getXrayLogs } from '@/api/server';
import PageHeader from '@/components/PageHeader.vue';
import VirtualLogViewer from '@/components/VirtualLogViewer.vue';
import type { XrayAccessLogEntry } from '@/types/server';
import { hasInjectedRuntimeConfig } from '@/types/runtime';
import { formatDateTime } from '@/utils/format';
import { copyText, downloadText } from '@/utils/textExport';

type LogSource = 'panel' | 'xray';

const logSource = ref<LogSource>('panel');
const lineCount = ref(200);
const panelLevel = ref('all');
const keyword = ref('');
const syslog = ref(false);
const showDirect = ref(true);
const showBlocked = ref(true);
const showProxy = ref(true);
const autoFollow = ref(true);
const loading = ref(false);
const error = ref('');
const panelLines = ref<string[]>([]);
const xrayEntries = ref<XrayAccessLogEntry[]>([]);

const sourceOptions = [
  { label: 'Panel', value: 'panel' },
  { label: 'Xray', value: 'xray' },
];
const lineOptions = [100, 200, 500, 1000].map((value) => ({ label: `${value} lines`, value }));
const levelOptions = ['all', 'debug', 'info', 'notice', 'warning', 'error'].map((value) => ({
  label: value,
  value,
}));

const xrayLines = computed(() => xrayEntries.value.map(formatXrayLogEntry));
const activeLines = computed(() =>
  logSource.value === 'panel' ? panelLines.value : xrayLines.value,
);
const visibleLines = computed(() => {
  const filter = keyword.value.trim().toLowerCase();
  if (!filter || logSource.value === 'xray') {
    return activeLines.value;
  }
  return activeLines.value.filter((line) => line.toLowerCase().includes(filter));
});
const logText = computed(() => visibleLines.value.join('\n'));

watch([logSource, lineCount, panelLevel, syslog, showDirect, showBlocked, showProxy], () => {
  void refreshLogs();
});

async function refreshLogs() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    if (logSource.value === 'panel') {
      panelLines.value = await getPanelLogs(
        {
          count: lineCount.value,
          level: panelLevel.value,
          syslog: syslog.value,
        },
        { notifyOnError: false },
      );
    } else {
      xrayEntries.value = await getXrayLogs(
        {
          count: lineCount.value,
          filter: keyword.value,
          showDirect: showDirect.value,
          showBlocked: showBlocked.value,
          showProxy: showProxy.value,
        },
        { notifyOnError: false },
      );
    }
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load logs';
  } finally {
    loading.value = false;
  }
}

async function copyLogs() {
  await copyText(logText.value);
  void message.success('Copied');
}

function downloadLogs() {
  downloadText(`superxray-${logSource.value}-logs.txt`, logText.value);
}

function formatXrayLogEntry(entry: XrayAccessLogEntry): string {
  const parts = [
    formatDateTime(entry.DateTime),
    entry.FromAddress && `from ${entry.FromAddress}`,
    entry.ToAddress && `to ${entry.ToAddress}`,
    entry.Inbound && `[${entry.Inbound}`,
    entry.Outbound && `${entry.Outbound}]`,
    entry.Email && `email ${entry.Email}`,
    `event ${entry.Event}`,
  ].filter(Boolean);

  return parts.join(' ');
}

onMounted(() => {
  void refreshLogs();
});
</script>
