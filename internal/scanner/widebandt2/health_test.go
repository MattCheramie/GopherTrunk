package widebandt2

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/iqpower"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// healthRig is one engine with a P25 Phase 1 control channel whose frame
// counters the test advances, driven window by window through the real
// diagnostics loop on an injected clock.
type healthRig struct {
	t        *testing.T
	e        *Engine
	ec       *engineChannel
	now      time.Time
	ok, fail int64
}

func newHealthRig(t *testing.T, proto trunking.Protocol, withFailures bool) *healthRig {
	t.Helper()
	bus := events.NewBus(64)
	t.Cleanup(bus.Close)
	r := &healthRig{t: t, now: time.Unix(1_700_000_000, 0)}
	r.ec = &engineChannel{
		freqHz:   851_012_500,
		sysName:  "metro",
		protoTag: "p25-phase1",
		sysProto: proto,
		decoded:  func() uint64 { return uint64(r.ok) },
	}
	if withFailures {
		r.ec.frameCounts = func() (int64, int64) { return r.ok, r.fail }
	}
	r.e = &Engine{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)), bus: bus,
		channels: []*engineChannel{r.ec}, serial: "wb-1",
		now: func() time.Time { return r.now },
	}
	r.e.maybeLogDiagnostics(r.now)
	return r
}

// window advances the clock one diagnostics window, adds ok/fail frames and
// a window of IQ, and closes the window.
func (r *healthRig) window(ok, fail int64) {
	r.now = r.now.Add(iqpower.Window)
	r.ok += ok
	r.fail += fail
	r.ec.pwr.Add(loudIQ(4096))
	r.e.maybeLogDiagnostics(r.now)
}

func (r *healthRig) health() ChannelHealth {
	r.t.Helper()
	hs := r.e.ChannelHealth()
	if len(hs) != 1 {
		r.t.Fatalf("ChannelHealth returned %d channels, want 1", len(hs))
	}
	return hs[0]
}

func TestChannelHealthListsAChannelBeforeAnyIQ(t *testing.T) {
	r := newHealthRig(t, trunking.ProtocolP25, true)
	h := r.health()
	if h.System != "metro" || h.Protocol != "p25" || h.FreqHz != 851_012_500 || h.Serial != "wb-1" {
		t.Fatalf("identity = %+v", h)
	}
	if h.Decoding || h.HasDecodeHealth || h.HasSignal {
		t.Fatalf("a channel with no IQ yet reports %+v, want nothing measured", h)
	}
}

func TestChannelHealthReportsADecodingChannel(t *testing.T) {
	r := newHealthRig(t, trunking.ProtocolP25, true)
	start := r.now.Add(iqpower.Window)
	for i := 0; i < 5; i++ {
		r.window(40, 0)
	}
	h := r.health()
	if !h.Decoding {
		t.Fatalf("a channel decoding 40 TSBKs a second is not reported decoding: %+v", h)
	}
	if !h.DecodingSince.Equal(start) {
		t.Errorf("DecodingSince = %v, want the first decoding window %v", h.DecodingSince, start)
	}
	if !h.HasDecodeHealth || h.DecodeQuality != "clean" {
		t.Errorf("decode quality = %q (has %v), want clean", h.DecodeQuality, h.HasDecodeHealth)
	}
	if !h.HasSignal || h.SignalDbFS > -1 || h.SignalDbFS < -10 {
		t.Errorf("signal = %.1f dBFS (has %v), want the window's level", h.SignalDbFS, h.HasSignal)
	}
	if h.RecentDecodes != 160 {
		t.Errorf("RecentDecodes = %d, want the 4 windows since the first tally (160)", h.RecentDecodes)
	}
}

func TestChannelHealthBucketsTheErrorRate(t *testing.T) {
	cases := []struct {
		ok, fail int64
		want     string
	}{
		{99, 1, "clean"},
		{97, 3, "marginal"},
		{41, 59, "poor"}, // the #1242 reporter's TSBK error rate
	}
	for _, c := range cases {
		r := newHealthRig(t, trunking.ProtocolP25, true)
		for i := 0; i < 3; i++ {
			r.window(c.ok, c.fail)
		}
		if h := r.health(); h.DecodeQuality != c.want || !h.HasDecodeHealth {
			t.Errorf("%d ok / %d failed per window: quality %q (has %v), want %q",
				c.ok, c.fail, h.DecodeQuality, h.HasDecodeHealth, c.want)
		}
	}
}

// TestChannelHealthRateFollowsTheChannel pins the rolling span: a channel
// that failed badly and then recovered reads clean once the bad windows have
// left the span, where a lifetime rate would read poor for good.
func TestChannelHealthRateFollowsTheChannel(t *testing.T) {
	r := newHealthRig(t, trunking.ProtocolP25, true)
	for i := 0; i < 20; i++ {
		r.window(10, 30)
	}
	if h := r.health(); h.DecodeQuality != "poor" {
		t.Fatalf("after 20 windows at 75 %% failed: %q, want poor", h.DecodeQuality)
	}
	for i := 0; i < healthSpanWindows; i++ {
		r.window(40, 0)
	}
	if h := r.health(); h.DecodeQuality != "clean" {
		t.Fatalf("after a full span of clean windows: %q, want clean", h.DecodeQuality)
	}
}

func TestChannelHealthNeedsEnoughAttempts(t *testing.T) {
	r := newHealthRig(t, trunking.ProtocolP25, true)
	r.window(5, 5)
	r.window(5, 4)
	h := r.health()
	if !h.Decoding {
		t.Fatalf("not decoding: %+v", h)
	}
	if h.HasDecodeHealth {
		t.Fatalf("bucketed %q from %d attempts, want none below %d", h.DecodeQuality, 9, healthMinAttempts)
	}
}

// TestChannelHealthWithoutFailureCountsHasNoQuality: a protocol that counts
// only its successes (DMR Tier III CSBKs, Phase 2 MAC PDUs) is reported
// decoding, with no clean/marginal/poor it could not have measured.
func TestChannelHealthWithoutFailureCountsHasNoQuality(t *testing.T) {
	r := newHealthRig(t, trunking.ProtocolDMR, false)
	for i := 0; i < 5; i++ {
		r.window(30, 0)
	}
	h := r.health()
	if !h.Decoding || h.Protocol != "dmr" {
		t.Fatalf("health = %+v, want a decoding dmr channel", h)
	}
	if h.HasDecodeHealth {
		t.Fatalf("quality %q reported without a failure count", h.DecodeQuality)
	}
}

// TestChannelHealthGoesStaleWhenTheStreamStops: the pump refreshes the
// record only while IQ arrives. A stopped stream must stop reading as a
// decoding channel with a level, not freeze its last good reading.
func TestChannelHealthGoesStaleWhenTheStreamStops(t *testing.T) {
	r := newHealthRig(t, trunking.ProtocolP25, true)
	for i := 0; i < 5; i++ {
		r.window(40, 0)
	}
	r.now = r.now.Add(healthSignalFresh + time.Second)
	if h := r.health(); h.HasSignal {
		t.Errorf("level still reported %v after the stream stopped", healthSignalFresh+time.Second)
	}
	r.now = r.now.Add(lowPowerDecodeGrace)
	if h := r.health(); h.Decoding || h.HasDecodeHealth {
		t.Errorf("still decoding (quality %q) %v after the last frame", h.DecodeQuality, lowPowerDecodeGrace)
	}
}

// TestChannelHealthStartsANewStretchAfterADrought: DecodingSince marks the
// current stretch, so a channel that went quiet past the grace and came back
// reports when it came back.
func TestChannelHealthStartsANewStretchAfterADrought(t *testing.T) {
	r := newHealthRig(t, trunking.ProtocolP25, true)
	r.window(40, 0)
	for i := 0; i < int(lowPowerDecodeGrace/iqpower.Window)+2; i++ {
		r.window(0, 0)
	}
	if h := r.health(); h.Decoding {
		t.Fatalf("still decoding after a %v drought", lowPowerDecodeGrace)
	}
	r.window(40, 0)
	back := r.now
	r.window(40, 0)
	if h := r.health(); !h.DecodingSince.Equal(back) {
		t.Fatalf("DecodingSince = %v, want the window decoding resumed (%v)", h.DecodingSince, back)
	}
}

func TestChannelHealthLeavesOutConventionalChannels(t *testing.T) {
	r := newHealthRig(t, trunking.ProtocolP25, true)
	conv := &engineChannel{freqHz: 154_130_000, sysName: "conventional", protoTag: conventionalProtoTag, conventional: true}
	r.e.channels = append(r.e.channels, conv)
	conv.pwr.Add(loudIQ(4096))
	r.window(40, 0)
	hs := r.e.ChannelHealth()
	if len(hs) != 1 || hs[0].FreqHz != 851_012_500 {
		t.Fatalf("ChannelHealth = %+v, want only the control channel", hs)
	}
}

// TestChannelHealthEndToEndFromSynthesizedIQ runs the real pump: synthesized
// DMR Tier II IQ through the mock device, DDC tap, receiver and channel
// state machine, on a clock that advances one chunk (100 ms) per pump step.
// The channel the engine decoded is the one ChannelHealth reports decoding,
// with the level of its own tap.
func TestChannelHealthEndToEndFromSynthesizedIQ(t *testing.T) {
	const (
		rateHz       = 48_000.0
		centerHz     = 460_000_000
		chunkSamples = 4800 // 100 ms
	)
	dibits := buildT2VoiceLCHeaderDibits(200, 0x7, 0x123, 0x456789)
	iq := demod.ModulateC4FM(dibits, 10, 8, 0.20, rateHz, dmrDeviationHz)
	dev := newMockDevice(chunkComplex(iq, chunkSamples))

	bus := events.NewBus(64)
	defer bus.Close()
	var mu sync.Mutex
	clock := time.Unix(1_700_000_000, 0)
	now := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		clock = clock.Add(time.Duration(chunkSamples / rateHz * float64(time.Second)))
		return clock
	}
	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		Device:       dev,
		Bus:          bus,
		Serial:       "wb-e2e",
		SampleRateHz: uint32(rateHz),
		CenterFreqHz: centerHz,
		Channels:     []ChannelConfig{{FrequencyHz: centerHz, SystemName: "regional-t2"}},
		Systems:      []trunking.System{t2System("regional-t2")},
		Now:          now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hs := e.ChannelHealth(); len(hs) != 1 || hs[0].Decoding {
		t.Fatalf("before any IQ: %+v, want one channel not decoding", hs)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = e.Run(ctx) // returns when the mock stream closes

	hs := e.ChannelHealth()
	if len(hs) != 1 {
		t.Fatalf("ChannelHealth returned %d channels, want 1", len(hs))
	}
	h := hs[0]
	if !h.Decoding || h.RecentDecodes == 0 {
		t.Fatalf("the channel the engine decoded is not reported decoding: %+v", h)
	}
	if h.System != "regional-t2" || h.Protocol != "dmr-tier2" || h.Serial != "wb-e2e" || !h.CampsWhenIdle {
		t.Errorf("identity = %+v", h)
	}
	if !h.HasSignal {
		t.Errorf("no level reported for a channel that just carried IQ")
	}
}
