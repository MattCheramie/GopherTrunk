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

// TestTCHSSoftRoundTrip proves the soft-decision TCH/S decoder recovers both
// speech frames from a clean (noise-free) ideal-LLR type-5 stream, matching the
// hard decoder bit-for-bit — the soft chain is a faithful mirror of the hard one.
func TestTCHSSoftRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for trial := 0; trial < 50; trial++ {
		a := randBits(rng, tchSpeechFrameBits)
		b := randBits(rng, tchSpeechFrameBits)
		bits := framing.UnpackBitsMSB(EncodeTCHS(a, b), tchType3Bits)
		gotA, gotB, crcOK, _, ok := DecodeTCHSSoft(idealLLR(bits, 1))
		if !ok || !crcOK {
			t.Fatalf("trial %d: soft decode failed on a clean frame (ok=%v crc=%v)", trial, ok, crcOK)
		}
		if !bitsEqual(gotA, a) || !bitsEqual(gotB, b) {
			t.Errorf("trial %d: soft path did not recover the speech frames", trial)
		}
	}
}

// TestTCHSSoftCodingGain is the regression that ties the fix to the reported
// symptom. On the SAME noisy channel, the soft-decision decoder passes the
// class-2 CRC on far more bursts than the hard-decision decoder — the ~2 dB
// coding gain the control channel already enjoyed but TCH/S did not. That gap is
// why ~70% of a same-carrier call's own bursts failed CRC and the recordings
// came out short/garbled (audio_pct ≈ 32%). Fails before the soft path exists.
func TestTCHSSoftCodingGain(t *testing.T) {
	rng := rand.New(rand.NewSource(20260729))
	const trials = 300
	const sigma = 1.0 // AWGN std-dev on unit-amplitude BPSK soft samples
	var hardPass, softPass int
	for i := 0; i < trials; i++ {
		a := randBits(rng, tchSpeechFrameBits)
		b := randBits(rng, tchSpeechFrameBits)
		bits := framing.UnpackBitsMSB(EncodeTCHS(a, b), tchType3Bits)
		llr := noisyLLR(rng, bits, sigma)
		hard := make([]byte, len(llr))
		for j, v := range llr {
			if v < 0 {
				hard[j] = 1
			}
		}
		if len(TCHSpeechFrames(framing.PackBitsMSB(hard))) == 2 {
			hardPass++
		}
		if len(TCHSpeechFramesSoft(llr)) == 2 {
			softPass++
		}
	}
	t.Logf("sigma=%.2f: hardPass=%d/%d softPass=%d/%d", sigma, hardPass, trials, softPass, trials)
	if softPass < 2*hardPass {
		t.Fatalf("soft coding gain too small: soft=%d hard=%d of %d (want soft >= 2x hard)", softPass, hardPass, trials)
	}
}

func idealLLR(bits []byte, amp float32) []float32 {
	out := make([]float32, len(bits))
	for i, b := range bits {
		if b&1 == 0 {
			out[i] = amp
		} else {
			out[i] = -amp
		}
	}
	return out
}

func noisyLLR(rng *rand.Rand, bits []byte, sigma float64) []float32 {
	out := make([]float32, len(bits))
	for i, b := range bits {
		mean := 1.0
		if b&1 == 1 {
			mean = -1.0
		}
		out[i] = float32(mean + rng.NormFloat64()*sigma)
	}
	return out
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

// TestTCHClass2CRCMatchesETSI pins the class-2 CRC to the ETSI EN 300 395-2
// reference (its TAB_CRC parity matrix). The two vectors' expected outputs were
// produced by the reference codec's Build_Crc; an earlier G(X)=1+X³+X⁷ LFSR
// approximation produced different bits, so every real on-air TCH/S burst failed
// the CRC and was dropped (no decoded voice). This guards against regressing to
// any CRC that disagrees with real air.
func TestTCHClass2CRCMatchesETSI(t *testing.T) {
	cases := []struct {
		class2 []byte
		want   []byte
	}{
		{
			class2: []byte{1, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 1, 0, 1},
			want:   []byte{1, 0, 1, 1, 0, 0, 1, 1},
		},
		{
			class2: []byte{1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 0, 1, 0, 1, 1, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0},
			want:   []byte{1, 1, 1, 0, 0, 1, 1, 1},
		},
	}
	for i, c := range cases {
		got := crcTCHClass2(c.class2)
		if !bitsEqual(got, c.want) {
			t.Errorf("case %d: crcTCHClass2 = %v, want %v (ETSI reference)", i, got, c.want)
		}
	}
}

// TestTCHSpeechFramesGate checks the CRC gate: a well-formed traffic frame
// yields the two 137-bit speech frames packed to 18 bytes each, matching the
// input; a random (non-TCH/S) frame is rejected (nil) by the class-2 CRC.
func TestTCHSpeechFramesGate(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	a := randBits(rng, tchSpeechFrameBits)
	b := randBits(rng, tchSpeechFrameBits)
	traffic := EncodeTCHS(a, b)

	frames := TCHSpeechFrames(traffic)
	if len(frames) != 2 {
		t.Fatalf("TCHSpeechFrames returned %d frames, want 2", len(frames))
	}
	wantBytes := (tchSpeechFrameBits + 7) / 8
	for i, f := range frames {
		if len(f) != wantBytes {
			t.Errorf("frame %d = %d bytes, want %d", i, len(f), wantBytes)
		}
	}
	if !bitsEqual(framing.UnpackBitsMSB(frames[0], tchSpeechFrameBits), a) {
		t.Error("frame A not recovered from packed speech frame")
	}
	if !bitsEqual(framing.UnpackBitsMSB(frames[1], tchSpeechFrameBits), b) {
		t.Error("frame B not recovered from packed speech frame")
	}

	// A random 54-byte frame is overwhelmingly not TCH/S: the CRC gate rejects
	// it. Sweep a handful of seeds; none should pass.
	passes := 0
	for s := 0; s < 64; s++ {
		junk := make([]byte, TrafficFrameBytes)
		rng.Read(junk)
		if TCHSpeechFrames(junk) != nil {
			passes++
		}
	}
	if passes > 1 { // ~1/256 chance each; >1 in 64 would be suspicious
		t.Errorf("CRC gate let %d/64 random frames through", passes)
	}
}
