package dstar

// D-STAR sync words per the JARL D-STAR DV-mode specification.
// Two distinct patterns mark the two burst types a receiver tracks:
//
//   FrameSync     32-bit "Frame Sync" prefix that opens every voice
//                 + data superframe (the "FRMSYNC" bit pattern).
//   DataSync      24-bit Slow Data sync inside the voice payload's
//                 alternating-frame slow-data channel.
//
// Stored as canonical hex; helpers materialise the dibits MSB-first
// for the symbol-domain detector (D-STAR is GMSK so symbols are
// bits, not dibits, but the higher-level detector is the same shape
// across all protocols here — call-sites pack their bits into the
// dibit slots two-bits-at-a-time when they reach this stage).
const (
	// FrameSyncHex is the 32-bit Frame Sync that prefixes every D-STAR
	// PCH (Preamble + Header). The bit pattern is the well-known
	// 0x55555555 toggle preceded by the JARL fixed identifier; we
	// store the 32 bits MSB-first as the lower 32 bits of the
	// constant.
	FrameSyncHex uint64 = 0x55_5555_5555_5555 & 0xFFFFFFFF

	// SlowDataSyncHex is the 24-bit Slow Data sync used inside voice
	// frames to demarcate the slow-data channel. JARL spec value.
	SlowDataSyncHex uint64 = 0x55_2D16

	FrameSyncBits    = 32
	SlowDataSyncBits = 24
)

// FrameSyncBitsSlice returns the 32 bits of the Frame Sync, MSB-first.
func FrameSyncBitsSlice() []uint8 { return hexToBits(FrameSyncHex, FrameSyncBits) }

// SlowDataSyncBitsSlice returns the 24 bits of the Slow Data sync,
// MSB-first.
func SlowDataSyncBitsSlice() []uint8 { return hexToBits(SlowDataSyncHex, SlowDataSyncBits) }

func hexToBits(hex uint64, n int) []uint8 {
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		shift := uint(n - 1 - i)
		out[i] = uint8((hex >> shift) & 0x1)
	}
	return out
}

// SyncDetector slides a window over a bit stream and reports indices
// where the configured pattern matches within `tolerance` bit
// mismatches. Same shape as the Phase 1 / DMR / NXDN / dPMR / TETRA
// sync detectors, but operating on bits rather than dibits — D-STAR
// is GMSK at 4800 bps so the symbol stream is per-bit.
type SyncDetector struct {
	pattern   []uint8
	tolerance int
	hist      []uint8
	primed    int
	pos       int
}

// NewSyncDetector accepts an N-bit pattern (typically
// FrameSyncBitsSlice or SlowDataSyncBitsSlice) and a tolerance.
// A zero tolerance demands an exact match.
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

// Process scans src and appends to dst the absolute bit indices where
// the sync pattern ends within tolerance.
func (s *SyncDetector) Process(dst []int, src []uint8, baseIndex int) ([]int, int) {
	n := len(s.pattern)
	for i, b := range src {
		s.hist[s.pos] = b & 1
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
