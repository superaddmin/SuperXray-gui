<template>
  <section class="page-stack">
    <PageHeader eyebrow="Overview" title="Dashboard">
      <AButton :loading="refreshing" type="primary" @click="refreshDashboard">
        <template #icon><ReloadOutlined /></template>
        Refresh
      </AButton>
    </PageHeader>

    <AAlert
      v-if="serverStore.statusError || inboundsError"
      banner
      type="warning"
      :message="serverStore.statusError || inboundsError"
    />

    <div class="status-grid">
      <StatusTile label="Xray State" :value="xrayStateLabel" :hint="xrayVersionLabel" />
      <StatusTile label="CPU" :value="formatPercent(status?.cpu)" :hint="cpuHint" />
      <StatusTile label="Memory" :value="memoryPercent" :hint="memoryHint" />
      <StatusTile label="Traffic" :value="trafficTotal" :hint="trafficHint" />
      <StatusTile label="Inbounds" :value="formatCount(inboundCount)" :hint="enabledInboundHint" />
      <StatusTile label="Clients" :value="formatCount(clientCount)" :hint="enabledClientHint" />
      <StatusTile label="Panel Uptime" :value="formatDuration(status?.uptime)" :hint="appHint" />
      <StatusTile label="Connections" :value="connectionTotal" :hint="connectionHint" />
    </div>

    <ACard class="work-panel geo-panel" :bordered="false" title="Geo Maintenance">
      <template #extra>
        <ASpace wrap>
          <AButton :loading="geoAction === 'geoip.dat'" @click="handleUpdateGeofile('geoip.dat')">
            <template #icon><CloudDownloadOutlined /></template>
            Update geoip.dat
          </AButton>
          <AButton
            :loading="geoAction === 'geosite.dat'"
            @click="handleUpdateGeofile('geosite.dat')"
          >
            <template #icon><CloudDownloadOutlined /></template>
            Update geosite.dat
          </AButton>
          <AButton
            :loading="geoAction === 'all-geofiles'"
            type="primary"
            @click="handleUpdateGeofile()"
          >
            <template #icon><ReloadOutlined /></template>
            Update all
          </AButton>
        </ASpace>
      </template>

      <div class="section-toolbar">
        <div>
          <h2>Custom Geo</h2>
          <p>Manage external geoip/geosite resources without leaving the new UI.</p>
        </div>
        <ASpace wrap>
          <AButton :loading="customGeoUpdatingAll" @click="handleUpdateAllCustomGeo">
            <template #icon><ReloadOutlined /></template>
            Update Resources
          </AButton>
          <AButton type="primary" @click="openCustomGeoModal()">
            <template #icon><PlusOutlined /></template>
            Add Resource
          </AButton>
        </ASpace>
      </div>

      <ATable
        :columns="customGeoColumns"
        :data-source="customGeoRows"
        :loading="customGeoLoading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <ATag color="blue">{{ record.type }}</ATag>
          </template>
          <template v-else-if="column.key === 'ext'">
            <code>{{ customGeoExtDisplay(record) }}</code>
          </template>
          <template v-else-if="column.key === 'updated'">
            {{ formatCustomGeoTime(record.lastUpdatedAt) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <ASpace>
              <AButton size="small" type="link" @click="openCustomGeoModal(record)">Edit</AButton>
              <AButton
                size="small"
                type="link"
                :loading="customGeoActionId === record.id"
                @click="handleDownloadCustomGeo(record.id)"
              >
                Download
              </AButton>
              <AButton danger size="small" type="link" @click="handleDeleteCustomGeo(record)">
                Delete
              </AButton>
            </ASpace>
          </template>
        </template>
      </ATable>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <ATable
        :columns="columns"
        :data-source="rows"
        :pagination="false"
        size="middle"
        row-key="key"
      />
    </ACard>

    <AModal
      v-model:open="customGeoModalOpen"
      :confirm-loading="customGeoSaving"
      :title="customGeoEditingId ? 'Edit Resource' : 'Add Resource'"
      @ok="handleSaveCustomGeo"
    >
      <AForm layout="vertical" :model="customGeoForm">
        <AFormItem label="Type">
          <ASelect v-model:value="customGeoForm.type" :disabled="Boolean(customGeoEditingId)">
            <ASelectOption value="geoip">geoip</ASelectOption>
            <ASelectOption value="geosite">geosite</ASelectOption>
          </ASelect>
        </AFormItem>
        <AFormItem label="Alias">
          <AInput
            v-model:value="customGeoForm.alias"
            :disabled="Boolean(customGeoEditingId)"
            placeholder="private"
          />
        </AFormItem>
        <AFormItem label="URL">
          <AInput v-model:value="customGeoForm.url" placeholder="https://example.com/geo.dat" />
        </AFormItem>
      </AForm>
    </AModal>
  </section>
</template>

<script setup lang="ts">
import { CloudDownloadOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Form as AForm,
  FormItem as AFormItem,
  Input as AInput,
  message,
  Modal,
  Modal as AModal,
  Select as ASelect,
  SelectOption as ASelectOption,
  Space as ASpace,
  Table as ATable,
  Tag as ATag,
} from 'ant-design-vue';
import { computed, onMounted, ref } from 'vue';

import {
  addCustomGeo,
  deleteCustomGeo,
  downloadCustomGeo,
  listCustomGeo,
  updateAllCustomGeo,
  updateCustomGeo,
} from '@/api/customGeo';
import { listInbounds } from '@/api/inbounds';
import { updateGeofile } from '@/api/server';
import PageHeader from '@/components/PageHeader.vue';
import StatusTile from '@/components/StatusTile.vue';
import { useServerStore } from '@/stores/server';
import type { CustomGeoForm, CustomGeoResource, CustomGeoType } from '@/types/customGeo';
import type { Inbound } from '@/types/inbound';
import { hasInjectedRuntimeConfig } from '@/types/runtime';
import { formatBytes, formatCount, formatDuration, formatPercent } from '@/utils/format';

const serverStore = useServerStore();
const inbounds = ref<Inbound[]>([]);
const loadingInbounds = ref(false);
const inboundsError = ref('');
const customGeoRows = ref<CustomGeoResource[]>([]);
const customGeoLoading = ref(false);
const customGeoSaving = ref(false);
const customGeoUpdatingAll = ref(false);
const customGeoActionId = ref<number>();
const customGeoEditingId = ref<number>();
const customGeoModalOpen = ref(false);
const geoAction = ref('');
const customGeoForm = ref<CustomGeoForm>({
  alias: '',
  type: 'geoip',
  url: '',
});
const status = computed(() => serverStore.status);
const refreshing = computed(
  () => serverStore.loadingStatus || loadingInbounds.value || customGeoLoading.value,
);

const inboundCount = computed(() => inbounds.value.length);
const enabledInboundCount = computed(
  () => inbounds.value.filter((inbound) => inbound.enable).length,
);
const clientCount = computed(() =>
  inbounds.value.reduce((total, inbound) => total + (inbound.clientStats?.length || 0), 0),
);
const enabledClientCount = computed(() =>
  inbounds.value.reduce(
    (total, inbound) =>
      total + (inbound.clientStats || []).filter((client) => client.enable).length,
    0,
  ),
);

const xrayStateLabel = computed(() => {
  const state = status.value?.xray.state;
  if (!state) {
    return '-';
  }
  return state.charAt(0).toUpperCase() + state.slice(1);
});
const xrayVersionLabel = computed(() => `Version ${status.value?.xray.version || '-'}`);
const cpuHint = computed(() => {
  const cores = status.value?.cpuCores;
  const logical = status.value?.logicalPro;
  if (!cores && !logical) {
    return '-';
  }
  return `${cores || '-'} cores / ${logical || '-'} logical`;
});
const memoryPercent = computed(() => {
  const mem = status.value?.mem;
  if (!mem?.total) {
    return '-';
  }
  return formatPercent((mem.current / mem.total) * 100);
});
const memoryHint = computed(() => {
  const mem = status.value?.mem;
  if (!mem) {
    return '-';
  }
  return `${formatBytes(mem.current)} / ${formatBytes(mem.total)}`;
});
const trafficTotal = computed(() => {
  const traffic = status.value?.netTraffic;
  if (!traffic) {
    return '-';
  }
  return formatBytes(traffic.sent + traffic.recv);
});
const trafficHint = computed(() => {
  const netIO = status.value?.netIO;
  if (!netIO) {
    return '-';
  }
  return `${formatBytes(netIO.up)}/s up, ${formatBytes(netIO.down)}/s down`;
});
const enabledInboundHint = computed(() => `${formatCount(enabledInboundCount.value)} enabled`);
const enabledClientHint = computed(() => `${formatCount(enabledClientCount.value)} enabled`);
const appHint = computed(() => {
  const appStats = status.value?.appStats;
  if (!appStats) {
    return '-';
  }
  return `${formatCount(appStats.threads)} threads, ${formatBytes(appStats.mem)} app memory`;
});
const connectionTotal = computed(() => {
  const current = status.value;
  if (!current) {
    return '-';
  }
  return formatCount(current.tcpCount + current.udpCount);
});
const connectionHint = computed(() => {
  const current = status.value;
  if (!current) {
    return '-';
  }
  return `${formatCount(current.tcpCount)} TCP / ${formatCount(current.udpCount)} UDP`;
});

const columns = [
  { title: 'Metric', dataIndex: 'metric', key: 'metric' },
  { title: 'Value', dataIndex: 'value', key: 'value' },
  { title: 'Detail', dataIndex: 'detail', key: 'detail' },
];

const customGeoColumns = [
  { title: 'Type', dataIndex: 'type', key: 'type', width: 110 },
  { title: 'Alias', dataIndex: 'alias', key: 'alias', width: 150 },
  { title: 'External Code', key: 'ext' },
  { title: 'URL', dataIndex: 'url', key: 'url', ellipsis: true },
  { title: 'Last Updated', key: 'updated', width: 180 },
  { title: 'Actions', key: 'action', width: 220 },
];

const rows = computed(() => [
  {
    key: 'public-ip',
    metric: 'Public IP',
    value: status.value?.publicIP.ipv4 || '-',
    detail: status.value?.publicIP.ipv6 || '-',
  },
  {
    key: 'load',
    metric: 'Load Average',
    value: status.value?.loads?.join(' / ') || '-',
    detail: `CPU speed ${status.value?.cpuSpeedMhz ? `${status.value.cpuSpeedMhz.toFixed(0)} MHz` : '-'}`,
  },
  {
    key: 'disk',
    metric: 'Disk',
    value: diskPercent.value,
    detail: diskHint.value,
  },
  {
    key: 'swap',
    metric: 'Swap',
    value: swapPercent.value,
    detail: swapHint.value,
  },
]);

const diskPercent = computed(() => {
  const disk = status.value?.disk;
  return disk?.total ? formatPercent((disk.current / disk.total) * 100) : '-';
});
const diskHint = computed(() => {
  const disk = status.value?.disk;
  return disk ? `${formatBytes(disk.current)} / ${formatBytes(disk.total)}` : '-';
});
const swapPercent = computed(() => {
  const swap = status.value?.swap;
  return swap?.total ? formatPercent((swap.current / swap.total) * 100) : '-';
});
const swapHint = computed(() => {
  const swap = status.value?.swap;
  return swap ? `${formatBytes(swap.current)} / ${formatBytes(swap.total)}` : '-';
});

async function refreshDashboard() {
  void serverStore.refreshStatus();
  await Promise.all([refreshInbounds(), refreshCustomGeo()]);
}

async function refreshInbounds() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingInbounds.value = true;
  inboundsError.value = '';
  try {
    inbounds.value = await listInbounds({ notifyOnError: false });
  } catch (error) {
    inboundsError.value = error instanceof Error ? error.message : 'Failed to load inbounds';
  } finally {
    loadingInbounds.value = false;
  }
}

async function refreshCustomGeo() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  customGeoLoading.value = true;
  try {
    customGeoRows.value = await listCustomGeo({ notifyOnError: false });
  } catch (error) {
    inboundsError.value = error instanceof Error ? error.message : 'Failed to load custom geo';
  } finally {
    customGeoLoading.value = false;
  }
}

async function handleUpdateGeofile(fileName?: string) {
  geoAction.value = fileName || 'all-geofiles';
  try {
    await updateGeofile(fileName);
    void message.success(fileName ? `${fileName} update started` : 'Geofile update started');
  } finally {
    geoAction.value = '';
  }
}

function openCustomGeoModal(record?: Partial<CustomGeoResource>) {
  customGeoEditingId.value = record?.id;
  customGeoForm.value = {
    alias: record?.alias || '',
    type: (record?.type || 'geoip') as CustomGeoType,
    url: record?.url || '',
  };
  customGeoModalOpen.value = true;
}

async function handleSaveCustomGeo() {
  const form = customGeoForm.value;
  if (!/^[A-Za-z0-9_-]+$/.test(form.alias)) {
    void message.error('Alias can only contain letters, numbers, underscores, and hyphens');
    return;
  }
  if (!/^https?:\/\/[^/\s]+/i.test(form.url)) {
    void message.error('URL must start with http:// or https:// and include a host');
    return;
  }

  customGeoSaving.value = true;
  try {
    if (customGeoEditingId.value) {
      await updateCustomGeo(customGeoEditingId.value, form);
    } else {
      await addCustomGeo(form);
    }
    customGeoModalOpen.value = false;
    await refreshCustomGeo();
  } finally {
    customGeoSaving.value = false;
  }
}

function handleDeleteCustomGeo(record: Partial<CustomGeoResource>) {
  if (!record.id || !record.alias) {
    return;
  }
  const id = record.id;
  Modal.confirm({
    title: `Delete ${record.alias}?`,
    content: 'This removes the custom geo resource from the panel configuration.',
    okButtonProps: { danger: true },
    onOk: async () => {
      await deleteCustomGeo(id);
      await refreshCustomGeo();
    },
  });
}

async function handleDownloadCustomGeo(id: number) {
  customGeoActionId.value = id;
  try {
    await downloadCustomGeo(id);
    void message.success('Custom geo update started');
    await refreshCustomGeo();
  } finally {
    customGeoActionId.value = undefined;
  }
}

async function handleUpdateAllCustomGeo() {
  customGeoUpdatingAll.value = true;
  try {
    await updateAllCustomGeo();
    void message.success('Custom geo resource updates started');
    await refreshCustomGeo();
  } finally {
    customGeoUpdatingAll.value = false;
  }
}

function customGeoExtDisplay(record: Partial<CustomGeoResource>): string {
  const fileName =
    record.type === 'geoip' ? `geoip_${record.alias}.dat` : `geosite_${record.alias}.dat`;
  return `ext:${fileName}:tag`;
}

function formatCustomGeoTime(value: number): string {
  if (!value) {
    return '-';
  }
  return new Date(value * 1000).toLocaleString();
}

onMounted(() => {
  serverStore.connectRealtime();
  void refreshDashboard();
});
</script>
