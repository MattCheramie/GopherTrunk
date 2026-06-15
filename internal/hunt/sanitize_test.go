package hunt

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// TestDetectedSignalSanitize asserts that a signal whose measurements went
// non-finite — an infinite SNR from a noise-free residual, an infinite siglab
// confidence, a -Inf analog power from log(0), NaN classifier features — is
// clamped to finite values and becomes JSON-encodable. Before the fix the same
// payload broke json.Marshal with "unsupported value: +Inf", which on the
// WebSocket event stream tore the live survey's connection down (issue #648).
func TestDetectedSignalSanitize(t *testing.T) {
	ds := DetectedSignal{
		FreqHz:     440_087_400,
		SNRDb:      float32(math.Inf(1)),
		Class:      survey.ClassPSK,
		Confidence: math.NaN(),
		BaudHz:     math.Inf(-1),
		Trunking:   &TrunkingRef{Protocol: "tetra", Confidence: math.Inf(1)},
		Analog:     &survey.AnalogReport{Active: true, PowerDbFS: math.Inf(-1), CTCSSHz: math.NaN()},
		Features: survey.ClassFeatures{
			SNRDb:          math.Inf(1),
			EnvelopeCV:     math.NaN(),
			IFStd:          math.Inf(-1),
			IFKurtosis:     math.NaN(),
			BaudHz:         math.Inf(1),
			BaudProminence: math.Inf(1),
		},
	}

	// The raw signal must fail to marshal — that is the bug we are guarding.
	if _, err := json.Marshal(ds); err == nil {
		t.Fatal("expected json.Marshal to fail on the un-sanitized non-finite signal")
	}

	ds.sanitize()

	finite := []struct {
		name string
		v    float64
	}{
		{"SNRDb", float64(ds.SNRDb)},
		{"Confidence", ds.Confidence},
		{"BaudHz", ds.BaudHz},
		{"Trunking.Confidence", ds.Trunking.Confidence},
		{"Analog.PowerDbFS", ds.Analog.PowerDbFS},
		{"Analog.CTCSSHz", ds.Analog.CTCSSHz},
		{"Features.SNRDb", ds.Features.SNRDb},
		{"Features.EnvelopeCV", ds.Features.EnvelopeCV},
		{"Features.IFStd", ds.Features.IFStd},
		{"Features.IFKurtosis", ds.Features.IFKurtosis},
		{"Features.BaudHz", ds.Features.BaudHz},
		{"Features.BaudProminence", ds.Features.BaudProminence},
	}
	for _, f := range finite {
		if math.IsNaN(f.v) || math.IsInf(f.v, 0) {
			t.Errorf("%s = %v, want finite after sanitize", f.name, f.v)
		}
	}

	if _, err := json.Marshal(ds); err != nil {
		t.Fatalf("json.Marshal after sanitize: %v", err)
	}
}

// TestDetectedSignalSanitizePreservesFinite makes sure sanitize only touches
// non-finite values and leaves real measurements alone.
func TestDetectedSignalSanitizePreservesFinite(t *testing.T) {
	ds := DetectedSignal{
		FreqHz:     440_260_000,
		SNRDb:      61.6,
		Confidence: 0.85,
		BaudHz:     4800,
		Trunking:   &TrunkingRef{Protocol: "dmr", Confidence: 0.7},
		Analog:     &survey.AnalogReport{PowerDbFS: -12.5, CTCSSHz: 67.0},
		Features:   survey.ClassFeatures{SNRDb: 61.6, BaudProminence: 22.5},
	}
	ds.sanitize()

	if ds.SNRDb != 61.6 || ds.Confidence != 0.85 || ds.BaudHz != 4800 {
		t.Errorf("sanitize altered finite top-level fields: %+v", ds)
	}
	if ds.Trunking.Confidence != 0.7 {
		t.Errorf("Trunking.Confidence = %v, want 0.7", ds.Trunking.Confidence)
	}
	if ds.Analog.PowerDbFS != -12.5 || ds.Analog.CTCSSHz != 67.0 {
		t.Errorf("sanitize altered finite analog fields: %+v", ds.Analog)
	}
	if ds.Features.SNRDb != 61.6 || ds.Features.BaudProminence != 22.5 {
		t.Errorf("sanitize altered finite feature fields: %+v", ds.Features)
	}
}
