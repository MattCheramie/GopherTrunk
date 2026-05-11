package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Log        LogConfig        `yaml:"log"`
	SDR        SDRConfig        `yaml:"sdr"`
	Trunking   TrunkingConfig   `yaml:"trunking"`
	API        APIConfig        `yaml:"api"`
	Storage    StorageConfig    `yaml:"storage"`
	Recordings RecordingsConfig `yaml:"recordings"`
	Metrics    MetricsConfig    `yaml:"metrics"`
	Retention  RetentionConfig  `yaml:"retention"`
	ToneOut    ToneOutConfig    `yaml:"tone_out"`
	Scanner    ScannerConfig    `yaml:"scanner"`
}

// ScannerConfig controls the police-scanner subsystems: the CC hunter,
// the talkgroup scan-list mode, and the conventional FM scanner.
// Empty == defaults; the daemon stays backwards compatible with
// pre-scanner configs.
type ScannerConfig struct {
	// ScanMode is "all" (every non-locked-out grant is followed,
	// the original behavior) or "list" (only TGs with Scan=true).
	// Empty string defaults to "all". Operators can flip this at
	// runtime from the TUI via PATCH /api/v1/scanner.
	ScanMode string `yaml:"scan_mode"`
	// CCHunt configures the multi-system control-channel hunter.
	CCHunt CCHuntConfig `yaml:"cc_hunt"`
	// Conventional is the fixed-frequency analog scan list.
	Conventional []ConvChannelConfig `yaml:"conventional"`
}

// CCHuntConfig tunes the hunter's dwell + exponential backoff.
type CCHuntConfig struct {
	// Enabled defaults to true when any trunked system is configured.
	// Set explicitly to false to ship without the hunter.
	Enabled bool `yaml:"enabled"`
	// DwellMs is the per-frequency wait window before declaring no
	// lock. Defaults to 3000.
	DwellMs int `yaml:"dwell_ms"`
	// BackoffMs is the initial sleep after exhausting a system's CC
	// list. Defaults to 5000. Doubles per failure up to MaxBackoffMs.
	BackoffMs int `yaml:"backoff_ms"`
	// MaxBackoffMs caps the exponential backoff. Defaults to 60000.
	MaxBackoffMs int `yaml:"max_backoff_ms"`
}

// ConvChannelConfig is one entry in the conventional scan list.
type ConvChannelConfig struct {
	Label       string  `yaml:"label"`
	FrequencyHz uint32  `yaml:"frequency_hz"`
	Mode        string  `yaml:"mode"`        // "fm" | "nfm"
	SquelchDbFS float64 `yaml:"squelch_dbfs"` // default -50
	HangtimeMs  int     `yaml:"hangtime_ms"`  // default 1500
	Priority    int     `yaml:"priority"`     // 1..10, 0 = unset
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type SDRConfig struct {
	SampleRate uint32          `yaml:"sample_rate"`
	Devices    []DeviceConfig  `yaml:"devices"`
}

type DeviceConfig struct {
	Serial  string `yaml:"serial"`
	Role    string `yaml:"role"`
	PPM     int    `yaml:"ppm"`
	// Gain is the tuner gain setting. "auto" (or empty) selects
	// the dongle's automatic gain control; any other value is
	// parsed as a tenths-of-dB integer matching librtlsdr's
	// gain table (e.g. "496" → 49.6 dB). Use `gophertrunk sdr
	// list` to see the supported values per device.
	Gain    string `yaml:"gain"`
	// BiasTee enables the dongle's 5V bias-tee output, used to
	// power external LNAs through the antenna SMA. Off by
	// default. Most modern RTL-SDR clones (e.g. NooElec NESDR
	// Smart v5) wire this through; older units may toggle a
	// GPIO bit that goes nowhere — librtlsdr accepts the call
	// either way.
	BiasTee bool `yaml:"bias_tee"`
}

type TrunkingConfig struct {
	Systems []SystemConfig `yaml:"systems"`
}

type SystemConfig struct {
	Name             string   `yaml:"name"`
	Protocol         string   `yaml:"protocol"`
	ControlChannels  []uint32 `yaml:"control_channels"`
	TalkgroupFile    string   `yaml:"talkgroup_file"`
}

// APIConfig controls the HTTP REST + SSE + WebSocket and gRPC servers.
// Both addresses are TCP listen specifiers (":8080", "127.0.0.1:9000",
// etc.). An empty value disables that surface.
//
// AllowMutations gates the write endpoints (end call, set talkgroup
// priority/lockout, retention sweep, tone-detector reset). Off by
// default — the daemon's HTTP API has no authentication, so any
// network-reachable instance is unauthenticated by definition. Only
// turn this on if you trust everything that can reach the listener.
type APIConfig struct {
	HTTPAddr       string `yaml:"http_addr"`
	GRPCAddr       string `yaml:"grpc_addr"`
	AllowMutations bool   `yaml:"allow_mutations"`
}

// StorageConfig configures the SQLite call log. An empty Path disables
// persistence (the daemon still runs, just without a call history).
type StorageConfig struct {
	Path string `yaml:"path"`
	// CCCacheFile is the JSON cache used by the CC hunter. Empty disables.
	CCCacheFile string `yaml:"cc_cache_file"`
}

// RecordingsConfig configures the per-call WAV recorder.
type RecordingsConfig struct {
	Dir         string `yaml:"dir"`
	SampleRate  uint32 `yaml:"sample_rate"`
	WriteRaw    bool   `yaml:"write_raw"`
	// Equalizer enables the per-call CMA blind equalizer that the FM
	// composer chain runs between the front-end LPF and the FM demod.
	// Off by default; useful when receiving simulcast systems with
	// multiple transmitters at slightly different arrival delays.
	Equalizer EqualizerConfig `yaml:"equalizer"`
}

// EqualizerConfig is the YAML shape of the optional CMA equalizer in
// the per-call FM voice chain.
type EqualizerConfig struct {
	Enabled  bool    `yaml:"enabled"`
	Taps     int     `yaml:"taps"`      // default 8 when enabled
	StepSize float32 `yaml:"step_size"` // default 1e-4 when enabled
}

// MetricsConfig toggles the Prometheus collector. The /metrics endpoint
// is mounted on the API HTTP server when both Enabled is true and the
// API HTTP address is configured.
type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

// RetentionConfig configures the background sweeper that ages out call
// log rows and recorded files. Zero values disable the corresponding
// sweep; both can be active independently.
type RetentionConfig struct {
	CallLogDays int           `yaml:"call_log_days"`
	FilesDays   int           `yaml:"files_days"`
	Interval    string        `yaml:"interval"` // Go duration string; default 1h
}

// ToneOutConfig describes paging-tone profiles to monitor. Empty
// Profiles disables the detector. Each ToneProfileConfig maps to one
// internal/voice/toneout.Profile.
type ToneOutConfig struct {
	Profiles []ToneProfileConfig `yaml:"profiles"`
}

// ToneProfileConfig is the YAML shape of one tone-out alarm.
//
//   - For two-tone sequential paging (most US fire/EMS) supply two
//     entries in `tones`: A-tone first, then B-tone.
//   - For single-tone supervision pages supply one tone.
//
// Durations are Go duration strings ("250ms", "1.5s"). MaxDuration
// of 0 disables the upper bound.
type ToneProfileConfig struct {
	Name               string                  `yaml:"name"`
	AlphaTag           string                  `yaml:"alpha_tag"`
	Tones              []ToneProfileToneConfig `yaml:"tones"`
	ToleranceHz        float64                 `yaml:"tolerance_hz"`
	MagnitudeThreshold float64                 `yaml:"magnitude_threshold"`
	MaxGap             string                  `yaml:"max_gap"`
	Cooldown           string                  `yaml:"cooldown"`
	System             string                  `yaml:"system"`
	GroupID            uint32                  `yaml:"group_id"`
}

// ToneProfileToneConfig is one tone within a profile sequence.
type ToneProfileToneConfig struct {
	FrequencyHz float64 `yaml:"frequency_hz"`
	MinDuration string  `yaml:"min_duration"`
	MaxDuration string  `yaml:"max_duration"`
}

func Default() Config {
	return Config{
		Log: LogConfig{Level: "info", Format: "text"},
		SDR: SDRConfig{SampleRate: 2_400_000},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.SDR.SampleRate != 0 && (c.SDR.SampleRate < 225_000 || c.SDR.SampleRate > 3_200_000) {
		return errors.New("sdr.sample_rate must be between 225 kHz and 3.2 MHz")
	}
	for i, d := range c.SDR.Devices {
		switch d.Role {
		case "", "control", "voice", "auto":
		default:
			return fmt.Errorf("sdr.devices[%d]: role must be control|voice|auto", i)
		}
	}
	for i, s := range c.Trunking.Systems {
		if s.Name == "" {
			return fmt.Errorf("trunking.systems[%d]: name required", i)
		}
		switch s.Protocol {
		case "p25", "dmr", "nxdn":
		default:
			return fmt.Errorf("trunking.systems[%d]: protocol must be p25|dmr|nxdn", i)
		}
	}
	if c.Recordings.SampleRate != 0 && (c.Recordings.SampleRate < 4000 || c.Recordings.SampleRate > 48_000) {
		return fmt.Errorf("recordings.sample_rate %d outside 4000..48000", c.Recordings.SampleRate)
	}
	if c.Retention.Interval != "" {
		if _, err := parseDurationFlexible(c.Retention.Interval); err != nil {
			return fmt.Errorf("retention.interval: %w", err)
		}
	}
	switch c.Scanner.ScanMode {
	case "", "all", "list":
	default:
		return fmt.Errorf("scanner.scan_mode must be \"all\" or \"list\"")
	}
	for i, ch := range c.Scanner.Conventional {
		if ch.FrequencyHz == 0 {
			return fmt.Errorf("scanner.conventional[%d]: frequency_hz required", i)
		}
		switch ch.Mode {
		case "", "fm", "nfm":
		default:
			return fmt.Errorf("scanner.conventional[%d]: mode must be fm|nfm", i)
		}
	}
	return nil
}

// parseDurationFlexible accepts a Go duration string. Wrapped here so
// the dependency lives in one place and tests can lean on it.
func parseDurationFlexible(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
