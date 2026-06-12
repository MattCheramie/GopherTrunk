package main

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// captureSink records the samples forwarded to it, per serial.
type captureSink struct {
	mu      sync.Mutex
	serial  string
	samples []int16
}

func (c *captureSink) WritePCM(serial string, samples []int16) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.serial = serial
	c.samples = append(c.samples[:0], samples...)
	return nil
}

func (c *captureSink) last() (string, []int16) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]int16, len(c.samples))
	copy(out, c.samples)
	return c.serial, out
}

func rms(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func sineInt16(n int, freq, rate float64, amp float64) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = int16(amp * 32767 * math.Sin(2*math.Pi*freq*float64(i)/rate))
	}
	return out
}

func TestLiveLoudnessSinkSilenceStaysSilent(t *testing.T) {
	bus := events.NewBus(16)
	cap := &captureSink{}
	s := newLiveLoudnessSink(cap, 8000, bus, nil)
	defer s.Close()

	in := make([]int16, 320)
	if err := s.WritePCM("dev1", in); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	serial, out := cap.last()
	if serial != "dev1" {
		t.Fatalf("serial = %q, want dev1", serial)
	}
	for i, v := range out {
		if v != 0 {
			t.Fatalf("silence produced non-zero sample %d at %d", v, i)
		}
	}
}

func TestLiveLoudnessSinkBoostsQuietAudio(t *testing.T) {
	bus := events.NewBus(16)
	cap := &captureSink{}
	s := newLiveLoudnessSink(cap, 8000, bus, nil)
	defer s.Close()

	// A quiet tone (~2% full scale) should be lifted toward the AGC
	// reference (0.3 ≈ 9830 int16 peak) over a few hundred ms.
	quiet := sineInt16(320, 1000, 8000, 0.02)
	inRMS := rms(quiet)
	for i := 0; i < 20; i++ { // ~0.8 s, past the attack/release settle
		_ = s.WritePCM("dev1", quiet)
	}
	_, out := cap.last()
	if rms(out) <= inRMS*2 {
		t.Fatalf("quiet audio not boosted: in rms %.1f, out rms %.1f", inRMS, rms(out))
	}
}

func TestLiveLoudnessSinkDoesNotClipPastInt16(t *testing.T) {
	bus := events.NewBus(16)
	cap := &captureSink{}
	s := newLiveLoudnessSink(cap, 8000, bus, nil)
	defer s.Close()

	// A near-full-scale tone should be ATTENUATED toward the AGC reference,
	// not amplified — so once the envelope settles the output is quieter than
	// the input and never wraps (clampToInt16 saturates by construction; the
	// int16 type bounds the range, so the meaningful check is attenuation).
	loud := sineInt16(320, 1000, 8000, 0.99)
	inRMS := rms(loud)
	for i := 0; i < 20; i++ {
		_ = s.WritePCM("dev1", loud)
	}
	_, out := cap.last()
	if rms(out) >= inRMS {
		t.Fatalf("loud audio not attenuated: in rms %.1f, out rms %.1f", inRMS, rms(out))
	}
}

func TestLiveLoudnessSinkResetsOnCallStart(t *testing.T) {
	bus := events.NewBus(16)
	cap := &captureSink{}
	s := newLiveLoudnessSink(cap, 8000, bus, nil)
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()

	// Drive a loud call so the envelope level climbs high.
	loud := sineInt16(320, 1000, 8000, 0.9)
	for i := 0; i < 10; i++ {
		_ = s.WritePCM("dev1", loud)
	}

	// CallEnd should drop the per-serial entry.
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{DeviceSerial: "dev1"}})
	deadline := time.Now().Add(time.Second)
	for {
		if s.lookup("dev1") == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("CallEnd did not drop the serial entry")
		}
		time.Sleep(2 * time.Millisecond)
	}

	// CallStart resets the envelope so the next call's quiet onset isn't
	// under-amplified by a stale high level. After reset, a quiet tone is
	// boosted, exactly as if the serial were fresh.
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{DeviceSerial: "dev1"}})
	quiet := sineInt16(320, 1000, 8000, 0.02)
	inRMS := rms(quiet)
	for i := 0; i < 20; i++ {
		_ = s.WritePCM("dev1", quiet)
	}
	_, out := cap.last()
	if rms(out) <= inRMS*2 {
		t.Fatalf("after reset, quiet audio not boosted: in %.1f out %.1f", inRMS, rms(out))
	}

	cancel()
	<-done
}

func TestClampToInt16(t *testing.T) {
	cases := []struct {
		in   float32
		want int16
	}{
		{0, 0},
		{1, math.MaxInt16},  // 1.0 * 32767
		{-1, -32767},        // symmetric ×32767 scaling
		{2, math.MaxInt16},  // saturate, don't wrap
		{-2, math.MinInt16}, // saturate, don't wrap
		{0.5, 16384},        // round(0.5 * 32767)
	}
	for _, c := range cases {
		if got := clampToInt16(c.in); got != c.want {
			t.Errorf("clampToInt16(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
