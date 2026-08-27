package phase1

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TSBK is one Trunking Signaling Block parsed from the TSDU. The header
// fields are common to every opcode; Payload holds the 8 opcode-specific
// octets (bits 16..79 of the 96-bit info block).
type TSBK struct {
	LB      bool   // Last Block in TSDU sequence
	P       bool   // Protected
	Opcode  Opcode // 6-bit opcode
	MFID    uint8  // Manufacturer ID (0x00 = standard)
	Payload [8]byte
}

// CRCError is returned by ParseTSBK when the trailer CRC fails.
var CRCError = fmt.Errorf("p25/phase1: TSBK CRC check failed")

// ErrTSBKInfoLength is returned when ParseTSBK receives an info block
// whose length isn't 12 bytes.
var ErrTSBKInfoLength = errors.New("p25/phase1: TSBK info must be 12 bytes")

// ParseTSBK consumes 96 info bits (12 bytes) and returns a parsed block.
// The trailer CRC is verified per the P25 TSBK convention
// (framing.CRCCCITTAugmented over all 12 bytes returns 0 on a valid
// codeword); CRCError is returned on mismatch (the partially-parsed
// TSBK is still returned so callers can log the contents for
// diagnostics).
//
// Issue #275 Phase B part 3: the prior implementation used the
// CRC-CCITT/FALSE algorithm (init 0xFFFF, no final XOR) and inverted
// the stored trailer for comparison. That algorithm is documented in
// many P25 references but is not what the spec uses on-air: TSBK
// trailers in the Mt Anakie capture verify cleanly under the
// "augmented codeword" CRC variant (init 0, final XOR 0xFFFF, run
// over all 12 bytes, expect 0). See framing.CRCCCITTAugmented and the
// OP25 cross-reference for the algorithmic derivation; the field
// symptom was 195/197 TSBK CRC failures on Mt Anakie even when the
// upstream trellis decoder reported metric=0 (clean path).
func ParseTSBK(info []byte) (TSBK, error) {
	if len(info) != 12 {
		return TSBK{}, fmt.Errorf("%w, got %d", ErrTSBKInfoLength, len(info))
	}
	var t TSBK
	t.LB = info[0]&0x80 != 0
	t.P = info[0]&0x40 != 0
	t.Opcode = Opcode(info[0] & 0x3F)
	t.MFID = info[1]
	copy(t.Payload[:], info[2:10])

	if framing.CRCCCITTAugmented(info) != 0 {
		return t, CRCError
	}
	return t, nil
}

// AssembleTSBK constructs a 12-byte TSBK info block from the structured
// fields. Used in tests and for any future encoder work. The trailer
// is the augmented-CRC value of (info ‖ 16 zero bits) per the P25
// spec convention — that's the unique 16-bit value V such that
// CRCCCITTAugmented(info ‖ V) returns 0.
func AssembleTSBK(t TSBK) []byte {
	out := make([]byte, 12)
	if t.LB {
		out[0] |= 0x80
	}
	if t.P {
		out[0] |= 0x40
	}
	out[0] |= byte(t.Opcode) & 0x3F
	out[1] = t.MFID
	copy(out[2:10], t.Payload[:])
	binary.BigEndian.PutUint16(out[10:12], framing.CRCCCITTAugmented(out))
	return out
}

// EncodeTSBKChannel turns a 12-byte TSBK info block into the 98 channel
// dibits transmitted on-air: trellis encode → block interleave. Useful
// for synthetic test streams.
func EncodeTSBKChannel(info []byte) []uint8 {
	if len(info) != 12 {
		panic("p25/phase1: TSBK info must be 12 bytes")
	}
	dibits := make([]uint8, 48)
	for i := 0; i < 12; i++ {
		b := info[i]
		dibits[4*i+0] = (b >> 6) & 0x3
		dibits[4*i+1] = (b >> 4) & 0x3
		dibits[4*i+2] = (b >> 2) & 0x3
		dibits[4*i+3] = b & 0x3
	}
	coding := EncodeTrellis(dibits)
	return InterleaveTSBK(coding)
}

// DecodeTSBKChannel runs the receive-side pipeline over 98 channel
// dibits: deinterleave → Viterbi → repack into 12 bytes → ParseTSBK.
// Returns the parsed TSBK, the Viterbi path metric (sum of dibit-
// distance penalties; 0 = clean channel, higher = more correction),
// and an error chained from CRCError or ErrTSBKInfoLength on mismatch.
// CRC failure still returns the partially-parsed block so callers can
// log it for diagnostics.
func DecodeTSBKChannel(channel []uint8) (TSBK, int, error) {
	if len(channel) != 98 {
		return TSBK{}, -1, fmt.Errorf("p25/phase1: TSBK channel must be 98 dibits, got %d", len(channel))
	}
	coding := DeinterleaveTSBK(channel)
	infoDibits, metric := DecodeTrellis(coding)
	info := make([]byte, 12)
	for i := 0; i < 12; i++ {
		info[i] = (infoDibits[4*i+0] << 6) |
			(infoDibits[4*i+1] << 4) |
			(infoDibits[4*i+2] << 2) |
			infoDibits[4*i+3]
	}
	tsbk, err := ParseTSBK(info)
	return tsbk, metric, err
}

// DecodeTSBKChannelSoft is the soft-decision analog of DecodeTSBKChannel
// (p25_phase1_soft_decision): the same deinterleave → Viterbi → repack →
// ParseTSBK pipeline, carrying per-bit log-likelihood ratios instead of
// sliced dibits. llr holds TWO values per channel dibit — llr[2i] for
// channel dibit i's MSB, llr[2i+1] for its LSB, in the framing convention
// (LLR > 0 ⇒ bit 0) — 196 values for the 98-dibit TSBK block. The trellis
// runs framing.DecodeP25TrellisSoft (true per-bit soft Viterbi), and the
// TSBK's CRC16 still corroborates, so a soft mis-decode cannot manufacture
// a grant. The returned metric is the soft path metric (a correlation sum,
// lower = better — NOT the hard metric's corrected-dibit count).
func DecodeTSBKChannelSoft(llr []float32) (TSBK, float64, error) {
	if len(llr) != 2*98 {
		return TSBK{}, 1e30, fmt.Errorf("p25/phase1: TSBK soft channel must be 196 LLRs, got %d", len(llr))
	}
	// Deinterleave the per-dibit LLR pairs with the same permutation the
	// hard path applies to dibits: coding pair i ← channel pair perm[i].
	coding := make([]float32, 2*98)
	for i := 0; i < 98; i++ {
		j := tsbkDeinterleavePerm[i]
		coding[2*i] = llr[2*j]
		coding[2*i+1] = llr[2*j+1]
	}
	infoDibits, metric := framing.DecodeP25TrellisSoft(coding)
	if len(infoDibits) != 48 {
		return TSBK{}, 1e30, fmt.Errorf("p25/phase1: soft trellis returned %d info dibits, want 48", len(infoDibits))
	}
	info := make([]byte, 12)
	for i := 0; i < 12; i++ {
		info[i] = (infoDibits[4*i+0] << 6) |
			(infoDibits[4*i+1] << 4) |
			(infoDibits[4*i+2] << 2) |
			infoDibits[4*i+3]
	}
	tsbk, err := ParseTSBK(info)
	return tsbk, metric, err
}
