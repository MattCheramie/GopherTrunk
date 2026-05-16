// Package client is the TUI's network layer. It wraps the daemon's
// HTTP REST + SSE surfaces in typed methods so the bubbletea model
// can stay focused on rendering.
//
// DTO shapes mirror internal/api/types.go and internal/storage's
// CallRow, copied by value rather than imported so that the TUI
// stays a pure client of the wire protocol — `internal/tui` doesn't
// pull in the server-side packages.
package client

import (
	"encoding/json"
	"time"
)

// Health is the response shape of GET /api/v1/health.
type Health struct {
	Status string    `json:"status"`
	Now    time.Time `json:"now"`
}

// Version is the response shape of GET /api/v1/version.
type Version struct {
	Version string `json:"version"`
}

// SystemDTO mirrors api.SystemDTO.
type SystemDTO struct {
	Name            string   `json:"name"`
	Protocol        string   `json:"protocol"`
	ControlChannels []uint32 `json:"control_channels"`
	WACN            uint32   `json:"wacn,omitempty"`
	SystemID        uint16   `json:"system_id,omitempty"`
	RFSS            uint8    `json:"rfss,omitempty"`
	Site            uint8    `json:"site,omitempty"`

	// Per-protocol FEC opt-out surface (mirrors api.SystemDTO).
	// The TUI Settings panel renders these.
	TETRAColourCode        uint32  `json:"tetra_colour_code,omitempty"`
	TETRAChannel           string  `json:"tetra_channel,omitempty"`
	TETRAChannelCoding     string  `json:"tetra_channel_coding,omitempty"`
	LTRFCSMode             string  `json:"ltr_fcs_mode,omitempty"`
	LTRManchesterMode      string  `json:"ltr_manchester_mode,omitempty"`
	P25Phase2TrellisMode   string  `json:"p25_phase2_trellis_mode,omitempty"`
	P25Phase2RSMode        string  `json:"p25_phase2_rs_mode,omitempty"`
	P25Phase2ScramblerMode string  `json:"p25_phase2_scrambler_mode,omitempty"`
	NXDNViterbiMode        string  `json:"nxdn_viterbi_mode,omitempty"`
	NXDNDeviationHz        float64 `json:"nxdn_deviation_hz,omitempty"`
	EDACSBCHMode           string  `json:"edacs_bch_mode,omitempty"`
	MPT1327BCHMode         string  `json:"mpt1327_bch_mode,omitempty"`
	MPT1327CWSCTolerance   string  `json:"mpt1327_cwsc_tolerance,omitempty"`
	MotorolaBCHMode        string  `json:"motorola_bch_mode,omitempty"`
}

// TalkgroupDTO mirrors api.TalkgroupDTO.
type TalkgroupDTO struct {
	ID          uint32 `json:"id"`
	AlphaTag    string `json:"alpha_tag"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Group       string `json:"group,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Lockout     bool   `json:"lockout,omitempty"`
	Scan        bool   `json:"scan"`
}

// GrantDTO mirrors api.GrantDTO.
type GrantDTO struct {
	System        string `json:"system"`
	Protocol      string `json:"protocol"`
	GroupID       uint32 `json:"group_id"`
	SourceID      uint32 `json:"source_id"`
	FrequencyHz   uint32 `json:"frequency_hz"`
	ChannelID     uint8  `json:"channel_id,omitempty"`
	ChannelNumber uint16 `json:"channel_number,omitempty"`
	Encrypted     bool   `json:"encrypted,omitempty"`
	Emergency     bool   `json:"emergency,omitempty"`
	DataCall      bool   `json:"data_call,omitempty"`
}

// ActiveCallDTO mirrors api.ActiveCallDTO.
type ActiveCallDTO struct {
	Grant        GrantDTO      `json:"grant"`
	Talkgroup    *TalkgroupDTO `json:"talkgroup,omitempty"`
	DeviceSerial string        `json:"device_serial"`
	StartedAt    time.Time     `json:"started_at"`
	LastHeardAt  time.Time     `json:"last_heard_at"`
}

// CallRow mirrors storage.CallRow — the shape returned by GET
// /api/v1/calls/history.
type CallRow struct {
	ID             int64     `json:"id"`
	System         string    `json:"system"`
	Protocol       string    `json:"protocol"`
	GroupID        uint32    `json:"group_id"`
	SourceID       uint32    `json:"source_id"`
	FrequencyHz    uint32    `json:"frequency_hz"`
	Encrypted      bool      `json:"encrypted"`
	Emergency      bool      `json:"emergency"`
	DataCall       bool      `json:"data_call"`
	DeviceSerial   string    `json:"device_serial"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at,omitempty"`
	DurationMs     int64     `json:"duration_ms,omitempty"`
	EndReason      string    `json:"end_reason,omitempty"`
	TalkgroupAlpha string    `json:"talkgroup_alpha,omitempty"`
}

// HistoryFilter is the query-parameter shape for GET
// /api/v1/calls/history. Zero-valued fields are omitted from the
// outgoing query string.
type HistoryFilter struct {
	System    string
	GroupID   uint32
	Since     time.Time
	Until     time.Time
	Limit     int // 0 → server default (100)
	OnlyEnded bool
}

// Event is one decoded SSE event. Kind matches the events.Kind
// constants ("cc.locked", "grant", "call.start", …); Time is the
// envelope timestamp; Raw is the un-decoded JSON of the kind-
// specific payload, which the caller can decode into a typed
// payload as needed (Grant, Active, Tone, …).
type Event struct {
	Kind string
	Time time.Time
	Raw  json.RawMessage
}

// ScannerStatusDTO mirrors api.ScannerStatus — the unified scanner
// snapshot returned by GET /api/v1/scanner. Fields are zero-valued
// when the underlying subsystem isn't wired (e.g. no CC hunter →
// empty Systems list).
type ScannerStatusDTO struct {
	ScanMode            string                `json:"scan_mode"`
	Systems             []SystemHuntStatusDTO `json:"systems"`
	Conventional        ConvScannerStatusDTO  `json:"conventional"`
	TalkgroupScanCount  int                   `json:"tg_scan_count"`
	TalkgroupTotalCount int                   `json:"tg_total"`
}

// SystemHuntStatusDTO mirrors api.SystemHuntStatusDTO.
type SystemHuntStatusDTO struct {
	Name            string    `json:"name"`
	Protocol        string    `json:"protocol"`
	State           string    `json:"state"`
	AttemptedFreqHz uint32    `json:"attempted_freq_hz,omitempty"`
	AttemptIndex    int       `json:"attempt_index,omitempty"`
	TotalCandidates int       `json:"total_candidates,omitempty"`
	LockedFreqHz    uint32    `json:"locked_freq_hz,omitempty"`
	LockedAt        time.Time `json:"locked_at,omitempty"`
	NAC             uint16    `json:"nac,omitempty"`
	LastFailedAt    time.Time `json:"last_failed_at,omitempty"`
	BackoffMs       int       `json:"backoff_ms,omitempty"`
	LastGrantAt     time.Time `json:"last_grant_at,omitempty"`
}

// ConvScannerStatusDTO mirrors api.ConvScannerStatusDTO.
type ConvScannerStatusDTO struct {
	Enabled      bool                   `json:"enabled"`
	State        string                 `json:"state,omitempty"`
	DeviceSerial string                 `json:"device_serial,omitempty"`
	CursorIndex  int                    `json:"cursor_index,omitempty"`
	Channels     []ConvChannelStatusDTO `json:"channels"`
}

// ConvChannelStatusDTO mirrors api.ConvChannelStatusDTO.
type ConvChannelStatusDTO struct {
	Index       int       `json:"index"`
	Label       string    `json:"label"`
	FrequencyHz uint32    `json:"frequency_hz"`
	Mode        string    `json:"mode"`
	Active      bool      `json:"active"`
	LockedOut   bool      `json:"locked_out,omitempty"`
	LastBreakAt time.Time `json:"last_break_at,omitempty"`
}

// SDRStatus mirrors api.sdr.SDRStatus — the per-device payload
// returned by GET /api/v1/devices and embedded in sdr.attached /
// sdr.detached SSE events.
type SDRStatus struct {
	Driver       string `json:"driver"`
	Serial       string `json:"serial"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Product      string `json:"product,omitempty"`
	TunerName    string `json:"tuner_name,omitempty"`
	Role         string `json:"role"`
	Attached     bool   `json:"attached"`
	GainTenthDB  int    `json:"gain_tenth_db"`
	GainAuto     bool   `json:"gain_auto"`
	PPM          int    `json:"ppm"`
	BiasTee      bool   `json:"bias_tee"`
	Gains        []int  `json:"gains,omitempty"`
}

// Tone is the payload shape of a `tone.alert` event.
type Tone struct {
	Profile       string    `json:"profile"`
	AlphaTag      string    `json:"alpha_tag,omitempty"`
	System        string    `json:"system,omitempty"`
	GroupID       uint32    `json:"group_id,omitempty"`
	DeviceSerial  string    `json:"device_serial"`
	MatchedAt     time.Time `json:"matched_at"`
	FrequenciesHz []float64 `json:"frequencies_hz"`
}

// LockState is the payload of cc.locked / cc.lost events.
// FrequencyHz is always present; the other fields are protocol-
// specific.
type LockState struct {
	FrequencyHz uint32 `json:"FrequencyHz"`
	NAC         uint16 `json:"NAC,omitempty"`
	SystemID    uint32 `json:"SystemID,omitempty"`
	Repeater    string `json:"Repeater,omitempty"`
	MCC         uint16 `json:"MCC,omitempty"`
	MNC         uint16 `json:"MNC,omitempty"`
}

// HTTPError is returned by the REST methods on a non-2xx response.
// Status carries the HTTP code; Body carries up to a few hundred
// bytes of the response for the toast UI to show.
// RuntimeDTO mirrors api.RuntimeDTO. Consumed by the TUI's tabbed
// Settings inspector to render every configured feature, output,
// protocol surface, etc. — the touch-point inventory the operator
// expects to see in one place.
type RuntimeDTO struct {
	HTTPAddr       string `json:"http_addr,omitempty"`
	GRPCAddr       string `json:"grpc_addr,omitempty"`
	WSPath         string `json:"ws_path,omitempty"`
	SSEPath        string `json:"sse_path,omitempty"`
	MetricsPath    string `json:"metrics_path,omitempty"`
	AllowMutations bool   `json:"allow_mutations"`

	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`
	Version   string `json:"version,omitempty"`

	StorageDBPath  string `json:"storage_db_path,omitempty"`
	StorageCCCache string `json:"storage_cc_cache,omitempty"`

	RetentionCallLogDays int           `json:"retention_call_log_days"`
	RetentionFilesDays   int           `json:"retention_files_days"`
	RetentionInterval    time.Duration `json:"retention_interval_ns"`

	RecordingDir        string `json:"recording_dir,omitempty"`
	RecordingSampleRate int    `json:"recording_sample_rate"`
	RecordingWriteRaw   bool   `json:"recording_write_raw"`
	RecordingEQEnabled  bool   `json:"recording_eq_enabled"`
	RecordingEQTaps     int    `json:"recording_eq_taps,omitempty"`
	RecordingEQStepSize string `json:"recording_eq_step_size,omitempty"`

	AudioEnabled       bool     `json:"audio_enabled"`
	AudioDevice        string   `json:"audio_device,omitempty"`
	AudioSampleRate    int      `json:"audio_sample_rate"`
	AudioBufferMs      int      `json:"audio_buffer_ms"`
	AudioBackends      []string `json:"audio_backends"`
	AudioDisableFallbk bool     `json:"audio_disable_fallback"`

	SDRSampleRate int      `json:"sdr_sample_rate"`
	SDRBackends   []string `json:"sdr_backends"`

	ScannerScanMode          string `json:"scanner_scan_mode"`
	ScannerCCHuntEnabled     bool   `json:"scanner_cc_hunt_enabled"`
	ScannerCCHuntDwellMs     int    `json:"scanner_cc_hunt_dwell_ms"`
	ScannerCCHuntBackoffMs   int    `json:"scanner_cc_hunt_backoff_ms"`
	ScannerCCMaxBackoffMs    int    `json:"scanner_cc_max_backoff_ms"`
	ScannerManualTuneEnabled bool   `json:"scanner_manual_tune_enabled"`

	ToneProfiles []ToneProfileDTO `json:"tone_profiles,omitempty"`

	VocoderMap     map[string]string `json:"vocoder_map"`
	MetricsEnabled bool              `json:"metrics_enabled"`

	// ConfigPath is the path of the config.yaml backing the daemon,
	// or empty when the daemon was started without -config. The TUI
	// uses it to render the Settings panel as read-only when empty.
	ConfigPath string `json:"config_path,omitempty"`
	// StartupWarnings are the non-fatal observations the daemon
	// collected during NewDaemon. The Dashboard pins them as a
	// one-shot banner.
	StartupWarnings []string `json:"startup_warnings,omitempty"`
}

// ToneProfileDTO mirrors api.ToneProfileDTO.
type ToneProfileDTO struct {
	Name      string        `json:"name"`
	AlphaTag  string        `json:"alpha_tag,omitempty"`
	Cooldown  time.Duration `json:"cooldown_ns"`
	ToneCount int           `json:"tone_count"`
}

type HTTPError struct {
	Status int
	Method string
	URL    string
	Body   string
}

func (e *HTTPError) Error() string {
	return e.Method + " " + e.URL + ": " + httpCodeText(e.Status) + " — " + e.Body
}

func httpCodeText(code int) string {
	switch code {
	case 400:
		return "400 Bad Request"
	case 401:
		return "401 Unauthorized"
	case 404:
		return "404 Not Found"
	case 500:
		return "500 Internal Server Error"
	case 502:
		return "502 Bad Gateway"
	case 503:
		return "503 Service Unavailable"
	}
	return shortInt(code)
}

func shortInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [11]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
