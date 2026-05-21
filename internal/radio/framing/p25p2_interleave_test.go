package framing

import (
	"bytes"
	"testing"
)

func TestMACBurstInterleaveRoundTrip(t *testing.T) {
	for _, n := range []int{0, 2, 72, 146, 360} {
		in := make([]uint8, n)
		for i := range in {
			in[i] = uint8((i*7 + 1) & 3)
		}
		got := DeinterleaveMACBurst(InterleaveMACBurst(in))
		if !bytes.Equal(got, in) {
			t.Errorf("n=%d: round-trip mismatch", n)
		}
	}
}

func TestMACBurstInterleaveIsPermutation(t *testing.T) {
	const n = 146 // the trellis-coded MAC-burst dibit count
	in := make([]uint8, n)
	for i := range in {
		in[i] = uint8(i)
	}
	out := InterleaveMACBurst(in)
	if len(out) != n {
		t.Fatalf("interleave changed length: %d -> %d", n, len(out))
	}
	seen := make([]bool, n)
	for _, v := range out {
		if seen[v] {
			t.Fatalf("interleaver is not a bijection: index %d repeats", v)
		}
		seen[v] = true
	}
	if bytes.Equal(out, in) {
		t.Error("interleaver is the identity permutation")
	}
}

func TestMACBurstInterleaveOddLengthPassthrough(t *testing.T) {
	in := []uint8{1, 2, 3}
	if got := InterleaveMACBurst(in); !bytes.Equal(got, in) {
		t.Errorf("odd-length input should pass through unchanged, got %v", got)
	}
}
