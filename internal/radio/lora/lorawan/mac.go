// Package lorawan parses the LoRaWAN MAC layer carried in a decoded LoRa
// PHY payload and, when the operator supplies the device session keys,
// verifies the message integrity code and decrypts the frame payload.
//
// GopherTrunk decrypts only with keys the operator already holds and is
// authorised to use — it performs no key recovery (the same posture as the
// trunking encryption_keys feature). Messages whose keys are unknown, or
// whose MIC does not verify, are still surfaced with the cleartext MAC
// header fields and the ciphertext payload.
//
// This implements the LoRaWAN 1.0.x layout (the common field deployment).
// MIC and payload-encryption use AES-128 (crypto/aes, pure Go) and an
// in-package AES-CMAC (RFC 4493); see mic.go and crypt.go.
package lorawan

import (
	"encoding/binary"
	"errors"
)

// MType is the LoRaWAN message type (the top 3 bits of MHDR).
type MType uint8

const (
	JoinRequest         MType = 0x00
	JoinAccept          MType = 0x01
	UnconfirmedDataUp   MType = 0x02
	UnconfirmedDataDown MType = 0x03
	ConfirmedDataUp     MType = 0x04
	ConfirmedDataDown   MType = 0x05
	RFU                 MType = 0x06
	Proprietary         MType = 0x07
)

// String returns a stable lowercase label for the message type.
func (t MType) String() string {
	switch t {
	case JoinRequest:
		return "join-request"
	case JoinAccept:
		return "join-accept"
	case UnconfirmedDataUp:
		return "unconfirmed-up"
	case UnconfirmedDataDown:
		return "unconfirmed-down"
	case ConfirmedDataUp:
		return "confirmed-up"
	case ConfirmedDataDown:
		return "confirmed-down"
	case Proprietary:
		return "proprietary"
	default:
		return "rfu"
	}
}

// IsUplink reports whether the message travels device → network (Dir=0 for
// the MIC / payload-encryption blocks).
func (t MType) IsUplink() bool {
	return t == JoinRequest || t == UnconfirmedDataUp || t == ConfirmedDataUp
}

// IsData reports whether the message carries an FHDR (a data frame, as
// opposed to a join exchange or proprietary message).
func (t MType) IsData() bool {
	switch t {
	case UnconfirmedDataUp, UnconfirmedDataDown, ConfirmedDataUp, ConfirmedDataDown:
		return true
	}
	return false
}

// Message is a parsed LoRaWAN frame. For data frames the FHDR fields are
// populated; FPort is -1 when absent and FRMPayload is the raw (still
// encrypted) frame payload. For join messages only MType, the raw EUIs /
// nonce (JoinEUI/DevEUI/DevNonce) and MIC are populated.
type Message struct {
	MType      MType
	DevAddr    uint32 // data frames
	FCtrl      uint8
	FCnt       uint16 // 16-bit counter from the FHDR
	FOpts      []byte
	FPort      int    // -1 when no FRMPayload
	FRMPayload []byte // ciphertext as received
	MIC        [4]byte

	// Join-request fields (raw, little-endian on the wire).
	JoinEUI  uint64
	DevEUI   uint64
	DevNonce uint16

	mac []byte // MHDR|FHDR|FPort|FRMPayload — the MIC-protected region
}

var (
	// ErrTooShort is returned when the PHY payload is smaller than the
	// minimum LoRaWAN frame.
	ErrTooShort = errors.New("lorawan: payload too short")
	// ErrBadFOptsLen is returned when the declared FOpts length exceeds the
	// available bytes.
	ErrBadFOptsLen = errors.New("lorawan: FOptsLen exceeds payload")
)

// Parse decodes the MAC layer from a LoRa PHY payload (MHDR | … | MIC).
func Parse(phy []byte) (*Message, error) {
	if len(phy) < 5 { // MHDR + at least 4-byte MIC
		return nil, ErrTooShort
	}
	m := &Message{FPort: -1}
	m.MType = MType(phy[0] >> 5)
	copy(m.MIC[:], phy[len(phy)-4:])
	m.mac = phy[:len(phy)-4]
	body := phy[1 : len(phy)-4] // between MHDR and MIC

	switch {
	case m.MType == JoinRequest:
		if len(body) != 18 {
			return nil, ErrTooShort
		}
		m.JoinEUI = binary.LittleEndian.Uint64(body[0:8])
		m.DevEUI = binary.LittleEndian.Uint64(body[8:16])
		m.DevNonce = binary.LittleEndian.Uint16(body[16:18])
		return m, nil
	case m.MType.IsData():
		if len(body) < 7 { // DevAddr(4)+FCtrl(1)+FCnt(2)
			return nil, ErrTooShort
		}
		m.DevAddr = binary.LittleEndian.Uint32(body[0:4])
		m.FCtrl = body[4]
		m.FCnt = binary.LittleEndian.Uint16(body[5:7])
		fOptsLen := int(m.FCtrl & 0x0f)
		off := 7
		if off+fOptsLen > len(body) {
			return nil, ErrBadFOptsLen
		}
		m.FOpts = append([]byte(nil), body[off:off+fOptsLen]...)
		off += fOptsLen
		if off < len(body) {
			m.FPort = int(body[off])
			off++
			m.FRMPayload = append([]byte(nil), body[off:]...)
		}
		return m, nil
	default:
		// Join-accept / proprietary: header parsed, body left raw.
		return m, nil
	}
}

// FCnt32 returns the frame counter widened to 32 bits. LoRaWAN frames only
// carry the low 16 bits on the wire; without session state we assume the
// upper half is zero (correct until the first rollover).
func (m *Message) FCnt32() uint32 { return uint32(m.FCnt) }
