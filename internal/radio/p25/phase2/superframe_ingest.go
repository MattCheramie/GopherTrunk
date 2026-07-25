package phase2

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
	// SoftDecision requests the receiver-level soft-decision demod path
	// (issue #915): when set, the traffic-channel receiver is built with
	// receiver.Options.SoftDecision so it emits per-symbol soft
	// differentials alongside the hard dibits, and the superframe decoder's
	// ProcessSoft carries them into Subframe.Soft. The MAC-decode routines
	// here consume Subframe.Soft automatically when it is present
	// (DecodeSuperframeMACPDUsWithSlot), so this field is inert to the
	// decode side itself — it exists so the flag can travel with the rest
	// of the per-channel FEC config from the grant to the composer /
	// sigfollow receiver setup. Default false keeps the hard slicer.
	SoftDecision bool
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
// The PN44 descrambler is handed the sub-frame's channel-bit offset
// (slotChannelPN44Offset) because superframe sync pins which of the 12
// TDMA slots each sub-frame occupies, and the descramble runs on the
// coded channel bits before FEC (issue #915).
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
		offset := slotChannelPN44Offset(sub.Index)
		var pdu MACPDU
		var ok bool
		if len(sub.Soft) >= MACPayloadOffset+macLen {
			// Soft-decision path (issue #915): true per-bit soft Viterbi over
			// the diagonal-frame differential. Byte-identical to the hard path
			// on a clean signal; recovers ~1.5-2 dB otherwise.
			soft := sub.Soft[MACPayloadOffset : MACPayloadOffset+macLen]
			pdu, ok = decodeMACPDUDibitsSoftC(macDibits, soft, cfg.Trellis, cfg.RS,
				cfg.Interleave, cfg.Scrambler, cfg.Seed, offset)
		} else {
			pdu, ok = decodeMACPDUDibits(macDibits, cfg.Trellis, cfg.RS,
				cfg.Interleave, cfg.Scrambler, cfg.Seed, offset)
		}
		if ok {
			out = append(out, DecodedMACPDU{
				SlotType: sub.SlotType,
				PDU:      pdu,
				RSValid:  verifyMACPDURS(AssembleMACPDU(pdu)),
			})
		}
	}
	return out
}

// DecodeSuperframeMACPDUsWithSlotPinned is DecodeSuperframeMACPDUsWithSlot
// with a self-aligning ScramblerProbe. When cfg.Scrambler is ScramblerProbe
// and pin is non-nil, each MAC sub-frame is descrambled at the channel phase
// the pin has discovered (or, until one is found, an RS-gated sweep of the
// full 4320-bit sequence), so probe recovers a channel whose true intra-slot
// PN44 offset differs from GopherTrunk's superframe-grid assumption
// (slotChannelPN44Offset) — the mac_rs_valid=0 case in issue #915, Finding B.
// The slot-base-only sweep in DecodeSuperframeMACPDUsWithSlot holds that
// intra-slot offset fixed, so it cannot. For any other scrambler mode, or a
// nil pin, this is identical to DecodeSuperframeMACPDUsWithSlot.
func DecodeSuperframeMACPDUsWithSlotPinned(sf Superframe, cfg MACDecodeConfig, pin *ScramblerPin) []DecodedMACPDU {
	if pin == nil || cfg.Scrambler != ScramblerProbe {
		return DecodeSuperframeMACPDUsWithSlot(sf, cfg)
	}
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
		if pdu, ok := pin.probeSubframe(macDibits, cfg.Trellis, cfg.Interleave, cfg.Seed, sub.Index); ok {
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

// slotChannelPN44Offset returns the PN44 sequence offset, in CHANNEL
// bits, at which sub-frame index's MAC payload begins within the
// continuous 4320-bit superframe scrambling sequence (TIA-102.BBAC-1
// §7.2.5). Because the descramble is applied to the coded channel bits
// before FEC (issue #915), the offset is the absolute channel-bit
// position of the MAC payload: the sub-frame's start in the superframe
// (index × DibitsPerSubframe × 2 bits) plus the MAC payload's offset
// within the sub-frame (MACPayloadOffset × 2 bits), which follows the
// sync + ISCH region.
//
// GopherTrunk's superframe grid (sync width, ISCH placement) is a
// documented working assumption, so this offset is the principled
// default for ScramblerOn; ScramblerProbe confirms the true phase
// against the RS gate rather than trusting it. Out-of-range indices
// fall back to offset 0.
func slotChannelPN44Offset(index int) int {
	if index < 0 || index >= SubframesPerSuperframe {
		return 0
	}
	return (index*DibitsPerSubframe + MACPayloadOffset) * 2
}
