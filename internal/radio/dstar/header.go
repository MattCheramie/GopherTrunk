package dstar

import (
	"errors"
	"strings"
)

// CallsignLen is the fixed callsign length D-STAR uses across its
// header fields. Callsigns shorter than 8 characters are padded with
// trailing ASCII spaces (0x20).
const CallsignLen = 8

// Header is the Preamble + Header (PCH) packet that opens every D-STAR
// transmission. After the upstream FEC removes the convolutional /
// scrambler / interleave layers, the structured layout per the JARL
// DV-mode specification is:
//
//	bytes  0     Flag byte 1 (FLAG1) — Repeater / Interrupted /
//	             Control / EMR / Data flags.
//	bytes  1     Flag byte 2 (FLAG2) — supplementary flags.
//	bytes  2     Flag byte 3 (FLAG3) — supplementary flags.
//	bytes  3-10  RPT2 callsign — destination repeater (or "        ").
//	bytes 11-18  RPT1 callsign — gateway / source repeater.
//	bytes 19-26  UR   callsign — destination station ("CQCQCQ" =
//	             group call, "/<rpt>" = repeater routing).
//	bytes 27-34  MY1  callsign — source / own station callsign.
//	bytes 35-38  MY2  4-char short suffix.
//	bytes 39-40  CRC-16-CCITT over bytes 0..38.
//
// The structure is intentionally permissive: callers parse the
// callsign + flag fields and dispatch on the UR field.
type Header struct {
	Flag1, Flag2, Flag3 uint8
	RPT2                string // 8-char, space-padded
	RPT1                string
	UR                  string
	MY1                 string
	MY2                 string // 4-char suffix
	CRC                 uint16 // CRC-CCITT over bytes 0..38, MSB-first
}

// Flag bits within FLAG1 per the JARL spec.
const (
	Flag1Data        uint8 = 0x80 // Data Type indicator
	Flag1RepeaterMux uint8 = 0x40 // Repeater multiplexing
	Flag1Interrupted uint8 = 0x20
	Flag1Control     uint8 = 0x10
	Flag1Urgent      uint8 = 0x08
	Flag1EMR         uint8 = 0x04 // Emergency
	Flag1BreakIn     uint8 = 0x02
)

// IsEmergency reports whether the FLAG1 EMR bit is set.
func (h Header) IsEmergency() bool { return h.Flag1&Flag1EMR != 0 }

// IsBreakIn reports whether the FLAG1 Break-In bit is set.
func (h Header) IsBreakIn() bool { return h.Flag1&Flag1BreakIn != 0 }

// IsData reports whether the FLAG1 Data bit is set (as opposed to a
// voice + slow-data transmission).
func (h Header) IsData() bool { return h.Flag1&Flag1Data != 0 }

// IsGroupCall reports whether the UR callsign is the broadcast tag
// "CQCQCQ" or any /-prefixed repeater routing — both of which a
// trunking-style follower treats as a group transmission.
func (h Header) IsGroupCall() bool {
	ur := strings.TrimSpace(h.UR)
	if ur == "CQCQCQ" {
		return true
	}
	if len(ur) > 0 && ur[0] == '/' {
		return true
	}
	return false
}

// AssembleHeader packs a Header into 41 bytes (FLAG1..3, four 8-byte
// callsigns, the 4-char MY2, and the 16-bit CRC). Used by tests and
// by any future encoder work.
func AssembleHeader(h Header) []byte {
	out := make([]byte, 41)
	out[0] = h.Flag1
	out[1] = h.Flag2
	out[2] = h.Flag3
	copyCallsign(out[3:11], h.RPT2)
	copyCallsign(out[11:19], h.RPT1)
	copyCallsign(out[19:27], h.UR)
	copyCallsign(out[27:35], h.MY1)
	copyShort(out[35:39], h.MY2)
	out[39] = byte(h.CRC >> 8)
	out[40] = byte(h.CRC & 0xFF)
	return out
}

// ParseHeader consumes 41 bytes (as packed by AssembleHeader) and
// returns the structured Header. Trailing ASCII-space padding is
// preserved on the callsign fields so encoder round-trips are exact;
// IsGroupCall and similar accessors trim before comparing.
func ParseHeader(info []byte) (Header, error) {
	if len(info) != 41 {
		return Header{}, errors.New("dstar: header info must be 41 bytes")
	}
	return Header{
		Flag1: info[0],
		Flag2: info[1],
		Flag3: info[2],
		RPT2:  string(info[3:11]),
		RPT1:  string(info[11:19]),
		UR:    string(info[19:27]),
		MY1:   string(info[27:35]),
		MY2:   string(info[35:39]),
		CRC:   uint16(info[39])<<8 | uint16(info[40]),
	}, nil
}

func copyCallsign(dst []byte, s string) {
	for i := 0; i < CallsignLen; i++ {
		if i < len(s) {
			dst[i] = s[i]
		} else {
			dst[i] = ' '
		}
	}
}

func copyShort(dst []byte, s string) {
	for i := 0; i < 4; i++ {
		if i < len(s) {
			dst[i] = s[i]
		} else {
			dst[i] = ' '
		}
	}
}

// ComputeCRC computes a CRC-16-CCITT (poly 0x1021, init 0xFFFF) over
// the supplied bytes — the algorithm D-STAR uses for the header
// integrity field. The function is exported so tests can assert
// header round-trips include a valid CRC and fixtures can populate it.
func ComputeCRC(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
