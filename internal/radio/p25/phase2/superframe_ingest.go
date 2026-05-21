package phase2

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// MACPayloadOffset is the dibit offset of the MAC PDU within a
// 180-dibit sub-frame: it follows the SyncDibits-wide region and the
// ISCH, sharing the same start as a voice sub-frame's first voice
// frame (VoiceFrameOffset). The MAC PDU width itself is macPDUDibits
// (raw) or macPDUDibitsTrellis (trellis-coded), selected by TrellisMode.
const MACPayloadOffset = ISCHOffset + ISCHDibits

// IngestSuperframe routes every MAC-bearing sub-frame of sf through the
// MAC-PDU FEC chain into Ingest. It is the superframe-structured
// counterpart of the flat Process adapter: the SuperframeDecoder has
// already locked the 360 ms superframe, sliced the 12 sub-frames, and
// decoded each ISCH SlotType, so this routes only the sub-frames whose
// SlotType.IsMAC() and skips voice sub-frames — the composer voice
// chain (internal/voice/composer/p25p2_voice.go) owns voice extraction.
//
// Because superframe sync pins which of the 12 TDMA slots a sub-frame
// occupies, the PN44 descrambler can be handed its true per-slot offset
// (slotPN44Offset) instead of blind-probing every offset.
func (c *ControlChannel) IngestSuperframe(sf Superframe) {
	c.mu.Lock()
	mode := c.trellisMode
	rsMode := c.rsMode
	interleaveMode := c.interleaveMode
	scramblerMode := c.scramblerMode
	scramblerSeed := c.scramblerSeed
	c.mu.Unlock()

	macLen := macPDUDibits
	if mode == TrellisOn {
		macLen = macPDUDibitsTrellis
	}
	for _, sub := range sf.Subframes {
		if !sub.SlotType.IsMAC() {
			continue
		}
		if len(sub.Dibits) < MACPayloadOffset+macLen {
			continue
		}
		macDibits := sub.Dibits[MACPayloadOffset : MACPayloadOffset+macLen]
		offset := slotPN44Offset(sub.Index)
		if pdu, ok := decodeMACPDUDibits(macDibits, mode, rsMode, interleaveMode,
			scramblerMode, scramblerSeed, offset); ok {
			c.Ingest(pdu)
		}
	}
}

// slotPN44Offset returns the PN44 sequence offset for sub-frame index
// (0..11) — the spec-defined per-slot offset, known here because
// superframe sync pins which slot a sub-frame occupies. Out-of-range
// indices fall back to offset 0.
func slotPN44Offset(index int) int {
	offs := framing.PN44SlotOffsetsOutbound
	if index < 0 || index >= len(offs) {
		return 0
	}
	return offs[index]
}
