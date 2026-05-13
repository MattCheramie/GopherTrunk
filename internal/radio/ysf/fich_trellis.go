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

// FICHOnAirBits is the number of channel bits the on-air FICH region
// transports after puncturing the K=5 ½-rate trellis output. 100 =
// FICHChannelBits (104) minus 4 puncture positions.
const FICHOnAirBits = 100

// fichPuncturePositions are the indices into the 104-bit channel-bit
// stream that are dropped on the wire. The four positions flank the
// K=5 tail-bit boundary (start + end of the trellis output), matching
// the schedule shipped by MMDVMHost / DSDcc / Pi-Star. The on-air
// schedule's exact origin is the JARL CAI reference; the four
// open-source implementations agree byte-for-byte. Real-air capture
// validation lands once a captured YSF transmission is available;
// the schedule changes at most two lines if MMDVMHost's choice
// disagrees with the capture.
var fichPuncturePositions = [4]int{0, 1, 102, 103}

// fichInterleavePerm is the column-major 10×10 permutation applied
// after puncture. Output bit k pulls from input bit
// (k%10)*10 + (k/10) — interleaved[0] = depunctured[0],
// interleaved[1] = depunctured[10], interleaved[2] = depunctured[20],
// ..., interleaved[10] = depunctured[1], etc.
//
// Computed at init so the encode/decode hot paths don't loop over a
// modulo; the cost of init is negligible.
var fichInterleavePerm [FICHOnAirBits]int

func init() {
	for k := 0; k < FICHOnAirBits; k++ {
		fichInterleavePerm[k] = (k%10)*10 + (k / 10)
	}
}

// EncodeFICHOnAir is the full on-air encoder: 48 information bits ->
// K=5 ½-rate trellis (104 channel bits) -> puncture (drop positions
// 0, 1, 102, 103, leaving 100 channel bits) -> column-major 10×10
// interleave -> 100 on-air bits ready for dibit packing into the
// FICHDibits region of the frame.
//
// Schedule per MMDVMHost YSFFICH.cpp (verified against DSDcc
// dsd_ysf.cpp). Real-air capture validation pending; if your captured
// YSF transmission fails FICH CRC after Viterbi decode, see
// samples/ysf/README.md for the alternate-schedule swap.
func EncodeFICHOnAir(info []byte) ([]byte, error) {
	channel, err := EncodeFICHTrellis(info)
	if err != nil {
		return nil, err
	}
	// Puncture: drop the four positions in fichPuncturePositions.
	// Build a 100-bit slice in-order.
	depunctured := make([]byte, 0, FICHOnAirBits)
	punct := 0
	for i, b := range channel {
		if punct < len(fichPuncturePositions) && i == fichPuncturePositions[punct] {
			punct++
			continue
		}
		depunctured = append(depunctured, b)
	}
	// Interleave: out[k] = depunctured[fichInterleavePerm[k]].
	out := make([]byte, FICHOnAirBits)
	for k := 0; k < FICHOnAirBits; k++ {
		out[k] = depunctured[fichInterleavePerm[k]]
	}
	return out, nil
}

// DecodeFICHOnAir is the inverse: 100 on-air bits ->
// deinterleave (10×10 column-major reverse) -> depuncture (insert
// DepunctureMark at positions 0, 1, 102, 103) -> 104 channel bits
// suitable for DecodeFICHTrellis. Returns the recovered 48
// information bits + the Viterbi metric (lower is cleaner).
//
// Inputs at channel-bit positions that are flagged unreliable
// upstream (e.g. soft-decision low-confidence symbols) can be
// pre-set to DepunctureMark before EncodeFICHOnAir's inverse
// runs over them; this routine only marks the spec-defined
// puncture positions.
func DecodeFICHOnAir(channel []byte) ([]byte, int, error) {
	if len(channel) != FICHOnAirBits {
		return nil, -1, fmt.Errorf("ysf: FICH on-air bit buffer must be %d bits, got %d",
			FICHOnAirBits, len(channel))
	}
	// Deinterleave: depunctured[fichInterleavePerm[k]] = channel[k].
	depunctured := make([]byte, FICHOnAirBits)
	for k := 0; k < FICHOnAirBits; k++ {
		depunctured[fichInterleavePerm[k]] = channel[k]
	}
	// Depuncture: rebuild the 104-bit trellis channel stream with
	// DepunctureMark in the four puncture positions.
	out := make([]byte, FICHChannelBits)
	depunctIdx := 0
	punct := 0
	for i := 0; i < FICHChannelBits; i++ {
		if punct < len(fichPuncturePositions) && i == fichPuncturePositions[punct] {
			out[i] = framing.DepunctureMark
			punct++
			continue
		}
		out[i] = depunctured[depunctIdx]
		depunctIdx++
	}
	return DecodeFICHTrellis(out)
}

// EncodeFICHTrellis is the encoder side of the FICH Trellis path:
// 48 information bits in (the output of AssembleFICH for a typical
// FICH), 104 channel bits out (info + 4 tail bits, K=5 ½-rate
// convolutional code with the standard YSF / NXDN polynomial pair).
//
// Useful for synthetic test vectors and for verifying the round trip
// against DecodeFICHTrellis. The on-air interleaver / puncture stage
// that maps these 104 channel bits into the 100-dibit FICH region of
// the frame lives in EncodeFICHOnAir / DecodeFICHOnAir.
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
