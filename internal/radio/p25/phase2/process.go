package phase2

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// processState is the cross-call dibit buffering + sync-detection
// state the Process adapter holds. Lazily initialised.
type processState struct {
	det          *SyncDetector
	remaining    int
	macDibits    []uint8
	matchScratch []int
}

// macPDUDibits is the count of dibits the adapter collects after
// each 20-dibit outbound sync match when SetTrellisMode is
// TrellisOff. A MAC PDU after FEC removal is 18 bytes = 144 bits =
// 72 dibits (1 opcode + up to 17 payload bytes). This mode reads
// the 72 information dibits straight from the wire — works for
// test fixtures + clean signals where the MAC bits aren't
// channel-coded.
const macPDUDibits = 72

// macPDUDibitsTrellis is the count of dibits the adapter collects
// after each sync match when SetTrellisMode is TrellisOn. The
// 4-state ½-rate trellis encoder produces 1 channel dibit pair
// per input dibit (2 channel dibits) plus 1 finisher transition,
// so 72 info dibits → 2 × (72 + 1) = 146 channel dibits.
const macPDUDibitsTrellis = 146

// Process consumes a window of raw dibits from the Phase 2
// receiver (the IQ → H-DQPSK dibit chain in
// internal/radio/p25/phase2/receiver/), runs the 20-dibit
// outbound sync detector, slices the following 72-dibit MAC PDU
// out of the stream, parses it via ParseMACPDU, and forwards the
// result to Ingest.
//
// baseIdx is the absolute dibit index of dibits[0]. The adapter's
// internal countdown survives across Process calls so a sync
// match in one chunk and the MAC PDU payload in the next still
// decode cleanly.
//
// Returns baseIdx + len(dibits) to match the ControlChannel.Process
// contracts shared across protocols.
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det:       NewSyncDetector(OutboundSyncDibits(), 2),
			macDibits: make([]uint8, 0, macPDUDibitsTrellis),
		}
	}
	p := c.proc
	c.mu.Lock()
	mode := c.trellisMode
	c.mu.Unlock()
	frameLen := macPDUDibits
	if mode == TrellisOn {
		frameLen = macPDUDibitsTrellis
	}

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], dibits, baseIdx)
	matchIdx := 0

	for i, d := range dibits {
		absPos := baseIdx + i
		if p.remaining > 0 {
			p.macDibits = append(p.macDibits, d)
			p.remaining--
			if p.remaining == 0 {
				c.tryIngestMACPDU(p.macDibits, mode)
				p.macDibits = p.macDibits[:0]
			}
		}
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == absPos {
			p.remaining = frameLen
			p.macDibits = p.macDibits[:0]
			matchIdx++
		}
	}
	return baseIdx + len(dibits)
}

// tryIngestMACPDU recovers an 18-byte MAC PDU from the collected
// post-sync dibits. The dibit slice layout depends on TrellisMode:
//
//   - TrellisOff: macDibits is exactly macPDUDibits (72) raw
//     dibits whose bits ARE the MAC PDU information bits.
//
//   - TrellisOn: macDibits is exactly macPDUDibitsTrellis (146)
//     channel dibits = the trellis-encoded form of the 72 info
//     dibits + 1 finisher transition. DecodeP25Trellis recovers
//     the 72 information dibits.
func (c *ControlChannel) tryIngestMACPDU(macDibits []uint8, mode TrellisMode) {
	var infoDibits []uint8
	switch mode {
	case TrellisOn:
		if len(macDibits) != macPDUDibitsTrellis {
			return
		}
		decoded, _ := framing.DecodeP25Trellis(macDibits)
		if len(decoded) != macPDUDibits {
			return
		}
		infoDibits = decoded
	default:
		if len(macDibits) != macPDUDibits {
			return
		}
		infoDibits = macDibits
	}
	bits := framing.DibitsToBits(infoDibits)
	info := framing.PackBitsMSB(bits)
	if len(info) < 18 {
		return
	}
	if pdu, err := ParseMACPDU(info[:18]); err == nil {
		c.Ingest(pdu)
	}
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
