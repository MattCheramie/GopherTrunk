package tetra

import (
	"errors"
	"fmt"
)

// Discriminator is the 4-bit tag that opens every TETRA Layer-3 PDU
// and selects the sub-protocol (MLE / MM / CMCE / SDS / …). Per
// ETSI EN 300 392-2 §14.7 the discriminator is structured:
//
//	bits 3..2  Protocol Identifier (00 = MLE, 01 = MM, 10 = CMCE,
//	           11 = SDS).
//	bits 1..0  PDU type within the sub-protocol (the upper 2 bits of
//	           the type field for that sub-protocol).
//
// The combined 4-bit value is what callers see. The per-PDU accessors
// in cmce.go interpret it.
type Discriminator uint8

const (
	DiscMLE  Discriminator = 0x0 // 00xx — MLE (Mobile Link Entity)
	DiscMM   Discriminator = 0x4 // 01xx — Mobility Management
	DiscCMCE Discriminator = 0x8 // 10xx — Circuit-Mode Control Entity
	DiscSDS  Discriminator = 0xC // 11xx — Short Data Service
)

// Protocol returns the high 2 bits of the Discriminator — the
// Protocol Identifier per the spec.
func (d Discriminator) Protocol() uint8 { return uint8(d) >> 2 }

func (d Discriminator) String() string {
	switch d & 0xC {
	case 0x0:
		return "MLE"
	case 0x4:
		return "MM"
	case 0x8:
		return "CMCE"
	case 0xC:
		return "SDS"
	}
	return fmt.Sprintf("Disc(%X)", uint8(d))
}

// PDU is one TETRA Layer-3 signalling unit. After the upstream FEC
// removes coding the structured layout is:
//
//	byte 0, bits 7..4   Discriminator (4 bits)
//	byte 0, bits 3..0   PDU type (4 bits)
//	bytes 1..N          opcode-specific information bits
//
// The structure is intentionally permissive: callers parse the
// discriminator + type, then dispatch to a per-PDU accessor.
type PDU struct {
	Disc    Discriminator
	Type    uint8 // 4-bit
	Payload []byte
}

// AssemblePDU packs a PDU back into bytes (header + payload). Used by
// tests and by any future encoder work.
func AssemblePDU(p PDU) []byte {
	out := make([]byte, 1+len(p.Payload))
	out[0] = byte((uint8(p.Disc)&0xF)<<4) | (p.Type & 0xF)
	copy(out[1:], p.Payload)
	return out
}

// ParsePDU consumes bytes emitted by AssemblePDU and returns the
// structured PDU. The payload slice in the result is a copy so the
// caller can safely modify the input afterwards.
func ParsePDU(info []byte) (PDU, error) {
	if len(info) < 1 {
		return PDU{}, errors.New("tetra: PDU info needs at least 1 byte")
	}
	pdu := PDU{
		Disc: Discriminator(info[0] >> 4),
		Type: info[0] & 0xF,
	}
	if len(info) > 1 {
		pdu.Payload = make([]byte, len(info)-1)
		copy(pdu.Payload, info[1:])
	}
	return pdu, nil
}

// PDUFromBits packs an arbitrary number of MSB-first bits (8 minimum)
// into a PDU. Convenience for tests; live captures arrive as bytes.
func PDUFromBits(bits []byte) (PDU, error) {
	if len(bits) < 8 {
		return PDU{}, errors.New("tetra: PDU requires at least 8 bits")
	}
	bytesLen := (len(bits) + 7) / 8
	bs := make([]byte, bytesLen)
	for i, b := range bits {
		if b&1 != 0 {
			bs[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return ParsePDU(bs)
}

// PDUBits returns the MSB-first bits of a PDU. The length is
// 8 + 8*len(payload).
func PDUBits(p PDU) []byte {
	bytes := AssemblePDU(p)
	out := make([]byte, len(bytes)*8)
	for i := 0; i < len(out); i++ {
		if bytes[i>>3]&(1<<uint(7-(i&7))) != 0 {
			out[i] = 1
		}
	}
	return out
}

// IsCMCE reports whether the PDU is a CMCE (call-control) PDU.
func (p PDU) IsCMCE() bool { return p.Disc&0xC == DiscCMCE }

// IsMLE reports whether the PDU is an MLE (system-broadcast) PDU.
func (p PDU) IsMLE() bool { return p.Disc&0xC == DiscMLE }
