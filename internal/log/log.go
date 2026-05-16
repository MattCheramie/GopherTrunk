package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// SwappableWriter is an io.Writer that can be redirected to a
// different target at runtime without re-wiring the slog handler.
// The daemon installs one as the slog sink at startup; the launcher
// redirects it to a tempfile while the in-process TUI owns the
// terminal, then restores it on exit so headless mode keeps stderr.
type SwappableWriter struct {
	mu    sync.Mutex
	out   io.Writer
	saved io.Writer
}

// NewSwappableWriter returns a SwappableWriter pointed at w.
func NewSwappableWriter(w io.Writer) *SwappableWriter {
	return &SwappableWriter{out: w}
}

// Write delegates to the active target. Holds the mutex briefly so
// concurrent log lines never interleave with a redirect transition.
func (s *SwappableWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	w := s.out
	s.mu.Unlock()
	if w == nil {
		return len(p), nil
	}
	return w.Write(p)
}

// Redirect replaces the active target. The previous target is
// remembered so Restore can reinstate it.
func (s *SwappableWriter) Redirect(w io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = s.out
	s.out = w
}

// Restore reverts to the writer that was active before the most
// recent Redirect. No-op when nothing was saved.
func (s *SwappableWriter) Restore() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saved != nil {
		s.out = s.saved
		s.saved = nil
	}
}

// New builds a slog.Logger whose handler writes to os.Stderr via a
// SwappableWriter. The default constructor is preserved for callers
// that don't need the swap surface.
func New(level, format string) *slog.Logger {
	logger, _ := NewWithSwap(level, format)
	return logger
}

// NewWithSwap is the variant used by main: it returns both the slog
// Logger and the SwappableWriter behind its handler, so the launcher
// can redirect stderr while the TUI runs.
func NewWithSwap(level, format string) (*slog.Logger, *SwappableWriter) {
	sw := NewSwappableWriter(os.Stderr)
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if strings.EqualFold(format, "json") {
		h = slog.NewJSONHandler(sw, opts)
	} else {
		h = slog.NewTextHandler(sw, opts)
	}
	return slog.New(h), sw
}
