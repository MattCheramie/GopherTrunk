package widebandt2

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/iqpower"
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
	holdOpen     bool // keep the stream open until ctx cancels (models a live stream)
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
			// holdOpen models a live SDR whose stream only ends on
			// ctx-cancel (a clean stop) rather than closing on its own —
			// the case Run must report as nil, not a stream death.
			if m.holdOpen {
				<-ctx.Done()
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

// warnCapture records WARN-level log messages so tests can assert on the
// operator-facing advisories New emits.
type warnCapture struct{ msgs []string }

func (h *warnCapture) Enabled(context.Context, slog.Level) bool { return true }
func (h *warnCapture) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *warnCapture) WithGroup(string) slog.Handler            { return h }
func (h *warnCapture) Handle(_ context.Context, r slog.Record) error {
	if r.Level == slog.LevelWarn {
		h.msgs = append(h.msgs, r.Message)
	}
	return nil
}
func (h *warnCapture) sawAdvisory() bool {
	for _, m := range h.msgs {
		if strings.Contains(m, "oversampled for the channel plan") {
			return true
		}
	}
	return false
}

// TestEngineNewAdvisesOversampledCapture is the Fix-D regression: a plan whose
// carriers span far less than half the captured band should draw a startup WARN
// suggesting a lower sdr.sample_rate (the durable cure for the host-can't-keep-up
// overruns), and a right-sized capture should stay silent. Mirrors the reported
// config: 71 DMR carriers spanning ±1.95 MHz captured at 10 MS/s.
func TestEngineNewAdvisesOversampledCapture(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	// Two outer carriers of the reported plan span ≈ ±1.9 MHz of centre, so the
	// span needs ≈ 4.25 MS/s; everything above ~6.4 MS/s is flagged.
	const center = 441_700_000
	channels := []ChannelConfig{
		{FrequencyHz: 439_787_500, SystemName: "x"}, // −1.9125 MHz
		{FrequencyHz: 443_600_000, SystemName: "x"}, // +1.9000 MHz
	}

	t.Run("oversampled warns", func(t *testing.T) {
		h := &warnCapture{}
		_, err := New(Options{
			Log: slog.New(h), Device: newMockDevice(nil), Bus: bus,
			SampleRateHz: 10_000_000, CenterFreqHz: center, TunerStrategy: "polyphase",
			Channels: channels, Systems: []trunking.System{t2System("x")},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !h.sawAdvisory() {
			t.Errorf("no oversampled-capture advisory at 10 MS/s for a ±1.9 MHz plan; got WARNs %q", h.msgs)
		}
	})

	t.Run("right-sized is silent", func(t *testing.T) {
		h := &warnCapture{}
		_, err := New(Options{
			Log: slog.New(h), Device: newMockDevice(nil), Bus: bus,
			SampleRateHz: 5_000_000, CenterFreqHz: center, TunerStrategy: "polyphase",
			Channels: channels, Systems: []trunking.System{t2System("x")},
		})
		if err != nil {
			t.Fatal(err)
		}
		if h.sawAdvisory() {
			t.Errorf("advised lowering rate for a right-sized 5 MS/s capture; got WARNs %q", h.msgs)
		}
	})
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

// TestChannelizerBinsFor locks the sample-rate-scaled bin count: 16 bins at
// the legacy 2.4 MS/s rate (≈150 kHz/bin, unchanged), growing with the rate so
// the per-bin width stays near channelizerBinWidthHz instead of ballooning to
// 625 kHz at 10 MS/s and collapsing adjacent carriers (issue #764).
func TestChannelizerBinsFor(t *testing.T) {
	cases := []struct {
		rate uint32
		want int
	}{
		{2_400_000, 16},
		{2_500_000, 16},
		{5_000_000, 32},
		{10_000_000, 64},
		{20_000_000, 128},
	}
	for _, c := range cases {
		if got := channelizerBinsFor(c.rate); got != c.want {
			t.Errorf("channelizerBinsFor(%d) = %d, want %d (bin width %.0f kHz)",
				c.rate, got, c.want, float64(c.rate)/float64(got)/1e3)
		}
	}
}

// TestEngineHighRateChannelizerDistinctBins is the channelizer half of issue
// #764: a 10 MS/s polyphase survey with carriers 200 kHz apart must place each
// in a distinct bin. With the old fixed 16 bins the bin width is 625 kHz and
// the carriers collide (AddTap → ErrBinAlreadyClaimed, engine construction
// fails); with the scaled bin count they land in separate bins.
func TestEngineHighRateChannelizerDistinctBins(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	channels := []ChannelConfig{}
	for i := -3; i <= 3; i++ {
		channels = append(channels, ChannelConfig{
			FrequencyHz: uint32(int64(453_500_000) + int64(i)*200_000),
			SystemName:  "x",
		})
	}
	e, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 10_000_000, CenterFreqHz: 453_500_000,
		TunerStrategy: "polyphase",
		Channels:      channels,
		Systems:       []trunking.System{t2System("x")},
	})
	if err != nil {
		t.Fatalf("10 MS/s polyphase with 200 kHz spacing should build distinct bins, got: %v", err)
	}
	if e.Strategy() != "polyphase" {
		t.Errorf("strategy = %q, want polyphase", e.Strategy())
	}
}

// TestEngineHighRateReporterTapsInBand is the DDC half of issue #764: the
// reporter's four closely-spaced P25 control taps at a 10 MS/s capture must all
// be accepted by the auto-selected DDC bank (they sit inside the post-decimation
// usable band), rather than being rejected or routed to a channelizer that
// cannot separate sub-bin spacing.
func TestEngineHighRateReporterTapsInBand(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := newMockDevice(nil)
	const center = 420_900_000
	offsets := []int64{-887_500, -862_500, -812_500, +937_500}
	channels := make([]ChannelConfig, 0, len(offsets))
	systems := make([]trunking.System, 0, len(offsets))
	for i, off := range offsets {
		freq := uint32(int64(center) + off)
		name := "mmr" + string(rune('a'+i))
		channels = append(channels, ChannelConfig{FrequencyHz: freq, SystemName: name})
		systems = append(systems, p25Phase1System(name, freq))
	}
	e, err := New(Options{
		Device: dev, Bus: bus, SampleRateHz: 10_000_000, CenterFreqHz: center,
		Channels: channels,
		Systems:  systems,
	})
	if err != nil {
		t.Fatalf("reporter's 4 taps at 10 MS/s should construct, got: %v", err)
	}
	if e.Strategy() != "auto(ddc)" {
		t.Errorf("strategy = %q, want auto(ddc) (closely-spaced taps need per-tap DDC)", e.Strategy())
	}
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

// TestEngineHostsDenseDMRPlan is the regression for the 70-channel DMR stress
// test that crashed daemon init ("two tap offsets fall in the same channelizer
// bin"). The reporter's plan packs 71 DMR repeaters across ±1.9 MHz of a
// 10 MS/s capture on a 12.5 kHz grid that never aligns to the channelizer bins.
// It must now build on the polyphase channelizer via bin sharing — both when
// requested explicitly and via auto (which keeps high tap counts on the
// channelizer because a per-tap DDC at 71 taps is far too heavy to stay
// real-time). Bin sharing means no AddTap error; every tap is hosted.
func TestEngineHostsDenseDMRPlan(t *testing.T) {
	const center = 441_700_000
	freqs := []uint32{
		439787500, 440012500, 440062500, 440087500, 440112500, 440187500, 440262500,
		440312500, 440362500, 440387500, 440437500, 440462500, 440487500, 440512500,
		440562500, 440587500, 440712500, 440737500, 440762500, 440787500, 440862500,
		440912500, 440937500, 441025000, 441312500, 441337500, 441362500, 441387500,
		441412500, 441437500, 441462500, 441487500, 441512500, 441537500, 441562500,
		441587500, 441612500, 441637500, 441662500, 441687500, 441787500, 441812500,
		441821500, 441837500, 441862500, 441887500, 441912500, 442287500, 442337500,
		442387500, 442412500, 442432500, 442437500, 442512500, 442562500, 442837500,
		442887500, 442937500, 442962500, 443037500, 443137500, 443162500, 443187500,
		443237500, 443412500, 443437500, 443462500, 443487500, 443587500, 443600000,
		443612500,
	}
	channels := make([]ChannelConfig, len(freqs))
	for i, f := range freqs {
		channels[i] = ChannelConfig{FrequencyHz: f, SystemName: "regional-dmr-hi"}
	}
	systems := []trunking.System{t2System("regional-dmr-hi")}

	cases := []struct {
		strategy, want string
	}{
		{"polyphase", "polyphase"},  // honoured via bin sharing — no crash
		{"auto", "auto(polyphase)"}, // high count stays on the channelizer
		{"", "auto(polyphase)"},     // default is auto
	}
	for _, tc := range cases {
		t.Run("strategy="+tc.strategy, func(t *testing.T) {
			bus := events.NewBus(8)
			defer bus.Close()
			e, err := New(Options{
				Device: newMockDevice(nil), Bus: bus,
				SampleRateHz: 10_000_000, CenterFreqHz: center,
				TunerStrategy: tc.strategy,
				Channels:      channels,
				Systems:       systems,
			})
			if err != nil {
				t.Fatalf("dense 71-DMR plan (%q) should build, got: %v", tc.strategy, err)
			}
			if e.Strategy() != tc.want {
				t.Errorf("strategy = %q, want %q", e.Strategy(), tc.want)
			}
			if len(e.channels) != len(freqs) {
				t.Errorf("hosted %d channels, want %d", len(e.channels), len(freqs))
			}
		})
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
	if err := e.Run(ctx); err != nil && !errors.Is(err, ErrIQStreamClosed) {
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

// countingIQObserver counts RecordWidebandInputPowerDbFS calls so a test can
// tell whether the Run pump processed any chunks (it records once per
// diagnostics window) or merely drained them while suspended.
type countingIQObserver struct{ wbPower atomic.Int64 }

func (c *countingIQObserver) RecordIQPowerDbFS(string, float64) {}
func (c *countingIQObserver) ClearIQPowerDbFS(string)           {}
func (c *countingIQObserver) RecordWidebandInputPowerDbFS(string, float64) {
	c.wbPower.Add(1)
}
func (c *countingIQObserver) RecordWidebandInputClipRatio(string, float64) {}
func (c *countingIQObserver) ClearWidebandInput(string)                    {}

// engineWithChunks builds a P25 engine fed `n` silence chunks, with a now()
// that jumps a full diagnostics window per chunk so every PROCESSED chunk
// flushes maybeLogDiagnostics (and thus records wideband power).
func engineWithChunks(t *testing.T, obs IQPowerObserver, serial string, n int) (*Engine, *mockDevice) {
	t.Helper()
	bus := events.NewBus(64)
	t.Cleanup(bus.Close)
	const chunkLen = 4800
	chunks := make([][]complex64, n)
	for i := range chunks {
		chunks[i] = make([]complex64, chunkLen)
	}
	dev := newMockDevice(chunks)
	var tick atomic.Int64
	e, err := New(Options{
		Log:          slog.Default(),
		Device:       dev,
		Bus:          bus,
		Metrics:      obs,
		Serial:       serial,
		SampleRateHz: 2_400_000,
		CenterFreqHz: 453_500_000,
		Channels:     []ChannelConfig{{FrequencyHz: 453_125_000, SystemName: "wbsys"}},
		Systems:      []trunking.System{t2System("wbsys")},
		Now: func() time.Time {
			// Each call advances two windows so a processed chunk always
			// crosses the diagnostics-window boundary.
			n := tick.Add(int64(2 * iqpower.Window))
			return time.Unix(0, 0).Add(time.Duration(n))
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return e, dev
}

// TestEngineRunReturnsStreamClosedOnUnexpectedClose is the self-heal
// regression: when the SDR's IQ channel closes while the engine was NOT
// asked to stop (a USB reaper death or the macOS stall watchdog aborting a
// frozen endpoint), Run must return ErrIQStreamClosed so the daemon's
// runWidebandWithRetry supervisor reacquires and restarts. A plain nil
// (the pre-fix behaviour) would silently stop decoding with no recovery.
func TestEngineRunReturnsStreamClosedOnUnexpectedClose(t *testing.T) {
	e, _ := engineWithChunks(t, &countingIQObserver{}, "WB-DEAD", 3)
	// A long-lived ctx: the mock closes the stream after its 3 chunks
	// while the ctx is still active, modelling a stream death.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := e.Run(ctx)
	if !errors.Is(err, ErrIQStreamClosed) {
		t.Fatalf("Run on unexpected stream close = %v, want ErrIQStreamClosed", err)
	}
}

// TestEngineRunReturnsNilOnContextCancel is the paired control: a clean
// shutdown (ctx cancelled) must NOT be reported as a stream death, so the
// supervisor lets the engine stop instead of thrashing on reacquire.
func TestEngineRunReturnsNilOnContextCancel(t *testing.T) {
	// A stream that stays open (no chunks queued, channel never closed by
	// the mock until ctx cancels its producer) so Run blocks until cancel.
	e, dev := engineWithChunks(t, &countingIQObserver{}, "WB-STOP", 0)
	dev.holdOpen = true
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- e.Run(ctx) }()
	// Give Run a moment to enter its pump loop, then cancel.
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run on ctx-cancel = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after ctx cancel")
	}
}

// TestEngineSuspendDrainsWithoutDecoding: a suspended engine consumes its IQ
// stream (so the SDR/broker never blocks) but processes nothing — no per-window
// diagnostics are recorded. Resume restores the dongle's centre frequency.
func TestEngineSuspendDrainsWithoutDecoding(t *testing.T) {
	obs := &countingIQObserver{}
	e, dev := engineWithChunks(t, obs, "WB-SUSP", 6)

	if got := e.Serial(); got != "WB-SUSP" {
		t.Errorf("Serial() = %q, want WB-SUSP", got)
	}

	// Suspend before Run consumes any chunk; the pump must drain all 6 and exit
	// cleanly on stream close without recording a single diagnostics window.
	e.Suspend()
	// A borrowing hunt would have retuned the SDR; emulate that so Resume's
	// retune-back is observable.
	_ = dev.SetCenterFreq(440_000_000)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := e.Run(ctx); err != nil && !errors.Is(err, ErrIQStreamClosed) {
		t.Fatalf("Run: %v", err)
	}
	if got := obs.wbPower.Load(); got != 0 {
		t.Errorf("suspended engine recorded %d diagnostics windows; want 0 (must not decode)", got)
	}

	// Resume must reprogram the dongle back to the engine's centre frequency.
	e.Resume()
	if got := dev.centerFreqHz.Load(); got != 453_500_000 {
		t.Errorf("after Resume, centre freq = %d, want 453_500_000 (retune-back)", got)
	}
}

// TestEngineProcessesWhenNotSuspended is the control: the same fixture, not
// suspended, DOES record diagnostics windows — proving the suspend guard, not
// the test setup, is what suppresses processing above.
func TestEngineProcessesWhenNotSuspended(t *testing.T) {
	obs := &countingIQObserver{}
	e, _ := engineWithChunks(t, obs, "WB-RUN", 6)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := e.Run(ctx); err != nil && !errors.Is(err, ErrIQStreamClosed) {
		t.Fatalf("Run: %v", err)
	}
	if got := obs.wbPower.Load(); got == 0 {
		t.Error("running engine recorded 0 diagnostics windows; want > 0 (fixture should process)")
	}
}
