package main

import (
	"context"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// streamIQSource adapts a continuous <-chan []complex64 IQ stream (from a live
// SDR device or an iqtap broker) to hunt.IQSource. A rolling buffer backs
// Capture; Tune retunes the underlying source and flushes the in-flight
// transient so the next Capture sees only settled IQ at the new center. It is
// the shared core behind both the standalone device path and the daemon broker
// path.
type streamIQSource struct {
	ch      <-chan []complex64
	tune    func(uint32) error
	rate    func() uint32
	pending []complex64
	settle  time.Duration
}

func (s *streamIQSource) SampleRateHz() uint32 { return s.rate() }

func (s *streamIQSource) Tune(centerHz uint32) error {
	if err := s.tune(centerHz); err != nil {
		return err
	}
	// Discard buffered + in-flight samples captured before/at the retune.
	s.pending = nil
	deadline := time.NewTimer(s.settle)
	defer deadline.Stop()
	for {
		select {
		case <-s.ch:
			// drain
		case <-deadline.C:
			return nil
		}
	}
}

func (s *streamIQSource) Capture(ctx context.Context, n int) ([]complex64, error) {
	for len(s.pending) < n {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case chunk, ok := <-s.ch:
			if !ok {
				out := s.pending
				s.pending = nil
				return out, nil // stream ended
			}
			s.pending = append(s.pending, chunk...)
		}
	}
	out := s.pending[:n:n]
	s.pending = s.pending[n:]
	return out, nil
}

// newDeviceIQSource opens a continuous IQ stream on a raw SDR device (standalone
// CLI live hunt). The caller's ctx governs the stream lifetime.
func newDeviceIQSource(ctx context.Context, dev sdr.Device, rate uint32) (*streamIQSource, error) {
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		return nil, err
	}
	return &streamIQSource{
		ch:     ch,
		tune:   dev.SetCenterFreq,
		rate:   func() uint32 { return rate },
		settle: 50 * time.Millisecond,
	}, nil
}

// newBrokerIQSource adapts a daemon iqtap.Broker (sharing a live SDR with the
// rest of the pipeline) to hunt.IQSource via a secondary subscription. Tune
// routes through the broker so its cached center stays consistent with the
// spectrum/rigctld views. Close the returned subscriber when the run ends.
func newBrokerIQSource(broker *iqtap.Broker) (*streamIQSource, *iqtap.Subscriber) {
	sub := broker.Subscribe()
	return &streamIQSource{
		ch:     sub.C,
		tune:   broker.SetCenterFreq,
		rate:   broker.SampleRateHz,
		settle: 50 * time.Millisecond,
	}, sub
}
