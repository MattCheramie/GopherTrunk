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
		{"CU8", 0, false},
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
