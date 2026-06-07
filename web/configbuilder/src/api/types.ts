// GopherTrunk config types. The daemon marshals config.Config to JSON
// using Go field names (the config structs carry yaml tags, not json
// tags), so these keys are the Go exported field names, NOT snake_case.
// Sections without a bespoke editor are typed as unknown and round-trip
// untouched (seeded from GET /api/v1/config/defaults).

export interface LogConfig {
  Level: string;
  Format: string;
  MessageLog?: unknown;
}

export interface DiagnosticsConfig {
  VerboseErrors: boolean;
}

export interface RadioReferenceConfig {
  APIKey: string;
  Username: string;
  Password: string;
}

export interface APIConfig {
  HTTPAddr: string;
  GRPCAddr: string;
  AllowMutations: boolean;
  Rigctld: string;
  Auth?: unknown;
  CORS?: unknown;
  TLSCert: string;
  TLSKey: string;
}

export interface DeviceChannelConfig {
  FrequencyHz: number;
  System: string;
}

export interface DeviceConfig {
  Serial: string;
  Role: string;
  PPM: number;
  Gain: string;
  BiasTee: boolean;
  CenterFreqHz: number;
  Channels?: DeviceChannelConfig[] | null;
  TunerStrategy?: string;
  VoiceTaps?: number;
  // Other device flags (BlogV4, IQCorrect, IQInvert, …) round-trip via the
  // index signature and are reachable through AdvancedJSON.
  [k: string]: unknown;
}

export interface RTLTCPConfig {
  Addr: string;
  Serial: string;
  Role: string;
  PPM: number;
  Gain: string;
  BiasTee: boolean;
  ConnectTimeoutMs: number;
}

export interface SoapyRemoteConfig {
  Addr: string;
  Driver: string;
  Args: string;
  Serial: string;
  Role: string;
  Format: string;
  StreamProtocol: string;
  PPM: number;
  Gain: string;
  BiasTee: boolean;
  ConnectTimeoutMs: number;
}

export interface SDRConfig {
  SampleRate: number;
  Devices: DeviceConfig[];
  RTLTCP?: RTLTCPConfig[] | null;
  SoapyRemote?: SoapyRemoteConfig[] | null;
  [k: string]: unknown;
}

export interface StorageConfig {
  Path: string;
  CCCacheFile: string;
}

export interface RecordingsConfig {
  Dir: string;
  SampleRate: number;
  WriteRaw: boolean;
  Equalizer?: unknown;
}

export interface MetricsConfig {
  Enabled: boolean;
}

export interface RetentionConfig {
  CallLogDays: number;
  LogDays: number;
  FilesDays: number;
  Interval: string;
}

export interface ScannerConfig {
  ScanMode: string;
  CCHunt?: unknown;
  Conventional?: unknown;
  [k: string]: unknown;
}

export interface AudioConfig {
  Enabled: boolean;
  Device: string;
  SampleRate: number;
  BufferMs: number;
  Volume: number;
  Muted: boolean;
}

export interface WebConfig {
  Tabs: Record<string, boolean> | null;
}

export interface P25BandPlanEntry {
  ChannelID: number;
  BaseHz: number;
  SpacingHz: number;
  TxOffsetHz: number;
  BandwidthHz: number;
}

export interface DMRLinearBandPlan {
  BaseHz: number;
  SpacingHz: number;
  Offset: number;
}

export interface DMRBandPlanTableEntry {
  LCN: number;
  FreqHz: number;
}

export interface DMRBandPlan {
  Linear: DMRLinearBandPlan | null;
  Table: DMRBandPlanTableEntry[] | null;
}

export interface EncryptionKey {
  KeyID: number;
  Algorithm: string;
  Key: string;
}

export interface SystemConfig {
  Name: string;
  Protocol: string;
  ControlChannels: number[] | null;
  TalkgroupFile: string;
  RIDAliasFile?: string;
  P25BandPlan?: P25BandPlanEntry[] | null;
  DMRBandPlan?: DMRBandPlan | null;
  EncryptionKeys?: EncryptionKey[] | null;
  // Long-tail protocol decoder knobs (TETRA*, LTR*, P25Phase1/2*, NXDN*,
  // EDACS*, MPT1327*, Motorola*, DStarFEC, DMRInterleavedVoice) round-trip
  // through this index signature and are edited via AdvancedJSON.
  [k: string]: unknown;
}

export interface TrunkingConfig {
  CallTimeoutMs: number;
  Systems: SystemConfig[] | null;
  [k: string]: unknown;
}

export interface GTConfig {
  Log: LogConfig;
  SDR: SDRConfig;
  Trunking: TrunkingConfig;
  API: APIConfig;
  Storage: StorageConfig;
  Recordings: RecordingsConfig;
  Metrics: MetricsConfig;
  Retention: RetentionConfig;
  ToneOut: unknown;
  Scanner: ScannerConfig;
  Audio: AudioConfig;
  Broadcast: unknown;
  Baseband: unknown;
  Paging: unknown;
  APRS: unknown;
  AIS: unknown;
  DSC: unknown;
  MDC1200: unknown;
  ADSB: unknown;
  M17: unknown;
  Web: WebConfig;
  Diagnostics: DiagnosticsConfig;
  RadioReference: RadioReferenceConfig;
  [k: string]: unknown;
}

// ---- API wire shapes -----------------------------------------------------

export interface ConfigFileInfo {
  path: string;
  dir: string;
  name: string;
  size: number;
  modified: string;
  valid: boolean;
  error?: string;
}

export interface ConfigListResponse {
  dirs: string[];
  files: ConfigFileInfo[] | null;
}

export interface ValidationError {
  section: string;
  message: string;
}

export interface ValidationResult {
  ok: boolean;
  errors: ValidationError[] | null;
}

export interface ConfigLoadResponse {
  path: string;
  config: GTConfig;
  validation: ValidationResult;
  mtime: number;
}

export interface ConfigSaveResponse {
  path: string;
  mtime: number;
  talkgroup_csvs?: string[];
}

export interface TalkgroupCSVRow {
  decimal: number;
  alpha_tag?: string;
  description?: string;
  tag?: string;
  group?: string;
  mode?: string;
}

export interface DocLink {
  title: string;
  url: string;
  description: string;
}

export interface RRSearchHit {
  sid: number;
  name: string;
  type: string;
  county?: string;
  state?: string;
}

export interface RRSiteDetail {
  rfss?: number;
  site_number?: number;
  description?: string;
  county?: string;
  control_channels: number[] | null;
  frequencies: number[] | null;
}

export interface RRTalkgroupDetail {
  dec: number;
  alpha_tag?: string;
  description?: string;
  tag?: string;
  group?: string;
  mode?: string;
  encrypted?: boolean;
}

export interface RRFullSystem {
  sid: number;
  name: string;
  type: string;
  flavor?: string;
  protocol: string;
  system_id?: number;
  wacn?: number;
  nac?: number;
  city?: string;
  county?: string;
  state?: string;
  sites: RRSiteDetail[] | null;
  talkgroups: RRTalkgroupDetail[] | null;
}

export interface RRSystemResponse {
  system: RRFullSystem;
  config: SystemConfig;
  talkgroups: TalkgroupCSVRow[] | null;
}

export interface ParsedSystemDTO {
  name: string;
  protocol: string;
  sysid?: string;
  wacn?: string;
  system_type?: string;
  site_count: number;
  talkgroup_count: number;
  source_path?: string;
  control_channels?: number[];
  talkgroups?: RRTalkgroupDetail[];
}
