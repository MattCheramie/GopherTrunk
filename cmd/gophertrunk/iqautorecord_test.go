package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeClock is a manually-advanced clock for deterministic cooldown tests.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// captureRecorder records every capture the auto-recorder actuates and blocks
// until released, so in-flight bookkeeping can be tested.
type captureRecorder struct {
	mu    sync.Mutex
	paths []string
	block chan struct{} // when non-nil, capture blocks on it
}

func (r *captureRecorder) fn(_ context.Context, path string, _ siglab.SampleFormat, _ int) (int64, uint64, error) {
	r.mu.Lock()
	r.paths = append(r.paths, path)
	block := r.block
	r.mu.Unlock()
	if block != nil {
		<-block
	}
	return 100, 0, nil
}

func (r *captureRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.paths)
}

// newTestAutoRecorder wires an auto-recorder with a fake clock + capture stub.
func newTestAutoRecorder(t *testing.T, cfg config.BasebandAutoRecordConfig) (*iqAutoRecorder, *captureRecorder, *fakeClock) {
	t.Helper()
	cfg.Enabled = true
	if cfg.Dir == "" {
		cfg.Dir = t.TempDir()
	}
	if cfg.Seconds == 0 {
		cfg.Seconds = 4
	}
	a := newIQAutoRecorder(cfg, "TETRA_Site_1", "tetra", "cc:same-carrier", nil, nil)
	if a == nil {
		t.Fatal("newIQAutoRecorder returned nil for enabled config")
	}
	clock := &fakeClock{t: time.Date(2026, 7, 23, 23, 0, 0, 0, time.UTC)}
	a.now = clock.now
	rec := &captureRecorder{}
	a.capture = rec.fn
	// A capture goroutine writes its metadata sidecar into cfg.Dir AFTER the
	// capture func returns, so wait for the recorder to go idle before t.TempDir's
	// RemoveAll runs (registered earlier ⇒ this LIFO cleanup runs first). Release
	// any capture still blocked on rec.block so the drain can complete.
	t.Cleanup(func() {
		rec.mu.Lock()
		if rec.block != nil {
			select {
			case <-rec.block: // already closed
			default:
				close(rec.block)
			}
			rec.block = nil
		}
		rec.mu.Unlock()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) && a.inFlightCount() > 0 {
			time.Sleep(2 * time.Millisecond)
		}
	})
	return a, rec, clock
}

// waitFor polls until pred is true or the deadline elapses (captures run in
// their own goroutine).
func waitFor(t *testing.T, pred func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

func grant(system string, tg uint32, ts uint8) trunking.Grant {
	return trunking.Grant{System: system, Protocol: "tetra", GroupID: tg, Timeslot: ts, FrequencyHz: 467913000}
}

func TestAutoRecordDisabledReturnsNil(t *testing.T) {
	if a := newIQAutoRecorder(config.BasebandAutoRecordConfig{Enabled: false}, "s", "tetra", "x", nil, nil); a != nil {
		t.Fatal("expected nil for disabled config")
	}
}

func TestAutoRecordConcurrentTrigger(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnConcurrentCalls: 2})

	// One active call: below threshold, no capture.
	a.onGrant(grant("TETRA_Site_1", 1001, 1))
	if rec.count() != 0 {
		t.Fatalf("single call should not trigger; got %d captures", rec.count())
	}
	// A distinct second call within the window: at threshold, capture fires.
	a.onGrant(grant("TETRA_Site_1", 1002, 2))
	waitFor(t, func() bool { return rec.count() == 1 })
}

func TestAutoRecordConcurrencyWindowExpiry(t *testing.T) {
	a, rec, clock := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnConcurrentCalls: 2})
	a.onGrant(grant("TETRA_Site_1", 1001, 1))
	// Advance past the window so the first call ages out before the second.
	clock.advance(autoRecordConcurrencyWindow + time.Second)
	a.onGrant(grant("TETRA_Site_1", 1002, 2))
	// Only one call is live in the window -> below threshold -> no capture.
	time.Sleep(20 * time.Millisecond)
	if rec.count() != 0 {
		t.Fatalf("aged-out call should not count toward concurrency; got %d captures", rec.count())
	}
}

func TestAutoRecordEncryptedAndEmergency(t *testing.T) {
	a, rec, clock := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, OnEmergency: true, Cooldown: "0s"})
	g := grant("TETRA_Site_1", 1001, 1)
	g.Encrypted = true
	a.onGrant(g)
	waitFor(t, func() bool { return rec.count() == 1 })

	clock.advance(time.Second)
	g2 := grant("TETRA_Site_1", 1002, 1)
	g2.Emergency = true
	a.onGrant(g2)
	waitFor(t, func() bool { return rec.count() == 2 })
}

func TestAutoRecordNoTriggerWhenClear(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, OnEmergency: true, OnConcurrentCalls: 5})
	a.onGrant(grant("TETRA_Site_1", 1001, 1)) // clear, single call
	time.Sleep(20 * time.Millisecond)
	if rec.count() != 0 {
		t.Fatalf("clear single call should not trigger; got %d captures", rec.count())
	}
}

func TestAutoRecordNoVoiceDeviceTrigger(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnNoVoiceDevice: true})
	a.onGrantUnserved(grant("TETRA_Site_1", 1001, 1))
	waitFor(t, func() bool { return rec.count() == 1 })

	// With the trigger off, an unserved grant is ignored.
	b, rec2, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnConcurrentCalls: 2})
	b.onGrantUnserved(grant("TETRA_Site_1", 1001, 1))
	time.Sleep(20 * time.Millisecond)
	if rec2.count() != 0 {
		t.Fatalf("on_no_voice_device off should ignore unserved grant; got %d captures", rec2.count())
	}
}

func TestAutoRecordCooldownThrottlesAuto(t *testing.T) {
	a, rec, clock := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, Cooldown: "10s"})
	enc := func(tg uint32) trunking.Grant { g := grant("TETRA_Site_1", tg, 1); g.Encrypted = true; return g }

	a.onGrant(enc(1))
	waitFor(t, func() bool { return rec.count() == 1 })

	// Within cooldown: suppressed.
	clock.advance(3 * time.Second)
	a.onGrant(enc(2))
	time.Sleep(20 * time.Millisecond)
	if rec.count() != 1 {
		t.Fatalf("trigger within cooldown should be suppressed; got %d captures", rec.count())
	}
	// Past cooldown: fires again.
	clock.advance(8 * time.Second)
	a.onGrant(enc(3))
	waitFor(t, func() bool { return rec.count() == 2 })
}

func TestAutoRecordManualBypassesCooldown(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, Cooldown: "1h"})
	a.TriggerManual()
	waitFor(t, func() bool { return rec.count() == 1 })
	// A second manual trigger fires despite the long cooldown.
	a.TriggerManual()
	waitFor(t, func() bool { return rec.count() == 2 })
}

func TestAutoRecordInFlightCap(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, Cooldown: "0s"})
	rec.mu.Lock()
	rec.block = make(chan struct{})
	rec.mu.Unlock()

	// Fire more than the cap while captures are blocked.
	for i := 0; i < autoRecordMaxInFlight+3; i++ {
		a.TriggerManual()
	}
	waitFor(t, func() bool { return rec.count() == autoRecordMaxInFlight })
	// Give any surplus goroutines a chance; the cap must hold.
	time.Sleep(20 * time.Millisecond)
	if rec.count() != autoRecordMaxInFlight {
		t.Fatalf("in-flight cap breached: got %d, want %d", rec.count(), autoRecordMaxInFlight)
	}
	// The blocked captures are released and drained by the t.Cleanup registered
	// in newTestAutoRecorder before the temp dir is removed.
}

func TestAutoRecordFilename(t *testing.T) {
	at := time.Date(2026, 7, 23, 20, 16, 8, 0, time.UTC)
	name := autoRecordFilename("TETRA_Site_1", "concurrent", at, 467913000, 50000, siglab.FormatS16)
	want := "TETRA_Site_1_20260723T201608Z_concurrent_467913000_50000hz.cs16"
	if name != want {
		t.Fatalf("filename = %q, want %q", name, want)
	}
}

// TestAutoRecordRunDispatch drives Run with a real event bus and checks the
// event switch routes KindGrant / KindGrantUnserved to the right handlers.
func TestAutoRecordRunDispatch(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnNoVoiceDevice: true, OnEncrypted: true, Cooldown: "0s"})
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = a.Run(ctx, sub) }()

	enc := grant("TETRA_Site_1", 1, 1)
	enc.Encrypted = true
	bus.Publish(events.Event{Kind: events.KindGrant, Payload: enc})
	bus.Publish(events.Event{Kind: events.KindGrantUnserved, Payload: grant("TETRA_Site_1", 2, 1)})
	waitFor(t, func() bool { return rec.count() == 2 })
}
