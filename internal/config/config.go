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
	Serial string `yaml:"serial"`
	Role   string `yaml:"role"`
	PPM    int    `yaml:"ppm"`
	Gain   string `yaml:"gain"`
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
type APIConfig struct {
	HTTPAddr string `yaml:"http_addr"`
	GRPCAddr string `yaml:"grpc_addr"`
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
	return nil
}

// parseDurationFlexible accepts a Go duration string. Wrapped here so
// the dependency lives in one place and tests can lean on it.
func parseDurationFlexible(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
