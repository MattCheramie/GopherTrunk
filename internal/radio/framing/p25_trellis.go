package framing

// P25Trellis1Half is the 4-state 1/2-rate convolutional code defined by
// TIA-102.BAAA-A Annex A and reused by TIA-102.BBAB for the P25 Phase 2
// MAC PDU channel coding. The code is table-driven rather than
// generator-polynomial based: one input dibit drives a state transition
// that emits one output (hi, lo) dibit pair via a 16-entry constellation
// table.
//
// State diagram (state → input dibit → next state, output (hi, lo)):
//
//	state ∈ {0..3} = previous input dibit
//	for each input dibit d in {0..3}:
//	    next  = d
//	    idx   = p25TrellisStates[state][next]
//	    (hi, lo) = p25TrellisPairs[idx]
//	    state = next
//
// At end-of-block the encoder feeds one finisher dibit (= 0) so the
// state machine returns to state 0 — that's the (N+1)th transition that
// produces the (N+1)th output pair.
//
// Identical tables back both P25 Phase 1 TSBK trellis encoding (48 info
// dibits → 98 channel dibits, 49 transitions) and P25 Phase 2 MAC PDU
// trellis encoding (variable info length). Phase 1's per-package
// implementation predates this primitive; the new shared exports here
// let Phase 2 (and any other consumer) reuse the same code without
// duplicating tables. See internal/radio/p25/phase1/trellis.go for the
// Phase 1 wrappers that delegate to these primitives (or keep their
// own legacy copy until callers migrate).

// p25TrellisStates is the (cur, next) → constellation-index table from
// TIA-102.BAAA-A Annex A Table A.1.
var p25TrellisStates = [4][4]int{
	{0, 15, 12, 3},
	{4, 11, 8, 7},
	{13, 2, 1, 14},
	{9, 6, 5, 10},
}

// p25TrellisPairs is the 16-entry (hi, lo) constellation table.
var p25TrellisPairs = [16][2]uint8{
	{0b00, 0b10},
	{0b10, 0b10},
	{0b01, 0b11},
	{0b11, 0b11},
	{0b11, 0b10},
	{0b01, 0b10},
	{0b10, 0b11},
	{0b00, 0b11},
	{0b11, 0b01},
	{0b01, 0b01},
	{0b10, 0b00},
	{0b00, 0b00},
	{0b00, 0b01},
	{0b10, 0b01},
	{0b01, 0b00},
	{0b11, 0b00},
}

// EncodeP25Trellis encodes N information dibits into 2*(N+1) channel
// dibits via the TIA-102 Annex A 4-state 1/2-rate trellis. The (N+1)th
// transition is the standard "finisher" (input dibit = 0) that flushes
// the encoder back to state 0, so the decoder can pick state 0 as the
// terminal-state constraint.
func EncodeP25Trellis(info []uint8) []uint8 {
	out := make([]uint8, 0, 2*(len(info)+1))
	state := 0
	for _, d := range info {
		next := int(d & 0x3)
		idx := p25TrellisStates[state][next]
		out = append(out, p25TrellisPairs[idx][0], p25TrellisPairs[idx][1])
		state = next
	}
	idx := p25TrellisStates[state][0]
	out = append(out, p25TrellisPairs[idx][0], p25TrellisPairs[idx][1])
	return out
}

// DecodeP25Trellis runs hard-decision Viterbi over a channel-dibit
// sequence produced by EncodeP25Trellis and returns the most-likely
// information dibits plus the path metric of the surviving path (the
// sum of dibit-distance penalties along the chosen path). A metric of
// 0 means the channel was clean; positive values represent corrected
// dibit errors. Callers can compare the metric against a threshold to
// flag low-confidence decodes.
//
// The channel input length must be even and at least 2 (one finisher
// transition). The returned info-dibit slice has length (len(channel)
// / 2) - 1.
func DecodeP25Trellis(channel []uint8) ([]uint8, int) {
	const inf = 1 << 30
	if len(channel) < 2 || len(channel)%2 != 0 {
		return nil, inf
	}
	stages := len(channel) / 2

	pm := [4]int{0, inf, inf, inf}
	trace := make([][4]uint8, stages)

	for s := 0; s < stages; s++ {
		var npm [4]int
		for i := range npm {
			npm[i] = inf
		}
		hi := channel[2*s]
		lo := channel[2*s+1]
		for cur := 0; cur < 4; cur++ {
			if pm[cur] >= inf {
				continue
			}
			for next := 0; next < 4; next++ {
				idx := p25TrellisStates[cur][next]
				cost := pm[cur] +
					p25TrellisDibitDist(p25TrellisPairs[idx][0], hi) +
					p25TrellisDibitDist(p25TrellisPairs[idx][1], lo)
				if cost < npm[next] {
					npm[next] = cost
					trace[s][next] = uint8(cur)
				}
			}
		}
		pm = npm
	}

	// Encoder is flushed to state 0 on the final transition; if a
	// clean terminal state isn't 0, the closest one wins.
	finalState := 0
	for s := 1; s < 4; s++ {
		if pm[s] < pm[finalState] {
			finalState = s
		}
	}
	finalMetric := pm[finalState]

	out := make([]uint8, stages)
	state := finalState
	for s := stages - 1; s >= 0; s-- {
		out[s] = uint8(state)
		state = int(trace[s][state])
	}
	// Drop the terminal finisher transition — its "input dibit" was 0
	// and isn't carrying information.
	return out[:stages-1], finalMetric
}

func p25TrellisDibitDist(a, b uint8) int {
	d := (a ^ b) & 0x3
	switch d {
	case 0:
		return 0
	case 1, 2:
		return 1
	default:
		return 2
	}
}
