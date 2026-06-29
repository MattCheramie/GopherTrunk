package blind

import (
	"math"
	"testing"
)

// genFSK synthesizes a clean M-level FSK carrier: nSym symbols at baudHz over a
// rateHz stream, each symbol a frequency from a deterministic level pattern,
// integrated into a unit-amplitude complex phasor. This exercises the estimator
// without pulling in the siglab synthesizer (which would import this package).
func genFSK(rateHz, baudHz, devHz float64, levels, nSym int) []complex64 {
	sps := int(math.Round(rateHz / baudHz))
	out := make([]complex64, 0, sps*nSym)
	phase := 0.0
	// Level pattern: a fixed pseudo-random-ish walk over the M levels so the
	// transition density is realistic but reproducible.
	lvlVals := make([]float64, levels)
	for i := range lvlVals {
		lvlVals[i] = float64(2*i-(levels-1)) / float64(levels-1) // ±1, ±1/3 for M=4
	}
	seed := 12345
	for sym := 0; sym < nSym; sym++ {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		// Use the high bits: an LCG's low-order bits have very short periods
		// (seed%4 cycles every 4), which would plant a false sub-harmonic.
		lvl := lvlVals[(seed>>16)%levels]
		dphi := 2 * math.Pi * (lvl * devHz) / rateHz
		for k := 0; k < sps; k++ {
			phase += dphi
			out = append(out, complex64(complex(math.Cos(phase), math.Sin(phase))))
		}
	}
	return out
}

func TestEstimateSymbolRate(t *testing.T) {
	cases := []struct {
		name            string
		rate, baud, dev float64
		levels          int
	}{
		{"p25-c4fm-4800", 48000, 4800, 1800, 4},
		{"dpmr-2400", 48000, 2400, 1800, 4},
		{"tetra-ish-18000", 144000, 18000, 4000, 4},
		{"gfsk-4800-2lvl", 48000, 4800, 2400, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			iq := genFSK(tc.rate, tc.baud, tc.dev, tc.levels, 3000)
			est := Estimate(iq, tc.rate)
			t.Logf("baud=%.0f conf=%.2f class=%s levels=%d obw=%.0f",
				est.SymbolRateHz, est.SymbolRateConf, est.ModClass, est.Levels, est.OccupiedBWHz)
			if est.SymbolRateHz <= 0 {
				t.Fatalf("no symbol-rate estimate")
			}
			if rel := math.Abs(est.SymbolRateHz-tc.baud) / tc.baud; rel > 0.05 {
				t.Errorf("symbol rate = %.0f Hz, want ≈ %.0f Hz (%.1f%% off)", est.SymbolRateHz, tc.baud, rel*100)
			}
		})
	}
}

func TestEstimateLevelClass(t *testing.T) {
	// A clean 4-FSK should classify as the FSK4 family with 4 levels.
	iq := genFSK(48000, 4800, 1800, 4, 3000)
	est := Estimate(iq, 48000)
	if est.ModClass != ModFSK4 || est.Levels != 4 {
		t.Errorf("4-FSK classified as %s/%d, want %s/4", est.ModClass, est.Levels, ModFSK4)
	}
}

func TestEstimateRejectsShortOrInvalid(t *testing.T) {
	if got := Estimate(nil, 48000); got.SymbolRateHz != 0 {
		t.Errorf("nil input: got %+v, want zero", got)
	}
	if got := Estimate(make([]complex64, 1024), 0); got.SymbolRateHz != 0 {
		t.Errorf("zero rate: got %+v, want zero", got)
	}
}
