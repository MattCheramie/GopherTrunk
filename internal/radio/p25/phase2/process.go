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
				c.tryIngestMACPDU(p.macDibits, mode, rsMode, scramblerMode, scramblerSeed, scramblerOffset)
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
//
// When scramblerMode is ScramblerOn the recovered 144 info bits
// are XORed with 144 bits of the PN44 sequence starting at
// scramblerOffset (per TIA-102.BBAC-1 §7.2.5) before RS check or
// MAC parse runs.
//
// When scramblerMode is ScramblerProbe the adapter tries each of
// the 12 spec-defined slot offsets in
// framing.PN44SlotOffsetsOutbound (Figure 7-5) and accepts the
// first candidate whose outer RS(24, 16, 9) syndromes are zero —
// the blind-probe form for receivers without superframe
// synchronization. ScramblerProbe requires rsMode == RSOn;
// without RS verification there is no way to tell which
// descrambled candidate is the true PDU, so the adapter degrades
// to ScramblerOn behaviour (offset 0).
//
// When rsMode is RSOn the (post-descramble) 18-byte MAC PDU is
// re-grouped into 24 hex symbols and verified against the
// RS(24, 16, 9) outer code per TIA-102.BAAA-A §5.9. PDUs whose
// syndromes are non-zero are dropped silently.
func (c *ControlChannel) tryIngestMACPDU(macDibits []uint8, mode TrellisMode, rsMode RSMode, scramblerMode ScramblerMode, scramblerSeed uint64, scramblerOffset int) {
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
	rawBits := framing.DibitsToBits(infoDibits)

	switch scramblerMode {
	case ScramblerProbe:
		// Probe each spec-defined slot offset; accept the first
		// whose RS(24, 16, 9) syndromes are zero. Falls through to
		// the ScramblerOn-with-offset-0 path when rsMode is RSOff
		// since there's no way to pick a winning candidate without
		// the verifier.
		if rsMode != RSOn {
			c.applyDescrambleAndIngest(rawBits, scramblerSeed, scramblerOffset, rsMode)
			return
		}
		for _, off := range framing.PN44SlotOffsetsOutbound {
			candidate := append([]byte(nil), rawBits...)
			descrambleAtOffset(candidate, scramblerSeed, off)
			info := framing.PackBitsMSB(candidate)
			if len(info) < 18 || !verifyMACPDURS(info[:18]) {
				continue
			}
			if pdu, err := ParseMACPDU(info[:18]); err == nil {
				c.Ingest(pdu)
			}
			return
		}
		return
	case ScramblerOn:
		c.applyDescrambleAndIngest(rawBits, scramblerSeed, scramblerOffset, rsMode)
		return
	default:
		info := framing.PackBitsMSB(rawBits)
		if len(info) < 18 {
			return
		}
		if rsMode == RSOn && !verifyMACPDURS(info[:18]) {
			return
		}
		if pdu, err := ParseMACPDU(info[:18]); err == nil {
			c.Ingest(pdu)
		}
	}
}

// applyDescrambleAndIngest descrambles in-place at the supplied
// offset, RS-verifies (when rsMode == RSOn), and forwards the PDU
// to Ingest if parsing succeeds.
func (c *ControlChannel) applyDescrambleAndIngest(bits []byte, seed uint64, offset int, rsMode RSMode) {
	descrambleAtOffset(bits, seed, offset)
	info := framing.PackBitsMSB(bits)
	if len(info) < 18 {
		return
	}
	if rsMode == RSOn && !verifyMACPDURS(info[:18]) {
		return
	}
	if pdu, err := ParseMACPDU(info[:18]); err == nil {
		c.Ingest(pdu)
	}
}

// descrambleAtOffset XOR-descrambles bits in-place with the PN44
// sequence starting at offset (folded into the 4320-bit
// superframe period).
func descrambleAtOffset(bits []byte, seed uint64, offset int) {
	s := framing.NewPN44Scrambler(seed)
	const superframeBits = 4320
	offset = offset % superframeBits
	if offset < 0 {
		offset += superframeBits
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

// Reset clears the SyncDetector's history so a stale match doesn't
// fire after a stream re-sync.
func (s *SyncDetector) Reset() {
	for i := range s.hist {
		s.hist[i] = 0
	}
	s.primed = 0
	s.pos = 0
}
