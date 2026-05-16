package dstar

// BitSink consumes the raw stream of bits a D-STAR receiver decodes
// from the GMSK demodulator + matched filter + symbol-clock recovery
// chain. D-STAR is 2-level (GMSK at 4800 bps) so every symbol carries
// exactly one bit — unlike the 4-level C4FM family which fans into
// DibitSink. Wire it through `internal/radio/dstar/receiver.Options`.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime so downstream sync detection / window slicing can report
// stable positions.
type BitSink func(bits []byte, baseIdx int)

// D-STAR sync words per the JARL D-STAR DV-mode specification.
// Two distinct patterns mark the two burst types a receiver tracks:
//
//	FrameSync     32-bit "Frame Sync" prefix that opens every voice
//	              + data superframe (the "FRMSYNC" bit pattern).
//	DataSync      24-bit Slow Data sync inside the voice payload's
//	              alternating-frame slow-data channel.
//
// Stored as canonical hex; helpers materialise the dibits MSB-first
// for the symbol-domain detector (D-STAR is GMSK so symbols are
// bits, not dibits, but the higher-level detector is the same shape
// across all protocols here — call-sites pack their bits into the
// dibit slots two-bits-at-a-time when they reach this stage).
const (
	// FrameSyncHex is the 24-bit Header Frame Sync that prefixes
	// every D-STAR PCH (Preamble + Header) per the JARL DV-mode
	// specification. The canonical value 0xEAA060 ("111 0101 0101
	// 0000 0011 0000 0000" MSB-first, then 0x60 padding) is the
	// pattern emitted by MMDVMHost / DSDcc / OpenDV transmitters.
	// On-air it follows a long bit-sync preamble (alternating
	// 1010… for clock-recovery convergence); the bit-sync isn't
	// part of the sync correlator because it has no information
	// alignment value.
	FrameSyncHex uint64 = 0xEAA060

	// SlowDataSyncHex is the 24-bit Slow Data sync used inside voice
	// frames to demarcate the slow-data channel. JARL spec value.
	SlowDataSyncHex uint64 = 0x55_2D16

	FrameSyncBits    = 24
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
