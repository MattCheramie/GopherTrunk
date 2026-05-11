package cchunt

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeTuner records every SetCenterFreq call.
type fakeTuner struct {
	mu     sync.Mutex
	calls  []uint32
	err    error
}

func (f *fakeTuner) SetCenterFreq(hz uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, hz)
	return f.err
}

func (f *fakeTuner) tuned() []uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]uint32, len(f.calls))
	copy(out, f.calls)
	return out
}

// fakeLock is a payload satisfying trunking.LockedPayload so the
// supervisor's listen() goroutine accepts our synthetic cc.locked.
type fakeLock struct {
	freq uint32
	nac  uint16
}

func (f fakeLock) LockedFrequencyHz() uint32 { return f.freq }
func (f fakeLock) LockedNAC() uint16          { return f.nac }

func TestSupervisorFailsClosedWhenNothingLocks(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	tuner := &fakeTuner{}
	sys := trunking.System{
		Name:            "Demo",
		Protocol:        trunking.ProtocolP25,
		ControlChannels: []uint32{851_000_000, 852_000_000},
	}
	sup, err := New(Options{
		Bus:            bus,
		Tuner:          tuner,
		Systems:        []trunking.System{sys},
		Dwell:          50 * time.Millisecond,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     200 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Capture HuntFailed events.
	sub := bus.Subscribe()
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = sup.Run(ctx) }()

	// Wait until we see KindHuntFailed for the Demo system.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindHuntFailed {
				if f, ok := ev.Payload.(trunking.HuntFailed); ok && f.System == "Demo" {
					cancel()
					// Drain remaining events briefly so the goroutine exits.
					done := make(chan struct{})
					go func() { time.Sleep(100 * time.Millisecond); close(done) }()
					<-done
					if calls := tuner.tuned(); len(calls) < 2 {
						t.Errorf("tuner saw %d retunes, want at least 2 (both CCs)", len(calls))
					}
					snap := sup.Snapshot()
					if len(snap) != 1 || snap[0].Name != "Demo" {
						t.Fatalf("snapshot = %+v", snap)
					}
					if snap[0].State != StateFailed {
						t.Errorf("snapshot state = %q, want failed", snap[0].State)
					}
					return
				}
			}
		case <-deadline:
			cancel()
			t.Fatal("never saw KindHuntFailed for Demo")
		}
	}
}

func TestSupervisorLocksAndParks(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	tuner := &fakeTuner{}
	sys := trunking.System{
		Name:            "Locks",
		Protocol:        trunking.ProtocolP25,
		ControlChannels: []uint32{851_000_000, 852_000_000},
	}
	sup, err := New(Options{
		Bus:            bus,
		Tuner:          tuner,
		Systems:        []trunking.System{sys},
		Dwell:          200 * time.Millisecond,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = sup.Run(ctx) }()

	// Give the supervisor a moment to start hunting, then publish a
	// cc.locked for the second candidate.
	time.Sleep(80 * time.Millisecond)
	bus.Publish(events.Event{
		Kind:    events.KindCCLocked,
		Payload: fakeLock{freq: 852_000_000, nac: 0xBEEF},
	})

	// Snapshot should report StateLocked within the dwell window.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snap := sup.Snapshot()
		if len(snap) == 1 && snap[0].State == StateLocked && snap[0].LockedFreqHz == 852_000_000 {
			if snap[0].NAC != 0xBEEF {
				t.Errorf("NAC = %X, want BEEF", snap[0].NAC)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("snapshot never reached StateLocked: %+v", sup.Snapshot())
}

func TestSupervisorHoldAndResume(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sup, err := New(Options{
		Bus:     bus,
		Tuner:   &fakeTuner{},
		Systems: []trunking.System{{Name: "H", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{1}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sup.Hold("H") {
		t.Fatal("Hold('H') = false")
	}
	if got := sup.Snapshot()[0].State; got != StateHeld {
		t.Errorf("state after Hold = %q, want held", got)
	}
	if sup.Hold("missing") {
		t.Errorf("Hold('missing') = true, want false")
	}
	if !sup.Resume("H") {
		t.Fatal("Resume('H') = false")
	}
	if got := sup.Snapshot()[0].State; got != StateIdle {
		t.Errorf("state after Resume = %q, want idle", got)
	}
}

func TestSupervisorForceRetuneClearsBackoff(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sup, err := New(Options{
		Bus:     bus,
		Tuner:   &fakeTuner{},
		Systems: []trunking.System{{Name: "R", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{1}}},
		InitialBackoff: time.Hour,
		MaxBackoff:     time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Mark as failed with a long backoff.
	sup.markFailed("R")
	if sup.backoffRemaining("R") <= 0 {
		t.Fatal("expected non-zero backoff after markFailed")
	}
	if !sup.ForceRetune("R") {
		t.Fatal("ForceRetune('R') = false")
	}
	if sup.backoffRemaining("R") > 0 {
		t.Errorf("backoff not cleared after ForceRetune")
	}
}

func TestSupervisorRunReturnsCtxErr(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	sup, err := New(Options{Bus: bus, Tuner: &fakeTuner{}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	got := sup.Run(ctx)
	if got != context.DeadlineExceeded {
		t.Errorf("Run() = %v, want context.DeadlineExceeded", got)
	}
}
