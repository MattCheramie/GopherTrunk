package imbe

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
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
//     channel bits of u_1..u_6 to whiten the spectrum (§7.4).
//
//  3. A 144-bit interleaver permutation that scatters adjacent
//     codeword bits across the frame so a localised burst error
//     spreads across vectors (§7.5).
//
// This file ships layer 1 — the per-vector FEC encode/decode plus
// the bit-position constants. Layers 2 and 3 are a self-contained
// follow-up; the public DecodeChannel / EncodeChannel functions
// here operate on already-deinterleaved + already-descrambled
// channel bits so the per-vector FEC can be reviewed in isolation.

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

	// Golay(23,12) vectors u_0..u_3 — 12 info bits each.
	for vi, off := range []int{u0Offset, u1Offset, u2Offset, u3Offset} {
		cw := bitsToUint32(channel[off : off+23])
		data, errs := framing.GolayDecode23_12(cw)
		if errs < 0 {
			uncorrectable = true
		} else {
			totalErrs += errs
		}
		uint16ToBits(data, 12, info[vi*12:vi*12+12])
	}

	// Hamming(15,11) vectors u_4..u_6 — 11 info bits each, packed
	// after the 48 Golay info bits.
	const hOffset = 48
	for vi, off := range []int{u4Offset, u5Offset, u6Offset} {
		cw := uint16(bitsToUint32(channel[off : off+15]))
		data, errs := framing.HammingDecode15_11(cw)
		if errs < 0 {
			uncorrectable = true
		} else {
			totalErrs += errs
		}
		uint16ToBits(data, 11, info[hOffset+vi*11:hOffset+vi*11+11])
	}

	// u_7 — 7 info bits, no FEC, copied through.
	copy(info[hOffset+33:hOffset+33+7], channel[u7Offset:u7Offset+7])

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
		data := uint16(bitsToUint32(info[vi*12 : vi*12+12]))
		cw := framing.GolayEncode23_12(data)
		uint32ToBits(cw, 23, channel[off:off+23])
	}

	const hOffset = 48
	for vi, off := range []int{u4Offset, u5Offset, u6Offset} {
		data := bitsToUint32(info[hOffset+vi*11 : hOffset+vi*11+11])
		cw := framing.HammingEncode15_11(uint16(data))
		uint32ToBits(uint32(cw), 15, channel[off:off+15])
	}

	copy(channel[u7Offset:u7Offset+7], info[hOffset+33:hOffset+33+7])

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
// "144 channel bits, post-deinterleave" → "FrameBytes-byte
// recorder-ready IMBE frame". It runs the §7.4 PRBS descrambler,
// then the per-vector Golay+Hamming FEC inverse, then packs the
// 88 recovered information bits MSB-first.
//
// Used by upstream protocol decoders (P25 Phase 1 LDU extraction,
// future) that hand off post-deinterleave channel bursts: each
// LDU carries 9 IMBE voice frames, each 144 bits — call this
// helper for each slot and forward the resulting frame to
// voice.Recorder.WriteRawFrame.
//
// Returns the total bit-errors corrected across all FEC vectors.
// Uncorrectable codewords surface as ErrUncorrectable from
// DecodeChannel; the partially-recovered frame is still returned
// so callers can log + frame-repeat upstream.
func DecodeChannelToFrame(channel []byte) (frame []byte, errs int, err error) {
	descrambled, err := Descramble(channel)
	if err != nil {
		return nil, 0, err
	}
	info, errs, decErr := DecodeChannel(descrambled)
	frame, packErr := PackInfoBitsToFrame(info)
	if decErr != nil {
		return frame, errs, decErr
	}
	return frame, errs, packErr
}

// bitsToUint32 packs up to 32 bits (MSB-first) from src into the
// low bits of a uint32. Used for codewords of any width that fits.
func bitsToUint32(src []byte) uint32 {
	var out uint32
	for _, b := range src {
		out = (out << 1) | uint32(b&1)
	}
	return out
}

// uint32ToBits writes the low n bits of v to dst MSB-first. dst
// must already be the right length.
func uint32ToBits(v uint32, n int, dst []byte) {
	for i := 0; i < n; i++ {
		dst[i] = byte((v >> uint(n-1-i)) & 1)
	}
}

// uint16ToBits is shorthand for uint32ToBits when callers already
// hold a uint16 (Hamming-decoded data). Keeps the call site
// clean without an extra cast at the use site.
func uint16ToBits(v uint16, n int, dst []byte) {
	uint32ToBits(uint32(v), n, dst)
}
