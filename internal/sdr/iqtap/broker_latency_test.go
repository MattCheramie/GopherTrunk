package iqtap

import (
	"context"
	"testing"
	"time"
)

// soapyChunkSamples is the samples-per-chunk a SoapyRemote stream delivers at
// its default 1500-byte MTU with cs16 samples ((1500-hdr)/4 ≈ 360). At
// 2.4 MS/s that is ~0.15 ms of IQ per chunk.
const soapyChunkSamples = 360

// pushChunks drives n soapyChunkSamples-sized chunks through the broker's
// primary while the subscriber is NOT read, the shape of a voice tap whose
// goroutine is paused (GC, a busy core) for n·0.15 ms.
func pushChunks(t *testing.T, b *Broker, dev *fakeDevice, n int) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out, err := b.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	go func() {
		for i := 0; i < n; i++ {
			dev.Out <- make([]complex64, soapyChunkSamples)
		}
	}()
	deadline := time.After(10 * time.Second)
	for i := 0; i < n; i++ {
		select {
		case <-out:
		case <-deadline:
			t.Fatalf("primary stalled at chunk %d", i)
		}
	}
}

// TestBrokerSubscriberRidesOutShortStallAtSoapyChunkSize is the #1242
// regression. The subscriber buffer was a fixed 16 CHUNKS, sized for ~4 ms
// RTL-SDR chunks; a SoapyRemote stream at 2.4 MS/s delivers ~0.15 ms chunks,
// so a wideband voice tap had ~2.4 ms of slack and any longer pause in its
// DDC goroutine silently dropped IQ. On the reporter's Pi every voice capture
// was missing 2–3.5 % of its samples in 8–14 ms holes — one in a third of the
// LDUs — which garbled the IMBE frames while the control channel (the
// primary, behind the driver's own 400 ms buffer) decoded cleanly. A 20 ms
// stall must now cost nothing.
func TestBrokerSubscriberRidesOutShortStallAtSoapyChunkSize(t *testing.T) {
	dev := newFakeDevice("soapy")
	b := New(dev, 0, nil)
	b.Seed(0, 2_400_000)
	sub := b.Subscribe()
	defer sub.Close()

	stall := 20 * time.Millisecond
	n := int(stall.Seconds() * 2_400_000 / soapyChunkSamples) // ≈133 chunks
	pushChunks(t, b, dev, n)

	if d := sub.Dropped(); d != 0 {
		t.Fatalf("subscriber dropped %d of %d chunks during a %v stall; want 0", d, n, stall)
	}
	if got := len(sub.C); got != n {
		t.Fatalf("subscriber holds %d chunks, want all %d", got, n)
	}
}

// TestBrokerSubscriberBufferIsBoundedInTime pins the other side: a consumer
// that falls behind for good still drops (the primary is never blocked), and
// what it holds is bounded by SubBufferLatency of IQ at the stream rate, not
// by an ever-growing queue.
func TestBrokerSubscriberBufferIsBoundedInTime(t *testing.T) {
	dev := newFakeDevice("soapy")
	b := New(dev, 0, nil)
	const rate = 2_400_000
	b.Seed(0, rate)
	sub := b.Subscribe()
	defer sub.Close()

	n := int(2 * SubBufferLatency.Seconds() * rate / soapyChunkSamples)
	pushChunks(t, b, dev, n)

	if sub.Dropped() == 0 {
		t.Fatal("a subscriber that never reads must eventually drop")
	}
	held := time.Duration(float64(len(sub.C)*soapyChunkSamples) / rate * float64(time.Second))
	if held > SubBufferLatency+time.Millisecond || held < SubBufferLatency/2 {
		t.Fatalf("subscriber holds %v of IQ, want ≈%v", held, SubBufferLatency)
	}
}

// TestBrokerSubscriberBufferUnknownRateKeepsChunkFloor: with no rate known
// the broker cannot size in time and keeps at least the old chunk depth.
func TestBrokerSubscriberBufferUnknownRateKeepsChunkFloor(t *testing.T) {
	dev := newFakeDevice("norate")
	b := New(dev, 0, nil)
	sub := b.Subscribe()
	defer sub.Close()
	pushChunks(t, b, dev, DefaultSubBuffer)
	if d := sub.Dropped(); d != 0 {
		t.Fatalf("dropped %d of %d chunks with an unknown rate; want 0", d, DefaultSubBuffer)
	}
}
