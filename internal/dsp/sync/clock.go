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
	if cap(dst) < len(src) {
		dst = make([]float32, 0, len(src)/int(m.sps)+1)
	} else {
		dst = dst[:0]
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

		if m.have {
			// Mueller-Muller error: e = sgn(prev)*current - sgn(current)*prev
			err := sgn(m.prevSym)*float64(sym) - sgn(sym)*float64(m.prevSym)
			m.mu += m.sps + m.gain*err
		} else {
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
	return dst
}

func sgn(x float32) float64 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
