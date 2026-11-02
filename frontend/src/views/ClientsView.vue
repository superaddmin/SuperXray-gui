<template>
  <section class="page-stack clients-page" :aria-busy="loading">
    <PageHeader
      eyebrow="Overview"
      title="Clients"
      :description="translate('clients.description', appStore.locale)"
    >
      <AButton
        :aria-label="translate('clients.refreshLabel', appStore.locale)"
        class="clients-refresh-button"
        :disabled="Boolean(clientActionKey)"
        :loading="loading"
        type="primary"
        @click="refresh"
      >
        <template #icon><ReloadOutlined /></template>
        Refresh
      </AButton>
    </PageHeader>

    <AAlert
      v-if="error"
      class="clients-error"
      banner
      closable
      show-icon
      type="warning"
      :message="error"
      @close="error = ''"
    />

    <div class="status-grid">
      <StatusTile
        label="Clients"
        :value="formatCount(rows.length)"
        hint="Across all inbounds"
        tone="success"
      />
      <StatusTile
        label="Enabled"
        :value="formatCount(enabledCount)"
        hint="Enabled client records"
        tone="success"
      />
      <StatusTile
        label="Online"
        :value="formatCount(onlineCount)"
        hint="Currently online"
        tone="info"
      />
      <StatusTile
        label="Inbounds"
        :value="formatCount(inbounds.length)"
        hint="Source inbounds"
        tone="neutral"
      />
    </div>

    <ACard class="work-panel" :bordered="false">
      <div class="section-toolbar clients-toolbar">
        <div class="clients-toolbar-copy">
          <div class="clients-title-row">
            <h2>{{ translate('clients.detailsTitle', appStore.locale) }}</h2>
            <ATag class="clients-result-tag" color="blue">
              {{ resultSummary }}
            </ATag>
          </div>
          <p>
            {{ translate('clients.detailsDescription', appStore.locale) }}
          </p>
        </div>
        <div
          class="client-filter-toolbar"
          role="search"
          :aria-label="translate('clients.filterLabel', appStore.locale)"
        >
          <AInput
            v-model:value="search"
            :aria-label="translate('clients.searchLabel', appStore.locale)"
            class="client-filter-input"
            allow-clear
            :placeholder="translate('clients.searchPlaceholder', appStore.locale)"
          >
            <template #prefix><SearchOutlined /></template>
          </AInput>
          <ASelect
            v-model:value="inboundFilter"
            :aria-label="translate('clients.filterInboundLabel', appStore.locale)"
            class="client-filter-select"
          >
            <ASelectOption value="all">
              {{ translate('clients.allInbounds', appStore.locale) }}
            </ASelectOption>
            <ASelectOption v-for="inbound in inbounds" :key="inbound.id" :value="inbound.id">
              {{ inboundLabel(inbound) }}
            </ASelectOption>
          </ASelect>
          <AButton v-if="hasFilters" class="client-clear-filters" type="text" @click="clearFilters">
            {{ translate('clients.clearFilters', appStore.locale) }}
          </AButton>
        </div>
      </div>

      <div class="client-table-meta" aria-live="polite">
        <span>{{ resultSummary }}</span>
        <span v-if="hasFilters" class="client-table-meta-hint">
          {{ translate('clients.filtersActive', appStore.locale) }}
        </span>
      </div>

      <ATable
        class="client-table"
        :columns="columns"
        :data-source="filteredRows"
        :locale="tableLocale"
        :loading="loading"
        :pagination="{ pageSize: 20, showSizeChanger: true }"
        :row-class-name="rowClassName"
        row-key="key"
        size="middle"
        :scroll="{ x: 1500 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'inbound'">
            <strong>{{ record.inboundLabel }}</strong>
            <div class="muted-text">{{ record.inboundMeta }}</div>
          </template>
          <template v-else-if="column.key === 'email'">
            {{ record.email || '-' }}
          </template>
          <template v-else-if="column.key === 'protocol'">
            <ATag>{{ record.protocol }}</ATag>
          </template>
          <template v-else-if="column.key === 'credential'">
            <code class="client-credential">{{ maskSecret(record.credential) || '-' }}</code>
          </template>
          <template v-else-if="column.key === 'subId'">
            <code class="client-credential">{{ maskSecret(record.subId) || '-' }}</code>
          </template>
          <template v-else-if="column.key === 'traffic'">
            {{ record.trafficLabel }}
          </template>
          <template v-else-if="column.key === 'expiry'">
            {{ record.expiryLabel }}
          </template>
          <template v-else-if="column.key === 'enabled'">
            <ATag :color="record.enabled ? 'green' : 'red'">
              {{ record.enabled ? 'Enabled' : 'Disabled' }}
            </ATag>
          </template>
          <template v-else-if="column.key === 'online'">
            <ATag :color="record.online ? 'green' : undefined">
              {{ record.online ? 'Online' : 'Offline' }}
            </ATag>
          </template>
          <template v-else-if="column.key === 'lastOnline'">
            {{ record.lastOnlineLabel }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <AButton
              :aria-label="clientActionAriaLabel(asClientRow(record))"
              class="client-action-button"
              :danger="record.clientEnabled"
              :disabled="clientActionDisabled(asClientRow(record))"
              :loading="clientActionKey === record.key"
              size="middle"
              type="text"
              @click="toggleClient(asClientRow(record))"
            >
              <template #icon>
                <StopOutlined v-if="record.clientEnabled" />
                <PlayCircleOutlined v-else />
              </template>
              {{ record.clientEnabled ? 'Stop' : 'Start' }}
            </AButton>
          </template>
        </template>
      </ATable>
    </ACard>
  </section>
</template>

<script setup lang="ts">
import {
  PlayCircleOutlined,
  ReloadOutlined,
  SearchOutlined,
  StopOutlined,
} from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Input as AInput,
  Select as ASelect,
  SelectOption as ASelectOption,
  Table as ATable,
  Tag as ATag,
} from 'ant-design-vue';
import type { TableColumnsType } from 'ant-design-vue';
import { computed, onMounted, ref } from 'vue';

import {
  getClientsLastOnline,
  getOnlineClients,
  listInbounds,
  updateInbound,
  updateInboundClient,
} from '@/api/inbounds';
import PageHeader from '@/components/PageHeader.vue';
import StatusTile from '@/components/StatusTile.vue';
import { translate } from '@/i18n/messages';
import { protocolSupportsClients } from '@/schemas/protocolRegistry';
import { useAppStore } from '@/stores/app';
import type { Inbound, InboundClient } from '@/types/inbound';
import { hasInjectedRuntimeConfig } from '@/types/runtime';
import { formatBytes, formatCount } from '@/utils/format';
import {
  getClientPrimaryId,
  getInboundClients,
  parseInboundSettings,
  stringifyJson,
} from '@/utils/inboundCompat';

interface AggregatedClientRow {
  key: string;
  inboundId: number;
  inboundLabel: string;
  inboundMeta: string;
  email: string;
  protocol: string;
  credential: string;
  subId: string;
  trafficLabel: string;
  expiryLabel: string;
  enabled: boolean;
  client: InboundClient;
  clientEnabled: boolean;
  online: boolean;
  lastOnlineLabel: string;
}

const appStore = useAppStore();
const inbounds = ref<Inbound[]>([]);
const onlineClients = ref<string[]>([]);
const lastOnlineMap = ref<Record<string, number>>({});
const loading = ref(false);
const error = ref('');
const search = ref('');
const inboundFilter = ref<number | 'all'>('all');
const clientActionKey = ref('');

const columns: TableColumnsType<AggregatedClientRow> = [
  { title: 'Inbound', key: 'inbound', width: 230 },
  { title: 'Email', key: 'email', width: 220 },
  { title: 'Protocol', key: 'protocol', width: 110 },
  { title: 'Credential', key: 'credential', width: 280 },
  { title: 'Sub ID', key: 'subId', width: 220 },
  { title: 'Traffic', key: 'traffic', width: 180 },
  { title: 'Expiry', key: 'expiry', width: 190 },
  { title: 'Enabled', key: 'enabled', width: 110 },
  { title: 'Online', key: 'online', width: 110 },
  { title: 'Last Online', key: 'lastOnline', width: 190 },
  { title: 'Actions', key: 'actions', width: 110, fixed: 'right' },
];

const rows = computed<AggregatedClientRow[]>(() =>
  inbounds.value.flatMap((inbound) => {
    if (!protocolSupportsClients(inbound.protocol)) {
      return [];
    }
    return getInboundClients(inbound).map((client, index) => {
      const stats = inbound.clientStats?.find((item) => item.email === client.email);
      const credential = getClientPrimaryId(inbound.protocol, client);
      const expiryTime = Number(stats?.expiryTime || client.expiryTime || 0);
      const used = Number(stats?.up || 0) + Number(stats?.down || 0);
      const total = Number(stats?.total ?? client.totalGB ?? 0);
      const lastOnline = lastOnlineMap.value[client.email] || stats?.lastOnline || 0;
      return {
        key: `${inbound.id}:${credential || client.email || index}`,
        inboundId: inbound.id,
        inboundLabel: inboundLabel(inbound),
        inboundMeta: `${inbound.protocol} · ${inbound.listen || '0.0.0.0'}:${inbound.port}`,
        email: client.email || '',
        protocol: inbound.protocol,
        credential,
        subId: String(client.subId || stats?.subId || ''),
        trafficLabel: formatTraffic(used, total),
        expiryLabel: formatTimestamp(expiryTime),
        enabled: client.enable !== false && (stats?.enable ?? true),
        client: { ...client },
        clientEnabled: client.enable !== false,
        online: onlineClients.value.includes(client.email),
        lastOnlineLabel: formatTimestamp(lastOnline),
      };
    });
  }),
);

const filteredRows = computed(() => {
  const term = search.value.trim().toLowerCase();
  return rows.value.filter((row) => {
    if (inboundFilter.value !== 'all' && row.inboundId !== inboundFilter.value) {
      return false;
    }
    if (!term) {
      return true;
    }
    return [row.email, row.credential, row.subId, row.inboundLabel].some((value) =>
      String(value).toLowerCase().includes(term),
    );
  });
});

const enabledCount = computed(() => rows.value.filter((row) => row.enabled).length);
const onlineCount = computed(() => rows.value.filter((row) => row.online).length);
const hasFilters = computed(() => Boolean(search.value.trim()) || inboundFilter.value !== 'all');
const resultSummary = computed(() => {
  const visible = filteredRows.value.length;
  const total = rows.value.length;
  return appStore.locale === 'zh-CN'
    ? hasFilters.value
      ? `${visible} / ${total} 个客户端`
      : `${total} 个客户端`
    : hasFilters.value
      ? `${visible} of ${total} clients shown`
      : `${total} clients`;
});
const tableLocale = computed(() => ({
  emptyText:
    rows.value.length === 0
      ? translate('clients.noSupported', appStore.locale)
      : translate('clients.noMatches', appStore.locale),
}));

function inboundLabel(inbound: Inbound): string {
  return inbound.remark || inbound.tag || `Inbound ${inbound.id}`;
}

function formatTraffic(used: number, total: number): string {
  const usedLabel = formatBytes(used);
  return total > 0 ? `${usedLabel} / ${formatBytes(total)}` : `${usedLabel} / Unlimited`;
}

function maskSecret(value: string): string {
  return value.trim() ? '••••••••' : '';
}

function formatTimestamp(value: number): string {
  if (!value) {
    return '-';
  }
  if (value < 0) {
    return `${Math.abs(Math.round(value / 86_400_000))} days`;
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
}

function asClientRow(record: AggregatedClientRow | Record<string, unknown>): AggregatedClientRow {
  return record as AggregatedClientRow;
}

function clearFilters() {
  search.value = '';
  inboundFilter.value = 'all';
}

function rowClassName(record: AggregatedClientRow): string {
  return record.clientEnabled ? '' : 'client-row-disabled';
}

function clientActionAriaLabel(row: AggregatedClientRow): string {
  const action = translate(row.clientEnabled ? 'action.stop' : 'action.start', appStore.locale);
  const client = translate('common.clients', appStore.locale);
  return appStore.locale === 'zh-CN'
    ? `${action}${client} ${row.email || row.key}`
    : `${action} client ${row.email || row.key}`;
}

function clientActionDisabled(row: AggregatedClientRow): boolean {
  return (
    Boolean(clientActionKey.value) ||
    !inbounds.value.some((inbound) => inbound.id === row.inboundId) ||
    !getClientPrimaryId(row.protocol, row.client)
  );
}

async function toggleClient(row: AggregatedClientRow) {
  if (clientActionDisabled(row)) {
    return;
  }
  const inbound = inbounds.value.find((item) => item.id === row.inboundId);
  if (!inbound) {
    return;
  }

  const enabled = !row.clientEnabled;
  const clientId = getClientPrimaryId(row.protocol, row.client);
  clientActionKey.value = row.key;
  error.value = '';
  try {
    if (row.protocol === 'wireguard') {
      const settings = parseInboundSettings(inbound);
      const peers = Array.isArray(settings.peers) ? settings.peers : [];
      let matched = false;
      settings.peers = peers.map((peer) => {
        if (getClientPrimaryId(row.protocol, peer) !== clientId) {
          return peer;
        }
        matched = true;
        return { ...peer, enable: enabled };
      });
      if (!matched) {
        throw new Error('Client no longer exists');
      }
      await updateInbound(
        inbound.id,
        { ...inbound, settings: stringifyJson(settings) },
        { notifyOnError: false },
      );
    } else {
      await updateInboundClient(
        clientId,
        {
          id: inbound.id,
          settings: stringifyJson({ clients: [{ ...row.client, enable: enabled }] }),
        },
        { notifyOnError: false },
      );
    }
    await refresh();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to update client';
  } finally {
    clientActionKey.value = '';
  }
}

async function refresh() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }
  loading.value = true;
  error.value = '';
  try {
    const [inboundList, online, lastOnline] = await Promise.all([
      listInbounds({ notifyOnError: false }),
      getOnlineClients({ notifyOnError: false }),
      getClientsLastOnline({ notifyOnError: false }),
    ]);
    inbounds.value = inboundList;
    onlineClients.value = Array.isArray(online) ? online : [];
    lastOnlineMap.value = lastOnline || {};
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load clients';
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void refresh();
});
</script>

<style scoped>
.clients-page {
  --clients-surface: rgba(255, 255, 255, 0.035);
}

.clients-refresh-button {
  min-width: 112px;
  min-height: 44px;
}

.clients-error {
  margin-bottom: 0;
}

.clients-toolbar {
  align-items: stretch;
  gap: 24px;
}

.clients-toolbar-copy {
  flex: 1 1 360px;
  min-width: 0;
}

.clients-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.clients-title-row h2 {
  margin-bottom: 0;
}

.clients-result-tag {
  margin: 0;
  border: 1px solid rgba(82, 139, 255, 0.32);
  background: rgba(0, 102, 255, 0.12);
  color: var(--brand-primary-soft);
  font-size: 12px;
  line-height: 24px;
}

.client-filter-toolbar {
  display: grid;
  grid-template-columns: minmax(240px, 1.1fr) minmax(190px, 0.9fr) auto;
  align-items: center;
  gap: 10px;
  min-width: min(640px, 100%);
  padding: 10px;
  border: 1px solid var(--brand-border-soft);
  border-radius: 14px;
  background: var(--clients-surface);
}

.client-filter-input,
.client-filter-select {
  min-width: 0;
  width: 100%;
}

.client-filter-input :deep(.ant-input-affix-wrapper),
.client-filter-select :deep(.ant-select-selector) {
  min-height: 44px;
}

.client-clear-filters {
  min-height: 44px;
  padding-inline: 12px;
  color: var(--brand-muted-strong);
  white-space: nowrap;
}

.client-clear-filters:hover,
.client-clear-filters:focus {
  color: var(--brand-ink) !important;
}

.client-table-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: -2px 0 12px;
  color: var(--brand-muted-strong);
  font-size: 13px;
  font-weight: 700;
}

.client-table-meta-hint {
  color: var(--brand-primary-soft);
  font-weight: 600;
}

.client-table :deep(.client-row-disabled > td) {
  opacity: 0.68;
}

.client-table :deep(.client-row-disabled:hover > td) {
  opacity: 0.86;
}

.client-table :deep(.client-row-disabled .client-action-button) {
  opacity: 1;
}

.client-action-button {
  min-width: 84px;
  min-height: 44px;
  padding-inline: 8px;
  font-weight: 800;
}

.client-credential {
  overflow-wrap: anywhere;
  font-size: 12px;
  word-break: normal;
}

@media (max-width: 1100px) {
  .clients-toolbar {
    flex-direction: column;
    gap: 14px;
  }

  .client-filter-toolbar {
    min-width: 100%;
  }
}

@media (max-width: 760px) {
  .clients-toolbar {
    gap: 14px;
  }

  .client-filter-toolbar {
    grid-template-columns: 1fr;
    padding: 8px;
  }

  .client-clear-filters {
    width: 100%;
  }

  .client-table-meta {
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
  }
}
</style>
