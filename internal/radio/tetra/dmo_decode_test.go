package tetra

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// These tests close the loop dmo.go's slicer left open: they build a spec-shaped
// DSB / DNB out of REAL encoded channel blocks (EncodeBSCH / EncodeSCHHD /
// EncodeSCHF / EncodeTCHS — the inverses of the decoders this file exercises),
// lay them out per EN 300 396-2 Tables 15/16, run the full
// ExtractDMBursts → descramble → channel-decode path, and assert the original
// content comes back CRC-valid. A round-trip is not proof of on-air decode (the
// #764/#771 lesson — the skip-guarded replay harness is that gate), but it does
// pin that the DMO descramble/slice/decode wiring is the exact inverse of a
// spec-built burst, and it fails without dmo_decode.go.

// dmoBitsToDibits packs on-air type-5 bits into the dibit block the demodulator
// emits for them (the inverse the receiver would produce).
func dmoBitsToDibits(bits []byte) []uint8 { return TetraBitsToDibits(bits) }

// buildDSB lays out a Direct Mode Synchronisation Burst per Table 16:
// preamble/phase (folded into a leading ramp) + 80-bit freq-corr + 120-bit
// SCH/S + 38-bit sync training seq + 216-bit SCH/H (BKN2). Blocks are supplied
// as already-encoded type-5 bit slices.
func buildDSB(schsType5, schhType5 []byte) []uint8 {
	freqCorr := ramp(0, 40) // 80 bits → 40 dibits, content irrelevant to the slicer
	schs := dmoBitsToDibits(schsType5)
	sts := SyncTrainingDibits()
	bkn2 := dmoBitsToDibits(schhType5)
	return concatDibits(ramp(3, 12), freqCorr, schs, sts, bkn2, ramp(3, 12))
}

// buildDNB lays out a Direct Mode Normal Burst per Table 15: 216-bit BKN1 +
// 22-bit normal training seq + 216-bit BKN2. block1/block2 are already-encoded
// type-5 bit slices (108 dibits each).
func buildDNB(block1, block2 []byte) []uint8 {
	return concatDibits(ramp(3, 12), dmoBitsToDibits(block1), NormalSyncDibits(), dmoBitsToDibits(block2), ramp(3, 12))
}

func findBurst(bursts []DMBurst, kind DMBurstKind) *DMBurst {
	for i := range bursts {
		if bursts[i].Kind == kind {
			return &bursts[i]
		}
	}
	return nil
}

func seqBits(seed, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte((seed + i*5) % 2)
	}
	return out
}

// TestDMSCHSRoundTrip encodes a 60-bit SYNC block through the BSCH chain (a DSB's
// SCH/S is scrambled with colour 0, §8.2.5.2), frames it as a DSB, and asserts
// DecodeDMSCHS recovers it CRC-valid.
func TestDMSCHSRoundTrip(t *testing.T) {
	sync := seqBits(1, 60)
	schs := EncodeBSCH(sync)                // 120 type-5 bits, colour 0
	schh := EncodeSCHHD(seqBits(2, 124), 0) // DSB BKN2 (SCH/H), colour 0

	bursts := ExtractDMBursts(buildDSB(schs, schh), 0)
	dsb := findBurst(bursts, DMBurstSync)
	if dsb == nil {
		t.Fatalf("no DSB detected (bursts=%d)", len(bursts))
	}
	got, ok := DecodeDMSCHS(*dsb)
	if !ok {
		t.Fatalf("DecodeDMSCHS CRC failed")
	}
	if !reflect.DeepEqual(got, sync) {
		t.Errorf("SCH/S mismatch:\n got %v\nwant %v", got, sync)
	}

	gotH, okH := DecodeDMSCHH(*dsb)
	if !okH {
		t.Fatalf("DecodeDMSCHH CRC failed")
	}
	if !reflect.DeepEqual(gotH, seqBits(2, 124)) {
		t.Errorf("SCH/H mismatch")
	}
}

// TestDMTCHSpeechRoundTrip is the voice path: two 137-bit speech frames → the
// EN 300 395-2 TCH/S coding (EncodeTCHS) → scramble with the DM colour code →
// split into a DNB's two blocks → ExtractDMBursts → DMBurstTCHSpeech recovers
// the exact speech frames, CRC-valid. Runs at colour 0 (the default) and a
// non-zero DM colour code to exercise the descramble seed.
func TestDMTCHSpeechRoundTrip(t *testing.T) {
	for _, colour := range []uint32{0, 0x0AB1F} {
		frameA := seqBits(7, tchSpeechFrameBits)
		frameB := seqBits(9, tchSpeechFrameBits)

		descrambled := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), tchType3Bits) // 432 type-4 bits
		onair := descrambled
		if colour != 0 {
			onair = framing.ScrambleTetra(descrambled, colour)
		}
		block1 := onair[:dmBlockDibits*2] // first 216 bits → BKN1
		block2 := onair[dmBlockDibits*2:] // last 216 bits → BKN2

		bursts := ExtractDMBursts(buildDNB(block1, block2), 0)
		dnb := findBurst(bursts, DMBurstNormal)
		if dnb == nil {
			t.Fatalf("colour %#x: no DNB detected (bursts=%d)", colour, len(bursts))
		}
		frames := DMBurstTCHSpeech(*dnb, colour)
		if len(frames) != 2 {
			t.Fatalf("colour %#x: DMBurstTCHSpeech returned %d frames, want 2 (CRC failed?)", colour, len(frames))
		}
		if !reflect.DeepEqual(frames[0], framing.PackBitsMSB(frameA)) {
			t.Errorf("colour %#x: speech frame A mismatch", colour)
		}
		if !reflect.DeepEqual(frames[1], framing.PackBitsMSB(frameB)) {
			t.Errorf("colour %#x: speech frame B mismatch", colour)
		}
	}
}

// TestDMTCHSpeechRotation proves the reported burst rotation de-rotates the
// speech blocks back to the transmitted values through the full decode: the same
// four-rotation robustness the DSB slicer test covers, but carried all the way
// to CRC-valid speech.
func TestDMTCHSpeechRotation(t *testing.T) {
	frameA := seqBits(3, tchSpeechFrameBits)
	frameB := seqBits(11, tchSpeechFrameBits)
	onair := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), tchType3Bits)
	base := buildDNB(onair[:dmBlockDibits*2], onair[dmBlockDibits*2:])

	for rot := uint8(1); rot < 4; rot++ {
		bursts := ExtractDMBursts(rotateDibits(base, rot), 0)
		dnb := findBurst(bursts, DMBurstNormal)
		if dnb == nil {
			t.Fatalf("rot %d: no DNB detected", rot)
		}
		if dnb.Rotation != rot {
			t.Errorf("rot %d: reported rotation %d", rot, dnb.Rotation)
		}
		frames := DMBurstTCHSpeech(*dnb, 0)
		if len(frames) != 2 {
			t.Fatalf("rot %d: speech CRC failed (frames=%d)", rot, len(frames))
		}
		if !reflect.DeepEqual(frames[0], framing.PackBitsMSB(frameA)) {
			t.Errorf("rot %d: frame A mismatch after de-rotation", rot)
		}
	}
}

// dmoNoisyDNB builds a DNB carrying the 432 on-air (post-scramble) type-5 bits
// onair as BOTH the hard dibit blocks (BKN1/BKN2) and the parallel per-symbol
// complex differentials (SoftBKN1/SoftBKN2), adding Gaussian noise of the given
// sigma to each π/4-DQPSK quadrature. Crucially the hard dibits are sign-sliced
// from the SAME noisy differentials the soft path reads, so the hard and soft
// decoders see one identical channel realisation — the only difference is hard
// vs soft decision, which is exactly what the outperform test must isolate.
// Constellation rotation 0 (softType5FromDiffs polarity: Im→b1, Re→b2, +⇒bit 0).
func dmoNoisyDNB(onair []byte, sigma float64, rng *rand.Rand) DMBurst {
	half := dmBlockDibits * 2 // 216 on-air bits per block
	block := func(bits []byte) ([]uint8, []complex64) {
		nsym := len(bits) / 2
		diffs := make([]complex64, nsym)
		hard := make([]byte, len(bits))
		amp := func(b byte) float64 {
			if b&1 == 0 {
				return 1 // bit 0 ⇒ +LLR
			}
			return -1 // bit 1 ⇒ -LLR
		}
		for i := 0; i < nsym; i++ {
			im := amp(bits[2*i]) + rng.NormFloat64()*sigma   // b1 → Im
			re := amp(bits[2*i+1]) + rng.NormFloat64()*sigma // b2 → Re
			diffs[i] = complex(float32(re), float32(im))
			if im < 0 {
				hard[2*i] = 1
			}
			if re < 0 {
				hard[2*i+1] = 1
			}
		}
		return TetraBitsToDibits(hard), diffs
	}
	bkn1, soft1 := block(onair[:half])
	bkn2, soft2 := block(onair[half:])
	return DMBurst{Kind: DMBurstNormal, BKN1: bkn1, BKN2: bkn2, SoftBKN1: soft1, SoftBKN2: soft2}
}

// TestDMTCHSpeechSoftRoundTripClean pins that the soft DNB path is the exact
// soft twin of the hard one: on a noiseless channel ExtractDMBurstsSoft carries
// the differentials, and DMBurstTCHSpeechSoft recovers the same two speech
// frames the hard DMBurstTCHSpeech does — through the four-rotation search and at
// a non-zero DM colour code (the descramble seed).
func TestDMTCHSpeechSoftRoundTripClean(t *testing.T) {
	for _, colour := range []uint32{0, 0x0AB1F} {
		frameA := seqBits(7, tchSpeechFrameBits)
		frameB := seqBits(9, tchSpeechFrameBits)
		descrambled := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), tchType3Bits)
		onair := descrambled
		if colour != 0 {
			onair = framing.ScrambleTetra(descrambled, colour)
		}

		// Build a DNB and its parallel ideal differentials (sigma 0), then run the
		// soft-capable extractor over the correlated stream so the soft fields come
		// from ExtractDMBurstsSoft, not hand-set — exercising the real slicing.
		src := dmoNoisyDNB(onair, 0, rand.New(rand.NewSource(1)))
		dibits := buildDNB(TetraDibitsToBits(src.BKN1), TetraDibitsToBits(src.BKN2))
		diffs := make([]complex64, len(dibits))
		// Place the block differentials at the same offsets buildDNB lays the
		// blocks (ramp(3,12) lead, BKN1, 11-dibit train, BKN2, trailing ramp).
		lead := 12
		copy(diffs[lead:], src.SoftBKN1)
		copy(diffs[lead+dmBlockDibits+len(NormalSyncDibits()):], src.SoftBKN2)

		bursts := ExtractDMBurstsSoft(dibits, diffs, 0)
		dnb := findBurst(bursts, DMBurstNormal)
		if dnb == nil {
			t.Fatalf("colour %#x: no DNB detected", colour)
		}
		if len(dnb.SoftBKN1) != dmBlockDibits || len(dnb.SoftBKN2) != dmBlockDibits {
			t.Fatalf("colour %#x: soft blocks not carried (%d/%d)", colour, len(dnb.SoftBKN1), len(dnb.SoftBKN2))
		}
		frames := DMBurstTCHSpeechSoft(*dnb, colour)
		if len(frames) != 2 {
			t.Fatalf("colour %#x: soft TCH/S returned %d frames, want 2", colour, len(frames))
		}
		if !reflect.DeepEqual(frames[0], framing.PackBitsMSB(frameA)) || !reflect.DeepEqual(frames[1], framing.PackBitsMSB(frameB)) {
			t.Errorf("colour %#x: soft speech frames mismatch", colour)
		}
	}
}

// TestDMTCHSpeechSoftOutperformsHardUnderNoise is the failing-first regression
// for the #1003 soft-decision lever: at a noise level where the hard
// DMBurstTCHSpeech mostly fails the class-2 CRC, the soft DMBurstTCHSpeechSoft
// passes far more of the SAME noisy bursts — the soft-Viterbi coding gain that
// ~doubled the TMO same-carrier yield (#1001), now on the DMO path. The metric
// is class-2-CRC yield (a returned frame pair), exactly what the replay harness
// counts as a CRC-valid TCH/S burst; the CLAUDE.md rule "CRC yield is the only
// trustworthy metric" applies. Exact speech recovery is pinned separately by the
// noiseless TestDMTCHSpeechSoftRoundTripClean — under noise the uncoded class-0
// speech bits legitimately differ even on a CRC pass, so they are not asserted
// here (that is why class-0 is unprotected).
func TestDMTCHSpeechSoftOutperformsHardUnderNoise(t *testing.T) {
	frameA := seqBits(7, tchSpeechFrameBits)
	frameB := seqBits(9, tchSpeechFrameBits)
	onair := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), tchType3Bits) // colour 0

	rng := rand.New(rand.NewSource(42))
	const (
		sigma  = 0.9
		trials = 200
	)
	hardPass, softPass := 0, 0
	for tr := 0; tr < trials; tr++ {
		b := dmoNoisyDNB(onair, sigma, rng)
		if frames := DMBurstTCHSpeech(b, 0); len(frames) == 2 {
			hardPass++
		}
		if frames := DMBurstTCHSpeechSoft(b, 0); len(frames) == 2 {
			softPass++
		}
	}
	t.Logf("sigma=%.2f trials=%d hardPass=%d softPass=%d", sigma, trials, hardPass, softPass)
	if softPass <= hardPass {
		t.Fatalf("soft-decision did not outperform hard: softPass=%d hardPass=%d", softPass, hardPass)
	}
	// Guard against a degenerate "both near zero" pass: soft must recover a
	// meaningful share the hard path misses.
	if softPass < hardPass+trials/10 {
		t.Errorf("soft-decision uplift too small: softPass=%d hardPass=%d (trials=%d)", softPass, hardPass, trials)
	}
}

// TestDMSCHFRoundTrip covers a DNB carrying SCH/F short-data (as opposed to
// speech): the 268→432 chain, recovered CRC-valid.
func TestDMSCHFRoundTrip(t *testing.T) {
	const colour = uint32(0x1234)
	msg := seqBits(5, 268)
	onair := EncodeSCHF(msg, colour) // 432 type-5 bits (bit-per-byte)
	bursts := ExtractDMBursts(buildDNB(onair[:dmBlockDibits*2], onair[dmBlockDibits*2:]), 0)
	dnb := findBurst(bursts, DMBurstNormal)
	if dnb == nil {
		t.Fatalf("no DNB detected")
	}
	got, ok := DecodeDMSCHF(*dnb, colour)
	if !ok {
		t.Fatalf("DecodeDMSCHF CRC failed")
	}
	if !reflect.DeepEqual(got, msg) {
		t.Errorf("SCH/F message mismatch")
	}
}
