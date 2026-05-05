package framing

import "testing"

func TestHamming13_9RoundTrip(t *testing.T) {
	for d := uint16(0); d < 1<<9; d++ {
		cw := HammingEncode13_9(d)
		got, errs := HammingDecode13_9(cw)
		if errs != 0 {
			t.Fatalf("clean codeword d=%x reported %d errors", d, errs)
		}
		if got != d {
			t.Fatalf("decode(encode(%x)) = %x", d, got)
		}
	}
}

func TestHamming13_9CorrectsSingleError(t *testing.T) {
	for d := uint16(0); d < 1<<9; d += 7 {
		cw := HammingEncode13_9(d)
		for bit := 0; bit < 13; bit++ {
			corrupted := cw ^ (1 << uint(bit))
			got, errs := HammingDecode13_9(corrupted)
			if errs != 1 || got != d {
				t.Errorf("d=%x bit=%d: got=%x errs=%d", d, bit, got, errs)
			}
		}
	}
}
