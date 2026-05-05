package motorola

// SyncWord is the 24-bit sync sequence prefacing each Motorola Type II
// SmartZone control-channel frame. Stored as MSB-first bits to mesh
// with the rest of the radio stack's framing layer.
//
// The standard Motorola SmartZone outbound control-channel sync is
// 0xA4D7AA (binary `1010 0100 1101 0111 1010 1010`) per public
// reference implementations. As with the other protocol packages, the
// constant is best-effort and should be cross-checked against an
// authoritative reference before trusting live captures.
const (
	OutboundSyncHex uint32 = 0xA4D7AA
	SyncBits               = 24
)

// OutboundSyncBits returns the 24 bits of the outbound sync MSB-first.
func OutboundSyncBits() []byte {
	out := make([]byte, SyncBits)
	for i := 0; i < SyncBits; i++ {
		out[i] = byte((OutboundSyncHex >> uint(SyncBits-1-i)) & 1)
	}
	return out
}

// SyncDetector slides a window over a 0/1 bit stream and reports
// indices where the sync word matches within `tolerance` mismatched
// bits. Mirrors the same shape used by the P25 / DMR / NXDN sync
// detectors so the higher-level state machine stays consistent.
type SyncDetector struct {
	pattern   []byte
	tolerance int
	hist      []byte
	primed    int
	pos       int
}

// NewSyncDetector returns a detector for the supplied pattern. Pass
// the length-24 result of OutboundSyncBits() unless you have a
// vendor-specific sync to track.
func NewSyncDetector(pattern []byte, tolerance int) *SyncDetector {
	if tolerance < 0 {
		tolerance = 1
	}
	cp := make([]byte, len(pattern))
	copy(cp, pattern)
	return &SyncDetector{
		pattern:   cp,
		tolerance: tolerance,
		hist:      make([]byte, len(cp)),
	}
}

// Process scans src and appends to dst the absolute bit indices where
// the sync word ends within tolerance.
func (s *SyncDetector) Process(dst []int, src []byte, baseIndex int) ([]int, int) {
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
