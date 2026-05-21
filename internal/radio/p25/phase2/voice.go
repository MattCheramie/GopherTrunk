package phase2

import (
	"errors"
	"fmt"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// P25 Phase 2 voice-frame extraction.
//
// A voice-bearing sub-frame (SlotTypeVoice4V / SlotTypeVoice2V) carries
// 4 or 2 AMBE+2 voice frames after its ISCH. Each on-air voice frame is
// the 72-bit FEC-wrapped form of the 49-bit AMBE+2 "3600x2450" vocoder
// payload. ExtractVoiceFrames undoes that wrapping and hands back
// 7-byte frames ready for internal/voice/ambe2.Decoder.Decode.
//
// The AMBE+2 FEC (C0/C1 Golay(23,12) + C0-seeded keystream over C1) is
// the same codec family P25 Phase 2 and DMR share, so the FEC decode is
// delegated to internal/radio/dmr/voice (DecodeAMBEFrame). TIA-102.BBAB
// §7 may interleave the 4/2 voice frames across the sub-frame
// differently from the contiguous layout assumed here; that on-wire
// detail — like the ISCH code and the superframe sync cadence — is
// flagged for real-capture calibration. The layout constants below are
// the single point of change if a capture shows otherwise.

// Voice-frame on-wire geometry within a sub-frame.
const (
	// VoiceOnAirFrameDibits is the on-wire width of one FEC-wrapped
	// AMBE+2 voice frame: 72 bits = 36 dibits.
	VoiceOnAirFrameDibits = 36
	// VoiceFrameOffset is the dibit offset of the first voice frame in
	// a voice sub-frame — immediately after the ISCH field.
	VoiceFrameOffset = ISCHOffset + ISCHDibits
	// Voice4VFrameCount / Voice2VFrameCount are the AMBE+2 voice-frame
	// counts in a SlotTypeVoice4V / SlotTypeVoice2V sub-frame.
	Voice4VFrameCount = 4
	Voice2VFrameCount = 2
	// VoiceFrameBytes is the size of one extracted, FEC-decoded AMBE+2
	// frame: 49 info bits packed MSB-first into 7 bytes — the frame
	// size internal/voice/ambe2.Decoder.Decode expects.
	VoiceFrameBytes = 7
	// voiceInfoBits is the AMBE+2 vocoder-payload bit count per frame.
	voiceInfoBits = 49
)

// ErrNotVoiceSubframe is returned by ExtractVoiceFrames when the
// sub-frame's SlotType is not voice-bearing.
var ErrNotVoiceSubframe = errors.New("p25/phase2: sub-frame SlotType is not voice-bearing")

// VoiceFrameCount returns how many AMBE+2 voice frames a sub-frame of
// the given SlotType carries, or 0 if it is not voice-bearing.
func VoiceFrameCount(s SlotType) int {
	switch s {
	case SlotTypeVoice4V:
		return Voice4VFrameCount
	case SlotTypeVoice2V:
		return Voice2VFrameCount
	default:
		return 0
	}
}

// ExtractVoiceFrames pulls the AMBE+2 voice frames out of a voice-
// bearing sub-frame and FEC-decodes each to a 7-byte vocoder frame.
//
// frames[i] is the i-th AMBE+2 frame, a VoiceFrameBytes-long slice
// consumable by ambe2.Decoder.Decode. totalErrs is the sum of Golay
// errors corrected across all frames. A non-nil err surfaces an
// uncorrectable frame; the failing slot still holds a zero frame so the
// caller can frame-repeat rather than skip a 20 ms slot.
func ExtractVoiceFrames(sf Subframe) (frames [][]byte, totalErrs int, err error) {
	n := VoiceFrameCount(sf.SlotType)
	if n == 0 {
		return nil, 0, fmt.Errorf("%w: %v", ErrNotVoiceSubframe, sf.SlotType)
	}
	need := VoiceFrameOffset + n*VoiceOnAirFrameDibits
	if len(sf.Dibits) < need {
		return nil, 0, fmt.Errorf("p25/phase2: voice sub-frame needs %d dibits, got %d",
			need, len(sf.Dibits))
	}
	frames = make([][]byte, n)
	var firstErr error
	for i := 0; i < n; i++ {
		off := VoiceFrameOffset + i*VoiceOnAirFrameDibits
		onAir := framing.DibitsToBits(sf.Dibits[off : off+VoiceOnAirFrameDibits])
		info, errs, decErr := dmrvoice.DecodeAMBEFrame(onAir)
		totalErrs += errs
		if decErr != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("voice frame %d: %w", i, decErr)
			}
			frames[i] = make([]byte, VoiceFrameBytes)
			continue
		}
		frames[i] = framing.PackBitsMSB(info)
	}
	return frames, totalErrs, firstErr
}
