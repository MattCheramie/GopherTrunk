package phase1

import (
	"errors"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// P25 LDU1 Link Control word.
//
// The 240-bit LC field (6 × 40-bit LC/ES blocks, as returned by
// ExtractLCESBlocks) carries a 72-bit Link Control word wrapped in two
// FEC layers: 24 shortened Hamming(10,6,3) codewords protect 144 bits,
// which are then the 24 six-bit symbols of an outer RS(24,12,13) code
// (TIA-102.BAAA). The first 12 symbols (72 bits) are the LC content; the
// trailing 12 are RS parity. ParseLinkControl decodes the inner Hamming
// layer and then runs the outer RS layer (framing.DecodeRS24_12) to
// correct up to t=6 residual symbol errors — without this, residual bit
// errors that survive the single-error Hamming layer corrupt the
// talkgroup/source under marginal SNR.
//
// LC content layout for the Group Voice Channel User LCO (0x00), per
// TIA-102.AABF. A symmetric encoder (AssembleLinkControl) keeps the
// on-wire mapping in one place:
//
//	octet 0   : Link Control Format (the LCO opcode)
//	octet 1   : manufacturer's ID (MFID)
//	octet 2   : service options
//	octet 3   : reserved
//	octets 4-5: talkgroup / destination group address (16 bits)
//	octets 6-8: source unit ID (24 bits)
//
// NOTE: a previous working model placed the talkgroup at octets 2-3 and
// the source at octets 4-6. That shifted both fields: the talkgroup
// decoded to the (constant) service-options byte — e.g. 0x0400 = 1024 —
// while the real talkgroup landed inside the misread source field. With
// the on-air talkgroup never matching the granted talkgroup, the voice
// composer's foreign-TG gate ended every call after ~2 LDU1s, fragmenting
// a single transmission into many tiny recordings (the field capture).
type LinkControl struct {
	LCFormat       uint8
	MFID           uint8
	ServiceOptions uint8
	TalkgroupID    uint16
	SourceID       uint32
}

// lcContentOctets is the LC content size in octets (72 bits).
const lcContentOctets = 9

// Standard TIA-102.AABF Link Control Opcodes the talker-alias path
// needs. These LCOs carry a radio's display name across an active
// voice channel as a HEADER + data BLOCK fragments — distinct from
// the Motorola vendor TSBK alias format the control channel already
// handles (see tsbk_vendor.go's OpVendorTalkerAlias).
//
// The LDU1 voice-channel form is decoded today: p25p1_voice.go
// dispatches IsTalkerAliasLCO LCs into MotorolaTalkerAliasBuf. The
// remaining #376 gap is the TDULC carriage — field reports
// (SDRTrunk ground truth on Victorian MMR) show Phase 1 systems
// emit the same 0x15/0x17 alias LCs in the TERMINATOR (TDULC, DUID
// 0xF) link control, NOT LDU1, so the voice chain misses them: it
// returns at the terminator before parsing the TDULC LC, and no
// TDULC LC extractor exists yet. The constants live here so
// consumers can recognise the opcodes when the LC is parsed.
const (
	LCOGroupVoiceChannelUser = 0x00 // the common LCO carrying TG + SRC
	LCOTalkerAliasHeader     = 0x15 // first frame: char-set + total length
	LCOTalkerAliasBlock1     = 0x16 // first alias-data fragment
	LCOTalkerAliasBlock2     = 0x17 // second alias-data fragment
)

// IsTalkerAliasLCO reports whether the given LC opcode carries a
// fragment of a standard P25 voice-channel talker alias. Callers in
// the voice composer use this to spot LCs they should buffer rather
// than treat as a group voice channel user.
func IsTalkerAliasLCO(lcf uint8) bool {
	return lcf == LCOTalkerAliasHeader ||
		lcf == LCOTalkerAliasBlock1 ||
		lcf == LCOTalkerAliasBlock2
}

// ErrLinkControlLength is returned when ParseLinkControl is handed
// blocks that are not each LDULCESBlockBits long.
var ErrLinkControlLength = errors.New("p25/phase1: LC blocks must each be 40 bits")

// ErrLinkControlUncorrectable is returned when the outer RS(24,12,13)
// layer cannot correct the Link Control word — more than t=6 symbol
// errors survived the inner Hamming layer. The decoded content is then
// low-confidence and callers should not trust its talkgroup/source.
var ErrLinkControlUncorrectable = errors.New("p25/phase1: Link Control RS-uncorrectable")

// lcesSymbolCount is the number of 6-bit GF(2^6) symbols in the 144-bit
// inner-decoded LC/ES field: 24 symbols, the RS codeword. Each symbol
// occupies exactly one inner Hamming(10,6,3) codeword's data bits.
const lcesSymbolCount = 24

// bitsToRSSymbols groups the 144 inner-decoded data bits (one bit per
// byte, MSB-first) into the 24 six-bit GF(2^6) RS symbols. The 6-bit
// grouping aligns with the inner Hamming(10,6,3) codeword boundaries.
func bitsToRSSymbols(bits []byte) [lcesSymbolCount]byte {
	var sym [lcesSymbolCount]byte
	for i := 0; i < lcesSymbolCount; i++ {
		var s byte
		for j := 0; j < 6; j++ {
			s = s<<1 | (bits[i*6+j] & 1)
		}
		sym[i] = s
	}
	return sym
}

// rsSymbolsToBits is the inverse of bitsToRSSymbols for the first k
// symbols, producing k*6 content bits (MSB-first).
func rsSymbolsToBits(sym []byte) []byte {
	out := make([]byte, len(sym)*6)
	for i, s := range sym {
		for j := 0; j < 6; j++ {
			out[i*6+j] = (s >> uint(5-j)) & 1
		}
	}
	return out
}

// rsCorrectLC applies the outer RS(24,12,13) layer to the 144 data bits
// returned by lcInnerDecode and returns the corrected 72 content bits
// (the 12 information symbols), the symbol-error count, and
// ErrLinkControlUncorrectable when beyond the correction radius.
func rsCorrectLC(data []byte) ([]byte, int, error) {
	cw := bitsToRSSymbols(data)
	info, nErr, err := framing.DecodeRS24_12(cw[:])
	if err != nil {
		return nil, 0, ErrLinkControlUncorrectable
	}
	return rsSymbolsToBits(info[:]), nErr, nil
}

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
// LinkControl. The inner Hamming(10,6,3) layer is decoded first, then the
// outer RS(24,12,13) layer corrects the residual symbol errors that
// otherwise corrupt the talkgroup/source under marginal SNR. It returns
// the parsed word, the total corrected-error count (inner bit errors plus
// outer symbol errors), and ErrLinkControlUncorrectable when the RS layer
// cannot recover the word.
func ParseLinkControl(blocks [LDULCESBlockCount][]byte) (LinkControl, int, error) {
	data, innerErrs, err := lcInnerDecode(blocks)
	if err != nil {
		return LinkControl{}, 0, err
	}
	content, rsErrs, rerr := rsCorrectLC(data)
	if rerr != nil {
		return LinkControl{}, innerErrs, rerr
	}
	oct := bitsToOctets(content)
	return LinkControl{
		LCFormat:       oct[0],
		MFID:           oct[1],
		ServiceOptions: oct[2],
		TalkgroupID:    uint16(oct[4])<<8 | uint16(oct[5]),
		SourceID:       uint32(oct[6])<<16 | uint32(oct[7])<<8 | uint32(oct[8]),
	}, innerErrs + rsErrs, nil
}

// ParseLinkControlContent decodes the 6 LC blocks of an LDU1 into the
// raw 9 LC content octets plus the corrected-error count, applying the
// same inner Hamming + outer RS correction as ParseLinkControl. Used by
// the talker-alias path so opcodes other than 0x00 (group voice channel
// user) can read their own octet layout without going through
// ParseLinkControl's LCO-0-shaped struct.
func ParseLinkControlContent(blocks [LDULCESBlockCount][]byte) ([lcContentOctets]byte, int, error) {
	var out [lcContentOctets]byte
	data, innerErrs, err := lcInnerDecode(blocks)
	if err != nil {
		return out, 0, err
	}
	content, rsErrs, rerr := rsCorrectLC(data)
	if rerr != nil {
		return out, innerErrs, rerr
	}
	copy(out[:], bitsToOctets(content))
	return out, innerErrs + rsErrs, nil
}

// AssembleLinkControl is the inverse of ParseLinkControl; it builds the
// 6 on-wire LC blocks for a LinkControl word, computing the outer
// RS(24,12,13) parity so the word round-trips through the RS-correcting
// parsers (and so synthetic test signals carry valid outer FEC).
func AssembleLinkControl(lc LinkControl) [LDULCESBlockCount][]byte {
	oct := make([]byte, lcContentOctets)
	oct[0] = lc.LCFormat
	oct[1] = lc.MFID
	oct[2] = lc.ServiceOptions
	// oct[3] reserved (0)
	oct[4], oct[5] = byte(lc.TalkgroupID>>8), byte(lc.TalkgroupID)
	oct[6], oct[7], oct[8] = byte(lc.SourceID>>16), byte(lc.SourceID>>8), byte(lc.SourceID)

	return lcInnerEncode(rsEncodeContentBits(octetsToBits(oct), 12))
}

// rsEncodeContentBits takes the leading content bits (k*6 of them) of an
// LC/ES word and returns the full 144-bit inner-data field with the
// outer RS parity symbols appended. k is 12 for Link Control
// (RS(24,12,13)) and 16 for Encryption Sync (RS(24,16,9)).
func rsEncodeContentBits(content []byte, k int) []byte {
	switch k {
	case 12:
		var info [12]byte
		for i := 0; i < 12; i++ {
			for j := 0; j < 6; j++ {
				info[i] = info[i]<<1 | (content[i*6+j] & 1)
			}
		}
		cw := framing.EncodeRS24_12(info)
		return rsSymbolsToBits(cw[:])
	case 16:
		var info [16]byte
		for i := 0; i < 16; i++ {
			for j := 0; j < 6; j++ {
				info[i] = info[i]<<1 | (content[i*6+j] & 1)
			}
		}
		cw := framing.EncodeRS24_16(info)
		return rsSymbolsToBits(cw[:])
	default:
		return make([]byte, 144)
	}
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
