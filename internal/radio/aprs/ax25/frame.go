// Package ax25 decodes the AX.25 link-layer frames APRS rides on top
// of. AX.25 is a HDLC-derived frame format used by amateur packet
// radio — modern terrestrial use is dominated by APRS (Automatic
// Packet Reporting System) on the North-American 144.39 MHz channel,
// the European 144.800 MHz channel, and other regional allocations.
//
// Frame structure (numbers are bytes):
//
//	+-----+-----+-----+-----+--------+--------+--------+
//	| dst | src |  path (0..8)       | control |  PID  |
//	|  7  |  7  | 0 .. 56            |    1    |   1   |
//	+-----+-----+-----+-----+--------+--------+--------+
//	|        info (0..256)            |   FCS (CRC-16) |
//	|             bytes               |       2 bytes  |
//	+---------------------------------+----------------+
//
// Each address is 6 bytes of base-40 callsign + 1 byte SSID/flags.
// The path lists digipeaters; the last byte of the LAST address has
// its bit 0 set ("end of address list") to terminate the chain.
// FCS is HDLC-style CRC-16-CCITT over the de-stuffed body, with the
// receive convention that a correct frame has residue 0xF0B8 after
// running the CRC over the body+FCS bytes.
//
// This package handles AX.25 frame parsing only; the on-air bit-
// stuffing / 0x7E flag delimiting / NRZI decoding happens one layer
// up (in the DSP receiver). The APRS info-field interpretation
// happens one layer further up (in package aprs).
package ax25

import (
	"errors"
	"fmt"
	"strings"
)

// Address is one entry in the {dst, src, path…} chain. AX.25 packs
// each as 6 bytes of left-justified callsign (each byte shifted
// left 1 — bit 0 of every callsign byte is reserved as the HDLC
// "extension" flag) plus 1 byte carrying the SSID, the H/R bits,
// and the end-of-address sentinel.
type Address struct {
	Callsign string // 0..6 ASCII chars, trailing spaces stripped
	SSID     uint8  // 0..15
	// HBit is the "has been digipeated" / repeater-acknowledged
	// flag (bit 7 of the SSID byte for destination + path entries;
	// the source address re-purposes it as the C-bit Command/
	// Response indicator). True when the path entry's frame has
	// passed through this digipeater.
	HBit bool
}

// String returns the conventional CALLSIGN-SSID format
// (e.g. "W1AW-9"), or just "CALLSIGN" when SSID is 0. Suffix "*"
// is appended when HBit is set — APRS console convention.
func (a Address) String() string {
	s := a.Callsign
	if a.SSID != 0 {
		s += fmt.Sprintf("-%d", a.SSID)
	}
	if a.HBit {
		s += "*"
	}
	return s
}

// Frame is one decoded AX.25 frame.
type Frame struct {
	Dst     Address
	Src     Address
	Path    []Address
	Control uint8
	PID     uint8
	Info    []byte
	// FCSOK reports whether the CRC trailing the frame matched.
	// Set by Parse; callers should drop frames where FCSOK is
	// false unless they're explicitly studying noise.
	FCSOK bool
}

// Errors returned by Parse.
var (
	ErrFrameTooShort = errors.New("ax25: frame body shorter than minimum (16 bytes)")
	ErrBadAddress    = errors.New("ax25: address chain malformed (no end-of-address flag in first 9 entries)")
	ErrUnsupported   = errors.New("ax25: unsupported control / PID combination")
)

// MinFrameBytes is dst (7) + src (7) + control (1) + PID (1) +
// FCS (2) = 18 bytes. A frame shorter than this can't possibly be
// well-formed.
const MinFrameBytes = 18

// MaxPathEntries is the AX.25 spec limit on digipeater addresses.
const MaxPathEntries = 8

// Parse decodes one bit-stuffed-and-flag-stripped AX.25 frame body.
// The input MUST be the byte stream that landed between two 0x7E
// HDLC flag bytes, after NRZI decoding and de-bit-stuffing. The
// trailing 2 bytes are the FCS; Parse validates them and reports
// FCSOK accordingly.
//
// Returns the decoded Frame plus a nil error on structural
// success; FCSOK on the returned frame distinguishes CRC-clean
// frames from CRC-failed ones. A structural error (too short,
// malformed address chain) is returned as the second value.
func Parse(body []byte) (Frame, error) {
	if len(body) < MinFrameBytes {
		return Frame{}, ErrFrameTooShort
	}
	// Address chain: dst + src + 0..8 path. Each entry is 7 bytes;
	// the end-of-address sentinel is bit 0 of the last byte of the
	// final entry.
	addresses := make([]Address, 0, 2+MaxPathEntries)
	off := 0
	endFound := false
	for i := 0; i < 2+MaxPathEntries; i++ {
		if off+7 > len(body) {
			return Frame{}, ErrBadAddress
		}
		raw := body[off : off+7]
		// HBit is only meaningful for path entries (i >= 2). For
		// the destination + source addresses bit 7 is the C-bit
		// (Command/Response indicator), which the APRS console
		// convention ignores. Only set HBit when this is a path
		// entry.
		addr := parseAddress(raw, i >= 2)
		addresses = append(addresses, addr)
		off += 7
		// End-of-address: bit 0 of byte 6 is set.
		if raw[6]&0x01 != 0 {
			endFound = true
			break
		}
	}
	if !endFound || len(addresses) < 2 {
		return Frame{}, ErrBadAddress
	}
	// Control + PID + Info + FCS.
	if len(body)-off < 4 { // control + pid + 2-byte FCS minimum
		return Frame{}, ErrFrameTooShort
	}
	control := body[off]
	off++
	// PID is only meaningful for I-frames + UI frames; for now we
	// always read it. Real-world APRS only ever uses UI frames
	// (control byte = 0x03, PID = 0xF0); the parser is generic
	// enough that other frame types pass through with their PID
	// preserved.
	pid := body[off]
	off++

	// FCS is the last 2 bytes; everything in between is the info
	// field.
	if len(body)-off < 2 {
		return Frame{}, ErrFrameTooShort
	}
	infoEnd := len(body) - 2
	info := append([]byte(nil), body[off:infoEnd]...)
	fcsBytes := body[infoEnd:]

	// CRC-16-CCITT, HDLC variant: poly 0x1021, init 0xFFFF, output
	// inverted, reflected input + output. The transmitted FCS is
	// the inverted CRC of everything in front of it; running the
	// CRC over body+FCS produces the magic residue 0xF0B8 when
	// clean.
	calc := computeFCS(body[:infoEnd])
	wireFCS := uint16(fcsBytes[0]) | uint16(fcsBytes[1])<<8
	frame := Frame{
		Dst:     addresses[0],
		Src:     addresses[1],
		Path:    addresses[2:],
		Control: control,
		PID:     pid,
		Info:    info,
		FCSOK:   calc == wireFCS,
	}
	return frame, nil
}

// parseAddress unpacks one 7-byte address entry. isPathEntry is
// true for entries 2..9 of the address chain (digipeater path);
// false for dst (entry 0) and src (entry 1) where bit 7 of the
// SSID byte is the C-bit rather than the H-bit and is ignored by
// the APRS console convention.
func parseAddress(raw []byte, isPathEntry bool) Address {
	var cs strings.Builder
	for i := 0; i < 6; i++ {
		c := raw[i] >> 1
		if c == ' ' || c == 0 {
			break
		}
		cs.WriteByte(c)
	}
	ssidByte := raw[6]
	// SSID is bits 1..4. Bit 7 is the H/C bit (H for path,
	// C for dst/src). Bit 0 is end-of-address (handled by caller).
	ssid := (ssidByte >> 1) & 0x0F
	hbit := isPathEntry && ssidByte&0x80 != 0
	return Address{
		Callsign: strings.TrimRight(cs.String(), " "),
		SSID:     ssid,
		HBit:     hbit,
	}
}

// computeFCS computes the HDLC CRC-16-CCITT over the supplied
// bytes. The FCS shipped on-wire is the bit-inverted, byte-swapped
// version of this; the parser compares the wire-form FCS directly,
// so this helper applies the same inversion + swap convention.
func computeFCS(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0x8408
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

// IsUI reports whether this is a UI (Unnumbered Information)
// frame — the only frame type APRS uses on the wire. Control byte
// 0x03, PID 0xF0 ("no layer 3 protocol").
func (f Frame) IsUI() bool {
	return f.Control == 0x03 && f.PID == 0xF0
}

// PathString renders the digipeater path in APRS console form
// (e.g. "WIDE1-1,WIDE2-1*"). Empty path returns the empty string.
func (f Frame) PathString() string {
	if len(f.Path) == 0 {
		return ""
	}
	parts := make([]string, len(f.Path))
	for i, a := range f.Path {
		parts[i] = a.String()
	}
	return strings.Join(parts, ",")
}
