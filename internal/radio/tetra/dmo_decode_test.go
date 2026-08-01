package tetra

import (
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
