package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// EngineSnapshot is the subset of trunking.Engine the API needs. Decoupling
// from the concrete type keeps the API testable with a fake engine.
type EngineSnapshot interface {
	ActiveCalls() []*trunking.ActiveCall
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
	Enabled      bool                  `json:"enabled"`
	State        string                `json:"state,omitempty"`
	DeviceSerial string                `json:"device_serial,omitempty"`
	CursorIndex  int                   `json:"cursor_index,omitempty"`
	Channels     []ConvChannelStatusDTO `json:"channels"`
}

// ConvChannelStatusDTO mirrors conventional.ChannelStatus.
type ConvChannelStatusDTO struct {
	Index       int       `json:"index"`
	Label       string    `json:"label"`
	FrequencyHz uint32    `json:"frequency_hz"`
	Mode        string    `json:"mode"`
	Active      bool      `json:"active"`
	LastBreakAt time.Time `json:"last_break_at,omitempty"`
}

// Server hosts the GopherTrunk HTTP/SSE/WebSocket API. A separate gRPC
// server (internal/api/grpc.go) shares the same in-process state.
type Server struct {
	addr       string
	bus        *events.Bus
	engine     EngineSnapshot
	mutator    EngineMutator
	retention  RetentionSweeper
	tones      ToneDetectorReset
	devices    DevicesProvider
	scanner    ScannerCockpit
	audio      AudioController
	talkgroups *trunking.TalkgroupDB
	systems    []trunking.System
	history    HistoryQuery
	metrics    http.Handler
	log        *slog.Logger
	version    string

	allowMutations bool

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

// HistoryFilter mirrors storage.HistoryFilter for the api layer's
// purposes (passed through to whatever HistoryQuery implementation the
// daemon wires up).
type HistoryFilter struct {
	System    string
	GroupID   uint32
	Since     time.Time
	Until     time.Time
	Limit     int
	OnlyEnded bool
}

// CallRow mirrors storage.CallRow as a JSON-friendly row. Lives in the
// api package so the storage package can stay free of API concerns.
type CallRow struct {
	ID             int64     `json:"id"`
	System         string    `json:"system"`
	Protocol       string    `json:"protocol"`
	GroupID        uint32    `json:"group_id"`
	SourceID       uint32    `json:"source_id"`
	FrequencyHz    uint32    `json:"frequency_hz"`
	Encrypted      bool      `json:"encrypted"`
	Emergency      bool      `json:"emergency"`
	DataCall       bool      `json:"data_call"`
	DeviceSerial   string    `json:"device_serial"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at,omitempty"`
	DurationMs     int64     `json:"duration_ms,omitempty"`
	EndReason      string    `json:"end_reason,omitempty"`
	TalkgroupAlpha string    `json:"talkgroup_alpha,omitempty"`
}

// ServerOptions configure a new Server.
type ServerOptions struct {
	// Addr is the listen address (e.g. ":8080" or "127.0.0.1:9000").
	Addr       string
	Bus        *events.Bus
	Engine     EngineSnapshot
	Talkgroups *trunking.TalkgroupDB
	Systems    []trunking.System
	// History is optional. When non-nil the server exposes
	// GET /api/v1/calls/history.
	History HistoryQuery
	// MetricsHandler is optional. When non-nil it is mounted at
	// GET /metrics; the daemon passes internal/metrics.Metrics.Handler()
	// here. Decoupling via http.Handler keeps the api package free of a
	// hard dependency on the metrics package.
	MetricsHandler http.Handler
	Log            *slog.Logger
	// Version is reported by GET /api/v1/version.
	Version string
	// AllowMutations gates the write endpoints. Off by default —
	// the HTTP API has no authentication, so mutations are unsafe
	// to expose unless the operator explicitly opts in.
	AllowMutations bool
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
	// Audio exposes the live-audio player + recorder gate for
	// GET + PATCH /api/v1/audio. Optional; when nil, the routes
	// return 503.
	Audio AudioController
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
	return &Server{
		addr:           opts.Addr,
		bus:            opts.Bus,
		engine:         opts.Engine,
		mutator:        opts.Mutator,
		retention:      opts.Retention,
		tones:          opts.Tones,
		devices:        opts.Devices,
		scanner:        opts.Scanner,
		audio:          opts.Audio,
		talkgroups:     opts.Talkgroups,
		systems:        append([]trunking.System(nil), opts.Systems...),
		history:        opts.History,
		metrics:        opts.MetricsHandler,
		log:            log,
		version:        opts.Version,
		allowMutations: opts.AllowMutations,
	}, nil
}

// Run binds the listener and serves until ctx cancels.
func (s *Server) Run(ctx context.Context) error {
	mux := s.routes()
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("api: listening", "addr", listener.Addr().String())
		if err := s.srv.Serve(listener); err != nil && err != http.ErrServerClosed {
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

func (s *Server) shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.srv == nil {
		s.closed = true
		return nil
	}
	s.closed = true
	shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(shutCtx)
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/version", s.handleVersion)
	mux.HandleFunc("GET /api/v1/systems", s.handleListSystems)
	mux.HandleFunc("GET /api/v1/systems/{name}", s.handleGetSystem)
	mux.HandleFunc("GET /api/v1/talkgroups", s.handleListTalkgroups)
	mux.HandleFunc("GET /api/v1/talkgroups/{id}", s.handleGetTalkgroup)
	mux.HandleFunc("GET /api/v1/calls/active", s.handleActiveCalls)
	mux.HandleFunc("GET /api/v1/calls/history", s.handleCallHistory)
	mux.HandleFunc("GET /api/v1/devices", s.handleListDevices)
	mux.HandleFunc("GET /api/v1/events", s.handleSSE)
	mux.HandleFunc("GET /api/v1/events/ws", s.handleWS)
	if s.metrics != nil {
		mux.Handle("GET /metrics", s.metrics)
	}

	// Mutation routes — wrapped in s.gate so a non-AllowMutations
	// daemon returns 403 without dispatching to the handler. The
	// gate also reports the daemon's mutation capability via
	// GET /api/v1/mutations so clients can light up keybindings.
	mux.HandleFunc("GET /api/v1/mutations", s.handleMutationStatus)
	mux.HandleFunc("POST /api/v1/calls/{deviceSerial}/end", s.gate(s.handleEndCall))
	mux.HandleFunc("PATCH /api/v1/talkgroups/{id}", s.gate(s.handleUpdateTalkgroup))
	mux.HandleFunc("POST /api/v1/retention/sweep", s.gate(s.handleRetentionSweep))
	mux.HandleFunc("POST /api/v1/devices/{serial}/tone-reset", s.gate(s.handleToneReset))

	// Scanner cockpit — read endpoint is always open; mutations are
	// gated behind allow_mutations like every other write route.
	mux.HandleFunc("GET /api/v1/scanner", s.handleScannerStatus)
	mux.HandleFunc("PATCH /api/v1/scanner", s.gate(s.handleScannerSetMode))
	mux.HandleFunc("POST /api/v1/scanner/hunt/{system}/hold", s.gate(s.handleHuntHold))
	mux.HandleFunc("POST /api/v1/scanner/hunt/{system}/resume", s.gate(s.handleHuntResume))
	mux.HandleFunc("POST /api/v1/scanner/hunt/{system}/retune", s.gate(s.handleHuntRetune))
	mux.HandleFunc("POST /api/v1/scanner/conventional/hold", s.gate(s.handleConvHold))
	mux.HandleFunc("POST /api/v1/scanner/conventional/resume", s.gate(s.handleConvResume))
	mux.HandleFunc("POST /api/v1/scanner/conventional/{index}/dwell", s.gate(s.handleConvDwell))

	// Audio cockpit — read endpoint is always open; the PATCH is
	// gated behind allow_mutations like every other write route.
	mux.HandleFunc("GET /api/v1/audio", s.handleAudioStatus)
	mux.HandleFunc("PATCH /api/v1/audio", s.gate(s.handleAudioPatch))

	return mux
}

// gate wraps a handler so it short-circuits with 403 when the
// daemon was started without api.allow_mutations. The body carries
// the same {"error":"..."} envelope existing 4xx handlers use.
func (s *Server) gate(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.allowMutations {
			writeError(w, http.StatusForbidden, "mutations disabled (set api.allow_mutations: true to enable)")
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

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
