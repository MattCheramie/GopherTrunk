// HTTP client for the GopherTrunk daemon. Mirrors the read surface of
// internal/tui/client/client.go. Every method takes the server's
// base URL + optional bearer token so a single SPA instance can
// hop between daemons (e.g. dev → Raspberry Pi → laptop) without
// reloading.

import type {
  ActiveCallDTO,
  AudioStatusDTO,
  CallRow,
  LocationFix,
  ConfigListResponse,
  DeviceDTO,
  Health,
  HuntRRReport,
  HuntStatus,
  Mutations,
  RIDDTO,
  RuntimeDTO,
  ScannerStatusDTO,
  SystemDTO,
  TalkgroupDTO,
  Version,
} from "./types";

export interface ClientConfig {
  baseURL: string;
  token: string | null;
}

// DiagInfo is the optional diagnostics payload the daemon attaches to
// an error envelope when diagnostics.verbose_errors is enabled. The
// banner carries version/OS/system/dongle context useful for triage.
export interface DiagInfo {
  banner: string;
  trace?: string[];
  stack?: string;
}

export class HTTPError extends Error {
  constructor(
    public readonly status: number,
    public readonly body: string,
    message: string,
    // diag is populated when the daemon returned a verbose error
    // envelope ({"error":..., "diag":{"banner":...}}). undefined when
    // verbose errors are off or the body wasn't the JSON envelope.
    public readonly diag?: DiagInfo,
  ) {
    super(message);
    this.name = "HTTPError";
  }
}

// parseErrorBody extracts the {"error", "diag"} envelope from a JSON
// error body. Returns the concise message and the optional diag block;
// falls back to the raw text for non-JSON bodies.
function parseErrorBody(text: string): { message: string; diag?: DiagInfo } {
  try {
    const obj = JSON.parse(text) as { error?: string; diag?: DiagInfo };
    if (obj && typeof obj === "object" && (obj.error || obj.diag)) {
      return { message: obj.error ?? text, diag: obj.diag };
    }
  } catch {
    // not JSON — fall through to raw text
  }
  return { message: text };
}

/** Default per-request timeout. Long enough for a slow Pi, short enough
 *  that a wedged daemon doesn't freeze the UI. */
const DEFAULT_TIMEOUT_MS = 10_000;

async function request<T>(
  cfg: ClientConfig,
  method: string,
  path: string,
  body?: unknown,
  timeoutMs = DEFAULT_TIMEOUT_MS,
): Promise<T> {
  const url = joinURL(cfg.baseURL, path);
  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";

  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), timeoutMs);

  let res: Response;
  try {
    res = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
      // CORS-enabled daemons get credentials so the bearer token
      // header passes through preflight. Same-origin reverse-proxy
      // deployments are unaffected.
      credentials: "include",
    });
  } finally {
    window.clearTimeout(timer);
  }

  if (!res.ok) {
    const text = await safeReadText(res);
    const { message, diag } = parseErrorBody(text);
    throw new HTTPError(
      res.status,
      text,
      `${method} ${path} → ${res.status}: ${message || res.statusText}`,
      diag,
    );
  }

  if (res.status === 204) return undefined as T;
  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) {
    return (await res.text()) as unknown as T;
  }
  return (await res.json()) as T;
}

async function safeReadText(res: Response): Promise<string> {
  try {
    return await res.text();
  } catch {
    return "";
  }
}

export function joinURL(base: string, path: string): string {
  // Trim trailing slashes off base and leading slashes off path so
  // we never end up with "/api//v1/health" or "//api/v1/health".
  const b = base.replace(/\/+$/, "");
  const p = path.replace(/^\/+/, "");
  return `${b}/${p}`;
}

export const api = {
  health: (c: ClientConfig) => request<Health>(c, "GET", "/api/v1/health"),
  version: (c: ClientConfig) => request<Version>(c, "GET", "/api/v1/version"),
  mutations: (c: ClientConfig) =>
    request<Mutations>(c, "GET", "/api/v1/mutations"),
  runtime: (c: ClientConfig) =>
    request<RuntimeDTO>(c, "GET", "/api/v1/runtime"),
  systems: (c: ClientConfig) =>
    request<{ systems: SystemDTO[] }>(c, "GET", "/api/v1/systems").then(
      (r) => r.systems,
    ),
  talkgroups: (c: ClientConfig) =>
    request<{ talkgroups: TalkgroupDTO[] }>(
      c,
      "GET",
      "/api/v1/talkgroups",
    ).then((r) => r.talkgroups),
  rids: (c: ClientConfig, opts: { system?: string } = {}) => {
    const q = new URLSearchParams();
    if (opts.system) q.set("system", opts.system);
    const qs = q.toString();
    return request<{ rids: RIDDTO[] }>(
      c,
      "GET",
      `/api/v1/rids${qs ? `?${qs}` : ""}`,
    ).then((r) => r.rids);
  },
  // Fetch one merged RID row by id. Used to open the /rids/:id deep link
  // (e.g. a source-radio click in CC Activity / call history) even when the
  // radio isn't in the currently-filtered list. Returns the RIDDTO directly
  // (the endpoint is unwrapped, unlike the list route).
  rid: (c: ClientConfig, id: number) =>
    request<RIDDTO>(c, "GET", `/api/v1/rids/${id}`),
  ridHistory: (
    c: ClientConfig,
    id: number,
    opts: { limit?: number } = {},
  ) => {
    const q = new URLSearchParams();
    if (opts.limit != null) q.set("limit", String(opts.limit));
    const qs = q.toString();
    return request<{ calls: CallRow[] }>(
      c,
      "GET",
      `/api/v1/rids/${id}/history${qs ? `?${qs}` : ""}`,
    ).then((r) => r.calls ?? []);
  },
  activeCalls: (c: ClientConfig) =>
    request<{ calls: ActiveCallDTO[] }>(
      c,
      "GET",
      "/api/v1/calls/active",
    ).then((r) => r.calls),
  // Recent subscriber-radio position fixes (DMR LRRP / unit GPS) for the
  // unit-location map. Optionally scoped by radio_id client-side.
  locations: (c: ClientConfig, opts: { limit?: number } = {}) => {
    const q = new URLSearchParams();
    if (opts.limit != null) q.set("limit", String(opts.limit));
    const qs = q.toString();
    return request<{ locations: LocationFix[] }>(
      c,
      "GET",
      `/api/v1/locations${qs ? `?${qs}` : ""}`,
    ).then((r) => r.locations ?? []);
  },
  history: (
    c: ClientConfig,
    opts: { limit?: number; system?: string; group_id?: number } = {},
  ) => {
    const q = new URLSearchParams();
    if (opts.limit != null) q.set("limit", String(opts.limit));
    if (opts.system) q.set("system", opts.system);
    if (opts.group_id != null) q.set("group_id", String(opts.group_id));
    const qs = q.toString();
    return request<{ rows: CallRow[] }>(
      c,
      "GET",
      `/api/v1/calls/history${qs ? `?${qs}` : ""}`,
    ).then((r) => r.rows ?? []);
  },
  devices: (c: ClientConfig) =>
    request<{ devices: DeviceDTO[] }>(c, "GET", "/api/v1/devices").then(
      (r) => r.devices,
    ),
  scanner: (c: ClientConfig) =>
    request<ScannerStatusDTO>(c, "GET", "/api/v1/scanner"),
  hunt: (c: ClientConfig) => request<HuntStatus>(c, "GET", "/api/v1/hunt"),
  huntRadioReference: (
    c: ClientConfig,
    params: { countyID?: number; checkSIDs?: number[] },
  ) => {
    const q = new URLSearchParams();
    if (params.countyID) q.set("county_id", String(params.countyID));
    for (const sid of params.checkSIDs ?? []) q.append("check_sid", String(sid));
    return request<HuntRRReport>(c, "GET", `/api/v1/hunt/radioreference?${q.toString()}`);
  },
  audio: (c: ClientConfig) =>
    request<AudioStatusDTO>(c, "GET", "/api/v1/audio"),
  metricsText: (c: ClientConfig) =>
    request<string>(c, "GET", "/metrics", undefined, 5_000),
  // Config files discovered by the Config Builder subsystem, for the
  // Settings config-file picker. Returns 503 (caught by the caller) when
  // the daemon was built/started without the config-builder subsystem.
  configFiles: (c: ClientConfig) =>
    request<ConfigListResponse>(c, "GET", "/api/v1/config/files"),
};

// Re-export the request helper for the write module.
export { request };

// Probe the supplied (URL, token) by hitting /api/v1/health. Returns
// the health body on success, throws an HTTPError otherwise. Used by
// the connect screen to validate before saving credentials.
export async function probe(cfg: ClientConfig): Promise<Health> {
  return await api.health(cfg);
}

// audioStreamURL composes the URL of the live PCM stream. The WebUI
// player (see ../audio/streamPlayer.ts) fetches this URL and decodes the
// WAV body through the Web Audio API, so it can attach the
// Authorization: Bearer header on auth-gated daemons — the old hidden
// <audio> element could not, which is why the stream had to be served
// from a trusted-network bind. The token is sent as a header by the
// fetch, never as a query parameter, so it stays out of access logs.
export function audioStreamURL(
  cfg: ClientConfig,
  filter: { device?: string; talkgroup?: number } = {},
): string {
  const q = new URLSearchParams();
  if (filter.device) q.set("device", filter.device);
  if (filter.talkgroup != null) q.set("talkgroup", String(filter.talkgroup));
  const qs = q.toString();
  return joinURL(cfg.baseURL, `/api/v1/audio/stream${qs ? `?${qs}` : ""}`);
}

// eventsWebSocketURL composes the WS URL. The scheme is derived from
// the base URL (http→ws, https→wss); an empty baseURL means the SPA is
// served same-origin by the daemon, so we resolve against the current
// document location. The URL constructor (rather than string concat)
// avoids malformed output like "ws:/api/v1/events/ws" when baseURL
// lacks a host, and normalizes an uppercase scheme.
export function eventsWebSocketURL(cfg: ClientConfig): string {
  const u = new URL("/api/v1/events/ws", cfg.baseURL || window.location.href);
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  return u.toString();
}
