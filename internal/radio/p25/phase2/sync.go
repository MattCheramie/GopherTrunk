package phase2

// DibitSink consumes the raw stream of dibits a Phase 2 receiver
// decodes from IQ. baseIdx is the absolute dibit index of dibits[0]
// across the stream lifetime — monotonically non-decreasing across
// calls, and reset to 0 by Receiver.Reset so a retune produces a
// fresh baseline. Phase 2 is dibit-oriented (H-DQPSK, 6000 sym/s,
// 1 dibit per symbol). Wire this into a future
// ControlChannel.Process adapter (20-dibit sync detect → MAC PDU
// slice → MAC opcode dispatch) so the connector can drive the
// Phase 2 CC state machine on live IQ.
type DibitSink func(dibits []uint8, baseIdx int)

// Phase 2 sync words. Outbound (BS → MS) and inbound (MS → BS) carry distinct
// patterns so a receiver can lock onto the appropriate side of the link.
// hexToDibits materialises SyncDibits (20) dibits MSB-first from the low 40 bits
// of each constant, so a correct value must be exactly 40 bits — the previous
// OutboundSyncHex was 48 bits and its top byte was silently dropped (issue #813).
//
// OutboundSyncHex is expressed in GopherTrunk's *decoder* dibit convention, not
// the P25 standard's. The authoritative P25 Phase 2 outbound frame sync is
// 0x575D57F7FF (OP25 frame_sync_magics.h / SDRtrunk P25P2_FRAME_SYNC_MAGIC,
// TIA-102.BBAC). On air that sync is a 20-symbol π/4-DQPSK-family sequence whose
// differential phases are, per dibit, {00→+π/4, 01→+3π/4, 10→−π/4, 11→−3π/4}
// (SDRtrunk Dibit.java). Running exactly that physical sequence through this
// package's differential decoder (demod.DQPSK.Decode, the receiver's actual
// output) yields the dibit sequence below — 0x565956A6AA — which is what the
// SyncDetector must match on live traffic. (GopherTrunk's DQPSK.Decode assigns
// the two negative-phase symbols the dibit values 2 and 3 swapped relative to
// the P25 standard, so the standard 0x575D57F7FF becomes 0x565956A6AA in the
// decoded stream. That same swap means the MAC/voice *payload* would also decode
// with 2↔3 transposed — a likely remaining blocker for ALGID/KID that lives in
// the shared demod.DQPSK map and must be confirmed against a real capture before
// touching, since it is also used by the TETRA and P25 Phase 1 CQPSK paths.)
//
// The old OutboundSyncHex (0x575F7DFF77FF) was neither the standard value nor
// this decoded form — a garbled 48-bit constant that never matched real air, so
// Phase 2 superframes never locked while every synthesized round-trip test (which
// encodes with the same constant) passed. See TestSuperframeLocksOnSpecSync.
//
// InboundSyncHex is the uplink (MS → BS) sync, used only by a diagnostic
// detector (internal/siglab/detail.go); it is unverified against real air and
// against this decoder's convention — do not rely on it until confirmed.
const (
	OutboundSyncHex uint64 = 0x565956A6AA
	InboundSyncHex  uint64 = 0xDFF57D75DF5D
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
