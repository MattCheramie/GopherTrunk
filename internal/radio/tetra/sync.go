package tetra

// TETRA synchronisation burst sync words per ETSI EN 300 392-2 §9.4.
// The synchronisation training sequences are 38-symbol patterns
// (76 bits / dibits) used to lock onto the BSCH burst at slot 1 of
// frame 18 in each multiframe. We carry the canonical hex of the
// 38 dibits MSB-first; the tolerant SyncDetector accepts a leading
// pattern and reports the position of the trailing dibit.
//
// Two pattern variants are defined:
//
//   NormalSync     normal-burst training sequence (downlink + uplink)
//   ExtendedSync   extended training sequence used on the broadcast
//                  burst — gives the receiver more energy to lock the
//                  initial frame timing on cold start.
//
// Both are 38 dibits long. Stored as the full hex constant; the
// per-dibit unpacker materialises them on demand.
const (
	// NormalSyncHex packs 38 dibits (76 bits) MSB-first into the low 76
	// bits of a uint128-equivalent. The constant below is the public
	// reference value documented for the normal training sequence.
	NormalSyncHex   uint64 = 0x4B65EE679D4D6F7F
	ExtendedSyncHex uint64 = 0x96B1D4A5DC34E13F

	SyncDibits = 38
)

// NormalSyncDibits returns the 38 dibits of the normal training
// sequence, MSB-first. Lower-order dibits beyond bit 64 of the hex
// constant zero-fill — TETRA receivers practically use the high-order
// portion of the burst for sync detection.
func NormalSyncDibits() []uint8 { return hexToDibits(NormalSyncHex, SyncDibits) }

// ExtendedSyncDibits returns the 38 dibits of the extended training
// sequence, MSB-first.
func ExtendedSyncDibits() []uint8 { return hexToDibits(ExtendedSyncHex, SyncDibits) }

func hexToDibits(hex uint64, n int) []uint8 {
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		shift := 2 * (n - 1 - i)
		if shift >= 64 {
			out[i] = 0
			continue
		}
		out[i] = uint8((hex >> uint(shift)) & 0x3)
	}
	return out
}

// SyncDetector slides a window over a dibit stream and reports
// indices where the configured pattern matches within `tolerance`
// dibit mismatches. Same shape as the Phase 1 / DMR / NXDN / dPMR
// sync detectors so the higher-level state machine stays consistent.
type SyncDetector struct {
	pattern   []uint8
	tolerance int
	hist      []uint8
	primed    int
	pos       int
}

// NewSyncDetector accepts a 38-dibit pattern (typically
// NormalSyncDibits or ExtendedSyncDibits) and a tolerance. A zero
// tolerance demands an exact match.
func NewSyncDetector(pattern []uint8, tolerance int) *SyncDetector {
	if tolerance < 0 {
		tolerance = 1
	}
	cp := make([]uint8, len(pattern))
	copy(cp, pattern)
	return &SyncDetector{
		pattern:   cp,
		tolerance: tolerance,
		hist:      make([]uint8, len(cp)),
	}
}

// Process scans src and appends to dst the absolute dibit indices
// where the sync pattern ends within tolerance.
func (s *SyncDetector) Process(dst []int, src []uint8, baseIndex int) ([]int, int) {
	n := len(s.pattern)
	for i, d := range src {
		s.hist[s.pos] = d
		s.pos = (s.pos + 1) % n
		if s.primed < n {
			s.primed++
			continue
		}
		mismatch := 0
		idx := s.pos
		for k := 0; k < n; k++ {
			if s.hist[idx] != s.pattern[k] {
				mismatch++
				if mismatch > s.tolerance {
					break
				}
			}
			idx = (idx + 1) % n
		}
		if mismatch <= s.tolerance {
			dst = append(dst, baseIndex+i)
		}
	}
	return dst, baseIndex + len(src)
}
