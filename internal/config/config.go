package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/pathutil"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Log            LogConfig            `yaml:"log"`
	SDR            SDRConfig            `yaml:"sdr"`
	Trunking       TrunkingConfig       `yaml:"trunking"`
	API            APIConfig            `yaml:"api"`
	Storage        StorageConfig        `yaml:"storage"`
	Recordings     RecordingsConfig     `yaml:"recordings"`
	Metrics        MetricsConfig        `yaml:"metrics"`
	Retention      RetentionConfig      `yaml:"retention"`
	ToneOut        ToneOutConfig        `yaml:"tone_out"`
	Scanner        ScannerConfig        `yaml:"scanner"`
	Audio          AudioConfig          `yaml:"audio"`
	Broadcast      BroadcastConfig      `yaml:"broadcast"`
	Baseband       BasebandConfig       `yaml:"baseband"`
	Paging         PagingConfig         `yaml:"paging"`
	APRS           APRSConfig           `yaml:"aprs"`
	AIS            AISConfig            `yaml:"ais"`
	DSC            DSCConfig            `yaml:"dsc"`
	MDC1200        MDC1200Config        `yaml:"mdc1200"`
	ADSB           ADSBConfig           `yaml:"adsb"`
	M17            M17Config            `yaml:"m17"`
	LoRa           LoRaConfig           `yaml:"lora"`
	Web            WebConfig            `yaml:"web"`
	Display        DisplayConfig        `yaml:"display"`
	Diagnostics    DiagnosticsConfig    `yaml:"diagnostics"`
	RadioReference RadioReferenceConfig `yaml:"radioreference"`
}

// DisplayConfig controls how the daemon renders timestamps for humans. By
// default GopherTrunk historically forced UTC ("…Z") in the decoded-message
// log, the power log and the TUI, which surprises operators expecting their
// own wall-clock time. Timezone selects the location those displayed/logged
// timestamps are rendered in:
//
//   - "" or "Local" → the host's local timezone (the default)
//   - "UTC"         → UTC (the historical behaviour)
//   - any IANA name → that zone, e.g. "America/New_York", "Europe/Paris"
//
// Machine-interchange surfaces (the JSON/gRPC API, webhooks, rdioscanner
// uploads) are intentionally NOT affected — they stay UTC RFC3339 so external
// consumers parse an unambiguous instant. This only changes human-facing
// display, mirroring how SDRtrunk shows local time.
type DisplayConfig struct {
	Timezone string `yaml:"timezone"`
}

// Location resolves Timezone to a *time.Location for formatting displayed
// timestamps. An empty value or "Local" yields time.Local; "UTC" yields
// time.UTC; any other value is looked up as an IANA name. An unparseable name
// falls back to time.Local so a typo degrades to a sensible default rather than
// crashing the daemon — callers that want to warn can use LocationStrict.
func (d DisplayConfig) Location() *time.Location {
	loc, _ := d.LocationStrict()
	return loc
}

// LocationStrict is Location plus the lookup error (nil on success). On error it
// still returns time.Local so the result is always usable; the daemon logs the
// error once at startup.
func (d DisplayConfig) LocationStrict() (*time.Location, error) {
	switch strings.TrimSpace(d.Timezone) {
	case "", "Local", "local":
		return time.Local, nil
	case "UTC", "utc":
		return time.UTC, nil
	default:
		loc, err := time.LoadLocation(strings.TrimSpace(d.Timezone))
		if err != nil {
			return time.Local, err
		}
		return loc, nil
	}
}

// RadioReferenceConfig holds credentials for RadioReference.com's read-only
// SOAP web service. It is consumed by `gophertrunk hunt` to check whether a
// discovered system already exists in RadioReference before producing a
// submission package (RadioReference has no public write API, so nothing is
// ever posted — this is a read-only duplicate check). All fields are optional;
// when APIKey is empty the duplicate check is skipped and the hunt still
// exports its files. The values are also overridable by the GOPHERTRUNK_RR_KEY
// / GOPHERTRUNK_RR_USER / GOPHERTRUNK_RR_PASS environment variables and the
// hunt -rr-key flag, so the secret need not live in config.yaml.
type RadioReferenceConfig struct {
	APIKey   string `yaml:"api_key"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
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

	// MemoryLimitMB sets a soft heap limit (Go runtime/debug.SetMemoryLimit)
	// so the GC keeps the resident footprint bounded instead of letting it
	// balloon under sustained high-allocation load — the mitigation for a
	// daemon being SIGKILLed by the OS memory-pressure killer / macOS jetsam
	// after a few minutes with no in-process trace (issue #492). 0 (the
	// default) auto-derives ~70% of physical RAM when that is known, or
	// leaves the runtime unbounded when it is not. The GOMEMLIMIT env var,
	// if set, always wins (the runtime applies it before this).
	MemoryLimitMB int `yaml:"memory_limit_mb"`

	// HeartbeatSeconds controls a periodic runtime health log (uptime,
	// goroutine count, heap/sys bytes). It turns a silent stop into a
	// timeline: a climbing goroutine/heap curve points at a leak, a frozen
	// heartbeat on a live process points at a hang, and the last line before
	// a cut pins the pre-kill footprint (issue #492). 0 uses the 60 s
	// default; negative disables it.
	HeartbeatSeconds int `yaml:"heartbeat_seconds"`
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
	// IDBase selects how P25/identity numbers (WACN, System ID, NAC, RFSS,
	// Site) are shown in the bundled web SPA: "hex" (the P25 field
	// convention, default) or "dec". The raw decimal values are always
	// present in the JSON API and event payloads (with *_hex strings
	// alongside); this only changes the web display. Empty/unset means "hex".
	IDBase string `yaml:"id_base"`
}

// IDBaseOrDefault returns the configured identity number base ("hex" or
// "dec"), defaulting to "hex" when unset or unrecognised. Feeds
// /api/v1/runtime so both UIs render identity numbers from one source.
func (w WebConfig) IDBaseOrDefault() string {
	if w.IDBase == "dec" {
		return "dec"
	}
	return "hex"
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
	"hunt":          true,
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

// BasebandConfig configures wideband IQ recording and offline replay.
// Empty == disabled. `record` taps live tuners and writes their IQ to
// WAV; `replay` mounts recorded WAVs as virtual tuners so a capture can
// be decoded offline. Replay recordings should have been made at the
// same rate as sdr.sample_rate for real-time-correct playback.
type BasebandConfig struct {
	Record []BasebandRecordConfig `yaml:"record"`
	Replay []BasebandReplayConfig `yaml:"replay"`
	// AutoRecord grabs a short slice of the control SDR's raw IQ whenever a
	// classified event fires (concurrent calls, an unserved grant, an
	// encrypted/emergency grant, or a manual API trigger). The captures are
	// self-describing (`.metadata.json` sidecar) so they drop straight into
	// `replay`/siglab — the event-driven debugging hook the operator asked for.
	AutoRecord BasebandAutoRecordConfig `yaml:"auto_record"`
}

// BasebandAutoRecordConfig configures event-triggered raw-IQ capture of the
// control SDR. Empty/disabled == off (zero cost). Files land in Dir named
// `<system>_<UTC>_<reason>_<freqHz>_<rateHz>.<ext>` so the capture time and
// trigger are obvious from the filename alone.
type BasebandAutoRecordConfig struct {
	// Enabled turns the whole feature on. When false every trigger is a no-op.
	Enabled bool `yaml:"enabled"`
	// Dir is the directory triggered captures (and their metadata sidecars)
	// are written into. Required when Enabled.
	Dir string `yaml:"dir"`
	// Format is the on-disk sample format: "cs16" (default), "f32"
	// (GNU Radio cfile), or "u8" (rtl_sdr native). Parsed by
	// siglab.ParseSampleFormat.
	Format string `yaml:"format"`
	// Seconds is the length of each triggered capture. Required when Enabled.
	Seconds int `yaml:"seconds"`
	// Cooldown is a Go duration string ("10s") — the minimum gap between two
	// automatic triggers, so a burst of grants doesn't spawn a capture storm.
	// Empty defaults to 10s. The manual API trigger bypasses the cooldown.
	Cooldown string `yaml:"cooldown"`
	// OnConcurrentCalls fires a capture when this many voice calls (or more)
	// are active on the system at once. 0 disables this trigger. This is the
	// operator's headline "record when more than one grant at the same time".
	OnConcurrentCalls int `yaml:"on_concurrent_calls"`
	// OnNoVoiceDevice fires when a grant arrives but every voice tuner is busy
	// (the "no voice device available for grant" overload moment).
	OnNoVoiceDevice bool `yaml:"on_no_voice_device"`
	// OnEncrypted fires on a grant flagged encrypted (TEA/AES/DES).
	OnEncrypted bool `yaml:"on_encrypted"`
	// OnEmergency fires on an emergency-flagged grant.
	OnEmergency bool `yaml:"on_emergency"`
	// OnCCSyncLoss fires when a locked control channel suddenly loses sync
	// (events.KindCCLost, which only fires after a genuine lock — never for a
	// hunt that never locked). It captures the seconds AFTER the loss, i.e. the
	// re-acquisition attempt, which is exactly the raw IQ needed to debug
	// sync-loss and slow warm-up-lock episodes. Most useful with the "ddc" tap.
	OnCCSyncLoss bool `yaml:"on_cc_sync_loss"`
	// Tap selects what IQ is captured, mirroring BasebandRecordConfig.Tap:
	//   "wideband" (default) — the control SDR's full-rate raw IQ (the historical
	//     behaviour; large files, e.g. ~50 MB per 30 s at 2.5 MS/s).
	//   "ddc" — the control decoder's narrowband digital-down-converter output
	//     (the channelised stream at the pipeline rate: 144 kHz for TETRA, ~48 kHz
	//     for the C4FM family). Orders of magnitude smaller and directly replayable
	//     with `replay -format wav` / siglab. For a same-carrier TETRA site the DDC
	//     tap holds all four voice timeslots of the control carrier — exactly the
	//     channel worth sharing when a hard-to-decode call fires a trigger.
	Tap string `yaml:"tap"`
	// Decimate is the integer software-decimation factor applied to the
	// wideband tap: the capture is anti-alias filtered and written at
	// SDR-rate / Decimate, cutting the file size (and effective bandwidth) by
	// the same factor while the metadata sidecar records the reduced rate so
	// it still replays. 0 or 1 records the full SDR rate (the default). This
	// is the remedy for a source like the USRP B210 whose hardware
	// sample-rate floor (~1 MS/s) is far above the bandwidth a single
	// narrowband channel needs — run the radio at the floor and decimate in
	// software to a manageable long capture. Only valid with the wideband tap
	// (the ddc tap is already narrowband); a value > 1 with tap: ddc is
	// rejected at config load.
	Decimate int `yaml:"decimate"`
}

// TapDDC reports whether triggered captures tap the narrowband DDC output rather
// than the wideband SDR stream. Mirrors BasebandRecordConfig.TapDDC.
func (a BasebandAutoRecordConfig) TapDDC() bool {
	return strings.EqualFold(strings.TrimSpace(a.Tap), "ddc")
}

// autoRecordDefaultCooldown is used when Cooldown is empty.
const autoRecordDefaultCooldown = 10 * time.Second

// CooldownDuration parses Cooldown, falling back to autoRecordDefaultCooldown
// on an empty string. Validate rejects a malformed value, so callers may
// ignore the error at runtime.
func (a BasebandAutoRecordConfig) CooldownDuration() (time.Duration, error) {
	s := strings.TrimSpace(a.Cooldown)
	if s == "" {
		return autoRecordDefaultCooldown, nil
	}
	return time.ParseDuration(s)
}

// BasebandRecordConfig taps one tuner's IQ to WAV recordings.
type BasebandRecordConfig struct {
	// Serial is the SDR serial whose IQ stream is recorded.
	Serial string `yaml:"serial"`
	// Dir is the directory recordings are written into.
	Dir string `yaml:"dir"`
	// Tap selects what is recorded:
	//   "wideband" (default) — the full-rate raw SDR IQ (large files; the
	//     historical behaviour), useful for re-channelizing or wideband work.
	//   "ddc" — the narrowband digital-down-converter output (the channelized
	//     stream the decoder actually sees, at the pipeline rate: 144 kHz for
	//     TETRA, ~48 kHz for the C4FM family). Orders of magnitude smaller than
	//     wideband, and directly replayable with `replay -format wav`. This is
	//     the "record the DDC output" tap for sharing a hard-to-decode channel.
	Tap string `yaml:"tap"`
}

// TapDDC reports whether this record entry taps the narrowband DDC output
// rather than the wideband SDR stream.
func (b BasebandRecordConfig) TapDDC() bool {
	return strings.EqualFold(strings.TrimSpace(b.Tap), "ddc")
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
	// Broadcastify, RdioScanner, OpenMHz, Icecast and Webhook each list
	// zero or more feeds. A feed with enabled=false is parsed but skipped.
	Broadcastify []BroadcastifyFeedConfig `yaml:"broadcastify"`
	RdioScanner  []RdioScannerFeedConfig  `yaml:"rdioscanner"`
	OpenMHz      []OpenMHzFeedConfig      `yaml:"openmhz"`
	Icecast      []IcecastFeedConfig      `yaml:"icecast"`
	Webhook      []WebhookFeedConfig      `yaml:"webhook"`
	// GrantWebhook lists zero or more push grant-webhook sinks. Each POSTs
	// one JSON object per control-channel grant as it is decoded — the push
	// form of GET /api/v1/grants (issue #915 / #268). A feed with
	// enabled=false is parsed but skipped.
	GrantWebhook []GrantWebhookFeedConfig `yaml:"grant_webhook"`
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

// WebhookFeedConfig is one generic JSON-webhook call sink: each
// completed call is POSTed as one JSON object to URL (issue #404 / #268).
type WebhookFeedConfig struct {
	Enabled bool   `yaml:"enabled"`
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	// AuthHeader is sent verbatim as the Authorization header (e.g.
	// "Bearer <token>"). Empty omits the header.
	AuthHeader string `yaml:"auth_header"`
	// IncludeAudio embeds the base64 MP3 in the payload. Off by default
	// keeps the webhook a lightweight metadata feed.
	IncludeAudio bool     `yaml:"include_audio"`
	Systems      []string `yaml:"systems"`
}

// GrantWebhookFeedConfig is one push grant-webhook sink: each control-channel
// grant is POSTed as one JSON object to URL as it is decoded (issue #915 /
// #268). The payload is the same GrantDTO schema GET /api/v1/grants and the
// KindGrant SSE stream publish, so a consumer reads one schema across all
// three. Unlike the per-call webhook it carries no audio — a grant is a
// control-channel event, not a recording.
type GrantWebhookFeedConfig struct {
	Enabled bool   `yaml:"enabled"`
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	// AuthHeader is sent verbatim as the Authorization header (e.g.
	// "Bearer <token>"). Empty omits the header.
	AuthHeader string   `yaml:"auth_header"`
	Systems    []string `yaml:"systems"`
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
	// LiveLoudness applies a real-time envelope-follower AGC to the
	// decoded digital-voice PCM fed to the live network stream
	// (WebUI / gRPC), so live loudness tracks the loudness-normalized
	// recordings instead of arriving raw and quieter/inconsistent.
	// Default false. Note: this only touches the live stream — the
	// on-disk WAV keeps its own per-call EBU R128 normalization
	// (recordings.normalize) and is never double-processed. It is a
	// real-time approximation, not bit-exact R128 (which needs the
	// whole call), and only affects digital decoded audio; analog FM
	// loudness is shaped by the composer's own optional audio AGC.
	LiveLoudness bool `yaml:"live_loudness"`
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
	// ActivityDebounceMs is the minimum sustained above-threshold time
	// that counts as renewed activity resetting the hangtime countdown.
	// De-bounces the trailing edge so a brief blip can't hold squelch
	// open indefinitely (issue #1090). Default 50 ms; 0 = default.
	ActivityDebounceMs int `yaml:"activity_debounce_ms"`
	// SquelchHysteresisDb is the close-side level margin below
	// squelch_dbfs before a chunk counts as inactive during a call.
	// Default 3 dB; 0 = default.
	SquelchHysteresisDb float64 `yaml:"squelch_hysteresis_db"`
	Priority            int     `yaml:"priority"` // 1..10, 0 = unset
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
	// PowerLog configures the optional per-channel IQ-power log. It
	// records each wideband channel's signal level, gated on decode
	// activity (so only channels actually carrying traffic appear) and,
	// by default, only when that level is below the low-power threshold —
	// the "decoding but weak signal" diagnostic.
	PowerLog PowerLogConfig `yaml:"power_log"`
	// EventLog configures the optional full event log — every bus event
	// written as one JSON line (JSONL/NDJSON), in the same envelope the
	// SSE/WS streams emit. Unlike MessageLog it captures all event kinds,
	// so a session can be recorded and replayed/inspected offline.
	EventLog EventLogConfig `yaml:"event_log"`
}

// MessageLogConfig configures the decoded-message log. Empty Path (or
// Enabled false) disables it.
type MessageLogConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Path      string `yaml:"path"`
	MaxSizeMB int    `yaml:"max_size_mb"` // default 16
}

// EventLogConfig configures the full JSONL event log. Empty Path (or
// Enabled false) disables it.
type EventLogConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Path      string `yaml:"path"`
	MaxSizeMB int    `yaml:"max_size_mb"` // default 16
}

// PowerLogConfig configures the decode-activity-gated IQ-power log. Empty
// Path (or Enabled false) disables it.
type PowerLogConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Path      string `yaml:"path"`
	MaxSizeMB int    `yaml:"max_size_mb"` // default 16
	// AllWindows logs every decode-active window. When false (default)
	// only low-power windows are written.
	AllWindows bool `yaml:"all_windows"`
}

type SDRConfig struct {
	// SampleRate is the IQ rate (Hz) every tuner is programmed to.
	// Default 2_400_000 (2.4 MS/s). Valid range 200_000..20_000_000; the
	// RTL2832U quantizes to its 28.4 fixed-point divisor so the streamed
	// rate may differ slightly (see Device.ActualSampleRate). Note that
	// RTL2832U hardware still caps at 3.2 MHz at the device level (the
	// resampler produces garbage above that), so rates beyond 3.2 MHz are
	// only usable with wideband sources such as soapy_remote (USRP, Lime,
	// bladeRF, …) that can stream them — an RTL dongle handed a higher
	// rate is rejected at open and skipped. This is
	// also the primary load lever on CPU-bound hosts: convert + resample
	// cost scales with it, so if the daemon logs "sdr: dropping live IQ
	// chunks; consumer can't keep up" (iq_underruns_total climbing),
	// lowering it — e.g. to 1_024_000 — roughly halves per-chunk decode
	// work. Running fewer simultaneous dongles on a weak CPU has the same
	// effect.
	SampleRate uint32 `yaml:"sample_rate"`
	// InputSampleRate, when non-zero, is the NATIVE rate (Hz) each tuner is
	// programmed to, and the daemon integer-decimates that wideband stream
	// down to SampleRate with an anti-alias filter BEFORE it reaches any
	// down-converter, demodulator, recording tap, or auto-IQ capture — a
	// systemwide pre-decimation stage. Use it when the front end must run
	// faster than you want to decode/record at: an Airspy pinned to 10 MS/s,
	// or any SDR whose useful signal is a narrow slice of a wide capture.
	// SampleRate then stays the rate everything downstream sees (the DDC bank,
	// the symbol clocks, auto_record/iqtap files, spectrum), so those files
	// are the decode rate and not multi-gigabyte native-rate captures.
	//
	// Must be an exact integer multiple of SampleRate (the decimation factor
	// M = InputSampleRate / SampleRate); a non-integer ratio is rejected at
	// config load. Zero (the default) or a value equal to SampleRate disables
	// the stage — the hardware runs at SampleRate exactly as before. The
	// anti-alias FIR is a real per-chunk cost at the native rate, so this is a
	// trade: cheaper downstream decode/recording for one wideband filter pass.
	// Applies to every source that streams complex IQ (RTL, soapy_remote,
	// ka9q, replay); the pre-combine diversity_capture tap sits below it and
	// still records the native branches.
	InputSampleRate uint32 `yaml:"input_sample_rate"`
	// Autotune enables per-dongle carrier-error tracking and digital
	// frequency correction (ported from trunk-recorder's autotune). When
	// on, the daemon watches each source's locked carrier offset (P25
	// Phase-1 control + voice), keeps a running average per device serial,
	// and shifts the channel's digital down-converter so the demodulator's
	// AFC starts close to lock. It never rewrites the dongle's hardware
	// ppm — it only logs a suggested ppm/error value you can bake into the
	// device block by hand. Off by default; harmless to leave on for
	// TCXO-equipped units (the correction simply stays near zero).
	Autotune bool `yaml:"autotune"`
	// CarrierOffsetWarnHz is the magnitude of the locked control-channel carrier
	// offset (Hz) above which the decoder logs a WARN that GT may be decoding an
	// adjacent site's stronger carrier rather than the configured frequency
	// (issue #815). Zero selects the built-in default (4000 Hz); raise it for a
	// high-drift dongle whose legitimate crystal error would otherwise trip the
	// warning. Advisory only — it never changes tuning or decode.
	CarrierOffsetWarnHz uint32         `yaml:"carrier_offset_warn_hz"`
	Devices             []DeviceConfig `yaml:"devices"`
	// RTLTCP lists remote rtl_tcp endpoints (host:port + optional
	// per-endpoint metadata) to mount as virtual tuners. Each entry
	// becomes a pool device alongside any locally-attached USB
	// dongles. Useful when the SDR hardware lives on a different
	// host from the daemon (e.g. a Raspberry Pi by the antenna +
	// a beefier machine for decode). rtl_tcp is plaintext — use it
	// on trusted networks only or through an SSH/wireguard tunnel.
	RTLTCP []RTLTCPConfig `yaml:"rtl_tcp"`
	// SoapyRemote lists remote SoapySDRServer endpoints to mount as
	// virtual tuners. SoapySDRServer (from pothosware/SoapyRemote)
	// exposes any SoapySDR-supported radio — USRP, LimeSDR, bladeRF,
	// HackRF, Airspy, RTL-SDR, SDRplay — over the network with a real
	// control plane and high bit depth (16-bit CS16 / 32-bit CF32),
	// unlike rtl_tcp's hardcoded 8-bit stream. Each entry becomes one
	// pool device. Plaintext like rtl_tcp — use on trusted networks
	// only or through an SSH/wireguard tunnel.
	SoapyRemote []SoapyRemoteConfig `yaml:"soapy_remote"`
	// Ka9qRadio lists channels from remote ka9q-radio `radiod` instances to
	// mount as virtual tuners. radiod runs fast-convolution downconverters on
	// a front end and multicasts each channel as RTP over IP; a channel in raw
	// "linear" IQ mode (output_channels=2) streams interleaved complex samples
	// GopherTrunk can demodulate directly. Each entry becomes one pool device
	// addressed by the channel's RTP SSRC. The driver discovers the channel's
	// IQ multicast group / sample rate / encoding by polling radiod's status
	// group, and resolves `.local` instance names via mDNS. Plaintext
	// multicast like rtl_tcp / soapy_remote — use on trusted LANs (issue #765).
	Ka9qRadio []Ka9qRadioConfig `yaml:"ka9q_radio"`
	// Sidecar lists external IQ producers to mount as virtual tuners. A
	// sidecar is any process that owns a radio and writes raw IQ to a pipe or
	// socket — a UHD/RFNoC program, a GNU Radio flowgraph, a hardware-specific
	// tool with no SoapySDR support — steered by a small datagram control
	// channel. It keeps hardware and DSP that would need CGO out of
	// GopherTrunk's process while still giving it a real tuner. See
	// docs/reference/sdr-sidecar.md for the wire format.
	Sidecar []SidecarConfig `yaml:"sidecar"`
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

// SoapyRemoteConfig describes one remote SoapySDRServer endpoint to expose
// as a virtual tuner. Addr is required; Serial / Role / PPM / Gain / BiasTee
// follow the same semantics as the local SDR devices and rtl_tcp blocks.
type SoapyRemoteConfig struct {
	// Addr is the SoapySDRServer host:port, e.g. "192.168.1.60:55132".
	// A bare host gets the default port (55132) appended. Required.
	Addr string `yaml:"addr"`
	// Driver is the SoapySDR device key used to select the radio on the
	// server (e.g. "uhd", "lime", "bladerf", "hackrf", "airspy",
	// "rtlsdr"). Empty selects the server's first/only device.
	Driver string `yaml:"driver"`
	// Args are extra SoapySDR device kwargs passed to the remote make(),
	// as a "key=value,key2=value2" string (e.g.
	// "rx_subdev_spec=A:0,antenna=RX1" for a USRP TwinRX). They are merged
	// with Driver; an explicit "driver=" here wins over the Driver field.
	// This is server-side device selection/configuration and is distinct
	// from the top-level Serial, which is the local virtual pool name.
	Args string `yaml:"args"`
	// MasterClockHz sets the radio's master/reference clock in Hz via the
	// SoapySDR make() arg master_clock_rate. A USRP can only deliver rates
	// that are integer decimations of this clock, so setting it lets the
	// device hit an exact-divisor sample_rate instead of having UHD coerce
	// the request to the nearest achievable rate — e.g. a B210 needs
	// master_clock_rate 61_440_000 to stream 6_144_000 (÷10) cleanly, while
	// an X310's default 200 MHz clock already divides evenly into 6_250_000
	// (÷32). Zero leaves the device default. This is a convenience shorthand
	// for putting "master_clock_rate=..." in Args; an explicit value in Args
	// wins.
	MasterClockHz uint32 `yaml:"master_clock_rate"`
	// Serial is the virtual device serial reported on the pool's
	// /api/v1/devices snapshot. Empty generates one from Addr.
	Serial string `yaml:"serial"`
	// Role hints the pool's role assignment: control|voice|auto.
	Role string `yaml:"role"`
	// Format is the requested wire sample format: "CS16" (16-bit, the
	// default) or "CF32" (32-bit float). The server converts from the
	// device's native format as needed.
	Format string `yaml:"format"`
	// StreamProtocol selects the stream transport. Only "tcp" (the
	// default) is currently implemented.
	StreamProtocol string `yaml:"stream_protocol"`
	// StreamMTU sets the SoapyRemote stream endpoint MTU in bytes. It is
	// forwarded to the server's setupStream as the "remote:mtu" stream arg
	// and used to size the client's flow-control window so both ends agree.
	// This is a stream argument, distinct from the device make() kwargs in
	// Args, so it cannot be expressed there. Zero leaves SoapyRemote's
	// default (1500); raise it (e.g. 8192) on jumbo-frame / high-throughput
	// links.
	StreamMTU int `yaml:"stream_mtu"`
	// StreamWindow sets the SoapyRemote stream flow-control window in bytes.
	// It is forwarded to the server's setupStream as the "remote:window"
	// stream arg (sizing its socket buffers) and used as the client's
	// in-flight credit ceiling, advertised as window/stream_mtu sequences, so
	// both ends agree. Like StreamMTU this is a stream argument and cannot be
	// expressed in Args. Zero uses the client default (8 MiB); raise it on
	// high-latency / high-bandwidth links where a larger in-flight window
	// keeps the pipe full.
	StreamWindow int `yaml:"stream_window"`
	// PPM is the frequency-correction tuning applied on open (best-effort;
	// ignored by SoapySDR drivers without frequency-correction support).
	PPM int `yaml:"ppm"`
	// Gain follows the same rule as DeviceConfig.Gain — "auto"/"" selects
	// AGC, any other value parses as tenths of dB.
	Gain string `yaml:"gain"`
	// BiasTee toggles the remote device's bias-tee (best-effort; mapped to
	// a SoapySDR writeSetting and ignored by drivers without the knob).
	BiasTee bool `yaml:"bias_tee"`
	// ConnectTimeoutMs caps the TCP dial in milliseconds. Zero picks the
	// driver default (3000).
	ConnectTimeoutMs int `yaml:"connect_timeout_ms"`
	// Diversity selects a spatial-diversity combiner over a multi-channel RX
	// stream. "" / "none" (default) streams a single channel. "mrc" opens RX
	// channels 0 and 1 and phase-coherently combines them into one maximised-SNR
	// stream, re-estimating the branch gain from the stream and smoothing it, so
	// it suits both a shared-LO front-end (USRP B210 / AD9361, where the branch
	// phase really is a constant) and two receivers on separate daughterboards
	// with independent PLLs (a USRP X310 with two TwinRX cards), whose relative
	// phase is random at each lock and walks afterwards.
	//
	// "mrc-static" is the escape hatch: the same combine with a ONE-SHOT frozen
	// gain, for comparing against the pre-tracking behaviour on hardware where
	// the constant is genuinely constant.
	//
	// Whether the combine helps at all is reported as "coherence" in the
	// periodic MRC log line — |rho| between the branches. The calibration gate
	// scales with the estimation window, so nobody should ever need to raise a
	// gain or narrow a bandwidth just to make diversity engage; if the WARN
	// says the branches are not coherent, believe it and check the antennas —
	// or a branch whose branch_dbfs sits far below the other, which may be
	// buried under its own front-end noise floor (there, raising THAT branch's
	// gain genuinely helps). EXPERIMENTAL (issue #1062).
	Diversity string `yaml:"diversity"`
	// Antennas selects the RX antenna port per channel (SoapySDR setAntenna),
	// applied in channel order after the device opens — antennas[0] to RX
	// channel 0, antennas[1] to channel 1. A multi-value antenna cannot be
	// expressed in the flat "key=value" args string (a comma there splits
	// arguments), so a USRP X310 in MRC diversity that needs RX1 on channel 0
	// and RX2 on channel 1 uses this list instead of args. Empty leaves the
	// device default. More than one entry requires diversity: mrc (only one RX
	// channel is opened otherwise).
	Antennas []string `yaml:"antennas"`
	// DiversityCapture, when non-empty, is a path PREFIX under which the driver
	// writes a one-shot raw dump of the PRE-COMBINE per-branch IQ streams:
	// <prefix>.br0.cs16, <prefix>.br1.cs16 and a <prefix>.diversity.json
	// sidecar. Every other IQ tap in GopherTrunk sits downstream of the
	// combiner, so a capture taken anywhere else has already had one particular
	// combiner applied and cannot be replayed through a different one — this is
	// the only tap that can answer "would a different combiner have done
	// better on this signal?" offline. Requires diversity: mrc or mrc-static.
	DiversityCapture string `yaml:"diversity_capture"`
	// DiversityCaptureSeconds bounds the dump; 0 selects 5 s. Two CS16 branches
	// at 6.25 MS/s is roughly 50 MB/s, so this is deliberately short. A 1 GiB
	// per-branch cap applies regardless.
	DiversityCaptureSeconds int `yaml:"diversity_capture_seconds"`
	// VerboseDebug logs every control-channel RPC exchanged with this
	// SoapySDRServer — decoded call name and arguments plus a hex dump of the
	// frame — at DEBUG level. The SoapyRemote wire carries no schema, so when
	// the server reports "~SoapyRPCUnpacker: Unconsumed payload bytes N" it
	// does not say which call was mis-shaped; this trace is the other half of
	// that conversation. Off by default, and per endpoint so a multi-radio
	// config can follow one server. Needs log.level: debug to be visible.
	VerboseDebug bool `yaml:"verbose_debug"`
}

// SidecarConfig describes one external IQ producer mounted as a virtual tuner.
//
// The contract is deliberately small: the sidecar streams raw interleaved IQ
// one way, and GopherTrunk sends 5-byte tuning commands the other way. Both
// halves are documented in docs/reference/sdr-sidecar.md, and the control
// opcodes match rtl_tcp's so an existing rtl_tcp-shaped tool works unmodified.
type SidecarConfig struct {
	// Transport selects how the IQ arrives: "unix_pipe" (a FIFO path),
	// "tcp" (GopherTrunk dials the sidecar) or "udp" (GopherTrunk binds and
	// the sidecar sends datagrams). Defaults to "tcp".
	Transport string `yaml:"transport"`
	// DataAddr is the FIFO path for unix_pipe, or host:port for tcp/udp.
	// Required.
	DataAddr string `yaml:"data_addr"`
	// ControlAddr is the sidecar's UDP command socket, host:port. Empty means
	// the sidecar owns tuning: GopherTrunk's setters become no-ops, which is
	// correct for a fixed-frequency feed but means it cannot follow a
	// trunked system's voice grants.
	ControlAddr string `yaml:"control_addr"`
	// Format is the wire sample format: "cs16" (16-bit signed I/Q, default) or
	// "complex64" (32-bit float I/Q, native Go layout, no conversion cost
	// locally but twice the bytes over a network).
	Format string `yaml:"format"`
	// SampleRateHz is the rate the sidecar delivers. GopherTrunk cannot probe
	// for it, and every downstream filter is sized from it, so a wrong value
	// mis-tunes every channel. Required.
	SampleRateHz uint32 `yaml:"sample_rate_hz"`
	// FreqMinHz / FreqMaxHz declare the tuning range for the whole-device hunt
	// sweep. Both zero leaves the range unknown, and a sweep that needs one
	// must be given an explicit band.
	FreqMinHz uint32 `yaml:"freq_min_hz"`
	FreqMaxHz uint32 `yaml:"freq_max_hz"`
	// Serial is the virtual device serial the pool reports. Empty derives one
	// from the data address.
	Serial string `yaml:"serial"`
	// Role hints the pool's role assignment: "control" | "voice" | "auto".
	Role string `yaml:"role"`
	// Gain is passed to the sidecar on open: "auto"/empty for AGC, else tenths
	// of a dB. What it means is the sidecar's business.
	Gain string `yaml:"gain"`
	// ConnectTimeoutMs bounds the TCP dial and the FIFO open. Zero uses the
	// driver default (3000). A FIFO open blocks until a writer attaches, so
	// this is what keeps a sidecar that never starts from hanging the daemon.
	ConnectTimeoutMs int `yaml:"connect_timeout_ms"`
}

// parseDeviceArgs parses a SoapySDR-style "key=value,key2=value2" argument
// string into a map. Empty input yields an empty map. Whitespace around keys
// and values is trimmed and empty segments are skipped. A segment with no "="
// or an empty key is an error.
func parseDeviceArgs(s string) (map[string]string, error) {
	out := map[string]string{}
	for _, seg := range strings.Split(s, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		k, v, ok := strings.Cut(seg, "=")
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid arg %q (want key=value)", seg)
		}
		out[k] = v
	}
	return out, nil
}

// reservedStreamArgs maps SoapyRemote stream-setup keys that GopherTrunk owns
// (it constructs the SETUP_STREAM frame itself) to the config key that actually
// applies them. Left in the free-form args string they reach the remote make()
// call and are silently dropped for streaming, so config load rejects them.
var reservedStreamArgs = map[string]string{
	"remote:mtu":    "stream_mtu",
	"remote:window": "stream_window",
	"remote:prot":   "stream_protocol",
}

// reservedStreamArg reports the first reserved SoapyRemote stream key present
// in the free-form args string, along with the config key that supersedes it.
// It returns "", "" when args carries no reserved stream key (or is malformed,
// which DeviceArgs reports separately).
func reservedStreamArg(argsStr string) (key, dest string) {
	args, err := parseDeviceArgs(argsStr)
	if err != nil {
		return "", ""
	}
	// Iterate in a stable order so the reported key is deterministic when args
	// names more than one.
	for _, k := range []string{"remote:mtu", "remote:window", "remote:prot"} {
		if _, ok := args[k]; ok {
			return k, reservedStreamArgs[k]
		}
	}
	return "", ""
}

// DeviceArgs returns the SoapySDR make() kwargs for this endpoint: any
// key=value pairs from Args, merged with the Driver shorthand. An explicit
// "driver=" in Args wins over the Driver field. It returns nil when no args
// apply (matching the driver's "select the server's first device" default),
// or an error when Args is malformed.
func (s SoapyRemoteConfig) DeviceArgs() (map[string]string, error) {
	args, err := parseDeviceArgs(s.Args)
	if err != nil {
		return nil, err
	}
	if s.Driver != "" {
		if _, ok := args["driver"]; !ok {
			args["driver"] = s.Driver
		}
	}
	if s.MasterClockHz != 0 {
		if _, ok := args["master_clock_rate"]; !ok {
			args["master_clock_rate"] = strconv.FormatUint(uint64(s.MasterClockHz), 10)
		}
	}
	if len(args) == 0 {
		return nil, nil
	}
	return args, nil
}

// Ka9qRadioConfig describes one ka9q-radio `radiod` channel to expose as a
// virtual tuner. Addr (the status multicast group) and SSRC are required;
// Serial / Role follow the same semantics as the local SDR devices block. The
// optional Data / SampleRate / Encoding / Channels fields pin the channel
// parameters and skip the status-group poll that otherwise discovers them.
type Ka9qRadioConfig struct {
	// Addr is the status+command multicast group: an mDNS name like
	// "hf.local" / "hf.local:5006", or a literal "239.1.2.3:5006". A missing
	// port defaults to 5006. Required.
	Addr string `yaml:"addr"`
	// SSRC selects the channel within the radiod instance (the channel's RTP
	// SSRC, e.g. 162550 for 162.550 MHz). Required, non-zero.
	SSRC uint32 `yaml:"ssrc"`
	// Serial is the virtual device serial reported on the pool's
	// /api/v1/devices snapshot. Empty generates one from Addr+SSRC.
	Serial string `yaml:"serial"`
	// Role hints the pool's role assignment: control|voice|auto.
	Role string `yaml:"role"`
	// Data optionally pins the IQ multicast group ("239.4.5.6:5004"),
	// skipping discovery of the data socket from the status poll. A missing
	// port defaults to 5004.
	Data string `yaml:"data"`
	// SampleRate optionally pins the channel's IQ rate (Hz), skipping
	// OUTPUT_SAMPRATE discovery.
	SampleRate uint32 `yaml:"sample_rate"`
	// Encoding optionally pins the wire sample encoding: s16be|s16le|f32le|
	// f32be (raw IQ encodings), skipping OUTPUT_ENCODING discovery.
	Encoding string `yaml:"encoding"`
	// Channels optionally pins the output channel count (2 for raw IQ).
	Channels int `yaml:"channels"`
	// ConnectTimeoutMs caps mDNS resolution and the status poll in
	// milliseconds. Zero picks the driver default (3000).
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

	// NarrowbandFilter engages the HackRF Pro's switchable narrowband
	// anti-alias filter. It tightens adjacent-channel rejection for
	// narrowband voice channels (e.g. 12.5 kHz P25) at the cost of
	// usable RF bandwidth, which can lift a marginal decode on a crowded
	// band. HackRF Pro only — ignored, with a startup warning, on any
	// other device (including the original HackRF One). Off by default.
	NarrowbandFilter bool `yaml:"narrowband_filter"`

	// FPGADCBlock engages the HackRF Pro's FPGA-side DC-offset blocker,
	// which strips the zero-IF DC spike in the gateware before the
	// samples leave the device. It's a hardware alternative to the P25
	// voice path's software DC-block and, unlike that block, also cleans
	// the control channel. HackRF Pro only — ignored, with a startup
	// warning, on any other device. Off by default.
	FPGADCBlock bool `yaml:"fpga_dc_block"`

	// RFAmp engages the front-end RF amplifier on the "auto" gain preset
	// (the HackRF's ~14 dB amp). It lowers the noise figure for weak
	// signals but adds gain ahead of everything, so a front end near a
	// strong transmitter can overload — leave it off unless a weak-signal
	// site needs it. Applies to devices with a switchable amp (the HackRF);
	// ignored, with a startup warning, on devices without one. Off by
	// default. Manual (positive) `gain:` targets are unaffected.
	RFAmp bool `yaml:"rf_amp"`

	// BlogV4 forces RTL-SDR Blog V4 mode (28.8 MHz reference crystal +
	// per-band HF/VHF/UHF input routing) regardless of the dongle's USB
	// iManufacturer/iProduct strings. Use it when a V4's EEPROM strings
	// are blank or non-standard so auto-detection misses it and the
	// R828D mistunes every frequency by ~1.8× (issue #264). Off by
	// default; leave false for any non-V4 dongle. BlogV4Lite selects the
	// two-band "Lite" variant — set it only on a V4L. When set, the
	// config value wins over auto-detection (it is applied after open).
	BlogV4     bool `yaml:"blog_v4"`
	BlogV4Lite bool `yaml:"blog_v4_lite"`

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
	// physical voice SDR when one is configured. No hard upper
	// bound, but each tap runs an independent DDC so CPU scales
	// roughly linearly per tap — the daemon logs a warning above
	// 16 so a typo doesn't silently peg a core.
	VoiceTaps int `yaml:"voice_taps"`

	// SignallingTaps is the number of per-grant DDC tuners the daemon
	// allocates from this wideband dongle's IQ stream for
	// signalling-only follows that harvest P25 Phase 2 talker aliases
	// off the traffic channel's FACCH-S signalling, independent of the
	// voice pool (issue #376). Unlike VoiceTaps these never record
	// audio — they decode the MAC signalling and publish the alias —
	// so the alias surfaces even when no voice tuner is free or the
	// call is encrypted and torn down before hangtime.
	//
	// Defaults to 0 (no signalling follows). Set to 2-4 on a busy
	// multi-site Phase 2 system where most grants never get a voice
	// tuner. Each tap runs an independent DDC, so CPU scales roughly
	// linearly per tap; the daemon warns above 16. Out-of-window grants
	// are skipped silently.
	SignallingTaps int `yaml:"signalling_taps"`

	// IQCorrect enables blind I/Q-imbalance correction on this device's
	// raw IQ before decimation (issue #402). Off by default. An
	// uncorrected RTL-SDR I/Q imbalance distorts the demodulated symbol
	// eye (worst at the on-channel DC the control decoder's DDC sits on);
	// validate the benefit with `gophertrunk replay -iq-correct -diag`
	// on a capture from this device before enabling it here.
	IQCorrect bool `yaml:"iq_correct"`

	// IQInvert conjugates this device's raw IQ (negates Q) before
	// channelization, undoing a spectrum-inverted / I-Q-swapped front
	// end. Some SoapySDR / soapy_remote front-ends (and a few USRP /
	// upconverter chains) deliver an inverted spectrum; on a π/4-DQPSK
	// protocol like TETRA an inverted spectrum reverses every phase
	// transition, so the constellation collapses and nothing locks even
	// though the signal looks clean. Off by default. Confirm against a
	// capture with `gophertrunk replay -conjugate -diag` before enabling.
	// Equivalent to the replay subcommand's -conjugate flag (issue #264).
	IQInvert bool `yaml:"iq_invert"`

	// DCAvoid enables live LO-offset (DC-spike-avoidance) tuning (issue #402):
	// the hardware LO is tuned below the target frequency and the channel is
	// mixed back to baseband, off the front-end DC spur, 1/f noise and the
	// channel's own I/Q-imbalance image (all of which corrupt a C4FM channel
	// sitting at zero-IF on a HackRF / RTL-SDR). This is the same offset tuning
	// SDRTrunk/OP25 apply by channelising every carrier off-DC.
	//
	// Applies to BOTH roles:
	//   - control: the ccdecoder down-converter mixes the control channel back.
	//   - voice: each granted call is offset-tuned and mixed back per-call (the
	//     composer sees an on-channel stream). On a zero-IF dongle a granted
	//     voice carrier tuned exactly on-channel sits on the DC spike, which
	//     leaves a good average EVM but corrupts frame-sync so voice decodes
	//     zero LDUs — this is the fix. Strongly recommended for a role:voice
	//     HackRF/RTL following a simulcast/marginal system.
	//
	// Off by default; enable per device and confirm the tsbk-crc / nid-bch
	// rates drop (control) or voice grants start decoding LDUs (voice).
	DCAvoid bool `yaml:"dc_avoid"`

	// DCAvoidOffsetHz pins the LO offset in Hz used when DCAvoid is set.
	// 0 (the default) auto-selects sample_rate/4. Must be < sample_rate/2;
	// ignored when DCAvoid is false or when the delivered sample rate is
	// at/below the channel rate (no room to offset). Applies to control and
	// voice roles alike.
	DCAvoidOffsetHz int `yaml:"dc_avoid_offset_hz"`
}

// DeviceChannelConfig is one repeater carrier carried by a
// `role: wideband` dongle. FrequencyHz must lie inside the dongle's
// IQ band (CenterFreqHz ± sample_rate/2 minus a guard); System must
// match an existing trunking.systems[].name with a supported
// per-channel protocol.
type DeviceChannelConfig struct {
	FrequencyHz uint32 `yaml:"frequency_hz"`
	System      string `yaml:"system"`

	// P25Phase1DemodMode optionally overrides the parent system's
	// p25_phase1_demod_mode for this one control-channel tap. Within a
	// single P25 system, individual sites can transmit different
	// modulation — e.g. a site that genuinely transmits linear/CQPSK
	// (Linear Simulcast Modulation, LSM) alongside sites on straight
	// C4FM (issue #935). Because a wideband dongle decodes every site's
	// control channel in parallel and the correct demod path must be
	// chosen before the CC locks (it is what lets it lock), the override
	// is keyed here per control-channel frequency — the one place a
	// site's identity is known at config time — rather than by the
	// RFSS/Site the CC only reveals after it decodes.
	//
	// Do NOT set this to cqpsk merely because a site is simulcast:
	// simulcast is a transmitter-coordination technique, not a
	// modulation, and most simulcast systems transmit C4FM (Victoria's
	// MMR, all sites, is the worked example in #935 — forcing CQPSK
	// there kills the decode). Override to cqpsk only for sites that a
	// strong, clean signal proves won't lock in C4FM.
	//
	// Recognised values match the system-level key (case-insensitive):
	// "" (inherit the system's p25_phase1_demod_mode) / "c4fm" / "fm"
	// (FM discriminator + 4-level slicer) or "cqpsk" / "lsm" / "linear"
	// (the linear path — complex RRC + Gardner + differential QPSK).
	// Applies to both the control-channel decoder and the voice grants
	// this tap issues, so a granted voice call on a CQPSK site is decoded
	// on the linear path too. Ignored for non-P25-Phase-1 channels.
	P25Phase1DemodMode string `yaml:"p25_phase1_demod_mode"`
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

	// VoiceHangtimeMs is the universal "end of transmission" window
	// applied to EVERY voice protocol (FM, DMR, P25 Phase 1 / 2, TETRA):
	// once a call has been decoding voice, the composer ends it this long
	// after the last decoded voice frame, instead of waiting out the much
	// longer CallTimeoutMs watchdog. Keeps recordings tightly bounded to
	// the actual transmission. Defaults to 3500 (3.5 s) when zero;
	// negative values are rejected by Validate.
	//
	// Note this governs when the SDR/voice chain is released, NOT the audio
	// length. For digital protocols the recording is exactly the decoded
	// voice frames, so raising this cannot lengthen a recording that ends
	// early — that is a decode-completeness matter (e.g. CRC-failed tail
	// bursts), not a hangtime one.
	VoiceHangtimeMs int `yaml:"voice_hangtime_ms"`

	// VoiceCallGrouping controls how voice recordings are split, for
	// EVERY voice protocol. "transmission" (default) writes one file per
	// over/PTT — the recording rolls to a fresh file at each
	// end-of-transmission boundary. "conversation" keeps consecutive
	// overs of the same talkgroup in one file, splitting only when a
	// different talkgroup takes the (shared) frequency or the channel
	// goes idle past VoiceHangtimeMs. Empty defaults to "transmission";
	// any other value is rejected by Validate.
	VoiceCallGrouping string `yaml:"voice_call_grouping"`
}

// EncryptedCallsConfig configures handling of calls discovered to be
// encrypted, so encrypted traffic can't monopolize a limited pool of
// voice SDRs and starve clear calls. Set per system under
// trunking.systems[].encrypted_calls so an operator can run "metadata" on
// one system and "follow" / "ignore" on another. Issue #711.
type EncryptedCallsConfig struct {
	// Mode selects the policy:
	//   "follow" (default / empty) — hold a voice SDR for the full call,
	//     exactly like a clear call (backwards-compatible behaviour).
	//   "metadata" — follow the call briefly so traffic-channel metadata
	//     (P25 Phase 2 talker alias, source RID, encryption sync) is
	//     captured, then release the voice SDR MetadataFollowMs after the
	//     call is first known to be encrypted, or as soon as the talker
	//     alias completes — whichever comes first.
	//   "ignore" — never tie up a voice SDR on an encrypted call.
	// Any other value is rejected by Validate. A call whose KeyID matches
	// a configured trunking.systems[].encryption_keys entry is exempt and
	// always followed regardless of mode.
	Mode string `yaml:"mode"`

	// MetadataFollowMs is how long (milliseconds) an encrypted call is
	// followed under mode "metadata" before its voice SDR is released,
	// measured from when the call is first known to be encrypted. 0 uses
	// the engine default (1500 ms). Negative values are rejected by
	// Validate. Ignored in "follow" / "ignore" modes.
	MetadataFollowMs int `yaml:"metadata_follow_ms"`
}

// SiteConfig names one P25 site by its RFSS and Site IDs. It is pure
// presentation metadata: GT discovers a site's identity from the
// control channel and merges the Name into GET /api/v1/sites so the
// site reads as a place rather than a pair of integers (issue #698).
type SiteConfig struct {
	RFSS uint8  `yaml:"rfss"`
	Site uint8  `yaml:"site"`
	Name string `yaml:"name"`
	// Latitude / Longitude are the site's optional geographic position
	// (decimal degrees, WGS84). When set they are merged into
	// GET /api/v1/sites so the web console can plot the site on a map.
	// Both zero means "no position" (the console hides such sites from the
	// map). RadioReference import fills these in automatically; operators can
	// also set them by hand for a site RR doesn't geolocate.
	Latitude  float64 `yaml:"latitude"`
	Longitude float64 `yaml:"longitude"`
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

	// Sites is the optional catalogue of human-readable names for the
	// P25 sites of this system, keyed by their RFSS and Site IDs
	// (issue #698). GT discovers a site's RFSS/Site from the control
	// channel; these names are pure presentation metadata merged into
	// GET /api/v1/sites so a discovered site reads as e.g.
	// "Mt Anakie" instead of "rfss 1 / site 1". Ignored for non-P25
	// protocols. Leave empty to surface discovered sites without names.
	Sites []SiteConfig `yaml:"sites"`

	// TETRAColourCode is the 30-bit extended colour code the TETRA
	// scrambler uses to seed its LFSR (ETSI EN 300 392-2 §8.2.5).
	// Set this to the per-cell colour code of the TETRA TMO system
	// being decoded so the descrambler can recover the type-3
	// stream. Bits 30..31 are silently ignored. Zero is valid only
	// for BSCH (§8.2.5.2); non-BSCH channels need the per-cell
	// colour code or descrambling produces garbage. Ignored for
	// non-TETRA protocols.
	TETRAColourCode uint32 `yaml:"tetra_colour_code"`
	// TETRAMCC / TETRAMNC give the network's Mobile Country Code
	// (10-bit, 0..1023) and Mobile Network Code (14-bit, 0..16383) —
	// the MNI half of the 30-bit extended colour code that seeds the
	// TETRA scrambler (ExtendedColourCode(MCC, MNC, colour)). They exist
	// for TETRA DMO (Direct Mode): the sync burst is always colour-0
	// scrambled and carries no MNI on air, but the voice traffic (TCH/S)
	// is scrambled with the FULL extended code, so on a network with a
	// non-zero MNI the traffic seed is ExtendedColourCode(MCC, MNC,
	// colour) and DMO decode fails unless the MNI is known. Set these to
	// the network's MNI (e.g. a Motorola DMO codeplug's MCC 250 / MNC 1)
	// so the DMO colour recovery folds it into every candidate; the
	// colour code itself is still auto-recovered. Zero (the default) is a
	// radio-to-radio DMO with MNI 0. Ignored for non-DMO / non-TETRA.
	TETRAMCC uint16 `yaml:"tetra_mcc"`
	TETRAMNC uint16 `yaml:"tetra_mnc"`
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

	// TETRAStatusIntervalSecs sets how often (seconds) the throttled
	// "tetra: decode status" DEBUG line is emitted, and therefore the
	// window its per-interval counters (sb_bursts, bsch_ok/fail,
	// sysinfo, sch_pdus/sch_pdus_fail, grants) accumulate over. 0 or
	// unset ⇒ the 5 s default. Debug-only diagnostics; has no effect
	// unless the log level is debug. Ignored for non-TETRA protocols.
	TETRAStatusIntervalSecs float64 `yaml:"tetra_status_interval_secs"`

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
	// (the default — FM discriminator + 4-level slicer; matches every
	// previously shipping config and is correct for the large majority
	// of P25 systems, including most simulcast systems, which transmit
	// C4FM) or "cqpsk" / "lsm" / "linear" (the linear / LSM path —
	// complex RRC + Gardner + differential QPSK; for the minority of
	// systems that transmit Linear Simulcast Modulation, a linear
	// π/4-DQPSK waveform, rather than C4FM — see issue #275 and
	// TIA-102.BAAA). The modulation is NOT implied by a system being
	// simulcast, nor readable from emission-designator/licensing data;
	// determine it empirically and set cqpsk only when a strong, clean
	// signal will not lock in C4FM (issue #935). Applies to both the
	// control channel decoder and the per-call voice chain — without
	// the voice-chain side a CQPSK site would lock the CC fine but
	// never decode an LDU on a granted voice call (issue #356
	// follow-up). Ignored for non-P25-Phase-1 protocols.
	P25Phase1DemodMode string `yaml:"p25_phase1_demod_mode"`
	// DMRInterleavedVoice overrides the 2-slot interleaved voice decoder.
	// A DMR carrier is 2-slot TDMA, so the demodulated stream interleaves
	// both timeslots' bursts; the interleaved decoder pulls each call's own
	// timeslot out (auto-detecting the on-air same-slot cadence, with or
	// without inter-burst CACH) and routes it by embedded-LC talkgroup,
	// where the single-slot decoder would splice the two slots together
	// into garbled audio (issue #644).
	//
	// It is a tri-state: nil (unset) uses the per-protocol default —
	// interleaved ON for DMR Tier III, single-slot for everything else.
	// An explicit true/false forces interleaved on or off for that system.
	// Ignored for non-DMR systems.
	DMRInterleavedVoice *bool `yaml:"dmr_interleaved_voice,omitempty"`
	// DMRColorCode pins a conventional DMR (Tier II / Tier I) system to a
	// single colour code (0..15). When set, any burst whose decoded colour
	// code differs is dropped before it can grant, lock, or raise a decode
	// error — the "list of frequencies for one Colour Code" IPSC /
	// linked-repeater profile, which keeps a co-channel system on a different
	// colour code (bleeding into the shared wideband passband) out of the call
	// log. It is a tri-state: nil (unset) accepts every colour code (the
	// historical default — GT reports whatever it reads off air). Ignored for
	// non-conventional-DMR protocols.
	DMRColorCode *uint8 `yaml:"color_code,omitempty"`
	// P25Phase2TrellisMode enables the 4-state ½-rate trellis FEC
	// decoder on the P25 Phase 2 MAC PDU window. Recognised values:
	// "" / "on" / "true" / "1" (the new default — 146 channel
	// dibits via the TIA-102.AABF trellis decoder) or "off" /
	// "false" / "0" (legacy 72-dibit raw-MAC-PDU path, opt-out for
	// pre-stripped fixtures). Ignored for non-P25-Phase-2 protocols.
	P25Phase2TrellisMode string `yaml:"p25_phase2_trellis_mode"`
	// P25Phase2RSMode enables the outer Reed-Solomon RS(24, 16, 9)
	// layer on top of the trellis-decoded MAC PDU. Recognised values:
	// "" / "off" / "false" / "0" (the default — no outer RS; matches
	// historical decoder behaviour); "on" / "true" / "1" (verify RS
	// syndromes per TIA-102.BAAA-A §5.9 and drop MAC PDUs whose
	// syndromes are non-zero — detection only); or "correct" / "fix" /
	// "ecc" (bounded-distance error *correction* of up to t=4 symbol
	// errors before parsing, instead of dropping the PDU — the
	// weak-frame recovery path for issue #915 that lets marginal-SNR
	// traffic-channel MAC PDUs still yield a source RID). "correct"
	// strictly supersets "on"; a corrected PDU is additionally gated on
	// a recognised opcode so a wrong-phase window cannot be miscorrected
	// into a bogus PDU. Ignored for non-P25-Phase-2 protocols.
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
	// P25Phase2SoftDecision enables the soft-decision demod path on the
	// P25 Phase 2 traffic-channel receiver (issue #915). Recognised
	// values: "" / "off" / "false" / "0" (the default — the hard
	// slicer, byte-for-byte the historical behaviour) or "on" / "true"
	// / "1" (feed the demodulator's soft symbol differentials into a
	// per-bit soft Viterbi on the MAC trellis, recovering the ~1.5-2 dB
	// of coding gain the hard slicer discards). Helps recover the
	// clear-MAC source RID on weak signals; neutral on strong ones.
	// Ignored for non-P25-Phase-2 protocols.
	P25Phase2SoftDecision string `yaml:"p25_phase2_soft_decision"`
	// P25Phase2Equalizer enables a blind constant-modulus (CMA) adaptive
	// equalizer on the P25 Phase 2 traffic-channel receiver (issue #915).
	// Recognised values: "" / "off" / "false" / "0" (the default — no
	// equalization, the symbol stream is untouched) or "on" / "true" / "1" /
	// "cma" (remove residual inter-symbol interference — RRC mismatch, a
	// fractional timing error, mild multipath — on the symbol stream ahead of
	// the differential decode, so a burst the ISI would otherwise corrupt still
	// satisfies the outer RS). Helps ISI-limited channels; neutral-to-slightly-
	// noisy on a purely AWGN-limited one, so it is opt-in. Ignored for
	// non-P25-Phase-2 protocols.
	P25Phase2Equalizer string `yaml:"p25_phase2_equalizer"`
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

	// DMRBandPlan maps the 12-bit Logical (Physical) Channel Number (LCN)
	// carried in each DMR Tier III voice-grant CSBK to a downlink frequency.
	// REQUIRED for T3 voice — T3 grants reference a channel by LCN, not
	// an absolute frequency, so without this plan every grant is
	// dropped with decode.error stage=no-bandplan. Provide exactly one
	// of `linear` (regular base+spacing grid) or `table` (explicit
	// LCN→Hz list). Ignored for non-dmr protocols.
	DMRBandPlan *DMRBandPlanConfig `yaml:"dmr_band_plan"`

	// NXDNBandPlan maps the traffic-channel number carried in each NXDN
	// VCALL_ASSGN message to a downlink frequency. Required for NXDN
	// voice follow — NXDN grants reference a channel by number, not an
	// absolute frequency, so without this plan every voice grant is
	// dropped. Provide exactly one of `linear` (regular base+spacing
	// grid) or `table` (explicit channel→Hz list). Ignored for non-nxdn
	// protocols.
	NXDNBandPlan *NXDNBandPlanConfig `yaml:"nxdn_band_plan"`

	// EncryptionKeys lists operator-supplied decryption keys for this
	// system. GopherTrunk decrypts only with keys the operator
	// already holds and is authorized to use — it performs no key
	// recovery. Today only DMR ARC4/RC4 ("Enhanced Privacy") is
	// recognised; the per-key `algorithm` field keeps the schema open
	// so AES can be added later without a config break. Ignored for
	// protocols without an encryption decoder. See issue #276.
	EncryptionKeys []EncryptionKeyConfig `yaml:"encryption_keys"`

	// EncryptedCalls controls how the engine allocates scarce voice SDRs
	// to encrypted calls on this system, so a few long encrypted calls
	// can't monopolize a limited voice pool and starve clear traffic.
	// Per-system so an operator can run "metadata" on one system and
	// "follow" / "ignore" on another. Empty mode = "follow" (the
	// pre-issue-711 behaviour). Issue #711.
	EncryptedCalls EncryptedCallsConfig `yaml:"encrypted_calls"`
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

// DMRBandPlanConfig is the operator-supplied DMR Tier III LCN→frequency
// band plan for a system. Exactly one of Linear or Table must be set
// (enforced by Config.Validate). See internal/radio/dmr/tier3/bandplan.go
// for the resolution math.
type DMRBandPlanConfig struct {
	Linear *DMRLinearBandPlanConfig      `yaml:"linear"`
	Table  []DMRBandPlanTableEntryConfig `yaml:"table"`
}

// DMRLinearBandPlanConfig lays channels out on a regular grid:
// freq = base_hz + (lcn - offset) × spacing_hz. Set offset=1 for the
// common case of sites that number LCNs from 1.
type DMRLinearBandPlanConfig struct {
	BaseHz    uint32 `yaml:"base_hz"`
	SpacingHz uint32 `yaml:"spacing_hz"`
	Offset    int8   `yaml:"offset"`
}

// DMRBandPlanTableEntryConfig is one explicit LCN→downlink-frequency
// mapping for sites whose channels don't fall on a regular grid.
type DMRBandPlanTableEntryConfig struct {
	LCN    uint16 `yaml:"lcn"`
	FreqHz uint32 `yaml:"freq_hz"`
}

// NXDNBandPlanConfig is the operator-supplied NXDN traffic-channel →
// frequency band plan for a system. Exactly one of Linear or Table
// must be set (enforced by Config.Validate). See
// internal/radio/nxdn/bandplan.go for the resolution math.
type NXDNBandPlanConfig struct {
	Linear *NXDNLinearBandPlanConfig      `yaml:"linear"`
	Table  []NXDNBandPlanTableEntryConfig `yaml:"table"`
}

// NXDNLinearBandPlanConfig lays channels out on a regular grid:
// freq = base_hz + (channel - offset) × spacing_hz. Set offset=1 for
// sites that number channels from 1.
type NXDNLinearBandPlanConfig struct {
	BaseHz    uint32 `yaml:"base_hz"`
	SpacingHz uint32 `yaml:"spacing_hz"`
	Offset    int8   `yaml:"offset"`
}

// NXDNBandPlanTableEntryConfig is one explicit channel→downlink-frequency
// mapping for sites whose channels don't fall on a regular grid.
type NXDNBandPlanTableEntryConfig struct {
	Channel uint16 `yaml:"channel"`
	FreqHz  uint32 `yaml:"freq_hz"`
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
	// Enhance is the opt-in "sound-good" voice enhancement chain. When
	// enabled it band-limits decoded digital voice to the telephone band,
	// warms the bright software-AMBE+2 timbre, runs the AGC to a louder
	// target, and (optionally) compresses — shaping BOTH the recorded WAV
	// and live monitoring. It deliberately trades a little faithfulness
	// for the cleaner/louder sound the rival decoders (OP25, Trunk
	// Recorder, DSDPlus) produce. Off by default; the faithful path is
	// byte-identical when disabled. Listed high so it's easy to find.
	Enhance  EnhanceConfig `yaml:"enhance"`
	WriteRaw bool          `yaml:"write_raw"`
	// MBEFiles additionally writes a dsd-fme-playable MBE sidecar per call
	// for protocols dsd-fme can play offline: <basename>.imb (P25 Phase 1
	// IMBE) / <basename>.amb (DMR / NXDN / P25 Phase 2 AMBE+2), in dsd-fme's
	// own cookie-headed container so `dsd-fme -r <file>` decodes it directly.
	// Unlike the flat .raw (write_raw), these need no external knowledge of
	// the frame size or protocol. TETRA ACELP / ProVoice / analog calls
	// produce no MBE file (dsd-fme has no playback mode for them).
	MBEFiles bool `yaml:"mbe_files"`
	// SkipEncrypted, when true, suppresses recording of calls flagged
	// encrypted. A call whose grant already signals encryption is never
	// opened; a call whose encryption is only discovered mid-stream has
	// its in-progress WAV/raw files closed and deleted. Live follow /
	// playback is unaffected. Default false (record everything).
	//
	// Note it also suppresses the completed-call broadcast backends
	// (webhook, rdioscanner, …) for encrypted calls: because the recorder
	// discards the call without publishing a CallComplete, no completed-call
	// event reaches those backends. Set skip_encrypted: false to deliver
	// encrypted calls (with their backfilled encryption / alg / key
	// metadata) to them. See issue #897.
	SkipEncrypted bool `yaml:"skip_encrypted"`
	// CryptoCapturePath, when set, opts into the cryptolab crypto-frame
	// bridge: for each encrypted P25 Phase 1 superframe the voice composer
	// appends a JSON line {label, iv (Message Indicator), ct (encrypted
	// voice frames), algid, keyid, …} to this file. The artifact feeds
	// `gophertrunk cryptolab assess`, the security-test harness that attempts
	// decryption by every applicable method (keystream reuse, known plaintext,
	// default/weak keys, keystream-LFSR) and grades the deployment's
	// resistance. Empty (default) disables the capture entirely — no
	// extraction work runs on the voice path. Research/offline use.
	CryptoCapturePath string `yaml:"crypto_capture_path"`
	// Equalizer enables the per-call CMA blind equalizer that the FM
	// composer chain runs between the front-end LPF and the FM demod.
	// Off by default; useful when receiving simulcast systems with
	// multiple transmitters at slightly different arrival delays.
	Equalizer EqualizerConfig `yaml:"equalizer"`
	// Normalize enables per-call EBU R128 / BS.1770 loudness
	// normalization. When enabled, each finished recording is measured and
	// rewritten in place to a perceptual loudness target (true-peak
	// limited), so calls from different talkgroups/sources play back at a
	// consistent level. Off by default. This is post-processing of the
	// recorded WAV only; live monitoring/playback is unaffected.
	Normalize NormalizeConfig `yaml:"normalize"`
	// Dedup suppresses near-simultaneous duplicate recordings of the same call
	// heard on more than one monitored system — networked / simulcast sites
	// where the same talkgroup carries the same traffic, so a call is saved once
	// instead of once per site. Off by default. See DedupConfig.
	Dedup DedupConfig `yaml:"dedup"`
	// WarmDMRAudio selects the opt-in "ambe2-dmr-warm" vocoder for DMR
	// voice instead of the default "ambe2-dmr". It applies a gentle
	// output high-shelf that trims ~2 dB above ~1.5 kHz, softening the
	// bright/thin "digital" timbre of software AMBE+2 decode (issue #644).
	// It is a listener tone preference, not a codec-quality fix — the
	// residual synthetic character of low-bitrate AMBE+2 is intrinsic to
	// software decoding (only a DVSI hardware vocoder removes it). Off by
	// default; affects DMR only.
	WarmDMRAudio bool `yaml:"warm_dmr_audio"`
	// SpecAmplitudeEnhance toggles the spec-faithful §6.2 spectral-amplitude
	// enhancement for the software MBE-family vocoders (DMR AMBE+2 + P25 IMBE).
	// The spec-faithful form restores the π/ω₀ factor from TIA-102.BABA §6.2 /
	// mbelib's mbe_spectralAmpEnhance that the legacy form dropped, lifting the
	// high-band harmonic weights off the attenuate clamp (worst on small-L /
	// higher-pitched voices). Tri-state pointer: nil (unset) defaults ON, so
	// recorded + live audio gets the corrected envelope out of the box; set to
	// false to fall back to the legacy formula for an A/B comparison. Unlike
	// warm_dmr_audio this is a codec-correctness fix, not a tone preference, and
	// it affects DMR and P25 alike. On-air quality sign-off is still owed via
	// internal/voice/calibrate once reference WAVs are supplied.
	SpecAmplitudeEnhance *bool `yaml:"spec_amplitude_enhance"`
	// WriteCallJSON writes a trunk-recorder-compatible <basename>.json metadata
	// sidecar next to each recording (per WAV / per-transmission segment). It
	// carries the call's talkgroup, source, frequency, timing, flags, and the
	// per-talker srcList / freqList so the existing trunk-recorder parsers work
	// unchanged. Tri-state: unset (nil) defaults ON; set false to disable.
	WriteCallJSON *bool `yaml:"write_call_json,omitempty"`
	// FilenameTemplate customises the recording basename (no extension). Tokens
	// in braces are substituted per call: {date} (YYYYMMDD), {time} (HHMMSS),
	// {datetime} (YYYYMMDDTHHMMSS), {year} {month} {day}, {tg} (talkgroup/dest
	// id), {alpha} (talkgroup alpha tag, else id), {freq} (Hz), {src} (source
	// radio id), {ts} (timeslot), {proto}, {system}, {callid} (per-run serial).
	// All rendered in the display timezone. Empty selects the default
	// "{date}_{time}_{tg}" — e.g. 20260810_230041_1005479 — which drops the
	// timezone offset, source, and timeslot the older scheme put in every name
	// (they remain in the .json sidecar). Two calls that resolve to the same
	// basename (concurrent DMR TS1/TS2 on one talkgroup and second) are made
	// unique with a -2/-3 suffix, so a template need not include a disambiguator.
	FilenameTemplate string `yaml:"filename_template"`
	// PathTemplate customises the per-call directory tree under `dir`, using the
	// same tokens as FilenameTemplate plus "/" as the separator — e.g.
	// "{system}/{year}/{month}/{day}" for a trunk-recorder-style date tree.
	// Empty keeps the default layout: <system>/<talkgroup>/ for group calls and
	// <system>/individual/<dest-id>/ for individual calls. When set, the template
	// is used verbatim for every call (group and individual alike), so include
	// whatever grouping you want; the individual/group flag is still in the .json.
	PathTemplate string `yaml:"path_template"`
	// VoiceTapBufferChunks sizes the per-consumer buffer on the same-carrier
	// voice tap — the queue of post-DDC IQ chunks the control decoder fans to
	// each followed TETRA call. A deeper buffer absorbs more scheduling jitter
	// before a lagging voice consumer starts dropping IQ (the issue #402
	// "dropped IQ to a lagging voice consumer" starvation), at a small memory
	// cost: roughly depth × (post-DDC chunk length) × 8 bytes per active carrier
	// (concurrent same-carrier calls share one tap consumer, so the multiplier
	// is one, not one-per-call) — order tens to low-hundreds of KB. It is
	// headroom for bursty stalls, not a cure for sustained CPU starvation.
	// 0 selects the built-in default (128); the accepted range is 1..1024.
	VoiceTapBufferChunks int `yaml:"voice_tap_buffer_chunks"`
}

// EqualizerConfig is the YAML shape of the optional CMA equalizer in
// the per-call FM voice chain.
type EqualizerConfig struct {
	Enabled  bool    `yaml:"enabled"`
	Taps     int     `yaml:"taps"`      // default 8 when enabled
	StepSize float32 `yaml:"step_size"` // default 1e-4 when enabled
}

// NormalizeConfig is the YAML shape of the optional per-call loudness
// normalization. Defaults (applied when enabled and a field is zero):
// target -16 LUFS, true peak -1.5 dBTP, max gain ±12 dB.
type NormalizeConfig struct {
	Enabled      bool    `yaml:"enabled"`
	TargetLUFS   float64 `yaml:"target_lufs"`    // default -16.0 when enabled
	TruePeakDBTP float64 `yaml:"true_peak_dbtp"` // default -1.5 when enabled
	MaxBoostDB   float64 `yaml:"max_boost_db"`   // default 12.0 when enabled
	// ApplyTo selects which artifacts are normalized:
	//   "" / "recording" → rewrite the on-disk WAV (the distributed MP3,
	//                       encoded from that WAV, inherits the result)
	//   "distributed"    → leave the WAV pristine; normalize only the
	//                       outbound broadcast/stream MP3 copy
	//   "both"           → normalize the WAV and the distributed copy
	ApplyTo string `yaml:"apply_to"`
}

// AppliesToRecording reports whether the on-disk WAV should be normalized.
func (n NormalizeConfig) AppliesToRecording() bool {
	return n.Enabled && (n.ApplyTo == "" || n.ApplyTo == "recording" || n.ApplyTo == "both")
}

// AppliesToDistributed reports whether the outbound broadcast/stream MP3
// copy should be normalized in the broadcast subsystem.
func (n NormalizeConfig) AppliesToDistributed() bool {
	return n.Enabled && (n.ApplyTo == "distributed" || n.ApplyTo == "both")
}

// DedupConfig is the YAML shape of cross-site duplicate-recording suppression.
// When enabled, a call whose (talkgroup, source) was already recorded from a
// DIFFERENT monitored system within the window is skipped — so a call heard on
// several networked / simulcast sites is saved once instead of once per site. A
// re-key on the SAME system is always recorded. Keying on the source RID (which
// is globally unique in a network) means two genuinely different calls that
// happen to share a talkgroup number across systems still both record whenever
// their sources are known and differ.
type DedupConfig struct {
	Enabled bool `yaml:"enabled"`
	// WindowSeconds is how long a recorded call suppresses another system's copy
	// of the same (talkgroup, source); it should cover a typical over plus
	// inter-site skew. Default 60 when enabled.
	WindowSeconds int `yaml:"window_seconds"`
}

// Window returns the dedup suppression window, applying the 60 s default when
// enabled with an unset/invalid WindowSeconds.
func (d DedupConfig) Window() time.Duration {
	if d.WindowSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(d.WindowSeconds) * time.Second
}

// EnhanceConfig is the YAML shape of the opt-in voice enhancement chain
// (recordings.enhance). When Enabled, the recorder installs a
// post-vocoder chain — rumble high-pass, presence/warmth high-shelf,
// telephone-band low-pass, a louder AGC target, and an optional
// compressor — on the decoder it builds for each digital-voice call, so
// the chain shapes BOTH the recorded WAV and live monitoring. It trades
// strict faithfulness for the cleaner/louder subjective sound the rival
// decoders produce. Off by default; zero-value numeric fields backfill
// from the runtime defaults (see internal/voice/mbe.DefaultEnhancerConfig:
// HPF 250 Hz, LPF 3400 Hz, shelf 1.5 kHz/-2 dB, AGC target 22000).
type EnhanceConfig struct {
	Enabled bool `yaml:"enabled"`
	// HPFHz / LPFHz bound the output band (Hz). 0 ⇒ default; negative ⇒
	// that stage disabled.
	HPFHz float64 `yaml:"hpf_hz"`
	LPFHz float64 `yaml:"lpf_hz"`
	// ShelfHz / ShelfDB define the warmth high-shelf: ShelfDB dB of cut
	// above ShelfHz. ShelfDB ≤ 0 disables the shelf.
	ShelfHz float64 `yaml:"shelf_hz"`
	ShelfDB float64 `yaml:"shelf_db"`
	// AGCTarget overrides the decoder AGC peak target (int16 units, e.g.
	// 22000) so calls play back louder. 0 ⇒ default.
	AGCTarget float64 `yaml:"agc_target"`
	// Compress is the optional soft-knee compressor (default off).
	Compress CompressConfig `yaml:"compress"`
}

// CompressConfig is the YAML shape of the optional output compressor in
// the voice enhancement chain. Off by default.
type CompressConfig struct {
	Enabled     bool    `yaml:"enabled"`
	ThresholdDB float64 `yaml:"threshold_db"` // default -18 when enabled
	Ratio       float64 `yaml:"ratio"`        // default 2 when enabled
	AttackMs    float64 `yaml:"attack_ms"`    // default 5 when enabled
	ReleaseMs   float64 `yaml:"release_ms"`   // default 80 when enabled
	MakeupDB    float64 `yaml:"makeup_db"`    // default 0
}

// MetricsConfig toggles the Prometheus collector. The /metrics endpoint
// is mounted on the API HTTP server when both Enabled is true and the
// API HTTP address is configured.
type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
	// DetailedFEC opts into the per-protocol FEC correction-depth
	// histograms (today: gophertrunk_tetra_viterbi_corrections). Off by
	// default — the buckets only make sense to an operator profiling
	// on-air recovery margins against a known capture. See
	// docs/opt-in-features.md §5.
	DetailedFEC bool `yaml:"detailed_fec"`
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
		return cfg, fmt.Errorf("config %s: %w\n  hint: check YAML syntax (indentation must be spaces, keys end with ':'). Run `gophertrunk config` to build/repair a config interactively.", path, err)
	}
	// Resolve every filesystem path the config carries against the
	// directory holding config.yaml, so a portable config can ship with
	// config-relative defaults (../recordings, ../data/calls.db, …) that
	// land under the operator's chosen data root regardless of platform
	// or current working directory. Absolute and env-expanded-to-absolute
	// paths pass through untouched (see resolvePaths).
	cfg.resolvePaths(filepath.Dir(path))
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("config %s: %w", path, err)
	}
	return cfg, nil
}

// resolvePaths expands ~/$VAR/%VAR% references in every filesystem-path
// field and, when the result is still relative, anchors it to base (the
// directory containing the loaded config.yaml). Empty fields are left
// empty — they are "feature disabled" sentinels (storage.path,
// cc_cache_file, token_file, message_log.path) — and already-absolute
// paths are preserved, so existing absolute-path configs are unaffected.
func (c *Config) resolvePaths(base string) {
	resolve := func(p string) string {
		if p == "" {
			return ""
		}
		// Expand first, THEN test IsAbs: an expanded ${HOME}/x or
		// %USERPROFILE%\x is absolute and must not be re-anchored.
		p = pathutil.Expand(p)
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(base, p)
	}

	c.Storage.Path = resolve(c.Storage.Path)
	c.Storage.CCCacheFile = resolve(c.Storage.CCCacheFile)
	c.Recordings.Dir = resolve(c.Recordings.Dir)
	c.Log.MessageLog.Path = resolve(c.Log.MessageLog.Path)
	c.Log.PowerLog.Path = resolve(c.Log.PowerLog.Path)
	c.Log.EventLog.Path = resolve(c.Log.EventLog.Path)
	c.API.Auth.TokenFile = resolve(c.API.Auth.TokenFile)
	if c.Baseband.AutoRecord.Dir != "" {
		c.Baseband.AutoRecord.Dir = resolve(c.Baseband.AutoRecord.Dir)
	}
	for i := range c.Baseband.Record {
		c.Baseband.Record[i].Dir = resolve(c.Baseband.Record[i].Dir)
	}
	for i := range c.Baseband.Replay {
		c.Baseband.Replay[i].File = resolve(c.Baseband.Replay[i].File)
	}
	for i := range c.Trunking.Systems {
		c.Trunking.Systems[i].TalkgroupFile = resolve(c.Trunking.Systems[i].TalkgroupFile)
		c.Trunking.Systems[i].RIDAliasFile = resolve(c.Trunking.Systems[i].RIDAliasFile)
	}
	for i := range c.SDR.SoapyRemote {
		p := c.SDR.SoapyRemote[i].DiversityCapture
		if p == "" {
			continue
		}
		r := resolve(p)
		// filepath.Join strips a trailing separator, but the branch recorder
		// uses it to tell "directory to drop timestamped captures into" from
		// "filename prefix" — keep the operator's intent visible.
		if strings.HasSuffix(p, "/") || strings.HasSuffix(p, string(os.PathSeparator)) {
			r += string(os.PathSeparator)
		}
		c.SDR.SoapyRemote[i].DiversityCapture = r
	}
}
