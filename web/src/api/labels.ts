// Operator-applied radio / talkgroup names. Mirrors the routes registered in
// internal/api/handlers_labels.go.
//
// Writes go through the existing PATCH /api/v1/rids/{id} and
// /api/v1/talkgroups/{id} (see write.ts) — this module covers reading the
// persisted rows back and building the CSV export URL.

import { type ClientConfig, joinURL } from "./client";

export type LabelKind = "rid" | "talkgroup";

export interface LabelDTO {
  id: number;
  kind: LabelKind;
  system?: string;
  target_id: number;
  name?: string;
  description?: string;
  tag?: string;
  group?: string;
  owner?: string;
  icon?: string;
  created_at: string;
  updated_at: string;
}

// exportURL builds the download link for a labels CSV. The endpoint is
// read-only and ungated, so the UI can use a plain anchor rather than a fetch
// plus an object URL.
export function exportURL(
  cfg: ClientConfig,
  kind: LabelKind,
  opts: { system?: string; scope?: "labels" | "all" } = {},
): string {
  const q = new URLSearchParams({ kind });
  if (opts.system) q.set("system", opts.system);
  if (opts.scope) q.set("scope", opts.scope);
  return joinURL(cfg.baseURL, `/api/v1/labels/export?${q.toString()}`);
}

export const labels = {
  list: async (
    cfg: ClientConfig,
    opts: { kind?: LabelKind; system?: string } = {},
  ): Promise<LabelDTO[]> => {
    const q = new URLSearchParams();
    if (opts.kind) q.set("kind", opts.kind);
    if (opts.system) q.set("system", opts.system);
    const qs = q.toString();
    const url = joinURL(cfg.baseURL, `/api/v1/labels${qs ? `?${qs}` : ""}`);
    const headers: Record<string, string> = { Accept: "application/json" };
    if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
    const res = await fetch(url, { headers });
    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(`/api/v1/labels ${res.status}: ${text || res.statusText}`);
    }
    const body = (await res.json()) as { labels?: LabelDTO[] };
    return body.labels ?? [];
  },

  remove: async (
    cfg: ClientConfig,
    kind: LabelKind,
    id: number,
    system?: string,
  ): Promise<void> => {
    const q = system ? `?system=${encodeURIComponent(system)}` : "";
    const url = joinURL(cfg.baseURL, `/api/v1/labels/${kind}/${id}${q}`);
    const headers: Record<string, string> = {};
    if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
    const res = await fetch(url, { method: "DELETE", headers });
    if (!res.ok && res.status !== 404) {
      const text = await res.text().catch(() => "");
      throw new Error(`/api/v1/labels ${res.status}: ${text || res.statusText}`);
    }
  },
};

export { exportURL as labelExportURL };
