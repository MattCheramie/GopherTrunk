package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// iqSlackDepth is the number of IQ chunks the forwarder buffers between the raw
// SDR delivery channel and Capture. It is pure slack: its only job is to absorb
// brief scheduling jitter so a momentary stall doesn't drop RF. Sustained
// overload (a multi-second decode of a captured dwell) still drops here — by
// design — because the forwarder keeps reading the SDR regardless, which is
// what prevents the server-side "Received overrun message".
const iqSlackDepth = 64

// streamIQSource adapts a continuous <-chan []complex64 IQ stream (from a live
// SDR device or an iqtap broker) to hunt.IQSource. A forwarder goroutine drains
// the raw stream into a bounded slack queue (bufCh) so the SDR/SoapyRemote is
// read in real time even while the hunt loop is busy decoding a captured dwell;
// Capture pulls from the slack queue. Tune retunes the underlying source and
// flushes the in-flight transient so the next Capture sees only settled IQ at
// the new center. It is the shared core behind both the standalone device path
// and the daemon broker path.
type streamIQSource struct {
	ch      <-chan []complex64
	tune    func(uint32) error
	rate    func() uint32
	setGain func(int) error // nil on the shared-SDR broker path (gain control unavailable)
	settle  time.Duration

	bufCh   chan []complex64 // slack queue: forwarder → Capture
	pending []complex64      // Capture-side leftover (< n) carried between calls
	dropped atomic.Uint64    // chunks dropped at bufCh when decode falls behind real time
}

func (s *streamIQSource) SampleRateHz() uint32 { return s.rate() }

// SetGain implements hunt.GainSettable for the auto-gain sweep. It is only wired
// on the standalone device path; the broker path returns an error because
// changing gain would disrupt the daemon's other consumers of the shared SDR.
func (s *streamIQSource) SetGain(tenthDB int) error {
	if s.setGain == nil {
		return fmt.Errorf("gain control unavailable on this IQ source")
	}
	return s.setGain(tenthDB)
}

// forward continuously drains the raw SDR/broker channel into the slack queue so
// the device is read at real-time rate even while the hunt loop is decoding a
// previously captured dwell. Without this, nothing reads s.ch during the
// (multi-second) decode phase, the driver's delivery channel backs up, and
// SoapyRemote logs "Received overrun message". When the slack queue is full —
// decode is sustainedly behind real time — the newest chunk is dropped rather
// than blocking ingestion; the next Tune discards the gap anyway. The goroutine
// exits when ctx is cancelled or the raw stream closes.
func (s *streamIQSource) forward(ctx context.Context) {
	defer close(s.bufCh)
	var lastLogAt time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-s.ch:
			if !ok {
				return
			}
			select {
			case s.bufCh <- chunk:
			default:
				// Slack queue full: drop and keep draining s.ch so the SDR
				// never blocks. Rate-limit the warning to once per second.
				n := s.dropped.Add(1)
				if now := time.Now(); now.Sub(lastLogAt) >= time.Second {
					slog.Default().Warn("hunt: decode falling behind real time; dropping buffered IQ (the SDR is still drained, so this is a CPU/throughput limit, not an RF overrun)",
						"dropped_total", n)
					lastLogAt = now
				}
			}
		}
	}
}

func (s *streamIQSource) Tune(centerHz uint32) error {
	if err := s.tune(centerHz); err != nil {
		return err
	}
	// Discard buffered + in-flight samples captured before/at the retune: empty
	// the slack queue and keep discarding for the settle window so the next
	// Capture sees only IQ produced after the retune transient.
	s.pending = nil
	deadline := time.NewTimer(s.settle)
	defer deadline.Stop()
	for {
		select {
		case <-s.bufCh:
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
		case chunk, ok := <-s.bufCh:
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
// CLI live hunt). The caller's ctx governs both the stream and the forwarder
// goroutine lifetime.
func newDeviceIQSource(ctx context.Context, dev sdr.Device, rate uint32) (*streamIQSource, error) {
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		return nil, err
	}
	s := &streamIQSource{
		ch:      ch,
		tune:    dev.SetCenterFreq,
		rate:    func() uint32 { return rate },
		setGain: dev.SetGain,
		settle:  50 * time.Millisecond,
		bufCh:   make(chan []complex64, iqSlackDepth),
	}
	go s.forward(ctx)
	return s, nil
}

// newBrokerIQSource adapts a daemon iqtap.Broker (sharing a live SDR with the
// rest of the pipeline) to hunt.IQSource via a secondary subscription. Tune
// routes through the broker so its cached center stays consistent with the
// spectrum/rigctld views. Close the returned subscriber when the run ends — that
// closes the broker's delivery channel, which stops the forwarder goroutine.
func newBrokerIQSource(broker *iqtap.Broker) (*streamIQSource, *iqtap.Subscriber) {
	sub := broker.Subscribe()
	s := &streamIQSource{
		ch:     sub.C,
		tune:   broker.SetCenterFreq,
		rate:   broker.SampleRateHz,
		settle: 50 * time.Millisecond,
		bufCh:  make(chan []complex64, iqSlackDepth),
	}
	// Teardown is driven by sub.Close() closing sub.C; the forwarder returns
	// when the channel closes, so context.Background is sufficient here.
	go s.forward(context.Background())
	return s, sub
}
