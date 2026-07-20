package lora

// Explicit-header codec. The header is the first interleaver block of a
// LoRa frame and is always sent at the most robust coding rate (4/8) so a
// receiver can decode it without yet knowing the payload's coding rate.
// It carries the payload length, the payload coding rate, and whether a
// payload CRC follows. Spreading factor is recovered from the preamble
// (see demod.syncFrame), not the header.
//
// This is a compact, self-consistent header layout — sufficient for SF and
// CR auto-detection across this package's own modulator/demodulator. A
// bit-exact Semtech header (with its reduced-rate SF-2 region) is a
// follow-up calibrated against captured frames.

// HeaderCR is the fixed coding rate the header block is transmitted at.
const HeaderCR = 4

// headerNibbles is the number of 4-bit fields the explicit header occupies
// at the start of the header block.
const headerNibbles = 4

// Header holds the decoded explicit-header fields.
type Header struct {
	PayloadLen int  // payload byte count (excludes CRC)
	CR         int  // payload coding rate, 1..4 (4/5..4/8)
	HasCRC     bool // a 2-byte payload CRC follows
	Valid      bool // header checksum passed
}

// encodeHeader returns the headerNibbles header fields for a frame.
func encodeHeader(payloadLen, cr int, hasCRC bool) []uint8 {
	crcBit := uint8(0)
	if hasCRC {
		crcBit = 1
	}
	n0 := uint8(payloadLen>>4) & 0x0f
	n1 := uint8(payloadLen) & 0x0f
	n2 := (uint8(cr) & 0x07) | (crcBit << 3)
	n3 := (n0 ^ n1 ^ n2) & 0x0f // checksum
	return []uint8{n0, n1, n2, n3}
}

// ParseHeader decodes the leading header fields from a decoded block's
// nibbles. nibbles must hold at least headerNibbles entries.
func ParseHeader(nibbles []uint8) Header {
	if len(nibbles) < headerNibbles {
		return Header{}
	}
	n0, n1, n2, n3 := nibbles[0]&0x0f, nibbles[1]&0x0f, nibbles[2]&0x0f, nibbles[3]&0x0f
	want := (n0 ^ n1 ^ n2) & 0x0f
	h := Header{
		PayloadLen: int(n0)<<4 | int(n1),
		CR:         int(n2 & 0x07),
		HasCRC:     n2&0x08 != 0,
		Valid:      want == n3,
	}
	if h.CR < 1 || h.CR > 4 {
		h.Valid = false
	}
	return h
}

// blockNibbles is the number of data nibbles carried by one interleaver
// block with ppm codewords (rows): one nibble per row. ppm is the spreading
// factor for a full-rate block, or SF-2 for a reduced-rate (LDRO / header)
// block.
func blockNibbles(ppm int) int { return ppm }

// blockSymbols is the number of symbols one interleaver block occupies at
// coding rate cr: (4+cr) symbols. It depends only on cr (the column count),
// not on ppm, so a reduced-rate block still spans (4+cr) symbols — it just
// carries fewer nibbles.
func blockSymbols(cr int) int { return 4 + cr }

// payloadBlockCount returns how many payload interleaver blocks a frame of
// payloadLen bytes (plus an optional 2-byte CRC) needs at spreading factor
// sf. Each block carries blockNibbles(payloadPPM(sf, ldro)) nibbles — SF
// normally, SF-2 when LDRO reduces the rate.
func payloadBlockCount(sf, payloadLen int, hasCRC, ldro bool) int {
	nib := payloadLen * 2
	if hasCRC {
		nib += 4 // 2 CRC bytes
	}
	per := blockNibbles(payloadPPM(sf, ldro))
	if per <= 0 {
		return 0
	}
	return (nib + per - 1) / per
}

// PayloadSymbolCount returns the number of symbols the payload (after the
// header block) occupies for the given parameters. LDRO increases the count
// because each reduced-rate block carries fewer nibbles.
func PayloadSymbolCount(sf, cr, payloadLen int, hasCRC, ldro bool) int {
	return payloadBlockCount(sf, payloadLen, hasCRC, ldro) * blockSymbols(cr)
}
