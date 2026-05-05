// BPTC(196,96) — Block Product Turbo Code used by DMR (ETSI TS 102 361-1
// Annex B) to protect 96-bit information blocks (e.g. CSBKs, voice-header
// LCs) over a 196-bit channel codeword.
//
// Internal matrix layout (13 rows × 15 cols + 1 R bit = 196):
//
//   - m[0..8][0..10]   : 9 × 11 = 99 info cells; we use the first 96 and
//                        zero-pad the trailing 3 (m[8][8..10]).
//   - m[0..8][11..14]  : 4 row-parity bits per info row, computed by
//                        Hamming(15,11,3) over m[r][0..10].
//   - m[9..12][0..14]  : 4 × 15 = 60 column-parity bits, computed by
//                        Hamming(13,9,3) over m[0..8][c].
//   - R bit            : reserved 0, sits at the end of the de-interleaved
//                        stream (index 195).
//
// Stream packing is row-major: bit i = m[i/15][i%15] for i = 0..194.
//
// Channel interleaving (ETSI §B.2.1): channelBit[i] = streamBit[(i*181) %
// 196]. The inverse map (de-interleave) uses the modular inverse of 181
// mod 196, which is 13.
//
// NOTE: this implementation is internally consistent (encode→interleave→
// deinterleave→decode round-trips and corrects single-bit errors per row
// and per column). The exact mapping from spec field bit positions to the
// matrix info cells will need a final cross-check against ETSI TS 102
// 361-1 Annex B before live DMR captures; see internal/radio/dmr/burst.go.

package framing

const bptcN = 196

// InterleaveBPTC applies the BPTC(196,96) interleaver: out[i] =
// in[(i*181) mod 196].
func InterleaveBPTC(in []byte) []byte {
	if len(in) != bptcN {
		panic("framing: InterleaveBPTC requires 196 bits")
	}
	out := make([]byte, bptcN)
	for i := 0; i < bptcN; i++ {
		out[i] = in[(i*181)%bptcN]
	}
	return out
}

// DeinterleaveBPTC reverses InterleaveBPTC.
func DeinterleaveBPTC(in []byte) []byte {
	if len(in) != bptcN {
		panic("framing: DeinterleaveBPTC requires 196 bits")
	}
	out := make([]byte, bptcN)
	for i := 0; i < bptcN; i++ {
		out[(i*181)%bptcN] = in[i]
	}
	return out
}

// EncodeBPTC196_96 takes 96 information bits (each entry 0/1) and returns
// the 196 channel bits post-interleave.
func EncodeBPTC196_96(info []byte) []byte {
	if len(info) != 96 {
		panic("framing: EncodeBPTC196_96 requires 96 info bits")
	}
	var m [13][15]byte
	// Place info into the upper-left 9×11 block. The last 3 cells of
	// row 8 are reserved zero.
	for r := 0; r < 9; r++ {
		for c := 0; c < 11; c++ {
			i := r*11 + c
			if i < 96 {
				m[r][c] = info[i] & 1
			}
		}
	}
	// Row parity: Hamming(15,11,3) over each info row.
	for r := 0; r < 9; r++ {
		var d uint16
		for c := 0; c < 11; c++ {
			d |= uint16(m[r][c]) << uint(c)
		}
		cw := HammingEncode15_11(d)
		// cw bit 0..3 = parity bits → matrix columns 11..14.
		m[r][11] = byte(cw & 1)
		m[r][12] = byte((cw >> 1) & 1)
		m[r][13] = byte((cw >> 2) & 1)
		m[r][14] = byte((cw >> 3) & 1)
	}
	// Column parity: Hamming(13,9,3) over each column.
	for c := 0; c < 15; c++ {
		var d uint16
		for r := 0; r < 9; r++ {
			d |= uint16(m[r][c]) << uint(r)
		}
		cw := HammingEncode13_9(d)
		// cw bit 0..3 = parity bits → matrix rows 9..12.
		m[9][c] = byte(cw & 1)
		m[10][c] = byte((cw >> 1) & 1)
		m[11][c] = byte((cw >> 2) & 1)
		m[12][c] = byte((cw >> 3) & 1)
	}
	// Flatten row-major into 195 cells; R bit at index 195.
	stream := make([]byte, bptcN)
	idx := 0
	for r := 0; r < 13; r++ {
		for c := 0; c < 15; c++ {
			stream[idx] = m[r][c]
			idx++
		}
	}
	stream[bptcN-1] = 0 // R bit
	return InterleaveBPTC(stream)
}

// DecodeBPTC196_96 reverses the channel encoding: 196 channel bits →
// 96 information bits. Returns (info, totalCorrected). totalCorrected is
// -1 if any row or column was uncorrectable after iterative passes; it is
// otherwise the number of single-bit corrections applied.
func DecodeBPTC196_96(channel []byte) ([]byte, int) {
	if len(channel) != bptcN {
		panic("framing: DecodeBPTC196_96 requires 196 bits")
	}
	stream := DeinterleaveBPTC(channel)
	var m [13][15]byte
	idx := 0
	for r := 0; r < 13; r++ {
		for c := 0; c < 15; c++ {
			m[r][c] = stream[idx] & 1
			idx++
		}
	}

	totalCorrected := 0
	failed := false
	for pass := 0; pass < 5; pass++ {
		anyChanged := false
		// Row pass.
		for r := 0; r < 9; r++ {
			var cw uint16
			for c := 0; c < 11; c++ {
				cw |= uint16(m[r][c]) << uint(c+4) // info → cw bits 4..14
			}
			cw |= uint16(m[r][11])      // parity bits → cw bits 0..3
			cw |= uint16(m[r][12]) << 1
			cw |= uint16(m[r][13]) << 2
			cw |= uint16(m[r][14]) << 3
			data, errs := HammingDecode15_11(cw)
			if errs == 1 {
				totalCorrected++
				anyChanged = true
			} else if errs < 0 {
				failed = true
			}
			cw2 := HammingEncode15_11(data)
			for c := 0; c < 11; c++ {
				m[r][c] = byte((data >> uint(c)) & 1)
			}
			m[r][11] = byte(cw2 & 1)
			m[r][12] = byte((cw2 >> 1) & 1)
			m[r][13] = byte((cw2 >> 2) & 1)
			m[r][14] = byte((cw2 >> 3) & 1)
		}
		// Column pass.
		for c := 0; c < 15; c++ {
			var cw uint16
			for r := 0; r < 9; r++ {
				cw |= uint16(m[r][c]) << uint(r+4) // info → cw bits 4..12
			}
			cw |= uint16(m[9][c])        // parity bits → cw bits 0..3
			cw |= uint16(m[10][c]) << 1
			cw |= uint16(m[11][c]) << 2
			cw |= uint16(m[12][c]) << 3
			data, errs := HammingDecode13_9(cw)
			if errs == 1 {
				totalCorrected++
				anyChanged = true
			} else if errs < 0 {
				failed = true
			}
			cw2 := HammingEncode13_9(data)
			for r := 0; r < 9; r++ {
				m[r][c] = byte((data >> uint(r)) & 1)
			}
			m[9][c] = byte(cw2 & 1)
			m[10][c] = byte((cw2 >> 1) & 1)
			m[11][c] = byte((cw2 >> 2) & 1)
			m[12][c] = byte((cw2 >> 3) & 1)
		}
		if !anyChanged {
			break
		}
	}

	info := make([]byte, 96)
	for r := 0; r < 9; r++ {
		for c := 0; c < 11; c++ {
			i := r*11 + c
			if i < 96 {
				info[i] = m[r][c]
			}
		}
	}
	if failed {
		return info, -1
	}
	return info, totalCorrected
}
