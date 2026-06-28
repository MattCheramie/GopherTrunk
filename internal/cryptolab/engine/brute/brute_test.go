package brute

import (
	"bytes"
	"testing"
)

func TestSolveSingleByteXOR(t *testing.T) {
	t.Parallel()
	pt := []byte("The quick brown fox jumps over the lazy dog.")
	const key = 0x5A
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ key
	}
	got := SolveSingleByteXOR(ct, English{})
	if len(got.Key) != 1 || got.Key[0] != key {
		t.Fatalf("recovered key = %v, want %#x", got.Key, key)
	}
	if !bytes.Equal(got.Plaintext, pt) {
		t.Fatalf("plaintext = %q, want %q", got.Plaintext, pt)
	}
}

func TestSolveRepeatingXOR(t *testing.T) {
	t.Parallel()
	pt := []byte("WHEN IN THE COURSE OF HUMAN EVENTS IT BECOMES NECESSARY FOR ONE PEOPLE TO DISSOLVE")
	key := []byte("RADIO")
	ct := XOR{}.Decrypt(key, pt) // XOR is symmetric
	got := SolveRepeatingXOR(ct, len(key), English{})
	if !bytes.Equal(got.Key, key) {
		t.Fatalf("recovered key = %q, want %q", got.Key, key)
	}
	if !bytes.Equal(got.Plaintext, pt) {
		t.Fatalf("plaintext mismatch:\n got %q\nwant %q", got.Plaintext, pt)
	}
}

func TestSolveCaesar(t *testing.T) {
	t.Parallel()
	pt := []byte("The quick brown fox jumps over the lazy dog repeatedly today.")
	const shift = 19
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] + shift
	}
	got := SolveCaesar(ct, English{})
	if len(got.Key) != 1 || got.Key[0] != shift {
		t.Fatalf("recovered shift = %v, want %d", got.Key, shift)
	}
	if !bytes.Equal(got.Plaintext, pt) {
		t.Fatalf("plaintext = %q", got.Plaintext)
	}
}

func TestSolveVigenere(t *testing.T) {
	t.Parallel()
	pt := []byte("WHEN IN THE COURSE OF HUMAN EVENTS IT BECOMES NECESSARY FOR ONE PEOPLE TO DISSOLVE")
	key := []byte{3, 14, 7, 22, 1}
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] + key[i%len(key)]
	}
	got := SolveVigenere(ct, len(key), English{})
	if !bytes.Equal(got.Key, key) {
		t.Fatalf("recovered key = %v, want %v", got.Key, key)
	}
	if !bytes.Equal(got.Plaintext, pt) {
		t.Fatalf("plaintext mismatch:\n got %q\nwant %q", got.Plaintext, pt)
	}
}

func TestCribScorer(t *testing.T) {
	t.Parallel()
	withCrib := Crib{Crib: []byte("RADIO"), Base: Printable{}}
	hit := withCrib.Score([]byte("CALL RADIO NOW"))
	miss := withCrib.Score([]byte("nothing here xx"))
	if hit <= miss+500 {
		t.Fatalf("crib hit (%v) should dominate miss (%v)", hit, miss)
	}
}

func TestByteKeySpaceEnumerates(t *testing.T) {
	t.Parallel()
	ks := &ByteKeySpace{Len: 1}
	if ks.Size() != 256 {
		t.Fatalf("size = %d, want 256", ks.Size())
	}
	var count int
	for {
		_, ok := ks.Next()
		if !ok {
			break
		}
		count++
	}
	if count != 256 {
		t.Fatalf("enumerated %d keys, want 256", count)
	}
	// Seek resets the sweep.
	ks.Seek(254)
	if _, ok := ks.Next(); !ok {
		t.Fatal("expected key at position 254")
	}
}
