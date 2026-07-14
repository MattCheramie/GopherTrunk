package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"strings"
	"sync"
	"time"

	gtdiag "github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// EngineSnapshot is the subset of trunking.Engine the API needs. Decoupling
// from the concrete type keeps the API testable with a fake engine.
type EngineSnapshot interface {
	ActiveCalls() []*trunking.ActiveCall
	// ObservedCalls returns voice calls the control channel announced that no
	// voice tuner is following (every device busy). The active-calls endpoint
	// merges them with ActiveCalls so the operator sees every talkgroup up on
	// the system, not just the ones a tuner is decoding.
	ObservedCalls() []*trunking.ActiveCall
}

// EngineMutator is the optional write side of the engine. Daemons
// that have AllowMutations enabled supply a real engine; tests can
// inject a fake. When nil the end-call route returns 503.
type EngineMutator interface {
	EndCall(deviceSerial string, reason trunking.EndReason) bool
}

// RetentionSweeper is the optional write side of the retention
// system: kick off one ad-hoc sweep. The daemon supplies the real
// sweeper from internal/storage; tests can fake it.
type RetentionSweeper interface {
	SweepOnce(ctx context.Context)
}

// ToneDetectorReset is the optional write side of the tone-out
// detector: clear per-device match progress without throwing away
// the cooldown clock. Daemons that wire the detector supply the
// real impl; tests can fake it.
type ToneDetectorReset interface {
	ResetDevice(serial string)
}

// DevicesProvider returns a snapshot of the SDR pool. The api package
// stays free of a hard dependency on internal/sdr's implementation
// details; the daemon supplies *sdr.Pool, tests supply a fake.
type DevicesProvider interface {
	Snapshot() []sdr.SDRStatus
}

// AudioController is the API surface for the live-audio subsystem
// (the voice.Player sink + the WAV recorder gate). All four methods
// are safe to call from any goroutine; the daemon supplies a single
// adapter that fans into player.Player + voice.Recorder, tests use a
// fake.
type AudioController interface {
	// Volume returns the current software gain (0..1).
	Volume() float32
	// SetVolume clamps to 0..1 and applies immediately.
	SetVolume(v float32)
	// Muted reports the mute state.
	Muted() bool
	// SetMuted toggles mute. Mute is a software-gain bypass, not a
	// device-level operation — toggling is instant.
	SetMuted(m bool)
	// RecordingEnabled reports whether the recorder's "create new
	// sessions" gate is open. In-flight sessions are not affected
	// by this gate.
	RecordingEnabled() bool
	// SetRecordingEnabled flips the recorder gate. False stops new
	// WAVs from landing on disk; in-flight sessions complete.
	SetRecordingEnabled(enabled bool)
	// DropsTotal is a monotonically increasing counter of PCM
	// samples lost because the playback queue was full. Surfaced
	// so operators can spot scheduling-jitter problems from the
	// TUI without reaching for /metrics.
	DropsTotal() uint64
	// SampleRate is the host playback rate the player was opened
	// at, in Hz. Read-only; reopening the device with a different
	// rate requires a daemon restart.
	SampleRate() uint32
	// BackendEnabled reports whether a real audio backend is
	// attached. False means audio.enabled was off in config or the
	// backend failed to init, and writes are silently dropped.
	BackendEnabled() bool
	// BackendError returns the reason the backend isn't attached when
	// audio was enabled but the sink failed to open (e.g. the ioctl
	// device rejected the pinned format). Empty when audio is healthy
	// or was intentionally disabled — lets the UI distinguish "off by
	// config" from "tried to start and failed".
	BackendError() string
}

// BroadcastStatusProvider is the read side of the outbound
// call-streaming subsystem (internal/broadcast). BroadcastStats
// returns a JSON-serialisable counter snapshot; the daemon adapts
// broadcast.Manager.Stats() to this interface so the api package
// keeps no compile-time dependency on internal/broadcast.
type BroadcastStatusProvider interface {
	BroadcastStats() any
}

// HuntCockpit is the API surface for the live system-discovery ("hunt")
// subsystem: read the current run state + discovered map, start/stop a blind
// spectrum-sweep run, and export or commit the result. The daemon supplies a
// single implementation wrapping the hunt.Manager; tests can stub it.
type HuntCockpit interface {
	// Status returns the current run snapshot (+ discovered system when ready).
	Status() HuntStatus
	// Start launches a live hunt; returns the new run id, or an error (e.g. a
	// run is already active, or no SDR is available).
	Start(req HuntStartRequest) (runID int, err error)
	// Stop cancels the active run; false when nothing is running.
	Stop() bool
	// Export serializes a run's discovered system in the named format
	// (bundle|trunk-recorder|rr), returning the bytes + a suggested filename.
	// id 0 = the latest run. Returns ErrHuntNoSuchRun for an unknown id.
	Export(id int, format string) (data []byte, filename string, err error)
	// ExportSurvey serializes a run's classified signal inventory in the named
	// format (json|csv). id 0 = the latest run. Returns ErrHuntNoSuchRun for an
	// unknown id, or an error when the run was a plain hunt (no survey).
	ExportSurvey(id int, format string) (data []byte, filename string, err error)
	// CaptureSignal records raw IQ of one signal from the inventory and routes
	// it to SigLab/CryptoLab — the daemon side of the list-driven capture.
	// Returns an "in progress" error while a run is active.
	CaptureSignal(req HuntCaptureRequest) (HuntCaptureResult, error)
	// Commit merges a run's discovered system into config.yaml (id 0 = latest),
	// returning a human-readable list of changes.
	Commit(id int, force, dryRun bool) (changes []string, err error)
	// RadioReference cross-references a run's discovered system against
	// RadioReference (id 0 = latest): duplicate-system hints plus a
	// frequency/talkgroup diff. countyID and checkSIDs select the existing
	// systems to compare against. Returns an error when RR credentials are not
	// configured or no system has been discovered yet.
	RadioReference(id, countyID int, checkSIDs []int) (HuntRRReport, error)
}

// ScannerCockpit is the API surface for the police-scanner subsystem:
// reads the current state (per-system CC hunt, conventional channel
// list, talkgroup-scan stats) and applies operator mutations from
// the TUI (hold/resume/retune the hunter, hold/resume/dwell-on the
// conventional scanner, flip the global scan mode).
//
// The daemon supplies a single ScannerCockpit implementation that
// aggregates the cchunt.Supervisor + conventional.Scanner + engine;
// tests can stub a single struct that satisfies the whole interface.
type ScannerCockpit interface {
	// Status returns the unified read snapshot the TUI panel renders.
	Status() ScannerStatus
	// SetScanMode flips the global TG-scan-list mode at runtime.
	// Returns the previous mode for audit / UX feedback.
	SetScanMode(mode string) (prev string, err error)
	// HoldHunt / ResumeHunt / ForceRetuneHunt apply to a single
	// trunked system. Returns false when the system isn't configured.
	HoldHunt(system string) bool
	ResumeHunt(system string) bool
	ForceRetuneHunt(system string) bool
	// HoldConventional / ResumeConventional / DwellConventional
	// drive the conventional FM scanner. DwellConventional indexes
	// into the configured Channels list. The Hold/Resume operations
	// return false when the conventional scanner isn't configured.
	HoldConventional() bool
	ResumeConventional() bool
	DwellConventional(index int) bool
	// LockoutConventional / UnlockoutConventional toggle the per-
	// channel lockout flag the scan loop respects. Locked-out
	// channels are skipped by pickNextChannel. Returns false when
	// the conventional scanner isn't configured or the index is
	// out of range.
	LockoutConventional(index int) bool
	UnlockoutConventional(index int) bool
	// ManualTune appends a VFO-style temporary channel to the
	// conventional scanner and forces dwell on it. Returns the new
	// index + ok=true on success; ok=false when the conventional
	// scanner isn't configured (no Voice SDR carved out for it).
	ManualTune(req ManualTuneRequest) (index int, ok bool)
	// ClearManualTune removes a previously-added temp channel by
	// index. Returns false if the index isn't a temp channel or
	// the scanner isn't configured.
	ClearManualTune(index int) bool
}

// ManualTuneRequest is the shape of POST /api/v1/scanner/manual_tune.
// FrequencyHz is required; everything else falls back to scanner
// defaults (Mode=fm, SquelchDbFS=-50, Hangtime=1500ms).
type ManualTuneRequest struct {
	FrequencyHz uint32  `json:"frequency_hz"`
	Label       string  `json:"label"`
	Mode        string  `json:"mode"`
	SquelchDbFS float64 `json:"squelch_dbfs"`
	HangtimeMs  int     `json:"hangtime_ms"`
}

// ScannerStatus is the JSON shape returned by GET /api/v1/scanner —
// a unified view over all three scanner-subsystem read surfaces.
type ScannerStatus struct {
	ScanMode            string                `json:"scan_mode"`
	Systems             []SystemHuntStatusDTO `json:"systems"`
	Conventional        ConvScannerStatusDTO  `json:"conventional"`
	TalkgroupScanCount  int                   `json:"tg_scan_count"`
	TalkgroupTotalCount int                   `json:"tg_total"`
}

// SystemHuntStatusDTO mirrors cchunt.SystemStatus for the wire layer
// so the api package doesn't import internal/scanner.
type SystemHuntStatusDTO struct {
	Name            string    `json:"name"`
	Protocol        string    `json:"protocol"`
	State           string    `json:"state"`
	AttemptedFreqHz uint32    `json:"attempted_freq_hz,omitempty"`
	AttemptIndex    int       `json:"attempt_index,omitempty"`
	TotalCandidates int       `json:"total_candidates,omitempty"`
	LockedFreqHz    uint32    `json:"locked_freq_hz,omitempty"`
	LockedAt        time.Time `json:"locked_at,omitempty"`
	NAC             uint16    `json:"nac,omitempty"`
	LastFailedAt    time.Time `json:"last_failed_at,omitempty"`
	BackoffMs       int       `json:"backoff_ms,omitempty"`
	LastGrantAt     time.Time `json:"last_grant_at,omitempty"`
}

// ConvScannerStatusDTO is the conventional FM scanner's read shape.
type ConvScannerStatusDTO struct {
	Enabled      bool                   `json:"enabled"`
	State        string                 `json:"state,omitempty"`
	DeviceSerial string                 `json:"device_serial,omitempty"`
	CursorIndex  int                    `json:"cursor_index,omitempty"`
	Channels     []ConvChannelStatusDTO `json:"channels"`
}

// ConvChannelStatusDTO mirrors conventional.ChannelStatus.
type ConvChannelStatusDTO struct {
	Index       int       `json:"index"`
	Label       string    `json:"label"`
	FrequencyHz uint32    `json:"frequency_hz"`
	Mode        string    `json:"mode"`
	Active      bool      `json:"active"`
	LockedOut   bool      `json:"locked_out,omitempty"`
	LastBreakAt time.Time `json:"last_break_at,omitempty"`
}

// Server hosts the GopherTrunk HTTP/SSE/WebSocket API. A separate gRPC
// server (internal/api/grpc.go) shares the same in-process state.
type Server struct {
	addr string
	// boundAddr is populated by Run() with the listener's actual
	// address after net.Listen — important for ":0" / "127.0.0.1:0"
	// configurations where the kernel picks the port. Read via
	// BoundAddr(). Empty until Run() has bound (or after Close).
	boundAddr    string
	bus          *events.Bus
	engine       EngineSnapshot
	mutator      EngineMutator
	retention    RetentionSweeper
	tones        ToneDetectorReset
	devices      DevicesProvider
	scanner      ScannerCockpit
	hunt         HuntCockpit
	audio        AudioController
	broadcast    BroadcastStatusProvider
	runtime      RuntimeProvider
	configWriter ConfigWriter
	configActiv  ConfigActivator
	settings     SettingsApplier
	importer     Importer
	imports      *importStaging
	webAssets    fs.FS
	talkgroups   *trunking.TalkgroupDB
	rids         *trunking.RIDDB
	systems      []trunking.System
	history      HistoryQuery
	locations    LocationQuery
	affiliations AffiliationProvider
	sites        SitesProvider
	metrics      http.Handler
	log          *slog.Logger
	version      string

	auth *authState
	// allowMutations is kept for backwards compatibility with
	// callers that haven't migrated to AuthConfig yet. When set
	// without an explicit AuthConfig the server constructs an
	// AuthModeDisabled state (legacy wide-open behaviour).
	allowMutations bool

	tlsCert string
	tlsKey  string

	cors CORSConfig
	// audioPub is the optional publisher feeding the new
	// /api/v1/audio/stream HTTP endpoint. The daemon shares its
	// existing *AudioPublisher (the same instance backing gRPC
	// StreamAudio) so the HTTP stream is a parallel subscriber on
	// the same fan-out. nil disables the route.
	audioPub *AudioPublisher

	// spectrum is the optional provider backing /api/v1/spectrum/...
	// routes. nil disables the routes. Implemented by the daemon
	// over its iqtap broker map.
	spectrum SpectrumProvider

	// capture is the optional provider backing the runtime
	// POST /api/v1/siglab/capture route (record raw IQ off a live
	// tuner). nil keeps the route returning 503. Implemented by the
	// daemon over the same iqtap broker map as spectrum.
	capture CaptureProvider

	// bookmarks is the optional provider backing /api/v1/bookmarks/...
	// routes. nil disables the routes (503). Implemented by the
	// daemon over storage.BookmarkStore.
	bookmarks BookmarkProvider

	// diag is the optional provider backing /api/v1/diag/...
	// routes (decimated-IQ stream for the Constellation panel).
	// nil disables the routes (503). Implemented by the daemon
	// over the iqtap broker + internal/dsp/diag.
	diag DiagProvider

	// symbols is the optional provider backing the
	// WS /api/v1/diag/symbols recovered-symbol stream for the Symbol
	// scope panel. nil disables the route (503). Implemented by the
	// daemon over the iqtap broker + internal/scanner/symbolscope.
	symbols SymbolProvider

	// mixer is the optional provider backing the WS /api/v1/diag/mixer
	// channel-baseband spectrum stream for the Mixer plot (OP25's Raw /
	// Tuned Mixer tabs). nil disables the route (503). Implemented by the
	// daemon over the iqtap broker + internal/scanner/symbolscope.
	mixer MixerProvider

	// pager is the optional provider backing /api/v1/pager/...
	// routes (pager log). nil disables the routes. Implemented
	// by the daemon over the SQLite-backed storage.PagerLog.
	pager PagerProvider

	// aprs is the optional provider backing /api/v1/aprs/...
	// routes (APRS / AX.25 packet log). nil disables the routes.
	// Implemented by the daemon over the SQLite-backed
	// storage.APRSLog.
	aprs APRSProvider

	// ais is the optional provider backing /api/v1/ais/...
	// routes (AIS / vessel-tracking log). nil disables the routes.
	// Implemented by the daemon over the SQLite-backed
	// storage.VesselLog.
	ais AISProvider

	// dsc is the optional provider backing /api/v1/dsc/...
	// routes (marine DSC sequence log). nil disables the routes.
	// Implemented by the daemon over the SQLite-backed
	// storage.DSCLog.
	dsc DSCProvider

	// m17 is the optional provider backing /api/v1/m17/... routes
	// (M17 link-setup log). nil disables the routes. Implemented by
	// the daemon over the SQLite-backed storage.M17Log.
	m17 M17Provider
	// lora is the optional provider backing /api/v1/lora/... routes
	// (LoRa frame log). nil disables the routes. Implemented by the
	// daemon over the SQLite-backed storage.LoRaLog.
	lora LoRaProvider

	// adsb is the optional provider backing /api/v1/adsb/...
	// routes (ADS-B aircraft report log). nil disables the
	// routes. Implemented by the daemon over the SQLite-backed
	// storage.AircraftLog.
	adsb ADSBProvider

	// mdc1200 is the optional provider backing /api/v1/mdc1200/...
	// routes (MDC1200 signaling-burst log). nil disables the routes.
	// Implemented by the daemon over the SQLite-backed
	// storage.MDC1200Log.
	mdc1200 MDC1200Provider

	// siglab is the optional offline signal-analysis subsystem backing
	// the /api/v1/siglab/* routes (capture upload, engine run + SSE,
	// identify, synthesize, export, decimated-IQ retrieval). nil disables
	// the routes. Constructed by NewServer when ServerOptions.Siglab.Enabled.
	siglab *siglabService
	// siglabStop signals the siglab TTL sweeper to exit on shutdown.
	siglabStop chan struct{}
	// siglabAssets is the optional Signal Lab SPA file tree. When non-nil (and
	// carrying an index.html) the daemon mounts the console at /siglab/. Set
	// from ServerOptions.Siglab.Assets.
	siglabAssets fs.FS
	// siglabConsole is set true when the Signal Lab SPA is mounted at /siglab/.
	// The runtime endpoint surfaces it so the other consoles only link to
	// Signal Lab when it is actually reachable.
	siglabConsole bool

	// rfscopeOpts holds the RF Scope console configuration (enabled flag + SPA
	// assets). mountRFScope reads it when building the mux. Set from
	// ServerOptions.RFScope.
	rfscopeOpts RFScopeOptions
	// rfscopeCloser holds the optional RF Scope web subsystem (the /rfscope/
	// console + /api/v1/rfscope/* routes). Constructed by mountRFScope when
	// ServerOptions.RFScope.Enabled. Typed as a minimal closer so shutdown can
	// reclaim its temp dir without server.go importing the concrete type.
	rfscopeCloser interface{ Close() error }
	// rfscopeConsole is set true when the RF Scope SPA is mounted at /rfscope/.
	// Surfaced on the runtime endpoint so the other consoles only link to RF
	// Scope when it is actually reachable.
	rfscopeConsole bool

	// cryptolabCloser holds the optional cryptolab web subsystem (the
	// /cryptolab/ console + /api/v1/cryptolab/* routes). It is only
	// constructed in builds tagged `cryptolab` (see cryptolab_enabled.go);
	// the default build leaves it nil. Typed as a minimal closer so the
	// non-tagged server.go does not import the cryptolab packages.
	cryptolabCloser interface{ Close() error }

	// cryptolabConsole is set true when the cryptolab SPA is mounted at
	// /cryptolab/ (a -tags cryptolab build with the SPA bundled). The runtime
	// endpoint surfaces it so the other consoles only link to the Crypto Lab
	// when it is actually reachable.
	cryptolabConsole bool

	// bundleRoot, when non-empty, confines the /api/v1/bundle/* endpoints to
	// GopherTrunk Bundles under this directory, so a web client cannot read
	// arbitrary files off the host. Empty allows any .gtb path the daemon can
	// stat (the standalone/dev default).
	bundleRoot string

	// configBuilder backs the standalone web Config Builder/Editor:
	// the /api/v1/config/* routes (browse / load / validate / save /
	// RadioReference browse) and, when assets are wired, the builder SPA
	// mounted at /config/. nil disables all of it. Set by the daemon and
	// the standalone `gophertrunk config serve` command.
	configBuilder *configBuilderService

	// diagnostics is the shared error-diagnostics collector (banner +
	// system info). nil disables banner enrichment of error responses
	// and the GET /api/v1/diag/banner route.
	diagnostics *gtdiag.Collector
	// verboseErrors mirrors diagnostics.verbose_errors; see
	// ServerOptions.VerboseErrors.
	verboseErrors bool
	// displayLoc is the timezone the daemon-time field in GET /api/v1/health
	// renders in. Never nil after construction; mirrors display.timezone.
	displayLoc *time.Location

	mu     sync.Mutex
	srv    *http.Server
	closed bool
}

// HistoryQuery is the subset of storage.DB the history endpoint needs.
// Decoupling keeps the api package free of a hard dependency on the
// storage package and lets tests inject fakes.
type HistoryQuery interface {
	History(ctx context.Context, f HistoryFilter) ([]CallRow, error)
}

// LocationFix is one geographic fix returned by GET /api/v1/locations.
type LocationFix struct {
	System     string  `json:"system"`
	Protocol   string  `json:"protocol"`
	RadioID    uint32  `json:"radio_id"`
	Talkgroup  uint32  `json:"talkgroup"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	SpeedKnots float64 `json:"speed_knots"`
	HeadingDeg float64 `json:"heading_deg"`
	ReportedAt string  `json:"reported_at"` // RFC3339
}

// LocationQuery is the read side of the GPS/location subsystem,
// supplying recent fixes for GET /api/v1/locations and the web map.
type LocationQuery interface {
	RecentLocations(limit int) ([]LocationFix, error)
}

// AffiliationProvider is the read side of the affiliation tracker,
// supplying the unit-activity table for GET /api/v1/affiliations.
type AffiliationProvider interface {
	Affiliations() []trunking.UnitActivity
}

// SitesProvider is the read side of the site tracker, supplying the
// P25 sites discovered from the control channel for GET /api/v1/sites
// (issue #698).
type SitesProvider interface {
	Sites() []trunking.SiteInfo
}

// NetworkReporter is the optional capability a SitesProvider may implement to
// render the live, human-readable network-configuration report for a system
// (the SiteTracker does). GET /api/v1/systems/{name}/report uses it when the
// wired provider supports it; the report reflects the most recently camped site
// of the system. The bool is false when no topology has been observed yet.
type NetworkReporter interface {
	Report(system string) (string, bool)
}

// orLocal returns loc, or time.Local when loc is nil, so timestamp rendering
// always has a usable *time.Location.
func orLocal(loc *time.Location) *time.Location {
	if loc == nil {
		return time.Local
	}
	return loc
}

// NetworkTopologyProvider is the optional capability a SitesProvider may
// implement to expose the live topology snapshot (identity + neighbours + band
// plan) for a system. The SiteTracker implements it; the systems endpoints use
// it to overlay the decoded adjacent-site ("neighbor") list onto a SystemDTO so
// the Systems panels can show neighbours without the drill-in report. The bool
// is false when no topology has been observed for the system yet.
type NetworkTopologyProvider interface {
	Topology(system string) (*trunking.TopologySnapshot, bool)
}

// HistoryFilter mirrors storage.HistoryFilter for the api layer's
// purposes (passed through to whatever HistoryQuery implementation the
// daemon wires up).
type HistoryFilter struct {
	System    string
	GroupID   uint32
	SourceID  uint32
	Since     time.Time
	Until     time.Time
	Limit     int
	OnlyEnded bool
}

// CallRow mirrors storage.CallRow as a JSON-friendly row. Lives in the
// api package so the storage package can stay free of API concerns.
type CallRow struct {
	ID          int64  `json:"id"`
	System      string `json:"system"`
	Protocol    string `json:"protocol"`
	GroupID     uint32 `json:"group_id"`
	SourceID    uint32 `json:"source_id"`
	FrequencyHz uint32 `json:"frequency_hz"`
	Encrypted   bool   `json:"encrypted"`
	Emergency   bool   `json:"emergency"`
	DataCall    bool   `json:"data_call"`
	// Timeslot is the 1-based DMR TDMA slot (0 = n/a, 1 = TS1, 2 = TS2).
	Timeslot       uint8     `json:"timeslot,omitempty"`
	DeviceSerial   string    `json:"device_serial"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at,omitempty"`
	DurationMs     int64     `json:"duration_ms,omitempty"`
	EndReason      string    `json:"end_reason,omitempty"`
	TalkgroupAlpha string    `json:"talkgroup_alpha,omitempty"`
	// SignalDbFS is the call's mean received channel power in dBFS
	// (channel power, not calibrated RSSI or SNR). nil when unmeasured.
	SignalDbFS *float64 `json:"signal_dbfs,omitempty"`
}

// ServerOptions configure a new Server.
type ServerOptions struct {
	// Addr is the listen address (e.g. ":8080" or "127.0.0.1:9000").
	Addr       string
	Bus        *events.Bus
	Engine     EngineSnapshot
	Talkgroups *trunking.TalkgroupDB
	// RIDs is the operator-configured radio-ID alias table. When nil
	// the server allocates an empty one so the routes serve a stable
	// shape; the daemon passes a populated DB loaded from each
	// system's rid_alias_file.
	RIDs    *trunking.RIDDB
	Systems []trunking.System
	// History is optional. When non-nil the server exposes
	// GET /api/v1/calls/history.
	History HistoryQuery
	// Locations is optional. When non-nil the server exposes
	// GET /api/v1/locations for the web map.
	Locations LocationQuery
	// Affiliations is optional. When non-nil the server exposes
	// GET /api/v1/affiliations (the unit-activity table).
	Affiliations AffiliationProvider
	// Sites is optional. When non-nil the server exposes
	// GET /api/v1/sites (the discovered-P25-site table, issue #698).
	Sites SitesProvider
	// MetricsHandler is optional. When non-nil it is mounted at
	// GET /metrics; the daemon passes internal/metrics.Metrics.Handler()
	// here. Decoupling via http.Handler keeps the api package free of a
	// hard dependency on the metrics package.
	MetricsHandler http.Handler
	Log            *slog.Logger
	// Version is reported by GET /api/v1/version.
	Version string
	// AllowMutations is the legacy mutation gate. Deprecated in
	// favour of Auth — set Auth.Mode = AuthModeDisabled to get the
	// same wide-open semantics, or AuthModeAuto / AuthModeRequired
	// for the bearer-token middleware. When Auth.Mode is the zero
	// value (AuthModeAuto) and AllowMutations is true, the daemon
	// emits a deprecation warning and treats the daemon as
	// AuthModeDisabled to preserve the existing behaviour.
	AllowMutations bool
	// Auth configures the mutation auth middleware. See AuthMode
	// for the policy semantics. Zero-value is AuthModeAuto, which
	// requires a token on non-loopback binds and bypasses the
	// check on loopback (peer-cred trust on a single-host
	// deployment).
	Auth AuthConfig
	// Mutator is the engine's write side (end call). Optional;
	// when nil the corresponding routes return 503.
	Mutator EngineMutator
	// Retention is the storage sweeper's write side (run a sweep
	// now). Optional.
	Retention RetentionSweeper
	// Tones is the tone-out detector's write side (reset per-device
	// match state). Optional.
	Tones ToneDetectorReset
	// Devices exposes the SDR pool snapshot for GET /api/v1/devices.
	// Optional; the route returns 503 when nil.
	Devices DevicesProvider
	// Scanner exposes the police-scanner cockpit (CC hunter,
	// conventional FM scanner, TG scan list) for GET + PATCH
	// /api/v1/scanner and the related mutation routes. Optional;
	// when nil, the routes return 503.
	Scanner ScannerCockpit
	// Hunt exposes the live system-discovery cockpit for GET /api/v1/hunt and
	// its start/stop/export/commit routes. Optional; when nil, the routes
	// return 503 (or an empty idle status for the read).
	Hunt HuntCockpit
	// Audio exposes the live-audio player + recorder gate for
	// GET + PATCH /api/v1/audio. Optional; when nil, the routes
	// return 503.
	Audio AudioController
	// Broadcast exposes the outbound call-streaming subsystem's
	// counters for GET /api/v1/broadcast. Optional; when nil, the
	// route reports the subsystem as disabled.
	Broadcast BroadcastStatusProvider
	// Runtime exposes the read-only daemon config snapshot served at
	// GET /api/v1/runtime. The TUI's tabbed Settings inspector uses
	// it to surface every config knob. Optional; when nil, the
	// route returns 503.
	Runtime RuntimeProvider
	// ConfigWriter, when supplied, enables PATCH /api/v1/settings:
	// the daemon writes the operator's edits to config.yaml
	// (preserving comments) and feeds the result back to
	// SettingsApplier for hot-reload. nil disables the endpoint
	// (returns 503) so daemons started without a -config file don't
	// pretend the SPA's edits will persist.
	ConfigWriter ConfigWriter
	// ConfigActivator, when supplied, enables POST /api/v1/config/activate:
	// the daemon loads/hot-swaps the config file the operator picks in the
	// web Config Builder, either reloading in place (hot-applying what it
	// can) or re-execing into the new file. nil disables the endpoint
	// (returns 503) — e.g. the standalone `config serve`, which has no
	// daemon to reconfigure.
	ConfigActivator ConfigActivator
	// SettingsApplier is the in-process hot-reload surface invoked
	// by handleSettingsPatch after the on-disk write succeeds.
	// Optional: when nil, every field is reported as
	// "restart_required" in the response.
	SettingsApplier SettingsApplier
	// Importer enables the live system-import endpoints
	// (POST /api/v1/import, POST /api/v1/import/{id}/commit,
	// DELETE /api/v1/import/{id}). nil disables the endpoints —
	// the daemon emits 503 so the SPA can present a clear "import
	// disabled" message.
	Importer Importer
	// WebAssets, when non-nil and containing an `index.html`, is
	// served from `/` (and as the SPA fallback for any non-/api
	// path). Set this to the embedded web/dist filesystem so the
	// daemon hosts the operator console without a sibling
	// gophertrunk-web/ directory. Leave nil to keep the SPA
	// out-of-process.
	WebAssets fs.FS
	// AudioPublisher, when non-nil, enables the
	// GET /api/v1/audio/stream HTTP endpoint that streams live
	// composed PCM as a continuous WAV body. Reuses the same
	// publisher that backs gRPC StreamAudio so the HTTP stream is
	// a parallel subscriber rather than a second fan-out.
	AudioPublisher *AudioPublisher
	// Spectrum, when non-nil, enables the
	// GET /api/v1/spectrum/devices read endpoint and the
	// WS /api/v1/spectrum/stream live FFT frame stream. The daemon
	// implements this on top of its iqtap.Broker map; nil keeps the
	// routes returning 503 so a build without SDRs doesn't pretend
	// to have a waterfall.
	Spectrum SpectrumProvider
	// Capture, when non-nil, enables the runtime
	// POST /api/v1/siglab/capture route and its GET
	// /api/v1/siglab/capture/devices picker — record a fixed-length
	// raw-IQ capture off a live tuner, stage it into siglab, and hand
	// back a .cfile download URL. The daemon implements this over its
	// iqtap.Broker map; nil keeps the routes returning 503.
	Capture CaptureProvider
	// Bookmarks, when non-nil, enables the
	// GET/POST/PATCH/DELETE /api/v1/bookmarks routes for operator-
	// managed conventional channel bookmarks. nil keeps the routes
	// returning 503. Wired by the daemon over the SQLite-backed
	// storage.BookmarkStore.
	Bookmarks BookmarkProvider
	// Diag, when non-nil, enables the
	// WS /api/v1/diag/iq decimated-IQ live stream that backs the
	// web Constellation panel. The daemon implements this over
	// the iqtap broker + internal/dsp/diag; nil keeps the route
	// returning 503.
	Diag DiagProvider
	// Symbols, when non-nil, enables the WS /api/v1/diag/symbols
	// recovered-symbol live stream that backs the web Symbol scope
	// panel. The daemon implements this over the iqtap broker +
	// internal/scanner/symbolscope; nil keeps the route returning 503.
	Symbols SymbolProvider
	// Mixer, when non-nil, enables the WS /api/v1/diag/mixer
	// channel-baseband spectrum stream that backs the web Mixer plot.
	// The daemon implements this over the iqtap broker +
	// internal/scanner/symbolscope; nil keeps the route returning 503.
	Mixer MixerProvider
	// Siglab, when Enabled, mounts the offline signal-analysis routes
	// (/api/v1/siglab/*) and constructs the in-memory job/capture store.
	// The daemon and the standalone `siglab serve` command both set it.
	Siglab SiglabOptions
	// ConfigBuilder, when Enabled, mounts the web Config Builder/Editor
	// routes (/api/v1/config/*) and — when Assets is non-nil — the builder
	// SPA at /config/. The daemon sets it so the operator can edit config
	// from the main UI in a new tab; the standalone `gophertrunk config
	// serve` command sets it (with the SPA in WebAssets, served at /).
	ConfigBuilder ConfigBuilderOptions
	// RFScope, when Enabled, mounts the RF Scope analysis routes
	// (/api/v1/rfscope/*) and — when Assets is non-nil — the RF Scope SPA at
	// /rfscope/. The daemon sets it so operators can reach the offline
	// RF-analysis console from the main UI; the standalone `gophertrunk rfscope
	// serve` command keeps its own webserver instead.
	RFScope RFScopeOptions
	// Diagnostics is the shared error-diagnostics collector (banner +
	// system info). The daemon injects one seeded with its SDR pool
	// snapshot so the error path never re-enumerates USB. Optional; nil
	// disables banner enrichment and the GET /api/v1/diag/banner route.
	Diagnostics *gtdiag.Collector
	// VerboseErrors mirrors diagnostics.verbose_errors. When true, every
	// JSON error envelope includes the diagnostics banner (exposing host
	// + dongle info), and GET /api/v1/diag/banner is served to any
	// caller; when false the banner is only attached on ?verbose=1 from
	// a trusted (authorized) request.
	VerboseErrors bool
	// DisplayLoc is the timezone the GET /api/v1/health daemon-time renders in.
	// Nil means time.Local (the daemon passes display.timezone).
	DisplayLoc *time.Location
	// Pager, when non-nil, enables the
	// GET /api/v1/pager/messages route serving recent decoded
	// POCSAG (and eventually FLEX) pager messages. Wired by the
	// daemon over the SQLite-backed storage.PagerLog.
	Pager PagerProvider
	// APRS, when non-nil, enables the
	// GET /api/v1/aprs/packets route serving recent decoded
	// APRS / AX.25 packets. Wired by the daemon over the SQLite-
	// backed storage.APRSLog.
	APRS APRSProvider
	// AIS, when non-nil, enables the
	// GET /api/v1/ais/vessels route serving recent decoded
	// AIS messages. Wired by the daemon over the SQLite-backed
	// storage.VesselLog.
	AIS AISProvider
	// DSC, when non-nil, enables the
	// GET /api/v1/dsc/messages route serving recent decoded
	// marine DSC sequences. Wired by the daemon over the
	// SQLite-backed storage.DSCLog.
	DSC DSCProvider
	// M17, when non-nil, enables the
	// GET /api/v1/m17/linksetups route serving recent decoded M17
	// link-setup frames. Wired by the daemon over the SQLite-backed
	// storage.M17Log.
	M17 M17Provider
	// LoRa, when non-nil, enables the
	// GET /api/v1/lora/frames route serving recent decoded LoRa
	// frames. Wired by the daemon over the SQLite-backed
	// storage.LoRaLog.
	LoRa LoRaProvider
	// ADSB, when non-nil, enables the
	// GET /api/v1/adsb/aircraft route serving recent decoded
	// Mode-S frames. Wired by the daemon over the SQLite-backed
	// storage.AircraftLog.
	ADSB ADSBProvider
	// MDC1200, when non-nil, enables the
	// GET /api/v1/mdc1200/messages route serving recent decoded
	// MDC1200 signaling bursts. Wired by the daemon over the
	// SQLite-backed storage.MDC1200Log.
	MDC1200 MDC1200Provider
	// CORS configures the cross-origin middleware. Off when
	// AllowedOrigins is empty (the daemon emits no CORS headers).
	// Set this when the browser-served SPA is loaded from an
	// origin different to the daemon's (most commonly file://,
	// whose Origin header is the literal string "null").
	CORS CORSConfig
	// TLSCert and TLSKey, when both non-empty, switch the HTTP
	// server to TLS. Paths point at PEM-encoded files on disk that
	// the daemon reads at start-up. Leaving either empty serves
	// plain HTTP (the default — appropriate for loopback / private-
	// network deployments where the bearer-token auth gate is the
	// only protection on mutations).
	TLSCert string
	TLSKey  string
}

// NewServer constructs a server but does not yet bind a listener; call
// Run.
func NewServer(opts ServerOptions) (*Server, error) {
	if opts.Addr == "" {
		return nil, errors.New("api: Addr is required")
	}
	if opts.Bus == nil {
		return nil, errors.New("api: events.Bus is required")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	if opts.Talkgroups == nil {
		opts.Talkgroups = trunking.NewTalkgroupDB()
	}
	if opts.RIDs == nil {
		opts.RIDs = trunking.NewRIDDB()
	}
	authCfg := opts.Auth
	// Legacy migration: AllowMutations: true with no explicit Auth
	// config maps to AuthModeDisabled so the existing wide-open
	// behaviour is preserved. The daemon logs a deprecation
	// warning so operators know to migrate to the explicit auth
	// config.
	if opts.AllowMutations && authCfg.Mode == AuthModeAuto && authCfg.Token == "" && authCfg.TokenFile == "" && len(authCfg.TrustedNetworks) == 0 {
		log.Warn("api: AllowMutations is deprecated; migrate to api.auth (mapping to auth.mode=disabled for backwards compatibility)")
		authCfg.Mode = AuthModeDisabled
	}
	auth, err := newAuthState(authCfg, opts.Addr)
	if err != nil {
		return nil, err
	}
	if authCfg.Mode == AuthModeDisabled {
		log.Warn("api: auth disabled — mutation endpoints are not authenticated; bind to loopback or trusted network only")
	}
	// CORS permissive default warning: only surfaces on non-loopback
	// binds so the common file:// + loopback workflow stays quiet.
	if opts.CORS.IsDefaultPermissive() && !bindsToLoopback(opts.Addr) {
		log.Warn("api: CORS open to any origin (default) on a non-loopback bind — set api.cors.allowed_origins to clamp it down on hostile networks")
	}
	// TLS: both files must be set to enable TLS; one without the
	// other is a misconfiguration the operator should hear about
	// rather than silently fall back to plain HTTP.
	if (opts.TLSCert == "") != (opts.TLSKey == "") {
		return nil, errors.New("api: tls_cert and tls_key must both be set or both be empty")
	}
	var siglabSvc *siglabService
	var siglabStop chan struct{}
	if opts.Siglab.Enabled {
		svc, serr := newSiglabService(opts.Siglab)
		if serr != nil {
			return nil, serr
		}
		siglabSvc = svc
		siglabStop = make(chan struct{})
		go svc.runSweeper(siglabStop)
	}
	var configBuilderSvc *configBuilderService
	if opts.ConfigBuilder.Enabled {
		svc, cberr := newConfigBuilderService(opts.ConfigBuilder)
		if cberr != nil {
			return nil, cberr
		}
		configBuilderSvc = svc
	}
	return &Server{
		addr:           opts.Addr,
		bus:            opts.Bus,
		engine:         opts.Engine,
		mutator:        opts.Mutator,
		retention:      opts.Retention,
		tones:          opts.Tones,
		devices:        opts.Devices,
		scanner:        opts.Scanner,
		hunt:           opts.Hunt,
		audio:          opts.Audio,
		broadcast:      opts.Broadcast,
		runtime:        opts.Runtime,
		configWriter:   opts.ConfigWriter,
		configActiv:    opts.ConfigActivator,
		settings:       opts.SettingsApplier,
		importer:       opts.Importer,
		imports:        newImportStaging(5 * time.Minute),
		webAssets:      opts.WebAssets,
		talkgroups:     opts.Talkgroups,
		rids:           opts.RIDs,
		systems:        append([]trunking.System(nil), opts.Systems...),
		history:        opts.History,
		locations:      opts.Locations,
		affiliations:   opts.Affiliations,
		sites:          opts.Sites,
		metrics:        opts.MetricsHandler,
		log:            log,
		version:        opts.Version,
		auth:           auth,
		allowMutations: opts.AllowMutations,
		tlsCert:        opts.TLSCert,
		tlsKey:         opts.TLSKey,
		cors:           opts.CORS,
		audioPub:       opts.AudioPublisher,
		spectrum:       opts.Spectrum,
		capture:        opts.Capture,
		bookmarks:      opts.Bookmarks,
		diag:           opts.Diag,
		symbols:        opts.Symbols,
		mixer:          opts.Mixer,
		siglab:         siglabSvc,
		siglabStop:     siglabStop,
		siglabAssets:   opts.Siglab.Assets,
		rfscopeOpts:    opts.RFScope,
		configBuilder:  configBuilderSvc,
		diagnostics:    opts.Diagnostics,
		verboseErrors:  opts.VerboseErrors,
		displayLoc:     orLocal(opts.DisplayLoc),
		pager:          opts.Pager,
		aprs:           opts.APRS,
		ais:            opts.AIS,
		dsc:            opts.DSC,
		m17:            opts.M17,
		lora:           opts.LoRa,
		adsb:           opts.ADSB,
		mdc1200:        opts.MDC1200,
	}, nil
}

// Run binds the listener and serves until ctx cancels.
func (s *Server) Run(ctx context.Context) error {
	mux := s.routes()
	var handler http.Handler = mux
	if s.cors.enabled() {
		handler = corsMiddleware(s.cors, handler)
	}
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.boundAddr = listener.Addr().String()
	s.srv = &http.Server{
		Handler: handler,
		// ReadHeaderTimeout protects against Slowloris attacks; the
		// existing 10 s bound stays.
		ReadHeaderTimeout: 10 * time.Second,
		// ReadTimeout / WriteTimeout / IdleTimeout cap per-request
		// resource use so slow clients can't pin a worker or a
		// socket. Streaming endpoints (SSE at /api/v1/events,
		// WebSocket at /api/v1/events/ws, the per-call audio stream
		// in api/audio.go) disable WriteTimeout per-request via
		// http.ResponseController so the long-lived connections keep
		// working — the standard REST handlers are bounded by these
		// at the server level.
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	s.mu.Unlock()

	errCh := make(chan error, 1)
	tlsEnabled := s.tlsCert != "" && s.tlsKey != ""
	go func() {
		s.log.Info("api: listening",
			"addr", listener.Addr().String(),
			"tls", tlsEnabled)
		var err error
		if tlsEnabled {
			// ServeTLS reads the cert / key off disk at start;
			// rotation requires a daemon restart. Document this
			// in docs/hardening.md.
			err = s.srv.ServeTLS(listener, s.tlsCert, s.tlsKey)
		} else {
			err = s.srv.Serve(listener)
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		return s.shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

// Close gracefully shuts down the server. Safe to call after Run returns.
func (s *Server) Close() error {
	return s.shutdown(context.Background())
}

// BoundAddr returns the actual TCP address the listener bound to,
// useful when callers configured ":0" / "127.0.0.1:0" and need the
// kernel-assigned port. Returns "" before Run() has bound.
func (s *Server) BoundAddr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.boundAddr
}

func (s *Server) shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.srv == nil {
		s.closed = true
		return nil
	}
	s.closed = true
	// Tear down the siglab subsystem: stop the TTL sweeper and remove the
	// temp directory (and any staged captures) the service owns.
	if s.siglab != nil {
		if s.siglabStop != nil {
			close(s.siglabStop)
			s.siglabStop = nil
		}
		s.siglab.close()
	}
	// Tear down the optional cryptolab subsystem (removes its temp dir).
	if s.cryptolabCloser != nil {
		_ = s.cryptolabCloser.Close()
		s.cryptolabCloser = nil
	}
	// Tear down the optional RF Scope subsystem (removes its temp dir).
	if s.rfscopeCloser != nil {
		_ = s.rfscopeCloser.Close()
		s.rfscopeCloser = nil
	}
	// 30 s shutdown window: SSE / WebSocket / audio-stream subscribers
	// get up to 30 s to drain rather than the 5 s the old bound gave
	// them. Cuts user-visible connection drops on a clean restart.
	// Static HTTP requests complete in milliseconds either way; the
	// extra headroom only matters for long-lived streams.
	shutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return s.srv.Shutdown(shutCtx)
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/runtime", s.handleRuntime)
	mux.HandleFunc("GET /api/v1/version", s.handleVersion)
	mux.HandleFunc("GET /api/v1/diag/banner", s.handleDiagBanner)
	mux.HandleFunc("GET /api/v1/systems", s.handleListSystems)
	mux.HandleFunc("GET /api/v1/systems/{name}", s.handleGetSystem)
	mux.HandleFunc("GET /api/v1/systems/{name}/report", s.handleGetSystemReport)
	mux.HandleFunc("GET /api/v1/talkgroups", s.handleListTalkgroups)
	mux.HandleFunc("GET /api/v1/talkgroups/{id}", s.handleGetTalkgroup)
	mux.HandleFunc("GET /api/v1/calls/active", s.handleActiveCalls)
	mux.HandleFunc("GET /api/v1/calls/history", s.handleCallHistory)
	mux.HandleFunc("GET /api/v1/locations", s.handleLocations)
	mux.HandleFunc("GET /api/v1/affiliations", s.handleAffiliations)
	mux.HandleFunc("GET /api/v1/sites", s.handleListSites)
	mux.HandleFunc("GET /api/v1/rids", s.handleListRIDs)
	mux.HandleFunc("GET /api/v1/rids/{id}", s.handleGetRID)
	mux.HandleFunc("GET /api/v1/rids/{id}/history", s.handleRIDHistory)
	mux.HandleFunc("GET /api/v1/devices", s.handleListDevices)
	mux.HandleFunc("GET /api/v1/events", s.handleSSE)
	mux.HandleFunc("GET /api/v1/events/ws", s.handleWS)

	// GopherTrunk Bundle endpoints: inspect, verify, and download a .gtb case
	// from the web consoles. Part of the standard build (read-side only; packing
	// runs through the CLI). Confined to bundleRoot when set.
	mux.HandleFunc("GET /api/v1/bundle/info", s.handleBundleInfo)
	mux.HandleFunc("POST /api/v1/bundle/verify", s.handleBundleVerify)
	mux.HandleFunc("GET /api/v1/bundle/download", s.handleBundleDownload)
	if s.metrics != nil {
		mux.Handle("GET /metrics", s.metrics)
	}

	// Opt-in live profiling for diagnosing leaks / hangs / footprint growth
	// (issue #492). Off by default — it exposes profiling and host detail —
	// and gated behind GOPHERTRUNK_PPROF so an operator can scrape
	// /debug/pprof/{heap,goroutine,allocs,profile} during a repro without a
	// rebuild. Bind the API to loopback when enabling.
	if os.Getenv("GOPHERTRUNK_PPROF") != "" {
		mux.HandleFunc("GET /debug/pprof/", httppprof.Index)
		mux.HandleFunc("GET /debug/pprof/cmdline", httppprof.Cmdline)
		mux.HandleFunc("GET /debug/pprof/profile", httppprof.Profile)
		mux.HandleFunc("GET /debug/pprof/symbol", httppprof.Symbol)
		mux.HandleFunc("GET /debug/pprof/trace", httppprof.Trace)
		s.log.Warn("api: pprof debug endpoints enabled (GOPHERTRUNK_PPROF) — exposes profiling/host detail; bind to loopback only")
	}

	// Mutation routes — wrapped in s.gate so a non-AllowMutations
	// daemon returns 403 without dispatching to the handler. The
	// gate also reports the daemon's mutation capability via
	// GET /api/v1/mutations so clients can light up keybindings.
	mux.HandleFunc("GET /api/v1/mutations", s.handleMutationStatus)
	mux.HandleFunc("POST /api/v1/calls/{deviceSerial}/end", s.gate(s.handleEndCall))
	mux.HandleFunc("PATCH /api/v1/talkgroups/{id}", s.gate(s.handleUpdateTalkgroup))
	mux.HandleFunc("PATCH /api/v1/rids/{id}", s.gate(s.handleUpdateRID))
	mux.HandleFunc("POST /api/v1/retention/sweep", s.gate(s.handleRetentionSweep))
	mux.HandleFunc("POST /api/v1/devices/{serial}/tone-reset", s.gate(s.handleToneReset))

	// Scanner cockpit — read endpoint is always open; mutations are
	// gated behind allow_mutations like every other write route.
	mux.HandleFunc("GET /api/v1/broadcast", s.handleBroadcastStatus)
	// Live system-discovery (hunt) cockpit — read is open; mutations gated.
	mux.HandleFunc("GET /api/v1/hunt", s.handleHuntStatus)
	mux.HandleFunc("POST /api/v1/hunt/start", s.gate(s.handleHuntStart))
	mux.HandleFunc("POST /api/v1/hunt/stop", s.gate(s.handleHuntStop))
	mux.HandleFunc("GET /api/v1/hunt/export", s.handleHuntExport)
	mux.HandleFunc("GET /api/v1/hunt/{id}/export", s.handleHuntExport)
	mux.HandleFunc("GET /api/v1/hunt/survey", s.handleHuntSurveyExport)
	mux.HandleFunc("GET /api/v1/hunt/{id}/survey", s.handleHuntSurveyExport)
	mux.HandleFunc("POST /api/v1/hunt/capture", s.gate(s.handleHuntCapture))
	mux.HandleFunc("POST /api/v1/hunt/commit", s.gate(s.handleHuntCommit))
	mux.HandleFunc("GET /api/v1/hunt/radioreference", s.handleHuntRadioReference)
	mux.HandleFunc("GET /api/v1/hunt/{id}/radioreference", s.handleHuntRadioReference)
	mux.HandleFunc("POST /api/v1/hunt/{id}/commit", s.gate(s.handleHuntCommit))

	mux.HandleFunc("GET /api/v1/scanner", s.handleScannerStatus)
	mux.HandleFunc("PATCH /api/v1/scanner", s.gate(s.handleScannerSetMode))
	mux.HandleFunc("POST /api/v1/scanner/hunt/{system}/hold", s.gate(s.handleHuntHold))
	mux.HandleFunc("POST /api/v1/scanner/hunt/{system}/resume", s.gate(s.handleHuntResume))
	mux.HandleFunc("POST /api/v1/scanner/hunt/{system}/retune", s.gate(s.handleHuntRetune))
	mux.HandleFunc("POST /api/v1/scanner/conventional/hold", s.gate(s.handleConvHold))
	mux.HandleFunc("POST /api/v1/scanner/conventional/resume", s.gate(s.handleConvResume))
	mux.HandleFunc("POST /api/v1/scanner/conventional/{index}/dwell", s.gate(s.handleConvDwell))
	mux.HandleFunc("POST /api/v1/scanner/conventional/{index}/lockout", s.gate(s.handleConvLockout))
	mux.HandleFunc("POST /api/v1/scanner/conventional/{index}/unlockout", s.gate(s.handleConvUnlockout))
	mux.HandleFunc("POST /api/v1/scanner/manual_tune", s.gate(s.handleScannerManualTune))
	mux.HandleFunc("DELETE /api/v1/scanner/manual_tune/{index}", s.gate(s.handleScannerClearManualTune))

	// Audio cockpit — read endpoint is always open; the PATCH is
	// gated behind allow_mutations like every other write route.
	mux.HandleFunc("GET /api/v1/audio", s.handleAudioStatus)
	mux.HandleFunc("PATCH /api/v1/audio", s.gate(s.handleAudioPatch))
	// Settings cockpit — PATCH writes the supplied fields back to
	// config.yaml (preserving comments + formatting) and hot-applies
	// the ones the daemon knows how to reload in-process. The
	// response carries the new runtime DTO + a per-field list of
	// what applied vs what needs a daemon restart.
	mux.HandleFunc("PATCH /api/v1/settings", s.gate(s.handleSettingsPatch))

	// Live import — upload one or more RadioReference PDFs / CSV
	// bundles, preview the parsed systems, then commit (or discard)
	// the staged batch. Multipart upload at POST /api/v1/import;
	// commit/discard keyed by the returned staging ID.
	mux.HandleFunc("POST /api/v1/import", s.gate(s.handleImportUpload))
	mux.HandleFunc("POST /api/v1/import/{id}/commit", s.gate(s.handleImportCommit))
	mux.HandleFunc("DELETE /api/v1/import/{id}", s.gate(s.handleImportDiscard))
	// Live audio stream — open like every other read route. Emits
	// a continuous WAV body of composed PCM frames; browsers play
	// it via <audio src="/api/v1/audio/stream">. Returns 503 when
	// the daemon was started without an audio publisher.
	mux.HandleFunc("GET /api/v1/audio/stream", s.handleAudioStream)

	// Spectrum / waterfall — list devices that can be streamed and
	// open a WS feed of FFT magnitude frames. Foundation for the
	// browser + TUI live spectrum panel. Returns 503 when the
	// daemon was started without a SpectrumProvider (no SDRs, or
	// the broker map wasn't wired).
	mux.HandleFunc("GET /api/v1/spectrum/devices", s.handleSpectrumDevices)
	mux.HandleFunc("GET /api/v1/spectrum/stream", s.handleSpectrumStream)
	mux.HandleFunc("POST /api/v1/spectrum/devices/{serial}/tune", s.gate(s.handleSpectrumTune))

	// Bookmarks / frequency manager. Read endpoint is always open;
	// create / update / delete are gated behind allow_mutations
	// like every other write route.
	mux.HandleFunc("GET /api/v1/bookmarks", s.handleListBookmarks)
	mux.HandleFunc("POST /api/v1/bookmarks", s.gate(s.handleCreateBookmark))
	mux.HandleFunc("PATCH /api/v1/bookmarks/{id}", s.gate(s.handleUpdateBookmark))
	mux.HandleFunc("DELETE /api/v1/bookmarks/{id}", s.gate(s.handleDeleteBookmark))

	// Diagnostic IQ stream — feeds the web Constellation panel.
	// Read-only; the daemon doesn't expose any way to inject IQ
	// via this path. Returns 503 when no SDR is in the pool.
	mux.HandleFunc("GET /api/v1/diag/iq", s.handleDiagStream)
	mux.HandleFunc("GET /api/v1/diag/symbols", s.handleSymbolStream)
	mux.HandleFunc("GET /api/v1/diag/mixer", s.handleMixerStream)

	// Siglab — offline signal-analysis console. Read routes (protocols,
	// job result/stream/iq, export) are open; routes that stage data or
	// spend CPU (upload, run, identify, synthesize, delete) are gated like
	// every other write route. Mounted only when the subsystem is enabled
	// (the daemon and the standalone `siglab serve` command).
	if s.siglab != nil {
		mux.HandleFunc("GET /api/v1/siglab/protocols", s.handleSiglabProtocols)
		mux.HandleFunc("POST /api/v1/siglab/captures", s.gate(s.handleSiglabUpload))
		mux.HandleFunc("DELETE /api/v1/siglab/captures/{id}", s.gate(s.handleSiglabDeleteCapture))
		mux.HandleFunc("POST /api/v1/siglab/run", s.gate(s.handleSiglabRun))
		mux.HandleFunc("GET /api/v1/siglab/jobs/{id}", s.handleSiglabJobResult)
		mux.HandleFunc("GET /api/v1/siglab/jobs/{id}/events", s.handleSiglabJobStream)
		mux.HandleFunc("GET /api/v1/siglab/jobs/{id}/iq", s.handleSiglabJobIQ)
		mux.HandleFunc("GET /api/v1/siglab/jobs/{id}/psd", s.handleSiglabJobPSD)
		mux.HandleFunc("GET /api/v1/siglab/jobs/{id}/spectrogram", s.handleSiglabJobSpectrogram)
		mux.HandleFunc("GET /api/v1/siglab/jobs/{id}/occupancy", s.handleSiglabJobOccupancy)
		mux.HandleFunc("GET /api/v1/siglab/jobs/{id}/export", s.handleSiglabExport)
		mux.HandleFunc("POST /api/v1/siglab/identify", s.gate(s.handleSiglabIdentify))
		mux.HandleFunc("POST /api/v1/siglab/wideband", s.gate(s.handleSiglabWideband))
		mux.HandleFunc("POST /api/v1/siglab/synthesize", s.gate(s.handleSiglabSynthesize))
		// Live raw-IQ capture off a tuner: list devices (read), record a
		// fixed-length capture (gated mutation — it spends an SDR + CPU),
		// and download a staged capture's raw bytes. 503 when no
		// CaptureProvider is wired (offline `siglab serve`, or no SDRs).
		mux.HandleFunc("GET /api/v1/siglab/capture/devices", s.handleSiglabCaptureDevices)
		mux.HandleFunc("POST /api/v1/siglab/capture", s.gate(s.handleSiglabCapture))
		mux.HandleFunc("GET /api/v1/siglab/captures/{id}/download", s.handleSiglabCaptureDownload)

		// Secondary SPA at /siglab/ (daemon path). The standalone `siglab
		// serve` puts the SPA in WebAssets (served at /) instead, so this only
		// fires on the daemon. The /siglab/ matcher is more specific than the
		// GET / catch-all, so both trees coexist. Advertised via
		// RuntimeDTO.SiglabConsole so the other consoles only link here when the
		// SPA is actually reachable.
		if s.siglabAssets != nil {
			if _, err := fs.Stat(s.siglabAssets, "index.html"); err == nil {
				mux.Handle("GET /siglab/", s.siglabSpaHandler())
				mux.HandleFunc("GET /siglab", func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, "/siglab/", http.StatusMovedPermanently)
				})
				s.siglabConsole = true
			}
		}
	}

	// Optional RF Scope console (/rfscope/ SPA + /api/v1/rfscope/* routes).
	// Mounted only when ServerOptions.RFScope.Enabled; a no-op otherwise.
	s.mountRFScope(mux)

	// Config Builder/Editor — browse/load/validate/save config files,
	// browse RadioReference, and parse PDF/CSV into a draft. Reads are
	// open (like siglab); save is gated behind allow_mutations. Mounted
	// only when the subsystem is enabled (the daemon and the standalone
	// `gophertrunk config serve` command).
	if s.configBuilder != nil {
		mux.HandleFunc("GET /api/v1/config/files", s.handleConfigList)
		mux.HandleFunc("GET /api/v1/config/file", s.handleConfigLoad)
		mux.HandleFunc("GET /api/v1/config/file/talkgroups", s.handleConfigTalkgroups)
		mux.HandleFunc("GET /api/v1/config/defaults", s.handleConfigDefaults)
		mux.HandleFunc("GET /api/v1/config/docs", s.handleConfigDocs)
		mux.HandleFunc("GET /api/v1/config/fieldmeta", s.handleConfigFieldMeta)
		mux.HandleFunc("POST /api/v1/config/validate", s.handleConfigValidate)
		mux.HandleFunc("POST /api/v1/config/marshal", s.handleConfigMarshal)
		mux.HandleFunc("POST /api/v1/config/file", s.gate(s.handleConfigSave))
		mux.HandleFunc("DELETE /api/v1/config/file", s.gate(s.handleConfigDelete))
		mux.HandleFunc("POST /api/v1/config/file/rename", s.gate(s.handleConfigRename))
		mux.HandleFunc("POST /api/v1/config/dir", s.gate(s.handleConfigMkdir))
		mux.HandleFunc("POST /api/v1/config/activate", s.gate(s.handleConfigActivate))
		mux.HandleFunc("POST /api/v1/config/parse", s.gate(s.handleConfigParse))
		mux.HandleFunc("GET /api/v1/config/rr/search", s.handleConfigRRSearch)
		mux.HandleFunc("GET /api/v1/config/rr/states", s.handleConfigRRStates)
		mux.HandleFunc("GET /api/v1/config/rr/counties", s.handleConfigRRCounties)
		mux.HandleFunc("GET /api/v1/config/rr/system/{sid}", s.handleConfigRRSystem)
		mux.HandleFunc("POST /api/v1/config/rr/verify", s.handleConfigRRVerify)

		// Secondary SPA at /config/ (daemon path). The standalone
		// `config serve` puts the SPA in WebAssets (served at /) instead,
		// so this only fires on the daemon. The /config/ matcher is more
		// specific than the GET / catch-all, so both trees coexist.
		if s.configBuilder.assets != nil {
			if _, err := fs.Stat(s.configBuilder.assets, "index.html"); err == nil {
				mux.Handle("GET /config/", s.configSpaHandler())
				mux.HandleFunc("GET /config", func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, "/config/", http.StatusMovedPermanently)
				})
			}
		}
	}

	// Optional cryptolab console (/cryptolab/ SPA + /api/v1/cryptolab/*).
	// A no-op in the default build; the cryptolab-tagged build registers the
	// routes here (see cryptolab_enabled.go).
	s.mountCryptolab(mux)

	// Pager log — recent POCSAG (and eventually FLEX) messages.
	// Read-only; the decoder writes via the events bus → PagerLog.
	mux.HandleFunc("GET /api/v1/pager/messages", s.handlePagerMessages)

	// APRS / AX.25 packet log — recent decoded packets. Read-only;
	// the decoder writes via the events bus → APRSLog.
	mux.HandleFunc("GET /api/v1/aprs/packets", s.handleAPRSPackets)
	mux.HandleFunc("GET /api/v1/ais/vessels", s.handleAISMessages)
	mux.HandleFunc("GET /api/v1/dsc/messages", s.handleDSCMessages)
	mux.HandleFunc("GET /api/v1/m17/linksetups", s.handleM17LinkSetups)
	mux.HandleFunc("GET /api/v1/lora/frames", s.handleLoRaFrames)
	mux.HandleFunc("GET /api/v1/adsb/aircraft", s.handleADSBAircraft)
	mux.HandleFunc("GET /api/v1/adsb/aircraft/current", s.handleADSBAircraftCurrent)
	mux.HandleFunc("GET /api/v1/mdc1200/messages", s.handleMDC1200Messages)

	// Embedded SPA at "/" — served only when the daemon was linked
	// against a populated web/dist embed. SPA history routes
	// (/scanner, /settings, /import, ...) fall back to index.html
	// so React-Router takes over on the client side. When the embed
	// is empty (the binary was built without `make web-build` first),
	// a fallback handler answers exactly `GET /` with an explanatory
	// 404 instead of stdlib's blank "404 page not found".
	embeddedSPA := false
	if s.webAssets != nil {
		if _, err := fs.Stat(s.webAssets, "index.html"); err == nil {
			mux.Handle("GET /", s.spaHandler())
			embeddedSPA = true
		}
	}
	if !embeddedSPA {
		mux.Handle("GET /{$}", s.spaMissingHandler())
	}

	return mux
}

// spaHandler serves the embedded SPA, falling back to index.html
// for client-side routes so React-Router can pick them up.
func (s *Server) spaHandler() http.Handler {
	fileSrv := http.FileServerFS(s.webAssets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API routes never reach this handler — the mux's
		// more-specific matchers own /api/v1/* and /metrics.
		// Defence in depth: refuse those paths so a hypothetical
		// embed override surfaces loudly in tests.
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/metrics" {
			http.NotFound(w, r)
			return
		}
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			fileSrv.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(s.webAssets, clean); err == nil {
			fileSrv.ServeHTTP(w, r)
			return
		}
		// Fallback to index.html so the SPA's router resolves
		// /scanner, /settings, /import, ... on the client.
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileSrv.ServeHTTP(w, r2)
	})
}

// configSpaHandler serves the Config Builder SPA mounted under /config/
// on the daemon (the standalone `config serve` serves it at / via
// spaHandler). It strips the /config/ prefix before hitting the embedded
// file tree and falls back to the builder's index.html for client-side
// routes, mirroring spaHandler but rooted one level down.
func (s *Server) configSpaHandler() http.Handler {
	assets := s.configBuilder.assets
	fileSrv := http.FileServerFS(assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/config/"), "/")
		if rel == "" {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileSrv.ServeHTTP(w, r2)
			return
		}
		if _, err := fs.Stat(assets, rel); err == nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/" + rel
			fileSrv.ServeHTTP(w, r2)
			return
		}
		// Unknown sub-path → serve index.html so the builder's client
		// router can resolve it.
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileSrv.ServeHTTP(w, r2)
	})
}

// siglabSpaHandler serves the Signal Lab SPA mounted under /siglab/ on the
// daemon (the standalone `siglab serve` serves it at / via spaHandler). It
// strips the /siglab/ prefix before hitting the embedded file tree and falls
// back to the SPA's index.html for client-side routes, mirroring
// configSpaHandler but rooted at /siglab/.
func (s *Server) siglabSpaHandler() http.Handler {
	assets := s.siglabAssets
	fileSrv := http.FileServerFS(assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/siglab/"), "/")
		if rel != "" {
			if _, err := fs.Stat(assets, rel); err == nil {
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/" + rel
				fileSrv.ServeHTTP(w, r2)
				return
			}
		}
		// Root or unknown sub-path → serve index.html so the SPA's client
		// router can resolve it.
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileSrv.ServeHTTP(w, r2)
	})
}

// spaMissingHandler answers exactly `GET /` with an HTML 404 that
// tells the operator how to bundle the SPA. Registered when the
// daemon binary was built without `make web-build` first, so the
// `//go:embed all:dist` snapshot contains only the `.gitkeep`
// sentinel. Status stays 404 so proxies/curl/healthchecks still
// treat the resource as missing — only the body changes.
func (s *Server) spaMissingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(spaMissingBody))
	})
}

const spaMissingBody = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>GopherTrunk — web console not bundled</title>
<style>
  body { font-family: system-ui, sans-serif; max-width: 40rem; margin: 4rem auto; padding: 0 1rem; color: #1a1a1a; line-height: 1.5; }
  code, pre { background: #f4f4f5; padding: 0.1rem 0.3rem; border-radius: 3px; font-size: 0.95em; }
  pre { padding: 0.8rem 1rem; overflow-x: auto; }
  h1 { font-size: 1.4rem; }
  a { color: #2563eb; }
</style>
</head>
<body>
<h1>Web console wasn&rsquo;t bundled into this daemon</h1>
<p>
  The daemon&rsquo;s API is healthy (try
  <a href="/api/v1/health"><code>/api/v1/health</code></a>),
  but the operator console at <code>/</code> wasn&rsquo;t embedded into this
  binary. The Go <code>//go:embed</code> directive snapshots
  <code>web/dist/</code> at compile time, and this build was produced before
  the SPA was built.
</p>
<p>To fix it, rebuild with the SPA bundled in:</p>
<pre>make dist
./bin/gophertrunk</pre>
<p>
  Or, equivalently:
  <code>make web-build &amp;&amp; make build</code>.
  See <code>docs/web.md</code> for details.
</p>
<p>
  The REST and WebSocket APIs continue to work normally — only the
  embedded SPA is missing.
</p>
</body>
</html>
`

// gate wraps a mutation handler in the auth middleware. The middleware
// returns 401 when a token is required but missing / wrong, 403 when
// auth is disabled by misconfiguration, and otherwise dispatches to
// the handler. The body always carries the same {"error":"..."}
// envelope existing 4xx handlers use.
func (s *Server) gate(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if status, reason := s.auth.authorize(r); status != 0 {
			s.writeError(w, status, reason)
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

// writeError emits the standard {"error": "..."} JSON envelope. When
// diagnostics.verbose_errors is enabled and a collector is wired, it
// also attaches a "diag" object carrying the diagnostics banner
// (version, OS, system specs, detected dongles) so an API consumer
// triaging a failure gets the same macro context the CLI banner shows.
// The banner exposes host + dongle info, so it is only included when
// verbose is explicitly enabled (see ServerOptions.VerboseErrors).
func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	if s.verboseErrors && s.diagnostics != nil {
		writeJSON(w, status, map[string]any{
			"error": msg,
			"diag": map[string]any{
				"banner": gtdiag.FormatBannerPlain(s.diagnostics.SysInfo()),
			},
		})
		return
	}
	writeJSON(w, status, map[string]string{"error": msg})
}
