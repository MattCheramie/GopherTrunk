package phase1

import (
	"encoding/binary"
	"fmt"
)

// Opcode is the 6-bit TSBK opcode field per TIA-102.AABF Table 7.
type Opcode uint8

// TSBK opcodes follow the OP25 mapping (which is the de-facto reference
// implementation of TIA-102.AABF). Vendor-specific extensions live behind
// MFID != 0x00.
const (
	OpGroupVoiceChannelGrant       Opcode = 0x00
	OpGroupVoiceChannelUpdate      Opcode = 0x02
	OpGroupVoiceChannelUpdateExpl  Opcode = 0x03
	OpUnitToUnitVoiceChannelGrant  Opcode = 0x04
	OpUnitToUnitAnswerRequest      Opcode = 0x05
	OpUnitToUnitVoiceChannelUpdate Opcode = 0x06
	OpTelephoneInterconnectGrant   Opcode = 0x08
	OpTelephoneAnswerRequest       Opcode = 0x0A
	OpSNDCPDataChannelGrant        Opcode = 0x14
	OpStatusUpdate                 Opcode = 0x18
	OpStatusQuery                  Opcode = 0x1A
	OpMessageUpdate                Opcode = 0x1C
	OpRadioUnitMonitor             Opcode = 0x1D
	OpCallAlert                    Opcode = 0x1F
	OpAcknowledgeResponse          Opcode = 0x20
	OpQueuedResponse               Opcode = 0x21
	OpExtendedFunctionCommand      Opcode = 0x24
	OpDenyResponse                 Opcode = 0x27
	OpGroupAffiliationResponse     Opcode = 0x28
	OpGroupAffiliationQuery        Opcode = 0x2A
	OpLocationRegistrationResponse Opcode = 0x2B
	OpUnitRegistrationResponse     Opcode = 0x2C
	OpUnitRegistrationCommand      Opcode = 0x2D
	OpDeregistrationAck            Opcode = 0x2F
	OpIdentifierUpdateTDMA         Opcode = 0x33
	OpIdentifierUpdateVUHF         Opcode = 0x34
	OpProtectionParamUpdate        Opcode = 0x35
	OpSecondaryControlChannel      Opcode = 0x39
	OpRFSSStatusBroadcast          Opcode = 0x3A
	OpNetworkStatusBroadcast       Opcode = 0x3B
	OpAdjacentSiteStatusBroadcast  Opcode = 0x3C
	OpIdentifierUpdate             Opcode = 0x3D
	OpProtectionParamBroadcast     Opcode = 0x3F
)

func (o Opcode) String() string {
	switch o {
	case OpGroupVoiceChannelGrant:
		return "GroupVoiceChannelGrant"
	case OpGroupVoiceChannelUpdate:
		return "GroupVoiceChannelUpdate"
	case OpGroupVoiceChannelUpdateExpl:
		return "GroupVoiceChannelUpdateExplicit"
	case OpUnitToUnitVoiceChannelGrant:
		return "UnitToUnitVoiceChannelGrant"
	case OpUnitToUnitAnswerRequest:
		return "UnitToUnitAnswerRequest"
	case OpAdjacentSiteStatusBroadcast:
		return "AdjacentSiteStatusBroadcast"
	case OpRFSSStatusBroadcast:
		return "RFSSStatusBroadcast"
	case OpNetworkStatusBroadcast:
		return "NetworkStatusBroadcast"
	case OpSecondaryControlChannel:
		return "SecondaryControlChannelBroadcast"
	case OpIdentifierUpdate:
		return "IdentifierUpdate"
	case OpIdentifierUpdateVUHF:
		return "IdentifierUpdateVUHF"
	case OpIdentifierUpdateTDMA:
		return "IdentifierUpdateTDMA"
	case OpGroupAffiliationResponse:
		return "GroupAffiliationResponse"
	case OpUnitRegistrationResponse:
		return "UnitRegistrationResponse"
	default:
		return fmt.Sprintf("Opcode(%02X)", uint8(o))
	}
}

// GroupVoiceChannelGrant (opcode 0x00) — base format. Payload layout:
//
//	byte 0:    service options
//	byte 1-2:  channel (4-bit ID + 12-bit number)
//	byte 3-4:  group address (talkgroup)
//	byte 5-7:  source unit (24-bit)
type GroupVoiceChannelGrant struct {
	ServiceOptions uint8
	ChannelID      uint8
	ChannelNumber  uint16
	GroupAddress   uint16
	SourceID       uint32
}

// ParseGroupVoiceChannelGrant decodes payload bytes for opcode 0x00.
func ParseGroupVoiceChannelGrant(p [8]byte) GroupVoiceChannelGrant {
	chanField := binary.BigEndian.Uint16(p[1:3])
	return GroupVoiceChannelGrant{
		ServiceOptions: p[0],
		ChannelID:      uint8(chanField >> 12),
		ChannelNumber:  chanField & 0x0FFF,
		GroupAddress:   binary.BigEndian.Uint16(p[3:5]),
		SourceID:       uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
	}
}

// GroupVoiceChannelUpdate (opcode 0x02) — channel announcement: two
// (channel, group) pairs in the same payload. The B fields are zero when
// only one call is being announced.
type GroupVoiceChannelUpdate struct {
	ChannelAID, ChannelBID         uint8
	ChannelANumber, ChannelBNumber uint16
	GroupAddressA, GroupAddressB   uint16
}

func ParseGroupVoiceChannelUpdate(p [8]byte) GroupVoiceChannelUpdate {
	cA := binary.BigEndian.Uint16(p[0:2])
	cB := binary.BigEndian.Uint16(p[4:6])
	return GroupVoiceChannelUpdate{
		ChannelAID:     uint8(cA >> 12),
		ChannelANumber: cA & 0x0FFF,
		GroupAddressA:  binary.BigEndian.Uint16(p[2:4]),
		ChannelBID:     uint8(cB >> 12),
		ChannelBNumber: cB & 0x0FFF,
		GroupAddressB:  binary.BigEndian.Uint16(p[6:8]),
	}
}

// AssembleGroupVoiceChannelUpdate is the inverse of
// ParseGroupVoiceChannelUpdate; used by tests.
func AssembleGroupVoiceChannelUpdate(u GroupVoiceChannelUpdate) [8]byte {
	var p [8]byte
	binary.BigEndian.PutUint16(p[0:2], uint16(u.ChannelAID&0x0F)<<12|u.ChannelANumber&0x0FFF)
	binary.BigEndian.PutUint16(p[2:4], u.GroupAddressA)
	binary.BigEndian.PutUint16(p[4:6], uint16(u.ChannelBID&0x0F)<<12|u.ChannelBNumber&0x0FFF)
	binary.BigEndian.PutUint16(p[6:8], u.GroupAddressB)
	return p
}

// ParseGroupVoiceChannelUpdateExplicit decodes opcode 0x03, the
// explicit-channel group voice update. It carries the same channel +
// group + source fields as a group voice grant (opcode 0x00).
func ParseGroupVoiceChannelUpdateExplicit(p [8]byte) GroupVoiceChannelGrant {
	return ParseGroupVoiceChannelGrant(p)
}

// ServiceOptions decodes the 8-bit SVC_OPTIONS field a P25 voice grant
// carries (TIA-102.AABF): bit 7 = Emergency, bit 6 = Protected (the
// encryption indicator), bits 0-2 = call priority.
type ServiceOptions uint8

// Emergency reports the emergency-call bit.
func (s ServiceOptions) Emergency() bool { return s&0x80 != 0 }

// Encrypted reports the protected (encryption) bit.
func (s ServiceOptions) Encrypted() bool { return s&0x40 != 0 }

// Priority returns the 3-bit call-priority field (0 = lowest).
func (s ServiceOptions) Priority() uint8 { return uint8(s) & 0x07 }

// UnitToUnitVoiceChannelGrant (opcode 0x04) — a private unit-to-unit
// voice call. Working-model payload layout (TIA-102.AABF):
//
//	bytes 0-1 : channel (4-bit ID + 12-bit number)
//	bytes 2-4 : target unit ID (24 bits)
//	bytes 5-7 : source unit ID (24 bits)
type UnitToUnitVoiceChannelGrant struct {
	ChannelID     uint8
	ChannelNumber uint16
	TargetID      uint32
	SourceID      uint32
}

// ParseUnitToUnitVoiceChannelGrant decodes payload bytes for opcode 0x04.
func ParseUnitToUnitVoiceChannelGrant(p [8]byte) UnitToUnitVoiceChannelGrant {
	ch := binary.BigEndian.Uint16(p[0:2])
	return UnitToUnitVoiceChannelGrant{
		ChannelID:     uint8(ch >> 12),
		ChannelNumber: ch & 0x0FFF,
		TargetID:      uint32(p[2])<<16 | uint32(p[3])<<8 | uint32(p[4]),
		SourceID:      uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
	}
}

// AssembleUnitToUnitVoiceChannelGrant is the inverse; used by tests.
func AssembleUnitToUnitVoiceChannelGrant(g UnitToUnitVoiceChannelGrant) [8]byte {
	var p [8]byte
	binary.BigEndian.PutUint16(p[0:2], uint16(g.ChannelID&0x0F)<<12|g.ChannelNumber&0x0FFF)
	p[2], p[3], p[4] = byte(g.TargetID>>16), byte(g.TargetID>>8), byte(g.TargetID)
	p[5], p[6], p[7] = byte(g.SourceID>>16), byte(g.SourceID>>8), byte(g.SourceID)
	return p
}

// TelephoneInterconnectGrant (opcode 0x08) — a grant for a call bridged
// to the public telephone network. Working-model payload layout:
//
//	byte 0    : service options
//	bytes 1-2 : channel
//	bytes 3-4 : call timer (units of 100 ms)
//	bytes 5-7 : target unit ID — the radio on the call
type TelephoneInterconnectGrant struct {
	ServiceOptions uint8
	ChannelID      uint8
	ChannelNumber  uint16
	CallTimer      uint16
	TargetID       uint32
}

// ParseTelephoneInterconnectGrant decodes payload bytes for opcode 0x08.
func ParseTelephoneInterconnectGrant(p [8]byte) TelephoneInterconnectGrant {
	ch := binary.BigEndian.Uint16(p[1:3])
	return TelephoneInterconnectGrant{
		ServiceOptions: p[0],
		ChannelID:      uint8(ch >> 12),
		ChannelNumber:  ch & 0x0FFF,
		CallTimer:      binary.BigEndian.Uint16(p[3:5]),
		TargetID:       uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
	}
}

// AssembleTelephoneInterconnectGrant is the inverse; used by tests.
func AssembleTelephoneInterconnectGrant(g TelephoneInterconnectGrant) [8]byte {
	var p [8]byte
	p[0] = g.ServiceOptions
	binary.BigEndian.PutUint16(p[1:3], uint16(g.ChannelID&0x0F)<<12|g.ChannelNumber&0x0FFF)
	binary.BigEndian.PutUint16(p[3:5], g.CallTimer)
	p[5], p[6], p[7] = byte(g.TargetID>>16), byte(g.TargetID>>8), byte(g.TargetID)
	return p
}

// SNDCPDataChannelGrant (opcode 0x14) — a grant for a packet-data
// (SNDCP) channel. Working-model payload layout:
//
//	byte 0    : data service options
//	bytes 1-2 : channel
//	bytes 3-4 : reserved
//	bytes 5-7 : target unit ID
type SNDCPDataChannelGrant struct {
	ServiceOptions uint8
	ChannelID      uint8
	ChannelNumber  uint16
	TargetID       uint32
}

// ParseSNDCPDataChannelGrant decodes payload bytes for opcode 0x14.
func ParseSNDCPDataChannelGrant(p [8]byte) SNDCPDataChannelGrant {
	ch := binary.BigEndian.Uint16(p[1:3])
	return SNDCPDataChannelGrant{
		ServiceOptions: p[0],
		ChannelID:      uint8(ch >> 12),
		ChannelNumber:  ch & 0x0FFF,
		TargetID:       uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
	}
}

// AssembleSNDCPDataChannelGrant is the inverse; used by tests.
func AssembleSNDCPDataChannelGrant(g SNDCPDataChannelGrant) [8]byte {
	var p [8]byte
	p[0] = g.ServiceOptions
	binary.BigEndian.PutUint16(p[1:3], uint16(g.ChannelID&0x0F)<<12|g.ChannelNumber&0x0FFF)
	p[5], p[6], p[7] = byte(g.TargetID>>16), byte(g.TargetID>>8), byte(g.TargetID)
	return p
}

// GroupAffiliationResponse (opcode 0x28) — published when a radio unit
// is granted (or denied) affiliation with a talkgroup. Payload layout
// follows OP25's reference decoder (`trunk_p25.py`):
//
//	byte 0:    bits 1-0 = Affiliation Response Value
//	bytes 1-2: Announcement Group Address (16 bits)
//	bytes 3-4: Group Address (16 bits) — the talkgroup
//	bytes 5-7: Target ID (24 bits) — the radio being responded to
type GroupAffiliationResponse struct {
	Response          uint8
	AnnouncementGroup uint16
	GroupAddress      uint16
	TargetID          uint32
}

// ParseGroupAffiliationResponse decodes payload bytes for opcode 0x28.
func ParseGroupAffiliationResponse(p [8]byte) GroupAffiliationResponse {
	return GroupAffiliationResponse{
		Response:          p[0] & 0x03,
		AnnouncementGroup: binary.BigEndian.Uint16(p[1:3]),
		GroupAddress:      binary.BigEndian.Uint16(p[3:5]),
		TargetID:          uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
	}
}

// UnitRegistrationResponse (opcode 0x2C) — published when a radio
// completes (or is denied) registration on a site. Payload layout
// follows OP25's reference decoder (`trunk_p25.py`):
//
//	byte 0:        bits 1-0 = Registration Response Value
//	bytes 1-3 + top nibble of byte 3: WACN (20 bits)
//	bottom nibble of byte 3 + byte 4: System ID (12 bits)
//	bytes 5-7:     Source ID (24 bits) — the radio's WUID
type UnitRegistrationResponse struct {
	Response uint8
	WACN     uint32 // 20-bit
	SystemID uint16 // 12-bit
	SourceID uint32 // 24-bit
}

// ParseUnitRegistrationResponse decodes payload bytes for opcode 0x2C.
func ParseUnitRegistrationResponse(p [8]byte) UnitRegistrationResponse {
	return UnitRegistrationResponse{
		Response: p[0] & 0x03,
		WACN:     uint32(p[1])<<12 | uint32(p[2])<<4 | uint32(p[3])>>4,
		SystemID: uint16(p[3]&0x0F)<<8 | uint16(p[4]),
		SourceID: uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
	}
}

// NetworkStatusBroadcast (opcode 0x3B) carries the system's network
// identity. Payload layout per TIA-102.AABF — identical to the Phase 2
// Network Status Broadcast (see phase2/mac.go AsNetworkStatusBroadcast):
//
//	byte 0:     LRA (Location Registration Area)
//	bytes 1-3:  WACN ID (20 bits in the upper 20 of bytes 1-3)
//	bytes 3-4:  System ID (low nibble of byte 3 + byte 4)
//	bytes 5-6:  channel (4-bit ID + 12-bit number)
//	byte 7:     system service class
type NetworkStatusBroadcast struct {
	LRA           uint8
	WACN          uint32 // 20-bit
	SystemID      uint16 // 12-bit
	ChannelID     uint8
	ChannelNumber uint16
	ServiceClass  uint16
}

func ParseNetworkStatusBroadcast(p [8]byte) NetworkStatusBroadcast {
	wacn := uint32(p[1])<<12 | uint32(p[2])<<4 | uint32(p[3])>>4
	sysid := uint16(p[3]&0x0F)<<8 | uint16(p[4])
	chanField := binary.BigEndian.Uint16(p[5:7])
	return NetworkStatusBroadcast{
		LRA:           p[0],
		WACN:          wacn,
		SystemID:      sysid,
		ChannelID:     uint8(chanField >> 12),
		ChannelNumber: chanField & 0x0FFF,
		ServiceClass:  uint16(p[7]),
	}
}

// RFSSStatusBroadcast (opcode 0x3A) names the site the receiver is
// camped on. Payload layout per TIA-102.AABF — identical to the Phase 2
// RFSS Status Broadcast (see phase2/mac_standard.go AsRFSSStatusBroadcast):
//
//	byte 0:     LRA (Location Registration Area)
//	bytes 1-2:  System ID (12 bits, low nibble of byte 1 + byte 2)
//	byte 3:     RFSS ID
//	byte 4:     Site ID
//	bytes 5-6:  channel (4-bit ID + 12-bit number)
//	byte 7:     system service class
type RFSSStatusBroadcast struct {
	LRA           uint8
	SystemID      uint16 // 12-bit
	RFSS          uint8
	Site          uint8
	ChannelID     uint8
	ChannelNumber uint16
	ServiceClass  uint16
}

// SecondaryControlChannelBroadcast (opcode 0x39) announces alternate
// control-channel frequencies for the site. Working-model payload
// layout:
//
//	byte 0    : RFSS ID
//	byte 1    : Site ID
//	bytes 2-3 : secondary control channel A
//	bytes 4-5 : secondary control channel B (zero when only one)
//	bytes 6-7 : system service class
type SecondaryControlChannelBroadcast struct {
	RFSS, Site                     uint8
	ChannelAID, ChannelBID         uint8
	ChannelANumber, ChannelBNumber uint16
}

// ParseSecondaryControlChannelBroadcast decodes payload for opcode 0x39.
func ParseSecondaryControlChannelBroadcast(p [8]byte) SecondaryControlChannelBroadcast {
	cA := binary.BigEndian.Uint16(p[2:4])
	cB := binary.BigEndian.Uint16(p[4:6])
	return SecondaryControlChannelBroadcast{
		RFSS:           p[0],
		Site:           p[1],
		ChannelAID:     uint8(cA >> 12),
		ChannelANumber: cA & 0x0FFF,
		ChannelBID:     uint8(cB >> 12),
		ChannelBNumber: cB & 0x0FFF,
	}
}

// AssembleSecondaryControlChannelBroadcast is the inverse; for tests.
func AssembleSecondaryControlChannelBroadcast(s SecondaryControlChannelBroadcast) [8]byte {
	var p [8]byte
	p[0], p[1] = s.RFSS, s.Site
	binary.BigEndian.PutUint16(p[2:4], uint16(s.ChannelAID&0x0F)<<12|s.ChannelANumber&0x0FFF)
	binary.BigEndian.PutUint16(p[4:6], uint16(s.ChannelBID&0x0F)<<12|s.ChannelBNumber&0x0FFF)
	return p
}

// AdjacentSiteStatusBroadcast (opcode 0x3C) announces a neighbouring
// site a radio may roam to. Payload layout per TIA-102.AABF — it shares
// the RFSS Status Broadcast field layout, except byte 1's high nibble
// carries the CFVA (conventional/failsoft/valid/active) flags rather
// than reserved bits, and it still names the neighbour's System ID:
//
//	byte 0    : LRA (Location Registration Area)
//	bytes 1-2 : System ID (12 bits, low nibble of byte 1 + byte 2)
//	byte 3    : RFSS ID
//	byte 4    : Site ID
//	bytes 5-6 : the neighbour's control channel (4-bit ID + 12-bit number)
//	byte 7    : system service class
type AdjacentSiteStatusBroadcast struct {
	LRA           uint8
	SystemID      uint16 // 12-bit
	RFSS, Site    uint8
	ChannelID     uint8
	ChannelNumber uint16
}

// ParseAdjacentSiteStatusBroadcast decodes payload for opcode 0x3C.
func ParseAdjacentSiteStatusBroadcast(p [8]byte) AdjacentSiteStatusBroadcast {
	ch := binary.BigEndian.Uint16(p[5:7])
	return AdjacentSiteStatusBroadcast{
		LRA:           p[0],
		SystemID:      uint16(p[1]&0x0F)<<8 | uint16(p[2]),
		RFSS:          p[3],
		Site:          p[4],
		ChannelID:     uint8(ch >> 12),
		ChannelNumber: ch & 0x0FFF,
	}
}

// AssembleAdjacentSiteStatusBroadcast is the inverse; for tests.
func AssembleAdjacentSiteStatusBroadcast(a AdjacentSiteStatusBroadcast) [8]byte {
	var p [8]byte
	p[0] = a.LRA
	p[1] = uint8(a.SystemID>>8) & 0x0F
	p[2] = uint8(a.SystemID)
	p[3] = a.RFSS
	p[4] = a.Site
	binary.BigEndian.PutUint16(p[5:7], uint16(a.ChannelID&0x0F)<<12|a.ChannelNumber&0x0FFF)
	return p
}

func ParseRFSSStatusBroadcast(p [8]byte) RFSSStatusBroadcast {
	chanField := binary.BigEndian.Uint16(p[5:7])
	return RFSSStatusBroadcast{
		LRA:           p[0],
		SystemID:      uint16(p[1]&0x0F)<<8 | uint16(p[2]),
		RFSS:          p[3],
		Site:          p[4],
		ChannelID:     uint8(chanField >> 12),
		ChannelNumber: chanField & 0x0FFF,
		ServiceClass:  uint16(p[7]),
	}
}
