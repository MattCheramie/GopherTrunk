package sidecar

import (
	"encoding/binary"
	"fmt"
	"math"
)

// sampleFormat is the on-the-wire sample encoding. Both are little-endian and
// interleaved I then Q, matching what `gophertrunk replay` reads, so a sidecar
// can be tested by piping a capture file into it.
type sampleFormat int

const (
	formatCS16 sampleFormat = iota
	formatComplex64
)

func parseFormat(s string) (sampleFormat, error) {
	switch s {
	case "", "cs16", "CS16":
		return formatCS16, nil
	case "complex64", "cf32", "CF32":
		return formatComplex64, nil
	}
	return 0, fmt.Errorf("sidecar: unsupported format %q (want cs16 or complex64)", s)
}

func (f sampleFormat) name() string {
	if f == formatComplex64 {
		return "complex64"
	}
	return "cs16"
}

func (f sampleFormat) bytesPerSample() int {
	if f == formatComplex64 {
		return 8
	}
	return 4
}

// convert decodes a whole number of samples. Callers guarantee the length is a
// multiple of bytesPerSample; a partial trailing sample is dropped rather than
// mis-decoded.
func (f sampleFormat) convert(buf []byte) []complex64 {
	if f == formatComplex64 {
		return convertComplex64(buf)
	}
	return convertCS16(buf)
}

// convertCS16 decodes interleaved int16 I/Q. The 32768 divisor matches the
// other drivers, so a level measured on a sidecar stream is comparable to one
// measured on a SoapyRemote stream.
func convertCS16(buf []byte) []complex64 {
	n := len(buf) / 4
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		o := i * 4
		iv := int16(binary.LittleEndian.Uint16(buf[o:]))
		qv := int16(binary.LittleEndian.Uint16(buf[o+2:]))
		out[i] = complex(float32(iv)/32768, float32(qv)/32768)
	}
	return out
}

// convertComplex64 decodes interleaved float32 I/Q. math.Float32frombits, not
// an unsafe.Pointer reinterpret: the bit pattern is the same, and this one is
// defined behaviour that a future compiler cannot rearrange.
func convertComplex64(buf []byte) []complex64 {
	n := len(buf) / 8
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		o := i * 8
		iv := math.Float32frombits(binary.LittleEndian.Uint32(buf[o:]))
		qv := math.Float32frombits(binary.LittleEndian.Uint32(buf[o+4:]))
		out[i] = complex(iv, qv)
	}
	return out
}
