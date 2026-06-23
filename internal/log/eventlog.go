package log

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// EventLog writes every bus event to a newline-delimited JSON (JSONL / NDJSON)
// file — one self-contained JSON object per line — so an operator can record a
// full session for offline inspection (and hand it to support / Claude). Unlike
// the human-readable MessageLog it captures ALL event kinds, not just the
// trunking subset, in the same envelope the SSE/WS streams emit. The file
// rotates to "<path>.1" when it exceeds the configured size cap.
type EventLog struct {
	bus       *events.Bus
	path      string
	maxBytes  int64
	encode    func(events.Event) ([]byte, error)
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	mu   sync.Mutex
	f    *os.File
	size int64
}

// EventLogOptions configure an EventLog.
type EventLogOptions struct {
	Bus *events.Bus
	// Path is the log file path. Required.
	Path string
	// MaxSizeMB caps the file size before rotation. Default 16.
	MaxSizeMB int
	// Encode renders one event to a single JSON line's worth of bytes (without
	// the trailing newline). Optional: when nil a minimal {kind,timestamp,
	// payload} envelope is used. The daemon injects the api package's wire
	// encoder so the file matches the live SSE/WS stream byte-for-byte.
	Encode func(events.Event) ([]byte, error)
}

// eventEnvelope is the default JSON shape when no Encode is supplied. It mirrors
// the api.EventDTO field names so a consumer sees the same keys either way.
type eventEnvelope struct {
	Kind      string    `json:"kind"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

func defaultEncode(ev events.Event) ([]byte, error) {
	return json.Marshal(eventEnvelope{
		Kind:      string(ev.Kind),
		Timestamp: ev.Timestamp,
		Payload:   ev.Payload,
	})
}

// NewEventLog opens the log file and subscribes to the bus.
func NewEventLog(opts EventLogOptions) (*EventLog, error) {
	if opts.Bus == nil {
		return nil, errors.New("log/eventlog: events.Bus is required")
	}
	if opts.Path == "" {
		return nil, errors.New("log/eventlog: Path is required")
	}
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = 16
	}
	if opts.Encode == nil {
		opts.Encode = defaultEncode
	}
	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return nil, fmt.Errorf("log/eventlog: mkdir: %w", err)
	}
	f, err := os.OpenFile(opts.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("log/eventlog: open: %w", err)
	}
	info, _ := f.Stat()
	el := &EventLog{
		bus:      opts.Bus,
		path:     opts.Path,
		maxBytes: int64(opts.MaxSizeMB) * 1024 * 1024,
		encode:   opts.Encode,
		runDone:  make(chan struct{}),
		f:        f,
	}
	if info != nil {
		el.size = info.Size()
	}
	el.sub = opts.Bus.Subscribe()
	return el, nil
}

// Run drains events until ctx cancels or the bus closes.
func (e *EventLog) Run(ctx context.Context) error {
	defer close(e.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-e.sub.C:
			if !ok {
				return nil
			}
			b, err := e.encode(ev)
			if err != nil {
				// A single unencodable event must never tear down the sink;
				// drop it and keep recording the rest of the session.
				continue
			}
			b = append(b, '\n')
			e.write(b)
		}
	}
}

func (e *EventLog) write(line []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.f == nil {
		return
	}
	if e.maxBytes > 0 && e.size+int64(len(line)) > e.maxBytes {
		e.rotate()
	}
	n, err := e.f.Write(line)
	e.size += int64(n)
	_ = err
}

// rotate closes the current file, renames it to "<path>.1", and opens a fresh
// one. Caller holds e.mu.
func (e *EventLog) rotate() {
	e.f.Close()
	_ = os.Rename(e.path, e.path+".1")
	f, err := os.OpenFile(e.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		e.f = nil
		return
	}
	e.f = f
	e.size = 0
}

// Close releases the bus subscription, waits for Run to drain, and closes the
// file.
func (e *EventLog) Close() error {
	e.closeOnce.Do(func() {
		e.sub.Close()
		select {
		case <-e.runDone:
		case <-time.After(time.Second):
		}
		e.mu.Lock()
		if e.f != nil {
			e.f.Close()
			e.f = nil
		}
		e.mu.Unlock()
	})
	return nil
}
