package main

import (
	"context"
	"testing"
	"time"
)

// newTestStreamSource wires a streamIQSource to a caller-controlled raw channel
// and launches the forwarder, mirroring the constructors without an SDR.
func newTestStreamSource(t *testing.T, ch <-chan []complex64) *streamIQSource {
	t.Helper()
	s := &streamIQSource{
		ch:     ch,
		tune:   func(uint32) error { return nil },
		rate:   func() uint32 { return 48_000 },
		settle: 0,
		bufCh:  make(chan []complex64, iqSlackDepth),
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go s.forward(ctx)
	return s
}

// TestStreamIQSourceDrainsDuringDecodeStall is the core overrun-fix guarantee:
// the forwarder keeps reading the raw SDR channel even while the consumer is
// busy (the "decode" phase), so the producer never blocks. Without the
// forwarder, a producer sending more than one chunk between Captures would
// block — exactly the back-pressure that makes SoapyRemote log "Received
// overrun message".
func TestStreamIQSourceDrainsDuringDecodeStall(t *testing.T) {
	raw := make(chan []complex64)
	s := newTestStreamSource(t, raw)

	// Simulate the decode phase: no Capture/Tune calls for a moment while the
	// producer keeps delivering. Every send must complete (be drained) within
	// the deadline; a block here would be the overrun bug.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < iqSlackDepth*4; i++ {
			raw <- []complex64{complex(float32(i), 0)}
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("producer blocked: forwarder did not keep draining the raw stream during the decode stall")
	}

	// Over-supply beyond the slack depth must have dropped, not blocked.
	if got := s.dropped.Load(); got == 0 {
		t.Fatal("expected some dropped chunks once the slack queue filled, got 0")
	}
}

// TestStreamIQSourceCaptureServesFromBuffer verifies Capture assembles exactly
// n samples from the forwarded chunks and carries the leftover to the next call.
func TestStreamIQSourceCaptureServesFromBuffer(t *testing.T) {
	raw := make(chan []complex64, 8)
	s := newTestStreamSource(t, raw)

	// Three chunks of two samples each = six samples available.
	for i := 0; i < 3; i++ {
		raw <- []complex64{complex(float32(2*i), 0), complex(float32(2*i+1), 0)}
	}

	got, err := s.Capture(context.Background(), 5)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("Capture returned %d samples, want 5", len(got))
	}
	for i, v := range got {
		if real(v) != float32(i) {
			t.Fatalf("sample %d = %v, want %d (FIFO order broken)", i, real(v), i)
		}
	}
	// The 6th sample is leftover; a follow-up Capture(1) must return it.
	rest, err := s.Capture(context.Background(), 1)
	if err != nil {
		t.Fatalf("Capture(leftover): %v", err)
	}
	if len(rest) != 1 || real(rest[0]) != 5 {
		t.Fatalf("leftover = %v, want [5]", rest)
	}
}

// TestStreamIQSourceCaptureContextCancel ensures Capture honours ctx when the
// buffer can never reach n.
func TestStreamIQSourceCaptureContextCancel(t *testing.T) {
	raw := make(chan []complex64)
	s := newTestStreamSource(t, raw)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Capture(ctx, 4); err == nil {
		t.Fatal("Capture with cancelled context returned nil error, want ctx.Err()")
	}
}

// TestStreamIQSourceForwarderStopsOnClose ensures closing the raw stream closes
// the slack queue, which Capture surfaces as a graceful end-of-stream.
func TestStreamIQSourceForwarderStopsOnClose(t *testing.T) {
	raw := make(chan []complex64)
	s := newTestStreamSource(t, raw)
	close(raw)

	// Drain whatever the forwarder forwards, then expect a closed bufCh giving
	// back the partial (empty) buffer without blocking.
	deadline := time.After(time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("Capture did not return after the raw stream closed")
		default:
		}
		out, err := s.Capture(context.Background(), 4)
		if err != nil {
			t.Fatalf("Capture: %v", err)
		}
		if len(out) < 4 {
			return // stream ended, partial returned
		}
	}
}
