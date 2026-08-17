// Recovered-symbol stream client. Mirrors WS /api/v1/diag/symbols.
// Same connect/reconnect scaffolding openIQStream (api/diag.ts) uses.

import { type ClientConfig } from "./client";
import {
  openReconnectingSocket,
  type SocketStatus,
} from "./reconnectingSocket";

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

// autoProtoFor picks the symbol-stream receiver for a device in the panels'
// "Auto" mode. It prefers the daemon's protocol-aware symbol_proto and only
// falls back to inferring one from p25_modulation for a daemon too old to send
// it.
//
// The distinction matters: p25_modulation is populated only for P25 Phase 1
// systems, so on a TETRA, TETRA DMO or DMR rig it is absent and the fallback
// yields p25-c4fm. That opens a 4-level C4FM receiver on a π/4-DQPSK carrier,
// whose soft track is meaningless but non-empty — which is what made the
// symbol-quality chip compute an MER around 9 dB and read "symbol: poor"
// permanently while the decode chip correctly read "decode: clean".
export function autoProtoFor(
  device: { symbol_proto?: string; p25_modulation?: string } | null | undefined,
): string {
  return device?.symbol_proto || demodModeToProto(device?.p25_modulation);
}

// Map a device's reported P25 demod mode (from SpectrumDevice
// .p25_modulation — the daemon's canonical "c4fm"/"cqpsk", with the
// other ParseDemodMode spellings tolerated) to the symbol-stream proto
// selector. Unknown / empty falls back to C4FM, matching the receiver's
// default. Prefer autoProtoFor, which consults the protocol-aware
// symbol_proto first.
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
export type StatusHandler = (s: SocketStatus) => void;

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
  /** Fired once when the stream gives up on this device; re-enumerate. */
  onGone?: () => void;
  onFrame: SymbolFrameHandler;
  onStatus?: StatusHandler;
}

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

export function openSymbolStream(
  cfg: ClientConfig,
  opts: SymbolOptions,
): SymbolStream {
  return openReconnectingSocket({
    url: () => symbolWebSocketURL(cfg, opts),
    onStatus: opts.onStatus,
    onGone: opts.onGone,
    onMessage: (data) => {
      try {
        const frame = JSON.parse(data) as SymbolFrame;
        if (frame && Array.isArray(frame.dibits)) {
          opts.onFrame(frame);
        }
      } catch {
        // Drop malformed.
      }
    },
  });
}
