package phase2

import (
	"encoding/binary"
	"fmt"
)

// Manufacturer-specific P25 Phase 2 MAC messages: opcodes 0x80..0xBF, whose
// second octet is the MFID. The same opcode value is a different message
// under each manufacturer, so every accessor here checks the (MFID, Opcode)
// pair, and MACPDU.Name names a message from both.
//
// Opcodes and field positions are SDRTrunk's (its Motorola MacStructure
// classes), and the group regroup layout is checked against a real MMR
// SACCH (realair_mmr_test.go). Until issue #1364 the regroup parser read the
// supergroup one octet early (from the message's length octet), "patch
// delete" was opcode 0x83 (which is a regroup voice channel update: a call in
// progress), and a Harris regroup and a plain-text talker alias were parsed
// from opcodes 0x81 / 0x82 with layouts no reference has.

// Manufacturer IDs (MFID) per the TIA-registered P25 manufacturer list.
const (
	MFIDStandard    uint8 = 0x00 // standard TIA-102 messages
	MFIDStandardAlt uint8 = 0x01 // also-standard MFID some systems emit
	MFIDMotorola    uint8 = 0x90
	MFIDHarris      uint8 = 0xA4
)

// Vendor MAC opcodes. They live in the manufacturer-specific opcode range
// (Opcode.IsManufacturerSpecific); the MFID says whose message it is.
const (
	// Motorola (MFID 0x90) group regroup, the "patch" that joins member
	// talkgroups under one supergroup address.
	OpMotorolaRegroupVoiceChannelUser         Opcode = 0x80 // in-call talker of a regroup call
	OpMotorolaRegroupAdd                      Opcode = 0x81
	OpMotorolaRegroupVoiceChannelUpdate       Opcode = 0x83 // a regroup call in progress
	OpMotorolaRegroupDelete                   Opcode = 0x89
	OpMotorolaRegroupVoiceChannelUserExtended Opcode = 0xA0

	// OpVendorGroupRegroup is OpMotorolaRegroupAdd's older name.
	OpVendorGroupRegroup = OpMotorolaRegroupAdd
	// OpMotorolaPatchDelete is OpMotorolaRegroupDelete's older name.
	OpMotorolaPatchDelete = OpMotorolaRegroupDelete

	// OpVendorTalkerAlias is the older working model's plain-ASCII alias
	// fragment (0x82). It matches nothing on the air (Motorola's 0x82 lists
	// a group's active radios) and the control channel no longer acts on it;
	// it remains for the assembler's own tests. The real Motorola Phase 2
	// alias is OpMotorolaAlias{Header,Data} below.
	OpVendorTalkerAlias Opcode = 0x82

	// OpMotorolaAliasHeader / OpMotorolaAliasData carry the real
	// Motorola Phase 2 talker alias (#376). SDRTrunk ground truth on
	// Victorian MMR shows it rides on FACCH-S during hangtime as a
	// HEADER opcode 0x91 + DATA opcodes 0x95 (MFID 0x90). The fragments
	// reassemble to the same Motorola message framing as Phase 1
	// (WACN|System|RadioID|cipher-alias|CRC-16) — so the inline source
	// RID falls out of the reassembled prefix and the shared
	// internal/radio/p25/motorola decode applies. See talker_alias.go's
	// AsMotorolaAliasHeader / AsMotorolaAliasData + MotorolaAliasAssembler.
	OpMotorolaAliasHeader Opcode = 0x91
	OpMotorolaAliasData   Opcode = 0x95
)

// vendorOpcodeNames names the manufacturer-specific messages SDRTrunk
// identifies, keyed MFID<<8 | opcode.
var vendorOpcodeNames = map[uint16]string{
	0x9080: "Motorola.GroupRegroupVoiceChannelUser",
	0x9081: "Motorola.GroupRegroupAdd",
	0x9082: "Motorola.ActiveGroupRadios",
	0x9083: "Motorola.GroupRegroupVoiceChannelUpdate",
	0x9085: "Motorola.GroupRegroupExtendedFunctionCommand",
	0x9089: "Motorola.GroupRegroupDelete",
	0x908B: "Motorola.TDMADataChannel",
	0x908F: "Motorola.ActiveGroupRadios",
	0x9091: "Motorola.TalkerAliasHeader",
	0x9095: "Motorola.TalkerAliasDataBlock",
	0x90A0: "Motorola.GroupRegroupVoiceChannelUserExtended",
	0x90A3: "Motorola.GroupRegroupChannelGrant",
	0x90A4: "Motorola.GroupRegroupChannelGrantExplicit",
	0x90A5: "Motorola.GroupRegroupChannelGrantUpdate",
	0x90A6: "Motorola.QueuedResponse",
	0x90A7: "Motorola.DenyResponse",
	0x90A8: "Motorola.AcknowledgeResponse",
	0x90BF: "Motorola.ActiveGroupRadios",
	0xA4A0: "Harris.PrivateDataChannelGrant",
	0xA4A8: "Harris.TalkerAlias",
	0xA4AA: "Harris.GPSLocation",
	0xA4AC: "Harris.UnitToUnitDataChannelGrant",
	0xA4B0: "Harris.GroupRegroupExplicitEncryptionCommand",
}

// ManufacturerName returns a human-readable label for an MFID.
func ManufacturerName(mfid uint8) string {
	switch mfid {
	case MFIDStandard, MFIDStandardAlt:
		return "Standard"
	case MFIDMotorola:
		return "Motorola"
	case MFIDHarris:
		return "Harris"
	default:
		return fmt.Sprintf("MFID(%02X)", mfid)
	}
}

// MotorolaPatchGroup is a Motorola group regroup add or delete command: the
// supergroup address and the member talkgroups it names.
type MotorolaPatchGroup struct {
	SuperGroup uint16
	Patched    []uint16
}

// motorolaRegroup parses the layout the add (0x81) and delete (0x89)
// commands share. Octet 0 the opcode (SDRTrunk MotorolaGroupRegroupAdd
// Command / DeleteCommand):
//
//	1     MFID 0x90
//	2     the message's length in octets
//	3-4   supergroup
//	5..   up to six member talkgroups, two octets each
//
// A real MMR SACCH carries 81 90 0b 0c82 00c9 00c9 00c9: length 11,
// supergroup 3202, talkgroup 201 three times. A message repeats a member to
// fill itself, so duplicates (and zero) are dropped.
func (p MACPDU) motorolaRegroup(op Opcode) (MotorolaPatchGroup, bool) {
	b := p.Payload
	if p.Opcode != op || p.MFID != MFIDMotorola || len(b) < 3 {
		return MotorolaPatchGroup{}, false
	}
	// Payload starts at octet 2, so the message's last octet is b[length-3].
	end := len(b)
	if n := int(b[0]&0x3F) - 2; n >= 3 && n < end {
		end = n
	}
	g := MotorolaPatchGroup{SuperGroup: binary.BigEndian.Uint16(b[1:3])}
next:
	for i := 3; i+2 <= end && len(g.Patched) < 6; i += 2 {
		tg := binary.BigEndian.Uint16(b[i : i+2])
		if tg == 0 {
			continue
		}
		for _, have := range g.Patched {
			if have == tg {
				continue next
			}
		}
		g.Patched = append(g.Patched, tg)
	}
	return g, true
}

// AsMotorolaPatchGroup returns the supergroup and members of a Motorola
// group regroup add command, otherwise (zero, false).
func (p MACPDU) AsMotorolaPatchGroup() (MotorolaPatchGroup, bool) {
	return p.motorolaRegroup(OpMotorolaRegroupAdd)
}

// AsMotorolaPatchDelete returns the supergroup address a Motorola group
// regroup delete command names, otherwise (0, false).
func (p MACPDU) AsMotorolaPatchDelete() (uint32, bool) {
	g, ok := p.motorolaRegroup(OpMotorolaRegroupDelete)
	return uint32(g.SuperGroup), ok
}

// MotorolaRegroupVoiceChannelUser is the in-call message of a Motorola
// regroup (patch) call: the supergroup the call is on and the radio talking.
// A patched call sends this in place of GROUP_VOICE_CHANNEL_USER.
type MotorolaRegroupVoiceChannelUser struct {
	SuperGroup uint16
	SourceID   uint32
}

// AsMotorolaRegroupVoiceChannelUser returns the in-call user of a Motorola
// regroup call, otherwise (zero, false). Octet 0 the opcode (SDRTrunk
// MotorolaGroupRegroupVoiceChannelUserAbbreviated / Extended):
//
//	0x80   1 MFID | 2 (see below) | 3-4 supergroup | 5-7 source
//	0xA0   1 MFID | 2 length | 3 service options | 4-5 supergroup |
//	       6-8 source | 9.. the talker's SUID
//
// Octet 2 of the 0x80 form is service options to SDRTrunk and a length to
// OP25 (whose message walk reads it as one); it is not used here.
func (p MACPDU) AsMotorolaRegroupVoiceChannelUser() (MotorolaRegroupVoiceChannelUser, bool) {
	if p.MFID != MFIDMotorola {
		return MotorolaRegroupVoiceChannelUser{}, false
	}
	b := p.Payload
	var at int // payload offset of the supergroup
	switch {
	case p.Opcode == OpMotorolaRegroupVoiceChannelUser && len(b) >= 6:
		at = 1
	case p.Opcode == OpMotorolaRegroupVoiceChannelUserExtended && len(b) >= 7:
		at = 2
	default:
		return MotorolaRegroupVoiceChannelUser{}, false
	}
	return MotorolaRegroupVoiceChannelUser{
		SuperGroup: binary.BigEndian.Uint16(b[at:]),
		SourceID:   uint32(b[at+2])<<16 | uint32(b[at+3])<<8 | uint32(b[at+4]),
	}, true
}
