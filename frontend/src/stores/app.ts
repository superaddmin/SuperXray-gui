import { defineStore } from 'pinia';

import {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  getNextLocale,
  normalizeLocale,
  type AppLocale,
} from '@/i18n/messages';
import { getRuntimeConfig } from '@/types/runtime';

function getInitialLocale(): AppLocale {
  if (typeof window === 'undefined') {
    return DEFAULT_LOCALE;
  }
  return normalizeLocale(window.localStorage.getItem(LOCALE_STORAGE_KEY));
}

export const useAppStore = defineStore('app', {
  state: () => ({
    collapsed: false,
    locale: getInitialLocale(),
    runtimeConfig: getRuntimeConfig(),
  }),
  actions: {
    setLocale(locale: AppLocale) {
      this.locale = normalizeLocale(locale);
      if (typeof window !== 'undefined') {
        window.localStorage.setItem(LOCALE_STORAGE_KEY, this.locale);
      }
    },
    toggleLocale() {
      this.setLocale(getNextLocale(this.locale));
    },
    toggleCollapsed() {
      this.collapsed = !this.collapsed;
    },
  },
});
