package dmrlcn

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// loopDevice is a minimal sdr.Device that streams a fixed IQ buffer on
// repeat until its context cancels — enough to exercise the confirmer's
// DDC-tap + DMR-decode path through a real iqtap.Broker. The end-to-end
// "synthetic DMR confirms" case lives with real/known-good captures (the
// IQ→dibit→sync chain itself is covered by the dmr receiver + sync
// package tests); here we pin the confirmer's control flow: it rejects
// noise, rejects out-of-band carriers, and honours its timeout.
type loopDevice struct {
	buf   []complex64
	chunk int
}

func (d *loopDevice) Info() sdr.Info             { return sdr.Info{Serial: "loop"} }
func (d *loopDevice) SetCenterFreq(uint32) error { return nil }
func (d *loopDevice) SetSampleRate(uint32) error { return nil }
func (d *loopDevice) SetGain(int) error          { return nil }
func (d *loopDevice) SetPPM(int) error           { return nil }
func (d *loopDevice) SetBiasTee(bool) error      { return nil }
func (d *loopDevice) Close() error               { return nil }

func (d *loopDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	out := make(chan []complex64)
	go func() {
		defer close(out)
		pos := 0
		for {
			end := pos + d.chunk
			if end > len(d.buf) {
				pos, end = 0, d.chunk
			}
			cp := make([]complex64, end-pos)
			copy(cp, d.buf[pos:end])
			pos = end
			select {
			case <-ctx.Done():
				return
			case out <- cp:
			}
		}
	}()
	return out, nil
}

func newLoopBroker(buf []complex64, chunk int) *iqtap.Broker {
	return iqtap.New(&loopDevice{buf: buf, chunk: chunk}, 0, nil)
}

func TestDDCConfirmerRejectsNoise(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	noise := make([]complex64, 48_000)
	for i := range noise {
		noise[i] = complex64(complex(rng.Float64()*2-1, rng.Float64()*2-1) * 0.001)
	}
	br := newLoopBroker(noise, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := br.StreamIQ(ctx); err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	c := newDDCConfirmer(br, 450_000_000, uint32(confirmNarrowbandHz), 0.05)
	c.timeout = 300 * time.Millisecond
	if ok, _ := c.Confirm(ctx, 450_000_000); ok {
		t.Error("noise should not confirm as DMR")
	}
}

func TestDDCConfirmerRejectsOutOfBand(t *testing.T) {
	br := newLoopBroker(make([]complex64, 4096), 1024)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := br.StreamIQ(ctx); err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	// Carrier 1.5 MHz off a 2.4 MS/s center sits outside the usable band.
	c := newDDCConfirmer(br, 851_500_000, 2_400_000, 0.05)
	c.timeout = 200 * time.Millisecond
	if ok, _ := c.Confirm(ctx, 853_000_000); ok {
		t.Error("out-of-band carrier should not confirm")
	}
}
