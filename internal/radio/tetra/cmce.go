package tetra

import (
	"encoding/binary"
	"fmt"
)

// PDUType is the 4-bit type field that follows the discriminator.
// Values follow ETSI EN 300 392-2 Table 14.x; only the trunking-grant
// subset is enumerated here.
type PDUType uint8

const (
	// CMCE PDU types (Disc = DiscCMCE).
	CMCEDSetup          PDUType = 0x1 // D-SETUP — incoming call setup
	CMCEDConnect        PDUType = 0x2 // D-CONNECT — call connected (carries grant)
	CMCEDRelease        PDUType = 0x4 // D-RELEASE — call released
	CMCEDTxCeased       PDUType = 0x5 // D-TX-CEASED — talker stopped
	CMCEDTxGranted      PDUType = 0x7 // D-TX-GRANTED — late-grant transmission
	CMCEDInfo           PDUType = 0x9 // D-INFO — supplementary services
	CMCEDCallProceeding PDUType = 0xA // D-CALL-PROCEEDING

	// MLE PDU types (Disc = DiscMLE).
	MLESystemInfo PDUType = 0x3 // SYSINFO — system identity broadcast
)

// String renders a stable label for log output. The string includes
// the discriminator the type was observed under so that opcode IDs
// that overlap across sub-protocols stay distinguishable.
func (p PDU) TypeString() string {
	if p.IsCMCE() {
		switch PDUType(p.Type) {
		case CMCEDSetup:
			return "D-SETUP"
		case CMCEDConnect:
			return "D-CONNECT"
		case CMCEDRelease:
			return "D-RELEASE"
		case CMCEDTxCeased:
			return "D-TX-CEASED"
		case CMCEDTxGranted:
			return "D-TX-GRANTED"
		case CMCEDInfo:
			return "D-INFO"
		case CMCEDCallProceeding:
			return "D-CALL-PROCEEDING"
		}
	}
	if p.IsMLE() && PDUType(p.Type) == MLESystemInfo {
		return "MLE-SYSINFO"
	}
	return fmt.Sprintf("%s/Type(%X)", p.Disc, p.Type)
}

// VoiceGrant is the structured shape of a CMCE D-CONNECT PDU. The
// fields surface what a trunking follower needs to retune a Voice
// device to the assigned slot:
//
//	bytes 0-1  Call Identifier (14 bits) + flags
//	bytes 2-4  Source SSI (24 bits)
//	bytes 5-7  Destination SSI (24 bits)
//	byte  8    Communication type / flags
//	bytes 9-10 Carrier Number (12 bits) + Timeslot (2 bits) + flags
//
// The exact bit positions follow the most-cited public reference for
// CMCE D-CONNECT; vendor extensions repurpose the high bits of
// byte 8 and the trailing bytes. Cross-check before trusting live
// captures.
type VoiceGrant struct {
	CallIdentifier uint16 // 14-bit
	SourceSSI      uint32 // 24-bit
	DestSSI        uint32 // 24-bit
	CarrierNumber  uint16 // 12-bit
	Timeslot       uint8  // 2-bit (0..3)
	Group          bool
	Emergency      bool
	Encrypted      bool
}

// AsVoiceGrant returns the structured grant if the PDU is a CMCE
// D-CONNECT (or D-TX-GRANTED), otherwise (zero, false). Both PDUs
// carry the same channel-allocation sub-element layout.
func (p PDU) AsVoiceGrant() (VoiceGrant, bool) {
	if !p.IsCMCE() {
		return VoiceGrant{}, false
	}
	switch PDUType(p.Type) {
	case CMCEDConnect, CMCEDTxGranted:
	default:
		return VoiceGrant{}, false
	}
	if len(p.Payload) < 11 {
		return VoiceGrant{}, false
	}
	cidAndFlags := binary.BigEndian.Uint16(p.Payload[0:2])
	flagsByte := p.Payload[8]
	carrierAndSlot := binary.BigEndian.Uint16(p.Payload[9:11])
	return VoiceGrant{
		CallIdentifier: cidAndFlags >> 2, // upper 14 bits
		SourceSSI: uint32(p.Payload[2])<<16 |
			uint32(p.Payload[3])<<8 | uint32(p.Payload[4]),
		DestSSI: uint32(p.Payload[5])<<16 |
			uint32(p.Payload[6])<<8 | uint32(p.Payload[7]),
		CarrierNumber: carrierAndSlot >> 4, // upper 12 bits
		Timeslot:      uint8((carrierAndSlot >> 2) & 0x3),
		Group:         flagsByte&0x80 != 0,
		Emergency:     flagsByte&0x40 != 0,
		Encrypted:     flagsByte&0x20 != 0,
	}, true
}

// CallRelease is the structured shape of a CMCE D-RELEASE PDU. We
// surface the call identifier so a higher-layer state machine can
// match the release to a previously-seen D-CONNECT.
type CallRelease struct {
	CallIdentifier  uint16
	DisconnectCause uint8
}

// AsRelease returns the structured release if the PDU is a CMCE
// D-RELEASE, otherwise (zero, false).
func (p PDU) AsRelease() (CallRelease, bool) {
	if !p.IsCMCE() || PDUType(p.Type) != CMCEDRelease {
		return CallRelease{}, false
	}
	if len(p.Payload) < 3 {
		return CallRelease{}, false
	}
	cid := binary.BigEndian.Uint16(p.Payload[0:2]) >> 2
	return CallRelease{
		CallIdentifier:  cid,
		DisconnectCause: p.Payload[2],
	}, true
}

// SystemBroadcast is the structured shape of an MLE SYSINFO PDU. The
// network identifiers (MCC + MNC) uniquely tag a TETRA system; the
// state machine treats the first SYSINFO as the cc.locked trigger
// and surfaces the identifier in the LockState payload.
type SystemBroadcast struct {
	MCC          uint16 // 10-bit Mobile Country Code
	MNC          uint16 // 14-bit Mobile Network Code
	LocationArea uint16 // 14-bit
}

// AsSystemBroadcast returns the structured broadcast if the PDU is an
// MLE SYSINFO, otherwise (zero, false).
func (p PDU) AsSystemBroadcast() (SystemBroadcast, bool) {
	if !p.IsMLE() || PDUType(p.Type) != MLESystemInfo {
		return SystemBroadcast{}, false
	}
	if len(p.Payload) < 5 {
		return SystemBroadcast{}, false
	}
	// 10 bits MCC + 14 bits MNC + 14 bits LA = 38 bits across 5 bytes.
	mcc := (uint16(p.Payload[0]) << 2) | uint16(p.Payload[1]>>6)
	mnc := (uint16(p.Payload[1]&0x3F) << 8) | uint16(p.Payload[2])
	la := (uint16(p.Payload[3]) << 6) | uint16(p.Payload[4]>>2)
	return SystemBroadcast{
		MCC:          mcc & 0x3FF,
		MNC:          mnc & 0x3FFF,
		LocationArea: la & 0x3FFF,
	}, true
}

// IsIdle reports whether the PDU is a CMCE filler (D-INFO with no
// service indication, D-TX-CEASED) the state machine should silently
// absorb at the trunking layer.
func (p PDU) IsIdle() bool {
	if !p.IsCMCE() {
		return false
	}
	switch PDUType(p.Type) {
	case CMCEDTxCeased:
		return true
	}
	return false
}

// IsKnown reports whether the PDU's (Discriminator, Type) pair is
// one of the documented ETSI EN 300 392-2 values the state machine
// recognises. Used by SetStrictValidation to drop PDUs whose 4-bit
// type field falls in the unallocated range for the sub-protocol
// the Discriminator selects.
func (p PDU) IsKnown() bool {
	t := PDUType(p.Type)
	if p.IsCMCE() {
		switch t {
		case CMCEDSetup, CMCEDConnect, CMCEDRelease, CMCEDTxCeased,
			CMCEDTxGranted, CMCEDInfo, CMCEDCallProceeding:
			return true
		}
		return false
	}
	if p.IsMLE() {
		switch t {
		case MLESystemInfo:
			return true
		}
		return false
	}
	// MM and SDS sub-protocols don't carry trunking-relevant grants;
	// strict mode drops them entirely.
	return false
}
