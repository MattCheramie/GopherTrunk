package equalizer

// SnapshotLMS is a training-sequence-aided linear equalizer that is safe to place
// ahead of a DIFFERENTIAL demodulator — the trained counterpart to SnapshotCMA.
//
// Where SnapshotCMA adapts BLINDLY to a constant-modulus target, SnapshotLMS
// trains on a burst's KNOWN training sequence (the π/4-DQPSK normal/synchronisation
// midamble). That difference matters: CMA's cost J = E[(|y|²−R²)²] is satisfied by
// any equalizer output that has the right modulus, so it has spurious minima and
// can converge to a constant-modulus but WRONG solution — the failure the package
// history records (a numerically-unstable CMA once drove differential EVM 34%→8%
// while CRC stayed 0). Training to a known reference removes that ambiguity: the
// LMS error e = d − y is minimised only when the composite channel·equalizer maps
// the reference back to itself, so a good training sequence pins the true channel
// inverse (phase and all), not merely its modulus. This is the #1001 follow-up:
// per-burst training-sequence-aided equalization for the same-carrier TETRA voice
// that blind CMA only partially recovers.
//
// It shares SnapshotCMA's differential-safety principle: ADAPT on the training
// sequence, then APPLY a FROZEN snapshot of the taps to the whole burst. A held-
// constant filter imposes only a constant phase/scale, which cancels in the
// downstream s[n]·conj(s[n−1]) differential decode; a continuously-adapting filter
// would not (its inter-symbol phase drift corrupts every dibit). So the intended
// use is: Train(midamble_rx, midamble_ref) once per burst, then Equalize the whole
// burst's symbols through the frozen taps before the differential decoder.
//
// The taps are centre-spike initialised, so before training (or on a benign
// channel with a short/absent reference) Equalize is a near pass-through — the
// equalizer can only help or leave a clean burst essentially unchanged. The
// forward FIR convention matches LMS exactly (tap 0 multiplies the newest sample),
// so a snapshot of an LMS weight vector means the same thing in Equalize.
type SnapshotLMS struct {
	adapt   *LMS        // trained on the reference; never applied while adapting
	apply   []complex64 // frozen snapshot of adapt's taps, applied to the burst
	hist    []complex64 // Equalize delay line (newest written at histPos-1)
	histPos int
}

// NewSnapshotLMS builds a training-sequence equalizer with taps complex weights
// and LMS step size stepSize. taps must be > 0; an odd count gives a well-defined
// centre spike. Panics on taps <= 0 (mirroring NewLMS).
func NewSnapshotLMS(taps int, stepSize float32) *SnapshotLMS {
	e := &SnapshotLMS{
		adapt: NewLMS(taps, stepSize),
		apply: make([]complex64, taps),
		hist:  make([]complex64, taps),
	}
	e.apply[taps/2] = 1 // centre-spike pass-through until trained
	return e
}

// Train adapts the tracking filter over the aligned (received, reference) symbol
// pairs, then freezes the result into the applied taps. rx[i] is the received
// symbol for reference symbol ref[i]; both must be the same length (the shorter
// prevails). passes repeats the sweep over the reference so a short midamble still
// converges (1 for a long training sequence, several for a burst-length one); a
// passes < 1 is treated as 1. Training does NOT reset the adapt taps, so repeated
// Train calls refine the same estimate — call Reset first for a fresh channel.
func (e *SnapshotLMS) Train(rx, ref []complex64, passes int) {
	n := len(rx)
	if len(ref) < n {
		n = len(ref)
	}
	if n == 0 {
		return
	}
	if passes < 1 {
		passes = 1
	}
	for p := 0; p < passes; p++ {
		for i := 0; i < n; i++ {
			e.adapt.Process(rx[i], ref[i])
		}
	}
	copy(e.apply, e.adapt.Taps())
}

// Equalize applies the frozen taps to src as a plain FIR, appending the equalized
// symbols to dst[:0] and returning it (dst may be nil or reused across bursts).
// The delay line persists across calls so a burst can be streamed in chunks; call
// Reset between independent bursts. Using the frozen taps for the whole burst is
// what keeps the output safe for a differential decoder (constant phase cancels).
func (e *SnapshotLMS) Equalize(dst, src []complex64) []complex64 {
	dst = dst[:0]
	n := len(e.apply)
	for _, x := range src {
		e.hist[e.histPos] = x
		e.histPos = (e.histPos + 1) % n

		var yr, yi float32
		idx := e.histPos - 1
		if idx < 0 {
			idx = n - 1
		}
		for i := 0; i < n; i++ {
			hr, hi := real(e.hist[idx]), imag(e.hist[idx])
			tr, ti := real(e.apply[i]), imag(e.apply[i])
			yr += tr*hr - ti*hi
			yi += tr*hi + ti*hr
			idx--
			if idx < 0 {
				idx = n - 1
			}
		}
		dst = append(dst, complex(yr, yi))
	}
	return dst
}

// Taps returns a copy of the currently-applied (frozen) weight vector.
func (e *SnapshotLMS) Taps() []complex64 {
	out := make([]complex64, len(e.apply))
	copy(out, e.apply)
	return out
}

// Reset returns the equalizer to its centre-spike pass-through state and clears
// the Equalize delay line — call between independent bursts / on re-sync so a
// stale channel estimate does not corrupt the next burst.
func (e *SnapshotLMS) Reset() {
	e.adapt.Reset()
	for i := range e.apply {
		e.apply[i] = 0
		e.hist[i] = 0
	}
	e.apply[len(e.apply)/2] = 1
	e.histPos = 0
}
