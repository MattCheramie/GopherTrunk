package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestCarveSliceRate synthesizes a P25 capture, carves a narrowband slice, and
// asserts the slice lands at the per-protocol channel rate as a non-empty
// 2-channel s16 WAV.
func TestCarveSliceRate(t *testing.T) {
	iq, meta, err := siglab.Synthesize(siglab.SynthOptions{Protocol: trunking.ProtocolP25, Format: siglab.FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	dir := t.TempDir()
	capPath := filepath.Join(dir, "cc.cfile")
	if err := os.WriteFile(capPath, siglab.EncodeCapture(iq, siglab.FormatF32), 0o644); err != nil {
		t.Fatalf("write capture: %v", err)
	}

	outWav := filepath.Join(dir, "cc.slice.wav")
	rate, err := carveSlice(capPath, siglab.FormatF32, meta.SampleRateHz, 0, "p25", gtbundle.IntentSurvey, outWav)
	if err != nil {
		t.Fatalf("carveSlice: %v", err)
	}
	wantRate := uint32(ccdecoder.DDCTargetForProtocol(trunking.ProtocolP25) + 0.5)
	if rate != wantRate {
		t.Errorf("slice rate = %d, want %d", rate, wantRate)
	}
	info, err := os.Stat(outWav)
	if err != nil {
		t.Fatalf("stat slice: %v", err)
	}
	// A WAV header is 44 bytes; a real slice must carry samples beyond it.
	if info.Size() <= 44 {
		t.Errorf("slice WAV too small (%d bytes) — no samples carved", info.Size())
	}
}

// TestCarveSliceUnknownProtocol confirms an undecoded carrier (empty protocol)
// still carves at the C4FM-family default rate rather than erroring.
func TestCarveSliceUnknownProtocol(t *testing.T) {
	iq, meta, err := siglab.Synthesize(siglab.SynthOptions{Protocol: trunking.ProtocolP25, Format: siglab.FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	dir := t.TempDir()
	capPath := filepath.Join(dir, "u.cfile")
	if err := os.WriteFile(capPath, siglab.EncodeCapture(iq, siglab.FormatF32), 0o644); err != nil {
		t.Fatalf("write capture: %v", err)
	}
	rate, err := carveSlice(capPath, siglab.FormatF32, meta.SampleRateHz, 0, "", gtbundle.IntentGeneral, filepath.Join(dir, "u.slice.wav"))
	if err != nil {
		t.Fatalf("carveSlice(empty proto): %v", err)
	}
	if rate == 0 {
		t.Errorf("slice rate = 0 for undecoded carrier")
	}
}
