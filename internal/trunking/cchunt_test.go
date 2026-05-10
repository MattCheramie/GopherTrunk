package trunking_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeTuner records SetCenterFreq calls and lets the test publish a fake
// cc.locked event matching the latest tuned freq.
type fakeTuner struct {
	mu    sync.Mutex
	freqs []uint32
	err   error
}

func (f *fakeTuner) SetCenterFreq(hz uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.freqs = append(f.freqs, hz)
	return f.err
}

func (f *fakeTuner) tuned() []uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uint32(nil), f.freqs...)
}

func newSystem() trunking.System {
	return trunking.System{
		Name:            "TestSys",
		Protocol:        trunking.ProtocolP25,
		ControlChannels: []uint32{851_000_000, 852_000_000, 853_000_000},
	}
}

func TestHunterLocksOnFirstResponsiveFreq(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tuner := &fakeTuner{}
	cachePath := filepath.Join(t.TempDir(), "cc.json")
	cache, err := trunking.OpenCache(cachePath)
	if err != nil {
		t.Fatal(err)
	}

	h, err := trunking.NewHunter(trunking.HunterOptions{
		System: newSystem(),
		Tuner:  tuner,
		Bus:    bus,
		Cache:  cache,
		Dwell:  150 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Publish a lock for the second candidate after a short delay.
	go func() {
		time.Sleep(200 * time.Millisecond) // 1st freq dwell expires
		bus.Publish(events.Event{
			Kind:    events.KindCCLocked,
			Payload: phase1.LockState{FrequencyHz: 852_000_000, NAC: 0x123, DUID: phase1.DUIDTrunkingSignaling},
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := h.Hunt(ctx)
	if err != nil {
		t.Fatalf("Hunt: %v", err)
	}
	if res.Frequency != 852_000_000 || res.NAC != 0x123 {
		t.Errorf("LockResult = %+v", res)
	}
	freqs := tuner.tuned()
	if len(freqs) != 2 || freqs[0] != 851_000_000 || freqs[1] != 852_000_000 {
		t.Errorf("tuned sequence = %v, want [851M, 852M]", freqs)
	}
	cached, _ := cache.Get("TestSys")
	if cached.LastFrequencyHz != 852_000_000 {
		t.Errorf("cache = %+v", cached)
	}
}

func TestHunterReturnsErrWhenAllExpire(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tuner := &fakeTuner{}
	h, _ := trunking.NewHunter(trunking.HunterOptions{
		System: newSystem(),
		Tuner:  tuner,
		Bus:    bus,
		Dwell:  20 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := h.Hunt(ctx)
	if !errors.Is(err, trunking.ErrNoControlChannel) {
		t.Errorf("err = %v, want ErrNoControlChannel", err)
	}
	if got := tuner.tuned(); len(got) != 3 {
		t.Errorf("tuned %d freqs, want 3 (full sweep)", len(got))
	}
}

func TestHunterStartsFromCachedFreq(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tuner := &fakeTuner{}
	cache, err := trunking.OpenCache(filepath.Join(t.TempDir(), "cc.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Set("TestSys", trunking.CachedSystem{LastFrequencyHz: 853_000_000, NAC: 0xABC}); err != nil {
		t.Fatal(err)
	}
	h, _ := trunking.NewHunter(trunking.HunterOptions{
		System: newSystem(),
		Tuner:  tuner,
		Bus:    bus,
		Cache:  cache,
		Dwell:  150 * time.Millisecond,
	})

	go func() {
		time.Sleep(20 * time.Millisecond)
		bus.Publish(events.Event{
			Kind:    events.KindCCLocked,
			Payload: phase1.LockState{FrequencyHz: 853_000_000, NAC: 0xABC, DUID: phase1.DUIDTrunkingSignaling},
		})
	}()

	res, err := h.Hunt(context.Background())
	if err != nil {
		t.Fatalf("Hunt: %v", err)
	}
	if res.Frequency != 853_000_000 {
		t.Errorf("locked freq = %d, want 853M", res.Frequency)
	}
	if got := tuner.tuned(); got[0] != 853_000_000 {
		t.Errorf("first tune = %d, want 853M (cached)", got[0])
	}
}

func TestHunterIgnoresMismatchedFreq(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tuner := &fakeTuner{}
	h, _ := trunking.NewHunter(trunking.HunterOptions{
		System: newSystem(),
		Tuner:  tuner,
		Bus:    bus,
		Dwell:  100 * time.Millisecond,
	})

	// Publish a lock for the wrong frequency. Hunter should ignore it,
	// dwell out, and continue scanning.
	go func() {
		time.Sleep(10 * time.Millisecond)
		bus.Publish(events.Event{
			Kind:    events.KindCCLocked,
			Payload: phase1.LockState{FrequencyHz: 999_999_999, NAC: 0x42, DUID: phase1.DUIDTrunkingSignaling},
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := h.Hunt(ctx)
	if !errors.Is(err, trunking.ErrNoControlChannel) {
		t.Errorf("err = %v, want ErrNoControlChannel", err)
	}
	if got := tuner.tuned(); len(got) != 3 {
		t.Errorf("expected full sweep, got %d freqs", len(got))
	}
}

func TestHunterReturnsCtxErr(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	h, _ := trunking.NewHunter(trunking.HunterOptions{
		System: newSystem(),
		Tuner:  &fakeTuner{},
		Bus:    bus,
		Dwell:  500 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.Hunt(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestNewHunterValidatesSystem(t *testing.T) {
	bus := events.NewBus(1)
	defer bus.Close()
	_, err := trunking.NewHunter(trunking.HunterOptions{
		System: trunking.System{Name: "X"}, // missing protocol + channels
		Tuner:  &fakeTuner{},
		Bus:    bus,
	})
	if err == nil {
		t.Error("expected validation error")
	}
}
