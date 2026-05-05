package motorola

import (
	"encoding/binary"
	"fmt"
)

// OSW is one Outbound Status Word — a 32-bit information block that
// rides every Motorola Type II / SmartZone control-channel frame
// after sync, BCH(64,16,11) error correction, and de-interleaving.
//
// Field layout follows the most-cited public reference: the upper 16
// bits carry an Address (talkgroup or radio ID, depending on opcode);
// the lower 16 bits carry a Command field that combines a 12-bit
// opcode with a 4-bit per-opcode parameter (typically the LCN for
// voice grants).
//
// The package's higher-level helpers (in opcodes.go) interpret the
// Command field per-opcode so callers don't need to know the
// bit-packing.
type OSW struct {
	Address uint16
	Command uint16
}

// AssembleOSW packs an OSW into 4 bytes (32 bits) MSB-first. Used by
// tests and for any future encoder work.
func AssembleOSW(o OSW) []byte {
	out := make([]byte, 4)
	binary.BigEndian.PutUint16(out[0:2], o.Address)
	binary.BigEndian.PutUint16(out[2:4], o.Command)
	return out
}

// ParseOSW reads 4 bytes (32 bits MSB-first) into an OSW.
func ParseOSW(info []byte) (OSW, error) {
	if len(info) != 4 {
		return OSW{}, fmt.Errorf("motorola: OSW info must be 4 bytes, got %d", len(info))
	}
	return OSW{
		Address: binary.BigEndian.Uint16(info[0:2]),
		Command: binary.BigEndian.Uint16(info[2:4]),
	}, nil
}

// OSWFromBits packs 32 MSB-first bits (each entry 0/1) into an OSW.
func OSWFromBits(bits []byte) (OSW, error) {
	if len(bits) != 32 {
		return OSW{}, fmt.Errorf("motorola: OSW requires 32 bits, got %d", len(bits))
	}
	info := make([]byte, 4)
	for i := 0; i < 32; i++ {
		if bits[i]&1 != 0 {
			info[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return ParseOSW(info)
}

// OSWBits returns the 32 MSB-first bits of an OSW.
func OSWBits(o OSW) []byte {
	bytes := AssembleOSW(o)
	out := make([]byte, 32)
	for i := 0; i < 32; i++ {
		if bytes[i>>3]&(1<<uint(7-(i&7))) != 0 {
			out[i] = 1
		}
	}
	return out
}
