package ambe2

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// TestImplementsErrorAware pins that *Decoder satisfies voice.ErrorAware so
// the recorder hands it the per-frame FEC corrected-bit count.
func TestImplementsErrorAware(t *testing.T) {
	var _ voice.ErrorAware = New()
	var _ voice.ErrorAware = NewDMR()
}

// TestSetFrameErrorsZeroIsInert: feeding 0 corrected bits before each Decode
// produces output byte-identical to never calling SetFrameErrors — adaptive
// smoothing must not perturb clean audio. This is the no-regression guard.
func TestSetFrameErrorsZeroIsInert(t *testing.T) {
	frame := frameWithB2()
	a := New()
	b := New()
	for i := 0; i < 30; i++ {
		oa, err := a.Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		b.SetFrameErrors(0)
		ob, err := b.Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		for j := range oa {
			if oa[j] != ob[j] {
				t.Fatalf("frame %d sample %d differs: %d vs %d (smoothing not inert at 0 errors)", i, j, oa[j], ob[j])
			}
		}
	}
}

// TestSustainedErrorsMute: feeding a sustained high corrected-bit count drives
// the running error-rate estimate past the mute threshold, so the output
// decays to silence. (Real DMR Golay budgets rarely reach the threshold, so
// this exercises the safety-net path with an out-of-band count.)
func TestSustainedErrorsMute(t *testing.T) {
	d := New()
	frame := frameWithB2()
	var lastPeak int16
	for i := 0; i < 300; i++ {
		d.SetFrameErrors(30)
		out, err := d.Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		lastPeak = peak(out)
	}
	if lastPeak != 0 {
		t.Errorf("after sustained max errors, output peak = %d, want 0 (muted)", lastPeak)
	}
}

// TestSynthFrameDispersionAffectsVoiced: the §6.3 voiced-phase dispersion is
// wired into synthFrame. Two decoders with identical noise sources but
// different phase sources must diverge on a voiced frame — proof the
// per-harmonic phase offset (the metallic-buzz fix) actually perturbs the
// voiced synthesis. A coherent (no-dispersion) decoder would produce
// identical output regardless of the phase source.
func TestSynthFrameDispersionAffectsVoiced(t *testing.T) {
	p, M, log2M := voicedFrame()

	d1, d2 := New(), New() // identical rng (seed 0) and phaseRng
	// Diverge only the phase source; the unvoiced-noise source stays in sync.
	d2.phaseRng = rand.New(rand.NewSource(0xABCDEF))

	pcm1 := make([]float64, mbe.SamplesPerFrame)
	pcm2 := make([]float64, mbe.SamplesPerFrame)
	d1.synthFrame(p, &log2M, &M, pcm1)
	d2.synthFrame(p, &log2M, &M, pcm2)

	if equalF(pcm1, pcm2) {
		t.Fatal("voiced output identical across different phase sources; dispersion not applied")
	}
}

// TestSynthFrameNoDispersionWhenUnvoiced: dispersion is gated on voiced
// upper harmonics. With an all-unvoiced frame there are no voiced harmonics
// to disperse, so two decoders that differ only in their phase source (but
// share the noise source) must produce identical output.
func TestSynthFrameNoDispersionWhenUnvoiced(t *testing.T) {
	p, M, log2M := voicedFrame()
	for l := 1; l <= p.L; l++ {
		p.Vl[l] = 0 // make every harmonic unvoiced
	}

	d1, d2 := New(), New()
	d2.phaseRng = rand.New(rand.NewSource(0xABCDEF))

	pcm1 := make([]float64, mbe.SamplesPerFrame)
	pcm2 := make([]float64, mbe.SamplesPerFrame)
	d1.synthFrame(p, &log2M, &M, pcm1)
	d2.synthFrame(p, &log2M, &M, pcm2)

	if !equalF(pcm1, pcm2) {
		t.Fatal("all-unvoiced output differs across phase sources; dispersion leaked onto unvoiced harmonics")
	}
}

// voicedFrame builds a mixed-voicing frame (L=12, harmonics 1..8 voiced and
// 9..12 unvoiced) plus the matching M / log2M arrays synthFrame consumes.
// The non-zero unvoiced fraction gives dispScale > 0, and the voiced upper
// harmonics (l > L/4 = 3, i.e. l=4..8) are the ones the §6.3 dispersion
// perturbs — so this frame actually exercises the de-buzz path.
func voicedFrame() (mbe.Params, [mbe.MaxL + 1]float64, [mbe.MaxL + 1]float64) {
	var p mbe.Params
	p.L = 12
	p.W0 = 0.18
	var M, log2M [mbe.MaxL + 1]float64
	for l := 1; l <= p.L; l++ {
		if l <= 8 {
			p.Vl[l] = 1
		}
		M[l] = 1000
	}
	return p, M, log2M
}

func equalF(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func peak(out []int16) int16 {
	var m int16
	for _, v := range out {
		if v < 0 {
			v = -v
		}
		if v > m {
			m = v
		}
	}
	return m
}
