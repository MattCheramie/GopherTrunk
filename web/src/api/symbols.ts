// Recovered-symbol stream client. Mirrors WS /api/v1/diag/symbols.
// Same connect/reconnect scaffolding openIQStream (api/diag.ts) uses.

import { type ClientConfig } from "./client";

export interface SymbolFrame {
  ts_ns: number;
  symbol_rate_hz: number;
  center_hz: number;
  offset_hz: number;
  // Pre-slicer soft waveform; empty when the demod path has no soft tap
  // (e.g. P25 CQPSK). When present it is aligned index-for-index with
  // dibits.
  soft: number[];
  // Complex symbol-decision points (the true constellation): in-phase /
  // quadrature components sampled at each symbol instant. Populated only
  // on the linear/CQPSK path; empty on C4FM. When present they are
  // aligned index-for-index with dibits.
  sym_i: number[];
  sym_q: number[];
  // Bounded window of oversampled, AGC-scaled matched-filter output
  // (eye_sps samples per symbol) for the C4FM eye diagram; empty on
  // CQPSK. Fold over eye_sps to render the 4-level eye.
  eye_soft: number[];
  eye_sps: number;
  // Sliced decisions: 0..3 when is_bits is false, 0..1 when true.
  dibits: number[];
  is_bits: boolean;
  base_idx: number;
  // Receiver-state metrics for the Tuning panel. carrier_offset_hz is the
  // AFC estimate (C4FM) or carrier-recovery estimate (CQPSK); agc_level
  // the C4FM symbol-AGC mean|x| or CQPSK matched-filter gain; agc_target
  // the C4FM AGC target (0 on CQPSK); clock_mu/clock_sps the symbol-clock
  // loop state; cma_error the CQPSK equalizer convergence proxy (0 on C4FM).
  carrier_offset_hz: number;
  agc_level: number;
  agc_target: number;
  clock_mu: number;
  clock_sps: number;
  cma_error: number;
}

// Map a device's reported P25 demod mode (from SpectrumDevice
// .p25_modulation — the daemon's canonical "c4fm"/"cqpsk", with the
// other ParseDemodMode spellings tolerated) to the symbol-stream proto
// selector. Unknown / empty falls back to C4FM, matching the receiver's
// default. Used by the panels' "Auto" mode to pick the receiver from the
// system the SDR is actually decoding.
export function demodModeToProto(mod: string | undefined | null): string {
  switch ((mod ?? "").toLowerCase()) {
    case "cqpsk":
    case "lsm":
    case "linear":
      return "p25-cqpsk";
    case "tetra":
    case "pi/4-dqpsk":
    case "pi4dqpsk":
    case "dqpsk":
      return "tetra";
    default:
      return "p25-c4fm";
  }
}

export type SymbolFrameHandler = (f: SymbolFrame) => void;
export type StatusHandler = (s: "connecting" | "open" | "closed") => void;

export interface SymbolStream {
  close(): void;
}

export interface SymbolOptions {
  serial: string;
  // Receiver selector: "p25-c4fm" (soft + dibits) or "p25-cqpsk"
  // (dibits only). Default "p25-c4fm".
  proto?: string;
  // Frequency offset in Hz, relative to the SDR centre, tuned down to
  // baseband before channelizing. Default 0.
  offset?: number;
  onFrame: SymbolFrameHandler;
  onStatus?: StatusHandler;
}

const INITIAL_BACKOFF = 500;
const MAX_BACKOFF = 30_000;

export function symbolWebSocketURL(cfg: ClientConfig, opts: SymbolOptions): string {
  const params = new URLSearchParams({ device: opts.serial });
  if (opts.proto) params.set("proto", opts.proto);
  if (opts.offset != null && opts.offset !== 0)
    params.set("offset", String(Math.round(opts.offset)));
  const u = new URL(
    `/api/v1/diag/symbols?${params.toString()}`,
    cfg.baseURL || window.location.href,
  );
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  return u.toString();
}

export function openSymbolStream(cfg: ClientConfig, opts: SymbolOptions): SymbolStream {
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
    const url = symbolWebSocketURL(cfg, opts);
    ws = new WebSocket(url);

    ws.onopen = () => {
      backoff = INITIAL_BACKOFF;
      setStatus("open");
    };

    ws.onmessage = (ev) => {
      if (closed) return;
      try {
        const frame = JSON.parse(ev.data) as SymbolFrame;
        if (frame && Array.isArray(frame.dibits)) {
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
