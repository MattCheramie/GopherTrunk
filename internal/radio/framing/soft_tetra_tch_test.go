package framing

import (
	"math/rand"
	"testing"
)

// bitsToLLR maps hard bits to ideal soft LLRs under the package convention
// LLR > 0 ⇒ bit 0, LLR < 0 ⇒ bit 1.
func bitsToLLR(bits []byte) []float32 {
	out := make([]float32, len(bits))
	for i, b := range bits {
		if b&1 == 0 {
			out[i] = 1
		} else {
			out[i] = -1
		}
	}
	return out
}

// TestDecodeRCPCTetraMotherSoftMatchesHard: on a clean channel the soft
// rate-1/3 TCH mother decoder recovers the same information bits as the hard
// decoder, and a single flipped soft bit is still corrected (K=5 free distance
// 12). This pins the soft trellis (3 outputs/stage, TCH polynomials) as a
// faithful mirror of DecodeRCPCTetraMother.
func TestDecodeRCPCTetraMotherSoftMatchesHard(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for trial := 0; trial < 20; trial++ {
		info := make([]byte, 40)
		for i := range info {
			info[i] = byte(rng.Intn(2))
		}
		stages := len(info) + 4 // K-1 tail bits
		in := make([]byte, stages)
		copy(in, info)
		channel := EncodeRCPCTetraMother(in)

		soft := bitsToLLR(channel)
		out, _ := DecodeRCPCTetraMotherSoft(soft, stages)
		for i, want := range in {
			if out[i] != want {
				t.Fatalf("trial %d clean: bit %d = %d, want %d", trial, i, out[i], want)
			}
		}
		// One corrupted soft bit (sign flip) must still be corrected.
		soft[7] = -soft[7]
		out2, _ := DecodeRCPCTetraMotherSoft(soft, stages)
		for i, want := range in {
			if out2[i] != want {
				t.Fatalf("trial %d 1-error: bit %d = %d, want %d", trial, i, out2[i], want)
			}
		}
	}
}

// TestDepunctureRCPCTetraSoftMirrorsHard: the soft depuncture places received
// LLRs at exactly the same mother-code positions the hard DepunctureRCPCTetra
// writes bits to, and leaves every punctured position at the 0.0 soft erasure
// (never the hard DepunctureMark). Checked for both TCH puncture schemes.
func TestDepunctureRCPCTetraSoftMirrorsHard(t *testing.T) {
	cases := []struct {
		name      string
		coded     int
		period    int
		puncture  []int
		motherLen int
	}{
		{"rate23-class1", 168, RCPCTetraPeriod23, RCPCTetraPuncture23, 3 * 112},
		{"rate818-class2", 162, RCPCTetraPeriod818, RCPCTetraPuncture818, 3 * (60 + 8 + 4)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hardIn := make([]byte, tc.coded)
			softIn := make([]float32, tc.coded)
			for i := range hardIn {
				hardIn[i] = byte((i % 2) ^ 1) // arbitrary 0/1 pattern
				softIn[i] = bitsToLLR(hardIn[i : i+1])[0]
			}
			hard := DepunctureRCPCTetra(hardIn, tc.period, tc.puncture, tc.motherLen)
			soft := DepunctureRCPCTetraSoft(softIn, tc.period, tc.puncture, tc.motherLen)
			if len(soft) != len(hard) {
				t.Fatalf("length mismatch: soft=%d hard=%d", len(soft), len(hard))
			}
			for i := range hard {
				if hard[i] == DepunctureMark {
					if soft[i] != 0 {
						t.Errorf("pos %d: punctured, soft=%v want 0.0 erasure", i, soft[i])
					}
					continue
				}
				// A carried position: soft sign must agree with the hard bit.
				wantNeg := hard[i] == 1
				if (soft[i] < 0) != wantNeg {
					t.Errorf("pos %d: hard bit %d but soft LLR %v", i, hard[i], soft[i])
				}
			}
		})
	}
}
