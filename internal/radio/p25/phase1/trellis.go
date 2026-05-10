package phase1

// P25 Phase 1 1/2-rate trellis convolutional code, per TIA-102.BAAA-A
// Annex A. The TSBK (and other data-block channels) carry 48 information
// dibits — i.e. 12 bytes — which the trellis encoder turns into 98
// channel dibits (= 196 transmitted bits). This is NOT the standard
// (7,5) octal NASA convolutional code; it's a table-driven code whose
// state IS the most-recent input dibit and whose transition outputs are
// chosen from a 16-entry constellation table (Annex A Table A.1).
//
// State diagram (state → input dibit → next state, output (hi, lo)):
//
//	state ∈ {0..3} = previous input dibit
//	for each input dibit d in {0..3}:
//	    next  = d
//	    idx   = trellisStates[state][next]
//	    (hi, lo) = trellisPairs[idx]
//	    state = next
//
// At end-of-block the encoder feeds one finisher dibit (= 0) so the
// state machine returns to state 0 — that's the 49-th transition that
// produces the 98-th output dibit.

// trellisStates is the (cur, next) → constellation-index table from
// TIA-102.BAAA-A Annex A Table A.1.
var trellisStates = [4][4]int{
	{0, 15, 12, 3},
	{4, 11, 8, 7},
	{13, 2, 1, 14},
	{9, 6, 5, 10},
}

// trellisPairs is the 16-entry (hi, lo) constellation table.
var trellisPairs = [16][2]uint8{
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

// EncodeTrellis encodes 48 information dibits into 98 channel dibits.
// The 49-th transition is the standard "finisher" (input dibit = 0)
// that flushes the encoder back to state 0.
func EncodeTrellis(info []uint8) []uint8 {
	if len(info) != 48 {
		panic("p25/phase1: trellis input must be 48 dibits")
	}
	out := make([]uint8, 0, 98)
	state := 0
	for _, d := range info {
		next := int(d & 0x3)
		idx := trellisStates[state][next]
		out = append(out, trellisPairs[idx][0], trellisPairs[idx][1])
		state = next
	}
	idx := trellisStates[state][0]
	out = append(out, trellisPairs[idx][0], trellisPairs[idx][1])
	return out
}

// DecodeTrellis runs hard-decision Viterbi over 98 channel dibits and
// returns the most-likely 48 information dibits plus the path metric of
// the surviving path (the sum of dibit-distance penalties along the
// chosen path). A metric of 0 means the channel was clean; positive
// values represent corrected dibit errors. Callers can compare the
// metric against a threshold to flag low-confidence decodes.
func DecodeTrellis(channel []uint8) ([]uint8, int) {
	if len(channel) != 98 {
		panic("p25/phase1: trellis input must be 98 dibits")
	}
	const inf = 1 << 30
	const stages = 49

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
				idx := trellisStates[cur][next]
				cost := pm[cur] +
					trellisDibitDist(trellisPairs[idx][0], hi) +
					trellisDibitDist(trellisPairs[idx][1], lo)
				if cost < npm[next] {
					npm[next] = cost
					trace[s][next] = uint8(cur)
				}
			}
		}
		pm = npm
	}

	// Encoder is flushed to state 0 on the last transition; if a clean
	// terminal state isn't 0, the closest one wins.
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
	return out[:48], finalMetric
}

func trellisDibitDist(a, b uint8) int {
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
