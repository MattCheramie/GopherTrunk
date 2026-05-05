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
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// EngineSnapshot is the subset of trunking.Engine the API needs. Decoupling
// from the concrete type keeps the API testable with a fake engine.
type EngineSnapshot interface {
	ActiveCalls() []*trunking.ActiveCall
}

// Server hosts the GopherTrunk HTTP/SSE/WebSocket API. A separate gRPC
// server (internal/api/grpc.go) shares the same in-process state.
type Server struct {
	addr       string
	bus        *events.Bus
	engine     EngineSnapshot
	talkgroups *trunking.TalkgroupDB
	systems    []trunking.System
	history    HistoryQuery
	metrics    http.Handler
	log        *slog.Logger
	version    string

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
		addr:       opts.Addr,
		bus:        opts.Bus,
		engine:     opts.Engine,
		talkgroups: opts.Talkgroups,
		systems:    append([]trunking.System(nil), opts.Systems...),
		history:    opts.History,
		metrics:    opts.MetricsHandler,
		log:        log,
		version:    opts.Version,
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
	mux.HandleFunc("GET /api/v1/events", s.handleSSE)
	mux.HandleFunc("GET /api/v1/events/ws", s.handleWS)
	if s.metrics != nil {
		mux.Handle("GET /metrics", s.metrics)
	}
	return mux
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
