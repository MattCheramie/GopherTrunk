package phase1

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// DUID is the 4-bit Data Unit ID that identifies the kind of frame
// following the NID (TIA-102.BAAA §6.2).
type DUID uint8

const (
	DUIDHeader            DUID = 0x0 // HDU
	DUIDTerminator        DUID = 0x3 // TDU (without LC)
	DUIDLogicalLink1      DUID = 0x5 // LDU1
	DUIDTrunkingSignaling DUID = 0x7 // TSDU (control channel)
	DUIDLogicalLink2      DUID = 0xA // LDU2
	DUIDPacketDataUnit    DUID = 0xC // PDU
	DUIDTerminatorWithLC  DUID = 0xF // TDULC
)

func (d DUID) String() string {
	switch d {
	case DUIDHeader:
		return "HDU"
	case DUIDTerminator:
		return "TDU"
	case DUIDLogicalLink1:
		return "LDU1"
	case DUIDTrunkingSignaling:
		return "TSDU"
	case DUIDLogicalLink2:
		return "LDU2"
	case DUIDPacketDataUnit:
		return "PDU"
	case DUIDTerminatorWithLC:
		return "TDULC"
	default:
		return fmt.Sprintf("DUID(%X)", uint8(d))
	}
}

// NID is the 64-bit Network ID immediately following the FSW. Bits 0..11
// are the NAC (Network Access Code), bits 12..15 are the DUID, bits
// 16..62 are the BCH(63,16,11) parity field, and bit 63 is a fixed
// per-DUID flag (see expectedNIDParity).
type NID struct {
	NAC  uint16
	DUID DUID
}

// ErrNIDUncorrectable is returned by ParseNID when the BCH decoder
// cannot recover the codeword within its t=11 error-correction radius.
var ErrNIDUncorrectable = errors.New("p25/phase1: NID BCH uncorrectable")

// ErrNIDReservedDUID reports a BCH-corrected NID whose DUID is one of the
// nine values no P25 Phase 1 frame carries (1, 2, 4, 6, 8, 9, B, D, E).
// A codeword that corrects to one of them is a mis-aligned or noise
// hypothesis the BCH happened to close on, not a frame: on a weak
// control channel 22% of the NIDs the search accepted carried one, and
// each cost a block's worth of trellis work downstream before the CRC
// threw it out. Treated as uncorrectable.
var ErrNIDReservedDUID = errors.New("p25/phase1: NID carries a reserved DUID")

// Valid reports whether d is a DUID a P25 Phase 1 transmitter emits:
// HDU, TDU, LDU1, TSDU, LDU2, PDU or TDULC.
func (d DUID) Valid() bool {
	switch d {
	case DUIDHeader, DUIDTerminator, DUIDLogicalLink1, DUIDTrunkingSignaling,
		DUIDLogicalLink2, DUIDPacketDataUnit, DUIDTerminatorWithLC:
		return true
	}
	return false
}

// ErrNIDParity is returned by ParseNID when the BCH decoder accepted a
// codeword but the trailing NID flag bit disagrees with the
// DUID-dictated value (see expectedNIDParity). Treated as uncorrectable.
var ErrNIDParity = errors.New("p25/phase1: NID parity mismatch")

// expectedNIDParity returns the value of the 64th NID bit a P25 Phase 1
// transmitter sets for a given DUID — NOT an overall parity over the
// 63-bit codeword (the obvious-but-wrong assumption that masked issue
// #275 for as long as it did). Per TIA-102.BAAA Annex A and confirmed
// against the OP25 reference implementation, the bit is a fixed flag:
//
//   - 0 for HDU (0), TDU (3), TSDU (7), PDU (12), TDULC (15)
//   - 1 for LDU1 (5), LDU2 (10)
//
// Reserved DUIDs never reach here (ParseNID rejects them first); 0 is
// returned for completeness.
func expectedNIDParity(duid DUID) byte {
	switch duid {
	case DUIDLogicalLink1, DUIDLogicalLink2:
		return 1
	default:
		return 0
	}
}

// ParseNID extracts the NAC and DUID from 64 received bits (MSB-first),
// running BCH(63,16,11) error correction over the first 63 bits and
// validating the trailing per-DUID flag bit. Returns the corrected NID,
// the number of bit errors corrected (0 on a clean codeword), and a
// non-nil error if the codeword is uncorrectable.
func ParseNID(bits []byte) (NID, int, error) {
	if len(bits) < 64 {
		return NID{}, -1, fmt.Errorf("p25/phase1: NID requires 64 bits, got %d", len(bits))
	}
	var cw uint64
	for i := 0; i < 63; i++ {
		if bits[i]&1 != 0 {
			cw |= uint64(1) << uint(62-i)
		}
	}
	rxParity := bits[63] & 1

	data, errs := framing.BCHDecode63_16(cw)
	if errs < 0 {
		return NID{}, -1, ErrNIDUncorrectable
	}
	duid := DUID(data & 0xF)
	if !duid.Valid() {
		return NID{}, errs, ErrNIDReservedDUID
	}
	if expectedNIDParity(duid) != rxParity {
		return NID{}, errs, ErrNIDParity
	}
	nac := uint16((data >> 4) & 0xFFF)
	return NID{NAC: nac, DUID: duid}, errs, nil
}

// NIDFromDibits is a convenience wrapper that takes 32 dibits (= 64 bits).
func NIDFromDibits(dibits []uint8) (NID, int, error) {
	nid, errs, _, err := NIDFromDibitsWithErrors(dibits)
	return nid, errs, err
}

// NIDFromDibitsWithErrors is identical to NIDFromDibits but additionally
// returns a 32-entry per-dibit bit-error count of the received NID
// against the BCH-corrected codeword (with the corrected parity bit
// folded into dibit 31). Each entry is 0, 1, or 2. The pattern is
// always returned; on an uncorrectable codeword it is the zero array
// (no comparison was possible). On ErrNIDParity the pattern reflects
// the parity bit's contribution to dibit 31.
//
// The searchNID closest-miss diag uses this to surface *where* in the
// NID the residual bit errors cluster — distinguishing post-FSW timing
// slip (errors at one end of the NID), a status-symbol-phase fault
// (errors clustered around the tail dibits), and SNR-limited demod
// corruption (errors distributed across all 32 dibits). Issue #275.
func NIDFromDibitsWithErrors(dibits []uint8) (NID, int, [32]uint8, error) {
	var pattern [32]uint8
	if len(dibits) < 32 {
		return NID{}, -1, pattern, fmt.Errorf("p25/phase1: NID requires 32 dibits, got %d", len(dibits))
	}
	bits := make([]byte, 64)
	for i := 0; i < 32; i++ {
		bits[2*i] = (dibits[i] >> 1) & 1
		bits[2*i+1] = dibits[i] & 1
	}
	var cw uint64
	for i := 0; i < 63; i++ {
		if bits[i]&1 != 0 {
			cw |= uint64(1) << uint(62-i)
		}
	}
	rxParity := bits[63] & 1

	data, errs := framing.BCHDecode63_16(cw)
	if errs < 0 {
		return NID{}, -1, pattern, ErrNIDUncorrectable
	}
	corrected := framing.BCHEncode63_16(data)
	duid := DUID(data & 0xF)
	correctedParity := expectedNIDParity(duid)

	// Per-dibit error count: bit i of the received codeword vs bit i of
	// the BCH-corrected codeword, with the trailing per-DUID flag bit
	// (bit 63) folded into dibit 31.
	for i := 0; i < 63; i++ {
		var corBit byte
		if corrected&(uint64(1)<<uint(62-i)) != 0 {
			corBit = 1
		}
		if bits[i] != corBit {
			pattern[i/2]++
		}
	}
	if rxParity != correctedParity {
		pattern[31]++
	}

	if !duid.Valid() {
		return NID{}, errs, pattern, ErrNIDReservedDUID
	}
	if correctedParity != rxParity {
		return NID{}, errs, pattern, ErrNIDParity
	}
	nac := uint16((data >> 4) & 0xFFF)
	return NID{NAC: nac, DUID: duid}, errs, pattern, nil
}

// EncodeNIDBits builds the 64 transmitted NID bits (MSB-first) for a
// given NAC + DUID. Useful for tests and synthetic streams.
func EncodeNIDBits(nac uint16, duid DUID) []byte {
	info := (uint16(nac&0x0FFF) << 4) | uint16(uint8(duid)&0x0F)
	cw := framing.BCHEncode63_16(info)
	bits := make([]byte, 64)
	for i := 0; i < 63; i++ {
		if cw&(uint64(1)<<uint(62-i)) != 0 {
			bits[i] = 1
		}
	}
	bits[63] = expectedNIDParity(duid)
	return bits
}
