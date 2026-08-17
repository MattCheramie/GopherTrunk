// Channel-baseband spectrum stream client. Mirrors WS /api/v1/diag/mixer
// (see internal/api/mixer.go). Same connect/reconnect scaffolding the
// symbol stream uses; the wire shape is one JSON MixerFrame per message.

import { type ClientConfig } from "./client";
import {
  openReconnectingSocket,
  type SocketStatus,
} from "./reconnectingSocket";

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
export type StatusHandler = (s: SocketStatus) => void;

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
  /** Fired once when the stream gives up on this device; re-enumerate. */
  onGone?: () => void;
  onFrame: MixerFrameHandler;
  onStatus?: StatusHandler;
}

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

export function openMixerStream(
  cfg: ClientConfig,
  opts: MixerOptions,
): MixerStream {
  return openReconnectingSocket({
    url: () => mixerWebSocketURL(cfg, opts),
    onStatus: opts.onStatus,
    onGone: opts.onGone,
    onMessage: (data) => {
      try {
        const frame = JSON.parse(data) as MixerFrame;
        if (frame && Array.isArray(frame.tuned_bins)) {
          opts.onFrame(frame);
        }
      } catch {
        // Drop malformed.
      }
    },
  });
}
