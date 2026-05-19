// Package phase1 decodes the P25 Phase 1 (C4FM, FDMA) frame structure.
// Inputs are dibits (0..3) recovered by the upstream C4FM demodulator and
// symbol-time recovery; outputs are parsed NID and TSBK records.
package phase1

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// FrameSyncWord is the 48-bit P25 Phase 1 frame sync word, expressed as
// 24 dibits (TIA-102.BAAA §6.1.1). Hex: 0x5575F5FF77FF, MSB-first dibits.
var FrameSyncWord = [24]uint8{
	1, 1, 1, 1, 1, 3, 1, 1, 3, 3, 1, 1,
	3, 3, 3, 3, 1, 3, 1, 3, 3, 3, 3, 3,
}

// C4FM symbol-to-dibit mapping per TIA-102.BAAA: +3→01, +1→00, -1→10, -3→11.
// Input is the slicer output {-3,-1,+1,+3}; output is the dibit value 0..3.
func SymbolToDibit(sym int8) uint8 {
	switch sym {
	case 1:
		return 0
	case 3:
		return 1
	case -1:
		return 2
	case -3:
		return 3
	}
	return 0
}

// SymbolsToDibits converts a slicer-output stream into dibits.
func SymbolsToDibits(syms []int8) []uint8 {
	out := make([]uint8, len(syms))
	for i, s := range syms {
		out[i] = SymbolToDibit(s)
	}
	return out
}

// FrameSyncBits returns the 48 bits of the FSW MSB-first.
func FrameSyncBits() []byte { return framing.DibitsToBits(FrameSyncWord[:]) }

// SyncDetector slides a window over an incoming dibit stream and emits the
// (zero-based) dibit index where the FSW best matches above tolerance.
// Tolerance is the maximum allowed dibit-symbol mismatch; default 4.
//
// The detector tries all four cyclic rotations of the dibit alphabet
// (k ∈ {0, 1, 2, 3}, applied as (dibit + k) mod 4 before comparing
// against the canonical FrameSyncWord) and records the rotation that
// matched. The rotation absorbs residual symbol-polarity / I-Q-swap
// ambiguities the front-end may have introduced — without it the C4FM
// path slipped to dibit 3↔0 / 1↔2 on conjugated IQ inputs, and the
// CQPSK path on rare DQPSK quadrant slips. Rotation=0 wins on ties so
// existing clean-fixture tests bind the same hit they always have.
//
// Callers needing the rotation per hit use ProcessWithRotation; the
// simpler Process API stays at the same signature for the rest of the
// pipeline.
type SyncDetector struct {
	tolerance int
	hist      [24]uint8
	primed    int
	pos       int
}

func NewSyncDetector(tolerance int) *SyncDetector {
	if tolerance < 0 {
		tolerance = 4
	}
	return &SyncDetector{tolerance: tolerance}
}

// Process appends to dst the indices (relative to baseIndex) where the FSW
// is detected. baseIndex is the absolute dibit index of src[0].
//
// Equivalent to ProcessWithRotation with the rotation values discarded.
func (s *SyncDetector) Process(dst []int, src []uint8, baseIndex int) ([]int, int) {
	dst, _, next := s.ProcessWithRotation(dst, nil, src, baseIndex)
	return dst, next
}

// ProcessWithRotation behaves like Process but also returns the
// rotation k ∈ {0, 1, 2, 3} that produced each emitted hit. The
// returned dst and rots slices stay in lockstep — rots[i] is the
// rotation that recovered the hit at dst[i]. Callers that only need
// the indices can use Process and discard rots.
//
// Pass nil for rots to let the detector allocate; the returned rots
// slice is non-nil whenever dst is non-empty after this call.
func (s *SyncDetector) ProcessWithRotation(dst []int, rots []uint8, src []uint8, baseIndex int) ([]int, []uint8, int) {
	for i, d := range src {
		s.hist[s.pos] = d
		s.pos = (s.pos + 1) % 24
		if s.primed < 24 {
			s.primed++
			continue
		}
		bestMis := s.tolerance + 1
		var bestRot uint8
		for k := uint8(0); k < 4; k++ {
			mismatch := 0
			idx := s.pos
			for kk := 0; kk < 24; kk++ {
				if ((s.hist[idx] + k) & 3) != FrameSyncWord[kk] {
					mismatch++
					if mismatch >= bestMis {
						break
					}
				}
				idx = (idx + 1) % 24
			}
			if mismatch < bestMis {
				bestMis = mismatch
				bestRot = k
				if bestMis == 0 {
					break
				}
			}
		}
		if bestMis <= s.tolerance {
			dst = append(dst, baseIndex+i)
			rots = append(rots, bestRot)
		}
	}
	return dst, rots, baseIndex + len(src)
}
