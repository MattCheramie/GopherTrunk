package player

import (
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// loopback is a Backend that captures every Write into a slice so
// tests can assert what the Player produced.
type loopback struct {
	mu      sync.Mutex
	samples []int16
	closed  bool
}

func (l *loopback) Write(samples []int16) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return io.ErrClosedPipe
	}
	l.samples = append(l.samples, samples...)
	return nil
}

func (l *loopback) Close() error {
	l.mu.Lock()
	l.closed = true
	l.mu.Unlock()
	return nil
}

func (l *loopback) snapshot() []int16 {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]int16, len(l.samples))
	copy(out, l.samples)
	return out
}

func newTestPlayer(t *testing.T, cfg Config) (*Player, *loopback) {
	t.Helper()
	lb := &loopback{}
	cfg.SampleRate = 8000
	cfg.Volume = 1.0
	p, err := NewWithBackend(lb, cfg, slog.Default())
	if err != nil {
		t.Fatalf("NewWithBackend: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	return p, lb
}

func TestPlayerWritePCMPassThrough(t *testing.T) {
	p, lb := newTestPlayer(t, Config{})
	in := []int16{100, 200, -300, 400}
	if err := p.WritePCM("dev", in); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	// give the run goroutine a tick to drain
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(lb.snapshot()) == len(in)
	})
	got := lb.snapshot()
	if len(got) != len(in) {
		t.Fatalf("loopback length: got %d, want %d", len(got), len(in))
	}
	for i := range in {
		if got[i] != in[i] {
			t.Errorf("sample %d: got %d, want %d", i, got[i], in[i])
		}
	}
}

func TestPlayerMuted(t *testing.T) {
	p, lb := newTestPlayer(t, Config{Muted: true})
	if err := p.WritePCM("dev", []int16{1, 2, 3}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if got := lb.snapshot(); len(got) != 0 {
		t.Fatalf("muted player wrote %d samples", len(got))
	}

	// Unmute and confirm samples flow.
	p.SetMuted(false)
	if err := p.WritePCM("dev", []int16{10, 20}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(lb.snapshot()) == 2
	})
	got := lb.snapshot()
	if len(got) != 2 || got[0] != 10 || got[1] != 20 {
		t.Fatalf("unexpected samples after unmute: %v", got)
	}
}

func TestPlayerVolumeHalves(t *testing.T) {
	p, lb := newTestPlayer(t, Config{})
	p.SetVolume(0.5)
	if err := p.WritePCM("dev", []int16{1000, -2000, 4000}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(lb.snapshot()) == 3
	})
	got := lb.snapshot()
	want := []int16{500, -1000, 2000}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("sample %d: got %d, want %d", i, got[i], w)
		}
	}
}

func TestPlayerNullBackend(t *testing.T) {
	// Enabled=false should produce a Player whose WritePCM is a
	// silent no-op, with backend == nil and Enabled stat false.
	p, err := New(Config{Enabled: false}, slog.Default())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()
	if err := p.WritePCM("dev", []int16{1, 2, 3}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	if st := p.Stats(); st.Enabled {
		t.Fatalf("expected Enabled=false on null player")
	}
}

func TestPlayerClosed(t *testing.T) {
	p, _ := newTestPlayer(t, Config{})
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Second Close is a no-op.
	if err := p.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func waitFor(t *testing.T, d time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", d)
}
