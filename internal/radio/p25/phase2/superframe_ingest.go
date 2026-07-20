package phase2

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// MACPayloadOffset is the dibit offset of the MAC PDU within a
// 180-dibit sub-frame: it follows the SyncDibits-wide region and the
// ISCH, sharing the same start as a voice sub-frame's first voice
// frame (VoiceFrameOffset). The MAC PDU width itself is macPDUDibits
// (raw) or macPDUDibitsTrellis (trellis-coded), selected by TrellisMode.
const MACPayloadOffset = ISCHOffset + ISCHDibits

// MACDecodeConfig collects the per-channel FEC parameters
// DecodeSuperframeMACPDUs needs to lift MAC PDUs out of a Phase 2
// superframe's MAC sub-frames. The fields mirror the ControlChannel
// setters one-for-one so a CC can hand its current config to a voice
// composer (which does not own a CC) without exposing internal state.
type MACDecodeConfig struct {
	Trellis    TrellisMode
	RS         RSMode
	Interleave InterleaveMode
	Scrambler  ScramblerMode
	Seed       uint64
}

// DecodedMACPDU pairs a decoded MAC PDU with the SlotType of the
// sub-frame it rode in. Phase 2 carries the encryption sync
// (ALGID/KID/MI) in the MAC_PTT message (SlotTypeMACPTT) that begins a
// transmission, not in a distinct MAC opcode, so a caller that needs to
// tell PTT signalling from ordinary FACCH/SACCH signalling must see the
// slot type — DecodeSuperframeMACPDUs alone discards it. Issue #813.
type DecodedMACPDU struct {
	SlotType SlotType
	PDU      MACPDU
	// RSValid reports whether the recovered 18-byte MAC PDU satisfies
	// the outer RS(24, 16, 9) parity check (TIA-102.BAAA-A §5.9),
	// computed regardless of the channel's RSMode. It is the per-PDU
	// FEC-integrity signal: a real over-the-air MAC PDU that framed and
	// descrambled correctly verifies clean, whereas a mis-framed /
	// mis-descrambled window packs into random bytes that (almost)
	// never satisfy the RS syndromes. The MAC path runs with the outer
	// RS check off by default (a permissive parse for weak signals), so
	// without this flag the caller cannot tell a genuine PDU from
	// garbage that merely happened to parse — which lets a mis-decoded
	// GROUP_VOICE_CHANNEL_USER inject a bogus source RID (issue #915,
	// the source-side analogue of the #924 algid gate). Callers that
	// backfill trust-sensitive fields (the completed-call source RID)
	// gate on this; the framing-health census counts it.
	RSValid bool
}

// DecodeSuperframeMACPDUsWithSlot returns every successfully decoded MAC
// PDU found in sf's MAC-typed sub-frames, in sub-frame order, each
// tagged with the SlotType of the sub-frame it came from. Voice
// sub-frames are skipped. It is the slot-aware form of
// DecodeSuperframeMACPDUs; callers that route on the PTT slot
// (encryption sync) use this one.
//
// The PN44 descrambler is handed the spec's per-slot offset
// (slotPN44Offset) because superframe sync pins which of the 12 TDMA
// slots each sub-frame occupies.
func DecodeSuperframeMACPDUsWithSlot(sf Superframe, cfg MACDecodeConfig) []DecodedMACPDU {
	macLen := macPDUDibits
	if cfg.Trellis == TrellisOn {
		macLen = macPDUDibitsTrellis
	}
	var out []DecodedMACPDU
	for _, sub := range sf.Subframes {
		if !sub.SlotType.IsMAC() {
			continue
		}
		if len(sub.Dibits) < MACPayloadOffset+macLen {
			continue
		}
		macDibits := sub.Dibits[MACPayloadOffset : MACPayloadOffset+macLen]
		offset := slotPN44Offset(sub.Index)
		if pdu, ok := decodeMACPDUDibits(macDibits, cfg.Trellis, cfg.RS,
			cfg.Interleave, cfg.Scrambler, cfg.Seed, offset); ok {
			out = append(out, DecodedMACPDU{
				SlotType: sub.SlotType,
				PDU:      pdu,
				RSValid:  verifyMACPDURS(AssembleMACPDU(pdu)),
			})
		}
	}
	return out
}

// DecodeSuperframeMACPDUs returns every successfully decoded MAC PDU
// found in sf's MAC-typed sub-frames, in sub-frame order. Voice
// sub-frames are skipped. Both the control-channel ingest path and the
// voice-channel composer call this — Phase 2 voice traffic channels
// interleave MAC sub-frames (signalling, talker alias, encryption
// sync, …) with voice sub-frames, and the composer needs the same MAC
// dispatch the CC runs. Callers that need each PDU's originating slot
// type use DecodeSuperframeMACPDUsWithSlot.
func DecodeSuperframeMACPDUs(sf Superframe, cfg MACDecodeConfig) []MACPDU {
	decoded := DecodeSuperframeMACPDUsWithSlot(sf, cfg)
	if len(decoded) == 0 {
		return nil
	}
	out := make([]MACPDU, len(decoded))
	for i, d := range decoded {
		out[i] = d.PDU
	}
	return out
}

// IngestSuperframe routes every MAC-bearing sub-frame of sf through the
// MAC-PDU FEC chain into Ingest. It is the superframe-structured
// counterpart of the flat Process adapter: the SuperframeDecoder has
// already locked the 360 ms superframe, sliced the 12 sub-frames, and
// decoded each ISCH SlotType, so this routes only the sub-frames whose
// SlotType.IsMAC() and skips voice sub-frames — the composer voice
// chain (internal/voice/composer/p25p2_voice.go) owns voice extraction
// and runs its own MAC dispatch for talker-alias fragments via
// DecodeSuperframeMACPDUs.
func (c *ControlChannel) IngestSuperframe(sf Superframe) {
	c.mu.Lock()
	cfg := MACDecodeConfig{
		Trellis:    c.trellisMode,
		RS:         c.rsMode,
		Interleave: c.interleaveMode,
		Scrambler:  c.scramblerMode,
		Seed:       c.scramblerSeed,
	}
	c.mu.Unlock()
	for _, pdu := range DecodeSuperframeMACPDUs(sf, cfg) {
		c.Ingest(pdu)
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
