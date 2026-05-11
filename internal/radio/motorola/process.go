package motorola

// processState is the cross-call bit buffering + sync-detection
// state the Process adapter holds. Lazily initialised on the first
// Process call.
type processState struct {
	det          *SyncDetector
	remaining    int
	osw          []byte
	matchScratch []int
}

// oswInfoBits is the count of bits the adapter collects after each
// 24-bit sync match: 32 information bits = one OSW. The on-air
// 84-bit OSW frame (with BCH(64,16,11) FEC over the upper 64 bits)
// isn't reversed here — the adapter reads the 32 information bits
// straight from the wire, which works for test fixtures + clean
// signals but typically fails on noisy on-air captures. Adding the
// BCH layer is a documented follow-up.
const oswInfoBits = 32

// Process consumes a window of raw bits from the Motorola receiver
// (the IQ → MSK bit chain in internal/radio/motorola/receiver/),
// runs the 24-bit outbound sync detector, slices the following
// 32-bit OSW out of the stream, parses it via OSWFromBits, and
// forwards the result to Ingest.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime. The adapter's internal countdown survives across
// Process calls so a sync match in one chunk and the payload in
// the next still decode cleanly.
//
// Returns baseIdx + len(bits) to match the YSF / P25 Phase 1 /
// dPMR / NXDN / EDACS ControlChannel.Process contracts.
func (c *ControlChannel) Process(bits []byte, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det: NewSyncDetector(OutboundSyncBits(), 1),
			osw: make([]byte, 0, oswInfoBits),
		}
	}
	p := c.proc

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], bits, baseIdx)
	matchIdx := 0

	for i, b := range bits {
		absPos := baseIdx + i
		if p.remaining > 0 {
			p.osw = append(p.osw, b&1)
			p.remaining--
			if p.remaining == 0 {
				if osw, err := OSWFromBits(p.osw); err == nil {
					c.Ingest(osw)
				}
				p.osw = p.osw[:0]
			}
		}
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == absPos {
			p.remaining = oswInfoBits
			p.osw = p.osw[:0]
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
