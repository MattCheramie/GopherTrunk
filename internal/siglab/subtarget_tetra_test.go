package siglab

import (
	"bytes"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestReplaySubTargetTETRALocks is the end-to-end reproduction of the field
// report: a TETRA control channel recorded as a narrowband slice BELOW the
// 144 kHz channel target (the natural 48/50 kHz rate SDR++/SDRTrunk pick) must
// still lock. Before the fix, the engine fed the sub-target stream to the
// receiver raw; the receiver rounded 50000/18000 = 2.78 → 3 samples/symbol and
// the Gardner loop — told a 3.0 nominal against a 2.78 true rate — never locked,
// emitting 50000/3 ≈ 16667 baud (−7.4%). Normalising the input up to the
// channel rate restores 8 sps and the lock.
//
// The capture is derived from the in-tree 144 kHz TETRA synth fixture by
// decimating it to 50 kHz through the (already-working) in>target DDC path, so
// the test needs no committed binary and exercises exactly the sub-target
// upsample the field capture requires.
func TestReplaySubTargetTETRALocks(t *testing.T) {
	const captureRate = 50_000.0

	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolTETRA, Format: FormatS16})
	if err != nil {
		t.Fatalf("Synthesize TETRA: %v", err)
	}
	if meta.SampleRateHz <= captureRate {
		t.Fatalf("fixture rate %.0f must exceed the sub-target capture rate", meta.SampleRateHz)
	}

	// Decimate the 144 kHz fixture down to a 50 kHz narrowband slice — the
	// shape of the operator's SDR++ recording. This uses the in>target path,
	// which already works; the fix is on the way back up.
	slice, sliceRate := Downconvert(iq, meta.SampleRateHz, 0, captureRate)
	if math.Abs(sliceRate-captureRate) > 1 {
		t.Fatalf("slice rate = %.1f, want ~%.0f", sliceRate, captureRate)
	}

	sys := trunking.System{Name: "siglab", Protocol: trunking.ProtocolTETRA}
	applySystemKnobs(&sys, meta.System)

	cfg := Config{
		Protocol:      trunking.ProtocolTETRA,
		System:        sys,
		SystemName:    "siglab",
		SampleRateHz:  captureRate,
		Format:        FormatS16,
		CollectIQDiag: true,
	}
	data := EncodeCapture(slice, FormatS16)
	res, err := RunReader(bytes.NewReader(data), "subtarget-tetra", cfg)
	if err != nil {
		t.Fatalf("RunReader: %v", err)
	}

	if !res.Locked {
		t.Errorf("sub-target (%.0f Hz) TETRA did not lock; NoLockReason=%q, EffectiveBaud=%.1f (%.2f%%)",
			captureRate, res.NoLockReason, res.EffectiveBaud, res.BaudDeviationPct)
	}
	// The recovered symbol rate must be ~18000, not the ~16667 the un-normalised
	// path produced. Allow 1% for the fixture's residual clock error.
	if math.Abs(res.BaudDeviationPct) > 1.0 {
		t.Errorf("EffectiveBaud = %.1f (%.2f%% off 18000), want within 1%%",
			res.EffectiveBaud, res.BaudDeviationPct)
	}
}
