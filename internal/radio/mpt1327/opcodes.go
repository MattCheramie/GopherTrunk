package mpt1327

import "fmt"

// CodewordType is the 1-bit Codeword.Type field — 0 = address
// codeword (trunked signalling), 1 = data codeword (short messages).
type CodewordType uint8

const (
	TypeAddress CodewordType = 0
	TypeData    CodewordType = 1
)

func (t CodewordType) String() string {
	switch t {
	case TypeAddress:
		return "Address"
	case TypeData:
		return "Data"
	default:
		return fmt.Sprintf("CodewordType(%d)", uint8(t))
	}
}

// CodewordKind enumerates the address-codeword sub-types most useful
// for trunking follow-along. The kind is decoded from the upper 4
// bits of the 17-bit Function field, which the spec calls the
// "Address Categorisation" subfield.
type CodewordKind uint8

const (
	KindUnknown    CodewordKind = 0x0
	KindAloha      CodewordKind = 0x1 // ALH  — control-channel idle
	KindAhoy       CodewordKind = 0x2 // AHY  — paging / inquiry
	KindAhoyChan   CodewordKind = 0x3 // AHYC — broadcast / system info
	KindGoToChan   CodewordKind = 0x4 // GTC  — voice grant ("go to channel")
	KindAck        CodewordKind = 0x5 // ACK
	KindDisconnect CodewordKind = 0x6 // DUL — disconnect-unique-to-line
	KindData       CodewordKind = 0x7 // SAMO / data-request shorthand
	KindEmergency  CodewordKind = 0xE
)

func (k CodewordKind) String() string {
	switch k {
	case KindAloha:
		return "Aloha"
	case KindAhoy:
		return "Ahoy"
	case KindAhoyChan:
		return "AhoyChan"
	case KindGoToChan:
		return "GoToChannel"
	case KindAck:
		return "Ack"
	case KindDisconnect:
		return "Disconnect"
	case KindData:
		return "Data"
	case KindEmergency:
		return "Emergency"
	default:
		return fmt.Sprintf("CodewordKind(%X)", uint8(k))
	}
}

// Kind returns the CodewordKind embedded in the Function field's
// upper 4 bits (Address Categorisation subfield).
func (c Codeword) Kind() CodewordKind {
	return CodewordKind((c.Function >> 13) & 0xF)
}

// FunctionPayload returns the lower 13 bits of the Function field —
// the subfield that carries kind-specific data. For GTC this is the
// assigned channel number; for AHYC it carries broadcast info.
func (c Codeword) FunctionPayload() uint16 {
	return uint16(c.Function & 0x1FFF)
}

// IsAloha reports whether the codeword is an ALH idle frame on the
// control channel.
func (c Codeword) IsAloha() bool { return c.Kind() == KindAloha }

// GoToChannel is the high-level shape of an MPT 1327 GTC voice grant.
// The codeword's Prefix + Ident form the called party; the lower 13
// bits of Function carry the assigned channel number that the
// receiving radio retunes to.
type GoToChannel struct {
	Prefix  uint8
	Ident   uint16
	Channel uint16
}

// AsGoToChannel returns the structured grant if the codeword is a
// GTC, otherwise (zero, false).
func (c Codeword) AsGoToChannel() (GoToChannel, bool) {
	if c.Type != TypeAddress || c.Kind() != KindGoToChan {
		return GoToChannel{}, false
	}
	return GoToChannel{
		Prefix:  c.Prefix,
		Ident:   c.Ident,
		Channel: c.FunctionPayload(),
	}, true
}

// AhoyChannel describes an AHYC system broadcast. The Function
// payload carries a sub-system identifier the engine treats as the
// SystemID for cc.locked events.
type AhoyChannel struct {
	Prefix uint8
	Ident  uint16
	System uint16
}

// AsAhoyChannel returns the structured AHYC if applicable.
func (c Codeword) AsAhoyChannel() (AhoyChannel, bool) {
	if c.Type != TypeAddress || c.Kind() != KindAhoyChan {
		return AhoyChannel{}, false
	}
	return AhoyChannel{
		Prefix: c.Prefix,
		Ident:  c.Ident,
		System: c.FunctionPayload(),
	}, true
}
