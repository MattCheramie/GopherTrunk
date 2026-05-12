package nxdn

import (
	"math/rand"
	"testing"
)

// fillBitsCAC populates a 0/1 bit slice from a deterministic PRNG
// so each round-trip test exercises non-trivial content.
func fillBitsCAC(n int, seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(r.Intn(2))
	}
	return out
}

func TestEncodeDecodeCACChannelRoundTrip(t *testing.T) {
	for _, seed := range []int64{1, 2, 42, 0xC0FFEE} {
		info := fillBitsCAC(CACInfoBits, seed)
		channel := EncodeCACChannel(info)
		if len(channel) != CACChannelBits {
			t.Fatalf("seed %d: encoded length = %d, want %d", seed, len(channel), CACChannelBits)
		}
		recovered, ok := DecodeCACChannel(channel)
		if !ok {
			t.Fatalf("seed %d: CRC fail on clean round-trip", seed)
		}
		if len(recovered) != CACInfoBits {
			t.Fatalf("seed %d: recovered length = %d, want %d", seed, len(recovered), CACInfoBits)
		}
		for i := range info {
			if info[i] != recovered[i] {
				t.Errorf("seed %d: bit %d differs: got %d, want %d", seed, i, recovered[i], info[i])
				break
			}
		}
	}
}

// TestDecodeCACChannelCorrectsSingleBitError: a single-bit flip in
// the on-air channel stream should be absorbed by the K=5 Viterbi
// without breaking the CRC.
func TestDecodeCACChannelCorrectsSingleBitError(t *testing.T) {
	info := fillBitsCAC(CACInfoBits, 7)
	channel := EncodeCACChannel(info)
	corrupted := append([]byte{}, channel...)
	corrupted[123] ^= 1
	recovered, ok := DecodeCACChannel(corrupted)
	if !ok {
		t.Fatalf("DecodeCACChannel: CRC fail after single-bit correction; want pass")
	}
	for i := range info {
		if info[i] != recovered[i] {
			t.Errorf("bit %d: got %d, want %d (single-bit correction)", i, recovered[i], info[i])
			break
		}
	}
}

// TestDecodeCACChannelDetectsHeavyCorruption: flip enough bits in
// the on-air stream to defeat the inner Viterbi (with the
// 25-position interleaver, adjacent bursts of corruption spread
// across many independent decoder steps, but the CRC still has to
// fire). 60 adjacent bit flips guarantees the CRC observes
// corruption.
func TestDecodeCACChannelDetectsHeavyCorruption(t *testing.T) {
	info := fillBitsCAC(CACInfoBits, 11)
	channel := EncodeCACChannel(info)
	corrupted := append([]byte{}, channel...)
	for i := 100; i < 160; i++ {
		corrupted[i] ^= 1
	}
	if _, ok := DecodeCACChannel(corrupted); ok {
		t.Errorf("DecodeCACChannel reported CRC pass on heavily-corrupted stream")
	}
}

// TestEncodeCACChannelRejectsWrongSize: Encode returns nil on
// non-CACInfoBits input so callers can spot misuse early.
func TestEncodeCACChannelRejectsWrongSize(t *testing.T) {
	if got := EncodeCACChannel(make([]byte, CACInfoBits-1)); got != nil {
		t.Errorf("EncodeCACChannel accepted %d-bit input, want nil", CACInfoBits-1)
	}
	if got := EncodeCACChannel(make([]byte, CACInfoBits+1)); got != nil {
		t.Errorf("EncodeCACChannel accepted %d-bit input, want nil", CACInfoBits+1)
	}
}

// TestDecodeCACChannelRejectsWrongSize: Decode returns false on
// non-CACChannelBits input.
func TestDecodeCACChannelRejectsWrongSize(t *testing.T) {
	if _, ok := DecodeCACChannel(make([]byte, CACChannelBits-1)); ok {
		t.Errorf("DecodeCACChannel accepted %d-bit input, want false", CACChannelBits-1)
	}
	if _, ok := DecodeCACChannel(make([]byte, CACChannelBits+1)); ok {
		t.Errorf("DecodeCACChannel accepted %d-bit input, want false", CACChannelBits+1)
	}
}

// TestCACInterleavePermIsBijection: the 25×12 column-major
// readout must cover every position 0..299 exactly once.
func TestCACInterleavePermIsBijection(t *testing.T) {
	var seen [CACChannelBits]bool
	for k := 0; k < CACChannelBits; k++ {
		j := cacInterleavePerm[k]
		if j < 0 || j >= CACChannelBits {
			t.Fatalf("perm[%d] = %d out of range", k, j)
		}
		if seen[j] {
			t.Fatalf("perm[%d] = %d already seen — not a bijection", k, j)
		}
		seen[j] = true
	}
}

// TestCACPuncturePositionsMatchMatrix: the computed positions must
// match the spec's 1111111/1011101 matrix (G2 dropped at i mod 7
// ∈ {1, 5}) for the full 175-bit encoder run.
func TestCACPuncturePositionsMatchMatrix(t *testing.T) {
	if len(cacPuncturePositions) != 50 {
		t.Fatalf("len(cacPuncturePositions) = %d, want 50", len(cacPuncturePositions))
	}
	for k, pos := range cacPuncturePositions {
		// Every position is an odd index (G2 bit, 2i+1).
		if pos%2 != 1 {
			t.Errorf("cacPuncturePositions[%d] = %d, not a G2 bit", k, pos)
		}
		i := (pos - 1) / 2
		if i%7 != 1 && i%7 != 5 {
			t.Errorf("cacPuncturePositions[%d] = %d (encoder step %d, mod 7 = %d), want mod 7 ∈ {1, 5}",
				k, pos, i, i%7)
		}
	}
	// Sorted ascending, no duplicates.
	for k := 1; k < len(cacPuncturePositions); k++ {
		if cacPuncturePositions[k] <= cacPuncturePositions[k-1] {
			t.Errorf("cacPuncturePositions not strictly ascending at index %d (%d <= %d)",
				k, cacPuncturePositions[k], cacPuncturePositions[k-1])
		}
	}
}

// TestCACCRC16SanityAgainstByteWiseCRC: when the input bits pack
// into whole bytes, cacCRC16 must produce the same value as the
// existing framing.CRCCCITT byte-level helper. Lets us anchor
// cacCRC16 against a well-tested implementation.
func TestCACCRC16SanityAgainstByteWiseCRC(t *testing.T) {
	// 88-bit input — same shape as the existing AssembleCAC.
	bytes := []byte{0x3C, 0xAA, 0xAA, 0x12, 0x34, 0x56, 0x78, 0x00, 0x00, 0x00, 0x00}
	bits := make([]byte, 0, 88)
	for _, b := range bytes[:9] { // 9 bytes = 72 bits, plus 2 more = 88
		for i := 7; i >= 0; i-- {
			bits = append(bits, (b>>uint(i))&1)
		}
	}
	for _, b := range bytes[9:11] {
		for i := 7; i >= 0; i-- {
			bits = append(bits, (b>>uint(i))&1)
		}
	}
	got := cacCRC16(bits)
	// framing.CRCCCITT over the same 11 bytes should match exactly
	// when the inputs align on byte boundaries.
	want := framingCRCCCITTBytes(bytes)
	if got != want {
		t.Errorf("cacCRC16 = %#04x, framing.CRCCCITT over same bytes = %#04x", got, want)
	}
}

// framingCRCCCITTBytes is a local inline copy of the byte-level
// CRC-CCITT/FALSE pattern so the test doesn't need to import the
// framing package — keeps the dependency direction stable.
func framingCRCCCITTBytes(msg []byte) uint16 {
	const poly uint16 = 0x1021
	crc := uint16(0xFFFF)
	for _, b := range msg {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
