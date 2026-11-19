package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestSingleChannelDecoderReopensItsStreamAfterItEnds: a decoder that owns
// its SDR's stream must own it again after that stream ends. The first
// decoder on a serial claims the primary (openSingleChannelIQ) and that claim
// was never released, so when the stream died the restart found the serial
// "already owned", only Subscribed, and waited for ever on a broker nothing
// was streaming — behind a "decoder stopped — restarting" line that read as
// a recovery. The stale claim also kept a live view from streaming the
// dongle (acquireViewerIQ skips an owned serial), so nothing brought it back.
func TestSingleChannelDecoderReopensItsStreamAfterItEnds(t *testing.T) {
	withFastRetryBackoff(t)
	dev := &pagingAirFake{serial: pagingTestSerial, rateHz: pagingTestRateHz,
		txHz: pagingTestTxHz, syms: pagingTestFrame(t), dieAfter: 5}
	d, _ := newPagingRunDaemon(t, dev, pagingTestRateHz, pagingTestRateHz)

	var passes, chunksAfterRestart atomic.Int32
	decode := func(ctx context.Context, in <-chan []complex64) error {
		pass := passes.Add(1)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case _, ok := <-in:
				if !ok {
					return nil
				}
				if pass > 1 {
					chunksAfterRestart.Add(1)
				}
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = d.runSingleChannelDecoder(ctx, "test", pagingTestSerial, pagingTestTxHz, nil, decode)
	}()
	defer func() {
		cancel()
		<-done
	}()

	deadline := time.Now().Add(5 * time.Second)
	for chunksAfterRestart.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("no IQ after the stream ended and the decoder restarted "+
				"(decode passes=%d, StreamIQ sessions=%d): the restart did not reopen the stream",
				passes.Load(), dev.streams.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := dev.streams.Load(); got != 2 {
		t.Fatalf("StreamIQ sessions = %d, want 2 (the one that died and its replacement)", got)
	}
}

// TestWidebandPagingDecodesAfterStreamDeath: the same recovery end to end
// for a paging.wideband group — the stream dies under it, the group reopens
// it, rebuilds its DDC bank and decodes the next page.
func TestWidebandPagingDecodesAfterStreamDeath(t *testing.T) {
	withFastRetryBackoff(t)
	dev := &pagingAirFake{serial: pagingTestSerial, rateHz: pagingTestRateHz,
		txHz: pagingTestTxHz, syms: pagingTestFrame(t), dieAfter: 20}
	d, _ := newPagingRunDaemon(t, dev, pagingTestRateHz, pagingTestRateHz)
	pages, _ := startPagingGroup(t, d, pagingTestRateHz)
	awaitTestPage(t, pages, 15*time.Second,
		"the group did not come back after its SDR's stream ended")
	if got := dev.streams.Load(); got != 2 {
		t.Fatalf("StreamIQ sessions = %d, want 2", got)
	}
}
