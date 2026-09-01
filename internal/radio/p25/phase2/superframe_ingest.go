package phase2

import "sync"

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
	// Equalizer requests the blind CMA adaptive equalizer on the Phase 2
	// traffic-channel receiver (receiver.Options.Equalizer, issue #915): it
	// removes residual inter-symbol interference on the symbol stream ahead of
	// the differential decode. Like SoftDecision this field is inert to the
	// MAC-decode routines here — it only travels with the per-channel config
	// from the grant to the composer / sigfollow receiver setup. Default false
	// leaves the symbol stream untouched.
	Equalizer bool
	// DCBlock requests the DC-removal high-pass on the Phase 2
	// traffic-channel receiver (receiver.Options.EnableDCBlock): it strips a
	// zero-IF front end's LO-leakage spur from an on-channel voice DDC, the
	// same stage the P1 / TETRA voice receivers run. Like SoftDecision this
	// field is inert to the MAC-decode routines here — it only travels with
	// the per-channel config from the grant to the composer / sigfollow
	// receiver setup. Default false leaves the IQ untouched.
	DCBlock bool
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

// sISCHSlots are the six superframe slots that carry the S-ISCH, and so the
// only slots a superframe anchor can land on: the SuperframeDecoder locks onto
// the outbound frame sync, which is transmitted in these and only these.
//
// Both reference decoders agree on the set, and real air confirms it from two
// directions — the sync arrives at inter-hit deltas of 540, 180, 540, 180
// dibits, and every scramble phase that ever decodes a burst resolves to one
// of these six. Searching six candidates rather than twelve halves the
// exposure to a false accept.
var sISCHSlots = [6]int{2, 3, 6, 7, 10, 11}

// voiceSlotVoteMaxErrs is how many bits the Golay may correct in a voice
// burst's first frame and still have that burst vote for a slot phase.
//
// Zero. The vote is a discriminator, not a decoder: at the right phase a clean
// frame needs no correction at all, while a wrong phase leaves the decoder
// working near its radius of 3. Allowing even one correction lets a wrong
// phase collect votes from noise, and a wrong phase that wins the vote costs
// the whole superframe — both its signalling and its audio.
const voiceSlotVoteMaxErrs = 0

// scrambleCache memoises the 4320-bit PN44 sequence per seed. A channel's seed
// never changes mid-call, so this turns a per-superframe LFSR run into a map
// lookup. Entries are read-only once stored.
var scrambleCache sync.Map // uint64 seed -> []byte

func cachedScrambleSequence(seed uint64) []byte {
	if v, ok := scrambleCache.Load(seed); ok {
		return v.([]byte)
	}
	seq := ScrambleSequence(seed)
	scrambleCache.Store(seed, seq)
	return seq
}

// ResolveSuperframeSlotOffset finds the constant that maps a sub-frame's Index
// to its true slot within the scrambling sequence, and reports how many ACCH
// bursts decoded under it. Exported for the voice path, which needs the same
// slot to descramble a voice burst; score <= 0 means no offset was resolved
// and nothing in the superframe can be descrambled.
//
// The superframe anchor is whichever S-ISCH slot the sync matched, so Index is
// offset from the true slot by an amount that is fixed for the superframe but
// unknown to the decoder. This resolves it by vote: each candidate offset is
// scored by how many of the superframe's ACCH bursts decode under it, and the
// best is taken.
//
// Voting over the whole superframe is what makes this safe. A wrong offset
// turns a burst into noise, and noise clears the outer RS about once in a
// billion and the CRC-12 once in 4096 — but for a wrong offset to win a vote
// it would have to do that on more bursts than the correct offset does, which
// does not happen. Deciding per burst instead, as the field probe does, gives
// up that margin.
func ResolveSuperframeSlotOffset(sf Superframe, seq []byte) (offset, score int) {
	return resolveSuperframeSlotOffset(sf, seq)
}

func resolveSuperframeSlotOffset(sf Superframe, seq []byte) (offset, score int) {
	// Score with the cheap CRC-only probe first. It sees any burst the air
	// delivered intact, which is nearly all of them, and costs a fraction of
	// the full decode. A superframe whose only signalling burst needs the
	// outer RS to close would score zero under it, so fall back to the full
	// decode rather than declaring the superframe unresolvable.
	if off, n := resolveSlotOffsetScored(sf, seq, false); n > 0 {
		return off, n
	}
	return resolveSlotOffsetScored(sf, seq, true)
}

func resolveSlotOffsetScored(sf Superframe, seq []byte, deep bool) (offset, score int) {
	// How many bursts could vote at all, so a candidate that satisfies every
	// one of them can be taken without scoring the rest.
	eligible := 0
	for _, sub := range sf.Subframes {
		if len(sub.Dibits) < BurstDibits {
			continue
		}
		if bt := BurstTypeOf(sub.Dibits); bt.IsACCH() || bt.IsVoice() {
			eligible++
		}
	}
	best, bestScore := 0, -1
	for _, cand := range sISCHSlots {
		n := 0
		for _, sub := range sf.Subframes {
			if len(sub.Dibits) < BurstDibits {
				continue
			}
			slot := (sub.Index + cand) % SubframesPerSuperframe
			switch bt := BurstTypeOf(sub.Dibits); {
			case bt.IsACCH():
				ok := acchSlotProbe(sub.Dibits, slot, seq)
				if !ok && deep {
					_, ok = DecodeACCHBurst(sub.Dibits, slot, seq)
				}
				if ok {
					n++
				}
			case bt.IsVoice():
				// A voice burst votes too. Its AMBE codewords are
				// Golay-protected, so the right phase leaves the FEC with
				// almost nothing to correct while a wrong one sits near the
				// correction radius on every frame — a margin wide enough to
				// separate the phases on its own. Without this a superframe
				// of nothing but voice could not be descrambled at all,
				// because it carries no signalling to resolve the phase from.
				if errs, ok := voiceSlotProbe(sub.Dibits, slot, seq); ok && errs <= voiceSlotVoteMaxErrs {
					n++
				}
			}
		}
		if n > bestScore {
			best, bestScore = cand, n
		}
		if bestScore == eligible && eligible > 0 {
			break // nothing can beat a clean sweep
		}
	}
	return best, bestScore
}

// DecodeSuperframeMACPDUsWithSlot returns every MAC PDU found in sf's ACCH
// bursts, in sub-frame order, each tagged with the channel state of the burst
// it rode in.
//
// A burst is selected by its DUID — the 8-bit code scattered through the
// payload that names the burst type — not by the ISCH SlotType, which this
// package models on a working assumption rather than the spec. The DUID is
// authoritative, it is protected by its own (8,4) code, and it is what both
// reference decoders dispatch on.
//
// One ACCH burst yields one *message*, which may carry several MAC structures;
// each becomes a DecodedMACPDU, all sharing the burst's channel state and FEC
// verdict. That is why a single GROUP VOICE CHANNEL USER burst can also report
// the neighbour site it was packed alongside.
//
// Only cfg.Seed is consulted. The Trellis / RS / Interleave / Scrambler mode
// fields configured a FEC chain that never decoded real air (issue #915) and
// are inert; see MACDecodeConfig.
func DecodeSuperframeMACPDUsWithSlot(sf Superframe, cfg MACDecodeConfig) []DecodedMACPDU {
	seq := cachedScrambleSequence(cfg.Seed)
	offset, score := resolveSuperframeSlotOffset(sf, seq)
	if score <= 0 {
		return nil
	}
	var out []DecodedMACPDU
	for _, sub := range sf.Subframes {
		if len(sub.Dibits) < BurstDibits || !BurstTypeOf(sub.Dibits).IsACCH() {
			continue
		}
		slot := (sub.Index + offset) % SubframesPerSuperframe
		res, ok := DecodeACCHBurst(sub.Dibits, slot, seq)
		if !ok {
			continue
		}
		msg, err := ParseACCHMessage(res.Message)
		if err != nil {
			continue
		}
		st := msg.Type.SlotType()
		for _, pdu := range msg.Structures {
			out = append(out, DecodedMACPDU{SlotType: st, PDU: pdu, RSValid: res.RSValid})
		}
	}
	return out
}

// DecodeSuperframeMACPDUsWithSlotPinned is retained for callers that still
// pass a ScramblerPin; it delegates to DecodeSuperframeMACPDUsWithSlot and
// ignores the pin.
//
// The pin existed to hunt for a channel's true intra-slot PN44 offset by
// sweeping the whole 4320-bit sequence, because the superframe-grid assumption
// put the MAC payload at the wrong place and nothing ever descrambled (issue
// #915, Finding B). The offset is no longer unknown: the payload begins at
// burst dibit 20 and the sequence origin at slot 0 bit 20, both confirmed
// against two independent decoders and real air, and what remains — which
// S-ISCH slot the superframe anchored on — is resolved per superframe by
// resolveSuperframeSlotOffset. There is nothing left to sweep for.
func DecodeSuperframeMACPDUsWithSlotPinned(sf Superframe, cfg MACDecodeConfig, _ *ScramblerPin) []DecodedMACPDU {
	return DecodeSuperframeMACPDUsWithSlot(sf, cfg)
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
