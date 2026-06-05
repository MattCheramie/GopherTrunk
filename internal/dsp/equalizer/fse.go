package equalizer

import "math"

// FSE is a T/2 fractionally-spaced blind (CMA) adaptive equalizer. It
// consumes two input samples per symbol — the half-symbol-earlier midpoint
// interpolant and the on-time symbol sample produced by the timing loop — and
// emits one equalized symbol.
//
// Why fractionally spaced: a symbol-spaced equalizer can only invert the
// channel at the symbol rate, so it cannot repair a sub-symbol pulse-shape
// mismatch. The P25 CQPSK/LSM receiver matched-filters with an RRC, but a real
// P25 C4FM transmitter shapes a different (CPM) pulse, so the outer ±1800 Hz
// rails arrive under-shot and the constellation closes — the symbol-spaced CMA
// cannot open it (issue #492). A T/2 equalizer spans two phases per symbol and
// therefore synthesizes the receive matched filter implicitly, correcting the
// pulse mismatch (and any multipath ISI) and is insensitive to residual symbol
// timing phase.
//
// Like CMA it is blind: the constant-modulus cost J = E[(|y|²−R²)²] needs no
// training symbols and is rotation-invariant, so the FSE runs ahead of the
// carrier (Costas) loop. The fractional spacing enlarges the cost's null space
// (degrees of freedom that leave |y| unchanged), which lets the taps random-
// walk on a clean, ISI-free channel; a small leakage term w←(1−leak)·w shrinks
// unexcited taps toward zero so real ISI sustains the taps but noise alone does
// not. The on-time centre tap is phase-pinned to the positive real axis after
// each update (as in CMA) so the downstream carrier loop does not read tap-phase
// drift as a frequency offset.
type FSE struct {
	taps     []complex64 // 2*symbolSpan taps at T/2 spacing
	hist     []complex64 // ring of the last len(taps) input samples
	histPos  int         // next write index into hist
	stepSize float32     // CMA step μ
	target   float32     // R²
	leak     float32     // per-update leakage toward zero (0 = vanilla CMA)
	center   int         // on-time centre-tap index (phase pin / output alignment)
}

// NewFSE builds a T/2 fractionally-spaced CMA equalizer. symbolSpan is the
// equalizer length in symbols; the filter holds 2*symbolSpan T/2-spaced taps.
// Use an even symbolSpan so the centre tap aligns with an on-time sample (the
// emission order is [mid, on], so even tap indices are on-time samples).
// stepSize is the CMA step; target is R² (1.0 for unit-modulus π/4-DQPSK after
// AGC); leak is the leakage coefficient (0 disables it; ~1e-4..1e-3 typical).
func NewFSE(symbolSpan int, stepSize, target, leak float32) *FSE {
	if symbolSpan <= 0 {
		panic("equalizer: FSE symbolSpan must be > 0")
	}
	if target <= 0 {
		panic("equalizer: FSE target must be > 0")
	}
	n := 2 * symbolSpan
	f := &FSE{
		taps:     make([]complex64, n),
		hist:     make([]complex64, n),
		stepSize: stepSize,
		target:   target,
		leak:     leak,
		center:   n / 2,
	}
	f.taps[f.center] = complex(1, 0)
	return f
}

// Reset returns the equalizer to centre-spike initial state.
func (f *FSE) Reset() {
	for i := range f.taps {
		f.taps[i] = 0
	}
	f.taps[f.center] = complex(1, 0)
	for i := range f.hist {
		f.hist[i] = 0
	}
	f.histPos = 0
}

// Taps returns a copy of the current weight vector.
func (f *FSE) Taps() []complex64 {
	out := make([]complex64, len(f.taps))
	copy(out, f.taps)
	return out
}

func (f *FSE) push(x complex64) {
	f.hist[f.histPos] = x
	f.histPos++
	if f.histPos == len(f.hist) {
		f.histPos = 0
	}
}

// Process consumes the two T/2-spaced samples of one symbol period (mid = the
// half-symbol-earlier interpolant, on = the symbol-time sample) and returns the
// equalized symbol y plus the CMA error proxy |y|²−R². The output is computed
// once, at the symbol instant, over the full 2-sps tap line; the gradient
// updates every tap against its own aligned T/2 input sample so both new
// samples participate.
func (f *FSE) Process(mid, on complex64) (complex64, float32) {
	// Shift both T/2 samples in, oldest first: the on-time sample becomes the
	// newest, the mid sample one before it.
	f.push(mid)
	f.push(on)

	n := len(f.taps)
	newest := f.histPos - 1
	if newest < 0 {
		newest = n - 1
	}

	// FIR output y = Σ_k taps[k] · hist[newest-k].
	var yr, yi float32
	idx := newest
	for k := 0; k < n; k++ {
		hr, hi := real(f.hist[idx]), imag(f.hist[idx])
		tr, ti := real(f.taps[k]), imag(f.taps[k])
		yr += tr*hr - ti*hi
		yi += tr*hi + ti*hr
		idx--
		if idx < 0 {
			idx = n - 1
		}
	}
	y := complex(yr, yi)

	mag2 := yr*yr + yi*yi
	err := mag2 - f.target

	// Leaky CMA weight update: w_k ← (1-leak)·w_k − μ·err·y·conj(x[newest-k]).
	mu := f.stepSize
	keep := 1 - f.leak
	idx = newest
	for k := 0; k < n; k++ {
		xr, xi := real(f.hist[idx]), imag(f.hist[idx])
		// y · conj(x) = (yr*xr + yi*xi) + j(yi*xr - yr*xi)
		ur := yr*xr + yi*xi
		ui := yi*xr - yr*xi
		tr, ti := real(f.taps[k]), imag(f.taps[k])
		f.taps[k] = complex(keep*tr-mu*err*ur, keep*ti-mu*err*ui)
		idx--
		if idx < 0 {
			idx = n - 1
		}
	}

	// Pin the on-time centre tap to the positive real axis (removes the CMA
	// rotation ambiguity so the carrier loop sees only the true residual).
	if ct := f.taps[f.center]; ct != 0 {
		mag := float32(math.Hypot(float64(real(ct)), float64(imag(ct))))
		derot := complex(real(ct)/mag, -imag(ct)/mag)
		for k := range f.taps {
			f.taps[k] *= derot
		}
	}
	return y, err
}
