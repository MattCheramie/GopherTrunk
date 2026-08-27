// Spectrum stream client. Mirrors the openEventStream pattern in
// events.ts: WebSocket with auto-reconnect backoff and an injection
// point for tests. The wire shape is one JSON SpectrumFrame per
// message; see internal/api/spectrum.go for the server side.

import { type ClientConfig, joinURL } from "./client";
import {
  openReconnectingSocket,
  type SocketStatus,
} from "./reconnectingSocket";

export interface SpectrumDevice {
  serial: string;
  driver: string;
  product?: string;
  role: string;
  center_hz: number;
  sample_rate_hz: number;
  // Configured P25 Phase 1 demod mode of the system this SDR is
  // decoding ("c4fm" | "cqpsk"), or absent when no P25 Phase 1 system
  // applies. The symbol/constellation panels' "Auto" mode reads this to
  // pick the receiver without operator input.
  p25_modulation?: string;
  // Symbol-stream receiver selector for the system this SDR is decoding
  // ("p25-c4fm" | "p25-cqpsk" | "p25-phase2" | "tetra" | "dmr" | "nxdn"), or absent when no
  // configured system matches. Protocol-aware, unlike p25_modulation,
  // which only ever describes P25 Phase 1 systems — on a TETRA/DMO/DMR
  // rig that was absent and "Auto" fell back to a P25 C4FM receiver.
  symbol_proto?: string;
  // Configured control-channel frequency in this SDR's passband (Hz), or
  // absent when none matches. The Plots panels rest their view here so
  // they default off the centre DC spike onto a decodable channel. (#557)
  control_channel_hz?: number;
}

// defaultSymbolDevice picks the SDR the symbol-domain panels (Eye
// Diagram, Symbol Scope, Tuning, Histogram) should select on first load.
// It prefers the control-role device so a panel opened during active
// control-channel decoding lands on the SDR that is actually carrying a
// decodable C4FM channel, rather than the enumeration's first entry —
// which on a multi-SDR rig is often an idle voice/aux dongle, leaving the
// scope showing nothing (issue #402). Falls back to the first device when
// no control role is present, and returns null for an empty list.
export function defaultSymbolDevice(
  list: SpectrumDevice[],
): SpectrumDevice | null {
  if (list.length === 0) return null;
  return list.find((d) => d.role === "control") ?? list[0];
}

// initialDeviceSerial picks which SDR a signal-plot panel selects on first
// load. When a `?device=<serial>` deep link names a device that is actually
// in the pool, that wins — this is how a "signal detail" link from a live
// call or a scanner hit lands the scope on the right SDR instead of the
// enumeration default. Otherwise it falls back to the panel's normal default
// (`fallback`, e.g. defaultSymbolDevice or list[0]). Returns null for an
// empty pool.
export function initialDeviceSerial(
  list: SpectrumDevice[],
  wanted: string | null | undefined,
  fallback: (l: SpectrumDevice[]) => SpectrumDevice | null,
): string | null {
  if (wanted && list.some((d) => d.serial === wanted)) return wanted;
  return fallback(list)?.serial ?? null;
}

// parentSerial resolves a wideband virtual voice-tap serial back to its
// parent wideband device. The wbvoice manager registers taps as
// "wb:<parent>:tap-N" (internal/sdr/wbvoice), and active calls on a wideband
// SDR run on those tap serials — but the Plots SDR selector lists the parent
// device. A direct device_serial === selected comparison therefore never
// matches, so the follow-the-call logic falls back to the control channel
// forever. Mapping the tap back to its parent fixes that; non-tap serials
// (direct dongles) pass through unchanged.
export function parentSerial(deviceSerial: string): string {
  const m = deviceSerial.match(/^wb:(.+):tap-\d+$/);
  return m ? m[1] : deviceSerial;
}

export interface SpectrumFrame {
  ts_ns: number;
  center_hz: number;
  sample_rate_hz: number;
  bins: number[];
}

export type FrameHandler = (f: SpectrumFrame) => void;
export type StatusHandler = (s: SocketStatus) => void;

export interface SpectrumStream {
  close(): void;
}

export interface SpectrumOptions {
  serial: string;
  bins?: number; // FFT size, power of two, 64..16384, default 2048
  fps?: number; // 1..30, default 10
  /** Fired once when the stream gives up on this device; re-enumerate. */
  onGone?: () => void;
  onFrame: FrameHandler;
  onStatus?: StatusHandler;
}

export function streamWebSocketURL(cfg: ClientConfig, opts: SpectrumOptions): string {
  const params = new URLSearchParams({ device: opts.serial });
  if (opts.bins != null) params.set("bins", String(opts.bins));
  if (opts.fps != null) params.set("fps", String(opts.fps));
  const u = new URL(
    `/api/v1/spectrum/stream?${params.toString()}`,
    cfg.baseURL || window.location.href,
  );
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  return u.toString();
}

export function openSpectrumStream(
  cfg: ClientConfig,
  opts: SpectrumOptions,
): SpectrumStream {
  return openReconnectingSocket({
    url: () => streamWebSocketURL(cfg, opts),
    onStatus: opts.onStatus,
    onGone: opts.onGone,
    onMessage: (data) => {
      try {
        const frame = JSON.parse(data) as SpectrumFrame;
        if (frame && Array.isArray(frame.bins)) {
          opts.onFrame(frame);
        }
      } catch {
        // Drop malformed.
      }
    },
  });
}

export async function fetchSpectrumDevices(
  cfg: ClientConfig,
): Promise<SpectrumDevice[]> {
  const url = joinURL(cfg.baseURL, "/api/v1/spectrum/devices");
  const headers: Record<string, string> = { Accept: "application/json" };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
  const res = await fetch(url, { headers });
  if (!res.ok) throw new Error(`spectrum/devices ${res.status}`);
  return (await res.json()) as SpectrumDevice[];
}

// tuneSpectrumDevice posts a new centre frequency for the named
// SDR. Wraps POST /api/v1/spectrum/devices/{serial}/tune. Gated
// like every other mutation route — daemons started with auth
// required will reject without a bearer token.
export async function tuneSpectrumDevice(
  cfg: ClientConfig,
  serial: string,
  centerHz: number,
): Promise<void> {
  const url = joinURL(
    cfg.baseURL,
    `/api/v1/spectrum/devices/${encodeURIComponent(serial)}/tune`,
  );
  const headers: Record<string, string> = {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
  const res = await fetch(url, {
    method: "POST",
    headers,
    body: JSON.stringify({ center_hz: Math.round(centerHz) }),
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`tune ${res.status}: ${text || res.statusText}`);
  }
}
