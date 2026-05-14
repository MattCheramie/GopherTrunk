// HTTP client for the GopherTrunk daemon. Mirrors the read surface of
// internal/tui/client/client.go. Every method takes the server's
// base URL + optional bearer token so a single SPA instance can
// hop between daemons (e.g. dev → Raspberry Pi → laptop) without
// reloading.

import type {
  ActiveCallDTO,
  AudioStatusDTO,
  CallRow,
  DeviceDTO,
  Health,
  Mutations,
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

export class HTTPError extends Error {
  constructor(
    public readonly status: number,
    public readonly body: string,
    message: string,
  ) {
    super(message);
    this.name = "HTTPError";
  }
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
    throw new HTTPError(
      res.status,
      text,
      `${method} ${path} → ${res.status}: ${text || res.statusText}`,
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
  activeCalls: (c: ClientConfig) =>
    request<{ calls: ActiveCallDTO[] }>(
      c,
      "GET",
      "/api/v1/calls/active",
    ).then((r) => r.calls),
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
  audio: (c: ClientConfig) =>
    request<AudioStatusDTO>(c, "GET", "/api/v1/audio"),
  metricsText: (c: ClientConfig) =>
    request<string>(c, "GET", "/metrics", undefined, 5_000),
};

// Re-export the request helper for the write module.
export { request };

// Probe the supplied (URL, token) by hitting /api/v1/health. Returns
// the health body on success, throws an HTTPError otherwise. Used by
// the connect screen to validate before saving credentials.
export async function probe(cfg: ClientConfig): Promise<Health> {
  return await api.health(cfg);
}

// audioStreamURL composes the URL of the live PCM stream. Browsers
// cannot attach an Authorization header to <audio> elements, so we
// pass the token (if any) as a query parameter is *not* supported by
// the daemon — instead, deployments that require auth must serve the
// stream from a trusted-network bind (auto mode) or use the
// fetch + AudioWorklet path which can supply headers. For the simple
// <audio> tag path we rely on the daemon's "reads are open" policy.
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

// eventsWebSocketURL composes the WS URL. WebSocket's protocol is
// derived from the base URL (http→ws, https→wss).
export function eventsWebSocketURL(cfg: ClientConfig): string {
  const http = joinURL(cfg.baseURL, "/api/v1/events/ws");
  return http.replace(/^http(s?):/, (_m, s) => `ws${s}:`);
}
