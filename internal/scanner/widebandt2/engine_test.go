package widebandt2

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// t2System / t3System are test helpers that build the trunking.System
// entries the engine needs for Tier II conventional and Tier III
// trunked channels respectively.
func t2System(name string) trunking.System {
	return trunking.System{Name: name, Protocol: trunking.ProtocolDMRTier2}
}

func t3System(name string, ccFreq uint32) trunking.System {
	return trunking.System{Name: name, Protocol: trunking.ProtocolDMR, ControlChannels: []uint32{ccFreq}}
}

// p25Phase1System / p25Phase2System are test helpers for P25 wideband
// channels — both phases run on declared control channels, just like
// DMR Tier III.
func p25Phase1System(name string, ccFreq uint32) trunking.System {
	return trunking.System{Name: name, Protocol: trunking.ProtocolP25, ControlChannels: []uint32{ccFreq}}
}

func p25Phase2System(name string, ccFreq uint32) trunking.System {
	return trunking.System{Name: name, Protocol: trunking.ProtocolP25Phase2, ControlChannels: []uint32{ccFreq}}
}

// mockDevice is a synchronous sdr.Device that emits a caller-supplied
// sequence of IQ chunks, then closes the stream. The test goroutine
// blocks on producing each chunk so the engine's loop is driven
// deterministically.
type mockDevice struct {
	chunks       [][]complex64
	chunkCh      chan []complex64
	streamErr    error
	centerFreqHz atomic.Uint32
	sampleRateHz atomic.Uint32
	startOnce    sync.Once
}

func newMockDevice(chunks [][]complex64) *mockDevice {
	return &mockDevice{chunks: chunks, chunkCh: make(chan []complex64, len(chunks)+1)}
}

func (m *mockDevice) Info() sdr.Info                { return sdr.Info{Driver: "mock", Serial: "MOCK1"} }
func (m *mockDevice) SetCenterFreq(hz uint32) error { m.centerFreqHz.Store(hz); return nil }
func (m *mockDevice) SetSampleRate(hz uint32) error { m.sampleRateHz.Store(hz); return nil }
func (m *mockDevice) SetGain(int) error             { return nil }
func (m *mockDevice) SetPPM(int) error              { return nil }
func (m *mockDevice) SetBiasTee(bool) error         { return nil }
func (m *mockDevice) Close() error                  { return nil }

func (m *mockDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	m.startOnce.Do(func() {
		go func() {
			defer close(m.chunkCh)
			for _, c := range m.chunks {
				select {
				case <-ctx.Done():
					return
				case m.chunkCh <- c:
				}
			}
		}()
	})
	return m.chunkCh, nil
}

func TestEngineNewRejectsMissingDevice(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	if _, err := New(Options{Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
		Channels: []ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "x"}},
		Systems:  []trunking.System{t2System("x")}}); err == nil {
		t.Errorf("expected error for missing Device")
	}
}

func TestEngineNewRejectsMissingBus(t *testing.T) {
	dev := newMockDevice(nil)
	if _, err := New(Options{Device: dev, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
		Channels: []ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "x"}},
		Systems:  []trunking.System{t2System("x")}}); err == nil {
		t.Errorf("expected error for missing Bus")
	}
}

func TestEngineNewRejectsEmptyChannels(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	if _, err := New(Options{Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000}); err == nil {
		t.Errorf("expected error for empty channels")
	}
}

func TestEngineNewRejectsMissingSystems(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	_, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
		Channels: []ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "x"}},
	})
	if err == nil {
		t.Errorf("expected error for missing Systems table")
	}
}

func TestEngineNewRejectsUnknownSystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	_, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
		Channels: []ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "missing"}},
		Systems:  []trunking.System{t2System("other")},
	})
	if err == nil {
		t.Errorf("expected error for channel referencing unknown system")
	}
}

func TestEngineNewRejectsT3ChannelNotInCCList(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	_, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
		// Channel freq doesn't match the system's CC list — engine
		// must reject because a T3 wideband tap MUST sit on a
		// declared control channel.
		Channels: []ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "t3sys"}},
		Systems:  []trunking.System{t3System("t3sys", 453_775_000)},
	})
	if err == nil {
		t.Errorf("expected error for T3 channel not in system.ControlChannels")
	}
}

func TestEngineNewRejectsOutOfBandChannel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	_, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
		Channels: []ChannelConfig{
			{FrequencyHz: 470_000_000, SystemName: "x"}, // > 16 MHz away
		},
		Systems: []trunking.System{t2System("x")},
	})
	if err == nil {
		t.Errorf("expected error for out-of-band channel")
	}
}

func TestEngineStrategyAuto(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	t.Run("small fleet picks ddc", func(t *testing.T) {
		dev := newMockDevice(nil)
		e, err := New(Options{
			Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
			Channels: []ChannelConfig{
				{FrequencyHz: 453_125_000, SystemName: "x"},
				{FrequencyHz: 453_775_000, SystemName: "x"},
			},
			Systems: []trunking.System{t2System("x")},
		})
		if err != nil {
			t.Fatal(err)
		}
		if e.Strategy() != "auto(ddc)" {
			t.Errorf("strategy = %q, want auto(ddc)", e.Strategy())
		}
	})

	t.Run("large fleet picks polyphase", func(t *testing.T) {
		dev := newMockDevice(nil)
		// 7 channels exceeds strategyAutoThreshold, so auto picks
		// the channelizer. Frequencies are 200 kHz apart so they
		// occupy distinct bins (150 kHz bin width at M=16,
		// 2.4 MS/s).
		channels := []ChannelConfig{}
		for i := -3; i <= 3; i++ {
			channels = append(channels, ChannelConfig{
				FrequencyHz: uint32(int64(453_500_000) + int64(i)*200_000),
				SystemName:  "x",
			})
		}
		e, err := New(Options{
			Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
			Channels: channels,
			Systems:  []trunking.System{t2System("x")},
		})
		if err != nil {
			t.Fatal(err)
		}
		if e.Strategy() != "auto(polyphase)" {
			t.Errorf("strategy = %q, want auto(polyphase)", e.Strategy())
		}
	})
}

func TestEngineStrategyExplicit(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	cases := []struct {
		req, want string
	}{
		{"ddc", "ddc"},
		{"polyphase", "polyphase"},
	}
	for _, tc := range cases {
		e, err := New(Options{
			Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
			TunerStrategy: tc.req,
			Channels: []ChannelConfig{
				{FrequencyHz: 453_125_000, SystemName: "x"},
				{FrequencyHz: 453_775_000, SystemName: "x"},
			},
			Systems: []trunking.System{t2System("x")},
		})
		if err != nil {
			t.Fatalf("strategy %q: %v", tc.req, err)
		}
		if e.Strategy() != tc.want {
			t.Errorf("strategy %q: got %q, want %q", tc.req, e.Strategy(), tc.want)
		}
	}
}

func TestEngineMixedT2AndT3Channels(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	// One wideband dongle covers a T2 repeater cluster AND a T3 CC.
	// The engine must instantiate the right state machine per
	// channel based on the referenced system's protocol.
	e, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
		Channels: []ChannelConfig{
			{FrequencyHz: 453_125_000, SystemName: "t2sys"},
			{FrequencyHz: 453_775_000, SystemName: "t3sys"},
		},
		Systems: []trunking.System{
			t2System("t2sys"),
			t3System("t3sys", 453_775_000),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tags := e.ChannelProtocolTags()
	if len(tags) != 2 {
		t.Fatalf("got %d channel tags, want 2", len(tags))
	}
	if tags[453_125_000] != "dmr-tier2" {
		t.Errorf("freq 453_125_000 tag = %q, want dmr-tier2", tags[453_125_000])
	}
	if tags[453_775_000] != "dmr-tier3" {
		t.Errorf("freq 453_775_000 tag = %q, want dmr-tier3", tags[453_775_000])
	}
}

func TestEngineRunSetsCenterFreqAndDrainsStream(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()

	// 4 silence chunks of 4800 wide-band samples each. The engine
	// must consume all of them then exit cleanly when the stream
	// closes.
	const chunkLen = 4800
	chunks := make([][]complex64, 4)
	for i := range chunks {
		chunks[i] = make([]complex64, chunkLen)
	}
	dev := newMockDevice(chunks)

	e, err := New(Options{
		Log:          slog.Default(),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: 2_400_000,
		CenterFreqHz: 453_500_000,
		Channels: []ChannelConfig{
			{FrequencyHz: 453_125_000, SystemName: "regional-t2"},
			{FrequencyHz: 453_775_000, SystemName: "regional-t2"},
		},
		Systems: []trunking.System{t2System("regional-t2")},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := e.Run(ctx); err != nil {
		t.Errorf("Run: %v", err)
	}
	if got := dev.centerFreqHz.Load(); got != 453_500_000 {
		t.Errorf("device centre frequency = %d, want 453_500_000", got)
	}
	if got := len(e.Channels()); got != 2 {
		t.Errorf("Channels() len = %d, want 2", got)
	}
}

func TestEngineP25Phase1Channel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	e, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 851_500_000,
		Channels: []ChannelConfig{{FrequencyHz: 851_037_500, SystemName: "p25-sys"}},
		Systems:  []trunking.System{p25Phase1System("p25-sys", 851_037_500)},
	})
	if err != nil {
		t.Fatal(err)
	}
	tags := e.ChannelProtocolTags()
	if tags[851_037_500] != "p25-phase1" {
		t.Errorf("tag = %q, want p25-phase1", tags[851_037_500])
	}
}

// TestEnginePolyphase625MSBuildsCorrectRate reproduces the issue #550 config
// (USRP X310 at 6.25 MS/s, strategy=polyphase, P25 CC 625 kHz below the wideband
// center) and asserts the engine's bank reports a per-tap rate within 0.1% of
// 48 kHz. Before the fix the polyphase fine-tune emitted 48828 Hz (1.7% fast)
// while receivers were hardcoded to 48000 — the symbol-clock error that left the
// control channel deaf. The receivers are now built from bank.OutputRateHz(), so
// a correct rate here means the receiver is clocked correctly.
func TestEnginePolyphase625MSBuildsCorrectRate(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	const ccHz = 450_125_000
	e, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 6_250_000, CenterFreqHz: 450_750_000,
		TunerStrategy: "polyphase",
		Channels:      []ChannelConfig{{FrequencyHz: ccHz, SystemName: "p25-sys"}},
		Systems:       []trunking.System{p25Phase1System("p25-sys", ccHz)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.Strategy() != "polyphase" {
		t.Fatalf("strategy = %q, want polyphase", e.Strategy())
	}
	rate := e.bank.OutputRateHz()
	if relErr := (rate - 48_000.0) / 48_000.0; relErr > 1e-3 || relErr < -1e-3 {
		t.Errorf("polyphase 6.25 MS/s per-tap rate = %.2f Hz, %.3f%% off 48 kHz (receivers are built from this)",
			rate, relErr*100)
	}
}

func TestEngineP25Phase2Channel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	e, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 851_500_000,
		Channels: []ChannelConfig{{FrequencyHz: 851_006_250, SystemName: "p25p2-sys"}},
		Systems:  []trunking.System{p25Phase2System("p25p2-sys", 851_006_250)},
	})
	if err != nil {
		t.Fatal(err)
	}
	tags := e.ChannelProtocolTags()
	if tags[851_006_250] != "p25-phase2" {
		t.Errorf("tag = %q, want p25-phase2", tags[851_006_250])
	}
}

func TestEngineRejectsP25ChannelNotInCCList(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	// CC declared at 851_037_500 but the wideband channel claims
	// 851_125_000 — must reject so we don't try to decode a TSBK
	// chain on a voice carrier.
	_, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 851_500_000,
		Channels: []ChannelConfig{{FrequencyHz: 851_125_000, SystemName: "p25-sys"}},
		Systems:  []trunking.System{p25Phase1System("p25-sys", 851_037_500)},
	})
	if err == nil {
		t.Fatal("expected error: P25 wideband channel must sit on a declared control channel")
	}
}

func TestEngineMixedDMRAndP25Channels(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	// One wideband dongle hosts a DMR T2 cluster and a P25 Phase 1
	// control channel at the other end of its IQ band. The dispatcher
	// must pick the right state machine per channel based on the
	// referenced system's protocol.
	e, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 852_000_000,
		Channels: []ChannelConfig{
			{FrequencyHz: 851_037_500, SystemName: "p25-sys"},
			{FrequencyHz: 852_775_000, SystemName: "t2-sys"},
		},
		Systems: []trunking.System{
			p25Phase1System("p25-sys", 851_037_500),
			t2System("t2-sys"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tags := e.ChannelProtocolTags()
	if tags[851_037_500] != "p25-phase1" {
		t.Errorf("freq 851_037_500 tag = %q, want p25-phase1", tags[851_037_500])
	}
	if tags[852_775_000] != "dmr-tier2" {
		t.Errorf("freq 852_775_000 tag = %q, want dmr-tier2", tags[852_775_000])
	}
}

func TestEngineRunPropagatesStreamError(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	dev.streamErr = errors.New("device dead")
	e, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 2_400_000, CenterFreqHz: 453_500_000,
		Channels: []ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "x"}},
		Systems:  []trunking.System{t2System("x")},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := e.Run(ctx); err == nil {
		t.Errorf("expected error from StreamIQ propagated, got nil")
	}
}
