package tetra

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestTCHSRoundTrip proves the TCH/S §5.5 chain is invertible end to end:
// two 137-bit speech frames encode to a 54-byte type-4 frame and decode back
// bit-for-bit with a valid CRC.
func TestTCHSRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 50; trial++ {
		a := randBits(rng, tchSpeechFrameBits)
		b := randBits(rng, tchSpeechFrameBits)
		frame := EncodeTCHS(a, b)
		if len(frame) != TrafficFrameBytes {
			t.Fatalf("encoded frame = %d bytes, want %d", len(frame), TrafficFrameBytes)
		}
		gotA, gotB, crcOK, errs, ok := DecodeTCHS(frame)
		if !ok {
			t.Fatal("DecodeTCHS ok=false on a well-formed frame")
		}
		if !crcOK {
			t.Errorf("trial %d: CRC failed on a clean frame", trial)
		}
		if errs != 0 {
			t.Errorf("trial %d: Viterbi corrected %d bits on a clean frame", trial, errs)
		}
		if !bitsEqual(gotA, a) || !bitsEqual(gotB, b) {
			t.Errorf("trial %d: speech frames not recovered", trial)
		}
	}
}

// TestTCHSCorrectsSingleError verifies the convolutional protection fixes a
// single channel-bit error in the coded region and still recovers the speech.
func TestTCHSCorrectsSingleError(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	a := randBits(rng, tchSpeechFrameBits)
	b := randBits(rng, tchSpeechFrameBits)
	bits := framing.UnpackBitsMSB(EncodeTCHS(a, b), tchType3Bits)
	// Flip one type-4 bit in the coded (non-class-0) region — deinterleave
	// spreads it, and the RCPC corrects it.
	bits[400] ^= 1
	gotA, gotB, crcOK, errs, _ := DecodeTCHS(framing.PackBitsMSB(bits))
	if !crcOK {
		t.Fatalf("CRC failed after a correctable single-bit error")
	}
	if errs == 0 {
		t.Errorf("expected the Viterbi metric to reflect a corrected bit")
	}
	if !bitsEqual(gotA, a) || !bitsEqual(gotB, b) {
		t.Errorf("speech not recovered after single-bit error correction")
	}
}

func randBits(rng *rand.Rand, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(rng.Intn(2))
	}
	return out
}

func bitsEqual(x, y []byte) bool {
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if (x[i] & 1) != (y[i] & 1) {
			return false
		}
	}
	return true
}
