package nxdn

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// CAC (Common Access Channel) carries the trunked-mode signaling on the
// NXDN control channel. Per the NXDN technical specification §6.4, a CAC
// message consists of:
//
//   - 8 bits message type (RCCH opcode)
//   - 64 bits payload
//   - 16 bits CRC-CCITT (covering message type + payload)
//
// = 88 information bits. On the air the CAC field carries this 88-bit
// payload plus FEC and a sub-frame interleaver across the 144-dibit
// information field; that wire-format pipeline (convolutional /
// scrambler / interleaver) is deferred and called out below.
//
// This file exposes the message-level structure so callers that already
// have the 88 information bits (e.g. supplied by tests or a future
// channel-coder) can parse RCCH messages and validate their CRCs.
type CACMessage struct {
	Type    RCCHType
	Payload [8]byte // 64 bits
}

// RCCHType is the 8-bit opcode field of an RCCH (Radio Channel Common
// Channel) message — common values per NXDN technical spec §7.3.
type RCCHType uint8

const (
	RCCHReserved00 RCCHType = 0x00
	RCCHVCALL      RCCHType = 0x01 // VCALL — voice call
	RCCHVCALLACK   RCCHType = 0x02 // VCALL_ACK
	RCCHVCALLASGN  RCCHType = 0x04 // VCALL_ASSGN
	RCCHDCALL      RCCHType = 0x09 // DCALL — data call
	RCCHDCALLACK   RCCHType = 0x0A // DCALL_ACK
	RCCHDCALLASGN  RCCHType = 0x0D // DCALL_ASSGN
	RCCHSDCALL     RCCHType = 0x38 // SDCALL — short-data
	RCCHSITEINFO   RCCHType = 0x3C // SITE_INFO_REQ / RSP
	RCCHSRVINFO    RCCHType = 0x3D // SRV_INFO
	RCCHCCH        RCCHType = 0x3F // CCH header (a.k.a. CC announcement)
)

func (r RCCHType) String() string {
	switch r {
	case RCCHVCALL:
		return "VCALL"
	case RCCHVCALLACK:
		return "VCALL_ACK"
	case RCCHVCALLASGN:
		return "VCALL_ASSGN"
	case RCCHDCALL:
		return "DCALL"
	case RCCHDCALLACK:
		return "DCALL_ACK"
	case RCCHDCALLASGN:
		return "DCALL_ASSGN"
	case RCCHSDCALL:
		return "SDCALL"
	case RCCHSITEINFO:
		return "SITE_INFO"
	case RCCHSRVINFO:
		return "SRV_INFO"
	case RCCHCCH:
		return "CCH_ANNOUNCE"
	default:
		return fmt.Sprintf("RCCHType(%02X)", uint8(r))
	}
}

// CRCError indicates the CRC trailer of a CAC message did not match the
// locally-computed CRC. The partially-parsed message is still returned.
var CRCError = errors.New("nxdn: CAC CRC mismatch")

// ParseCAC consumes 11 bytes (88 information bits, MSB-first) and returns
// a parsed message. Layout:
//
//	byte 0     : message type (RCCH opcode)
//	bytes 1-8  : payload
//	bytes 9-10 : CRC-CCITT (covering bytes 0..8)
func ParseCAC(info []byte) (CACMessage, error) {
	if len(info) != 11 {
		return CACMessage{}, fmt.Errorf("nxdn: CAC info must be 11 bytes, got %d", len(info))
	}
	msg := CACMessage{Type: RCCHType(info[0])}
	copy(msg.Payload[:], info[1:9])
	stored := binary.BigEndian.Uint16(info[9:11])
	want := framing.CRCCCITT(info[:9])
	if stored != want {
		return msg, CRCError
	}
	return msg, nil
}

// AssembleCAC builds an 11-byte CAC info block from structured fields.
func AssembleCAC(m CACMessage) []byte {
	out := make([]byte, 11)
	out[0] = byte(m.Type)
	copy(out[1:9], m.Payload[:])
	binary.BigEndian.PutUint16(out[9:11], framing.CRCCCITT(out[:9]))
	return out
}

// VCallPayload represents the payload of a VCALL (RCCH 0x01) message.
// The exact bit layout depends on the NXDN service variant; the fields
// below match the common Type-C trunked variant.
type VCallPayload struct {
	ServiceOptions uint8
	GroupAddress   uint16
	SourceID       uint16
	Reserved       uint16
}

// ParseVCall extracts the VCALL fields from the 8-byte payload.
func ParseVCall(p [8]byte) VCallPayload {
	return VCallPayload{
		ServiceOptions: p[0],
		GroupAddress:   binary.BigEndian.Uint16(p[1:3]),
		SourceID:       binary.BigEndian.Uint16(p[3:5]),
		Reserved:       binary.BigEndian.Uint16(p[5:7]),
	}
}

// SiteInfoPayload represents the SITE_INFO message used by the trunked
// variant for site identification broadcasts.
type SiteInfoPayload struct {
	LocationID uint16
	SiteID     uint16
	SystemID   uint16
}

func ParseSiteInfo(p [8]byte) SiteInfoPayload {
	return SiteInfoPayload{
		LocationID: binary.BigEndian.Uint16(p[0:2]),
		SiteID:     binary.BigEndian.Uint16(p[2:4]),
		SystemID:   binary.BigEndian.Uint16(p[4:6]),
	}
}
