package survey

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// TestDemodBitsRoundTrip checks the buffer-mode FM→resample→slice adapter
// recovers a known bit pattern from a GFSK-modulated carrier. It validates the
// slicer that feeds the POCSAG/FLEX syncers without needing full page framing
// (end-to-end page decode is covered against real fixtures).
func TestDemodBitsRoundTrip(t *testing.T) {
	const baud = 1200
	const sps = testRateHz / baud // 40 samples/symbol at 48 kHz

	want := randBits(2000, 42)
	iq := demod.ModulateGFSK(want, int(sps), 4, 0.5, testRateHz, baud/2)

	got := demodBits(iq, testRateHz, baud)
	if len(got) < len(want)/2 {
		t.Fatalf("recovered only %d bits from %d symbols", len(got), len(want))
	}

	// The slicer has filter/group delay and unknown polarity; find the best
	// alignment (offset + inversion) and require a high match there.
	best := bestBitMatch(want, got)
	if best < 0.9 {
		t.Errorf("best bit match = %.1f%%, want ≥ 90%%", best*100)
	}
}

// bestBitMatch returns the highest fractional agreement between want and got
// over a small offset search, considering both polarities.
func bestBitMatch(want, got []byte) float64 {
	best := 0.0
	for _, invert := range []bool{false, true} {
		for off := 0; off < 80 && off < len(got); off++ {
			match, n := 0, 0
			for i := 0; i+off < len(got) && i < len(want); i++ {
				b := got[i+off]
				if invert {
					b ^= 1
				}
				if b == want[i] {
					match++
				}
				n++
			}
			if n > 200 {
				if f := float64(match) / float64(n); f > best {
					best = f
				}
			}
		}
	}
	return best
}
