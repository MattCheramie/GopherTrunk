package main

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// dyingWidebandDevice is a minimal sdr.Device whose IQ stream always
// closes immediately without a ctx-cancel — modelling a wideband dongle
// whose USB reaper keeps dying (or the macOS stall watchdog repeatedly
// aborting a frozen endpoint). Every StreamIQ hands back a channel that
// closes at once, so widebandt2.Engine.Run returns ErrIQStreamClosed on
// each attempt.
type dyingWidebandDevice struct{}

func (dyingWidebandDevice) Info() sdr.Info             { return sdr.Info{Driver: "mock", Serial: "WB-FATAL"} }
func (dyingWidebandDevice) SetCenterFreq(uint32) error { return nil }
func (dyingWidebandDevice) SetSampleRate(uint32) error { return nil }
func (dyingWidebandDevice) SetGain(int) error          { return nil }
func (dyingWidebandDevice) SetPPM(int) error           { return nil }
func (dyingWidebandDevice) SetBiasTee(bool) error      { return nil }
func (dyingWidebandDevice) Close() error               { return nil }
func (dyingWidebandDevice) StreamIQ(context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64)
	close(ch) // immediate reaper death: channel closes with the ctx still live
	return ch, nil
}

// TestRunWidebandWithRetry_FatalsAfterRetriesExhausted pins the wideband
// self-heal supervisor's terminal escalation: when the engine's IQ stream
// keeps dying (widebandt2.ErrIQStreamClosed) with no healthy window, the
// retry loop must exhaust its backoff schedule and record a fatal so an
// external supervisor restarts a clean process — the recovery the bare
// eng.Run(ctx) spawn (which just returned nil and stopped decoding) never
// provided for the macOS Airspy freeze.
func TestRunWidebandWithRetry_FatalsAfterRetriesExhausted(t *testing.T) {
	oldBackoffs := ccDecoderRetryBackoffs
	ccDecoderRetryBackoffs = []time.Duration{
		5 * time.Millisecond,
		5 * time.Millisecond,
	}
	defer func() { ccDecoderRetryBackoffs = oldBackoffs }()

	bus := events.NewBus(8)
	defer bus.Close()
	eng, err := widebandt2.New(widebandt2.Options{
		Log:          slog.New(slog.DiscardHandler),
		Bus:          bus,
		Device:       dyingWidebandDevice{},
		SampleRateHz: 2_400_000,
		CenterFreqHz: 453_500_000,
		Channels:     []widebandt2.ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "wbsys"}},
		Systems:      []trunking.System{{Name: "wbsys", Protocol: trunking.ProtocolDMRTier2}},
	})
	if err != nil {
		t.Fatalf("widebandt2.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// pool nil: the retry loop skips reacquire and just re-runs the engine,
	// which keeps dying, so the backoff schedule is exhausted.
	d := &Daemon{
		log:    slog.New(slog.DiscardHandler),
		bus:    bus,
		cancel: cancel,
	}

	got := d.runWidebandWithRetry(ctx, eng)
	if !errors.Is(got, widebandt2.ErrIQStreamClosed) {
		t.Errorf("runWidebandWithRetry = %v, want wrapped ErrIQStreamClosed", got)
	}
	if d.takeFatal() == nil {
		t.Error("expected recordFatal to fire after wideband retries exhausted")
	}
}

// TestRunWidebandWithRetry_CleanStopOnCancel confirms a ctx-cancel is a
// clean shutdown, not a stream death: the supervisor returns without
// recording a fatal so daemon teardown stays quiet.
func TestRunWidebandWithRetry_CleanStopOnCancel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	// A device whose stream stays open until ctx cancels.
	dev := &liveWidebandDevice{}
	eng, err := widebandt2.New(widebandt2.Options{
		Log:          slog.New(slog.DiscardHandler),
		Bus:          bus,
		Device:       dev,
		SampleRateHz: 2_400_000,
		CenterFreqHz: 453_500_000,
		Channels:     []widebandt2.ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "wbsys"}},
		Systems:      []trunking.System{{Name: "wbsys", Protocol: trunking.ProtocolDMRTier2}},
	})
	if err != nil {
		t.Fatalf("widebandt2.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	d := &Daemon{log: slog.New(slog.DiscardHandler), bus: bus, cancel: cancel}
	done := make(chan error, 1)
	go func() { done <- d.runWidebandWithRetry(ctx, eng) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("runWidebandWithRetry on cancel = %v, want nil/Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("supervisor did not return after ctx cancel")
	}
	if d.takeFatal() != nil {
		t.Errorf("recordFatal fired on a clean ctx-cancel stop: %v", d.takeFatal())
	}
}

// flappingWidebandDevice models a dongle that keeps recovering: each of its
// first maxDeaths StreamIQ sessions streams for `healthy` (long enough to count
// as real progress) then the reaper dies with the ctx still live; after that it
// parks the stream open until ctx cancels. It is the 9-Jul Airspy signature — a
// stream that heals, runs a few seconds, dies, repeatedly — never sustaining a
// single 60 s Run.
type flappingWidebandDevice struct {
	healthy   time.Duration
	maxDeaths int64
	calls     atomic.Int64
}

func (*flappingWidebandDevice) Info() sdr.Info          { return sdr.Info{Driver: "mock", Serial: "WB-FLAP"} }
func (*flappingWidebandDevice) SetCenterFreq(uint32) error { return nil }
func (*flappingWidebandDevice) SetSampleRate(uint32) error { return nil }
func (*flappingWidebandDevice) SetGain(int) error          { return nil }
func (*flappingWidebandDevice) SetPPM(int) error           { return nil }
func (*flappingWidebandDevice) SetBiasTee(bool) error      { return nil }
func (*flappingWidebandDevice) Close() error               { return nil }
func (d *flappingWidebandDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	n := d.calls.Add(1)
	ch := make(chan []complex64)
	go func() {
		defer close(ch)
		if n > d.maxDeaths {
			<-ctx.Done() // stopped flapping: keep the stream healthy
			return
		}
		select {
		case <-time.After(d.healthy): // healed, ran a while, now the reaper dies
		case <-ctx.Done():
		}
	}()
	return ch, nil
}

// TestRunWidebandWithRetry_SurvivesRecoveringFlaps pins the decay fix: a stream
// that dies, self-heals, streams past the healthy-progress window, and dies
// again — repeatedly, more times than the backoff schedule is long — must NOT
// escalate the whole daemon to a fatal. Before the decay (reset only after one
// uninterrupted 60 s Run) `attempt` climbed monotonically across these ~seconds-
// long runs and the daemon killed itself after a handful of deaths even though
// every restart succeeded (the 9-Jul captures). It fails on the pre-fix code
// (fatal recorded, supervisor returns before the device stops flapping) and
// passes with the decay.
func TestRunWidebandWithRetry_SurvivesRecoveringFlaps(t *testing.T) {
	oldBackoffs := ccDecoderRetryBackoffs
	oldProgress := ccDecoderHealthyProgressDuration
	oldHealthy := ccDecoderHealthyRunDuration
	ccDecoderRetryBackoffs = []time.Duration{2 * time.Millisecond, 2 * time.Millisecond}
	ccDecoderHealthyProgressDuration = 15 * time.Millisecond
	ccDecoderHealthyRunDuration = 10 * time.Second // never reached in-test
	defer func() {
		ccDecoderRetryBackoffs = oldBackoffs
		ccDecoderHealthyProgressDuration = oldProgress
		ccDecoderHealthyRunDuration = oldHealthy
	}()

	const maxDeaths = 6 // > len(backoffs): pre-fix this escalates long before here
	dev := &flappingWidebandDevice{healthy: 30 * time.Millisecond, maxDeaths: maxDeaths}

	bus := events.NewBus(8)
	defer bus.Close()
	eng, err := widebandt2.New(widebandt2.Options{
		Log:          slog.New(slog.DiscardHandler),
		Bus:          bus,
		Device:       dev,
		SampleRateHz: 2_400_000,
		CenterFreqHz: 453_500_000,
		Channels:     []widebandt2.ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "wbsys"}},
		Systems:      []trunking.System{{Name: "wbsys", Protocol: trunking.ProtocolDMRTier2}},
	})
	if err != nil {
		t.Fatalf("widebandt2.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := &Daemon{log: slog.New(slog.DiscardHandler), bus: bus, cancel: cancel}
	done := make(chan error, 1)
	go func() { done <- d.runWidebandWithRetry(ctx, eng) }()

	// The supervisor must survive every flap and reach the parked session
	// (call maxDeaths+1). If it escalated instead, it returns early and calls
	// stalls below maxDeaths+1 — the poll times out and the test fails.
	deadline := time.After(4 * time.Second)
	for dev.calls.Load() <= maxDeaths {
		select {
		case err := <-done:
			t.Fatalf("supervisor returned early (%v) instead of surviving flaps; fatal=%v", err, d.takeFatal())
		case <-deadline:
			t.Fatalf("supervisor did not survive flaps: reached only %d/%d StreamIQ calls", dev.calls.Load(), maxDeaths+1)
		case <-time.After(2 * time.Millisecond):
		}
	}
	if f := d.takeFatal(); f != nil {
		t.Fatalf("recordFatal fired on a recovering stream: %v", f)
	}

	// Clean shutdown from the parked state.
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("runWidebandWithRetry on cancel = %v, want nil/Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("supervisor did not return after ctx cancel")
	}
}

// liveWidebandDevice keeps its IQ stream open until ctx cancels.
type liveWidebandDevice struct{}

func (liveWidebandDevice) Info() sdr.Info             { return sdr.Info{Driver: "mock", Serial: "WB-LIVE"} }
func (liveWidebandDevice) SetCenterFreq(uint32) error { return nil }
func (liveWidebandDevice) SetSampleRate(uint32) error { return nil }
func (liveWidebandDevice) SetGain(int) error          { return nil }
func (liveWidebandDevice) SetPPM(int) error           { return nil }
func (liveWidebandDevice) SetBiasTee(bool) error      { return nil }
func (liveWidebandDevice) Close() error               { return nil }
func (liveWidebandDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64)
	go func() {
		defer close(ch)
		<-ctx.Done()
	}()
	return ch, nil
}
