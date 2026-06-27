package lfsr

import (
	"reflect"
	"testing"
)

func TestBitsRoundTrip(t *testing.T) {
	t.Parallel()
	data := []byte{0xA5, 0x3C, 0xFF, 0x00}
	if got := BitsToBytes(BitsFromBytes(data)); !reflect.DeepEqual(got, data) {
		t.Fatalf("round trip = %x, want %x", got, data)
	}
}

func TestBerlekampMasseyRecoversLFSR(t *testing.T) {
	t.Parallel()
	// A degree-4 LFSR x^4 + x + 1 (taps at 4 and 1), classic maximal-length.
	taps := []int{1, 4}
	seed := []byte{1, 0, 0, 0}
	seq, err := Generate(taps, seed, 60)
	if err != nil {
		t.Fatal(err)
	}
	res := BerlekampMassey(seq)
	if res.LinearComplexity != 4 {
		t.Fatalf("linear complexity = %d, want 4", res.LinearComplexity)
	}
	// BM should recover the same feedback taps.
	got := res.Taps()
	if !reflect.DeepEqual(got, taps) {
		t.Fatalf("recovered taps = %v, want %v (poly %v)", got, taps, res.Polynomial)
	}
}

func TestBerlekampMasseyRandomLooksComplex(t *testing.T) {
	t.Parallel()
	// A sequence with no short LFSR should yield complexity ~ n/2.
	seq := BitsFromBytes([]byte{0x9e, 0x37, 0x79, 0xb9, 0x7f, 0x4a, 0x7c, 0x15})
	res := BerlekampMassey(seq)
	if res.LinearComplexity < len(seq)/3 {
		t.Fatalf("complexity %d unexpectedly low for %d bits", res.LinearComplexity, len(seq))
	}
}

func TestKeystreamExtraction(t *testing.T) {
	t.Parallel()
	pt := []byte("HELLO RADIO")
	ks := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ ks[i]
	}
	if got := Keystream(pt, ct); !reflect.DeepEqual(got, ks) {
		t.Fatalf("keystream = %v, want %v", got, ks)
	}
}
