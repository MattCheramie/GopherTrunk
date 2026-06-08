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

// RotationSet enumerates the cyclic dibit rotations the FSW correlator
// and the NID alignment search probe. The zero value (nil) means "use
// the default all-rotation set" — every existing caller stays
// unchanged. Callers that know the demod mode can pass a restricted
// set: only rotation 0 (identity) and rotation 2 (FM-discriminator
// polarity flip) are physically meaningful on a C4FM stream, so rot 1
// and rot 3 wins there are structurally BCH miscorrections (issue
// #275: post-#321 retests converged on rot=3 on a C4FM site).
type RotationSet []uint8

// RotationsAll covers every cyclic dibit-alphabet rotation; the
// CQPSK / π/4-DQPSK path needs all four (the differential decode has
// a genuine four-fold phase ambiguity) and it stays the default.
var RotationsAll = RotationSet{0, 1, 2, 3}

// RotationsC4FM restricts the search to the rotations a C4FM
// FM-discriminator stream can physically present: 0 (identity) and 2
// (discriminator polarity flip). Rot 1 and rot 3 are non-physical on
// C4FM, so excluding them prevents the BCH decoder from miscorrecting
// misaligned dibits into a parity-valid pseudo-NID at a wrong rotation.
var RotationsC4FM = RotationSet{0, 2}

// resolveRotations returns the rotation set to iterate, falling back
// to RotationsAll when the caller passed nil.
func resolveRotations(r RotationSet) RotationSet {
	if len(r) == 0 {
		return RotationsAll
	}
	return r
}

// SyncDetector slides a window over an incoming dibit stream and emits the
// (zero-based) dibit index where the FSW best matches above tolerance.
// Tolerance is the maximum allowed dibit-symbol mismatch; default 4.
//
// The detector tries cyclic rotations of the dibit alphabet (applied as
// (dibit + k) mod 4 before comparing against the canonical
// FrameSyncWord) and records the rotation that matched. The default is
// all four rotations — k ∈ {0, 1, 2, 3} — to absorb residual
// symbol-polarity / I-Q-swap ambiguities the front-end may have
// introduced; without it the C4FM path slipped to dibit 3↔0 / 1↔2 on
// conjugated IQ inputs, and the CQPSK path on rare DQPSK quadrant
// slips. Rotation=0 wins on ties so existing clean-fixture tests bind
// the same hit they always have.
//
// Callers that know the demod mode can call SetRotations to restrict
// the set — see RotationsC4FM for the C4FM-specific subset.
//
// Callers needing the rotation per hit use ProcessWithRotation; the
// simpler Process API stays at the same signature for the rest of the
// pipeline.
type SyncDetector struct {
	tolerance int
	rotations RotationSet
	hist      [24]uint8
	primed    int
	pos       int
}

func NewSyncDetector(tolerance int) *SyncDetector {
	if tolerance < 0 {
		tolerance = 4
	}
	return &SyncDetector{tolerance: tolerance, rotations: RotationsAll}
}

// SetRotations restricts (or restores) the rotations the FSW
// correlator tries. Passing nil or an empty set restores the default
// all-rotation behaviour.
func (s *SyncDetector) SetRotations(r RotationSet) {
	s.rotations = resolveRotations(r)
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
	dst, rots, _, next := s.ProcessWithMargin(dst, rots, nil, src, baseIndex)
	return dst, rots, next
}

// ProcessWithMargin behaves like ProcessWithRotation but additionally returns,
// per emitted hit, the correlation margin: tolerance+1 − bestMismatch, where
// bestMismatch is the fewest dibit-symbol mismatches the FSW matched at across
// the tried rotations. A perfect 24/24 hit at tolerance 4 has margin 5; a hit
// that only just cleared tolerance has margin 1. The distribution of this
// number over a capture is the sync-health metric the measurement harness
// reports — a margin pressed against 1 means sync is barely holding and a
// little more noise would drop the lock. dst, rots and margins stay in
// lockstep; pass nil for any to let the detector allocate.
func (s *SyncDetector) ProcessWithMargin(dst []int, rots []uint8, margins []int, src []uint8, baseIndex int) ([]int, []uint8, []int, int) {
	rotations := resolveRotations(s.rotations)
	for i, d := range src {
		s.hist[s.pos] = d
		s.pos = (s.pos + 1) % 24
		if s.primed < 24 {
			s.primed++
			continue
		}
		bestMis := s.tolerance + 1
		var bestRot uint8
		for _, k := range rotations {
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
			margins = append(margins, s.tolerance+1-bestMis)
		}
	}
	return dst, rots, margins, baseIndex + len(src)
}
