package phase2

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// processState is the cross-call dibit buffering + sync-detection
// state the Process adapter holds. Lazily initialised.
type processState struct {
	det          *SyncDetector
	remaining    int
	macDibits    []uint8
	matchScratch []int
	// lastNoHitsAt throttles the "no sync hits" debug log so the
	// chunk-rate emission doesn't flood at debug level. See Process.
	lastNoHitsAt time.Time
}

// noHitsThrottle bounds how often Process emits its "no sync hits"
// debug log when the sync detector is finding nothing. Issue #275
// surfaced because that state produced zero logs at all — operators
// couldn't tell whether the IQ pipeline was alive but unsynchronized
// or wholly silent.
const noHitsThrottle = 2 * time.Second

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
	rsMode := c.rsMode
	interleaveMode := c.interleaveMode
	scramblerMode := c.scramblerMode
	scramblerSeed := c.scramblerSeed
	scramblerOffset := c.scramblerOffset
	c.mu.Unlock()
	frameLen := macPDUDibits
	if mode == TrellisOn {
		frameLen = macPDUDibitsTrellis
	}

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], dibits, baseIdx)
	matchIdx := 0

	c.mu.Lock()
	alreadyLocked := c.locked
	c.mu.Unlock()
	if len(p.matchScratch) == 0 && p.remaining == 0 && len(dibits) > 0 && !alreadyLocked {
		now := c.now()
		if now.Sub(p.lastNoHitsAt) >= noHitsThrottle {
			c.log.Debug("p25/phase2: no sync hits in chunk",
				"system", c.systemName, "freq_hz", c.freqHz, "dibits", len(dibits))
			p.lastNoHitsAt = now
		}
	}

	for i, d := range dibits {
		absPos := baseIdx + i
		if p.remaining > 0 {
			p.macDibits = append(p.macDibits, d)
			p.remaining--
			if p.remaining == 0 {
				c.tryIngestMACPDU(p.macDibits, mode, rsMode, interleaveMode, scramblerMode, scramblerSeed, scramblerOffset)
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
// post-sync dibits and, on success, forwards it to Ingest. It is the
// flat sync-window path; the FEC chain itself lives in the pure
// decodeMACPDUDibits helper, which the superframe-structured path
// (superframe_ingest.go) reuses.
func (c *ControlChannel) tryIngestMACPDU(macDibits []uint8, mode TrellisMode, rsMode RSMode, interleaveMode InterleaveMode, scramblerMode ScramblerMode, scramblerSeed uint64, scramblerOffset int) {
	if pdu, ok := decodeMACPDUDibits(macDibits, mode, rsMode, interleaveMode, scramblerMode, scramblerSeed, scramblerOffset); ok {
		c.Ingest(pdu)
	}
}

// decodeMACPDUDibits runs the full MAC-PDU recovery chain over a
// collected dibit window and returns the parsed PDU. It is a pure
// function (no ControlChannel state) so both the flat Process adapter
// and the superframe-structured IngestSuperframe can share it. The
// dibit slice layout depends on TrellisMode:
//
//   - TrellisOff: macDibits is exactly macPDUDibits (72) raw
//     dibits whose bits ARE the MAC PDU information bits.
//
//   - TrellisOn: macDibits is exactly macPDUDibitsTrellis (146)
//     channel dibits = the trellis-encoded form of the 72 info
//     dibits + 1 finisher transition. DecodeP25Trellis recovers
//     the 72 information dibits.
//
// Pipeline order (issue #915). Per TIA-102.BBAC-1 §7.2.5 the PN44
// scrambler wraps the CODED channel bits "between demodulation and
// FEC", so the descramble runs FIRST — on the raw channel dibits —
// and only then does the FEC chain (deinterleave, then trellis) run.
// This matches the reference decoders (SDRtrunk's Timeslot.xor() and
// OP25's handle_packet() both XOR the mask over the coded burst before
// deinterleave/trellis/RS). GopherTrunk previously descrambled the
// recovered 144 INFORMATION bits AFTER the trellis; because the trellis
// code does not commute with the XOR, that domain can never satisfy the
// outer RS(24, 16, 9) check on a genuinely scrambled burst, which is
// why real Phase 2 traffic decoded with mac_rs_valid = 0 regardless of
// seed (issue #915, Finding B).
//
// When scramblerMode is ScramblerOn the channel bits are XORed with the
// PN44 sequence starting at scramblerOffset — the caller passes the
// slot's channel-bit offset into the continuous 4320-bit superframe
// sequence (slotChannelPN44Offset); the sequence runs continuously
// across the 360 ms superframe rather than restarting per burst.
//
// When scramblerMode is ScramblerProbe each of the 12 slot channel-bit
// phases (slotChannelPN44Offset) is tried and the first candidate whose
// outer RS(24, 16, 9) syndromes are zero is accepted — the self-aligning
// form for a receiver without superframe synchronization, where the
// sub-frame's slot within the superframe (hence its phase into the
// continuous 4320-bit sequence) is not yet pinned. The RS gate makes the
// sweep safe: a wrong offset packs into random bytes that satisfy the
// 8-parity-symbol syndrome with probability ≈2^-48, so the sweep never
// accepts garbage. ScramblerProbe requires an enabled rsMode (RSOn or
// RSCorrect); otherwise it degrades to ScramblerOn at scramblerOffset.
//
// When rsMode is RSOn the (descrambled, FEC-decoded) 18-byte MAC PDU is
// re-grouped into 24 hex symbols and verified against the RS(24, 16, 9)
// outer code per TIA-102.BAAA-A §5.9; PDUs whose syndromes are non-zero
// are rejected. When rsMode is RSCorrect the same layer instead corrects
// up to t=4 symbol errors (framing.DecodeRS24_16) and accepts the repaired
// PDU if it carries a recognised opcode — the weak-frame recovery for
// issue #915. See gateMACPDURS.
func decodeMACPDUDibits(macDibits []uint8, mode TrellisMode, rsMode RSMode, interleaveMode InterleaveMode, scramblerMode ScramblerMode, scramblerSeed uint64, scramblerOffset int) (MACPDU, bool) {
	switch scramblerMode {
	case ScramblerProbe:
		if !rsMode.Enabled() {
			// Without the RS gate there is no way to tell a correct
			// offset from a wrong one, so fall back to the fixed offset.
			return descrambleChannelAndParse(macDibits, mode, interleaveMode, rsMode, scramblerSeed, scramblerOffset)
		}
		for slot := 0; slot < SubframesPerSuperframe; slot++ {
			off := slotChannelPN44Offset(slot)
			// Pass the caller's rsMode through so RSCorrect repairs each
			// candidate (up to t=4) rather than only verifying it — the
			// known-opcode gate inside decodeMACChannelAndParse keeps the
			// sweep from accepting a wrong-phase garbage window.
			if pdu, ok := descrambleChannelAndParse(macDibits, mode, interleaveMode, rsMode, scramblerSeed, off); ok {
				return pdu, true
			}
		}
		return MACPDU{}, false
	case ScramblerOn:
		return descrambleChannelAndParse(macDibits, mode, interleaveMode, rsMode, scramblerSeed, scramblerOffset)
	default: // ScramblerOff
		return decodeMACChannelAndParse(macDibits, mode, interleaveMode, rsMode)
	}
}

// descrambleChannelAndParse XOR-descrambles the coded channel bits of
// macDibits at channelOffset (per TIA-102.BBAC-1 §7.2.5 — the scrambler
// domain is the channel bits, not the post-FEC info bits) and then runs
// the FEC + parse chain over the descrambled burst. The input slice is
// left untouched.
func descrambleChannelAndParse(macDibits []uint8, mode TrellisMode, interleaveMode InterleaveMode, rsMode RSMode, seed uint64, channelOffset int) (MACPDU, bool) {
	bits := framing.DibitsToBits(macDibits)
	descrambleAtOffset(bits, seed, channelOffset)
	return decodeMACChannelAndParse(framing.BitsToDibits(bits), mode, interleaveMode, rsMode)
}

// decodeMACChannelAndParse runs the FEC chain (deinterleave, then
// trellis) over channel dibits, re-groups the recovered 144 information
// bits into the 18-byte MAC PDU, optionally RS-verifies, and parses.
func decodeMACChannelAndParse(macDibits []uint8, mode TrellisMode, interleaveMode InterleaveMode, rsMode RSMode) (MACPDU, bool) {
	// Undo the per-burst block interleaver before trellis decoding,
	// when enabled. DeinterleaveMACBurst returns a fresh slice so the
	// caller's buffer is untouched.
	if interleaveMode == InterleaveOn {
		macDibits = framing.DeinterleaveMACBurst(macDibits)
	}
	var infoDibits []uint8
	switch mode {
	case TrellisOn:
		if len(macDibits) != macPDUDibitsTrellis {
			return MACPDU{}, false
		}
		decoded, _ := framing.DecodeP25Trellis(macDibits)
		if len(decoded) != macPDUDibits {
			return MACPDU{}, false
		}
		infoDibits = decoded
	default:
		if len(macDibits) != macPDUDibits {
			return MACPDU{}, false
		}
		infoDibits = macDibits
	}
	info := framing.PackBitsMSB(framing.DibitsToBits(infoDibits))
	if len(info) < 18 {
		return MACPDU{}, false
	}
	pduBytes, ok := gateMACPDURS(info[:18], rsMode)
	if !ok {
		return MACPDU{}, false
	}
	if pdu, err := ParseMACPDU(pduBytes); err == nil {
		return pdu, true
	}
	return MACPDU{}, false
}

// gateMACPDURS applies the outer RS(24, 16, 9) layer to an 18-byte MAC PDU
// according to rsMode and returns the bytes to parse:
//
//   - RSOff:     the PDU passes through unchanged (permissive parse).
//   - RSOn:      the PDU is accepted only if its syndromes are already
//     zero (detection-only); any residual symbol error rejects it.
//   - RSCorrect: the PDU is run through the bounded-distance corrector
//     (up to t=4 symbol errors). A clean codeword (0 corrections)
//     passes through exactly as RSOn would accept it, so RSCorrect
//     is a strict superset of RSOn. A PDU that actually needed
//     correction is additionally gated on a recognised opcode,
//     because t=4 correction admits ~6e-4 of random windows (versus
//     2^-48 for verify) — the gate stops a wrong-phase / weak-frame
//     window from being miscorrected into a plausible-but-bogus PDU
//     that injects a bad source RID (issue #915 / #924).
//
// The returned slice is safe to parse directly; ok=false means the PDU was
// rejected.
func gateMACPDURS(pdu []byte, rsMode RSMode) ([]byte, bool) {
	switch rsMode {
	case RSOn:
		if !verifyMACPDURS(pdu) {
			return nil, false
		}
		return pdu, true
	case RSCorrect:
		corrected, nErr, ok := correctMACPDURS(pdu)
		if !ok {
			return nil, false
		}
		if nErr > 0 && !Opcode(corrected[0]).IsKnown() {
			return nil, false
		}
		return corrected, true
	default: // RSOff
		return pdu, true
	}
}

// decodeMACPDUDibitsSoftC is the soft-decision counterpart of
// decodeMACPDUDibits: it drives the true per-bit soft Viterbi
// (framing.DecodeP25TrellisSoftC) off the diagonal-frame differential samples
// carried alongside the hard dibits. It mirrors the scrambler dispatch exactly.
// soft must be the same length as macDibits; a nil / mismatched soft degrades to
// the hard path. Issue #915.
func decodeMACPDUDibitsSoftC(macDibits []uint8, soft []complex64, mode TrellisMode, rsMode RSMode, interleaveMode InterleaveMode, scramblerMode ScramblerMode, scramblerSeed uint64, scramblerOffset int) (MACPDU, bool) {
	if len(soft) != len(macDibits) || mode != TrellisOn {
		return decodeMACPDUDibits(macDibits, mode, rsMode, interleaveMode, scramblerMode, scramblerSeed, scramblerOffset)
	}
	switch scramblerMode {
	case ScramblerProbe:
		if !rsMode.Enabled() {
			return descrambleAndParseSoftC(soft, interleaveMode, rsMode, scramblerSeed, scramblerOffset)
		}
		for slot := 0; slot < SubframesPerSuperframe; slot++ {
			off := slotChannelPN44Offset(slot)
			if pdu, ok := descrambleAndParseSoftC(soft, interleaveMode, rsMode, scramblerSeed, off); ok {
				return pdu, true
			}
		}
		return MACPDU{}, false
	case ScramblerOn:
		return descrambleAndParseSoftC(soft, interleaveMode, rsMode, scramblerSeed, scramblerOffset)
	default: // ScramblerOff
		return decodeMACChannelAndParseSoftC(soft, interleaveMode, rsMode)
	}
}

// descrambleAndParseSoftC descrambles the diagonal-frame soft samples (a set
// PN44 channel bit flips the sign of the matching soft component: bit b1 → Im,
// bit b0 → Re) at channelOffset, then runs the soft FEC chain. The input slice
// is left untouched.
func descrambleAndParseSoftC(soft []complex64, interleaveMode InterleaveMode, rsMode RSMode, seed uint64, channelOffset int) (MACPDU, bool) {
	ds := make([]complex64, len(soft))
	copy(ds, soft)
	seq := make([]byte, len(soft)*2)
	descrambleAtOffset(seq, seed, channelOffset) // XOR into zeros ⇒ seq = PN44 bits
	for i := range ds {
		if seq[2*i] == 1 { // b1 (Im)
			ds[i] = complex(real(ds[i]), -imag(ds[i]))
		}
		if seq[2*i+1] == 1 { // b0 (Re)
			ds[i] = complex(-real(ds[i]), imag(ds[i]))
		}
	}
	return decodeMACChannelAndParseSoftC(ds, interleaveMode, rsMode)
}

// decodeMACChannelAndParseSoftC deinterleaves the soft samples (same permutation
// as the dibits), runs the soft Viterbi, then RS-verifies and parses.
func decodeMACChannelAndParseSoftC(soft []complex64, interleaveMode InterleaveMode, rsMode RSMode) (MACPDU, bool) {
	if len(soft) != macPDUDibitsTrellis {
		return MACPDU{}, false
	}
	if interleaveMode == InterleaveOn {
		soft = framing.DeinterleaveMACBurstC(soft)
	}
	decoded, _ := framing.DecodeP25TrellisSoftC(soft)
	if len(decoded) != macPDUDibits {
		return MACPDU{}, false
	}
	info := framing.PackBitsMSB(framing.DibitsToBits(decoded))
	if len(info) < 18 {
		return MACPDU{}, false
	}
	pduBytes, ok := gateMACPDURS(info[:18], rsMode)
	if !ok {
		return MACPDU{}, false
	}
	if pdu, err := ParseMACPDU(pduBytes); err == nil {
		return pdu, true
	}
	return MACPDU{}, false
}

// pn44SuperframeBits is the length in channel bits of one 360 ms P25
// Phase 2 superframe (12 sub-frames × 180 dibits × 2 bits) — the period
// of the PN44 scrambling sequence per TIA-102.BBAC-1 §7.2.5. Channel-bit
// offsets fold into this period.
const pn44SuperframeBits = DibitsPerSuperframe * 2

// descrambleAtOffset XOR-descrambles bits in-place with the PN44
// sequence starting at offset (folded into the 4320-bit
// superframe period).
func descrambleAtOffset(bits []byte, seed uint64, offset int) {
	s := framing.NewPN44Scrambler(seed)
	offset = offset % pn44SuperframeBits
	if offset < 0 {
		offset += pn44SuperframeBits
	}
	if offset > 0 {
		s.Advance(offset)
	}
	s.Apply(bits)
}

// verifyMACPDURS treats the 18-byte (144-bit) MAC PDU as 24 hex
// symbols and runs the RS(24, 16, 9) outer-code syndrome check per
// TIA-102.BAAA-A §5.9. Bit packing: the 144 information bits are
// read MSB-first from the byte stream, then grouped into 24 6-bit
// hex symbols where the first 6 bits form symbol 0.
func verifyMACPDURS(pdu []byte) bool {
	if len(pdu) != 18 {
		return false
	}
	var bits [144]byte
	for i := 0; i < 18; i++ {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (pdu[i] >> uint(7-j)) & 1
		}
	}
	var syms [24]byte
	for i := 0; i < 24; i++ {
		var s byte
		for j := 0; j < 6; j++ {
			s = (s << 1) | bits[i*6+j]
		}
		syms[i] = s
	}
	return framing.VerifyRS24_16(syms[:])
}

// correctMACPDURS runs bounded-distance RS(24, 16, 9) error correction over an
// 18-byte MAC PDU and returns the corrected 18 bytes plus the number of symbol
// errors repaired. It uses the same 24-symbol MSB-first grouping as
// verifyMACPDURS: the 144 information bits are read MSB-first from the byte
// stream and grouped into 24 six-bit GF(2^6) symbols, of which the first 16 are
// information and the last 8 are parity.
//
// Unlike verifyMACPDURS (accepts only syndromes==0), this repairs up to t=4
// symbol errors via framing.DecodeRS24_16 (Berlekamp-Massey + Chien + Forney,
// with a defensive re-syndrome), then re-encodes to a clean codeword and
// repacks it into 18 bytes. ok is false when the codeword lies beyond the
// correction radius. This is the weak-frame recovery path for issue #915: a MAC
// PDU that framed and descrambled at the right phase but carries a handful of
// symbol errors from marginal-SNR demod, which the detection-only gate drops.
func correctMACPDURS(pdu []byte) (corrected []byte, nErr int, ok bool) {
	if len(pdu) != 18 {
		return nil, 0, false
	}
	var bits [144]byte
	for i := 0; i < 18; i++ {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (pdu[i] >> uint(7-j)) & 1
		}
	}
	var syms [24]byte
	for i := 0; i < 24; i++ {
		var s byte
		for j := 0; j < 6; j++ {
			s = (s << 1) | bits[i*6+j]
		}
		syms[i] = s
	}
	info, nErr, err := framing.DecodeRS24_16(syms[:])
	if err != nil {
		return nil, 0, false
	}
	// Re-encode the corrected information to a full, self-consistent codeword,
	// then repack all 24 symbols back into 18 bytes — the exact inverse of the
	// grouping above, so ParseMACPDU sees the corrected information bytes in the
	// same layout verifyMACPDURS would accept.
	cw := framing.EncodeRS24_16(info)
	var outBits [144]byte
	for i := 0; i < 24; i++ {
		for j := 0; j < 6; j++ {
			outBits[i*6+j] = (cw[i] >> uint(5-j)) & 1
		}
	}
	out := make([]byte, 18)
	for i := 0; i < 18; i++ {
		var b byte
		for j := 0; j < 8; j++ {
			b = (b << 1) | outBits[i*8+j]
		}
		out[i] = b
	}
	return out, nErr, true
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
