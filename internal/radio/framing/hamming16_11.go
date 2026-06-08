package framing

// Hamming(16,11,4): the DMR variable-length BPTC row code (ETSI TS 102
// 361-1 Annex B.2.2 / C, embedded-signalling Link Control). It is the
// (15,11,3) single-error-correcting Hamming code extended with one
// overall even-parity bit, raising the minimum distance to 4 (SEC-DED:
// it corrects any single-bit error and detects any double-bit error).
//
// Codeword layout (low 16 bits): bits 1..15 hold the 15-bit
// Hamming(15,11) codeword (data in bits 5..15, parity in bits 1..4 —
// i.e. the HammingEncode15_11 output shifted up by one), and bit 0 is
// the overall even parity computed over those 15 bits.

func parityLow15(v uint16) uint16 {
	v &= 0x7FFF
	v ^= v >> 8
	v ^= v >> 4
	v ^= v >> 2
	v ^= v >> 1
	return v & 1
}

// HammingEncode16_11 encodes 11 data bits (low 11 bits of input) into the
// 16-bit codeword.
func HammingEncode16_11(data uint16) uint16 {
	cw15 := HammingEncode15_11(data) & 0x7FFF
	return cw15<<1 | parityLow15(cw15)
}

// HammingDecode16_11 decodes a 16-bit codeword. Returns (data, errors):
// 0 = clean, 1 = a single-bit error was corrected (in the 15-bit body or
// the overall-parity bit), -1 = uncorrectable (double-bit error detected
// by the overall parity).
func HammingDecode16_11(cw uint16) (uint16, int) {
	cw &= 0xFFFF
	body := (cw >> 1) & 0x7FFF
	recvOverall := cw & 1
	// p is the overall-parity consistency of the *received* word: 0 when
	// the received overall bit matches the received body's parity.
	p := parityLow15(body) ^ recvOverall

	data, errs := HammingDecode15_11(body)
	switch {
	case errs < 0:
		return data, -1
	case errs == 0 && p == 0:
		// No Hamming syndrome and overall parity consistent → clean.
		return data, 0
	case errs == 1 && p == 1:
		// Hamming syndrome points at one body bit and overall parity is
		// inconsistent — the classic single-bit error. Corrected.
		return data, 1
	case errs == 0 && p == 1:
		// Body consistent but overall parity off → the error is in the
		// overall-parity bit itself; the recovered data is clean.
		return data, 1
	default: // errs == 1 && p == 0 — syndrome set but overall consistent
		// Two bit errors: SEC-DED detects but cannot correct.
		return data, -1
	}
}
