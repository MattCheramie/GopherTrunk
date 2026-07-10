package mp3

import "encoding/binary"

// This file adds a LAME-style Xing/Info header frame to Shine's raw output.
//
// Shine emits bare MPEG Layer-III frames with no Xing/LAME info frame and no
// ID3 tag. On short clips (an individual radio call) ffmpeg's mp3 demuxer scores
// the format low-confidence from the first frame, then tries to confirm by
// seeking to a computed next-frame offset — which, for the final frame of a
// short clip, lands past end-of-file and is treated as fatal
// ("Failed to read frame size: Could not seek to …" / "Invalid data found").
// LAME (and therefore Trunk Recorder) always writes an Info frame as the very
// first frame precisely so demuxers read the stream length instead of probing.
// We reproduce that frame in pure Go, keeping the package's zero-CGO guarantee.

// l3BitrateMPEG1 and l3BitrateMPEG2 map a 4-bit bitrate index to kbps for
// MPEG-1 and MPEG-2/2.5 Layer III respectively. Index 0 (free) and 15 (bad)
// are unused by the encoder.
var (
	l3BitrateMPEG1 = [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
	l3BitrateMPEG2 = [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}
	srMPEG1        = [4]int{44100, 48000, 32000, 0}
	srMPEG2        = [4]int{22050, 24000, 16000, 0}
	srMPEG25       = [4]int{11025, 12000, 8000, 0}
)

// bitrateIndex returns the 4-bit Layer-III bitrate index for kbps at the given
// sample rate, or 0 (the "free" slot) if kbps is not a legal value there.
func bitrateIndex(kbps, sampleRate int) int {
	tbl := l3BitrateMPEG2
	if sampleRate >= 32000 { // MPEG-1 family
		tbl = l3BitrateMPEG1
	}
	for i, v := range tbl {
		if v == kbps {
			return i
		}
	}
	return 0
}

// frameHeader is the decoded content of a 4-byte MPEG Layer-III frame header.
type frameHeader struct {
	versionBits uint8 // 0=MPEG-2.5, 2=MPEG-2, 3=MPEG-1 (raw header field)
	bitrate     int   // bits per second
	sampleRate  int   // Hz
	padding     int   // 0 or 1 padding slot
	mono        bool  // single-channel (mode bits == 11)
}

// parseFrameHeader decodes the 4-byte Layer-III header at the start of b.
// It returns ok=false for anything that is not a valid Layer-III frame header
// (bad sync, reserved version, non-Layer-III, or free/reserved bitrate/rate).
func parseFrameHeader(b []byte) (frameHeader, bool) {
	if len(b) < 4 || b[0] != 0xFF || b[1]&0xE0 != 0xE0 {
		return frameHeader{}, false
	}
	verBits := (b[1] >> 3) & 0x3
	layerBits := (b[1] >> 1) & 0x3
	if verBits == 1 || layerBits != 1 { // reserved version / not Layer III
		return frameHeader{}, false
	}
	brIndex := (b[2] >> 4) & 0xF
	srIndex := (b[2] >> 2) & 0x3
	var br, sr int
	if verBits == 3 { // MPEG-1
		br = l3BitrateMPEG1[brIndex]
		sr = srMPEG1[srIndex]
	} else {
		br = l3BitrateMPEG2[brIndex]
		if verBits == 2 {
			sr = srMPEG2[srIndex]
		} else {
			sr = srMPEG25[srIndex]
		}
	}
	if br == 0 || sr == 0 {
		return frameHeader{}, false
	}
	return frameHeader{
		versionBits: verBits,
		bitrate:     br * 1000,
		sampleRate:  sr,
		padding:     int((b[2] >> 1) & 0x1),
		mono:        (b[3]>>6)&0x3 == 0x3,
	}, true
}

// baseLen is the frame length in bytes excluding the padding slot:
// 144·bitrate/rate for MPEG-1 (1152 samples/frame), 72·bitrate/rate for
// MPEG-2/2.5 (576 samples/frame).
func (h frameHeader) baseLen() int {
	coef := 72
	if h.versionBits == 3 {
		coef = 144
	}
	return coef * h.bitrate / h.sampleRate
}

// frameLen is the total frame length in bytes including any padding slot.
func (h frameHeader) frameLen() int { return h.baseLen() + h.padding }

// sideInfoLen is the length in bytes of the side-information region that
// follows the 4-byte header, per MPEG version and channel count.
func (h frameHeader) sideInfoLen() int {
	if h.versionBits == 3 { // MPEG-1
		if h.mono {
			return 17
		}
		return 32
	}
	if h.mono {
		return 9
	}
	return 17
}

// countFrames returns the number of whole MPEG Layer-III frames in a bare
// Shine stream. It stops at the first byte that is not a valid frame header.
func countFrames(stream []byte) int {
	n := 0
	for pos := 0; pos+4 <= len(stream); {
		h, ok := parseFrameHeader(stream[pos:])
		if !ok {
			break
		}
		fl := h.frameLen()
		if fl <= 0 {
			break
		}
		pos += fl
		n++
	}
	return n
}

// xingInfoFrame builds a single CBR "Info" header frame that mirrors header h
// and advertises totalFrames/totalBytes for the whole stream. The audio payload
// is silent (all-zero side info and main data), so it decodes to a brief silence
// and is safe to prepend to any stream.
func xingInfoFrame(h frameHeader, headerBytes []byte, totalFrames, totalBytes int) []byte {
	frame := make([]byte, h.baseLen())
	// Reuse Shine's own 4-byte header so version/rate/mode/bitrate always match
	// the stream, but clear the padding bit — this synthetic frame is exactly
	// baseLen bytes (no padding slot).
	copy(frame[:4], headerBytes[:4])
	frame[2] &^= 0x02
	off := 4 + h.sideInfoLen()
	copy(frame[off:], "Info")                                     // CBR tag ("Xing" would signal VBR)
	binary.BigEndian.PutUint32(frame[off+4:], 0x0003)             // flags: frames + bytes present
	binary.BigEndian.PutUint32(frame[off+8:], uint32(totalFrames)) //nolint:gosec // small non-negative counts
	binary.BigEndian.PutUint32(frame[off+12:], uint32(totalBytes)) //nolint:gosec
	return frame
}

// withInfoHeader returns stream prefixed with a Xing/Info header frame that
// describes it. If stream does not begin with a parseable Layer-III frame it is
// returned unchanged.
func withInfoHeader(stream []byte) []byte {
	h, ok := parseFrameHeader(stream)
	if !ok {
		return stream
	}
	frames := countFrames(stream)
	info := xingInfoFrame(h, stream, frames+1, h.baseLen()+len(stream))
	out := make([]byte, 0, len(info)+len(stream))
	out = append(out, info...)
	out = append(out, stream...)
	return out
}
