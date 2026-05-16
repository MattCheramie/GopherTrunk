package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// wizardAnswers is the typed bundle of choices the config-builder
// wizard collected. configTemplate consumes it deterministically to
// produce a fully-annotated config.yaml that the daemon's
// internal/config loader accepts without modification.
//
// Every field has a documented zero value that matches the daemon's
// hard-coded defaults so an operator who hits Enter through every
// step still gets a valid, runnable config.
type wizardAnswers struct {
	// Where the rendered config will be written. Tracked here so the
	// review screen can show the destination; the template ignores it.
	ConfigPath string

	// log
	LogLevel  string // "info" | "debug" | "warn" | "error"
	LogFormat string // "text" | "json"

	// api
	HTTPAddr           string // e.g. "127.0.0.1:8080" or "0.0.0.0:8080"
	GRPCAddr           string // e.g. "127.0.0.1:50051" — "" disables gRPC
	AuthMode           string // "auto" | "required" | "disabled"
	AuthTokenFile      string // optional path to a bearer-token file
	CORSAllowedOrigins []string

	// metrics
	MetricsEnabled bool

	// storage
	StoragePath        string
	StorageCCCacheFile string

	// recordings
	RecordingsDir      string
	RecordingsSampleHz int
	RecordingsWriteRaw bool
	EqualizerEnabled   bool

	// retention
	RetentionCallLogDays int
	RetentionFilesDays   int
	RetentionInterval    string // Go duration ("1h", "30m")

	// sdr
	SDRSampleHz int
	SDRDevices  []wizardSDR

	// scanner
	ScannerScanMode    string // "all" | "list"
	CCHuntEnabled      bool
	CCHuntDwellMs      int
	CCHuntBackoffMs    int
	CCHuntMaxBackoffMs int
	ManualTuneEnabled  bool

	// audio
	AudioEnabled  bool
	AudioDevice   string
	AudioSampleHz int
	AudioBufferMs int
	AudioVolume   float64
	AudioMuted    bool
}

// wizardSDR is one entry in sdr.devices. Mirrors internal/config.SDRDevice
// but keeps the wizard's representation deliberately simple.
type wizardSDR struct {
	Serial  string
	Role    string // "control" | "voice" | "auto"
	PPM     int
	Gain    string // "auto" or tenths-of-dB integer ("496" = 49.6 dB)
	BiasTee bool
}

// defaultWizardAnswers seeds the wizard with values that satisfy
// internal/config.Validate() across the board. An operator who walks
// through the wizard hitting Enter on every step still produces a
// usable config — the daemon starts, the API listens on loopback,
// and the only thing left for the operator is to populate SDR
// serials and (optionally) trunked systems via -pdf / -csv.
func defaultWizardAnswers() wizardAnswers {
	return wizardAnswers{
		ConfigPath: defaultConfigPath(),

		LogLevel:  "info",
		LogFormat: "text",

		HTTPAddr:           "127.0.0.1:8080",
		GRPCAddr:           "127.0.0.1:50051",
		AuthMode:           "auto",
		AuthTokenFile:      "",
		CORSAllowedOrigins: nil,

		MetricsEnabled: true,

		StoragePath:        "/var/lib/gophertrunk/calls.db",
		StorageCCCacheFile: "/var/lib/gophertrunk/cc-cache.json",

		RecordingsDir:      "/var/lib/gophertrunk/recordings",
		RecordingsSampleHz: 8000,
		RecordingsWriteRaw: true,
		EqualizerEnabled:   false,

		RetentionCallLogDays: 30,
		RetentionFilesDays:   14,
		RetentionInterval:    "1h",

		SDRSampleHz: 2_400_000,
		SDRDevices:  nil,

		ScannerScanMode:    "all",
		CCHuntEnabled:      true,
		CCHuntDwellMs:      3000,
		CCHuntBackoffMs:    5000,
		CCHuntMaxBackoffMs: 60000,
		ManualTuneEnabled:  false,

		AudioEnabled:  false,
		AudioDevice:   "",
		AudioSampleHz: 8000,
		AudioBufferMs: 80,
		AudioVolume:   0.8,
		AudioMuted:    false,
	}
}

// renderConfigYAML emits a complete, annotated config.yaml from the
// wizard's answers. The output mirrors config.example.yaml's section
// structure so an operator who skims it sees the same comments they'd
// see in the canonical example.
//
// trunking.systems is left empty here — the importer's existing merge
// path handles -pdf / -csv inputs on top of this file. When the wizard
// is run without imports the user gets the daemon-startable scaffold
// with a placeholder comment showing where to add systems.
func renderConfigYAML(a wizardAnswers) ([]byte, error) {
	t, err := template.New("config").Funcs(template.FuncMap{
		"yamlString": yamlString,
	}).Parse(configTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, a); err != nil {
		return nil, fmt.Errorf("render template: %w", err)
	}
	return buf.Bytes(), nil
}

// yamlString quotes a string for embedding inside double-quoted YAML
// scalars. Conservative: always emits with double quotes so the
// output is unambiguous even for values that look like numbers or
// booleans. Returns the bare value when it would shadow a key name.
func yamlString(s string) string {
	// Escape backslashes and double-quotes for YAML double-quoted form.
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

// configTemplate is the canonical output shape. Keeping it in a Go
// raw string (rather than an embedded file) avoids the embed.FS
// indirection — the wizard is single-binary, no external assets.
//
// Section order matches config.example.yaml so a reader who knows
// one knows the other.
const configTemplate = `# GopherTrunk daemon configuration. Generated by ` + "`gophertrunk import-pdf -wizard`" + `.
# Edit this file by hand whenever you like; re-running the wizard
# rebuilds it from scratch. Section comments mirror config.example.yaml.

log:
  level: {{.LogLevel}}       # debug | info | warn | error
  format: {{.LogFormat}}      # text | json

api:
  http_addr: {{yamlString .HTTPAddr}}   # HTTP REST + SSE + WebSocket
  grpc_addr: {{yamlString .GRPCAddr}}  # gRPC; empty disables the gRPC server

  # Bearer-token authentication on every mutation endpoint (end call,
  # talkgroup priority/lockout, retention sweep, tone-out reset,
  # scanner cockpit, audio cockpit).
  #
  # mode:
  #   auto      — require a token on non-loopback binds; bypass on
  #               127.0.0.1 / ::1 (peer-cred via reachability is a
  #               reasonable trust proxy on a single-host operator
  #               box). Recommended for most deployments.
  #   required  — require a token on every mutation, even loopback.
  #   disabled  — wide-open mutations, no auth. The daemon logs a
  #               startup warning.
  auth:
    mode: {{yamlString .AuthMode}}
{{- if .AuthTokenFile}}
    token_file: {{yamlString .AuthTokenFile}}
{{- else}}
    # token_file: "/etc/gophertrunk/api-token"   # preferred over inline token
{{- end}}
    # trusted_networks:
    #   - "10.0.0.0/8"
    #   - "192.168.0.0/16"

  # Legacy gate. Setting true logs a deprecation warning and maps to
  # auth.mode: disabled. Prefer auth.mode for new deployments.
  allow_mutations: false

  # Cross-origin browser access. Off by default — the daemon emits
  # no Access-Control-* headers and rejects WebSocket upgrades whose
  # Origin header isn't on the allow-list. Enable when serving the
  # bundled web UI (gophertrunk-web/) from a different origin than
  # the daemon. Use "null" for the file:// case (laptop opens
  # gophertrunk-web/index.html directly).
  cors:
{{- if .CORSAllowedOrigins}}
    allowed_origins:
{{- range .CORSAllowedOrigins}}
      - {{yamlString .}}
{{- end}}
{{- else}}
    allowed_origins: []
    # allowed_origins:
    #   - "null"
    #   - "http://laptop.local:8000"
{{- end}}

metrics:
  enabled: {{.MetricsEnabled}}     # mounts /metrics on the HTTP API

storage:
  path: {{yamlString .StoragePath}}
  cc_cache_file: {{yamlString .StorageCCCacheFile}}

recordings:
  dir: {{yamlString .RecordingsDir}}
  sample_rate: {{.RecordingsSampleHz}}
  write_raw: {{.RecordingsWriteRaw}}   # also append a .raw sidecar with vocoder frames
  equalizer:
    enabled: {{.EqualizerEnabled}}   # CMA blind equalizer in the FM voice chain (simulcast mitigation)
    taps: 8
    step_size: 0.0001

retention:
  call_log_days: {{.RetentionCallLogDays}}   # 0 disables call-log row sweep
  files_days: {{.RetentionFilesDays}}        # 0 disables filesystem sweep
  interval: {{yamlString .RetentionInterval}}      # how often the sweeper runs

sdr:
  sample_rate: {{.SDRSampleHz}}
{{- if .SDRDevices}}
  devices:
{{- range .SDRDevices}}
    - serial: {{yamlString .Serial}}
      role: {{.Role}}           # control | voice | auto
      ppm: {{.PPM}}
      gain: {{yamlString .Gain}}        # "auto" or tenths-of-dB string ("496" = 49.6 dB)
      bias_tee: {{.BiasTee}}
{{- end}}
{{- else}}
  # devices is empty — the daemon will start but won't lock any control
  # channel until you add at least one entry below and re-run. Match
  # the serial(s) reported by ` + "`gophertrunk sdr list`" + `.
  devices: []
  # devices:
  #   - serial: "00000001"
  #     role: control          # control | voice | auto
  #     ppm: 0                 # 0 is fine for TCXO-equipped units (NESDR Smart v5)
  #     gain: "auto"           # "auto" or tenths-of-dB ("496" = 49.6 dB)
  #     bias_tee: false
{{- end}}

trunking:
  # trunking.systems is normally populated by ` + "`gophertrunk import-pdf`" + `
  # from a RadioReference PDF export or a structured CSV bundle.
  # Re-running the importer merges new entries here without disturbing
  # the surrounding comments.
  systems: []

# Police-scanner subsystems:
#   - scan_mode controls the engine's per-grant gate. "all" follows every
#     non-locked-out grant (the original behaviour). "list" only follows
#     grants whose talkgroup carries Scan=true; Emergency grants bypass.
#   - cc_hunt is the multi-system control-channel scanner. Operator can
#     hold/resume/force-retune any system from the TUI Scanner panel.
#   - conventional is the fixed-frequency analog FM scan list. Requires
#     a dedicated Voice SDR (the last voice device in the pool is used).
scanner:
  scan_mode: {{.ScannerScanMode}}       # all | list
  cc_hunt:
    enabled: {{.CCHuntEnabled}}
    dwell_ms: {{.CCHuntDwellMs}}
    backoff_ms: {{.CCHuntBackoffMs}}
    max_backoff_ms: {{.CCHuntMaxBackoffMs}}
  manual_tune_enabled: {{.ManualTuneEnabled}}
  manual_tune_disabled: false
  conventional: []
  # Example conventional channels:
  # conventional:
  #   - label: "Sheriff Repeater"
  #     frequency_hz: 155895000
  #     mode: fm
  #     squelch_dbfs: -48
  #     hangtime_ms: 1500
  #     priority: 4
  #     tone:
  #       mode: ctcss            # ctcss | dcs | none
  #       ctcss_hz: 100.0

# audio routes decoded PCM to the host's speakers. Disabled by default;
# WAV recording is unaffected and continues whether audio is on or off.
audio:
  enabled: {{.AudioEnabled}}       # set true to play decoded calls live
  device: {{yamlString .AudioDevice}}            # empty = system default sink
  sample_rate: {{.AudioSampleHz}}    # must match recordings.sample_rate
  buffer_ms: {{.AudioBufferMs}}       # playback queue depth
  volume: {{printf "%.2f" .AudioVolume}}         # 0..1 software gain
  muted: {{.AudioMuted}}

# tone_out fires events.KindToneAlert when the configured paging tones
# are detected on a Voice device's PCM stream. Wizard leaves the list
# empty — operators with two-tone QC-II / supervisory pages to detect
# should add profile entries by hand. Schema documented in
# internal/voice/toneout/profile.go.
tone_out:
  profiles: []
`
