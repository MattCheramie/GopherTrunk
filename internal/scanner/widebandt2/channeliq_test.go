package widebandt2

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// testEngineWithChannel builds a bare engine hosting one channel on freqHz at
// rateHz (no bank, no device) — enough to exercise the per-channel IQ
// fan-out and the same-carrier voice source in isolation.
func testEngineWithChannel(freqHz uint32, rateHz float64) *Engine {
	ec := &engineChannel{freqHz: freqHz, sysName: "T", protoTag: "tetra", rateHz: rateHz,
		fan: newChannelFanout(quietLog(), "wb-00", freqHz, rateHz)}
	return &Engine{channels: []*engineChannel{ec}, serial: "wb-00", log: quietLog()}
}

// TestChannelVoiceSourceSameCarrierGate mirrors ccdecoder's
// TestCCVoiceSourceSameCarrierGate for the wideband twin: the source binds
// only its own channel's carrier on a live engine, reports the tap's ACTUAL
// rate, is inert before the engine exists, and streams copies of what the
// channel's tap broadcasts until its context ends.
func TestChannelVoiceSourceSameCarrierGate(t *testing.T) {
	const cc = uint32(467_912_500)
	const rate = 144_230.7 // an ActualRateFor-style non-nominal tap rate
	var eng *Engine
	src := NewChannelVoiceSource(func() *Engine { return eng }, cc, "wb:wb-00:same-carrier:467912500:1")

	// Inert before the engine is built.
	if src.CanTune(cc) || src.SampleRateHz() != 0 || src.CarrierKey() != "" {
		t.Fatal("source must be inert with no engine")
	}
	if _, err := src.StreamIQ(context.Background()); err == nil {
		t.Fatal("StreamIQ must fail with no engine")
	}

	eng = testEngineWithChannel(cc, rate)
	if src.Serial() != "wb:wb-00:same-carrier:467912500:1" {
		t.Errorf("Serial = %q", src.Serial())
	}
	if !src.CanTune(cc) {
		t.Error("CanTune(own carrier) = false, want true")
	}
	if src.CanTune(cc + 12_500) {
		t.Error("CanTune(other carrier) = true, want false")
	}
	if got := src.SampleRateExactHz(); got != rate {
		t.Errorf("SampleRateExactHz = %v, want the tap's actual rate %v", got, rate)
	}
	if got := src.SampleRateHz(); got != 144_231 {
		t.Errorf("SampleRateHz = %d, want 144231 (rounded)", got)
	}
	if src.CarrierKey() == "" {
		t.Error("CarrierKey empty on a live engine")
	}
	other := NewChannelVoiceSource(func() *Engine { return eng }, cc+25_000, "x")
	if other.CarrierKey() == src.CarrierKey() {
		t.Error("two channels on one engine must not share a CarrierKey (each needs its own demux)")
	}
	if err := src.SetCenterFreq(cc); err != nil {
		t.Errorf("SetCenterFreq: %v", err)
	}

	// A suspended engine (hunt borrowed the SDR) emits nothing — don't bind.
	eng.suspended.Store(true)
	if src.CanTune(cc) {
		t.Error("CanTune must be false while the engine is suspended")
	}
	eng.suspended.Store(false)

	// Stream: chunks broadcast on the channel's fan-out arrive as copies.
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := src.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	chunk := []complex64{1, 2, 3}
	eng.channels[0].fan.broadcast(chunk)
	chunk[0] = 99 // the bank reuses its slice; the subscriber must hold a copy
	select {
	case got := <-ch:
		if len(got) != 3 || got[0] != 1 {
			t.Errorf("subscriber got %v, want a copy of [1 2 3]", got)
		}
	case <-time.After(time.Second):
		t.Fatal("no chunk delivered")
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected the IQ channel to close after ctx cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("IQ channel did not close after ctx cancel")
	}
	// Broadcasting with no subscribers is a no-op, never a block.
	eng.channels[0].fan.broadcast(chunk)
}

// TestChannelFanoutDropsWhenLagging pins the pump-protection contract: a
// subscriber that stops draining loses chunks (counted) rather than stalling
// the wideband pump that feeds every channel on the dongle.
func TestChannelFanoutDropsWhenLagging(t *testing.T) {
	f := newChannelFanout(quietLog(), "wb-00", 1, 0) // rate 0 ⇒ minimum depth
	ch, unsub := f.subscribe()
	for i := 0; i < minChannelTapChunks+5; i++ {
		f.broadcast([]complex64{complex64(complex(float32(i), 0))})
	}
	if len(ch) != minChannelTapChunks {
		t.Fatalf("buffered %d chunks, want the depth %d", len(ch), minChannelTapChunks)
	}
	if drops := unsub(); drops != 5 {
		t.Errorf("dropped = %d, want 5", drops)
	}
	if drops := unsub(); drops != 5 {
		t.Errorf("second unsubscribe reported %d, want the same final total", drops)
	}
}

// TestEngineChannelIQAccessors covers the freq-keyed lookups.
func TestEngineChannelIQAccessors(t *testing.T) {
	eng := testEngineWithChannel(460_000_000, 48_000)
	if got := eng.ChannelRateHz(460_000_000); got != 48_000 {
		t.Errorf("ChannelRateHz = %v", got)
	}
	if got := eng.ChannelRateHz(1); got != 0 {
		t.Errorf("ChannelRateHz(unknown) = %v, want 0", got)
	}
	if _, _, ok := eng.SubscribeChannelIQ(1); ok {
		t.Error("SubscribeChannelIQ(unknown) ok, want false")
	}
	ch, unsub, ok := eng.SubscribeChannelIQ(460_000_000)
	if !ok || ch == nil {
		t.Fatal("SubscribeChannelIQ(known) failed")
	}
	unsub()
}
