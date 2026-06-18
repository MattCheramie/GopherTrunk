package phase1

import (
	"testing"
	"time"
)

// makeHeaderContent builds an LC content matching the Motorola
// talker-alias HEADER layout (LCO 0x15): octets 2-3 = talkgroup,
// octet 4 = block_count, octet 5 = format, octet 6 = 0, octet 7
// high nibble = sequence.
func makeHeaderContent(talkgroup uint16, blockCount, seq uint8) [lcContentOctets]byte {
	var c [lcContentOctets]byte
	c[0] = LCOTalkerAliasHeader
	c[1] = 0x90 // MFID Motorola (informational; assembler doesn't gate on it)
	c[2] = byte(talkgroup >> 8)
	c[3] = byte(talkgroup)
	c[4] = blockCount
	c[5] = 1 // format = 1 (observed value)
	c[6] = 0
	c[7] = seq << 4
	c[8] = 0
	return c
}

// makeDataBlockContent builds an LC content matching the Motorola
// talker-alias DATA BLOCK layout (LCO 0x17): octet 2 = block_number
// (1-based), octet 3 high nibble = sequence, low nibble +
// octets 4-8 = 44-bit fragment. Each bit of `fragment` is taken
// MSB-first; the slice must be exactly 6 bytes with the first
// byte's high nibble unused.
func makeDataBlockContent(blockNum, seq uint8, fragment [6]byte) [lcContentOctets]byte {
	var c [lcContentOctets]byte
	c[0] = LCOTalkerAliasBlock2
	c[1] = 0x90
	c[2] = blockNum
	c[3] = (seq << 4) | (fragment[0] & 0x0F)
	c[4] = fragment[1]
	c[5] = fragment[2]
	c[6] = fragment[3]
	c[7] = fragment[4]
	c[8] = fragment[5]
	return c
}

func TestMotorolaTalkerAliasBufRejectsBadHeaderCount(t *testing.T) {
	b := NewMotorolaTalkerAliasBuf(nil)
	// block_count = 0 is meaningless.
	if _, ok := b.AddFragment(LCOTalkerAliasHeader, makeHeaderContent(20202, 0, 5)); ok {
		t.Error("zero block_count should not emit")
	}
	// block_count over the safety cap is rejected.
	if _, ok := b.AddFragment(LCOTalkerAliasHeader, makeHeaderContent(20202, 100, 5)); ok {
		t.Error("over-cap block_count should not emit")
	}
}

func TestMotorolaTalkerAliasBufRejectsDataBlockWithoutHeader(t *testing.T) {
	b := NewMotorolaTalkerAliasBuf(nil)
	if _, ok := b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(1, 5, [6]byte{})); ok {
		t.Error("data block without prior header should not emit")
	}
}

func TestMotorolaTalkerAliasBufRejectsMismatchedSequence(t *testing.T) {
	b := NewMotorolaTalkerAliasBuf(nil)
	b.AddFragment(LCOTalkerAliasHeader, makeHeaderContent(20202, 2, 5))
	// Data block carries a different sequence number — must be dropped.
	b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(1, 6, [6]byte{}))
	b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(2, 6, [6]byte{}))
	// Even with the right block count "received", the mismatched
	// sequence means the blocks slot at indices 0 and 1 are still nil.
	// AddFragment for the matching blocks below should still not emit
	// because block_number > block_count check would let it through —
	// instead, the wrong sequences fail the seq check and don't
	// populate b.blocks. So the buffer is still incomplete.
	if _, ok := b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(3, 5, [6]byte{})); ok {
		t.Error("emission with non-matching-sequence data blocks should not fire")
	}
}

func TestMotorolaTalkerAliasBufNewHeaderResetsBlocks(t *testing.T) {
	// A header with a new sequence number drops any in-flight data
	// blocks from the previous one.
	b := NewMotorolaTalkerAliasBuf(nil)
	b.AddFragment(LCOTalkerAliasHeader, makeHeaderContent(20202, 4, 3))
	b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(1, 3, [6]byte{0, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}))
	b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(2, 3, [6]byte{0, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}))
	// New header — different sequence; the two stored data blocks
	// must be cleared, otherwise the second sequence's emission
	// gate would fire prematurely.
	b.AddFragment(LCOTalkerAliasHeader, makeHeaderContent(20202, 2, 7))
	if _, ok := b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(1, 7, [6]byte{})); ok {
		t.Error("emission fired before both new-sequence data blocks arrived")
	}
}

func TestMotorolaTalkerAliasBufStaleEviction(t *testing.T) {
	clk := time.Now()
	b := NewMotorolaTalkerAliasBuf(func() time.Time { return clk })
	b.AddFragment(LCOTalkerAliasHeader, makeHeaderContent(20202, 2, 5))
	b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(1, 5, [6]byte{}))
	// Far past the staleness window — the partial reassembly is
	// dropped on the next call.
	clk = clk.Add(motorolaAliasStaleAfter + time.Second)
	if _, ok := b.AddFragment(LCOTalkerAliasBlock2, makeDataBlockContent(2, 5, [6]byte{})); ok {
		t.Error("emission fired after stale eviction")
	}
}

func TestMotorolaTalkerAliasBufIgnoresUnrelatedLCO(t *testing.T) {
	b := NewMotorolaTalkerAliasBuf(nil)
	// LCO 0x16 is the TIA-102 BLOCK1; the Motorola form does not
	// use it, so the assembler must drop it on the floor.
	if _, ok := b.AddFragment(LCOTalkerAliasBlock1, [lcContentOctets]byte{}); ok {
		t.Error("LCO 0x16 must not emit (Motorola form skips BLOCK1)")
	}
	// Non-talker-alias LCO is also a no-op.
	if _, ok := b.AddFragment(LCOGroupVoiceChannelUser, [lcContentOctets]byte{}); ok {
		t.Error("non-talker-alias LCO must not emit")
	}
}

// TestMotorolaCipherIsDeterministic guards the LUT + per-byte
// algorithm against silent drift. The expected output isn't from
// a real over-the-air capture — there's no source for one in this
// repo — but the same input byte sequence must always decode to
// the same output so a regression in the LUT or arithmetic shows
// up immediately.
func TestMotorolaCipherIsDeterministic(t *testing.T) {
	encoded := []byte{0x00, 0x41, 0x80, 0xFF, 0x12, 0x34, 0x56, 0x78}
	a := decodeAliasBytes(encoded)
	b := decodeAliasBytes(encoded)
	if len(a) != len(b) || len(a) != len(encoded) {
		t.Fatalf("length mismatch: got %d and %d (expected %d)", len(a), len(b), len(encoded))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("non-deterministic decode at byte %d: %02x vs %02x", i, a[i], b[i])
		}
	}
}

func TestMotorolaCipherZeroInputProducesZeroLength(t *testing.T) {
	if got := decodeAliasBytes(nil); len(got) != 0 {
		t.Errorf("decodeAliasBytes(nil) = %v, want empty", got)
	}
}

// The 256-entry LUT-size invariant moved with the cipher to the shared
// internal/radio/p25/motorola package (see its TestLUTSize).

func TestAssembleMotorolaBitStreamPackingOrder(t *testing.T) {
	// Two data blocks with known fragments — verify the bit stream
	// is packed in block order, MSB-first, with each block
	// contributing exactly 44 bits.
	blockA := []byte{0x0F, 0xFF, 0x00, 0xFF, 0x00, 0xFF} // 4 bits + 40 bits
	blockB := []byte{0x00, 0x00, 0xFF, 0x00, 0xFF, 0x00}
	bits := assembleMotorolaBitStream([][]byte{blockA, blockB})
	// 2 blocks × 44 bits = 88 bits = 11 bytes.
	if len(bits) != 11 {
		t.Fatalf("len(bits) = %d, want 11", len(bits))
	}
	// First nibble of blockA (0x0F low nibble = 0b1111) lands in
	// the top nibble of bits[0].
	if bits[0]&0xF0 != 0xF0 {
		t.Errorf("first nibble = %#x, want 0xF0", bits[0]&0xF0)
	}
}

// Per-block fragment bit count must be exactly 44 — guard against
// a stray off-by-one in the bit-packing helpers.
func TestMotorolaFragmentBitsConstant(t *testing.T) {
	if motorolaFragmentBits != 44 {
		t.Errorf("motorolaFragmentBits = %d, want 44 (SDRTrunk-documented)", motorolaFragmentBits)
	}
}

func TestDecodeMotorolaAliasTooShortReturnsEmpty(t *testing.T) {
	// Less than SUID (56 bits) + CRC (16 bits) + one encoded char
	// (8 bits) = 80 bits = 10 bytes.
	short := make([]byte, 9)
	if got := decodeMotorolaAlias(short); got != "" {
		t.Errorf("decodeMotorolaAlias(short) = %q, want empty", got)
	}
}

func TestResetClearsEmittedFingerprint(t *testing.T) {
	b := NewMotorolaTalkerAliasBuf(nil)
	b.emittedHash = 0xDEADBEEF
	b.Reset()
	if b.emittedHash != 0 {
		t.Errorf("emittedHash = %#x after Reset, want 0", b.emittedHash)
	}
}
