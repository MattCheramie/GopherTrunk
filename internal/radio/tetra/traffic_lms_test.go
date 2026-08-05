package tetra

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// These tests exercise the #1001 per-burst training-sequence equalizer wired into
// TrafficExtractor: a full Normal Continuous Downlink Burst is synthesized as a
// π/4-DQPSK symbol stream, pushed through a multipath channel that closes the
// constellation eye, and fed to the extractor exactly as the receiver feeds it
// (hard dibits + the raw differentials via StashSoft + the raw symbols via
// StashSymbols). With EnableLMSEqualizer on, the extractor trains on the burst's
// known midamble, equalizes BKN1/BKN2, and re-derives the soft LLRs — which should
// recover the payload the un-equalized soft path loses to the ISI.
//
// Metric is type-5 bit-error rate on the emitted softType5 stream (the
// constant-phase-invariant, decode-level number CLAUDE.md flags as the trustworthy
// one), measured against the known transmitted payload.

// tsEncodeSyms differentially encodes a dibit stream into unit-modulus π/4-DQPSK
// symbols such that idealDiff(dibits[i]) == sym[i]·conj(sym[i-1]) — i.e. the
// extractor's dibit index i lines up 1:1 with symbol index i, matching how the
// receiver emits symbols and dibits in lockstep. Index 0 is the phase anchor
// (its differential is undefined and never read; the burst sits far from it).
func tsEncodeSyms(dibits []uint8) []complex64 {
	s := make([]complex64, len(dibits))
	s[0] = 1
	for i := 1; i < len(dibits); i++ {
		s[i] = s[i-1] * idealDiff(dibits[i])
	}
	return s
}

// tsMultipath convolves sym with a causal channel h (h[0] the direct path) and
// adds complex Gaussian noise — the same minimum-phase ISI model the equalizer
// package's tests use, so a midamble-trained LMS has a near-zero-delay inverse.
func tsMultipath(sym, h []complex64, sigma float64, rng *rand.Rand) []complex64 {
	out := make([]complex64, len(sym))
	for n := range sym {
		var acc complex64
		for k := range h {
			if n-k >= 0 {
				acc += h[k] * sym[n-k]
			}
		}
		acc += complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
		out[n] = acc
	}
	return out
}

// tsSliceBits hard-slices a softType5 LLR stream (positive ⇒ bit 0) to bits.
func tsSliceBits(llr []float32) []byte {
	out := make([]byte, len(llr))
	for i, v := range llr {
		if v < 0 {
			out[i] = 1
		}
	}
	return out
}

func tsBitErrs(got, want []byte) int {
	n := len(got)
	if len(want) < n {
		n = len(want)
	}
	bad := 0
	for i := 0; i < n; i++ {
		if got[i] != want[i] {
			bad++
		}
	}
	return bad
}

// tsBuildBurst lays a detectable NCDB (NTS1 midamble at index L) into a random
// dibit stream, runs it through the channel, and returns everything the extractor
// needs: the channel-decoded hard dibits, the raw differentials, the raw symbols,
// and the known transmitted payload bits (BKN1+BKN2) to score against.
func tsBuildBurst(t *testing.T, h []complex64, sigma float64, seed int64) (hard []uint8, diffs, rx []complex64, wantBits []byte, L int) {
	t.Helper()
	const M = 600
	L = 300
	rng := rand.New(rand.NewSource(seed))
	edib := make([]uint8, M)
	for i := range edib {
		edib[i] = uint8(rng.Intn(4))
	}
	nts := NormalSyncDibits()
	copy(edib[L:L+len(nts)], nts)

	tx := tsEncodeSyms(edib)
	rx = tsMultipath(tx, h, sigma, rng)

	// Decode the received symbols exactly as the receiver does (π/4 rotation),
	// yielding the hard dibits and the parallel differentials in one pass.
	dq := demod.NewDQPSK()
	dq.SetRotation(math.Pi / 4)
	hard, diffs = dq.DecodeBoth(nil, nil, rx)

	// Known transmitted payload: BKN1 = [L-115, L-7), BKN2 = [L+19, L+127).
	var payload []uint8
	payload = append(payload, edib[L+ndbBKN1Start:L+ndbBKN1Start+ndbBlockDibits]...)
	payload = append(payload, edib[L+ndbBKN2Start:L+ndbBKN2Start+ndbBlockDibits]...)
	wantBits = TetraDibitsToBits(payload)
	return hard, diffs, rx, wantBits, L
}

// tsRunSoft feeds one prepared burst through an extractor and returns the emitted
// softType5 and how many bursts were detected. When lms is true the
// training-sequence equalizer is enabled and symbols are stashed. Callers assert
// the count is 1 so a spurious rotated-NTS match on the random filler can never
// silently swap in a garbage burst's soft stream.
func tsRunSoft(hard []uint8, diffs, rx []complex64, lms bool) ([]float32, int) {
	var got []float32
	n := 0
	te := NewTrafficExtractor(0, func(_ []byte, softType5 []float32, _, _ uint8) {
		if softType5 != nil {
			got = softType5
			n++
		}
	})
	if lms {
		te.EnableLMSEqualizer(0, 0)
		te.StashSymbols(rx, 0)
	}
	te.StashSoft(diffs, 0)
	te.Process(hard, 0)
	return got, n
}

// TestTrafficExtractorLMSRecoversMultipathBurst is the failing-first regression:
// on a multipath channel that garbles the raw soft decode, enabling the per-burst
// training-sequence equalizer drives the emitted softType5 payload bit-error rate
// down sharply — proving the plumbing trains on the midamble and equalizes the
// data blocks, not just that a burst was emitted.
func TestTrafficExtractorLMSRecoversMultipathBurst(t *testing.T) {
	// A moderate two-tap echo: harsh enough to garble the 216-dibit payload, mild
	// enough that the 11-dibit midamble still correlates within the detector's
	// tolerance so the burst is found at all.
	h := []complex64{1, complex(0.45, 0.25)}
	hard, diffs, rx, wantBits, _ := tsBuildBurst(t, h, 0.03, 11)

	rawSoft, rawN := tsRunSoft(hard, diffs, rx, false)
	eqSoft, eqN := tsRunSoft(hard, diffs, rx, true)
	if rawN != 1 {
		t.Fatalf("expected exactly one burst on the raw path, got %d", rawN)
	}
	if eqN != 1 {
		t.Fatalf("expected exactly one burst on the equalized path, got %d", eqN)
	}

	total := len(wantBits)
	rawErr := float64(tsBitErrs(tsSliceBits(rawSoft), wantBits)) / float64(total)
	eqErr := float64(tsBitErrs(tsSliceBits(eqSoft), wantBits)) / float64(total)
	t.Logf("payload type-5 bit-error: raw=%.3f equalized=%.3f (n=%d)", rawErr, eqErr, total)

	if rawErr < 0.08 {
		t.Fatalf("channel not hard enough: raw soft bit-error %.3f (want the ISI to actually garble it)", rawErr)
	}
	if eqErr > 0.02 {
		t.Fatalf("LMS-equalized soft decode still garbled: bit-error %.3f (raw %.3f)", eqErr, rawErr)
	}
	if eqErr >= rawErr {
		t.Fatalf("equalizer did not improve the decode: equalized %.3f vs raw %.3f", eqErr, rawErr)
	}
}

// TestTrafficExtractorLMSNoHarmOnCleanChannel: on an identity channel the frozen
// taps stay a near pass-through, so enabling the equalizer does not degrade an
// already-clean burst's soft decode (the SnapshotLMS/SnapshotCMA no-harm rule).
func TestTrafficExtractorLMSNoHarmOnCleanChannel(t *testing.T) {
	h := []complex64{1} // identity channel
	hard, diffs, rx, wantBits, _ := tsBuildBurst(t, h, 0.01, 3)

	eqSoft, eqN := tsRunSoft(hard, diffs, rx, true)
	if eqN != 1 {
		t.Fatalf("expected exactly one burst on a clean channel, got %d", eqN)
	}
	eqErr := float64(tsBitErrs(tsSliceBits(eqSoft), wantBits)) / float64(len(wantBits))
	t.Logf("clean-channel equalized payload bit-error: %.4f", eqErr)
	if eqErr > 0.01 {
		t.Fatalf("equalizer harmed a clean burst: soft bit-error %.4f", eqErr)
	}
}

// TestTrafficExtractorSoftUnchangedWithoutEqualizer guards the byte-identical
// promise: with the equalizer off, stashing symbols or not, the emitted softType5
// is exactly the raw-differential path (the pre-#1001 behaviour).
func TestTrafficExtractorSoftUnchangedWithoutEqualizer(t *testing.T) {
	h := []complex64{1, complex(0.45, 0.25)}
	hard, diffs, rx, _, _ := tsBuildBurst(t, h, 0.03, 11)

	base, baseN := tsRunSoft(hard, diffs, rx, false)
	if baseN != 1 {
		t.Fatalf("expected exactly one burst, got %d", baseN)
	}
	// Even with symbols stashed, an extractor that never enabled the equalizer must
	// produce the identical soft stream.
	var withSyms []float32
	te := NewTrafficExtractor(0, func(_ []byte, softType5 []float32, _, _ uint8) {
		if softType5 != nil {
			withSyms = softType5
		}
	})
	te.StashSymbols(rx, 0)
	te.StashSoft(diffs, 0)
	te.Process(hard, 0)
	if withSyms == nil {
		t.Fatal("no burst detected (symbols stashed, equalizer off)")
	}
	if len(withSyms) != len(base) {
		t.Fatalf("soft length changed: %d vs %d", len(withSyms), len(base))
	}
	for i := range base {
		if withSyms[i] != base[i] {
			t.Fatalf("soft LLR %d changed with equalizer off: %v vs %v", i, withSyms[i], base[i])
		}
	}
}
