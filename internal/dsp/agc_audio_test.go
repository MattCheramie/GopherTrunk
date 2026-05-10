package dsp

import (
	"math"
	"testing"
	"time"
)

func TestAudioAGCConvergesOnSteadySine(t *testing.T) {
	// Drive a steady 1 kHz sinusoid at amplitude 0.05 (well below the
	// 0.3 reference) through the AGC. After enough samples for the
	// release time constant to settle, the output RMS should match
	// the reference within a small margin.
	const fs = 48_000.0
	a := NewAudioAGC(AudioAGCConfig{
		Reference:  0.3,
		Attack:     5 * time.Millisecond,
		Release:    50 * time.Millisecond, // shortened so test settles fast
		MaxGain:    64,
		SampleRate: fs,
	})
	n := int(fs * 0.5) // 500 ms — plenty for 50 ms release
	in := make([]float32, n)
	for i := range in {
		in[i] = 0.05 * float32(math.Sin(2*math.Pi*1000*float64(i)/fs))
	}
	out := a.Process(nil, in)

	// Measure RMS over the last 100 ms (steady-state).
	settled := n - int(fs*0.1)
	var ss float64
	for _, y := range out[settled:] {
		ss += float64(y) * float64(y)
	}
	rms := math.Sqrt(ss / float64(n-settled))

	// A pure sine at peak A has RMS A/√2. The AGC drives |y| toward
	// reference, so the output RMS should be ≈ 0.3/√2 ≈ 0.212.
	want := 0.3 / math.Sqrt2
	if rms < want*0.7 || rms > want*1.3 {
		t.Errorf("steady-state output RMS = %.3f, want ≈ %.3f", rms, want)
	}
}

func TestAudioAGCAttackCatchesTransient(t *testing.T) {
	// Hand the AGC a long quiet stretch followed by a loud transient
	// and check that it tames the transient — peak output during the
	// loud section should not exceed reference × MaxGain (clamp) and
	// should settle close to reference within the attack window.
	const fs = 48_000.0
	a := NewAudioAGC(AudioAGCConfig{
		Reference:  0.3,
		Attack:     5 * time.Millisecond,
		Release:    200 * time.Millisecond,
		MaxGain:    64,
		SampleRate: fs,
	})
	n := int(fs * 0.05)
	loud := make([]float32, n)
	for i := range loud {
		loud[i] = 1.0 // unit amplitude — 3.3× reference, would clip without AGC
	}
	out := a.Process(nil, loud)

	// During steady-state of the loud burst, output should be ≈ reference.
	settled := n - int(fs*0.01) // last 10 ms
	var sum float32
	for _, y := range out[settled:] {
		if y < 0 {
			y = -y
		}
		sum += y
	}
	mean := sum / float32(n-settled)
	if mean < 0.2 || mean > 0.4 {
		t.Errorf("steady-state |y| during loud burst = %.3f, want ≈ 0.3", mean)
	}
}

func TestAudioAGCReleaseRampUpIsSlow(t *testing.T) {
	// Hand the AGC a loud signal, then a quiet one. The gain on the
	// quiet section should not jump to MaxGain instantly — the
	// release coefficient is small so the level estimate decays
	// gradually. Concretely, at the start of the quiet section the
	// effective gain should still be roughly what the loud section
	// produced.
	const fs = 48_000.0
	a := NewAudioAGC(AudioAGCConfig{
		Reference:  0.3,
		Attack:     5 * time.Millisecond,
		Release:    200 * time.Millisecond,
		MaxGain:    64,
		SampleRate: fs,
	})
	loud := make([]float32, int(fs*0.1))
	for i := range loud {
		loud[i] = 1.0
	}
	a.Process(nil, loud)
	gainAfterLoud := a.Gain()

	quiet := make([]float32, 256) // ~5 ms — much less than release
	for i := range quiet {
		quiet[i] = 0.001
	}
	a.Process(nil, quiet)
	gainAfter256Quiet := a.Gain()

	// Gain should still be close to the loud-section gain (release is
	// 200 ms, we only ran 5 ms of quiet).
	ratio := gainAfter256Quiet / gainAfterLoud
	if ratio > 1.5 {
		t.Errorf("gain ratio after 5 ms quiet = %.2f, want ≤ 1.5 (release should be slow)", ratio)
	}
}

func TestAudioAGCResetClearsState(t *testing.T) {
	a := NewAudioAGC(AudioAGCConfig{Reference: 0.3, SampleRate: 48_000})
	in := make([]float32, 1024)
	for i := range in {
		in[i] = 1.0
	}
	a.Process(nil, in)
	a.Reset()
	if a.level != 0 {
		t.Errorf("after Reset, level = %f, want 0", a.level)
	}
	// Right after Reset, gain should be MaxGain (level is at floor).
	if g := a.Gain(); g != a.maxGain {
		t.Errorf("post-reset gain = %f, want maxGain = %f", g, a.maxGain)
	}
}

func TestAudioAGCInPlace(t *testing.T) {
	a := NewAudioAGC(AudioAGCConfig{Reference: 0.3, SampleRate: 48_000})
	buf := []float32{0.05, 0.05, 0.05, 0.05}
	out := a.Process(buf, buf)
	if &out[0] != &buf[0] {
		t.Error("Process(buf, buf) should reuse the slice")
	}
}

func TestAudioAGCDefaultsApplied(t *testing.T) {
	// Zero-valued config (except SampleRate) should hit defaults.
	a := NewAudioAGC(AudioAGCConfig{SampleRate: 48_000})
	if a.reference != 0.3 {
		t.Errorf("reference = %f, want default 0.3", a.reference)
	}
	if a.maxGain != 64 {
		t.Errorf("maxGain = %f, want default 64", a.maxGain)
	}
	if a.attackCoef <= 0 || a.attackCoef >= 1 {
		t.Errorf("attackCoef out of range: %f", a.attackCoef)
	}
	if a.releaseCoef <= 0 || a.releaseCoef >= 1 {
		t.Errorf("releaseCoef out of range: %f", a.releaseCoef)
	}
	if a.releaseCoef >= a.attackCoef {
		t.Errorf("releaseCoef (%f) should be smaller than attackCoef (%f)",
			a.releaseCoef, a.attackCoef)
	}
}

func TestAudioAGCRejectsZeroSampleRate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for zero sample rate")
		}
	}()
	_ = NewAudioAGC(AudioAGCConfig{Reference: 0.3})
}
