import { defineStore } from 'pinia';

import { getRuntimeConfig } from '@/types/runtime';

export const useAppStore = defineStore('app', {
  state: () => ({
    collapsed: false,
    runtimeConfig: getRuntimeConfig(),
  }),
  actions: {
    toggleCollapsed() {
      this.collapsed = !this.collapsed;
    },
  },
});
