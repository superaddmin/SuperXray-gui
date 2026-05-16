<template>
  <div class="status-bar">
    <ATag class="status-tag status-tag-runtime" :color="xrayStatusColor">
      {{ xrayStatusLabel }}
    </ATag>
    <ATag class="status-tag status-tag-phase">{{ phaseLabel }}</ATag>
    <AButton
      class="language-toggle"
      size="small"
      :title="languageToggleAriaLabel"
      @click="appStore.toggleLocale"
    >
      <span>{{ languageButtonLabel }}</span>
      <span class="visually-hidden">: {{ languageToggleAriaLabel }}</span>
    </AButton>
    <span class="icon-button theme-indicator" :aria-label="themeLabel" role="img">
      <span class="theme-crescent" aria-hidden="true" />
    </span>
  </div>
</template>

<script setup lang="ts">
import { Button as AButton, Tag as ATag } from 'ant-design-vue';
import { computed, onMounted } from 'vue';

import { getLanguageButtonLabel, getLanguageToggleAriaLabel, translate } from '@/i18n/messages';
import { useAppStore } from '@/stores/app';
import { useServerStore } from '@/stores/server';

const appStore = useAppStore();
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

const languageButtonLabel = computed(() => getLanguageButtonLabel(appStore.locale));
const languageToggleAriaLabel = computed(() => getLanguageToggleAriaLabel(appStore.locale));
const phaseLabel = computed(() => translate('status.phase10', appStore.locale));
const themeLabel = computed(() => translate('status.darkTheme', appStore.locale));

onMounted(() => {
  serverStore.connectRealtime();
  void serverStore.refreshStatus();
});
</script>
