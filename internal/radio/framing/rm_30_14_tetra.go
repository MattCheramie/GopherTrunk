package framing

// Shortened (30,14) Reed-Muller block code per ETSI EN 300 392-2
// §8.2.3.2. Used by the TETRA AACH (Access Assignment Channel) to
// encode 14 type-1 information bits into 30 type-2 channel bits with
// no further convolutional coding or interleaving — AACH's
// type-4 block equals its type-2 block.
//
// Generator matrix G is [I_14 | P] where I_14 is the 14×14 identity
// (so the code is systematic — the first 14 type-2 bits equal the
// 14 type-1 bits) and P is the 14×16 parity matrix from
// equation (8.13):
//
//	row 1:  1 0 0 1 1 0 1 1 0 1 1 0 0 0 0 0
//	row 2:  0 0 1 0 1 1 0 1 1 1 1 0 0 0 0 0
//	row 3:  1 1 1 1 1 1 0 0 0 0 1 0 0 0 0 0
//	row 4:  1 1 1 0 0 0 0 0 0 0 1 1 1 1 0 0
//	row 5:  1 0 0 1 1 0 0 0 0 0 1 1 1 0 1 0
//	row 6:  0 1 0 1 0 1 0 0 0 0 1 1 0 1 1 0
//	row 7:  0 0 1 0 1 1 0 0 0 0 1 0 1 1 1 0
//	row 8:  1 1 1 1 1 1 1 1 1 1 0 1 1 1 1 1
//	row 9:  1 0 0 0 0 0 1 1 0 0 1 1 1 0 0 1
//	row 10: 0 1 0 0 0 0 1 0 1 0 1 1 0 1 0 1
//	row 11: 0 0 1 0 0 0 0 1 1 0 1 0 1 1 0 1
//	row 12: 0 0 0 1 0 0 1 0 0 1 1 1 0 0 1 1
//	row 13: 0 0 0 0 1 0 0 1 0 1 1 0 1 0 1 1
//	row 14: 0 0 0 0 0 1 0 0 1 1 1 0 0 1 1 1
//
// Encoding rule: b_2 = b_1 * G over GF(2).

// rm3014ParityMatrix is the 14×16 parity portion P of the (30,14)
// generator matrix. rm3014ParityMatrix[r][c] is row r, column c.
var rm3014ParityMatrix = [14][16]byte{
	{1, 0, 0, 1, 1, 0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0},
	{0, 0, 1, 0, 1, 1, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
	{1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0},
	{1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 0},
	{1, 0, 0, 1, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0},
	{0, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 1, 0},
	{0, 0, 1, 0, 1, 1, 0, 0, 0, 0, 1, 0, 1, 1, 1, 0},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1},
	{1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1},
	{0, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1},
	{0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 0, 1, 1, 0, 1},
	{0, 0, 0, 1, 0, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1},
	{0, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 1, 1},
	{0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1},
}

// EncodeRM3014Tetra encodes 14 type-1 information bits (each entry
// 0/1, MSB-first per the spec convention) into 30 type-2 channel
// bits via the shortened (30,14) Reed-Muller code from
// EN 300 392-2 §8.2.3.2. The first 14 output bits equal the input
// (systematic encoding); the trailing 16 bits are the parity sum
// of the input vector dotted into each column of the parity matrix.
func EncodeRM3014Tetra(info []byte) []byte {
	if len(info) != 14 {
		return nil
	}
	out := make([]byte, 30)
	// Systematic: first 14 output bits equal the input bits.
	for i := 0; i < 14; i++ {
		out[i] = info[i] & 1
	}
	// Parity: for each of the 16 parity columns, XOR the input
	// bits whose row has a 1 in that column.
	for c := 0; c < 16; c++ {
		var p byte
		for r := 0; r < 14; r++ {
			if info[r]&1 != 0 && rm3014ParityMatrix[r][c] != 0 {
				p ^= 1
			}
		}
		out[14+c] = p
	}
	return out
}

// DecodeRM3014Tetra decodes 30 received bits via minimum-Hamming-
// distance search across all 2^14 = 16 384 valid codewords. Returns
// (info, errs) where info is the 14-bit information vector closest
// to the received word and errs is the Hamming distance to that
// codeword. errs = 0 means the received word is a valid codeword.
//
// For a clean channel this recovers the original info exactly; for
// noisier captures the (30,14) RM code has minimum distance ≥ 4
// (giving guaranteed single-bit correction across the 30-bit
// codeword) and detect-up-to-3 capability — beyond that the
// closest-codeword decoder may mis-correct.
func DecodeRM3014Tetra(received []byte) ([]byte, int) {
	if len(received) != 30 {
		return nil, -1
	}
	bestDist := 31
	var bestInfo [14]byte
	for v := 0; v < (1 << 14); v++ {
		var info [14]byte
		for i := 0; i < 14; i++ {
			info[i] = byte((v >> uint(13-i)) & 1)
		}
		cw := EncodeRM3014Tetra(info[:])
		dist := 0
		for i := 0; i < 30; i++ {
			if cw[i] != received[i]&1 {
				dist++
			}
		}
		if dist < bestDist {
			bestDist = dist
			bestInfo = info
			if dist == 0 {
				break
			}
		}
	}
	out := make([]byte, 14)
	copy(out, bestInfo[:])
	return out, bestDist
}
