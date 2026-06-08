package framing

// Soft-decision (LLR) variants of the TETRA signalling channel-coding
// primitives. They mirror the hard-decision functions in
// scramble_tetra.go, interleave_tetra.go and rcpc_tetra_sig.go exactly,
// but carry per-bit log-likelihood ratios (float32) instead of hard
// bits. Convention throughout: LLR > 0 ⇒ bit 0, LLR < 0 ⇒ bit 1, and
// magnitude is reliability (0 = erasure / no information). The soft
// Viterbi's surviving path is scale-invariant, so the LLRs need no
// normalisation — any consistent scaling works.

// DescrambleTetraSoft is the soft-LLR analog of DescrambleTetra:
// XOR-ing a bit with the scrambling PN becomes a conditional sign flip
// of its LLR (PN bit 1 inverts the bit, hence the LLR sign). Reuses the
// same LFSR (NewScramblerTetra). Scrambling is XOR-symmetric, so this
// is both scramble and descramble in the soft domain.
func DescrambleTetraSoft(in []float32, colourCode uint32) []float32 {
	s := NewScramblerTetra(colourCode)
	out := make([]float32, len(in))
	for i, v := range in {
		if s.Next() == 1 {
			out[i] = -v
		} else {
			out[i] = v
		}
	}
	return out
}

// BlockDeinterleaveTetraSoft is the soft-LLR analog of
// BlockDeinterleaveTetra — the same (K, a) inverse permutation applied
// to LLRs.
func BlockDeinterleaveTetraSoft(b4 []float32, K, a int) []float32 {
	if len(b4) != K {
		return nil
	}
	out := make([]float32, K)
	for i := 1; i <= K; i++ {
		k := 1 + ((a * i) % K)
		out[i-1] = b4[k-1]
	}
	return out
}

// DepunctureRCPCTetraSigSoft is the soft-LLR analog of
// DepunctureRCPCTetraSig: it allocates a motherLen LLR buffer, leaves
// punctured positions at 0 (a soft erasure — no information, zero
// branch-metric contribution in the Viterbi), and copies received LLRs
// at the puncture-map positions. Same index map as the hard version.
func DepunctureRCPCTetraSigSoft(punctured []float32, puncture []int, motherLen int, indexShift func(j int) int) []float32 {
	const period = 8
	t := len(puncture)
	if indexShift == nil {
		indexShift = func(j int) int { return j }
	}
	out := make([]float32, motherLen) // zero-valued = erasure
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

// DecodeRCPCTetraSigMotherSoft is the soft-decision analog of
// DecodeRCPCTetraSigMother: the same 16-state K=5 R=1/4 mother-code
// trellis, with the hard Hamming branch metric replaced by the soft
// correlation cost. For an expected mother bit g and received LLR L,
// the branch cost is L·(2g-1): a match (g=0 with L>0, or g=1 with L<0)
// lowers the path metric, a mismatch raises it, and an erasure (L=0)
// contributes nothing. The minimum-cost path is the maximum-likelihood
// sequence. Returns the recovered stages-bit input plus the surviving
// path metric.
func DecodeRCPCTetraSigMotherSoft(channel []float32, stages int) ([]byte, float32) {
	const numStates = 16
	const inf = float32(1e30)
	pm := make([]float32, numStates)
	for i := range pm {
		pm[i] = inf
	}
	pm[0] = 0
	trace := make([][numStates]uint8, stages)

	for s := 0; s < stages; s++ {
		var npm [numStates]float32
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
				g1 := (input ^ d1 ^ d4) & 1
				g2 := (input ^ d2 ^ d3 ^ d4) & 1
				g3 := (input ^ d1 ^ d2 ^ d4) & 1
				g4 := (input ^ d1 ^ d3 ^ d4) & 1
				cost := pm[cur]
				cost += rxG1 * float32(2*g1-1)
				cost += rxG2 * float32(2*g2-1)
				cost += rxG3 * float32(2*g3-1)
				cost += rxG4 * float32(2*g4-1)
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
