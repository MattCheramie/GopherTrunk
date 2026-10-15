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
	// region: Golay-coded, convolutionally coded and interleaved FICH
	// bits, decoded by DecodeAirFICH.
	FICHDibits = 100

	// PayloadDibits covers the DCH (Data CHannel) region — voice or
	// data depending on the FICH's frame indicator and data type.
	PayloadDibits = FrameDibits - FSWDibits - FICHDibits // 360
)

// Frame offsets for callers that want to slice a 480-dibit window.
const (
	FSWOffset     = 0
	FICHOffset    = FSWOffset + FSWDibits   // 20
	PayloadOffset = FICHOffset + FICHDibits // 120
)
