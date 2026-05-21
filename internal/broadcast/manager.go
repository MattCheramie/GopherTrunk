package broadcast

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Default tuning for the Manager.
const (
	defaultWorkers    = 2
	defaultMaxRetries = 3
	defaultRetryBase  = 2 * time.Second
	defaultQueueDepth = 64
)

// Options configure a Manager.
type Options struct {
	Bus *events.Bus
	Log *slog.Logger
	// Backends is the set of outbound destinations. A Manager with no
	// backends is valid but inert.
	Backends []Backend
	// MinDuration drops calls shorter than this from every backend.
	// Zero streams calls of any length.
	MinDuration time.Duration
	// Workers is the number of concurrent upload goroutines. Default 2.
	Workers int
	// MaxRetries is the per-backend retry budget on a failed Send.
	// Default 3.
	MaxRetries int
	// RetryBase is the first backoff delay; it doubles per attempt.
	// Default 2s.
	RetryBase time.Duration
}

// Stats is a point-in-time snapshot of a Manager's counters.
type Stats struct {
	Queued   int            `json:"queued"`
	Dropped  int            `json:"dropped"`
	Sent     map[string]int `json:"sent"`
	Failed   map[string]int `json:"failed"`
	Backends []string       `json:"backends"`
}

// Manager fans completed calls out to the configured backends.
type Manager struct {
	bus         *events.Bus
	log         *slog.Logger
	backends    []Backend
	minDuration time.Duration
	maxRetries  int
	retryBase   time.Duration

	sub       *events.Subscription
	jobs      chan *Call
	wg        sync.WaitGroup
	runDone   chan struct{}
	closeOnce sync.Once

	mu      sync.Mutex
	queued  int
	dropped int
	sent    map[string]int
	failed  map[string]int
}

// NewManager validates opts and returns a Manager that has already
// subscribed to the bus (so calls completed before Run starts are not
// lost). The caller must invoke Run and, on shutdown, Close.
func NewManager(opts Options) (*Manager, error) {
	if opts.Bus == nil {
		return nil, errors.New("broadcast: events.Bus is required")
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.Workers <= 0 {
		opts.Workers = defaultWorkers
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = defaultMaxRetries
	}
	if opts.RetryBase <= 0 {
		opts.RetryBase = defaultRetryBase
	}
	m := &Manager{
		bus:         opts.Bus,
		log:         opts.Log,
		backends:    opts.Backends,
		minDuration: opts.MinDuration,
		maxRetries:  opts.MaxRetries,
		retryBase:   opts.RetryBase,
		jobs:        make(chan *Call, defaultQueueDepth),
		runDone:     make(chan struct{}),
		sent:        make(map[string]int),
		failed:      make(map[string]int),
	}
	m.sub = opts.Bus.Subscribe()
	for i := 0; i < opts.Workers; i++ {
		m.wg.Add(1)
		go m.worker()
	}
	return m, nil
}

// Backends reports how many outbound destinations are configured.
func (m *Manager) Backends() int { return len(m.backends) }

// Run drains KindCallComplete events until ctx is cancelled or the bus
// subscription closes.
func (m *Manager) Run(ctx context.Context) error {
	defer close(m.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-m.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindCallComplete {
				continue
			}
			cc, ok := ev.Payload.(trunking.CallComplete)
			if !ok {
				continue
			}
			m.dispatch(cc)
		}
	}
}

// dispatch applies the stream gate + minimum-duration filter and
// enqueues the call when at least one backend wants it.
func (m *Manager) dispatch(cc trunking.CallComplete) {
	if len(m.backends) == 0 {
		return
	}
	if cc.Talkgroup != nil && !cc.Talkgroup.Stream {
		return // talkgroup opted out of all feeds
	}
	if m.minDuration > 0 && cc.Duration() < m.minDuration {
		return
	}
	call := callFromEvent(cc)
	wanted := false
	for _, b := range m.backends {
		if b.Accepts(call.System) {
			wanted = true
			break
		}
	}
	if !wanted {
		return
	}
	select {
	case m.jobs <- call:
		m.mu.Lock()
		m.queued++
		m.mu.Unlock()
	default:
		// Queue full — a backend is wedged. Drop rather than block
		// the event loop (and the recorder behind it).
		m.mu.Lock()
		m.dropped++
		m.mu.Unlock()
		m.log.Warn("broadcast: upload queue full, dropping call",
			"system", call.System, "tg", call.Talkgroup)
	}
}

func (m *Manager) worker() {
	defer m.wg.Done()
	for call := range m.jobs {
		for _, b := range m.backends {
			if !b.Accepts(call.System) {
				continue
			}
			m.sendWithRetry(b, call)
		}
	}
}

// sendWithRetry attempts b.Send up to maxRetries+1 times with exponential
// backoff between attempts.
func (m *Manager) sendWithRetry(b Backend, call *Call) {
	backoff := m.retryBase
	for attempt := 0; attempt <= m.maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		err := b.Send(ctx, call)
		cancel()
		if err == nil {
			m.mu.Lock()
			m.sent[b.Name()]++
			m.mu.Unlock()
			m.log.Info("broadcast: call streamed",
				"backend", b.Name(), "system", call.System,
				"tg", call.Talkgroup, "attempt", attempt+1)
			return
		}
		m.log.Warn("broadcast: upload failed",
			"backend", b.Name(), "system", call.System, "tg", call.Talkgroup,
			"attempt", attempt+1, "of", m.maxRetries+1, "err", err)
		if attempt < m.maxRetries {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	m.mu.Lock()
	m.failed[b.Name()]++
	m.mu.Unlock()
	m.log.Error("broadcast: giving up on call",
		"backend", b.Name(), "system", call.System, "tg", call.Talkgroup)
}

// Stats returns a snapshot of the Manager's counters.
func (m *Manager) Stats() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := Stats{
		Queued:  m.queued,
		Dropped: m.dropped,
		Sent:    make(map[string]int, len(m.sent)),
		Failed:  make(map[string]int, len(m.failed)),
	}
	for k, v := range m.sent {
		s.Sent[k] = v
	}
	for k, v := range m.failed {
		s.Failed[k] = v
	}
	for _, b := range m.backends {
		s.Backends = append(s.Backends, b.Name())
	}
	return s
}

// Close releases the bus subscription, waits for Run to drain, then
// stops the upload workers. Safe to call multiple times.
func (m *Manager) Close() error {
	m.closeOnce.Do(func() {
		m.sub.Close()
		select {
		case <-m.runDone:
		case <-time.After(2 * time.Second):
		}
		close(m.jobs)
		m.wg.Wait()
		for _, b := range m.backends {
			if c, ok := b.(interface{ Close() error }); ok {
				if err := c.Close(); err != nil {
					m.log.Warn("broadcast: backend close", "backend", b.Name(), "err", err)
				}
			}
		}
	})
	return nil
}

// String renders a one-line Manager summary for log output.
func (m *Manager) String() string {
	return fmt.Sprintf("broadcast.Manager(%d backends)", len(m.backends))
}
