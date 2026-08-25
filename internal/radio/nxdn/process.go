package nxdn

import (
	"encoding/binary"

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
	// frameSoft accumulates the per-dibit soft LLRs (2 per dibit,
	// MSB then LSB) parallel to frame on the ProcessSoft path; empty
	// on the hard Process path.
	frameSoft []float32
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
// puncture / interleave content this simplified path doesn't
// model; the spec-correct ViterbiSpec path covers them.
const postSyncDibitsViterbi = 8 + 32 + 92

// postSyncDibitsViterbiSpec is the count of dibits the adapter
// collects after the 8-dibit FSW match when SetViterbiMode is
// ViterbiSpec: 8 LICH + 150 CAC = 158 dibits. The 150 CAC dibits =
// 300 channel bits run through the full NXDN-TS-1-A §4.5.1.1
// outbound chain (deinterleave 25×12 + depuncture 50/350 + K=5
// R=½ Viterbi + 16-bit CRC verify + tail strip), recovering the
// 155-bit info block (8 SR + 144 L3 + 3 Null). The RCCH outbound
// frame layout is FSW(20) + LICH(16) + CAC(300) + E(24) + Post(24)
// per §4.6, so the post-CAC 48 bits aren't read here; an upstream
// PR can extend this if E / Post become useful.
const postSyncDibitsViterbiSpec = 8 + 150

// cacViterbiInfoBits is the number of source bits the K=5 ½-rate
// Viterbi decode recovers from the 92 encoded CAC dibits under
// ViterbiOn: 88 CAC information bits + 4 zero tail bits to flush
// the encoder.
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
	switch c.viterbiMode {
	case ViterbiOn:
		frameLen = postSyncDibitsViterbi
	case ViterbiSpec:
		frameLen = postSyncDibitsViterbiSpec
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

// ProcessSoft is the soft-decision sibling of Process (opt-in via
// nxdn_soft_decision): identical FSW detection and frame slicing on the
// hard dibits, but each collected frame also carries the per-dibit soft
// LLRs (2 per dibit) so the ViterbiSpec CAC decode runs the true per-bit
// soft Viterbi (DecodeCACChannelSoft) instead of hard-slicing first — the
// coding gain that recovers marginal on-air CAC bursts. len(soft) must be
// 2*len(dibits) (the SoftDibitSink contract); a mismatched call falls back
// to the hard path for the whole chunk. Non-ViterbiSpec modes decode hard
// regardless (their legacy fixture layouts carry no benefit).
func (c *ControlChannel) ProcessSoft(dibits []uint8, soft []float32, baseIdx int) int {
	if len(soft) != 2*len(dibits) {
		return c.Process(dibits, baseIdx)
	}
	if c.proc == nil {
		c.proc = &processState{
			det:   NewSyncDetector([][]uint8{FSWDibitsOutbound}, 1),
			frame: make([]uint8, 0, postSyncDibitsViterbi),
		}
	}
	p := c.proc
	frameLen := postSyncDibits
	switch c.viterbiMode {
	case ViterbiOn:
		frameLen = postSyncDibitsViterbi
	case ViterbiSpec:
		frameLen = postSyncDibitsViterbiSpec
	}

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], dibits, baseIdx)
	matchIdx := 0

	for i, d := range dibits {
		absPos := baseIdx + i
		if p.remaining > 0 {
			p.frame = append(p.frame, d)
			p.frameSoft = append(p.frameSoft, soft[2*i], soft[2*i+1])
			p.remaining--
			if p.remaining == 0 {
				c.tryIngestFrameSoft(p.frame, p.frameSoft)
				p.frame = p.frame[:0]
				p.frameSoft = p.frameSoft[:0]
			}
		}
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx].Index == absPos {
			if !p.matchScratch[matchIdx].Inbound {
				p.remaining = frameLen
				p.frame = p.frame[:0]
				p.frameSoft = p.frameSoft[:0]
			}
			matchIdx++
		}
	}
	return baseIdx + len(dibits)
}

// tryIngestFrameSoft is tryIngestFrame with the ViterbiSpec CAC decode
// running soft (DecodeCACChannelSoft over the frame's per-bit LLRs). The
// LICH stays a hard decode — its (16, 8) wire code is trivially strong at
// any SNR where the FSW correlates — and non-ViterbiSpec modes fall back to
// the hard path entirely.
func (c *ControlChannel) tryIngestFrameSoft(frame []uint8, soft []float32) {
	if c.viterbiMode != ViterbiSpec || len(frame) != postSyncDibitsViterbiSpec ||
		len(soft) != 2*len(frame) {
		c.tryIngestFrame(frame)
		return
	}
	if len(frame) < 8 {
		return
	}
	lichBits := framing.DibitsToBits(frame[0:8])
	lichByte, _ := DecodeLICHWire(lichBits)
	lich := ParseLICH(lichByte)

	// CAC LLRs: everything after the 8 LICH dibits (150 dibits = 300 LLRs).
	info, ok := DecodeCACChannelSoft(soft[8*2:])
	if !ok {
		return
	}
	block, ok := packCACBlockFromInfo(info)
	if !ok {
		return
	}
	cac, err := ParseCAC(block)
	if err != nil {
		return
	}
	c.IngestFrame(lich, &cac)
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
//
//   - ViterbiSpec: frame is 158 dibits (8 LICH + 150 CAC) per the
//     §4.6 RCCH outbound layout. Offsets 8..158 are the 150 CAC
//     dibits = 300 channel bits. DecodeCACChannel runs the
//     spec's full chain (deinterleave 25×12 + depuncture + K=5
//     Viterbi + 16-bit CRC verify) and returns 155 info bits. The
//     8-bit SR header is dropped; the next 88 bits of the L3
//     payload feed the existing ParseCAC. CRC-fail drops the
//     frame silently.
func (c *ControlChannel) extractCACBytes(frame []uint8) ([]byte, bool) {
	switch c.viterbiMode {
	case ViterbiSpec:
		if len(frame) != postSyncDibitsViterbiSpec {
			return nil, false
		}
		channelBits := framing.DibitsToBits(frame[8:postSyncDibitsViterbiSpec])
		if len(channelBits) != CACChannelBits {
			return nil, false
		}
		info, ok := DecodeCACChannel(channelBits)
		if !ok {
			return nil, false
		}
		return packCACBlockFromInfo(info)
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

// packCACBlockFromInfo repacks a CRC-validated 155-bit §4.5.1.1 info block
// into the 11-byte block the legacy ParseCAC path expects. Layout per step
// ①: 8 bits SR ‖ 144 bits L3 Data ‖ 3 Null. The first 8 L3 bits carry the
// RCCH message type; the next 64 carry the existing CAC payload. Drop SR,
// pack the 72-bit L3 prefix into 9 bytes, and synthesize the trailing
// 16-bit inner CRC — the spec's outer CRC has already validated the whole
// info block, so the inner-CRC sentinel is a no-op here. binary.BigEndian
// keeps the layout identical to AssembleCAC. Shared by the hard
// (extractCACBytes) and soft (tryIngestFrameSoft) ViterbiSpec paths.
func packCACBlockFromInfo(info []byte) ([]byte, bool) {
	if len(info) < 8+72 {
		return nil, false
	}
	l3 := framing.PackBitsMSB(info[8 : 8+72])
	if len(l3) < 9 {
		return nil, false
	}
	block := make([]byte, 11)
	copy(block, l3[:9])
	binary.BigEndian.PutUint16(block[9:11], framing.CRCCCITT(block[:9]))
	return block, true
}

// ResyncReset drops the Process adapter's sync-detection + partial-frame
// state so a receiver-side Reset (which restarts the dibit index at 0) can
// reacquire cleanly — a stale mid-frame countdown would otherwise splice
// pre-reset dibits onto post-reset ones into one garbage frame. Mirrors the
// TETRA / DMR Tier III ControlChannels' ResyncReset; call from the pipeline
// whenever the receiver is reset mid-stream.
//
// Precondition: called on the same goroutine as Process (from the pipeline,
// after rx.Process returns), so the proc swap never races a Process call.
func (c *ControlChannel) ResyncReset() {
	c.proc = nil
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
