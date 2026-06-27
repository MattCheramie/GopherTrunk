package stats

import (
	"bytes"
	"math"
	"testing"
)

func TestShannonEntropyBounds(t *testing.T) {
	t.Parallel()
	// All-same byte -> 0 entropy.
	if h := ShannonEntropy(bytes.Repeat([]byte{0x41}, 100)); h != 0 {
		t.Fatalf("constant data entropy = %v, want 0", h)
	}
	// Uniform 0..255 -> 8 bits.
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	if h := ShannonEntropy(all); math.Abs(h-8) > 1e-9 {
		t.Fatalf("uniform entropy = %v, want 8", h)
	}
}

func TestIndexOfCoincidence(t *testing.T) {
	t.Parallel()
	text := []byte("the quick brown fox jumps over the lazy dog the end")
	ic := IndexOfCoincidence(text)
	// Structured English-ish text sits well above the uniform 1/256.
	if ic < 0.03 {
		t.Fatalf("IC of text = %v, expected > 0.03", ic)
	}
}

func TestGuessXORKeyLength(t *testing.T) {
	t.Parallel()
	pt := []byte("WHEN IN THE COURSE OF HUMAN EVENTS IT BECOMES NECESSARY FOR ONE PEOPLE")
	key := []byte("RADIO")
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ key[i%len(key)]
	}
	scores := GuessXORKeyLength(ct, 16)
	if len(scores) == 0 {
		t.Fatal("no key-length scores")
	}
	// The true length (5) or a multiple should rank near the top.
	topFew := scores
	if len(topFew) > 4 {
		topFew = topFew[:4]
	}
	found := false
	for _, s := range topFew {
		if s.Length == 5 || s.Length == 10 || s.Length == 15 {
			found = true
		}
	}
	if !found {
		t.Fatalf("true key length 5 not in top results: %+v", topFew)
	}
}

func TestCribDrag(t *testing.T) {
	t.Parallel()
	ks := []byte("\x10\x33\x91\x4a\x05\xde\xad\xbe\xef\x77\x21\x44")
	pa := []byte("ATTACK AT DAWN")
	pb := []byte("RETREAT NOWXYZ")
	n := len(ks)
	ca := make([]byte, n)
	cb := make([]byte, n)
	for i := 0; i < n; i++ {
		ca[i] = pa[i] ^ ks[i]
		cb[i] = pb[i] ^ ks[i]
	}
	hits := CribDrag(ca, cb, []byte("ATTACK"))
	// At offset 0, the crib "ATTACK" against (pa^pb) reveals the partner
	// plaintext "RETREA" — printable.
	if len(hits) == 0 || !bytes.Equal(hits[0].Revealed, []byte("RETREA")) {
		t.Fatalf("crib drag offset 0 = %q, want RETREA", hits[0].Revealed)
	}
	if !hits[0].Printable {
		t.Fatal("offset 0 should be printable")
	}
}
