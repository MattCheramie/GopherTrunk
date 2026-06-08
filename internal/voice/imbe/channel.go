package imbe

import (
	"errors"
	"fmt"
)

// IMBE 4400 channel coding (TIA-102.BABA §7) maps 88 information
// bits to 144 transmitted channel bits via three layers:
//
//  1. Per-vector FEC over eight u_n vectors:
//
//     vector  channel bits  info bits  FEC
//     u_0     23            12         Golay(23,12,7)
//     u_1     23            12         Golay(23,12,7)
//     u_2     23            12         Golay(23,12,7)
//     u_3     23            12         Golay(23,12,7)
//     u_4     15            11         Hamming(15,11,3)
//     u_5     15            11         Hamming(15,11,3)
//     u_6     15            11         Hamming(15,11,3)
//     u_7      7             7         (no FEC — least-sensitive bits)
//                                      ────
//     total  144            88
//
//  2. A pseudo-random scrambler keyed off u_0 — XOR'd onto the
//     channel bits of u_1..u_6 to whiten the spectrum (§7.4). See
//     scrambler.go.
//
//  3. A 144-bit interleaver permutation that scatters adjacent
//     codeword bits across the frame so a localised burst error
//     spreads across vectors (§7.5). See interleave.go.
//
// This file ships layer 1 — the per-vector FEC encode/decode plus
// the bit-position constants. DecodeChannel / EncodeChannel here
// operate on channel bits in vector order — i.e. already
// deinterleaved (§7.5) and descrambled (§7.4) — so the per-vector
// FEC can be reviewed in isolation. The DecodeChannelToFrame /
// EncodeFrameToChannel helpers wrap those two layers around the
// per-vector FEC to consume / produce raw on-air bursts.

// Per-vector geometry. Offsets are measured in bits from the start
// of the (already-deinterleaved + already-descrambled) 144-bit
// channel buffer.
const (
	ChannelBits   = 144
	InfoBitsTotal = InfoBits // re-exported so callers don't have to recall the name

	u0Offset, u0Bits, u0InfoBits = 0, 23, 12
	u1Offset, u1Bits, u1InfoBits = 23, 23, 12
	u2Offset, u2Bits, u2InfoBits = 46, 23, 12
	u3Offset, u3Bits, u3InfoBits = 69, 23, 12
	u4Offset, u4Bits, u4InfoBits = 92, 15, 11
	u5Offset, u5Bits, u5InfoBits = 107, 15, 11
	u6Offset, u6Bits, u6InfoBits = 122, 15, 11
	u7Offset, u7Bits, u7InfoBits = 137, 7, 7
)

// ErrChannelLength is returned by DecodeChannel / EncodeChannel
// when the supplied buffer isn't the expected channel/info length.
var ErrChannelLength = errors.New("imbe: channel buffer must be 144 bits / info buffer must be 88 bits")

// ErrUncorrectable is returned by DecodeChannel when one of the
// FEC-protected vectors exceeds its correction radius. The
// partially-recovered info bits are still returned so callers can
// log + frame-repeat upstream.
var ErrUncorrectable = errors.New("imbe: per-vector FEC uncorrectable")

// DecodeChannel runs the per-vector FEC inverse over 144 channel
// bits (post-deinterleave, post-descramble) and returns 88
// recovered information bits, the total number of bit-errors
// corrected across all vectors, and an error when any vector's
// codeword exceeds its correction radius. Bits are stored one per
// byte (0/1) in MSB-first order matching what the deinterleaver
// will emit.
func DecodeChannel(channel []byte) ([]byte, int, error) {
	if len(channel) != ChannelBits {
		return nil, -1, fmt.Errorf("%w: got %d channel bits", ErrChannelLength, len(channel))
	}
	info := make([]byte, InfoBits)
	totalErrs := 0
	uncorrectable := false

	// Golay(23,12) vectors u_0..u_3 — 12 info bits each, data in the
	// high columns (col 22..11), MSB-first.
	for vi, off := range []int{u0Offset, u1Offset, u2Offset, u3Offset} {
		data, errs := golay23Decode(vectorToBlock(channel[off : off+23]))
		if errs < 0 {
			uncorrectable = true
		} else {
			totalErrs += errs
		}
		dataToInfo(data, 12, info[vi*12:vi*12+12])
	}

	// Hamming(15,11) vectors u_4..u_6 — 11 info bits each, packed
	// after the 48 Golay info bits.
	const hOffset = 48
	for vi, off := range []int{u4Offset, u5Offset, u6Offset} {
		data, errs := hamming1511Decode(uint16(vectorToBlock(channel[off : off+15])))
		if errs < 0 {
			uncorrectable = true
		} else {
			totalErrs += errs
		}
		dataToInfo(data, 11, info[hOffset+vi*11:hOffset+vi*11+11])
	}

	// u_7 — 7 info bits, no FEC. The on-air column order is the
	// reverse of the info-bit order: info[0] of this group is the
	// highest column (col 6).
	for i := 0; i < u7InfoBits; i++ {
		info[hOffset+33+i] = channel[u7Offset+(u7Bits-1-i)] & 1
	}

	if uncorrectable {
		return info, totalErrs, ErrUncorrectable
	}
	return info, totalErrs, nil
}

// EncodeChannel is the inverse: 88 info bits → 144 channel bits
// (still pre-scramble + pre-interleave). The encode path matches
// the decode path bit-for-bit so the future scrambler + interleaver
// land as a thin wrapper that doesn't need to touch the FEC math.
func EncodeChannel(info []byte) ([]byte, error) {
	if len(info) != InfoBits {
		return nil, fmt.Errorf("%w: got %d info bits", ErrChannelLength, len(info))
	}
	channel := make([]byte, ChannelBits)

	for vi, off := range []int{u0Offset, u1Offset, u2Offset, u3Offset} {
		cw := golay23Encode(infoToData(info[vi*12:vi*12+12], 12))
		blockToVector(cw, channel[off:off+23])
	}

	const hOffset = 48
	for vi, off := range []int{u4Offset, u5Offset, u6Offset} {
		cw := hamming1511Encode(infoToData(info[hOffset+vi*11:hOffset+vi*11+11], 11))
		blockToVector(uint32(cw), channel[off:off+15])
	}

	for i := 0; i < u7InfoBits; i++ {
		channel[u7Offset+(u7Bits-1-i)] = info[hOffset+33+i] & 1
	}

	return channel, nil
}

// PackInfoBitsToFrame packs InfoBits (88) information bits — one
// bit per byte (0/1) — into a FrameBytes (11)-byte buffer
// MSB-first, the wire shape Decoder.Decode and
// voice.DecodeStream consume. Inverse of the unpacking those
// functions perform internally.
//
// Used by upstream protocol decoders (P25 Phase 1 LDU extractor
// and friends) to produce a recorder-ready frame after running
// the per-vector channel-coding inverse.
func PackInfoBitsToFrame(info []byte) ([]byte, error) {
	if len(info) != InfoBits {
		return nil, fmt.Errorf("%w: got %d info bits, want %d",
			ErrChannelLength, len(info), InfoBits)
	}
	frame := make([]byte, FrameBytes)
	for i, b := range info {
		if b&1 != 0 {
			frame[i/8] |= 1 << (7 - uint(i)%8)
		}
	}
	return frame, nil
}

// DecodeChannelToFrame is the convenience pipeline that bridges
// "144 raw on-air channel bits" → "FrameBytes-byte recorder-ready
// IMBE frame". It runs the §7.5 deinterleaver, then the §7.4 PRBS
// descrambler (whose u_0 seed is only valid once the bits are in
// vector order), then the per-vector Golay+Hamming FEC inverse,
// then packs the 88 recovered information bits MSB-first.
//
// Used by upstream protocol decoders (P25 Phase 1 LDU extraction)
// that hand off raw on-air channel bursts: each LDU carries 9 IMBE
// voice frames, each 144 bits — call this helper for each slot and
// forward the resulting frame to voice.Recorder.WriteRawFrame.
//
// Returns the total bit-errors corrected across all FEC vectors.
// Uncorrectable codewords surface as ErrUncorrectable from
// DecodeChannel; the partially-recovered frame is still returned
// so callers can log + frame-repeat upstream.
func DecodeChannelToFrame(channel []byte) (frame []byte, errs int, err error) {
	deinterleaved, err := Deinterleave(channel)
	if err != nil {
		return nil, 0, err
	}
	// Correct u_0's Golay before deriving the descrambler seed, so a
	// channel-bit error in u_0 doesn't desync the entire u_1..u_6
	// descramble. mbelib runs eccImbe7200x4400C0 before
	// mbe_demodulateImbe7200x4400Data for the same reason. DecodeChannel
	// re-decodes u_0 from the unmodified bits afterwards, so its error
	// telemetry still reflects the real u_0 error count.
	u0Data, _ := golay23Decode(vectorToBlock(deinterleaved[u0Offset : u0Offset+u0Bits]))
	descrambleWithSeed(deinterleaved, u0Data<<4)
	info, errs, decErr := DecodeChannel(deinterleaved)
	frame, packErr := PackInfoBitsToFrame(info)
	if decErr != nil {
		return frame, errs, decErr
	}
	return frame, errs, packErr
}

// vectorToBlock packs a vector's channel bits into an integer with
// column c at bit c (LSB-first by column). The IMBE per-vector FEC
// (p25fec.go) addresses codewords this way: column index == codeword
// bit index, data in the high columns.
func vectorToBlock(v []byte) uint32 {
	var b uint32
	for c, bit := range v {
		b |= uint32(bit&1) << uint(c)
	}
	return b
}

// blockToVector writes an integer back into a vector's channel bits,
// column c taking bit c. Inverse of vectorToBlock.
func blockToVector(b uint32, dst []byte) {
	for c := range dst {
		dst[c] = byte((b >> uint(c)) & 1)
	}
}

// dataToInfo writes an n-bit data value MSB-first into dst (dst[0] is
// the most-significant data bit).
func dataToInfo(data uint16, n int, dst []byte) {
	for i := 0; i < n; i++ {
		dst[i] = byte((data >> uint(n-1-i)) & 1)
	}
}

// infoToData reads n MSB-first info bits into a data value.
func infoToData(src []byte, n int) uint16 {
	var d uint16
	for i := 0; i < n; i++ {
		d = d<<1 | uint16(src[i]&1)
	}
	return d
}
