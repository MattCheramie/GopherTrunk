// BPTC(196,96) — Block Product Turbo Code used by DMR (ETSI TS 102 361-1
// Annex B) to protect 96-bit information blocks (e.g. CSBKs, voice-header
// LCs) over a 196-bit channel codeword.
//
// On-air → matrix (decode):
//
//   - Deinterleave: deInter[a] = channel[(a*181) mod 196], a = 0..195
//     (ETSI §B.2.1). deInter[0] is the reserved R bit.
//   - The remaining 195 cells form a 13-row × 15-col product-code matrix,
//     cell (r,c) = deInter[r*15 + c + 1].
//   - Each of the 15 columns is a Hamming(13,9,3) codeword over rows 0..12
//     (data rows 0..8, parity rows 9..12).
//   - Each of the 9 data rows is a Hamming(15,11,3) codeword over cols 0..14
//     (data cols 0..10, parity cols 11..14).
//   - The 96 info bits are read from row 0 cols 3..10 (8 bits) then rows
//     1..8 cols 0..10 (11 bits each). Row 0 cols 0..2 are reserved zero.
//
// This matches the de-facto on-air implementation (MMDVM CBPTC19696, OP25,
// DSD) verified against ETSI TS 102 361-1 Annex B; see bptc_test.go for the
// canonical-layout golden vector that pins the bit positions and Hamming
// conventions independently of this file's encoder.

package framing

const bptcN = 196

// InterleaveBPTC applies the BPTC(196,96) interleaver: out[i] =
// in[(i*181) mod 196]. This is the on-air → deinterleaved mapping used by
// the decoder (deInter[a] = channel[(a*181) mod 196]).
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

// DeinterleaveBPTC reverses InterleaveBPTC: out[(i*181) mod 196] = in[i].
// This is the deinterleaved → on-air mapping used by the encoder.
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
	// Place the 96 info bits: row 0 cols 3..10 (8 bits), then rows 1..8
	// cols 0..10 (11 bits each). Row 0 cols 0..2 stay reserved zero.
	idx := 0
	for c := 3; c <= 10; c++ {
		m[0][c] = info[idx] & 1
		idx++
	}
	for r := 1; r <= 8; r++ {
		for c := 0; c <= 10; c++ {
			m[r][c] = info[idx] & 1
			idx++
		}
	}
	// Row parity: Hamming(15,11,3) over cols 0..10 of rows 0..8.
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
	// Column parity: Hamming(13,9,3) over rows 0..8 of each column.
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
	// Flatten to the deinterleaved stream (R bit at index 0, cell (r,c) at
	// index r*15+c+1) and interleave onto the channel.
	deInter := make([]byte, bptcN)
	for r := 0; r < 13; r++ {
		for c := 0; c < 15; c++ {
			deInter[r*15+c+1] = m[r][c]
		}
	}
	return DeinterleaveBPTC(deInter)
}

// DecodeBPTC196_96 reverses the channel encoding: 196 channel bits →
// 96 information bits. Returns (info, totalCorrected). totalCorrected is
// -1 if any row or column was uncorrectable after iterative passes; it is
// otherwise the number of single-bit corrections applied.
func DecodeBPTC196_96(channel []byte) ([]byte, int) {
	if len(channel) != bptcN {
		panic("framing: DecodeBPTC196_96 requires 196 bits")
	}
	// Deinterleave: deInter[a] = channel[(a*181) mod 196]. deInter[0] is
	// the reserved R bit; cell (r,c) sits at deInter[r*15+c+1].
	deInter := InterleaveBPTC(channel)
	var m [13][15]byte
	for r := 0; r < 13; r++ {
		for c := 0; c < 15; c++ {
			m[r][c] = deInter[r*15+c+1] & 1
		}
	}

	totalCorrected := 0
	failed := false
	for pass := 0; pass < 5; pass++ {
		anyChanged := false
		// Column pass: Hamming(13,9) over rows 0..12 of each column.
		for c := 0; c < 15; c++ {
			var cw uint16
			for r := 0; r < 9; r++ {
				cw |= uint16(m[r][c]) << uint(r+4) // info → cw bits 4..12
			}
			cw |= uint16(m[9][c]) // parity bits → cw bits 0..3
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
		// Row pass: Hamming(15,11) over the 15 cells of rows 0..8.
		for r := 0; r < 9; r++ {
			var cw uint16
			for c := 0; c < 11; c++ {
				cw |= uint16(m[r][c]) << uint(c+4) // info → cw bits 4..14
			}
			cw |= uint16(m[r][11]) // parity bits → cw bits 0..3
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
		if !anyChanged {
			break
		}
	}

	// Extract the 96 info bits: row 0 cols 3..10, then rows 1..8 cols 0..10.
	info := make([]byte, 96)
	idx := 0
	for c := 3; c <= 10; c++ {
		info[idx] = m[0][c]
		idx++
	}
	for r := 1; r <= 8; r++ {
		for c := 0; c <= 10; c++ {
			info[idx] = m[r][c]
			idx++
		}
	}
	if failed {
		return info, -1
	}
	return info, totalCorrected
}
