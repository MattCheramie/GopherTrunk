package mbe

import (
	"math"
	"testing"
)

// TestDefaultAGCConfigValues pins the field-tested defaults so an
// accidental edit (e.g. re-introducing the MinGain=10 floor that forced
// 10× over-amplification) trips a test rather than shipping over-driven,
// clipped audio.
func TestDefaultAGCConfigValues(t *testing.T) {
	d := DefaultAGCConfig()
	if d.TargetPeak != 18000 {
		t.Errorf("TargetPeak = %v, want 18000", d.TargetPeak)
	}
	if d.Attack != 0.5 {
		t.Errorf("Attack = %v, want 0.5", d.Attack)
	}
	if d.MinGain != 0.05 {
		t.Errorf("MinGain = %v, want 0.05 (a high floor over-amplifies into the limiter)", d.MinGain)
	}
	if d.MinGain >= 1 {
		t.Errorf("MinGain = %v must be < 1 so the AGC can attenuate already-loud frames", d.MinGain)
	}
}

// TestSoftLimitBounds checks softLimit never reaches the int16 rail and
// passes sub-knee samples through unchanged.
func TestSoftLimitBounds(t *testing.T) {
	for _, x := range []float64{0, 1000, softLimitKnee - 1, softLimitKnee} {
		if y, limited := softLimit(x); limited || y != x {
			t.Errorf("softLimit(%v) = (%v, %v), want (%v, false)", x, y, limited, x)
		}
	}
	for _, x := range []float64{30000, 50000, 200000, 1e7} {
		y, limited := softLimit(x)
		if !limited {
			t.Errorf("softLimit(%v): limited = false, want true", x)
		}
		if y > softLimitRail {
			t.Errorf("softLimit(%v) = %v, must not exceed rail %v", x, y, softLimitRail)
		}
		if y2, _ := softLimit(-x); y2 != -y {
			t.Errorf("softLimit not odd-symmetric: %v vs %v", y2, -y)
		}
	}
	// A realistic overshoot (a few× the knee) is smoothly compressed and
	// stays strictly below the rail — no abrupt clip corner.
	if y, _ := softLimit(45000); y >= softLimitRail {
		t.Errorf("softLimit(45000) = %v, want strictly below rail %v", y, softLimitRail)
	}
}

// TestAGCNeverHardClips drives the AGC with a signal that, at the applied
// gain, would blow well past the int16 rail and asserts the output is
// strictly inside int16 range (the soft limiter compresses rather than
// hard-clipping).
func TestAGCNeverHardClips(t *testing.T) {
	a := NewAGC(DefaultAGCConfig())
	pcm := make([]float64, SamplesPerFrame)
	out := make([]int16, SamplesPerFrame)
	// A quiet seed frame then a sudden loud frame, to provoke overshoot.
	for i := range pcm {
		pcm[i] = 50 * math.Sin(float64(i))
	}
	a.Apply(pcm, out, false)
	for i := range pcm {
		pcm[i] = 12000 * math.Sin(float64(i)) // loud onset
	}
	a.Apply(pcm, out, false)
	for _, s := range out {
		if s == 32767 || s == -32768 {
			t.Fatalf("output hit the int16 rail (%d) — soft limiter should keep it inside", s)
		}
	}
	if a.LastClipSamples() == 0 {
		t.Log("note: no samples engaged the limiter on this frame (acceptable)")
	}
}

// TestAGCAttenuatesLoudFrames confirms the AGC can apply gain < 1 to a
// frame already louder than TargetPeak — impossible under the old
// MinGain=10 floor, which forced amplification and clipping.
func TestAGCAttenuatesLoudFrames(t *testing.T) {
	a := NewAGC(DefaultAGCConfig())
	pcm := make([]float64, SamplesPerFrame)
	out := make([]int16, SamplesPerFrame)
	for i := range pcm {
		pcm[i] = 30000 * math.Sin(float64(i)) // peak 30000 > TargetPeak 18000
	}
	a.Apply(pcm, out, false)
	if g := a.LastGain(); g >= 1 {
		t.Errorf("gain = %v on a 30000-peak frame; want < 1 (attenuation)", g)
	}
}

// TestAGCPreservesDynamics feeds frames whose per-frame peak alternates
// loud/quiet and checks the output retains a healthy crest factor (the
// over-driven AGC flattened speech to crest ~3). Fast-attack/slow-release
// keeps the loud frames louder than the quiet ones in the output.
func TestAGCPreservesDynamics(t *testing.T) {
	a := NewAGC(DefaultAGCConfig())
	pcm := make([]float64, SamplesPerFrame)
	out := make([]int16, SamplesPerFrame)
	var sumSq, peak float64
	var n int
	for f := 0; f < 60; f++ {
		amp := 2000.0
		if f%4 == 0 {
			amp = 12000.0 // periodic loud frame
		}
		for i := range pcm {
			pcm[i] = amp * math.Sin(2*math.Pi*float64(i)/16)
		}
		a.Apply(pcm, out, false)
		for _, s := range out {
			v := float64(s)
			sumSq += v * v
			if av := math.Abs(v); av > peak {
				peak = av
			}
			n++
		}
	}
	rms := math.Sqrt(sumSq / float64(n))
	crest := peak / rms
	if crest < 2.0 {
		t.Errorf("crest factor = %.2f, want > 2.0 (dynamics flattened — the over-AGC regression)", crest)
	}
}
