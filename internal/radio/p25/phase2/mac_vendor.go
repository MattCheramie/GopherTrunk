package phase2

import (
	"encoding/binary"
	"fmt"
)

// Manufacturer-specific P25 Phase 2 MAC PDUs. SDRtrunk organises its
// P25 message taxonomy into standard / Motorola / Harris / unknown
// namespaces; this file is the Phase 2 MAC equivalent — vendor accessors
// that dispatch on the (MFID, Opcode) pair so the same opcode value
// decodes differently per manufacturer.
//
// A manufacturer-specific PDU carries its MFID in the octet right after
// the opcode (see ParseMACPDU); the opcode itself sits in the
// Opcode.IsManufacturerSpecific range. The opcode value and per-vendor
// payload layouts below are the project's working model — TIA-102 vendor
// extensions are not in the repo's spec PDFs — and are confined to this
// file so a correction stays local.

// Manufacturer IDs (MFID) per the TIA-registered P25 manufacturer list.
const (
	MFIDStandard    uint8 = 0x00 // standard TIA-102 messages
	MFIDStandardAlt uint8 = 0x01 // also-standard MFID some systems emit
	MFIDMotorola    uint8 = 0x90
	MFIDHarris      uint8 = 0xA4
)

// Vendor MAC opcodes. They live in the manufacturer-specific opcode
// range (Opcode.IsManufacturerSpecific); the MFID disambiguates which
// manufacturer's message a given opcode value carries.
const (
	// OpVendorGroupRegroup is a group-regroup / patch message. Decoded
	// as a Motorola patch group (AsMotorolaPatchGroup) under MFID 0x90
	// and as a Harris regroup (AsHarrisRegroup) under MFID 0xA4.
	OpVendorGroupRegroup Opcode = 0x81
)

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

// MotorolaPatchGroup is a Motorola group-regroup ("patch") MAC PDU
// (MFID 0x90, opcode OpVendorGroupRegroup): it aggregates member
// talkgroups under one super-group address so a patched call is heard
// on every member group.
//
//	bytes 0-1 : super-group address
//	bytes 2-7 : up to 3 member talkgroups (16 bits each; 0 = unused)
type MotorolaPatchGroup struct {
	SuperGroup uint16
	Patched    []uint16
}

// AsMotorolaPatchGroup returns the structured patch group if the PDU is
// a Motorola group-regroup, otherwise (zero, false).
func (p MACPDU) AsMotorolaPatchGroup() (MotorolaPatchGroup, bool) {
	if p.Opcode != OpVendorGroupRegroup || p.MFID != MFIDMotorola {
		return MotorolaPatchGroup{}, false
	}
	if len(p.Payload) < 8 {
		return MotorolaPatchGroup{}, false
	}
	g := MotorolaPatchGroup{SuperGroup: binary.BigEndian.Uint16(p.Payload[0:2])}
	for i := 2; i+2 <= 8; i += 2 {
		if tg := binary.BigEndian.Uint16(p.Payload[i : i+2]); tg != 0 {
			g.Patched = append(g.Patched, tg)
		}
	}
	return g, true
}

// HarrisRegroup is a Harris group-regroup MAC PDU (MFID 0xA4, opcode
// OpVendorGroupRegroup): it points one regroup talkgroup at a target
// unit.
//
//	bytes 0-1 : regroup talkgroup
//	bytes 2-4 : target unit ID (24 bits)
type HarrisRegroup struct {
	RegroupGroup uint16
	TargetID     uint32
}

// AsHarrisRegroup returns the structured regroup if the PDU is a Harris
// group-regroup, otherwise (zero, false).
func (p MACPDU) AsHarrisRegroup() (HarrisRegroup, bool) {
	if p.Opcode != OpVendorGroupRegroup || p.MFID != MFIDHarris {
		return HarrisRegroup{}, false
	}
	if len(p.Payload) < 5 {
		return HarrisRegroup{}, false
	}
	return HarrisRegroup{
		RegroupGroup: binary.BigEndian.Uint16(p.Payload[0:2]),
		TargetID:     uint32(p.Payload[2])<<16 | uint32(p.Payload[3])<<8 | uint32(p.Payload[4]),
	}, true
}
