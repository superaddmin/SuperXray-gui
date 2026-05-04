import { getRuntimeConfig, hasInjectedRuntimeConfig } from '@/types/runtime';

type PanelWebSocketType =
  | 'status'
  | 'traffic'
  | 'inbounds'
  | 'notification'
  | 'xray_state'
  | 'outbounds'
  | 'invalidate';

export interface PanelWebSocketMessage<T = unknown> {
  type: PanelWebSocketType;
  payload: T;
  time: number;
}

export function openPanelWebSocket(
  onMessage: (message: PanelWebSocketMessage) => void,
  onClose?: () => void,
): WebSocket | null {
  if (!hasInjectedRuntimeConfig()) {
    return null;
  }

  const url = new URL(`${getRuntimeConfig().basePath}ws`, window.location.origin);
  url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';

  const socket = new WebSocket(url);
  socket.addEventListener('message', (event) => {
    const parsed = parsePanelWebSocketMessage(event.data);
    if (parsed) {
      onMessage(parsed);
    }
  });
  if (onClose) {
    socket.addEventListener('close', onClose);
  }
  return socket;
}

function parsePanelWebSocketMessage(data: unknown): PanelWebSocketMessage | null {
  if (typeof data !== 'string') {
    return null;
  }

  try {
    const parsed = JSON.parse(data) as Partial<PanelWebSocketMessage>;
    if (typeof parsed.type !== 'string') {
      return null;
    }
    return {
      type: parsed.type as PanelWebSocketType,
      payload: parsed.payload,
      time: typeof parsed.time === 'number' ? parsed.time : Date.now(),
    };
  } catch {
    return null;
  }
}
