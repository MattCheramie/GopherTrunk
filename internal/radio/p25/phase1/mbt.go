package phase1

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Multi-Block Trunking (MBT) — trunking signalling carried in a PDU
// (DUID 0xC) on the control channel instead of a TSDU. Many systems
// (notably Motorola) broadcast their richest network data — Network
// Status with WACN, Adjacent Status with explicit downlink AND uplink
// channels, RFSS Status — only in the Alternate MBT (AMBT) format, so a
// decoder that drops DUID 0xC as "non-control" never surfaces most
// neighbour sites or (on some systems) the WACN at all. That was the
// operator-reported gap: SDRTrunk lists 12 neighbours with uplinks in
// 15 s while GT showed one and "No Network Status Broadcast yet",
// while the very frames carrying that data were logged as
// `non-control DUID duid=PDU` spam.
//
// Channel coding is identical to the TSDU: each block is 196 bits,
// 1/2-rate trellis coded and interleaved exactly like a TSBK, so the
// receive chain reuses the deinterleave → Viterbi pipeline. What
// differs is the block content: a 12-octet PDU header protected by the
// same augmented CRC-CCITT16 the TSBK trailer uses, followed by
// BlocksToFollow 12-octet data blocks whose concatenation ends in a
// 4-octet CRC-32.
//
// Layouts are cross-checked against two independent decoders — OP25's
// p25p1_fdma::process_PDU (header/format/SAP/opcode extraction, both
// CRC algorithms) and SDRTrunk's PDUHeader / AMBTCHeader / AMBTC*
// message classes (per-field bit offsets) — so this is not a working
// model.

// MBT PDU header formats (5-bit PDU_FORMAT field, header octet 0 low 5
// bits) that carry trunking control, per OP25 process_PDU.
const (
	MBTFormatUnconfirmed = 0x15 // Unconfirmed MBT: opcode in data block 0
	MBTFormatAlternate   = 0x17 // Alternate MBT (AMBT): opcode in header octet 7
)

// MBTSAPTrunkingControl is the PDU Service Access Point value for
// trunking control signalling (61); any other SAP is user data, not MBT.
const MBTSAPTrunkingControl = 0x3D

// maxMBTDataBlocks bounds how many data blocks an MBT message may carry.
// AMBT trunking broadcasts use 1-2 data blocks; the cap keeps a corrupt
// BlocksToFollow from stalling the frame parser on a block train that
// never arrives.
const maxMBTDataBlocks = 3

// ErrMBTHeaderCRC is returned when the PDU header block fails its
// CRC-CCITT16 check.
var ErrMBTHeaderCRC = errors.New("p25/phase1: MBT PDU header CRC check failed")

// ErrMBTDataCRC is returned when the concatenated data blocks fail
// their trailing CRC-32 check.
var ErrMBTDataCRC = errors.New("p25/phase1: MBT data block CRC-32 check failed")

// MBTHeader is the decoded 12-octet PDU header of a trunking MBT.
// Field offsets per SDRTrunk PDUHeader/AMBTCHeader and OP25:
//
//	octet 0     : bits 4..0 = format (0x17 AMBT / 0x15 unconfirmed)
//	octet 1     : bits 5..0 = SAP (61 = trunking control)
//	octet 2     : MFID
//	octets 3-5  : AMBT: message-specific (LRA / System ID for the
//	              status broadcasts); plain PDU: logical link ID
//	octet 6     : bits 6..0 = blocks to follow
//	octet 7     : bits 5..0 = opcode (AMBT only)
//	octets 8-9  : AMBT: two message-specific data octets
//	octets 10-11: header CRC (CCITT-16, validated by ParseMBTHeader)
type MBTHeader struct {
	Format         uint8
	SAP            uint8
	MFID           uint8
	BlocksToFollow uint8
	Opcode         Opcode // AMBT opcode (octet 7); meaningless for 0x15
	Raw            [12]byte
}

// IsTrunkingControl reports whether this header carries trunking
// signalling this decoder understands (SAP 61, AMBT or unconfirmed MBT).
func (h MBTHeader) IsTrunkingControl() bool {
	return h.SAP == MBTSAPTrunkingControl &&
		(h.Format == MBTFormatAlternate || h.Format == MBTFormatUnconfirmed)
}

// ParseMBTHeader decodes and CRC-checks a 12-octet PDU header block.
// The partially-parsed header is returned even on CRC failure so
// callers can log it.
func ParseMBTHeader(info []byte) (MBTHeader, error) {
	if len(info) != 12 {
		return MBTHeader{}, fmt.Errorf("%w, got %d", ErrTSBKInfoLength, len(info))
	}
	var h MBTHeader
	copy(h.Raw[:], info)
	h.Format = info[0] & 0x1F
	h.SAP = info[1] & 0x3F
	h.MFID = info[2]
	h.BlocksToFollow = info[6] & 0x7F
	h.Opcode = Opcode(info[7] & 0x3F)
	if framing.CRCCCITTAugmented(info) != 0 {
		return h, ErrMBTHeaderCRC
	}
	return h, nil
}

// AssembleMBTHeader is the inverse of ParseMBTHeader; for tests. raw37
// and raw89 fill the message-specific header octets 3-5 and 8-9.
func AssembleMBTHeader(h MBTHeader, raw35 [3]byte, raw89 [2]byte) []byte {
	out := make([]byte, 12)
	out[0] = h.Format & 0x1F
	out[1] = h.SAP & 0x3F
	out[2] = h.MFID
	copy(out[3:6], raw35[:])
	out[6] = h.BlocksToFollow & 0x7F
	out[7] = byte(h.Opcode) & 0x3F
	copy(out[8:10], raw89[:])
	binary.BigEndian.PutUint16(out[10:12], framing.CRCCCITTAugmented(out))
	return out
}

// DecodeMBTBlockChannel runs the shared TSBK channel pipeline —
// deinterleave → 1/2-rate trellis Viterbi → repack — over 98 channel
// dibits and returns the 12 info octets plus the Viterbi path metric
// (0 = clean). PDU blocks use exactly the TSBK block coding; only the
// CRC conventions differ, and those are checked by the callers
// (ParseMBTHeader / ValidateMBTData).
func DecodeMBTBlockChannel(channel []uint8) ([]byte, int) {
	coding := DeinterleaveTSBK(channel)
	infoDibits, metric := DecodeTrellis(coding)
	info := make([]byte, 12)
	for i := 0; i < 12; i++ {
		info[i] = (infoDibits[4*i+0] << 6) |
			(infoDibits[4*i+1] << 4) |
			(infoDibits[4*i+2] << 2) |
			infoDibits[4*i+3]
	}
	return info, metric
}

// mbtCRC32 is the P25 packet CRC-32 (poly 0x04C11DB7, init 0, final
// XOR 0xFFFFFFFF, MSB-first, non-reflected) over the first bitLen bits
// of buf — the algorithm OP25 applies to a PDU's concatenated data
// blocks (translated there from p25craft.py).
func mbtCRC32(buf []byte, bitLen int) uint32 {
	const g = 0x04C11DB7
	var crc uint64
	for i := 0; i < bitLen; i++ {
		crc <<= 1
		b := uint64((buf[i/8] >> (7 - (i % 8))) & 1)
		if ((crc>>32)^b)&1 != 0 {
			crc ^= g
		}
	}
	return uint32(crc&0xFFFFFFFF) ^ 0xFFFFFFFF
}

// ValidateMBTData verifies the CRC-32 that terminates an MBT's data
// blocks: computed over all data octets except the last 4, compared to
// the last 4 octets. data is the concatenation of the BlocksToFollow
// 12-octet blocks.
func ValidateMBTData(data []byte) error {
	if len(data) < 4 || len(data)%12 != 0 {
		return fmt.Errorf("%w (len %d)", ErrMBTDataCRC, len(data))
	}
	want := binary.BigEndian.Uint32(data[len(data)-4:])
	if mbtCRC32(data, (len(data)-4)*8) != want {
		return ErrMBTDataCRC
	}
	return nil
}

// appendMBTDataCRC32 stamps the trailing CRC-32 into the last 4 octets
// of data (in place) so tests can build valid block trains.
func appendMBTDataCRC32(data []byte) {
	binary.BigEndian.PutUint32(data[len(data)-4:],
		mbtCRC32(data, (len(data)-4)*8))
}

// MBTNetworkStatusBroadcast is the AMBT form of opcode 0x3B. Unlike the
// TSBK form it carries explicit downlink AND uplink channels for the
// system's control channel. Offsets per SDRTrunk
// AMBTCNetworkStatusBroadcast:
//
//	header octets 3   : LRA
//	header octets 4-5 : System ID (12 bits)
//	block0 octets 0-2 : WACN (20 bits, top of the 24)
//	block0 octet 3-4  : downlink channel (4-bit band + 12-bit number)
//	block0 octet 5-6  : uplink channel   (4-bit band + 12-bit number)
//	block0 octet 7    : system service class
type MBTNetworkStatusBroadcast struct {
	LRA           uint8
	SystemID      uint16
	WACN          uint32
	ChannelID     uint8
	ChannelNumber uint16
	UplinkID      uint8
	UplinkNumber  uint16
	ServiceClass  uint8
}

// ParseMBTNetworkStatusBroadcast decodes an AMBT 0x3B from its header
// and first data block.
func ParseMBTNetworkStatusBroadcast(h MBTHeader, block0 []byte) MBTNetworkStatusBroadcast {
	dl := binary.BigEndian.Uint16(block0[3:5])
	ul := binary.BigEndian.Uint16(block0[5:7])
	return MBTNetworkStatusBroadcast{
		LRA:           h.Raw[3],
		SystemID:      uint16(h.Raw[4]&0x0F)<<8 | uint16(h.Raw[5]),
		WACN:          uint32(block0[0])<<12 | uint32(block0[1])<<4 | uint32(block0[2])>>4,
		ChannelID:     uint8(dl >> 12),
		ChannelNumber: dl & 0x0FFF,
		UplinkID:      uint8(ul >> 12),
		UplinkNumber:  ul & 0x0FFF,
		ServiceClass:  block0[7],
	}
}

// MBTAdjacentSiteStatusBroadcast is the AMBT form of opcode 0x3C, with
// the neighbour's explicit downlink and uplink channels. Offsets per
// SDRTrunk AMBTCAdjacentStatusBroadcast:
//
//	header octet 3    : LRA
//	header octets 4-5 : System ID (12 bits)
//	header octet 8    : RFSS ID
//	header octet 9    : Site ID
//	block0 octets 0-1 : downlink channel (4-bit band + 12-bit number)
//	block0 octets 2-3 : uplink channel   (4-bit band + 12-bit number)
type MBTAdjacentSiteStatusBroadcast struct {
	LRA           uint8
	SystemID      uint16
	RFSS, Site    uint8
	ChannelID     uint8
	ChannelNumber uint16
	UplinkID      uint8
	UplinkNumber  uint16
}

// ParseMBTAdjacentSiteStatusBroadcast decodes an AMBT 0x3C from its
// header and first data block.
func ParseMBTAdjacentSiteStatusBroadcast(h MBTHeader, block0 []byte) MBTAdjacentSiteStatusBroadcast {
	dl := binary.BigEndian.Uint16(block0[0:2])
	ul := binary.BigEndian.Uint16(block0[2:4])
	return MBTAdjacentSiteStatusBroadcast{
		LRA:           h.Raw[3],
		SystemID:      uint16(h.Raw[4]&0x0F)<<8 | uint16(h.Raw[5]),
		RFSS:          h.Raw[8],
		Site:          h.Raw[9],
		ChannelID:     uint8(dl >> 12),
		ChannelNumber: dl & 0x0FFF,
		UplinkID:      uint8(ul >> 12),
		UplinkNumber:  ul & 0x0FFF,
	}
}

// MBTRFSSStatusBroadcast is the AMBT form of opcode 0x3A. Offsets per
// SDRTrunk AMBTCRFSSStatusBroadcast:
//
//	header octet 3    : LRA
//	header octets 4-5 : System ID (12 bits)
//	block0 octet 0    : RFSS ID
//	block0 octet 1    : Site ID
//	block0 octets 2-3 : downlink channel (4-bit band + 12-bit number)
//	block0 octets 4-5 : uplink channel   (4-bit band + 12-bit number)
type MBTRFSSStatusBroadcast struct {
	LRA           uint8
	SystemID      uint16
	RFSS, Site    uint8
	ChannelID     uint8
	ChannelNumber uint16
	UplinkID      uint8
	UplinkNumber  uint16
}

// ParseMBTRFSSStatusBroadcast decodes an AMBT 0x3A from its header and
// first data block.
func ParseMBTRFSSStatusBroadcast(h MBTHeader, block0 []byte) MBTRFSSStatusBroadcast {
	dl := binary.BigEndian.Uint16(block0[2:4])
	ul := binary.BigEndian.Uint16(block0[4:6])
	return MBTRFSSStatusBroadcast{
		LRA:           h.Raw[3],
		SystemID:      uint16(h.Raw[4]&0x0F)<<8 | uint16(h.Raw[5]),
		RFSS:          block0[0],
		Site:          block0[1],
		ChannelID:     uint8(dl >> 12),
		ChannelNumber: dl & 0x0FFF,
		UplinkID:      uint8(ul >> 12),
		UplinkNumber:  ul & 0x0FFF,
	}
}
