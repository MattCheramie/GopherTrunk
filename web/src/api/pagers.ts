// Pager log client. Mirrors GET /api/v1/pager/messages.

import { type ClientConfig, joinURL } from "./client";

export interface PagerMessage {
  id: number;
  received_at: string;
  protocol: string;
  ric: number;
  func: number;
  encoding: string;
  body: string;
  corrected: number;
}

export async function fetchPagerMessages(
  cfg: ClientConfig,
  limit = 200,
): Promise<PagerMessage[]> {
  const url = joinURL(
    cfg.baseURL,
    `/api/v1/pager/messages?limit=${encodeURIComponent(String(limit))}`,
  );
  const headers: Record<string, string> = { Accept: "application/json" };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
  const res = await fetch(url, { headers });
  if (!res.ok) throw new Error(`pager/messages ${res.status}`);
  return (await res.json()) as PagerMessage[];
}

export function functionLabel(fn: number): string {
  if (fn >= 0 && fn <= 3) return String.fromCharCode("A".charCodeAt(0) + fn);
  return "?";
}
