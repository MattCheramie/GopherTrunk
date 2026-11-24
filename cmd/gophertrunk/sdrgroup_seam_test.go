package main

import (
	"math"
	"math/cmplx"
	"math/rand"
	"strings"
	"testing"
)

// seamScene synthesises one band of noise-like signal (many random tones
// around f0) as two group members would record it: member b's clock runs
// clockPPM fast, it started startDelay seconds after member a, and its LO is
// offset by loHz. Each member adds its own receiver noise.
func seamScene(seconds, rate, ca, cb, f0, clockPPM, startDelay, loHz float64, common bool) (a, b []complex64) {
	rng := rand.New(rand.NewSource(7))
	const tones = 96
	freqs := make([]float64, tones)
	amps := make([]complex128, tones)
	for k := range freqs {
		freqs[k] = f0 + (rng.Float64()-0.5)*30_000
		amps[k] = cmplx.Rect(1, rng.Float64()*2*math.Pi)
	}
	n := int(seconds * rate)
	gen := func(center, clock, t0, lo float64) []complex64 {
		out := make([]complex64, n)
		ph := make([]complex128, tones)
		step := make([]complex128, tones)
		for k := range ph {
			w := 2 * math.Pi * (freqs[k] - center - lo)
			ph[k] = amps[k] * cmplx.Rect(1, w*t0)
			step[k] = cmplx.Rect(1, w/(rate*clock))
		}
		for i := range out {
			var s complex128
			if common {
				for k := range ph {
					s += ph[k]
					ph[k] *= step[k]
				}
				s /= tones
			}
			s += complex(rng.NormFloat64()*0.03, rng.NormFloat64()*0.03)
			out[i] = complex64(s)
		}
		return out
	}
	a = gen(ca, 1, 0, 0)
	b = gen(cb, 1+clockPPM*1e-6, startDelay, loHz)
	return a, b
}

// TestAnalyzeSeam pins the phase-2 feasibility instrument: it must tell the
// three hardware classes apart from the overlap band alone, and must refuse
// to guess when the overlap carries no common signal.
func TestAnalyzeSeam(t *testing.T) {
	const rate, f0 = 240_000.0, 852_000_000.0
	ca, cb := f0-80_000, f0+80_000
	for _, tc := range []struct {
		name    string
		ppm, lo float64
		common  bool
		want    string
	}{
		{"shared clock", 0, 0, true, "coherent stitching feasible"},
		{"independent crystals", 5, 0, true, "independent sample clocks"},
		// Two RTL-SDRs 20 ppm apart: the clocks drift AND the LOs sit
		// 17 kHz apart at 852 MHz. The clock drift decides the verdict.
		{"two RTL crystals", 20, 17_040, true, "independent sample clocks"},
		{"locked clocks, LO offset", 0, 20, true, "phase walks"},
		{"quiet overlap", 0, 0, false, "no common signal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := seamScene(4, rate, ca, cb, f0, tc.ppm, 0.0123, tc.lo, tc.common)
			rep := analyzeSeam(a, b, rate, ca, cb, f0)
			if !strings.Contains(rep.Verdict, tc.want) {
				t.Fatalf("verdict %q, want %q (coherence %.2f, drift %.3f ppm, %.2f °/s)",
					rep.Verdict, tc.want, rep.MedianCoherence, rep.LagDriftPPM, rep.PhaseDriftDegPerS)
			}
			if tc.ppm != 0 && math.Abs(rep.LagDriftPPM-tc.ppm) > 1.5 {
				t.Errorf("measured %.2f ppm, scene has %.0f", rep.LagDriftPPM, tc.ppm)
			}
			if tc.lo != 0 && math.Abs(math.Abs(rep.PhaseDriftDegPerS)-360*tc.lo) > 0.1*360*tc.lo {
				t.Errorf("measured %.1f °/s, scene's %.0f Hz LO offset gives %.0f", rep.PhaseDriftDegPerS, tc.lo, 360*tc.lo)
			}
		})
	}
}
