package dmr

import (
	"errors"
	"fmt"
)

// FLCO is the 6-bit Full Link Control Opcode field.
type FLCO uint8

// FLCO values per ETSI TS 102 361-2 §7.1.1 Table 7.1. Only the most
// commonly-seen voice-call opcodes are listed here; vendor extensions
// live behind FID != 0.
const (
	FLCOGroupVoiceUser FLCO = 0x00 // Group voice channel user
	FLCOUnitToUnitVoice FLCO = 0x03 // Unit-to-Unit voice channel user
	FLCOTalkerAlias    FLCO = 0x04 // Talker alias header
	FLCOGPS            FLCO = 0x08 // GPS info
	FLCOTerminator     FLCO = 0x30 // Terminator
)

func (o FLCO) String() string {
	switch o {
	case FLCOGroupVoiceUser:
		return "GroupVoiceChannelUser"
	case FLCOUnitToUnitVoice:
		return "UnitToUnitVoiceChannelUser"
	case FLCOTalkerAlias:
		return "TalkerAlias"
	case FLCOGPS:
		return "GPSInfo"
	case FLCOTerminator:
		return "Terminator"
	default:
		return fmt.Sprintf("FLCO(%02X)", uint8(o))
	}
}

// FLC is a parsed 72-bit Full Link Control PDU. The Voice LC Header
// burst (DataType 0x1) and the Terminator with LC burst (DataType
// 0x2) both carry an FLC followed by a 24-bit RS(12,9) parity trailer
// inside the BPTC(196,96) info block.
//
// Layout per ETSI TS 102 361-2 §7.1 (9 octets / 72 bits):
//
//	octet 0 : PF(1) | Reserved(1) | FLCO(6)
//	octet 1 : FID
//	octet 2 : ServiceOptions
//	octet 3-5 : Destination address (24-bit; group address for
//	            FLCOGroupVoiceUser, subscriber for FLCOUnitToUnitVoice)
//	octet 6-8 : Source address (24-bit subscriber)
type FLC struct {
	PF             bool
	FLCO           FLCO
	FID            uint8
	ServiceOptions uint8
	DstAddr        uint32 // 24-bit
	SrcAddr        uint32 // 24-bit
}

// ErrFLCLength is returned by ParseFLC when the supplied buffer is
// shorter than the 9 FLC octets.
var ErrFLCLength = errors.New("dmr: FLC requires 9 octets")

// ParseFLC decodes the leading 9 octets of a BPTC(196,96)-recovered
// info block as a Full Link Control PDU. The remaining 3 octets carry
// the RS(12,9) parity trailer; verifying it is intentionally
// out-of-scope for this pass — the BPTC layer already supplies error
// correction over the same bits, and adding RS(12,9) over hexadecimal
// symbols is a separate, self-contained piece of work.
func ParseFLC(info []byte) (FLC, error) {
	if len(info) < 9 {
		return FLC{}, fmt.Errorf("%w, got %d", ErrFLCLength, len(info))
	}
	return FLC{
		PF:             info[0]&0x80 != 0,
		FLCO:           FLCO(info[0] & 0x3F),
		FID:            info[1],
		ServiceOptions: info[2],
		DstAddr:        uint32(info[3])<<16 | uint32(info[4])<<8 | uint32(info[5]),
		SrcAddr:        uint32(info[6])<<16 | uint32(info[7])<<8 | uint32(info[8]),
	}, nil
}

// AssembleFLC packs an FLC into 9 octets (the data-symbol portion of
// the 12-octet RS(12,9) frame). RS parity is not generated here;
// callers building synthetic streams concatenate three zero octets to
// reach the 12-byte info block expected by BPTC encode.
func AssembleFLC(f FLC) []byte {
	out := make([]byte, 9)
	out[0] = byte(f.FLCO) & 0x3F
	if f.PF {
		out[0] |= 0x80
	}
	out[1] = f.FID
	out[2] = f.ServiceOptions
	out[3] = byte(f.DstAddr >> 16)
	out[4] = byte(f.DstAddr >> 8)
	out[5] = byte(f.DstAddr)
	out[6] = byte(f.SrcAddr >> 16)
	out[7] = byte(f.SrcAddr >> 8)
	out[8] = byte(f.SrcAddr)
	return out
}

// GroupVoiceChannelUser is the structured shape of an FLC whose FLCO
// is FLCOGroupVoiceUser. Mirrors the EDACS / P25 voice-grant accessor
// pattern: pure data, no parsing side-effects.
type GroupVoiceChannelUser struct {
	GroupAddress uint32 // 24-bit
	SourceID     uint32 // 24-bit
	Encrypted    bool
	Emergency    bool
}

// AsGroupVoiceUser decodes the FLC into the typed group-voice payload
// when its FLCO matches; otherwise (zero, false). ServiceOptions bit 7
// is the emergency flag and bit 6 is the privacy/encrypted flag, per
// §7.2.1 Table 7.13.
func (f FLC) AsGroupVoiceUser() (GroupVoiceChannelUser, bool) {
	if f.FLCO != FLCOGroupVoiceUser {
		return GroupVoiceChannelUser{}, false
	}
	return GroupVoiceChannelUser{
		GroupAddress: f.DstAddr,
		SourceID:     f.SrcAddr,
		Emergency:    f.ServiceOptions&0x80 != 0,
		Encrypted:    f.ServiceOptions&0x40 != 0,
	}, true
}
