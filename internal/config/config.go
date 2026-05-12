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
	Audio      AudioConfig      `yaml:"audio"`
}

// AudioConfig controls live audio playback to the host's speakers.
// The daemon mixes decoded PCM from the per-call composer and the
// conventional scanner into a single output stream, applied with
// software gain so volume / mute changes are instant.
//
// Disabled by default — headless servers stay silent unless
// audio.enabled is set true. Backend init failure (e.g. no audio
// device, no PulseAudio / ALSA on the host) falls back to the null
// player automatically.
type AudioConfig struct {
	// Enabled gates live playback. Default false. The recorder
	// path is unaffected: WAVs land on disk whether audio is on
	// or off.
	Enabled bool `yaml:"enabled"`
	// Device is the backend-specific output device name. Empty
	// (or "default") routes to the system default sink. "null"
	// forces the no-op backend even when Enabled=true.
	Device string `yaml:"device"`
	// SampleRate is the host playback rate in Hz. Default 8000;
	// must match recordings.sample_rate so the composer's PCM
	// frames don't need a resample stage.
	SampleRate uint32 `yaml:"sample_rate"`
	// BufferMs is the depth of the playback queue. Default 80.
	BufferMs int `yaml:"buffer_ms"`
	// Volume is the initial software gain (0..1). Default 0.8.
	Volume float32 `yaml:"volume"`
	// Muted is the initial mute state. Default false.
	Muted bool `yaml:"muted"`
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
	// ManualTuneEnabled forces construction of the conventional
	// scanner so the TUI's `f` key (or POST
	// /api/v1/scanner/manual_tune) can VFO-tune at runtime even
	// when no static channels are configured. With this set the
	// scanner steals one Voice SDR from the trunking pool
	// regardless of how many Voice SDRs are available.
	//
	// Default false; the daemon auto-detects when at least two
	// Voice SDRs are present (sum >= 2) and constructs the
	// scanner from the spare without requiring this flag. To
	// keep all Voice SDRs reserved for trunking even with a
	// spare, leave this false and the auto-detect rule still
	// holds — set ManualTuneDisabled to opt out entirely.
	ManualTuneEnabled bool `yaml:"manual_tune_enabled"`
	// ManualTuneDisabled vetoes the auto-detect rule. When true,
	// the conventional scanner is constructed only when
	// `conventional` channels are explicitly listed or
	// ManualTuneEnabled is set true.
	ManualTuneDisabled bool `yaml:"manual_tune_disabled"`
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
	// Tone is the optional CTCSS / DCS sub-audible squelch gate.
	// Zero / "none" disables tone gating (default).
	Tone ConvToneConfig `yaml:"tone"`
}

// ConvToneConfig configures CTCSS / DCS gating for one conventional
// channel.
type ConvToneConfig struct {
	// Mode is "ctcss", "dcs", or "" / "none".
	Mode string `yaml:"mode"`
	// CTCSSHz is the target CTCSS frequency (50..300 Hz).
	// Required when Mode is "ctcss".
	CTCSSHz float64 `yaml:"ctcss_hz"`
	// DCSCode is the 3-digit octal DCS code. Required when
	// Mode is "dcs". Detector wiring is a tracked follow-up; the
	// config is accepted now so deployments can pre-stage YAML.
	DCSCode string `yaml:"dcs_code"`
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
	Name            string   `yaml:"name"`
	Protocol        string   `yaml:"protocol"`
	ControlChannels []uint32 `yaml:"control_channels"`
	TalkgroupFile   string   `yaml:"talkgroup_file"`

	// TETRAColourCode is the 30-bit extended colour code the TETRA
	// scrambler uses to seed its LFSR (ETSI EN 300 392-2 §8.2.5).
	// Set this to the per-cell colour code of the TETRA TMO system
	// being decoded so the descrambler can recover the type-3
	// stream. Bits 30..31 are silently ignored. Zero is valid only
	// for BSCH (§8.2.5.2); non-BSCH channels need the per-cell
	// colour code or descrambling produces garbage. Ignored for
	// non-TETRA protocols.
	TETRAColourCode uint32 `yaml:"tetra_colour_code"`
	// TETRAChannel selects which TETRA logical channel lives in
	// each burst window under ChannelCodingOn. Recognised values:
	// "sch/hd" | "sch/f" | "sch/hu" | "bsch" | "aach". Empty
	// defaults to "sch/hd" — the standard signaling channel for
	// cc.locked / Grant events. Ignored for non-TETRA protocols.
	TETRAChannel string `yaml:"tetra_channel"`
	// TETRAChannelCoding gates the full ETSI EN 300 392-2 §8.3.1
	// channel-coding chain (descramble + deinterleave + depuncture
	// + Viterbi + CRC-16 verify + tail strip). Recognised values:
	// "" / "on" / "true" / "1" (the new default — full chain;
	// required for live on-air captures) or "off" / "false" / "0"
	// (legacy raw-dibit path, opt-out for operators feeding pre-
	// stripped DSD-FME / OP25 fixtures). Ignored for non-TETRA
	// protocols.
	TETRAChannelCoding string `yaml:"tetra_channel_coding"`

	// LTRFCSMode enables the CRC-7 FCS check on the LTR Status
	// Ingest path. Recognised values: "" / "on" / "true" / "1"
	// (the new default — drop Status words whose FCS trailer
	// doesn't match) or "off" / "false" / "0" (no verification —
	// opt-out for synthesized fixtures whose FCS trailer isn't
	// populated). Ignored for non-LTR protocols.
	LTRFCSMode string `yaml:"ltr_fcs_mode"`
	// LTRManchesterMode controls Manchester decoding of the
	// sub-audible LTR bit stream. Recognised values: "" / "on" /
	// "soft" (the new default — majority-decode + tolerate noise
	// bursts; matches the dominant on-air encoding), "strict"
	// (require a mid-bit transition per pair, drop transition-less
	// pairs), "off" / "nrz" (raw NRZ — opt-out for synthesized NRZ
	// fixtures). Ignored for non-LTR protocols.
	LTRManchesterMode string `yaml:"ltr_manchester_mode"`

	// P25Phase2TrellisMode enables the 4-state ½-rate trellis FEC
	// decoder on the P25 Phase 2 MAC PDU window. Recognised values:
	// "" / "on" / "true" / "1" (the new default — 146 channel
	// dibits via the TIA-102.AABF trellis decoder) or "off" /
	// "false" / "0" (legacy 72-dibit raw-MAC-PDU path, opt-out for
	// pre-stripped fixtures). Ignored for non-P25-Phase-2 protocols.
	P25Phase2TrellisMode string `yaml:"p25_phase2_trellis_mode"`
	// P25Phase2RSMode enables the outer Reed-Solomon RS(24, 16, 9)
	// verification layer on top of the trellis-decoded MAC PDU.
	// Recognised values: "" / "off" / "false" / "0" (the default —
	// no outer RS verification; matches historical decoder
	// behaviour) or "on" / "true" / "1" (verify RS syndromes per
	// TIA-102.BAAA-A §5.9; drop MAC PDUs whose syndromes are
	// non-zero before parsing). The per-burst block interleaver
	// schedule defined in TIA-102.BBAC remains a follow-up.
	// Ignored for non-P25-Phase-2 protocols.
	P25Phase2RSMode string `yaml:"p25_phase2_rs_mode"`
	// P25Phase2ClockMode selects the symbol-timing-recovery strategy
	// for the P25 Phase 2 receiver. Recognised values: "" /
	// "gardner" / "on" (the new default — non-data-aided Gardner
	// loop; recommended for live SDR captures) or "naive" / "off"
	// (decimate every sps-th sample; works on sample-aligned
	// synthesized IQ). Ignored for non-P25-Phase-2 protocols.
	P25Phase2ClockMode string `yaml:"p25_phase2_clock_mode"`
	// TETRAClockMode mirrors P25Phase2ClockMode for the TETRA
	// receiver. Recognised values: "" / "gardner" / "on" (the new
	// default) or "naive" / "off". Ignored for non-TETRA protocols.
	TETRAClockMode string `yaml:"tetra_clock_mode"`
	// NXDNViterbiMode enables the K=5 ½-rate Viterbi FEC decoder
	// on the NXDN CAC region. Recognised values: "" / "spec" (the
	// new default — full NXDN-TS-1-A §4.5.1.1 outbound CAC chain),
	// "on" / "true" / "1" (intermediate 92-dibit K=5 Viterbi path
	// for older MMDVMHost / DSDcc fixtures), or "off" / "false" /
	// "0" (legacy 44-dibit raw-CAC path, opt-out for pre-stripped
	// fixtures). Ignored for non-NXDN protocols.
	NXDNViterbiMode string `yaml:"nxdn_viterbi_mode"`
	// EDACSBCHMode enables the BCH(40, 28, 2) FEC layer on the
	// EDACS CCW. Recognised values: "" / "on" / "true" / "1" (the
	// new default — 40-bit on-wire BCH decode with single/double-
	// bit correction) or "off" / "false" / "0" (legacy pre-stripped
	// 40-bit CCW, opt-out for pre-stripped fixtures). Ignored for
	// non-EDACS protocols.
	EDACSBCHMode string `yaml:"edacs_bch_mode"`
	// MPT1327BCHMode enables the BCH(63, 38) FEC layer on the MPT
	// 1327 codeword. Recognised values: "" / "on" / "true" / "1"
	// (the new default — 64-bit on-wire BCH decode) or "off" /
	// "false" / "0" (legacy 38-bit pre-stripped codeword, opt-out
	// for pre-stripped fixtures). Ignored for non-MPT-1327
	// protocols.
	MPT1327BCHMode string `yaml:"mpt1327_bch_mode"`
	// MotorolaBCHMode enables the BCH(64, 16, 11) FEC layer on the
	// Motorola Type II OSW. Recognised values: "" / "on" / "true" /
	// "1" (the new default — two 64-bit BCH(64, 16, 11) codewords
	// reassembled into the 32-bit OSW with single- through 11-bit-
	// error correction) or "off" / "false" / "0" (legacy 32-bit
	// raw-OSW path, opt-out for pre-stripped fixtures). Ignored
	// for non-Motorola protocols.
	MotorolaBCHMode string `yaml:"motorola_bch_mode"`
	// DStarFECMode enables the JARL DV-mode header FEC chain on
	// the D-STAR Process adapter (conv R=1/2 K=5 + PN15 scrambler
	// + 22×30 block interleaver). Recognised values: "" / "off" /
	// "false" / "0" (the default — 328 info bits straight off the
	// wire) or "on" / "true" / "1" (660 on-wire bits → full FEC
	// chain → 328 info bits → ParseHeader). Ignored for non-D-STAR
	// protocols.
	DStarFECMode string `yaml:"dstar_fec_mode"`
}

// APIConfig controls the HTTP REST + SSE + WebSocket and gRPC servers.
// Both addresses are TCP listen specifiers (":8080", "127.0.0.1:9000",
// etc.). An empty value disables that surface.
//
// Auth gates the write endpoints (end call, set talkgroup
// priority/lockout, retention sweep, tone-detector reset, scanner
// cockpit, audio cockpit). See APIAuthConfig for the policy modes;
// the default `auto` mode bypasses auth on loopback binds and
// requires a bearer token on public binds.
//
// AllowMutations is the legacy gate. Setting it to true logs a
// deprecation warning and maps to `auth.mode: disabled` so the
// daemon's existing wide-open behaviour is preserved.
type APIConfig struct {
	HTTPAddr       string         `yaml:"http_addr"`
	GRPCAddr       string         `yaml:"grpc_addr"`
	AllowMutations bool           `yaml:"allow_mutations"`
	Auth           APIAuthConfig  `yaml:"auth"`
}

// APIAuthConfig configures bearer-token authentication on the HTTP
// API's mutation endpoints. See internal/api/AuthMode for the policy
// modes.
type APIAuthConfig struct {
	// Mode picks the auth policy. Recognised values:
	//   "" / "auto"     → auto (the default — require a token on
	//                     non-loopback binds, bypass on loopback)
	//   "required" / "on" → require a token on every mutation
	//   "disabled" / "off" → no auth, mutations wide open (the
	//                       legacy `allow_mutations: true` behaviour)
	Mode string `yaml:"mode"`
	// Token is the inline bearer token (compared via crypto/subtle).
	// Prefer TokenFile so the token doesn't live in config.yaml.
	Token string `yaml:"token"`
	// TokenFile is a path to a file containing the bearer token
	// (whitespace stripped). The daemon re-reads it on every
	// request so operators can rotate without a restart.
	TokenFile string `yaml:"token_file"`
	// TrustedNetworks is a list of CIDRs whose source addresses
	// bypass the token check under `auto` mode. Loopback
	// (127.0.0.1/32 and ::1/128) is implicitly trusted under
	// `auto` and does not need to be listed here.
	TrustedNetworks []string `yaml:"trusted_networks"`
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
	if c.Audio.SampleRate != 0 && (c.Audio.SampleRate < 4000 || c.Audio.SampleRate > 48_000) {
		return fmt.Errorf("audio.sample_rate %d outside 4000..48000", c.Audio.SampleRate)
	}
	if c.Audio.Volume != 0 && (c.Audio.Volume < 0 || c.Audio.Volume > 1) {
		return fmt.Errorf("audio.volume %f outside 0..1", c.Audio.Volume)
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
		switch ch.Tone.Mode {
		case "", "none":
		case "ctcss":
			if ch.Tone.CTCSSHz < 50 || ch.Tone.CTCSSHz > 300 {
				return fmt.Errorf("scanner.conventional[%d].tone.ctcss_hz %v outside 50..300 Hz",
					i, ch.Tone.CTCSSHz)
			}
		case "dcs":
			if len(ch.Tone.DCSCode) != 3 {
				return fmt.Errorf("scanner.conventional[%d].tone.dcs_code must be 3 octal digits", i)
			}
			for _, r := range ch.Tone.DCSCode {
				if r < '0' || r > '7' {
					return fmt.Errorf("scanner.conventional[%d].tone.dcs_code %q must be octal 0..7",
						i, ch.Tone.DCSCode)
				}
			}
		default:
			return fmt.Errorf("scanner.conventional[%d].tone.mode must be ctcss|dcs|none", i)
		}
	}
	return nil
}

// parseDurationFlexible accepts a Go duration string. Wrapped here so
// the dependency lives in one place and tests can lean on it.
func parseDurationFlexible(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
