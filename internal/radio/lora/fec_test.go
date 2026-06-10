package lora

import (
	"math/rand"
	"testing"
)

func TestGrayRoundTrip(t *testing.T) {
	for v := 0; v < 4096; v++ {
		if got := grayDecode(grayEncode(uint16(v))); got != uint16(v) {
			t.Fatalf("gray round-trip v=%d got=%d", v, got)
		}
	}
}

func TestHammingRoundTrip(t *testing.T) {
	for cr := 1; cr <= 4; cr++ {
		for nib := 0; nib < 16; nib++ {
			cw := HammingEncode4(uint8(nib), cr)
			got, corr := HammingDecode4(cw, cr)
			if got != uint8(nib) || corr != 0 {
				t.Fatalf("cr=%d nib=%d -> cw=%05b decoded=%d corr=%d", cr, nib, cw, got, corr)
			}
		}
	}
}

func TestHammingSingleErrorCorrection(t *testing.T) {
	// CR 4/7 (cr=3) and 4/8 (cr=4) must correct any single bit error.
	for _, cr := range []int{3, 4} {
		width := 4 + cr
		for nib := 0; nib < 16; nib++ {
			cw := HammingEncode4(uint8(nib), cr)
			for bit := 0; bit < width; bit++ {
				corrupt := cw ^ (1 << uint(bit))
				got, corr := HammingDecode4(corrupt, cr)
				if got != uint8(nib) || corr != 1 {
					t.Fatalf("cr=%d nib=%d bit=%d: decoded=%d corr=%d (want %d,1)",
						cr, nib, bit, got, corr, nib)
				}
			}
		}
	}
}

func TestInterleaverRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, sf := range []int{7, 9, 12} {
		for cr := 1; cr <= 4; cr++ {
			ppm := sf
			cw := 4 + cr
			codewords := make([]uint16, ppm)
			for i := range codewords {
				codewords[i] = uint16(rng.Intn(1 << uint(cw)))
			}
			syms := Interleave(codewords, ppm, cr)
			if len(syms) != cw {
				t.Fatalf("sf=%d cr=%d: got %d syms want %d", sf, cr, len(syms), cw)
			}
			back := Deinterleave(syms, ppm, cr)
			for i := range codewords {
				if back[i] != codewords[i] {
					t.Fatalf("sf=%d cr=%d codeword %d: got %d want %d", sf, cr, i, back[i], codewords[i])
				}
			}
		}
	}
}

func TestWhitenSelfInverse(t *testing.T) {
	orig := []byte("GopherTrunk LoRa whitening self-inverse check 0123456789")
	buf := make([]byte, len(orig))
	copy(buf, orig)
	Whiten(buf)
	if string(buf) == string(orig) {
		t.Fatal("whitening did not change the data")
	}
	Whiten(buf)
	if string(buf) != string(orig) {
		t.Fatalf("whitening not self-inverse: got %q", buf)
	}
}

func TestFrameSymbolRoundTrip(t *testing.T) {
	// Pure symbol-domain round trip (no DSP): encode -> decode.
	payloads := [][]byte{
		{0x01},
		{0xDE, 0xAD, 0xBE, 0xEF},
		[]byte("hello lora"),
	}
	for _, sf := range []int{7, 8, 10, 12} {
		for cr := 1; cr <= 4; cr++ {
			for _, pl := range payloads {
				syms := encodeFrameSymbols(pl, sf, cr, true)
				f, ok := decodeFrameFromSymbols(syms, sf)
				if !ok {
					t.Fatalf("sf=%d cr=%d len=%d: decode failed", sf, cr, len(pl))
				}
				if !f.HeaderOK || !f.CRCOK {
					t.Fatalf("sf=%d cr=%d: headerOK=%v crcOK=%v", sf, cr, f.HeaderOK, f.CRCOK)
				}
				if f.CR != cr {
					t.Fatalf("sf=%d: decoded CR=%d want %d", sf, f.CR, cr)
				}
				if string(f.Payload) != string(pl) {
					t.Fatalf("sf=%d cr=%d: payload %x want %x", sf, cr, f.Payload, pl)
				}
			}
		}
	}
}
