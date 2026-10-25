package main

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// reasonChainEngine is voiceChainEngine that also records how each call
// ended and when.
type reasonChainEngine struct {
	*voiceChainEngine
	mu      sync.Mutex
	reasons []trunking.EndReason
	endedAt time.Time
}

func (e *reasonChainEngine) EndSyntheticCall(serial string, r trunking.EndReason) bool {
	e.mu.Lock()
	e.reasons = append(e.reasons, r)
	e.endedAt = time.Now()
	e.mu.Unlock()
	return e.voiceChainEngine.EndSyntheticCall(serial, r)
}

// TestWidebandConvCallEndsWhenStreamStops is issue #1288's test G for a
// wideband conventional channel: the dongle's stream stops in the middle of a
// transmission (USB loss, a wedged pump). The call must end on its own,
// within the scanner's stream-stall bound plus hangtime, instead of staying
// open as a phantom with no audio; and it must end exactly once.
func TestWidebandConvCallEndsWhenStreamStops(t *testing.T) {
	const (
		rate     = 960_000.0
		centerHz = 460_500_000
		chunk    = 9_600 // 10 ms
		seconds  = 1.5   // the stream ends here, mid-transmission
	)
	chHz := uint32(centerHz - 200_000)
	// Keyed from 0.5 s to well past the end of the stream.
	iq := synthToneGated(rate, seconds, -200_000, 1000, 0, 0.5, 10)
	var chunks [][]complex64
	for i := 0; i+chunk <= len(iq); i += chunk {
		chunks = append(chunks, iq[i:i+chunk])
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := events.NewBus(64)
	defer bus.Close()
	eng, err := widebandt2.New(widebandt2.Options{
		Log: log, Bus: bus,
		Device:       &pacedWidebandDevice{chunks: chunks, period: 10 * time.Millisecond},
		SampleRateHz: uint32(rate), CenterFreqHz: centerHz,
		Channels: widebandConvTapChannels([]config.ConvChannelConfig{{FrequencyHz: chHz, Mode: "nfm"}}),
		Serial:   "WB1",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := widebandt2.NewChannelVoiceSource(func() *widebandt2.Engine { return eng }, chHz, wbConvSerial("WB1", chHz))
	ch := config.ConvChannelConfig{Label: "A", FrequencyHz: chHz, Mode: "nfm", SquelchDbFS: -30, HangtimeMs: 500}
	chain := &reasonChainEngine{voiceChainEngine: &voiceChainEngine{src: src}}
	cs, err := newWidebandConvScanner(src, convChannelFromConfig(ch, config.WidebandConvPositionalBase), chain, discardRecorder{}, nil, log)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = cs.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	streamStart := time.Now()
	go func() { _ = eng.Run(ctx) }()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		chain.mu.Lock()
		n := len(chain.reasons)
		chain.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(4 * time.Second) // a second (wrong) end or a phantom re-open would show here
	cancel()

	chain.voiceChainEngine.mu.Lock()
	calls := chain.calls
	chain.voiceChainEngine.mu.Unlock()
	chain.mu.Lock()
	defer chain.mu.Unlock()
	if calls != 1 {
		t.Fatalf("calls = %d, want the one call the transmission opened", calls)
	}
	if len(chain.reasons) != 1 {
		t.Fatalf("call ended %d times %v, want exactly once", len(chain.reasons), chain.reasons)
	}
	// Stream ends ~1.5 s in; stall bound 3 s; hangtime 0.5 s; slack for a
	// loaded runner.
	took := chain.endedAt.Sub(streamStart)
	t.Logf("call ended %.2f s after the stream started (stream stopped at %.1f s), reason %v", took.Seconds(), seconds, chain.reasons[0])
	if took > 8*time.Second {
		t.Errorf("call ended %.1f s after the stream started; a stopped stream must end it within the stall bound", took.Seconds())
	}
}
