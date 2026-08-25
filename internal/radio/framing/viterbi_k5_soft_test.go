package framing

import (
	"math/rand"
	"testing"
)

// bitToLLR maps a hard channel bit to its ideal LLR under the soft_tetra.go
// convention: LLR > 0 ⇒ bit 0, LLR < 0 ⇒ bit 1.
func bitToLLR(b byte) float32 {
	if b&1 == 1 {
		return -1
	}
	return 1
}

// TestViterbiK5SoftCleanMatchesHard: on ideal (noiseless) LLRs the soft
// decoder recovers exactly the encoder's input — anchors the bit and
// polynomial conventions against EncodeK5/ViterbiK5.
func TestViterbiK5SoftCleanMatchesHard(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 200; trial++ {
		stages := 24 + rng.Intn(64)
		in := make([]byte, stages)
		for i := 0; i < stages-4; i++ { // last 4 stay zero (tail flush)
			in[i] = byte(rng.Intn(2))
		}
		ch := EncodeK5(in)
		llr := make([]float32, len(ch))
		for i, b := range ch {
			llr[i] = bitToLLR(b)
		}
		got, _ := ViterbiK5Soft(llr, stages)
		for i := range in {
			if got[i] != in[i] {
				t.Fatalf("trial %d: soft decode wrong at %d (got %d want %d)", trial, i, got[i], in[i])
			}
		}
	}
}

// TestViterbiK5SoftErasuresMatchDepunctureMark: an LLR of 0.0 must behave
// exactly like the hard path's DepunctureMark — no cost contribution — so a
// punctured stream decodes identically through both.
func TestViterbiK5SoftErasuresMatchDepunctureMark(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 100; trial++ {
		stages := 40
		in := make([]byte, stages)
		for i := 0; i < stages-4; i++ {
			in[i] = byte(rng.Intn(2))
		}
		ch := EncodeK5(in)
		hard := make([]byte, len(ch))
		llr := make([]float32, len(ch))
		for i, b := range ch {
			if rng.Intn(7) == 0 { // ~1/7 punctured, like the NXDN CAC rate
				hard[i] = DepunctureMark
				llr[i] = 0
				continue
			}
			hard[i] = b
			llr[i] = bitToLLR(b)
		}
		wantBits, _ := ViterbiK5(hard, stages)
		gotBits, _ := ViterbiK5Soft(llr, stages)
		for i := range wantBits {
			if gotBits[i] != wantBits[i] {
				t.Fatalf("trial %d: soft/hard disagree at %d under erasures", trial, i)
			}
		}
	}
}

// TestViterbiK5SoftBeatsHard: over an AWGN channel the soft decoder has a
// strictly lower info-bit error rate than hard-sliced ViterbiK5 — the
// coding-gain claim the NXDN nxdn_soft_decision knob rests on.
func TestViterbiK5SoftBeatsHard(t *testing.T) {
	const stages = 175 // the NXDN CAC's 155 info + 16 CRC + 4 tail shape
	for _, sigma := range []float64{0.8, 1.0} {
		rng := rand.New(rand.NewSource(9))
		var hardErr, softErr, total int
		for trial := 0; trial < 200; trial++ {
			in := make([]byte, stages)
			for i := 0; i < stages-4; i++ {
				in[i] = byte(rng.Intn(2))
			}
			ch := EncodeK5(in)
			hard := make([]byte, len(ch))
			llr := make([]float32, len(ch))
			for i, b := range ch {
				v := float64(bitToLLR(b)) + rng.NormFloat64()*sigma
				llr[i] = float32(v)
				if v < 0 {
					hard[i] = 1
				}
			}
			hbits, _ := ViterbiK5(hard, stages)
			sbits, _ := ViterbiK5Soft(llr, stages)
			for i := range in {
				if hbits[i] != in[i] {
					hardErr++
				}
				if sbits[i] != in[i] {
					softErr++
				}
				total++
			}
		}
		hBER := float64(hardErr) / float64(total)
		sBER := float64(softErr) / float64(total)
		t.Logf("sigma=%.2f  hard BER=%.4f  soft BER=%.4f", sigma, hBER, sBER)
		if sBER >= hBER {
			t.Errorf("sigma=%.2f: soft BER %.4f not better than hard %.4f", sigma, sBER, hBER)
		}
	}
}
