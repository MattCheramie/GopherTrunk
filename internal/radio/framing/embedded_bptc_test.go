package framing

import (
	"math/rand"
	"testing"
)

func TestHamming16_11RoundTrip(t *testing.T) {
	for d := uint16(0); d < 1<<11; d++ {
		cw := HammingEncode16_11(d)
		got, errs := HammingDecode16_11(cw)
		if errs != 0 || got != d {
			t.Fatalf("clean data %#x: got %#x errs %d", d, got, errs)
		}
	}
}

func TestHamming16_11CorrectsSingleBit(t *testing.T) {
	for _, d := range []uint16{0, 1, 0x2AA, 0x555, 0x7FF, 0x123} {
		cw := HammingEncode16_11(d)
		for bit := 0; bit < 16; bit++ {
			got, errs := HammingDecode16_11(cw ^ (1 << uint(bit)))
			if got != d || errs != 1 {
				t.Errorf("data %#x bit %d: got %#x errs %d, want %#x errs 1", d, bit, got, errs, d)
			}
		}
	}
}

func TestHamming16_11DetectsDoubleBit(t *testing.T) {
	// A genuine 2-bit error must never be silently mis-decoded to the
	// wrong 11-bit value reported as clean/corrected.
	d := uint16(0x2C5)
	cw := HammingEncode16_11(d)
	for i := 0; i < 15; i++ {
		for j := i + 1; j < 16; j++ {
			got, errs := HammingDecode16_11(cw ^ (1 << uint(i)) ^ (1 << uint(j)))
			if errs >= 0 && got != d {
				t.Errorf("2-bit error (%d,%d): decoded wrong data %#x as errs=%d", i, j, got, errs)
			}
		}
	}
}

func randomLC(r *rand.Rand) []byte {
	lc := make([]byte, EmbLCBits)
	for i := range lc {
		lc[i] = byte(r.Intn(2))
	}
	return lc
}

func TestEmbeddedLCRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for n := 0; n < 200; n++ {
		lc := randomLC(r)
		ch := EncodeEmbeddedLC(lc)
		if len(ch) != EmbChannelBits {
			t.Fatalf("channel length = %d, want %d", len(ch), EmbChannelBits)
		}
		got, corrected := DecodeEmbeddedLC(ch)
		if corrected != 0 {
			t.Fatalf("clean LC: corrected = %d, want 0", corrected)
		}
		for i := range lc {
			if got[i] != lc[i] {
				t.Fatalf("LC bit %d mismatch on clean round trip", i)
			}
		}
	}
}

func TestEmbeddedLCCorrectsSingleBit(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	lc := randomLC(r)
	clean := EncodeEmbeddedLC(lc)
	for bit := 0; bit < EmbChannelBits; bit++ {
		ch := append([]byte(nil), clean...)
		ch[bit] ^= 1
		got, corrected := DecodeEmbeddedLC(ch)
		if corrected < 0 {
			t.Errorf("single bit %d flip: decode failed", bit)
			continue
		}
		for i := range lc {
			if got[i] != lc[i] {
				t.Errorf("single bit %d flip: LC bit %d corrupted", bit, i)
				break
			}
		}
	}
}

func TestEmbeddedLCRejectsHeavyCorruption(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	lc := randomLC(r)
	ch := EncodeEmbeddedLC(lc)
	// Put TWO errors in the same payload row (row 0 = on-air indices
	// c*8+0): columns 0 and 1 → indices 0 and 8. That exceeds the row
	// Hamming(16,11,4) single-error correction, so the SEC-DED row
	// decode flags it uncorrectable and the block is rejected.
	ch[0] ^= 1
	ch[8] ^= 1
	if _, corrected := DecodeEmbeddedLC(ch); corrected >= 0 {
		t.Errorf("two errors in one row accepted (corrected=%d); SEC-DED/CRC gate not effective", corrected)
	}
}
