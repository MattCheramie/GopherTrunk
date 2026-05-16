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
}

export interface GrantDTO {
  system: string;
  protocol: string;
  group_id: number;
  source_id?: number;
  frequency_hz: number;
  channel_id?: number;
  channel_number?: number;
  encrypted?: boolean;
  emergency?: boolean;
  data_call?: boolean;
}

export interface ActiveCallDTO {
  grant: GrantDTO;
  talkgroup?: TalkgroupDTO;
  device_serial: string;
  started_at: string;
  ended_at?: string;
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
  emergency?: boolean;
  data_call?: boolean;
  device_serial?: string;
  started_at: string;
  ended_at?: string;
  duration_ms?: number;
  end_reason?: string;
  talkgroup_alpha?: string;
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
  // ConfigPath is non-empty when the daemon was started with a
  // -config file. The SPA renders the Settings panel as editable
  // only when this is set; an empty value means PATCH /api/v1/settings
  // returns 503 and edits would be lost.
  config_path?: string;
  // StartupWarnings are the non-fatal observations the daemon
  // collected during NewDaemon. The Dashboard pins them until the
  // operator dismisses them.
  startup_warnings?: string[];
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
