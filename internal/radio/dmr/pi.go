package dmr

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// PIHeader is the DMR Privacy Indicator header (ETSI TS 102 361-1 data
// type 0x0), the data burst an encrypted transmission sends after its
// Voice LC Header to announce HOW the voice that follows is scrambled:
// the algorithm, the key the receiving radio must hold, and the Message
// Indicator (the per-transmission initialisation vector).
//
// Octet layout of the 12-octet BPTC(196,96)-recovered info block, pinned
// against two independent decoders (SDRTrunk `EncryptionParameters` and
// DSD-FME `dmr_pi.c` / `dmr_dburst.c`, which agree byte for byte):
//
//	octet 0     : algorithm identifier (DMRA: 0x21 RC4 "Enhanced Privacy",
//	              0x22 DES, 0x24 AES-128, 0x25 AES-256; SDRTrunk reads it as
//	              the Full LC opcode field, DSD-FME keys on the low 3 bits)
//	octet 1     : feature-set / manufacturer ID (0x10 DMRA, 0x68 Hytera,
//	              0x0A Kirisun)
//	octet 2     : key identifier
//	octets 3-6  : 32-bit Message Indicator, MSB first (Hytera carries a
//	              40-bit MI in octets 3-7 instead)
//	octets 7-9  : 24-bit destination address
//	octets 10-11: CRC-CCITT (poly 0x1021, init 0) over octets 0-9, XORed
//	              with the PI-header data-type mask
//
// The mask is 0x6969 in the ETSI convention (§B.3.12), which defines the
// CRC with its output inverted; framing.CRCCCITTWithInit is the plain,
// non-inverted form, so the equivalent mask here is ^0x6969 = 0x9696 —
// the same relation csbkCRCMask (0x5A5A) bears to the ETSI CSBK mask
// 0xA5A5, and the value SDRTrunk feeds its own non-inverting checker.
type PIHeader struct {
	AlgID   uint8
	FID     uint8
	KeyID   uint8
	MI      [4]byte // Message Indicator, MSB first
	DstAddr uint32  // 24-bit destination
	// Raw keeps the 12 recovered octets so an operator log can carry the
	// whole header (the instrument for a vendor variant this parser does
	// not name).
	Raw [12]byte
}

// Feature-set IDs the PI header carries in octet 1.
const (
	PIFIDDMRA    uint8 = 0x10
	PIFIDHytera  uint8 = 0x68
	PIFIDKirisun uint8 = 0x0A
)

// DMRA algorithm identifiers carried in octet 0 of a DMRA PI header.
const (
	PIAlgRC4    uint8 = 0x21 // "Enhanced Privacy" (ARC4)
	PIAlgDES    uint8 = 0x22
	PIAlgAES128 uint8 = 0x24
	PIAlgAES256 uint8 = 0x25
)

// piHeaderCRCMask is the PI-header data-type CRC mask in this package's
// (non-inverted CRC-CCITT) convention — see the PIHeader doc.
const piHeaderCRCMask uint16 = 0x9696

// PIHeaderBytes is the length of the recovered info block a PI header
// occupies (9 data octets + 1 reserved-by-layout + 2 CRC octets = 12).
const PIHeaderBytes = 12

// ErrPIHeaderLength / ErrPIHeaderCRC are the two ParsePIHeader failures.
var (
	ErrPIHeaderLength = errors.New("dmr: PI header requires 12 octets")
	ErrPIHeaderCRC    = errors.New("dmr: PI header CRC mismatch")
)

// ParsePIHeader decodes a BPTC(196,96)-recovered 12-octet info block as a
// Privacy Indicator header, verifying its CRC. The CRC is the only integrity
// check the header carries (there is no RS(12,9) trailer as on a Voice LC
// Header), so a BPTC-clean block that fails it is dropped, never guessed.
func ParsePIHeader(info []byte) (PIHeader, error) {
	if len(info) < PIHeaderBytes {
		return PIHeader{}, fmt.Errorf("%w, got %d", ErrPIHeaderLength, len(info))
	}
	want := uint16(info[10])<<8 | uint16(info[11])
	if got := framing.CRCCCITTWithInit(info[:10], 0x0000) ^ piHeaderCRCMask; got != want {
		return PIHeader{}, fmt.Errorf("%w: computed 0x%04X, carried 0x%04X", ErrPIHeaderCRC, got, want)
	}
	h := PIHeader{
		AlgID:   info[0],
		FID:     info[1],
		KeyID:   info[2],
		DstAddr: uint32(info[7])<<16 | uint32(info[8])<<8 | uint32(info[9]),
	}
	copy(h.MI[:], info[3:7])
	copy(h.Raw[:], info[:12])
	return h, nil
}

// AssemblePIHeader packs a PI header into its 12-octet info block with a
// valid CRC (the inverse of ParsePIHeader). Raw is ignored.
func AssemblePIHeader(h PIHeader) []byte {
	out := make([]byte, PIHeaderBytes)
	out[0] = h.AlgID
	out[1] = h.FID
	out[2] = h.KeyID
	copy(out[3:7], h.MI[:])
	out[7] = byte(h.DstAddr >> 16)
	out[8] = byte(h.DstAddr >> 8)
	out[9] = byte(h.DstAddr)
	crc := framing.CRCCCITTWithInit(out[:10], 0x0000) ^ piHeaderCRCMask
	out[10] = byte(crc >> 8)
	out[11] = byte(crc)
	return out
}

// MI32 returns the 32-bit Message Indicator as an integer (MSB first).
func (h PIHeader) MI32() uint32 {
	return uint32(h.MI[0])<<24 | uint32(h.MI[1])<<16 | uint32(h.MI[2])<<8 | uint32(h.MI[3])
}

// IsRC4 reports whether the header announces the DMRA RC4 ("Enhanced
// Privacy") algorithm. It keys on the low three bits of the algorithm
// octet like DSD-FME does, so a "DMRA-compatible" radio that sends 0x01
// instead of 0x21 still reads as RC4; the vendor must be DMRA (0x10) —
// Hytera's 0x02 Enhanced Privacy is a different construction (40-bit MI,
// vendor key schedule) and is deliberately not claimed here.
func (h PIHeader) IsRC4() bool {
	return h.FID == PIFIDDMRA && h.AlgID&0x07 == 0x01
}

// AlgName renders the algorithm octet for logs.
func (h PIHeader) AlgName() string {
	switch {
	case h.FID == PIFIDHytera:
		return fmt.Sprintf("hytera-0x%02X", h.AlgID)
	case h.FID == PIFIDKirisun:
		return fmt.Sprintf("kirisun-0x%02X", h.AlgID)
	case h.IsRC4():
		return "rc4"
	case h.FID == PIFIDDMRA && h.AlgID&0x07 == 0x02:
		return "des"
	case h.FID == PIFIDDMRA && h.AlgID&0x07 == 0x04:
		return "aes-128"
	case h.FID == PIFIDDMRA && h.AlgID&0x07 == 0x05:
		return "aes-256"
	}
	return fmt.Sprintf("fid-0x%02X-alg-0x%02X", h.FID, h.AlgID)
}
