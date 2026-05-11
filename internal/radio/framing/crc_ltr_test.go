package framing

import (
	"math/rand"
	"testing"
)

func TestCRC7LTRZeroMessage(t *testing.T) {
	msg := make([]byte, 24)
	if got := CRC7LTR(msg); got != 0 {
		t.Errorf("CRC of all-zero message = %#x, want 0x00", got)
	}
}

func TestCRC7LTRSingleBitMatchesTable(t *testing.T) {
	for i := 0; i < 24; i++ {
		msg := make([]byte, 24)
		msg[i] = 1
		got := CRC7LTR(msg)
		want := crc7LTRChecksums[i] & 0x7F
		if got != want {
			t.Errorf("CRC of bit %d set = %#x, want %#x", i, got, want)
		}
	}
}

func TestCRC7LTRMultiBitXORsTable(t *testing.T) {
	msg := make([]byte, 24)
	msg[0], msg[1], msg[11] = 1, 1, 1
	want := (crc7LTRChecksums[0] ^ crc7LTRChecksums[1] ^ crc7LTRChecksums[11]) & 0x7F
	if got := CRC7LTR(msg); got != want {
		t.Errorf("CRC = %#x, want %#x", got, want)
	}
}

func TestCRC7LTRRoundTripOSW(t *testing.T) {
	r := rand.New(rand.NewSource(0xC0DE))
	for trial := 0; trial < 256; trial++ {
		msg := make([]byte, 24)
		for i := range msg {
			msg[i] = byte(r.Intn(2))
		}
		crc := CRC7LTR(msg)
		if !VerifyCRC7LTR(msg, crc, false) {
			t.Errorf("trial %d: OSW round-trip failed: msg=%v crc=%#x", trial, msg, crc)
		}
	}
}

func TestCRC7LTRDetectsSingleBitErrorInMessage(t *testing.T) {
	msg := make([]byte, 24)
	for i := 0; i < 24; i += 2 {
		msg[i] = 1
	}
	crc := CRC7LTR(msg)
	for pos := 0; pos < 24; pos++ {
		corrupted := append([]byte{}, msg...)
		corrupted[pos] ^= 1
		if VerifyCRC7LTR(corrupted, crc, false) {
			t.Errorf("bit %d flip went undetected", pos)
		}
	}
}

func TestVerifyCRC7LTRInboundInvertsChecksum(t *testing.T) {
	msg := make([]byte, 24)
	for i := 0; i < 24; i += 3 {
		msg[i] = 1
	}
	crc := CRC7LTR(msg)
	inv := crc ^ 0x7F
	// OSW path: matches the as-is checksum.
	if !VerifyCRC7LTR(msg, crc, false) {
		t.Errorf("OSW: VerifyCRC7LTR(_, crc, false) = false, want true")
	}
	if VerifyCRC7LTR(msg, inv, false) {
		t.Errorf("OSW: VerifyCRC7LTR(_, ~crc, false) = true, want false")
	}
	// ISW path: matches the bit-inverted checksum.
	if !VerifyCRC7LTR(msg, inv, true) {
		t.Errorf("ISW: VerifyCRC7LTR(_, ~crc, true) = false, want true")
	}
	if VerifyCRC7LTR(msg, crc, true) {
		t.Errorf("ISW: VerifyCRC7LTR(_, crc, true) = true, want false")
	}
}

func TestCRC7LTRChecksumsAreUnique(t *testing.T) {
	// Each table entry must be distinct (single-bit-error
	// detection requires unique syndromes per bit position).
	seen := map[byte]int{}
	for i, c := range crc7LTRChecksums {
		if prev, dup := seen[c]; dup {
			t.Errorf("duplicate checksum %#x at positions %d and %d", c, prev, i)
		}
		seen[c] = i
	}
}

func TestCRC7LTRChecksumsAreSevenBit(t *testing.T) {
	for i, c := range crc7LTRChecksums {
		if c&0x80 != 0 {
			t.Errorf("checksum at index %d = %#x has high bit set", i, c)
		}
	}
}
