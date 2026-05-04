import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';

import MainLayout from '@/layouts/MainLayout.vue';
import { getRuntimeConfig } from '@/types/runtime';

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { title: 'Login' },
  },
  {
    path: '/',
    component: MainLayout,
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/views/DashboardView.vue'),
        meta: { title: 'Dashboard' },
      },
      {
        path: 'dashboard',
        redirect: { name: 'dashboard' },
      },
      {
        path: 'logs',
        name: 'logs',
        component: () => import('@/views/LogsView.vue'),
        meta: { title: 'Logs' },
      },
      {
        path: 'xray',
        name: 'xray',
        component: () => import('@/views/XrayView.vue'),
        meta: { title: 'Xray' },
      },
      {
        path: 'inbounds',
        name: 'inbounds',
        component: () => import('@/views/InboundsView.vue'),
        meta: { title: 'Inbounds' },
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/SettingsView.vue'),
        meta: { title: 'Settings' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: 'Not Found' },
  },
];

export const router = createRouter({
  history: createWebHistory(getRuntimeConfig().uiBasePath || import.meta.env.BASE_URL),
  routes,
});

router.beforeEach((to) => {
  document.title = `${String(to.meta.title || 'Panel')} - SuperXray`;
  return true;
});
