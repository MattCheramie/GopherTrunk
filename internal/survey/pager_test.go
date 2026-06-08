package survey

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/pocsag"
)

// pocsagIQ builds a clean POCSAG transmission as FM IQ: a 576-bit preamble, a
// sync codeword, then one batch carrying one numeric page for addr18, at
// baud × Oversample × 10 sample rate (an exact resampler ratio) and POCSAG's
// 4500 Hz deviation.
func pocsagIQ(addr18 uint32, fn pocsag.Function, baud uint32) ([]complex64, uint32) {
	rate := baud * oversample * 10
	cws := []uint32{pocsag.SyncCodeword}
	for i := 0; i < pocsag.BatchCodewords; i++ {
		switch i {
		case 0:
			cws = append(cws, pocsag.EncodeAddress(addr18, fn))
		case 1:
			cws = append(cws, pocsag.EncodeMessage(0))
		default:
			cws = append(cws, pocsag.IdleCodeword)
		}
	}
	bits := make([]byte, 0, 576+len(cws)*32)
	for i := 0; i < 576; i++ {
		bits = append(bits, byte((i+1)%2)) // 1010… preamble
	}
	for _, cw := range cws {
		for b := 31; b >= 0; b-- {
			bits = append(bits, byte((cw>>uint(b))&1))
		}
	}
	sps := int(rate / baud)
	msg := make([]float64, 0, len(bits)*sps)
	for _, b := range bits {
		v := -1.0
		if b == 1 {
			v = 1.0
		}
		for j := 0; j < sps; j++ {
			msg = append(msg, v)
		}
	}
	return fmModulate(msg, float64(rate), 4500), rate
}

// TestDecodePOCSAGEndToEnd builds a clean POCSAG transmission as IQ and asserts
// the buffer-mode adapter recovers the page (RIC + encoding), exercising the
// full FM→resample→slice→syncer→batch→message chain.
func TestDecodePOCSAGEndToEnd(t *testing.T) {
	const addr18 uint32 = 0x12345
	iq, rate := pocsagIQ(addr18, 1 /* numeric */, 1200)
	pages := DecodePOCSAG(iq, rate)
	if len(pages) == 0 {
		t.Fatalf("no pages decoded")
	}
	wantRIC := addr18 << 3 // address occupies bits 3..20; function is separate
	found := false
	for _, p := range pages {
		if p.Capcode == wantRIC {
			found = true
			if p.Encoding != "numeric" {
				t.Errorf("encoding = %q, want numeric", p.Encoding)
			}
			if p.Protocol != "pocsag" || p.BaudHz != 1200 {
				t.Errorf("got protocol=%q baud=%d, want pocsag 1200", p.Protocol, p.BaudHz)
			}
		}
	}
	if !found {
		t.Errorf("RIC 0x%x not among decoded pages %+v", wantRIC, pages)
	}
}

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
