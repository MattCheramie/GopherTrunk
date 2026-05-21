package phase2

import (
	"encoding/binary"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Synthesized-fixture encoders for the P25 Phase 2 superframe layer.
// These are the inverse of SuperframeDecoder / the ISCH + voice
// decoders and exist so unit and integration tests can build a
// well-formed dibit stream without a real over-the-air capture. They
// panic on misuse — they are test scaffolding, not a production
// transmit path (GopherTrunk is receive-only).

// IdleSubframe returns a DibitsPerSubframe-long all-zero sub-frame body
// for building synthesized superframe fixtures.
func IdleSubframe() []uint8 {
	return make([]uint8, DibitsPerSubframe)
}

// WriteISCH overwrites the ISCH region of a DibitsPerSubframe-long
// sub-frame body in place with the Golay-encoded ISCH for slotType +
// counter, so SuperframeDecoder decodes the sub-frame as that SlotType.
// Panics if sub is not exactly DibitsPerSubframe dibits.
func WriteISCH(sub []uint8, slotType SlotType, counter uint8) {
	if len(sub) != DibitsPerSubframe {
		panic("p25/phase2: WriteISCH sub-frame must be DibitsPerSubframe dibits")
	}
	copy(sub[ISCHOffset:ISCHOffset+ISCHDibits],
		EncodeISCH(ISCH{SlotType: slotType, Counter: counter}))
}

// EncodeSuperframe concatenates 12 sub-frame dibit bodies into one
// DibitsPerSuperframe-long superframe stream and injects the 20-dibit
// outbound sync word at the head of sub-frame SyncSubframeIndex so a
// SuperframeDecoder can anchor on it. Each body must be exactly
// DibitsPerSubframe dibits; EncodeSuperframe panics otherwise.
func EncodeSuperframe(subframes [SubframesPerSuperframe][]uint8) []uint8 {
	out := make([]uint8, 0, DibitsPerSuperframe)
	for _, sub := range subframes {
		if len(sub) != DibitsPerSubframe {
			panic("p25/phase2: EncodeSuperframe sub-frame must be DibitsPerSubframe dibits")
		}
		out = append(out, sub...)
	}
	sync := OutboundSyncDibits()
	base := SyncSubframeIndex * DibitsPerSubframe
	copy(out[base:base+SyncDibits], sync)
	return out
}

// EncodeMACSubframe builds a DibitsPerSubframe-long MAC sub-frame: the
// ISCH for slotType (which must be a MAC SlotType) at counter, followed
// by the FEC-encoded MAC PDU at MACPayloadOffset. It is the inverse of
// the slice + decodeMACPDUDibits step IngestSuperframe runs. The PDU is
// assembled and zero-padded/truncated to the 18-byte (144-bit) MAC PDU
// width; mode and interleave must match the decoder's configuration.
// Panics if slotType is not a MAC SlotType.
func EncodeMACSubframe(slotType SlotType, counter uint8, pdu MACPDU, mode TrellisMode, interleave InterleaveMode) []uint8 {
	if !slotType.IsMAC() {
		panic("p25/phase2: EncodeMACSubframe slotType is not a MAC SlotType")
	}
	macBytes := AssembleMACPDU(pdu)
	full := make([]byte, 18)
	copy(full, macBytes) // pad short / truncate long to the 144-bit MAC PDU
	infoDibits := framing.BitsToDibits(framing.UnpackBitsMSB(full, 144))

	channelDibits := infoDibits
	if mode == TrellisOn {
		channelDibits = framing.EncodeP25Trellis(infoDibits)
	}
	if interleave == InterleaveOn {
		channelDibits = framing.InterleaveMACBurst(channelDibits)
	}

	sub := IdleSubframe()
	WriteISCH(sub, slotType, counter)
	copy(sub[MACPayloadOffset:MACPayloadOffset+len(channelDibits)], channelDibits)
	return sub
}

// EncodeEncryptionSync builds the MAC PDU form of an Encryption Sync —
// the inverse of MACPDU.AsEncryptionSync.
func EncodeEncryptionSync(es EncryptionSync) MACPDU {
	payload := make([]byte, 12)
	payload[0] = es.AlgorithmID
	binary.BigEndian.PutUint16(payload[1:3], es.KeyID)
	copy(payload[3:12], es.MessageIndicator[:])
	return MACPDU{Opcode: OpEncryptionSync, Payload: payload}
}

// EncodeGroupAffiliationResponse builds the MAC PDU form of a Group
// Affiliation Response — the inverse of AsGroupAffiliationResponse.
func EncodeGroupAffiliationResponse(g GroupAffiliationResponse) MACPDU {
	p := make([]byte, 8)
	p[0] = g.Response & 0x03
	binary.BigEndian.PutUint16(p[1:3], g.AnnouncementGroup)
	binary.BigEndian.PutUint16(p[3:5], g.GroupAddress)
	p[5] = byte(g.TargetID >> 16)
	p[6] = byte(g.TargetID >> 8)
	p[7] = byte(g.TargetID)
	return MACPDU{Opcode: OpGroupAffiliationResponse, Payload: p}
}

// EncodeUnitRegistrationResponse builds the MAC PDU form of a Unit
// Registration Response — the inverse of AsUnitRegistrationResponse.
func EncodeUnitRegistrationResponse(u UnitRegistrationResponse) MACPDU {
	p := make([]byte, 8)
	p[0] = u.Response & 0x03
	p[1] = byte(u.WACN >> 12)
	p[2] = byte(u.WACN >> 4)
	p[3] = byte(u.WACN<<4) | byte((u.SystemID>>8)&0x0F)
	p[4] = byte(u.SystemID)
	p[5] = byte(u.SourceID >> 16)
	p[6] = byte(u.SourceID >> 8)
	p[7] = byte(u.SourceID)
	return MACPDU{Opcode: OpUnitRegistrationResponse, Payload: p}
}

// EncodeTalkerAliasFragment builds the MAC PDU form of a talker-alias
// fragment — the inverse of MACPDU.AsTalkerAliasFragment.
func EncodeTalkerAliasFragment(f TalkerAliasFragment) MACPDU {
	payload := make([]byte, 5+len(f.Data))
	payload[0] = byte(f.SourceID >> 16)
	payload[1] = byte(f.SourceID >> 8)
	payload[2] = byte(f.SourceID)
	payload[3] = f.BlockIndex
	payload[4] = f.BlockCount
	copy(payload[5:], f.Data)
	return MACPDU{Opcode: OpVendorTalkerAlias, MFID: MFIDMotorola, Payload: payload}
}

// EncodeVoiceSubframe builds a DibitsPerSubframe-long voice sub-frame:
// the ISCH for slotType (which must be SlotTypeVoice4V or
// SlotTypeVoice2V) at counter, followed by the AMBE+2-FEC-encoded voice
// frames. payloads holds one VoiceFrameBytes-long AMBE+2 frame per
// voice slot; len(payloads) must equal VoiceFrameCount(slotType). It is
// the inverse of ExtractVoiceFrames. Panics on misuse.
func EncodeVoiceSubframe(slotType SlotType, counter uint8, payloads [][]byte) []uint8 {
	n := VoiceFrameCount(slotType)
	if n == 0 {
		panic("p25/phase2: EncodeVoiceSubframe slotType is not voice-bearing")
	}
	if len(payloads) != n {
		panic("p25/phase2: EncodeVoiceSubframe payload count mismatch")
	}
	sub := IdleSubframe()
	WriteISCH(sub, slotType, counter)
	for i, p := range payloads {
		if len(p) != VoiceFrameBytes {
			panic("p25/phase2: EncodeVoiceSubframe payload must be VoiceFrameBytes")
		}
		onAir, err := dmrvoice.EncodeAMBEFrame(framing.UnpackBitsMSB(p, voiceInfoBits))
		if err != nil {
			panic("p25/phase2: EncodeVoiceSubframe: " + err.Error())
		}
		off := VoiceFrameOffset + i*VoiceOnAirFrameDibits
		copy(sub[off:off+VoiceOnAirFrameDibits], framing.BitsToDibits(onAir))
	}
	return sub
}
