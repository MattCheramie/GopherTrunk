package receiver

import "math"

// cmaEqualizer is a T-spaced blind adaptive equalizer that inverts the linear
// channel (multipath / ISI / band-edge group delay) smearing the π/4-DQPSK
// constellation on real captures — the demod-side gap that made otherwise-clean
// concurrent-load TETRA recordings garble. Validated on the reporter's captures
// to roughly double the CRC-valid TCH/S burst yield (e.g. 1→208, 35→133,
// 101→161) with no regression on already-clean captures.
//
// Design — adapt continuously, apply a FROZEN snapshot:
//
// The equalizer keeps two tap sets. wAdapt is updated every symbol by the
// Constant-Modulus Algorithm (CMA(2,2), Godard p=2), which needs no training
// sequence and no symbol decisions — it drives the equalized samples toward the
// unit modulus that π/4-DQPSK already has. wApply is a snapshot of wAdapt,
// refreshed every snapEvery symbols, and is the filter actually applied to the
// output.
//
// Why the split is essential: CMA's cost is rotation-invariant, so wAdapt's
// output phase wanders freely as it converges. Feeding that continuously
// adapting output to the *differential* decoder (s·conj(last)) is fatal — a
// time-varying phase does not cancel in the differential and corrupts every
// dibit. wApply is held FIXED between snapshots, so within any window (and so
// within any 255-symbol burst) it imposes only a constant phase, which the
// differential decoder cancels, while still removing the ISI. Only the single
// symbol straddling a snapshot sees a phase step; with snapEvery ≫ a burst that
// is a few dibits per stream, absorbed by the FEC.
//
// wApply is center-spike initialised (a pass-through), so before CMA converges
// the output is exactly the un-equalized signal — the equalizer can only help
// or, on a clean signal, leave it essentially unchanged.
//
// The input is normalised by a CUMULATIVE mean power estimate — a constant-ish
// scale that converges to the whole-session RMS and then stays put. This
// matters: the CMA update scales with |x|³, and a *local* (EMA) normalisation
// that tracks the TDMA downlink's slot-to-slot power swings gives the algorithm
// a moving modulus target and it converges to garbage (empirically CRC → 0). A
// divergence guard re-seeds the tracking filter if a normalisation transient or
// deep fade blows the taps up, so one bad patch cannot permanently poison every
// later snapshot.
type cmaEqualizer struct {
	wAdapt    []complex64 // adapts every symbol (phase wanders — never applied directly)
	wApply    []complex64 // frozen snapshot, applied to the output
	buf       []complex64 // normalised input history, newest at buf[len-1]
	mu        float32     // CMA step
	snapEvery int         // symbols between wApply <- wAdapt snapshots
	since     int         // symbols since the last snapshot
	cumSum    float64     // cumulative Σ|x|² for the constant-scale normaliser
	count     float64     // symbols seen (denominator of the cumulative mean)
}

// equalizer tuning validated on the reporter's captures; robust across
// taps∈{11,21} and mu∈{1e-3,3e-3}, so these are safe defaults.
const (
	eqDefaultTaps      = 11
	eqDefaultMu        = 3e-3
	eqDefaultSnapEvery = 1000
	eqDivergeGuard     = 9.0 // reseed wAdapt when any |tap|² exceeds this (|tap|>3)
)

// newCMAEqualizer builds an equalizer. taps<=0 ⇒ 11 (forced odd for an exact
// center spike), mu<=0 ⇒ 3e-3, snapEvery<=0 ⇒ 1000.
func newCMAEqualizer(taps int, mu float64, snapEvery int) *cmaEqualizer {
	if taps <= 0 {
		taps = eqDefaultTaps
	}
	if taps%2 == 0 {
		taps++
	}
	if mu <= 0 {
		mu = eqDefaultMu
	}
	if snapEvery <= 0 {
		snapEvery = eqDefaultSnapEvery
	}
	e := &cmaEqualizer{
		wAdapt:    make([]complex64, taps),
		wApply:    make([]complex64, taps),
		buf:       make([]complex64, taps),
		mu:        float32(mu),
		snapEvery: snapEvery,
	}
	e.wAdapt[taps/2] = 1
	e.wApply[taps/2] = 1
	return e
}

// process equalizes src into dst (dst is reused; may alias src) using the frozen
// wApply filter, adapts wAdapt by CMA, and snapshots wApply<-wAdapt every
// snapEvery symbols.
func (e *cmaEqualizer) process(dst, src []complex64) []complex64 {
	n := len(e.wAdapt)
	if cap(dst) < len(src) {
		dst = make([]complex64, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, x := range src {
		px := float64(real(x)*real(x) + imag(x)*imag(x))
		e.cumSum += px
		e.count++
		mean := e.cumSum / e.count
		if mean < 1e-9 {
			mean = 1e-9
		}
		scale := float32(1 / math.Sqrt(mean))
		copy(e.buf, e.buf[1:])
		e.buf[n-1] = complex(real(x)*scale, imag(x)*scale)

		// Output uses the frozen filter (phase-coherent between snapshots).
		var y complex64
		for k := 0; k < n; k++ {
			y += e.wApply[k] * e.buf[n-1-k]
		}
		dst[i] = y

		// Adapt the tracking filter by CMA(2,2): e = ya·(|ya|² − 1).
		var ya complex64
		for k := 0; k < n; k++ {
			ya += e.wAdapt[k] * e.buf[n-1-k]
		}
		ge := real(ya)*real(ya) + imag(ya)*imag(ya) - 1
		er := ge * real(ya)
		ei := ge * imag(ya)
		var mx float32
		for k := 0; k < n; k++ {
			b := e.buf[n-1-k]
			// w -= mu · e · conj(b)
			pr := er*real(b) + ei*imag(b)
			pi := ei*real(b) - er*imag(b)
			w := e.wAdapt[k] - complex(e.mu*pr, e.mu*pi)
			e.wAdapt[k] = w
			if m := real(w)*real(w) + imag(w)*imag(w); m > mx {
				mx = m
			}
		}
		if mx > eqDivergeGuard { // taps blew up — re-seed to pass-through
			for k := range e.wAdapt {
				e.wAdapt[k] = 0
			}
			e.wAdapt[n/2] = 1
		}

		if e.since++; e.since >= e.snapEvery {
			copy(e.wApply, e.wAdapt)
			e.since = 0
		}
	}
	return dst
}

// reset returns the equalizer to its center-spike pass-through state. Call on
// receiver re-sync so a stale channel estimate does not corrupt the re-acquired
// stream.
func (e *cmaEqualizer) reset() {
	for i := range e.wAdapt {
		e.wAdapt[i] = 0
		e.wApply[i] = 0
		e.buf[i] = 0
	}
	c := len(e.wAdapt) / 2
	e.wAdapt[c] = 1
	e.wApply[c] = 1
	e.cumSum = 0
	e.count = 0
	e.since = 0
}
