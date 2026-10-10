package wmbus

import (
	"errors"
	"fmt"
)

// Telegram is a parsed link-layer frame: the data link header every frame
// carries, plus whatever of the transport and application layers is
// readable without a key.
type Telegram struct {
	Mode   Mode
	Format Format
	CRCOK  bool

	// Data link layer.
	C                byte   // C-field: 0x44 SND_NR (a meter's periodic broadcast), …
	ManufacturerCode uint16 // M-field
	Manufacturer     string // three-letter FLAG code, e.g. "TCH", "KAM"
	ID               string // identification number, 8 BCD digits
	Version          byte
	DeviceType       byte

	// CI is the first control-information field after the header (an ELL
	// one when present); AppCI is the one that introduces the transport
	// and application layers.
	CI    byte
	AppCI byte

	// Extended link layer (CI 0x8C / 0x8D), when present.
	HasELL    bool
	ELLAccess byte
	ELLSec    int // ELL encryption field (0 = none); >0 = AES-CTR payload

	// HasAFL reports an authentication and fragmentation layer (CI 0x90).
	HasAFL bool

	// Transport layer header (short 0x7A / long 0x72), when present.
	HasTPL       bool
	Access       byte   // access number, increments per telegram
	Status       byte   // meter status byte
	ConfigWord   uint16 // TPL configuration field
	SecurityMode int    // 0 = none, 5 = AES-CBC with IV, 7 = AES-CBC, …
	// Long TPL headers name the meter behind a radio converter.
	TPLManufacturer string
	TPLID           string
	TPLVersion      byte
	TPLDeviceType   byte

	// Encrypted reports that the application data could not be read
	// because it is encrypted (TPL security mode or ELL encryption).
	Encrypted bool
	// Payload is the application data, starting after the last header
	// parsed (still encrypted when Encrypted is set).
	Payload []byte
	// Records are the decoded data records of an unencrypted EN 13757-3
	// application layer. Empty for encrypted, compact or
	// manufacturer-specific payloads.
	Records []Record
	// RecordsErr explains why record parsing stopped early, if it did.
	RecordsErr string
}

// DeviceTypeName is the EN 13757-3 name of the telegram's device type.
func (t Telegram) DeviceTypeName() string { return DeviceTypeName(t.DeviceType) }

// ErrTooShort is returned for a frame too short to hold the data link
// header.
var ErrTooShort = errors.New("wmbus: telegram shorter than the data link header")

// ParseTelegram parses a frame's CRC-stripped data.
func ParseTelegram(f Frame) (Telegram, error) {
	t := Telegram{Mode: f.Mode, Format: f.Format, CRCOK: f.CRCOK}
	d := f.Data
	if len(d) < 11 {
		return t, ErrTooShort
	}
	t.C = d[1]
	t.ManufacturerCode = uint16(d[2]) | uint16(d[3])<<8
	t.Manufacturer = ManufacturerFlag(t.ManufacturerCode)
	t.ID = bcdID(d[4:8])
	t.Version = d[8]
	t.DeviceType = d[9]
	t.CI = d[10]
	p := d[11:]
	ci := t.CI

	// Extended link layer: one of these may sit between the data link
	// header and the transport layer.
	switch ci {
	case 0x8C, 0x8D:
		need := 2
		if ci == 0x8D {
			need = 8
		}
		if len(p) < need+1 {
			return t, fmt.Errorf("wmbus: CI 0x%02X: extended link layer truncated", ci)
		}
		t.HasELL = true
		t.ELLAccess = p[1]
		if ci == 0x8D {
			sn := uint32(p[2]) | uint32(p[3])<<8 | uint32(p[4])<<16 | uint32(p[5])<<24
			t.ELLSec = int(sn >> 29 & 0x7)
		}
		ci, p = p[need], p[need+1:]
		if t.ELLSec != 0 {
			// The rest, next CI included, is AES-CTR ciphertext.
			t.AppCI, t.Encrypted, t.Payload = 0, true, p
			return t, nil
		}
	}
	// Authentication and fragmentation layer (CI 0x90): a length byte,
	// that many bytes (fragment control, message counter, MAC), then the
	// transport layer's CI. OMS security mode 7 meters send one.
	if ci == 0x90 {
		if len(p) < 1 || len(p) < 2+int(p[0]) {
			return t, fmt.Errorf("wmbus: CI 0x90: authentication layer truncated")
		}
		t.HasAFL = true
		ci, p = p[1+int(p[0])], p[2+int(p[0]):]
	}
	t.AppCI = ci

	switch ci {
	case 0x7A, 0x7B: // short transport layer header
		if len(p) < 4 {
			return t, fmt.Errorf("wmbus: CI 0x%02X: short header truncated", ci)
		}
		t.HasTPL = true
		t.Access, t.Status = p[0], p[1]
		t.ConfigWord = uint16(p[2]) | uint16(p[3])<<8
		p = p[4:]
	case 0x72, 0x73: // long transport layer header
		if len(p) < 12 {
			return t, fmt.Errorf("wmbus: CI 0x%02X: long header truncated", ci)
		}
		t.HasTPL = true
		t.TPLID = bcdID(p[0:4])
		t.TPLManufacturer = ManufacturerFlag(uint16(p[4]) | uint16(p[5])<<8)
		t.TPLVersion, t.TPLDeviceType = p[6], p[7]
		t.Access, t.Status = p[8], p[9]
		t.ConfigWord = uint16(p[10]) | uint16(p[11])<<8
		p = p[12:]
	}
	if t.HasTPL {
		t.SecurityMode = int(t.ConfigWord >> 8 & 0x1F)
	}
	t.Payload = p
	if t.SecurityMode != 0 {
		t.Encrypted = true
		return t, nil
	}
	switch ci {
	case 0x72, 0x7A, 0x78:
		// EN 13757-3 application layer with full data records. (0x73,
		// 0x79 and 0x7B are compact frames whose records need the format
		// signature of an earlier full frame; 0xA0–0xB7 are
		// manufacturer-specific.)
		recs, err := ParseRecords(p)
		t.Records = recs
		if err != nil {
			t.RecordsErr = err.Error()
		}
	}
	return t, nil
}

// SecurityModeName names a TPL security mode.
func SecurityModeName(m int) string {
	switch m {
	case 0:
		return "none"
	case 5:
		return "AES-128-CBC with IV (mode 5)"
	case 7:
		return "AES-128-CBC, ephemeral key (mode 7)"
	case 10:
		return "AES-128-CCM (mode 10)"
	}
	return fmt.Sprintf("mode %d", m)
}

// ManufacturerFlag decodes an M-field into its three-letter FLAG code:
// three 5-bit letters, 'A' = 1, most significant first.
func ManufacturerFlag(m uint16) string {
	b := [3]byte{
		byte(m>>10&0x1F) + 64,
		byte(m>>5&0x1F) + 64,
		byte(m&0x1F) + 64,
	}
	for _, c := range b {
		if c < 'A' || c > 'Z' {
			return fmt.Sprintf("0x%04X", m)
		}
	}
	return string(b[:])
}

// bcdID renders a 4-byte little-endian BCD identification number as its
// 8 digits, most significant first, as printed on the meter.
func bcdID(b []byte) string {
	const hex = "0123456789ABCDEF"
	out := make([]byte, 0, 8)
	for i := len(b) - 1; i >= 0; i-- {
		out = append(out, hex[b[i]>>4], hex[b[i]&0xF])
	}
	return string(out)
}

// DeviceTypeName is the EN 13757-3 name of a device type (A-field byte 6).
func DeviceTypeName(t byte) string {
	switch t {
	case 0x00:
		return "other"
	case 0x01:
		return "oil"
	case 0x02:
		return "electricity"
	case 0x03:
		return "gas"
	case 0x04:
		return "heat"
	case 0x05:
		return "steam"
	case 0x06:
		return "warm water"
	case 0x07:
		return "water"
	case 0x08:
		return "heat cost allocator"
	case 0x09:
		return "compressed air"
	case 0x0A:
		return "cooling (outlet)"
	case 0x0B:
		return "cooling (inlet)"
	case 0x0C:
		return "heat (inlet)"
	case 0x0D:
		return "heat/cooling"
	case 0x0E:
		return "bus/system component"
	case 0x15:
		return "hot water"
	case 0x16:
		return "cold water"
	case 0x17:
		return "hot/cold water"
	case 0x18:
		return "pressure"
	case 0x19:
		return "A/D converter"
	case 0x1A:
		return "smoke detector"
	case 0x1B:
		return "room sensor"
	case 0x1C:
		return "gas detector"
	case 0x20:
		return "breaker (electricity)"
	case 0x21:
		return "valve (gas or water)"
	case 0x25:
		return "customer unit (display)"
	case 0x28:
		return "waste water"
	case 0x29:
		return "garbage"
	case 0x31:
		return "communication controller"
	case 0x32:
		return "unidirectional repeater"
	case 0x33:
		return "bidirectional repeater"
	case 0x36:
		return "radio converter (system side)"
	case 0x37:
		return "radio converter (meter side)"
	}
	return fmt.Sprintf("type 0x%02X", t)
}
