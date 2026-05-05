// Package sync provides symbol-time recovery and frame sync correlators.
package sync

// MuellerMuller is a feedback symbol-timing recovery loop for real-valued
// PAM signals. The loop adjusts a sub-sample symbol clock toward the
// optimum sampling instant by minimizing |s[n] - sgn(s[n-1])*s[mid]|.
//
// Inputs are oversampled samples (e.g. 8 sps after the matched filter).
// Output is one sample per recovered symbol.
type MuellerMuller struct {
	sps      float64 // nominal samples per symbol
	mu       float64 // current sub-sample phase in [0, sps)
	gain     float64 // loop gain
	prevSym  float32
	prevMid  float32
	have     bool
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
// per nominal symbol period. dst is reused if it has capacity.
func (m *MuellerMuller) Process(dst []float32, src []float32) []float32 {
	if cap(dst) < len(src) {
		dst = make([]float32, 0, len(src)/int(m.sps)+1)
	} else {
		dst = dst[:0]
	}
	for i := 1; i < len(src); i++ {
		m.mu -= 1.0
		if m.mu > 0 {
			continue
		}
		// We've crossed a symbol boundary; interpolate at this point.
		// Use linear interpolation between src[i-1] and src[i].
		frac := 1.0 + m.mu // mu is in (-1, 0]; frac is in (0, 1]
		sym := float32(float64(src[i-1])*(1-frac) + float64(src[i])*frac)

		if m.have {
			// Mueller-Muller error: e = sgn(prev)*current - sgn(current)*prev
			err := sgn(m.prevSym)*float64(sym) - sgn(float32(sym))*float64(m.prevSym)
			m.mu += m.sps + m.gain*err
		} else {
			m.mu += m.sps
			m.have = true
		}
		m.prevSym = sym
		dst = append(dst, sym)
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
