package tuner

import "testing"

// The offsets from issue #881: two role=wideband RTL dongles, both at
// 2.048 MS/s, both auto(ddc), both 4 channels. The "failing" plan (RMR,
// VHF) clusters its carriers in a 225 kHz span — GT flags it as
// oversampled — while the "working" plan (MMR, UHF) spreads them across
// 1.75 MHz. The reporter's hypothesis was that the clustered/oversampled
// geometry makes the DDC hand the P25 demod sub-frame-sync-word chunks
// (~19 dibits < the 24-dibit FSW) so it never syncs. These vectors let us
// test that mechanism directly.
var (
	// center 166_550_000
	rmrClusteredOffsets = []float64{-125_000, -100_000, -87_500, +100_000}
	// center 420_962_500
	mmrWideSpanOffsets = []float64{-875_000, -587_500, -237_500, +875_000}
)

const (
	issue881RateHz  = 2_048_000.0
	issue881TapHz   = 48_000.0
	issue881Guard   = 0.05
	rmrMaxAbsOffset = 125_000.0
	mmrMaxAbsOffset = 875_000.0
	// A 16 KiB RTL-SDR USB transfer carries 8192 complex u8 IQ samples —
	// the chunk size that, decimated 2.048 MS/s → 48 kHz, yields the
	// ~19-dibit output the reporter measured.
	rtlUSBChunkSamples = 8192
)

// bankWithCapturingTaps builds a span-sized DDCBank at the issue #881
// capture rate and registers one length-capturing tap per offset. The
// returned slice of pointers accumulates each tap's total output length
// across Process calls, in offsets order.
func bankWithCapturingTaps(t *testing.T, offsets []float64, requiredHalfHz float64) (*DDCBank, []*int) {
	t.Helper()
	b := NewDDCBankForSpan(issue881RateHz, issue881TapHz, issue881Guard, requiredHalfHz)
	lens := make([]*int, len(offsets))
	for i, off := range offsets {
		n := new(int)
		lens[i] = n
		if err := b.AddTap(off, func(out []complex64) { *n += len(out) }); err != nil {
			t.Fatalf("AddTap(%.0f): %v", off, err)
		}
	}
	return b, lens
}

// TestDDCChunkSizeIndependentOfPlanGeometry is the issue #881 regression
// guard. It refutes the reported root cause — "oversampled/tightly-
// clustered channel plan starves the P25 demod with sub-24-dibit chunks"
// — by showing the per-tap output chunk size is a pure function of the
// capture rate and tap rate, NOT of the channel-plan span or clustering.
//
// Both plans run through structurally identical banks: at 2.048 MS/s the
// shared decimator is a no-op (2.048M/2 = 1.024M is below the 2.5 MS/s
// floor) regardless of span, so both use the same L=3/M=128 per-tap
// resampler. Feeding both the same 8192-sample RTL USB chunk therefore
// yields byte-for-byte identical per-tap output lengths — and both land
// well under the 24-dibit frame sync word. In other words the ~19-dibit
// chunk the reporter measured on the failing plan is ALSO what the
// working plan gets; chunk size cannot be why one syncs and the other
// doesn't. (The P25 SyncDetector buffers a 24-dibit history across
// Process calls — see phase1.SyncDetector and the passing
// TestControlChannelLocksAndGrantsAcrossSmallChunks with 19-dibit
// batches — so a sub-FSW chunk never blocks sync in the first place.)
func TestDDCChunkSizeIndependentOfPlanGeometry(t *testing.T) {
	rmr, rmrLens := bankWithCapturingTaps(t, rmrClusteredOffsets, rmrMaxAbsOffset)
	mmr, mmrLens := bankWithCapturingTaps(t, mmrWideSpanOffsets, mmrMaxAbsOffset)

	// Identical input signal into both banks: one RTL USB transfer's worth
	// of samples. A flat tone is fine — we care about output *length*, not
	// spectral content, here.
	gen := newToneGen(issue881RateHz, 0.3, 0)
	chunk := gen.Next(rtlUSBChunkSamples)
	rmr.Process(chunk)
	mmr.Process(chunk)

	if rmr.OutputRateHz() != mmr.OutputRateHz() {
		t.Errorf("per-tap output rate differs by geometry: RMR=%.3f MMR=%.3f — both should be %.0f",
			rmr.OutputRateHz(), mmr.OutputRateHz(), issue881TapHz)
	}

	rmr0 := *rmrLens[0]
	if rmr0 == 0 {
		t.Fatal("clustered plan produced no per-tap output for a full RTL USB chunk")
	}
	// Every tap in both plans must emit the SAME number of samples for the
	// same input — the differentiator the report attributes to geometry.
	for i, p := range rmrLens {
		if *p != rmr0 {
			t.Errorf("RMR tap %d (offset %.0f) len=%d, want %d — chunk size varies within the clustered plan",
				i, rmrClusteredOffsets[i], *p, rmr0)
		}
	}
	for i, p := range mmrLens {
		if *p != rmr0 {
			t.Errorf("MMR tap %d (offset %.0f) len=%d != RMR len=%d — chunk size depends on plan geometry",
				i, mmrWideSpanOffsets[i], *p, rmr0)
		}
	}

	// Tie the length back to the reporter's observation: 48 kHz / 4800 baud
	// = 10 samples per P25 symbol (dibit). One RTL USB chunk yields well
	// under the 24-dibit FSW on BOTH plans — the "tiny chunk" is not unique
	// to the failing device.
	dibitsPerChunk := rmr0 / 10
	if dibitsPerChunk >= 24 {
		t.Fatalf("expected the modelled RTL chunk to yield < 24 dibits (the reporter measured ~19); got %d", dibitsPerChunk)
	}
	t.Logf("both plans: %d samples/tap per 8192-sample RTL chunk = ~%d dibits (reporter measured ~19); "+
		"identical across clustered and wide-span geometry", rmr0, dibitsPerChunk)
}

// TestDDCClusteredOffsetsTuneToDC shows the second half of the issue #881
// refutation: every carrier in the oversampled/clustered RMR plan — including
// the 12.5 kHz-adjacent pair (-100 kHz / -87.5 kHz) and the ±100 kHz image
// pair — is mixed cleanly to DC by its own tap at the reporter's 2.048 MS/s
// capture rate. If the clustered geometry corrupted the channelizer output
// (the other half of the "never syncs" hypothesis), the target carrier would
// not concentrate at DC. It does, so the DDC hands the demod a correctly-tuned
// channel; whatever blocks sync on the reporter's live capture is upstream of
// GT's channelizer (front end / captured RF), which needs the raw .cfile to
// pin — consistent with the #764/#771 findings in CLAUDE.md.
func TestDDCClusteredOffsetsTuneToDC(t *testing.T) {
	for _, off := range rmrClusteredOffsets {
		off := off
		b := NewDDCBankForSpan(issue881RateHz, issue881TapHz, issue881Guard, rmrMaxAbsOffset)
		var got []complex64
		if err := b.AddTap(off, func(out []complex64) { got = append(got, out...) }); err != nil {
			t.Fatalf("AddTap(%.0f): %v", off, err)
		}
		// A single carrier at this offset, run long enough for the per-tap
		// resampler to settle and produce a few thousand narrow-band samples.
		gen := newToneGen(issue881RateHz, 0.5, off)
		for i := 0; i < 40; i++ {
			b.Process(gen.Next(4096))
		}
		if len(got) < 1024 {
			t.Fatalf("offset %.0f: too few output samples (%d)", off, len(got))
		}
		settled := got[len(got)/2:]
		if frac := powerNearDC(settled, issue881TapHz, 500); frac < 0.95 {
			t.Errorf("clustered offset %.0f Hz: only %.1f%% of tap power within ±500 Hz of DC — mis-tuned",
				off, frac*100)
		}
	}
}
