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

// NetworkStatusBroadcast (opcode 0x3B explicit / 0x3C is Adjacent in some
// vendor variants — TIA-102.AABF is unfortunately revised over time;
// callers should look at MFID + opcode together to disambiguate). This
// parser handles the standard explicit-form payload:
//
//	bytes 0-2:  WACN ID (20 bits in upper 20 of 24)
//	bytes 2-3:  System ID (12 bits)
//	bytes 4-5:  channel
//	bytes 6-7:  system service class
type NetworkStatusBroadcast struct {
	WACN          uint32 // 20-bit
	SystemID      uint16 // 12-bit
	ChannelID     uint8
	ChannelNumber uint16
	ServiceClass  uint16
}

func ParseNetworkStatusBroadcast(p [8]byte) NetworkStatusBroadcast {
	wacn := uint32(p[0])<<12 | uint32(p[1])<<4 | uint32(p[2]>>4)
	sysid := uint16(p[2]&0x0F)<<8 | uint16(p[3])
	chanField := binary.BigEndian.Uint16(p[4:6])
	return NetworkStatusBroadcast{
		WACN:          wacn,
		SystemID:      sysid,
		ChannelID:     uint8(chanField >> 12),
		ChannelNumber: chanField & 0x0FFF,
		ServiceClass:  binary.BigEndian.Uint16(p[6:8]),
	}
}

// RFSSStatusBroadcast (opcode 0x3A in standard form). Payload:
//
//	byte 0:     LRA (Location Registration Area)
//	byte 1:     System ID high nibble + RFSS ID
//	byte 2:     RFSS ID continued / Site ID
//	byte 3:     Site ID continued
//	bytes 4-5:  channel
//	bytes 6-7:  system service class
type RFSSStatusBroadcast struct {
	LRA           uint8
	SystemID      uint16 // 12-bit
	RFSS          uint8
	Site          uint8
	ChannelID     uint8
	ChannelNumber uint16
	ServiceClass  uint16
}

func ParseRFSSStatusBroadcast(p [8]byte) RFSSStatusBroadcast {
	chanField := binary.BigEndian.Uint16(p[4:6])
	return RFSSStatusBroadcast{
		LRA:           p[0],
		SystemID:      uint16(p[1]&0x0F)<<8 | uint16(p[2]),
		RFSS:          p[2], // some variants split high/low nibble; documented per-implementation
		Site:          p[3],
		ChannelID:     uint8(chanField >> 12),
		ChannelNumber: chanField & 0x0FFF,
		ServiceClass:  binary.BigEndian.Uint16(p[6:8]),
	}
}
