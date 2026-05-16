// Package tier3 decodes DMR Tier III (trunked-mode) Control Signaling
// Blocks. CSBKs are carried as 96-bit information blocks encoded with
// BPTC(196,96), and contain the opcodes used for trunking signaling
// (Aloha, channel grants, system info, neighbor lists).
//
// Layout per ETSI TS 102 361-4 §7 of the 96-bit info block:
//
//	bit 0     : LB  (Last Block in CSBK chain)
//	bit 1     : PF  (Protected Flag)
//	bits 2-7  : CSBKO (Control Signaling Block Opcode, 6 bits)
//	bits 8-15 : FID (Feature set ID; 0x00 = standard ETSI)
//	bits 16-79: opcode-specific payload (64 bits, 8 octets)
//	bits 80-95: CRC-16 (CCITT) of bits 0-79; transmitted bitwise-NOT.
package tier3

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// CSBKOpcode is the 6-bit CSBKO field.
type CSBKOpcode uint8

const (
	// Standard FID (0x00) opcodes — ETSI TS 102 361-4 §7. Only the most
	// common opcodes are listed; vendor extensions live behind FID != 0.
	OpUnknown   CSBKOpcode = 0x00
	OpAloha     CSBKOpcode = 0x04 // ALOHA
	OpRAND      CSBKOpcode = 0x06 // Random access service request
	OpAckResp   CSBKOpcode = 0x16 // ACK response
	OpAhoy      CSBKOpcode = 0x18 // AHOY
	OpProtect   CSBKOpcode = 0x1A // Protect message
	OpMoveTSCC  CSBKOpcode = 0x1B // Move trunked-system control channel
	OpPreamble  CSBKOpcode = 0x28 // Preamble CSBK
	OpTVGrant   CSBKOpcode = 0x30 // TalkGroup voice channel grant
	OpPVGrant   CSBKOpcode = 0x31 // Private voice channel grant
	OpTDGrant   CSBKOpcode = 0x32 // TalkGroup data channel grant
	OpPDGrant   CSBKOpcode = 0x33 // Private data channel grant
	OpAdjStatus CSBKOpcode = 0x38 // Adjacent site status
	OpSysInfo   CSBKOpcode = 0x39 // System information broadcast
)

func (o CSBKOpcode) String() string {
	switch o {
	case OpAloha:
		return "Aloha"
	case OpRAND:
		return "RAND"
	case OpAckResp:
		return "AckResponse"
	case OpAhoy:
		return "Ahoy"
	case OpMoveTSCC:
		return "MoveTSCC"
	case OpPreamble:
		return "Preamble"
	case OpTVGrant:
		return "TalkGroupVoiceChannelGrant"
	case OpPVGrant:
		return "PrivateVoiceChannelGrant"
	case OpTDGrant:
		return "TalkGroupDataChannelGrant"
	case OpPDGrant:
		return "PrivateDataChannelGrant"
	case OpAdjStatus:
		return "AdjacentSiteStatus"
	case OpSysInfo:
		return "SystemInfoBroadcast"
	default:
		return fmt.Sprintf("CSBKOpcode(%02X)", uint8(o))
	}
}

// CSBK is one parsed Control Signaling Block.
type CSBK struct {
	LB      bool
	PF      bool
	Opcode  CSBKOpcode
	FID     uint8
	Payload [8]byte
}

// CRCError indicates the transmitted CRC didn't match the locally-computed
// CRC over the info bits. The partially-parsed CSBK is still returned so
// callers can log diagnostics.
var CRCError = errors.New("dmr/tier3: CSBK CRC mismatch")

// ParseCSBK consumes 96 information bits (12 bytes, MSB-first) and returns
// a parsed CSBK. The 16-bit trailer is the bitwise complement of the
// CRC-CCITT of the leading 10 bytes; CRCError is returned on mismatch.
func ParseCSBK(info []byte) (CSBK, error) {
	if len(info) != 12 {
		return CSBK{}, fmt.Errorf("dmr/tier3: CSBK info must be 12 bytes, got %d", len(info))
	}
	var c CSBK
	c.LB = info[0]&0x80 != 0
	c.PF = info[0]&0x40 != 0
	c.Opcode = CSBKOpcode(info[0] & 0x3F)
	c.FID = info[1]
	copy(c.Payload[:], info[2:10])

	storedCRC := binary.BigEndian.Uint16(info[10:12]) ^ 0xFFFF
	want := framing.CRCCCITT(info[:10])
	if storedCRC != want {
		return c, CRCError
	}
	return c, nil
}

// AssembleCSBK builds a 12-byte CSBK info block from structured fields. The
// 16-bit CRC trailer is stored as the bitwise complement of CRC-CCITT.
func AssembleCSBK(c CSBK) []byte {
	out := make([]byte, 12)
	if c.LB {
		out[0] |= 0x80
	}
	if c.PF {
		out[0] |= 0x40
	}
	out[0] |= byte(c.Opcode) & 0x3F
	out[1] = c.FID
	copy(out[2:10], c.Payload[:])
	crc := framing.CRCCCITT(out[:10]) ^ 0xFFFF
	binary.BigEndian.PutUint16(out[10:12], crc)
	return out
}

// InfoBitsToBytes packs a 96-bit slice (each entry 0/1, MSB-first) into 12
// bytes. Useful when handing off the BPTC decode result to ParseCSBK.
func InfoBitsToBytes(bits []byte) []byte {
	if len(bits) != 96 {
		panic("dmr/tier3: InfoBitsToBytes requires 96 bits")
	}
	out := make([]byte, 12)
	for i := 0; i < 96; i++ {
		if bits[i]&1 != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}
