package equalizer

// LMS is a complex-valued tapped-delay-line adaptive equalizer trained
// with the standard Least-Mean-Squares update rule:
//
//	y[n]   = sum_k w_k(n) * x[n-k]              // FIR output
//	e[n]   = d[n] - y[n]                        // training error
//	w[n+1] = w[n] + μ · e[n] · conj(x[n-k])     // weight update
//
// Notes:
//
//   - The reference signal d[n] can be a true training preamble or, in
//     decision-directed mode, the slicer's hard decision on y[n].
//   - μ (StepSize) sets the trade-off between convergence speed and
//     mean-squared-error floor; 0.005 to 0.05 are reasonable starting
//     points for symbol-spaced channels.
//   - Initialised to a centre spike: w_{N/2}(0) = 1, others zero. That
//     starts the equaliser as a pass-through so a benign channel
//     stays roughly intact while training begins.
//
// The struct is not safe for concurrent use; one equaliser belongs
// to one demod chain.
type LMS struct {
	taps     []complex64
	hist     []complex64
	histPos  int
	stepSize float32
}

// NewLMS constructs an equaliser with `taps` complex weights and the
// supplied step size. taps must be > 0; an odd taps count is
// recommended so the centre spike is well-defined.
func NewLMS(taps int, stepSize float32) *LMS {
	if taps <= 0 {
		panic("equalizer: LMS taps must be > 0")
	}
	e := &LMS{
		taps:     make([]complex64, taps),
		hist:     make([]complex64, taps),
		stepSize: stepSize,
	}
	e.taps[taps/2] = complex(1, 0) // centre-spike init
	return e
}

// Reset returns the equaliser to its centre-spike initial state.
func (e *LMS) Reset() {
	for i := range e.taps {
		e.taps[i] = 0
	}
	e.taps[len(e.taps)/2] = complex(1, 0)
	for i := range e.hist {
		e.hist[i] = 0
	}
	e.histPos = 0
}

// Taps returns a copy of the current weight vector. Useful in tests
// and when an operator wants to inspect what the equaliser has
// learned (or stash and restore taps across calls on the same
// channel).
func (e *LMS) Taps() []complex64 {
	out := make([]complex64, len(e.taps))
	copy(out, e.taps)
	return out
}

// SetStepSize updates μ. Larger steps converge faster but settle to a
// noisier weight vector.
func (e *LMS) SetStepSize(step float32) { e.stepSize = step }

// Process consumes one input sample x and updates the filter. The
// `desired` argument is the reference / training symbol; in
// decision-directed mode supply the upstream slicer's hard decision
// on the previous output. Returns the equalised output y[n] and the
// instantaneous error e[n].
func (e *LMS) Process(x, desired complex64) (complex64, complex64) {
	// Push x into the history buffer.
	e.hist[e.histPos] = x
	e.histPos = (e.histPos + 1) % len(e.hist)

	// FIR output: y = sum_k taps[k] * hist_in_chronological_order[k].
	// The most recent sample is at hist[histPos-1] (mod N).
	var yr, yi float32
	idx := e.histPos - 1
	if idx < 0 {
		idx = len(e.hist) - 1
	}
	for i := 0; i < len(e.taps); i++ {
		hr, hi := real(e.hist[idx]), imag(e.hist[idx])
		tr, ti := real(e.taps[i]), imag(e.taps[i])
		yr += tr*hr - ti*hi
		yi += tr*hi + ti*hr
		idx--
		if idx < 0 {
			idx = len(e.hist) - 1
		}
	}
	y := complex(yr, yi)

	// Error e = d - y.
	err := complex(real(desired)-yr, imag(desired)-yi)

	// Weight update: w_k += μ · e · conj(x[n-k]).
	//
	// For the non-Hermitian filter y = Σ_k w_k·x_k, Wirtinger calculus
	// gives ∂J/∂w_k* = −e·conj(x_k) with J = |d−y|², so steepest descent
	// is w_k += μ·e·conj(x_k). The conjugate is on x, NOT on e: the two
	// differ only in the sign of the imaginary cross-term, so a real-only
	// channel can't tell them apart — but on a COMPLEX channel the wrong
	// sign turns the update into ascent on the imaginary axis and the taps
	// diverge. (The earlier code computed x·conj(e), which is that wrong
	// sign; it survived only because the package tests used a real-valued
	// channel coefficient. See TestLMSConvergesOnComplexChannel.)
	mu := e.stepSize
	er, ei := real(err), imag(err)
	idx = e.histPos - 1
	if idx < 0 {
		idx = len(e.hist) - 1
	}
	for i := 0; i < len(e.taps); i++ {
		xr, xi := real(e.hist[idx]), imag(e.hist[idx])
		// e · conj(x) = (er + j*ei)(xr - j*xi)
		//             = (er*xr + ei*xi) + j*(ei*xr - er*xi)
		ur := er*xr + ei*xi
		ui := ei*xr - er*xi
		tr, ti := real(e.taps[i]), imag(e.taps[i])
		e.taps[i] = complex(tr+mu*ur, ti+mu*ui)
		idx--
		if idx < 0 {
			idx = len(e.hist) - 1
		}
	}
	return y, err
}
