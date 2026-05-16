// Mutation endpoints. Mirrors internal/tui/client/write.go.

import { type ClientConfig, joinURL, HTTPError, request } from "./client";
import type {
  AudioStatusDTO,
  ImportPreview,
  ImportResult,
  SettingsPatch,
  SettingsResponse,
  TalkgroupDTO,
} from "./types";

export interface TalkgroupPatch {
  priority?: number;
  lockout?: boolean;
  scan?: boolean;
}

export interface AudioPatch {
  volume?: number;
  muted?: boolean;
  recording_enabled?: boolean;
}

export const writes = {
  endCall: (c: ClientConfig, deviceSerial: string) =>
    request<void>(c, "POST", `/api/v1/calls/${encodeURIComponent(deviceSerial)}/end`),

  updateTalkgroup: (c: ClientConfig, id: number, patch: TalkgroupPatch) =>
    request<TalkgroupDTO>(c, "PATCH", `/api/v1/talkgroups/${id}`, patch),

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
