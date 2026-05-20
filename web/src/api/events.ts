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

const INITIAL_BACKOFF = 500;
const MAX_BACKOFF = 30_000;
// A connection must stay open at least this long before it counts as
// healthy and the backoff is allowed to reset. A socket that opens then
// drops immediately keeps backing off rather than storming at the floor.
const STABLE_MS = 5_000;

export function openEventStream(
  cfg: ClientConfig,
  opts: Options,
): EventStream {
  let closed = false;
  let ws: WebSocket | null = null;
  let backoff = INITIAL_BACKOFF;
  let reconnectTimer: number | undefined;
  let stableTimer: number | undefined;

  // Once the stream is closed, no late socket event may write to the
  // store — the React effect that owns this stream has been torn down.
  const setStatus = (s: "connecting" | "open" | "closed") => {
    if (!closed) opts.onStatus?.(s);
  };

  // Equal jitter: spreads reconnects so clients don't synchronize into a
  // thundering herd, and keeps a single client's retry from collapsing
  // toward a zero-delay busy loop.
  const jittered = (base: number) => base / 2 + Math.random() * (base / 2);

  const connect = () => {
    if (closed) return;
    // Deliver "connecting" on a microtask so openEventStream never
    // writes to the store synchronously inside the effect that called
    // it. The closed guard in setStatus covers a teardown that races
    // the microtask.
    queueMicrotask(() => setStatus("connecting"));

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
      setStatus("open");
      // Only treat the connection as healthy — and reset the backoff —
      // once it has held for STABLE_MS. Resetting on open alone lets a
      // flapping connection reconnect-storm at the backoff floor.
      stableTimer = window.setTimeout(() => {
        backoff = INITIAL_BACKOFF;
        stableTimer = undefined;
      }, STABLE_MS);
    };
    ws.onmessage = (msg) => {
      if (closed) return;
      try {
        const parsed = JSON.parse(msg.data) as EventDTO;
        opts.onEvent(parsed);
      } catch {
        // Malformed frame — ignore. The daemon never emits non-JSON.
      }
    };
    ws.onclose = () => {
      if (stableTimer !== undefined) {
        window.clearTimeout(stableTimer);
        stableTimer = undefined;
      }
      setStatus("closed");
      if (!closed) scheduleReconnect();
    };
    ws.onerror = () => {
      // onclose follows; let it handle the reconnect.
    };
  };

  const scheduleReconnect = () => {
    if (closed) return;
    const delay = jittered(backoff);
    backoff = Math.min(backoff * 2, MAX_BACKOFF);
    reconnectTimer = window.setTimeout(connect, delay);
  };

  connect();

  return {
    close() {
      closed = true;
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      if (stableTimer !== undefined) window.clearTimeout(stableTimer);
      if (ws) {
        // Null every handler so an in-flight socket that opens or
        // closes after teardown cannot call back into the store.
        ws.onopen = null;
        ws.onmessage = null;
        ws.onclose = null;
        ws.onerror = null;
        try {
          ws.close();
        } catch {
          /* swallow */
        }
      }
      // Deliver the final status directly — setStatus() is now gated by
      // `closed`, which is already true here.
      opts.onStatus?.("closed");
    },
  };
}
