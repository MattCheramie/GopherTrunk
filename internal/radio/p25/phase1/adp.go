package phase1

import (
	"encoding/binary"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// P25 ADP ("Advanced Digital Privacy", ALGID 0xAA — RC4) known-key
// descramble for the Phase 1 voice path (issue #1187).
//
// The keystream construction is the one OP25 and DSD-FME use on air and the
// cryptolab already realises (p25crypto.Keystream): RC4 keyed with the
// 40-bit key followed by the first 8 octets of the Message Indicator, with
// the first 256 keystream bytes discarded. What this file adds is the
// mapping of that byte stream onto the LDU voice frames and the Message
// Indicator schedule — the two things a "keystream generator" leaves to the
// caller, and the two things a wrong guess silently breaks:
//
//   - One superframe (LDU1 + LDU2, 360 ms) consumes 469 absolute keystream
//     bytes: 256 discarded, 11 skipped, then LDU1's nine 88-bit (11-byte)
//     IMBE frames contiguously from byte 267 and LDU2's from byte 368, with
//     a 2-byte gap before the ninth frame of each LDU (the Low Speed Data
//     word sits between u7 and u8). The XOR is applied to the FEC-decoded
//     88-bit frame, MSB-first packed, exactly as the recorder stores it.
//     (Reference: OP25 p25_crypt_algs::adp_process — offset 267 + 11·i,
//     +2 at i = 8, +101 for LDU2.)
//   - The Encryption Sync an LDU2 carries announces the MI of the NEXT
//     superframe, and successive MIs advance through the 64-bit LFSR
//     x^64 + x^62 + x^46 + x^38 + x^27 + x^15 + 1 (TIA-102.AAAD; DSD-FME's
//     LFSR64). AdvanceMI runs it forward; RewindMI inverts it, so a chain
//     that enters on an LDU2 — or that never sees the HDU, which
//     GopherTrunk's LDU assembler does not deliver — recovers the CURRENT
//     superframe's MI from the first ES it decodes instead of losing the
//     first 360 ms.
//
// Both facts are capture-pinned, not spec-derived (this repo's #764/#771
// rule): adp_test.go replays the #1187 reporter's own 6.2 s ADP call
// (testdata/adp_issue1187_ldus.json, key 1234567890) — the OP25 layout
// under the next-superframe rule turns the IMBE pitch track from
// ciphertext-random to speech-continuous, the alternatives do not, and
// every consecutive pair of the fifteen MIs it carries satisfies AdvanceMI.

const (
	// ADPKeyBytes is the ADP key length (40 bits).
	ADPKeyBytes = 5
	// ADPMIBytes is how much of the 72-bit Message Indicator seeds RC4.
	ADPMIBytes = 8
	// adpKeystreamDrop is the RC4 warm-up discarded before use.
	adpKeystreamDrop = 256
	// adpSuperframeKeystreamBytes is one superframe's absolute keystream
	// span (discard included): 267 + 2·(9·11 + 2).
	adpSuperframeKeystreamBytes = 469
	// ADPSuperframeKeystreamBytes is the keystream one superframe consumes
	// after the warm-up — the length ADPSuperframeKeystream returns.
	ADPSuperframeKeystreamBytes = adpSuperframeKeystreamBytes - adpKeystreamDrop
	// adpLDU1VoiceBase / adpLDU2VoiceBase are the absolute keystream
	// offsets of voice subframe u0 in LDU1 and LDU2.
	adpLDU1VoiceBase = 267
	adpLDU2VoiceBase = 368
	// adpLSDSkip is the keystream the Low Speed Data word consumes between
	// u7 and u8.
	adpLSDSkip = 2
)

// ADPVoiceFrameOffset returns the absolute keystream offset (counting the
// 256 discarded warm-up bytes, as OP25 does) of voice subframe `subframe`
// of an LDU of type duid. ok is false for a non-voice DUID or an
// out-of-range subframe.
func ADPVoiceFrameOffset(duid DUID, subframe int) (off int, ok bool) {
	if subframe < 0 || subframe >= LDUVoiceSubframeCount {
		return 0, false
	}
	switch duid {
	case DUIDLogicalLink1:
		off = adpLDU1VoiceBase
	case DUIDLogicalLink2:
		off = adpLDU2VoiceBase
	default:
		return 0, false
	}
	off += imbe.FrameBytes * subframe
	if subframe == LDUVoiceSubframeCount-1 {
		off += adpLSDSkip
	}
	return off, true
}

// ADPSuperframeKeystream returns the keystream one superframe (LDU1 + LDU2)
// scrambled under mi consumes, warm-up already discarded: index it with
// ADPVoiceFrameOffset(…) − 256, or hand it to ADPDescrambleVoiceFrames.
func ADPSuperframeKeystream(key []byte, mi [9]byte) ([]byte, error) {
	if len(key) != ADPKeyBytes {
		return nil, fmt.Errorf("p25/phase1: ADP key must be %d bytes, got %d", ADPKeyBytes, len(key))
	}
	return p25crypto.Keystream(p25crypto.AlgADP, key, mi[:ADPMIBytes], ADPSuperframeKeystreamBytes)
}

// ADPDescrambleVoiceFrames XORs the nine FEC-decoded 11-byte IMBE frames of
// an LDU of type duid, in place, with their slots of ks (as returned by
// ADPSuperframeKeystream for the superframe's MI). A nil frame — one whose
// FEC failed — is skipped but still keeps its keystream slot, so the rest
// stay aligned. Returns the number of frames descrambled.
func ADPDescrambleVoiceFrames(ks []byte, duid DUID, frames *[LDUVoiceSubframeCount][]byte) (int, error) {
	if len(ks) < ADPSuperframeKeystreamBytes {
		return 0, fmt.Errorf("p25/phase1: ADP keystream is %d bytes, need %d", len(ks), ADPSuperframeKeystreamBytes)
	}
	if duid != DUIDLogicalLink1 && duid != DUIDLogicalLink2 {
		return 0, fmt.Errorf("p25/phase1: ADP descramble of a %v (not a voice LDU)", duid)
	}
	n := 0
	for i := range frames {
		f := frames[i]
		if f == nil {
			continue
		}
		if len(f) != imbe.FrameBytes {
			return n, fmt.Errorf("p25/phase1: ADP voice subframe %d is %d bytes, want %d", i, len(f), imbe.FrameBytes)
		}
		off, _ := ADPVoiceFrameOffset(duid, i)
		off -= adpKeystreamDrop
		for j := range f {
			f[j] ^= ks[off+j]
		}
		n++
	}
	return n, nil
}

// AdvanceMI returns the Message Indicator of the superframe that follows
// one scrambled under mi: the 64-bit LFSR x^64 + x^62 + x^46 + x^38 + x^27
// + x^15 + 1 clocked 64 times over the first eight octets (TIA-102.AAAD).
// The ninth octet is carried through unchanged.
func AdvanceMI(mi [9]byte) [9]byte {
	v := binary.BigEndian.Uint64(mi[:8])
	for i := 0; i < 64; i++ {
		bit := ((v >> 63) ^ (v >> 61) ^ (v >> 45) ^ (v >> 37) ^ (v >> 26) ^ (v >> 14)) & 1
		v = v<<1 | bit
	}
	var out [9]byte
	binary.BigEndian.PutUint64(out[:8], v)
	out[8] = mi[8]
	return out
}

// RewindMI inverts AdvanceMI: given the MI an LDU2's Encryption Sync
// announces for the next superframe, it returns the MI of the superframe
// that LDU2 belongs to. Each LFSR step is invertible — the bit shifted out
// at the top is recovered from the feedback bit that was shifted in.
func RewindMI(mi [9]byte) [9]byte {
	v := binary.BigEndian.Uint64(mi[:8])
	for i := 0; i < 64; i++ {
		fed := v & 1
		prev := v >> 1
		// fed = old63 ^ old61 ^ old45 ^ old37 ^ old26 ^ old14, and every old
		// bit but old63 is one position higher in the shifted value.
		top := fed ^ ((prev>>61)^(prev>>45)^(prev>>37)^(prev>>26)^(prev>>14))&1
		v = prev | top<<63
	}
	var out [9]byte
	binary.BigEndian.PutUint64(out[:8], v)
	out[8] = mi[8]
	return out
}
