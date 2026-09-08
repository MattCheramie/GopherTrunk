// Package fleetsync decodes Kenwood FleetSync (and FleetSync II)
// in-band ANI signaling.
//
// FleetSync is the analog in-band data burst Kenwood commercial two-way
// radios key at the start of a PTT transmission. It carries the
// transmitting radio's identity — a Fleet number and a Unit ID (the ANI,
// automatic number identification) — plus status and messaging, on
// otherwise-analog conventional NBFM voice channels. Scanner operators
// use it to see *which* radio is transmitting on a system that is
// otherwise just FM voice, the same role MDC1200 plays for Motorola.
//
// On the air it is a 1200-baud FFSK burst (CCIR tones: mark = 1200 Hz,
// space = 1800 Hz) carried inside the narrowband-FM voice channel — the
// same modulation class GopherTrunk already demodulates for MDC1200 and
// MPT 1327, so a DSP front end can reuse internal/dsp/demod.FFSK. This
// package owns the protocol layer only: a stream of sliced FSK bits is
// framed by [Framer] (sync hunt → capture), and each captured frame is
// handed to [DecodeFrame] here, which validates it and returns a typed
// [Message] carrying the Fleet/Unit ANI.
//
// Frame layout, after a ≥24-bit alternating preamble and the 16-bit sync
// word 0xA23E (most-significant bit first, read straight off the FSK
// slicer): the payload begins 4 bits into the captured stream and is two
// 32-bit words —
//
//	word1[31:0], word2[31:16] : data (fleet / unit / status)
//	word2[15:0]               : the 16-bit block check (CRC)
//
// FleetSync I validates word2's low 16 bits against [fsyncCRC]. FleetSync
// II adds forward error correction: four consecutive 64-bit blocks, each
// 16-bit half-word carrying a single-error-correcting code ([fs2ECCRepair]);
// the corrected nibbles reassemble word1/word2, which are then CRC-checked
// the same way. The ANI is read identically from the recovered words.
//
// The Fleet/Unit field extraction, the CRC (polynomial 0x6815, parity
// bit, 0x0002 final term) and the FS-II parity-check / repair tables are
// the FleetSync framing facts as implemented by the multimon-ng `fsync`
// decoder, which is proven on air; this is a clean-room Go port of that
// framing, cross-checked against a working Python reference contributed
// on issue #437. No third-party source is incorporated.
//
// Verification status: the protocol constants below are pinned to the
// multimon-ng reference and exercised by reference-literal + single-bit
// ECC-correction tests, but this decoder has NOT yet been confirmed
// against a real Kenwood off-air capture. Wiring it to a live DSP front
// end, the events bus, storage and the REST/web surface is deliberately
// staged for after that on-air A/B (issue #437).
package fleetsync

import (
	"fmt"
	"math/bits"
)

const (
	// SyncWord is the 16-bit FleetSync frame synchronization word, most-
	// significant bit first, read directly off the FSK slicer.
	SyncWord uint16 = 0xA23E

	// SyncBits is the length of SyncWord in bits.
	SyncBits = 16

	// FrameBits is the number of payload bits captured after the sync
	// word. This is the FleetSync II maximum (a 4-bit lead-in plus four
	// 64-bit ECC blocks); a FleetSync I frame uses only the first
	// fs1FrameBits of them.
	FrameBits = frameOffset + 4*64 // 260

	// frameOffset is the number of bits between the end of the sync word
	// and the first payload word (empirically fixed by the reference).
	frameOffset = 4

	// fs1FrameBits is the payload length a FleetSync I frame needs: the
	// lead-in plus the two 32-bit words.
	fs1FrameBits = frameOffset + 64 // 68
)

// fsyncPoly is the FleetSync block-check polynomial used by the bit-
// serial CRC (multimon-ng fsync_common.c).
const fsyncPoly uint16 = 0x6815

// fs2ParityCheck are the eight parity-check masks of the FleetSync II
// single-error-correcting code; fs2RepairPos maps a non-zero syndrome to
// the bit to flip. Both are transcribed verbatim from the multimon-ng
// fsync2 tables and pinned by TestFS2ECCRepairCorrectsEverySingleBit.
var (
	fs2ParityCheck = [8]uint16{0x4045, 0x2067, 0x1076, 0x083b, 0x0458, 0x022c, 0x0116, 0x008b}
	fs2RepairPos   = [15]uint16{
		0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80,
		0x17, 0x2e, 0x5c, 0xb8, 0x67, 0xce, 0x8b,
	}
)

// Message is one decoded FleetSync ANI burst.
type Message struct {
	Fleet  int    // transmitting radio's Fleet number
	Unit   int    // transmitting radio's Unit ID
	IsFS2  bool   // decoded via the FleetSync II ECC path
	CRCOK  bool   // the block check validated
	RawHex string // hex of the two recovered 32-bit words
	Body   string // one-line summary for logs / panel
}

// DecodeFrame decodes the raw FSK bits captured immediately after the
// 16-bit sync word (one byte per bit; only bit 0 of each is read). It
// tries FleetSync I first, then falls back to the FleetSync II ECC path
// when enough bits are present. The bool reports whether the block check
// validated; a failure still returns the best-effort FleetSync I words
// with CRCOK=false so a caller can surface marginal bursts.
func DecodeFrame(raw []byte) (Message, bool) {
	if len(raw) < fs1FrameBits {
		return Message{Body: "incomplete"}, false
	}
	word1 := bitsToUint32(raw[frameOffset : frameOffset+32])
	word2 := bitsToUint32(raw[frameOffset+32 : frameOffset+64])

	if fsyncCRC(word1, word2) == uint16(word2&0xFFFF) {
		return newMessage(word1, word2, false, true), true
	}

	if len(raw) >= FrameBits {
		if w1, w2, ok := decodeFS2(raw); ok {
			return newMessage(w1, w2, true, true), true
		}
	}

	// Best-effort: return the FleetSync I words uncorrected.
	return newMessage(word1, word2, false, false), false
}

// decodeFS2 runs the FleetSync II path: four 64-bit blocks, each 16-bit
// half-word ECC-repaired, the corrected high-nibbles reassembled into the
// two data words and CRC-checked. Returns (word1, word2, true) on a clean
// FS-II decode, or (_, _, false) if any half-word is uncorrectable or the
// reassembled CRC fails.
func decodeFS2(raw []byte) (uint32, uint32, bool) {
	var nibblesW1, nibblesW2 []uint32
	for blk := 0; blk < 4; blk++ {
		boff := frameOffset + blk*64
		bw1 := bitsToUint32(raw[boff : boff+32])
		bw2 := bitsToUint32(raw[boff+32 : boff+64])

		ec1, ok1 := fs2ECCRepair(uint16(bw1 & 0xFFFF))
		ec2, ok2 := fs2ECCRepair(uint16((bw1 >> 16) & 0xFFFF))
		ec3, ok3 := fs2ECCRepair(uint16(bw2 & 0xFFFF))
		ec4, ok4 := fs2ECCRepair(uint16((bw2 >> 16) & 0xFFFF))
		if !(ok1 && ok2 && ok3 && ok4) {
			return 0, 0, false
		}

		nibs := [4]uint32{
			uint32((ec1 >> 8) & 0xF),
			uint32((ec2 >> 8) & 0xF),
			uint32((ec3 >> 8) & 0xF),
			uint32((ec4 >> 8) & 0xF),
		}
		if blk < 2 {
			nibblesW1 = append(nibblesW1, nibs[:]...)
		} else {
			nibblesW2 = append(nibblesW2, nibs[:]...)
		}
	}

	var w1, w2 uint32
	for _, n := range nibblesW1 {
		w1 = (w1 << 4) | n
	}
	for _, n := range nibblesW2 {
		w2 = (w2 << 4) | n
	}
	if fsyncCRC(w1, w2) != uint16(w2&0xFFFF) {
		return 0, 0, false
	}
	return w1, w2, true
}

// newMessage extracts the Fleet/Unit ANI from the two recovered data
// words. The field layout — fleet from word1 byte 2 (+99), unit from
// word1 byte 3 and the high nibble of word2 byte 0 (+999) — matches the
// multimon-ng fsync_decode dispatch.
func newMessage(word1, word2 uint32, isFS2, crcOK bool) Message {
	b2 := int((word1 >> 8) & 0xFF)  // fleet source byte
	b3 := int(word1 & 0xFF)         // unit high byte
	b4 := int((word2 >> 24) & 0xFF) // unit low nibble carrier

	m := Message{
		Fleet:  b2 + 99,
		Unit:   (b3 << 4) + ((b4 >> 4) & 0x0F) + 999,
		IsFS2:  isFS2,
		CRCOK:  crcOK,
		RawHex: fmt.Sprintf("%08X%08X", word1, word2),
	}
	m.Body = m.summary()
	return m
}

// summary renders the one-line Body string used by logs and the panel.
func (m Message) summary() string {
	label := "FleetSync"
	if m.IsFS2 {
		label = "FleetSync II"
	}
	s := fmt.Sprintf("%s ANI: fleet=%d unit=%d", label, m.Fleet, m.Unit)
	if !m.CRCOK {
		s += " (CRC?)"
	}
	return s
}

// fsyncCRC computes the FleetSync block check over the two 32-bit words.
// The first 48 bits (word1, then word2's high 16) clock a bit-serial LFSR
// with polynomial 0x6815; word2's low 15 bits (the received check field)
// contribute only to a running parity bit. The final value is
// (lfsr ^ 0x0002 + parity) & 0xFFFF, compared by the caller against
// word2's low 16 bits. Ported bit-for-bit from multimon-ng fsync_common.c.
func fsyncCRC(word1, word2 uint32) uint16 {
	var paritybit, crcsr uint16
	for bit := 0; bit < 48; bit++ {
		var cur uint16
		if bit < 32 {
			cur = uint16((word1 >> (31 - bit)) & 1)
		} else {
			cur = uint16((word2 >> (31 - (bit - 32))) & 1)
		}
		if cur != 0 {
			paritybit ^= 1
		}
		if (cur ^ ((crcsr >> 15) & 1)) != 0 {
			crcsr ^= fsyncPoly
		}
		crcsr <<= 1 // uint16 truncates to 16 bits, matching (<<1)&0xFFFF
	}
	for bit := 48; bit < 63; bit++ {
		if uint16((word2>>(31-(bit-32)))&1) != 0 {
			paritybit ^= 1
		}
	}
	crcsr ^= 0x0002
	return uint16((uint32(crcsr) + uint32(paritybit)) & 0xFFFF)
}

// fs2ECCRepair corrects a single bit error in one 15-bit FleetSync II
// code word. It returns (val, true) when the syndrome is zero (no error),
// (corrected, true) when a single-bit syndrome is found, and (0, false)
// when the error is uncorrectable. Ported from multimon-ng fsync2.
func fs2ECCRepair(val uint16) (uint16, bool) {
	var syndrome uint16
	for i := 0; i < 8; i++ {
		if bits.OnesCount16(fs2ParityCheck[i]&val)&1 == 1 {
			syndrome |= 1 << uint(i)
		}
	}
	if syndrome == 0 {
		return val, true
	}
	for i := 0; i < 15; i++ {
		if syndrome == fs2RepairPos[i] {
			return val ^ (1 << uint(14-i)), true
		}
	}
	return 0, false
}

// bitsToUint32 packs up to 32 one-bit-per-byte values MSB-first into a
// uint32. Only bit 0 of each input byte is read.
func bitsToUint32(b []byte) uint32 {
	var v uint32
	for _, x := range b {
		v = (v << 1) | uint32(x&1)
	}
	return v
}
