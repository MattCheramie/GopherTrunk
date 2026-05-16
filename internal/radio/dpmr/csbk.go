package dpmr

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// CSBK (Common Signalling Block) is the 80-bit Mode 3 signalling unit
// the control channel transmits between voice grants. After FEC
// removal the structured layout is:
//
//	bits  0..4   (5)  Message type      (see opcodes.go)
//	bits  5..7   (3)  Format / flags    (reserved + emergency + group)
//	bits  8..31  (24) Source ID         (calling subscriber)
//	bits 32..55  (24) Destination ID    (callee — group or subscriber)
//	bits 56..63  (8)  Service info      (priority / payload type)
//	bits 64..79  (16) Opcode-specific   (channel number, status, …)
//
// Field layout follows ETSI TS 102 658 §6.5; vendor extensions
// repurpose Service info and the trailing 16 bits, so cross-check
// before trusting live captures.
type CSBK struct {
	Type        MessageType
	Flags       uint8  // 3-bit
	SourceID    uint32 // 24-bit
	DestID      uint32 // 24-bit
	ServiceInfo uint8
	Extra       uint16 // opcode-specific 16-bit field (e.g. channel number)
}

// AssembleCSBK packs a CSBK into 10 bytes (80 bits, MSB-first across
// the byte boundaries described above). Used by tests and by any
// future encoder work.
func AssembleCSBK(c CSBK) []byte {
	out := make([]byte, 10)
	out[0] = (uint8(c.Type)&0x1F)<<3 | (c.Flags & 0x07)
	out[1] = byte((c.SourceID >> 16) & 0xFF)
	out[2] = byte((c.SourceID >> 8) & 0xFF)
	out[3] = byte(c.SourceID & 0xFF)
	out[4] = byte((c.DestID >> 16) & 0xFF)
	out[5] = byte((c.DestID >> 8) & 0xFF)
	out[6] = byte(c.DestID & 0xFF)
	out[7] = c.ServiceInfo
	binary.BigEndian.PutUint16(out[8:10], c.Extra)
	return out
}

// ParseCSBK consumes 10 bytes (80 bits, as packed by AssembleCSBK)
// into a CSBK.
func ParseCSBK(info []byte) (CSBK, error) {
	if len(info) != 10 {
		return CSBK{}, fmt.Errorf("dpmr: CSBK info must be 10 bytes, got %d", len(info))
	}
	return CSBK{
		Type:        MessageType((info[0] >> 3) & 0x1F),
		Flags:       info[0] & 0x07,
		SourceID:    uint32(info[1])<<16 | uint32(info[2])<<8 | uint32(info[3]),
		DestID:      uint32(info[4])<<16 | uint32(info[5])<<8 | uint32(info[6]),
		ServiceInfo: info[7],
		Extra:       binary.BigEndian.Uint16(info[8:10]),
	}, nil
}

// CSBKFromBits packs 80 MSB-first bits (each entry 0/1) into a CSBK.
func CSBKFromBits(bits []byte) (CSBK, error) {
	if len(bits) != 80 {
		return CSBK{}, errors.New("dpmr: CSBK requires 80 bits")
	}
	info := make([]byte, 10)
	for i := 0; i < 80; i++ {
		if bits[i]&1 != 0 {
			info[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return ParseCSBK(info)
}

// CSBKBits returns the 80 MSB-first bits of a CSBK.
func CSBKBits(c CSBK) []byte {
	bytes := AssembleCSBK(c)
	out := make([]byte, 80)
	for i := 0; i < 80; i++ {
		if bytes[i>>3]&(1<<uint(7-(i&7))) != 0 {
			out[i] = 1
		}
	}
	return out
}

// Flag bits inside the 3-bit Flags field.
const (
	FlagGroupCall byte = 0x4
	FlagEmergency byte = 0x2
	FlagEncrypted byte = 0x1
)

// IsGroup reports whether the CSBK Flags field marks the call as a
// group (rather than individual) call.
func (c CSBK) IsGroup() bool { return c.Flags&FlagGroupCall != 0 }

// IsEmergency reports whether the Flags field marks the call as
// emergency.
func (c CSBK) IsEmergency() bool { return c.Flags&FlagEmergency != 0 }

// IsEncrypted reports whether the Flags field marks the call as
// encrypted.
func (c CSBK) IsEncrypted() bool { return c.Flags&FlagEncrypted != 0 }
