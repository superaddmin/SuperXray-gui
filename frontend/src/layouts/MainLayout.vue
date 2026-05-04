<template>
  <ALayout class="app-shell">
    <ALayoutSider
      v-model:collapsed="appStore.collapsed"
      breakpoint="lg"
      class="app-sider"
      collapsible
      :trigger="null"
      width="236"
    >
      <RouterLink aria-label="SuperXray dashboard" class="brand" to="/">
        <!-- eslint-disable vue/html-self-closing -->
        <img
          v-if="appStore.collapsed"
          class="brand-logo brand-logo-collapsed"
          :src="logoIconUrl"
          alt="SuperXray"
        />
        <img v-else class="brand-logo" :src="logoDarkUrl" alt="SuperXray" />
        <!-- eslint-enable vue/html-self-closing -->
      </RouterLink>

      <AMenu
        class="app-menu"
        mode="inline"
        :items="menuItems"
        :selected-keys="selectedKeys"
        @click="handleMenuClick"
      />
    </ALayoutSider>

    <ALayout>
      <ALayoutHeader class="app-header">
        <AButton
          type="text"
          class="icon-button"
          :aria-label="appStore.collapsed ? 'Expand navigation' : 'Collapse navigation'"
          @click="appStore.toggleCollapsed"
        >
          <MenuUnfoldOutlined v-if="appStore.collapsed" />
          <MenuFoldOutlined v-else />
        </AButton>
        <AppStatusBar />
      </ALayoutHeader>

      <ALayoutContent class="app-content">
        <RouterView />
      </ALayoutContent>
    </ALayout>
  </ALayout>
</template>

<script setup lang="ts">
import {
  ApiOutlined,
  DashboardOutlined,
  FileTextOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SettingOutlined,
  SwapOutlined,
} from '@ant-design/icons-vue';
import {
  Button as AButton,
  Layout as ALayout,
  LayoutContent as ALayoutContent,
  LayoutHeader as ALayoutHeader,
  LayoutSider as ALayoutSider,
  Menu as AMenu,
} from 'ant-design-vue';
import type { ItemType } from 'ant-design-vue';
import type { MenuInfo } from 'ant-design-vue/es/menu/src/interface';
import { computed, h } from 'vue';
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router';

import logoDarkUrl from '@/assets/logo-dark.svg';
import logoIconUrl from '@/assets/logo-icon.svg';
import AppStatusBar from '@/components/AppStatusBar.vue';
import { useAppStore } from '@/stores/app';

const appStore = useAppStore();
const route = useRoute();
const router = useRouter();

const menuItems: ItemType[] = [
  { key: 'dashboard', icon: () => h(DashboardOutlined), label: 'Dashboard' },
  { key: 'logs', icon: () => h(FileTextOutlined), label: 'Logs' },
  { key: 'xray', icon: () => h(ApiOutlined), label: 'Xray' },
  { key: 'inbounds', icon: () => h(SwapOutlined), label: 'Inbounds' },
  { key: 'settings', icon: () => h(SettingOutlined), label: 'Settings' },
];

const selectedKeys = computed(() => [String(route.name || 'dashboard')]);

function handleMenuClick({ key }: MenuInfo) {
  const routeKey = String(key);
  router.push(routeKey === 'dashboard' ? '/' : `/${routeKey}`);
}
</script>
