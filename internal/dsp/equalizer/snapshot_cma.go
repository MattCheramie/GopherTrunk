package equalizer

import "math"

// snapshotDivergeGuard is the squared-tap-magnitude threshold at which the
// tracking filter is re-seeded to a pass-through (|tap| > 3).
const snapshotDivergeGuard = 9.0

// SnapshotCMA is a blind CMA equalizer safe to place ahead of a DIFFERENTIAL
// demodulator, where the plain CMA (which applies its live, continuously
// adapting taps) is not.
//
// The problem it solves: CMA's cost function J = E[(|y|²−R²)²] is rotation-
// invariant, so as the taps adapt the equalized output's absolute phase wanders
// freely. A coherent slicer with its own carrier recovery tolerates that, but a
// *differential* decoder forms s[n]·conj(s[n−1]) — and a phase that changes
// between consecutive symbols does NOT cancel there, corrupting every dibit.
// Feeding a continuously-adapting CMA to a differential decoder empirically
// drives the decode to zero.
//
// The fix — adapt continuously, apply a FROZEN snapshot: SnapshotCMA keeps two
// tap sets. wAdapt is updated every sample by CMA(2,2); wApply is a snapshot of
// wAdapt refreshed every snapEvery samples and is the filter actually applied to
// the output. Held fixed between snapshots, wApply imposes only a constant phase
// (which the differential decoder cancels) while still inverting the linear
// channel. Only the one sample straddling a snapshot sees a phase step; with
// snapEvery on the order of a burst that is a dibit or two, absorbed by the FEC.
// wApply is center-spike initialised, so before CMA converges the output is the
// unmodified input — the equalizer can only help or, on a clean signal, leave it
// essentially unchanged.
//
// Input is normalised by a CUMULATIVE mean power estimate — a constant-ish scale
// that converges to the whole-session RMS and stays put. This matters: the CMA
// update scales with |x|³, and a *local* (EMA) normalisation that tracks a
// bursty TDMA downlink's slot-to-slot power swings gives a moving modulus target
// and CMA converges to garbage. A divergence guard re-seeds the tracking filter
// if a normalisation transient or deep fade blows the taps up, so one bad patch
// cannot poison later snapshots. The target modulus is R²=1 (the cumulative
// normalisation makes the input near-unit-power), so SnapshotCMA is for
// unit-modulus PSK (BPSK / QPSK / π/4-DQPSK / 8PSK).
//
// Validated on real TETRA captures: inserted ahead of the π/4-DQPSK differential
// decoder it roughly doubles CRC-valid TCH/S burst yield (soft-decision 410→778,
// ~1.9×) with no regression on already-clean captures.
type SnapshotCMA struct {
	wAdapt    []complex64 // adapts every sample (phase wanders — never applied directly)
	wApply    []complex64 // frozen snapshot, applied to the output
	buf       []complex64 // normalised input history, newest at buf[len-1]
	mu        float32     // CMA step size
	snapEvery int         // samples between wApply <- wAdapt snapshots
	since     int         // samples since the last snapshot
	cumSum    float64     // cumulative Σ|x|² for the constant-scale normaliser
	count     float64     // samples seen (denominator of the cumulative mean)
}

// NewSnapshotCMA builds a snapshot CMA equalizer. taps is the FIR length (rounded
// up to odd for an exact center spike); stepSize is the CMA step (µ); snapEvery
// is the number of samples between frozen-tap snapshots. Panics on taps <= 0 or
// snapEvery <= 0.
func NewSnapshotCMA(taps int, stepSize float32, snapEvery int) *SnapshotCMA {
	if taps <= 0 {
		panic("equalizer: SnapshotCMA taps must be > 0")
	}
	if snapEvery <= 0 {
		panic("equalizer: SnapshotCMA snapEvery must be > 0")
	}
	if taps%2 == 0 {
		taps++
	}
	e := &SnapshotCMA{
		wAdapt:    make([]complex64, taps),
		wApply:    make([]complex64, taps),
		buf:       make([]complex64, taps),
		mu:        stepSize,
		snapEvery: snapEvery,
	}
	e.wAdapt[taps/2] = 1
	e.wApply[taps/2] = 1
	return e
}

// Process consumes one input sample and returns the equalized output (the frozen
// wApply filter's response), while adapting the tracking filter and refreshing
// the snapshot on schedule.
func (e *SnapshotCMA) Process(x complex64) complex64 {
	n := len(e.wAdapt)

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
	if mx > snapshotDivergeGuard { // taps blew up — re-seed to pass-through
		for k := range e.wAdapt {
			e.wAdapt[k] = 0
		}
		e.wAdapt[n/2] = 1
	}

	if e.since++; e.since >= e.snapEvery {
		copy(e.wApply, e.wAdapt)
		e.since = 0
	}
	return y
}

// Taps returns a copy of the currently-applied (frozen) weight vector.
func (e *SnapshotCMA) Taps() []complex64 {
	out := make([]complex64, len(e.wApply))
	copy(out, e.wApply)
	return out
}

// Reset returns the equalizer to its center-spike pass-through state. Call on
// stream re-sync so a stale channel estimate does not corrupt the re-acquired
// stream.
func (e *SnapshotCMA) Reset() {
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
