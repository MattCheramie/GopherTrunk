package spectrum

import (
	"math"
	"sort"
)

// StitchMember is one member SDR of a stitched (sdr.groups) spectrum: the
// slice of the group's span it owns in the panorama. Adjacent members'
// windows overlap; each owns its window up to the middle of each overlap.
type StitchMember struct {
	OwnLowHz, OwnHighHz uint32
}

// Stitcher builds one panorama frame from the latest frame of each member
// of an SDR group. The members are separate streams on separate clocks, so
// this is a display composite, not a coherent wideband spectrum: every
// output bin is copied from the one member that owns its frequency.
//
// Three things keep the composite honest:
//   - each member's DC spur (its centre bins) is replaced by an
//     interpolation of its neighbours, since it is tuner leakage, not RF;
//   - each member's level is matched to member 0's across the overlaps
//     (an EMA of the median bin difference), so a different gain on one
//     stick does not leave a step at the seam;
//   - a member with no frame yet is filled at the frame's floor, never
//     NaN, so the wire stays finite.
type Stitcher struct {
	members []StitchMember
	binHz   float64
	lowHz   float64 // frequency of output bin 0
	nOut    int
	alpha   float64
	// offset[i] is the dB added to member i's bins (member 0 is the anchor).
	offset   []float64
	haveOff  []bool
	diffsBuf []float64
}

// stitchDCHalfBins is how many bins either side of a member's centre bin are
// replaced as DC spur.
const stitchDCHalfBins = 1

// stitchLevelAlpha is the EMA weight of one frame's seam-level estimate:
// slow enough that a burst in the overlap does not jolt the levels.
const stitchLevelAlpha = 0.05

// NewStitcher builds a stitcher for members that all stream at sampleRateHz
// with fftSize-point frames. members must be sorted by frequency and their
// owned slices must not overlap.
func NewStitcher(members []StitchMember, sampleRateHz uint32, fftSize int) *Stitcher {
	s := &Stitcher{
		members: append([]StitchMember(nil), members...),
		binHz:   float64(sampleRateHz) / float64(fftSize),
		alpha:   stitchLevelAlpha,
		offset:  make([]float64, len(members)),
		haveOff: make([]bool, len(members)),
	}
	if len(members) == 0 || s.binHz <= 0 {
		return s
	}
	lo := float64(members[0].OwnLowHz)
	hi := float64(members[len(members)-1].OwnHighHz)
	s.lowHz = lo
	s.nOut = int(math.Ceil((hi - lo + 1) / s.binHz))
	return s
}

// Span returns the panorama's centre frequency and width as the producer's
// frames report them: bins[0] sits at centre - span/2 and the spacing is one
// member bin.
func (s *Stitcher) Span() (centerHz, spanHz uint32) {
	span := float64(s.nOut) * s.binHz
	return uint32(math.Round(s.lowHz + span/2)), uint32(math.Round(span))
}

// Stitch composes the panorama from the latest frame of each member (in the
// stitcher's member order). A frame with no bins means "no frame yet".
func (s *Stitcher) Stitch(frames []Frame) []float32 {
	if s.nOut <= 0 || len(frames) != len(s.members) {
		return nil
	}
	// Despur each member's copy once.
	clean := make([][]float32, len(frames))
	for i, f := range frames {
		if len(f.Bins) == 0 {
			continue
		}
		clean[i] = despurDC(f.Bins)
	}
	s.updateLevels(frames, clean)

	out := make([]float32, s.nOut)
	floor := float32(math.Inf(1))
	m := 0
	for k := range out {
		f := s.lowHz + (float64(k)+0.5)*s.binHz
		for m+1 < len(s.members) && f > float64(s.members[m].OwnHighHz) {
			m++
		}
		v, ok := s.binAt(frames[m], clean[m], f)
		if !ok {
			out[k] = float32(math.NaN())
			continue
		}
		v += float32(s.offset[m])
		out[k] = v
		if v < floor {
			floor = v
		}
	}
	if math.IsInf(float64(floor), 1) {
		floor = -150
	}
	for k, v := range out {
		if v != v { // NaN: member missing, or outside its frame
			out[k] = floor
		}
	}
	return out
}

// binAt returns member frame fr's (despurred) value at frequency f.
func (s *Stitcher) binAt(fr Frame, bins []float32, f float64) (float32, bool) {
	if len(bins) == 0 || fr.SampleRate == 0 {
		return 0, false
	}
	n := len(bins)
	bw := float64(fr.SampleRate) / float64(n)
	idx := int(math.Floor((f - (float64(fr.CenterHz) - float64(fr.SampleRate)/2)) / bw))
	if idx < 0 || idx >= n {
		return 0, false
	}
	return bins[idx], true
}

// updateLevels refreshes each member's level offset from the overlap with the
// member below it: the median over the shared bins of (lower member, offset
// applied) − (this member), EMA-smoothed.
func (s *Stitcher) updateLevels(frames []Frame, clean [][]float32) {
	if len(frames) > 0 && len(clean[0]) > 0 {
		s.haveOff[0] = true
	}
	for i := 1; i < len(frames); i++ {
		a, b := frames[i-1], frames[i]
		if len(clean[i-1]) == 0 || len(clean[i]) == 0 || !s.haveOff[i-1] {
			continue
		}
		lo := math.Max(float64(b.CenterHz)-float64(b.SampleRate)/2, float64(a.CenterHz)-float64(a.SampleRate)/2)
		hi := math.Min(float64(a.CenterHz)+float64(a.SampleRate)/2, float64(b.CenterHz)+float64(b.SampleRate)/2)
		if hi <= lo {
			continue
		}
		s.diffsBuf = s.diffsBuf[:0]
		for f := lo + s.binHz/2; f < hi; f += s.binHz {
			va, okA := s.binAt(a, clean[i-1], f)
			vb, okB := s.binAt(b, clean[i], f)
			if okA && okB {
				s.diffsBuf = append(s.diffsBuf, float64(va)+s.offset[i-1]-float64(vb))
			}
		}
		if len(s.diffsBuf) < 8 {
			continue
		}
		sort.Float64s(s.diffsBuf)
		med := s.diffsBuf[len(s.diffsBuf)/2]
		if !s.haveOff[i] {
			s.offset[i], s.haveOff[i] = med, true
		} else {
			s.offset[i] += s.alpha * (med - s.offset[i])
		}
	}
}

// despurDC returns a copy of bins with the centre (DC) bins replaced by a
// straight line between the bins just outside them.
func despurDC(bins []float32) []float32 {
	out := append([]float32(nil), bins...)
	n := len(out)
	c := n / 2
	lo, hi := c-stitchDCHalfBins-1, c+stitchDCHalfBins+1
	if lo < 0 || hi >= n {
		return out
	}
	a, b := out[lo], out[hi]
	for k := lo + 1; k < hi; k++ {
		t := float32(k-lo) / float32(hi-lo)
		out[k] = a + t*(b-a)
	}
	return out
}
