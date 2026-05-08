import { defineStore } from 'pinia';

import {
  getCoreStatus,
  listCoreInstances,
  restartCore,
  startCore,
  stopCore,
  validateCore,
} from '@/api/cores';
import { getApiErrorMessage } from '@/api/request';
import type { CoreInstance, CoreLifecycleResult, CoreStatus } from '@/types/core';
import { hasInjectedRuntimeConfig } from '@/types/runtime';

interface CoreStoreState {
  instances: CoreInstance[];
  loading: boolean;
  actionLoading: Record<string, boolean>;
  error: string;
  selectedId: string;
}

export const useCoreStore = defineStore('core', {
  state: (): CoreStoreState => ({
    instances: [],
    loading: false,
    actionLoading: {},
    error: '',
    selectedId: '',
  }),
  getters: {
    selectedInstance: (state) => state.instances.find((instance) => instance.id === state.selectedId),
    instanceCount: (state) => state.instances.length,
    runningCount: (state) =>
      state.instances.filter((instance) => instance.status.state === 'running').length,
    experimentalCount: (state) => state.instances.filter((instance) => instance.experimentalOnly).length,
  },
  actions: {
    /** 刷新当前面板可见的所有内核实例。 */
    async refreshInstances() {
      if (!hasInjectedRuntimeConfig()) {
        return;
      }

      this.loading = true;
      this.error = '';
      try {
        this.instances = await listCoreInstances({ notifyOnError: false });
        if (!this.selectedId && this.instances.length > 0) {
          this.selectedId = this.instances[0].id;
        }
      } catch (error) {
        this.error = getApiErrorMessage(error);
      } finally {
        this.loading = false;
      }
    },

    /** 刷新指定内核实例状态并合并到本地列表。 */
    async refreshStatus(id: string) {
      const status = await getCoreStatus(id, { notifyOnError: false });
      this.patchStatus(id, status);
      return status;
    },

    /** 执行内核生命周期动作并刷新对应实例状态。 */
    async runLifecycleAction(
      id: string,
      action: 'validate' | 'start' | 'stop' | 'restart',
    ): Promise<CoreLifecycleResult> {
      const actionKey = `${id}:${action}`;
      this.actionLoading[actionKey] = true;
      this.error = '';
      try {
        const handlers = {
          validate: validateCore,
          start: startCore,
          stop: stopCore,
          restart: restartCore,
        };
        const result = await handlers[action](id, { notifyOnError: false });
        this.patchStatus(id, {
          state: result.state,
          version: this.instances.find((instance) => instance.id === id)?.status.version || '',
          errorMsg: result.errorMsg,
          pid: result.pid,
        });
        void this.refreshStatus(id).catch(() => undefined);
        return result;
      } catch (error) {
        this.error = getApiErrorMessage(error);
        throw error;
      } finally {
        this.actionLoading[actionKey] = false;
      }
    },

    /** 更新指定内核实例的状态快照。 */
    patchStatus(id: string, status: CoreStatus) {
      this.instances = this.instances.map((instance) =>
        instance.id === id
          ? {
              ...instance,
              status: {
                ...instance.status,
                ...status,
              },
            }
          : instance,
      );
    },

    /** 判断指定内核实例的生命周期动作是否正在执行。 */
    isActionLoading(id: string, action: string) {
      return Boolean(this.actionLoading[`${id}:${action}`]);
    },
  },
});
