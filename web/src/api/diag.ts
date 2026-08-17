// Diagnostic IQ-stream client. Mirrors WS /api/v1/diag/iq.
// Same connect/reconnect scaffolding pattern openSpectrumStream
// (api/spectrum.ts) uses.

import { type ClientConfig } from "./client";
import {
  openReconnectingSocket,
  type SocketStatus,
} from "./reconnectingSocket";

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
export type StatusHandler = (s: SocketStatus) => void;

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
  /** Fired once when the stream gives up on this device; re-enumerate. */
  onGone?: () => void;
  onFrame: FrameHandler;
  onStatus?: StatusHandler;
}

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

export function openIQStream(
  cfg: ClientConfig,
  opts: IQOptions,
): IQStream {
  return openReconnectingSocket({
    url: () => diagWebSocketURL(cfg, opts),
    onStatus: opts.onStatus,
    onGone: opts.onGone,
    onMessage: (data) => {
      try {
        const frame = JSON.parse(data) as IQFrame;
        if (frame && Array.isArray(frame.points)) {
          opts.onFrame(frame);
        }
      } catch {
        // Drop malformed.
      }
    },
  });
}
