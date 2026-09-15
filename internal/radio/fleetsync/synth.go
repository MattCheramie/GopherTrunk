package fleetsync

import "fmt"

// Burst synthesis — test vectors for the DSP front end and the capture
// replay harness.
//
// These builders are the EXACT inverse of DecodeFrame: they lay a
// Fleet/Unit ANI out as the two 32-bit words, solve the block check the
// way a transmitter must (word2's low 16 bits are found by search, not
// assumed — the check field feeds only the CRC's parity bit, so a
// closed-form inverse is not worth the risk of a subtly-wrong constant),
// and, for FleetSync II, spread the words' nibbles across the four
// single-error-correcting 64-bit blocks.
//
// They are self-consistent with the decoder by construction, so a green
// round-trip through them proves the DSP → framer → decoder wiring, NOT
// the framing itself (#764/#771: encoder+decoder sharing one wrong layout
// pass every synthetic test). The framing facts are pinned separately
// against the multimon-ng reference literals in fleetsync_test.go, and
// the on-air gate is cmd/gophertrunk's TestFleetSyncReplay against a real
// Kenwood capture (#437, #1184).

// ANI range the field layout can express: fleet = byte2 + 99 with one
// byte of headroom, unit = (byte3<<4 | nibble) + 999 with 12 bits.
const (
	MinFleet = 99
	MaxFleet = 99 + 0xFF
	MinUnit  = 999
	MaxUnit  = 999 + 0xFFF
)

// ANIWords maps a Fleet/Unit ANI into the word1 data bytes and the
// word2 high-16 data field — the inverse of newMessage. The remaining
// data bytes (word1 bytes 0–1, word2's low data nibble and byte 1) are
// zero; a real burst carries status/message there.
func ANIWords(fleet, unit int) (word1 uint32, dataHi16 uint16, err error) {
	if fleet < MinFleet || fleet > MaxFleet {
		return 0, 0, fmt.Errorf("fleetsync: fleet %d outside %d..%d", fleet, MinFleet, MaxFleet)
	}
	if unit < MinUnit || unit > MaxUnit {
		return 0, 0, fmt.Errorf("fleetsync: unit %d outside %d..%d", unit, MinUnit, MaxUnit)
	}
	b2 := uint32(fleet - 99)
	u := uint32(unit - 999)
	b3 := (u >> 4) & 0xFF
	b4hi := u & 0xF
	word1 = b2<<8 | b3
	dataHi16 = uint16(b4hi << 12)
	return word1, dataHi16, nil
}

// SolveBlockCheck finds the 16-bit block check that makes (word1, word2)
// validate under fsyncCRC and returns the assembled word2. ok is false
// if no check value satisfies the CRC (never observed; kept so callers
// cannot silently emit an invalid frame).
func SolveBlockCheck(word1 uint32, dataHi16 uint16) (word2 uint32, ok bool) {
	for crc := uint32(0); crc <= 0xFFFF; crc++ {
		w2 := uint32(dataHi16)<<16 | crc
		if fsyncCRC(word1, w2) == uint16(crc) {
			return w2, true
		}
	}
	return 0, false
}

// FS1Payload renders a FleetSync I capture as DecodeFrame expects it: the
// 4-bit lead-in, word1 and word2 MSB-first, zero-padded to FrameBits.
func FS1Payload(word1, word2 uint32) []byte {
	raw := make([]byte, FrameBits)
	putUint32Bits(raw[frameOffset:], word1)
	putUint32Bits(raw[frameOffset+32:], word2)
	return raw
}

// FS2Payload renders a FleetSync II capture: the 4-bit lead-in, then four
// 64-bit blocks whose four 16-bit half-words each carry one data nibble
// inside a valid 15-bit ECC code word — blocks 0–1 carry word1's eight
// nibbles, blocks 2–3 word2's, in decodeFS2's extraction order.
func FS2Payload(word1, word2 uint32) []byte {
	raw := make([]byte, FrameBits)
	nibbles := func(w uint32) [8]uint32 {
		var n [8]uint32
		for i := 0; i < 8; i++ {
			n[i] = (w >> uint(28-4*i)) & 0xF
		}
		return n
	}
	n1, n2 := nibbles(word1), nibbles(word2)
	writeBlock := func(blk int, nibs [4]uint32) {
		boff := frameOffset + blk*64
		h0 := uint32(fs2Codeword(nibs[0]))
		h1 := uint32(fs2Codeword(nibs[1]))
		h2 := uint32(fs2Codeword(nibs[2]))
		h3 := uint32(fs2Codeword(nibs[3]))
		putUint32Bits(raw[boff:], (h1<<16)|h0)
		putUint32Bits(raw[boff+32:], (h3<<16)|h2)
	}
	writeBlock(0, [4]uint32{n1[0], n1[1], n1[2], n1[3]})
	writeBlock(1, [4]uint32{n1[4], n1[5], n1[6], n1[7]})
	writeBlock(2, [4]uint32{n2[0], n2[1], n2[2], n2[3]})
	writeBlock(3, [4]uint32{n2[4], n2[5], n2[6], n2[7]})
	return raw
}

// SynthPreambleBits is the alternating lead-in SynthBurst emits ahead of
// the sync word — longer than the 24 bits the framer checks so the
// register is clean when the sync arrives, matching the on-air "3 × 0xAA
// bytes before sync" observation with headroom.
const SynthPreambleBits = 32

// SynthBurst assembles the wire bits of one FleetSync I (fs2=false) or
// FleetSync II (fs2=true) ANI burst: SynthPreambleBits of alternating
// preamble, the 16-bit sync word MSB-first, then the FrameBits payload.
// One byte per bit, mark (binary 1) = 1. Feed it to a Framer directly or
// to demod.ModulateFFSK to produce IQ for the DSP front end.
func SynthBurst(fleet, unit int, fs2 bool) ([]byte, error) {
	word1, hi16, err := ANIWords(fleet, unit)
	if err != nil {
		return nil, err
	}
	word2, ok := SolveBlockCheck(word1, hi16)
	if !ok {
		return nil, fmt.Errorf("fleetsync: no block check satisfies word1=%08x hi16=%04x", word1, hi16)
	}
	var payload []byte
	if fs2 {
		payload = FS2Payload(word1, word2)
	} else {
		payload = FS1Payload(word1, word2)
	}
	out := make([]byte, 0, SynthPreambleBits+SyncBits+len(payload))
	out = append(out, Dotting(SynthPreambleBits)...)
	for i := SyncBits - 1; i >= 0; i-- {
		out = append(out, byte((SyncWord>>uint(i))&1))
	}
	return append(out, payload...), nil
}

// Dotting returns n alternating bits starting with 1 (1,0,1,0,…) — the
// FleetSync preamble pattern, also useful as a symbol-clock training run
// ahead of a synthesised burst.
func Dotting(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte((i + 1) & 1)
	}
	return out
}

// fs2Codeword returns a valid 15-bit FleetSync II code word (syndrome
// zero) whose data nibble (bits 11..8) equals nibble — the first such
// word. Every nibble has several; the decoder reads only the nibble.
func fs2Codeword(nibble uint32) uint16 {
	for v := uint16(0); v < 0x8000; v++ {
		if (uint32(v)>>8)&0xF != nibble {
			continue
		}
		if fixed, ok := fs2ECCRepair(v); ok && fixed == v {
			return v
		}
	}
	panic(fmt.Sprintf("fleetsync: no valid FS2 codeword carries nibble 0x%x", nibble))
}

// putUint32Bits writes v MSB-first as one byte per bit into dst[:32].
func putUint32Bits(dst []byte, v uint32) {
	for i := 0; i < 32; i++ {
		dst[i] = byte((v >> (31 - i)) & 1)
	}
}
