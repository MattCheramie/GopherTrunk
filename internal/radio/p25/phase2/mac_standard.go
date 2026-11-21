package phase2

import "encoding/binary"

// Structured accessors for the standard (non-manufacturer-specific) P25
// Phase 2 MAC opcodes beyond the group-voice-grant / NSB / IdentifierUp
// trio handled in mac.go. Field positions are SDRTrunk's (its bit indexes
// count from the opcode's first bit); issue #1364. Each As...() accessor returns (struct, false)
// when the PDU's opcode does not match, so a caller can probe a PDU
// against several accessors in turn.

// UnitToUnitVoiceChannelGrant is a Phase 2 unit-to-unit (private call) voice
// grant or grant update. Unlike a group grant it carries no service options.
type UnitToUnitVoiceChannelGrant struct {
	ChannelID     uint8
	ChannelNumber uint16
	TargetID      uint32
	SourceID      uint32
}

// AsUnitToUnitVoiceChannelGrant returns the grant if the PDU is a
// Unit-to-Unit Voice Channel Grant (0x44) or its update (0x46), abbreviated
// form, otherwise (zero, false). Octet 0 the opcode (SDRTrunk
// UnitToUnitVoiceServiceChannelGrantAbbreviated / UnitToUnitVoiceChannel
// GrantUpdateAbbreviated):
//
//	1-2 channel | 3-5 target | 6-8 source
func (p MACPDU) AsUnitToUnitVoiceChannelGrant() (UnitToUnitVoiceChannelGrant, bool) {
	switch p.Opcode {
	case OpUnitToUnitVoiceChannelGrant, OpUnitToUnitVoiceChannelGrantUpdate:
	default:
		return UnitToUnitVoiceChannelGrant{}, false
	}
	b := p.Payload
	if len(b) < 8 {
		return UnitToUnitVoiceChannelGrant{}, false
	}
	u := UnitToUnitVoiceChannelGrant{
		TargetID: uint32(b[2])<<16 | uint32(b[3])<<8 | uint32(b[4]),
		SourceID: uint32(b[5])<<16 | uint32(b[6])<<8 | uint32(b[7]),
	}
	u.ChannelID, u.ChannelNumber = macChannel(b)
	return u, true
}

// RFSSStatusBroadcast is the structured shape of a Phase 2 RFSS Status
// Broadcast (0xFA explicit, 0x7A implicit). It names the site the receiver
// is camped on so a scanner can log RFSS / site topology. Octet 0 the opcode
// (SDRTrunk RfssStatusBroadcast):
//
//	1    LRA (Location Registration Area)
//	2-3  System ID (low 12 bits)
//	4    RFSS ID
//	5    Site ID
//	6-7  channel (explicit: the transmit channel)
type RFSSStatusBroadcast struct {
	LRA           uint8
	SystemID      uint16 // 12-bit
	RFSS          uint8
	Site          uint8
	ChannelID     uint8
	ChannelNumber uint16
}

// AsRFSSStatusBroadcast returns the structured RFSS status if the PDU is
// either form of the RFSS Status Broadcast, otherwise (zero, false).
func (p MACPDU) AsRFSSStatusBroadcast() (RFSSStatusBroadcast, bool) {
	if p.Opcode != OpRFSSStatusBroadcastUpdate && p.Opcode != OpRFSSStatusBroadcastImplicit {
		return RFSSStatusBroadcast{}, false
	}
	if len(p.Payload) < 7 {
		return RFSSStatusBroadcast{}, false
	}
	chanField := binary.BigEndian.Uint16(p.Payload[5:7])
	return RFSSStatusBroadcast{
		LRA:           p.Payload[0],
		SystemID:      uint16(p.Payload[1]&0x0F)<<8 | uint16(p.Payload[2]),
		RFSS:          p.Payload[3],
		Site:          p.Payload[4],
		ChannelID:     uint8(chanField >> 12),
		ChannelNumber: chanField & 0x0FFF,
	}, true
}

// ServiceOptions decodes the 8-bit SVC_OPTIONS field a P25 voice grant
// carries (TIA-102.AABF, reused by the Phase 2 MAC). Bit 7 = Emergency,
// bit 6 = Protected (the encryption indicator), bits 0-2 = call
// priority.
type ServiceOptions uint8

// Emergency reports the emergency call bit.
func (s ServiceOptions) Emergency() bool { return s&0x80 != 0 }

// Encrypted reports the protected (encryption) bit.
func (s ServiceOptions) Encrypted() bool { return s&0x40 != 0 }

// Priority returns the 3-bit call-priority field (0 = lowest).
func (s ServiceOptions) Priority() uint8 { return uint8(s) & 0x07 }

// EncryptionSync is a call's encryption identification: the Algorithm ID,
// Key ID and 72-bit Message Indicator. On the air it rides two places, and
// neither is a MAC message of its own: the MAC_PTT PDU at key-up (AirMAC.PTT)
// and the Encryption Synchronization Signal spread over the voice bursts
// (ESSTracker), which is how a call joined after its key-up learns them.
type EncryptionSync struct {
	AlgorithmID      uint8
	KeyID            uint16
	MessageIndicator [9]byte
}

// GroupAffiliationResponse is a Phase 2 Group Affiliation Response (0x68,
// abbreviated): a radio was granted or denied affiliation with a talkgroup.
// Octet 0 the opcode (SDRTrunk GroupAffiliationResponseAbbreviated):
//
//	2     bits 1-0 the response value (bit 7 local / global)
//	3-4   announcement group
//	5-6   group (talkgroup)
//	7-9   target unit
type GroupAffiliationResponse struct {
	Response          uint8
	AnnouncementGroup uint16
	GroupAddress      uint16
	TargetID          uint32
}

// AsGroupAffiliationResponse returns the structured response if the PDU
// opcode is OpGroupAffiliationResponse, otherwise (zero, false).
func (p MACPDU) AsGroupAffiliationResponse() (GroupAffiliationResponse, bool) {
	b := p.Payload
	if p.Opcode != OpGroupAffiliationResponse || len(b) < 9 {
		return GroupAffiliationResponse{}, false
	}
	return GroupAffiliationResponse{
		Response:          b[1] & 0x03,
		AnnouncementGroup: binary.BigEndian.Uint16(b[2:4]),
		GroupAddress:      binary.BigEndian.Uint16(b[4:6]),
		TargetID:          uint32(b[6])<<16 | uint32(b[7])<<8 | uint32(b[8]),
	}, true
}

// UnitRegistrationResponse is a Phase 2 Unit Registration Response (0x6C,
// abbreviated): a radio completed or was denied registration on a site.
// Octet 0 the opcode (SDRTrunk UnitRegistrationResponseAbbreviated):
//
//	2     bits 5-4 the response value, bits 3-0 the top of the System ID
//	3     the rest of the System ID
//	4-6   the radio's own unit ID
//	7-9   the address the site assigned it (the ID its calls carry)
//
// The abbreviated form carries no WACN; WACN stays 0.
type UnitRegistrationResponse struct {
	Response uint8
	WACN     uint32 // 20-bit; 0 (not carried by the abbreviated form)
	SystemID uint16 // 12-bit
	UnitID   uint32 // 24-bit
	SourceID uint32 // 24-bit, the assigned address
}

// AsUnitRegistrationResponse returns the structured response if the PDU
// opcode is OpUnitRegistrationResponse, otherwise (zero, false).
func (p MACPDU) AsUnitRegistrationResponse() (UnitRegistrationResponse, bool) {
	b := p.Payload
	if p.Opcode != OpUnitRegistrationResponse || len(b) < 9 {
		return UnitRegistrationResponse{}, false
	}
	return UnitRegistrationResponse{
		Response: b[1] >> 4 & 0x03,
		SystemID: uint16(b[1]&0x0F)<<8 | uint16(b[2]),
		UnitID:   uint32(b[3])<<16 | uint32(b[4])<<8 | uint32(b[5]),
		SourceID: uint32(b[6])<<16 | uint32(b[7])<<8 | uint32(b[8]),
	}, true
}

// AsPushToTalk extracts the Encryption Sync (ALGID/KID/MI) that a P25
// Phase 2 MAC_PTT message carries at key-up, for the older superframe
// working model's callers (the air path uses AirMAC.PTT). The PTT message is
// identified by its slot type (SlotTypeMACPTT), not by a MAC opcode, so this
// does NOT gate on the opcode byte. Callers must only invoke it for a
// PTT-slot PDU (see DecodeSuperframeMACPDUsWithSlot), otherwise a long
// enough non-PTT payload would parse as garbage.
//
// Layout note: field offsets follow SDRtrunk's PushToTalk MacStructure —
// MESSAGE_INDICATOR at message bits 8-79 (72 bits), ALGORITHM_ID at
// 80-87, KEY_ID at 88-103. Relative to the MACPDU payload, which starts
// after the 1-octet opcode (message bits 0-7), that is:
//
//	bytes 0-8  : 72-bit Message Indicator
//	byte 9     : Algorithm ID
//	bytes 10-11: Key ID
//
// This is the project's working model pending issue #813 on-air
// confirmation (the repo has no TIA-102.BBAC PDF). It is confined here
// so a correction is a one-file change, and the grant's ServiceOptions
// "protected" bit flags encryption independently, so a wrong layout
// degrades gracefully to today's behaviour (Encrypted true, alg/key 0).
func (p MACPDU) AsPushToTalk() (EncryptionSync, bool) {
	if len(p.Payload) < 12 {
		return EncryptionSync{}, false
	}
	es := EncryptionSync{
		AlgorithmID: p.Payload[9],
		KeyID:       binary.BigEndian.Uint16(p.Payload[10:12]),
	}
	copy(es.MessageIndicator[:], p.Payload[0:9])
	return es, true
}
