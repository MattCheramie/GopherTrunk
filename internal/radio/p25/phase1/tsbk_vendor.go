package phase1

import "encoding/binary"

// Manufacturer-specific P25 Phase 1 TSBKs. SDRtrunk organises its P25
// TSBK taxonomy into standard / Motorola / Harris / unknown namespaces;
// this file is the Phase 1 vendor side — accessors that dispatch on the
// TSBK's MFID header byte (TSBK.MFID, parsed by ParseTSBK) so the same
// 6-bit opcode value decodes differently per manufacturer.
//
// The vendor opcode values and payload layouts below are the project's
// working model — TIA-102 vendor extensions are not in the repo's spec
// PDFs — and are confined to this file with symmetric Assemble* encoders
// and defensive As* accessors, so a spec correction stays local.

// Manufacturer IDs (MFID) per the TIA-registered P25 manufacturer list.
const (
	MFIDStandard    uint8 = 0x00 // standard TIA-102 TSBKs
	MFIDStandardAlt uint8 = 0x01 // also-standard MFID some systems emit
	MFIDMotorola    uint8 = 0x90
	MFIDHarris      uint8 = 0xA4
)

// Vendor TSBK opcodes. These are interpreted only under their owning
// MFID — the same 6-bit value means something else under MFIDStandard.
const (
	// OpMotorolaPatchGroupAdd / Delete are the Motorola group-regroup
	// ("patch") add and cancel commands under MFID 0x90.
	OpMotorolaPatchGroupAdd    Opcode = 0x00
	OpMotorolaPatchGroupDelete Opcode = 0x01
	// OpHarrisRegroup is the Harris dynamic-regroup command under
	// MFID 0xA4.
	OpHarrisRegroup Opcode = 0x00
	// OpVendorTalkerAlias carries one fragment of a radio's display
	// name; emitted under either vendor MFID.
	OpVendorTalkerAlias Opcode = 0x15
)

// IsVendorMFID reports whether the TSBK's MFID is a recognised
// manufacturer-specific ID (Motorola or Harris) rather than standard.
func (t TSBK) IsVendorMFID() bool {
	return t.MFID == MFIDMotorola || t.MFID == MFIDHarris
}

// MotorolaPatchGroup is a Motorola group-regroup ("patch") TSBK
// (MFID 0x90, opcode OpMotorolaPatchGroupAdd): it aggregates member
// talkgroups under one super-group address.
//
//	bytes 0-1 : super-group address
//	bytes 2-7 : up to 3 member talkgroups (16 bits each; 0 = unused)
type MotorolaPatchGroup struct {
	SuperGroup uint16
	Patched    []uint16
}

// AsMotorolaPatchGroup returns the structured patch group if the TSBK
// is a Motorola group-regroup add, otherwise (zero, false).
func (t TSBK) AsMotorolaPatchGroup() (MotorolaPatchGroup, bool) {
	if t.MFID != MFIDMotorola || t.Opcode != OpMotorolaPatchGroupAdd {
		return MotorolaPatchGroup{}, false
	}
	g := MotorolaPatchGroup{SuperGroup: binary.BigEndian.Uint16(t.Payload[0:2])}
	for i := 2; i+2 <= 8; i += 2 {
		if tg := binary.BigEndian.Uint16(t.Payload[i : i+2]); tg != 0 {
			g.Patched = append(g.Patched, tg)
		}
	}
	return g, true
}

// AssembleMotorolaPatchGroup builds the 8-byte payload; used by tests.
func AssembleMotorolaPatchGroup(g MotorolaPatchGroup) [8]byte {
	var p [8]byte
	binary.BigEndian.PutUint16(p[0:2], g.SuperGroup)
	for i, tg := range g.Patched {
		if i >= 3 {
			break
		}
		binary.BigEndian.PutUint16(p[2+i*2:4+i*2], tg)
	}
	return p
}

// AsMotorolaPatchDelete returns the super-group address a Motorola
// group-regroup delete cancels, if the TSBK is one, otherwise (0, false).
func (t TSBK) AsMotorolaPatchDelete() (uint16, bool) {
	if t.MFID != MFIDMotorola || t.Opcode != OpMotorolaPatchGroupDelete {
		return 0, false
	}
	return binary.BigEndian.Uint16(t.Payload[0:2]), true
}

// HarrisRegroup is a Harris dynamic-regroup TSBK (MFID 0xA4, opcode
// OpHarrisRegroup): it points a regroup talkgroup at a target unit.
//
//	bytes 0-1 : regroup talkgroup
//	bytes 2-4 : target unit ID (24 bits)
type HarrisRegroup struct {
	RegroupGroup uint16
	TargetID     uint32
}

// AsHarrisRegroup returns the structured regroup if the TSBK is a
// Harris dynamic-regroup, otherwise (zero, false).
func (t TSBK) AsHarrisRegroup() (HarrisRegroup, bool) {
	if t.MFID != MFIDHarris || t.Opcode != OpHarrisRegroup {
		return HarrisRegroup{}, false
	}
	return HarrisRegroup{
		RegroupGroup: binary.BigEndian.Uint16(t.Payload[0:2]),
		TargetID:     uint32(t.Payload[2])<<16 | uint32(t.Payload[3])<<8 | uint32(t.Payload[4]),
	}, true
}

// AssembleHarrisRegroup builds the 8-byte payload; used by tests.
func AssembleHarrisRegroup(r HarrisRegroup) [8]byte {
	var p [8]byte
	binary.BigEndian.PutUint16(p[0:2], r.RegroupGroup)
	p[2], p[3], p[4] = byte(r.TargetID>>16), byte(r.TargetID>>8), byte(r.TargetID)
	return p
}

// AsTalkerAliasFragment returns one talker-alias fragment if the TSBK
// is a vendor talker-alias message, otherwise (zero, false). It is
// MFID-agnostic — both Motorola and Harris alias TSBKs decode through
// it. Working-model payload layout:
//
//	bytes 0-2 : source unit ID (24 bits)
//	byte 3    : block index
//	byte 4    : block count
//	bytes 5-7 : alias data for this block
func (t TSBK) AsTalkerAliasFragment() (TalkerAliasFragment, bool) {
	if !t.IsVendorMFID() || t.Opcode != OpVendorTalkerAlias {
		return TalkerAliasFragment{}, false
	}
	return TalkerAliasFragment{
		SourceID:   uint32(t.Payload[0])<<16 | uint32(t.Payload[1])<<8 | uint32(t.Payload[2]),
		BlockIndex: t.Payload[3],
		BlockCount: t.Payload[4],
		Data:       append([]byte(nil), t.Payload[5:8]...),
	}, true
}

// AssembleTalkerAliasFragment builds the 8-byte payload; used by tests.
// Data beyond 3 bytes is truncated to the block's on-wire width.
func AssembleTalkerAliasFragment(f TalkerAliasFragment) [8]byte {
	var p [8]byte
	p[0], p[1], p[2] = byte(f.SourceID>>16), byte(f.SourceID>>8), byte(f.SourceID)
	p[3] = f.BlockIndex
	p[4] = f.BlockCount
	copy(p[5:8], f.Data)
	return p
}
