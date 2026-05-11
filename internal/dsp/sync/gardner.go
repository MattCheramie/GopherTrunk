package sync

// Gardner is a feedback symbol-timing recovery loop for complex IQ
// signals. It's the standard non-data-aided timing detector used in
// PSK / QAM demodulators where the symbol decisions aren't yet
// available — useful for the π/4-DQPSK family (TETRA TMO, P25
// Phase 2) where the receivers currently rely on naive decimation.
//
// Algorithm (Gardner 1986): for each symbol period the detector
// samples once at the symbol time t_s and once at the midpoint
// t_s − sps/2. The timing error is
//
//	e[n] = Re{ (s[n] − s[n−1])^* · m[n] }
//
// where s[n] is the symbol-time sample, s[n−1] is the previous
// symbol-time sample, and m[n] is the midpoint sample between
// them. The error has zero mean at the correct sampling instant
// and signed deviation otherwise; a scalar update applies it to
// the sub-sample phase.
//
// Compared to MuellerMuller (this package) which is real-valued
// and decision-directed, Gardner:
//
//   - Works on complex samples without sign decisions, so it
//     starts converging before the demod has acquired symbol
//     polarity.
//   - Has zero self-noise at the optimum sampling instant for
//     RRC-filtered PSK signals, so the surviving phase estimate
//     is stable.
//   - Requires a midpoint sample, so the natural input is
//     samples at ≥ 2 sps (4 sps is typical for noisier
//     captures).
//
// Inputs are oversampled complex samples (e.g. 8 sps after the
// matched filter). Output is one sample per recovered symbol.
type Gardner struct {
	sps     float64 // nominal samples per symbol
	mu      float64 // current sub-sample phase
	gain    float64 // loop gain
	prevSym complex64
	prevMid complex64
	have    bool
	// stash holds samples from a previous Process call that
	// didn't yet line up with a symbol boundary; reused as
	// extra context so a long-running stream can be processed
	// in chunks without losing the timing estimate.
	stashed []complex64
}

// NewGardner constructs a Gardner timing-recovery loop. sps is the
// nominal samples-per-symbol (≥ 2); gain is the loop step (a small
// positive value, typical range 0.01..0.1). Panics on invalid sps.
// A non-positive gain defaults to 0.02 (slightly slower convergence
// than MuellerMuller, picked for stability on noisier IQ).
func NewGardner(sps, gain float64) *Gardner {
	if sps < 2 {
		panic("gardner: sps must be >= 2")
	}
	if gain <= 0 {
		gain = 0.02
	}
	return &Gardner{sps: sps, gain: gain, mu: sps}
}

// Process consumes oversampled complex IQ samples and emits one
// recovered symbol per nominal symbol period. dst is reused if it
// has capacity. Symbols are interpolated linearly between adjacent
// input samples at the loop's current sub-sample phase. Cross-call
// state preserves the timing estimate so chunked streams converge
// once rather than per-chunk.
func (g *Gardner) Process(dst, src []complex64) []complex64 {
	if cap(dst) < len(src)/int(g.sps)+1 {
		dst = make([]complex64, 0, len(src)/int(g.sps)+1)
	} else {
		dst = dst[:0]
	}
	// Concatenate any stashed tail from the previous call so the
	// midpoint look-back across the chunk boundary is correct.
	var buf []complex64
	if len(g.stashed) > 0 {
		buf = make([]complex64, 0, len(g.stashed)+len(src))
		buf = append(buf, g.stashed...)
		buf = append(buf, src...)
	} else {
		buf = src
	}

	// midOffset is half the symbol period (in input samples).
	half := g.sps / 2

	i := 1
	for i < len(buf) {
		g.mu -= 1.0
		if g.mu > 0 {
			i++
			continue
		}
		// At a symbol boundary. Interpolate the symbol-time sample
		// between buf[i-1] and buf[i].
		frac := 1.0 + g.mu // mu is in (-1, 0]; frac in (0, 1]
		sym := interpComplex(buf[i-1], buf[i], frac)

		// Interpolate the midpoint sample half a symbol period
		// before the current symbol-time sample. If we don't yet
		// have enough back-history, skip the error update for
		// this iteration.
		midPos := float64(i) - 1.0 + frac - half
		midSym, midOK := interpAt(buf, midPos)

		if g.have && midOK {
			// Gardner error: Re{ conj(sym - prevSym) * midSym }
			// is the canonical form for complex signals.
			diff := complex64(sym - g.prevSym)
			err := float64(real(diff)*real(midSym) + imag(diff)*imag(midSym))
			g.mu += g.sps + g.gain*err
		} else {
			g.mu += g.sps
			g.have = true
		}
		g.prevSym = sym
		g.prevMid = midSym
		dst = append(dst, sym)
		i++
	}
	// Keep enough tail samples in stash for the next call's first
	// midpoint look-back (one full symbol period of context).
	keep := int(g.sps) + 1
	if keep > len(buf) {
		keep = len(buf)
	}
	stash := make([]complex64, keep)
	copy(stash, buf[len(buf)-keep:])
	g.stashed = stash
	return dst
}

// Reset clears the loop state. Call on stream re-tune so the next
// chunk doesn't carry a stale timing estimate.
func (g *Gardner) Reset() {
	g.mu = g.sps
	g.have = false
	g.prevSym = 0
	g.prevMid = 0
	g.stashed = nil
}

// interpComplex returns a linear interpolation between a and b at
// fraction t (0..1).
func interpComplex(a, b complex64, t float64) complex64 {
	rt := 1.0 - t
	return complex64(complex(
		float64(real(a))*rt+float64(real(b))*t,
		float64(imag(a))*rt+float64(imag(b))*t,
	))
}

// interpAt returns the linearly-interpolated complex sample at
// fractional index pos in buf. Returns false if pos is outside the
// valid range [0, len(buf)-1].
func interpAt(buf []complex64, pos float64) (complex64, bool) {
	if pos < 0 || pos > float64(len(buf)-1) {
		return 0, false
	}
	lo := int(pos)
	if lo+1 >= len(buf) {
		return buf[lo], true
	}
	t := pos - float64(lo)
	return interpComplex(buf[lo], buf[lo+1], t), true
}
