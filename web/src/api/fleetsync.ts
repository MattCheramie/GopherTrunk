// Kenwood FleetSync signalling client. Mirrors GET /api/v1/fleetsync/messages.

import { type ClientConfig, joinURL } from "./client";

export interface FleetSyncMessage {
  id: number;
  received_at: string;
  fleet: number;
  unit: number;
  fs2: boolean;
  body?: string;
  raw_hex?: string;
  crc_ok: boolean;
}

export async function fetchFleetSyncMessages(
  cfg: ClientConfig,
  limit = 200,
): Promise<FleetSyncMessage[]> {
  const url = joinURL(
    cfg.baseURL,
    `/api/v1/fleetsync/messages?limit=${encodeURIComponent(String(limit))}`,
  );
  const headers: Record<string, string> = { Accept: "application/json" };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
  const res = await fetch(url, { headers });
  if (!res.ok) throw new Error(`fleetsync/messages ${res.status}`);
  return (await res.json()) as FleetSyncMessage[];
}
