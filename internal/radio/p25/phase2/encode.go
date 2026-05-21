package phase2

import (
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
