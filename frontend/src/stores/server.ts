import { defineStore } from 'pinia';

import { getApiErrorMessage } from '@/api/request';
import { getServerStatus } from '@/api/server';
import { openPanelWebSocket, type PanelWebSocketMessage } from '@/api/websocket';
import type { ServerStatus } from '@/types/server';
import { hasInjectedRuntimeConfig } from '@/types/runtime';

export const useServerStore = defineStore('server', {
  state: () => ({
    loadingStatus: false,
    status: null as ServerStatus | null,
    statusError: '',
    websocketConnected: false,
    websocketStarted: false,
    websocket: null as WebSocket | null,
  }),
  actions: {
    async refreshStatus() {
      if (!hasInjectedRuntimeConfig()) {
        return;
      }

      this.loadingStatus = true;
      this.statusError = '';
      try {
        this.status = await getServerStatus({ notifyOnError: false });
      } catch (error) {
        this.statusError = getApiErrorMessage(error);
      } finally {
        this.loadingStatus = false;
      }
    },
    connectRealtime() {
      if (this.websocketStarted || !hasInjectedRuntimeConfig()) {
        return;
      }

      this.websocketStarted = true;
      this.websocket = openPanelWebSocket(
        (message) => this.handleRealtimeMessage(message),
        () => {
          this.websocketConnected = false;
          this.websocketStarted = false;
          this.websocket = null;
        },
      );
      this.websocketConnected = Boolean(this.websocket);
    },
    handleRealtimeMessage(message: PanelWebSocketMessage) {
      if (message.type === 'status') {
        this.status = message.payload as ServerStatus;
        return;
      }

      if (message.type === 'xray_state' && this.status) {
        const payload = message.payload as Partial<ServerStatus['xray']>;
        this.status.xray = {
          ...this.status.xray,
          state: payload.state || this.status.xray.state,
          errorMsg: payload.errorMsg || '',
        };
        return;
      }

      if (message.type === 'invalidate') {
        void this.refreshStatus();
      }
    },
  },
});
