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
