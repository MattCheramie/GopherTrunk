package framing

// TETRA signaling-channel RCPC primitive per ETSI EN 300 392-2 §8.2.3.1.
//
// This is the K=5 R=1/4 16-state convolutional mother code used by
// every TETRA π/4-DQPSK signaling channel (BSCH, SCH/HD, BNCH,
// STCH, SCH/HU, SCH/F) — distinct from the K=5 R=1/3 speech-traffic-
// channel code in `rcpc_tetra.go` (which is from EN 300 395-2 §5.4.3).
// Same 16-state structure, but with four generator polynomials and a
// different puncturing table family.
//
// Mother code generator polynomials:
//
//	G_1(D) = 1 + D + D^4              (0x13)
//	G_2(D) = 1 + D^2 + D^3 + D^4      (0x1D)
//	G_3(D) = 1 + D + D^2 + D^4        (0x17)
//	G_4(D) = 1 + D + D^3 + D^4        (0x1B)
//
// Encoder convention: state bits stored as (d1<<3)|(d2<<2)|(d3<<1)|d4
// where d1 is the most-recent bit shifted in before the current
// input, d4 is the oldest. For a register of [input, d1, d2, d3, d4]
// the polynomials evaluate to:
//
//	g1 = input ^ d1 ^ d4
//	g2 = input ^ d2 ^ d3 ^ d4
//	g3 = input ^ d1 ^ d2 ^ d4
//	g4 = input ^ d1 ^ d3 ^ d4
//
// Callers flush the encoder with K-1 = 4 zero tail bits at the end
// of each block so the Viterbi decoder can pick the surviving path
// ending in state 0.

// EncodeRCPCTetraSigMother encodes len(input) information bits into
// 4 * len(input) channel bits via the TETRA K=5 1/4-rate signaling-
// channel mother code. Caller is responsible for appending the K-1
// = 4 tail bits (zeros) to the input before calling.
func EncodeRCPCTetraSigMother(input []byte) []byte {
	out := make([]byte, 4*len(input))
	var d1, d2, d3, d4 byte
	for i, in := range input {
		bit := in & 1
		out[4*i] = bit ^ d1 ^ d4
		out[4*i+1] = bit ^ d2 ^ d3 ^ d4
		out[4*i+2] = bit ^ d1 ^ d2 ^ d4
		out[4*i+3] = bit ^ d1 ^ d3 ^ d4
		d4 = d3
		d3 = d2
		d2 = d1
		d1 = bit
	}
	return out
}

// DecodeRCPCTetraSigMother runs hard-decision 16-state Viterbi over
// the K=5 1/4-rate mother code. channel holds 4*stages bits arranged
// as (g1, g2, g3, g4) tuples per encoder stage; output is the
// recovered stages-bit input sequence plus the path metric of the
// surviving path (0 = clean channel).
//
// Bits at depunctured positions should be set to DepunctureMark so
// the cost accumulator skips them — same convention as the other
// K=5 primitives in this package.
//
// The Viterbi survivor is forced to the state-0 terminal constraint
// (the tail bits flush the encoder there).
func DecodeRCPCTetraSigMother(channel []byte, stages int) ([]byte, int) {
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
		rxG1 := channel[4*s]
		rxG2 := channel[4*s+1]
		rxG3 := channel[4*s+2]
		rxG4 := channel[4*s+3]
		for cur := 0; cur < numStates; cur++ {
			if pm[cur] >= inf {
				continue
			}
			d1 := (cur >> 3) & 1
			d2 := (cur >> 2) & 1
			d3 := (cur >> 1) & 1
			d4 := cur & 1
			for input := 0; input < 2; input++ {
				g1 := byte(input^d1^d4) & 1
				g2 := byte(input^d2^d3^d4) & 1
				g3 := byte(input^d1^d2^d4) & 1
				g4 := byte(input^d1^d3^d4) & 1
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
				if rxG4 != DepunctureMark && g4 != rxG4 {
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

// PunctureRCPCTetraSig selects k3 bits from the mother-code output
// per the TETRA signaling-channel puncturing scheme — §8.2.3.1.2
// formula:
//
//	k = 8 * ((i-1) div t) + P[((i-1) mod t)]    (1-indexed)
//	b_3(j) = V(k)                                j = 1..k3
//
// where i = j for the simple rates (2/3, 1/3) and i = j +
// (j-1) div 65 / 35 for the 292/432 and 148/432 special rates.
// Callers pass the indexShift slice that maps 1-indexed j to the
// 1-indexed mother-code "i" — pass nil for the identity case
// (i = j). Period is fixed at 8 for all four schemes.
func PunctureRCPCTetraSig(mother []byte, puncture []int, k3 int, indexShift func(j int) int) []byte {
	const period = 8
	t := len(puncture)
	if indexShift == nil {
		indexShift = func(j int) int { return j }
	}
	out := make([]byte, k3)
	for j := 1; j <= k3; j++ {
		i := indexShift(j)
		blockIdx := (i - 1) / t
		offset := i - t*blockIdx // 1..t
		k := period*blockIdx + puncture[offset-1]
		out[j-1] = mother[k-1]
	}
	return out
}

// DepunctureRCPCTetraSig is the inverse of PunctureRCPCTetraSig:
// allocates a motherLen-sized buffer, fills it with DepunctureMark,
// and copies received bits at the puncture-map positions.
func DepunctureRCPCTetraSig(punctured []byte, puncture []int, motherLen int, indexShift func(j int) int) []byte {
	const period = 8
	t := len(puncture)
	if indexShift == nil {
		indexShift = func(j int) int { return j }
	}
	out := make([]byte, motherLen)
	for i := range out {
		out[i] = DepunctureMark
	}
	for j := 1; j <= len(punctured); j++ {
		i := indexShift(j)
		blockIdx := (i - 1) / t
		offset := i - t*blockIdx
		k := period*blockIdx + puncture[offset-1]
		if k-1 < motherLen {
			out[k-1] = punctured[j-1]
		}
	}
	return out
}

// TETRA signaling-channel RCPC puncturing schemes per ETSI
// EN 300 392-2 §8.2.3.1.3 / .4 / .5 / .6. Each pattern is 1-indexed
// exactly as the spec lists it. Period is fixed at 8.
//
// RCPCTetraSigPuncture23 is the rate-2/3 scheme (§8.2.3.1.3) used
// by every standard signaling channel (BSCH, SCH/HD, BNCH, STCH,
// SCH/HU, SCH/F).
//
// RCPCTetraSigPuncture13 is the rate-1/3 scheme (§8.2.3.1.4) used
// by data channels that need stronger protection.
//
// RCPCTetraSigPuncture292_432 / RCPCTetraSigPuncture148_432 are
// the special rate-292/432 and rate-148/432 schemes (§8.2.3.1.5 /
// .6) that pair with the index-shift helpers below.
var (
	RCPCTetraSigPuncture23      = []int{1, 2, 5}
	RCPCTetraSigPuncture13      = []int{1, 2, 3, 5, 6, 7}
	RCPCTetraSigPuncture292_432 = []int{1, 2, 5}
	RCPCTetraSigPuncture148_432 = []int{1, 2, 3, 5, 6, 7}
)

// RCPCTetraSigIndexShift292_432 implements i = j + (j-1) div 65
// for the rate-292/432 scheme per §8.2.3.1.5.
func RCPCTetraSigIndexShift292_432(j int) int {
	return j + (j-1)/65
}

// RCPCTetraSigIndexShift148_432 implements i = j + (j-1) div 35
// for the rate-148/432 scheme per §8.2.3.1.6.
func RCPCTetraSigIndexShift148_432(j int) int {
	return j + (j-1)/35
}
