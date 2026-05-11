package ysf

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// FICHTailBits is the number of zero tail bits the YSF FICH encoder
// appends so the K=5 ½-rate Viterbi can fall back on the terminal-
// state constraint at decode time. K-1 = 4.
const FICHTailBits = 4

// FICHInfoBits is the number of FICH information bits the Trellis
// stage protects (32 info + 16 CRC = 48 bits ready for ParseFICH).
const FICHInfoBits = 48

// FICHChannelBits is the number of channel bits the K=5 ½-rate
// encoder produces for one FICH block: 2 * (FICHInfoBits + FICHTailBits).
// That's 104 hard channel bits the on-air interleaver permutes
// across the FICHDibits region of the frame.
const FICHChannelBits = 2 * (FICHInfoBits + FICHTailBits)

// ErrFICHTrellisLength is returned by DecodeFICHTrellis when the
// supplied channel-bit buffer isn't FICHChannelBits long.
var ErrFICHTrellisLength = errors.New("ysf: FICH channel-bit buffer length mismatch")

// EncodeFICHTrellis is the encoder side of the FICH Trellis path:
// 48 information bits in (the output of AssembleFICH for a typical
// FICH), 104 channel bits out (info + 4 tail bits, K=5 ½-rate
// convolutional code with the standard YSF / NXDN polynomial pair).
//
// Useful for synthetic test vectors and for verifying the round trip
// against DecodeFICHTrellis. The on-air interleaver / puncture
// stage that maps these 104 channel bits into the 100-dibit FICH
// region of the frame lives in a follow-up — every published YSF
// reference (DSDcc, MMDVMHost) uses a slightly different table and
// we want to validate against a captured stream before pinning one
// here.
func EncodeFICHTrellis(info []byte) ([]byte, error) {
	if len(info) != FICHInfoBits {
		return nil, fmt.Errorf("ysf: FICH info must be %d bits, got %d", FICHInfoBits, len(info))
	}
	input := make([]byte, FICHInfoBits+FICHTailBits)
	copy(input, info)
	// Tail bits stay 0 — flush the encoder back to state 0.
	return framing.EncodeK5(input), nil
}

// DecodeFICHTrellis runs the K=5 ½-rate Viterbi over the supplied
// hard channel bits and returns the recovered 48 FICH information
// bits + the path metric (0 means clean — every survivor branch
// matched the channel observation; higher means more bit-flips were
// repaired). The recovered info-bit slice is ready to be packed into
// 6 octets and fed to ParseFICH for CRC verification.
//
// Inputs at puncture positions (when an interleaver+puncture stage
// is layered above this primitive) should be set to
// framing.DepunctureMark so the metric accumulator skips them.
func DecodeFICHTrellis(channel []byte) ([]byte, int, error) {
	if len(channel) != FICHChannelBits {
		return nil, -1, fmt.Errorf("%w: got %d, want %d", ErrFICHTrellisLength, len(channel), FICHChannelBits)
	}
	const stages = FICHInfoBits + FICHTailBits
	bits, metric := framing.ViterbiK5(channel, stages)
	return bits[:FICHInfoBits], metric, nil
}

// PackBits packs a length-multiple-of-8 bit slice (each entry 0/1)
// MSB-first into octets. Used to bridge the Viterbi output into the
// 6-octet shape ParseFICH expects.
func PackBits(bits []byte) []byte {
	out := make([]byte, len(bits)/8)
	for i := range out {
		var b byte
		for k := 0; k < 8; k++ {
			b = (b << 1) | (bits[i*8+k] & 1)
		}
		out[i] = b
	}
	return out
}

// UnpackBits is the inverse of PackBits: 1 octet → 8 MSB-first bits.
func UnpackBits(octets []byte) []byte {
	out := make([]byte, len(octets)*8)
	for i, o := range octets {
		for k := 0; k < 8; k++ {
			out[i*8+k] = (o >> (7 - k)) & 1
		}
	}
	return out
}
