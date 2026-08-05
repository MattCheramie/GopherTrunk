package tetra

import (
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

// VoiceGrant is the grant a trunking follower needs to retune a Voice device to
// the assigned traffic slot. It is filled field-by-field from the bit-accurate
// MAC layer (see downlink.go publishGrantFromMAC / classifyParties): the
// physical resource (carrier + timeslot + usage marker) from the MAC-RESOURCE
// channel-allocation element, and the party SSIs / emergency flag from the CMCE
// TM-SDU riding in the same PDU.
type VoiceGrant struct {
	CallIdentifier uint16 // 14-bit
	SourceSSI      uint32 // 24-bit
	DestSSI        uint32 // 24-bit
	CarrierNumber  uint16 // 12-bit
	Timeslot       uint8  // 2-bit (0..3)
	UsageMarker    uint8  // downlink usage marker (AACH §21.4.7); 0 = none
	// Individual marks a grant whose DestSSI is an individual subscriber ISSI
	// (a unit-to-unit / individual-addressed call), not a talkgroup GSSI — so the
	// engine and UI do not surface the radio ID as a phantom talkgroup. Set by
	// classifyParties.
	Individual bool
	Emergency  bool
	Encrypted  bool
	// Priority is the 4-bit CMCE Call priority (EN 300 392-2 Table 14.46;
	// 0 = lowest, 0xF = pre-emptive priority 4 / emergency). Emergency is
	// derived from it. Zero when no CMCE priority element was decoded.
	Priority uint8
}

// Voice grants and call teardown are decoded bit-accurately from the MAC layer
// (see downlink.go ingestMAC / ParseCMCE), not from this byte-aligned PDU view:
// a real CMCE TM-SDU is bit-packed (3-bit MLE discriminator + 5-bit PDU type),
// which the byte-oriented ParsePDU cannot frame. The VoiceGrant struct below is
// retained as the grant carrier the MAC path fills.

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
