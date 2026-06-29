package randomness

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
)

// randomBits returns n pseudo-random 0/1 values from a seeded generator — a
// good PRNG, so the battery should clear every applicable test.
func randomBits(seed int64, n int) []byte {
	r := rand.New(rand.NewSource(seed))
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(r.Intn(2))
	}
	return out
}

func TestBatteryAcceptsRandomStream(t *testing.T) {
	t.Parallel()
	rep := Battery(randomBits(1, 200000), DefaultAlpha)
	if rep.Failed > 1 {
		t.Fatalf("random stream failed %d tests (want <=1): %+v", rep.Failed, rep.WeakestTests()[:3])
	}
	if rep.Passed < 8 {
		t.Fatalf("random stream only passed %d tests: %+v", rep.Passed, rep.Tests)
	}
}

func TestBatteryRejectsShortLFSR(t *testing.T) {
	t.Parallel()
	// A degree-9 LFSR has linear complexity 9 over any block, far below the
	// ~M/2 a random stream shows — the linear-complexity and spectral tests
	// must catch it.
	seq, err := lfsr.Generate([]int{9, 5}, []byte{1, 0, 1, 1, 0, 0, 1, 0, 1}, 200000)
	if err != nil {
		t.Fatal(err)
	}
	rep := Battery(seq, DefaultAlpha)
	if rep.LooksRandom() {
		t.Fatalf("short LFSR stream judged random: %+v", rep.Tests)
	}
	lc := find(rep, "linear_complexity")
	if lc == nil || !lc.Applicable || lc.Passed {
		t.Fatalf("linear_complexity should fail on a short LFSR, got %+v", lc)
	}
}

func TestBatteryRejectsConstantStream(t *testing.T) {
	t.Parallel()
	bits := make([]byte, 100000) // all zeros
	rep := Battery(bits, DefaultAlpha)
	mb := find(rep, "monobit")
	if mb == nil || mb.Passed {
		t.Fatalf("monobit should fail on all-zero stream, got %+v", mb)
	}
	if rep.LooksRandom() {
		t.Fatal("all-zero stream judged random")
	}
}

func TestBatteryRejectsPeriodicStream(t *testing.T) {
	t.Parallel()
	// A short repeating pattern is highly periodic — the spectral test should
	// flag it.
	pat := []byte{1, 1, 0, 0, 1, 0, 1, 0}
	bits := make([]byte, 0, 100000)
	for len(bits) < 100000 {
		bits = append(bits, pat...)
	}
	rep := Battery(bits[:100000], DefaultAlpha)
	if rep.LooksRandom() {
		t.Fatalf("periodic stream judged random: %+v", rep.Tests)
	}
}

func TestQuickSubset(t *testing.T) {
	t.Parallel()
	rep := Quick(randomBits(7, 20000), DefaultAlpha)
	if len(rep.Tests) != 4 {
		t.Fatalf("quick should run 4 tests, ran %d", len(rep.Tests))
	}
	if rep.Failed > 1 {
		t.Fatalf("quick failed %d on a random stream", rep.Failed)
	}
}

func find(r Report, name string) *TestResult {
	for i := range r.Tests {
		if r.Tests[i].Name == name {
			return &r.Tests[i]
		}
	}
	return nil
}
