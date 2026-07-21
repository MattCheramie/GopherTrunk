package framing

import (
	"math/rand"
	"testing"
)

// TestScrambleTetraXORIsSymmetric: scrambling twice with the same
// colour code returns the original data — the XOR mask is the
// same on both passes.
func TestScrambleTetraXORIsSymmetric(t *testing.T) {
	cases := []struct {
		name       string
		colourCode uint32
	}{
		{"BSCH (all zeros)", 0},
		{"single bit", 1},
		{"alternating", 0x2AAAAAAA},
		{"full 30-bit", 0x3FFFFFFF},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0, 0, 1, 0, 1}
			scrambled := ScrambleTetra(data, tc.colourCode)
			restored := DescrambleTetra(scrambled, tc.colourCode)
			for i, want := range data {
				if restored[i] != want {
					t.Errorf("bit %d: scrambled then descrambled = %d, want %d", i, restored[i], want)
				}
			}
		})
	}
}

// TestScrambleTetraBSCHInitialState: with colour code 0 (BSCH),
// the initial state is 0xC0000000 (p(-31) = p(-30) = 1, the rest
// zero). The first output bit p(1) is the XOR of the tapped state
// bits — tap mask 0x82608EDB AND state 0xC0000000 = 0x80000000
// (only bit 31 set), so popcount is 1 → p(1) = 1.
func TestScrambleTetraBSCHInitialState(t *testing.T) {
	s := NewScramblerTetra(0)
	if got := s.Next(); got != 1 {
		t.Errorf("p(1) for BSCH (colour 0) = %d, want 1", got)
	}
}

// TestScrambleTetraDifferentColourCodes: two different colour
// codes produce different scrambling sequences (sanity check on
// the seeding path).
func TestScrambleTetraDifferentColourCodes(t *testing.T) {
	a := NewScramblerTetra(0x12345).Generate(120)
	b := NewScramblerTetra(0x54321).Generate(120)
	if equalBytes(a, b) {
		t.Errorf("scrambling sequences for colour codes 0x12345 and 0x54321 are identical")
	}
}

// TestScrambleTetraSequenceHasReasonableEntropy: a 1 000-bit
// scrambling sequence should be roughly balanced (more than 30%
// ones, fewer than 70% — the LFSR's m-sequence properties give
// near-50/50 in practice).
func TestScrambleTetraSequenceHasReasonableEntropy(t *testing.T) {
	seq := NewScramblerTetra(0x1B2D4F).Generate(1000)
	ones := 0
	for _, b := range seq {
		ones += int(b)
	}
	if ones < 300 || ones > 700 {
		t.Errorf("balance of scrambling sequence = %d ones in 1000 bits (out of expected ~500)", ones)
	}
}

// TestScrambleTetraGenerateLengthBalance: scrambling a buffer of
// all zeros under a random colour code gives back the raw
// scrambling sequence — useful sanity check that ScrambleTetra
// applies the sequence the same way Generate produces it.
func TestScrambleTetraGenerateLengthBalance(t *testing.T) {
	zeros := make([]byte, 200)
	seq := NewScramblerTetra(0xDEAD).Generate(200)
	scrambled := ScrambleTetra(zeros, 0xDEAD)
	for i, want := range seq {
		if scrambled[i] != want {
			t.Errorf("scramble(zeros)[%d] = %d, want %d (matches Generate)", i, scrambled[i], want)
		}
	}
}

// TestScrambleTetraRandomRoundTrip: 64 random data buffers and
// colour codes round-trip cleanly.
func TestScrambleTetraRandomRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(0xBEEF))
	for trial := 0; trial < 64; trial++ {
		n := 30 + r.Intn(200)
		data := make([]byte, n)
		for i := range data {
			data[i] = byte(r.Intn(2))
		}
		colourCode := r.Uint32() & 0x3FFFFFFF
		scrambled := ScrambleTetra(data, colourCode)
		restored := DescrambleTetra(scrambled, colourCode)
		for i, want := range data {
			if restored[i] != want {
				t.Errorf("trial %d bit %d: got %d, want %d", trial, i, restored[i], want)
				break
			}
		}
	}
}

// refScrambleTetraETSI is an independent, textbook implementation of the
// ETSI EN 300 392-2 §8.2.5 scrambling sequence, built directly from the
// recurrence rather than from an LFSR register-shift — precisely so it can
// not share a shift-direction bug with the production ScramblerTetra.
//
//	p(k) = Σ_{i∈taps} p(k-i)                    (eq. 8.41, mod 2)
//	taps = {1,2,4,5,7,8,10,11,12,16,22,23,26,32}
//	init: p(-31)=p(-30)=1, p(-29)=e(30), …, p(0)=e(1)   (eq. 8.42)
//
// The extended colour code is MCC || MNC || colour, MSB first
// (§8.2.5.3, the packing tetra.ExtendedColourCode produces), so e(1) is
// its most-significant (bit 29) bit and e(30) its least-significant
// (bit 0): e(m) = bit (30-m) of ext. Getting this endianness wrong (e(1)
// as the LSB) still round-trips and still passes BSCH, but does not match
// the sequence a real base station scrambles with, so no BNCH/SCH burst
// decodes.
//
// Output is p(1), p(2), … For BSCH the extended colour code is 0.
func refScrambleTetraETSI(n int, ext uint32) []byte {
	taps := []int{1, 2, 4, 5, 7, 8, 10, 11, 12, 16, 22, 23, 26, 32}
	// p[idx] with idx = k+31, so p(-31) is p[0] and p(1) is p[32].
	p := make([]byte, n+32)
	p[0], p[1] = 1, 1 // p(-31), p(-30)
	for j := 0; j < 30; j++ {
		// p(-29+j) = e(30-j); e(m) = bit (30-m) of ext, so e(30-j) = bit j.
		p[2+j] = byte((ext >> uint(j)) & 1)
	}
	out := make([]byte, n)
	for k := 1; k <= n; k++ {
		idx := k + 31
		var b byte
		for _, i := range taps {
			b ^= p[idx-i]
		}
		p[idx] = b
		out[k-1] = b
	}
	return out
}

// TestScrambleTetraMatchesETSIReference pins the production scrambler to the
// ETSI §8.2.5 sequence via an independent reference. This is the regression
// guard for issue #925: the LFSR previously shifted the wrong way, so its
// sequence diverged from the standard at p(2). Because scramble and
// descramble share the generator, every round-trip test still passed while
// no real (externally scrambled) TETRA burst — the whole BSCH/BNCH/SCH
// family — could be decoded, so the control channel never locked. Colour 0
// is the BSCH case that gates cold-start acquisition; the others guard the
// data-carrying channels.
func TestScrambleTetraMatchesETSIReference(t *testing.T) {
	for _, ext := range []uint32{0, 0x2C, 0x12345, 0x2AAAAAAA, 0x3FFFFFFF} {
		want := refScrambleTetraETSI(200, ext)
		got := NewScramblerTetra(ext).Generate(200)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("colour %#x: scrambler diverges from ETSI §8.2.5 reference at p(%d): got %d, want %d",
					ext, i+1, got[i], want[i])
			}
		}
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
