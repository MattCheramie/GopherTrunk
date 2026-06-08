package soapyremote

import (
	"math"
	"net"
	"testing"
)

// roundtrip serializes the values written by build through the real framing
// (header + trailer) over an in-memory pipe and returns an unpacker over the
// decoded payload.
func roundtrip(t *testing.T, build func(*packer)) *unpacker {
	t.Helper()
	c1, c2 := net.Pipe()
	p := newPacker()
	build(p)
	go func() {
		_ = p.writeTo(c1, 0)
		_ = c1.Close()
	}()
	u, err := readResponse(c2, 0)
	if err != nil {
		t.Fatalf("readResponse: %v", err)
	}
	_ = c2.Close()
	return u
}

func TestFloat64RoundTrip(t *testing.T) {
	// Values that matter for tuning: center frequencies, sample rates,
	// gains. Integers below 2^53 must round-trip bit-exactly through the
	// frexp/ldexp codec; fractionals to within a tiny epsilon.
	cases := []float64{0, 1, -1, 30.0, 49.6, 2400000, 851000000, 1296000000.5, -123.25}
	for _, want := range cases {
		u := roundtrip(t, func(p *packer) { p.f64(want) })
		got, err := u.f64()
		if err != nil {
			t.Fatalf("f64(%v): %v", want, err)
		}
		if math.Abs(got-want) > 1e-6 {
			t.Errorf("f64 round-trip: got %v, want %v", got, want)
		}
	}
}

func TestIntegerAndStringRoundTrip(t *testing.T) {
	u := roundtrip(t, func(p *packer) {
		p.call(callSetFrequency)
		p.char(dirRX)
		p.i32(-7)
		p.i64(1 << 40)
		p.str("")
		p.str("hello world")
		p.boolean(true)
	})
	if id, err := u.call(); err != nil || id != callSetFrequency {
		t.Fatalf("call: id=%d err=%v", id, err)
	}
	if c, err := u.char(); err != nil || c != dirRX {
		t.Fatalf("char: %d err=%v", c, err)
	}
	if v, err := u.i32(); err != nil || v != -7 {
		t.Fatalf("i32: %d err=%v", v, err)
	}
	if v, err := u.i64(); err != nil || v != 1<<40 {
		t.Fatalf("i64: %d err=%v", v, err)
	}
	if s, err := u.str(); err != nil || s != "" {
		t.Fatalf("str empty: %q err=%v", s, err)
	}
	if s, err := u.str(); err != nil || s != "hello world" {
		t.Fatalf("str: %q err=%v", s, err)
	}
	if b, err := u.boolean(); err != nil || !b {
		t.Fatalf("boolean: %v err=%v", b, err)
	}
}

func TestStringValues(t *testing.T) {
	u := roundtrip(t, func(p *packer) {
		p.str("CS16")
		p.str("55140")
	})
	a, err := u.str()
	if err != nil || a != "CS16" {
		t.Fatalf("str a: %q err=%v", a, err)
	}
	b, err := u.str()
	if err != nil || b != "55140" {
		t.Fatalf("str b: %q err=%v", b, err)
	}
}

func TestCheckExceptionDetectsRemoteError(t *testing.T) {
	u := roundtrip(t, func(p *packer) {
		p.raw8(tException)
		p.str("device not found")
	})
	err := u.checkException()
	if err == nil {
		t.Fatal("checkException: expected error, got nil")
	}
	if want := "device not found"; !contains(err.Error(), want) {
		t.Errorf("checkException error %q does not contain %q", err.Error(), want)
	}
}

func TestCheckExceptionPassesVoid(t *testing.T) {
	u := roundtrip(t, func(p *packer) { p.raw8(tVoid) })
	if err := u.checkException(); err != nil {
		t.Errorf("checkException on VOID: %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
