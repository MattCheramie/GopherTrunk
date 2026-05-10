package framing

// K=5 ½-rate convolutional code shared by NXDN SACCH, YSF FICH, and
// other open-spec digital-voice systems built on the same de-facto
// "16-state" code. The polynomial pair is fixed:
//
//	g1 = 1 + x^3 + x^4   (= 0x19, octal 31)
//	g2 = 1 + x + x^2 + x^4 (= 0x17, octal 27)
//
// Which is the constraint-length-5, rate-½ code OP25 / DSDcc /
// MMDVMHost all use under different protocol-specific framing.
// The systems that share this code differ in their puncturing,
// interleaving, CRC/RS trailers, and tail-bit count — keep those
// pieces in the protocol package and call this primitive for the
// inner Viterbi search.
//
// Encoder convention: state bits stored as (d1<<3)|(d2<<2)|(d3<<1)|d4
// where d1 is the most-recent bit shifted in (after the current
// input) and d4 is the oldest. For a register of [input, d1, d2, d3,
// d4], the polynomials evaluate to:
//
//	g1 = input ^ d3 ^ d4
//	g2 = input ^ d1 ^ d2 ^ d4

// DepunctureMark is the sentinel byte callers store at puncture
// positions in the depunctured channel-bit stream. ViterbiK5
// recognises any byte equal to this constant as "no information at
// this slot — skip the cost contribution." Picked at 0xFF so it
// can't collide with real channel bits (which are 0 or 1).
const DepunctureMark byte = 0xFF

// ViterbiK5 runs hard-decision Viterbi over the K=5 ½-rate code.
// Input is `2*stages` channel bits arranged as (g1, g2) pairs;
// output is `stages` recovered input bits and the path metric of the
// surviving path (sum of dibit-distance penalties; 0 = clean
// channel). Tail bits in the encoder flush state back to 0, so the
// decoder picks the survivor ending in state 0.
//
// Channel bits at puncture positions should be set to DepunctureMark
// so the cost accumulator skips them; non-punctured systems can
// ignore the marker — pass real 0/1 bits and DepunctureMark never
// matches.
func ViterbiK5(channel []byte, stages int) ([]byte, int) {
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
		rxG1 := channel[2*s]
		rxG2 := channel[2*s+1]
		for cur := 0; cur < numStates; cur++ {
			if pm[cur] >= inf {
				continue
			}
			d1 := (cur >> 3) & 1
			d2 := (cur >> 2) & 1
			d3 := (cur >> 1) & 1
			d4 := cur & 1
			for input := 0; input < 2; input++ {
				g1 := byte(input^d3^d4) & 1
				g2 := byte(input^d1^d2^d4) & 1
				cost := pm[cur]
				if rxG1 != DepunctureMark && g1 != rxG1 {
					cost++
				}
				if rxG2 != DepunctureMark && g2 != rxG2 {
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

	// Encoder is flushed back to state 0 by tail bits — pick state 0
	// unconditionally so the survivor uses the terminal-state
	// constraint.
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

// EncodeK5 is the inverse of ViterbiK5: given `stages` input bits
// (each 0 or 1, *including* the K-1 = 4 tail bits of zero needed to
// flush the encoder), returns `2*stages` channel bits as alternating
// (g1, g2) pairs. Useful for synthetic test streams.
func EncodeK5(input []byte) []byte {
	out := make([]byte, 2*len(input))
	var d1, d2, d3, d4 byte
	for i, in := range input {
		bit := in & 1
		g1 := bit ^ d3 ^ d4
		g2 := bit ^ d1 ^ d2 ^ d4
		out[2*i] = g1
		out[2*i+1] = g2
		// Shift register: new d1 = bit, others slide right.
		d4 = d3
		d3 = d2
		d2 = d1
		d1 = bit
	}
	return out
}
