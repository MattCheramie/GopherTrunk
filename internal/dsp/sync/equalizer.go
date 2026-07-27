package sync

import "math"

// LMSEqualizer is a T-spaced (symbol-spaced) complex adaptive linear
// equalizer for the π/4-DQPSK / H-DQPSK symbol stream. It sits after
// carrier recovery (QPSKCostas) and before the differential decode, and
// removes the residual inter-symbol interference (ISI) a real channel
// leaves on the *absolute* symbols: RRC pulse-shape mismatch, a fractional
// timing error the Gardner loop settles at, and mild multipath — all of
// which smear a symbol's energy into its neighbours and widen the
// differential-phase decision, so the outer FEC (the P25 Phase 2
// RS(24,16,9)) sees symbol errors on an otherwise-decodable burst
// (issue #915).
//
// It is a transversal FIR filter with N complex taps run over the symbol
// stream, adapted by the normalised least-mean-squares (NLMS) rule:
//
//	y[n] = Σ_k w[k]·x[n−k]                     (equalized output)
//	e[n] = d[n] − y[n]                          (error vs. reference d)
//	w[k] ← w[k] + μ · e[n] · conj(x[n−k]) / (ε + ‖x‖²)
//
// NLMS (rather than plain LMS) normalises the step by the delay-line
// energy, so convergence is insensitive to the absolute signal level the
// upstream AGC settles at. The taps initialise to a centre spike
// (w[centre] = 1, rest 0), so an un-adapted equalizer is a pure delay —
// a no-op on a clean signal, matching the primitive-first convention the
// Gardner / Costas loops follow (a centred, undistorted stream trains the
// taps back to ~the centre spike and the path stays transparent).
//
// The reference d[n] comes from one of two sources:
//
//   - Trained: the known transmitted symbol, during a sync/preamble whose
//     symbols the receiver knows a priori (the 20-symbol P25 Phase 2
//     outbound frame sync). This is the "sync-trained" mode — the taps
//     converge on known data, immune to decision errors.
//
//   - Decision-directed: the nearest constellation point of y[n], during
//     the unknown payload that follows. Correct once the sync-trained taps
//     have opened the eye; the caller supplies the slicer so the primitive
//     stays modulation-agnostic (SlicePiOver4DQPSK is the slicer for this
//     family).
//
// The filter is stateful (it carries the delay line across calls) and not
// safe for concurrent use. One per tuned frequency / per call chain, reset
// on stream re-sync.
type LMSEqualizer struct {
	w      []complex128 // adaptive taps; w[0] multiplies the most-recent input
	line   []complex128 // delay line, line[0] = most-recent input sample
	mu     float64      // NLMS step size (0 < μ < 2 for stability)
	eps    float64      // NLMS regulariser, guards the divide when the line is quiet
	center int          // centre-tap index used for the initial spike
	lastSq float64      // |e|² of the most recent adaptation, for convergence diagnostics
}

// eqRegulariser is the default NLMS denominator floor. It keeps the tap
// update finite when the delay line is (near) zero and bounds the effective
// step on very weak input.
const eqRegulariser = 1e-6

// NewLMSEqualizer builds a symbol-spaced equalizer with numTaps taps and
// NLMS step μ. numTaps is forced odd and ≥ 1 so a symmetric centre tap
// exists (an even count rounds up); μ is clamped to (0, 2) — the NLMS
// stability bound — defaulting to 0.3 when out of range. The taps start as
// a centre spike so the filter is initially a pure delay.
func NewLMSEqualizer(numTaps int, mu float64) *LMSEqualizer {
	if numTaps < 1 {
		numTaps = 1
	}
	if numTaps%2 == 0 {
		numTaps++
	}
	if mu <= 0 || mu >= 2 {
		mu = 0.3
	}
	e := &LMSEqualizer{
		w:      make([]complex128, numTaps),
		line:   make([]complex128, numTaps),
		mu:     mu,
		eps:    eqRegulariser,
		center: numTaps / 2,
	}
	e.w[e.center] = 1
	return e
}

// push shifts x into the front of the delay line, dropping the oldest sample.
func (e *LMSEqualizer) push(x complex128) {
	copy(e.line[1:], e.line[:len(e.line)-1])
	e.line[0] = x
}

// output computes the current filter output y = Σ_k w[k]·line[k].
func (e *LMSEqualizer) output() complex128 {
	var y complex128
	for k, wk := range e.w {
		y += wk * e.line[k]
	}
	return y
}

// adapt applies one NLMS tap update toward reference d given the current
// (already-pushed) delay line and pre-computed output y, and records |e|².
func (e *LMSEqualizer) adapt(d, y complex128) {
	err := d - y
	e.lastSq = real(err)*real(err) + imag(err)*imag(err)
	var norm float64
	for _, s := range e.line {
		norm += real(s)*real(s) + imag(s)*imag(s)
	}
	g := e.mu / (e.eps + norm)
	for k, s := range e.line {
		// w[k] += μ · e · conj(line[k]) / (ε + ‖line‖²)
		e.w[k] += complex(g, 0) * err * complex(real(s), -imag(s))
	}
}

// Train pushes one input symbol, equalizes it, adapts the taps toward the
// known reference symbol ref, and returns the equalized symbol. Use during
// a sync/preamble whose transmitted symbols are known a priori.
func (e *LMSEqualizer) Train(x, ref complex64) complex64 {
	e.push(complex(float64(real(x)), float64(imag(x))))
	y := e.output()
	e.adapt(complex(float64(real(ref)), float64(imag(ref))), y)
	return complex(float32(real(y)), float32(imag(y)))
}

// DecisionUpdate pushes one input symbol, equalizes it, adapts the taps
// toward the sliced (decided) constellation point, and returns the
// equalized symbol. slice maps an equalized sample to its nearest ideal
// constellation point. Use on the unknown payload after the sync-trained
// taps have opened the eye.
func (e *LMSEqualizer) DecisionUpdate(x complex64, slice func(complex128) complex128) complex64 {
	e.push(complex(float64(real(x)), float64(imag(x))))
	y := e.output()
	e.adapt(slice(y), y)
	return complex(float32(real(y)), float32(imag(y)))
}

// Apply pushes one input symbol and returns the equalized output WITHOUT
// adapting the taps (a frozen equalizer). Use when adaptation is disabled
// or the eye is known open and further adaptation would only add noise.
func (e *LMSEqualizer) Apply(x complex64) complex64 {
	e.push(complex(float64(real(x)), float64(imag(x))))
	y := e.output()
	return complex(float32(real(y)), float32(imag(y)))
}

// EqualizeGated pushes one input symbol, equalizes it, and adapts the taps
// toward the sliced decision ONLY when the equalized sample is within tol of
// that decision — i.e. the symbol is reliable. On an unreliable symbol (the
// equalized point sits far from every constellation point) the taps are frozen
// for that sample. It returns the equalized symbol.
//
// This is the streaming form used in the live receiver, where no known training
// sequence is fed in. Plain decision-directed LMS adapts on every symbol,
// including its own decision errors, and can walk the taps away from the
// solution when the eye starts closed; the reliability gate breaks that
// feedback — the taps only move on symbols the current equalizer already
// decodes confidently. The frame sync and clean payload are exactly those
// high-confidence symbols, so the equalizer sharpens the eye on the reliable
// structure and leaves the noise-dominated symbols alone (issue #915). tol is a
// Euclidean distance on the unit-radius constellation; ~0.4 keeps only symbols
// comfortably inside a decision region. A centre-spike (un-adapted) equalizer on
// a clean signal decides every symbol reliably and adapts toward its own
// (correct) decisions, so it stays transparent.
func (e *LMSEqualizer) EqualizeGated(x complex64, slice func(complex128) complex128, tol float64) complex64 {
	e.push(complex(float64(real(x)), float64(imag(x))))
	y := e.output()
	d := slice(y)
	dr, di := real(y)-real(d), imag(y)-imag(d)
	if dr*dr+di*di <= tol*tol {
		e.adapt(d, y)
	}
	return complex(float32(real(y)), float32(imag(y)))
}

// Delay returns the equalizer's reference delay in symbols: the centre-tap
// index. The equalized output at step n corresponds to the input symbol the
// centre tap sees, so a caller training on known symbols must align the
// reference to this delay — feed reference d[n] = tx[n − Delay()] (plus any
// channel bulk delay the taps will additionally absorb). Getting the
// alignment wrong makes the taps chase an unrelated symbol and never
// converge.
func (e *LMSEqualizer) Delay() int { return e.center }

// cmaRadius is the Godard / CMA dispersion constant R₂ = E[|s|⁴]/E[|s|²].
// Every symbol of a PSK (constant-modulus) constellation has |s| = 1, so
// R₂ = 1: the algorithm drives the equalized modulus toward unity.
const cmaRadius = 1.0

// CMAUpdate pushes one input symbol, equalizes it, and adapts the taps by the
// Constant Modulus Algorithm (Godard, p = 2) — it drives |y| toward the unit
// modulus every π/4-DQPSK / H-DQPSK symbol carries, using NO phase decision.
//
// This is the correct blind equalizer for P25 Phase 2. The H-DQPSK absolute
// constellation is not a fixed phase grid: the modulator adds a π/8 rotation
// *cumulatively every symbol* (and the carrier loop is designed to keep it, so
// it survives to the symbol stream), so the absolute symbols spin through every
// phase. A decision-directed phase slicer has no fixed grid to lock to and
// would fight the spin; CMA is immune to it — it only sees the modulus, which
// is constant regardless of rotation and of any residual carrier phase. It
// removes the amplitude ripple / inter-symbol interference a channel imposes;
// the leftover phase is cancelled downstream by the differential decode.
//
// The taps still initialise to a centre spike, so on a clean unit-modulus
// signal the CMA error is ~0 and the equalizer stays transparent. Returns the
// equalized symbol.
func (e *LMSEqualizer) CMAUpdate(x complex64) complex64 {
	e.push(complex(float64(real(x)), float64(imag(x))))
	y := e.output()
	yy := real(y)*real(y) + imag(y)*imag(y)
	// CMA cost J = (|y|² − R₂)². Gradient descent gives the tap update
	// w[k] += μ · (R₂ − |y|²)·y · conj(line[k]) / (ε + ‖line‖²).
	errTerm := complex(cmaRadius-yy, 0) * y
	e.lastSq = (yy - cmaRadius) * (yy - cmaRadius)
	var norm float64
	for _, s := range e.line {
		norm += real(s)*real(s) + imag(s)*imag(s)
	}
	g := e.mu / (e.eps + norm)
	for k, s := range e.line {
		e.w[k] += complex(g, 0) * errTerm * complex(real(s), -imag(s))
	}
	return complex(float32(real(y)), float32(imag(y)))
}

// LastErrorSq returns |e|² of the most recent Train / DecisionUpdate — a
// convergence diagnostic. It falls toward the noise floor as the taps
// converge; a value that stays high means the equalizer has not locked
// (too few taps, wrong reference, or a channel beyond a linear equalizer's
// reach).
func (e *LMSEqualizer) LastErrorSq() float64 { return e.lastSq }

// TapEnergy returns Σ|w[k]|² — the total tap energy. A linear equalizer
// inverting a channel notch grows its taps; an unbounded growth signals
// divergence (μ too large for the channel) or noise enhancement.
func (e *LMSEqualizer) TapEnergy() float64 {
	var s float64
	for _, wk := range e.w {
		s += real(wk)*real(wk) + imag(wk)*imag(wk)
	}
	return s
}

// Reset returns the equalizer to its initial centre-spike state and clears
// the delay line. Call on stream re-sync so the taps shed a stale channel
// estimate.
func (e *LMSEqualizer) Reset() {
	for i := range e.w {
		e.w[i] = 0
	}
	for i := range e.line {
		e.line[i] = 0
	}
	e.w[e.center] = 1
	e.lastSq = 0
}

// SlicePiOver4DQPSK maps an equalized sample to the nearest point of the
// 8-PSK super-constellation the π/4-DQPSK / H-DQPSK family visits: two
// QPSK constellations offset by π/4, i.e. the eight phases k·π/4 at unit
// radius. It is the decision slicer for DecisionUpdate on this modulation —
// the absolute symbols alternate between the {π/4, 3π/4, …} and {0, π/2, …}
// rings, and their union is the 8-PSK grid, so slicing to the nearest of
// the eight phases is the correct hard decision on the absolute symbol
// regardless of which ring it came from. Magnitude is normalised to unit
// radius so the reference does not fight the equalizer's gain.
func SlicePiOver4DQPSK(y complex128) complex128 {
	ang := math.Atan2(imag(y), real(y))
	// Snap to the nearest multiple of π/4.
	q := math.Round(ang/(math.Pi/4)) * (math.Pi / 4)
	return complex(math.Cos(q), math.Sin(q))
}
