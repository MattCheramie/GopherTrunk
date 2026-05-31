package ppm

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

func TestNewRejectsBadOptions(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	if _, err := New(Options{}); err == nil {
		t.Error("New without Bus: want error")
	}
	if _, err := New(Options{Bus: bus}); err == nil {
		t.Error("New without InputRateHz: want error")
	}
	if _, err := New(Options{Bus: bus, InputRateHz: 1_000_000}); err == nil {
		t.Error("New with sub-2-Msps rate: want error")
	}
	if _, err := New(Options{Bus: bus, InputRateHz: SampleRateHz}); err != nil {
		t.Errorf("New at 2 Msps: unexpected error %v", err)
	}
}

func TestProcessPropagatesContextCancel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, _ := New(Options{InputRateHz: SampleRateHz, Bus: bus})
	in := make(chan []complex64)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Process(ctx, in) }()
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Process err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Process did not exit after ctx cancel")
	}
}

func TestProcessNilInputErrors(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, _ := New(Options{InputRateHz: SampleRateHz, Bus: bus})
	if err := r.Process(context.Background(), nil); err == nil {
		t.Error("Process with nil input: want error")
	}
}

// modulatePPM synthesises a clean 2 Msps Mode-S IQ burst for one frame:
// quiet lead-in, the 8 µs preamble (pulses at samples 0, 2, 7, 9), the
// PPM data bits (1 = high-then-low, 0 = low-then-high), then a quiet
// tail. Magnitude is 1 for a pulse and 0 for quiet.
func modulatePPM(frame []byte) []complex64 {
	const lead = 30
	hi := complex64(complex(1, 0))
	var iq []complex64
	for i := 0; i < lead; i++ {
		iq = append(iq, 0)
	}
	// Preamble: high at 0, 2, 7, 9; low elsewhere across 16 samples.
	pre := [preambleSamples]complex64{}
	for _, p := range []int{0, 2, 7, 9} {
		pre[p] = hi
	}
	iq = append(iq, pre[:]...)
	// Data bits, MSB-first per byte.
	for _, by := range frame {
		for b := 7; b >= 0; b-- {
			if by&(1<<uint(b)) != 0 {
				iq = append(iq, hi, 0) // 1: high-then-low
			} else {
				iq = append(iq, 0, hi) // 0: low-then-high
			}
		}
	}
	for i := 0; i < frameSpan; i++ {
		iq = append(iq, 0) // tail so scan has frameSpan headroom
	}
	return iq
}

// TestEndToEndDF17Decode modulates a real DF17 identification frame to
// a clean IQ burst, runs it through the full PPM pipeline, and asserts
// the decoded AircraftReport lands on the bus with the right ICAO.
func TestEndToEndDF17Decode(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	r, err := New(Options{InputRateHz: SampleRateHz, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// 8D4840D6202CC371C32CE0576098 — ICAO 4840D6, an identification
	// extended squitter (matches internal/radio/adsb/adsb_test.go).
	frame, err := hex.DecodeString("8D4840D6202CC371C32CE0576098")
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	iq := modulatePPM(frame)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan []complex64, 1)
	done := make(chan struct{})
	go func() {
		_ = r.Process(ctx, in)
		close(done)
	}()
	in <- iq
	close(in)
	<-done

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindAircraftReport {
				continue
			}
			rep := ev.Payload.(storage.AircraftReport)
			if rep.ICAOHex != "4840D6" {
				t.Errorf("ICAOHex = %q, want 4840D6", rep.ICAOHex)
			}
			if rep.Callsign == "" {
				t.Errorf("Callsign empty; want the squitter callsign")
			}
			if got := r.Stats().FramesEmitted; got != 1 {
				t.Errorf("FramesEmitted = %d, want 1", got)
			}
			return
		case <-deadline:
			t.Fatalf("no aircraft report decoded; stats = %+v", r.Stats())
		}
	}
}

// TestChunkBoundarySplit feeds the same burst split across two chunks
// mid-frame to confirm the carry buffer reassembles it.
func TestChunkBoundarySplit(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	r, _ := New(Options{InputRateHz: SampleRateHz, Bus: bus})
	frame, _ := hex.DecodeString("8D4840D6202CC371C32CE0576098")
	iq := modulatePPM(frame)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan []complex64, 2)
	done := make(chan struct{})
	go func() {
		_ = r.Process(ctx, in)
		close(done)
	}()
	// Split partway through the preamble/data so a naive per-chunk
	// scan would miss it.
	split := 40
	in <- iq[:split]
	in <- iq[split:]
	close(in)
	<-done

	select {
	case ev := <-sub.C:
		rep := ev.Payload.(storage.AircraftReport)
		if rep.ICAOHex != "4840D6" {
			t.Errorf("ICAOHex = %q, want 4840D6", rep.ICAOHex)
		}
	case <-time.After(time.Second):
		t.Fatalf("no report across chunk boundary; stats = %+v", r.Stats())
	}
}
