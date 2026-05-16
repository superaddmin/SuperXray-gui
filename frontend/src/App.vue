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
    borderRadius: 8,
    colorBgBase: '#0a1628',
    colorBgContainer: '#111c33',
    colorBgElevated: '#14213a',
    colorBgLayout: '#0a1628',
    colorBorder: 'rgba(255, 255, 255, 0.1)',
    colorBorderSecondary: 'rgba(255, 255, 255, 0.08)',
    colorError: '#ef4444',
    colorInfo: '#0066ff',
    colorLink: '#0066ff',
    colorPrimary: '#0066ff',
    colorPrimaryHover: '#2f80ff',
    colorSuccess: '#36b37e',
    colorText: '#ffffff',
    colorTextDescription: '#a3b3c9',
    colorTextSecondary: '#c7d2e6',
    colorTextTertiary: '#8293b1',
    colorWarning: '#ffab00',
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
