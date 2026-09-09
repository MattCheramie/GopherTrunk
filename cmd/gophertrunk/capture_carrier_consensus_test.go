package main

import (
	"context"
	"io"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// synthNoise builds n IQ samples of deterministic noise with no carrier, so a
// probe window over it must yield no offset estimate.
func synthNoise(n int, amp float64) []complex64 {
	iq := make([]complex64, n)
	x := uint32(0x1234ABCD)
	nz := func() float64 {
		x = x*1664525 + 1013904223
		return (float64(x>>8)/float64(1<<24) - 0.5) * amp
	}
	for i := 0; i < n; i++ {
		iq[i] = complex(float32(nz()), float32(nz()))
	}
	return iq
}

// TestCarrierProbeSpreadsWindowsAcrossCapture pins the collection geometry:
// the probe windows must cover the whole expected recording in order and
// carry the exact recorded samples for their spans, whatever chunk sizes the
// stream arrives in.
func TestCarrierProbeSpreadsWindowsAcrossCapture(t *testing.T) {
	const total = captureProbeWindows * captureProbeSamples * 2 // stride = 2 windows
	p := newCarrierProbe(total)

	// Feed index-encoded samples in awkward chunk sizes.
	stream := make([]complex64, total)
	for i := range stream {
		stream[i] = complex(float32(i), -float32(i))
	}
	for off, sz := 0, 0; off < total; off += sz {
		sz = 3000 + (off % 5000)
		if off+sz > total {
			sz = total - off
		}
		p.feed(stream[off : off+sz])
	}

	wins := p.Windows()
	if len(wins) != captureProbeWindows {
		t.Fatalf("got %d windows, want %d", len(wins), captureProbeWindows)
	}
	stride := int64(total / captureProbeWindows)
	for i, w := range wins {
		if len(w) != captureProbeSamples {
			t.Fatalf("window %d has %d samples, want %d", i, len(w), captureProbeSamples)
		}
		start := int64(i) * stride
		for j, s := range []int{0, captureProbeSamples - 1} {
			want := stream[start+int64(s)]
			if w[s] != want {
				t.Fatalf("window %d sample %d = %v, want %v (probe %d)", i, s, w[s], want, j)
			}
		}
	}

	// A capture too short for multiple windows degenerates to one at the
	// start (the pre-existing behaviour), and a truncated capture drops the
	// windows it never reached instead of returning unusable stubs.
	short := newCarrierProbe(captureProbeSamples)
	short.feed(stream[:captureProbeSamples])
	if n := len(short.Windows()); n != 1 {
		t.Errorf("short capture: got %d windows, want 1", n)
	}
	cut := newCarrierProbe(total)
	cut.feed(stream[:total/4])
	for i, w := range cut.Windows() {
		if len(w) < 1024 {
			t.Errorf("truncated capture returned unusable window %d (%d samples)", i, len(w))
		}
	}
}

// TestCarrierOffsetConsensus pins the corroboration rules that separate a
// real tuner ppm error (the same offset wherever the signal is up) from the
// transient that produced #1143's false "≈550.3 ppm" capture warning (a
// strong carrier in a single probe window).
func TestCarrierOffsetConsensus(t *testing.T) {
	const rate = 262144.0
	const freq = 854_562_500
	tone := func(off float64) []complex64 { return synthTone(captureProbeSamples, off, rate, 0.1) }
	noise := func() []complex64 { return synthNoise(captureProbeSamples, 0.1) }

	t.Run("transient in one window is rejected", func(t *testing.T) {
		wins := [][]complex64{tone(-8700), noise(), noise(), noise(), noise(), noise(), noise(), noise()}
		c := carrierOffsetConsensus(wins, rate)
		if c.OK {
			t.Fatalf("single-window transient must not reach consensus: %+v", c)
		}
		if c.Est != 1 || c.Usable != 8 {
			t.Errorf("Est=%d Usable=%d, want 1/8", c.Est, c.Usable)
		}
		if w := carrierOffsetWarning(wins, rate, freq); w != "" {
			t.Errorf("transient carrier must not produce a ppm warning, got: %s", w)
		}
		if note := carrierOffsetInconclusiveNote(c); note == "" {
			t.Error("expected an inconclusive note for a strong single-window transient")
		}
	})

	t.Run("intermittent but consistent carrier warns", func(t *testing.T) {
		// A handheld keyed for part of the capture (issue #836's measurement
		// use): three windows see the same offset, the rest are noise.
		wins := [][]complex64{tone(-8700), noise(), tone(-8700), noise(), tone(-8700), noise(), noise(), noise()}
		c := carrierOffsetConsensus(wins, rate)
		if !c.OK {
			t.Fatalf("consistent carrier across 3 windows must reach consensus: %+v", c)
		}
		if math.Abs(c.OffsetHz+8700) > 200 {
			t.Errorf("consensus offset = %.0f Hz, want ≈ -8700", c.OffsetHz)
		}
		if w := carrierOffsetWarning(wins, rate, freq); w == "" {
			t.Error("a corroborated 8.7 kHz offset must warn")
		}
		if m := carrierOffsetMeasurement(c, freq); m == "" {
			t.Error("a consensus offset must produce the measured-offset line")
		}
	})

	t.Run("two disagreeing carriers are rejected", func(t *testing.T) {
		wins := [][]complex64{tone(-8700), tone(5000), noise(), noise(), noise(), noise(), noise(), noise()}
		if c := carrierOffsetConsensus(wins, rate); c.OK {
			t.Fatalf("disagreeing windows must not reach consensus: %+v", c)
		}
	})

	t.Run("single-window capture keeps the old behaviour", func(t *testing.T) {
		wins := [][]complex64{tone(-8700)}
		c := carrierOffsetConsensus(wins, rate)
		if !c.OK || math.Abs(c.OffsetHz+8700) > 200 {
			t.Fatalf("short capture (one window) must still estimate: %+v", c)
		}
		if note := carrierOffsetInconclusiveNote(c); note != "" {
			t.Errorf("single-window consensus must not print the inconclusive note: %s", note)
		}
	})

	t.Run("small consistent offset measures without warning", func(t *testing.T) {
		wins := [][]complex64{tone(-500), tone(-500), tone(-500), noise(), noise(), noise(), noise(), noise()}
		c := carrierOffsetConsensus(wins, rate)
		if !c.OK {
			t.Fatalf("expected consensus: %+v", c)
		}
		if w := carrierOffsetWarning(wins, rate, freq); w != "" {
			t.Errorf("500 Hz should not warn, got: %s", w)
		}
		if m := carrierOffsetMeasurement(c, freq); m == "" {
			t.Error("expected a measured-offset line for the ppm-measurement use (issue #836)")
		}
	})
}

// TestCaptureStreamProbeIgnoresStartupTransient is the end-to-end
// failing-first regression for the #1143 false capture warning: a strong
// carrier present ONLY in the first ~11 ms of the stream (front-end settling,
// or a neighbour keying up at stream start) must not be reported as a tuner
// ppm error, while the same carrier present throughout still is. The old
// probe collected exactly the first captureProbeSamples, so the transient
// case warned.
func TestCaptureStreamProbeIgnoresStartupTransient(t *testing.T) {
	const rate = uint32(262144)
	const freq = 854_562_500
	const seconds = 1.0

	run := func(stream []complex64) string {
		src := make(chan []complex64)
		go func() {
			defer close(src)
			for off := 0; off < len(stream); off += 4096 {
				end := off + 4096
				if end > len(stream) {
					end = len(stream)
				}
				src <- stream[off:end]
			}
		}()
		_, windows, err := captureStream(context.Background(), io.Discard, siglab.FormatF32, src, rate, seconds, nil)
		if err != nil {
			t.Fatalf("captureStream: %v", err)
		}
		return carrierOffsetWarning(windows, float64(rate), freq)
	}

	total := int(seconds * float64(rate))
	transient := append(synthTone(captureProbeSamples, -8700, float64(rate), 0.1),
		synthNoise(total-captureProbeSamples, 0.1)...)
	if w := run(transient); w != "" {
		t.Errorf("a carrier only in the first probe window must not warn, got: %s", w)
	}

	persistent := synthTone(total, -8700, float64(rate), 0.1)
	if w := run(persistent); w == "" {
		t.Error("a persistent 8.7 kHz offset must still warn")
	}
}

// TestCaptureEffectiveRate pins the capture command's ActualSampleRate check:
// a quantizing backend's real rate must flow into the recording metadata (and
// the operator note fires only on a mismatch).
func TestCaptureEffectiveRate(t *testing.T) {
	const requested = 2_048_000
	for _, c := range []struct {
		name     string
		dev      interface{ ActualSampleRate() (uint32, error) }
		plain    bool
		want     uint32
		wantNote bool
	}{
		{"quantized", quantizingDevice{actual: 2_048_001}, false, 2_048_001, true},
		{"exact", quantizingDevice{actual: requested}, false, requested, false},
		{"error falls back", quantizingDevice{actual: 0, err: io.EOF}, false, requested, false},
		{"no extension", nil, true, requested, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			var got uint32
			var note string
			if c.plain {
				got, note = captureEffectiveRate(rateOnlyDevice{}, requested)
			} else {
				got, note = captureEffectiveRate(c.dev.(quantizingDevice), requested)
			}
			if got != c.want {
				t.Errorf("rate = %d, want %d", got, c.want)
			}
			if (note != "") != c.wantNote {
				t.Errorf("note = %q, wantNote %v", note, c.wantNote)
			}
		})
	}
}
