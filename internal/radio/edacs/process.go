package edacs

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
// The interleaved Reed-Solomon-derived FEC over the CCW is NOT
// reversed here — the package's CCW parser assumes the upstream
// caller delivered the 40 information bits already, which is the
// case for test fixtures + clean signals but not for noisy on-air
// captures. Adding the RS FEC layer is a documented follow-up.
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
				if ccw, err := CCWFromBits(p.ccw); err == nil {
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

// Reset clears the SyncDetector's history so a stale match doesn't
// fire after a stream re-sync.
func (s *SyncDetector) Reset() {
	for i := range s.hist {
		s.hist[i] = 0
	}
	s.primed = 0
	s.pos = 0
}
