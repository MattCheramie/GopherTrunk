package main

import (
	"context"
	"errors"
	"log/slog"
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
