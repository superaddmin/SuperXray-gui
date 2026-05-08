import { legacyEndpoints } from './endpoints';
import { getJson, postForm, type ApiRequestOptions } from './request';

import type { CoreInstance, CoreLifecycleResult, CoreStatus } from '@/types/core';

/** 获取所有已注册的内核实例。 */
export function listCoreInstances(options?: ApiRequestOptions): Promise<CoreInstance[]> {
  return getJson<CoreInstance[]>(legacyEndpoints.cores.instances, options);
}

/** 获取指定内核实例的完整元数据。 */
export function getCoreInstance(id: string, options?: ApiRequestOptions): Promise<CoreInstance> {
  return getJson<CoreInstance>(legacyEndpoints.cores.instance(id), options);
}

/** 获取指定内核实例的实时状态。 */
export function getCoreStatus(id: string, options?: ApiRequestOptions): Promise<CoreStatus> {
  return getJson<CoreStatus>(legacyEndpoints.cores.status(id), options);
}

/** 请求后端校验指定内核实例配置。 */
export function validateCore(
  id: string,
  options?: ApiRequestOptions,
): Promise<CoreLifecycleResult> {
  return postForm<CoreLifecycleResult>(legacyEndpoints.cores.validate(id), {}, options);
}

/** 请求后端启动指定内核实例。 */
export function startCore(id: string, options?: ApiRequestOptions): Promise<CoreLifecycleResult> {
  return postForm<CoreLifecycleResult>(legacyEndpoints.cores.start(id), {}, options);
}

/** 请求后端停止指定内核实例。 */
export function stopCore(id: string, options?: ApiRequestOptions): Promise<CoreLifecycleResult> {
  return postForm<CoreLifecycleResult>(legacyEndpoints.cores.stop(id), {}, options);
}

/** 请求后端重启指定内核实例。 */
export function restartCore(
  id: string,
  options?: ApiRequestOptions,
): Promise<CoreLifecycleResult> {
  return postForm<CoreLifecycleResult>(legacyEndpoints.cores.restart(id), {}, options);
}
