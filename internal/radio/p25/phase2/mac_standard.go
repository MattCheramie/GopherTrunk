package phase2

import "encoding/binary"

// Structured accessors for the standard (non-manufacturer-specific) P25
// Phase 2 MAC opcodes beyond the group-voice-grant / NSB / IdentifierUp
// trio handled in mac.go. Each As...() accessor returns (struct, false)
// when the PDU's opcode does not match, so a caller can probe a PDU
// against several accessors in turn.

// UnitToUnitVoiceChannelGrant is the structured shape of a Phase 2
// unit-to-unit (private call) voice-grant MAC PDU (opcode 0x48):
//
//	byte 0    : service options
//	bytes 1-2 : channel ID + channel number (4 + 12 bits)
//	bytes 3-5 : target unit ID (24 bits)
//	bytes 6-8 : source unit ID (24 bits)
type UnitToUnitVoiceChannelGrant struct {
	ServiceOptions uint8
	ChannelID      uint8
	ChannelNumber  uint16
	TargetID       uint32
	SourceID       uint32
}

// AsUnitToUnitVoiceChannelGrant returns the structured grant if the PDU
// opcode is a unit-to-unit voice grant or grant-update, otherwise
// (zero, false).
func (p MACPDU) AsUnitToUnitVoiceChannelGrant() (UnitToUnitVoiceChannelGrant, bool) {
	switch p.Opcode {
	case OpUnitToUnitVoiceChannelGrant, OpUnitToUnitVoiceChannelGrantUpdate:
	default:
		return UnitToUnitVoiceChannelGrant{}, false
	}
	if len(p.Payload) < 9 {
		return UnitToUnitVoiceChannelGrant{}, false
	}
	chanField := binary.BigEndian.Uint16(p.Payload[1:3])
	return UnitToUnitVoiceChannelGrant{
		ServiceOptions: p.Payload[0],
		ChannelID:      uint8(chanField >> 12),
		ChannelNumber:  chanField & 0x0FFF,
		TargetID:       uint32(p.Payload[3])<<16 | uint32(p.Payload[4])<<8 | uint32(p.Payload[5]),
		SourceID:       uint32(p.Payload[6])<<16 | uint32(p.Payload[7])<<8 | uint32(p.Payload[8]),
	}, true
}

// RFSSStatusBroadcast is the structured shape of a Phase 2 RFSS Status
// Broadcast - Update MAC PDU (opcode 0xFA). It names the site the
// receiver is camped on so a scanner can log RFSS / site topology:
//
//	byte 0    : LRA (Location Registration Area)
//	bytes 1-2 : System ID (12 bits, low 12)
//	byte 3    : RFSS ID
//	byte 4    : Site ID
//	bytes 5-6 : channel ID + channel number (4 + 12 bits)
type RFSSStatusBroadcast struct {
	LRA           uint8
	SystemID      uint16 // 12-bit
	RFSS          uint8
	Site          uint8
	ChannelID     uint8
	ChannelNumber uint16
}

// AsRFSSStatusBroadcast returns the structured RFSS status if the PDU
// opcode is OpRFSSStatusBroadcastUpdate, otherwise (zero, false).
func (p MACPDU) AsRFSSStatusBroadcast() (RFSSStatusBroadcast, bool) {
	if p.Opcode != OpRFSSStatusBroadcastUpdate {
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

// EncryptionSync is the P25 Encryption Sync identification a Phase 2
// MAC PDU carries for a protected call — the Algorithm ID, Key ID, and
// 72-bit Message Indicator. GopherTrunk, like SDRtrunk, identifies
// encryption (surfaces which algorithm/key) but does not decrypt.
//
// Layout note: the encryption-sync MAC opcode (OpEncryptionSync) and
// this 12-byte payload packing are the project's working model —
// TIA-102.BBAB does not appear in the repo's spec PDFs. The accessor
// is confined here so a spec correction is a one-file change; the
// grant's ServiceOptions "protected" bit flags encryption independently
// of this, so the feature degrades gracefully if the layout is wrong.
//
//	byte 0     : Algorithm ID
//	bytes 1-2  : Key ID
//	bytes 3-11 : 72-bit Message Indicator
type EncryptionSync struct {
	AlgorithmID      uint8
	KeyID            uint16
	MessageIndicator [9]byte
}

// GroupAffiliationResponse is the structured shape of a Phase 2 Group
// Affiliation Response MAC PDU — published when a radio is granted or
// denied affiliation with a talkgroup. Layout mirrors the Phase 1 TSBK
// (phase1.GroupAffiliationResponse):
//
//	byte 0     : bits 1-0 = affiliation response value
//	bytes 1-2  : announcement group address
//	bytes 3-4  : group address (talkgroup)
//	bytes 5-7  : target unit ID (24 bits)
type GroupAffiliationResponse struct {
	Response          uint8
	AnnouncementGroup uint16
	GroupAddress      uint16
	TargetID          uint32
}

// AsGroupAffiliationResponse returns the structured response if the PDU
// opcode is OpGroupAffiliationResponse, otherwise (zero, false).
func (p MACPDU) AsGroupAffiliationResponse() (GroupAffiliationResponse, bool) {
	if p.Opcode != OpGroupAffiliationResponse {
		return GroupAffiliationResponse{}, false
	}
	if len(p.Payload) < 8 {
		return GroupAffiliationResponse{}, false
	}
	return GroupAffiliationResponse{
		Response:          p.Payload[0] & 0x03,
		AnnouncementGroup: binary.BigEndian.Uint16(p.Payload[1:3]),
		GroupAddress:      binary.BigEndian.Uint16(p.Payload[3:5]),
		TargetID:          uint32(p.Payload[5])<<16 | uint32(p.Payload[6])<<8 | uint32(p.Payload[7]),
	}, true
}

// UnitRegistrationResponse is the structured shape of a Phase 2 Unit
// Registration Response MAC PDU — published when a radio completes or
// is denied registration on a site. Layout mirrors the Phase 1 TSBK
// (phase1.UnitRegistrationResponse):
//
//	byte 0       : bits 1-0 = registration response value
//	bytes 1-3    : WACN (20 bits, upper 20 of bytes 1-3)
//	bytes 3-4    : System ID (low nibble of byte 3 + byte 4)
//	bytes 5-7    : source unit ID (24 bits)
type UnitRegistrationResponse struct {
	Response uint8
	WACN     uint32 // 20-bit
	SystemID uint16 // 12-bit
	SourceID uint32 // 24-bit
}

// AsUnitRegistrationResponse returns the structured response if the PDU
// opcode is OpUnitRegistrationResponse, otherwise (zero, false).
func (p MACPDU) AsUnitRegistrationResponse() (UnitRegistrationResponse, bool) {
	if p.Opcode != OpUnitRegistrationResponse {
		return UnitRegistrationResponse{}, false
	}
	if len(p.Payload) < 8 {
		return UnitRegistrationResponse{}, false
	}
	return UnitRegistrationResponse{
		Response: p.Payload[0] & 0x03,
		WACN:     uint32(p.Payload[1])<<12 | uint32(p.Payload[2])<<4 | uint32(p.Payload[3])>>4,
		SystemID: uint16(p.Payload[3]&0x0F)<<8 | uint16(p.Payload[4]),
		SourceID: uint32(p.Payload[5])<<16 | uint32(p.Payload[6])<<8 | uint32(p.Payload[7]),
	}, true
}

// AsEncryptionSync returns the structured Encryption Sync if the PDU
// opcode is OpEncryptionSync, otherwise (zero, false).
func (p MACPDU) AsEncryptionSync() (EncryptionSync, bool) {
	if p.Opcode != OpEncryptionSync {
		return EncryptionSync{}, false
	}
	if len(p.Payload) < 12 {
		return EncryptionSync{}, false
	}
	es := EncryptionSync{
		AlgorithmID: p.Payload[0],
		KeyID:       binary.BigEndian.Uint16(p.Payload[1:3]),
	}
	copy(es.MessageIndicator[:], p.Payload[3:12])
	return es, true
}
