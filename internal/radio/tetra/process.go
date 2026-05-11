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
// each 38-dibit training-sequence match under ChannelCodingOff —
// 48 dibits = 96 bits, large enough to cover SystemBroadcast /
// VoiceGrant / Release PDUs read raw from a synthesized fixture.
// Under ChannelCodingOn the slice size is channel-specific (see
// channelDibitCount).
const pduDibitCount = 48

// channelDibitCount returns the number of dibits the Process
// adapter collects after sync detection for each ChannelType
// under ChannelCodingOn. Each value is the type-5 bit count of
// the channel (per EN 300 392-2 §8.3.1) divided by 2 (one dibit
// = 2 bits).
func channelDibitCount(ch ChannelType) int {
	switch ch {
	case ChannelAACH:
		return 15 // 30 type-5 bits / 2
	case ChannelBSCH:
		return 60 // 120 / 2
	case ChannelSCHHD:
		return 108 // 216 / 2
	case ChannelSCHHU:
		return 84 // 168 / 2
	case ChannelSCHF:
		return 216 // 432 / 2
	default:
		return 108 // SCH/HD default
	}
}

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
	c.mu.Lock()
	mode := c.channelCoding
	channel := c.channelType
	colour := c.colourCode
	c.mu.Unlock()

	if c.proc == nil {
		c.proc = &processState{
			det:       NewSyncDetector(NormalSyncDibits(), 3),
			pduDibits: make([]uint8, 0, channelDibitCount(ChannelSCHF)),
		}
	}
	p := c.proc

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], dibits, baseIdx)
	matchIdx := 0

	sliceCount := pduDibitCount
	if mode == ChannelCodingOn {
		sliceCount = channelDibitCount(channel)
	}

	for i, d := range dibits {
		absPos := baseIdx + i
		if p.remaining > 0 {
			p.pduDibits = append(p.pduDibits, d)
			p.remaining--
			if p.remaining == 0 {
				c.dispatchSlice(p.pduDibits, mode, channel, colour)
				p.pduDibits = p.pduDibits[:0]
			}
		}
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == absPos {
			p.remaining = sliceCount
			p.pduDibits = p.pduDibits[:0]
			matchIdx++
		}
	}
	return baseIdx + len(dibits)
}

// dispatchSlice turns the collected post-sync dibit slice into a
// PDU and forwards it to Ingest. Under ChannelCodingOff the
// slice is interpreted as raw type-1 bits; under ChannelCodingOn
// the configured ChannelType drives a full type-5 → type-1 decode
// chain via the per-channel helpers in channel_coding.go.
func (c *ControlChannel) dispatchSlice(slice []uint8, mode ChannelCodingMode, channel ChannelType, colour uint32) {
	bits := framing.DibitsToBits(slice)
	var info []byte
	switch {
	case mode != ChannelCodingOn:
		info = framing.PackBitsMSB(bits)
	case channel == ChannelAACH:
		recovered, errs := DecodeAACH(bits, colour)
		if errs < 0 {
			return
		}
		info = framing.PackBitsMSB(recovered)
	case channel == ChannelBSCH:
		recovered, ok := DecodeBSCH(bits)
		if !ok {
			return
		}
		info = framing.PackBitsMSB(recovered)
	case channel == ChannelSCHHU:
		recovered, ok := DecodeSCHHU(bits, colour)
		if !ok {
			return
		}
		info = framing.PackBitsMSB(recovered)
	case channel == ChannelSCHF:
		recovered, ok := DecodeSCHF(bits, colour)
		if !ok {
			return
		}
		info = framing.PackBitsMSB(recovered)
	default: // ChannelSCHHD
		recovered, ok := DecodeSCHHD(bits, colour)
		if !ok {
			return
		}
		info = framing.PackBitsMSB(recovered)
	}
	if len(info) >= 1 {
		if pdu, err := ParsePDU(info); err == nil {
			c.Ingest(pdu)
		}
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
