package framing

import (
	"math/bits"
	"testing"
)

// reverseBits reverses the low n bits of x.
func reverseBits(x uint32, n int) uint32 {
	return bits.Reverse32(x) >> uint(32-n)
}

// TestFLEXBCHLayout pins the codeword layout to a vector worked by hand from
// the generator: information bit 20 is the coefficient of x^10, and
// x^10 mod g(x) = x^9+x^8+x^6+x^5+x^3+1, whose coefficient of x^k sits in
// bit 30-k — bits 21, 22, 24, 25, 27 and 30. Seven ones, so the even-parity
// bit (31) is set. (On-air words pin the same layout from the other side:
// flex.TestRealAirWordsAreCodewords.)
func TestFLEXBCHLayout(t *testing.T) {
	const want = 1<<20 | 1<<21 | 1<<22 | 1<<24 | 1<<25 | 1<<27 | 1<<30 | 1<<31
	if got := FLEXBCHEncode21(1 << 20); got != want {
		t.Fatalf("FLEXBCHEncode21(1<<20) = %08x, want %08x", got, uint32(want))
	}
}

// TestFLEXBCHMatchesPOCSAGCode: FLEX and POCSAG use one BCH(31,21) code with
// the bit order mirrored — information AND codeword. The POCSAG encoder is an
// independent implementation of the same generator.
func TestFLEXBCHMatchesPOCSAGCode(t *testing.T) {
	for _, info := range []uint32{0, 1, 0x1FFFFF, 0x12345, 0x0A5A5A, 0x155555, 0x100000, 0x000800} {
		want := reverseBits(BCHEncode31_21(reverseBits(info, 21)), 31)
		if got := FLEXBCHEncode21(info) & 0x7FFFFFFF; got != want {
			t.Errorf("info %06x: codeword %08x, POCSAG-mirrored %08x", info, got, want)
		}
	}
}

// TestFLEXBCHCorrectsTwoErrors: every single and double bit error in a word
// is corrected and counted; three are either flagged or (the code's limit)
// decoded to another word, never silently "corrected" back.
func TestFLEXBCHCorrectsTwoErrors(t *testing.T) {
	for _, info := range []uint32{0, 0x1FFFFF, 0x0F0F0F, 0x13579B} {
		cw := FLEXBCHEncode21(info)
		if cw&0x1FFFFF != info || bits.OnesCount32(cw)&1 != 0 {
			t.Fatalf("info %06x: codeword %08x is not systematic with even parity", info, cw)
		}
		if got, errs := FLEXBCHDecode32(cw); got != info || errs != 0 {
			t.Fatalf("clean word: got %06x errs %d", got, errs)
		}
		for a := 0; a < 31; a++ {
			if got, errs := FLEXBCHDecode32(cw ^ 1<<uint(a)); got != info || errs != 1 {
				t.Fatalf("bit %d flipped: got %06x errs %d", a, got, errs)
			}
			for b := a + 1; b < 31; b++ {
				e := uint32(1)<<uint(a) | 1<<uint(b)
				if got, errs := FLEXBCHDecode32(cw ^ e); got != info || errs != 2 {
					t.Fatalf("bits %d,%d flipped: got %06x errs %d", a, b, got, errs)
				}
				if c := b + 1; c < 31 {
					if got, errs := FLEXBCHDecode32(cw ^ e ^ 1<<uint(c)); errs >= 0 && got == info {
						t.Fatalf("bits %d,%d,%d flipped: decoded back to the sent word", a, b, c)
					}
				}
			}
		}
		// The parity bit is not part of the BCH word.
		if got, errs := FLEXBCHDecode32(cw ^ 1<<31); got != info || errs != 0 {
			t.Fatalf("parity bit flipped: got %06x errs %d", got, errs)
		}
	}
}

func BenchmarkFLEXBCHDecode32(b *testing.B) {
	cw := FLEXBCHEncode21(0x12345) ^ 0x00040010
	for i := 0; i < b.N; i++ {
		FLEXBCHDecode32(cw)
	}
}
