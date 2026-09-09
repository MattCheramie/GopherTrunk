package phase2

// P25 Phase 2 voice-frame extraction.
//
// A voice-bearing sub-frame (SlotTypeVoice4V / SlotTypeVoice2V) carries
// 4 or 2 AMBE+2 voice frames after its ISCH. Each on-air voice frame is
// the 72-bit FEC-wrapped form of the 49-bit AMBE+2 "3600x2450" vocoder
// payload. ExtractVoiceFrames undoes that wrapping and hands back
// 7-byte frames ready for internal/voice/ambe2.Decoder.Decode.
//
// The layout this file once assumed — four contiguous 36-dibit frames from
// dibit 32, unscrambled, FEC-decoded by the DMR AMBE path — was wrong on every
// count, and the real-capture calibration it was flagged for has now happened.
// Voice frames are threaded between the scattered DUID dibits, the payload is
// PN44-scrambled, and the Golay convention is the cyclic one P25 uses rather
// than the form internal/radio/framing's GolayDecode24_12 implements. See
// voice_burst.go, which is where voice extraction now lives.

// Voice-frame on-wire geometry within a sub-frame.
const (
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

// VoiceFrameCount returns how many AMBE+2 voice frames a sub-frame of the
// given SlotType carries, or 0 if it is not voice-bearing.
//
// Kept for fixtures that size a payload slice by slot type. Live extraction
// goes through voice_burst.go, which selects on the DUID instead: the SlotType
// comes from an ISCH model that does not match the air (issue #915).
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
