<template>
  <div class="status-bar">
    <ATag class="status-tag" :color="xrayStatusColor">{{ xrayStatusLabel }}</ATag>
    <ATag class="status-tag" color="blue">Phase 10</ATag>
    <span class="icon-button theme-indicator" aria-label="Dark theme" role="img">
      <span class="theme-crescent" aria-hidden="true" />
    </span>
  </div>
</template>

<script setup lang="ts">
import { Tag as ATag } from 'ant-design-vue';
import { computed, onMounted } from 'vue';

import { useServerStore } from '@/stores/server';

const serverStore = useServerStore();

const xrayStatusLabel = computed(() => {
  const status = serverStore.status?.xray;
  if (!status) {
    return 'default-xray';
  }
  return status.version ? `Xray ${status.version}` : `Xray ${status.state}`;
});

const xrayStatusColor = computed(() => {
  switch (serverStore.status?.xray.state) {
    case 'running':
      return 'green';
    case 'error':
      return 'red';
    case 'stop':
      return 'orange';
    default:
      return 'default';
  }
});

onMounted(() => {
  serverStore.connectRealtime();
  void serverStore.refreshStatus();
});
</script>
