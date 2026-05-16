package ysf

// Frame layout constants per the Yaesu System Fusion specification.
// All values measured in dibits unless suffixed otherwise.
const (
	// FrameDibits is the length of one YSF frame on-air. 480 dibits
	// = 960 bits = 100 ms at 4800 baud C4FM.
	FrameDibits = 480

	// FrameBits = FrameDibits * 2.
	FrameBits = FrameDibits * 2

	// FrameDurationMs is informational — every frame carries 100 ms of
	// air time at the fixed 4800 sym/s rate.
	FrameDurationMs = 100

	// FICHDibits is the size of the encoded Frame Information Channel
	// region. The 100-dibit field carries Trellis-encoded FICH bits;
	// after Viterbi decode the recovered info is a much smaller field
	// (handled by the future FICH decoder, not in this PR).
	FICHDibits = 100

	// PayloadDibits covers the DCH (Data CHannel) region — voice or
	// data depending on the FICH's Frame Type field.
	PayloadDibits = FrameDibits - FSWDibits - FICHDibits // 360
)

// Frame offsets for callers that want to slice a 480-dibit window.
const (
	FSWOffset     = 0
	FICHOffset    = FSWOffset + FSWDibits   // 20
	PayloadOffset = FICHOffset + FICHDibits // 120
)
