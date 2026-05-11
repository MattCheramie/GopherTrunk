package dpmr

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// processState is the cross-call dibit buffering + sync-detection
// state the Process adapter holds. It's stashed on the
// ControlChannel via a lazily-initialised pointer so the existing
// Ingest path stays callable from tests that hand in pre-parsed
// CSBKs.
type processState struct {
	det *SyncDetector
	// remaining > 0: collecting CSBK dibits after a sync match;
	// counts down to 0 as Process feeds dibits forward.
	remaining int
	// csbk accumulates the 40 dibits that make up one 80-bit CSBK.
	csbk []uint8
	// matchScratch is reused across calls so SyncDetector.Process
	// doesn't allocate fresh slices on every Process call.
	matchScratch []int
}

// Process consumes a window of raw dibits from the dPMR receiver
// (the IQ → C4FM dibit chain in internal/radio/dpmr/receiver/), runs
// the FS3 control-channel sync detector over them, slices the
// following 40-dibit / 80-bit CSBK out of the stream, parses it via
// CSBKFromBits, and forwards the result to Ingest.
//
// baseIdx is the absolute dibit index of dibits[0] across the
// stream lifetime. The adapter doesn't track absolute positions
// itself — it uses an internal countdown that survives across
// Process calls so a sync match in one chunk and the CSBK payload
// in the next chunk both decode correctly. CSBKs whose 80 bits
// fail to parse (truncated stream, etc.) are dropped silently;
// the next sync re-anchors the stream.
//
// Returns baseIdx + len(dibits) to match the YSF / P25 Phase 1
// ControlChannel.Process contract — the ccdecoder connector
// passes the result back as the next chunk's baseIdx, which the
// adapter doesn't currently need but the contract keeps stable
// across protocols.
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det:  NewSyncDetector(FS3Dibits(), 1),
			csbk: make([]uint8, 0, 40),
		}
	}
	p := c.proc

	// Detect sync matches across the new dibit chunk in one call —
	// SyncDetector keeps its own internal history so a sync that
	// spans two Process calls is still found.
	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], dibits, baseIdx)
	matchIdx := 0

	for i, d := range dibits {
		absPos := baseIdx + i
		// Collect first (this dibit completes a CSBK if remaining
		// counts down to 0). Doing this BEFORE the sync-match check
		// is important: the sync match's absolute index is the LAST
		// dibit of the 24-dibit sync, so the CSBK starts at the
		// NEXT dibit — i.e. one iteration later.
		if p.remaining > 0 {
			p.csbk = append(p.csbk, d)
			p.remaining--
			if p.remaining == 0 {
				bits := framing.DibitsToBits(p.csbk)
				if csbk, err := CSBKFromBits(bits); err == nil {
					c.Ingest(csbk)
				}
				p.csbk = p.csbk[:0]
			}
		}
		// Check if a sync ended at this position. If yes, start
		// collecting CSBK dibits from the NEXT iteration.
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == absPos {
			p.remaining = 40
			p.csbk = p.csbk[:0]
			matchIdx++
		}
	}
	return baseIdx + len(dibits)
}

// Reset clears the Process adapter's internal sync-detection state
// + the partial CSBK buffer. Call on stream re-sync (control-channel
// hunt success, IQ underrun recovery) so a stale FS3 history
// doesn't bleed across the discontinuity.
func (s *SyncDetector) Reset() {
	for i := range s.hist {
		s.hist[i] = 0
	}
	s.primed = 0
	s.pos = 0
}
