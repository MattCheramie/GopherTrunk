// Live event stream over WebSocket. Mirrors the SSE-driven update
// pattern of internal/tui/cmds.go connectSSE. WebSocket is used
// rather than SSE because browsers cannot attach the Authorization
// header to an EventSource; the WS upgrade carries the same payload
// shape (one JSON EventDTO per frame).

import type { EventDTO } from "./types";
import { type ClientConfig, eventsWebSocketURL } from "./client";

export type EventHandler = (ev: EventDTO) => void;
export type StatusHandler = (status: "connecting" | "open" | "closed") => void;

export interface EventStream {
  close(): void;
}

interface Options {
  onEvent: EventHandler;
  onStatus?: StatusHandler;
}

export function openEventStream(
  cfg: ClientConfig,
  opts: Options,
): EventStream {
  let closed = false;
  let ws: WebSocket | null = null;
  let backoff = 500;
  const MAX_BACKOFF = 30_000;
  let reconnectTimer: number | undefined;

  const setStatus = (s: "connecting" | "open" | "closed") =>
    opts.onStatus?.(s);

  const connect = () => {
    if (closed) return;
    setStatus("connecting");

    try {
      let url = eventsWebSocketURL(cfg);
      if (cfg.token) {
        // The daemon's WS upgrade does not currently accept a token
        // via query parameter; if auth is required, deployments must
        // bind to a trusted network (auto mode) or front the daemon
        // with a reverse proxy that adds the header. The token is
        // still forwarded via the optional Sec-WebSocket-Protocol
        // sub-protocol form as a future extension point.
        url += url.includes("?") ? "&" : "?";
        url += `token=${encodeURIComponent(cfg.token)}`;
      }
      ws = new WebSocket(url);
    } catch {
      // A malformed base URL (eventsWebSocketURL throwing) or a
      // WebSocket constructor rejection both land here.
      scheduleReconnect();
      return;
    }

    ws.onopen = () => {
      backoff = 500;
      setStatus("open");
    };
    ws.onmessage = (msg) => {
      try {
        const parsed = JSON.parse(msg.data) as EventDTO;
        opts.onEvent(parsed);
      } catch {
        // Malformed frame — ignore. The daemon never emits non-JSON.
      }
    };
    ws.onclose = () => {
      setStatus("closed");
      if (!closed) scheduleReconnect();
    };
    ws.onerror = () => {
      // onclose follows; let it handle the reconnect.
    };
  };

  const scheduleReconnect = () => {
    if (closed) return;
    reconnectTimer = window.setTimeout(() => {
      backoff = Math.min(backoff * 2, MAX_BACKOFF);
      connect();
    }, backoff);
  };

  connect();

  return {
    close() {
      closed = true;
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      if (ws) {
        ws.onclose = null;
        ws.onerror = null;
        try {
          ws.close();
        } catch {
          /* swallow */
        }
      }
      setStatus("closed");
    },
  };
}
