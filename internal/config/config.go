package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Log         LogConfig         `yaml:"log"`
	SDR         SDRConfig         `yaml:"sdr"`
	Trunking    TrunkingConfig    `yaml:"trunking"`
	API         APIConfig         `yaml:"api"`
	Storage     StorageConfig     `yaml:"storage"`
	Recordings  RecordingsConfig  `yaml:"recordings"`
	Metrics     MetricsConfig     `yaml:"metrics"`
	Retention   RetentionConfig   `yaml:"retention"`
	ToneOut     ToneOutConfig     `yaml:"tone_out"`
	Scanner     ScannerConfig     `yaml:"scanner"`
	Audio       AudioConfig       `yaml:"audio"`
	Broadcast   BroadcastConfig   `yaml:"broadcast"`
	Baseband    BasebandConfig    `yaml:"baseband"`
	Paging      PagingConfig      `yaml:"paging"`
	APRS        APRSConfig        `yaml:"aprs"`
	AIS         AISConfig         `yaml:"ais"`
	DSC         DSCConfig         `yaml:"dsc"`
	MDC1200     MDC1200Config     `yaml:"mdc1200"`
	ADSB        ADSBConfig        `yaml:"adsb"`
	M17         M17Config         `yaml:"m17"`
	Web         WebConfig         `yaml:"web"`
	Diagnostics DiagnosticsConfig `yaml:"diagnostics"`
}

// DiagnosticsConfig controls error-reporting verbosity. When
// VerboseErrors is true, every error surface (CLI, daemon log,
// HTTP/gRPC API) prints the full wrapped error chain plus a goroutine
// stack dump under the diagnostics banner, with no interactive prompt;
// the API also expands its error envelopes to include the banner +
// trace (which exposes host/dongle info — enable only on trusted
// networks). When false (the default) the CLI instead offers the trace
// interactively on a TTY. Overridable at runtime by the -verbose-errors
// flag and the GOPHERTRUNK_VERBOSE_ERRORS env var.
type DiagnosticsConfig struct {
	VerboseErrors bool `yaml:"verbose_errors"`
}

// WebConfig configures the bundled user interfaces (the embedded web SPA
// and the terminal TUI). Tabs maps a tab key (e.g. "pagers", "metrics")
// to whether it is shown in the navigation. Absent keys default to
// visible, so an empty/omitted section shows everything. Set a key to
// false to turn that tab off — operators running GopherTrunk for a single
// task can declutter the UI to just the panels they care about. Hiding a
// tab only removes it from the nav strip; the route/panel is still
// reachable directly.
type WebConfig struct {
	Tabs map[string]bool `yaml:"tabs"`
}

// KnownUITabs is the canonical set of navigation tab keys both UIs
// understand. The key is the web route path minus its leading slash; the
// TUI maps the same keys onto its panels via state.PanelKind.Key(). The
// web SPA owns the full set; the TUI owns only the core subset, so hiding
// a web-only tab (pagers/aprs/…) is simply a no-op there. Keep this in
// sync with web/src/App.tsx (TABS + EXTRA_TABS).
var KnownUITabs = map[string]bool{
	"dashboard":     true,
	"active":        true,
	"scanner":       true,
	"settings":      true,
	"systems":       true,
	"talkgroups":    true,
	"rids":          true,
	"history":       true,
	"events":        true,
	"cc":            true,
	"tones":         true,
	"pagers":        true,
	"aprs":          true,
	"ais":           true,
	"dsc":           true,
	"adsb":          true,
	"mdc1200":       true,
	"spectrum":      true,
	"constellation": true,
	"bookmarks":     true,
	"metrics":       true,
	"devices":       true,
	"import":        true,
}

// HiddenTabs returns the sorted list of tab keys explicitly switched off
// (mapped to false). The result feeds /api/v1/runtime so both UIs can
// filter their navigation from a single source of truth.
func (w WebConfig) HiddenTabs() []string {
	var hidden []string
	for key, visible := range w.Tabs {
		if !visible {
			hidden = append(hidden, key)
		}
	}
	sort.Strings(hidden)
	return hidden
}

// ADSBConfig configures the ADS-B aircraft-tracking input. The
// native 1 Msps PPM DSP frontend is planned; for now the BEAST
// upstream lets operators consume Mode-S frames from a separately-
// running dump1090 / readsb / BeastSplitter / commercial hub. Most
// 1090 MHz receiver chains already run dump1090 on a dedicated
// RTL-SDR + 1090 MHz filter + LNA; pointing GopherTrunk at it
// is a one-line config away.
type ADSBConfig struct {
	BeastUpstreams []ADSBBeastConfig   `yaml:"beast_upstreams"`
	Channels       []ADSBChannelConfig `yaml:"channels"`
}

// ADSBChannelConfig describes one SDR pinned to 1090 MHz for the
// native PPM Mode-S receiver — the alternative to a BEAST upstream for
// operators who want GopherTrunk to own the whole 1090 MHz chain
// rather than running a separate dump1090 / readsb. Serial picks the
// SDR; the daemon tunes it to FrequencyHz (default 1090 MHz) and runs
// the PPM demodulator against its full IQ stream. A 1090 MHz SAW
// filter + LNA ahead of the SDR is strongly recommended — Mode-S is a
// weak, bursty signal. The SDR must sample at ≥ 2 Msps; the receiver
// resamples to 2 Msps internally. Decoded frames merge into the same
// events.KindAircraftReport stream the BEAST upstreams feed, so the
// /aircraft panel and storage are shared.
type ADSBChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"` // defaults to 1090 MHz when zero
}

// ADSBBeastConfig describes one BEAST upstream to consume. Addr is
// typically "host:30005" — the standard dump1090 / readsb BEAST
// output port. Multiple upstreams can run side-by-side (e.g. a
// local antenna + a remote hub at the airport) and their frames
// merge into the same `events.KindAircraftReport` stream.
type ADSBBeastConfig struct {
	Addr string `yaml:"addr"`
	Name string `yaml:"name"` // log + metrics label
}

// M17Config configures the M17 digital-voice link-layer receiver.
// Each entry pins an SDR to an M17 frequency and runs the DSP frontend
// (FM demod → C4FM matched filter → symbol-timing recovery → 4FSK
// slice → sync hunt → LICH reassembly → Link Setup Frame parse).
// Decoded link metadata (source / destination callsigns, mode)
// publishes on events.KindM17LinkSetup; storage.M17Log persists it to
// the m17_log table and the REST endpoint at /api/v1/m17/linksetups
// returns the recent rows. Voice (Codec2) decode is a later milestone.
type M17Config struct {
	Channels []M17ChannelConfig `yaml:"channels"`
}

// M17ChannelConfig describes one M17 channel to decode. Serial picks
// the SDR; the daemon tunes it to FrequencyHz and runs the receiver
// against its full IQ stream. M17 simplex calling is commonly
// 144.975 MHz (2 m) / 433.475 MHz (70 cm) in many regions.
type M17ChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
}

// APRSConfig configures the APRS / AX.25 Bell-202 AFSK receiver.
// Each entry pins an SDR to a 2 m / 70 cm APRS frequency and runs
// the DSP frontend (FM demod → FFSK discriminator → symbol-timing
// recovery → NRZI decode → HDLC framer → AX.25 + APRS info-field
// parsing) against its full IQ stream. Decoded packets publish on
// events.KindAPRSPacket; the storage.APRSLog subscriber persists
// them, the REST endpoint at /api/v1/aprs/packets and the /aprs
// web panel render them.
type APRSConfig struct {
	Channels []APRSChannelConfig `yaml:"channels"`
}

// APRSChannelConfig describes one APRS channel to decode. Serial
// picks the SDR; the daemon tunes it to FrequencyHz and runs the
// AFSK receiver against its full IQ stream. The 144.39 MHz North-
// America primary channel is the most common target; other
// regions use 144.575 (EU R1), 144.64 (JP), 144.80 (EU R1 short-
// distance), 145.825 (ISS digipeater), 144.575 (AU). The DropBadFCS
// and DropNonUI toggles match the receiver's options — leave both
// false to see marginal traffic on the panel (highlighted in
// yellow); flip them on if the channel is dominated by noise.
type APRSChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	DropBadFCS  bool   `yaml:"drop_bad_fcs"`
	DropNonUI   bool   `yaml:"drop_non_ui"`
}

// AISConfig configures the marine-AIS GMSK receiver. Each entry
// pins an SDR to one of the AIS channels (87B = 161.975 MHz,
// 88B = 162.025 MHz — class A vessels alternate between them
// every second) and runs the DSP frontend (FM demod → GFSK
// matched filter → symbol-timing recovery → NRZI decode → HDLC
// framer → ITU-R M.1371-5 message parser) against its full IQ
// stream. Decoded messages publish on events.KindAISMessage;
// storage.VesselLog persists them, the REST endpoint at
// /api/v1/ais/vessels and the /ais web panel render them.
type AISConfig struct {
	Channels []AISChannelConfig `yaml:"channels"`
}

// AISChannelConfig describes one AIS channel to decode. Serial
// picks the SDR; the daemon tunes it to FrequencyHz and runs the
// GMSK receiver against its full IQ stream. Most operators pin
// one SDR to 161.975 (channel 87B) and another to 162.025 (88B)
// to catch both halves of the class-A alternation; one channel
// is enough for class-B-only or quiet-area monitoring. The
// DropBadFCS and DropNonPosition toggles match the receiver's
// options.
type AISChannelConfig struct {
	Serial          string `yaml:"serial"`
	FrequencyHz     uint32 `yaml:"frequency_hz"`
	DropBadFCS      bool   `yaml:"drop_bad_fcs"`
	DropNonPosition bool   `yaml:"drop_non_position"`
}

// DSCConfig configures the marine Digital Selective Calling receiver.
// Each entry pins an SDR to a DSC channel — VHF channel 70 is
// 156.525 MHz; HF DSC rides 2187.5 / 8414.5 / 12577 / 16804.5 kHz
// among others — and runs the DSP frontend (FM demod → FFSK tone
// discriminator at 1300/2100 Hz → symbol-timing recovery → direct-FSK
// slicer → BCH(10,7) character sync → ITU-R M.493 sequence parser)
// against its full IQ stream. Decoded sequences publish on
// events.KindDSCMessage; storage.DSCLog persists them, the REST
// endpoint at /api/v1/dsc/messages and the /dsc web panel render them.
type DSCConfig struct {
	Channels []DSCChannelConfig `yaml:"channels"`
}

// DSCChannelConfig describes one DSC channel to decode. Serial picks
// the SDR; the daemon tunes it to FrequencyHz and runs the FFSK
// receiver against its full IQ stream. The VHF calling channel 70
// (156.525 MHz) is the most common target — it carries distress /
// urgency / safety alerts and the routine call-ups that precede a
// voice working-channel hand-off. DropBadFCS matches the receiver's
// option: leave it false to see BCH-marginal sequences on the panel
// (flagged), flip it on for noisy channels.
type DSCChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	DropBadFCS  bool   `yaml:"drop_bad_fcs"`
}

// MDC1200Config configures the Motorola MDC1200 FFSK signaling
// receiver. Each entry pins an SDR to a conventional analog VHF / UHF
// voice channel and runs the DSP frontend (FM demod → FFSK
// discriminator at 1200/1800 Hz → symbol-timing recovery → NRZ
// slicer → sync framer → op/arg/unit-ID parser) against its full IQ
// stream. Decoded bursts publish on events.KindMDC1200Message;
// storage.MDC1200Log persists them, the REST endpoint at
// /api/v1/mdc1200/messages and the /mdc1200 web panel render them.
type MDC1200Config struct {
	Channels []MDC1200ChannelConfig `yaml:"channels"`
}

// MDC1200ChannelConfig describes one MDC1200 channel to decode. Serial
// picks the SDR; the daemon tunes it to FrequencyHz and runs the FFSK
// receiver against its full IQ stream. Target the conventional analog
// voice channels of the systems you monitor — MDC1200 bursts ride at
// the head (and optionally tail) of each transmission. DropBadCRC
// matches the receiver's option — leave it false to see CRC-failed
// bursts on the panel (flagged), flip it on for noisy channels.
type MDC1200ChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	DropBadCRC  bool   `yaml:"drop_bad_crc"`
}

// PagingConfig configures pager decoders (POCSAG today, FLEX
// follow-up). Each entry pins an SDR to a paging frequency and
// runs the per-protocol receiver against its IQ stream.
type PagingConfig struct {
	POCSAG []PagingPOCSAGConfig `yaml:"pocsag"`
	FLEX   []PagingFLEXConfig   `yaml:"flex"`
}

// PagingFLEXConfig describes one FLEX paging channel to decode. Serial
// picks the SDR; the daemon tunes it to FrequencyHz and runs the FLEX
// receiver against its full IQ stream. The frontend handles the
// 1600 bps / 2-level mode. Decoded pages publish on
// events.KindPagerMessage with protocol="flex" and share the pager_log
// table / web panel with POCSAG.
type PagingFLEXConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
}

// PagingPOCSAGConfig describes one POCSAG paging channel to
// decode. Serial picks the SDR; the daemon tunes it to FrequencyHz
// and runs the POCSAG receiver against its full IQ stream. Baud
// defaults to 1200 — the most common POCSAG rate; configure 512
// for legacy networks (e.g. some commercial paging providers) or
// 2400 for higher-throughput systems (DAPNET).
type PagingPOCSAGConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	BaudHz      uint32 `yaml:"baud_hz"`
}

// BasebandConfig configures wideband IQ recording and offline replay.
// Empty == disabled. `record` taps live tuners and writes their IQ to
// WAV; `replay` mounts recorded WAVs as virtual tuners so a capture can
// be decoded offline. Replay recordings should have been made at the
// same rate as sdr.sample_rate for real-time-correct playback.
type BasebandConfig struct {
	Record []BasebandRecordConfig `yaml:"record"`
	Replay []BasebandReplayConfig `yaml:"replay"`
}

// BasebandRecordConfig taps one tuner's live IQ to WAV recordings.
type BasebandRecordConfig struct {
	// Serial is the SDR serial whose IQ stream is recorded.
	Serial string `yaml:"serial"`
	// Dir is the directory recordings are written into.
	Dir string `yaml:"dir"`
}

// BasebandReplayConfig mounts one recorded WAV as a virtual tuner.
type BasebandReplayConfig struct {
	// File is the path to the baseband WAV recording.
	File string `yaml:"file"`
	// Serial is the virtual device serial the pool reports. Empty
	// generates one.
	Serial string `yaml:"serial"`
	// Role is the pool role: control|voice|auto (empty = auto).
	Role string `yaml:"role"`
	// Loop restarts the recording on EOF so the offline tuner is a
	// continuous source. nil defaults to true.
	Loop *bool `yaml:"loop"`
}

// BroadcastConfig configures the outbound call-streaming subsystem
// (internal/broadcast): completed calls are encoded to MP3 and uploaded
// to call aggregators or pushed to a live Icecast/ShoutCast mountpoint.
// Empty == disabled; the daemon runs no broadcast manager when no feed
// is configured.
type BroadcastConfig struct {
	// MinDurationMs drops calls shorter than this from every feed
	// (squelch crackle, failed decodes). 0 streams calls of any
	// length.
	MinDurationMs int `yaml:"min_duration_ms"`
	// Workers is the number of concurrent upload goroutines. 0 uses
	// the broadcast package default.
	Workers int `yaml:"workers"`
	// Broadcastify, RdioScanner, OpenMHz and Icecast each list zero
	// or more feeds. A feed with enabled=false is parsed but skipped.
	Broadcastify []BroadcastifyFeedConfig `yaml:"broadcastify"`
	RdioScanner  []RdioScannerFeedConfig  `yaml:"rdioscanner"`
	OpenMHz      []OpenMHzFeedConfig      `yaml:"openmhz"`
	Icecast      []IcecastFeedConfig      `yaml:"icecast"`
}

// BroadcastifyFeedConfig is one Broadcastify Calls upload feed.
type BroadcastifyFeedConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Name     string   `yaml:"name"`
	APIKey   string   `yaml:"api_key"`
	SystemID int      `yaml:"system_id"`
	Systems  []string `yaml:"systems"` // empty = every system
}

// RdioScannerFeedConfig is one RdioScanner call-upload feed.
type RdioScannerFeedConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Name     string   `yaml:"name"`
	URL      string   `yaml:"url"`
	APIKey   string   `yaml:"api_key"`
	SystemID int      `yaml:"system_id"`
	Systems  []string `yaml:"systems"`
}

// OpenMHzFeedConfig is one OpenMHz upload feed.
type OpenMHzFeedConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Name      string   `yaml:"name"`
	APIKey    string   `yaml:"api_key"`
	ShortName string   `yaml:"short_name"`
	Systems   []string `yaml:"systems"`
}

// IcecastFeedConfig is one live Icecast/ShoutCast feed.
type IcecastFeedConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Name       string   `yaml:"name"`
	Host       string   `yaml:"host"`
	Port       int      `yaml:"port"`
	Mount      string   `yaml:"mount"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
	StreamName string   `yaml:"stream_name"`
	Systems    []string `yaml:"systems"`
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
	Mode        string  `yaml:"mode"`         // "fm" | "nfm"
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
	// MessageLog configures the optional decoded-message log — a
	// human-readable, per-event text log of trunking activity
	// (grants, lock/loss, affiliations, patches, …), the analogue
	// of SDRtrunk's per-channel decoded message log.
	MessageLog MessageLogConfig `yaml:"message_log"`
}

// MessageLogConfig configures the decoded-message log. Empty Path (or
// Enabled false) disables it.
type MessageLogConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Path      string `yaml:"path"`
	MaxSizeMB int    `yaml:"max_size_mb"` // default 16
}

type SDRConfig struct {
	// SampleRate is the IQ rate (Hz) every tuner is programmed to.
	// Default 2_400_000 (2.4 MS/s). Valid range 225_000..3_200_000; the
	// RTL2832U quantizes to its 28.4 fixed-point divisor so the streamed
	// rate may differ slightly (see Device.ActualSampleRate). This is
	// also the primary load lever on CPU-bound hosts: convert + resample
	// cost scales with it, so if the daemon logs "sdr: dropping live IQ
	// chunks; consumer can't keep up" (iq_underruns_total climbing),
	// lowering it — e.g. to 1_024_000 — roughly halves per-chunk decode
	// work. Running fewer simultaneous dongles on a weak CPU has the same
	// effect.
	SampleRate uint32         `yaml:"sample_rate"`
	Devices    []DeviceConfig `yaml:"devices"`
	// RTLTCP lists remote rtl_tcp endpoints (host:port + optional
	// per-endpoint metadata) to mount as virtual tuners. Each entry
	// becomes a pool device alongside any locally-attached USB
	// dongles. Useful when the SDR hardware lives on a different
	// host from the daemon (e.g. a Raspberry Pi by the antenna +
	// a beefier machine for decode). rtl_tcp is plaintext — use it
	// on trusted networks only or through an SSH/wireguard tunnel.
	RTLTCP []RTLTCPConfig `yaml:"rtl_tcp"`
	// WatchdogIntervalMs governs the periodic USB-disconnect
	// watchdog that the SDR pool runs while the daemon is up. It
	// polls the registered drivers, surfaces serials that vanish
	// from the bus, and calls Pool.Reacquire on serials that
	// reappear so the next consumer touches a live handle instead
	// of the stale one. Zero (default) selects
	// sdr.DefaultWatchdogInterval (30 s). Negative disables the
	// watchdog entirely — useful when a host with intentionally
	// slow USB enumeration sees the periodic enumerate as a tax.
	// In-stream IQ-death recovery (ccdecoder retry loop, voice
	// Bind reacquire) is unaffected by this knob.
	WatchdogIntervalMs int `yaml:"watchdog_interval_ms"`
}

// RTLTCPConfig describes one remote rtl_tcp endpoint to expose as
// a virtual tuner. Addr is required; Serial / Role follow the same
// semantics as the local SDR devices block.
type RTLTCPConfig struct {
	// Addr is the host:port pair the rtl_tcp server is listening
	// on, e.g. "192.168.1.50:1234". Required.
	Addr string `yaml:"addr"`
	// Serial is the virtual device serial reported on the pool's
	// /api/v1/devices snapshot. Empty generates one from Addr.
	Serial string `yaml:"serial"`
	// Role hints the pool's role assignment: control|voice|auto.
	Role string `yaml:"role"`
	// PPM is the frequency-correction tuning sent to the remote on
	// open (the remote's local rtlsdr layer applies it). Optional;
	// zero matches every TCXO-equipped dongle.
	PPM int `yaml:"ppm"`
	// Gain follows the same rule as DeviceConfig.Gain — "auto" /
	// "" selects AGC, any other value parses as tenths of dB.
	Gain string `yaml:"gain"`
	// BiasTee toggles the remote dongle's 5 V bias-tee output.
	// Honoured only by servers running librtlsdr ≥ 0.7; older
	// servers silently ignore the command.
	BiasTee bool `yaml:"bias_tee"`
	// ConnectTimeoutMs caps the TCP dial in milliseconds. Zero
	// picks the driver default (3000).
	ConnectTimeoutMs int `yaml:"connect_timeout_ms"`
}

type DeviceConfig struct {
	Serial string `yaml:"serial"`
	Role   string `yaml:"role"`
	PPM    int    `yaml:"ppm"`
	// Gain is the tuner gain setting. "auto" (or empty) selects
	// the dongle's automatic gain control; any other value is
	// parsed as a tenths-of-dB integer matching librtlsdr's
	// gain table (e.g. "496" → 49.6 dB). Use `gophertrunk sdr
	// list` to see the supported values per device.
	Gain string `yaml:"gain"`
	// BiasTee enables the dongle's 5V bias-tee output, used to
	// power external LNAs through the antenna SMA. Off by
	// default. Most modern RTL-SDR clones (e.g. NooElec NESDR
	// Smart v5) wire this through; older units may toggle a
	// GPIO bit that goes nowhere — librtlsdr accepts the call
	// either way.
	BiasTee bool `yaml:"bias_tee"`

	// CenterFreqHz pins a `role: wideband` dongle to the centre of
	// the IQ band it should cover. Every Channels[].FrequencyHz must
	// fall within ±sample_rate/2 of this value, with a 5 % guard.
	// Required for wideband; ignored for other roles.
	CenterFreqHz uint32 `yaml:"center_freq_hz"`

	// TunerStrategy picks the DSP layout that extracts each per-
	// repeater narrow-band stream from the dongle's wide IQ stream:
	//   - ""        / "auto"      — auto-pick by Channel count
	//                                (≤ 6 channels: ddc; otherwise
	//                                polyphase)
	//   - "ddc"                   — independent NCO mixer + rational
	//                                resampler per channel.
	//   - "polyphase"             — shared M-channel polyphase
	//                                channelizer + fine-tune DDC.
	// Ignored for non-wideband roles. See internal/dsp/tuner for the
	// trade-offs.
	TunerStrategy string `yaml:"tuner_strategy"`

	// Channels is the list of repeater carriers a wideband dongle
	// should monitor inside its IQ band. Each entry binds a
	// frequency to a configured trunking.systems[].name. Ignored
	// for non-wideband roles.
	Channels []DeviceChannelConfig `yaml:"channels"`

	// VoiceTaps is the number of per-grant DDC tuners the daemon
	// allocates from this wideband dongle's IQ stream so trunked
	// voice grants can be followed without retuning a separate
	// `role: voice` SDR. Each tap subscribes to the dongle's
	// iqtap broker on demand and emits 48 kHz IQ centred on the
	// grant frequency.
	//
	// Defaults to 0 (no virtual voice taps; voice grants route to
	// the physical voice pool). Set to 2-4 on a wideband dongle
	// hosting a trunked CC tap (DMR T3, P25 Phase 1, P25 Phase 2)
	// so one SDR can cover the full system end-to-end. Out-of-
	// window grants surface ErrOutOfBand and fall back to a
	// physical voice SDR when one is configured. Capped at 8 to
	// keep CPU bounded.
	VoiceTaps int `yaml:"voice_taps"`

	// IQCorrect enables blind I/Q-imbalance correction on this device's
	// raw IQ before decimation (issue #402). Off by default. An
	// uncorrected RTL-SDR I/Q imbalance distorts the demodulated symbol
	// eye (worst at the on-channel DC the control decoder's DDC sits on);
	// validate the benefit with `gophertrunk replay -iq-correct -diag`
	// on a capture from this device before enabling it here.
	IQCorrect bool `yaml:"iq_correct"`
}

// DeviceChannelConfig is one repeater carrier carried by a
// `role: wideband` dongle. FrequencyHz must lie inside the dongle's
// IQ band (CenterFreqHz ± sample_rate/2 minus a guard); System must
// match an existing trunking.systems[].name with a supported
// per-channel protocol.
type DeviceChannelConfig struct {
	FrequencyHz uint32 `yaml:"frequency_hz"`
	System      string `yaml:"system"`
}

type TrunkingConfig struct {
	Systems []SystemConfig `yaml:"systems"`

	// CallTimeoutMs is the inactivity window after which the engine's
	// watchdog ends a call (publishes CallEnd with EndReasonTimeout
	// and releases the bound voice SDR). The watchdog only fires when
	// no voice frames have been decoded for this long — see
	// internal/voice/composer for the per-protocol activity gate.
	// Defaults to 30 000 (30 s) when zero. Negative values are
	// rejected by Validate; setting it explicitly lets operators tune
	// teardown on systems whose signaling is consistently clean
	// (lower) or chatty with long pauses (higher). Issue #356.
	CallTimeoutMs int `yaml:"call_timeout_ms"`
}

type SystemConfig struct {
	Name            string   `yaml:"name"`
	Protocol        string   `yaml:"protocol"`
	ControlChannels []uint32 `yaml:"control_channels"`
	TalkgroupFile   string   `yaml:"talkgroup_file"`
	// RIDAliasFile is the optional path to a per-system CSV or JSON
	// catalogue of radio-ID (subscriber unit) aliases — the per-RID
	// equivalent of TalkgroupFile. CSV format: a Decimal/DEC/ID column
	// plus optional Alias/AlphaTag, Description, Tag, Group, Owner,
	// Priority, Lockout, Watch, Icon columns. JSON format: an array
	// of {id, alias, description, ...} objects. Empty leaves the RID
	// catalogue blank for this system (live observations still
	// surface via the affiliation tracker).
	RIDAliasFile string `yaml:"rid_alias_file"`

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

	// P25Phase1DemodMode selects the symbol-recovery path for the
	// P25 Phase 1 receiver. Recognised values: "" / "c4fm" / "fm"
	// (the default — FM discriminator + 4-level slicer; matches
	// every previously shipping config and works on conventional
	// non-simulcast P25 transmitters) or "cqpsk" / "lsm" / "linear"
	// (the linear / LSM path — complex RRC + Gardner + differential
	// QPSK; required for simulcast P25 deployments whose control
	// channel transmits Linear Simulcast Modulation rather than
	// straight C4FM, see issue #275 and TIA-102.BAAA). Applies to
	// both the control channel decoder and the per-call voice
	// chain — without the voice-chain side a simulcast site would
	// lock the CC fine but never decode an LDU on a granted voice
	// call (issue #356 follow-up). Ignored for non-P25-Phase-1
	// protocols.
	P25Phase1DemodMode string `yaml:"p25_phase1_demod_mode"`
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
	// non-zero before parsing). Ignored for non-P25-Phase-2
	// protocols.
	P25Phase2RSMode string `yaml:"p25_phase2_rs_mode"`
	// P25Phase2InterleaveMode enables the TIA-102.BBAC per-burst block
	// deinterleaver applied to the MAC-burst dibits before trellis
	// decoding. Recognised values: "" / "off" / "false" / "0" (the
	// default — no deinterleave; matches synthesized-fixture
	// expectations) or "on" / "true" / "1". Ignored for
	// non-P25-Phase-2 protocols.
	P25Phase2InterleaveMode string `yaml:"p25_phase2_interleave_mode"`
	// P25Phase2ScramblerMode enables the PN44 descrambling layer
	// per TIA-102.BBAC-1 §7.2.5 on top of the trellis-decoded MAC
	// PDU. Recognised values: "" / "on" / "true" / "1" (the
	// default — every on-air P25 Phase 2 MAC PDU is PN44 scrambled,
	// so descrambling is required for live decode; XOR the
	// trellis-decoded 144-bit MAC PDU with the leading 144 bits of
	// the PN44 sequence) or "off" / "false" / "0" (no PN44
	// descrambling; the opt-out for synthesized, unscrambled
	// fixtures). The scrambler seed is derived from (WACN, SystemID,
	// Color Code = NAC) per spec equation (5); the zero-seed edge
	// case maps to (2^44 - 1). Ignored for non-P25-Phase-2 protocols.
	P25Phase2ScramblerMode string `yaml:"p25_phase2_scrambler_mode"`
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
	// NXDNDeviationHz overrides the peak frequency deviation (Hz)
	// the NXDN receiver's slicer is calibrated against. The Common
	// Air Interface spec value is 1800 Hz (matched against the
	// FM-discriminator output level so live captures slice
	// correctly). Some on-air transmitters deviate from spec —
	// captures whose dibit distribution is bimodal (outer ±3 levels
	// dominate, inner ±1 underrepresented) usually want a higher
	// value (e.g., 2400 Hz). Zero / unset uses the spec default.
	// Ignored for non-NXDN protocols.
	NXDNDeviationHz float64 `yaml:"nxdn_deviation_hz,omitempty"`
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
	// MPT1327CWSCTolerance sets the Hamming-distance threshold the
	// Process adapter uses when scanning for the 16-bit Codeword
	// Synchronisation Code that precedes every MPT 1327 message.
	// Recognised values: "" → default 2-bit tolerance (matches
	// commercial MPT 1327 receivers on noisy on-air captures);
	// "0" / "exact" / "off" → exact match (use for pre-stripped
	// synthesized fixtures); a decimal integer in [0, 15] for
	// custom thresholds. Ignored for non-MPT-1327 protocols.
	MPT1327CWSCTolerance string `yaml:"mpt1327_cwsc_tolerance"`
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

	// P25BandPlan seeds the Phase 1 receiver's BandPlan with static
	// IdentifierUpdate slot entries — the operator's escape hatch for
	// sites that route grants through a channel ID they never
	// broadcast an IDEN_UP TSBK for (issue #345). Over-the-air
	// IDEN_UPs take precedence; entries here are the startup floor.
	// Ignored for non-P25-Phase-1 protocols.
	P25BandPlan []P25BandPlanEntryConfig `yaml:"p25_band_plan"`

	// EncryptionKeys lists operator-supplied decryption keys for this
	// system. GopherTrunk decrypts only with keys the operator
	// already holds and is authorized to use — it performs no key
	// recovery. Today only DMR ARC4/RC4 ("Enhanced Privacy") is
	// recognised; the per-key `algorithm` field keeps the schema open
	// so AES can be added later without a config break. Ignored for
	// protocols without an encryption decoder. See issue #276.
	EncryptionKeys []EncryptionKeyConfig `yaml:"encryption_keys"`
}

// P25BandPlanEntryConfig is one operator-supplied IDEN_UP slot seed
// for the Phase 1 receiver. ChannelID is the 4-bit IDEN_UP slot index
// (0..15). BaseHz / SpacingHz / TxOffsetHz / BandwidthHz mirror the
// on-air IDEN_UP fields per TIA-102.AABF — see
// internal/radio/p25/phase1/identifier.go for the bit layout. Most
// operators only need to populate ChannelID + BaseHz + SpacingHz +
// TxOffsetHz; BandwidthHz is informational and BandPlan.Frequency
// does not consult it.
type P25BandPlanEntryConfig struct {
	ChannelID   uint8  `yaml:"channel_id"`
	BaseHz      uint64 `yaml:"base_hz"`
	SpacingHz   uint32 `yaml:"spacing_hz"`
	TxOffsetHz  int64  `yaml:"tx_offset_hz"`
	BandwidthHz uint32 `yaml:"bandwidth_hz"`
}

// EncryptionKeyConfig is one operator-supplied decryption key for a
// trunking system. KeyID matches the key identifier the radios carry
// in the protocol's privacy header, so a system that rotates between
// several keys still resolves to the right one. Key is the raw key
// hex-encoded; surrounding whitespace, internal spaces, and an
// optional "0x" prefix are tolerated.
type EncryptionKeyConfig struct {
	KeyID     uint16 `yaml:"key_id"`
	Algorithm string `yaml:"algorithm"`
	Key       string `yaml:"key"`
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
	HTTPAddr       string        `yaml:"http_addr"`
	GRPCAddr       string        `yaml:"grpc_addr"`
	AllowMutations bool          `yaml:"allow_mutations"`
	Auth           APIAuthConfig `yaml:"auth"`
	// Rigctld, when non-empty, exposes the control SDR's tuning over
	// the Hamlib rigctld TCP wire protocol on this address. Lets
	// external amateur-radio tooling (Cloudlog, logging programs,
	// satellite trackers) read and set the daemon's frequency
	// without learning the GopherTrunk REST API. Defaults to empty
	// (off). Typical value: "127.0.0.1:4532" (the rigctld default
	// port). The server is read-only beyond SetFreq; PTT is
	// always reported as 0. Bind to loopback unless the network
	// is trusted — the protocol has no authentication.
	Rigctld string `yaml:"rigctld"`
	// CORS gates cross-origin browser requests. Off by default
	// (no Access-Control-* headers emitted). Enable when serving
	// the bundled web UI from a different origin than the daemon
	// (e.g. opening web/index.html via file:// → Origin: null, or
	// hosting the SPA on a separate static server).
	CORS APICORSConfig `yaml:"cors"`
	// TLSCert / TLSKey, when both set, switch both the HTTP and
	// gRPC servers to TLS. Paths point at PEM-encoded files on
	// disk that the daemon reads at start-up (rotation requires a
	// restart). Leave both empty for plain TCP (the default;
	// appropriate for loopback / private-network deployments).
	// See docs/hardening.md §"Transport encryption (TLS)".
	TLSCert string `yaml:"tls_cert"`
	TLSKey  string `yaml:"tls_key"`
}

// APICORSConfig configures cross-origin browser access to the HTTP
// API + WebSocket upgrade. Off by default; the daemon emits no
// Access-Control-* headers and rejects WS upgrades whose Origin
// header is not in AllowedOrigins.
//
// Common values:
//
//	["null"]                       allow web UI opened via file://
//	["http://laptop.local:8000"]   allow a specific static host
//	["*"]                          allow any origin (use with auth)
type APICORSConfig struct {
	// AllowedOrigins is the exact origin string the daemon
	// echoes back in Access-Control-Allow-Origin. Browsers send
	// the literal "null" for file:// loads. Use "*" to allow
	// any origin (must not be combined with credentials).
	AllowedOrigins []string `yaml:"allowed_origins"`
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
	Dir        string `yaml:"dir"`
	SampleRate uint32 `yaml:"sample_rate"`
	WriteRaw   bool   `yaml:"write_raw"`
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
	CallLogDays int `yaml:"call_log_days"`
	// LogDays sweeps the decoder log tables (pager_log, aprs_log,
	// vessel_log, dsc_log, aircraft_log, mdc1200_log, m17_log,
	// location_log): rows older than this many days are deleted. Zero
	// (the default) disables the decoder-log sweep.
	LogDays   int    `yaml:"log_days"`
	FilesDays int    `yaml:"files_days"`
	Interval  string `yaml:"interval"` // Go duration string; default 1h
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
		// HTTP API on by default so the bundled launcher's TUI /
		// web paths have something to attach to without an explicit
		// config edit. Loopback bind keeps the auth-disabled default
		// (see api.ParseAuthMode) safe out-of-the-box; operators on
		// a closed LAN flip this to ":8080" or a LAN IP.
		API: APIConfig{HTTPAddr: "127.0.0.1:8080"},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("config %s: %w\n  hint: check YAML syntax (indentation must be spaces, keys end with ':'). Run `gophertrunk import-pdf -wizard` to regenerate a fresh scaffold.", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("config %s: %w", path, err)
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.SDR.SampleRate != 0 && (c.SDR.SampleRate < 225_000 || c.SDR.SampleRate > 3_200_000) {
		return errors.New("sdr.sample_rate must be between 225 kHz and 3.2 MHz")
	}
	seenSerials := make(map[string]int, len(c.SDR.Devices))
	for i, d := range c.SDR.Devices {
		switch d.Role {
		case "", "control", "voice", "auto", "wideband":
		default:
			return fmt.Errorf("sdr.devices[%d]: role must be control|voice|auto|wideband", i)
		}
		if d.Role == "wideband" {
			if err := validateWidebandDevice(i, d, c.SDR.SampleRate, c.Trunking.Systems); err != nil {
				return err
			}
		}
		if d.Serial == "" {
			continue
		}
		if prev, dup := seenSerials[d.Serial]; dup {
			return fmt.Errorf(
				"sdr.devices[%d]: duplicate serial %q (also at sdr.devices[%d]) — "+
					"one physical SDR cannot serve multiple roles; P25 trunking needs "+
					"separate dongles for control and voice",
				i, d.Serial, prev)
		}
		seenSerials[d.Serial] = i
	}
	// Validate rtl_tcp endpoints. Addr is required; role must match
	// the standard set; serial collisions with local devices are
	// rejected for the same reason serial dedup runs above.
	for i, r := range c.SDR.RTLTCP {
		if r.Addr == "" {
			return fmt.Errorf("sdr.rtl_tcp[%d]: addr is required (host:port)", i)
		}
		switch r.Role {
		case "", "control", "voice", "auto":
		default:
			return fmt.Errorf("sdr.rtl_tcp[%d]: role must be control|voice|auto", i)
		}
		if r.Serial == "" {
			continue
		}
		if prev, dup := seenSerials[r.Serial]; dup {
			return fmt.Errorf(
				"sdr.rtl_tcp[%d]: serial %q collides with sdr.devices[%d]",
				i, r.Serial, prev)
		}
		seenSerials[r.Serial] = i
	}
	if c.Trunking.CallTimeoutMs < 0 {
		return fmt.Errorf("trunking.call_timeout_ms: %d ms must be ≥ 0", c.Trunking.CallTimeoutMs)
	}
	for i, s := range c.Trunking.Systems {
		if s.Name == "" {
			return fmt.Errorf("trunking.systems[%d]: name required", i)
		}
		if _, err := trunking.ParseProtocol(s.Protocol); err != nil {
			return fmt.Errorf("trunking.systems[%d]: %w", i, err)
		}
		seenBandPlanIDs := make(map[uint8]int, len(s.P25BandPlan))
		for k, e := range s.P25BandPlan {
			if e.ChannelID > 15 {
				return fmt.Errorf("trunking.systems[%d].p25_band_plan[%d]: channel_id %d outside 0..15", i, k, e.ChannelID)
			}
			if prev, dup := seenBandPlanIDs[e.ChannelID]; dup {
				return fmt.Errorf("trunking.systems[%d].p25_band_plan[%d]: duplicate channel_id %d (also at p25_band_plan[%d])", i, k, e.ChannelID, prev)
			}
			seenBandPlanIDs[e.ChannelID] = k
			if e.SpacingHz == 0 {
				return fmt.Errorf("trunking.systems[%d].p25_band_plan[%d]: spacing_hz required (nonzero)", i, k)
			}
			if e.BaseHz == 0 {
				return fmt.Errorf("trunking.systems[%d].p25_band_plan[%d]: base_hz required (nonzero)", i, k)
			}
		}
		seenKeyIDs := make(map[uint16]struct{}, len(s.EncryptionKeys))
		for k, ek := range s.EncryptionKeys {
			switch strings.ToLower(strings.TrimSpace(ek.Algorithm)) {
			case "rc4", "arc4":
				// supported
			case "":
				return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: algorithm is required (use \"rc4\")", i, k)
			case "aes", "des":
				return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: algorithm %q is not supported yet (only \"rc4\")", i, k, ek.Algorithm)
			default:
				return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: unknown algorithm %q (use \"rc4\")", i, k, ek.Algorithm)
			}
			if _, dup := seenKeyIDs[ek.KeyID]; dup {
				return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: duplicate key_id %d", i, k, ek.KeyID)
			}
			seenKeyIDs[ek.KeyID] = struct{}{}
			b, err := decodeHexKey(ek.Key)
			if err != nil {
				return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: %w", i, k, err)
			}
			if len(b) > 32 {
				return fmt.Errorf("trunking.systems[%d].encryption_keys[%d]: key is %d bytes, must be 1..32", i, k, len(b))
			}
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
	if err := c.Broadcast.validate(); err != nil {
		return err
	}
	for i, r := range c.Baseband.Record {
		if r.Serial == "" {
			return fmt.Errorf("baseband.record[%d]: serial required", i)
		}
		if r.Dir == "" {
			return fmt.Errorf("baseband.record[%d]: dir required", i)
		}
	}
	for i, r := range c.Baseband.Replay {
		if r.File == "" {
			return fmt.Errorf("baseband.replay[%d]: file required", i)
		}
		switch r.Role {
		case "", "control", "voice", "auto":
		default:
			return fmt.Errorf("baseband.replay[%d]: role must be control|voice|auto", i)
		}
	}
	for key := range c.Web.Tabs {
		if !KnownUITabs[key] {
			valid := make([]string, 0, len(KnownUITabs))
			for k := range KnownUITabs {
				valid = append(valid, k)
			}
			sort.Strings(valid)
			return fmt.Errorf("web.tabs: unknown tab %q (valid: %s)", key, strings.Join(valid, ", "))
		}
	}
	return nil
}

// widebandGuardFrac reserves this fraction of the dongle's IQ band at
// each edge as a guard against alias roll-off. Channel frequencies
// outside the resulting usable interval are rejected at config load.
// Mirrors the default passed to internal/dsp/tuner.NewDDCBank.
const widebandGuardFrac = 0.05

// validateWidebandDevice checks a wideband SDR entry's centre-freq,
// strategy, and channel list. sampleRateHz may be zero — Validate has
// already accepted that as "fall back to the pool default" — in which
// case the in-band check uses sdr.DefaultSampleRateHz so a missing
// rate doesn't bypass the per-channel sanity check.
//
// Each channel must reference a system whose protocol is either:
//   - "dmr-tier2" — Tier II conventional; the channel frequency is one
//     repeater carrier.
//   - "dmr"       — Tier III trunked; the channel frequency must match
//     one of the system's control_channels (the wideband dongle is
//     hosting that CC).
func validateWidebandDevice(idx int, d DeviceConfig, sampleRateHz uint32, systems []SystemConfig) error {
	if d.Serial == "" {
		return fmt.Errorf("sdr.devices[%d]: role: wideband requires serial (the daemon binds the channel list to the device by USB serial)", idx)
	}
	if d.VoiceTaps < 0 || d.VoiceTaps > 8 {
		return fmt.Errorf("sdr.devices[%d]: voice_taps %d out of range; 0 disables, 1-8 allocate that many virtual voice DDC taps on the dongle",
			idx, d.VoiceTaps)
	}
	if d.CenterFreqHz == 0 {
		return fmt.Errorf("sdr.devices[%d]: role: wideband requires center_freq_hz", idx)
	}
	switch d.TunerStrategy {
	case "", "auto", "ddc", "polyphase":
	default:
		return fmt.Errorf("sdr.devices[%d]: tuner_strategy must be auto|ddc|polyphase, got %q", idx, d.TunerStrategy)
	}
	if len(d.Channels) == 0 {
		return fmt.Errorf("sdr.devices[%d]: role: wideband requires at least one channel", idx)
	}
	rate := sampleRateHz
	if rate == 0 {
		rate = 2_048_000 // sdr.DefaultSampleRateHz; avoid an import cycle by repeating it
	}
	usableHalfBand := float64(rate) * (0.5 - widebandGuardFrac)
	systemsByName := make(map[string]SystemConfig, len(systems))
	for _, s := range systems {
		systemsByName[s.Name] = s
	}
	seenFreq := make(map[uint32]int, len(d.Channels))
	for j, ch := range d.Channels {
		if ch.FrequencyHz == 0 {
			return fmt.Errorf("sdr.devices[%d].channels[%d]: frequency_hz required", idx, j)
		}
		if ch.System == "" {
			return fmt.Errorf("sdr.devices[%d].channels[%d]: system required", idx, j)
		}
		sys, ok := systemsByName[ch.System]
		if !ok {
			return fmt.Errorf("sdr.devices[%d].channels[%d]: system %q is not declared in trunking.systems", idx, j, ch.System)
		}
		switch sys.Protocol {
		case "dmr-tier2", "dmr_tier2", "dmr-t2", "dmrtier2":
			// Tier II conventional - channel freq is a repeater carrier,
			// no relationship to system.ControlChannels required.
		case "dmr", "p25", "p25-phase2", "p25_phase2", "p25p2":
			// Trunked control-channel protocols — the wideband channel
			// MUST be one of the system's declared control channels.
			// Tier III DMR's CSBK chain, P25 Phase 1's TSBK chain, and
			// P25 Phase 2's H-DQPSK MAC chain all run on a frequency
			// the system advertises in control_channels; voice grants
			// hop elsewhere.
			matched := false
			for _, cc := range sys.ControlChannels {
				if cc == ch.FrequencyHz {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf(
					"sdr.devices[%d].channels[%d]: frequency_hz %d does not match any of system %q's "+
						"control_channels %v (wideband %s channels must sit on a declared control channel)",
					idx, j, ch.FrequencyHz, ch.System, sys.ControlChannels, sys.Protocol)
			}
		default:
			return fmt.Errorf(
				"sdr.devices[%d].channels[%d]: system %q has protocol %q; wideband currently supports "+
					"dmr-tier2 (Tier II conventional), dmr (Tier III trunked control channel), "+
					"p25 (Phase 1 trunked control channel), and p25-phase2 (Phase 2 trunked control channel)",
				idx, j, ch.System, sys.Protocol)
		}
		offset := float64(ch.FrequencyHz) - float64(d.CenterFreqHz)
		if offset > usableHalfBand || offset < -usableHalfBand {
			return fmt.Errorf(
				"sdr.devices[%d].channels[%d]: frequency_hz %d is %.1f kHz from center; usable band is ±%.1f kHz "+
					"(sample_rate %d Hz minus %.0f%% guard)",
				idx, j, ch.FrequencyHz, offset/1000, usableHalfBand/1000, rate, widebandGuardFrac*100)
		}
		if prev, dup := seenFreq[ch.FrequencyHz]; dup {
			return fmt.Errorf("sdr.devices[%d].channels[%d]: duplicate frequency_hz %d (also at channels[%d])", idx, j, ch.FrequencyHz, prev)
		}
		seenFreq[ch.FrequencyHz] = j
	}
	return nil
}

// validate checks that every enabled broadcast feed carries the fields
// its backend requires. Disabled feeds are left unchecked so operators
// can pre-stage credentials.
func (b BroadcastConfig) validate() error {
	if b.MinDurationMs < 0 {
		return errors.New("broadcast.min_duration_ms must not be negative")
	}
	for i, f := range b.Broadcastify {
		if !f.Enabled {
			continue
		}
		if f.APIKey == "" {
			return fmt.Errorf("broadcast.broadcastify[%d]: api_key required", i)
		}
		if f.SystemID == 0 {
			return fmt.Errorf("broadcast.broadcastify[%d]: system_id required", i)
		}
	}
	for i, f := range b.RdioScanner {
		if !f.Enabled {
			continue
		}
		if f.URL == "" {
			return fmt.Errorf("broadcast.rdioscanner[%d]: url required", i)
		}
		if f.APIKey == "" {
			return fmt.Errorf("broadcast.rdioscanner[%d]: api_key required", i)
		}
		if f.SystemID == 0 {
			return fmt.Errorf("broadcast.rdioscanner[%d]: system_id required", i)
		}
	}
	for i, f := range b.OpenMHz {
		if !f.Enabled {
			continue
		}
		if f.APIKey == "" {
			return fmt.Errorf("broadcast.openmhz[%d]: api_key required", i)
		}
		if f.ShortName == "" {
			return fmt.Errorf("broadcast.openmhz[%d]: short_name required", i)
		}
	}
	for i, f := range b.Icecast {
		if !f.Enabled {
			continue
		}
		if f.Host == "" {
			return fmt.Errorf("broadcast.icecast[%d]: host required", i)
		}
		if f.Port == 0 {
			return fmt.Errorf("broadcast.icecast[%d]: port required", i)
		}
		if f.Password == "" {
			return fmt.Errorf("broadcast.icecast[%d]: password required", i)
		}
	}
	return nil
}

// parseDurationFlexible accepts a Go duration string. Wrapped here so
// the dependency lives in one place and tests can lean on it.
func parseDurationFlexible(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// decodeHexKey parses a hex-encoded encryption key. Surrounding and
// internal whitespace plus an optional "0x"/"0X" prefix are stripped
// so operators can paste keys in whatever form their radio-programming
// software displays them.
func decodeHexKey(s string) ([]byte, error) {
	clean := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		default:
			return r
		}
	}, s)
	clean = strings.TrimPrefix(clean, "0x")
	clean = strings.TrimPrefix(clean, "0X")
	if clean == "" {
		return nil, errors.New("key is empty")
	}
	b, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("key is not valid hex: %w", err)
	}
	return b, nil
}
