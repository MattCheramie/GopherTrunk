package soapyremote

import (
	"encoding/binary"
	"math"
	"testing"
)

// encodeStreamHeader builds the 24-byte datagram header (network byte order),
// mirroring what a SoapySDRServer sends. Shared with the fake server in
// driver_test.go.
func encodeStreamHeader(h streamHeader) []byte {
	b := make([]byte, streamHeaderSize)
	binary.BigEndian.PutUint32(b[0:4], h.bytes)
	binary.BigEndian.PutUint32(b[4:8], h.sequence)
	binary.BigEndian.PutUint32(b[8:12], uint32(h.elems))
	binary.BigEndian.PutUint32(b[12:16], uint32(h.flags))
	binary.BigEndian.PutUint64(b[16:24], uint64(h.time))
	return b
}

func TestStreamHeaderRoundTrip(t *testing.T) {
	want := streamHeader{bytes: 1452, sequence: 42, elems: 357, flags: 4, time: 1234567890}
	got := decodeStreamHeader(encodeStreamHeader(want))
	if got != want {
		t.Errorf("decodeStreamHeader: got %+v, want %+v", got, want)
	}
}

func TestStreamHeaderNegativeElems(t *testing.T) {
	// A negative elems is a SoapySDR error/status code, not a sample count.
	want := streamHeader{bytes: 24, sequence: 7, elems: -4, flags: 0, time: 0}
	got := decodeStreamHeader(encodeStreamHeader(want))
	if got.elems != -4 {
		t.Errorf("elems: got %d, want -4", got.elems)
	}
}

func TestConvertCS16(t *testing.T) {
	// Two samples: (16384, -16384) -> (0.5, -0.5); (32767, -32768) -> ~(1, -1).
	buf := make([]byte, 8)
	for i, v := range []int16{16384, -16384, 32767, -32768} {
		binary.LittleEndian.PutUint16(buf[2*i:], uint16(v))
	}
	got := convertCS16(buf)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if real(got[0]) != 0.5 || imag(got[0]) != -0.5 {
		t.Errorf("sample 0 = %v, want (0.5,-0.5)", got[0])
	}
	if math.Abs(float64(real(got[1]))-0.99997) > 1e-3 || imag(got[1]) != -1 {
		t.Errorf("sample 1 = %v, want ~(1,-1)", got[1])
	}
}

func TestConvertCF32(t *testing.T) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:], math.Float32bits(0.25))
	binary.LittleEndian.PutUint32(buf[4:], math.Float32bits(-0.75))
	got := convertCF32(buf)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if real(got[0]) != 0.25 || imag(got[0]) != -0.75 {
		t.Errorf("sample = %v, want (0.25,-0.75)", got[0])
	}
}

func TestParseFormat(t *testing.T) {
	for _, c := range []struct {
		in   string
		want sampleFormat
		ok   bool
	}{
		{"", formatCS16, true},
		{"CS16", formatCS16, true},
		{"CF32", formatCF32, true},
		{"cf32", formatCF32, true},
		{"CU8", sampleFormat{}, false},
	} {
		got, err := parseFormat(c.in)
		if c.ok != (err == nil) {
			t.Errorf("parseFormat(%q): err=%v, wantOK=%v", c.in, err, c.ok)
			continue
		}
		if c.ok && got != c.want {
			t.Errorf("parseFormat(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestWithNativeFormat pins the SoapyRemote reference client's scale rule
// (client/Streaming.cpp setupStream) as GopherTrunk applies it (issue #1329):
// the native full scale is used only when the CS16 wire format is the
// device's native format.
func TestWithNativeFormat(t *testing.T) {
	for _, c := range []struct {
		name   string
		f      sampleFormat
		native string
		scale  float64
		want   float64 // effective CS16 divisor
		cf32   bool
	}{
		{"pluto-12bit-cs16", formatCS16, "CS16", 2048, 2048, false},
		{"uhd-16bit-cs16", formatCS16, "CS16", 32768, 32768, false},
		// HackRF's native CS8: its driver converts to full-range CS16.
		{"native-cs8", formatCS16, "CS8", 128, 32768, false},
		// CF32 on the wire: the device driver scales to [-1, 1] itself.
		{"cf32-wire", formatCF32, "CS16", 2048, 32768, true},
		// Implausible int16 scales are ignored, never used as a divisor.
		{"zero-scale", formatCS16, "CS16", 0, 32768, false},
		{"unit-scale", formatCS16, "CS16", 1, 32768, false},
		{"oversize-scale", formatCS16, "CS16", 65536, 32768, false},
	} {
		got := c.f.withNativeFormat(c.native, c.scale)
		if got.fullScale() != c.want || got.cf32 != c.cf32 {
			t.Errorf("%s: withNativeFormat(%q, %g) = {cf32:%v fullScale:%g}, want {cf32:%v fullScale:%g}",
				c.name, c.native, c.scale, got.cf32, got.fullScale(), c.cf32, c.want)
		}
	}
}
