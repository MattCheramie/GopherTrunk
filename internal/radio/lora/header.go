package lora

// Explicit-header codec and frame geometry, in the Semtech air format
// (pinned in bitexact_test.go against gr-lora_sdr and jkadbear/LoRaPHY).
//
// The header is five nibbles: payload length (high, low), (CR<<1)|CRC-on,
// then a 5-bit checksum split as one bit in nibble 3 and four in nibble 4.
// It opens the first interleaver block, which is always sent at coding
// rate 4/8 in reduced-rate mode (SF-2 rows): that block carries SF-2
// nibbles — the five header nibbles plus the first SF-7 payload nibbles —
// in 8 symbols. Spreading factor is recovered from the preamble (see
// demod.syncFrame), not the header.

// HeaderCR is the fixed coding rate (4/8) of the first block.
const HeaderCR = 4

// HeaderNibbles is the number of nibbles the explicit header occupies.
const HeaderNibbles = 5

// HeaderSymbols is the number of symbols the first block spans (4+HeaderCR).
const HeaderSymbols = 4 + HeaderCR

// Header holds the decoded explicit-header fields.
type Header struct {
	PayloadLen int  // payload byte count (excludes CRC)
	CR         int  // payload coding rate, 1..4 (4/5..4/8)
	HasCRC     bool // a 2-byte payload CRC follows
	Valid      bool // header checksum passed and CR is in range
}

// headerChecksum computes the 5-bit header checksum (c4..c0) over the
// first three header nibbles.
func headerChecksum(h0, h1, h2 uint8) uint8 {
	b := func(x uint8, k uint) uint8 { return (x >> k) & 1 }
	c4 := b(h0, 3) ^ b(h0, 2) ^ b(h0, 1) ^ b(h0, 0)
	c3 := b(h0, 3) ^ b(h1, 3) ^ b(h1, 2) ^ b(h1, 1) ^ b(h2, 0)
	c2 := b(h0, 2) ^ b(h1, 3) ^ b(h1, 0) ^ b(h2, 3) ^ b(h2, 1)
	c1 := b(h0, 1) ^ b(h1, 2) ^ b(h1, 0) ^ b(h2, 2) ^ b(h2, 1) ^ b(h2, 0)
	c0 := b(h0, 0) ^ b(h1, 1) ^ b(h2, 3) ^ b(h2, 2) ^ b(h2, 1) ^ b(h2, 0)
	return c4<<4 | c3<<3 | c2<<2 | c1<<1 | c0
}

// encodeHeader returns the HeaderNibbles header nibbles for a frame.
func encodeHeader(payloadLen, cr int, hasCRC bool) []uint8 {
	h0 := uint8(payloadLen>>4) & 0x0f
	h1 := uint8(payloadLen) & 0x0f
	h2 := uint8(cr&0x07) << 1
	if hasCRC {
		h2 |= 1
	}
	chk := headerChecksum(h0, h1, h2)
	return []uint8{h0, h1, h2, chk >> 4, chk & 0x0f}
}

// ParseHeader decodes the explicit header from the first HeaderNibbles
// nibbles of the first block.
func ParseHeader(nibbles []uint8) Header {
	if len(nibbles) < HeaderNibbles {
		return Header{}
	}
	h0, h1, h2 := nibbles[0]&0x0f, nibbles[1]&0x0f, nibbles[2]&0x0f
	got := (nibbles[3]&1)<<4 | nibbles[4]&0x0f
	h := Header{
		PayloadLen: int(h0)<<4 | int(h1),
		CR:         int(h2 >> 1),
		HasCRC:     h2&1 != 0,
		Valid:      got == headerChecksum(h0, h1, h2),
	}
	if h.CR < 1 || h.CR > 4 {
		h.Valid = false
	}
	return h
}

// frameNibbles is the total nibble count of a frame: header, payload and
// optional CRC.
func frameNibbles(payloadLen int, hasCRC bool) int {
	n := HeaderNibbles + 2*payloadLen
	if hasCRC {
		n += 4
	}
	return n
}

// firstBlockNibbles is the number of nibbles the first (reduced-rate,
// 4/8) block carries: one per interleaver row, SF-2 rows.
func firstBlockNibbles(sf int) int { return sf - 2 }

// payloadBlockCount returns how many interleaver blocks follow the first
// block. Each carries payloadPPM(sf, ldro) nibbles — SF normally, SF-2
// when LDRO reduces the rate.
func payloadBlockCount(sf, payloadLen int, hasCRC, ldro bool) int {
	rem := frameNibbles(payloadLen, hasCRC) - firstBlockNibbles(sf)
	if rem <= 0 {
		return 0
	}
	per := payloadPPM(sf, ldro)
	return (rem + per - 1) / per
}

// PayloadSymbolCount returns the number of symbols that follow the first
// (header) block for the given parameters. LDRO increases the count
// because each reduced-rate block carries fewer nibbles.
func PayloadSymbolCount(sf, cr, payloadLen int, hasCRC, ldro bool) int {
	return payloadBlockCount(sf, payloadLen, hasCRC, ldro) * (4 + cr)
}

// FrameSymbolCount is the total number of data symbols after the SFD:
// the HeaderSymbols-long first block plus PayloadSymbolCount.
func FrameSymbolCount(sf, cr, payloadLen int, hasCRC, ldro bool) int {
	return HeaderSymbols + PayloadSymbolCount(sf, cr, payloadLen, hasCRC, ldro)
}
