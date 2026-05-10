package phase1

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// P25 Phase 1 Logical Data Unit (LDU) structural primitives.
//
// Reference: TIA-102.BAAA-A § 8 (Logical Link Data Units 1 and 2),
// as reproduced in Figures 8-3 and 8-4 of "Security Weaknesses in
// the APCO Project 25 Two-Way Radio System" (Clark, Metzger,
// Wasserman, Xu, Blaze; UPenn CIS Tech Report MS-CIS-10-34, 2010).
//
// An LDU is the unit of voice transmission. Each LDU carries 9 IMBE
// voice subframes (each 144 channel bits = 20 ms of audio @ 8 kHz)
// plus protocol metadata. LDU1 and LDU2 alternate during a voice
// transmission and differ only in whether the metadata field
// carries Link Control (LDU1) or Encryption Sync (LDU2):
//
//	1728 bits total = 48 (FS) + 64 (NID) + 9·144 (Voice)
//	                + 240 (LC or ES) + 32 (LSD) + 24·2 (Status)
//
// where:
//
//	FS  — frame sync, fixed pattern marking the LDU start.
//	NID — network ID + DUID (already parsed by the existing
//	      ParseNID / NIDFromDibits in nid.go).
//	Voice — 9 IMBE subframes, each 144 post-deinterleave bits.
//	        Hand each to imbe.DecodeChannelToFrame to get the
//	        11-byte recorder-ready frame.
//	LC  — 240 bits = 24 short Hamming(10,6,3) codewords for the
//	      Link Control word (24-bit source unit ID, 16/24-bit
//	      destination, etc.). LDU1 only.
//	ES  — same shape as LC but carries Encryption Sync (Message
//	      Indicator + Algorithm ID + Key ID). LDU2 only.
//	LSD — 32 bits = 2 cyclic codewords of low-speed data
//	      piggybacked on the voice channel.
//	Status — 24 status symbols, each 2 bits, INTERLEAVED into the
//	         on-air bit stream "after every 70 bits" (TIA Figure
//	         8-3 caption). Used for inbound/outbound channel
//	         signalling at the trunking layer.
//
// What this file ships today: the structural constants, the
// status-symbol deinterleaver (StripStatusSymbols + StatusSymbols),
// and the per-subframe voice-extractor signature stubbed at
// ErrLDUVoicePositionsUnknown. The bit-level interleaving order
// of voice / LC / LSD inside the 1680-bit payload — i.e. the
// answer to "where exactly is voice subframe N within the LDU"
// — is not spelled out in the PDF figures the project has access
// to and is the next gap.
const (
	// LDUTotalBits is the on-air length of an LDU including FS,
	// NID, voice, LC/ES, LSD, and the interleaved status symbols.
	LDUTotalBits = 1728

	// LDUStatusSymbolBits is the total bit count of all
	// interleaved status symbols (24 symbols × 2 bits).
	LDUStatusSymbolBits = 48

	// LDUStatusSymbolCount is the number of 2-bit status symbols
	// interleaved into the LDU stream.
	LDUStatusSymbolCount = 24

	// LDUStatusInterval is the number of payload bits that elapse
	// between consecutive status symbols. Per TIA-102.BAAA Figure
	// 8-3 / 8-4 caption: "2 bits after every 70 bits" — a status
	// symbol follows each run of 70 payload bits, repeated 24
	// times to consume the full 1680-bit payload.
	LDUStatusInterval = 70

	// LDUPayloadBits is the on-air length minus the status
	// symbols — what remains after StripStatusSymbols.
	// 1680 = 1728 − 48.
	LDUPayloadBits = LDUTotalBits - LDUStatusSymbolBits

	// LDUFrameSyncBits is the 48-bit frame-sync pattern at the
	// start of each LDU. Same constant used elsewhere in the
	// package; re-declared here so callers reading ldu.go don't
	// have to chase to the sync package.
	LDUFrameSyncBits = 48

	// LDUNIDBits is the 64-bit Network ID + DUID + BCH parity that
	// follows the frame sync. ParseNID consumes this region.
	LDUNIDBits = 64

	// LDUVoiceSubframeCount is the number of IMBE voice subframes
	// per LDU (one every 20 ms; a single LDU spans 180 ms of
	// audio).
	LDUVoiceSubframeCount = 9

	// LDUVoiceSubframeBits is the per-subframe channel-bit count
	// in IMBE 4400. Matches imbe.ChannelBits.
	LDUVoiceSubframeBits = imbe.ChannelBits

	// LDULCBits is the LDU1 Link Control field width (240 bits =
	// 24 × 10-bit short Hamming codewords). LDU2 carries the
	// Encryption Sync field at the same width.
	LDULCBits = 240
	LDUESBits = 240

	// LDULSDBits is the Low-Speed Data field width (32 bits = 2 ×
	// 16-bit cyclic codewords).
	LDULSDBits = 32
)

// Compile-time check that the structural constants add up to the
// total LDU length stated in TIA-102.BAAA. A future change that
// silently drops or grows a field would fail to compile here.
const _ = uintptr(LDUTotalBits - (LDUFrameSyncBits + LDUNIDBits +
	LDUVoiceSubframeCount*LDUVoiceSubframeBits + LDULCBits +
	LDULSDBits + LDUStatusSymbolBits))

// ErrLDULength is returned by StripStatusSymbols / StatusSymbols
// when the input doesn't have exactly LDUTotalBits bits.
var ErrLDULength = errors.New("p25/phase1: LDU input must be exactly 1728 bits (one bit per byte, 0/1)")

// ErrLDUVoicePositionsUnknown signals that the bit-level
// interleaving of voice / LC / LSD inside the 1680-bit LDU
// payload isn't yet implemented in this package. Callers who
// want recorder-ready IMBE frames from a captured LDU stream
// should wait for the follow-up that adds those bit positions
// (TIA-102.BAAA-A § 8 voice frame layout).
var ErrLDUVoicePositionsUnknown = errors.New(
	"p25/phase1: LDU voice-subframe bit positions not yet implemented (see TIA-102.BAAA-A §8)")

// StripStatusSymbols removes the 24 interleaved status symbols
// (each 2 bits) from a 1728-bit LDU stream and returns the
// resulting 1680-bit payload. The interleaving rule is "2 status
// bits after every 70 payload bits" (TIA-102.BAAA-A Figure 8-3 /
// 8-4 caption), repeated 24 times. The first 70 payload bits
// (positions 0..69 of the input) precede the first status symbol;
// the last 2 input bits (positions 1726..1727) are the 24th
// status symbol's bits.
//
// Bits are stored one per byte (0/1) — same shape the rest of
// the phase1 package and the imbe channel decoder use.
func StripStatusSymbols(ldu []byte) ([]byte, error) {
	if len(ldu) != LDUTotalBits {
		return nil, fmt.Errorf("%w: got %d bits", ErrLDULength, len(ldu))
	}
	payload := make([]byte, 0, LDUPayloadBits)
	stride := LDUStatusInterval + 2
	for i := 0; i < LDUStatusSymbolCount; i++ {
		start := i * stride
		payload = append(payload, ldu[start:start+LDUStatusInterval]...)
	}
	return payload, nil
}

// StatusSymbols extracts the 24 interleaved status symbols from
// a 1728-bit LDU stream. Each symbol is a 2-bit value packed into
// the low 2 bits of a uint8 (high bit first). Use this to inspect
// the trunking-layer signalling carried alongside voice; for
// payload extraction call StripStatusSymbols instead.
func StatusSymbols(ldu []byte) ([LDUStatusSymbolCount]uint8, error) {
	var out [LDUStatusSymbolCount]uint8
	if len(ldu) != LDUTotalBits {
		return out, fmt.Errorf("%w: got %d bits", ErrLDULength, len(ldu))
	}
	stride := LDUStatusInterval + 2
	for i := 0; i < LDUStatusSymbolCount; i++ {
		off := i*stride + LDUStatusInterval
		out[i] = (ldu[off] << 1) | ldu[off+1]
	}
	return out, nil
}

// InjectStatusSymbols is the inverse of StripStatusSymbols: take
// a 1680-bit payload + the 24 status symbols and produce the
// 1728-bit on-air LDU bit stream. Useful for round-trip tests
// and for upstream callers building synthetic LDUs.
func InjectStatusSymbols(payload []byte, status [LDUStatusSymbolCount]uint8) ([]byte, error) {
	if len(payload) != LDUPayloadBits {
		return nil, fmt.Errorf("p25/phase1: payload must be exactly %d bits, got %d",
			LDUPayloadBits, len(payload))
	}
	out := make([]byte, 0, LDUTotalBits)
	for i := 0; i < LDUStatusSymbolCount; i++ {
		out = append(out, payload[i*LDUStatusInterval:(i+1)*LDUStatusInterval]...)
		out = append(out, (status[i]>>1)&1, status[i]&1)
	}
	return out, nil
}

// ExtractVoiceFrames is the not-yet-implemented entry point for
// turning a 1728-bit LDU bit stream into 9 IMBE-frame byte
// buffers ready for recorder.WriteRawFrame. The TIA-102.BAAA-A
// figures embedded in available references show the high-level
// LDU structure but not the precise bit positions of each voice
// subframe inside the 1680-bit payload — that requires the
// section 8 voice-frame layout text.
//
// When the bit positions are sourced (TIA spec direct, or a
// non-copyleft reference; OP25 is GPLv3 and unsuitable as a
// transcription source for this project), the implementation
// shape will be:
//
//	payload, err := StripStatusSymbols(ldu)
//	if err != nil { return nil, err }
//	for i := 0; i < LDUVoiceSubframeCount; i++ {
//	    channel := payload[voiceOffsets[i] : voiceOffsets[i] + LDUVoiceSubframeBits]
//	    frame, _, _ := imbe.DecodeChannelToFrame(channel)
//	    frames[i] = frame
//	}
//
// Until those positions land, callers receive ErrLDUVoicePositionsUnknown.
// The recorder-side wire-up (PR-75) is ready to consume the
// frames once they're available.
func ExtractVoiceFrames(ldu []byte) ([LDUVoiceSubframeCount][]byte, error) {
	var frames [LDUVoiceSubframeCount][]byte
	if len(ldu) != LDUTotalBits {
		return frames, fmt.Errorf("%w: got %d bits", ErrLDULength, len(ldu))
	}
	return frames, ErrLDUVoicePositionsUnknown
}
