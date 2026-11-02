<template>
  <section class="page-stack">
    <PageHeader
      eyebrow="Overview"
      title="Clients"
      description="Aggregated client details across all inbounds, including traffic, expiry and online status."
    >
      <AButton
        :disabled="Boolean(clientActionKey)"
        :loading="loading"
        type="primary"
        @click="refresh"
      >
        <template #icon><ReloadOutlined /></template>
        Refresh
      </AButton>
    </PageHeader>

    <AAlert v-if="error" banner type="warning" :message="error" />

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
      <div class="section-toolbar">
        <div>
          <h2>Client Details</h2>
          <p>
            Aggregated view of VMess, VLESS, Trojan, Shadowsocks, Hysteria2 and WireGuard clients.
            SOCKS5/mixed accounts are excluded.
          </p>
        </div>
        <ASpace wrap>
          <AInput
            v-model:value="search"
            allow-clear
            placeholder="Search email, credential, subId..."
            style="width: 280px"
          />
          <ASelect v-model:value="inboundFilter" style="width: 220px">
            <ASelectOption value="all">All inbounds</ASelectOption>
            <ASelectOption v-for="inbound in inbounds" :key="inbound.id" :value="inbound.id">
              {{ inboundLabel(inbound) }}
            </ASelectOption>
          </ASelect>
        </ASpace>
      </div>

      <ATable
        :columns="columns"
        :data-source="filteredRows"
        :loading="loading"
        :pagination="{ pageSize: 20, showSizeChanger: true }"
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
              :aria-label="`${record.clientEnabled ? 'Stop' : 'Start'} client ${record.email || record.key}`"
              :danger="record.clientEnabled"
              :disabled="clientActionDisabled(asClientRow(record))"
              :loading="clientActionKey === record.key"
              size="small"
              type="link"
              @click="toggleClient(asClientRow(record))"
            >
              {{ record.clientEnabled ? 'Stop' : 'Start' }}
            </AButton>
          </template>
        </template>
      </ATable>
    </ACard>
  </section>
</template>

<script setup lang="ts">
import { ReloadOutlined } from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Input as AInput,
  Select as ASelect,
  SelectOption as ASelectOption,
  Space as ASpace,
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
import { protocolSupportsClients } from '@/schemas/protocolRegistry';
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
.client-credential {
  font-size: 12px;
  word-break: break-all;
}
</style>
