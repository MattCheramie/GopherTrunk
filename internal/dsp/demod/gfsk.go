package demod

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// GFSK is a two-level Gaussian-FSK demodulator used by EDACS / GE-Marc
// (9.6 kbps control / data, BT = 0.3) and other classic GFSK
// trunked-radio signalling layers. It operates on a real input stream
// produced by an FM discriminator, applies a Gaussian matched filter
// matched to the transmitter's premod pulse, and slices to {0, 1}
// around zero.
//
// Pipeline: IQ → FM.Process → GFSK.MatchedFilter → clock recovery
// (e.g. sync.MuellerMuller) → GFSK.Slice / SliceMany.
//
// The matched filter's BT product should match the transmitter's
// premod filter. Pick BT = 0.3 for EDACS / GE-Marc; BT = 0.5 for
// Bluetooth-class GFSK.
type GFSK struct {
	gauss   []float32
	hist    []float32
	histPos int
}

// NewGFSK constructs a GFSK demod with the given samples-per-symbol,
// Gaussian span (in symbols), and BT product. Typical EDACS / GE-Marc
// parameters: sps depending on the upstream rate, span = 4, bt = 0.3.
// Panics if any argument is non-positive.
func NewGFSK(sps, span int, bt float64) *GFSK {
	g := filter.Gaussian(sps, span, bt)
	return &GFSK{gauss: g, hist: make([]float32, len(g))}
}

// MatchedFilter applies the Gaussian matched filter and returns a
// same-length output. Internal history carries across calls so chunk
// boundaries do not corrupt the stream.
func (g *GFSK) MatchedFilter(dst, src []float32) []float32 {
	if cap(dst) < len(src) {
		dst = make([]float32, len(src))
	} else {
		dst = dst[:len(src)]
	}
	N := len(g.gauss)
	for i, x := range src {
		g.hist[g.histPos] = x
		g.histPos = (g.histPos + 1) % N
		var acc float32
		idx := g.histPos - 1
		if idx < 0 {
			idx = N - 1
		}
		for k := 0; k < N; k++ {
			acc += g.gauss[k] * g.hist[idx]
			idx--
			if idx < 0 {
				idx = N - 1
			}
		}
		dst[i] = acc
	}
	return dst
}

// Slice maps a soft sample to a binary symbol: positive → 1,
// non-positive → 0. The slicer threshold is fixed at zero — GFSK is
// symmetric around DC so there is no bias to compensate for.
func (g *GFSK) Slice(soft float32) int {
	if soft > 0 {
		return 1
	}
	return 0
}

// SliceMany applies Slice to a slice of soft samples.
func (g *GFSK) SliceMany(dst []int8, src []float32) []int8 {
	if cap(dst) < len(src) {
		dst = make([]int8, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		dst[i] = int8(g.Slice(s))
	}
	return dst
}

// Reset clears the matched-filter history. Call on stream re-sync
// (control-channel hunt success, IQ underrun recovery) so stale
// samples don't bleed into the new stream.
func (g *GFSK) Reset() {
	for i := range g.hist {
		g.hist[i] = 0
	}
	g.histPos = 0
}
