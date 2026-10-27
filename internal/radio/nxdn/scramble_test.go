package nxdn

import (
	"strings"
	"testing"
)

// TestScramblerKeystreamLiteral pins the register to the construction the
// issue #1261 known-key captures decode with (out = bit 0, feedback bit 0
// XOR bit 1 into bit 14): the first 40 keystream bits of three seeds. The
// earlier left-shift x^15+x^14+1 model produced "0000000000000001…" for
// key 1 and decoded neither capture.
func TestScramblerKeystreamLiteral(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		key  uint16
		bits string
	}{
		{1, "1000000000000001000000000000011000000000"},
		{0x4000, "0000000000000010000000000000110000000000"},
		{32767, "1111111111111110000000000000010000000000"},
		{12345, "1001110000001101010010000010111110110000"},
	} {
		var sb strings.Builder
		for _, b := range NewScrambler(c.key).Generate(len(c.bits)) {
			sb.WriteByte('0' + b)
		}
		if got := sb.String(); got != c.bits {
			t.Errorf("key %d keystream\n got %s\nwant %s", c.key, got, c.bits)
		}
	}
}

// TestScrambleRoundTrip is the core invariant: descrambling with the same key
// recovers the original bits for any key, because the keystream XOR is
// symmetric.
func TestScrambleRoundTrip(t *testing.T) {
	t.Parallel()
	// A deterministic pseudo-random bit pattern (no math/rand to keep the test
	// reproducible and -race friendly).
	bits := make([]byte, 288) // one NXDN information field's worth of bits
	x := uint32(0x1234_5678)
	for i := range bits {
		x = x*1664525 + 1013904223 // LCG
		bits[i] = byte((x >> 24) & 1)
	}
	for _, key := range []uint16{1, 2, 7, 0x1F, 0x2A3, 0x4000, ScramblerKeyMax} {
		sc := Scramble(bits, key)
		// Scrambling must actually change the data (key != 0).
		same := true
		for i := range bits {
			if sc[i] != bits[i] {
				same = false
				break
			}
		}
		if same {
			t.Errorf("key %d: scrambled output identical to input", key)
		}
		got := Descramble(sc, key)
		for i := range bits {
			if got[i] != bits[i] {
				t.Fatalf("key %d: round-trip mismatch at bit %d", key, i)
			}
		}
	}
}

// TestScrambleKeyZeroIsClear documents that key 0 is the "no scrambling" case:
// the keystream is all zero, so Scramble is the identity.
func TestScrambleKeyZeroIsClear(t *testing.T) {
	t.Parallel()
	bits := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0}
	got := Scramble(bits, 0)
	for i := range bits {
		if got[i] != bits[i] {
			t.Fatalf("key 0 changed bit %d: got %d want %d", i, got[i], bits[i])
		}
	}
}

// TestKeystreamIsBalancedMSequence checks the LFSR is maximal length: over one
// full period (2^15-1 = 32767 bits) a maximal m-sequence is balanced, with
// exactly 2^14 = 16384 ones and 16383 zeros, and the state returns to its
// seed. Skip must agree with Generate.
func TestKeystreamIsBalancedMSequence(t *testing.T) {
	t.Parallel()
	const period = ScramblerKeyMax // 32767
	ks := NewScrambler(1).Generate(period)
	ones := 0
	for _, b := range ks {
		ones += int(b)
	}
	if ones != 16384 {
		t.Fatalf("m-sequence not balanced: %d ones over period %d, want 16384", ones, period)
	}
	s := NewScrambler(1)
	s.Generate(period)
	if s.state != 1 {
		t.Fatalf("LFSR period != %d: state after one period is %#x, want seed 0x1", period, s.state)
	}
	a, b := NewScrambler(777), NewScrambler(777)
	a.Skip(ScramblerBitsPerFrame)
	b.Generate(ScramblerBitsPerFrame)
	if a.state != b.state {
		t.Fatalf("Skip and Generate disagree: %#x vs %#x", a.state, b.state)
	}
}
