package tuner

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

// toneGen produces an unbroken multi-tone IQ stream across many chunks.
// Phase accumulates in float64 between calls so chunk boundaries do not
// introduce a discontinuity (which would broaden the test signal's
// spectrum and mask DDC accuracy with leakage artifacts).
type toneGen struct {
	sampleRateHz float64
	amp          float32
	freqsHz      []float64
	n            uint64 // total samples emitted so far
}

func newToneGen(sampleRateHz float64, amp float32, freqsHz ...float64) *toneGen {
	return &toneGen{sampleRateHz: sampleRateHz, amp: amp, freqsHz: freqsHz}
}

func (g *toneGen) Next(n int) []complex64 {
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		var r, im float32
		idx := float64(g.n + uint64(i))
		for _, f := range g.freqsHz {
			theta := 2 * math.Pi * f * idx / g.sampleRateHz
			r += g.amp * float32(math.Cos(theta))
			im += g.amp * float32(math.Sin(theta))
		}
		out[i] = complex(r, im)
	}
	g.n += uint64(n)
	return out
}

// powerNearDC returns the fraction of total energy in the input IQ
// stream that lies within ±halfWidthHz of DC. A correctly-tuned tone
// concentrates almost all of its energy near DC; a mistuned tone
// spreads it across the band. This is more robust than peak-bin
// finding when the window is short and the DFT bin grid is coarse.
func powerNearDC(samples []complex64, sampleRateHz, halfWidthHz float64) float64 {
	N := len(samples)
	if N < 8 {
		return 0
	}
	binHz := sampleRateHz / float64(N)
	maxK := int(math.Ceil(halfWidthHz / binHz))
	totalPow := 0.0
	dcPow := 0.0
	for k := -N / 2; k < N/2; k++ {
		var sumR, sumI float64
		w := -2 * math.Pi * float64(k) / float64(N)
		for i, s := range samples {
			theta := w * float64(i)
			c := math.Cos(theta)
			si := math.Sin(theta)
			sumR += float64(real(s))*c - float64(imag(s))*si
			sumI += float64(real(s))*si + float64(imag(s))*c
		}
		p := sumR*sumR + sumI*sumI
		totalPow += p
		if k >= -maxK && k <= maxK {
			dcPow += p
		}
	}
	if totalPow == 0 {
		return 0
	}
	return dcPow / totalPow
}

func TestDDCBankSingleTapLandsAtDC(t *testing.T) {
	const (
		inRate  = 2_400_000.0
		outRate = 48_000.0
		toneAt  = 500_000.0
	)
	b := NewDDCBank(inRate, outRate, 0.05)
	var got []complex64
	if err := b.AddTap(toneAt, func(out []complex64) {
		got = append(got, out...)
	}); err != nil {
		t.Fatalf("AddTap: %v", err)
	}

	// Run enough chunks for the resampler to settle and produce a few
	// thousand narrow-band samples for the peak-finder.
	gen := newToneGen(inRate, 0.5, toneAt)
	for i := 0; i < 32; i++ {
		b.Process(gen.Next(4096))
	}
	if len(got) < 1024 {
		t.Fatalf("not enough output samples: %d", len(got))
	}
	settled := got[len(got)/2:]
	// Expect ≥ 95 % of post-tuning energy within ±500 Hz of DC.
	if frac := powerNearDC(settled, outRate, 500); frac < 0.95 {
		t.Errorf("only %.1f%% of power within ±500 Hz of DC after tuning to %.0f Hz",
			frac*100, toneAt)
	}
}

func TestDDCBankMultipleTapsExtractDistinctTones(t *testing.T) {
	const (
		inRate  = 2_400_000.0
		outRate = 48_000.0
	)
	offsets := []float64{-700_000, -150_000, +320_000, +900_000}
	b := NewDDCBank(inRate, outRate, 0.05)
	collected := make(map[float64]*[]complex64, len(offsets))
	for _, off := range offsets {
		off := off
		buf := &[]complex64{}
		collected[off] = buf
		if err := b.AddTap(off, func(out []complex64) {
			*buf = append(*buf, out...)
		}); err != nil {
			t.Fatalf("AddTap(%.0f): %v", off, err)
		}
	}

	gen := newToneGen(inRate, 0.2, offsets...)
	for i := 0; i < 16; i++ {
		b.Process(gen.Next(4096))
	}

	for off, bufPtr := range collected {
		buf := *bufPtr
		if len(buf) < 512 {
			t.Fatalf("tap %.0f: too few samples (%d)", off, len(buf))
		}
		settled := buf[len(buf)/2:]
		// With 4 simultaneous tones in the wideband stream, the tap of
		// interest collects ≥ 80 % of its narrow-band energy near DC -
		// the rest is filter sidelobe leakage from the other three.
		if frac := powerNearDC(settled, outRate, 500); frac < 0.80 {
			t.Errorf("tap %.0f Hz: only %.1f%% of power within ±500 Hz of DC", off, frac*100)
		}
	}
}

func TestDDCBankRejectsOutOfBandOffset(t *testing.T) {
	b := NewDDCBank(2_400_000, 48_000, 0.05)
	// Usable half-band is 2.4M * (0.5 - 0.05) = 1_080_000.
	if err := b.AddTap(1_200_000, func([]complex64) {}); !errors.Is(err, ErrOffsetOutOfBand) {
		t.Errorf("expected ErrOffsetOutOfBand, got %v", err)
	}
	if err := b.AddTap(1_000_000, func([]complex64) {}); err != nil {
		t.Errorf("in-band offset should succeed, got %v", err)
	}
}

func TestDDCBankProcessEmptyDoesNotCrash(t *testing.T) {
	b := NewDDCBank(2_400_000, 48_000, 0.05)
	called := 0
	_ = b.AddTap(0, func(out []complex64) {
		called++
		if len(out) != 0 {
			t.Errorf("expected empty output on empty input, got %d samples", len(out))
		}
	})
	b.Process(nil)
	b.Process([]complex64{})
	if called != 2 {
		t.Errorf("sink called %d times, want 2", called)
	}
}

func TestDDCBankResetClearsState(t *testing.T) {
	b := NewDDCBank(2_400_000, 48_000, 0.05)
	var got []complex64
	_ = b.AddTap(500_000, func(out []complex64) {
		got = append(got, out...)
	})
	gen := newToneGen(2_400_000, 0.5, 500_000)
	b.Process(gen.Next(4096))
	b.Reset()
	got = got[:0]
	b.Process(gen.Next(4096))
	// After reset, the first samples should look the same as the first
	// post-construction Process output. We only check that no panic
	// occurs and at least some output is produced.
	if len(got) == 0 {
		t.Errorf("no output after reset")
	}
}

// TestRationalRatioWidebandExact is a regression test for issue #550.
// A bug report claimed the polyphase resampler produces "fractional filter
// coefficients" on "non-power-of-two/fractional" wideband rates (singling out
// 5.0, 6.25 and 16.0 MHz as failing). In fact every wideband rate the report
// named reduces to an exact L/M ratio against the 48 kHz tap rate, so the
// resampler reproduces 48 kHz with zero rate error and the Kaiser coefficients
// (which depend only on L and M) are well-defined. This test locks that in.
func TestRationalRatioWidebandExact(t *testing.T) {
	const out = 48_000.0
	cases := []struct {
		in           float64
		wantL, wantM int
	}{
		{4_000_000, 3, 250},
		{5_000_000, 6, 625},
		{6_250_000, 24, 3125},
		{8_000_000, 3, 500},
		{10_000_000, 3, 625},
		{16_000_000, 3, 1000},
		{20_000_000, 3, 1250},
	}
	for _, c := range cases {
		l, m := rationalRatio(out, c.in)
		if l != c.wantL || m != c.wantM {
			t.Errorf("rationalRatio(%v, %v) = %d/%d, want %d/%d (exact ratio expected, no integer-decimation fallback)",
				out, c.in, l, m, c.wantL, c.wantM)
		}
		if gotOut := c.in * float64(l) / float64(m); gotOut != out {
			t.Errorf("rate %.0f Hz: output rate %.4f Hz != %.0f Hz (err %.6f%%)",
				c.in, gotOut, out, (gotOut-out)/out*100)
		}
	}
}

// TestDDCBankWidebandRatesLandAtDC is the empirical companion to
// TestRationalRatioWidebandExact (issue #550): it pushes a tone through a real
// DDCBank at each reported wideband rate and confirms it is cleanly extracted
// to baseband — directly refuting the report's "deaf / constellation collapse"
// claim for 5.0, 6.25 and 16.0 MHz.
func TestDDCBankWidebandRatesLandAtDC(t *testing.T) {
	const (
		outRate = 48_000.0
		toneAt  = 200_000.0 // comfortably in-band for every rate below
	)
	rates := []float64{4e6, 5e6, 6.25e6, 8e6, 10e6, 16e6, 20e6}
	for _, inRate := range rates {
		inRate := inRate
		t.Run(fmt.Sprintf("%gMHz", inRate/1e6), func(t *testing.T) {
			b := NewDDCBank(inRate, outRate, 0.05)
			var got []complex64
			if err := b.AddTap(toneAt, func(out []complex64) {
				got = append(got, out...)
			}); err != nil {
				t.Fatalf("AddTap: %v", err)
			}
			gen := newToneGen(inRate, 0.5, toneAt)
			// Feed until the resampler has emitted enough settled output for
			// the DFT-based power check (chunk yield shrinks as the rate rises).
			for i := 0; i < 8192 && len(got) < 2200; i++ {
				b.Process(gen.Next(4096))
			}
			if len(got) < 1024 {
				t.Fatalf("not enough output samples: %d", len(got))
			}
			settled := got[len(got)/2:]
			if frac := powerNearDC(settled, outRate, 500); frac < 0.95 {
				t.Errorf("rate %.0f Hz: only %.1f%% of power within ±500 Hz of DC", inRate, frac*100)
			}
		})
	}
}

// TestRationalRatioFallbackBestApproximation guards the issue #550 fix: when the
// exact reduction exceeds the resampler caps (L>64 or M>8192), rationalRatio must
// return the closest ratio under the caps, not the crude L=1/M=round integer
// decimator that drifted the rate by up to ~2%. These are the polyphase
// channelizer bin rates (fullRate/16) where the bin loses its factor of two.
func TestRationalRatioFallbackBestApproximation(t *testing.T) {
	const out = 48_000.0
	// binRate, and the old crude fallback's M that this must beat.
	cases := []struct {
		binRate  float64
		crudeOld int // M the previous fallback produced (L was 1)
	}{
		{390625, 8}, // 6.25 MS/s ÷16 — old: 1/8 → 48828 Hz (+1.7%)
		{312500, 7}, // 5.0 MS/s ÷16  — old: 1/7 → 44643 Hz (−7%)
	}
	for _, c := range cases {
		l, m := rationalRatio(out, c.binRate)
		if l < 1 || l > 64 || m < 1 || m > 8192 {
			t.Errorf("rationalRatio(%v, %v) = %d/%d violates caps (L≤64, M≤8192)", out, c.binRate, l, m)
		}
		emitted := c.binRate * float64(l) / float64(m)
		if relErr := math.Abs(emitted-out) / out; relErr > 1e-3 {
			t.Errorf("rationalRatio(%v, %v) = %d/%d → %.2f Hz, %.3f%% off 48 kHz (want <0.1%%)",
				out, c.binRate, l, m, emitted, relErr*100)
		}
		if l == 1 && m == c.crudeOld {
			t.Errorf("rationalRatio(%v, %v) still returns the crude 1/%d fallback", out, c.binRate, c.crudeOld)
		}
	}
}

func TestRationalRatioReduces(t *testing.T) {
	cases := []struct {
		in, out      float64
		wantL, wantM int
	}{
		{2_400_000, 48_000, 1, 50},
		{2_048_000, 48_000, 3, 128},
		{1_000_000, 250_000, 1, 4},
	}
	for _, c := range cases {
		gotL, gotM := rationalRatio(c.out, c.in)
		if gotL != c.wantL || gotM != c.wantM {
			t.Errorf("rationalRatio(%v, %v) = %d/%d, want %d/%d",
				c.out, c.in, gotL, gotM, c.wantL, c.wantM)
		}
	}
}
