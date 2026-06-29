// HTTP client for the siglab REST API (internal/api/handlers_siglab.go).
// Same-origin by default (the `siglab serve` command hosts both the SPA and
// the API), with an optional baseURL + bearer token so the SPA can also point
// at a remote daemon.

import type {
  CaptureDTO,
  CaptureDevice,
  CaptureRequest,
  CaptureResponse,
  IQTaps,
  IdentifyResult,
  JobDTO,
  OccupancyResult,
  ProtocolsDTO,
  PSDResult,
  RunConfig,
  SpectrogramResult,
  SynthRequest,
  WidebandRequest,
  WidebandResult,
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

function authHeaders(c: ClientConfig): Record<string, string> {
  const h: Record<string, string> = {};
  if (c.token) h["Authorization"] = `Bearer ${c.token}`;
  return h;
}

export function joinURL(base: string, path: string): string {
  const b = base.replace(/\/+$/, "");
  const p = path.replace(/^\/+/, "");
  return b ? `${b}/${p}` : `/${p}`;
}

async function req<T>(
  c: ClientConfig,
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    ...authHeaders(c),
  };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(joinURL(c.baseURL, path), {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: "include",
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    let msg = text;
    try {
      msg = (JSON.parse(text) as { error?: string }).error ?? text;
    } catch {
      /* not JSON */
    }
    throw new HTTPError(res.status, text, `${method} ${path} → ${res.status}: ${msg}`);
  }
  if (res.status === 204) return undefined as T;
  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) return (await res.text()) as unknown as T;
  return (await res.json()) as T;
}

export const api = {
  protocols: (c: ClientConfig) => req<ProtocolsDTO>(c, "GET", "/api/v1/siglab/protocols"),

  uploadCapture: async (
    c: ClientConfig,
    file: File,
    params: { format: string; sample_rate_hz?: number },
  ): Promise<CaptureDTO> => {
    const fd = new FormData();
    fd.append("file", file);
    fd.append("format", params.format);
    if (params.sample_rate_hz != null)
      fd.append("sample_rate_hz", String(params.sample_rate_hz));
    const res = await fetch(joinURL(c.baseURL, "/api/v1/siglab/captures"), {
      method: "POST",
      headers: authHeaders(c),
      body: fd,
      credentials: "include",
    });
    if (!res.ok) {
      const t = await res.text().catch(() => "");
      throw new HTTPError(res.status, t, `upload → ${res.status}: ${t}`);
    }
    return (await res.json()) as CaptureDTO;
  },

  deleteCapture: (c: ClientConfig, id: string) =>
    req<void>(c, "DELETE", `/api/v1/siglab/captures/${id}`),

  run: (c: ClientConfig, captureID: string, config: RunConfig) =>
    req<{ job_id: string }>(c, "POST", "/api/v1/siglab/run", {
      capture_id: captureID,
      config,
    }),

  job: (c: ClientConfig, id: string) => req<JobDTO>(c, "GET", `/api/v1/siglab/jobs/${id}`),

  jobIQ: (c: ClientConfig, id: string) =>
    req<IQTaps>(c, "GET", `/api/v1/siglab/jobs/${id}/iq`),

  // jobPSD / jobSpectrogram fetch the server-computed spectrum of a job's IQ,
  // so the UI plots them without an in-browser FFT (no TensorFlow.js).
  jobPSD: (c: ClientConfig, id: string, fft = 1024) =>
    req<PSDResult>(c, "GET", `/api/v1/siglab/jobs/${id}/psd?fft=${fft}`),

  jobSpectrogram: (c: ClientConfig, id: string, fft = 256, hop = 128) =>
    req<SpectrogramResult>(
      c,
      "GET",
      `/api/v1/siglab/jobs/${id}/spectrogram?fft=${fft}&hop=${hop}`,
    ),

  // jobOccupancy fetches the server-computed spectral-occupancy metrics
  // (occupied bandwidth, channel power, ACPR, flatness) off the same spectrum.
  jobOccupancy: (c: ClientConfig, id: string, fft = 1024) =>
    req<OccupancyResult>(c, "GET", `/api/v1/siglab/jobs/${id}/occupancy?fft=${fft}`),

  identify: (
    c: ClientConfig,
    captureID: string,
    opts: { sample_rate_hz?: number; format?: string; max_seconds?: number } = {},
  ) =>
    req<IdentifyResult>(c, "POST", "/api/v1/siglab/identify", {
      capture_id: captureID,
      ...opts,
    }),

  // wideband surveys a staged multi-carrier capture: averaged band spectrum +
  // per-carrier identification + trunked-system rollups.
  wideband: (c: ClientConfig, body: WidebandRequest) =>
    req<WidebandResult>(c, "POST", "/api/v1/siglab/wideband", body),

  // synthesize stages an idealized/impaired capture and returns its DTO.
  synthesize: (c: ClientConfig, body: SynthRequest) =>
    req<{ capture: CaptureDTO; metadata: unknown }>(
      c,
      "POST",
      "/api/v1/siglab/synthesize",
      body,
    ),

  // captureDevices lists the SDRs available to record from. 503 (HTTPError)
  // when the console is offline (`siglab serve`) or the daemon has no SDR.
  captureDevices: (c: ClientConfig) =>
    req<CaptureDevice[]>(c, "GET", "/api/v1/siglab/capture/devices"),

  // capture records a fixed-length raw-IQ capture off a live tuner, stages it,
  // and returns the staged capture + a download URL for the raw .cfile.
  capture: (c: ClientConfig, body: CaptureRequest) =>
    req<CaptureResponse>(c, "POST", "/api/v1/siglab/capture", body),

  // URLs for browser-native consumers (EventSource, <a download>).
  jobEventsURL: (c: ClientConfig, id: string) =>
    joinURL(c.baseURL, `/api/v1/siglab/jobs/${id}/events`),
  exportURL: (c: ClientConfig, id: string, format: string) =>
    joinURL(c.baseURL, `/api/v1/siglab/jobs/${id}/export?format=${format}`),
  synthDownloadURL: (c: ClientConfig) =>
    joinURL(c.baseURL, "/api/v1/siglab/synthesize?download=1"),
  captureDownloadURL: (c: ClientConfig, id: string) =>
    joinURL(c.baseURL, `/api/v1/siglab/captures/${id}/download`),
};
