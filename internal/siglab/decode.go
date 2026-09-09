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
	// FormatWAV is a two-channel 16-bit signed PCM RIFF/WAVE baseband
	// recording (I in channel 1, Q in channel 2) — the layout GopherTrunk's
	// own IQWriter and SDRtrunk/SDR++ write. The sample rate is read from the
	// WAV header, overriding -sample-rate, and the 44-byte header is skipped
	// before decoding.
	FormatWAV
	// FormatS16 is headerless interleaved little-endian 16-bit signed PCM IQ
	// (I then Q, normalised ÷32768) — the same sample layout as FormatWAV but
	// with no RIFF/WAVE header. Half the size of f32 while keeping the
	// resolution of a 12–14-bit ADC (Airspy/RSPdx/USRP), where u8 would throw
	// it away. The sample rate is not carried in-band, so it comes from the
	// metadata sidecar (or -sample-rate), like u8/f32.
	FormatS16
	// FormatFLAC is a FLAC-compressed wrapper around the exact FormatS16 body
	// (two-channel 16-bit, I=left/Q=right). Lossless and ~30–50% smaller than
	// cs16 on continuous baseband; the sample rate is carried in the FLAC
	// STREAMINFO. On read the stream is decoded back to the sw16 body and the
	// rate is taken from the header, mirroring FormatWAV. Write-only for the
	// live capture subsystems (auto_record / voice_iq_debug).
	FormatFLAC
)

// String renders the format as the flag value operators type.
func (f SampleFormat) String() string {
	switch f {
	case FormatF32:
		return "f32"
	case FormatWAV:
		return "wav"
	case FormatS16:
		return "cs16"
	case FormatFLAC:
		return "flac"
	default:
		return "u8"
	}
}

// ParseSampleFormat maps a -format flag value to a SampleFormat. It accepts
// the same spellings the replay subcommand always has (u8, f32, plus the
// float32/cfile aliases) so existing muscle memory keeps working, plus wav
// (aka sw16/s16) for two-channel 16-bit RIFF/WAVE baseband recordings, and
// cs16 (aka sc16/i16/raw) for the headerless interleaved 16-bit raw variant.
//
// NOTE: s16/sw16 remain WAV aliases (a documented, tested contract — reading a
// WAV file as headerless would misinterpret its 44-byte header as samples).
// The headerless format uses the unambiguous "cs16" (complex signed 16) family.
func ParseSampleFormat(s string) (SampleFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "u8", "":
		return FormatU8, nil
	case "f32", "float32", "cfile", "cf32", "fc32":
		// cf32/fc32 are the SoapySDR / OpenWebRX+ spellings of interleaved
		// little-endian float32 IQ — the format OWRX+ emits when piping a
		// channelized stream into `replay -in -` (issue #314).
		return FormatF32, nil
	case "wav", "sw16", "s16":
		return FormatWAV, nil
	case "cs16", "sc16", "i16", "raw":
		return FormatS16, nil
	case "flac":
		return FormatFLAC, nil
	default:
		return FormatU8, fmt.Errorf("siglab: unknown sample format %q (want u8, f32, cs16, wav, or flac)", s)
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
	case FormatWAV, FormatS16, FormatFLAC:
		// Same interleaved-16-bit decoder for all three. FormatWAV's 44-byte
		// RIFF/WAVE header is stripped upstream (prepareInput in engine.go) and
		// FormatFLAC is decoded upstream into the same sw16 body (with cfg.Format
		// rewritten to FormatS16), so by the time bytes reach this decoder they
		// are headerless interleaved 16-bit IQ; FormatS16 is headerless natively.
		return decodeSW16, 4
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

// decodeSW16 converts interleaved 16-bit signed PCM (I in channel 1, Q in
// channel 2) to complex64, matching the normalisation the baseband IQWriter
// and SDRtrunk/SDR++ use (÷32768). The 44-byte WAV header is stripped before
// this decoder is fed any bytes (see the FormatWAV handling in engine.go).
func decodeSW16(buf []byte, out []complex64) {
	n := len(buf) / 4
	for i := 0; i < n; i++ {
		iv := int16(binary.LittleEndian.Uint16(buf[4*i:]))
		qv := int16(binary.LittleEndian.Uint16(buf[4*i+2:]))
		out[i] = complex(float32(iv)/32768, float32(qv)/32768)
	}
}
