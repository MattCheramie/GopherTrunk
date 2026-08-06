package nxdn

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// VCH (traffic voice channel) decode for NXDN.
//
// An NXDN traffic frame's 288-bit Information field (frame.go
// Frame.Info() = Dibits[48:192], InfoFieldDibits=144) carries the VCH
// voice payload as 4 AMBE+2 voice frames of 72 on-air bits each
// (4 × 72 = 288). Each 20 ms AMBE+2 frame decodes (voice_ambe.go) to a
// 49-bit vocoder payload; four of them span the 80 ms NXDN frame.
//
// ⚠️ UNVERIFIED ON AIR — see voice_ambe.go. The 4×72 carve and the AMBE
// deinterleave are transcribed from spec/reference and validated only by
// the synthetic round-trip; a real NXDN VCH capture is needed to confirm
// them (and the codebook variant).

// VCHVoiceFramesPerFrame is the number of AMBE+2 voice frames the VCH
// carries in one 80 ms NXDN frame.
const VCHVoiceFramesPerFrame = 4

// VCHFrame is one decoded AMBE+2 voice frame carved from a VCH burst.
type VCHFrame struct {
	// Payload is the 49-bit vocoder payload, one bit per byte, in the
	// order DecodeVCHFrame emits (C0:12 + C1:12 + C2:11 + C3:14). Pack
	// it MSB-first to 7 bytes (framing.PackBitsMSB) for the ambe2
	// decoder.
	Payload []byte
	// Errors is the Golay-corrected bit count across the C0/C1
	// sub-vectors — a decode-quality signal to feed the vocoder's
	// error-aware smoothing.
	Errors int
}

// ExtractVCHFrames decodes the 144-dibit (288-bit) Information field of
// a VCH traffic frame into its VCHVoiceFramesPerFrame AMBE+2 voice
// frames. Returns nil if infoDibits is not exactly InfoFieldDibits long.
// Frames whose AMBE FEC hard-fails are skipped (so a partially corrupt
// burst still yields its good frames).
func ExtractVCHFrames(infoDibits []uint8) []VCHFrame {
	if len(infoDibits) != InfoFieldDibits {
		return nil
	}
	bitsAll := framing.DibitsToBits(infoDibits) // 288 bits
	out := make([]VCHFrame, 0, VCHVoiceFramesPerFrame)
	for i := 0; i < VCHVoiceFramesPerFrame; i++ {
		seg := bitsAll[i*nxdnAMBEOnAirBits : (i+1)*nxdnAMBEOnAirBits]
		payload, errs, err := DecodeVCHFrame(seg)
		if err != nil {
			continue
		}
		out = append(out, VCHFrame{Payload: payload, Errors: errs})
	}
	return out
}
