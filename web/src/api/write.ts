// Mutation endpoints. Mirrors internal/tui/client/write.go.

import { type ClientConfig, request } from "./client";
import type { AudioStatusDTO, TalkgroupDTO } from "./types";

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
};
