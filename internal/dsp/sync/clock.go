// Package sync provides symbol-time recovery and frame sync correlators.
package sync

// MuellerMuller is a feedback symbol-timing recovery loop for real-valued
// PAM signals. The loop adjusts a sub-sample symbol clock toward the
// optimum sampling instant by minimizing |s[n] - sgn(s[n-1])*s[mid]|.
//
// Inputs are oversampled samples (e.g. 8 sps after the matched filter).
// Output is one sample per recovered symbol.
type MuellerMuller struct {
	sps     float64 // nominal samples per symbol
	mu      float64 // current sub-sample phase in [0, sps)
	gain    float64 // loop gain
	prevSym float32
	prevMid float32
	have    bool
	// prevTail is the last sample of the previous Process chunk and
	// havePrev marks it valid. They let a symbol boundary landing on
	// src[0] of a later chunk interpolate against the correct preceding
	// sample. Without them the walk skipped src[0] on every call,
	// losing one sample of clock phase per chunk — on RTL-realistic
	// ~19-symbol chunks that drifted the C4FM symbol timing and
	// scattered dibit errors (issue #275). The first call has no
	// look-back, so it starts at src[1]; contiguous single-call
	// behaviour is unchanged.
	prevTail float32
	havePrev bool
}

func NewMuellerMuller(sps, gain float64) *MuellerMuller {
	if sps < 2 {
		panic("mm: sps must be >= 2")
	}
	if gain <= 0 {
		gain = 0.1
	}
	return &MuellerMuller{sps: sps, gain: gain, mu: sps}
}

// Process consumes oversampled real samples and emits one recovered symbol
// per nominal symbol period. dst is reused if it has capacity. The symbol
// clock carries across calls, so a long stream may be processed in chunks
// without the recovered symbol count depending on the chunk size: prevTail
// bridges each chunk boundary so src[0] of a continuation chunk is a real
// clock step rather than being skipped.
func (m *MuellerMuller) Process(dst []float32, src []float32) []float32 {
	dst, _ = m.ProcessGated(dst, nil, src, nil)
	return dst
}

// ProcessGated is Process with a per-sample carrier-presence gate. present,
// when non-nil, must be len(src) long; a symbol interpolated at a sample whose
// flag is false is still emitted (the stream stays contiguous for the
// downstream framers) but does NOT drive the timing-error update — the loop
// coasts at its nominal period, holding the symbol phase it had when the
// carrier dropped. When present is non-nil the returned symPresent carries one
// flag per emitted symbol (aligned with dst; the passed slice is reused when it
// has capacity). A nil present is all-true, which makes this byte-identical to
// Process, and returns a nil symPresent.
//
// The gate exists for burst-mode carriers. A DMR direct-mode (Tier I /
// simplex) MS transmits one 27.5 ms burst per 60 ms frame and is off for the
// other 32.5 ms; the FM discriminator of the receiver noise in that gap is
// uniformly distributed over ±π, so the matched-filter output there is
// several times LARGER than the signal's own symbol levels and the
// Mueller-Müller error term random-walks the phase between every burst.
// Freezing the update on absent samples keeps the phase the previous burst
// converged to — all bursts of one transmission share one clock (issue #836).
func (m *MuellerMuller) ProcessGated(dst []float32, symPresent []bool, src []float32, present []bool) ([]float32, []bool) {
	if cap(dst) < len(src) {
		dst = make([]float32, 0, len(src)/int(m.sps)+1)
	} else {
		dst = dst[:0]
	}
	if present == nil {
		symPresent = nil
	} else if cap(symPresent) < cap(dst) {
		symPresent = make([]bool, 0, cap(dst))
	} else {
		symPresent = symPresent[:0]
	}
	// On the first call there is no look-back sample, so the walk
	// starts at src[1] (src[0] only seeds the interpolation), leaving
	// single-call behaviour unchanged. Later calls carry the previous
	// chunk's last sample in prevTail, so src[0] is a real clock step
	// and the recovered symbol stream no longer depends on how the IQ
	// was chunked (issue #275).
	start := 0
	if !m.havePrev {
		start = 1
	}
	for i := start; i < len(src); i++ {
		m.mu -= 1.0
		if m.mu > 0 {
			continue
		}
		// We've crossed a symbol boundary; interpolate at this point
		// between the previous sample and src[i]. For i == 0 the
		// previous sample is the last sample of the previous chunk.
		prev := m.prevTail
		if i > 0 {
			prev = src[i-1]
		}
		frac := 1.0 + m.mu // mu is in (-1, 0]; frac is in (0, 1]
		sym := float32(float64(prev)*(1-frac) + float64(src[i])*frac)

		on := present == nil || present[i]
		if present != nil {
			symPresent = append(symPresent, on)
		}
		switch {
		case !on:
			// No carrier under this symbol: coast at the nominal period and
			// keep prevSym / have as they were, so the first present symbol of
			// the next burst forms its error against the last present one.
			m.mu += m.sps
			dst = append(dst, sym)
			continue
		case m.have:
			// Mueller-Muller error: e = sgn(prev)*current - sgn(current)*prev
			err := sgn(m.prevSym)*float64(sym) - sgn(sym)*float64(m.prevSym)
			m.mu += m.sps + m.gain*err
		default:
			m.mu += m.sps
			m.have = true
		}
		m.prevSym = sym
		dst = append(dst, sym)
	}
	if len(src) > 0 {
		m.prevTail = src[len(src)-1]
		m.havePrev = true
	}
	return dst, symPresent
}

// Reset returns the loop to its just-constructed state, so a downstream
// consumer that applies a large mid-stream correction (e.g. a coarse
// carrier de-rotation) can re-acquire symbol timing from scratch on the
// corrected signal instead of carrying a phase parked against the old one.
// The nominal sps and gain are preserved.
func (m *MuellerMuller) Reset() {
	m.mu = m.sps
	m.prevSym = 0
	m.prevMid = 0
	m.have = false
	m.prevTail = 0
	m.havePrev = false
}

// Mu returns the current sub-sample phase accumulator (rad-equivalent;
// in (-1, sps] depending on where the loop is in the symbol period).
// At steady state on a noise-free signal mu cycles deterministically
// around the symbol period; a slow monotonic drift indicates the
// nominal sps does not match the stream's actual sample-rate / baud
// ratio. Exposed read-only for issue-#402-style diagnostics where the
// daemon (or replay) periodically logs the loop's internal state so a
// persistent clock slip can be distinguished from a slicer / AFC
// failure. Not safe for concurrent calls with Process.
func (m *MuellerMuller) Mu() float64 { return m.mu }

// SPS returns the nominal samples-per-symbol the loop was constructed
// with. Paired with Mu() so a diagnostic log line can render mu as a
// fraction of the symbol period without re-deriving the construction
// value.
func (m *MuellerMuller) SPS() float64 { return m.sps }

func sgn(x float32) float64 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
