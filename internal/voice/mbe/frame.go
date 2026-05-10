package mbe

// Frame parameters shared across MBE-family vocoders. Every IMBE
// 4400 / AMBE+2 2400 frame carries 20 ms of audio at 8 kHz mono;
// the synthesizer produces 160 int16 PCM samples per call.
const (
	// SamplesPerFrame is the PCM count one synthesis call produces.
	// Fixed at 8 kHz × 20 ms = 160 samples for both IMBE and AMBE+2.
	SamplesPerFrame = 160

	// PCMSampleRate is the recorder's expected output rate.
	PCMSampleRate = 8000

	// FrameDurationMs documents the 20 ms cadence.
	FrameDurationMs = 20

	// MaxL is the upper bound on the harmonic count L for both
	// IMBE (9..56) and AMBE+2. Slice indices use the 1-based
	// convention from TIA-102.BABA — index 0 is unused. Storage
	// arrays are sized [MaxL+1] so [1..MaxL] is addressable.
	MaxL = 56
)
