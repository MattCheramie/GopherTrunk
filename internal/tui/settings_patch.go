package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
)

// buildSettingsPatch converts a (dotted field, stringified value)
// pair from the Settings panel into a typed client.SettingsPatch.
// Validation (parse failure, unrecognised field) returns an error
// the caller surfaces inline on the editing row.
func buildSettingsPatch(field, raw string) (client.SettingsPatch, error) {
	var p client.SettingsPatch
	switch field {
	case "log.level":
		s := raw
		p.LogLevel = &s
	case "log.format":
		s := raw
		p.LogFormat = &s
	case "api.http_addr":
		s := raw
		p.APIHTTPAddr = &s
	case "api.grpc_addr":
		s := raw
		p.APIGRPCAddr = &s
	case "api.auth.mode":
		s := raw
		p.APIAuthMode = &s
	case "metrics.enabled":
		b, err := parseBool(raw)
		if err != nil {
			return p, err
		}
		p.MetricsEnabled = &b
	case "audio.enabled":
		b, err := parseBool(raw)
		if err != nil {
			return p, err
		}
		p.AudioEnabled = &b
	case "audio.device":
		s := raw
		p.AudioDevice = &s
	case "audio.volume":
		f64, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return p, fmt.Errorf("audio.volume: %w (want a float 0..1)", err)
		}
		f := float32(f64)
		if f < 0 || f > 1 {
			return p, fmt.Errorf("audio.volume %v outside 0..1", f)
		}
		p.AudioVolume = &f
	case "audio.muted":
		b, err := parseBool(raw)
		if err != nil {
			return p, err
		}
		p.AudioMuted = &b
	case "audio.buffer_ms":
		n, err := strconv.Atoi(raw)
		if err != nil {
			return p, fmt.Errorf("audio.buffer_ms: %w (want an integer)", err)
		}
		p.AudioBufferMs = &n
	case "recordings.dir":
		s := raw
		p.RecordingsDir = &s
	case "recordings.sample_rate":
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return p, fmt.Errorf("recordings.sample_rate: %w", err)
		}
		v := uint32(n)
		p.RecordingsSampleRate = &v
	case "recordings.write_raw":
		b, err := parseBool(raw)
		if err != nil {
			return p, err
		}
		p.RecordingsWriteRaw = &b
	case "retention.call_log_days":
		n, err := strconv.Atoi(raw)
		if err != nil {
			return p, fmt.Errorf("retention.call_log_days: %w", err)
		}
		p.RetentionCallLogDays = &n
	case "retention.files_days":
		n, err := strconv.Atoi(raw)
		if err != nil {
			return p, fmt.Errorf("retention.files_days: %w", err)
		}
		p.RetentionFilesDays = &n
	case "retention.interval":
		s := raw
		p.RetentionInterval = &s
	case "sdr.sample_rate":
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return p, fmt.Errorf("sdr.sample_rate: %w", err)
		}
		v := uint32(n)
		p.SDRSampleRate = &v
	case "scanner.scan_mode":
		s := raw
		p.ScannerScanMode = &s
	case "scanner.manual_tune_enabled":
		b, err := parseBool(raw)
		if err != nil {
			return p, err
		}
		p.ScannerManualTuneEnabled = &b
	case "scanner.cc_hunt.enabled":
		b, err := parseBool(raw)
		if err != nil {
			return p, err
		}
		p.ScannerCCHuntEnabled = &b
	case "scanner.cc_hunt.dwell_ms":
		n, err := strconv.Atoi(raw)
		if err != nil {
			return p, fmt.Errorf("scanner.cc_hunt.dwell_ms: %w", err)
		}
		p.ScannerCCHuntDwellMs = &n
	case "scanner.cc_hunt.backoff_ms":
		n, err := strconv.Atoi(raw)
		if err != nil {
			return p, fmt.Errorf("scanner.cc_hunt.backoff_ms: %w", err)
		}
		p.ScannerCCHuntBackoffMs = &n
	case "scanner.cc_hunt.max_backoff_ms":
		n, err := strconv.Atoi(raw)
		if err != nil {
			return p, fmt.Errorf("scanner.cc_hunt.max_backoff_ms: %w", err)
		}
		p.ScannerCCHuntMaxBackoff = &n
	case "storage.path":
		s := raw
		p.StoragePath = &s
	case "storage.cc_cache_file":
		s := raw
		p.StorageCCCacheFile = &s
	default:
		return p, fmt.Errorf("unknown field %q", field)
	}
	return p, nil
}

// parseBool accepts the common forms operators type ("true",
// "false", "on", "off", "yes", "no") and returns a descriptive
// error otherwise.
func parseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "on", "yes", "1":
		return true, nil
	case "false", "off", "no", "0":
		return false, nil
	}
	return false, fmt.Errorf("want true/false (also accepts on/off, yes/no, 1/0)")
}
