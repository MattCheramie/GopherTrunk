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

	// SB-burst (synchronisation burst) path: a separate detector for
	// the synchronisation training sequence plus a rolling dibit buffer
	// with look-back (to the BSCH before the STS) and look-ahead (to the
	// BNCH after it). See processSB.
	stsDet     *SyncDetector
	stsScratch []int
	buf        []uint8 // rolling dibit window
	bufBase    int     // absolute dibit index of buf[0]
	pendingSTS []int   // STS leading indices awaiting look-ahead
}

// Synchronisation downlink burst (SB) geometry in dibits, per ETSI
// EN 300 392-2 §9.4.4 / osmo-tetra: the burst lays out
// freq-correction → BSCH(60) → STS(19) → broadcast(15) → BNCH(108).
// Relative to the STS leading dibit L: the BSCH block is [L-60, L) and
// the BNCH (SYSINFO) block is [L+34, L+142).
const (
	stsDibits         = 19  // SyncTrainingDibits length
	sbBSCHDibits      = 60  // BSCH block 1 (120 type-5 bits)
	sbBroadcastDibits = 15  // broadcast block between STS and BNCH
	sbBNCHDibits      = 108 // BNCH block 2 (216 type-5 bits, SCH/HD-coded)
	// sbTrimMargin is how many trailing dibits the rolling buffer keeps
	// so a future STS hit near the end still has its 60-dibit look-back.
	sbTrimMargin = 256
)

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
			stsDet:    NewSyncDetector(SyncTrainingDibits(), 3),
			pduDibits: make([]uint8, 0, channelDibitCount(ChannelSCHF)),
		}
	}
	p := c.proc

	// SB-burst path: auto-learn the colour code from the BSCH and decode
	// the BNCH SYSINFO. Only meaningful with channel coding on (real
	// bursts); skipped for the synthetic raw-PDU fixtures.
	if mode == ChannelCodingOn {
		c.processSB(p, dibits, baseIdx)
	}

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

// rotateDibits returns dibits cyclically rotated by rot (mod 4) — the
// π/4-DQPSK differential decode has a constant 90°/dibit ambiguity, so
// each block is tried under all four rotations.
func rotateDibits(dibits []uint8, rot uint8) []uint8 {
	out := make([]uint8, len(dibits))
	for i, d := range dibits {
		out[i] = (d + rot) & 3
	}
	return out
}

// processSB maintains the rolling dibit buffer, detects the
// synchronisation training sequence (STS), and once a buffer covers the
// whole synchronisation burst around an STS hit, decodes the BSCH (to
// learn the colour code) and the BNCH SYSINFO. The BSCH is scrambled
// with colour code 0, so this works on a cold receiver with no
// configured colour code (issue #553).
func (c *ControlChannel) processSB(p *processState, dibits []uint8, baseIdx int) {
	// Align the buffer base on the first call.
	if len(p.buf) == 0 {
		p.bufBase = baseIdx
	}
	p.buf = append(p.buf, dibits...)

	var hits []int
	hits, _ = p.stsDet.Process(p.stsScratch[:0], dibits, baseIdx)
	for _, trailing := range hits {
		p.pendingSTS = append(p.pendingSTS, trailing-(stsDibits-1)) // leading index L
	}

	bufEnd := p.bufBase + len(p.buf)
	kept := make([]int, 0, len(p.pendingSTS))
	for _, L := range p.pendingSTS {
		needStart := L - sbBSCHDibits
		needEnd := L + stsDibits + sbBroadcastDibits + sbBNCHDibits
		if needStart < p.bufBase {
			continue // look-back already trimmed; give up on this hit
		}
		if needEnd > bufEnd {
			kept = append(kept, L) // not enough look-ahead yet
			continue
		}
		c.decodeSB(p, L)
	}
	p.pendingSTS = kept

	// Trim the buffer, keeping the trailing margin (so future STS hits
	// retain look-back) and any unresolved pending hit's look-back.
	keepFrom := bufEnd - sbTrimMargin
	for _, L := range p.pendingSTS {
		if ns := L - sbBSCHDibits; ns < keepFrom {
			keepFrom = ns
		}
	}
	if keepFrom > p.bufBase {
		drop := keepFrom - p.bufBase
		if drop > len(p.buf) {
			drop = len(p.buf)
		}
		p.buf = append(p.buf[:0], p.buf[drop:]...)
		p.bufBase += drop
	}
}

// decodeSB decodes the BSCH and BNCH of the synchronisation burst whose
// STS leads at absolute dibit index L. It learns the colour code from
// the BSCH SYNC PDU, locks on the cell identity, then descrambles the
// BNCH SYSINFO with the learned code to fill the location area.
func (c *ControlChannel) decodeSB(p *processState, L int) {
	block := func(start, n int) []uint8 {
		s := start - p.bufBase
		return p.buf[s : s+n]
	}

	// BSCH (block 1), scrambled with colour code 0. Try all rotations.
	bsch := block(L-sbBSCHDibits, sbBSCHDibits)
	var sync SyncPDU
	found := false
	for rot := uint8(0); rot < 4 && !found; rot++ {
		bits := TetraDibitsToBits(rotateDibits(bsch, rot))
		if recovered, ok := DecodeBSCH(bits); ok {
			if s, ok := ParseSyncPDU(recovered); ok {
				sync, found = s, true
			}
		}
	}
	if !found {
		return
	}
	c.LearnColourCode(sync.ExtendedColourCode())

	// Build a single lock state: cell identity (MCC/MNC) from the BSCH
	// SYNC, location area from the BNCH SYSINFO. Firing one maybeLock
	// with MCC/MNC always populated avoids the "preserve learned
	// MCC/MNC" merge in maybeLock clobbering the location area.
	ls := LockState{FrequencyHz: c.freqHz, MCC: sync.MCC, MNC: sync.MNC}

	// BNCH (block 2): SCH/HD-coded SYSINFO scrambled with the learned
	// colour code. Fills the location area (and MCC/MNC if it carries
	// them).
	colour := c.ColourCode()
	bnch := block(L+stsDibits+sbBroadcastDibits, sbBNCHDibits)
	for rot := uint8(0); rot < 4; rot++ {
		bits := TetraDibitsToBits(rotateDibits(bnch, rot))
		recovered, ok := DecodeSCHHD(bits, colour)
		if !ok {
			continue
		}
		if pdu, err := ParsePDU(framing.PackBitsMSB(recovered)); err == nil {
			if sb, ok := pdu.AsSystemBroadcast(); ok {
				ls.LocationArea = sb.LocationArea
				if sb.MCC != 0 {
					ls.MCC, ls.MNC = sb.MCC, sb.MNC
				}
			}
			break
		}
	}
	c.maybeLock(ls)
}

// dispatchSlice turns the collected post-sync dibit slice into a
// PDU and forwards it to Ingest. Under ChannelCodingOff the
// slice is interpreted as raw type-1 bits; under ChannelCodingOn
// the configured ChannelType drives a full type-5 → type-1 decode
// chain via the per-channel helpers in channel_coding.go.
func (c *ControlChannel) dispatchSlice(slice []uint8, mode ChannelCodingMode, channel ChannelType, colour uint32) {
	// TETRA uses the π/4-DQPSK Gray bit↔dibit convention, not the
	// linear framing.DibitsToBits used by the C4FM protocols.
	bits := TetraDibitsToBits(slice)
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
