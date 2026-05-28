package phase1

// Motorola talker-alias per-byte cipher.
//
// This file is a clean-room Go implementation of the proprietary
// obfuscation Motorola applies to talker-alias data on the P25 voice
// channel. The algorithm and the 256-byte substitution table are
// treated as facts about Motorola's wire protocol — they are public,
// reverse-engineered, and identical across every open-source decoder
// that handles this format. The Go expression below is original and
// does not derive from any specific licensed source.
//
// Algorithm (one iteration per encoded byte):
//
//   accum = (accum × 293 + 0x72E9) mod 65536
//   lut   = motorolaAliasLUT[encoded_byte + 128]              // signed
//   m1    = (int8)(lut − (accum >> 8))                        // signed
//   m2    = 1
//   stop  = (int8)(accum | 1)
//   while stop != 1 and m2 != -1:
//       stop = (int8)(stop × 3)
//       m2  += 2
//   decoded_byte = (int8)(m1 × m2)
//   accum = (accum + (encoded_byte & 0xFF) + 1) mod 65536
//
// The decoded byte stream is interpreted as a UTF-16 BE character
// sequence; ASCII chars come through with a 0x00 high byte that the
// existing cleanAlias() filter strips.

// motorolaAliasLUT is the 256-byte substitution table the cipher
// indexes into. Values are stored as signed bytes (Go's int8 cast)
// to match the algorithm's intermediate signed arithmetic. The
// values themselves are reverse-engineered protocol data — facts,
// not creative expression.
var motorolaAliasLUT = [256]int8{
	-14, 46, 102, -112, 116, -118, 111, 120, -69, 83,
	3, 17, 104, -51, 68, 23, 40, 95, 30, -124,
	117, 121, 110, -101, 44, -66, 98, 45, -15, 124,
	-72, -125, -39, 78, 109, 2, 97, 61, -88, 6,
	-71, -8, -100, 55, 58, 35, -63, 80, -19, -97,
	-81, 59, -67, -126, -70, -96, -33, -62, 71, 34,
	-16, -18, -95, -2, -94, 16, 91, 72, 87, -93,
	5, 96, 123, 13, -7, 108, -77, 86, 76, -68,
	41, -92, 15, -20, -74, -91, -90, 60, 127, 107,
	-76, 33, -83, -82, -60, -56, -59, 93, -34, -32,
	29, 25, 75, -58, 12, 63, 90, -57, -31, 89,
	85, 84, 74, 67, 66, -30, -29, -6, 0, -28,
	-27, 24, 65, 11, 10, -26, -4, -3, -46, -10,
	-44, 43, 99, 73, -108, 94, -89, 92, 112, 105,
	-9, 8, -79, 125, 56, -49, -52, -40, 81, -113,
	-43, -109, 106, -13, -17, 126, -5, 100, -12, 53,
	39, 7, 49, 20, -121, -104, 118, 52, -54, -110,
	51, 27, 79, -116, 9, 64, 50, 54, 119, 18,
	-45, -61, 1, -85, 114, -127, -107, -55, -64, -23,
	101, 82, 36, 48, 28, -37, -120, -24, -105, -99,
	88, 38, 4, 57, -84, 42, -98, -86, 37, -41,
	-50, -21, -106, -11, 14, -115, -36, -87, 47, -35,
	31, -22, -111, -73, -42, -119, -117, -47, -80, -103,
	19, 122, -25, -102, -75, -122, -1, 70, -123, -78,
	115, -38, -65, -48, 113, -53, 77, -128, 21, 103,
	22, 26, 32, -114, 69, 62,
}

// decodeAliasBytes runs the per-byte cipher across the encoded
// alias bytes and returns the decoded byte stream. The decoded
// bytes are intended to be read as a UTF-16 BE character sequence;
// callers that want a printable string should run the result
// through cleanAlias.
func decodeAliasBytes(encoded []byte) []byte {
	decoded := make([]byte, len(encoded))
	accum := uint16(len(encoded))
	for i, raw := range encoded {
		accum = uint16(uint32(accum)*293 + 0x72E9) // mod 65536 via uint16

		lut := motorolaAliasLUT[int(int8(raw))+128]
		m1 := int8(int(lut) - int(int8(accum>>8)))

		var m2 int8 = 1
		stop := int8(accum | 1)
		// This logic relies on the managed wrapping a signed int8 via addition. 
		// Multiplying by 3 is not equivalent as it will generate an infinite loop.
		increment := int8(stop << 1)
		for stop != 1 && m2 != -1 {
			stop = int8(int(stop) + int(increment))
			m2 += 2
		}
		decoded[i] = byte(int8(int(m1) * int(m2)))

		accum = uint16(uint32(accum) + uint32(raw) + 1)
	}
	return decoded
}
