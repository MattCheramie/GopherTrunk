package siglab

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestEngineReportsADCRailOverload is the issue #836 capture-quality
// diagnosis: a capture whose samples sit at the ADC rail (a handheld a few
// metres from an RTL-SDR on AGC — 23 % of the reporter's samples were pinned)
// must be named as front-end overload by the no-lock verdict, not left as
// "no frame sync was found", and the raw clip ratio must be on the Result
// without -diag. A clean run of the same stream reads 0.
func TestEngineReportsADCRailOverload(t *testing.T) {
	const sampleRate = 48_000.0
	dibits := make([]uint8, 12_000)
	for i := range dibits {
		dibits[i] = uint8((i*7 + i/5) % 4)
	}
	// A healthy capture sits well below full scale (the modulator's unit
	// carrier would itself touch ±1 on each axis every quarter turn).
	iq := demod.ModulateC4FM(dibits, 10, 8, 0.20, sampleRate, 1944.0)
	for i := range iq {
		iq[i] *= 0.5
	}

	// Overload: the front end delivers the carrier at 2.5× full scale and the
	// ADC clips it to ±1 per axis — what an RTL-SDR's 8-bit ADC does at the
	// rail, so the bulk of every sample is pinned.
	clipped := make([]complex64, len(iq))
	for i, s := range iq {
		re, im := real(s)*5, imag(s)*5
		clipped[i] = complex(clampUnit(re), clampUnit(im))
	}

	dir := t.TempDir()
	cleanPath := filepath.Join(dir, "clean.cfile")
	clipPath := filepath.Join(dir, "clipped.cfile")
	writeF32(t, cleanPath, iq)
	writeF32(t, clipPath, clipped)

	cfg := Config{Protocol: trunking.ProtocolDMRTier2, SystemName: "t", SampleRateHz: sampleRate, Format: FormatF32}
	clean, err := Run(cleanPath, cfg)
	if err != nil {
		t.Fatalf("Run(clean): %v", err)
	}
	if clean.RawClipRatio != 0 {
		t.Errorf("clean capture RawClipRatio = %v, want 0", clean.RawClipRatio)
	}
	res, err := Run(clipPath, cfg)
	if err != nil {
		t.Fatalf("Run(clipped): %v", err)
	}
	if res.RawClipRatio < 0.5 {
		t.Errorf("clipped capture RawClipRatio = %.3f, want >= 0.5 (the bulk of a 2.5× overdriven carrier sits at the rail)", res.RawClipRatio)
	}
	if res.Locked {
		t.Fatalf("a rail-clipped random stream must not lock")
	}
	if !strings.Contains(res.NoLockReason, "ADC rail") {
		t.Errorf("NoLockReason = %q, want the front-end overload diagnosis", res.NoLockReason)
	}
}

func clampUnit(v float32) float32 {
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}
