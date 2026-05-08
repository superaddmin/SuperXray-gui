<template>
  <AConfigProvider :csp="cspConfig" :theme="antdTheme">
    <RouterView />
  </AConfigProvider>
</template>

<script setup lang="ts">
import { ConfigProvider as AConfigProvider, theme } from 'ant-design-vue';
import type { ThemeConfig } from 'ant-design-vue/es/config-provider/context';
import { computed, nextTick, onBeforeUnmount, onMounted, watch } from 'vue';
import { RouterView, useRoute } from 'vue-router';

import { applyLocaleToDocument, createLocaleDomObserver } from './i18n/domTranslator';
import { formatDocumentTitle } from './i18n/messages';
import { useAppStore } from './stores/app';

const appStore = useAppStore();
const route = useRoute();
let localeObserver: MutationObserver | undefined;

const antdTheme = computed<ThemeConfig>(() => ({
  algorithm: theme.darkAlgorithm,
  token: {
    borderRadius: 16,
    colorBgBase: '#0a0e27',
    colorBgContainer: '#0f1635',
    colorBgElevated: '#10183a',
    colorBgLayout: '#0a0e27',
    colorBorder: 'rgba(30, 38, 80, 0.72)',
    colorBorderSecondary: 'rgba(30, 38, 80, 0.48)',
    colorError: '#ef4444',
    colorInfo: '#0080ff',
    colorLink: '#0080ff',
    colorPrimary: '#39ff14',
    colorPrimaryHover: '#7dff6a',
    colorSuccess: '#39ff14',
    colorText: '#ffffff',
    colorTextDescription: '#8b92b3',
    colorTextSecondary: '#aab2d5',
    colorTextTertiary: '#70789f',
    colorWarning: '#f7931a',
    fontFamily:
      '"DM Sans", Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  },
}));

const cspConfig = computed(() =>
  appStore.runtimeConfig.cspNonce ? { nonce: appStore.runtimeConfig.cspNonce } : undefined,
);

function syncLocale() {
  document.title = formatDocumentTitle(route.name, appStore.locale);
  void nextTick(() => applyLocaleToDocument(appStore.locale));
}

watch(() => [appStore.locale, route.fullPath, route.name], syncLocale, { immediate: true });

onMounted(() => {
  localeObserver = createLocaleDomObserver(() => appStore.locale);
  syncLocale();
});

onBeforeUnmount(() => {
  localeObserver?.disconnect();
});
</script>
