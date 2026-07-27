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

// TestLMSEqualizerGatedOpensEyeWithoutTraining is the streaming-path check:
// with NO known training sequence, EqualizeGated (confidence-gated
// decision-directed) bootstraps from the centre spike and opens a moderately
// closed ISI eye, driving the symbol-error rate well below the raw rate. The
// gate is what makes this safe — plain DD on every symbol would adapt on its
// own errors and can diverge.
func TestLMSEqualizerGatedOpensEyeWithoutTraining(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5EED))
	const (
		n       = 8000
		chDelay = 1
	)
	// A moderate ISI channel — eye partly closed but decisions still mostly
	// right, the regime where a reliability-gated DD equalizer bootstraps.
	h := []complex128{complex(0.22, 0.04), complex(1.0, 0.0), complex(0.26, -0.06)}
	tx := make([]complex128, n)
	for i := range tx {
		tx[i] = qpskPoint(rng.Intn(4))
	}
	rx := isiChannel(tx, h, 0.04, rng)

	eq := NewLMSEqualizer(9, 0.4)
	delta := eq.Delay() + chDelay

	rawErrs, eqErrs, measured := 0, 0, 0
	for i := 0; i < n; i++ {
		y := eq.EqualizeGated(c64(rx[i]), SlicePiOver4DQPSK, 0.45)
		// Measure only the back half, after the gated loop has converged.
		if i >= n/2 && i-delta >= 0 {
			measured++
			if symbolError(rx[i], tx[i-chDelay]) {
				rawErrs++
			}
			if symbolError(complex(float64(real(y)), float64(imag(y))), tx[i-delta]) {
				eqErrs++
			}
		}
	}
	rawSER := float64(rawErrs) / float64(measured)
	eqSER := float64(eqErrs) / float64(measured)
	t.Logf("raw SER=%.4f  gated-equalized SER=%.4f  tapE=%.3f", rawSER, eqSER, eq.TapEnergy())

	if rawSER < 0.02 {
		t.Fatalf("channel too benign to prove anything: raw SER=%.4f", rawSER)
	}
	if eqSER >= rawSER*0.5 {
		t.Fatalf("gated equalizer did not open the eye: raw=%.4f eq=%.4f", rawSER, eqSER)
	}
}

// hdqpskStream builds a spinning H-DQPSK-like absolute symbol stream: each
// dibit's raw phase increment {0, π/2, π, −π/2} plus a fixed per-symbol
// rotation (π/8 for P25 Phase 2) accumulate into the absolute phase, so the
// constellation has no fixed grid — exactly the case CMA handles and a phase
// slicer cannot.
func hdqpskStream(dibits []int, rotation float64) []complex128 {
	out := make([]complex128, len(dibits))
	inc := [4]float64{0, math.Pi / 2, math.Pi, -math.Pi / 2}
	phase := 0.3 // arbitrary initial phase
	for i, b := range dibits {
		phase += inc[b&3] + rotation
		out[i] = complex(math.Cos(phase), math.Sin(phase))
	}
	return out
}

// diffDibit recovers a dibit from the differential phase of consecutive
// symbols, the same operation the receiver's differential decoder performs:
// arg(cur·conj(prev)) − rotation, sliced to the nearest π/2.
func diffDibit(cur, prev complex128, rotation float64) int {
	d := cur * complex(real(prev), -imag(prev)) // cur · conj(prev)
	ang := math.Atan2(imag(d), real(d)) - rotation
	return int(math.Round(ang/(math.Pi/2))) & 3
}

// TestLMSEqualizerCMARecoversSpinningISIChannel is the receiver-realistic
// check: a spinning H-DQPSK constellation through an ISI channel loses
// differential dibits, and CMA — which needs no fixed phase grid — reduces the
// modulus dispersion and drives the differential dibit-error rate down. A
// decision-directed phase slicer cannot help here because the absolute
// constellation never sits still.
func TestLMSEqualizerCMARecoversSpinningISIChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(0xB16B00B5))
	const (
		n        = 8000
		rotation = math.Pi / 8 // P25 Phase 2 H-DQPSK
		chDelay  = 1
	)
	h := []complex128{complex(0.30, 0.05), complex(1.0, 0.0), complex(0.30, -0.05)}
	tx := make([]int, n)
	for i := range tx {
		tx[i] = rng.Intn(4)
	}
	sym := hdqpskStream(tx, rotation)
	rx := isiChannel(sym, h, 0.03, rng)

	eq := NewLMSEqualizer(11, 0.3)
	delta := eq.Delay() + chDelay

	eqOut := make([]complex128, n)
	for i := 0; i < n; i++ {
		y := eq.CMAUpdate(c64(rx[i]))
		eqOut[i] = complex(float64(real(y)), float64(imag(y)))
	}

	// Compare differential dibit errors over the converged back half.
	rawErrs, eqErrs, measured := 0, 0, 0
	for i := n / 2; i < n; i++ {
		measured++
		if diffDibit(rx[i], rx[i-1], rotation) != tx[i] {
			rawErrs++
		}
		if i-delta-1 >= 0 && diffDibit(eqOut[i], eqOut[i-1], rotation) != tx[i-delta] {
			eqErrs++
		}
	}
	rawSER := float64(rawErrs) / float64(measured)
	eqSER := float64(eqErrs) / float64(measured)
	t.Logf("spinning H-DQPSK: raw diff-SER=%.4f  CMA diff-SER=%.4f  tapE=%.3f  lastErr=%.4g",
		rawSER, eqSER, eq.TapEnergy(), eq.LastErrorSq())

	if rawSER < 0.02 {
		t.Fatalf("channel too benign: raw diff-SER=%.4f", rawSER)
	}
	if eqSER >= rawSER*0.6 {
		t.Fatalf("CMA did not open the spinning eye: raw=%.4f eq=%.4f", rawSER, eqSER)
	}
}

// TestLMSEqualizerCMACleanSignalIsTransparent confirms CMA on a clean
// unit-modulus spinning stream stays near the centre-spike identity (the CMA
// error is ~0 when the modulus is already unity), so enabling it is a no-op on
// a strong signal.
func TestLMSEqualizerCMACleanSignalIsTransparent(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const rotation = math.Pi / 8
	tx := make([]int, 3000)
	for i := range tx {
		tx[i] = rng.Intn(4)
	}
	sym := hdqpskStream(tx, rotation)
	eq := NewLMSEqualizer(11, 0.3)
	var maxErr float64
	for i, s := range sym {
		eq.CMAUpdate(c64(s))
		if i > 200 && eq.LastErrorSq() > maxErr {
			maxErr = eq.LastErrorSq()
		}
	}
	if maxErr > 0.02 {
		t.Fatalf("CMA not transparent on a clean signal: max |e|^2=%.4g", maxErr)
	}
}

// c64 narrows a complex128 to complex64 for the equalizer's boundary API.
func c64(z complex128) complex64 { return complex(float32(real(z)), float32(imag(z))) }
