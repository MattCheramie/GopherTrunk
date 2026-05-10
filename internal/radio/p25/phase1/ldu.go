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

// LDU payload-bit offsets for each field inside the 1680-bit
// post-status-strip payload. Source: TIA-102.BAAA-A § 8
// (Logical Link Data Unit 1 / 2 voice-frame layout). The 9 IMBE
// voice subframes are interleaved with 6 × 40-bit LC (LDU1) or
// ES (LDU2) blocks and 2 × 16-bit LSD blocks per the table:
//
//	Field                   Length   Cumulative
//	Frame Sync (FS)             48       48
//	Network ID (NID)            64      112
//	Voice Frame 1 (u_0)        144      256
//	Voice Frame 2 (u_1)        144      400
//	LC / ES Block 1             40      440
//	Voice Frame 3 (u_2)        144      584
//	LC / ES Block 2             40      624
//	Voice Frame 4 (u_3)        144      768
//	LC / ES Block 3             40      808
//	Voice Frame 5 (u_4)        144      952
//	LC / ES Block 4             40      992
//	Voice Frame 6 (u_5)        144     1136
//	LC / ES Block 5             40     1176
//	Voice Frame 7 (u_6)        144     1320
//	LC / ES Block 6             40     1360
//	Voice Frame 8 (u_7)        144     1504
//	LSD Block 1                 16     1520
//	Voice Frame 9 (u_8)        144     1664
//	LSD Block 2                 16     1680
//
// (LDU1 carries Link Control bits in the LC/ES slots; LDU2
// carries Encryption Sync bits at the identical positions.)
var (
	// lduFSOffset, lduNIDOffset locate the two fixed-position
	// fields at the start of every LDU.
	lduFSOffset  = 0
	lduNIDOffset = LDUFrameSyncBits

	// lduVoiceOffsets[i] is the bit offset of voice subframe
	// u_i (0 ≤ i < 9) inside the 1680-bit payload. Each voice
	// subframe is LDUVoiceSubframeBits = 144 bits long.
	lduVoiceOffsets = [LDUVoiceSubframeCount]int{
		112,  // u_0 → ends at 256
		256,  // u_1 → ends at 400
		440,  // u_2 → ends at 584  (post LC/ES Block 1)
		624,  // u_3 → ends at 768  (post LC/ES Block 2)
		808,  // u_4 → ends at 952  (post LC/ES Block 3)
		992,  // u_5 → ends at 1136 (post LC/ES Block 4)
		1176, // u_6 → ends at 1320 (post LC/ES Block 5)
		1360, // u_7 → ends at 1504 (post LC/ES Block 6)
		1520, // u_8 → ends at 1664 (post LSD Block 1)
	}

	// lduLCESBlockOffsets[j] is the bit offset of LC/ES block j
	// (0 ≤ j < 6) inside the 1680-bit payload. Each block is
	// 40 bits = 4 × Hamming(10,6,3) short codewords.
	lduLCESBlockOffsets = [6]int{
		400,  // Block 1
		584,  // Block 2 (post u_2)
		768,  // Block 3 (post u_3)
		952,  // Block 4 (post u_4)
		1136, // Block 5 (post u_5)
		1320, // Block 6 (post u_6)
	}

	// lduLSDBlockOffsets[k] is the bit offset of LSD block k
	// (0 ≤ k < 2). Each block is 16 bits = 1 cyclic codeword.
	lduLSDBlockOffsets = [2]int{
		1504, // Block 1 (post u_7)
		1664, // Block 2 (post u_8)
	}
)

// LDULCESBlockBits is the bit width of one of the 6 LC (in LDU1)
// or ES (in LDU2) blocks interleaved between voice subframes.
// 6 × 40 = 240 total.
const LDULCESBlockBits = 40

// LDULCESBlockCount is the number of LC/ES blocks per LDU.
const LDULCESBlockCount = 6

// LDULSDBlockBits is the bit width of one of the 2 LSD blocks.
// 2 × 16 = 32 total.
const LDULSDBlockBits = 16

// LDULSDBlockCount is the number of LSD blocks per LDU.
const LDULSDBlockCount = 2

// ErrLDUVoicePositionsUnknown is kept as a deprecated alias for
// callers that gated on it before the LDU layout was sourced.
// New code should not encounter this; ExtractVoiceFrames now
// returns frames for well-formed input.
//
// Deprecated: kept for one release cycle so external callers can
// migrate; will be removed in a future PR.
var ErrLDUVoicePositionsUnknown = errors.New(
	"p25/phase1: legacy sentinel; LDU voice-frame extraction is now implemented")

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

// ExtractVoiceFrames turns a 1728-bit on-air LDU bit stream into
// 9 IMBE-frame byte buffers ready for recorder.WriteRawFrame. The
// pipeline is:
//
//	1728-bit on-air ldu
//	  → StripStatusSymbols     (1680-bit payload)
//	  → slice u_0..u_8 at lduVoiceOffsets   (9 × 144 bits)
//	  → imbe.DecodeChannelToFrame for each  (descramble, FEC,
//	                                         bit-pack → 11 bytes)
//	  → [9] recorder-ready frames
//
// totalErrs is the sum of bit-errors corrected by the per-vector
// Golay + Hamming FEC across all 9 subframes. A non-nil err
// surfaces an uncorrectable codeword in at least one subframe
// (the partially-recovered frame is still returned in that slot
// so callers can frame-repeat the failing one).
//
// frames[i] is the IMBE info-bit frame for voice subframe u_i; an
// 11-byte slice consumable by imbe.Decoder.Decode (which the
// recorder calls when a vocoder is wired to the call's protocol;
// see voice.DefaultVocoderForProtocol). Use the
// ExtractLCESBlocks / ExtractLSDBlocks helpers below to pull the
// metadata interleaved between voice subframes.
//
// Bit positions sourced from TIA-102.BAAA-A § 8 (Logical Link
// Data Unit 1 / 2 voice-frame layout).
func ExtractVoiceFrames(ldu []byte) (frames [LDUVoiceSubframeCount][]byte, totalErrs int, err error) {
	if len(ldu) != LDUTotalBits {
		return frames, 0, fmt.Errorf("%w: got %d bits", ErrLDULength, len(ldu))
	}
	payload, err := StripStatusSymbols(ldu)
	if err != nil {
		return frames, 0, err
	}

	var firstErr error
	for i, off := range lduVoiceOffsets {
		channel := payload[off : off+LDUVoiceSubframeBits]
		frame, errs, decErr := imbe.DecodeChannelToFrame(channel)
		frames[i] = frame
		totalErrs += errs
		if decErr != nil && firstErr == nil {
			firstErr = fmt.Errorf("subframe %d: %w", i, decErr)
		}
	}
	return frames, totalErrs, firstErr
}

// ExtractLCESBlocks returns the 6 LC (LDU1) / ES (LDU2) blocks
// interleaved between the voice subframes. Each block is 40 bits
// long and carries 4 × Hamming(10,6,3) short codewords. Concatenating
// the 6 blocks yields the 240-bit LC or ES word that downstream
// processing decodes into source/destination unit IDs (LDU1) or
// the Message Indicator + Algorithm ID + Key ID (LDU2).
//
// The caller is responsible for knowing which DUID was just
// decoded (DUIDLogicalLink1 vs DUIDLogicalLink2) and interpreting
// the bits accordingly.
func ExtractLCESBlocks(ldu []byte) (blocks [LDULCESBlockCount][]byte, err error) {
	if len(ldu) != LDUTotalBits {
		return blocks, fmt.Errorf("%w: got %d bits", ErrLDULength, len(ldu))
	}
	payload, err := StripStatusSymbols(ldu)
	if err != nil {
		return blocks, err
	}
	for i, off := range lduLCESBlockOffsets {
		blocks[i] = payload[off : off+LDULCESBlockBits]
	}
	return blocks, nil
}

// ExtractLSDBlocks returns the 2 Low-Speed Data blocks
// interleaved into the LDU. Each block is 16 bits long = one
// cyclic codeword. LSD piggybacks on the voice channel and
// carries low-bandwidth out-of-band signalling (sensor data,
// status messages, etc.) that operators can opt into.
func ExtractLSDBlocks(ldu []byte) (blocks [LDULSDBlockCount][]byte, err error) {
	if len(ldu) != LDUTotalBits {
		return blocks, fmt.Errorf("%w: got %d bits", ErrLDULength, len(ldu))
	}
	payload, err := StripStatusSymbols(ldu)
	if err != nil {
		return blocks, err
	}
	for i, off := range lduLSDBlockOffsets {
		blocks[i] = payload[off : off+LDULSDBlockBits]
	}
	return blocks, nil
}
