package acelp

import (
	"math"
	"testing"
)

// TestLog2Known checks log2fp against exact powers of two: log2(2^k) must have
// integer exponent k and ~zero fraction.
func TestLog2Known(t *testing.T) {
	for _, k := range []int16{1, 5, 10, 15, 20, 25, 30} {
		exp, frac := log2fp(int32(1) << uint(k))
		if exp != k {
			t.Errorf("log2(2^%d): exponent %d, want %d", k, exp, k)
		}
		if frac < -4 || frac > 4 {
			t.Errorf("log2(2^%d): fraction %d, want ~0", k, frac)
		}
	}
	if e, f := log2fp(0); e != 0 || f != 0 {
		t.Errorf("log2(0) = %d,%d want 0,0", e, f)
	}
}

// TestLog2Value checks the numeric accuracy of log2fp against math.Log2 for a
// spread of magnitudes.
func TestLog2Value(t *testing.T) {
	for _, x := range []int32{3, 100, 1000, 12345, 1 << 20, 3 << 24, 0x7fffffff} {
		exp, frac := log2fp(x)
		got := float64(exp) + float64(frac)/32768.0
		want := math.Log2(float64(x))
		if math.Abs(got-want) > 0.01 {
			t.Errorf("log2fp(%d) = %.4f, want %.4f", x, got, want)
		}
	}
}

// TestPow2Log2RoundTrip checks that pow2fp inverts log2fp within a small
// relative error over a range of inputs.
func TestPow2LogRoundTrip(t *testing.T) {
	for _, x := range []int32{5, 97, 1000, 50000, 1 << 18, 1 << 24, 1 << 28} {
		exp, frac := log2fp(x)
		got := pow2fp(exp, frac)
		rel := math.Abs(float64(got-x)) / float64(x)
		if rel > 0.001 {
			t.Errorf("pow2(log2(%d)) = %d, rel err %.4f", x, got, rel)
		}
	}
}

// TestPow2Value checks pow2fp for known fractional exponents. pow2fp(k, 0) with
// exponent k returns 2^k as an integer.
func TestPow2Value(t *testing.T) {
	for _, k := range []int16{4, 8, 12, 16, 20} {
		got := pow2fp(k, 0)
		want := int32(1) << uint(k)
		if d := got - want; d < -1 || d > 1 {
			t.Errorf("pow2(%d,0) = %d, want %d", k, got, want)
		}
	}
	// half-fraction: 2^(10 + 0.5) = 2^10 * sqrt(2)
	got := pow2fp(10, 1<<14) // 0.5 in Q15
	want := int32(math.Round(1024 * math.Sqrt2))
	if d := got - want; d < -2 || d > 2 {
		t.Errorf("pow2(10, .5) = %d, want ~%d", got, want)
	}
}

// TestLExtractComp checks the double-precision split/recombine round-trips
// within the 1-LSB precision of the format.
func TestLExtractComp(t *testing.T) {
	for _, l := range []int32{0, 1, 2, 0x12345678, -0x12345678, maxWord32, minWord32 + 2} {
		hi, lo := lExtract(l)
		got := lComp(hi, lo)
		if d := got - l; d < -2 || d > 2 {
			t.Errorf("lComp(lExtract(%#x)) = %#x, off by %d", l, got, d)
		}
	}
}

// TestShift16Ops checks the shift-by-16 accumulate helpers.
func TestShift16Ops(t *testing.T) {
	if loadSh16(3) != 3<<16 {
		t.Errorf("loadSh16(3) = %#x, want %#x", loadSh16(3), 3<<16)
	}
	if addSh16(0x10000, 2) != 0x30000 {
		t.Errorf("addSh16 failed: %#x", addSh16(0x10000, 2))
	}
	if subSh16(0x30000, 1) != 0x20000 {
		t.Errorf("subSh16 failed: %#x", subSh16(0x30000, 1))
	}
}
