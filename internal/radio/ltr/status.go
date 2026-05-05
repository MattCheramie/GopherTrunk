package ltr

import (
	"errors"
	"fmt"
)

// Status is one LTR repeater status word — 41 bits transmitted
// continuously at 300 bps under the in-band voice.
//
// Field layout (MSB-first across the 41-bit word):
//
//   bits 40..40 (1 bit)   Sync   — frame-start marker (always 1).
//   bits 39..35 (5 bits)  Area   — area code (0..31). Multiple LTR
//                                  systems on the same frequency are
//                                  disambiguated by Area.
//   bits 34..34 (1 bit)   Group  — the "F-bit". 1 = call is active
//                                  on this repeater for the named
//                                  group; 0 = idle / free.
//   bits 33..30 (4 bits)  Chan   — physical channel number (1..20)
//                                  this status word references.
//   bits 29..25 (5 bits)  Home   — home-repeater number (1..20)
//                                  for the active group.
//   bits 24..17 (8 bits)  Group  — group / talkgroup ID (1..250).
//   bits 16..12 (5 bits)  Free   — free-repeater hint
//                                  (which repeater is currently
//                                  unallocated, for handoff).
//   bits 11..0  (12 bits) FCS    — frame check / parity. Computed
//                                  by the encoder and verified by
//                                  the decoder.
//
// As with the other protocol packages, the field positions follow
// the most-cited public reference; some LTR-Net variants pack
// fields slightly differently. Cross-check before trusting live
// captures.
type Status struct {
	Sync    bool
	Area    uint8  // 5-bit
	Group   bool   // F-bit
	Channel uint8  // 4-bit (1..20)
	Home    uint8  // 5-bit (1..20)
	GroupID uint16 // 8-bit (1..250)
	Free    uint8  // 5-bit
	FCS     uint16 // 12-bit
}

// AssembleStatus packs a Status into 6 bytes (48 bits, with the upper
// 41 bits carrying the status word and the bottom 7 bits zero-padded).
// Used by tests and for any future encoder work.
func AssembleStatus(s Status) []byte {
	var v uint64
	if s.Sync {
		v |= 1 << 40
	}
	v |= uint64(s.Area&0x1F) << 35
	if s.Group {
		v |= 1 << 34
	}
	v |= uint64(s.Channel&0x0F) << 30
	v |= uint64(s.Home&0x1F) << 25
	v |= uint64(s.GroupID&0xFF) << 17
	v |= uint64(s.Free&0x1F) << 12
	v |= uint64(s.FCS & 0x0FFF)
	// Left-align into 6 bytes (48 bits) for clean MSB-first transport.
	v <<= (48 - 41)
	return []byte{
		byte((v >> 40) & 0xFF),
		byte((v >> 32) & 0xFF),
		byte((v >> 24) & 0xFF),
		byte((v >> 16) & 0xFF),
		byte((v >> 8) & 0xFF),
		byte(v & 0xFF),
	}
}

// ParseStatus reads 6 bytes (the upper 41 bits MSB-first as produced
// by AssembleStatus) into a Status.
func ParseStatus(info []byte) (Status, error) {
	if len(info) != 6 {
		return Status{}, fmt.Errorf("ltr: status info must be 6 bytes, got %d", len(info))
	}
	v := uint64(info[0])<<40 | uint64(info[1])<<32 | uint64(info[2])<<24 |
		uint64(info[3])<<16 | uint64(info[4])<<8 | uint64(info[5])
	v >>= (48 - 41) // strip the trailing zero-padding
	return Status{
		Sync:    v&(1<<40) != 0,
		Area:    uint8((v >> 35) & 0x1F),
		Group:   v&(1<<34) != 0,
		Channel: uint8((v >> 30) & 0x0F),
		Home:    uint8((v >> 25) & 0x1F),
		GroupID: uint16((v >> 17) & 0xFF),
		Free:    uint8((v >> 12) & 0x1F),
		FCS:     uint16(v & 0x0FFF),
	}, nil
}

// StatusFromBits packs 41 MSB-first bits (each entry 0/1) into a
// Status.
func StatusFromBits(bits []byte) (Status, error) {
	if len(bits) != 41 {
		return Status{}, errors.New("ltr: status requires 41 bits")
	}
	info := make([]byte, 6)
	for i := 0; i < 41; i++ {
		if bits[i]&1 != 0 {
			info[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return ParseStatus(info)
}

// StatusBits returns the 41 MSB-first bits of a Status.
func StatusBits(s Status) []byte {
	bytes := AssembleStatus(s)
	out := make([]byte, 41)
	for i := 0; i < 41; i++ {
		if bytes[i>>3]&(1<<uint(7-(i&7))) != 0 {
			out[i] = 1
		}
	}
	return out
}

// IsActive reports whether the status word indicates an active call
// on this repeater. By convention LTR sets the Group ("F") bit to 1
// while a group is transmitting and the GroupID is non-zero.
func (s Status) IsActive() bool { return s.Group && s.GroupID != 0 }
