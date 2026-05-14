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
  // RuntimeDTO is large and changes shape as the daemon grows. Read
  // unknown fields lazily.
  [key: string]: unknown;
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
