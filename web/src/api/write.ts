// Mutation endpoints. Mirrors internal/tui/client/write.go.

import { type ClientConfig, joinURL, HTTPError, request } from "./client";
import type {
  AudioStatusDTO,
  ConfigActivateResponse,
  HuntStartRequest,
  HuntCaptureRequest,
  HuntCaptureResult,
  ImportPreview,
  ImportResult,
  RIDDTO,
  SettingsPatch,
  SettingsResponse,
  TalkgroupDTO,
} from "./types";

export interface TalkgroupPatch {
  // alpha_tag / description / tag / group are the operator-applied name and
  // its descriptive fields; they persist when the daemon has a label store,
  // unlike the policy fields below.
  alpha_tag?: string;
  description?: string;
  tag?: string;
  group?: string;
  priority?: number;
  lockout?: boolean;
  scan?: boolean;
}

export interface RIDPatch {
  alias?: string;
  description?: string;
  tag?: string;
  group?: string;
  owner?: string;
  priority?: number;
  lockout?: boolean;
  watch?: boolean;
  icon?: string;
}

export interface AudioPatch {
  volume?: number;
  muted?: boolean;
  recording_enabled?: boolean;
}

function systemQuery(system?: string): string {
  return system ? `?system=${encodeURIComponent(system)}` : "";
}

export const writes = {
  endCall: (c: ClientConfig, deviceSerial: string) =>
    request<void>(c, "POST", `/api/v1/calls/${encodeURIComponent(deviceSerial)}/end`),

  // system, when given, scopes a persisted label to one trunking system —
  // talkgroup_file / rid_alias_file are per-system config keys, so an export
  // has to be able to emit one file per system. Omitted, the name applies
  // wherever the id is seen.
  updateTalkgroup: (
    c: ClientConfig,
    id: number,
    patch: TalkgroupPatch,
    system?: string,
  ) =>
    request<TalkgroupDTO>(
      c,
      "PATCH",
      `/api/v1/talkgroups/${id}${systemQuery(system)}`,
      patch,
    ),

  updateRID: (c: ClientConfig, id: number, patch: RIDPatch, system?: string) =>
    request<RIDDTO>(c, "PATCH", `/api/v1/rids/${id}${systemQuery(system)}`, patch),

  sweepRetention: (c: ClientConfig) =>
    request<void>(c, "POST", "/api/v1/retention/sweep"),

  toneReset: (c: ClientConfig, serial: string) =>
    request<void>(
      c,
      "POST",
      `/api/v1/devices/${encodeURIComponent(serial)}/tone-reset`,
    ),

  setAudio: (c: ClientConfig, patch: AudioPatch) =>
    request<AudioStatusDTO>(c, "PATCH", "/api/v1/audio", patch),

  setScanMode: (c: ClientConfig, mode: "all" | "list") =>
    request<void>(c, "PATCH", "/api/v1/scanner", { scan_mode: mode }),

  // Live system-discovery (hunt) controls.
  huntStart: (c: ClientConfig, req: HuntStartRequest) =>
    request<{ run_id: number }>(c, "POST", "/api/v1/hunt/start", req),
  huntStop: (c: ClientConfig) => request<void>(c, "POST", "/api/v1/hunt/stop"),

  // List-driven capture: record one signal from the inventory and route it to
  // SigLab / CryptoLab.
  huntCapture: (c: ClientConfig, req: HuntCaptureRequest) =>
    request<HuntCaptureResult>(c, "POST", "/api/v1/hunt/capture", req),

  huntHold: (c: ClientConfig, system: string) =>
    request<void>(
      c,
      "POST",
      `/api/v1/scanner/hunt/${encodeURIComponent(system)}/hold`,
    ),

  huntResume: (c: ClientConfig, system: string) =>
    request<void>(
      c,
      "POST",
      `/api/v1/scanner/hunt/${encodeURIComponent(system)}/resume`,
    ),

  huntRetune: (c: ClientConfig, system: string) =>
    request<void>(
      c,
      "POST",
      `/api/v1/scanner/hunt/${encodeURIComponent(system)}/retune`,
    ),

  convHold: (c: ClientConfig) =>
    request<void>(c, "POST", "/api/v1/scanner/conventional/hold"),
  convResume: (c: ClientConfig) =>
    request<void>(c, "POST", "/api/v1/scanner/conventional/resume"),
  convDwell: (c: ClientConfig, index: number) =>
    request<void>(c, "POST", `/api/v1/scanner/conventional/${index}/dwell`),
  convLockout: (c: ClientConfig, index: number) =>
    request<void>(c, "POST", `/api/v1/scanner/conventional/${index}/lockout`),
  convUnlockout: (c: ClientConfig, index: number) =>
    request<void>(
      c,
      "POST",
      `/api/v1/scanner/conventional/${index}/unlockout`,
    ),

  manualTune: (
    c: ClientConfig,
    body: {
      frequency_hz: number;
      label?: string;
      mode?: "fm" | "nfm";
      squelch_dbfs?: number;
      hangtime_ms?: number;
    },
  ) => request<{ index: number }>(c, "POST", "/api/v1/scanner/manual_tune", body),

  clearManualTune: (c: ClientConfig, index: number) =>
    request<void>(c, "DELETE", `/api/v1/scanner/manual_tune/${index}`),

  updateSettings: (c: ClientConfig, patch: SettingsPatch) =>
    request<SettingsResponse>(c, "PATCH", "/api/v1/settings", patch),

  // Load / hot-swap the config file the daemon runs on. "reload"
  // hot-applies what it can and flags the rest as restart-required;
  // "restart" re-execs the daemon into the new file (the connection
  // drops and the SPA reconnects).
  activateConfig: (c: ClientConfig, path: string, mode: "reload" | "restart") =>
    request<ConfigActivateResponse>(c, "POST", "/api/v1/config/activate", {
      path,
      mode,
    }),

  importUpload: (c: ClientConfig, files: File[]) =>
    importMultipart(c, files),

  importCommit: (c: ClientConfig, id: string, force = false) =>
    request<ImportResult>(
      c,
      "POST",
      `/api/v1/import/${encodeURIComponent(id)}/commit${force ? "?force=true" : ""}`,
    ),

  importDiscard: (c: ClientConfig, id: string) =>
    request<void>(c, "DELETE", `/api/v1/import/${encodeURIComponent(id)}`),
};

// importMultipart wraps the multipart upload separately because
// request() in client.ts only does JSON bodies. The shape matches
// the request helper otherwise (token, bearer, abort timeout).
async function importMultipart(
  cfg: ClientConfig,
  files: File[],
): Promise<ImportPreview> {
  const url = joinURL(cfg.baseURL, "/api/v1/import");
  const headers: Record<string, string> = { Accept: "application/json" };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;

  const body = new FormData();
  for (const f of files) body.append("files", f, f.name);

  const controller = new AbortController();
  // 60s — uploads can be large enough that the default 10s would
  // race a slow Pi over wifi.
  const timer = window.setTimeout(() => controller.abort(), 60_000);
  let res: Response;
  try {
    res = await fetch(url, {
      method: "POST",
      headers,
      body,
      credentials: "include",
      signal: controller.signal,
    });
  } finally {
    window.clearTimeout(timer);
  }
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new HTTPError(
      res.status,
      text,
      `POST /api/v1/import → ${res.status}: ${text || res.statusText}`,
    );
  }
  return (await res.json()) as ImportPreview;
}
