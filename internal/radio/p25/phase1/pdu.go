package phase1

import "errors"

// P25 Packet Data Unit (PDU) — the data-channel counterpart of the
// voice LDU and the trunking TSDU. A PDU is a header block followed by
// N data blocks, each a 12-byte (96-bit) post-FEC unit. This file
// decodes the structured PDU from already-FEC-decoded block bytes and
// reassembles the data blocks into a payload (typically an SNDCP
// message — see sndcp.go).
//
// The block bit layout is the project's working model — TIA-102.BAAA's
// PDU tables are not in the repo — confined here with symmetric
// encoders and defensive parsers. Receiver-level PDU framing (the
// dibit-stream FSW → NID → trellis-coded-block chain for DUID 0xC) is a
// documented follow-up: it needs the same machinery the TSDU path uses
// wired for a new DUID, best built against real packet-data captures.

// PDUFormat identifies the PDU's format/type.
type PDUFormat uint8

const (
	PDUFmtResponse        PDUFormat = 0x03 // response PDU
	PDUFmtUnconfirmedData PDUFormat = 0x15 // unconfirmed user data
	PDUFmtConfirmedData   PDUFormat = 0x16 // confirmed user data
)

// PDUBlockBytes is the byte length of one post-FEC PDU block.
const PDUBlockBytes = 12

// ErrPDUBlockLength is returned when a PDU block is not PDUBlockBytes long.
var ErrPDUBlockLength = errors.New("p25/phase1: PDU block must be 12 bytes")

// PDUHeader is the structured PDU header block. Working-model layout:
//
//	byte 0     : bit 6 = confirmed flag, bits 0-5 = format
//	byte 1     : bits 0-5 = SAP (Service Access Point)
//	byte 2     : MFID
//	bytes 3-5  : LLID — 24-bit logical link (destination) ID
//	byte 6     : bits 0-6 = data-block count
type PDUHeader struct {
	Format     PDUFormat
	Confirmed  bool
	SAP        uint8
	MFID       uint8
	LLID       uint32
	BlockCount uint8
}

// ParsePDUHeader decodes a 12-byte PDU header block.
func ParsePDUHeader(b []byte) (PDUHeader, error) {
	if len(b) < PDUBlockBytes {
		return PDUHeader{}, ErrPDUBlockLength
	}
	return PDUHeader{
		Format:     PDUFormat(b[0] & 0x3F),
		Confirmed:  b[0]&0x40 != 0,
		SAP:        b[1] & 0x3F,
		MFID:       b[2],
		LLID:       uint32(b[3])<<16 | uint32(b[4])<<8 | uint32(b[5]),
		BlockCount: b[6] & 0x7F,
	}, nil
}

// AssemblePDUHeader is the inverse of ParsePDUHeader.
func AssemblePDUHeader(h PDUHeader) []byte {
	b := make([]byte, PDUBlockBytes)
	b[0] = byte(h.Format) & 0x3F
	if h.Confirmed {
		b[0] |= 0x40
	}
	b[1] = h.SAP & 0x3F
	b[2] = h.MFID
	b[3], b[4], b[5] = byte(h.LLID>>16), byte(h.LLID>>8), byte(h.LLID)
	b[6] = h.BlockCount & 0x7F
	return b
}

// PDU is a fully-reassembled packet data unit.
type PDU struct {
	Header  PDUHeader
	Payload []byte
}

// ReassemblePDU concatenates a PDU's data blocks into its payload. The
// blocks are the post-FEC data blocks (each PDUBlockBytes long), in
// order. A confirmed PDU's per-block serial-number / CRC octets are not
// stripped here — that is a documented follow-up.
func ReassemblePDU(h PDUHeader, blocks [][]byte) (PDU, error) {
	payload := make([]byte, 0, len(blocks)*PDUBlockBytes)
	for _, blk := range blocks {
		if len(blk) != PDUBlockBytes {
			return PDU{}, ErrPDUBlockLength
		}
		payload = append(payload, blk...)
	}
	return PDU{Header: h, Payload: payload}, nil
}
