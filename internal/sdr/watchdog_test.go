package sdr

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestWatchdogTickPublishesDetachedOnFirstMissing pins the watchdog's
// first action when a pool serial disappears from EnumerateAll: flip
// the per-serial state to missing and emit one KindSDRDetached so the
// API / web / TUI snapshot consumers see the gap. The second tick
// against the same missing serial must NOT republish (idempotent).
func TestWatchdogTickPublishesDetachedOnFirstMissing(t *testing.T) {
	drv := &reacquireDriver{name: "fake-wd-missing", infos: []Info{
		{Driver: "fake-wd-missing", Index: 0, Serial: "WD1"},
	}}
	registerDriver(t, drv.name, drv)

	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	p := NewPool(nil)
	p.SetBus(bus)
	if err := p.Open(0, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	// Drain the open-time Attached event.
	select {
	case <-sub.C:
	case <-time.After(time.Second):
		t.Fatal("missing open-time event")
	}

	// Pull the device off the bus.
	drv.infos = nil
	missing := map[string]bool{}
	p.watchdogTick(missing, 0)
	if !missing["WD1"] {
		t.Error("watchdog did not mark WD1 missing")
	}
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindSDRDetached {
			t.Errorf("first tick event = %v, want sdr.detached", ev.Kind)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("watchdog did not publish detached event on first missing tick")
	}

	// Second tick against still-missing serial: no new event.
	p.watchdogTick(missing, 0)
	select {
	case ev := <-sub.C:
		t.Errorf("watchdog republished on still-missing tick: %v", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

// TestWatchdogTickReacquiresOnReappear pins the recovery path: a
// serial that was missing last tick and is back this tick triggers
// Reacquire. The reacquire itself publishes Detached+Attached, the
// pool entry's Device is swapped, and the missing map is cleared.
func TestWatchdogTickReacquiresOnReappear(t *testing.T) {
	drv := &reacquireDriver{name: "fake-wd-reappear", infos: []Info{
		{Driver: "fake-wd-reappear", Index: 0, Serial: "WD2"},
	}}
	registerDriver(t, drv.name, drv)

	p := NewPool(nil)
	if err := p.Open(0, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	original := p.FindBySerial("WD2")
	if original == nil {
		t.Fatal("missing initial WD2 entry")
	}
	oldDev := original.Device

	// Simulate missing → present.
	missing := map[string]bool{"WD2": true}
	// Pretend the kernel re-enumerated under a new index.
	drv.infos = []Info{{Driver: drv.name, Index: 9, Serial: "WD2"}}
	p.watchdogTick(missing, 0)

	if missing["WD2"] {
		t.Error("watchdog should clear missing map entry on successful reappear")
	}
	if p.FindBySerial("WD2").Device == oldDev {
		t.Error("watchdog did not trigger Reacquire on reappear (device handle unchanged)")
	}
	if p.FindBySerial("WD2").Info.Index != 9 {
		t.Errorf("Info.Index = %d, want 9 (refreshed from re-enumerate)",
			p.FindBySerial("WD2").Info.Index)
	}
}

// TestWatchdogTickQuietForPresentDevice ensures the watchdog leaves
// healthy serials alone — no Reacquire, no events.
func TestWatchdogTickQuietForPresentDevice(t *testing.T) {
	drv := &reacquireDriver{name: "fake-wd-healthy", infos: []Info{
		{Driver: "fake-wd-healthy", Index: 0, Serial: "WD3"},
	}}
	registerDriver(t, drv.name, drv)

	bus := events.NewBus(4)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	p := NewPool(nil)
	p.SetBus(bus)
	if err := p.Open(0, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	// Drain open-time event.
	<-sub.C

	missing := map[string]bool{}
	openCountBefore := atomic.LoadInt32(&driverOpenCounter(drv).i)
	p.watchdogTick(missing, 0)
	if len(missing) != 0 {
		t.Errorf("missing map = %v, want empty for healthy device", missing)
	}
	if atomic.LoadInt32(&driverOpenCounter(drv).i) != openCountBefore {
		t.Error("watchdog opened device on healthy tick (should be no-op)")
	}
	select {
	case ev := <-sub.C:
		t.Errorf("watchdog published event for healthy device: %v", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

// TestRunWatchdogDisabledOnZeroInterval pins the opt-out: a non-
// positive interval parks the watchdog on ctx without ever ticking
// or touching the driver registry.
func TestRunWatchdogDisabledOnZeroInterval(t *testing.T) {
	p := NewPool(nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.RunWatchdog(ctx, 0, 0) }()
	select {
	case err := <-done:
		t.Fatalf("watchdog exited before cancel: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Error("RunWatchdog returned nil after cancel, want ctx.Err()")
		}
	case <-time.After(time.Second):
		t.Fatal("RunWatchdog did not exit after ctx cancel")
	}
}

// driverOpenCounter is a small adapter so the healthy-tick test can
// peek at the open count even though the driver type itself uses an
// untyped int slice. Keeps the production type free of test-only
// accessors.
type counter struct{ i int32 }

func driverOpenCounter(r *reacquireDriver) *counter {
	return &counter{i: int32(len(r.opens))}
}
