package ccdecoder

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// TestVoiceFanoutDeliversCopies verifies subscribers get their own copy of each
// broadcast chunk and that a broadcast with no subscribers is a no-op.
func TestVoiceFanoutDeliversCopies(t *testing.T) {
	f := newVoiceFanout(nil)
	f.broadcast([]complex64{1, 2, 3}) // no subscribers: must not panic/block

	ch, unsub := f.subscribe()
	buf := []complex64{complex(1, 0), complex(2, 0)}
	f.broadcast(buf)
	buf[0] = complex(9, 0) // mutate the reused buffer after broadcast

	select {
	case got := <-ch:
		if len(got) != 2 || got[0] != complex(1, 0) {
			t.Fatalf("subscriber got %v, want an independent copy [1 2]", got)
		}
	case <-time.After(time.Second):
		t.Fatal("no chunk delivered")
	}

	unsub()
	unsub() // idempotent
	if _, ok := <-ch; ok {
		t.Fatal("channel not closed after unsubscribe")
	}
}

// TestVoiceFanoutDropsWhenFull proves a lagging subscriber can't stall the
// broadcaster (drop-on-full), protecting the decode hot path, and that the
// dropped IQ is counted and surfaced as a warning at unsubscribe (issue #402
// starvation diagnostic) rather than lost silently.
func TestVoiceFanoutDropsWhenFull(t *testing.T) {
	var logs bytes.Buffer
	f := newVoiceFanout(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})))
	_, unsub := f.subscribe()
	for i := 0; i < 1000; i++ {
		f.broadcast([]complex64{complex(float32(i), 0)}) // never blocks despite full buffer
	}
	unsub() // call-end: drops must be reported

	out := logs.String()
	if !strings.Contains(out, "same-carrier voice tap dropped IQ") {
		t.Fatalf("expected a dropped-IQ warning at unsubscribe; log was:\n%s", out)
	}
	// The 64-chunk buffer soaks the first 64; the rest (≥ 900) are counted drops.
	if !strings.Contains(out, "dropped_chunks=") {
		t.Errorf("warning missing dropped_chunks count; log: %s", out)
	}
}

// TestVoiceFanoutNoDropWarningWhenDrained verifies a subscriber that keeps up
// logs nothing at unsubscribe (no false starvation warnings).
func TestVoiceFanoutNoDropWarningWhenDrained(t *testing.T) {
	var logs bytes.Buffer
	f := newVoiceFanout(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})))
	ch, unsub := f.subscribe()
	done := make(chan struct{})
	go func() {
		for range ch { // drain promptly so nothing drops
		}
		close(done)
	}()
	for i := 0; i < 200; i++ {
		f.broadcast([]complex64{complex(float32(i), 0)})
		time.Sleep(time.Millisecond)
	}
	unsub()
	<-done
	if strings.Contains(logs.String(), "dropped IQ") {
		t.Fatalf("unexpected drop warning for a keeping-up subscriber:\n%s", logs.String())
	}
}

// TestCCVoiceSourceSameCarrierGate checks the tap binds only the CC's current
// carrier and streams until the context ends.
func TestCCVoiceSourceSameCarrierGate(t *testing.T) {
	d := &Decoder{voiceFan: newVoiceFanout(nil), activeFreqHz: 467_913_000, pipelineRateHz: 144_000}
	src := NewCCVoiceSource(func() *Decoder { return d }, "cc:SDR1")

	// A nil decoder (before startup / after teardown) stays inert.
	if NewCCVoiceSource(func() *Decoder { return nil }, "x").CanTune(467_913_000) {
		t.Error("nil-decoder source must not bind")
	}

	if src.Serial() != "cc:SDR1" {
		t.Errorf("serial = %q", src.Serial())
	}
	if src.SampleRateHz() != 144_000 || src.SampleRateExactHz() != 144_000 {
		t.Errorf("rate = %d / %g, want 144000", src.SampleRateHz(), src.SampleRateExactHz())
	}
	if !src.CanTune(467_913_000) {
		t.Error("should tune the CC's own carrier")
	}
	if src.CanTune(467_938_000) {
		t.Error("must not tune an off-carrier grant")
	}
	if err := src.SetCenterFreq(1); err != nil {
		t.Errorf("SetCenterFreq should be a no-op, got %v", err)
	}

	// Idle CC (0) never binds.
	d.activeFreqHz = 0
	if src.CanTune(467_913_000) {
		t.Error("an idle CC must not bind")
	}
	d.activeFreqHz = 467_913_000

	ctx, cancel := context.WithCancel(context.Background())
	iqCh, err := src.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	d.voiceFan.broadcast([]complex64{complex(1, 0)})
	select {
	case <-iqCh:
	case <-time.After(time.Second):
		t.Fatal("no IQ delivered to the tap")
	}
	cancel()
	// After cancel the subscription is torn down and the channel closes.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-iqCh:
			if !ok {
				return // closed — success
			}
		case <-deadline:
			t.Fatal("channel not closed after ctx cancel")
		}
	}
}
