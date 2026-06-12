package dmr

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// Reverse Channel (RC) signalling — ETSI TS 102 361-1 §6.2, §9.1.4.
//
// The DMR Reverse Channel is in-call signalling, distinct from the CSBKs on
// the trunking control channel. It rides one of two carriers:
//
//   - the 32-bit embedded-signalling fragment of an ordinary burst, when the
//     framing EMB marks a single, non-LC fragment (LCSS == LCSSSingle); or
//   - a short MS-sourced RC burst framed by the MS-RC sync (dmr.MSRC).
//
// In the 32-bit embedded field the RC is 11 bits of information protected by
// 21 bits of FEC parity, framed by the 16-bit EMB (colour code, PI, LCSS;
// itself QR(16,7)-protected). This file decodes the embedded carrier — the
// one a downlink receiver actually observes, since it is already pulling the
// embedded field out of every voice burst (see dmr/voice). The short MS-RC
// burst is a mobile *uplink* transmission a downlink scanner does not normally
// receive (its sync is detected by dmr.MSRC but the burst is not present on
// the downlink — see sync_test.go), so no payload decode is attempted for it.
//
// Provenance / honesty: the 11-info + 21-parity split and the EMB framing are
// pinned to ETSI TS 102 361-1, but GopherTrunk has no real off-air RC capture
// and no open reference decoder (dsd-fme, pd0mz/go-dmr) implements the (32,11)
// RC FEC, so the parity is preserved but not yet used to correct or validate
// the 11 information bits. This mirrors the capture-pending posture already
// documented for the EMB's own QR(16,7) (see emb.go) and the MBC last-block
// CRC. The integrity gate is therefore the caller's EMB framing check
// (LCSS == LCSSSingle) plus the null/idle rejection below; a future off-air
// capture can promote the parity to a real FEC gate without changing callers.

// RCInfoBits is the number of Reverse Channel information bits carried in the
// 32-bit embedded fragment; the remaining RCParityBits are FEC parity.
const (
	RCInfoBits   = 11
	RCParityBits = framing.EmbeddedFragmentBits - RCInfoBits // 21
)

// ReverseChannel is a decoded DMR Reverse Channel word.
type ReverseChannel struct {
	// Info is the 11-bit RC information field — the leading bits of the
	// 32-bit embedded fragment, MSB-first. Its per-opcode sub-field semantics
	// are left unparsed pending a real off-air RC capture.
	Info uint16
	// Parity is the 21-bit FEC parity protecting Info. Preserved raw; not yet
	// applied (see the package note above).
	Parity uint32
}

// DecodeReverseChannel parses the 32-bit embedded-signalling fragment (one bit
// per byte, MSB-first — the fragment returned by SplitEmbeddedField) as a
// Reverse Channel word. It returns ok=false when the fragment is the wrong
// length or is the ETSI null/idle embedded message (all-zero). The caller must
// confirm the framing EMB marks a single, non-LC fragment (its LCSS ==
// LCSSSingle) before treating a fragment as RC.
func DecodeReverseChannel(frag []byte) (ReverseChannel, bool) {
	if len(frag) != framing.EmbeddedFragmentBits {
		return ReverseChannel{}, false
	}
	ones := 0
	for _, b := range frag {
		ones += int(b & 1)
	}
	// The all-zero field is the ETSI "null embedded message" idle, not RC.
	if ones == 0 {
		return ReverseChannel{}, false
	}
	var rc ReverseChannel
	for i := 0; i < RCInfoBits; i++ {
		rc.Info = rc.Info<<1 | uint16(frag[i]&1)
	}
	for i := 0; i < RCParityBits; i++ {
		rc.Parity = rc.Parity<<1 | uint32(frag[RCInfoBits+i]&1)
	}
	return rc, true
}

// AssembleReverseChannel is the inverse of DecodeReverseChannel: it packs a
// Reverse Channel word into the 32-bit embedded fragment (11 info bits then 21
// parity bits, MSB-first). Used to build synthetic streams and round-trip
// tests.
func AssembleReverseChannel(rc ReverseChannel) []byte {
	frag := make([]byte, framing.EmbeddedFragmentBits)
	for i := 0; i < RCInfoBits; i++ {
		frag[i] = byte((rc.Info >> uint(RCInfoBits-1-i)) & 1)
	}
	for i := 0; i < RCParityBits; i++ {
		frag[RCInfoBits+i] = byte((rc.Parity >> uint(RCParityBits-1-i)) & 1)
	}
	return frag
}
