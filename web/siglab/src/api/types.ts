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
  // Aligned symbol series for the Symbol-scope viz. soft is aligned
  // index-for-index with dibits (empty on soft-less paths, e.g. CQPSK).
  symbol_dibits?: number[];
  symbol_soft?: number[];
  symbol_cardinality?: number;
  // Per-symbol differential phase (radians) of the complex constellation —
  // the π/4-DQPSK rotation signal. Present only on the CQPSK path.
  diff_phase?: number[];
}

// PSDResult is the server-computed Welch power spectrum from
// GET /jobs/{id}/psd: FFT-shifted dBFS bins (bin 0 = -sample_rate_hz/2).
export interface PSDResult {
  sample_rate_hz: number;
  fft_size: number;
  bins: number[];
}

// OccupancyResult is the server-computed spectral-occupancy metric set from
// GET /jobs/{id}/occupancy, derived from the same Welch spectrum as the PSD.
export interface OccupancyResult {
  sample_rate_hz: number;
  fft_size: number;
  channel_bw_hz: number;
  adj_offset_hz: number;
  adj_bw_hz: number;
  occupancy: {
    occupied_bandwidth_hz: number;
    channel_power_dbfs: number;
    acpr_upper_db: number;
    acpr_lower_db: number;
    spectral_flatness: number;
  };
}

// SpectrogramResult is the server-computed STFT from GET /jobs/{id}/spectrogram:
// z[frame][bin] dBFS, each row FFT-shifted (bin 0 = -sample_rate_hz/2).
export interface SpectrogramResult {
  sample_rate_hz: number;
  fft_size: number;
  hop: number;
  frames: number;
  z: number[][];
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
  demod?: DemodMetrics;
}

// DemodMetrics is the demodulator-quality summary (EVM + estimated SNR) the
// P25 deep path derives from the recovered soft symbols. Mirrors
// siglab.DemodMetrics.
export interface DemodMetrics {
  modulation: string;
  evm_pct: number;
  snr_estimate_db: number;
  symbols_analyzed: number;
  // VSA decomposition (see siglab.DemodMetrics). The phase/IQ/origin fields
  // are populated on the CQPSK path only; arrays are omitted when empty.
  peak_evm_pct: number;
  mag_err_pct: number;
  phase_err_deg: number;
  carrier_freq_error_hz: number;
  iq_gain_imbalance_db: number;
  quadrature_error_deg: number;
  origin_offset_pct: number;
  evm_trace?: number[];
  error_vector_spectrum?: number[];
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

// PDURecord is one dissected data/signaling PDU (mirrors siglab.PDURecord).
export interface PDURecord {
  seq: number;
  offset_sec: number;
  protocol: string;
  opcode: number;
  opcode_name: string;
  mfid?: number;
  nac?: number;
  source_id?: number;
  dest_id?: number;
  talkgroup?: number;
  fields?: Record<string, unknown>;
  raw_hex: string;
  crc_ok: boolean;
  fec_metric: number;
  dibit_start: number;
  dibit_len: number;
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
  pdus?: PDURecord[];
  pdus_truncated?: boolean;
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

// BlindEstimate is the offline blind symbol-rate / modulation estimate
// (mirrors blind.BlindEstimate). ReferenceMatch is one scored reference-DB
// candidate (mirrors sigref.Match).
export interface BlindEstimate {
  symbol_rate_hz: number;
  symbol_rate_conf: number;
  symbol_rate_cands?: number[];
  mod_class: string;
  levels: number;
  occupied_bw_hz: number;
  method: string;
}

export interface ReferenceMatch {
  entry: {
    protocol: string;
    display_name: string;
    mod_class: string;
    mod_labels: string[];
    symbol_rate_hz: number;
    channel_bw_hz: number;
    levels: number;
    decodable: boolean;
  };
  score: number;
  why: string;
}

export interface IdentifyResult {
  source: string;
  sample_rate_hz: number;
  winner: string;
  confidence: number;
  inconclusive: boolean;
  candidates: CandidateScore[];
  // Offline-signal-ID fallback, populated only when inconclusive.
  blind?: BlindEstimate;
  reference_matches?: ReferenceMatch[];
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

// CaptureDevice mirrors api.SpectrumDevice — an SDR available to record from.
export interface CaptureDevice {
  serial: string;
  driver: string;
  product?: string;
  role: string;
  center_hz: number;
  sample_rate_hz: number;
}

// CaptureRequest is the body of POST /api/v1/siglab/capture.
export interface CaptureRequest {
  serial: string;
  seconds: number;
  format: string;
  protocol?: string;
  source?: string;
}

// CaptureResponse is returned by a successful live capture.
export interface CaptureResponse {
  capture: CaptureDTO;
  metadata: unknown;
  download_url: string;
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
  collect_pdus?: boolean;
  demod_mode?: string;
  nid_search_span?: number;
  enable_dda?: boolean;
  enable_adaptive_slicer?: boolean;
  collect_receiver_state?: boolean;
}

// WidebandRequest is the body of POST /api/v1/siglab/wideband. center_hz is the
// absolute tuner center of the wideband grab; the survey maps detected carriers
// to absolute frequencies from it.
export interface WidebandRequest {
  capture_id: string;
  center_hz?: number;
  sample_rate_hz?: number;
  format?: string;
  fft_size?: number;
  channel_rate_hz?: number;
  peak_threshold_db?: number;
  tuner_strategy?: string;
  conjugate?: boolean;
  iq_correct?: boolean;
}

// SurveySpectrum is the Welch-averaged band power spectrum (dBFS), FFT-shifted
// so bins[0] is center_hz - sample_rate_hz/2. Mirrors siglab.SurveySpectrum.
export interface SurveySpectrum {
  center_hz: number;
  sample_rate_hz: number;
  bins: number[];
}

// WidebandCarrier is one surveyed carrier. role is "control" | "voice" | "".
export interface WidebandCarrier {
  offset_hz: number;
  freq_hz: number;
  snr_db: number;
  protocol: string;
  confidence: number;
  inconclusive: boolean;
  locked: boolean;
  role: string;
  color_code?: number;
  system_id?: number;
  grants?: GrantRecord[];
  score: number;
  // Reference-DB naming for a carrier that did not decode (protocol is blank).
  blind?: BlindEstimate;
  reference_matches?: ReferenceMatch[];
}

// WidebandSystem clusters carriers into one trunked-system verdict.
export interface WidebandSystem {
  protocol: string;
  tier3: boolean;
  system_id?: number;
  color_code?: number;
  control_count: number;
  voice_count: number;
  lo_hz: number;
  hi_hz: number;
  carrier_idx: number[];
  verdict: string;
}

// WidebandResult mirrors siglab.WidebandResult.
export interface WidebandResult {
  source: string;
  sample_rate_hz: number;
  center_hz: number;
  channel_rate_hz: number;
  strategy: string;
  spectrum?: SurveySpectrum;
  detected_count: number;
  carriers: WidebandCarrier[];
  systems: WidebandSystem[];
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
