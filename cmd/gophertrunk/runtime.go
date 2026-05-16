package main

import (
	"fmt"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/voice/player"
)

// runtimeSnapshot wraps the daemon config + a few process-global
// registries into the api.RuntimeProvider shape. The DTO is rebuilt
// on every /api/v1/runtime call since the daemon config is immutable
// at runtime; cost is a handful of slice allocs.
type runtimeSnapshot struct {
	cfg     *config.Config
	version string
	metrics bool
	daemon  *Daemon // for StartupWarnings, ConfigPath, ...
}

// vocoderProtocolMap is the canonical mapping the daemon hands to
// each protocol's voice decoder at construction time. Surfaced here
// so the TUI's Vocoders tab matches what the engine actually loads.
// Keep in sync with the per-protocol pipeline factories in
// internal/radio/*/pipeline.go.
var vocoderProtocolMap = map[string]string{
	"p25":        "imbe",
	"p25-phase2": "ambe2",
	"dmr":        "ambe2",
	"nxdn":       "ambe2",
	"dpmr":       "ambe2",
	"tetra":      "ambe2",
	"ysf":        "ambe2",
	"edacs":      "ambe2",
	"motorola":   "—",
	"ltr":        "—",
	"mpt1327":    "—",
}

func (r *runtimeSnapshot) Runtime() api.RuntimeDTO {
	cfg := r.cfg
	dto := api.RuntimeDTO{
		HTTPAddr:       cfg.API.HTTPAddr,
		GRPCAddr:       cfg.API.GRPCAddr,
		AllowMutations: cfg.API.AllowMutations,
		LogLevel:       cfg.Log.Level,
		LogFormat:      cfg.Log.Format,
		Version:        r.version,

		StorageDBPath:  cfg.Storage.Path,
		StorageCCCache: cfg.Storage.CCCacheFile,

		RetentionCallLogDays: cfg.Retention.CallLogDays,
		RetentionFilesDays:   cfg.Retention.FilesDays,

		RecordingDir:        cfg.Recordings.Dir,
		RecordingSampleRate: int(cfg.Recordings.SampleRate),
		RecordingWriteRaw:   cfg.Recordings.WriteRaw,
		RecordingEQEnabled:  cfg.Recordings.Equalizer.Enabled,
		RecordingEQTaps:     cfg.Recordings.Equalizer.Taps,

		AudioEnabled:    cfg.Audio.Enabled,
		AudioDevice:     cfg.Audio.Device,
		AudioSampleRate: int(cfg.Audio.SampleRate),
		AudioBufferMs:   cfg.Audio.BufferMs,
		AudioBackends:   player.ListDevices(),

		SDRSampleRate: int(cfg.SDR.SampleRate),
		SDRBackends:   sdrBackendNames(),

		ScannerScanMode:          cfg.Scanner.ScanMode,
		ScannerCCHuntEnabled:     cfg.Scanner.CCHunt.Enabled,
		ScannerCCHuntDwellMs:     cfg.Scanner.CCHunt.DwellMs,
		ScannerCCHuntBackoffMs:   cfg.Scanner.CCHunt.BackoffMs,
		ScannerCCMaxBackoffMs:    cfg.Scanner.CCHunt.MaxBackoffMs,
		ScannerManualTuneEnabled: cfg.Scanner.ManualTuneEnabled,

		MetricsEnabled: r.metrics,
		VocoderMap:     vocoderProtocolMap,
	}
	if cfg.Recordings.Equalizer.StepSize != 0 {
		dto.RecordingEQStepSize = formatFloat(float64(cfg.Recordings.Equalizer.StepSize))
	}
	if d, err := time.ParseDuration(cfg.Retention.Interval); err == nil {
		dto.RetentionInterval = d
	}
	for _, prof := range cfg.ToneOut.Profiles {
		cooldown, _ := time.ParseDuration(prof.Cooldown)
		dto.ToneProfiles = append(dto.ToneProfiles, api.ToneProfileDTO{
			Name:      prof.Name,
			AlphaTag:  prof.AlphaTag,
			Cooldown:  cooldown,
			ToneCount: len(prof.Tones),
		})
	}
	if r.daemon != nil {
		dto.ConfigPath = r.daemon.ConfigPath()
		dto.StartupWarnings = r.daemon.StartupWarnings()
	}
	return dto
}

func sdrBackendNames() []string {
	drivers := sdr.Drivers()
	out := make([]string, 0, len(drivers))
	for _, d := range drivers {
		out = append(out, d.Name())
	}
	return out
}

// formatFloat is a tiny helper to avoid pulling strconv into the API
// DTO encoding path. 6 significant digits is enough for step_size
// constants like 1e-4.
func formatFloat(v float64) string {
	// We deliberately go via fmt — strconv has no shorthand for "drop
	// trailing zeroes" without going through ParseFloat round-trips.
	const eps = 1e-12
	s := ""
	switch {
	case v == 0:
		return "0"
	case v < eps:
		s = fmt.Sprintf("%g", v)
	default:
		s = fmt.Sprintf("%g", v)
	}
	return s
}
