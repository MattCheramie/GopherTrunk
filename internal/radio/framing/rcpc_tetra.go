package framing

// TETRA channel-coding RCPC primitive per ETSI EN 300 395-2 §5.4.3.
//
// Mother code: 16-state, constraint length K=5, rate 1/3
// convolutional code with generator polynomials:
//
//	G_1(D) = 1 + D + D^2 + D^3 + D^4    (0x1F)
//	G_2(D) = 1 + D + D^3 + D^4          (0x1B)
//	G_3(D) = 1 + D^2 + D^4              (0x15)
//
// Different polynomials and rate from the K=5 1/2-rate code in
// viterbi_k5.go (which serves NXDN SACCH / YSF FICH), so this is a
// separate primitive — same 16-state structure, three outputs per
// input instead of two.
//
// Encoder convention: state bits stored as (d1<<3)|(d2<<2)|(d3<<1)|d4
// where d1 is the most-recent bit shifted in before the current
// input, and d4 is the oldest (4 input cycles back). For a register
// of [input, d1, d2, d3, d4], the polynomials evaluate to:
//
//	g1 = input ^ d1 ^ d2 ^ d3 ^ d4
//	g2 = input ^ d1 ^ d3 ^ d4
//	g3 = input ^ d2 ^ d4
//
// Caller flushes the encoder with K-1 = 4 tail bits (zeros) at the
// end of each block so the Viterbi decoder can pick the surviving
// path ending in state 0.
//
// Puncturing patterns and periods for the rates TETRA uses are
// supplied as exported tables; callers pick the matching
// (period, puncture) pair for their channel type.

// EncodeRCPCTetraMother encodes len(input) information bits into
// 3 * len(input) channel bits via the TETRA K=5 1/3-rate mother
// code. Caller is responsible for appending the K-1 = 4 tail bits
// (zeros) to the input before calling so the encoder flushes back
// to state 0.
func EncodeRCPCTetraMother(input []byte) []byte {
	out := make([]byte, 3*len(input))
	var d1, d2, d3, d4 byte
	for i, in := range input {
		bit := in & 1
		out[3*i] = bit ^ d1 ^ d2 ^ d3 ^ d4
		out[3*i+1] = bit ^ d1 ^ d3 ^ d4
		out[3*i+2] = bit ^ d2 ^ d4
		d4 = d3
		d3 = d2
		d2 = d1
		d1 = bit
	}
	return out
}

// DecodeRCPCTetraMother runs hard-decision 16-state Viterbi over
// the K=5 1/3-rate mother code. channel holds 3*stages bits arranged
// as (g1, g2, g3) triples per encoder stage; output is the recovered
// stages-bit input sequence plus the path metric of the surviving
// path (0 = clean channel, positive = corrected bit errors).
//
// Bits at depunctured positions should be set to DepunctureMark so
// the cost accumulator skips them — matches the same convention the
// K=5 1/2-rate primitive in viterbi_k5.go uses.
//
// The Viterbi survivor is forced to the state-0 terminal constraint
// because TETRA appends K-1 = 4 zero tail bits at every encoder
// flush.
func DecodeRCPCTetraMother(channel []byte, stages int) ([]byte, int) {
	const numStates = 16
	const inf = 1 << 30
	pm := make([]int, numStates)
	for i := range pm {
		pm[i] = inf
	}
	pm[0] = 0
	trace := make([][numStates]uint8, stages)

	for s := 0; s < stages; s++ {
		var npm [numStates]int
		for i := range npm {
			npm[i] = inf
		}
		rxG1 := channel[3*s]
		rxG2 := channel[3*s+1]
		rxG3 := channel[3*s+2]
		for cur := 0; cur < numStates; cur++ {
			if pm[cur] >= inf {
				continue
			}
			d1 := (cur >> 3) & 1
			d2 := (cur >> 2) & 1
			d3 := (cur >> 1) & 1
			d4 := cur & 1
			for input := 0; input < 2; input++ {
				g1 := byte(input^d1^d2^d3^d4) & 1
				g2 := byte(input^d1^d3^d4) & 1
				g3 := byte(input^d2^d4) & 1
				cost := pm[cur]
				if rxG1 != DepunctureMark && g1 != rxG1 {
					cost++
				}
				if rxG2 != DepunctureMark && g2 != rxG2 {
					cost++
				}
				if rxG3 != DepunctureMark && g3 != rxG3 {
					cost++
				}
				next := (input << 3) | (d1 << 2) | (d2 << 1) | d3
				if cost < npm[next] {
					npm[next] = cost
					trace[s][next] = uint8((cur << 1) | input)
				}
			}
		}
		copy(pm, npm[:])
	}

	// Encoder is flushed back to state 0 by the tail bits — pick
	// state 0 unconditionally so the survivor honours the
	// terminal-state constraint.
	final := 0
	metric := pm[final]

	out := make([]byte, stages)
	state := final
	for s := stages - 1; s >= 0; s-- {
		entry := trace[s][state]
		out[s] = entry & 1
		state = int(entry >> 1)
	}
	return out, metric
}

// PunctureRCPCTetra selects k3 bits from the mother-code output per
// the TETRA puncturing scheme. puncture[i] holds the 1-indexed
// position within each period window to keep — values in 1..period.
// puncture and period are passed verbatim from the spec (§5.5.2 /
// §5.6.2). The implementation follows the §5.4.3.2 formula:
//
//	k = period * ((j-1) div t) + puncture[((j-1) mod t)]   1-indexed
//	C_3(j) = V(k)                                          j = 1..k3
//
// where t = len(puncture).
func PunctureRCPCTetra(mother []byte, period int, puncture []int, k3 int) []byte {
	t := len(puncture)
	out := make([]byte, k3)
	for j := 1; j <= k3; j++ {
		blockIdx := (j - 1) / t
		offset := j - t*blockIdx // 1..t
		k := period*blockIdx + puncture[offset-1]
		out[j-1] = mother[k-1]
	}
	return out
}

// DepunctureRCPCTetra is the inverse of PunctureRCPCTetra: it
// allocates a motherLen-sized buffer, fills it with DepunctureMark,
// and copies received bits at the puncture-map positions. Pass the
// result straight to DecodeRCPCTetraMother.
func DepunctureRCPCTetra(punctured []byte, period int, puncture []int, motherLen int) []byte {
	t := len(puncture)
	out := make([]byte, motherLen)
	for i := range out {
		out[i] = DepunctureMark
	}
	for j := 1; j <= len(punctured); j++ {
		blockIdx := (j - 1) / t
		offset := j - t*blockIdx
		k := period*blockIdx + puncture[offset-1]
		if k-1 < motherLen {
			out[k-1] = punctured[j-1]
		}
	}
	return out
}

// TETRA RCPC puncturing schemes per ETSI EN 300 395-2.
//
// RCPCTetraPuncture23 is the rate-8/12 (= 2/3) scheme for class-1
// bits in the normal speech traffic channel (§5.5.2.1).
//
// RCPCTetraPuncture818 is the rate-8/18 scheme for class-2 bits in
// the normal speech traffic channel (§5.5.2.2).
//
// RCPCTetraPuncture817 is the rate-8/17 scheme for class-2 bits
// under frame-stealing mode (§5.6.2.1).
//
// Each pattern is 1-indexed exactly as the spec lists it; pair each
// with the matching Period* constant when calling
// PunctureRCPCTetra / DepunctureRCPCTetra.
var (
	RCPCTetraPuncture23  = []int{1, 2, 4}
	RCPCTetraPuncture818 = []int{1, 2, 3, 4, 5, 7, 8, 10, 11}
	RCPCTetraPuncture817 = []int{1, 2, 3, 4, 5, 7, 8, 10, 11, 13, 14, 16, 17, 19, 20, 22, 23}
)

const (
	RCPCTetraPeriod23  = 6
	RCPCTetraPeriod818 = 12
	RCPCTetraPeriod817 = 24
)
