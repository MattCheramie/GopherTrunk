// Package siglab is the offline signal replay / testing / analysis core
// shared by the gophertrunk replay, analyze, gen, test and siglab-TUI
// subcommands. It drives any protocol GopherTrunk can decode through the
// same production receiver + control-channel pipelines the daemon runs
// (via the ccdecoder factory map), collects a protocol-agnostic structured
// Result, and exposes signal-quality analysis, synthesis, and a
// metadata-driven acceptance harness on top of it.
//
// The engine deliberately mirrors the daemon's IQ → DDC → pipeline chain
// so a replay lock implies an on-air lock and a replay failure makes the
// offline capture a reproducible fixture (the same contract the original
// replay subcommand established for P25 in issue #402).
package siglab

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// SampleFormat is the on-disk IQ encoding of a capture file.
type SampleFormat int

const (
	// FormatU8 is rtl_sdr's 8-bit unsigned interleaved IQ.
	FormatU8 SampleFormat = iota
	// FormatF32 is GNU Radio's interleaved little-endian float32 cfile.
	FormatF32
)

// String renders the format as the flag value operators type.
func (f SampleFormat) String() string {
	switch f {
	case FormatF32:
		return "f32"
	default:
		return "u8"
	}
}

// ParseSampleFormat maps a -format flag value to a SampleFormat. It accepts
// the same spellings the replay subcommand always has (u8, f32, plus the
// float32/cfile aliases) so existing muscle memory keeps working.
func ParseSampleFormat(s string) (SampleFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "u8", "":
		return FormatU8, nil
	case "f32", "float32", "cfile":
		return FormatF32, nil
	default:
		return FormatU8, fmt.Errorf("siglab: unknown sample format %q (want u8 or f32)", s)
	}
}

// SampleDecoder converts a byte chunk to complex64 IQ in-place into out.
type SampleDecoder func(buf []byte, out []complex64)

// Decoder returns the sample decoder + bytes-per-IQ-pair for the format.
// This is the shared replacement for replay.go's pickSampleDecoder /
// decodeU8Replay / decodeF32Replay so the replay subcommand and the siglab
// engine read captures through one implementation.
func (f SampleFormat) Decoder() (SampleDecoder, int) {
	switch f {
	case FormatF32:
		return decodeF32, 8
	default:
		return decodeU8, 2
	}
}

// decodeU8 converts rtl_sdr 8-bit unsigned interleaved IQ to complex64 in
// [-1, +1]. Mirrors the historical replay decodeU8Replay byte-for-byte.
func decodeU8(buf []byte, out []complex64) {
	n := len(buf) / 2
	for i := 0; i < n; i++ {
		ir := float32(buf[2*i]) - 127.5
		qr := float32(buf[2*i+1]) - 127.5
		out[i] = complex(ir/127.5, qr/127.5)
	}
}

// decodeF32 converts an interleaved little-endian float32 GNU Radio cfile
// to complex64. Matches the format gnuradio-companion emits on every
// platform GopherTrunk supports.
func decodeF32(buf []byte, out []complex64) {
	n := len(buf) / 8
	for i := 0; i < n; i++ {
		ir := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i:]))
		qr := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i+4:]))
		out[i] = complex(ir, qr)
	}
}
