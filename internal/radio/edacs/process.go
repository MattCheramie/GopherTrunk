package edacs

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// processState is the cross-call bit buffering + sync-detection
// state the Process adapter holds. Lazily initialised on the first
// Process call.
type processState struct {
	det *SyncDetector
	// remaining > 0: collecting CCW bits after a sync match;
	// counts down to 0 as Process feeds bits forward.
	remaining int
	// ccw accumulates the 40 bits that make up one CCW info block.
	ccw []byte
	// matchScratch is reused across calls so SyncDetector.Process
	// doesn't allocate fresh slices.
	matchScratch []int
}

// Process consumes a window of raw bits from the EDACS receiver
// (the IQ → GFSK bit chain in internal/radio/edacs/receiver/), runs
// the 24-bit outbound sync detector, slices the following 40-bit
// CCW out of the stream, parses it via CCWFromBits, and forwards
// the result to Ingest.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime. The adapter's internal countdown survives across
// Process calls so a sync match in one chunk and the payload in
// the next still decode cleanly.
//
// The per-CCW BCH(40, 28, 2) FEC layer is gated by SetBCHMode —
// BCHOff (default) reads the 40 wire bits straight as the
// information field (matches test fixtures and clean fixtures);
// BCHOn runs framing.BCHDecodeEDACS over each 40-bit codeword and
// recovers up to 2 bit errors before parsing. Per the canonical
// open reference (lwvmobile/edacs-fm), BCH is the only on-wire FEC
// on the Standard EDACS CCW — no outer interleaved or
// Reed-Solomon-derived layer sits above it.
//
// Returns baseIdx + len(bits) to match the YSF / P25 Phase 1 /
// dPMR / NXDN ControlChannel.Process contracts.
func (c *ControlChannel) Process(bits []byte, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det: NewSyncDetector(OutboundSyncBits(), 1),
			ccw: make([]byte, 0, 40),
		}
	}
	p := c.proc
	c.mu.Lock()
	mode := c.bchMode
	c.mu.Unlock()

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], bits, baseIdx)
	matchIdx := 0

	for i, b := range bits {
		absPos := baseIdx + i
		// Collect first (this bit completes the 40-bit CCW if
		// remaining counts down to 0). Order matters: the sync
		// match's absolute index is the LAST bit of the 24-bit
		// sync, so the CCW starts at the NEXT iteration.
		if p.remaining > 0 {
			p.ccw = append(p.ccw, b&1)
			p.remaining--
			if p.remaining == 0 {
				if ccw, ok := c.parseCCW(p.ccw, mode); ok {
					c.Ingest(ccw)
				}
				p.ccw = p.ccw[:0]
			}
		}
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == absPos {
			p.remaining = 40
			p.ccw = p.ccw[:0]
			matchIdx++
		}
	}
	return baseIdx + len(bits)
}

// parseCCW turns a 40-bit wire window into a CCW. Under BCHOff
// the window is treated as 40 bits of pre-stripped information;
// under BCHOn the window is the on-wire BCH(40,28,2) codeword:
// the adapter packs it into the framing primitive's uint64
// convention, runs BCH validation + correction, then re-encodes
// the corrected 28-bit info field into a fresh 40-bit wire word
// (parity bits set to zero / cleared) so the existing
// CCWFromBits parser can interpret it. Uncorrectable codewords
// (errs == -1) are dropped by returning (zero, false).
func (c *ControlChannel) parseCCW(wire []byte, mode BCHMode) (CCW, bool) {
	if len(wire) != 40 {
		return CCW{}, false
	}
	if mode != BCHOn {
		w, err := CCWFromBits(wire)
		if err != nil {
			return CCW{}, false
		}
		return w, true
	}
	// Pack 40 wire bits into the framing primitive's uint64
	// convention. wire[0] is the MSB of the on-wire CCW (Command
	// bit 3), which maps to bit 39 of the codeword; wire[39] is
	// the LSB (Aux bit 0), which maps to bit 0.
	var cw uint64
	for i := 0; i < 40; i++ {
		if wire[i]&1 != 0 {
			cw |= uint64(1) << uint(39-i)
		}
	}
	info28, errs := framing.BCHDecodeEDACS(cw)
	if errs == -1 {
		return CCW{}, false
	}
	// Re-encode the corrected info field into a clean wire word.
	// The parity bits in positions 0..11 become a valid checksum
	// of the corrected info; CCWFromBits then reads the field
	// values out of the canonical bit positions.
	corrected := framing.BCHEncodeEDACS(info28)
	out := make([]byte, 40)
	for i := 0; i < 40; i++ {
		if corrected&(uint64(1)<<uint(39-i)) != 0 {
			out[i] = 1
		}
	}
	w, err := CCWFromBits(out)
	if err != nil {
		return CCW{}, false
	}
	return w, true
}

// Reset clears the SyncDetector's history so a stale match doesn't
// fire after a stream re-sync.
func (s *SyncDetector) Reset() {
	for i := range s.hist {
		s.hist[i] = 0
	}
	s.primed = 0
	s.pos = 0
}
