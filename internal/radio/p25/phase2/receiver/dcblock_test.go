package receiver

import "testing"

// addDC returns a copy of iq with a constant complex DC offset added —
// the static 0 Hz spur a zero-IF front end's LO self-mixing leaves on an
// on-channel voice DDC.
func addDC(iq []complex64, dc complex64) []complex64 {
	out := make([]complex64, len(iq))
	for i, x := range iq {
		out[i] = x + dc
	}
	return out
}

// TestReceiverDCBlockRecoversDCOffsetStream is the failing-first regression
// for the ported DC-block stage (the P25 Phase 1 / TETRA voice-receiver
// lever, issue-parity port): a DC spur comparable to the signal amplitude
// shifts every absolute H-DQPSK symbol by the same complex bias, which does
// NOT cancel in the differential decode and collapses the dibit stream.
// EnableDCBlock strips it and decode returns to clean.
func TestReceiverDCBlockRecoversDCOffsetStream(t *testing.T) {
	const fs = 48_000.0
	// The block's ~0.8 Hz corner needs ~20k samples (~2500 symbols) to bleed
	// off the step transient (same settling the Phase 1 dcblock test skips),
	// so use a long stream and score the settled tail only.
	tx := pseudoRandomDibits(8000, 0xC0FFEE)
	const settleSkip = 4000
	clean, _ := makeP2HDQPSKIQWithOffset(tx, 0, fs)
	// The modulated stream peaks near |x| ≈ 1; a comparable DC pedestal is
	// the worst case an on-channel zero-IF tune produces.
	dirty := addDC(clean, complex(0.7, 0.5))

	run := func(iq []complex64, dcBlock bool) float64 {
		var rx []uint8
		r := New(Options{
			SampleRateHz:  fs,
			ClockMode:     ClockGardner,
			GardnerGain:   0.005,
			EnableDCBlock: dcBlock,
			DibitSink:     func(d []uint8, _ int) { rx = append(rx, d...) },
		})
		for i := 0; i < len(iq); i += 192 {
			end := i + 192
			if end > len(iq) {
				end = len(iq)
			}
			r.Process(iq[i:end])
		}
		return bestTailSER(tx, rx, settleSkip)
	}

	if ser := run(dirty, true); ser > 0.02 {
		t.Errorf("DC-offset stream with EnableDCBlock: tail SER = %.3f, want <= 0.02 (block failed to strip the spur)", ser)
	}
	// Failing-first evidence: without the block the same stream is garbage.
	// Keep the assertion loose (>0.1) — it documents the failure mode rather
	// than pinning its exact severity.
	if ser := run(dirty, false); ser < 0.1 {
		t.Errorf("DC-offset stream without the block decoded cleanly (SER %.3f) — fixture no longer exercises the failure mode", ser)
	}
}

// TestReceiverDCBlockNoHarmOnCleanStream: with the block ON, a clean
// DC-free stream must still decode cleanly (the ~1 Hz corner is decades
// below the modulation); with the block OFF (the default) the path is
// untouched by construction.
func TestReceiverDCBlockNoHarmOnCleanStream(t *testing.T) {
	const fs = 48_000.0
	tx := pseudoRandomDibits(2400, 0xC0FFEE)
	clean, _ := makeP2HDQPSKIQWithOffset(tx, 0, fs)

	var rx []uint8
	r := New(Options{
		SampleRateHz:  fs,
		ClockMode:     ClockGardner,
		GardnerGain:   0.005,
		EnableDCBlock: true,
		DibitSink:     func(d []uint8, _ int) { rx = append(rx, d...) },
	})
	for i := 0; i < len(clean); i += 192 {
		end := i + 192
		if end > len(clean) {
			end = len(clean)
		}
		r.Process(clean[i:end])
	}
	if ser := bestTailSER(tx, rx, 900); ser > 0.02 {
		t.Errorf("clean stream with EnableDCBlock: tail SER = %.3f, want <= 0.02 (block harmed a clean signal)", ser)
	}
}
