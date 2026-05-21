package demod

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// C4FM is a four-level continuous-phase FSK demodulator (P25 Phase 1 control
// channel). Operates on a real input stream produced by an FM discriminator;
// applies a matched filter and slices to the four-level alphabet
// {+3, +1, -1, -3} (multiplied by a deviation scale).
type C4FM struct {
	taps      []float32
	hist      []float32
	histPos   int
	deviation float32
}

// NewC4FM returns a C4FM demod whose matched filter is an RRC with the given
// samples-per-symbol, span (in symbols), and roll-off α. The deviation
// scales slicer thresholds to the application; for P25 with 4800 sym/s and
// ±1.8 kHz outer deviation, downstream code should set this empirically.
//
// Note: an RRC matched filter is correct only for an RRC-shaped transmit
// signal. P25 Phase 1 C4FM is not RRC-shaped — use NewC4FMWithTaps with
// P25C4FMRxTaps for it (see c4fm_p25.go).
func NewC4FM(sps, span int, alpha, deviation float64) *C4FM {
	return NewC4FMWithTaps(filter.RootRaisedCosine(sps, span, alpha), deviation)
}

// NewC4FMWithTaps returns a C4FM demod whose matched filter uses the
// supplied FIR taps directly, instead of the default root-raised-cosine.
// P25 Phase 1 uses this with the spec C4FM receive filter (P25C4FMRxTaps).
func NewC4FMWithTaps(taps []float32, deviation float64) *C4FM {
	return &C4FM{taps: taps, hist: make([]float32, len(taps)), deviation: float32(deviation)}
}

// MatchedFilter applies the matched filter and returns a same-length output.
func (c *C4FM) MatchedFilter(dst, src []float32) []float32 {
	if cap(dst) < len(src) {
		dst = make([]float32, len(src))
	} else {
		dst = dst[:len(src)]
	}
	N := len(c.taps)
	for i, x := range src {
		c.hist[c.histPos] = x
		c.histPos = (c.histPos + 1) % N
		var acc float32
		idx := c.histPos - 1
		if idx < 0 {
			idx = N - 1
		}
		for k := 0; k < N; k++ {
			acc += c.taps[k] * c.hist[idx]
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
