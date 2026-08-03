package receiver

import (
	"math"
	"math/cmplx"
	"testing"
)

// TestDCBlockRemovesOffsetPreservesTone pins the DC-spike fix: the complex DC
// blocker drives a constant additive DC offset to ~0 while passing a modulated
// (zero-mean) tone essentially untouched. Without the blocker the DC pedestal
// survives and biases the π/4-DQPSK differential decode — the "DC leaks into
// voice under heavy multislot" report.
func TestDCBlockRemovesOffsetPreservesTone(t *testing.T) {
	const (
		fs   = 144000.0
		tone = 6000.0 // well inside the passband, away from DC
		dc   = complex(0.30, -0.20)
		n    = 1 << 15
	)
	// Build: a complex tone (zero mean over many cycles) plus a constant DC term.
	sig := make([]complex64, n)
	clean := make([]complex64, n)
	for i := 0; i < n; i++ {
		s := cmplx.Exp(complex(0, 2*math.Pi*tone*float64(i)/fs))
		clean[i] = complex64(s)
		sig[i] = complex64(s) + dc
	}

	// Baseline (no block): the mean stays at the DC offset — the bug.
	var rawMean complex128
	for _, x := range sig {
		rawMean += complex128(x)
	}
	rawMean /= complex(float64(n), 0)
	if cmplx.Abs(rawMean-complex128(dc)) > 1e-3 {
		t.Fatalf("precondition: input mean %.4f, want ~the DC offset %.4f", rawMean, dc)
	}

	d := &dcBlock{}
	out := d.process(nil, sig)

	// After the blocker: the steady-state mean (skip the filter's startup
	// transient) is ~0 — the DC pedestal is gone.
	const skip = 4096
	var mean complex128
	for _, x := range out[skip:] {
		mean += complex128(x)
	}
	mean /= complex(float64(n-skip), 0)
	if got := cmplx.Abs(mean); got > 5e-3 {
		t.Errorf("DC not removed: residual mean |%.5f| = %.5f, want < 0.005", mean, got)
	}

	// The tone amplitude is preserved (a ~23 Hz-corner high-pass barely touches a
	// 6 kHz tone). Compare steady-state RMS of output vs the clean tone.
	rms := func(v []complex64) float64 {
		var s float64
		for _, x := range v {
			s += float64(real(x)*real(x) + imag(x)*imag(x))
		}
		return math.Sqrt(s / float64(len(v)))
	}
	gotRMS := rms(out[skip:])
	wantRMS := rms(clean[skip:])
	if math.Abs(gotRMS-wantRMS)/wantRMS > 0.02 {
		t.Errorf("tone amplitude altered: RMS %.4f vs clean %.4f (>2%%)", gotRMS, wantRMS)
	}
}

// TestDCBlockResetClearsState confirms Reset zeroes the filter memory so a
// re-synced stream starts clean.
func TestDCBlockResetClearsState(t *testing.T) {
	d := &dcBlock{}
	d.process(nil, []complex64{complex(1, 1), complex(1, 1), complex(1, 1)})
	if d.prevX == 0 && d.prevY == 0 {
		t.Fatal("precondition: filter state should be non-zero after processing")
	}
	d.Reset()
	if d.prevX != 0 || d.prevY != 0 {
		t.Errorf("Reset left state: prevX=%v prevY=%v", d.prevX, d.prevY)
	}
}
