package phase2

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// MACPDU is one P25 Phase 2 MAC PDU — the signalling unit that
// rides MAC slots inside a Phase 2 traffic channel. After Reed-
// Solomon / Trellis FEC removal, a MAC PDU resolves to:
//
//	byte 0     : opcode (the MAC_PDU_OPCODE field)
//	bytes 1-N  : opcode-specific payload (typically up to 17 bytes
//	             across the MAC slot's 21-byte "MAC PDU SLOT" field;
//	             exact length depends on opcode + format).
//
// The structure is intentionally permissive: callers parse the
// opcode and then dispatch to a per-opcode accessor.
type MACPDU struct {
	Opcode  Opcode
	Payload []byte // copy of bytes 1..end
}

// Opcode is the MAC PDU opcode field. Values follow TIA-102.AABF /
// BBAB Table 8.x; only the subset relevant to trunking follow-along
// is enumerated below.
type Opcode uint8

const (
	OpUnknown                      Opcode = 0x00
	OpMACPTT                       Opcode = 0x01 // PTT-on
	OpMACEnd                       Opcode = 0x02 // End of transmission
	OpMACIdle                      Opcode = 0x03 // Channel idle
	OpMACHangtime                  Opcode = 0x05 // Hang-time
	OpMACActive                    Opcode = 0x06 // Late-grant active
	OpGroupVoiceChannelGrantUpdate Opcode = 0x40
	OpGroupVoiceChannelGrant       Opcode = 0x44
	OpGroupVoiceChannelUserExt     Opcode = 0x46
	OpUnitToUnitVoiceChannelGrant  Opcode = 0x48
	OpIdentifierUpdate             Opcode = 0x7D
	OpNetworkStatusBroadcastUpdate Opcode = 0xFB
	OpRFSSStatusBroadcastUpdate    Opcode = 0xFA
)

func (o Opcode) String() string {
	switch o {
	case OpMACPTT:
		return "MAC_PTT"
	case OpMACEnd:
		return "MAC_END"
	case OpMACIdle:
		return "MAC_IDLE"
	case OpMACHangtime:
		return "MAC_HANGTIME"
	case OpMACActive:
		return "MAC_ACTIVE"
	case OpGroupVoiceChannelGrant:
		return "GroupVoiceChannelGrant"
	case OpGroupVoiceChannelGrantUpdate:
		return "GroupVoiceChannelGrantUpdate"
	case OpGroupVoiceChannelUserExt:
		return "GroupVoiceChannelUserExt"
	case OpUnitToUnitVoiceChannelGrant:
		return "UnitToUnitVoiceChannelGrant"
	case OpIdentifierUpdate:
		return "IdentifierUpdate"
	case OpNetworkStatusBroadcastUpdate:
		return "NetworkStatusBroadcastUpdate"
	case OpRFSSStatusBroadcastUpdate:
		return "RFSSStatusBroadcastUpdate"
	default:
		return fmt.Sprintf("Opcode(%02X)", uint8(o))
	}
}

// ParseMACPDU consumes 18-byte MAC PDU information bytes (opcode +
// up to 17 payload bytes, the standard size after FEC removal) and
// returns the structured PDU.
func ParseMACPDU(info []byte) (MACPDU, error) {
	if len(info) < 1 {
		return MACPDU{}, errors.New("p25/phase2: MAC PDU info needs at least 1 byte")
	}
	pdu := MACPDU{Opcode: Opcode(info[0])}
	if len(info) > 1 {
		pdu.Payload = make([]byte, len(info)-1)
		copy(pdu.Payload, info[1:])
	}
	return pdu, nil
}

// AssembleMACPDU re-packs a MAC PDU into bytes (opcode + payload).
// Used by tests; encoder support for Phase 2 isn't a project goal.
func AssembleMACPDU(p MACPDU) []byte {
	out := make([]byte, 1+len(p.Payload))
	out[0] = byte(p.Opcode)
	copy(out[1:], p.Payload)
	return out
}

// GroupVoiceChannelGrant is the structured shape of a Phase 2
// voice-grant MAC PDU. Field positions follow the TIA-102 layout:
//
//	byte 0    : service options
//	bytes 1-2 : channel ID + channel number (4 + 12 bits)
//	bytes 3-4 : group address (talkgroup)
//	bytes 5-7 : source unit ID (24 bits)
type GroupVoiceChannelGrant struct {
	ServiceOptions uint8
	ChannelID      uint8
	ChannelNumber  uint16
	GroupAddress   uint16
	SourceID       uint32
}

// AsGroupVoiceChannelGrant returns the structured grant if the PDU
// opcode is a voice-grant variant, otherwise (zero, false).
func (p MACPDU) AsGroupVoiceChannelGrant() (GroupVoiceChannelGrant, bool) {
	switch p.Opcode {
	case OpGroupVoiceChannelGrant, OpGroupVoiceChannelGrantUpdate:
	default:
		return GroupVoiceChannelGrant{}, false
	}
	if len(p.Payload) < 8 {
		return GroupVoiceChannelGrant{}, false
	}
	chanField := binary.BigEndian.Uint16(p.Payload[1:3])
	return GroupVoiceChannelGrant{
		ServiceOptions: p.Payload[0],
		ChannelID:      uint8(chanField >> 12),
		ChannelNumber:  chanField & 0x0FFF,
		GroupAddress:   binary.BigEndian.Uint16(p.Payload[3:5]),
		SourceID:       uint32(p.Payload[5])<<16 | uint32(p.Payload[6])<<8 | uint32(p.Payload[7]),
	}, true
}

// NetworkStatusBroadcast is the structured shape of a Phase 2
// Network Status Broadcast - Update MAC PDU (opcode 0xFB).
// Carries the system-identity fields the PN44 descrambler per
// TIA-102.BBAC-1 §7.2.5 needs to compute its seed:
//
//   - WACN: 20-bit Wide-Area Communication Network ID
//   - SystemID: 12-bit P25 System ID
//   - ColorCode: 12-bit Color Code, equal to the Phase 1 NAC per
//     the spec's seed derivation rule
//
// Payload layout (after the 1-byte opcode the MAC PDU parses off):
//
//	byte 0       : LRA (Location Registration Area)
//	bytes 1..3   : WACN (20 bits in the upper 20 of bytes 1..3,
//	               i.e., bit 19 of WACN = bit 7 of byte 1, … bit 0
//	               of WACN = bit 4 of byte 3)
//	bytes 3..4   : SystemID (low nibble of byte 3 + byte 4)
//	bytes 5..6   : Color Code (12 bits — high 12 bits of bytes 5..6)
//	bytes 7..8   : channel info (ChannelID + ChannelNumber)
//
// The WACN + SystemID bit packing matches the Phase 1 NSB layout
// (TIA-102.AABF), which Phase 2 reuses. The Color Code position
// follows the most-cited public reference for Phase 2 NSB-Update;
// vendors occasionally repurpose the byte 5..6 field, so callers
// that hit a mismatch on a specific system can override the seed
// explicitly via SetScramblerSeed.
type NetworkStatusBroadcast struct {
	LRA           uint8  // 8-bit Location Registration Area
	WACN          uint32 // 20-bit Wide-Area Communication Network ID
	SystemID      uint16 // 12-bit System ID
	ColorCode     uint16 // 12-bit Color Code = Phase 1 NAC per spec
	ChannelID     uint8
	ChannelNumber uint16
}

// AsNetworkStatusBroadcast returns the structured NSB if the PDU
// opcode is OpNetworkStatusBroadcastUpdate (0xFB), otherwise
// (zero, false). Requires a payload of at least 9 bytes to cover
// LRA + WACN + SystemID + ColorCode + channel fields.
func (p MACPDU) AsNetworkStatusBroadcast() (NetworkStatusBroadcast, bool) {
	if p.Opcode != OpNetworkStatusBroadcastUpdate {
		return NetworkStatusBroadcast{}, false
	}
	if len(p.Payload) < 9 {
		return NetworkStatusBroadcast{}, false
	}
	wacn := uint32(p.Payload[1])<<12 | uint32(p.Payload[2])<<4 | uint32(p.Payload[3]>>4)
	sysid := uint16(p.Payload[3]&0x0F)<<8 | uint16(p.Payload[4])
	cc := uint16(p.Payload[5])<<4 | uint16(p.Payload[6]>>4)
	chanField := binary.BigEndian.Uint16(p.Payload[7:9])
	return NetworkStatusBroadcast{
		LRA:           p.Payload[0],
		WACN:          wacn,
		SystemID:      sysid,
		ColorCode:     cc,
		ChannelID:     uint8(chanField >> 12),
		ChannelNumber: chanField & 0x0FFF,
	}, true
}

// AsIdentifierUpdate returns the structured band-plan definition if the
// PDU opcode is OpIdentifierUpdate, otherwise (zero, false). The
// ControlChannel feeds the result to its BandPlan so later voice grants
// resolve a downlink frequency.
func (p MACPDU) AsIdentifierUpdate() (IdentifierUpdate, bool) {
	if p.Opcode != OpIdentifierUpdate {
		return IdentifierUpdate{}, false
	}
	return ParseIdentifierUpdate(p.Payload)
}

// IsIdle reports whether the PDU is one of the channel-idle / hang-
// time opcodes a state machine should silently absorb.
func (p MACPDU) IsIdle() bool {
	switch p.Opcode {
	case OpMACIdle, OpMACHangtime, OpMACEnd:
		return true
	}
	return false
}

// IsKnown reports whether the Opcode is one of the documented
// TIA-102.AABF / BBAB MAC PDU opcodes the state machine recognises.
// Used by SetStrictValidation to drop PDUs whose 8-bit opcode field
// falls outside the recognised set.
func (o Opcode) IsKnown() bool {
	switch o {
	case OpMACPTT, OpMACEnd, OpMACIdle, OpMACHangtime, OpMACActive,
		OpGroupVoiceChannelGrant, OpGroupVoiceChannelGrantUpdate,
		OpGroupVoiceChannelUserExt, OpUnitToUnitVoiceChannelGrant,
		OpIdentifierUpdate,
		OpNetworkStatusBroadcastUpdate, OpRFSSStatusBroadcastUpdate:
		return true
	}
	return false
}
