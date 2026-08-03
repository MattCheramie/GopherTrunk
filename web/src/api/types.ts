// DTOs that mirror internal/tui/client/types.go. Only the shapes the
// SPA currently consumes are typed here; the daemon emits richer
// objects that we forward through `unknown` until a panel needs them.

export interface Health {
  status: string;
  version?: string;
  now: string;
  active_calls?: number;
  pool_attached_count?: number;
  pool_total_count?: number;
}

export interface Version {
  version: string;
}

export interface Mutations {
  allow_mutations: boolean;
}

export interface SystemDTO {
  name: string;
  protocol: string;
  control_channels?: number[];
  wacn?: number;
  system_id?: number;
  rfss?: number;
  site?: number;
  // Active DMR Tier III LCN→frequency band plan (configured or learned),
  // surfaced so the Systems panel can show how voice grants resolve. (#638)
  dmr_band_plan?: DMRBandPlanDTO;
  // Live adjacent (neighbour) sites decoded from the control channel (P25
  // Adjacent Site Status Broadcast). Empty/absent until one is heard.
  neighbors?: NeighborDTO[];
  // Decoded P25 IDEN_UP frequency-band table (channel id → base/spacing/offset),
  // overlaid live from the same topology snapshot as neighbors. Empty/absent
  // until an IDEN_UP is heard. (#814)
  frequency_bands?: BandPlanSlotDTO[];
}

// NeighborDTO mirrors api.NeighborDTO — one adjacent site with resolved
// downlink/uplink frequencies (zero when the band plan was not yet known).
export interface NeighborDTO {
  rfss: number;
  site: number;
  channel_id?: number;
  channel_number?: number;
  downlink_hz?: number;
  uplink_hz?: number;
  rfss_hex?: string;
  site_hex?: string;
}

// BandPlanSlotDTO mirrors api.BandPlanSlotDTO — one P25 IDEN_UP frequency-band
// entry (channel id → downlink base, spacing, and signed transmit offset).
export interface BandPlanSlotDTO {
  channel_id: number;
  base_hz: number;
  spacing_hz: number;
  bandwidth_hz?: number;
  tx_offset_hz?: number;
  access_tdma?: boolean;
}

// DMRBandPlanDTO mirrors api.DMRBandPlanDTO — exactly one of linear/table.
export interface DMRBandPlanDTO {
  linear?: { base_hz: number; spacing_hz: number; offset?: number };
  table?: { lcn: number; freq_hz: number }[];
}

export interface TalkgroupDTO {
  id: number;
  alpha_tag?: string;
  description?: string;
  tag?: string;
  group?: string;
  mode?: string;
  priority?: number;
  lockout?: boolean;
  scan?: boolean;
  // discovered marks an auto-learned entry (Tag == "Discovered"). The UI badges
  // these and offers a "hide auto-discovered" filter to collapse the phantom
  // radio-ID entries the operator doesn't want cluttering the list.
  discovered?: boolean;
}

// RIDDTO mirrors api.RIDDTO. `configured` distinguishes a row backed
// by the operator's rid_alias_file (where alias / tag / owner are
// authoritative) from a row that only exists because the affiliation
// tracker observed it over the air. Live fields (last_seen,
// last_talkgroup, talker_alias, call_count) are zero/empty when the
// radio has not been seen since the daemon started.
export interface RIDDTO {
  id: number;
  alias?: string;
  description?: string;
  tag?: string;
  group?: string;
  owner?: string;
  priority?: number;
  lockout?: boolean;
  watch?: boolean;
  icon?: string;
  configured?: boolean;
  system?: string;
  protocol?: string;
  last_talkgroup?: number;
  talker_alias?: string;
  talker_alias_at?: string;
  talker_alias_unreliable?: boolean;
  call_count?: number;
  first_seen?: string;
  last_seen?: string;
}

export interface GrantDTO {
  system: string;
  protocol: string;
  group_id: number;
  source_id?: number;
  frequency_hz: number;
  channel_id?: number;
  channel_number?: number;
  // TDMA timeslot, 1-based: 1 = TS1, 2 = TS2 (DMR Tier III). Absent /
  // 0 for non-slotted protocols, where frequency alone identifies the
  // call.
  timeslot?: number;
  encrypted?: boolean;
  emergency?: boolean;
  data_call?: boolean;
  // P25 encryption metadata recovered from the in-call signalling.
  // Meaningful only when encrypted is true; on Phase 1 they arrive
  // after the grant via a `call.encryption` SSE event.
  algorithm_id?: number;
  key_id?: number;
}

export interface ActiveCallDTO {
  grant: GrantDTO;
  talkgroup?: TalkgroupDTO;
  device_serial: string;
  started_at: string;
  ended_at?: string;
  // true when a voice tuner is decoding this call; false for a call the
  // control channel announced but no tuner is following (every voice device
  // busy). An unfollowed call has an empty device_serial.
  following?: boolean;
}

export interface DeviceDTO {
  serial: string;
  driver: string;
  tuner?: string;
  role?: string;
  attached?: boolean;
  gain?: string;
  ppm?: number;
  bias_tee?: boolean;
}

export interface AudioStatusDTO {
  backend_enabled: boolean;
  sample_rate: number;
  volume: number;
  muted: boolean;
  recording_enabled: boolean;
  drops_total: number;
}

// HuntStatus mirrors api.HuntStatus — the live system-discovery run snapshot
// from GET /api/v1/hunt. `system` carries the full discovered map when ready.
export interface HuntStatus {
  run_id: number;
  state: string;
  running: boolean;
  mode?: string; // "hunt" | "survey"
  phase?: string;
  detail?: string;
  sites: number;
  talkgroups: number;
  system_name?: string;
  signals?: DetectedSignal[];
  error?: string;
  system?: DiscoveredSystem;
  // Per-capture decode results from the live run, so the Hunt panel can
  // show what each surveyed candidate decoded to (or why it was skipped).
  reports?: CaptureReport[];
  gain_recommendations?: GainRecommendation[];
  gain_note?: string;
}

// GainRecommendation mirrors hunt.GainRecommendation — the best front-end gain
// found for one control channel by an auto-gain sweep.
export interface GainRecommendation {
  freq_hz: number;
  best_gain_tenth_db: number;
  best_error_rate: number;
  locked: boolean;
}

// CaptureReport mirrors hunt.CaptureReport — one candidate's decode outcome
// during a live hunt run.
export interface CaptureReport {
  path?: string;
  protocol?: string;
  confidence?: number;
  locked?: boolean;
  control_hz?: number;
  talkgroups?: number;
  skipped?: boolean;
  skip_reason?: string;
  error?: string;
  // identity_note explains a locked P25 capture whose WACN/System ID are still
  // blank (e.g. the Network Status Broadcast never decoded in the dwell).
  identity_note?: string;
}

// DiscoveredSystem mirrors hunt.DiscoveredSystem — the map a completed hunt
// produced. Only the fields the Hunt panel renders are typed.
export interface DiscoveredSystem {
  name?: string;
  protocol?: string;
  state?: string;
  county?: string;
  confidence?: number;
  // P25 / generic identity decoded live from the control channel. Zero/absent
  // ⇒ not yet observed.
  wacn?: number;
  system_id?: number;
  nac?: number;
  sites?: DiscoveredSite[];
  // Talkgroups and the IDEN_UP band plan accumulate as the control channel is
  // decoded — the human-readable map an operator uses to program a radio.
  talkgroups?: DiscoveredTalkgroup[];
  band_plan?: BandPlanEntry[];
}

// DiscoveredTalkgroup mirrors hunt.DiscoveredTalkgroup — one talkgroup observed
// on the control channel, with its activity counter.
export interface DiscoveredTalkgroup {
  dec: number;
  hex: string;
  encrypted?: boolean;
  count: number;
}

// BandPlanEntry mirrors hunt.BandPlanEntry — one P25 IDEN_UP band-plan slot:
// the base/spacing math that turns a (channel id, channel number) into a
// downlink frequency. These are the "voice-channel bands" of the system.
export interface BandPlanEntry {
  channel_id: number;
  base_hz: number;
  spacing_hz: number;
  bandwidth_hz?: number;
  tx_offset_hz?: number;
}

export interface DiscoveredSite {
  rfss?: number;
  site_id?: number;
  site_name?: string;
  county?: string;
  control_channels?: { frequency_hz: number; is_control?: boolean; confidence?: number }[];
  secondary?: number[];
  neighbors?: NeighborRef[];
  // Distinct voice/traffic-channel frequencies (Hz) granted on this site.
  voice_channels?: number[];
}

// HuntRRReport mirrors api.HuntRRReport — the RadioReference cross-reference of
// a discovered system: duplicate-system hints + a frequency/talkgroup diff.
export interface HuntRRReport {
  hints?: DuplicateHint[];
  diff?: RRDiff;
  compared: number;
}

export interface DuplicateHint {
  sid: number;
  name: string;
  reason: string;
  confidence: number;
}

export interface RRDiff {
  sid: number;
  name: string;
  freq_offsets?: { discovered_hz: number; rr_hz: number; delta_hz: number }[];
  freqs_not_in_rr?: number[];
  talkgroups_not_in_rr?: number[];
}

// NeighborRef mirrors hunt.NeighborRef — an adjacent site advertised by the
// control channel. frequency_hz is set once the band plan / LCN resolver maps
// the channel to a downlink frequency.
export interface NeighborRef {
  rfss?: number;
  site?: number;
  channel_id?: number;
  channel_number?: number;
  frequency_hz?: number;
}

// DMRGrantObservedDTO mirrors api.DMRGrantObservedDTO — a raw DMR Tier III
// voice grant the control channel decoded (pre LCN resolution). (#638)
export interface DMRGrantObservedDTO {
  system: string;
  color_code: number;
  lcn: number;
  timeslot: number;
  group_id: number;
  source_id: number;
  cc_freq_hz: number;
  at: string;
}

// DMRBandPlanLearnedDTO mirrors api.DMRBandPlanLearnedDTO — the autoconfig
// learner's fitted band plan for a system. (#638)
export interface DMRBandPlanLearnedDTO {
  system: string;
  base_hz?: number;
  spacing_hz?: number;
  offset?: number;
  table?: { lcn: number; freq_hz: number }[];
  num_pairs: number;
  confidence: number;
  residual_hz?: number;
}

// DetectedSignal mirrors hunt.DetectedSignal — one classified carrier in a
// survey run.
export interface DetectedSignal {
  freq_hz: number;
  snr_db: number;
  occupied_bw_hz: number;
  class: string;
  confidence: number;
  baud_hz?: number;
  // Consolidated inventory naming (full-spectrum survey).
  name?: string;
  service?: string;
  purpose?: string;
  encrypted?: boolean;
  enc_type?: string;
  wideband?: boolean;
  trunking?: { protocol: string; locked: boolean; control_hz?: number; encrypted?: boolean; enc_type?: string };
  analog?: { active: boolean; ctcss_hz?: number; dcs_code?: string };
  pages?: { protocol: string; capcode: number; text: string }[];
}

// HuntCaptureRequest / HuntCaptureResult mirror the api types for the
// list-driven capture (POST /api/v1/hunt/capture).
export interface HuntCaptureRequest {
  freq_hz: number;
  seconds?: number;
  target?: "siglab" | "cryptolab";
}

export interface HuntCaptureResult {
  path: string;
  metadata_path?: string;
  sample_rate_hz: number;
  center_hz: number;
  samples: number;
  identify?: string;
  frames_path?: string;
  note?: string;
}

// HuntStartRequest mirrors api.HuntStartRequest (frequencies in MHz).
export interface HuntStartRequest {
  serial?: string;
  bands?: string[];
  candidates?: number[];
  no_sweep?: boolean;
  survey?: boolean;
  classify_only?: boolean;
  detect_wideband?: boolean;
  detect_encryption?: boolean;
  persist_survey?: boolean;
  resume?: boolean;
  auto_gain?: boolean;
  auto_gain_set?: string;
  max_dwell_seconds?: number;
  monitor_seconds?: number;
  protocol?: string;
  dwell_seconds?: number;
  name?: string;
  state?: string;
  county?: string;
  location?: string;
}

export interface ScannerStatusDTO {
  scan_mode: string;
  systems: SystemHuntStatusDTO[];
  conventional: ConvScannerStatusDTO;
  tg_scan_count: number;
  tg_total: number;
}

export interface SystemHuntStatusDTO {
  name: string;
  protocol: string;
  state: string;
  attempted_freq_hz?: number;
  attempt_index?: number;
  total_candidates?: number;
  locked_freq_hz?: number;
  locked_at?: string;
  nac?: number;
  last_failed_at?: string;
  backoff_ms?: number;
  last_grant_at?: string;
  // Live control-channel signal quality (clean/marginal/poor) and carrier offset
  // (Hz) for the locked CC. has_decode_health distinguishes a real 0 offset from
  // "no data yet"; decode_quality is empty until enough frames have decoded.
  decode_quality?: string;
  carrier_offset_hz?: number;
  has_decode_health?: boolean;
  // Locked carrier's mean channel power in dBFS — the raw front-end level for
  // antenna/LNA aiming, a different axis from decode_quality. has_signal
  // distinguishes a real reading from "no data" (0 is ambiguous since a genuine
  // level is negative). Present as soon as the CC locks.
  signal_dbfs?: number;
  has_signal?: boolean;
}

export interface ConvScannerStatusDTO {
  enabled: boolean;
  state?: string;
  device_serial?: string;
  cursor_index?: number;
  channels: ConvChannelStatusDTO[];
}

export interface ConvChannelStatusDTO {
  index: number;
  label: string;
  frequency_hz: number;
  mode: string;
  active: boolean;
  locked_out?: boolean;
  last_break_at?: string;
}

export interface CallRow {
  id: number;
  system: string;
  protocol: string;
  group_id: number;
  source_id?: number;
  frequency_hz: number;
  encrypted?: boolean;
  algorithm_id?: number;
  key_id?: number;
  emergency?: boolean;
  data_call?: boolean;
  // TDMA timeslot, 1-based (1 = TS1, 2 = TS2; absent for non-slotted).
  timeslot?: number;
  device_serial?: string;
  started_at: string;
  ended_at?: string;
  duration_ms?: number;
  end_reason?: string;
  talkgroup_alpha?: string;
}

// CallEncryptionEvent is the SSE payload published as `call.encryption`
// when the daemon recovers an Encryption Sync on an active call (P25
// Phase 1 — Phase 2 carries the values on the original grant). The
// SPA matches device_serial to the active-call row and patches its
// algorithm_id / key_id in flight.
export interface CallEncryptionEvent {
  device_serial: string;
  system?: string;
  protocol?: string;
  group_id?: number;
  algorithm_id: number;
  key_id: number;
  at: string;
}

export interface RuntimeDTO {
  version?: string;
  api?: {
    http_addr?: string;
    grpc_addr?: string;
    auth_mode?: string;
    cors_allowed_origins?: string[];
  };
  audio?: AudioStatusDTO;
  log_level?: string;
  log_format?: string;
  metrics_enabled?: boolean;
  // ConfigPath is non-empty when the daemon was started with a
  // -config file. The SPA renders the Settings panel as editable
  // only when this is set; an empty value means PATCH /api/v1/settings
  // returns 503 and edits would be lost.
  config_path?: string;
  // StartupWarnings are the non-fatal observations the daemon
  // collected during NewDaemon. The Dashboard pins them until the
  // operator dismisses them.
  startup_warnings?: string[];
  // HiddenTabs lists the navigation tab keys switched off via
  // web.tabs in config. The App filters them out of the nav strip.
  hidden_tabs?: string[];
  // IDBase selects how identity numbers (WACN, System ID, NAC, RFSS,
  // Site) are rendered: "hex" (default) or "dec". From web.id_base config.
  id_base?: "hex" | "dec";
  // CryptolabConsole is true when the daemon serves the Crypto Lab SPA at
  // /cryptolab/ (a -tags cryptolab build). Gates the Crypto Lab nav link.
  cryptolab_console?: boolean;
  // SiglabConsole is true when the daemon serves the Signal Lab SPA at
  // /siglab/. Gates the Signal Lab nav link.
  siglab_console?: boolean;
  // RFScopeConsole is true when the daemon serves the RF Scope SPA at
  // /rfscope/. Gates the RF Scope nav link.
  rfscope_console?: boolean;
  // RuntimeDTO is large and changes shape as the daemon grows. Read
  // unknown fields lazily.
  [key: string]: unknown;
}

// SettingsPatch mirrors the daemon's PATCH /api/v1/settings body.
// Every field is optional; the daemon leaves unspecified fields
// alone. Use snake_case keys to match the wire format directly.
export interface SettingsPatch {
  log_level?: string;
  log_format?: string;
  api_http_addr?: string;
  api_grpc_addr?: string;
  api_auth_mode?: string;
  audio_enabled?: boolean;
  audio_device?: string;
  audio_volume?: number;
  audio_muted?: boolean;
  audio_buffer_ms?: number;
  recordings_dir?: string;
  recordings_sample_rate?: number;
  recordings_write_raw?: boolean;
  recordings_skip_encrypted?: boolean;
  retention_call_log_days?: number;
  retention_files_days?: number;
  retention_interval?: string;
  sdr_sample_rate?: number;
  scanner_scan_mode?: string;
  scanner_manual_tune_enabled?: boolean;
  scanner_cc_hunt_enabled?: boolean;
  scanner_cc_hunt_dwell_ms?: number;
  scanner_cc_hunt_backoff_ms?: number;
  scanner_cc_hunt_max_backoff_ms?: number;
  storage_path?: string;
  storage_cc_cache_file?: string;
  metrics_enabled?: boolean;
}

export interface SettingsResponse {
  applied: string[];
  restart_required: string[];
  config_path?: string;
  runtime: RuntimeDTO;
}

// ConfigFileInfo describes one discovered config file (GET
// /api/v1/config/files), used by the Settings config-file picker.
export interface ConfigFileInfo {
  path: string;
  dir: string;
  name: string;
  size: number;
  modified: string; // RFC3339
  valid: boolean;
  error?: string;
}

export interface ConfigListResponse {
  dirs: string[];
  files: ConfigFileInfo[];
}

// ConfigActivateResponse is the outcome of loading/hot-swapping the active
// config file (POST /api/v1/config/activate).
export interface ConfigActivateResponse {
  path: string;
  mode: "reload" | "restart";
  applied?: string[];
  restart_required?: string[];
  restarting?: boolean;
}

// ParsedSystemDTO is one row in an import preview.
export interface ParsedSystemDTO {
  name: string;
  protocol: string;
  site_count: number;
  talkgroup_count: number;
  source_path?: string;
  location?: string;
  county?: string;
  sysid?: string;
  wacn?: string;
  system_type?: string;
}

export interface ImportPreview {
  id: string;
  systems: ParsedSystemDTO[];
}

export interface ImportResult {
  systems_added: string[];
  systems_replaced?: string[];
  csv_paths?: string[];
  config_path?: string;
}

export interface EventDTO {
  kind: string;
  timestamp: string;
  payload?: unknown;
}

export interface ToneAlertDTO {
  profile: string;
  alpha_tag?: string;
  system?: string;
  group_id?: number;
  device_serial: string;
  matched_at: string;
  frequencies_hz: number[];
}
