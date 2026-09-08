package fleetsync

import (
	"math/bits"
	"testing"
)

// ---- FleetSync II ECC ------------------------------------------------

// TestFS2ECCRepairPassesValidCodeword: a code word whose syndrome is zero
// is returned unchanged. The all-zero word is the trivial valid codeword.
func TestFS2ECCRepairPassesValidCodeword(t *testing.T) {
	got, ok := fs2ECCRepair(0x0000)
	if !ok || got != 0x0000 {
		t.Fatalf("fs2ECCRepair(0) = (0x%04x, %v), want (0x0000, true)", got, ok)
	}
}

// TestFS2ECCRepairCorrectsEverySingleBit is the reference-literal check of
// the parity-check / repair tables: for a valid codeword, flipping ANY
// single one of its 15 code bits must be corrected back to the original.
// Applied to the all-zero codeword, every single-bit error 1<<i must
// repair to 0. If either table were mistranscribed this fails, so it pins
// the FleetSync II ECC against the multimon-ng reference.
func TestFS2ECCRepairCorrectsEverySingleBit(t *testing.T) {
	for i := 0; i < 15; i++ {
		corrupted := uint16(1) << uint(i)
		got, ok := fs2ECCRepair(corrupted)
		if !ok {
			t.Errorf("bit %d: fs2ECCRepair(0x%04x) reported uncorrectable; a single-bit error must be correctable", i, corrupted)
			continue
		}
		if got != 0 {
			t.Errorf("bit %d: fs2ECCRepair(0x%04x) = 0x%04x, want 0x0000 (restore the valid codeword)", i, corrupted, got)
		}
	}
}

// TestFS2ECCRepairRejectsUncorrectable: a two-bit error that lands on no
// single-bit syndrome must be rejected rather than silently "corrected"
// to a wrong codeword.
func TestFS2ECCRepairRejectsUncorrectable(t *testing.T) {
	// Two bits flipped in the zero codeword. Its syndrome is the XOR of
	// the two single-bit syndromes; find a pair whose combined syndrome is
	// not itself a single-bit repair position.
	found := false
	for a := 0; a < 15 && !found; a++ {
		for b := a + 1; b < 15; b++ {
			corrupted := uint16(1)<<uint(a) | uint16(1)<<uint(b)
			if _, ok := fs2ECCRepair(corrupted); !ok {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("expected at least one two-bit pattern to be flagged uncorrectable")
	}
}

// ---- FleetSync I CRC + ANI extraction --------------------------------

// buildFS1Frame lays out a valid FleetSync I capture: a 4-bit lead-in,
// word1, then word2 whose low 16 bits are the block check computed over
// word1 and word2's high 16 bits. The check is found by search (it is a
// one-bit fixed point because word2's low bits feed only the parity bit),
// which keeps the builder honest: it does not assume the CRC value, it
// solves fsyncCRC(word1, word2) == word2&0xFFFF the way a real transmitter
// must. Returns the 260-bit capture and the assembled word2.
func buildFS1Frame(t *testing.T, word1 uint32, dataHi16 uint16) ([]byte, uint32) {
	t.Helper()
	for crc := 0; crc <= 0xFFFF; crc++ {
		word2 := uint32(dataHi16)<<16 | uint32(crc)
		if fsyncCRC(word1, word2) == uint16(crc) {
			return layoutFrame(word1, word2), word2
		}
	}
	t.Fatalf("no valid CRC found for word1=0x%08x dataHi16=0x%04x", word1, dataHi16)
	return nil, 0
}

// layoutFrame renders a 260-bit capture: 4 zero lead-in bits, then word1
// and word2 MSB-first, then zero padding to FrameBits.
func layoutFrame(word1, word2 uint32) []byte {
	raw := make([]byte, FrameBits)
	appendUint32(raw[frameOffset:], word1)
	appendUint32(raw[frameOffset+32:], word2)
	return raw
}

// appendUint32 writes v MSB-first as one byte per bit into dst[:32].
func appendUint32(dst []byte, v uint32) {
	for i := 0; i < 32; i++ {
		dst[i] = byte((v >> (31 - i)) & 1)
	}
}

func TestDecodeFrameFS1ExtractsANI(t *testing.T) {
	// fleet = byte2 + 99; unit = (byte3<<4) + (byte4_hi_nibble) + 999.
	// byte2 = 51 → fleet 150; byte3 = 0x7D, byte4 hi nibble = 0x2
	//   → unit = (0x7D<<4) + 0x2 + 999 = 2000 + 2 + 999 = 3001.
	const (
		byte0, byte1 = 0x00, 0x00
		byte2        = 51   // fleet 150
		byte3        = 0x7D // unit high byte
		byte4        = 0x2F // high nibble 0x2 is the unit low nibble; low nibble ignored
		byte5        = 0x00
	)
	word1 := uint32(byte0)<<24 | uint32(byte1)<<16 | uint32(byte2)<<8 | uint32(byte3)
	dataHi16 := uint16(byte4)<<8 | uint16(byte5)

	raw, _ := buildFS1Frame(t, word1, dataHi16)
	msg, ok := DecodeFrame(raw)
	if !ok {
		t.Fatalf("DecodeFrame reported CRC failure on a frame built with a valid check")
	}
	if msg.CRCOK != true || msg.IsFS2 {
		t.Errorf("msg = %+v, want CRCOK=true IsFS2=false", msg)
	}
	if msg.Fleet != 150 {
		t.Errorf("Fleet = %d, want 150", msg.Fleet)
	}
	if msg.Unit != 3001 {
		t.Errorf("Unit = %d, want 3001", msg.Unit)
	}
}

// TestDecodeFrameFS1RejectsCorruptedCheck: flipping a payload bit must
// break the block check so DecodeFrame reports CRCOK=false rather than a
// bogus ANI passing as valid.
func TestDecodeFrameFS1RejectsCorruptedCheck(t *testing.T) {
	word1 := uint32(0x0033_7D00)
	raw, _ := buildFS1Frame(t, word1, 0x2F00)

	// Flip a bit inside word1 (payload) — the check no longer matches.
	raw[frameOffset+10] ^= 1
	if _, ok := DecodeFrame(raw); ok {
		t.Error("DecodeFrame accepted a frame with a corrupted payload bit; the CRC must reject it")
	}
}

// TestDecodeFrameShortIsIncomplete: a capture shorter than a FleetSync I
// frame is reported incomplete, not decoded from garbage.
func TestDecodeFrameShortIsIncomplete(t *testing.T) {
	if _, ok := DecodeFrame(make([]byte, fs1FrameBits-1)); ok {
		t.Error("DecodeFrame accepted a too-short capture")
	}
}

// ---- FleetSync II round-trip -----------------------------------------

// fs2EncodeHalf finds a valid 15-bit FleetSync II code word (syndrome 0)
// whose data nibble (bits 11..8) equals nibble. Every nibble has several
// codewords; returns the first.
func fs2EncodeHalf(t *testing.T, nibble uint32) uint16 {
	t.Helper()
	for v := uint16(0); v < 0x8000; v++ {
		if (uint32(v)>>8)&0xF != nibble {
			continue
		}
		if syndromeZero(v) {
			return v
		}
	}
	t.Fatalf("no valid FS2 codeword carries nibble 0x%x", nibble)
	return 0
}

func syndromeZero(v uint16) bool {
	for i := 0; i < 8; i++ {
		if bits.OnesCount16(fs2ParityCheck[i]&v)&1 == 1 {
			return false
		}
	}
	return true
}

// buildFS2Frame lays out a FleetSync II capture that ECC-decodes to the
// given word1/word2. Each of the two words contributes 8 high-nibbles
// spread across four 64-bit blocks; each nibble becomes a valid 15-bit
// code word in one 16-bit half-word.
func buildFS2Frame(t *testing.T, word1, word2 uint32) []byte {
	t.Helper()
	// nibbles(word) yields the 8 high-nibbles MSB-first.
	nibbles := func(w uint32) [8]uint32 {
		var n [8]uint32
		for i := 0; i < 8; i++ {
			n[i] = (w >> uint(28-4*i)) & 0xF
		}
		return n
	}
	n1 := nibbles(word1)
	n2 := nibbles(word2)

	raw := make([]byte, FrameBits)
	// Blocks 0,1 carry word1's 8 nibbles; blocks 2,3 carry word2's.
	// Per block, four half-words in the order (bw1 lo, bw1 hi, bw2 lo, bw2 hi)
	// — matching decodeFS2's extraction order.
	writeBlock := func(blk int, nibs [4]uint32) {
		boff := frameOffset + blk*64
		h0 := uint32(fs2EncodeHalf(t, nibs[0]))
		h1 := uint32(fs2EncodeHalf(t, nibs[1]))
		h2 := uint32(fs2EncodeHalf(t, nibs[2]))
		h3 := uint32(fs2EncodeHalf(t, nibs[3]))
		bw1 := (h1 << 16) | h0
		bw2 := (h3 << 16) | h2
		appendUint32(raw[boff:], bw1)
		appendUint32(raw[boff+32:], bw2)
	}
	writeBlock(0, [4]uint32{n1[0], n1[1], n1[2], n1[3]})
	writeBlock(1, [4]uint32{n1[4], n1[5], n1[6], n1[7]})
	writeBlock(2, [4]uint32{n2[0], n2[1], n2[2], n2[3]})
	writeBlock(3, [4]uint32{n2[4], n2[5], n2[6], n2[7]})
	return raw
}

func TestDecodeFrameFS2RoundTrip(t *testing.T) {
	// A CRC-valid word pair, then re-encoded through the FS2 ECC blocks.
	word1 := uint32(0x0000_337D) // fleet 150 (byte2=0x33=51), unit high 0x7D
	var word2 uint32
	{
		_, w2 := buildFS1Frame(t, word1, 0x2F00) // solves the CRC for us
		word2 = w2
	}
	raw := buildFS2Frame(t, word1, word2)

	msg, ok := DecodeFrame(raw)
	if !ok {
		t.Fatalf("DecodeFrame(FS2) reported failure on a valid FleetSync II frame")
	}
	if !msg.IsFS2 {
		t.Errorf("IsFS2 = false, want true (FS2 path must be taken when FS1 CRC fails)")
	}
	if msg.Fleet != 150 || msg.Unit != 3001 {
		t.Errorf("ANI = fleet %d unit %d, want fleet 150 unit 3001", msg.Fleet, msg.Unit)
	}
}

// TestDecodeFrameFS2CorrectsSingleBit: a single sliced-bit error in one
// FS2 half-word must be corrected by the ECC and still yield the right ANI
// — the whole reason FleetSync II carries the code.
func TestDecodeFrameFS2CorrectsSingleBit(t *testing.T) {
	word1 := uint32(0x0000_337D)
	_, word2 := buildFS1Frame(t, word1, 0x2F00)
	raw := buildFS2Frame(t, word1, word2)

	// Corrupt one bit inside block 0's first half-word (raw[frameOffset..]).
	raw[frameOffset+20] ^= 1

	msg, ok := DecodeFrame(raw)
	if !ok {
		t.Fatalf("DecodeFrame(FS2) failed to correct a single-bit error")
	}
	if !msg.IsFS2 || msg.Fleet != 150 || msg.Unit != 3001 {
		t.Errorf("after single-bit correction: msg = %+v, want FS2 fleet 150 unit 3001", msg)
	}
}
