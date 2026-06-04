// TypeScript mirrors of the siglab engine's JSON output (internal/siglab)
// and the api package's wire DTOs (internal/api/handlers_siglab.go). Field
// names match the Go json tags exactly.

export interface IQPoint {
  i: number;
  q: number;
}

export interface IQTaps {
  decimated_iq: IQPoint[];
  soft_samples: number[];
  decimated_rate_hz: number;
  stride: number;
}

export interface LockInfo {
  frequency_hz: number;
  fields: Record<string, unknown>;
}

export interface GrantRecord {
  offset_sec: number;
  group_id: number;
  source_id: number;
  channel_id: number;
  channel_num: number;
  timeslot: number;
  frequency_hz: number;
  encrypted: boolean;
  emergency: boolean;
}

export interface EventRecord {
  seq: number;
  offset_sec: number;
  kind: string;
  fields: Record<string, unknown>;
}

export interface SignalQuality {
  symbol_cardinality: number;
  symbol_histogram: number[];
  symbol_histogram_pct: number[];
  iq_gain_imbalance_db: number;
  iq_phase_imbalance_deg: number;
  iq_image_rejection_db: number;
  iq_observed: boolean;
  decode_error_rate_per_ksym: number;
}

export interface RailStat {
  label: string;
  n: number;
  mean: number;
  std: number;
  p10: number;
  p50: number;
  p90: number;
}

export interface SpreadStat {
  label: string;
  steady_std: number;
  steady_n: number;
  trans_std: number;
  trans_n: number;
  ratio: number;
}

export interface SoftEye {
  soft_samples: number;
  signed_mean_dc: number;
  std_dev: number;
  mean_abs: number;
  max_abs: number;
  magnitude_histogram: number[];
  per_decided_symbol: RailStat[];
  true_outer_rail: RailStat[];
  outer_spread: SpreadStat[];
  verdict: string;
}

export interface ReceiverState {
  time_sec: number;
  cqpsk: boolean;
  afc_hz_est: number;
  agc_level: number;
  agc_target: number;
  mm_mu: number;
  mm_sps: number;
  dda_active: boolean;
  slicer_levels: number[];
  slicer_thresholds: number[];
  carrier_hz_est?: number;
  gardner_mu?: number;
  gardner_sps?: number;
  cqpsk_agc_gain?: number;
  cma_error?: number;
}

export interface SyncStat {
  variant: string;
  rotation: number;
  best_dist: number;
  best_pos: number;
  hits: number;
}

export interface SyncLandscape {
  sync_len: number;
  tolerance: number;
  stats: SyncStat[];
  winner_variant: string;
  winner_rotation: number;
  winner_hits: number;
  modal_spacing: number;
}

export interface FECStat {
  stage: string;
  frames: number;
  clean: number;
  corrected: number;
  uncorrectable: number;
  crc_pass: number;
  crc_fail: number;
}

// Detail is protocol-specific; we keep the well-known fields optional so the
// dashboard can render whatever the engine attached.
export interface Detail {
  // generic ProtocolDetail
  sync?: SyncLandscape;
  fec?: FECStat[];
  // P25P1Detail
  dibit_histogram?: number[];
  rotations?: { best_dist: number; best_pos: number; hits: number }[];
  winning_rotation?: number;
  winning_hits?: number;
  soft_eye?: SoftEye;
  receiver_states?: ReceiverState[];
  cc_stats?: Record<string, number>;
  [k: string]: unknown;
}

export interface Result {
  source: string;
  protocol: string;
  sample_rate_hz: number;
  pipeline_rate_hz: number;
  tune_hz: number;
  duration_sec: number;
  total_samples: number;
  symbols: number;
  effective_baud: number;
  expected_baud: number;
  baud_deviation_pct: number;
  locked: boolean;
  lock_latency_sec: number;
  lock?: LockInfo;
  grants: GrantRecord[];
  events: EventRecord[];
  event_counts: Record<string, number>;
  decode_errors: Record<string, number>;
  signal?: SignalQuality;
  detail?: Detail;
  verdict?: { pass: boolean; failures?: string[] };
  iq_taps?: IQTaps;
}

export interface CandidateScore {
  protocol: string;
  locked: boolean;
  lock_latency_sec: number;
  sync_hits: number;
  sync_variant: string;
  modal_spacing: number;
  fec_pass_rate: number;
  score: number;
}

export interface IdentifyResult {
  source: string;
  sample_rate_hz: number;
  winner: string;
  confidence: number;
  inconclusive: boolean;
  candidates: CandidateScore[];
}

export interface CaptureDTO {
  id: string;
  name: string;
  format: string;
  sample_rate_hz: number;
  size: number;
  created_at: string;
}

export interface JobDTO {
  id: string;
  capture_id: string;
  state: "running" | "done" | "error";
  error?: string;
  events: number;
  created_at: string;
  result?: Result;
  metadata?: unknown;
  has_iq: boolean;
}

export interface ProtocolsDTO {
  protocols: string[];
  fixtures: string[];
  formats: string[];
}

// RunConfig mirrors siglabRunConfig (the engine knobs).
export interface RunConfig {
  protocol: string;
  sample_rate_hz?: number;
  format?: string;
  frequency_hz?: number;
  tune_hz?: number;
  auto_tune?: boolean;
  conjugate?: boolean;
  iq_correct?: boolean;
  collect_iq_diag?: boolean;
  capture_iq?: boolean;
  capture_iq_max_points?: number;
  demod_mode?: string;
  nid_search_span?: number;
  enable_dda?: boolean;
  enable_adaptive_slicer?: boolean;
  collect_receiver_state?: boolean;
}

export interface SynthRequest {
  protocol: string;
  format: string;
  modulation?: string;
  snr_db?: number;
  freq_offset_hz?: number;
  freq_drift_hz_per_sec?: number;
  dc_offset?: number;
  iq_gain_imbalance?: number;
  iq_phase_skew_rad?: number;
  seed?: number;
  multipath?: string;
}
