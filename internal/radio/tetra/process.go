package tetra

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// processState is the cross-call dibit buffering + sync-detection
// state the Process adapter holds. Lazily initialised.
type processState struct {
	det          *SyncDetector
	remaining    int
	pduDibits    []uint8
	matchScratch []int
}

// pduDibitCount is the count of dibits the adapter collects after
// each 38-dibit training-sequence match. A TETRA VoiceGrant (CMCE
// D-CONNECT) needs 1 header byte + 11 payload bytes = 12 bytes =
// 96 bits = 48 dibits — large enough to cover SystemBroadcast +
// VoiceGrant + Release. The RCPC / RM FEC + interleaving across
// the full slot aren't reversed here; the adapter reads the
// information bits straight from the wire, which works on test
// fixtures but typically fails on captured TETRA traffic.
const pduDibitCount = 48

// Process consumes a window of raw dibits from the TETRA receiver
// (the IQ → π/4-DQPSK dibit chain in internal/radio/tetra/receiver/),
// runs the 38-dibit normal training-sequence detector, slices the
// following 48-dibit PDU out of the stream, parses it via ParsePDU,
// and forwards the result to Ingest.
//
// baseIdx is the absolute dibit index of dibits[0]. The adapter's
// internal countdown survives across Process calls so a sync match
// in one chunk and the PDU payload in the next still decode cleanly.
//
// Returns baseIdx + len(dibits) to match the ControlChannel.Process
// contracts shared across protocols.
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det:       NewSyncDetector(NormalSyncDibits(), 3),
			pduDibits: make([]uint8, 0, pduDibitCount),
		}
	}
	p := c.proc

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], dibits, baseIdx)
	matchIdx := 0

	for i, d := range dibits {
		absPos := baseIdx + i
		if p.remaining > 0 {
			p.pduDibits = append(p.pduDibits, d)
			p.remaining--
			if p.remaining == 0 {
				bits := framing.DibitsToBits(p.pduDibits)
				info := framing.PackBitsMSB(bits)
				if len(info) >= 1 {
					if pdu, err := ParsePDU(info); err == nil {
						c.Ingest(pdu)
					}
				}
				p.pduDibits = p.pduDibits[:0]
			}
		}
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == absPos {
			p.remaining = pduDibitCount
			p.pduDibits = p.pduDibits[:0]
			matchIdx++
		}
	}
	return baseIdx + len(dibits)
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
