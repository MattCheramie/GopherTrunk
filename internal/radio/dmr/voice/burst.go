package voice

import "github.com/MattCheramie/GopherTrunk/internal/radio/dmr"

const (
	// VoiceHalfDibits is the dibit count of each voice payload half.
	// A DMR voice burst is 132 dibits: 54 payload + 24 sync/embedded +
	// 54 payload, with no slot-type fields (unlike a data burst).
	VoiceHalfDibits = 54
	// VoicePayloadBits is the total voice-payload bit count per burst
	// — the two 108-bit halves concatenated.
	VoicePayloadBits = 216
	// AMBEFrameBits is the on-air size of one AMBE+2 voice frame.
	AMBEFrameBits = 72
	// BurstsPerSuperframe is the burst count of one DMR voice
	// superframe (bursts A–F).
	BurstsPerSuperframe = 6
	// FramesPerBurst is the number of AMBE frames in one voice burst.
	FramesPerBurst = VoicePayloadBits / AMBEFrameBits // 3
	// FramesPerSuperframe is the AMBE frame count across bursts A–F.
	FramesPerSuperframe = FramesPerBurst * BurstsPerSuperframe // 18
)

// VoiceBits returns the 216 voice-payload bits of a DMR voice burst,
// one bit per byte MSB-first: the first 54-dibit half followed by the
// second 54-dibit half, with the central 24-dibit sync / embedded-
// signalling field skipped. Voice bursts have no slot-type fields, so
// the payload occupies the dibit positions a data burst uses for slot
// type.
func VoiceBits(b *dmr.Burst) []byte {
	out := make([]byte, 0, VoicePayloadBits)
	emit := func(d uint8) { out = append(out, (d>>1)&1, d&1) }
	for i := 0; i < VoiceHalfDibits; i++ {
		emit(b.Dibits[i])
	}
	for i := dmr.BurstDibits - VoiceHalfDibits; i < dmr.BurstDibits; i++ {
		emit(b.Dibits[i])
	}
	return out
}

// AMBEFrames splits a voice burst into its three 72-bit on-air AMBE+2
// frames, each one bit per byte MSB-first. The frames are contiguous
// slices of the 216 voice bits; the middle frame straddles the two
// payload halves, exactly as transmitted around the sync field.
func AMBEFrames(b *dmr.Burst) [FramesPerBurst][]byte {
	bits := VoiceBits(b)
	var frames [FramesPerBurst][]byte
	for i := range frames {
		frames[i] = bits[i*AMBEFrameBits : (i+1)*AMBEFrameBits]
	}
	return frames
}
