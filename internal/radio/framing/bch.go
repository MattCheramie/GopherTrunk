package framing

// BCH(63,16,11) is the binary BCH code that protects the P25 Phase 1
// Network ID (NID) field on every transmitted frame. The code carries 16
// information bits in a 63-bit codeword (47 parity bits) and corrects up
// to 11 bit errors per codeword. P25 then appends a single bit (a fixed
// per-DUID value, NOT an overall parity — see internal/radio/p25/phase1
// nid.go for that lookup) to form the 64-bit NID actually transmitted
// on-air.
//
// Generator polynomial g(x), TIA-102.BAAA Annex A.4 — the product of
// the binary minimal polynomials of α, α^3, α^5, …, α^21 over
// GF(2^6) (primitive polynomial p(x) = x^6 + x + 1):
//
//	g(x) = x^47 + x^46 + x^43 + x^42 + x^40 + x^39 + x^36 + x^33
//	     + x^32 + x^27 + x^25 + x^24 + x^23 + x^22 + x^20 + x^19
//	     + x^18 + x^16 + x^13 + x^12 + x^11 + x^9  + x^8  + x^5
//	     + x^3  + x    + 1
//
// Codewords are stored systematically with the 16 information bits in
// positions 62..47 (bit 62 = MSB of info, bit 47 = LSB) and the 47
// parity bits in positions 46..0. We pack the 63-bit codeword into the
// low bits of a uint64 with bit 0 = lowest-order x^0 coefficient.
//
// Issue #275 / PR-after: the original constant here was wrong (off by
// 10 exponents — see bch_test.go), so encoded codewords had non-zero
// syndromes under the spec α^i evaluation and real on-air NIDs never
// decoded. Our synthetic-modulator tests passed only because the
// encoder and decoder both used the same wrong polynomial.

const bch6316Generator uint64 = 0xCD930BDD3B2B

// BCHEncode63_16 encodes 16 information bits into a 63-bit BCH codeword.
// Only the low 16 bits of data are used. The result occupies the low 63
// bits of the returned uint64.
func BCHEncode63_16(data uint16) uint64 {
	info := uint64(data) & 0xFFFF
	rem := info << 47
	for i := 62; i >= 47; i-- {
		if rem&(uint64(1)<<uint(i)) != 0 {
			rem ^= bch6316Generator << uint(i-47)
		}
	}
	return (info << 47) | (rem & ((uint64(1) << 47) - 1))
}

// BCHDecode63_16 decodes a 63-bit BCH codeword via the algebraic
// syndrome → Berlekamp-Massey → Chien decoder (see bch_decode.go).
// Returns (data, errors) where errors is the bit-error count corrected,
// or -1 if the word is more than 11 bit errors from any codeword
// (uncorrectable). On the -1 path data is unspecified (zero) — callers
// must treat the result as trustworthy only when errors >= 0.
func BCHDecode63_16(cw uint64) (uint16, int) {
	cw &= (uint64(1) << 63) - 1

	// Fast path: a zero-error codeword. The code is systematic with the
	// 16 info bits in positions 62..47, so a clean codeword decodes to
	// its own high bits — re-encode those and compare. This is the
	// dominant case on a locked signal and skips the syndrome work.
	info := uint16(cw >> 47)
	if BCHEncode63_16(info) == cw {
		return info, 0
	}

	synd, allZero := bchSyndromes6316(cw)
	if allZero {
		// No syndrome error but the fast path missed it — defensive; a
		// zero-syndrome word is a valid codeword, so trust its info bits.
		return info, 0
	}

	lambda, L := bchBerlekampMassey(synd)
	if L < 1 || L > bch6316T {
		return 0, -1
	}
	pos := bchChienSearch(lambda, L)
	if len(pos) != L {
		// Fewer roots than the locator degree ⇒ the error pattern is not
		// genuinely correctable (more than t errors).
		return 0, -1
	}

	corrected := cw
	for _, p := range pos {
		corrected ^= uint64(1) << uint(p)
	}
	// Self-check: the corrected word must be a valid codeword. This guards
	// against any convention slip in the algebraic stages — a mismatch
	// degrades safely to "uncorrectable" rather than a wrong trusted decode.
	data := uint16(corrected >> 47)
	if BCHEncode63_16(data) != corrected {
		return 0, -1
	}
	return data, L
}

// BCH6316ParityBit returns the even-parity bit over the 63 codeword
// bits — the trailing bit P25 appends to form the 64-bit NID.
func BCH6316ParityBit(cw uint64) byte {
	return byte(PopCount64(cw&((uint64(1)<<63)-1)) & 1)
}

// BCHEncode64_16 encodes 16 information bits into a 64-bit BCH
// codeword: the 63-bit BCH(63,16,11) codeword from BCHEncode63_16
// plus a trailing overall-even-parity bit. This is the FEC layer
// used by Motorola Type II / SmartZone control-channel OSWs (each
// OSW frame carries two BCH(64,16,11) codewords concatenated; two
// codewords' 16 info bits each combine into the OSW's 32-bit
// {Address, Command} field).
//
// Bit 0 of the returned uint64 is the parity bit; bits 1..63 are
// the 63-bit BCH codeword (info in the high 16 bits of that
// range, parity in the low 47). Encoders that want the canonical
// MSB-first 64-bit wire representation should pack [parity, bch63
// high → low].
func BCHEncode64_16(data uint16) uint64 {
	cw63 := BCHEncode63_16(data)
	parity := uint64(BCH6316ParityBit(cw63))
	return (cw63 << 1) | parity
}

// BCHDecode64_16 decodes a 64-bit BCH codeword by extracting the
// 63-bit BCH(63,16,11) codeword (top 63 bits) and the trailing
// parity bit, running BCHDecode63_16 on the 63-bit codeword, and
// reporting the corrected info + total error count.
//
// Returns (data, errors) where errors is the bit-error count
// corrected. errors = -1 means uncorrectable (the inner BCH
// decoder reported > 11 errors); the returned data is the closest
// codeword's info but should not be trusted.
//
// The trailing parity bit is included in the error count: if the
// 63-bit decode reports E errors AND the recomputed parity over
// the corrected 63-bit codeword doesn't match the received
// parity, the total is E + 1 (still uncorrectable if E > 10).
func BCHDecode64_16(cw uint64) (uint16, int) {
	parity := byte(cw & 1)
	cw63 := (cw >> 1) & ((uint64(1) << 63) - 1)
	data, errs := BCHDecode63_16(cw63)
	if errs < 0 {
		return data, -1
	}
	// Recompute parity over the corrected 63-bit codeword.
	gotParity := BCH6316ParityBit(BCHEncode63_16(data))
	if gotParity != parity {
		errs++
		if errs > 11 {
			return data, -1
		}
	}
	return data, errs
}
