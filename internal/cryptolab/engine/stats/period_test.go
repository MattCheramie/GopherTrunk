package stats

import (
	"strings"
	"testing"
)

func TestDetectPeriodFindsRepeatingXORKeyLength(t *testing.T) {
	t.Parallel()
	// Repeating-key XOR with a 7-byte key over English text: the ciphertext
	// autocorrelation should peak at lag 7 (and its multiples). Enough text is
	// needed for the peak to clear the baseline.
	pt := []byte(strings.Repeat("the quick brown fox jumps over the lazy dog ", 8))
	key := []byte("SECRET!")
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ key[i%len(key)]
	}
	cands := DetectPeriod(ct, 40, 5)
	if len(cands) == 0 {
		t.Fatal("no period detected")
	}
	found := false
	for _, c := range cands {
		if c.Lag%7 == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a multiple of 7 among candidates, got %+v", cands)
	}
}

func TestDetectPeriodQuietOnRandom(t *testing.T) {
	t.Parallel()
	// A simple LCG byte stream has no short byte period; expect few/no peaks.
	data := make([]byte, 4000)
	x := uint32(12345)
	for i := range data {
		x = x*1664525 + 1013904223
		data[i] = byte(x >> 24)
	}
	cands := DetectPeriod(data, 256, 5)
	if len(cands) > 2 {
		t.Fatalf("unexpected strong periodicity in pseudo-random data: %+v", cands)
	}
}

func TestTopNGramsCountsRepeats(t *testing.T) {
	t.Parallel()
	data := []byte("ababab")
	g := TopNGrams(data, 2, 3)
	if len(g) == 0 || g[0].Count < 2 {
		t.Fatalf("expected a repeated bigram, got %+v", g)
	}
}
