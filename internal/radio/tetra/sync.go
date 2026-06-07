package tetra

// DibitSink consumes the raw stream of dibits a TETRA receiver
// decodes from IQ. baseIdx is the absolute dibit index of dibits[0]
// across the stream lifetime — monotonically non-decreasing across
// calls, and reset to 0 by Receiver.Reset so a retune produces a
// fresh baseline. TETRA is dibit-oriented (π/4-DQPSK, 18000 sym/s,
// 1 dibit per symbol). Wire this into a future
// ControlChannel.Process adapter (38-dibit training sync detect →
// burst slice → channel decode → L3 PDU dispatch) so the connector
// can drive the TETRA CC state machine on live IQ.
type DibitSink func(dibits []uint8, baseIdx int)

// TETRA training sequences per ETSI EN 300 392-2 §9.4.4.3, as
// MSB-first on-air BIT arrays. These are the real over-the-air
// sequences — validated against live captures for issue #553. The
// previous uint64 "hex" constants were placeholders that (a) were
// truncated (a uint64 holds 64 bits but the values were declared as
// 76) and (b) matched no spec value, so the control channel could
// never lock on real air.
//
// Lengths, in on-air bits and recovered dibits (1 dibit / symbol):
//
//	NormalTrainingSeq1/2   22 bits / 11 dibits  (§9.4.4.3.2)
//	ExtendedTrainingSeq    30 bits / 15 dibits  (§9.4.4.3.3)
//	SyncTrainingSeq        38 bits / 19 dibits  (§9.4.4.3.4)
//
// Bit↔dibit convention: on-air TETRA is π/4-DQPSK; a transmitted bit
// pair (b1,b2) modulates to a differential phase the demodulator
// recovers as a dibit VALUE 0..3 per the TETRA Gray mapping
// (b1,b2) → (b1<<1)|(b1^b2): 00→0, 01→1, 11→2, 10→3. TetraBitsToDibits
// / TetraDibitsToBits are the single source of truth for this and are
// deliberately distinct from the linear framing.DibitsToBits used by
// the C4FM family (P25/DMR/NXDN/dPMR).
var (
	// NormalTrainingSeq1 — normal training sequence 1 (§9.4.4.3.2),
	// carried by the normal continuous downlink burst.
	NormalTrainingSeq1 = []uint8{1, 1, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0}
	// NormalTrainingSeq2 — normal training sequence 2 (§9.4.4.3.2).
	NormalTrainingSeq2 = []uint8{0, 1, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1, 0}
	// ExtendedTrainingSeq — extended training sequence (§9.4.4.3.3).
	ExtendedTrainingSeq = []uint8{1, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0}
	// SyncTrainingSeq — synchronisation training sequence (§9.4.4.3.4),
	// carried by the synchronisation downlink burst (SB) at slot 1 of
	// frame 18 in each multiframe.
	SyncTrainingSeq = []uint8{1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1}
)

// On-air bit lengths of the training sequences (each = 2× its dibit
// count).
const (
	NormalTrainingBits   = 22
	ExtendedTrainingBits = 30
	SyncTrainingBits     = 38
)

// TetraBitsToDibits maps an MSB-first on-air bit stream to the dibit
// VALUE sequence the π/4-DQPSK demodulator (internal/dsp/demod) emits
// for it, using the TETRA Gray mapping (see the package note above). A
// trailing odd bit, if any, is dropped.
func TetraBitsToDibits(bits []uint8) []uint8 {
	n := len(bits) / 2
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		b1 := bits[2*i] & 1
		b2 := bits[2*i+1] & 1
		out[i] = (b1 << 1) | (b1 ^ b2)
	}
	return out
}

// TetraDibitsToBits is the inverse of TetraBitsToDibits: it expands a
// demodulated dibit-value stream back to MSB-first on-air bit pairs.
// Use this — not framing.DibitsToBits (the linear C4FM convention) —
// for TETRA channel decoding.
func TetraDibitsToBits(dibits []uint8) []byte {
	out := make([]byte, len(dibits)*2)
	for i, d := range dibits {
		b1 := (d >> 1) & 1
		out[2*i] = b1
		out[2*i+1] = b1 ^ (d & 1)
	}
	return out
}

// NormalSyncDibits returns normal training sequence 1 as the 11-dibit
// pattern the demodulator emits for it. NormalSyncDibits2,
// ExtendedSyncDibits and SyncTrainingDibits expose the others.
func NormalSyncDibits() []uint8 { return TetraBitsToDibits(NormalTrainingSeq1) }

// NormalSyncDibits2 returns normal training sequence 2 as a dibit pattern.
func NormalSyncDibits2() []uint8 { return TetraBitsToDibits(NormalTrainingSeq2) }

// ExtendedSyncDibits returns the extended training sequence (15 dibits).
func ExtendedSyncDibits() []uint8 { return TetraBitsToDibits(ExtendedTrainingSeq) }

// SyncTrainingDibits returns the synchronisation training sequence (19 dibits).
func SyncTrainingDibits() []uint8 { return TetraBitsToDibits(SyncTrainingSeq) }

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
