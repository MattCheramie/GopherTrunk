package dmr

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// Embedded signalling carried by the 48-bit sync field of DMR voice
// bursts B–F (ETSI TS 102 361-1 §6.2, §9.1.4). The field is laid out as
//
//	bits  0..7   : EMB MSB half  (8 bits)
//	bits  8..39  : embedded signalling fragment (32 bits)
//	bits 40..47  : EMB LSB half  (8 bits)
//
// The 16-bit EMB (CC, PI, LCSS + FEC) frames the fragment; the four
// fragments of bursts B–E reassemble into a 128-bit embedded Link
// Control block — see framing.DecodeEmbeddedLC.
//
// NOTE: the EMB info bits are read systematically here; the EMB's own
// QR(16,7) error-correction is not yet applied (the embedded-LC BPTC +
// CRC downstream is the integrity gate). This mirrors the
// capture-pending posture documented for the BPTC layers; see
// docs/status.md.

// LCSS is the 2-bit Link Control Start/Stop field in the EMB, marking a
// fragment's position within a multi-burst embedded LC.
type LCSS uint8

const (
	LCSSSingle LCSS = 0 // 00 — single-fragment LC (CSBK/RC) or null
	LCSSFirst  LCSS = 1 // 01 — first fragment of a Full LC (burst B)
	LCSSLast   LCSS = 2 // 10 — last fragment of a Full LC (burst E)
	LCSSCont   LCSS = 3 // 11 — continuation fragment (bursts C, D)
)

// EMB is the decoded 16-bit Embedded signalling header.
type EMB struct {
	ColorCode uint8 // 4-bit
	PI        bool  // privacy indicator
	LCSS      LCSS  // 2-bit fragment position
}

// systematic EMB bit positions within the 16-bit field (MSB-first):
// CC in bits 15..12, PI in bit 11, LCSS in bits 10..9, bits 8..0 spare.
func encodeEMB(e EMB) uint16 {
	return uint16(e.ColorCode&0x0F)<<12 |
		boolToBit(e.PI)<<11 |
		uint16(e.LCSS&0x03)<<9
}

func decodeEMB(v uint16) EMB {
	return EMB{
		ColorCode: uint8((v >> 12) & 0x0F),
		PI:        (v>>11)&1 == 1,
		LCSS:      LCSS((v >> 9) & 0x03),
	}
}

func boolToBit(b bool) uint16 {
	if b {
		return 1
	}
	return 0
}

// SplitEmbeddedField splits the 48 bits of a voice burst's sync field
// (one bit per byte, MSB-first — i.e. bitsFromDibits(burst.Sync())) into
// the decoded EMB header and the 32-bit embedded-signalling fragment.
func SplitEmbeddedField(field []byte) (EMB, []byte) {
	if len(field) != SyncBitCount {
		panic("dmr: embedded field requires 48 bits")
	}
	var emb uint16
	for i := 0; i < 8; i++ { // EMB MSB half: field[0..7] → emb bits 15..8
		emb |= uint16(field[i]&1) << uint(15-i)
	}
	for i := 0; i < 8; i++ { // EMB LSB half: field[40..47] → emb bits 7..0
		emb |= uint16(field[40+i]&1) << uint(7-i)
	}
	frag := make([]byte, framing.EmbeddedFragmentBits) // field[8..39]
	copy(frag, field[8:8+framing.EmbeddedFragmentBits])
	return decodeEMB(emb), frag
}

// AssembleEmbeddedField is the inverse of SplitEmbeddedField: it packs an
// EMB header and a 32-bit fragment into the 48-bit voice-burst sync
// field. Used to build synthetic streams.
func AssembleEmbeddedField(e EMB, frag []byte) []byte {
	if len(frag) != framing.EmbeddedFragmentBits {
		panic("dmr: embedded fragment requires 32 bits")
	}
	field := make([]byte, SyncBitCount)
	emb := encodeEMB(e)
	for i := 0; i < 8; i++ {
		field[i] = byte((emb >> uint(15-i)) & 1)
	}
	copy(field[8:8+framing.EmbeddedFragmentBits], frag)
	for i := 0; i < 8; i++ {
		field[40+i] = byte((emb >> uint(7-i)) & 1)
	}
	return field
}

// ReassembleEmbeddedLC concatenates the four ordered 32-bit fragments
// from bursts B, C, D and E into the 128-bit embedded Link Control
// block, decodes it (variable BPTC(128,72) + CRC), and parses the
// recovered 72 bits as a Full Link Control PDU. It returns ok=false when
// any fragment is the wrong length, the BPTC/CRC check fails, or the FLC
// cannot be parsed.
func ReassembleEmbeddedLC(frags [4][]byte) (FLC, bool) {
	_, flc, ok := ReassembleEmbeddedLCInfo(frags)
	return flc, ok
}

// ReassembleEmbeddedLCInfo is ReassembleEmbeddedLC but also returns the
// raw 9-octet recovered info block. Callers that need the payload octets
// verbatim — e.g. the talker-alias fragment parser, whose octets 2..8 are
// alias text rather than the ServiceOptions/address fields ParseFLC
// assumes — use this to read the bits before the generic FLC view
// reinterprets them.
func ReassembleEmbeddedLCInfo(frags [4][]byte) ([]byte, FLC, bool) {
	channel := make([]byte, 0, framing.EmbChannelBits)
	for _, f := range frags {
		if len(f) != framing.EmbeddedFragmentBits {
			return nil, FLC{}, false
		}
		channel = append(channel, f...)
	}
	lcBits, corrected := framing.DecodeEmbeddedLC(channel)
	if corrected < 0 {
		return nil, FLC{}, false
	}
	info := make([]byte, framing.EmbLCBits/8) // 9 octets
	for i := 0; i < framing.EmbLCBits; i++ {
		if lcBits[i]&1 != 0 {
			info[i/8] |= 1 << uint(7-(i%8))
		}
	}
	flc, err := ParseFLC(info)
	if err != nil {
		return nil, FLC{}, false
	}
	return info, flc, true
}
