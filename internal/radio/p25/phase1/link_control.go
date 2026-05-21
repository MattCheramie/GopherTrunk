package phase1

import "errors"

// P25 LDU1 Link Control word.
//
// The 240-bit LC field (6 × 40-bit LC/ES blocks, as returned by
// ExtractLCESBlocks) carries a 72-bit Link Control word wrapped in FEC:
// 24 shortened Hamming(10,6,3) codewords protect 144 bits, of which the
// first 72 are the LC content and the remaining 72 are an RS(24,12,13)
// outer parity (TIA-102.BAAA). This file decodes the inner Hamming
// layer and parses the 72-bit LC content; the RS outer verification is
// a documented follow-up — without the spec's RS generator polynomial
// it cannot be done reliably, and the inner Hamming layer already
// single-error-corrects each codeword.
//
// LC content layout is the project's working model — TIA-102.BAAA's
// per-LCO field tables are not in the repo. It is confined here with a
// symmetric encoder so a spec correction stays local.
//
//	octet 0   : Link Control Format (the LCO opcode)
//	octet 1   : service options
//	octets 2-3: talkgroup / destination group
//	octets 4-6: source unit ID (24 bits)
//	octets 7-8: reserved
type LinkControl struct {
	LCFormat       uint8
	ServiceOptions uint8
	TalkgroupID    uint16
	SourceID       uint32
}

// lcContentOctets is the LC content size in octets (72 bits).
const lcContentOctets = 9

// ErrLinkControlLength is returned when ParseLinkControl is handed
// blocks that are not each LDULCESBlockBits long.
var ErrLinkControlLength = errors.New("p25/phase1: LC blocks must each be 40 bits")

// lcInnerDecode runs the 24 Hamming(10,6,3) inner codewords over the
// 240-bit LC/ES field and returns the 144 recovered data bits plus the
// total corrected-error count. It is shared by ParseLinkControl and the
// Encryption Sync parser.
func lcInnerDecode(blocks [LDULCESBlockCount][]byte) ([]byte, int, error) {
	var field []byte
	for _, b := range blocks {
		if len(b) != LDULCESBlockBits {
			return nil, 0, ErrLinkControlLength
		}
		field = append(field, b...)
	}
	// 240 bits = 24 codewords × 10 bits → 24 × 6 = 144 data bits.
	data := make([]byte, 0, 144)
	totalErrs := 0
	for i := 0; i < 24; i++ {
		dec, errs := decodeHamming10_6(field[i*10 : i*10+10])
		totalErrs += errs
		data = append(data, dec...)
	}
	return data, totalErrs, nil
}

// lcInnerEncode is the inverse of lcInnerDecode: 144 data bits → 240
// on-wire bits as 6 × 40-bit blocks.
func lcInnerEncode(data []byte) [LDULCESBlockCount][]byte {
	var field []byte
	for i := 0; i < 24; i++ {
		field = append(field, encodeHamming10_6(data[i*6:i*6+6])...)
	}
	var blocks [LDULCESBlockCount][]byte
	for j := range blocks {
		blocks[j] = append([]byte(nil), field[j*LDULCESBlockBits:(j+1)*LDULCESBlockBits]...)
	}
	return blocks
}

// ParseLinkControl decodes the 6 LC blocks of an LDU1 into a structured
// LinkControl. It returns the parsed word and the inner-FEC corrected-
// error count.
func ParseLinkControl(blocks [LDULCESBlockCount][]byte) (LinkControl, int, error) {
	data, errs, err := lcInnerDecode(blocks)
	if err != nil {
		return LinkControl{}, 0, err
	}
	oct := bitsToOctets(data[:lcContentOctets*8])
	return LinkControl{
		LCFormat:       oct[0],
		ServiceOptions: oct[1],
		TalkgroupID:    uint16(oct[2])<<8 | uint16(oct[3]),
		SourceID:       uint32(oct[4])<<16 | uint32(oct[5])<<8 | uint32(oct[6]),
	}, errs, nil
}

// AssembleLinkControl is the inverse of ParseLinkControl; it builds the
// 6 on-wire LC blocks for a LinkControl word. The RS-parity half of the
// 144-bit data field is left zero (see the package note above).
func AssembleLinkControl(lc LinkControl) [LDULCESBlockCount][]byte {
	oct := make([]byte, lcContentOctets)
	oct[0] = lc.LCFormat
	oct[1] = lc.ServiceOptions
	oct[2], oct[3] = byte(lc.TalkgroupID>>8), byte(lc.TalkgroupID)
	oct[4], oct[5], oct[6] = byte(lc.SourceID>>16), byte(lc.SourceID>>8), byte(lc.SourceID)

	data := make([]byte, 144)
	copy(data, octetsToBits(oct))
	return lcInnerEncode(data)
}

// LDUDuid returns the DUID encoded in an on-air LDU's NID — used to
// tell an LDU1 (Link Control) from an LDU2 (Encryption Sync) before
// interpreting its 6 LC/ES blocks.
func LDUDuid(ldu []byte) (DUID, error) {
	payload, err := StripStatusSymbols(ldu)
	if err != nil {
		return 0, err
	}
	nid, _, err := ParseNID(payload[lduNIDOffset : lduNIDOffset+LDUNIDBits])
	if err != nil {
		return 0, err
	}
	return nid.DUID, nil
}

// bitsToOctets packs a bit slice (0/1 per byte, MSB-first) into octets.
func bitsToOctets(bits []byte) []byte {
	out := make([]byte, len(bits)/8)
	for i := range out {
		var b byte
		for j := 0; j < 8; j++ {
			b = b<<1 | (bits[i*8+j] & 1)
		}
		out[i] = b
	}
	return out
}

// octetsToBits is the inverse of bitsToOctets.
func octetsToBits(oct []byte) []byte {
	out := make([]byte, len(oct)*8)
	for i, b := range oct {
		for j := 0; j < 8; j++ {
			out[i*8+j] = (b >> uint(7-j)) & 1
		}
	}
	return out
}
