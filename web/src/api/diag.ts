// Diagnostic IQ-stream client. Mirrors WS /api/v1/diag/iq.
// Same connect/reconnect scaffolding pattern openSpectrumStream
// (api/spectrum.ts) uses.

import { type ClientConfig } from "./client";

export interface IQPoint {
  i: number;
  q: number;
}

export interface IQFrame {
  ts_ns: number;
  sample_rate: number;
  center_hz: number;
  points: IQPoint[];
  energy_dbfs: number;
}

export type FrameHandler = (f: IQFrame) => void;
export type StatusHandler = (s: "connecting" | "open" | "closed") => void;

export interface IQStream {
  close(): void;
}

export interface IQOptions {
  serial: string;
  rate?: number; // target sample rate (sps), 100..20000, default 2000
  // Frequency offset in Hz, relative to the SDR centre, mixed down to
  // baseband before decimation. Lets the operator pull an off-centre
  // locked channel out from under the centre DC spike. Default 0.
  offset?: number;
  onFrame: FrameHandler;
  onStatus?: StatusHandler;
}

const INITIAL_BACKOFF = 500;
const MAX_BACKOFF = 30_000;

export function diagWebSocketURL(cfg: ClientConfig, opts: IQOptions): string {
  const params = new URLSearchParams({ device: opts.serial });
  if (opts.rate != null) params.set("rate", String(opts.rate));
  if (opts.offset != null && opts.offset !== 0)
    params.set("offset", String(Math.round(opts.offset)));
  const u = new URL(
    `/api/v1/diag/iq?${params.toString()}`,
    cfg.baseURL || window.location.href,
  );
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  return u.toString();
}

export function openIQStream(cfg: ClientConfig, opts: IQOptions): IQStream {
  let closed = false;
  let ws: WebSocket | null = null;
  let backoff = INITIAL_BACKOFF;
  let reconnectTimer: number | undefined;

  const setStatus = (s: "connecting" | "open" | "closed") => {
    if (!closed) opts.onStatus?.(s);
  };

  const jittered = (base: number) => base / 2 + Math.random() * (base / 2);

  const connect = () => {
    if (closed) return;
    setStatus("connecting");
    const url = diagWebSocketURL(cfg, opts);
    ws = new WebSocket(url);

    ws.onopen = () => {
      backoff = INITIAL_BACKOFF;
      setStatus("open");
    };

    ws.onmessage = (ev) => {
      if (closed) return;
      try {
        const frame = JSON.parse(ev.data) as IQFrame;
        if (frame && Array.isArray(frame.points)) {
          opts.onFrame(frame);
        }
      } catch {
        // Drop malformed.
      }
    };

    const onDown = () => {
      if (closed) return;
      ws = null;
      setStatus("closed");
      const wait = jittered(backoff);
      backoff = Math.min(backoff * 2, MAX_BACKOFF);
      reconnectTimer = window.setTimeout(connect, wait);
    };
    ws.onerror = onDown;
    ws.onclose = onDown;
  };

  connect();

  return {
    close() {
      closed = true;
      setStatus("closed");
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      if (ws) {
        try {
          ws.close();
        } catch {
          // ignore
        }
      }
    },
  };
}
