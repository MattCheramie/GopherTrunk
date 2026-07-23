package acelp

// Parameter unpacking and LSP dequantisation for the TETRA ACELP decoder,
// ETSI EN 300 395-2. A decoded 137-bit speech frame carries 23 quantiser
// indices; the decoder first unpacks them (bits2prm) and, for a good frame,
// dequantises the three LSP split-VQ indices back to 10 line spectral pairs in
// the cosine domain (dLsp334). Both are clean-room ports of the reference
// Bits2prm_Tetra / D_Lsp334 (see package doc for provenance).

const (
	lpcOrder  = 10 // LPC order p — 10 line spectral pairs
	numParams = 23 // quantiser indices per speech frame (excl. BFI)
	numFields = 24 // parm[] layout: BFI followed by the 23 indices
)

// Parameter layout of parm[] after unpacking. parm[0] is the bad-frame
// indicator; parm[1..3] are the LSP split-VQ indices; the remaining 20 are five
// per subframe (pitch, codebook, sign, shift, energy) for the four subframes.
const (
	pBFI  = 0 // bad-frame indicator (1 = frame erased)
	pLSP0 = 1 // LSP split-VQ index 1 (8 bits, dico1)
	pLSP1 = 2 // LSP split-VQ index 2 (9 bits, dico2)
	pLSP2 = 3 // LSP split-VQ index 3 (9 bits, dico3)
	pSub0 = 4 // first subframe field; 5 fields per subframe follow
)

// Per-subframe field offsets relative to a subframe's base index.
const (
	sfPitch  = 0 // pitch (adaptive codebook) lag index
	sfCode   = 1 // algebraic (fixed) codebook index
	sfSign   = 2 // pulse global sign
	sfShift  = 3 // pulse shift
	sfEnergy = 4 // pitch + innovation gain VQ index
	sfFields = 5 // fields per subframe
	numSubfr = 4 // subframes per 30 ms frame
)

// bitWidths is the number of bits carrying each of the 23 indices, in unpacking
// order (Bits2prm_Tetra bitno[]): split-VQ LSP, then the four subframes. The
// first subframe's pitch index is 8 bits, the others 5.
var bitWidths = [numParams]int{
	8, 9, 9, // split VQ LSP
	8, 14, 1, 1, 6, // subframe 1
	5, 14, 1, 1, 6, // subframe 2
	5, 14, 1, 1, 6, // subframe 3
	5, 14, 1, 1, 6, // subframe 4
}

// speechFrameBits is the total number of coded bits across the 23 indices (the
// 137-bit ACELP speech frame produced by the TCH/S channel decoder).
const speechFrameBits = 137

// binToInt reads n bits MSB-first from bits (bit-per-byte, values 0/1) into an
// unsigned integer, mirroring the reference bin2int.
func binToInt(bits []byte, n int) int16 {
	var v int16
	for i := 0; i < n; i++ {
		v <<= 1
		if bits[i]&1 != 0 {
			v |= 1
		}
	}
	return v
}

// intToBin writes the low n bits of v MSB-first into dst (bit-per-byte). Inverse
// of binToInt; used by the round-trip test and any re-encode path.
func intToBin(dst []byte, v int16, n int) {
	for i := 0; i < n; i++ {
		dst[i] = byte((v >> uint(n-1-i)) & 1)
	}
}

// bits2prm unpacks a 137-bit speech frame (bit-per-byte, MSB-first fields) plus
// a bad-frame flag into the 24-word parameter array. bfi=true marks the frame
// erased; the indices are still unpacked so the caller can fall back cleanly.
func bits2prm(bits []byte, bfi bool) [numFields]int16 {
	var prm [numFields]int16
	if bfi {
		prm[pBFI] = 1
	}
	pos := 0
	for i := 0; i < numParams; i++ {
		w := bitWidths[i]
		prm[1+i] = binToInt(bits[pos:pos+w], w)
		pos += w
	}
	return prm
}

// prm2bits packs the 23 indices of a parameter array back into a 137-bit speech
// frame (bit-per-byte). Inverse of bits2prm (the BFI is not carried in-frame).
func prm2bits(prm [numFields]int16) []byte {
	bits := make([]byte, speechFrameBits)
	pos := 0
	for i := 0; i < numParams; i++ {
		w := bitWidths[i]
		intToBin(bits[pos:pos+w], prm[1+i], w)
		pos += w
	}
	return bits
}

// dLsp334 dequantises the three LSP split-VQ indices to 10 line spectral pairs
// in the cosine domain (Q15). It enforces a minimum spacing between the split
// boundaries (lsp[2]/lsp[3] and lsp[5]/lsp[6]) and, if the result is not a
// strictly decreasing (ordered) LSP set, discards it and returns lspOld — a
// clean-room port of the reference D_Lsp334.
func dLsp334(idx [3]int16, lspOld [lpcOrder]int16) [lpcOrder]int16 {
	var lsp [lpcOrder]int16

	p := int(idx[0]) * 3
	lsp[0], lsp[1], lsp[2] = lspDico1[p], lspDico1[p+1], lspDico1[p+2]

	p = int(idx[1]) * 3
	lsp[3], lsp[4], lsp[5] = lspDico2[p], lspDico2[p+1], lspDico2[p+2]

	p = int(idx[2]) * 4
	lsp[6], lsp[7], lsp[8], lsp[9] = lspDico3[p], lspDico3[p+1], lspDico3[p+2], lspDico3[p+3]

	// Minimum distance between lsp[2] and lsp[3]: 917 = 0.028 in Q15 (~50 Hz
	// around 1000 Hz). If the split boundary is too close, spread them apart.
	temp := addOp(subOp(917, lsp[2]), lsp[3])
	if temp > 0 {
		temp = shr(temp, 1)
		lsp[2] = addOp(lsp[2], temp)
		lsp[3] = subOp(lsp[3], temp)
	}

	// Minimum distance between lsp[5] and lsp[6]: 1245 = 0.038 in Q15.
	temp = addOp(subOp(1245, lsp[5]), lsp[6])
	if temp > 0 {
		temp = shr(temp, 1)
		lsp[5] = addOp(lsp[5], temp)
		lsp[6] = subOp(lsp[6], temp)
	}

	// LSPs in the cosine domain must be strictly decreasing. If any pair is out
	// of order, keep the previous frame's ordered set.
	ordered := true
	for i := 0; i < lpcOrder-1; i++ {
		if subOp(lsp[i], lsp[i+1]) <= 0 {
			ordered = false
			break
		}
	}
	if !ordered {
		return lspOld
	}
	return lsp
}
