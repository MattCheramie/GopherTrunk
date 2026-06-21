// Package motorola holds the Motorola proprietary talker-alias decode
// shared by the P25 Phase 1 (voice-channel / TDULC link control) and
// Phase 2 (FACCH-S MAC) paths.
//
// All three on-air carriers — LDU1 link control, TDULC link control,
// and the Phase 2 FACCH-S MAC PDU — reassemble to the *same* message
// framing once their fragments are concatenated:
//
//	bits 0..19      : WACN
//	bits 20..31     : System ID
//	bits 32..55     : Radio (subscriber) ID
//	bits 56..end-16 : encoded alias bytes (proprietary per-byte cipher)
//	last 16 bits    : CRC-16/GSM over the preceding bits
//
// Keeping the cipher, the framing, and the CRC in one package lets each
// carrier own only its fragment transport and share the decode. The
// algorithm and 256-byte substitution table are reverse-engineered
// facts about Motorola's wire protocol (public, identical across every
// open-source decoder); the Go expression here is original.
package motorola

// motorolaAliasLUT is the 256-byte substitution table the per-byte
// cipher indexes into. Values are stored as signed bytes to match the
// algorithm's intermediate signed arithmetic.
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

// DecodeAliasBytes runs the per-byte cipher across the encoded alias
// bytes and returns the decoded byte stream. The decoded bytes are
// intended to be read as a UTF-16 BE character sequence; callers that
// want a printable string should run the result through CleanAlias.
//
//	accum = (accum × 293 + 0x72E9) mod 65536
//	lut   = motorolaAliasLUT[encoded_byte + 128]   (signed)
//	m1    = (int8)(lut − (accum >> 8))
//	stop  = (int8)(accum | 1); inc = (int8)(stop << 1); m2 = 1
//	while stop != 1 && m2 != -1: stop += inc; m2 += 2
//	decoded = (int8)(m1 × m2)
//	accum = (accum + encoded_byte + 1) mod 65536
func DecodeAliasBytes(encoded []byte) []byte {
	decoded := make([]byte, len(encoded))
	accum := uint16(len(encoded))
	for i, raw := range encoded {
		accum = uint16(uint32(accum)*293 + 0x72E9)

		lut := motorolaAliasLUT[int(int8(raw))+128]
		m1 := int8(int(lut) - int(int8(accum>>8)))

		var m2 int8 = 1
		stop := int8(accum | 1)
		increment := stop << 1
		for stop != 1 && m2 != -1 {
			stop += increment
			m2 += 2
		}
		decoded[i] = byte(int8(int(m1) * int(m2)))

		accum = uint16(uint32(accum) + uint32(raw) + 1)
	}
	return decoded
}

// DecodeAlias renders the decoded cipher output — a UTF-16 BE character
// sequence — as a printable-ASCII string and reports whether the decode
// looks reliable.
//
// reliable is false when the stream holds any character outside
// printable ASCII: a non-zero UTF-16 high byte, a control/DEL low byte,
// a non-NUL after the trailing NUL padding has started, or an odd
// trailing byte that can't form a UTF-16 unit. Those are the hallmark
// of bit-error corruption that survived the CRC (#711), so a caller
// should surface the result as suspect rather than as a confirmed name.
//
// The returned string is still the best-effort printable-ASCII rendering
// (corrupt characters dropped) so an operator has something to display
// alongside the unreliable flag. ASCII characters arrive as a 0x00 high
// byte plus the character; those high bytes and any trailing NUL padding
// are expected and do not make the decode unreliable.
func DecodeAlias(raw []byte) (alias string, reliable bool) {
	reliable = true
	out := make([]byte, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		hi, lo := raw[i], raw[i+1]
		if hi == 0 && lo == 0 {
			// NUL: expected only as trailing padding. Anything other
			// than NUL after it means the stream is corrupt.
			for j := i + 2; j+1 < len(raw); j += 2 {
				if raw[j] != 0 || raw[j+1] != 0 {
					reliable = false
					break
				}
			}
			break
		}
		if hi == 0 && lo >= 0x20 && lo < 0x7F {
			out = append(out, lo)
		} else {
			reliable = false
		}
	}
	// An odd trailing byte can't form a UTF-16 unit — treat as corruption.
	if len(raw)%2 == 1 {
		reliable = false
	}
	return string(out), reliable
}

// Message framing field sizes.
const (
	// SUIDBits = WACN(20) + System(12) + RadioID(24).
	SUIDBits = 56
	// CRCBits is the trailing CRC-16 field.
	CRCBits = 16
)

// Message is a fully reassembled Motorola alias message: the source
// SUID (WACN / System / Radio ID) followed by the decoded alias.
type Message struct {
	WACN     uint32
	SystemID uint16
	RadioID  uint32
	Alias    string
	// AliasReliable reports whether the decoded alias is all printable
	// ASCII. When false the decode contains non-ASCII-printable
	// characters — a hallmark of bit-error corruption that survived the
	// CRC — and callers should surface the alias as suspect (#711).
	AliasReliable bool
	// CRCOK reports whether the trailing CRC-16/GSM matched. It is
	// advisory: the CRC parameters are inferred from open-source
	// decoders and not yet confirmed against a committed real-frame
	// fixture (#376), so callers log it rather than gate on it — a
	// wrong polynomial must not suppress a valid alias.
	CRCOK bool
}

// DecodeMessage parses a reassembled Motorola alias message — a packed
// byte stream (8 bits per byte, MSB-first), the same shape phase1's
// assembleMotorolaBitStream produces — into its SUID + decoded alias.
// It returns (zero, false) when the stream is too short to contain a
// SUID, a CRC, and at least one alias character. The alias may still be
// empty if the decoded bytes are all non-printable.
func DecodeMessage(msg []byte) (Message, bool) {
	totalBits := len(msg) * 8
	if totalBits < SUIDBits+CRCBits+16 {
		return Message{}, false
	}
	aliasBits := totalBits - SUIDBits - CRCBits
	if aliasBits%8 != 0 {
		aliasBits -= aliasBits % 8
	}
	encStart := SUIDBits / 8 // SUID is exactly 7 bytes
	encByteLen := aliasBits / 8
	if encStart+encByteLen > len(msg) {
		return Message{}, false
	}

	wacn, sysID, rid := decodeSUID(msg)
	encoded := msg[encStart : encStart+encByteLen]
	alias, aliasReliable := DecodeAlias(DecodeAliasBytes(encoded))

	// CRC-16/GSM over everything before the trailing 16-bit CRC field.
	crcBytes := CRCBits / 8
	body := msg[:len(msg)-crcBytes]
	want := uint16(msg[len(msg)-2])<<8 | uint16(msg[len(msg)-1])
	crcOK := crc16GSM(body) == want

	return Message{
		WACN:          wacn,
		SystemID:      sysID,
		RadioID:       rid,
		Alias:         alias,
		AliasReliable: aliasReliable,
		CRCOK:         crcOK,
	}, true
}

// decodeSUID reads WACN(20) + System(12) + RadioID(24) from the front
// of a packed byte stream.
func decodeSUID(msg []byte) (wacn uint32, sysID uint16, rid uint32) {
	read := func(bitOff, n int) uint32 {
		var v uint32
		for i := 0; i < n; i++ {
			pos := bitOff + i
			bit := (msg[pos/8] >> uint(7-pos%8)) & 1
			v = v<<1 | uint32(bit)
		}
		return v
	}
	wacn = read(0, 20)
	sysID = uint16(read(20, 12))
	rid = read(32, 24)
	return wacn, sysID, rid
}

// crc16GSM computes CRC-16/GSM (poly 0x1021, init 0x0000, xorout
// 0xFFFF, no reflection) over the message bytes. This matches the CRC
// SDRTrunk runs over the reassembled encoded alias to reject a garbled
// block sequence.
func crc16GSM(data []byte) uint16 {
	crc := uint16(0x0000)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc ^ 0xFFFF
}
