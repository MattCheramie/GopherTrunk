package edacs

import (
	"errors"
	"fmt"
)

// CCW is one EDACS Control Channel Word — a 40-bit information block
// that rides every EDACS control-channel slot after sync detection.
// Under SetBCHMode(BCHOn) the 40 on-wire bits are the BCH(40, 28, 2)
// codeword (28 info high, 12 parity low) and the parse logic decodes
// + corrects up to 2 bit errors before reading the fields. The
// BCH layer is the only on-wire FEC on the Standard EDACS CCW per
// the lwvmobile/edacs-fm reference.
//
// Field layout follows the most-cited public reference:
//
//   bits 39..36 (4 bits)  Command  — operation type (voice grant,
//                                    data grant, system, idle, ...).
//   bits 35..32 (4 bits)  Status   — per-command flag bits
//                                    (encryption, emergency, etc.).
//   bits 31..16 (16 bits) Address  — talkgroup / radio ID / site ID,
//                                    interpretation depends on
//                                    Command.
//   bits 15..11 (5 bits)  LCN      — logical channel number; voice
//                                    grants reference the band plan.
//   bits 10..0  (11 bits) Aux      — command-specific auxiliary
//                                    parameter.
//
// Higher-level helpers in opcodes.go interpret these fields per
// command so callers don't need to touch the bit packing directly.
type CCW struct {
	Command Command
	Status  uint8  // 4-bit
	Address uint16 // 16-bit
	LCN     uint8  // 5-bit
	Aux     uint16 // 11-bit
}

// AssembleCCW packs a CCW into 5 bytes (40 bits) MSB-first. Used by
// tests and for any future encoder work.
func AssembleCCW(c CCW) []byte {
	cmd := uint64(c.Command) & 0xF
	status := uint64(c.Status) & 0xF
	addr := uint64(c.Address) & 0xFFFF
	lcn := uint64(c.LCN) & 0x1F
	aux := uint64(c.Aux) & 0x7FF
	v := cmd<<36 | status<<32 | addr<<16 | lcn<<11 | aux
	return []byte{
		byte((v >> 32) & 0xFF),
		byte((v >> 24) & 0xFF),
		byte((v >> 16) & 0xFF),
		byte((v >> 8) & 0xFF),
		byte(v & 0xFF),
	}
}

// ParseCCW reads 5 bytes (40 bits MSB-first) into a CCW.
func ParseCCW(info []byte) (CCW, error) {
	if len(info) != 5 {
		return CCW{}, fmt.Errorf("edacs: CCW info must be 5 bytes, got %d", len(info))
	}
	v := uint64(info[0])<<32 | uint64(info[1])<<24 | uint64(info[2])<<16 | uint64(info[3])<<8 | uint64(info[4])
	return CCW{
		Command: Command((v >> 36) & 0xF),
		Status:  uint8((v >> 32) & 0xF),
		Address: uint16((v >> 16) & 0xFFFF),
		LCN:     uint8((v >> 11) & 0x1F),
		Aux:     uint16(v & 0x7FF),
	}, nil
}

// CCWFromBits packs 40 MSB-first bits (each entry 0/1) into a CCW.
func CCWFromBits(bits []byte) (CCW, error) {
	if len(bits) != 40 {
		return CCW{}, errors.New("edacs: CCW requires 40 bits")
	}
	info := make([]byte, 5)
	for i := 0; i < 40; i++ {
		if bits[i]&1 != 0 {
			info[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return ParseCCW(info)
}

// CCWBits returns the 40 MSB-first bits of a CCW.
func CCWBits(c CCW) []byte {
	bytes := AssembleCCW(c)
	out := make([]byte, 40)
	for i := 0; i < 40; i++ {
		if bytes[i>>3]&(1<<uint(7-(i&7))) != 0 {
			out[i] = 1
		}
	}
	return out
}
