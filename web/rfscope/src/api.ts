// TypeScript mirrors of the rfscope JSON DTOs (internal/rfscope/scene.go) and a
// tiny fetch client for the rfscope web API.

export interface HierStats {
  emitter_count: number;
  burst_count: number;
  airtime_sec: number;
  spectrum_pct: number;
  time_pct: number;
  symbol_volume: number;
  byte_volume: number;
}

export interface HierNode {
  depth: number;
  label: string;
  class?: string;
  protocol?: string;
  stats: HierStats;
}

export interface Channel {
  freq_hz: number;
  occupied_bw_hz: number;
  dominant_class: string;
  burst_count: number;
  airtime_sec: number;
  duty_cycle: number;
  occupancy_pct: number;
  burst_rate_per_min: number;
  median_power_dbfs: number;
  median_flatness: number;
  timeline?: { t_sec: number; power_dbfs: number; occupied: boolean; burst_count: number }[];
  period_sec: number;
  period_confidence: number;
}

export interface Emitter {
  id: number;
  label: string;
  total_airtime_sec: number;
  first_sec: number;
  last_sec: number;
  hop_set?: number[];
}

export interface Conversation {
  id: number;
  emitter_ids: number[];
  kind: string;
  correlation: number;
  exchanges: number;
}

export interface Anomaly {
  severity: string;
  kind: string;
  freq_hz?: number;
  emitter_id?: number;
  detail: string;
}

export interface Scene {
  source: string;
  center_hz: number;
  span_hz: number;
  window_sec: number;
  noise_floor_dbfs: number;
  bursts?: unknown[];
  channels?: Channel[];
  emitters?: Emitter[];
  conversations?: Conversation[];
  hierarchy: { total: HierStats; nodes?: HierNode[] };
  anomalies?: Anomaly[];
}

export interface AnalyzeResponse {
  scene: Scene;
  frame_count: number;
  frames?: string;
}

export interface AnalyzerInfo {
  name: string;
  synopsis: string;
  depends_on?: string[];
}

async function asError(res: Response): Promise<never> {
  const text = await res.text().catch(() => "");
  let msg = text;
  try {
    msg = (JSON.parse(text) as { error?: string }).error ?? text;
  } catch {
    /* not JSON */
  }
  throw new Error(`${res.status}: ${msg}`);
}

export async function analyzers(): Promise<AnalyzerInfo[]> {
  const res = await fetch("/api/v1/rfscope/analyzers");
  if (!res.ok) return asError(res);
  const body = (await res.json()) as { analyzers: AnalyzerInfo[] };
  return body.analyzers ?? [];
}

export async function analyze(form: FormData): Promise<AnalyzeResponse> {
  const res = await fetch("/api/v1/rfscope/analyze", { method: "POST", body: form });
  if (!res.ok) return asError(res);
  return (await res.json()) as AnalyzeResponse;
}
