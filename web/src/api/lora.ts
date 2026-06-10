// LoRa frame-log client. Mirrors GET /api/v1/lora/frames.

import { type ClientConfig, joinURL } from "./client";

export interface LoRaFrame {
  id: number;
  received_at: string;
  frequency_hz: number;
  sf: number;
  cr: number;
  bandwidth_hz: number;
  rssi: number;
  snr: number;
  cfo_hz: number;
  crc_ok: boolean;
  payload_hex?: string;

  // LoRaWAN MAC fields — present when is_lorawan is true.
  is_lorawan: boolean;
  mtype?: string;
  dev_addr?: string;
  fcnt: number;
  fport: number;
  mic_ok: boolean;
  decrypted: boolean;
  decoded?: string;
}

export async function fetchLoRaFrames(
  cfg: ClientConfig,
  limit = 200,
): Promise<LoRaFrame[]> {
  const url = joinURL(
    cfg.baseURL,
    `/api/v1/lora/frames?limit=${encodeURIComponent(String(limit))}`,
  );
  const headers: Record<string, string> = { Accept: "application/json" };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
  const res = await fetch(url, { headers });
  if (!res.ok) throw new Error(`lora/frames ${res.status}`);
  return (await res.json()) as LoRaFrame[];
}

// crLabel renders the coding-rate index (1..4) as the 4/5..4/8 fraction.
export function crLabel(cr: number): string {
  return cr >= 1 && cr <= 4 ? `4/${cr + 4}` : "—";
}
