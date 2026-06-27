package soapyremote

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// The stream channel is the only cushion for consumer jitter because the read
// loop never blocks (sendOrDrop). These tests pin the regression where the buffer
// was a fixed 8 chunks: at a multi-MS/s remote rate that is only a few ms of IQ,
// so ordinary scheduling jitter overflowed it and sendOrDrop shed ~2% of chunks —
// silent IQ discontinuities that shredded P25 symbol framing even while the device
// itself kept up (device_overflows=0). The fix sizes the buffer from the delivered
// rate so a realistic jitter burst is absorbed losslessly.

// fieldSamplesPerChunk mirrors the reported config: remote:mtu=8000 with the
// default CS16 wire format → (8000-24)/4 ≈ 1994 complex samples per datagram.
const fieldSamplesPerChunk = (8000 - streamHeaderSize) / 4

// TestStreamBufferDepthAbsorbsRealisticJitter is the regression: at the reported
// 6.25 MS/s the buffer must hold far more than the old fixed 8 chunks — enough to
// ride out a realistic (~100 ms) consumer stall without dropping a single chunk.
func TestStreamBufferDepthAbsorbsRealisticJitter(t *testing.T) {
	const rateHz = 6_250_000

	depth := streamBufferDepth(rateHz, fieldSamplesPerChunk)

	// A 100 ms stall at this rate/chunk size delivers this many chunks; the
	// buffer must hold all of them so none are shed while the consumer catches up.
	chunksPerSec := float64(rateHz) / float64(fieldSamplesPerChunk)
	jitter100ms := int(0.100 * chunksPerSec)
	if depth < jitter100ms {
		t.Fatalf("depth %d cannot absorb a 100 ms stall (%d chunks) at %d Hz — the old depth-8 regression",
			depth, jitter100ms, rateHz)
	}
	if depth <= 8 {
		t.Fatalf("depth %d is not materially deeper than the pre-fix fixed 8", depth)
	}
	if depth > maxStreamBufferChunks {
		t.Fatalf("depth %d exceeds the ceiling %d", depth, maxStreamBufferChunks)
	}
}

func TestStreamBufferDepthClampsAndFallback(t *testing.T) {
	if got := streamBufferDepth(0, fieldSamplesPerChunk); got != defaultStreamBufferChunks {
		t.Errorf("unknown rate: depth = %d, want fallback %d", got, defaultStreamBufferChunks)
	}
	if got := streamBufferDepth(6_250_000, 0); got != defaultStreamBufferChunks {
		t.Errorf("unknown chunk size: depth = %d, want fallback %d", got, defaultStreamBufferChunks)
	}
	// A tiny rate floors at the minimum cushion (never back to the depth-8 regime).
	if got := streamBufferDepth(1, fieldSamplesPerChunk); got != minStreamBufferChunks {
		t.Errorf("tiny rate: depth = %d, want floor %d", got, minStreamBufferChunks)
	}
	// A very high rate is capped so the channel can't grow without bound.
	if got := streamBufferDepth(100_000_000, 64); got != maxStreamBufferChunks {
		t.Errorf("huge rate: depth = %d, want ceiling %d", got, maxStreamBufferChunks)
	}
}

// nonWarningThrottle returns an overrunThrottle whose counters accumulate without
// being reset by the periodic WARN, so a test can read total drops afterwards.
func nonWarningThrottle() *overrunThrottle {
	now := time.Unix(100, 0)
	return &overrunThrottle{
		log:      slog.New(&capHandler{}),
		addr:     "test",
		interval: time.Hour,
		now:      func() time.Time { return now },
		lastWarn: now, // already "warned" at now ⇒ maybeWarn early-returns, counts persist
	}
}

// TestSendOrDropLosslessWithinBuffer feeds a realistic jitter burst through
// sendOrDrop into a rate-sized buffer with no consumer draining (a stalled
// consumer), then drains. Every chunk must survive in order with zero host drops.
// With the pre-fix depth of 8 this burst sheds all but the last 8 chunks.
func TestSendOrDropLosslessWithinBuffer(t *testing.T) {
	const rateHz = 6_250_000
	depth := streamBufferDepth(rateHz, fieldSamplesPerChunk)
	out := make(chan []complex64, depth)
	tr := nonWarningThrottle()
	d := &device{}

	// A realistic ~100 ms consumer stall's worth of chunks. The rate-sized buffer
	// holds it losslessly; the pre-fix depth of 8 would shed all but the last 8.
	r := float64(rateHz)
	burst := int(0.100 * r / float64(fieldSamplesPerChunk))
	if burst > depth {
		t.Fatalf("test setup: burst %d exceeds buffer depth %d", burst, depth)
	}
	for i := 0; i < burst; i++ {
		if !d.sendOrDrop(context.Background(), out, []complex64{complex(float32(i), 0)}, tr) {
			t.Fatalf("sendOrDrop returned false (ctx not cancelled) at chunk %d", i)
		}
	}
	if tr.hostDrops != 0 {
		t.Fatalf("dropped %d chunks of a %d-chunk burst that fits the buffer", tr.hostDrops, burst)
	}
	close(out)
	got := 0
	for chunk := range out {
		if int(real(chunk[0])) != got {
			t.Fatalf("out-of-order: chunk %d carried %v", got, real(chunk[0]))
		}
		got++
	}
	if got != burst {
		t.Fatalf("drained %d chunks, want %d (lossless)", got, burst)
	}
}

// TestSendOrDropShedsUnderSustainedOverload guards efb284f's intent: when the
// buffer is genuinely full and the consumer never catches up, sendOrDrop must keep
// shedding the oldest chunk (never blocking the read loop) and counting the drops,
// leaving the freshest chunks in order.
func TestSendOrDropShedsUnderSustainedOverload(t *testing.T) {
	const bufCap = 4
	const burst = 20
	out := make(chan []complex64, bufCap)
	tr := nonWarningThrottle()
	d := &device{}

	for i := 0; i < burst; i++ {
		if !d.sendOrDrop(context.Background(), out, []complex64{complex(float32(i), 0)}, tr) {
			t.Fatalf("sendOrDrop returned false at chunk %d", i)
		}
	}
	if want := uint64(burst - bufCap); tr.hostDrops != want {
		t.Fatalf("hostDrops = %d, want %d (oldest shed under sustained overload)", tr.hostDrops, want)
	}
	// The bufCap freshest chunks remain, in order.
	close(out)
	want := burst - bufCap
	for chunk := range out {
		if int(real(chunk[0])) != want {
			t.Fatalf("remaining chunk = %v, want freshest %d", real(chunk[0]), want)
		}
		want++
	}
	if want != burst {
		t.Fatalf("buffer held up to chunk %d, want %d", want, burst)
	}
}
