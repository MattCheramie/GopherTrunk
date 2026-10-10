package wmbus

import "errors"

// CRC16 is the EN 13757 link-layer CRC (CRC-16/EN-13757 in the reveng
// catalogue): polynomial 0x3D65, initial value 0, not reflected, final
// value complemented. It is transmitted most-significant byte first.
func CRC16(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x3D65
			} else {
				crc <<= 1
			}
		}
	}
	return ^crc
}

// Mode is the radio mode a frame was received in.
type Mode uint8

const (
	// ModeT1 is meter-to-other T mode: 100 kchip/s, 3-of-6 coded.
	ModeT1 Mode = iota + 1
	// ModeC1 is meter-to-other C mode: 100 kchip/s NRZ.
	ModeC1
)

func (m Mode) String() string {
	switch m {
	case ModeT1:
		return "T1"
	case ModeC1:
		return "C1"
	}
	return "?"
}

// Format is the link-layer frame format.
type Format uint8

const (
	// FormatA: a 10-byte first block, then 16-byte blocks, each followed
	// by its own CRC. The L-field excludes the CRC bytes.
	FormatA Format = iota + 1
	// FormatB: one CRC over the first 126 bytes (blocks 1 and 2), then one
	// over the remainder. The L-field includes the CRC bytes. C mode only.
	FormatB
)

func (f Format) String() string {
	switch f {
	case FormatA:
		return "A"
	case FormatB:
		return "B"
	}
	return "?"
}

// minLField is the smallest L-field a telegram can carry: C, M (2) and
// A (6) follow L in every frame.
const minLField = 9

// frameLength is the number of bytes on the air for a frame with the given
// L-field and format, CRC bytes included, L itself included. It returns 0
// for an L-field too short to hold the data link header.
func frameLength(l byte, f Format) int {
	n := int(l)
	switch f {
	case FormatA:
		if n < minLField {
			return 0
		}
		// Block 1 holds L plus 9 header bytes; every further block holds
		// up to 16 data bytes. Each block carries a 2-byte CRC.
		blocks := 1 + (n-minLField+15)/16
		return 1 + n + 2*blocks
	case FormatB:
		if n < minLField+2 {
			return 0
		}
		return 1 + n
	}
	return 0
}

// errShort is returned for a frame shorter than its own L-field promises.
var errShort = errors.New("wmbus: frame shorter than its L-field")

// stripCRC checks every block CRC of a received frame and returns the data
// with the CRC bytes removed, L first. In the result the L-field always
// counts the data bytes after it (format A's convention), so callers parse
// both formats alike. ok is false when any block CRC fails; data is still
// returned so a marginal frame can be inspected.
func stripCRC(raw []byte, f Format) (data []byte, ok bool, err error) {
	if len(raw) == 0 || len(raw) < frameLength(raw[0], f) || frameLength(raw[0], f) == 0 {
		return nil, false, errShort
	}
	raw = raw[:frameLength(raw[0], f)]
	ok = true
	check := func(block []byte, crc []byte) {
		if CRC16(block) != uint16(crc[0])<<8|uint16(crc[1]) {
			ok = false
		}
	}
	switch f {
	case FormatA:
		data = append(data, raw[:10]...)
		check(raw[:10], raw[10:12])
		for p := 12; p < len(raw); {
			n := min(16, len(raw)-p-2)
			data = append(data, raw[p:p+n]...)
			check(raw[p:p+n], raw[p+n:p+n+2])
			p += n + 2
		}
	case FormatB:
		// Block 2's CRC covers blocks 1 and 2 together (up to 126 bytes);
		// an optional block 3 has its own.
		p := 0
		for p < len(raw) {
			n := min(126, len(raw)-p-2)
			data = append(data, raw[p:p+n]...)
			check(raw[p:p+n], raw[p+n:p+n+2])
			p += n + 2
		}
		data[0] = byte(len(data) - 1)
	}
	return data, ok, nil
}
