package mbe

import (
	"math"
	"testing"
)

// enhancerGainDB measures the steady-state magnitude response of the
// enhancement chain at freqHz (8 kHz PCM), in dB. Valid for a linear
// configuration (compressor disabled, as in the default chain).
func enhancerGainDB(e *VoiceEnhancer, freqHz float64) float64 {
	const n, fs = 8192, float64(PCMSampleRate)
	in := make([]float64, n)
	for i := range in {
		in[i] = 8000 * math.Sin(2*math.Pi*freqHz*float64(i)/fs)
	}
	out := make([]float64, n)
	copy(out, in)
	e.Process(out)
	rms := func(x []float64) float64 {
		var s float64
		for _, v := range x[n/2:] {
			s += v * v
		}
		return math.Sqrt(s / float64(n/2))
	}
	return 20 * math.Log10(rms(out)/rms(in))
}

func TestNewVoiceEnhancerDisabledIsNil(t *testing.T) {
	if e := NewVoiceEnhancer(8000, EnhancerConfig{}); e != nil {
		t.Error("zero-value (disabled) config should yield a nil pass-through enhancer")
	}
	if e := NewVoiceEnhancer(8000, EnhancerConfig{Enabled: false, HPFHz: 300}); e != nil {
		t.Error("explicitly disabled config should yield nil even with params set")
	}
	if e := NewVoiceEnhancer(0, DefaultEnhancerConfig()); e != nil {
		t.Error("non-positive sample rate should yield nil")
	}
}

func TestVoiceEnhancerNilIsPassThrough(t *testing.T) {
	var e *VoiceEnhancer
	pcm := []float64{1, -2, 3, -4}
	e.Process(pcm) // must not panic
	e.Reset()      // must not panic
	for i, want := range []float64{1, -2, 3, -4} {
		if pcm[i] != want {
			t.Fatalf("nil enhancer mutated input at %d: got %g want %g", i, pcm[i], want)
		}
	}
}

func TestEnhancerWithDefaultsBackfills(t *testing.T) {
	got := EnhancerConfig{Enabled: true}.WithDefaults()
	def := DefaultEnhancerConfig()
	if got.HPFHz != def.HPFHz || got.LPFHz != def.LPFHz || got.ShelfHz != def.ShelfHz ||
		got.ShelfDB != def.ShelfDB || got.AGCTarget != def.AGCTarget {
		t.Errorf("enabled config did not backfill defaults: %+v", got)
	}
	// A disabled config is returned untouched (no surprise params).
	if d := (EnhancerConfig{Enabled: false}).WithDefaults(); d.HPFHz != 0 || d.AGCTarget != 0 {
		t.Errorf("disabled config should not backfill, got %+v", d)
	}
	// A partial override keeps the caller's value.
	if p := (EnhancerConfig{Enabled: true, LPFHz: 3000}).WithDefaults(); p.LPFHz != 3000 {
		t.Errorf("partial override clobbered LPFHz: got %g want 3000", p.LPFHz)
	}
}

func TestVoiceEnhancerBandLimitsAndWarms(t *testing.T) {
	e := NewVoiceEnhancer(float64(PCMSampleRate), DefaultEnhancerConfig())
	if e == nil {
		t.Fatal("default config should build a non-nil enhancer")
	}
	// Rebuild per probe so each tone sees fresh filter state. The radio
	// tilt is disabled here so the band limits and the shelf are measured
	// in isolation (the tilt is −2 dB at 1 kHz by design; its own shape is
	// pinned by TestRadioTiltIsFirstOrderHighPass).
	untilted := func() *VoiceEnhancer {
		cfg := DefaultEnhancerConfig()
		cfg.TiltHz = -1
		return NewVoiceEnhancer(8000, cfg)
	}
	low := enhancerGainDB(untilted(), 120)
	mid := enhancerGainDB(untilted(), 1000)
	upperMid := enhancerGainDB(untilted(), 2500)
	high := enhancerGainDB(untilted(), 3900)

	if low > -4 {
		t.Errorf("120 Hz rumble not attenuated: %.2f dB", low)
	}
	if mid < -1.5 || mid > 0.5 {
		t.Errorf("1 kHz mid not near unity: %.2f dB", mid)
	}
	if high > -6 {
		t.Errorf("3900 Hz not band-limited: %.2f dB", high)
	}
	// Warmth shelf trims the upper-mid relative to the mid band.
	if upperMid >= mid-0.5 {
		t.Errorf("warmth shelf had no effect: 2.5 kHz=%.2f dB not below 1 kHz=%.2f dB", upperMid, mid)
	}
}

func TestVoiceEnhancerResetClearsState(t *testing.T) {
	e := NewVoiceEnhancer(8000, DefaultEnhancerConfig())
	imp := make([]float64, 64)
	imp[0] = 10000
	e.Process(imp)
	e.Reset()
	zeros := make([]float64, 64)
	e.Process(zeros)
	for i, v := range zeros {
		// Allow a vanishingly small numerical residue, but it must be
		// effectively silent — no ringing carried past the reset.
		if math.Abs(v) > 1e-6 {
			t.Fatalf("residual state after Reset at %d: %g", i, v)
		}
	}
}

func TestVoiceEnhancerCompressorReducesPeaks(t *testing.T) {
	cfg := DefaultEnhancerConfig()
	// Isolate the compressor: disable the filters so only dynamics change.
	cfg.HPFHz, cfg.LPFHz, cfg.ShelfDB = -1, -1, 0
	cfg.Compress = CompressConfig{Enabled: true, ThresholdDB: -20, Ratio: 4, AttackMs: 1, ReleaseMs: 50}
	e := NewVoiceEnhancer(8000, cfg)
	if e == nil || e.comp == nil {
		t.Fatal("compressor not built from enabled config")
	}
	// A loud steady tone well above threshold is brought down.
	const n = 4096
	loud := make([]float64, n)
	for i := range loud {
		loud[i] = 16000 * math.Sin(2*math.Pi*1000*float64(i)/8000)
	}
	e.Process(loud)
	var peak float64
	for _, v := range loud[n/2:] {
		if a := math.Abs(v); a > peak {
			peak = a
		}
	}
	if peak >= 16000 {
		t.Errorf("compressor did not reduce a 16000-peak tone: peak=%.0f", peak)
	}
}

func TestAGCSetTargetPeakIsLouder(t *testing.T) {
	// Identical input through two AGCs differing only in target peak: the
	// higher target must produce louder output. This is the loudness lever
	// the enhancement chain pulls via AGC.SetTargetPeak.
	run := func(target float64) int16 {
		agc := NewAGC(DefaultAGCConfig())
		agc.SetTargetPeak(target)
		var peak int16
		out := make([]int16, 160)
		// Several frames so the envelope settles toward the target.
		for f := 0; f < 20; f++ {
			pcm := make([]float64, 160)
			for i := range pcm {
				pcm[i] = 5000 * math.Sin(2*math.Pi*440*float64(f*160+i)/8000)
			}
			agc.Apply(pcm, out, false)
		}
		for _, s := range out {
			if s > peak {
				peak = s
			}
		}
		return peak
	}
	loud := run(22000)
	faithful := run(18000)
	if int(loud) <= int(float64(faithful)*1.1) {
		t.Errorf("target 22000 not meaningfully louder than 18000: %d vs %d", loud, faithful)
	}
}

// TestRadioTiltIsFirstOrderHighPass pins the tilt stage's shape: a first-
// order RC high-pass is −3 dB at its corner, ≈−6 dB an octave below, and
// within 1 dB of flat two octaves above — the curve measured between
// GopherTrunk's vocoder output and dsd-neo's default-on digital-voice
// high-pass (10 Sep calibration pairs), which the 2nd-order rumble HPF
// cannot reproduce.
func TestRadioTiltIsFirstOrderHighPass(t *testing.T) {
	const rate, fc = 8000.0, 750.0
	gainDB := func(f float64) float64 {
		tilt := newRadioTilt(rate, fc)
		n := 8000
		pcm := make([]float64, n)
		for i := range pcm {
			pcm[i] = math.Sin(2 * math.Pi * f * float64(i) / rate)
		}
		tilt.Process(pcm)
		var sum float64
		for _, v := range pcm[n/2:] { // steady state only
			sum += v * v
		}
		rms := math.Sqrt(sum / float64(n/2))
		return 20 * math.Log10(rms/math.Sqrt(0.5))
	}
	for _, tc := range []struct{ f, wantDB, tol float64 }{
		{fc, -3.0, 0.3},
		{fc / 2, -7.0, 0.5},
		{fc / 4, -12.3, 0.6},
		{fc * 4, -0.2, 0.3}, // bilinear warp pulls this slightly toward 0 dB
	} {
		if got := gainDB(tc.f); math.Abs(got-tc.wantDB) > tc.tol {
			t.Errorf("tilt gain at %.0f Hz = %.2f dB, want %.2f ± %.1f", tc.f, got, tc.wantDB, tc.tol)
		}
	}
	// Disabled by a negative corner; still enabled at the default.
	if e := NewVoiceEnhancer(rate, EnhancerConfig{Enabled: true, TiltHz: -1}); e.tilt != nil {
		t.Error("TiltHz < 0 must disable the tilt stage")
	}
	if e := NewVoiceEnhancer(rate, EnhancerConfig{Enabled: true}); e.tilt == nil {
		t.Error("default enabled config must carry the tilt stage")
	}
}
