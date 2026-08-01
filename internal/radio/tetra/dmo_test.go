package tetra

import (
	"reflect"
	"testing"
)

// TestDMOTrainingSequencesMatchSpec cross-checks that the shared training
// sequences this package already uses ARE the DMO ones — EN 300 396-2 §9.4.3.3.3
// equations (63) and (64) for the two normal training sequences. DMO and TMO
// share the π/4-DQPSK air interface, so the DMO burst slicer reuses these; if a
// future edit drifted the TMO constants, DMO framing would silently break, so pin
// the DMO spec values here independently. (The 38-bit synchronisation training
// sequence, eq (66), is likewise shared — SyncTrainingSeq, validated against live
// air for issue #553 — and its length is pinned below.)
func TestDMOTrainingSequencesMatchSpec(t *testing.T) {
	// EN 300 396-2 §9.4.3.3.3 eq (63): normal training sequence 1.
	spec1 := []uint8{1, 1, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0}
	// eq (64): normal training sequence 2.
	spec2 := []uint8{0, 1, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1, 0}

	if !reflect.DeepEqual(NormalTrainingSeq1, spec1) {
		t.Errorf("NormalTrainingSeq1 != EN 300 396-2 eq(63):\n got %v\nwant %v", NormalTrainingSeq1, spec1)
	}
	if !reflect.DeepEqual(NormalTrainingSeq2, spec2) {
		t.Errorf("NormalTrainingSeq2 != EN 300 396-2 eq(64):\n got %v\nwant %v", NormalTrainingSeq2, spec2)
	}
	if len(SyncTrainingSeq) != SyncTrainingBits {
		t.Errorf("SyncTrainingSeq length = %d, want %d (EN 300 396-2 eq(66) is a 38-bit word)", len(SyncTrainingSeq), SyncTrainingBits)
	}
}

// ramp fills n dibits with a deterministic, non-constant pattern so a slice can be
// compared exactly and won't accidentally correlate a training sequence.
func ramp(seed, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8((seed + i*7) % 4)
	}
	return out
}

func concatDibits(parts ...[]uint8) []uint8 {
	var out []uint8
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// TestExtractDMBurstsDSB builds a synthetic Direct Mode Synchronisation Burst
// (freq-corr + SCH/S + sync training seq + BKN2, EN 300 396-2 Table 16) and asserts
// the extractor detects it and slices the SCH/S (BKN1) and BKN2 blocks at the
// spec offsets.
func TestExtractDMBurstsDSB(t *testing.T) {
	freqCorr := ramp(0, 40)
	schs := ramp(1, dmSCHSDibits) // 60 dibits
	sts := SyncTrainingDibits()   // 19 dibits
	bkn2 := ramp(2, dmBlockDibits) // 108 dibits
	prefix := ramp(3, 12)
	suffix := ramp(3, 12)

	stream := concatDibits(prefix, freqCorr, schs, sts, bkn2, suffix)
	leadWant := len(prefix) + len(freqCorr) + len(schs) // sync-train lead dibit

	bursts := ExtractDMBursts(stream, 0)
	var dsb *DMBurst
	for i := range bursts {
		if bursts[i].Kind == DMBurstSync {
			dsb = &bursts[i]
			break
		}
	}
	if dsb == nil {
		t.Fatalf("no DSB detected (bursts=%d)", len(bursts))
	}
	if dsb.Lead != leadWant {
		t.Errorf("DSB lead = %d, want %d", dsb.Lead, leadWant)
	}
	if !reflect.DeepEqual(dsb.SCHS, schs) {
		t.Errorf("SCH/S block mis-sliced:\n got %v\nwant %v", dsb.SCHS, schs)
	}
	if !reflect.DeepEqual(dsb.BKN2, bkn2) {
		t.Errorf("BKN2 block mis-sliced:\n got %v\nwant %v", dsb.BKN2, bkn2)
	}
	if dsb.BKN1 != nil {
		t.Errorf("DSB BKN1 should be nil (block 1 is the SCH/S), got %v", dsb.BKN1)
	}
}

// TestExtractDMBurstsDNB builds a synthetic Direct Mode Normal Burst (BKN1 +
// normal training seq + BKN2, EN 300 396-2 Table 15) and asserts both 216-bit
// blocks are sliced at the spec offsets.
func TestExtractDMBurstsDNB(t *testing.T) {
	bkn1 := ramp(1, dmBlockDibits) // 108 dibits
	nts := NormalSyncDibits()      // 11 dibits (normal training seq 1)
	bkn2 := ramp(2, dmBlockDibits)
	prefix := ramp(3, 12)
	suffix := ramp(3, 12)

	stream := concatDibits(prefix, bkn1, nts, bkn2, suffix)
	leadWant := len(prefix) + len(bkn1)

	bursts := ExtractDMBursts(stream, 0)
	var dnb *DMBurst
	for i := range bursts {
		if bursts[i].Kind == DMBurstNormal {
			dnb = &bursts[i]
			break
		}
	}
	if dnb == nil {
		t.Fatalf("no DNB detected (bursts=%d)", len(bursts))
	}
	if dnb.Lead != leadWant {
		t.Errorf("DNB lead = %d, want %d", dnb.Lead, leadWant)
	}
	if !reflect.DeepEqual(dnb.BKN1, bkn1) {
		t.Errorf("BKN1 mis-sliced:\n got %v\nwant %v", dnb.BKN1, bkn1)
	}
	if !reflect.DeepEqual(dnb.BKN2, bkn2) {
		t.Errorf("BKN2 mis-sliced:\n got %v\nwant %v", dnb.BKN2, bkn2)
	}
}

// TestExtractDMBurstsRotation verifies a DSB is still detected when the whole
// dibit stream carries a residual π/4-DQPSK rotation (0..3), and that the reported
// Rotation de-rotates the sliced blocks back to the transmitted values — the same
// four-rotation correlation the TMO SB / traffic path relies on.
func TestExtractDMBurstsRotation(t *testing.T) {
	schs := ramp(1, dmSCHSDibits)
	bkn2 := ramp(2, dmBlockDibits)
	base := concatDibits(ramp(3, 12), ramp(0, 40), schs, SyncTrainingDibits(), bkn2, ramp(3, 12))

	for rot := uint8(1); rot < 4; rot++ {
		stream := rotateDibits(base, rot)
		bursts := ExtractDMBursts(stream, 0)
		var dsb *DMBurst
		for i := range bursts {
			if bursts[i].Kind == DMBurstSync {
				dsb = &bursts[i]
				break
			}
		}
		if dsb == nil {
			t.Fatalf("rot %d: DSB not detected", rot)
		}
		if dsb.Rotation != rot {
			t.Errorf("rot %d: reported rotation %d", rot, dsb.Rotation)
		}
		// De-rotating the sliced block by the reported rotation recovers the
		// transmitted SCH/S.
		got := rotateDibits(dsb.SCHS, (4-dsb.Rotation)&3)
		if !reflect.DeepEqual(got, schs) {
			t.Errorf("rot %d: de-rotated SCH/S mismatch", rot)
		}
	}
}
