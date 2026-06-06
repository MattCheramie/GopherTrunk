package soapyremote

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Stream datagram wire format (common/SoapyStreamEndpoint.cpp). Every transfer
// (one UDP datagram, or one framed unit on a TCP stream socket) begins with a
// 24-byte header. The header's integer fields are network byte order; the IQ
// sample payload that follows is copied RAW (native little-endian on the
// x86/ARM hosts that run SoapySDRServer) and is NOT byte-swapped.
const streamHeaderSize = 24

// streamHeader is the decoded 24-byte datagram header.
type streamHeader struct {
	bytes    uint32 // total transfer size including this header
	sequence uint32 // sender's monotonically increasing sequence
	elems    int32  // element count per channel, or a negative SoapySDR error code
	flags    int32  // SOAPY_SDR_* stream flags
	time     int64  // timestamp (ns)
}

func decodeStreamHeader(b []byte) streamHeader {
	return streamHeader{
		bytes:    binary.BigEndian.Uint32(b[0:4]),
		sequence: binary.BigEndian.Uint32(b[4:8]),
		elems:    int32(binary.BigEndian.Uint32(b[8:12])),
		flags:    int32(binary.BigEndian.Uint32(b[12:16])),
		time:     int64(binary.BigEndian.Uint64(b[16:24])),
	}
}

// sampleFormat identifies the wire sample encoding the server was asked to send.
type sampleFormat int

const (
	formatCS16 sampleFormat = iota // interleaved little-endian int16 I,Q
	formatCF32                     // interleaved little-endian float32 I,Q
)

// soapyName returns the SoapySDR format string for the SETUP_STREAM call.
func (f sampleFormat) soapyName() string {
	if f == formatCF32 {
		return "CF32"
	}
	return "CS16"
}

// bytesPerSample is the size of one complex sample on the wire.
func (f sampleFormat) bytesPerSample() int {
	if f == formatCF32 {
		return 8
	}
	return 4
}

func parseFormat(s string) (sampleFormat, error) {
	switch s {
	case "", "CS16", "cs16":
		return formatCS16, nil
	case "CF32", "cf32":
		return formatCF32, nil
	default:
		return 0, fmt.Errorf("soapyremote: unsupported format %q (want CS16 or CF32)", s)
	}
}

// convert turns one raw sample-payload buffer into complex64 normalized to
// roughly [-1, 1), matching every other GopherTrunk SDR backend.
func (f sampleFormat) convert(buf []byte) []complex64 {
	if f == formatCF32 {
		return convertCF32(buf)
	}
	return convertCS16(buf)
}

// convertCS16 maps interleaved little-endian int16 I,Q to complex64.
// Mirrors baseband.decodeIQ16 / rtltcp.convertU8 normalization.
func convertCS16(buf []byte) []complex64 {
	n := len(buf) / 4
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := int16(binary.LittleEndian.Uint16(buf[4*i:]))
		qv := int16(binary.LittleEndian.Uint16(buf[4*i+2:]))
		out[i] = complex(float32(iv)/32768, float32(qv)/32768)
	}
	return out
}

// convertCF32 maps interleaved little-endian float32 I,Q to complex64. The
// server already delivers samples in the conventional [-1, 1] float range.
func convertCF32(buf []byte) []complex64 {
	n := len(buf) / 8
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i:]))
		qv := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i+4:]))
		out[i] = complex(iv, qv)
	}
	return out
}
