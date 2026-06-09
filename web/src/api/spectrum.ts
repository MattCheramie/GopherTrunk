// Spectrum stream client. Mirrors the openEventStream pattern in
// events.ts: WebSocket with auto-reconnect backoff and an injection
// point for tests. The wire shape is one JSON SpectrumFrame per
// message; see internal/api/spectrum.go for the server side.

import { type ClientConfig, joinURL } from "./client";

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

export interface SpectrumFrame {
  ts_ns: number;
  center_hz: number;
  sample_rate_hz: number;
  bins: number[];
}

export type FrameHandler = (f: SpectrumFrame) => void;
export type StatusHandler = (s: "connecting" | "open" | "closed") => void;

export interface SpectrumStream {
  close(): void;
}

export interface SpectrumOptions {
  serial: string;
  bins?: number; // FFT size, power of two, 64..16384, default 2048
  fps?: number; // 1..30, default 10
  onFrame: FrameHandler;
  onStatus?: StatusHandler;
}

const INITIAL_BACKOFF = 500;
const MAX_BACKOFF = 30_000;

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
    const url = streamWebSocketURL(cfg, opts);
    ws = new WebSocket(url);

    ws.onopen = () => {
      backoff = INITIAL_BACKOFF;
      setStatus("open");
    };

    ws.onmessage = (ev) => {
      if (closed) return;
      try {
        const frame = JSON.parse(ev.data) as SpectrumFrame;
        if (frame && Array.isArray(frame.bins)) {
          opts.onFrame(frame);
        }
      } catch {
        // Drop malformed frame quietly.
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
