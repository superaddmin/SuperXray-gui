<template>
  <section class="page-stack core-page">
    <PageHeader
      eyebrow="Runtime"
      title="Core Instances"
      description="Default Xray and experimental core adapters under the current migration gate."
    >
      <ASpace wrap>
        <AButton :loading="coreStore.loading" @click="loadInstances">
          <template #icon><ReloadOutlined /></template>
          Refresh
        </AButton>
      </ASpace>
    </PageHeader>

    <AAlert v-if="coreStore.error" banner type="warning" :message="coreStore.error" />

    <div class="status-grid">
      <StatusTile
        label="Instances"
        :value="String(coreStore.instanceCount)"
        hint="Registered cores"
        tone="info"
      />
      <StatusTile
        label="Running"
        :value="String(coreStore.runningCount)"
        hint="Runtime state"
        tone="success"
      />
      <StatusTile
        label="Experimental"
        :value="String(coreStore.experimentalCount)"
        hint="Experimental adapters"
        tone="warning"
      />
    </div>

    <ACard class="work-panel" :bordered="false">
      <ATable
        :columns="columns"
        :data-source="coreStore.instances"
        :loading="coreStore.loading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'core'">
            <ASpace direction="vertical" :size="2">
              <ASpace wrap>
                <strong>{{ record.displayName || record.name || record.id }}</strong>
                <ATag :color="coreTypeColor(record.coreType)">{{ record.coreType }}</ATag>
                <ATag v-if="record.experimentalOnly" color="orange">Experimental</ATag>
                <ATag v-if="!record.writeSupported">Read only</ATag>
              </ASpace>
              <span class="muted-text">{{ record.id }}</span>
            </ASpace>
          </template>

          <template v-else-if="column.key === 'status'">
            <ASpace direction="vertical" :size="2">
              <ATag :color="stateColor(record.status.state)">{{ record.status.state }}</ATag>
              <span v-if="record.status.version" class="muted-text">{{
                record.status.version
              }}</span>
              <span v-if="record.status.errorMsg" class="muted-text">{{
                record.status.errorMsg
              }}</span>
            </ASpace>
          </template>

          <template v-else-if="column.key === 'source'">
            <ASpace direction="vertical" :size="2">
              <span>{{ record.mode || '-' }}</span>
              <span class="muted-text">{{ record.source || '-' }}</span>
              <span class="muted-text">{{ record.lifecycleOwner || '-' }}</span>
            </ASpace>
          </template>

          <template v-else-if="column.key === 'capabilities'">
            <ASpace wrap>
              <ATag :color="record.capabilities.read ? 'green' : 'default'">Read</ATag>
              <ATag :color="record.capabilities.write ? 'green' : 'default'">Write</ATag>
              <ATag :color="record.capabilities.validate ? 'green' : 'default'">Validate</ATag>
              <ATag :color="record.capabilities.start ? 'green' : 'default'">Start</ATag>
              <ATag :color="record.capabilities.stop ? 'green' : 'default'">Stop</ATag>
              <ATag :color="record.capabilities.restart ? 'green' : 'default'">Restart</ATag>
            </ASpace>
          </template>

          <template v-else-if="column.key === 'actions'">
            <ASpace wrap>
              <AButton size="small" @click="refreshCoreStatus(record.id)">Status</AButton>
              <AButton
                size="small"
                :disabled="!record.capabilities.validate"
                :loading="coreStore.isActionLoading(record.id, 'validate')"
                @click="runAction(record.id, 'validate')"
              >
                Validate
              </AButton>
              <AButton
                size="small"
                :disabled="!record.capabilities.start"
                :loading="coreStore.isActionLoading(record.id, 'start')"
                @click="runAction(record.id, 'start')"
              >
                Start
              </AButton>
              <AButton
                size="small"
                :disabled="!record.capabilities.stop"
                :loading="coreStore.isActionLoading(record.id, 'stop')"
                @click="runAction(record.id, 'stop')"
              >
                Stop
              </AButton>
              <AButton
                size="small"
                :disabled="!record.capabilities.restart"
                :loading="coreStore.isActionLoading(record.id, 'restart')"
                @click="runAction(record.id, 'restart')"
              >
                Restart
              </AButton>
            </ASpace>
          </template>
        </template>
      </ATable>
    </ACard>

    <ACard v-if="selectedInstance" class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Selected Core</p>
          <h2>{{ selectedInstance.displayName || selectedInstance.id }}</h2>
        </div>
        <ATag :color="stateColor(selectedInstance.status.state)">
          {{ selectedInstance.status.state }}
        </ATag>
      </div>
      <AAlert
        v-if="!selectedInstance.capabilities.lifecycleViaCoreManager"
        class="mb-12"
        type="info"
        message="This core is not fully controlled by Core Manager. Legacy runtime pages may still own lifecycle operations."
      />
      <pre class="code-preview compact-preview">{{ selectedInstanceJson }}</pre>
    </ACard>
  </section>
</template>

<script setup lang="ts">
import { ReloadOutlined } from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Space as ASpace,
  Table as ATable,
  Tag as ATag,
  message,
} from 'ant-design-vue';
import type { TableColumnsType } from 'ant-design-vue';
import { computed, onMounted } from 'vue';

import PageHeader from '@/components/PageHeader.vue';
import StatusTile from '@/components/StatusTile.vue';
import { useCoreStore } from '@/stores/core';
import type { CoreInstance, CoreState, CoreType } from '@/types/core';

const coreStore = useCoreStore();

const columns: TableColumnsType<CoreInstance> = [
  { title: 'Core', key: 'core', dataIndex: 'displayName' },
  { title: 'Status', key: 'status', dataIndex: 'status' },
  { title: 'Mode / Source', key: 'source', dataIndex: 'source' },
  { title: 'Capabilities', key: 'capabilities', dataIndex: 'capabilities' },
  { title: 'Actions', key: 'actions', fixed: 'right', width: 360 },
];

const selectedInstance = computed(() => coreStore.selectedInstance || coreStore.instances[0]);
const selectedInstanceJson = computed(() => JSON.stringify(selectedInstance.value, null, 2));

/** 根据内核类型返回稳定的标签颜色。 */
function coreTypeColor(coreType: CoreType) {
  if (coreType === 'xray') {
    return 'blue';
  }
  if (coreType === 'sing-box') {
    return 'purple';
  }
  return 'default';
}

/** 根据内核运行状态返回稳定的标签颜色。 */
function stateColor(state: CoreState) {
  if (state === 'running') {
    return 'green';
  }
  if (state === 'stopped') {
    return 'default';
  }
  if (state === 'error') {
    return 'red';
  }
  if (state === 'not-installed' || state === 'not-configured') {
    return 'orange';
  }
  return 'blue';
}

/** 刷新内核实例列表。 */
async function loadInstances() {
  await coreStore.refreshInstances();
}

/** 刷新单个内核的运行状态。 */
async function refreshCoreStatus(id: string) {
  try {
    await coreStore.refreshStatus(id);
    void message.success('Core status refreshed');
  } catch {
    void message.error(coreStore.error || 'Failed to refresh core status');
  }
}

/** 执行指定内核生命周期动作。 */
async function runAction(id: string, action: 'validate' | 'start' | 'stop' | 'restart') {
  try {
    const result = await coreStore.runLifecycleAction(id, action);
    void message.success(result.msg || `${action} completed`);
  } catch {
    void message.error(coreStore.error || `${action} failed`);
  }
}

onMounted(() => {
  void loadInstances();
});
</script>
