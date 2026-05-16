package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// RuntimeDTO is the sanitised, JSON-friendly snapshot of every config
// knob + runtime fact the TUI's tabbed Settings inspector renders.
// Keep this strictly read-only — no secrets, no credentials, no
// auth tokens. Operators expect /api/v1/runtime to be safe to scrape.
type RuntimeDTO struct {
	// API listener addresses (empty when disabled).
	HTTPAddr       string `json:"http_addr,omitempty"`
	GRPCAddr       string `json:"grpc_addr,omitempty"`
	WSPath         string `json:"ws_path,omitempty"`
	SSEPath        string `json:"sse_path,omitempty"`
	MetricsPath    string `json:"metrics_path,omitempty"`
	AllowMutations bool   `json:"allow_mutations"`

	// Daemon log + version.
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`
	Version   string `json:"version,omitempty"`

	// Storage paths (sanitised — paths only, never contents).
	StorageDBPath  string `json:"storage_db_path,omitempty"`
	StorageCCCache string `json:"storage_cc_cache,omitempty"`

	// Retention windows.
	RetentionCallLogDays int           `json:"retention_call_log_days"`
	RetentionFilesDays   int           `json:"retention_files_days"`
	RetentionInterval    time.Duration `json:"retention_interval_ns"`

	// Recording config.
	RecordingDir        string `json:"recording_dir,omitempty"`
	RecordingSampleRate int    `json:"recording_sample_rate"`
	RecordingWriteRaw   bool   `json:"recording_write_raw"`
	RecordingEQEnabled  bool   `json:"recording_eq_enabled"`
	RecordingEQTaps     int    `json:"recording_eq_taps,omitempty"`
	RecordingEQStepSize string `json:"recording_eq_step_size,omitempty"`

	// Audio runtime (mirrors AudioStatus but adds device list +
	// backend identity so operators can confirm whether the Linux
	// fallback path took effect).
	AudioEnabled       bool     `json:"audio_enabled"`
	AudioDevice        string   `json:"audio_device,omitempty"`
	AudioSampleRate    int      `json:"audio_sample_rate"`
	AudioBufferMs      int      `json:"audio_buffer_ms"`
	AudioBackends      []string `json:"audio_backends"`
	AudioDisableFallbk bool     `json:"audio_disable_fallback"`

	// SDR pool config (the live status is on /api/v1/devices).
	SDRSampleRate int      `json:"sdr_sample_rate"`
	SDRBackends   []string `json:"sdr_backends"`

	// Scanner config (the live state is on /api/v1/scanner).
	ScannerScanMode          string `json:"scanner_scan_mode"`
	ScannerCCHuntEnabled     bool   `json:"scanner_cc_hunt_enabled"`
	ScannerCCHuntDwellMs     int    `json:"scanner_cc_hunt_dwell_ms"`
	ScannerCCHuntBackoffMs   int    `json:"scanner_cc_hunt_backoff_ms"`
	ScannerCCMaxBackoffMs    int    `json:"scanner_cc_max_backoff_ms"`
	ScannerManualTuneEnabled bool   `json:"scanner_manual_tune_enabled"`

	// Tone-out profiles (names only, plus tone counts + cooldown).
	ToneProfiles []ToneProfileDTO `json:"tone_profiles,omitempty"`

	// Vocoder map by protocol — operator-facing names like
	// "p25-phase2" → "ambe2".
	VocoderMap map[string]string `json:"vocoder_map"`

	// MetricsEnabled mirrors metrics.enabled config.
	MetricsEnabled bool `json:"metrics_enabled"`

	// ConfigPath is the absolute path to the config.yaml backing this
	// daemon, or empty when the daemon was started without a -config
	// file. The SPA / TUI use it to gate the editable Settings panel:
	// empty = render read-only ("daemon running on built-in defaults").
	ConfigPath string `json:"config_path,omitempty"`
	// StartupWarnings carries the non-fatal observations the daemon
	// collected during NewDaemon (missing talkgroup CSV, SDR pool
	// failed to open, etc.). Surfaced so the SPA Dashboard can pin
	// them until the operator dismisses them.
	StartupWarnings []string `json:"startup_warnings,omitempty"`
}

// ToneProfileDTO is the minimal projection of a tone-out profile —
// no internal detector state, just the operator-relevant fields.
type ToneProfileDTO struct {
	Name      string        `json:"name"`
	AlphaTag  string        `json:"alpha_tag,omitempty"`
	Cooldown  time.Duration `json:"cooldown_ns"`
	ToneCount int           `json:"tone_count"`
}

// RuntimeProvider returns the runtime snapshot. The daemon supplies
// the production impl; tests supply a fake. Optional on ServerOptions —
// when nil, GET /api/v1/runtime returns 503.
type RuntimeProvider interface {
	Runtime() RuntimeDTO
}

func (s *Server) handleRuntime(w http.ResponseWriter, _ *http.Request) {
	if s.runtime == nil {
		http.Error(w, "runtime snapshot unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.runtime.Runtime())
}
