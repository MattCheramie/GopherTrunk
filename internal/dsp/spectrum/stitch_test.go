package spectrum

import (
	"math"
	"testing"
)

const (
	stRate = 2_400_000
	stFFT  = 1024
	stHalf = 1_080_000 // a member's usable half-window at 2.4 MS/s
)

// stitchPair is two members 1.9 MHz apart with the planner's ownership rule
// (each owns its window up to the middle of the overlap).
func stitchPair() (members []StitchMember, centers []uint32) {
	centers = []uint32{851_500_000, 853_400_000}
	mid := (centers[0] + stHalf + centers[1] - stHalf) / 2
	members = []StitchMember{
		{OwnLowHz: centers[0] - stHalf, OwnHighHz: mid},
		{OwnLowHz: mid + 1, OwnHighHz: centers[1] + stHalf},
	}
	return members, centers
}

// memberFrame is a flat floor at level dB with a +40 dB spur on the centre
// (DC) bin and optional tones.
func memberFrame(center uint32, level float32, tones ...uint32) Frame {
	bins := make([]float32, stFFT)
	for i := range bins {
		bins[i] = level
	}
	bins[stFFT/2] = level + 40
	bw := float64(stRate) / stFFT
	for _, t := range tones {
		idx := int(math.Floor((float64(t) - (float64(center) - stRate/2)) / bw))
		if idx >= 0 && idx < stFFT {
			bins[idx] = level + 30
		}
	}
	return Frame{CenterHz: center, SampleRate: stRate, Bins: bins}
}

func outIndex(s *Stitcher, f uint32) int {
	c, span := s.Span()
	return int(math.Floor((float64(f) - (float64(c) - float64(span)/2)) / s.binHz))
}

func TestStitcherPlacesToneAndLevelsSeams(t *testing.T) {
	members, centers := stitchPair()
	s := NewStitcher(members, stRate, stFFT)
	// Member 1 runs 6 dB hotter; a tone sits on member 1 only, and one sits
	// in member 0's owned slice.
	tone0, tone1 := uint32(851_000_000), uint32(853_900_000)
	var out []float32
	for i := 0; i < 200; i++ {
		out = s.Stitch([]Frame{
			memberFrame(centers[0], -100, tone0),
			memberFrame(centers[1], -94, tone1),
		})
	}
	if len(out) == 0 {
		t.Fatal("no output")
	}
	c, span := s.Span()
	if lo := float64(c) - float64(span)/2; math.Abs(lo-float64(members[0].OwnLowHz)) > s.binHz {
		t.Errorf("panorama starts at %.0f Hz, want %d", lo, members[0].OwnLowHz)
	}
	for _, tone := range []uint32{tone0, tone1} {
		k := outIndex(s, tone)
		if out[k] < -75 {
			t.Errorf("tone %d Hz missing at output bin %d: %.1f dB", tone, k, out[k])
		}
	}
	// The 6 dB step is levelled: the floor either side of the seam agrees.
	seam := members[0].OwnHighHz
	below, above := out[outIndex(s, seam-50_000)], out[outIndex(s, seam+50_000)]
	if d := math.Abs(float64(below - above)); d > 0.5 {
		t.Errorf("floor steps %.2f dB at the seam (%.1f / %.1f)", d, below, above)
	}
	// Each member's DC spur is gone.
	for _, cf := range centers {
		if v := out[outIndex(s, cf)]; v > -99 {
			t.Errorf("DC spur at %d Hz survived: %.1f dB", cf, v)
		}
	}
}

func TestStitcherFillsMissingMemberFinite(t *testing.T) {
	members, centers := stitchPair()
	s := NewStitcher(members, stRate, stFFT)
	out := s.Stitch([]Frame{memberFrame(centers[0], -100), {}})
	if len(out) == 0 {
		t.Fatal("no output")
	}
	for k, v := range out {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Fatalf("bin %d = %v", k, v)
		}
	}
	if v := out[outIndex(s, centers[1])]; v != -100 {
		t.Errorf("missing member filled at %.1f dB, want the frame floor -100", v)
	}
}
