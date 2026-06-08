// Package m17 decodes the link layer of the M17 digital voice/data
// protocol — an open, Codec2-based amateur mode (4FSK, 4800 sym/s).
// This package owns the *metadata* path: it recovers the Link Setup
// Frame (LSF) — source and destination callsigns, the TYPE field, and
// the META block — and publishes it on the bus. The Codec2 voice
// payload is a later milestone; this slice surfaces "who is
// transmitting to whom, in what mode" without decoding audio.
//
// Two routes carry the LSF:
//   - the dedicated LSF frame at the head of a transmission
//     (convolutionally coded + punctured — decoded elsewhere), and
//   - the LICH (Link Information CHannel) embedded in every stream
//     frame, which carries 1/6 of the LSF Golay(24,12)-coded. Six
//     consecutive LICH chunks reassemble the full LSF.
//
// This package implements the LICH route (Golay only, no convolutional
// machinery), so a receiver tuned to an in-progress transmission picks
// up the link metadata within ~240 ms. See lich.go.
//
// Constants (sync words, LSF layout, CRC-16 0x5935, base-40 alphabet,
// dibit↔symbol map, LICH structure) are from the M17 specification.
// The Golay(24,12) generator and LSF field parse are validated
// end-to-end against a synthetic encoder; the exact Golay matrix of
// real M17 traffic should be confirmed against a capture.
//
// Reference: M17 Protocol Specification (spec.m17project.org),
// data-link layer.
package m17

import "strings"

// LSFBits / LSFBytes is the Link Setup Frame size after FEC removal:
// DST(48) + SRC(48) + TYPE(16) + META(112) + CRC(16) = 240 bits.
const (
	LSFBits  = 240
	LSFBytes = LSFBits / 8 // 30
)

// base40 is the M17 address alphabet (index 0..39). Index 0 is the
// blank/terminator; the rest are the callsign characters.
const base40 = " ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-/."

// broadcastAddr is the all-ones 48-bit address (the M17 broadcast
// destination).
const broadcastAddr uint64 = 0xFFFFFFFFFFFF

// LSF is a decoded Link Setup Frame.
type LSF struct {
	Dst      string // destination callsign / "BROADCAST" / "#RESERVED"
	Src      string // source callsign
	Type     uint16 // raw TYPE field
	Meta     []byte // 14-byte META block (nonce / GNSS / extended callsign)
	CRCOK    bool   // CRC-16 over DST..META matched the trailing field
	Stream   bool   // TYPE bit 0: stream (vs packet) mode
	DataMode uint8  // TYPE bits 1..2: 1 = data, 2 = voice, 3 = voice+data
	CAN      uint8  // TYPE bits 7..10: channel access number
}

// DecodeAddress turns a 48-bit M17 address into a callsign string.
// Returns "BROADCAST" for the all-ones address, "#RESERVED" for values
// above the encodable range, and "" for the reserved zero address.
func DecodeAddress(addr uint64) string {
	addr &= 0xFFFFFFFFFFFF
	switch {
	case addr == 0:
		return ""
	case addr == broadcastAddr:
		return "BROADCAST"
	case addr > 0xEE6B28000000: // > 40^9: not a base-40 callsign
		return "#RESERVED"
	}
	var sb strings.Builder
	for addr > 0 {
		sb.WriteByte(base40[addr%40])
		addr /= 40
	}
	return reverse(sb.String())
}

func reverse(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// ParseLSF parses a 30-byte (240-bit) Link Setup Frame and verifies its
// CRC-16. Never errors — a bad CRC sets CRCOK=false and the fields are
// still populated (best-effort, for display of marginal decodes).
func ParseLSF(b []byte) LSF {
	var l LSF
	if len(b) < LSFBytes {
		return l
	}
	l.Dst = DecodeAddress(beUint48(b[0:6]))
	l.Src = DecodeAddress(beUint48(b[6:12]))
	l.Type = uint16(b[12])<<8 | uint16(b[13])
	l.Meta = append([]byte(nil), b[14:28]...)
	want := uint16(b[28])<<8 | uint16(b[29])
	l.CRCOK = CRC16(b[0:28]) == want

	l.Stream = l.Type&0x0001 != 0
	l.DataMode = uint8((l.Type >> 1) & 0x03)
	l.CAN = uint8((l.Type >> 7) & 0x0F)
	return l
}

// String renders the LSF for log / panel display.
func (l LSF) String() string {
	mode := "packet"
	if l.Stream {
		mode = "stream"
	}
	dst := l.Dst
	if dst == "" {
		dst = "(none)"
	}
	return l.Src + " → " + dst + " [" + mode + " CAN=" + itoa(int(l.CAN)) + "]"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// ModeString labels the TYPE data-mode field (bits 1..2): 1 = data,
// 2 = voice, 3 = voice+data, 0 = reserved. Packet-mode frames (TYPE
// bit 0 clear) report "packet".
func (l LSF) ModeString() string {
	if !l.Stream {
		return "packet"
	}
	switch l.DataMode {
	case 1:
		return "data"
	case 2:
		return "voice"
	case 3:
		return "voice+data"
	}
	return "reserved"
}

// MetaHex renders the 14-byte META block as a hex string.
func (l LSF) MetaHex() string {
	const hexdigits = "0123456789abcdef"
	b := make([]byte, 0, len(l.Meta)*2)
	for _, x := range l.Meta {
		b = append(b, hexdigits[x>>4], hexdigits[x&0x0F])
	}
	return string(b)
}

// beUint48 reads a 48-bit big-endian unsigned integer.
func beUint48(b []byte) uint64 {
	var v uint64
	for _, x := range b[:6] {
		v = v<<8 | uint64(x)
	}
	return v
}

// CRC16 computes the M17 CRC-16 (polynomial 0x5935, initial value
// 0xFFFF, MSB-first, no final XOR) over msg.
func CRC16(msg []byte) uint16 {
	const poly = 0x5935
	crc := uint16(0xFFFF)
	for _, b := range msg {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
