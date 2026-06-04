package siglab

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestSynthesizeAndAnalyzeLocks confirms the in-memory synthesize→decode
// bridge produces a locking Result for a clean fixture without writing a
// capture to disk — the path the web "idealized reference" compare uses.
func TestSynthesizeAndAnalyzeLocks(t *testing.T) {
	decode := Config{CollectIQDiag: true, CaptureIQ: true}
	res, meta, err := SynthesizeAndAnalyze(
		SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32},
		decode, nil,
	)
	if err != nil {
		t.Fatalf("SynthesizeAndAnalyze: %v", err)
	}
	if meta == nil {
		t.Fatal("expected synthesis metadata")
	}
	if res.Source != "synthesized" {
		t.Errorf("Source = %q, want synthesized", res.Source)
	}
	if !res.Locked {
		t.Errorf("expected clean synthesized P25 to lock; got %+v", res.Lock)
	}
	if res.IQTaps == nil || len(res.IQTaps.DecimatedIQ) == 0 {
		t.Error("expected IQ taps from the in-memory bridge")
	}
}

// TestSynthesizeAndAnalyzeStreams confirms the live event sink fires.
func TestSynthesizeAndAnalyzeStreams(t *testing.T) {
	var events int
	_, _, err := SynthesizeAndAnalyze(
		SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32},
		Config{CollectIQDiag: true},
		func(EventRecord) { events++ },
	)
	if err != nil {
		t.Fatalf("SynthesizeAndAnalyze: %v", err)
	}
	if events == 0 {
		t.Error("expected at least one streamed event")
	}
}
