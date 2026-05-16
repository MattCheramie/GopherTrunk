package dsp

import (
	"math"
	"testing"
)

func TestRealResamplerOutputRateMatchesLOverM(t *testing.T) {
	cases := []struct{ L, M int }{
		{1, 6},     // 48k → 8k (composer audio path)
		{1, 5},     // 40k → 8k
		{441, 480}, // 48k → 44.1k (rational, common audio rate)
		{2, 1},     // upsample by 2
	}
	for _, c := range cases {
		r := NewRealResampler(c.L, c.M, 16, 8.6)
		in := make([]float32, 2_000)
		out := r.Process(nil, in)
		want := len(in) * c.L / c.M
		// ±2 is OK to absorb the commutator startup transient.
		if abs := absInt(len(out) - want); abs > 2 {
			t.Errorf("L=%d M=%d: got %d output samples, want ≈ %d", c.L, c.M, len(out), want)
		}
	}
}

func TestRealResamplerPassesPassbandSinusoid(t *testing.T) {
	// 48 kHz → 8 kHz (M=6). A 1 kHz sinusoid is well inside the
	// resampler's prototype passband; output should reproduce it
	// (close to unit amplitude) without aliasing artefacts.
	const inFs = 48_000.0
	r := NewRealResampler(1, 6, 32, 8.6)
	settling := 4096
	n := settling + 8192
	in := make([]float32, n)
	for i := range in {
		in[i] = float32(math.Sin(2 * math.Pi * 1000 * float64(i) / inFs))
	}
	out := r.Process(nil, in)

	// Output rate is 8 kHz; settling samples in input ≈ settling/6 in
	// output. Skip them, then measure RMS of the output and compare to
	// expected RMS of a 1 kHz sinusoid (1/√2).
	skip := settling / 6
	var ss float64
	for _, y := range out[skip:] {
		ss += float64(y) * float64(y)
	}
	rms := math.Sqrt(ss / float64(len(out)-skip))
	want := 1 / math.Sqrt2
	if rms < want*0.9 || rms > want*1.1 {
		t.Errorf("1 kHz passband output RMS = %.3f, want ≈ %.3f", rms, want)
	}
}

func TestRealResamplerRejectsAliasingFromAboveNyquist(t *testing.T) {
	// 48 kHz → 8 kHz. Output Nyquist is 4 kHz; the prototype
	// rejects anything above 4 kHz so a 12 kHz input sinusoid (which
	// would alias down to 4 kHz under naive decimation) should be
	// heavily attenuated in the output.
	const inFs = 48_000.0
	r := NewRealResampler(1, 6, 32, 8.6)
	settling := 4096
	n := settling + 8192
	in := make([]float32, n)
	for i := range in {
		in[i] = float32(math.Sin(2 * math.Pi * 12_000 * float64(i) / inFs))
	}
	out := r.Process(nil, in)

	skip := settling / 6
	var ss float64
	for _, y := range out[skip:] {
		ss += float64(y) * float64(y)
	}
	rms := math.Sqrt(ss / float64(len(out)-skip))
	// Want at least 30 dB rejection on a clean stopband signal.
	if rms > 0.05 {
		t.Errorf("12 kHz stopband output RMS = %.3f, want < 0.05 (≥ 26 dB)", rms)
	}
}

func TestRealResamplerResetClearsState(t *testing.T) {
	r := NewRealResampler(1, 4, 16, 8.6)
	in := make([]float32, 256)
	for i := range in {
		in[i] = 1
	}
	r.Process(nil, in)
	r.Reset()
	if r.histPos != 0 || r.idx != 0 || r.mCount != 0 {
		t.Errorf("post-Reset state non-zero: histPos=%d idx=%d mCount=%d",
			r.histPos, r.idx, r.mCount)
	}
	for i, h := range r.hist {
		if h != 0 {
			t.Errorf("post-Reset hist[%d] = %f, want 0", i, h)
			break
		}
	}
}

func TestRealResamplerPanicsOnBadParams(t *testing.T) {
	cases := []struct{ L, M, taps int }{
		{0, 1, 16},
		{1, 0, 16},
		{1, 1, 0},
	}
	for _, c := range cases {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for L=%d M=%d taps=%d", c.L, c.M, c.taps)
				}
			}()
			_ = NewRealResampler(c.L, c.M, c.taps, 8.6)
		}()
	}
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
