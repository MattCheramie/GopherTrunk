package tetra

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// These tests exercise the #1003/#1001 per-burst training-sequence equalizer wired
// into the DMO (Direct Mode) extractor: a Direct Mode Normal Burst is synthesized
// as a π/4-DQPSK symbol stream, pushed through a multipath channel that closes the
// eye, and run through ExtractDMBurstsEqualized. Training on the burst's known
// normal training sequence and equalizing BKN1/BKN2 should recover the TCH/S soft
// payload the raw ExtractDMBurstsSoft loses to the ISI. Metric is type-5 bit-error
// on the re-derived soft differentials (the constant-phase-invariant number), the
// same shape as the TMO traffic_lms_test.

// dmBuildDNB lays a detectable DNB (NTS1 midamble at lead L) into a random dibit
// stream, runs it through the channel, and returns the channel-decoded hard
// dibits, the raw differentials, the raw symbols, the known transmitted BKN1+BKN2
// type-5 bits, and L.
func dmBuildDNB(t *testing.T, h []complex64, sigma float64, seed int64) (hard []uint8, diffs, rx []complex64, wantBits []byte, L int) {
	t.Helper()
	const M = 400
	L = 200
	rng := rand.New(rand.NewSource(seed))
	edib := make([]uint8, M)
	for i := range edib {
		edib[i] = uint8(rng.Intn(4))
	}
	nts := NormalSyncDibits()
	copy(edib[L:L+len(nts)], nts)

	tx := tsEncodeSyms(edib)
	rx = tsMultipath(tx, h, sigma, rng)
	dq := demod.NewDQPSK()
	dq.SetRotation(math.Pi / 4)
	hard, diffs = dq.DecodeBoth(nil, nil, rx)

	// DNB payload: BKN1 = [L-108, L), BKN2 = [L+11, L+119).
	var payload []uint8
	payload = append(payload, edib[L+dmDNBBKN1Start:L]...)
	payload = append(payload, edib[L+dmDNBBKN2Start:L+dmDNBBKN2Start+dmBlockDibits]...)
	wantBits = TetraDibitsToBits(payload)
	return hard, diffs, rx, wantBits, L
}

func dmFindDNB(bursts []DMBurst, lead int) *DMBurst {
	for i := range bursts {
		if bursts[i].Kind == DMBurstNormal && bursts[i].Lead == lead {
			return &bursts[i]
		}
	}
	return nil
}

// dmSoftPayloadBitErr hard-slices a DNB's soft blocks (de-rotated by the burst's
// own rotation) and returns the type-5 bit-error against the known payload.
func dmSoftPayloadBitErr(b *DMBurst, want []byte) float64 {
	d := make([]complex64, 0, len(b.SoftBKN1)+len(b.SoftBKN2))
	d = append(d, b.SoftBKN1...)
	d = append(d, b.SoftBKN2...)
	llr := softType5FromDiffs(d, (4-b.Rotation)&3)
	return float64(tsBitErrs(tsSliceBits(llr), want)) / float64(len(want))
}

// TestExtractDMBurstsEqualizedRecoversMultipath is the failing-first regression:
// on a multipath channel that garbles the raw soft DNB decode, ExtractDMBurstsEqualized
// drives the soft TCH/S payload bit-error down sharply while ExtractDMBurstsSoft
// (raw differentials) stays high.
func TestExtractDMBurstsEqualizedRecoversMultipath(t *testing.T) {
	h := []complex64{1, complex(0.45, 0.25)}
	hard, diffs, rx, wantBits, L := dmBuildDNB(t, h, 0.03, 7)

	rawB := dmFindDNB(tetraExtractSoft(hard, diffs), L)
	eqB := dmFindDNB(ExtractDMBurstsEqualized(hard, diffs, rx, 0, 0, 0), L)
	if rawB == nil {
		t.Fatal("DNB not detected on the raw path — midamble too corrupted; soften the channel")
	}
	if eqB == nil {
		t.Fatal("DNB not detected on the equalized path")
	}
	if eqB.Rotation != 0 {
		t.Fatalf("expected rotation-0 DNB for the equalized case, got %d", eqB.Rotation)
	}

	rawErr := dmSoftPayloadBitErr(rawB, wantBits)
	eqErr := dmSoftPayloadBitErr(eqB, wantBits)
	t.Logf("DNB soft payload type-5 bit-error: raw=%.3f equalized=%.3f (n=%d)", rawErr, eqErr, len(wantBits))

	if rawErr < 0.08 {
		t.Fatalf("channel not hard enough: raw soft bit-error %.3f", rawErr)
	}
	if eqErr > 0.02 {
		t.Fatalf("DMO LMS-equalized soft decode still garbled: bit-error %.3f (raw %.3f)", eqErr, rawErr)
	}
	if eqErr >= rawErr {
		t.Fatalf("equalizer did not improve the DNB decode: equalized %.3f vs raw %.3f", eqErr, rawErr)
	}
}

// TestExtractDMBurstsEqualizedNoHarmOnClean: on an identity channel the equalized
// soft blocks decode as cleanly as the raw path (frozen taps ≈ pass-through).
func TestExtractDMBurstsEqualizedNoHarmOnClean(t *testing.T) {
	h := []complex64{1}
	hard, diffs, rx, wantBits, L := dmBuildDNB(t, h, 0.01, 5)

	eqB := dmFindDNB(ExtractDMBurstsEqualized(hard, diffs, rx, 0, 0, 0), L)
	if eqB == nil {
		t.Fatal("DNB not detected on a clean channel")
	}
	eqErr := dmSoftPayloadBitErr(eqB, wantBits)
	t.Logf("clean-channel DMO equalized payload bit-error: %.4f", eqErr)
	if eqErr > 0.01 {
		t.Fatalf("equalizer harmed a clean DNB: bit-error %.4f", eqErr)
	}
}

// TestExtractDMBurstsEqualizedNoSymbolsUnchanged guards the fallback: with no
// symbols supplied, ExtractDMBurstsEqualized is identical to ExtractDMBurstsSoft
// (the soft differentials are the raw ones), so it can never regress the caller
// that hasn't wired the SymbolSink.
func TestExtractDMBurstsEqualizedNoSymbolsUnchanged(t *testing.T) {
	h := []complex64{1, complex(0.4, 0.2)}
	hard, diffs, _, wantBits, L := dmBuildDNB(t, h, 0.02, 7)

	soft := dmFindDNB(tetraExtractSoft(hard, diffs), L)
	// nil symbols ⇒ no equalization; must equal the raw soft path.
	noSym := dmFindDNB(ExtractDMBurstsEqualized(hard, diffs, nil, 0, 0, 0), L)
	if soft == nil || noSym == nil {
		t.Fatal("DNB not detected")
	}
	_ = wantBits
	if len(noSym.SoftBKN1) != len(soft.SoftBKN1) || len(noSym.SoftBKN2) != len(soft.SoftBKN2) {
		t.Fatal("soft block lengths diverged")
	}
	for i := range soft.SoftBKN1 {
		if noSym.SoftBKN1[i] != soft.SoftBKN1[i] {
			t.Fatalf("SoftBKN1[%d] changed without symbols: %v vs %v", i, noSym.SoftBKN1[i], soft.SoftBKN1[i])
		}
	}
	for i := range soft.SoftBKN2 {
		if noSym.SoftBKN2[i] != soft.SoftBKN2[i] {
			t.Fatalf("SoftBKN2[%d] changed without symbols: %v vs %v", i, noSym.SoftBKN2[i], soft.SoftBKN2[i])
		}
	}
}

// tetraExtractSoft is a tiny wrapper so the tests read symmetrically.
func tetraExtractSoft(hard []uint8, diffs []complex64) []DMBurst {
	return ExtractDMBurstsSoft(hard, diffs, 0)
}
