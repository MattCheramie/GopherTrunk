// Channel-baseband spectrum stream client. Mirrors WS /api/v1/diag/mixer
// (see internal/api/mixer.go). Same connect/reconnect scaffolding the
// symbol stream uses; the wire shape is one JSON MixerFrame per message.

import { type ClientConfig } from "./client";

export interface MixerFrame {
  ts_ns: number;
  center_hz: number;
  sample_rate_hz: number;
  offset_hz: number;
  // The receiver's residual carrier-offset estimate (Hz). The tuned view
  // is the raw window re-mixed by this, so a locked loop centres it.
  carrier_offset_hz: number;
  // FFT-shifted dBFS bins of the channelized baseband. raw_bins shows the
  // carrier at its residual offset; tuned_bins shows it after the
  // carrier-recovery correction. Both equal length; index 0 is the lowest
  // frequency, index n/2 is DC.
  raw_bins: number[];
  tuned_bins: number[];
}

export type MixerFrameHandler = (f: MixerFrame) => void;
export type StatusHandler = (s: "connecting" | "open" | "closed") => void;

export interface MixerStream {
  close(): void;
}

export interface MixerOptions {
  serial: string;
  // Receiver selector: "p25-c4fm" or "p25-cqpsk". Default "p25-c4fm".
  proto?: string;
  // Frequency offset in Hz, relative to the SDR centre, tuned down to
  // baseband before channelizing. Default 0.
  offset?: number;
  onFrame: MixerFrameHandler;
  onStatus?: StatusHandler;
}

const INITIAL_BACKOFF = 500;
const MAX_BACKOFF = 30_000;

export function mixerWebSocketURL(cfg: ClientConfig, opts: MixerOptions): string {
  const params = new URLSearchParams({ device: opts.serial });
  if (opts.proto) params.set("proto", opts.proto);
  if (opts.offset != null && opts.offset !== 0)
    params.set("offset", String(Math.round(opts.offset)));
  const u = new URL(
    `/api/v1/diag/mixer?${params.toString()}`,
    cfg.baseURL || window.location.href,
  );
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  return u.toString();
}

export function openMixerStream(cfg: ClientConfig, opts: MixerOptions): MixerStream {
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
    const url = mixerWebSocketURL(cfg, opts);
    ws = new WebSocket(url);

    ws.onopen = () => {
      backoff = INITIAL_BACKOFF;
      setStatus("open");
    };

    ws.onmessage = (ev) => {
      if (closed) return;
      try {
        const frame = JSON.parse(ev.data) as MixerFrame;
        if (frame && Array.isArray(frame.raw_bins)) {
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
