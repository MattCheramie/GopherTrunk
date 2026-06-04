// Variable-length BPTC(128,72) for DMR embedded signalling (ETSI TS 102
// 361-1 Annex B.2.2 / C). The four 32-bit embedded fragments carried by the
// sync field of voice bursts B–E concatenate into 128 channel bits that
// protect a 72-bit Link Control PDU.
//
// On-air → matrix (decode), matching the de-facto reference implementation
// (MMDVM CDMREmbeddedData) verified against ETSI Annex B/C:
//
//   - Deinterleave the 128 on-air bits into an 8-row × 16-col matrix with
//     data[b] = onair[a], b += 16, wrapping b -= 127 when b > 127.
//   - Rows 0..6 are each a Hamming(16,11,4) codeword (data cols 0..10,
//     parity cols 11..15).
//   - Row 7 holds even column parity over rows 0..6 (every column of the
//     full 8-row matrix XORs to 0).
//   - The 72 LC bits are read from rows 0..6 cols 0..10 — except cols 0..9
//     for rows 2..6, whose col 10 carries one bit of the 5-bit checksum.
//   - The 5-bit checksum (cells data[42],58,74,90,106, weights 16..1) is the
//     sum of the nine LC octets modulo 31.

package framing

const (
	EmbLCBits      = 72  // a Full Link Control PDU
	EmbChannelBits = 128 // on-air bits (4 × 32-bit fragments)
	embFragmentLen = EmbChannelBits / 4
)

// EmbeddedFragmentBits is the per-burst embedded-signalling fragment length
// (bits) carried by voice bursts B–E.
const EmbeddedFragmentBits = embFragmentLen

// embLCRanges lists the matrix index ranges (half-open) holding the 72 LC
// bits, in output order: rows 0..6 cols 0..10, but only cols 0..9 for rows
// 2..6 (their col 10 is a checksum bit). Verbatim from MMDVM
// CDMREmbeddedData.
var embLCRanges = [7][2]int{
	{0, 11}, {16, 27}, {32, 42}, {48, 58}, {64, 74}, {80, 90}, {96, 106},
}

// emb16114Parity computes the five Hamming(16,11,4) parity bits for the 11
// data bits d[0..10] (MMDVM CHamming::encode16114), returned for cells
// d[11..15].
func emb16114Parity(d []byte) [5]byte {
	b := func(i int) byte { return d[i] & 1 }
	return [5]byte{
		b(0) ^ b(1) ^ b(2) ^ b(3) ^ b(5) ^ b(7) ^ b(8),
		b(1) ^ b(2) ^ b(3) ^ b(4) ^ b(6) ^ b(8) ^ b(9),
		b(2) ^ b(3) ^ b(4) ^ b(5) ^ b(7) ^ b(9) ^ b(10),
		b(0) ^ b(1) ^ b(2) ^ b(4) ^ b(6) ^ b(7) ^ b(10),
		b(0) ^ b(2) ^ b(5) ^ b(6) ^ b(8) ^ b(9) ^ b(10),
	}
}

// emb16114Syndrome returns the 5-bit mismatch between the recomputed parity
// and the received parity cells d[11..15] of one 16-bit row.
func emb16114Syndrome(d []byte) byte {
	p := emb16114Parity(d)
	var syn byte
	for i := 0; i < 5; i++ {
		if p[i]^(d[11+i]&1) != 0 {
			syn |= 1 << uint(i)
		}
	}
	return syn
}

// decodeEmb16114 checks and, if possible, corrects one 16-bit row in place.
// Returns 0 (clean), 1 (single-bit corrected), or -1 (uncorrectable). The
// (16,11,4) code is SEC-DED: it corrects one error and detects two.
func decodeEmb16114(d []byte) int {
	if emb16114Syndrome(d) == 0 {
		return 0
	}
	for bit := 0; bit < 16; bit++ {
		d[bit] ^= 1
		if emb16114Syndrome(d) == 0 {
			return 1
		}
		d[bit] ^= 1 // revert
	}
	return -1
}

// embFiveBit computes the DMR embedded-LC 5-bit checksum: the sum of the
// nine LC octets (big-endian within the 72-bit block) modulo 31
// (MMDVM CCRC::encodeFiveBit).
func embFiveBit(lc []byte) byte {
	total := 0
	for i := 0; i < EmbLCBits; i += 8 {
		var c byte
		for j := 0; j < 8; j++ {
			c = c<<1 | (lc[i+j] & 1)
		}
		total += int(c)
	}
	return byte(total % 31)
}

// extractEmbLC reads the 72 LC bits out of the decoded matrix.
func extractEmbLC(data []byte) []byte {
	lc := make([]byte, EmbLCBits)
	idx := 0
	for _, rg := range embLCRanges {
		for a := rg[0]; a < rg[1]; a++ {
			lc[idx] = data[a] & 1
			idx++
		}
	}
	return lc
}

// EncodeEmbeddedLC packs a 72-bit Link Control PDU (one bit per byte,
// MSB-first) into the 128 embedded-signalling channel bits.
func EncodeEmbeddedLC(lc []byte) []byte {
	if len(lc) != EmbLCBits {
		panic("framing: EncodeEmbeddedLC requires 72 LC bits")
	}
	var data [128]byte
	// LC bits into the data cells.
	idx := 0
	for _, rg := range embLCRanges {
		for a := rg[0]; a < rg[1]; a++ {
			data[a] = lc[idx] & 1
			idx++
		}
	}
	// 5-bit checksum into col 10 of rows 2..6 (weights 16..1).
	crc := embFiveBit(lc)
	data[42] = (crc >> 4) & 1
	data[58] = (crc >> 3) & 1
	data[74] = (crc >> 2) & 1
	data[90] = (crc >> 1) & 1
	data[106] = crc & 1
	// Row Hamming(16,11) parity (rows 0..6) — must follow the checksum so
	// rows 2..6 protect their col-10 bit.
	for r := 0; r < 7; r++ {
		p := emb16114Parity(data[r*16 : r*16+11])
		for i := 0; i < 5; i++ {
			data[r*16+11+i] = p[i]
		}
	}
	// Column parity (row 7).
	for c := 0; c < 16; c++ {
		var par byte
		for r := 0; r < 7; r++ {
			par ^= data[c+r*16]
		}
		data[c+112] = par
	}
	// Interleave back to the on-air order: onair[a] = data[b], b += 16
	// wrapping at 127.
	onair := make([]byte, EmbChannelBits)
	b := 0
	for a := 0; a < EmbChannelBits; a++ {
		onair[a] = data[b]
		b += 16
		if b > 127 {
			b -= 127
		}
	}
	return onair
}

// DecodeEmbeddedLC reverses EncodeEmbeddedLC: 128 channel bits → the 72-bit
// Link Control. Returns (lc, corrected); corrected is the number of
// single-bit row corrections applied, or -1 if any row was uncorrectable,
// the column parity failed, or the recovered checksum did not verify.
func DecodeEmbeddedLC(onair []byte) ([]byte, int) {
	if len(onair) != EmbChannelBits {
		panic("framing: DecodeEmbeddedLC requires 128 channel bits")
	}
	// Deinterleave into the 8×16 matrix.
	var data [128]byte
	b := 0
	for a := 0; a < EmbChannelBits; a++ {
		data[b] = onair[a] & 1
		b += 16
		if b > 127 {
			b -= 127
		}
	}
	// Hamming(16,11) on the seven data rows.
	corrected := 0
	for r := 0; r < 7; r++ {
		c := decodeEmb16114(data[r*16 : r*16+16])
		if c < 0 {
			return extractEmbLC(data[:]), -1
		}
		corrected += c
	}
	// Column parity over all eight rows must be even.
	for c := 0; c < 16; c++ {
		var par byte
		for r := 0; r < 8; r++ {
			par ^= data[c+r*16]
		}
		if par != 0 {
			return extractEmbLC(data[:]), -1
		}
	}
	lc := extractEmbLC(data[:])
	// 5-bit checksum.
	var crc byte
	for i, pos := range [5]int{42, 58, 74, 90, 106} {
		if data[pos] != 0 {
			crc |= 1 << uint(4-i)
		}
	}
	if embFiveBit(lc) != crc {
		return lc, -1
	}
	return lc, corrected
}
