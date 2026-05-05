package demod

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// C4FM is a four-level continuous-phase FSK demodulator (P25 Phase 1 control
// channel). Operates on a real input stream produced by an FM discriminator;
// applies an RRC matched filter and slices to the four-level alphabet
// {+3, +1, -1, -3} (multiplied by a deviation scale).
type C4FM struct {
	rrc      []float32
	hist     []float32
	histPos  int
	deviation float32
}

// NewC4FM returns a C4FM demod whose matched filter is an RRC with the given
// samples-per-symbol, span (in symbols), and roll-off α. The deviation
// scales slicer thresholds to the application; for P25 with 4800 sym/s and
// ±1.8 kHz outer deviation, downstream code should set this empirically.
func NewC4FM(sps, span int, alpha, deviation float64) *C4FM {
	rrc := filter.RootRaisedCosine(sps, span, alpha)
	return &C4FM{rrc: rrc, hist: make([]float32, len(rrc)), deviation: float32(deviation)}
}

// MatchedFilter applies the RRC filter and returns a same-length output.
func (c *C4FM) MatchedFilter(dst, src []float32) []float32 {
	if cap(dst) < len(src) {
		dst = make([]float32, len(src))
	} else {
		dst = dst[:len(src)]
	}
	N := len(c.rrc)
	for i, x := range src {
		c.hist[c.histPos] = x
		c.histPos = (c.histPos + 1) % N
		var acc float32
		idx := c.histPos - 1
		if idx < 0 {
			idx = N - 1
		}
		for k := 0; k < N; k++ {
			acc += c.rrc[k] * c.hist[idx]
			idx--
			if idx < 0 {
				idx = N - 1
			}
		}
		dst[i] = acc
	}
	return dst
}

// Slice maps a soft sample to the four C4FM symbols {-3, -1, +1, +3}.
// Threshold spacing is 2*deviation/3 so that ±deviation lands at ±1.5*scale.
func (c *C4FM) Slice(soft float32) int {
	d := c.deviation
	switch {
	case soft >= 2*d/3:
		return 3
	case soft >= 0:
		return 1
	case soft >= -2*d/3:
		return -1
	default:
		return -3
	}
}

// SliceMany applies Slice to a slice of soft samples.
func (c *C4FM) SliceMany(dst []int8, src []float32) []int8 {
	if cap(dst) < len(src) {
		dst = make([]int8, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		dst[i] = int8(c.Slice(s))
	}
	return dst
}
