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

// Opcode is a MAC message's opcode: the first octet of each message in a
// MAC PDU. Its top two bits select a partition:
//
//	00  TDMA-only messages (in-call voice channel users, grant updates)
//	01  the Phase 1 TSBK message set in its abbreviated form: the TSBK's
//	    6-bit opcode is the low six bits, so GRP_V_CH_GRANT (TSBK 0x00) is
//	    MAC 0x40 and NET_STS_BCST (0x3B) is 0x7B
//	10  manufacturer-specific: the next octet is the MFID, and the opcode
//	    means something different under each one
//	11  the same Phase 1 set in its extended / explicit form
//
// The values are SDRTrunk's MacOpcode enum; OP25's decode_mac_msg and its
// mac_msg_len table agree on every one this package acts on. Until issue
// #1364 this table was an earlier working model's: 0x02 / 0x03 / 0x05 / 0x06
// were taken for MAC_END / MAC_IDLE / MAC_HANGTIME / MAC_ACTIVE (those are PDU
// types, AirMACType, not message opcodes), the group grant sat at 0x44 (a
// unit-to-unit grant), affiliation and registration responses at 0x4C / 0x4D
// (0x68 / 0x6C), and 0x70 (the Synchronization Broadcast, a clock) was parsed
// as an encryption sync.
type Opcode uint8

const (
	// OpNullInformation fills the rest of a PDU.
	OpNullInformation Opcode = 0x00
	// OpUnknown is OpNullInformation's older name.
	OpUnknown = OpNullInformation

	// In-call "voice channel user" messages: who is talking on this channel.
	OpGroupVoiceChannelUserAbbreviated      Opcode = 0x01
	OpUnitToUnitVoiceChannelUserAbbreviated Opcode = 0x02
	OpTelephoneInterconnectVoiceChannelUser Opcode = 0x03
	OpGroupVoiceChannelUserExtended         Opcode = 0x21 // adds the talker's SUID
	OpUnitToUnitVoiceChannelUserExtended    Opcode = 0x22

	// Grant updates: calls in progress on other channels.
	OpGroupVoiceChannelGrantUpdateMultiple         Opcode = 0x05 // three calls
	OpGroupVoiceChannelGrantUpdateMultipleExplicit Opcode = 0x25 // two, explicit channels
	OpGroupVoiceChannelGrantUpdate                 Opcode = 0x42 // two calls
	OpGroupVoiceChannelGrantUpdateExplicit         Opcode = 0xC3 // one, explicit channels

	OpNullAvoidZeroBias         Opcode = 0x08
	OpPowerControlSignalQuality Opcode = 0x30
	OpMACRelease                Opcode = 0x31

	// Grants.
	OpGroupVoiceChannelGrant            Opcode = 0x40
	OpGroupVoiceChannelGrantExplicit    Opcode = 0xC0
	OpUnitToUnitVoiceChannelGrant       Opcode = 0x44
	OpUnitToUnitVoiceChannelGrantUpdate Opcode = 0x46

	OpGroupAffiliationResponse Opcode = 0x68
	OpUnitRegistrationResponse Opcode = 0x6C

	// OpSynchronizationBroadcast carries the site's date, time and
	// superframe count. It is not an encryption sync: a call's ALGID / KID
	// / MI ride MAC_PTT (AirMAC.PTT) and the voice bursts' ESS (ESSTracker).
	OpSynchronizationBroadcast Opcode = 0x70

	// TDMA-form identifier updates (SDRTrunk MacOpcode PHASE1_73 /
	// PHASE1_F3): the abbreviated form carries the IDEN_UP_TDMA field group
	// straight after the opcode; the extended form (OP25 op 0xf3) has one
	// octet before it and WACN + System ID after it. Issue #1273.
	OpIdentifierUpdateTDMAAbbreviated Opcode = 0x73
	OpIdentifierUpdateTDMAExtended    Opcode = 0xF3
	OpIdentifierUpdate                Opcode = 0x7D

	OpRFSSStatusBroadcastImplicit Opcode = 0x7A
	OpRFSSStatusBroadcastUpdate   Opcode = 0xFA // explicit form
	// OpNetworkStatusBroadcastImplicit is the NSB's implicit form (one
	// channel field instead of a transmit / receive pair).
	OpNetworkStatusBroadcastImplicit  Opcode = 0x7B
	OpNetworkStatusBroadcastUpdate    Opcode = 0xFB // explicit form
	OpAdjacentStatusBroadcastImplicit Opcode = 0x7C
	OpAdjacentStatusBroadcastExplicit Opcode = 0xFC
)

// standardOpcodeNames names the standard MAC messages (partitions 00, 01 and
// 11), after SDRTrunk's MacOpcode.
var standardOpcodeNames = map[Opcode]string{
	0x00: "NullInformation",
	0x01: "GroupVoiceChannelUserAbbreviated",
	0x02: "UnitToUnitVoiceChannelUserAbbreviated",
	0x03: "TelephoneInterconnectVoiceChannelUser",
	0x05: "GroupVoiceChannelGrantUpdateMultiple",
	0x08: "NullAvoidZeroBias",
	0x10: "MultiFragmentContinuation",
	0x11: "IndirectGroupPagingWithoutPriority",
	0x12: "IndividualPagingWithPriority",
	0x21: "GroupVoiceChannelUserExtended",
	0x22: "UnitToUnitVoiceChannelUserExtended",
	0x25: "GroupVoiceChannelGrantUpdateMultipleExplicit",
	0x30: "PowerControlSignalQuality",
	0x31: "MACRelease",
	0x40: "GroupVoiceChannelGrant",
	0x41: "GroupVoiceServiceRequest",
	0x42: "GroupVoiceChannelGrantUpdate",
	0x44: "UnitToUnitVoiceChannelGrant",
	0x45: "UnitToUnitAnswerRequest",
	0x46: "UnitToUnitVoiceChannelGrantUpdate",
	0x48: "TelephoneInterconnectVoiceChannelGrant",
	0x49: "TelephoneInterconnectVoiceChannelGrantUpdate",
	0x4A: "TelephoneInterconnectAnswerRequest",
	0x4C: "RadioUnitMonitorCommand",
	0x52: "SNDCPDataChannelRequest",
	0x53: "SNDCPDataPageResponse",
	0x54: "SNDCPDataChannelGrant",
	0x55: "SNDCPDataPageRequest",
	0x58: "StatusUpdate",
	0x5A: "StatusQuery",
	0x5C: "MessageUpdate",
	0x5D: "RadioUnitMonitorCommandObsolete",
	0x5E: "RadioUnitMonitorEnhancedCommand",
	0x5F: "CallAlert",
	0x60: "AcknowledgeResponseFNE",
	0x61: "QueuedResponse",
	0x64: "ExtendedFunctionCommand",
	0x67: "DenyResponse",
	0x68: "GroupAffiliationResponse",
	0x6A: "GroupAffiliationQuery",
	0x6B: "LocationRegistrationResponse",
	0x6C: "UnitRegistrationResponse",
	0x6D: "UnitRegistrationCommand",
	0x6F: "DeregistrationAcknowledge",
	0x70: "SynchronizationBroadcast",
	0x71: "AuthenticationDemand",
	0x72: "AuthenticationFNEResponse",
	0x73: "IdentifierUpdateTDMAAbbreviated",
	0x74: "IdentifierUpdateVUHF",
	0x75: "TimeAndDateAnnouncement",
	0x76: "RoamingAddressCommand",
	0x77: "RoamingAddressUpdate",
	0x78: "SystemServiceBroadcast",
	0x79: "SecondaryControlChannelBroadcast",
	0x7A: "RFSSStatusBroadcastImplicit",
	0x7B: "NetworkStatusBroadcastImplicit",
	0x7C: "AdjacentStatusBroadcastImplicit",
	0x7D: "IdentifierUpdate",
	0xC0: "GroupVoiceChannelGrantExplicit",
	0xC3: "GroupVoiceChannelGrantUpdateExplicit",
	0xC4: "UnitToUnitVoiceChannelGrantExtended",
	0xC5: "UnitToUnitAnswerRequestExtended",
	0xC6: "UnitToUnitVoiceChannelGrantUpdateExtended",
	0xC7: "UnitToUnitVoiceChannelGrantUpdateExtendedLCCH",
	0xC8: "TelephoneInterconnectVoiceChannelGrantExplicit",
	0xC9: "TelephoneInterconnectVoiceChannelGrantUpdateExplicit",
	0xCB: "CallAlertExtendedLCCH",
	0xCC: "RadioUnitMonitorCommandExtended",
	0xCD: "RadioUnitMonitorCommandExtendedLCCH",
	0xCE: "MessageUpdateExtendedLCCH",
	0xCF: "UnitToUnitVoiceChannelGrantExtendedLCCH",
	0xD6: "SNDCPDataChannelAnnouncement",
	0xD8: "StatusUpdateExtended",
	0xD9: "StatusUpdateExtendedLCCH",
	0xDA: "StatusQueryExtended",
	0xDB: "StatusQueryExtendedLCCH",
	0xDC: "MessageUpdateExtended",
	0xDE: "RadioUnitMonitorEnhancedCommandExtended",
	0xDF: "CallAlertExtended",
	0xE0: "AcknowledgeResponseFNEExtended",
	0xE4: "ExtendedFunctionCommandExtended",
	0xE5: "ExtendedFunctionCommandExtendedLCCH",
	0xE8: "GroupAffiliationResponseExtended",
	0xE9: "SecondaryControlChannelBroadcastExplicit",
	0xEA: "GroupAffiliationQueryExtended",
	0xEC: "UnitRegistrationResponseExtended",
	0xF2: "AuthenticationFNEResponseExtended",
	0xF3: "IdentifierUpdateTDMAExtended",
	0xFA: "RFSSStatusBroadcastUpdate",
	0xFB: "NetworkStatusBroadcastUpdate",
	0xFC: "AdjacentStatusBroadcastExplicit",
	0xFE: "AdjacentStatusBroadcastExtendedExplicit",
}

// String names a standard opcode. A manufacturer-specific opcode
// (0x80..0xBF) prints as its number only, because what it means depends on
// the MFID that follows it: MACPDU.Name has both. Naming a vendor opcode from
// the opcode alone is how a message gets mislabelled.
func (o Opcode) String() string {
	if !o.IsManufacturerSpecific() {
		if name, ok := standardOpcodeNames[o]; ok {
			return name
		}
	}
	return fmt.Sprintf("Opcode(%02X)", uint8(o))
}

// Name names the message: the standard name, or for a manufacturer-specific
// opcode the name it has under this PDU's MFID.
func (p MACPDU) Name() string {
	if !p.Opcode.IsManufacturerSpecific() {
		return p.Opcode.String()
	}
	if name, ok := vendorOpcodeNames[uint16(p.MFID)<<8|uint16(p.Opcode)]; ok {
		return name
	}
	return fmt.Sprintf("%s.Opcode(%02X)", ManufacturerName(p.MFID), uint8(p.Opcode))
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

// GroupVoiceChannelGrant is a Phase 2 group voice grant, or one entry of a
// grant update. The explicit forms name a transmit and a receive channel;
// ChannelID / ChannelNumber are the transmit (downlink) one.
type GroupVoiceChannelGrant struct {
	ServiceOptions uint8
	ChannelID      uint8
	ChannelNumber  uint16
	GroupAddress   uint16
	SourceID       uint32 // 0 in a grant update, which names no radio
}

func macChannel(b []byte) (id uint8, number uint16) {
	f := binary.BigEndian.Uint16(b)
	return uint8(f >> 12), f & 0x0FFF
}

// AsGroupVoiceChannelGrant returns the grant if the PDU is a Group Voice
// Channel Grant, otherwise (zero, false). Message layouts, octet 0 the
// opcode (SDRTrunk GroupVoiceChannelGrantImplicit / Explicit):
//
//	0x40 implicit   1 service options | 2-3 channel | 4-5 group | 6-8 source
//	0xC0 explicit   1 service options | 2-3 transmit channel |
//	                4-5 receive channel | 6-7 group | 8-10 source
func (p MACPDU) AsGroupVoiceChannelGrant() (GroupVoiceChannelGrant, bool) {
	b := p.Payload
	var grp int // payload offset of the group address
	switch {
	case p.Opcode == OpGroupVoiceChannelGrant && len(b) >= 8:
		grp = 3
	case p.Opcode == OpGroupVoiceChannelGrantExplicit && len(b) >= 10:
		grp = 5
	default:
		return GroupVoiceChannelGrant{}, false
	}
	g := GroupVoiceChannelGrant{
		ServiceOptions: b[0],
		GroupAddress:   binary.BigEndian.Uint16(b[grp:]),
		SourceID:       uint32(b[grp+2])<<16 | uint32(b[grp+3])<<8 | uint32(b[grp+4]),
	}
	g.ChannelID, g.ChannelNumber = macChannel(b[1:])
	return g, true
}

// AsGroupVoiceChannelGrantUpdates returns the calls a Group Voice Channel
// Grant Update lists, otherwise (nil, false). A site repeats its calls in
// progress in these, on the control channel and on every traffic channel;
// they name no source radio. A message with fewer calls than it has room for
// repeats one, so duplicates are dropped. Layouts, octet 0 the opcode
// (SDRTrunk GroupVoiceChannelGrantUpdateImplicit / Explicit / Multiple
// Implicit / MultipleExplicit):
//
//	0x42   1-2 channel A | 3-4 group A | 5-6 channel B | 7-8 group B
//	0xC3   1 service options | 2-3 transmit channel | 4-5 receive channel |
//	       6-7 group
//	0x05   three of: service options | channel (2) | group (2)
//	0x25   two of: service options | transmit channel (2) |
//	       receive channel (2) | group (2)
func (p MACPDU) AsGroupVoiceChannelGrantUpdates() ([]GroupVoiceChannelGrant, bool) {
	b := p.Payload
	var out []GroupVoiceChannelGrant
	add := func(svc uint8, ch, grp []byte) {
		g := GroupVoiceChannelGrant{ServiceOptions: svc, GroupAddress: binary.BigEndian.Uint16(grp)}
		g.ChannelID, g.ChannelNumber = macChannel(ch)
		for _, have := range out {
			if have == g {
				return
			}
		}
		out = append(out, g)
	}
	switch {
	case p.Opcode == OpGroupVoiceChannelGrantUpdate && len(b) >= 8:
		add(0, b[0:], b[2:])
		add(0, b[4:], b[6:])
	case p.Opcode == OpGroupVoiceChannelGrantUpdateExplicit && len(b) >= 7:
		add(b[0], b[1:], b[5:])
	case p.Opcode == OpGroupVoiceChannelGrantUpdateMultiple && len(b) >= 15:
		for i := 0; i < 15; i += 5 {
			add(b[i], b[i+1:], b[i+3:])
		}
	case p.Opcode == OpGroupVoiceChannelGrantUpdateMultipleExplicit && len(b) >= 14:
		for i := 0; i < 14; i += 7 {
			add(b[i], b[i+1:], b[i+5:])
		}
	default:
		return nil, false
	}
	return out, true
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

// IsIdle reports whether the message is fill: Null Information or Null
// Avoid Zero Bias. AirMAC.Messages drops both, so this is true only for a
// PDU built by hand. (MAC_IDLE, MAC_HANGTIME and MAC_END_PTT are PDU types,
// AirMACType; they are not message opcodes.)
func (p MACPDU) IsIdle() bool {
	return p.Opcode == OpNullInformation || p.Opcode == OpNullAvoidZeroBias
}

// IsKnown reports whether the opcode is a defined standard MAC message, or
// lies in the manufacturer-specific range (which cannot be judged without
// its MFID). SetStrictValidation drops the rest.
func (o Opcode) IsKnown() bool {
	if o.IsManufacturerSpecific() {
		return true
	}
	_, ok := standardOpcodeNames[o]
	return ok
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

// UnitToUnitVoiceChannelUser is the in-call message of a unit-to-unit
// (private) call: the radio being called and the radio talking.
type UnitToUnitVoiceChannelUser struct {
	ServiceOptions uint8
	TargetID       uint32
	SourceID       uint32
}

// AsUnitToUnitVoiceChannelUser returns the in-call user of a unit-to-unit
// call (0x02 abbreviated, 0x22 extended, which adds the talker's SUID after
// these fields), otherwise (zero, false). Octet 0 the opcode (SDRTrunk
// UnitToUnitVoiceChannelUserAbbreviated):
//
//	1 service options | 2-4 target | 5-7 source
func (p MACPDU) AsUnitToUnitVoiceChannelUser() (UnitToUnitVoiceChannelUser, bool) {
	switch p.Opcode {
	case OpUnitToUnitVoiceChannelUserAbbreviated, OpUnitToUnitVoiceChannelUserExtended:
	default:
		return UnitToUnitVoiceChannelUser{}, false
	}
	b := p.Payload
	if len(b) < 7 {
		return UnitToUnitVoiceChannelUser{}, false
	}
	return UnitToUnitVoiceChannelUser{
		ServiceOptions: b[0],
		TargetID:       uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]),
		SourceID:       uint32(b[4])<<16 | uint32(b[5])<<8 | uint32(b[6]),
	}, true
}

// IsManufacturerSpecific reports whether the opcode falls in the P25
// MAC manufacturer-specific range (0x80..0xBF). A PDU with such an
// opcode carries an MFID octet right after the opcode; the vendor
// accessors in mac_vendor.go dispatch on the (MFID, Opcode) pair.
func (o Opcode) IsManufacturerSpecific() bool {
	return o >= 0x80 && o <= 0xBF
}
