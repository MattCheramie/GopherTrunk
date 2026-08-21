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
// carrier own only its fragment transport and share the decode.
//
// Provenance and verification status (see #773): both the SUID framing
// and the per-byte CIPHER are now verified. The cipher was recovered by
// clean-room reverse engineering — chosen-ciphertext differential analysis
// of a behavioral oracle (an existing GPLv3 decoder driven as an OPAQUE
// decode function; its source and table were never read) — and validated
// at 100% against that oracle (1242/1242 held-out characters, plus 85382
// training characters). No third-party code or table was copied: the
// structure and the substitution table below were derived solely from
// observed input->output behavior, so this stays a clean-room, Apache-2.0
// implementation. The recovered cipher is a 16-bit additive accumulator
// seeded by the character count (see DecodeAliasBytes). GopherTrunk's
// earlier inferred structure — a multiplicative accumulator, an accum=len
// seed, and a different table — was wrong and decoded nothing; the oracle
// disproved it and pinned the correct law.
package motorola

// motorolaAliasLUT is the 256-byte output substitution table — a bijection
// over 0..255 — that the per-byte cipher indexes by the encoded byte value.
// Recovered by clean-room chosen-ciphertext differential analysis of a
// behavioral oracle and validated 100% against a held-out set (#773). The
// values are facts about the wire cipher derived from observed behavior; no
// table was copied.
var motorolaAliasLUT = [256]byte{
	0x21, 0x45, 0x23, 0x7a, 0xb2, 0x98, 0xe3, 0xad, 0xf6, 0xab, 0xbf, 0xb8, 0x46, 0x57, 0x00, 0xcc,
	0x87, 0x1e, 0x1b, 0x27, 0xa0, 0xde, 0x24, 0xe2, 0xb9, 0x42, 0x3e, 0xcd, 0x4a, 0xb3, 0x43, 0x84,
	0x76, 0x56, 0x80, 0x63, 0xd6, 0xe7, 0xc5, 0x83, 0x19, 0xe1, 0x82, 0x6a, 0x9e, 0xdb, 0x58, 0x8f,
	0x81, 0x85, 0xc6, 0x61, 0x22, 0x12, 0x50, 0xfa, 0xc1, 0xd0, 0xe4, 0x18, 0x0f, 0x38, 0xb4, 0xa1,
	0x73, 0x7f, 0x6b, 0x2a, 0xd7, 0x37, 0xe6, 0xec, 0xa7, 0x75, 0x53, 0x88, 0xfb, 0x79, 0xed, 0xf9,
	0x74, 0x26, 0x1d, 0x3a, 0xe5, 0x44, 0x5d, 0xdc, 0x2b, 0xf8, 0x7e, 0x2c, 0x6e, 0x39, 0xe0, 0x06,
	0x25, 0xd8, 0xda, 0x20, 0xff, 0xe8, 0x62, 0xc9, 0x36, 0xe9, 0x04, 0xd5, 0x4e, 0x95, 0xd4, 0x01,
	0xc2, 0x29, 0x0e, 0x1f, 0xc0, 0x1a, 0x9c, 0xcf, 0x64, 0xb6, 0x65, 0x69, 0x6f, 0xdd, 0x94, 0x8d,
	0x41, 0x7d, 0xb5, 0xdf, 0xc3, 0xd9, 0xbe, 0xc7, 0x0a, 0xa2, 0x52, 0x60, 0xb7, 0x1c, 0x93, 0x66,
	0x77, 0xae, 0x6d, 0xd3, 0xc4, 0xc8, 0xbd, 0xea, 0x7b, 0x0d, 0xb1, 0x7c, 0x40, 0xcb, 0x07, 0xd2,
	0x28, 0x9d, 0xbc, 0x51, 0xb0, 0x8c, 0xf7, 0x55, 0x08, 0x47, 0xeb, 0x86, 0x89, 0x72, 0x10, 0x9f,
	0x3c, 0xee, 0xfe, 0x8a, 0x0c, 0xd1, 0x09, 0xef, 0x2e, 0x11, 0x96, 0x71, 0x3f, 0x3d, 0xf0, 0x4d,
	0xf1, 0x5f, 0xaa, 0x97, 0xa6, 0xf2, 0x54, 0xaf, 0xca, 0x5c, 0x48, 0xbb, 0x02, 0xa5, 0x9b, 0x0b,
	0x78, 0xf3, 0x5e, 0x3b, 0x05, 0xf4, 0xf5, 0x8b, 0xce, 0xba, 0x03, 0x70, 0xfc, 0xfd, 0x13, 0x17,
	0x14, 0xac, 0x2d, 0x2f, 0x6c, 0x68, 0x9a, 0x15, 0x5b, 0x8e, 0xa9, 0x16, 0x30, 0xa8, 0xa4, 0xa3,
	0x99, 0x92, 0x91, 0x31, 0x32, 0x49, 0x4f, 0x33, 0x34, 0x67, 0x90, 0x5a, 0x59, 0x35, 0x4b, 0x4c,
}

// motorolaAliasInvOdd[x] is the multiplicative inverse of x mod 256 for odd
// x (and 0 for even x, which never occurs — the accumulator low byte is
// forced odd by the |1 in DecodeAliasBytes).
var motorolaAliasInvOdd [256]byte

func init() {
	for a := 1; a < 256; a += 2 {
		for x := 1; x < 256; x += 2 {
			if byte(a*x) == 1 {
				motorolaAliasInvOdd[a] = byte(x)
				break
			}
		}
	}
}

// CipherVerified reports whether the per-byte alias cipher (the
// motorolaAliasLUT table plus the accumulator constants in
// DecodeAliasBytes) has been validated against ground truth — a
// captured frame whose decoded alias matches a known plaintext for the
// same RID.
//
// It is true. The cipher was recovered by clean-room reverse engineering
// (chosen-ciphertext differential analysis of a behavioral oracle — an
// existing decoder driven as an opaque function, its source never read) and
// reproduces the oracle's output at 100% on a held-out set (1242/1242
// characters), pinned by the fixtures in alias_test.go (#773). With this
// true, DecodeMessage reports a clean-ASCII alias as reliable so the decoded
// name surfaces to callers.
const CipherVerified = true

// DecodeAliasBytes runs the recovered per-byte alias cipher over the encoded
// ciphertext bytes and returns the decoded byte stream as UTF-16BE (two
// bytes per character); callers that want a printable string run the result
// through DecodeAlias/CleanAlias. A trailing unpaired encoded byte is dropped
// (odd input length).
//
// The cipher is a single 16-bit additive accumulator W, seeded by the
// character count n = len(encoded)/2. Per processed byte i (byte arithmetic
// wraps mod 256; W wraps mod 2^16):
//
//	W       = 0xC433 + 586*(n-1)                             // seed; 586 = 2·293
//	out[i]  = invOdd[loByte(W)|1] · (LUT[enc[i]] − hiByte(W)) // per-byte sub
//	W      += 293·(enc[i] + 1)                                // accumulator step
//
// Character k packs to (out[2k] high, out[2k+1] low), with the high byte
// forced to 0xFF when the low byte is a negative int8 (≥0x80). Recovered by
// clean-room oracle analysis, validated 100% on a held-out set (#773).
func DecodeAliasBytes(encoded []byte) []byte {
	n := len(encoded) / 2
	if n == 0 {
		return []byte{}
	}
	w := uint16(0xC433 + 586*(n-1))
	out := make([]byte, 2*n)
	for i := 0; i < 2*n; i++ {
		out[i] = motorolaAliasInvOdd[byte(w)|1] * (motorolaAliasLUT[encoded[i]] - byte(w>>8))
		w += uint16(293 * (int(encoded[i]) + 1))
	}
	decoded := make([]byte, 0, 2*n)
	for k := 0; k < n; k++ {
		lo, hi := out[2*k+1], out[2*k]
		if lo >= 0x80 {
			hi = 0xFF
		}
		decoded = append(decoded, hi, lo)
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
	// Encoded is the per-byte-cipher region — the exactly 2n encoded-alias
	// bytes that the cipher operates on, with the SUID prefix, the trailing
	// CRC-16, and any FACCH block zero-padding stripped. The cipher length is
	// recovered by the self-delimiting CRC (see DecodeMessage), confirmed
	// against the #376 real capture. Callers may still log it as known-RID
	// ground truth; it is the same shape the cryptolab `alias sweep` solver
	// consumes (pure cipher output, no CRC).
	Encoded []byte
	// AliasReliable reports whether the decoded alias can be trusted as a
	// confirmed name. It is true only when the decode is all printable
	// ASCII (no bit-error corruption hallmarks, #711) AND the cipher is
	// verified (CipherVerified). The cipher is verified, so this tracks the
	// clean-ASCII flag; the gate remains so a future regression that unsets
	// CipherVerified would immediately stop surfacing names (#773). Callers
	// should surface an unreliable alias as suspect, not a confirmed name.
	AliasReliable bool
	// CRCOK reports whether the trailing CRC-16/GSM matched. The CRC-16/GSM
	// parameters and their coverage (SUID + cipher) are now confirmed against
	// the #376 real capture — RID 200062's alias "CRIO 0062" carries CRC
	// 0x6A96, which crc16GSM(SUID+cipher) reproduces exactly. It stays
	// advisory to callers (a carrier whose CRC differs must not suppress a
	// valid alias), but a match is also what delimits the cipher region.
	CRCOK bool
}

// DecodeMessage parses a reassembled Motorola alias message — a packed
// byte stream (8 bits per byte, MSB-first), the same shape phase1's
// assembleMotorolaBitStream and phase2's assembleMotorolaAliasMessage
// produce — into its SUID + decoded alias. It returns (zero, false) when
// the stream is too short to contain a SUID, a CRC, and at least one alias
// character. The alias may still be empty if the decoded bytes are all
// non-printable.
//
// The on-air frame is SUID | cipher(2n) | CRC-16/GSM | zero-pad, where the
// trailing pad is FACCH block filler carried along by reassembly. The cipher
// length is not framed explicitly, so it is recovered from the self-delimiting
// CRC: the cipher region is the shortest span whose trailing CRC-16/GSM (over
// SUID + cipher) matches and after which the frame is all zero. This both
// strips the pad and picks the length the cipher's length-seed needs — feeding
// the pad in as ciphertext corrupts that seed and decodes to noise. Confirmed
// against the #376 real capture: RID 200062 → "CRIO 0062", CRC 0x6A96, 6 pad
// bytes. When no CRC matches (a carrier whose CRC differs, or a corrupt frame),
// it falls back to treating the final two bytes as the CRC so a pad-free frame
// still decodes.
func DecodeMessage(msg []byte) (Message, bool) {
	const crcBytes = CRCBits / 8
	encStart := SUIDBits / 8 // SUID is exactly 7 bytes
	// Need at least SUID + one 2-byte character + CRC.
	if len(msg) < encStart+2+crcBytes {
		return Message{}, false
	}
	wacn, sysID, rid := decodeSUID(msg)

	// Recover the cipher length from the self-delimiting CRC. The cipher is
	// an even number of bytes (2 per UTF-16 character).
	maxCipher := len(msg) - encStart - crcBytes
	cipherLen, crcOK := -1, false
	for L := 2; L <= maxCipher; L += 2 {
		crcPos := encStart + L
		want := uint16(msg[crcPos])<<8 | uint16(msg[crcPos+1])
		if crc16GSM(msg[:crcPos]) != want {
			continue
		}
		// Everything after the CRC must be zero pad for L to be the true
		// boundary; a real CRC byte in that region rules out a shorter span.
		allZero := true
		for _, b := range msg[crcPos+crcBytes:] {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			cipherLen, crcOK = L, true
			break
		}
	}
	if cipherLen < 0 {
		// No self-consistent CRC: legacy framing — the final two bytes are
		// the CRC and everything between them and the SUID is ciphertext.
		cipherLen = maxCipher
		want := uint16(msg[encStart+cipherLen])<<8 | uint16(msg[encStart+cipherLen+1])
		crcOK = crc16GSM(msg[:encStart+cipherLen]) == want
	}

	encoded := msg[encStart : encStart+cipherLen]
	alias, aliasClean := DecodeAlias(DecodeAliasBytes(encoded))
	// Gate on CipherVerified so a possibly-wrong table can never surface a
	// coincidentally-clean decode as a confirmed name (#773). The cipher is
	// verified, so a clean-ASCII decode is reported reliable.
	aliasReliable := aliasClean && CipherVerified

	return Message{
		WACN:          wacn,
		SystemID:      sysID,
		RadioID:       rid,
		Alias:         alias,
		Encoded:       append([]byte(nil), encoded...),
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
