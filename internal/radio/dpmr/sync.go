package dpmr

// DibitSink consumes the raw stream of dibits a dPMR receiver
// decodes from IQ. baseIdx is the absolute dibit index of
// dibits[0] across the stream lifetime — monotonically non-
// decreasing across calls, and reset to 0 by Receiver.Reset so a
// retune produces a fresh baseline. Wire this into a future
// ControlChannel.Process adapter (FS3 sync detect → 80-bit CSBK
// slice → CSBKFromBits → Ingest) so the connector can drive the
// dPMR CC state machine on live IQ.
type DibitSink func(dibits []uint8, baseIdx int)

// dPMR sync words per ETSI TS 102 658 §4.4. Three distinct 48-bit
// (24-dibit) patterns mark the three burst types Mode 3 uses:
//
//	FS1  start of a voice / data superframe
//	FS2  middle synchronisation inside a superframe (every 4th burst)
//	FS3  start of a CSBK (signalling) burst on the control channel
//
// All three are stored as canonical 48-bit hex constants; helpers
// materialise the dibits MSB-first for the symbol-domain detector.
const (
	FS1Hex uint64 = 0x57FF5F75F575
	FS2Hex uint64 = 0x5F7F77FD7DFD
	FS3Hex uint64 = 0x7DDFFD5F55D5

	SyncDibits = 24
)

// FS1Dibits returns the 24 dibits of FS1, MSB-first.
func FS1Dibits() []uint8 { return hexToDibits(FS1Hex, SyncDibits) }

// FS2Dibits returns the 24 dibits of FS2, MSB-first.
func FS2Dibits() []uint8 { return hexToDibits(FS2Hex, SyncDibits) }

// FS3Dibits returns the 24 dibits of FS3, MSB-first.
func FS3Dibits() []uint8 { return hexToDibits(FS3Hex, SyncDibits) }

func hexToDibits(hex uint64, n int) []uint8 {
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		shift := uint(2 * (n - 1 - i))
		out[i] = uint8((hex >> shift) & 0x3)
	}
	return out
}

// SyncDetector slides a window over a dibit stream and reports indices
// where the configured pattern matches within `tolerance` dibit
// mismatches. Same shape as the Phase 1 / DMR / NXDN sync detectors so
// the higher-level state machine stays consistent.
type SyncDetector struct {
	pattern   []uint8
	tolerance int
	hist      []uint8
	primed    int
	pos       int
}

// NewSyncDetector accepts a 24-dibit pattern (typically FS1Dibits,
// FS2Dibits, or FS3Dibits) and a tolerance. A zero tolerance demands
// an exact match.
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
