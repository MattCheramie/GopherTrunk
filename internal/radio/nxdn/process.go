package nxdn

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// processState is the cross-call dibit buffering + sync-detection
// state the Process adapter holds. Lazily initialised on the first
// Process call so the existing IngestFrame path stays callable
// from tests that hand in pre-parsed LICH + CAC structures.
type processState struct {
	det *SyncDetector
	// remaining > 0: collecting frame dibits after the FSW match;
	// counts down to 0 as Process feeds dibits forward.
	remaining int
	// frame accumulates the post-FSW frame dibits the adapter
	// slices into LICH wire bits + (skipped) SACCH + CAC info bits.
	frame []uint8
	// matchScratch is reused across calls so SyncDetector.Process
	// doesn't allocate fresh slices.
	matchScratch []Match
}

// postSyncDibits is the count of dibits the adapter collects after
// the 8-dibit FSW match: 8 LICH wire + 32 SACCH (skipped) + 44 CAC
// info dibits = 84 dibits. The remaining 100 dibits of the 144-dibit
// Info field carry the CAC FEC redundancy that a future package PR
// decodes; this adapter only uses the first 44 dibits, which is
// sufficient to drive cc.locked in test fixtures where the CAC
// information bits are placed directly in the wire (no FEC encoding).
// For real on-air NXDN traffic the CAC FEC layer is the next
// follow-up; until that lands this adapter will sync on the FSW
// but typically fail the CAC CRC and stay silent on production
// signals.
const postSyncDibits = 84

// Process consumes a window of raw dibits from the NXDN receiver
// (the IQ → C4FM dibit chain in internal/radio/nxdn/receiver/),
// runs the outbound-FSW detector, parses the LICH from the next 8
// wire dibits, and tries ParseCAC on the next 44 dibits' worth of
// information bits before handing the (lich, cac) pair to
// IngestFrame.
//
// baseIdx is the absolute dibit index of dibits[0] across the
// stream lifetime. The adapter's internal countdown survives
// across Process calls so a sync match in one chunk and the
// payload in the next still decode cleanly.
//
// Returns baseIdx + len(dibits) to match the YSF / P25 Phase 1 /
// dPMR ControlChannel.Process contracts.
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det:   NewSyncDetector([][]uint8{FSWDibitsOutbound}, 1),
			frame: make([]uint8, 0, postSyncDibits),
		}
	}
	p := c.proc

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], dibits, baseIdx)
	matchIdx := 0

	for i, d := range dibits {
		absPos := baseIdx + i
		// Collect first (this dibit completes the post-sync window
		// if remaining counts down to 0). Doing this BEFORE the
		// sync-match check is important: the sync match's absolute
		// index is the LAST dibit of the 8-dibit FSW, so the next
		// frame data starts at the NEXT iteration.
		if p.remaining > 0 {
			p.frame = append(p.frame, d)
			p.remaining--
			if p.remaining == 0 {
				c.tryIngestFrame(p.frame)
				p.frame = p.frame[:0]
			}
		}
		// Check if a sync ended at this position. If yes, start
		// collecting post-sync dibits from the NEXT iteration.
		// Only honour outbound matches — inbound (MS → BS) bursts
		// don't carry the CC announcement payloads the state
		// machine locks on.
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx].Index == absPos {
			if !p.matchScratch[matchIdx].Inbound {
				p.remaining = postSyncDibits
				p.frame = p.frame[:0]
			}
			matchIdx++
		}
	}
	return baseIdx + len(dibits)
}

// tryIngestFrame slices the collected post-sync dibits into LICH +
// CAC bits, parses each, and forwards the result to IngestFrame.
// Drops the frame silently on any parse / CRC error — the next
// FSW match anchors the stream again.
func (c *ControlChannel) tryIngestFrame(frame []uint8) {
	if len(frame) != postSyncDibits {
		return
	}
	// LICH: 8 wire dibits → 16 wire bits → DecodeLICHWire → info
	// byte → ParseLICH.
	lichBits := framing.DibitsToBits(frame[0:8])
	lichByte, _ := DecodeLICHWire(lichBits)
	lich := ParseLICH(lichByte)

	// CAC: 44 dibits → 88 information bits → 11 bytes → ParseCAC.
	// Frame offsets: 8..40 is the 32-dibit SACCH (skipped here;
	// SACCH carries channel signalling decoded via the K=5
	// Viterbi chain in sacch.go but isn't required for the
	// cc.locked path). Offsets 40..84 are the first 44 dibits
	// of the 144-dibit Info field.
	cacBits := framing.DibitsToBits(frame[40:84])
	cacBytes := framing.PackBitsMSB(cacBits)
	if len(cacBytes) < 11 {
		return
	}
	cac, _ := ParseCAC(cacBytes[:11])
	// CRC mismatch is non-fatal — IngestFrame will simply not
	// publish cc.locked unless the message type is one it
	// recognises (RCCHSITEINFO / RCCHCCH). For real-air signals
	// the CAC FEC layer (a documented follow-up) is required to
	// pull valid info bits out of the 288-wire-bit Info field.
	c.IngestFrame(lich, &cac)
}

// Reset clears the Process adapter's sync-detection + partial-frame
// state. The receiver-side Reset rewinds the absolute dibit index;
// callers that need to clear stream state on retune call this.
func (s *SyncDetector) Reset() {
	for i := range s.hist {
		s.hist[i] = 0
	}
	s.primed = 0
	s.pos = 0
}
