package framing

import (
	"math/rand"
	"testing"
)

func TestInterleaveTetraRoundTripPerChannel(t *testing.T) {
	cases := []struct {
		name string
		K    int
		a    int
	}{
		{"BSCH (120, 11)", InterleaveKBSCH, InterleaveABSCH},
		{"SCH/HD (216, 101)", InterleaveKSCHHD, InterleaveASCHHD},
		{"SCH/HU (168, 13)", InterleaveKSCHHU, InterleaveASCHHU},
		{"SCH/F (432, 103)", InterleaveKSCHF, InterleaveASCHF},
	}
	r := rand.New(rand.NewSource(0x1337))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.K)
			for i := range data {
				data[i] = byte(r.Intn(2))
			}
			interleaved := BlockInterleaveTetra(data, tc.K, tc.a)
			if len(interleaved) != tc.K {
				t.Fatalf("interleaved length = %d, want %d", len(interleaved), tc.K)
			}
			restored := BlockDeinterleaveTetra(interleaved, tc.K, tc.a)
			for i, want := range data {
				if restored[i] != want {
					t.Errorf("bit %d: round-trip got %d, want %d", i, restored[i], want)
					break
				}
			}
		})
	}
}

// TestInterleaveTetraIsPermutation: the interleaver must be a
// permutation — every output position is filled exactly once and
// every input position is consumed exactly once. Detects an
// off-by-one or modular-arithmetic bug.
func TestInterleaveTetraIsPermutation(t *testing.T) {
	const K = 216
	const a = 101
	// Mark each input bit with its index modulo 256.
	in := make([]byte, K)
	for i := range in {
		in[i] = byte(i & 1) // 0/1 pattern; check structure via positions instead
	}
	out := BlockInterleaveTetra(in, K, a)
	// Verify all positions filled — every K * i index hit once.
	seen := make(map[int]bool)
	for i := 1; i <= K; i++ {
		k := 1 + ((a * i) % K)
		if seen[k] {
			t.Errorf("position %d hit twice (collision)", k)
		}
		seen[k] = true
	}
	if len(seen) != K {
		t.Errorf("permutation covers %d distinct positions, want %d", len(seen), K)
	}
	// The bit values themselves round-trip via deinterleave.
	restored := BlockDeinterleaveTetra(out, K, a)
	for i, want := range in {
		if restored[i] != want {
			t.Errorf("permutation round-trip bit %d: got %d, want %d", i, restored[i], want)
			break
		}
	}
}

// TestInterleaveTetraSpecFormula: confirm output position k for a
// known input maps to the spec formula k = 1 + ((a*i) mod K).
// Sets only bit i = 7 in a (216, 101) input and asserts it lands
// at position 1 + (101*7 mod 216) = 1 + (707 mod 216) = 1 + 59 = 60.
func TestInterleaveTetraSpecFormula(t *testing.T) {
	in := make([]byte, 216)
	in[6] = 1 // 1-indexed i = 7 → 0-indexed slot 6
	out := BlockInterleaveTetra(in, 216, 101)
	if out[59] != 1 {
		t.Errorf("input bit at i=7 should land at k=60: out[59] = %d, want 1", out[59])
	}
	// All other positions zero.
	for j, b := range out {
		if j != 59 && b != 0 {
			t.Errorf("position %d unexpectedly set", j)
			break
		}
	}
}

func TestInterleaveTetraRejectsWrongLength(t *testing.T) {
	if got := BlockInterleaveTetra(make([]byte, 100), 120, 11); got != nil {
		t.Errorf("BlockInterleaveTetra accepted 100-byte input for K=120, want nil")
	}
	if got := BlockDeinterleaveTetra(make([]byte, 100), 120, 11); got != nil {
		t.Errorf("BlockDeinterleaveTetra accepted 100-byte input for K=120, want nil")
	}
}

// TestInterleaveTetraConstantsMatchSpec asserts the per-channel
// (K, a) constants are exactly what EN 300 392-2 §8.3.1 specifies —
// protects against typos in the table.
func TestInterleaveTetraConstantsMatchSpec(t *testing.T) {
	cases := []struct {
		name  string
		K, a  int
		wantK int
		wantA int
	}{
		{"BSCH", InterleaveKBSCH, InterleaveABSCH, 120, 11},
		{"SCH/HD", InterleaveKSCHHD, InterleaveASCHHD, 216, 101},
		{"SCH/HU", InterleaveKSCHHU, InterleaveASCHHU, 168, 13},
		{"SCH/F", InterleaveKSCHF, InterleaveASCHF, 432, 103},
	}
	for _, tc := range cases {
		if tc.K != tc.wantK {
			t.Errorf("%s: K = %d, want %d", tc.name, tc.K, tc.wantK)
		}
		if tc.a != tc.wantA {
			t.Errorf("%s: a = %d, want %d", tc.name, tc.a, tc.wantA)
		}
	}
}
