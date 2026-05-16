package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// SettingsPatchRequest is the JSON shape of PATCH /api/v1/settings.
// Pointer fields preserve "leave alone" semantics — JSON-omitted
// fields aren't zeroed in the daemon's running config.
//
// The shape mirrors config.Patch one-for-one; the names use snake-
// case-with-section-prefix so the wire format reads close to the
// YAML config keys an operator already knows from config.yaml.
type SettingsPatchRequest struct {
	LogLevel  *string `json:"log_level,omitempty"`
	LogFormat *string `json:"log_format,omitempty"`

	APIHTTPAddr *string `json:"api_http_addr,omitempty"`
	APIGRPCAddr *string `json:"api_grpc_addr,omitempty"`
	APIAuthMode *string `json:"api_auth_mode,omitempty"`

	AudioEnabled  *bool    `json:"audio_enabled,omitempty"`
	AudioDevice   *string  `json:"audio_device,omitempty"`
	AudioVolume   *float32 `json:"audio_volume,omitempty"`
	AudioMuted    *bool    `json:"audio_muted,omitempty"`
	AudioBufferMs *int     `json:"audio_buffer_ms,omitempty"`

	RecordingsDir        *string `json:"recordings_dir,omitempty"`
	RecordingsSampleRate *uint32 `json:"recordings_sample_rate,omitempty"`
	RecordingsWriteRaw   *bool   `json:"recordings_write_raw,omitempty"`

	RetentionCallLogDays *int    `json:"retention_call_log_days,omitempty"`
	RetentionFilesDays   *int    `json:"retention_files_days,omitempty"`
	RetentionInterval    *string `json:"retention_interval,omitempty"`

	SDRSampleRate *uint32 `json:"sdr_sample_rate,omitempty"`

	ScannerScanMode          *string `json:"scanner_scan_mode,omitempty"`
	ScannerManualTuneEnabled *bool   `json:"scanner_manual_tune_enabled,omitempty"`
	ScannerCCHuntEnabled     *bool   `json:"scanner_cc_hunt_enabled,omitempty"`
	ScannerCCHuntDwellMs     *int    `json:"scanner_cc_hunt_dwell_ms,omitempty"`
	ScannerCCHuntBackoffMs   *int    `json:"scanner_cc_hunt_backoff_ms,omitempty"`
	ScannerCCHuntMaxBackoff  *int    `json:"scanner_cc_hunt_max_backoff_ms,omitempty"`

	StoragePath        *string `json:"storage_path,omitempty"`
	StorageCCCacheFile *string `json:"storage_cc_cache_file,omitempty"`

	MetricsEnabled *bool `json:"metrics_enabled,omitempty"`
}

// toPatch converts the wire-level request into a config.Patch.
func (r SettingsPatchRequest) toPatch() config.Patch {
	return config.Patch{
		LogLevel:                 r.LogLevel,
		LogFormat:                r.LogFormat,
		APIHTTPAddr:              r.APIHTTPAddr,
		APIGRPCAddr:              r.APIGRPCAddr,
		APIAuthMode:              r.APIAuthMode,
		AudioEnabled:             r.AudioEnabled,
		AudioDevice:              r.AudioDevice,
		AudioVolume:              r.AudioVolume,
		AudioMuted:               r.AudioMuted,
		AudioBufferMs:            r.AudioBufferMs,
		RecordingsDir:            r.RecordingsDir,
		RecordingsSampleRate:     r.RecordingsSampleRate,
		RecordingsWriteRaw:       r.RecordingsWriteRaw,
		RetentionCallLogDays:     r.RetentionCallLogDays,
		RetentionFilesDays:       r.RetentionFilesDays,
		RetentionInterval:        r.RetentionInterval,
		SDRSampleRate:            r.SDRSampleRate,
		ScannerScanMode:          r.ScannerScanMode,
		ScannerManualTuneEnabled: r.ScannerManualTuneEnabled,
		ScannerCCHuntEnabled:     r.ScannerCCHuntEnabled,
		ScannerCCHuntDwellMs:     r.ScannerCCHuntDwellMs,
		ScannerCCHuntBackoffMs:   r.ScannerCCHuntBackoffMs,
		ScannerCCHuntMaxBackoff:  r.ScannerCCHuntMaxBackoff,
		StoragePath:              r.StoragePath,
		StorageCCCacheFile:       r.StorageCCCacheFile,
		MetricsEnabled:           r.MetricsEnabled,
	}
}

// SettingsResponse is the JSON shape returned by PATCH /api/v1/settings.
// Applied lists the keys that took effect immediately; RestartRequired
// lists keys that were written to config.yaml but need a daemon
// restart to take effect.
type SettingsResponse struct {
	Applied         []string   `json:"applied"`
	RestartRequired []string   `json:"restart_required"`
	ConfigPath      string     `json:"config_path,omitempty"`
	Runtime         RuntimeDTO `json:"runtime"`
}

// SettingsApplier is the optional in-process hot-reload surface for
// fields the daemon can change without a restart. Routes that don't
// have a matching Applier method are still written to disk; the
// response's RestartRequired list flags them so the UI can render
// "restart required" badges.
type SettingsApplier interface {
	SetLogLevel(level string) error
	SetAudioVolume(volume float32)
	SetAudioMuted(muted bool)
	SetAudioEnabled(enabled bool)
	SetRecordingEnabled(enabled bool)
	SetScannerScanMode(mode string) error
}

// ConfigWriter wraps the daemon's config.yaml writer. Decoupled via
// an interface so tests can fake it and the api package doesn't pull
// in the OS file machinery.
type ConfigWriter interface {
	// WritePatch applies the patch to the backing config.yaml and
	// returns the merged config so callers can route hot-reloadable
	// fields to the in-memory applier.
	WritePatch(p config.Patch) (config.Config, error)
	// Path is the path to the config.yaml the writer mutates.
	// Empty means "no config file backs this daemon" (the SPA / TUI
	// should render the Settings panel read-only).
	Path() string
}

// handleSettingsPatch is the entry for PATCH /api/v1/settings.
//
// 200 → response body carries applied / restart_required / runtime
// 400 → invalid JSON or empty patch
// 503 → daemon has no ConfigWriter wired (no -config at startup)
// 409 → config.yaml was edited externally; daemon refuses to clobber
func (s *Server) handleSettingsPatch(w http.ResponseWriter, r *http.Request) {
	if s.configWriter == nil {
		writeError(w, http.StatusServiceUnavailable, "settings: no config file backs this daemon (start with -config to enable live edits)")
		return
	}
	var req SettingsPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "settings: "+err.Error())
		return
	}
	patch := req.toPatch()
	if patch.IsEmpty() {
		writeError(w, http.StatusBadRequest, "settings: patch has no fields")
		return
	}
	if _, err := s.configWriter.WritePatch(patch); err != nil {
		// Distinguish external-edit conflict from a generic write
		// failure so the UI can present a clearer toast.
		if isExternalEditConflict(err) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	applied, restartReq := s.applyHotReload(patch)
	resp := SettingsResponse{
		Applied:         applied,
		RestartRequired: restartReq,
		ConfigPath:      s.configWriter.Path(),
	}
	if s.runtime != nil {
		resp.Runtime = s.runtime.Runtime()
	}
	writeJSON(w, http.StatusOK, resp)
}

// applyHotReload walks the patch and dispatches each non-nil field
// to the SettingsApplier when one is wired. Returns the lists used
// in the response so the UI can render "applied" vs "restart
// required" badges.
func (s *Server) applyHotReload(p config.Patch) (applied, restartRequired []string) {
	hot := func(key string) {
		applied = append(applied, key)
	}
	cold := func(key string) {
		restartRequired = append(restartRequired, key)
	}

	app := s.settings // may be nil — every field falls back to cold

	if p.AudioVolume != nil {
		if app != nil {
			app.SetAudioVolume(*p.AudioVolume)
			hot("audio.volume")
		} else {
			cold("audio.volume")
		}
	}
	if p.AudioMuted != nil {
		if app != nil {
			app.SetAudioMuted(*p.AudioMuted)
			hot("audio.muted")
		} else {
			cold("audio.muted")
		}
	}
	if p.AudioEnabled != nil {
		if app != nil {
			app.SetAudioEnabled(*p.AudioEnabled)
			hot("audio.enabled")
		} else {
			cold("audio.enabled")
		}
	}
	if p.ScannerScanMode != nil {
		if app != nil {
			if err := app.SetScannerScanMode(*p.ScannerScanMode); err == nil {
				hot("scanner.scan_mode")
			} else {
				cold("scanner.scan_mode")
			}
		} else {
			cold("scanner.scan_mode")
		}
	}
	if p.LogLevel != nil {
		if app != nil {
			if err := app.SetLogLevel(*p.LogLevel); err == nil {
				hot("log.level")
			} else {
				cold("log.level")
			}
		} else {
			cold("log.level")
		}
	}
	if p.RecordingsWriteRaw != nil {
		if app != nil {
			app.SetRecordingEnabled(*p.RecordingsWriteRaw)
			hot("recordings.write_raw")
		} else {
			cold("recordings.write_raw")
		}
	}

	// Cold-only fields (require a daemon restart to take effect).
	coldOnly := []struct {
		key string
		set bool
	}{
		{"log.format", p.LogFormat != nil},
		{"api.http_addr", p.APIHTTPAddr != nil},
		{"api.grpc_addr", p.APIGRPCAddr != nil},
		{"api.auth.mode", p.APIAuthMode != nil},
		{"audio.device", p.AudioDevice != nil},
		{"audio.buffer_ms", p.AudioBufferMs != nil},
		{"recordings.dir", p.RecordingsDir != nil},
		{"recordings.sample_rate", p.RecordingsSampleRate != nil},
		{"retention.call_log_days", p.RetentionCallLogDays != nil},
		{"retention.files_days", p.RetentionFilesDays != nil},
		{"retention.interval", p.RetentionInterval != nil},
		{"sdr.sample_rate", p.SDRSampleRate != nil},
		{"scanner.manual_tune_enabled", p.ScannerManualTuneEnabled != nil},
		{"scanner.cc_hunt.enabled", p.ScannerCCHuntEnabled != nil},
		{"scanner.cc_hunt.dwell_ms", p.ScannerCCHuntDwellMs != nil},
		{"scanner.cc_hunt.backoff_ms", p.ScannerCCHuntBackoffMs != nil},
		{"scanner.cc_hunt.max_backoff_ms", p.ScannerCCHuntMaxBackoff != nil},
		{"storage.path", p.StoragePath != nil},
		{"storage.cc_cache_file", p.StorageCCCacheFile != nil},
		{"metrics.enabled", p.MetricsEnabled != nil},
	}
	for _, c := range coldOnly {
		if c.set {
			cold(c.key)
		}
	}
	return applied, restartRequired
}

// isExternalEditConflict checks for the writer's mtime-guard error.
func isExternalEditConflict(err error) bool {
	if err == nil {
		return false
	}
	var msg = err.Error()
	for _, needle := range []string{"modified externally"} {
		if needle != "" && len(msg) >= len(needle) {
			// substring check without bringing in strings for one call
			for i := 0; i+len(needle) <= len(msg); i++ {
				if msg[i:i+len(needle)] == needle {
					return true
				}
			}
		}
	}
	return false
}

// ensureSettingsWired prevents the legacy callers that bypass the
// settings interface from silently degrading the new endpoint.
func (s *Server) ensureSettingsWired() error {
	if s.configWriter == nil {
		return errors.New("api: no ConfigWriter configured")
	}
	return nil
}
