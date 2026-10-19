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
//	byte 1     : MFID — present only for a manufacturer-specific
//	             opcode (Opcode.IsManufacturerSpecific); absent for
//	             standard opcodes.
//	bytes N..  : opcode-specific payload (typically up to 17 bytes
//	             across the MAC slot's 21-byte "MAC PDU SLOT" field;
//	             exact length depends on opcode + format).
//
// The structure is intentionally permissive: callers parse the
// opcode (and MFID, for vendor PDUs) and then dispatch to a per-opcode
// accessor — see mac_standard.go and mac_vendor.go.
type MACPDU struct {
	Opcode  Opcode
	MFID    uint8  // Manufacturer ID; MFIDStandard (0) for standard PDUs
	Payload []byte // copy of the opcode-specific payload bytes
}

// Opcode is the MAC PDU opcode field. Values follow TIA-102.AABF /
// BBAB Table 8.x; only the subset relevant to trunking follow-along
// is enumerated below.
type Opcode uint8

const (
	OpUnknown                           Opcode = 0x00
	OpGroupVoiceChannelUserAbbreviated  Opcode = 0x01 // In-call group voice channel user (abbreviated)
	OpMACEnd                            Opcode = 0x02 // End of transmission
	OpMACIdle                           Opcode = 0x03 // Channel idle
	OpMACHangtime                       Opcode = 0x05 // Hang-time
	OpMACActive                         Opcode = 0x06 // Late-grant active
	OpGroupVoiceChannelUserExtended     Opcode = 0x21 // In-call group voice channel user (extended; carries SUID)
	OpGroupVoiceChannelGrantUpdate      Opcode = 0x40
	OpGroupVoiceChannelGrant            Opcode = 0x44
	OpUnitToUnitGrantUpdateAbbreviated  Opcode = 0x46
	OpUnitToUnitVoiceChannelGrant       Opcode = 0x48
	OpUnitToUnitVoiceChannelGrantUpdate Opcode = 0x49
	OpGroupAffiliationResponse          Opcode = 0x4C
	OpUnitRegistrationResponse          Opcode = 0x4D
	OpEncryptionSync                    Opcode = 0x70
	OpIdentifierUpdate                  Opcode = 0x7D
	// TDMA-form identifier updates (SDRTrunk MacOpcode PHASE1_73 /
	// PHASE1_F3): the abbreviated form carries the IDEN_UP_TDMA field group
	// straight after the opcode; the extended form (OP25 op 0xf3) has one
	// octet before it and WACN + System ID after it. Issue #1273.
	OpIdentifierUpdateTDMAAbbreviated Opcode = 0x73
	OpIdentifierUpdateTDMAExtended    Opcode = 0xF3
	OpNetworkStatusBroadcastUpdate    Opcode = 0xFB // explicit form
	// OpNetworkStatusBroadcastImplicit is the NSB's implicit form (one
	// channel field instead of a transmit / receive pair).
	OpNetworkStatusBroadcastImplicit Opcode = 0x7B
	OpRFSSStatusBroadcastUpdate      Opcode = 0xFA
)

func (o Opcode) String() string {
	switch o {
	case OpGroupVoiceChannelUserAbbreviated:
		return "GroupVoiceChannelUserAbbreviated"
	case OpGroupVoiceChannelUserExtended:
		return "GroupVoiceChannelUserExtended"
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
	case OpUnitToUnitGrantUpdateAbbreviated:
		return "UnitToUnitGrantUpdateAbbreviated"
	case OpUnitToUnitVoiceChannelGrant:
		return "UnitToUnitVoiceChannelGrant"
	case OpUnitToUnitVoiceChannelGrantUpdate:
		return "UnitToUnitVoiceChannelGrantUpdate"
	case OpGroupAffiliationResponse:
		return "GroupAffiliationResponse"
	case OpUnitRegistrationResponse:
		return "UnitRegistrationResponse"
	case OpVendorGroupRegroup:
		return "VendorGroupRegroup"
	case OpVendorTalkerAlias:
		return "VendorTalkerAlias"
	case OpMotorolaAliasHeader:
		return "MotorolaAliasHeader"
	case OpMotorolaAliasData:
		return "MotorolaAliasData"
	case OpMotorolaPatchDelete:
		return "MotorolaPatchDelete"
	case OpEncryptionSync:
		return "EncryptionSync"
	case OpIdentifierUpdate:
		return "IdentifierUpdate"
	case OpIdentifierUpdateTDMAAbbreviated:
		return "IdentifierUpdateTDMAAbbreviated"
	case OpIdentifierUpdateTDMAExtended:
		return "IdentifierUpdateTDMAExtended"
	case OpNetworkStatusBroadcastUpdate:
		return "NetworkStatusBroadcastUpdate"
	case OpNetworkStatusBroadcastImplicit:
		return "NetworkStatusBroadcastImplicit"
	case OpRFSSStatusBroadcastUpdate:
		return "RFSSStatusBroadcastUpdate"
	default:
		return fmt.Sprintf("Opcode(%02X)", uint8(o))
	}
}

// ParseMACPDU consumes 18-byte MAC PDU information bytes (opcode +
// up to 17 payload bytes, the standard size after FEC removal) and
// returns the structured PDU. A manufacturer-specific opcode carries
// its 8-bit MFID in the octet right after the opcode; standard opcodes
// do not, so their payload starts at byte 1 exactly as before.
func ParseMACPDU(info []byte) (MACPDU, error) {
	if len(info) < 1 {
		return MACPDU{}, errors.New("p25/phase2: MAC PDU info needs at least 1 byte")
	}
	pdu := MACPDU{Opcode: Opcode(info[0])}
	rest := info[1:]
	if pdu.Opcode.IsManufacturerSpecific() && len(rest) >= 1 {
		pdu.MFID = rest[0]
		rest = rest[1:]
	}
	if len(rest) > 0 {
		pdu.Payload = make([]byte, len(rest))
		copy(pdu.Payload, rest)
	}
	return pdu, nil
}

// AssembleMACPDU re-packs a MAC PDU into bytes (opcode + optional MFID
// + payload). Used by tests; encoder support for Phase 2 isn't a
// project goal.
func AssembleMACPDU(p MACPDU) []byte {
	if p.Opcode.IsManufacturerSpecific() {
		out := make([]byte, 2+len(p.Payload))
		out[0] = byte(p.Opcode)
		out[1] = p.MFID
		copy(out[2:], p.Payload)
		return out
	}
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

// NetworkStatusBroadcast is the structured shape of a Phase 2 Network
// Status Broadcast MAC message (0xFB explicit, 0x7B implicit). It carries
// the system identity the PN44 descrambler's seed is computed from
// (TIA-102.BBAC-1 §7.2.5): WACN, System ID and the Color Code, which
// equals the Phase 1 NAC.
//
// Message layout, byte 0 the opcode (SDRTrunk NetworkStatusBroadcast
// Explicit / Implicit; OP25 tk_p25.py op 0xfb reads the same WACN,
// System ID and channel bytes):
//
//	1       LRA
//	2..4    WACN (20 bits, top of 2..4)
//	4..5    System ID (low nibble of 4, then 5)
//	6..7    channel (explicit: the transmit channel)
//	8..9    explicit: receive channel     | implicit: 8 service class,
//	10      explicit: service class       |   9..10 Color Code (low 12)
//	11..12  explicit: Color Code (low 12)
//
// GopherTrunk used to read the Color Code from bytes 6..7 — the channel —
// so a site's own NSB overwrote the descramble seed with a wrong one.
type NetworkStatusBroadcast struct {
	LRA           uint8  // 8-bit Location Registration Area
	WACN          uint32 // 20-bit Wide-Area Communication Network ID
	SystemID      uint16 // 12-bit System ID
	ColorCode     uint16 // 12-bit Color Code = Phase 1 NAC per spec
	ChannelID     uint8  // the (transmit) channel's identifier
	ChannelNumber uint16
	ServiceClass  uint8
}

// AsNetworkStatusBroadcast returns the structured NSB if the PDU is either
// NSB form with a payload long enough for its Color Code (11 bytes
// explicit, 9 implicit), otherwise (zero, false).
func (p MACPDU) AsNetworkStatusBroadcast() (NetworkStatusBroadcast, bool) {
	var svc, cc int // payload offsets of the service class and Color Code
	switch p.Opcode {
	case OpNetworkStatusBroadcastUpdate:
		svc, cc = 9, 10
	case OpNetworkStatusBroadcastImplicit:
		svc, cc = 7, 8
	default:
		return NetworkStatusBroadcast{}, false
	}
	if len(p.Payload) < cc+2 {
		return NetworkStatusBroadcast{}, false
	}
	chanField := binary.BigEndian.Uint16(p.Payload[5:7])
	return NetworkStatusBroadcast{
		LRA:           p.Payload[0],
		WACN:          uint32(p.Payload[1])<<12 | uint32(p.Payload[2])<<4 | uint32(p.Payload[3]>>4),
		SystemID:      uint16(p.Payload[3]&0x0F)<<8 | uint16(p.Payload[4]),
		ColorCode:     uint16(p.Payload[cc]&0x0F)<<8 | uint16(p.Payload[cc+1]),
		ChannelID:     uint8(chanField >> 12),
		ChannelNumber: chanField & 0x0FFF,
		ServiceClass:  p.Payload[svc],
	}, true
}

// AsIdentifierUpdate returns the structured band-plan definition if the
// PDU is an identifier update — the FDMA form (OpIdentifierUpdate) or
// either TDMA form — otherwise (zero, false). The ControlChannel feeds the
// result to its BandPlan so later voice grants resolve a downlink
// frequency.
func (p MACPDU) AsIdentifierUpdate() (IdentifierUpdate, bool) {
	switch p.Opcode {
	case OpIdentifierUpdate:
		return ParseIdentifierUpdate(p.Payload)
	case OpIdentifierUpdateTDMAAbbreviated:
		return ParseIdentifierUpdateTDMA(p.Payload)
	case OpIdentifierUpdateTDMAExtended:
		if len(p.Payload) < 9 {
			return IdentifierUpdate{}, false
		}
		return ParseIdentifierUpdateTDMA(p.Payload[1:])
	}
	return IdentifierUpdate{}, false
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
	case OpGroupVoiceChannelUserAbbreviated, OpGroupVoiceChannelUserExtended,
		OpMACEnd, OpMACIdle, OpMACHangtime, OpMACActive,
		OpGroupVoiceChannelGrant, OpGroupVoiceChannelGrantUpdate,
		OpUnitToUnitGrantUpdateAbbreviated, OpUnitToUnitVoiceChannelGrant,
		OpUnitToUnitVoiceChannelGrantUpdate, OpGroupAffiliationResponse,
		OpUnitRegistrationResponse,
		OpIdentifierUpdate, OpIdentifierUpdateTDMAAbbreviated,
		OpIdentifierUpdateTDMAExtended, OpVendorGroupRegroup, OpVendorTalkerAlias,
		OpMotorolaAliasHeader, OpMotorolaAliasData,
		OpMotorolaPatchDelete, OpEncryptionSync,
		OpNetworkStatusBroadcastUpdate, OpNetworkStatusBroadcastImplicit,
		OpRFSSStatusBroadcastUpdate:
		return true
	}
	return false
}

// GroupVoiceChannelUser is the structured shape of an in-call
// GROUP_VOICE_CHANNEL_USER MAC PDU — the traffic-channel broadcast a
// Phase 2 system emits during an active call to identify the source
// radio + carry the SVC_OPTIONS for the call. The CC's grant on
// real systems often arrives without these fields (compressed grant
// form: src=0, enc=false), so the source RID + encryption flag
// arrive in-call via this PDU on the voice channel. The voice
// composer publishes a trunking.CallSourceUpdate when one decodes
// so the trunking engine can backfill the bound ActiveCall.
//
// Both the Abbreviated form (0x01) and the Extended form (0x21)
// share the same layout for the first three fields per SDRTrunk's
// GroupVoiceChannelUserAbbreviated.java + Extended.java +
// MacStructureGroupVoiceService.java + MacStructureVoiceService.java
// (verified during PR drafting): SVC_OPTIONS at SDRTrunk octet 2,
// GROUP_ADDRESS at octets 3-4, SOURCE_ADDRESS at octets 5-7. The
// Extended form adds a 64-bit SUID (WACN+System+ID) after the
// source address; those fields aren't surfaced here yet — only the
// in-call source ID + svc_opts matter for the backfill flow.
type GroupVoiceChannelUser struct {
	ServiceOptions uint8
	GroupAddress   uint16
	SourceID       uint32 // 24-bit source radio ID
}

// AsGroupVoiceChannelUser returns the structured in-call user
// broadcast if the PDU opcode is OpGroupVoiceChannelUserAbbreviated
// or OpGroupVoiceChannelUserExtended, otherwise (zero, false). The
// payload needs at least 6 bytes to cover svc_opts (1) +
// group_address (2) + source_id (3); the Extended form's SUID tail
// is optional and ignored here.
func (p MACPDU) AsGroupVoiceChannelUser() (GroupVoiceChannelUser, bool) {
	switch p.Opcode {
	case OpGroupVoiceChannelUserAbbreviated, OpGroupVoiceChannelUserExtended:
	default:
		return GroupVoiceChannelUser{}, false
	}
	if len(p.Payload) < 6 {
		return GroupVoiceChannelUser{}, false
	}
	return GroupVoiceChannelUser{
		ServiceOptions: p.Payload[0],
		GroupAddress:   binary.BigEndian.Uint16(p.Payload[1:3]),
		SourceID:       uint32(p.Payload[3])<<16 | uint32(p.Payload[4])<<8 | uint32(p.Payload[5]),
	}, true
}

// IsManufacturerSpecific reports whether the opcode falls in the P25
// MAC manufacturer-specific range (0x80..0xBF). A PDU with such an
// opcode carries an MFID octet right after the opcode; the vendor
// accessors in mac_vendor.go dispatch on the (MFID, Opcode) pair.
func (o Opcode) IsManufacturerSpecific() bool {
	return o >= 0x80 && o <= 0xBF
}
