package sync

import (
	"math"
	"math/rand"
	"testing"
)

// qpskPoint maps 2 bits to a unit-radius QPSK symbol at ±π/4, ±3π/4 — one
// ring of the π/4-DQPSK family, so SlicePiOver4DQPSK is the matching slicer.
func qpskPoint(bits int) complex128 {
	switch bits & 3 {
	case 0:
		return complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))
	case 1:
		return complex(math.Cos(3*math.Pi/4), math.Sin(3*math.Pi/4))
	case 2:
		return complex(math.Cos(-math.Pi/4), math.Sin(-math.Pi/4))
	default:
		return complex(math.Cos(-3*math.Pi/4), math.Sin(-3*math.Pi/4))
	}
}

// isiChannel convolves a symbol stream with a fixed multipath FIR (a main
// tap plus pre/post echoes) and adds complex Gaussian noise. This is the
// ISI a linear equalizer is meant to invert: each symbol's energy smears
// into its neighbours, widening the decision regions.
func isiChannel(sym []complex128, h []complex128, noiseSigma float64, rng *rand.Rand) []complex128 {
	out := make([]complex128, len(sym))
	for n := range sym {
		var acc complex128
		for k, hk := range h {
			if n-k >= 0 {
				acc += hk * sym[n-k]
			}
		}
		acc += complex(rng.NormFloat64()*noiseSigma, rng.NormFloat64()*noiseSigma)
		out[n] = acc
	}
	return out
}

func symbolError(a, b complex128) bool {
	// Two absolute symbols differ if their nearest π/4 grid points differ.
	return SlicePiOver4DQPSK(a) != SlicePiOver4DQPSK(b)
}

// TestLMSEqualizerRecoversISIChannel is the failing-first regression check:
// a QPSK stream through a multipath ISI channel is mis-sliced at a high rate
// on the raw (unequalized) samples, but after a sync-trained warm-up + a
// decision-directed run the equalizer drives the symbol-error rate to ~zero.
// Deleting the equalizer (slicing the raw channel output) fails the assertion.
//
// The transmitted reference is aligned to the equalizer's own delay
// (eq.Delay()): a centre-spike transversal filter outputs the symbol its
// centre tap sees, so training reference d[n] = tx[n − Δ]. The channel's
// bulk delay (main tap at index 1) is additionally absorbed by the taps.
func TestLMSEqualizerRecoversISIChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(0xC0FFEE))

	const (
		nTrain  = 600  // sync-trained warm-up symbols (known reference)
		nData   = 4000 // payload symbols (decision-directed)
		chDelay = 1    // channel main-tap index in h below
	)
	// A main tap with a strong post-cursor echo and a weaker pre-cursor —
	// enough ISI to close the eye without the equalizer.
	h := []complex128{
		complex(0.28, 0.06),
		complex(1.0, 0.0),
		complex(0.34, -0.10),
	}
	const noiseSigma = 0.05

	// Generate the full symbol sequence and its channel output.
	total := nTrain + nData
	tx := make([]complex128, total)
	for n := range tx {
		tx[n] = qpskPoint(rng.Intn(4))
	}
	rx := isiChannel(tx, h, noiseSigma, rng)

	eq := NewLMSEqualizer(9, 0.4)
	// Total reference delay: the equalizer's centre-tap delay plus the
	// channel's bulk delay. y[n] best predicts tx[n-delta].
	delta := eq.Delay() + chDelay

	// Baseline: slice the raw channel output directly (no equalizer). rx[n] is
	// dominated by tx[n-chDelay], so measure how often the raw eye is wrong.
	rawErrs := 0
	for n := nTrain; n < total; n++ {
		if symbolError(rx[n], tx[n-chDelay]) {
			rawErrs++
		}
	}
	rawSER := float64(rawErrs) / float64(nData)

	// Equalized path: sync-train on the known preamble, then decision-direct.
	for n := 0; n < nTrain; n++ {
		if n < delta {
			eq.Apply(c64(rx[n])) // fill the delay line before the reference is valid
			continue
		}
		eq.Train(c64(rx[n]), c64(tx[n-delta]))
	}
	trainErrSq := eq.LastErrorSq()

	eqErrs := 0
	for n := nTrain; n < total; n++ {
		y := eq.DecisionUpdate(c64(rx[n]), SlicePiOver4DQPSK)
		if symbolError(complex(float64(real(y)), float64(imag(y))), tx[n-delta]) {
			eqErrs++
		}
	}
	eqSER := float64(eqErrs) / float64(nData)

	t.Logf("raw SER=%.4f  equalized SER=%.4f  train |e|^2=%.4g  tapE=%.3f  delta=%d",
		rawSER, eqSER, trainErrSq, eq.TapEnergy(), delta)

	// The channel must actually close the eye, or the test proves nothing.
	if rawSER < 0.05 {
		t.Fatalf("channel too benign: raw SER=%.4f, expected a meaningful error rate", rawSER)
	}
	// The trained equalizer must open it back up.
	if eqSER > 0.01 {
		t.Fatalf("equalized SER=%.4f too high; equalizer failed to open the eye (raw=%.4f)", eqSER, rawSER)
	}
	// Training must have converged (residual error well below a symbol's energy ~1).
	if trainErrSq > 0.1 {
		t.Fatalf("training did not converge: final |e|^2=%.4g", trainErrSq)
	}
}

// TestLMSEqualizerCleanSignalIsTransparent confirms an un-distorted stream is
// passed essentially unchanged (centre-spike init, small residual after
// light adaptation) — the no-op property on a clean signal.
func TestLMSEqualizerCleanSignalIsTransparent(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	eq := NewLMSEqualizer(9, 0.3)
	delta := eq.Delay() // identity channel: y[n] = s[n-delta]
	hist := make([]complex128, 0, 2100)
	var maxDev float64
	for n := 0; n < 2000; n++ {
		s := qpskPoint(rng.Intn(4))
		hist = append(hist, s)
		if n < delta {
			eq.Apply(c64(s))
			continue
		}
		ref := hist[n-delta]
		y := eq.Train(c64(s), c64(ref)) // identity channel, aligned reference
		dev := math.Hypot(float64(real(y))-real(ref), float64(imag(y))-imag(ref))
		if n > 50 && dev > maxDev {
			maxDev = dev
		}
	}
	if maxDev > 0.15 {
		t.Fatalf("clean-signal deviation too large: max=%.4f (expected near-transparent)", maxDev)
	}
}

// TestLMSEqualizerResetRestoresSpike verifies Reset returns the taps to the
// centre-spike identity so a re-sync sheds a stale channel estimate.
func TestLMSEqualizerResetRestoresSpike(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	eq := NewLMSEqualizer(7, 0.5)
	h := []complex128{complex(0.3, 0), complex(1, 0), complex(0.3, 0)}
	sym := make([]complex128, 500)
	for n := range sym {
		sym[n] = qpskPoint(rng.Intn(4))
	}
	rx := isiChannel(sym, h, 0.02, rng)
	for n := range sym {
		eq.Train(c64(rx[n]), c64(sym[n]))
	}
	if eq.TapEnergy() == 1.0 {
		t.Fatalf("taps never adapted away from the unit spike")
	}
	eq.Reset()
	if e := eq.TapEnergy(); math.Abs(e-1.0) > 1e-12 {
		t.Fatalf("Reset did not restore the centre spike: tap energy=%.6f, want 1.0", e)
	}
	if eq.LastErrorSq() != 0 {
		t.Fatalf("Reset did not clear the error diagnostic")
	}
}

// c64 narrows a complex128 to complex64 for the equalizer's boundary API.
func c64(z complex128) complex64 { return complex(float32(real(z)), float32(imag(z))) }
