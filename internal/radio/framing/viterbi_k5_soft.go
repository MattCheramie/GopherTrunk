package framing

// ViterbiK5Soft is the soft-decision analog of ViterbiK5: the same 16-state
// K=5 rate-½ trellis (g1 = in^d3^d4, g2 = in^d1^d2^d4 — see EncodeK5), with
// the hard Hamming branch metric replaced by the soft correlation cost the
// TETRA soft primitives use (soft_tetra.go). Input is `2*stages` per-bit
// log-likelihood ratios arranged as (g1, g2) pairs. Convention: LLR > 0 ⇒
// bit 0, LLR < 0 ⇒ bit 1, magnitude is reliability, and 0.0 is an erasure —
// the soft replacement for DepunctureMark, contributing nothing to the
// branch metric, so depuncture by leaving dropped positions at zero. For an
// expected channel bit g and received LLR L the branch cost is L·(2g−1): a
// match lowers the path metric, a mismatch raises it. The survivor is
// scale-invariant, so the LLRs need no normalisation. The encoder is flushed
// to state 0 by tail bits, so the survivor is forced to state 0, exactly as
// in ViterbiK5. Returns the recovered `stages` input bits plus the surviving
// path metric.
func ViterbiK5Soft(channel []float32, stages int) ([]byte, float32) {
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
				g1 := (input ^ d3 ^ d4) & 1
				g2 := (input ^ d1 ^ d2 ^ d4) & 1
				cost := pm[cur]
				cost += rxG1 * float32(2*g1-1)
				cost += rxG2 * float32(2*g2-1)
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
