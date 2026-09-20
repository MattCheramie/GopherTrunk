package filter

import (
	"math"
	"testing"
)

// toneGainDB measures the steady-state magnitude response of f at freqHz
// for a stream at sampleRate Hz, in dB (0 ⇒ unity). It drives a unit sine
// through the filter and compares output to input RMS over the second
// half of the buffer, after the IIR has settled.
func toneGainDB(f *Biquad, sampleRate, freqHz float64) float64 {
	const n = 8192
	in := make([]float64, n)
	for i := range in {
		in[i] = math.Sin(2 * math.Pi * freqHz * float64(i) / sampleRate)
	}
	out := make([]float64, n)
	copy(out, in)
	f.Process(out)
	rms := func(x []float64) float64 {
		var s float64
		for _, v := range x[n/2:] {
			s += v * v
		}
		return math.Sqrt(s / float64(n/2))
	}
	return 20 * math.Log10(rms(out)/rms(in))
}

func TestLowPassResponse(t *testing.T) {
	const fs, fc = 8000.0, 3400.0
	// Passband, corner, and well into the stopband near Nyquist.
	if g := toneGainDB(NewLowPass(fs, fc), fs, 800); g < -1.5 || g > 0.5 {
		t.Errorf("low-pass passband gain at 800 Hz = %.2f dB, want ≈0", g)
	}
	if g := toneGainDB(NewLowPass(fs, fc), fs, fc); g < -6 || g > -1 {
		t.Errorf("low-pass corner gain at %.0f Hz = %.2f dB, want ≈-3", fc, g)
	}
	if g := toneGainDB(NewLowPass(fs, fc), fs, 3900); g > -6 {
		t.Errorf("low-pass stopband gain at 3900 Hz = %.2f dB, want strong attenuation", g)
	}
}

func TestHighPassResponse(t *testing.T) {
	const fs, fc = 8000.0, 250.0
	if g := toneGainDB(NewHighPass(fs, fc), fs, 1500); g < -1.5 || g > 0.5 {
		t.Errorf("high-pass passband gain at 1500 Hz = %.2f dB, want ≈0", g)
	}
	if g := toneGainDB(NewHighPass(fs, fc), fs, fc); g < -6 || g > -1 {
		t.Errorf("high-pass corner gain at %.0f Hz = %.2f dB, want ≈-3", fc, g)
	}
	if g := toneGainDB(NewHighPass(fs, fc), fs, 80); g > -6 {
		t.Errorf("high-pass stopband gain at 80 Hz = %.2f dB, want strong attenuation", g)
	}
}

func TestBiquadPassThrough(t *testing.T) {
	// A non-positive sample rate / cutoff degrades to unity, not silence.
	if g := toneGainDB(NewLowPass(0, 3400), 8000, 1000); math.Abs(g) > 0.01 {
		t.Errorf("degenerate low-pass should pass through, gain = %.3f dB", g)
	}
	if g := toneGainDB(NewHighPass(8000, 0), 8000, 1000); math.Abs(g) > 0.01 {
		t.Errorf("degenerate high-pass should pass through, gain = %.3f dB", g)
	}
}

func TestBiquadNilIsNoOp(t *testing.T) {
	var f *Biquad
	pcm := []float64{1, 2, 3}
	f.Process(pcm) // must not panic
	f.Reset()      // must not panic
	if pcm[0] != 1 || pcm[1] != 2 || pcm[2] != 3 {
		t.Fatal("nil Biquad.Process mutated input")
	}
	f32 := []float32{1, 2, 3}
	f.ProcessFloat32(f32) // nil receiver must not panic
	if f32[0] != 1 || f32[1] != 2 || f32[2] != 3 {
		t.Fatal("nil Biquad.ProcessFloat32 mutated input")
	}
}

// TestProcessFloat32MatchesFloat64 checks that the float32 in-place path
// tracks the float64 Process it mirrors: the same difference equation, run
// in float64, written back as float32. The composer's analog-FM voice chain
// relies on this to reuse the tested RBJ sections without a []float64 copy.
func TestProcessFloat32MatchesFloat64(t *testing.T) {
	const fs, fc = 48000.0, 300.0
	n := 4096
	in := make([]float64, n)
	for i := range in {
		// DC bias + a sub-audible tone + a voice tone — the FM-audio shape.
		in[i] = 0.2 + 0.3*math.Sin(2*math.Pi*100*float64(i)/fs) + math.Sin(2*math.Pi*1200*float64(i)/fs)
	}
	f64 := make([]float64, n)
	copy(f64, in)
	NewHighPass(fs, fc).Process(f64)

	f32 := make([]float32, n)
	for i, v := range in {
		f32[i] = float32(v)
	}
	NewHighPass(fs, fc).ProcessFloat32(f32)

	for i := range f64 {
		if d := math.Abs(f64[i] - float64(f32[i])); d > 1e-4 {
			t.Fatalf("sample %d: float32 path %.6f vs float64 %.6f (diff %.2e)", i, f32[i], f64[i], d)
		}
	}
}

// TestProcessFloat32RemovesDCAndSubAudible pins the behaviour the analog-FM
// chain needs: the two-section (4th-order) 300 Hz high-pass the composer runs
// zeroes a DC bias and strongly attenuates a sub-audible CTCSS tone that sits
// just under the corner (241.8 Hz is a standard tone), while leaving
// voice-band energy essentially intact. A single 2nd-order section only trims
// the near-corner tone ~6 dB, so the composer cascades two.
func TestProcessFloat32RemovesDCAndSubAudible(t *testing.T) {
	const fs, fc = 48000.0, 300.0
	n := 48000
	const dc = 0.25
	tone := make([]float32, n)  // DC + 241.8 Hz CTCSS, amplitude 0.5 (RMS ≈ 0.354)
	voice := make([]float32, n) // 1 kHz voice tone (no DC)
	for i := range tone {
		tone[i] = float32(dc + 0.5*math.Sin(2*math.Pi*241.8*float64(i)/fs))
		voice[i] = float32(math.Sin(2 * math.Pi * 1000 * float64(i) / fs))
	}
	// The composer's cascade: two Butterworth high-pass sections in series.
	for i := 0; i < 2; i++ {
		NewHighPass(fs, fc).ProcessFloat32(tone)
		NewHighPass(fs, fc).ProcessFloat32(voice)
	}

	// DC removed: mean of the settled tail ≈ 0.
	var s float64
	for _, v := range tone[n/2:] {
		s += float64(v)
	}
	if mean := math.Abs(s / float64(n/2)); mean > 1e-3 {
		t.Errorf("DC not removed: residual mean %.4f, want ≈0", mean)
	}
	// Sub-audible tone strongly attenuated: settled-tail RMS well below the
	// input 0.354 (a single section leaves ~0.19; the cascade ~0.09).
	rms := func(x []float32) float64 {
		var ss float64
		for _, v := range x[n/2:] {
			ss += float64(v) * float64(v)
		}
		return math.Sqrt(ss / float64(len(x)-n/2))
	}
	if r := rms(tone); r > 0.13 {
		t.Errorf("241.8 Hz tone RMS after 4th-order high-pass = %.3f, want strongly attenuated (<0.13)", r)
	}
	// Voice band preserved: RMS stays near the unit-sine 0.707.
	if r := rms(voice); r < 0.65 {
		t.Errorf("1 kHz voice RMS after high-pass = %.3f, want ≈0.707 (preserved)", r)
	}
}

func TestBiquadResetClearsState(t *testing.T) {
	f := NewLowPass(8000, 1000)
	// Excite the filter, then reset.
	imp := make([]float64, 64)
	imp[0] = 1
	f.Process(imp)
	f.Reset()
	// After reset, a run of zeros must stay exactly zero — no residual
	// ringing leaking from the previous excitation.
	zeros := make([]float64, 64)
	f.Process(zeros)
	for i, v := range zeros {
		if v != 0 {
			t.Fatalf("residual state after Reset: sample %d = %g", i, v)
		}
	}
}
