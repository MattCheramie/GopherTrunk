package nxdn

// DibitSink consumes the raw stream of dibits an NXDN receiver
// decodes from IQ. baseIdx is the absolute dibit index of dibits[0]
// across the stream lifetime — monotonically non-decreasing across
// calls, and reset to 0 by Receiver.Reset so a retune produces a
// fresh baseline. Wire this into a future ControlChannel.Process
// adapter (sync detect → 192-dibit frame slice → LICH/SACCH decode
// → IngestFrame) so the connector can drive the NXDN CC state
// machine on live IQ.
type DibitSink func(dibits []uint8, baseIdx int)

// FSW patterns (16 bits = 8 dibits). These constants follow the NXDN
// Common Air Interface specification §6.2.1; the exact bit pattern
// should be cross-checked against the published technical document
// before live captures (the constants below match the most commonly
// cited values in public reference implementations).
const (
	FSWOutboundHex uint16 = 0xC55A // BS → MS
	FSWInboundHex  uint16 = 0x3AA5 // MS → BS
)

// FSWDibitsOutbound is the 8-dibit decomposition of FSWOutboundHex.
var FSWDibitsOutbound = hexToDibits(FSWOutboundHex, FSWDibits)

// FSWDibitsInbound is the 8-dibit decomposition of FSWInboundHex.
var FSWDibitsInbound = hexToDibits(FSWInboundHex, FSWDibits)

func hexToDibits(hex uint16, n int) []uint8 {
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		shift := uint(2 * (n - 1 - i))
		out[i] = uint8((hex >> shift) & 0x3)
	}
	return out
}

// SyncDetector slides a window over an incoming dibit stream and emits
// the index where any of the supplied FSW patterns matches within the
// configured tolerance. The detector compares dibit values directly, so
// callers should pre-map their slicer output via NXDN's symbol
// constellation.
type SyncDetector struct {
	patterns  [][]uint8
	tolerance int
	hist      []uint8
	primed    int
	pos       int
}

// NewSyncDetector accepts one or more FSW patterns (each FSWDibits
// long) and a max-mismatch tolerance.
func NewSyncDetector(patterns [][]uint8, tolerance int) *SyncDetector {
	if tolerance < 0 {
		tolerance = 1
	}
	if len(patterns) == 0 {
		patterns = [][]uint8{FSWDibitsOutbound}
	}
	for _, p := range patterns {
		if len(p) != FSWDibits {
			panic("nxdn: FSW pattern must be FSWDibits long")
		}
	}
	return &SyncDetector{
		patterns:  patterns,
		tolerance: tolerance,
		hist:      make([]uint8, FSWDibits),
	}
}

// Match reports a sync hit: the dibit index where the FSW ends.
type Match struct {
	Index   int
	Inbound bool // true if the matched pattern was FSWDibitsInbound
}

// Process scans src and appends matches to dst. baseIndex is the absolute
// dibit index of src[0]. Returns the new baseIndex.
func (s *SyncDetector) Process(dst []Match, src []uint8, baseIndex int) ([]Match, int) {
	for i, d := range src {
		s.hist[s.pos] = d
		s.pos = (s.pos + 1) % FSWDibits
		if s.primed < FSWDibits {
			s.primed++
			continue
		}
		bestErrs := s.tolerance + 1
		bestPattern := -1
		for pi, pat := range s.patterns {
			errs := 0
			idx := s.pos
			for k := 0; k < FSWDibits; k++ {
				if s.hist[idx] != pat[k] {
					errs++
					if errs >= bestErrs {
						break
					}
				}
				idx = (idx + 1) % FSWDibits
			}
			if errs < bestErrs {
				bestErrs = errs
				bestPattern = pi
			}
		}
		if bestErrs <= s.tolerance && bestPattern >= 0 {
			inbound := false
			if len(s.patterns) > bestPattern && len(s.patterns[bestPattern]) == FSWDibits {
				inbound = dibitsEqual(s.patterns[bestPattern], FSWDibitsInbound)
			}
			dst = append(dst, Match{Index: baseIndex + i, Inbound: inbound})
		}
	}
	return dst, baseIndex + len(src)
}

func dibitsEqual(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
