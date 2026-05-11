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
// the 8-dibit FSW match when SetViterbiMode is ViterbiOff: 8 LICH
// wire + 32 SACCH (skipped) + 44 CAC info dibits = 84 dibits. The
// remaining 100 dibits of the 144-dibit Info field carry FEC
// redundancy or other content the no-FEC path doesn't read. This
// mode drives cc.locked in test fixtures where the CAC bits are
// placed directly on the wire; on real on-air signals the CAC CRC
// almost always fails and the adapter silently drops the frame.
const postSyncDibits = 84

// postSyncDibitsViterbi is the count of dibits the adapter collects
// after the 8-dibit FSW match when SetViterbiMode is ViterbiOn: 8
// LICH + 32 SACCH (skipped) + 92 CAC-encoded dibits = 132 dibits.
// The 92 CAC-encoded dibits = 184 wire bits = (88 CAC info + 4 tail
// bits) × 2 — the K=5 ½-rate convolutional output. The remaining
// 52 dibits of the 144-dibit Info field carry per-protocol
// puncture / interleave content the public references don't fully
// document; those layers are documented follow-ups.
const postSyncDibitsViterbi = 8 + 32 + 92

// cacViterbiInfoBits is the number of source bits the K=5 ½-rate
// Viterbi decode recovers from the 92 encoded CAC dibits: 88 CAC
// information bits + 4 zero tail bits to flush the encoder.
const cacViterbiInfoBits = 92

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
			frame: make([]uint8, 0, postSyncDibitsViterbi),
		}
	}
	p := c.proc
	frameLen := postSyncDibits
	if c.viterbiMode == ViterbiOn {
		frameLen = postSyncDibitsViterbi
	}

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
				p.remaining = frameLen
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
	// LICH: 8 wire dibits → 16 wire bits → DecodeLICHWire → info
	// byte → ParseLICH. Layout is the same in both Viterbi modes.
	if len(frame) < 8 {
		return
	}
	lichBits := framing.DibitsToBits(frame[0:8])
	lichByte, _ := DecodeLICHWire(lichBits)
	lich := ParseLICH(lichByte)

	cacBytes, ok := c.extractCACBytes(frame)
	if !ok {
		return
	}
	cac, err := ParseCAC(cacBytes)
	if err != nil {
		// CRC-CCITT-16 mismatch — drop the frame silently.
		// ViterbiOff: the wire bits are read raw, so any noise
		// on the CAC slot fails the CRC. ViterbiOn: the K=5
		// decode recovers info bits but the per-protocol
		// interleave / puncture isn't reversed, so on-air
		// frames still typically fail; clean synthesized
		// streams (or a future PR that adds the interleave
		// reversal) pass.
		return
	}
	c.IngestFrame(lich, &cac)
}

// extractCACBytes pulls the 11 CAC bytes (88 information bits +
// CRC) out of the post-sync frame. The slice layout depends on
// ViterbiMode:
//
//   - ViterbiOff: frame is 84 dibits total. Offsets 8..40 are the
//     32-dibit SACCH (skipped). Offsets 40..84 are the first 44
//     dibits of the Info field; their 88 wire bits ARE the CAC
//     information bits (no FEC reversal).
//
//   - ViterbiOn: frame is 132 dibits total. Offsets 8..40 are
//     SACCH (skipped). Offsets 40..132 are the first 92 dibits
//     of the Info field = 184 wire bits = K=5 ½-rate-encoded
//     output. ViterbiK5 recovers 92 source bits; the first 88
//     are the CAC info bits.
func (c *ControlChannel) extractCACBytes(frame []uint8) ([]byte, bool) {
	switch c.viterbiMode {
	case ViterbiOn:
		if len(frame) != postSyncDibitsViterbi {
			return nil, false
		}
		channelBits := framing.DibitsToBits(frame[40:postSyncDibitsViterbi])
		if len(channelBits) != 2*cacViterbiInfoBits {
			return nil, false
		}
		info, _ := framing.ViterbiK5(channelBits, cacViterbiInfoBits)
		// Drop the 4 trailing tail bits; the first 88 source
		// bits are the CAC information field.
		cacBytes := framing.PackBitsMSB(info[:88])
		if len(cacBytes) < 11 {
			return nil, false
		}
		return cacBytes[:11], true
	default:
		if len(frame) != postSyncDibits {
			return nil, false
		}
		cacBits := framing.DibitsToBits(frame[40:postSyncDibits])
		cacBytes := framing.PackBitsMSB(cacBits)
		if len(cacBytes) < 11 {
			return nil, false
		}
		return cacBytes[:11], true
	}
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
