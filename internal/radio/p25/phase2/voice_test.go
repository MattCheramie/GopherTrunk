package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// voiceInfoPattern returns a packed vocoder frame with a deterministic bit
// pattern — the form the encoder takes and the decoder returns.
func voiceInfoPattern(seed byte) []byte {
	bits := make([]byte, VoiceInfoBits)
	for i := range bits {
		bits[i] = (seed + byte(i)) & 1
	}
	return framing.PackBitsMSB(bits)
}

// TestVoiceBurstRoundTrip exercises the encoder against the decoder across
// both voice burst kinds and every slot phase, including the PN44 descramble.
func TestVoiceBurstRoundTrip(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	for _, bt := range []BurstType{BurstVoice4, BurstVoice2} {
		n := len(VoiceFrameOffsets(bt))
		want := make([][]byte, n)
		for i := range want {
			want[i] = voiceInfoPattern(byte(i))
		}
		for slot := 0; slot < SubframesPerSuperframe; slot++ {
			burst := EncodeVoiceBurst(bt, want, slot, seq)
			if got := BurstTypeOf(burst); got != bt {
				t.Fatalf("burst type %d round-tripped as %d", bt, got)
			}
			got, errs, unc, err := ExtractBurstVoiceFrames(burst, slot, seq)
			if err != nil {
				t.Fatalf("type %d slot %d: %v", bt, slot, err)
			}
			if errs != 0 || unc != 0 {
				t.Errorf("type %d slot %d: clean burst reported errs=%d unc=%d", bt, slot, errs, unc)
			}
			if len(got) != n {
				t.Fatalf("type %d slot %d: %d frames, want %d", bt, slot, len(got), n)
			}
			for i := range got {
				if string(got[i]) != string(want[i]) {
					t.Errorf("type %d slot %d frame %d mismatch", bt, slot, i)
				}
			}
		}
	}
}

// TestVoiceBurstCorrectsBitErrors: the AMBE codeword's first two fields are
// Golay-protected, so up to three bit errors in each are repaired and the
// vocoder frame comes back intact.
func TestVoiceBurstCorrectsBitErrors(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	want := [][]byte{voiceInfoPattern(0), voiceInfoPattern(1), voiceInfoPattern(2), voiceInfoPattern(3)}
	const slot = 5
	burst := EncodeVoiceBurst(BurstVoice4, want, slot, seq)
	// Flip three dibits inside the first frame's c0 column (on-air bits 4k).
	for k := 0; k < 3; k++ {
		burst[voiceFrameOffsets[0]+2*k] ^= 2
	}
	got, errs, unc, err := ExtractBurstVoiceFrames(burst, slot, seq)
	if err != nil {
		t.Fatal(err)
	}
	if unc != 0 {
		t.Errorf("uncorrectable=%d, want 0", unc)
	}
	if errs == 0 {
		t.Error("errs=0; the damage should have been corrected, not absent")
	}
	if string(got[0]) != string(want[0]) {
		t.Error("frame 0 not repaired")
	}
}

// TestVoiceBurstWrongSlotIsNoise pins the discriminator the slot-phase vote
// relies on: at the wrong scramble phase the Golay decoders work near their
// correction radius on every frame.
func TestVoiceBurstWrongSlotIsNoise(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	want := [][]byte{voiceInfoPattern(0), voiceInfoPattern(1), voiceInfoPattern(2), voiceInfoPattern(3)}
	const slot = 4
	burst := EncodeVoiceBurst(BurstVoice4, want, slot, seq)
	right, ok := VoiceBurstGolayErrors(burst, slot, seq)
	if !ok || right != 0 {
		t.Fatalf("correct slot reported errs=%d ok=%v, want 0/true", right, ok)
	}
	for other := 0; other < SubframesPerSuperframe; other++ {
		if other == slot {
			continue
		}
		wrong, ok := VoiceBurstGolayErrors(burst, other, seq)
		if !ok {
			continue
		}
		if wrong <= voiceSlotVoteMaxErrs {
			t.Errorf("slot %d (wrong) reported only %d corrected bits; the vote "+
				"threshold is %d, so it would tie with the right phase",
				other, wrong, voiceSlotVoteMaxErrs)
		}
	}
}

func TestVoiceFrameOffsetsLandOnDUIDBoundaries(t *testing.T) {
	// The geometry is self-checking: each frame ends exactly where a DUID
	// dibit sits, and the ESS fills the gap after frame 1.
	if got := VoiceFrameOffsets(BurstVoice4); len(got) != 4 {
		t.Fatalf("4V has %d frames, want 4", len(got))
	}
	if got := len(VoiceFrameOffsets(BurstVoice2)); got != 2 {
		t.Fatalf("2V has %d frames, want 2", got)
	}
	if VoiceFrameOffsets(BurstFACCHScrambled) != nil {
		t.Error("an ACCH burst reported voice frames")
	}
	// The burst tiles exactly, with nothing left over:
	//   DUID 20 | f0 21..56 | DUID 57 | f1 58..93 | ESS 94..105 |
	//   f2 106..141 | DUID 142 | f3 143..178 | DUID 179
	duid := map[int]bool{}
	for _, p := range duidDibitPositions {
		duid[p] = true
	}
	off := VoiceFrameOffsets(BurstVoice4)
	for _, i := range []int{0, 1, 3} {
		if !duid[off[i]-1] {
			t.Errorf("frame %d starts at %d, which does not follow a DUID dibit", i, off[i])
		}
	}
	for _, i := range []int{0, 2, 3} {
		if end := off[i] + VoiceCodewordDibits; !duid[end] {
			t.Errorf("frame %d ends at %d, which is not a DUID dibit", i, end)
		}
	}
	if got := off[1] + VoiceCodewordDibits; got != ESSDibitOffset {
		t.Errorf("frame 1 ends at %d, but the ESS starts at %d", got, ESSDibitOffset)
	}
	if got := ESSDibitOffset + ESSDibits; got != off[2] {
		t.Errorf("ESS ends at %d, but frame 2 starts at %d", got, off[2])
	}
	if got := off[3] + VoiceCodewordDibits; got != BurstDibits-1 {
		t.Errorf("frame 3 ends at %d, want the final DUID dibit at %d", got, BurstDibits-1)
	}
}

// voicePayloads builds n silent vocoder payloads for fixtures that need voice
// sub-frames as filler.
func voicePayloads(n int) [][]byte {
	out := make([][]byte, n)
	for i := range out {
		out[i] = make([]byte, VoiceFrameBytes)
	}
	return out
}
