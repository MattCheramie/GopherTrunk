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
// The update SUBTRACTS the error, and that sign is load-bearing.
// Sampling late drives e positive — the midpoint has moved past the
// zero crossing toward the next symbol while the symbol difference
// keeps the opposite sign — so the next symbol instant must be
// pulled *earlier*. Adding it instead makes the loop's stable point
// the transition instant, the worst sampling phase there is: the
// loop settles there and never leaves, and its yield then degrades
// monotonically with loop gain. That inversion shipped here, and it
// was invisible to every test in this package because they shape
// symbols with a flat-topped triangular window and assert a quadrant
// match rate, which a badly mistimed sample still satisfies. It cost
// roughly 7x the recoverable P25 Phase 2 signalling (issue #915);
// see gardner_timing_test.go, which judges the loop against the
// transmitted dibits through the repo's real RRC modulator.
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
	// in chunks without losing the timing estimate. It is an
	// owned buffer, appended in place each call, so the per-chunk
	// keep-tail copy allocates only when it has to grow.
	stashed []complex64
	// catBuf is a reusable backing array for the stash+src
	// concatenation built at the head of Process/Process2x. Kept
	// on the struct so the hot loop (one Process call per IQ chunk)
	// does not allocate a fresh slice every chunk — the churn that
	// dominated the TETRA CC decode goroutine's GC pressure. A
	// distinct array from stashed, so copying buf's tail back into
	// stashed never aliases.
	catBuf []complex64
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
	// Reuse catBuf as the backing array so this concatenation does
	// not allocate every chunk.
	var buf []complex64
	if len(g.stashed) > 0 {
		g.catBuf = append(g.catBuf[:0], g.stashed...)
		g.catBuf = append(g.catBuf, src...)
		buf = g.catBuf
	} else {
		buf = src
	}

	// midOffset is half the symbol period (in input samples).
	half := g.sps / 2

	// Resume the walk at the first genuinely new sample. The stashed
	// samples (buf[:len(g.stashed)]) were already consumed by the
	// previous Process call; they are retained only as look-back
	// context for the midpoint interpolation below. Walking them
	// again would re-emit their symbols, inflating the recovered
	// symbol rate by ≈1 per chunk — which desynchronises every
	// chunked consumer (issue #275: the P25 CQPSK control-channel
	// path emitted ~5% surplus dibits on RTL-SDR-sized IQ chunks and
	// never locked). The walk needs buf[i-1], so the first call —
	// which has no stash — still starts at index 1.
	i := len(g.stashed)
	if i < 1 {
		i = 1
	}
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
			g.mu += g.sps - g.gain*err
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
	// Append in place into the owned stashed buffer (distinct array
	// from buf/catBuf, so no aliasing) — allocates only when it must
	// grow, which after warm-up it never does.
	keep := int(g.sps) + 1
	if keep > len(buf) {
		keep = len(buf)
	}
	g.stashed = append(g.stashed[:0], buf[len(buf)-keep:]...)
	return dst
}

// Process2x is identical to Process — same timing loop, error detector and
// cross-call stash — but emits TWO samples per recovered symbol: the
// half-symbol-earlier midpoint interpolant followed by the on-time symbol
// sample, i.e. dst grows by [mid, sym] each symbol. This is the T/2 feed for
// the fractionally-spaced equalizer (issue #492). When the midpoint look-back
// is not yet available (stream start / chunk head) the pair is emitted with
// mid = sym, a degenerate pre-lock symbol the equalizer's adaptation tolerates.
// The symbol cadence is bit-for-bit the same as Process, so chunked consumers
// stay in sync.
func (g *Gardner) Process2x(dst, src []complex64) []complex64 {
	if cap(dst) < 2*(len(src)/int(g.sps)+1) {
		dst = make([]complex64, 0, 2*(len(src)/int(g.sps)+1))
	} else {
		dst = dst[:0]
	}
	var buf []complex64
	if len(g.stashed) > 0 {
		g.catBuf = append(g.catBuf[:0], g.stashed...)
		g.catBuf = append(g.catBuf, src...)
		buf = g.catBuf
	} else {
		buf = src
	}

	half := g.sps / 2

	i := len(g.stashed)
	if i < 1 {
		i = 1
	}
	for i < len(buf) {
		g.mu -= 1.0
		if g.mu > 0 {
			i++
			continue
		}
		frac := 1.0 + g.mu
		sym := interpComplex(buf[i-1], buf[i], frac)
		midPos := float64(i) - 1.0 + frac - half
		midSym, midOK := interpAt(buf, midPos)

		if g.have && midOK {
			diff := complex64(sym - g.prevSym)
			err := float64(real(diff)*real(midSym) + imag(diff)*imag(midSym))
			g.mu += g.sps - g.gain*err
		} else {
			g.mu += g.sps
			g.have = true
		}
		g.prevSym = sym
		g.prevMid = midSym
		emitMid := midSym
		if !midOK {
			emitMid = sym // degenerate pre-lock pair
		}
		dst = append(dst, emitMid, sym) // [mid, on-time]
		i++
	}
	keep := int(g.sps) + 1
	if keep > len(buf) {
		keep = len(buf)
	}
	g.stashed = append(g.stashed[:0], buf[len(buf)-keep:]...)
	return dst
}

// Mu returns the loop's current sub-sample phase accumulator (it counts
// down from sps toward 0 each input sample and resets by +sps at every
// emitted symbol). Read-only; exposed for diagnostics so a CQPSK replay
// can render the timing-loop state the way the C4FM path renders its
// Mueller-Müller mu (issue #492).
func (g *Gardner) Mu() float64 { return g.mu }

// SPS returns the loop's nominal samples per symbol. Paired with Mu so a
// diagnostic can render mu as a fraction of the symbol period.
func (g *Gardner) SPS() float64 { return g.sps }

// Reset clears the loop state. Call on stream re-tune so the next
// chunk doesn't carry a stale timing estimate.
func (g *Gardner) Reset() {
	g.mu = g.sps
	g.have = false
	g.prevSym = 0
	g.prevMid = 0
	// Truncate rather than nil so the reused backing arrays survive a
	// re-tune and the next chunk stays allocation-free. Every consumer
	// gates on len(g.stashed), so an empty non-nil slice is equivalent.
	g.stashed = g.stashed[:0]
	g.catBuf = g.catBuf[:0]
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
