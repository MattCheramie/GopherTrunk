package edacs

// BitSink consumes the raw stream of bits an EDACS receiver decodes
// from IQ. baseIdx is the absolute bit index of bits[0] across the
// stream lifetime — monotonically non-decreasing across calls, and
// reset to 0 by Receiver.Reset so a retune produces a fresh
// baseline. EDACS is bit-oriented (2-level GFSK at 9600 baud); the
// other 4-level trunked protocols use a DibitSink instead. Wire
// this into a future ControlChannel.Process adapter (sync detect
// → 40-bit CCW slice → CCWFromBits → Ingest) so the connector can
// drive the EDACS CC state machine on live IQ.
type BitSink func(bits []byte, baseIdx int)

// SyncWord is the 24-bit sync sequence prefacing each EDACS Control
// Channel Word. Stored as MSB-first bits to mesh with the rest of the
// radio stack's framing layer.
//
// The standard EDACS outbound control-channel sync is documented as
// 0x55D5AA across multiple public reference implementations. As with
// the other protocol packages, the constant is best-effort and should
// be cross-checked against an authoritative reference before trusting
// live captures.
const (
	OutboundSyncHex uint32 = 0x55D5AA
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
// bits. Same shape as the Motorola / NXDN / DMR / P25 sync detectors.
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
