package conventional

import (
	"math"
	"testing"
)

// genFMModulatedTone synthesises an IQ stream representing a CTCSS
// tone at toneHz superimposed on a carrier and FM-modulated with the
// supplied per-sample frequency deviation. The output is what an FM
// receiver's discriminator would emit as a (toneHz, devHz) sine
// wave — the audio piece is what CTCSS sits in.
//
// We don't bother with the audio band — just the sub-audible tone —
// because the detector's low-pass filter rejects the audio anyway
// and unit tests should isolate one mechanism at a time.
func genFMModulatedTone(toneHz, devHz, sampleHz float64, n int) []complex64 {
	out := make([]complex64, n)
	phase := 0.0
	dt := 1.0 / sampleHz
	for i := range n {
		t := float64(i) * dt
		// Modulating signal: a CTCSS sinusoid at toneHz with peak
		// amplitude 1.0 (we'll let the deviation knob set the
		// FM index).
		m := math.Sin(2 * math.Pi * toneHz * t)
		// Integrate to get instantaneous phase. FM phase advance
		// per sample is 2π · devHz · m(t) · dt.
		phase += 2 * math.Pi * devHz * m * dt
		out[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	return out
}

func TestCTCSSDetector_RejectsSilence(t *testing.T) {
	d := NewCTCSSDetector(CTCSSConfig{SampleHz: 48_000, TargetHz: 100, BlockSize: 4800})
	// Constant-phase carrier (no modulation) — discriminator output
	// is silence, no tone.
	iq := make([]complex64, 4800)
	for i := range iq {
		iq[i] = complex(1, 0)
	}
	if d.Process(iq) {
		t.Error("detector matched on a silent carrier")
	}
}

func TestCTCSSDetector_MatchesConfiguredTone(t *testing.T) {
	d := NewCTCSSDetector(CTCSSConfig{SampleHz: 48_000, TargetHz: 100, BlockSize: 4800})
	// FM-modulate at 100 Hz with 1 kHz peak deviation — a typical
	// CTCSS injection level on commercial repeaters.
	iq := genFMModulatedTone(100, 1000, 48_000, 4800)
	if !d.Process(iq) {
		t.Error("detector failed to match a 100 Hz CTCSS tone")
	}
}

func TestCTCSSDetector_RejectsOffFrequencyTone(t *testing.T) {
	d := NewCTCSSDetector(CTCSSConfig{SampleHz: 48_000, TargetHz: 100, BlockSize: 4800})
	// Configure for 100 Hz but transmit 250 Hz — the Goertzel
	// magnitude at the 100 Hz bin should stay below threshold.
	iq := genFMModulatedTone(250, 1000, 48_000, 4800)
	if d.Process(iq) {
		t.Error("detector matched on a 250 Hz tone configured for 100 Hz")
	}
}

func TestCTCSSDetector_ResetClearsState(t *testing.T) {
	d := NewCTCSSDetector(CTCSSConfig{SampleHz: 48_000, TargetHz: 100, BlockSize: 4800})
	iq := genFMModulatedTone(100, 1000, 48_000, 4800)
	d.Process(iq)
	if !d.Present() {
		t.Fatal("test setup: detector never matched")
	}
	d.Reset()
	if d.Present() {
		t.Error("Present remained true after Reset")
	}
}

func TestCTCSSDetector_NilSafe(t *testing.T) {
	var d *CTCSSDetector // nil
	if d.Process(nil) {
		t.Error("nil detector should not report matched")
	}
}

func TestCTCSSDetector_BadConfigReturnsNil(t *testing.T) {
	if NewCTCSSDetector(CTCSSConfig{}) != nil {
		t.Error("zero config should yield nil")
	}
	if NewCTCSSDetector(CTCSSConfig{SampleHz: 48_000}) != nil {
		t.Error("missing TargetHz should yield nil")
	}
}

func TestValidateTone(t *testing.T) {
	cases := []struct {
		name string
		in   ToneConfig
		ok   bool
	}{
		{"empty mode ok", ToneConfig{}, true},
		{"none mode ok", ToneConfig{Mode: "none"}, true},
		{"ctcss in range ok", ToneConfig{Mode: "ctcss", CTCSSHz: 100.0}, true},
		{"ctcss too low fails", ToneConfig{Mode: "ctcss", CTCSSHz: 30}, false},
		{"ctcss too high fails", ToneConfig{Mode: "ctcss", CTCSSHz: 500}, false},
		{"dcs octal ok", ToneConfig{Mode: "dcs", DCSCode: "023"}, true},
		{"dcs non-octal fails", ToneConfig{Mode: "dcs", DCSCode: "089"}, false},
		{"dcs wrong length fails", ToneConfig{Mode: "dcs", DCSCode: "12"}, false},
		{"unknown mode fails", ToneConfig{Mode: "wat"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTone(tc.in)
			if tc.ok && err != nil {
				t.Errorf("want ok, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("want err, got nil")
			}
		})
	}
}

func TestBuildDetector_CTCSSReturnsDetector(t *testing.T) {
	d := buildDetector(ToneConfig{Mode: "ctcss", CTCSSHz: 100}, 48_000)
	if d == nil {
		t.Fatal("expected detector for ctcss/100Hz")
	}
}

func TestBuildDetector_DCSDefers(t *testing.T) {
	// DCS detector is a follow-up — build should return nil so
	// the scanner falls back to power-only squelch.
	d := buildDetector(ToneConfig{Mode: "dcs", DCSCode: "023"}, 48_000)
	if d != nil {
		t.Error("DCS detector should be nil until implemented")
	}
}

func TestBuildDetector_NoneReturnsNil(t *testing.T) {
	if buildDetector(ToneConfig{}, 48_000) != nil {
		t.Error("empty mode should yield nil")
	}
}

func TestBuildDetector_NoSampleRateReturnsNil(t *testing.T) {
	if buildDetector(ToneConfig{Mode: "ctcss", CTCSSHz: 100}, 0) != nil {
		t.Error("zero SampleHz should yield nil so the scanner runs without tone gating")
	}
}
