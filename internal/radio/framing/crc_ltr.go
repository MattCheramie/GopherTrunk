package framing

// LTR Standard message check (CRC-7).
//
// Per the most-cited public reference (DSheirer/sdrtrunk's
// CRCLTR.java), LTR Standard protects a 24-bit message field with
// a CRC-7 computed by XORing precomputed syndrome contributions
// from a 24-entry lookup table:
//
//	polynomial: 0xFD  (= x^7 + x^6 + x^5 + x^4 + x^3 + x^2 + 1
//	                     with the explicit x^7 leading term)
//	initial fill: 0x00
//
// The 24 message bits cover the four LTR fields the CRC protects:
//
//	Area     (1 bit)   — sdrtrunk's table[0]
//	Channel  (5 bits)  — table[1..5]   (MSB-first: Channel 4..0)
//	Home     (5 bits)  — table[6..10]  (MSB-first: Home 4..0)
//	Group    (8 bits)  — table[11..18] (MSB-first: Group 7..0)
//	Free     (5 bits)  — table[19..23] (MSB-first: Free 4..0)
//	= 24 message bits
//
// Note: sdrtrunk's "Area (1 bit)" reading is one of several LTR
// Standard interpretations in circulation. GopherTrunk's existing
// 41-bit Status struct models Area as 5 bits and Channel as 4 bits
// (different convention); reconciling those layouts before wiring
// this primitive into the LTR adapter is the documented follow-up.

// crc7LTRChecksums is the 24-entry syndrome lookup table copied
// from sdrtrunk's CRCLTR.java. Each entry is the 7-bit CRC
// contribution of one message bit set.
var crc7LTRChecksums = [24]byte{
	0x38, // Area
	0x1C, // Channel 4 (MSB)
	0x0E, // Channel 3
	0x46, // Channel 2
	0x23, // Channel 1
	0x51, // Channel 0 (LSB)
	0x68, // Home 4 (MSB)
	0x75, // Home 3
	0x7A, // Home 2
	0x3D, // Home 1
	0x1F, // Home 0 (LSB)
	0x4F, // Group 7 (MSB)
	0x26, // Group 6
	0x52, // Group 5
	0x29, // Group 4
	0x15, // Group 3
	0x0B, // Group 2
	0x45, // Group 1
	0x62, // Group 0 (LSB)
	0x31, // Free 4 (MSB)
	0x19, // Free 3
	0x0D, // Free 2
	0x07, // Free 1
	0x43, // Free 0 (LSB)
}

// CRC7LTR computes the 7-bit LTR Standard message CRC over a
// 24-bit message field. Each entry of msgBits is the value 0 or 1;
// msgBits[0] is the Area bit, msgBits[1..5] are the 5 Channel bits
// MSB-first, and so on per the field layout documented above.
//
// The returned checksum occupies the low 7 bits of a byte (the
// high bit is always zero). Pass the same value the LTR
// transmitter computed at encode time; verify that
// CRC7LTR(receivedMsg) == receivedChecksum.
func CRC7LTR(msgBits []byte) byte {
	var crc byte
	n := len(msgBits)
	if n > len(crc7LTRChecksums) {
		n = len(crc7LTRChecksums)
	}
	for i := 0; i < n; i++ {
		if msgBits[i]&1 != 0 {
			crc ^= crc7LTRChecksums[i]
		}
	}
	return crc & 0x7F
}

// VerifyCRC7LTR reports whether the 7-bit checksum for the
// supplied 24-bit message matches the expected value. The OSW
// (outbound from base) variant requires `calculated ==
// transmitted`; the ISW (inbound from subscriber) variant
// requires `calculated ^ 0x7F == transmitted` (per sdrtrunk's
// CRCLTR.check direction-aware logic). Most decoders consume
// only OSW frames; pass false for inbound otherwise.
func VerifyCRC7LTR(msgBits []byte, transmitted byte, inbound bool) bool {
	calculated := CRC7LTR(msgBits)
	if inbound {
		return (calculated^0x7F)&0x7F == transmitted&0x7F
	}
	return calculated == transmitted&0x7F
}
