package framing

// D-STAR DV-mode header FEC, as real D-STAR radios send it (#1261).
//
// The chain GopherTrunk used to carry here (K=5 punctured code, a PN15
// scrambler, a 22 × 30 interleaver) was invented: it matched no
// reference, and its encoder and decoder agreed only with each other.
// This one is the on-air format, transcribed from the MMDVM firmware
// (DStarTX::txHeader / DStarRX::rxHeader, the code D-STAR hotspots and
// repeaters run) and pinned by literal vectors that firmware's own
// encoder produced (dstar_header_test.go):
//
//	41 header bytes (FLAG1..3, four callsigns, MY2, CRC), each byte
//	  sent LSB first, + 2 zero tail bits          = 330 bits
//	→ K=3 rate-1/2 convolutional code, per input bit d:
//	    g1 = d ⊕ d₋₁ ⊕ d₋₂ (sent first), g0 = d ⊕ d₋₂ = 660 bits
//	→ interleave: the 660 coded bits fill rows of 24 (27 full rows and
//	  a 12-bit 28th), and go on air column by column, so the first 12
//	  columns are 28 bits deep and the last 12 are 27
//	→ scramble: XOR with the x⁷ + x⁴ + 1 sequence from the all-ones
//	  state (0000 1110 1111 0010 …, MMDVM's SCRAMBLE_TABLE_RX, period
//	  127; the same sequence scrambles the voice frames' slow data)
//
// The 660 bits follow the frame sync directly (dstar.FrameSyncBitsSlice
// ends with the four zero bits MMDVM's TX buffer leaves before them).

const (
	// DStarHeaderInfoBits is the size of the structured information
	// field that dstar.ParseHeader consumes (41 bytes * 8).
	DStarHeaderInfoBits = 41 * 8
	// DStarHeaderTailBits is the zero flush tail appended before the
	// convolutional code (K-1 for K=3).
	DStarHeaderTailBits = 2
	// DStarHeaderInputBits is the encoder's input bit count.
	DStarHeaderInputBits = DStarHeaderInfoBits + DStarHeaderTailBits
	// DStarHeaderChannelBits is the on-air bit count: rate 1/2, no
	// puncturing.
	DStarHeaderChannelBits = 2 * DStarHeaderInputBits // 660

	// dstarInterleaveCols is the interleaver's row width.
	dstarInterleaveCols = 24
)

// dstarHeaderPN is the header scrambling sequence, one bit per on-air
// position.
var dstarHeaderPN = DStarPN(DStarHeaderChannelBits)

// dstarInterleave[k] is the coded-bit index that goes on air at
// position k.
var dstarInterleave = func() []int {
	idx := make([]int, 0, DStarHeaderChannelBits)
	rows := (DStarHeaderChannelBits + dstarInterleaveCols - 1) / dstarInterleaveCols
	for c := 0; c < dstarInterleaveCols; c++ {
		for r := 0; r < rows; r++ {
			if i := r*dstarInterleaveCols + c; i < DStarHeaderChannelBits {
				idx = append(idx, i)
			}
		}
	}
	return idx
}()

// DStarPN returns the first n bits of D-STAR's scrambling sequence:
// x⁷ + x⁴ + 1 from the all-ones state. The header FEC section is
// scrambled from its first bit; a DV frame's 24 slow-data bits are
// scrambled with the first 24 (0x70 0x4F 0x93, LSB first).
func DStarPN(n int) []byte {
	out := make([]byte, n)
	s := uint8(0x7F)
	for i := range out {
		b := (s>>6 ^ s>>3) & 1
		out[i] = b
		s = (s<<1 | b) & 0x7F
	}
	return out
}

// EncodeDStarHeaderFEC takes 41 header bytes (the dstar.AssembleHeader
// output, CRC included) and returns the 660 on-air bits that follow the
// frame sync, in transmission order. Returns nil for the wrong length.
func EncodeDStarHeaderFEC(info []byte) []byte {
	if len(info) != 41 {
		return nil
	}
	coded := make([]byte, 0, DStarHeaderChannelBits)
	var d1, d2 byte
	for i := 0; i < DStarHeaderInputBits; i++ {
		var d byte
		if i < DStarHeaderInfoBits {
			d = info[i/8] >> uint(i%8) & 1
		}
		coded = append(coded, d^d1^d2, d^d2)
		d2, d1 = d1, d
	}
	out := make([]byte, DStarHeaderChannelBits)
	for k, i := range dstarInterleave {
		out[k] = coded[i] ^ dstarHeaderPN[k]
	}
	return out
}

// DecodeDStarHeaderFEC recovers the 41 header bytes from the 660 on-air
// bits after the frame sync (descramble, deinterleave, Viterbi). ok is
// false only for the wrong length: whether the result is a header is
// for its CRC to say. Use DecodeDStarHeaderFECErrors for the number of
// channel bits the decoder corrected.
func DecodeDStarHeaderFEC(channel []byte) ([]byte, bool) {
	info, _, ok := DecodeDStarHeaderFECErrors(channel)
	return info, ok
}

// DecodeDStarHeaderFECErrors is DecodeDStarHeaderFEC that also returns
// the Viterbi path metric: the number of on-air bits that differ from
// the re-encoded decision (0 on a clean header).
func DecodeDStarHeaderFECErrors(channel []byte) ([]byte, int, bool) {
	if len(channel) != DStarHeaderChannelBits {
		return nil, 0, false
	}
	coded := make([]byte, DStarHeaderChannelBits)
	for k, i := range dstarInterleave {
		coded[i] = (channel[k] ^ dstarHeaderPN[k]) & 1
	}
	bits, metric := viterbiK3DStar(coded)
	info := make([]byte, 41)
	for i := 0; i < DStarHeaderInfoBits; i++ {
		info[i/8] |= bits[i] << uint(i%8)
	}
	return info, metric, true
}

// viterbiK3DStar is a hard-decision Viterbi decoder for the header's
// 4-state code. coded holds (g1, g0) pairs; the encoder starts and, via
// its zero tail, ends in state 0. State = (d₋₁ << 1) | d₋₂.
func viterbiK3DStar(coded []byte) ([]byte, int) {
	const states = 4
	const inf = 1 << 30
	stages := len(coded) / 2
	metric := [states]int{0, inf, inf, inf}
	// prev[t][s] = the predecessor state of s at stage t; the input bit
	// is s >> 1.
	prev := make([][states]uint8, stages)
	for t := 0; t < stages; t++ {
		r1, r0 := coded[2*t], coded[2*t+1]
		next := [states]int{inf, inf, inf, inf}
		for s := 0; s < states; s++ {
			if metric[s] >= inf {
				continue
			}
			d1, d2 := byte(s>>1), byte(s&1)
			for d := byte(0); d < 2; d++ {
				g1, g0 := d^d1^d2, d^d2
				m := metric[s]
				if g1 != r1 {
					m++
				}
				if g0 != r0 {
					m++
				}
				ns := int(d)<<1 | int(d1)
				if m < next[ns] {
					next[ns] = m
					prev[t][ns] = uint8(s)
				}
			}
		}
		metric = next
	}
	out := make([]byte, stages)
	s := 0
	for t := stages - 1; t >= 0; t-- {
		out[t] = byte(s >> 1)
		s = int(prev[t][s])
	}
	return out, metric[0]
}
