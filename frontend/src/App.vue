<template>
  <AConfigProvider :csp="cspConfig" :theme="antdTheme">
    <RouterView />
  </AConfigProvider>
</template>

<script setup lang="ts">
import { ConfigProvider as AConfigProvider, theme } from 'ant-design-vue';
import type { ThemeConfig } from 'ant-design-vue/es/config-provider/context';
import { computed } from 'vue';
import { RouterView } from 'vue-router';

import { useAppStore } from './stores/app';

const appStore = useAppStore();

const antdTheme = computed<ThemeConfig>(() => ({
  algorithm: theme.darkAlgorithm,
  token: {
    borderRadius: 14,
    colorBgBase: '#080d1b',
    colorBgContainer: '#1f2433',
    colorBgElevated: '#232a3a',
    colorBgLayout: '#080d1b',
    colorBorder: 'rgba(148, 163, 184, 0.22)',
    colorBorderSecondary: 'rgba(148, 163, 184, 0.14)',
    colorError: '#ff7875',
    colorInfo: '#2f73f6',
    colorLink: '#3b82f6',
    colorPrimary: '#2f73f6',
    colorPrimaryHover: '#3b82f6',
    colorSuccess: '#22c55e',
    colorText: '#f8fafc',
    colorTextDescription: '#a6b0c3',
    colorTextSecondary: '#a6b0c3',
    colorTextTertiary: '#7d879b',
    colorWarning: '#f97316',
    fontFamily:
      'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  },
}));

const cspConfig = computed(() =>
  appStore.runtimeConfig.cspNonce ? { nonce: appStore.runtimeConfig.cspNonce } : undefined,
);
</script>
