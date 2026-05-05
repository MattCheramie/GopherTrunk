package phase2

// Phase 2 sync words per TIA-102.BBAB §6. Outbound (BS → MS) and
// inbound (MS → BS) carry distinct patterns so a receiver can lock
// onto the appropriate side of the link. Stored as the canonical
// 40-bit (20-dibit) hex constants, with helpers to materialise the
// dibits MSB-first.
const (
	OutboundSyncHex uint64 = 0x575_F7DFF_77FF
	InboundSyncHex  uint64 = 0xDFF_57D75_DF5D
	SyncDibits             = 20
)

// OutboundSyncDibits returns the 20 dibits of the outbound sync
// word, MSB-first.
func OutboundSyncDibits() []uint8 { return hexToDibits(OutboundSyncHex, SyncDibits) }

// InboundSyncDibits returns the 20 dibits of the inbound sync word,
// MSB-first.
func InboundSyncDibits() []uint8 { return hexToDibits(InboundSyncHex, SyncDibits) }

func hexToDibits(hex uint64, n int) []uint8 {
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		shift := uint(2 * (n - 1 - i))
		out[i] = uint8((hex >> shift) & 0x3)
	}
	return out
}

// SyncDetector slides a window over a dibit stream and reports
// indices where the supplied sync pattern matches within
// `tolerance` dibit mismatches. Same shape as the Phase 1 / DMR /
// NXDN sync detectors so the higher-level state machine stays
// consistent.
type SyncDetector struct {
	pattern   []uint8
	tolerance int
	hist      []uint8
	primed    int
	pos       int
}

// NewSyncDetector accepts a 20-dibit pattern (typically
// OutboundSyncDibits or InboundSyncDibits) and a tolerance. A zero
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
