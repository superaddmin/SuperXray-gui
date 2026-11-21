<template>
  <section class="page-stack clients-page" :aria-busy="loading">
    <PageHeader
      :title="translate('common.clients', appStore.locale)"
      :description="translate('clients.detailsDescription', appStore.locale)"
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
        {{ translate('action.refresh', appStore.locale) }}
      </AButton>
    </PageHeader>

    <AAlert
      v-if="error"
      banner
      closable
      show-icon
      type="warning"
      :message="error"
      @close="error = ''"
    />

    <div class="status-grid clients-stats">
      <StatusTile
        label="Clients"
        :value="formatCount(rows.length)"
        :hint="translate('clients.allInboundsHint', appStore.locale)"
        tone="success"
      />
      <StatusTile
        label="Enabled"
        :value="formatCount(enabledCount)"
        :hint="translate('clients.enabledHint', appStore.locale)"
        tone="success"
      />
      <StatusTile
        label="Online"
        :value="onlineCount === null ? '—' : formatCount(onlineCount)"
        :hint="translate('clients.onlineHint', appStore.locale)"
        tone="info"
      />
      <StatusTile
        label="Inbounds"
        :value="formatCount(inbounds.length)"
        :hint="translate('clients.inboundsHint', appStore.locale)"
        tone="neutral"
      />
    </div>

    <ACard class="work-panel clients-panel" :bordered="false">
      <div class="section-toolbar clients-toolbar">
        <h2>{{ translate('clients.detailsTitle', appStore.locale) }}</h2>
        <span class="clients-count" role="status">{{ resultSummary }}</span>
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
            <span>{{ translate('clients.allInbounds', appStore.locale) }}</span>
          </ASelectOption>
          <ASelectOption v-for="inbound in inbounds" :key="inbound.id" :value="inbound.id">
            {{ inboundLabel(inbound) }}
          </ASelectOption>
        </ASelect>
        <AButton v-if="hasFilters" class="client-clear-filters" @click="clearFilters">
          {{ translate('clients.clearFilters', appStore.locale) }}
        </AButton>
      </div>

      <ATable
        class="client-table"
        :columns="columns"
        :data-source="filteredRows"
        :locale="tableLocale"
        :loading="loading"
        :pagination="pagination"
        row-key="tableKey"
        size="middle"
        table-layout="fixed"
        @change="changePage"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'client'">
            <strong class="client-name">{{
              record.email || translate('clients.unnamed', appStore.locale)
            }}</strong>
            <ATag class="client-protocol">{{ record.protocol }}</ATag>
          </template>
          <template v-else-if="column.key === 'inbound'">
            <span class="client-mobile-label">{{ column.title }}</span>
            <span>{{ record.inboundLabel }}</span>
          </template>
          <template v-else-if="column.key === 'traffic'">
            <span class="client-mobile-label">{{ column.title }}</span>
            <span>{{ record.trafficLabel }}</span>
          </template>
          <template v-else-if="column.key === 'status'">
            <span class="client-mobile-label">{{ column.title }}</span>
            <div class="client-states">
              <ATag :color="record.enabled ? 'green' : 'red'">
                <span>{{ record.enabled ? 'Enabled' : 'Disabled' }}</span>
              </ATag>
              <span class="client-online">{{
                record.online === null
                  ? translate('clients.unknown', appStore.locale)
                  : translateDomText(record.online ? 'Online' : 'Offline', appStore.locale)
              }}</span>
            </div>
          </template>
          <template v-else-if="column.key === 'actions'">
            <div class="client-actions">
              <AButton aria-haspopup="dialog" @click="openClientDetails(record.key, $event)">
                {{ translate('clients.detailsAction', appStore.locale) }}
              </AButton>
              <AButton
                :aria-label="clientActionAriaLabel(asClientRow(record))"
                class="client-action-button"
                :danger="record.clientEnabled"
                :disabled="clientActionDisabled(asClientRow(record))"
                :loading="clientActionKey === record.key"
                @click="toggleClient(asClientRow(record))"
              >
                <template #icon>
                  <StopOutlined v-if="record.clientEnabled" aria-hidden="true" />
                  <PlayCircleOutlined v-else aria-hidden="true" />
                </template>
                {{
                  translate(record.clientEnabled ? 'action.stop' : 'action.start', appStore.locale)
                }}
              </AButton>
            </div>
          </template>
        </template>
      </ATable>
    </ACard>

    <AModal
      :open="Boolean(selectedClient)"
      :title="translate('clients.detailsTitle', appStore.locale)"
      :footer="null"
      :focus-trigger-after-close="false"
      :after-close="restoreDetailsFocus"
      width="min(32rem, calc(100vw - 24px))"
      @cancel="selectedClientKey = ''"
    >
      <template v-if="selectedClient">
        <h3 ref="detailHeading" class="client-detail-heading" tabindex="-1">
          {{ selectedClient.email || translate('clients.unnamed', appStore.locale) }}
        </h3>
        <dl class="client-detail-fields">
          <dt>Inbound</dt>
          <dd>{{ selectedClient.inboundLabel }}</dd>
          <dt>Address</dt>
          <dd>{{ selectedClient.inboundMeta }}</dd>
          <dt>Traffic</dt>
          <dd>{{ selectedClient.trafficLabel }}</dd>
          <dt>Expiry</dt>
          <dd>{{ selectedClient.expiryLabel }}</dd>
          <dt>Last Online</dt>
          <dd>{{ selectedClient.lastOnlineLabel }}</dd>
          <dt>{{ translate('common.credentials', appStore.locale) }}</dt>
          <dd>
            <code>{{ maskSecret(selectedClient.credential) || '—' }}</code>
          </dd>
          <dt>Sub ID</dt>
          <dd>
            <code>{{ maskSecret(selectedClient.subId) || '—' }}</code>
          </dd>
        </dl>
      </template>
    </AModal>
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
  Modal as AModal,
  Select as ASelect,
  SelectOption as ASelectOption,
  Table as ATable,
  Tag as ATag,
} from 'ant-design-vue';
import type { TableColumnsType, TablePaginationConfig } from 'ant-design-vue';
import zhPagination from 'ant-design-vue/es/vc-pagination/locale/zh_CN';
import { computed, onMounted, ref, watch } from 'vue';

import {
  getClientsLastOnline,
  getOnlineClients,
  listInbounds,
  updateInbound,
  updateInboundClient,
} from '@/api/inbounds';
import PageHeader from '@/components/PageHeader.vue';
import StatusTile from '@/components/StatusTile.vue';
import { translate, translateDomText } from '@/i18n/messages';
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
  tableKey: string;
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
  online: boolean | null;
  lastOnlineLabel: string;
}

const appStore = useAppStore();
const inbounds = ref<Inbound[]>([]);
const onlineClients = ref<Set<string> | null>(null);
const lastOnlineMap = ref<Record<string, number>>({});
const loading = ref(false);
const clientListCurrent = ref(false);
const error = ref('');
const search = ref('');
const inboundFilter = ref<number | 'all'>('all');
const clientActionKey = ref('');

const currentPage = ref(1);
const pageSize = ref(20);
const pagination = computed<TablePaginationConfig>(() => ({
  current: currentPage.value,
  pageSize: pageSize.value,
  showSizeChanger: true,
  showLessItems: true,
  locale: appStore.locale === 'zh-CN' ? zhPagination : undefined,
}));
const selectedClientKey = ref('');
const detailHeading = ref<HTMLElement>();
let detailTrigger: HTMLElement | null = null;
const columns = computed<TableColumnsType<AggregatedClientRow>>(() => [
  { title: translateDomText('Client', appStore.locale), key: 'client', className: 'identity-cell' },
  { title: translateDomText('Inbound', appStore.locale), key: 'inbound' },
  { title: translate('common.traffic', appStore.locale), key: 'traffic' },
  { title: translate('action.status', appStore.locale), key: 'status', width: 112 },
  {
    title: translate('common.actions', appStore.locale),
    key: 'actions',
    className: 'actions-cell',
    width: 184,
  },
]);

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
        // Ant renders its row key as a DOM attribute. Keep credentials out of that key.
        tableKey: `${inbound.id}:${index}`,
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
        online: onlineClients.value?.has(client.email) ?? null,
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
const selectedClient = computed(() =>
  rows.value.find((row) => row.key === selectedClientKey.value),
);
watch([search, inboundFilter], () => {
  currentPage.value = 1;
});
watch(
  () => filteredRows.value.length,
  (count) => {
    currentPage.value = Math.min(currentPage.value, Math.max(1, Math.ceil(count / pageSize.value)));
  },
);

function changePage(nextPage: TablePaginationConfig) {
  pageSize.value = nextPage.pageSize ?? pageSize.value;
  currentPage.value = nextPage.current ?? 1;
}
const onlineCount = computed(() =>
  onlineClients.value === null ? null : rows.value.filter((row) => row.online).length,
);
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
  const quotaLabel =
    total > 0 ? formatBytes(total) : translate('clients.unlimited', appStore.locale);
  return `${usedLabel} / ${quotaLabel}`;
}

function maskSecret(value: string): string {
  return value.trim() ? '••••••••' : '';
}

function formatTimestamp(value: number): string {
  if (!value) {
    return '-';
  }
  if (value < 0) {
    return new Intl.NumberFormat(appStore.locale, {
      style: 'unit',
      unit: 'day',
      unitDisplay: 'long',
    }).format(Math.abs(Math.round(value / 86_400_000)));
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString(appStore.locale);
}

function asClientRow(record: AggregatedClientRow | Record<string, unknown>): AggregatedClientRow {
  return record as AggregatedClientRow;
}

function clearFilters() {
  search.value = '';
  inboundFilter.value = 'all';
}

function openClientDetails(key: string, event: MouseEvent) {
  detailTrigger = event.currentTarget instanceof HTMLElement ? event.currentTarget : null;
  selectedClientKey.value = key;
}

// The dialog can receive keyboard input as soon as it mounts, including during its opening transition.
watch(detailHeading, (heading) => heading?.focus({ preventScroll: true }), { flush: 'post' });

function restoreDetailsFocus() {
  detailTrigger?.focus({ preventScroll: true });
  detailTrigger = null;
}

function clientActionAriaLabel(row: AggregatedClientRow): string {
  const action = translate(row.clientEnabled ? 'action.stop' : 'action.start', appStore.locale);
  const client = translate('common.clients', appStore.locale);
  return appStore.locale === 'zh-CN'
    ? `${action}${client} ${row.email || row.inboundLabel}`
    : `${action} client ${row.email || row.inboundLabel}`;
}

function clientActionDisabled(row: AggregatedClientRow): boolean {
  return (
    loading.value ||
    !clientListCurrent.value ||
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
  if (loading.value || !hasInjectedRuntimeConfig()) {
    return;
  }
  loading.value = true;
  error.value = '';
  try {
    const [inboundList, online, lastOnline] = await Promise.allSettled([
      listInbounds({ notifyOnError: false }),
      getOnlineClients({ notifyOnError: false }),
      getClientsLastOnline({ notifyOnError: false }),
    ]);
    clientListCurrent.value = inboundList.status === 'fulfilled';
    if (inboundList.status === 'fulfilled') inbounds.value = inboundList.value;
    onlineClients.value =
      online.status === 'fulfilled' && Array.isArray(online.value) ? new Set(online.value) : null;
    if (lastOnline.status === 'fulfilled') lastOnlineMap.value = lastOnline.value || {};
    const failure = [inboundList, online, lastOnline].find(
      (result) => result.status === 'rejected',
    );
    if (failure) throw failure.reason;
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
.clients-page,
.clients-panel {
  min-width: 0;
}

.clients-panel {
  container: clients-panel / inline-size;
}

.clients-stats {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.clients-refresh-button,
.client-clear-filters,
.client-filter-input {
  min-height: 44px;
}

.clients-toolbar {
  align-items: baseline;
  justify-content: flex-start;
  flex-flow: row wrap;
  gap: 8px 14px;
}

.clients-count,
.client-online,
.client-detail-fields dt,
.client-mobile-label {
  color: var(--brand-muted-strong);
}

.client-filter-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 18px;
}

.client-filter-input {
  flex: 2 1 16rem;
  min-width: 0;
}

.client-filter-select {
  flex: 1 1 12rem;
  min-width: 0;
}

.client-filter-select :deep(.ant-select-selector) {
  min-height: 44px;
  align-items: center;
}

.client-name,
.client-detail-heading,
.client-detail-fields dd,
.client-table :deep(td) {
  overflow-wrap: anywhere;
}

.client-name,
.client-online {
  display: block;
}

.client-protocol {
  margin: 4px 0 0;
}

.client-states .ant-tag {
  margin-inline-end: 0;
}

.client-online,
.client-mobile-label {
  font-size: 13px;
}

.client-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.client-actions .ant-btn {
  min-height: 44px;
  padding-inline: 10px;
}

.client-mobile-label {
  display: none;
}

.client-detail-heading {
  margin: 12px 0 18px;
  color: var(--brand-ink);
  font-size: 18px;
}

.client-detail-fields {
  display: grid;
  grid-template-columns: minmax(5rem, 0.9fr) minmax(0, 2fr);
  column-gap: 16px;
  margin: 0;
}

.client-detail-fields dt,
.client-detail-fields dd {
  margin: 0;
  padding: 10px 0;
  border-top: 1px solid var(--brand-border-soft);
}

.client-table :deep(.ant-pagination) {
  flex-wrap: wrap;
  row-gap: 8px;
}

@media (max-width: 760px) {
  .clients-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
}

/* Use the available panel width, which also changes when the sidebar opens. */
@container clients-panel (max-width: 760px) {
  .client-table :deep(.ant-table) {
    min-width: 0;
    background: transparent;
  }

  .client-table :deep(table),
  .client-table :deep(tbody) {
    display: block;
    width: 100%;
  }

  .client-table :deep(thead) {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }

  .client-table :deep(colgroup) {
    display: none;
  }

  .client-table :deep(.ant-pagination-options) {
    display: inline-block;
  }

  .client-table :deep(.ant-table-row) {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 8px;
    margin-bottom: 12px;
    padding: 14px;
    border: 1px solid var(--brand-border-soft);
    border-radius: 12px;
  }

  .client-table :deep(.ant-table-row > td) {
    display: grid;
    grid-template-columns: minmax(4rem, 0.4fr) minmax(0, 1fr);
    align-items: baseline;
    gap: 8px;
    padding: 0 !important;
    border: 0;
  }

  .client-table :deep(.ant-table-row > .identity-cell),
  .client-table :deep(.ant-table-row > .actions-cell) {
    display: block;
  }

  .client-mobile-label {
    display: inline;
  }

  .client-states {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 4px 8px;
  }

  .client-actions {
    padding-top: 8px;
    border-top: 1px solid var(--brand-border-soft);
  }

  .client-actions .ant-btn {
    flex: 1 1 5rem;
  }
}
</style>
