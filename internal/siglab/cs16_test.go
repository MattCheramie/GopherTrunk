package siglab

import (
	"math"
	"testing"
)

// TestParseSampleFormatCS16 pins the cs16 (headerless 16-bit) flag spellings
// and the decoder shape, and guards the deliberate split from the WAV aliases:
// s16/sw16 stay WAV (a documented, tested contract), cs16/sc16/i16/raw are the
// new headerless format.
func TestParseSampleFormatCS16(t *testing.T) {
	for _, s := range []string{"cs16", "sc16", "i16", "raw", "CS16"} {
		f, err := ParseSampleFormat(s)
		if err != nil || f != FormatS16 {
			t.Fatalf("ParseSampleFormat(%q) = %v, %v; want FormatS16", s, f, err)
		}
	}
	// The WAV aliases must NOT be captured by the new format.
	for _, s := range []string{"wav", "sw16", "s16"} {
		f, err := ParseSampleFormat(s)
		if err != nil || f != FormatWAV {
			t.Fatalf("ParseSampleFormat(%q) = %v, %v; want FormatWAV (unchanged contract)", s, f, err)
		}
	}
	if _, n := FormatS16.Decoder(); n != 4 {
		t.Fatalf("FormatS16 bytes-per-sample = %d, want 4", n)
	}
	if FormatS16.String() != "cs16" {
		t.Fatalf("FormatS16.String() = %q, want cs16", FormatS16.String())
	}
}

// TestEncodeCaptureCS16RoundTrip encodes known IQ as cs16, decodes it back
// through the format's own decoder, and asserts the samples survive within
// 16-bit quantisation. It also confirms cs16 is exactly half the size of f32
// (4 vs 8 bytes/sample) — the whole point of the format.
func TestEncodeCaptureCS16RoundTrip(t *testing.T) {
	in := []complex64{
		complex(0, 0),
		complex(0.5, -0.5),
		complex(-0.25, 0.75),
		complex(1, -1),
		complex(-1, 1),
	}

	raw := EncodeCapture(in, FormatS16)
	if got, want := len(raw), len(in)*4; got != want {
		t.Fatalf("cs16 encoded size = %d bytes, want %d (4 bytes/sample)", got, want)
	}
	if f32 := EncodeCapture(in, FormatF32); len(f32) != len(raw)*2 {
		t.Fatalf("f32 size %d should be 2× cs16 size %d", len(f32), len(raw))
	}

	decode, bps := FormatS16.Decoder()
	out := make([]complex64, len(raw)/bps)
	decode(raw, out)
	if len(out) != len(in) {
		t.Fatalf("decoded %d samples, want %d", len(out), len(in))
	}

	const tol = 1.0 / 32768 // one 16-bit LSB
	for i := range in {
		if d := math.Abs(float64(real(in[i]) - real(out[i]))); d > tol {
			t.Errorf("sample %d I: in %v out %v (Δ%v > %v)", i, real(in[i]), real(out[i]), d, tol)
		}
		if d := math.Abs(float64(imag(in[i]) - imag(out[i]))); d > tol {
			t.Errorf("sample %d Q: in %v out %v (Δ%v > %v)", i, imag(in[i]), imag(out[i]), d, tol)
		}
	}
}

// TestEncodeCaptureWAVNotU8 guards the latent-bug fix: a FormatWAV encode must
// no longer silently downgrade to 8-bit u8 — it emits the 16-bit body (4
// bytes/sample), the same as cs16.
func TestEncodeCaptureWAVNotU8(t *testing.T) {
	in := []complex64{complex(0.5, -0.5), complex(-0.25, 0.75)}
	wav := EncodeCapture(in, FormatWAV)
	if got, want := len(wav), len(in)*4; got != want {
		t.Fatalf("FormatWAV encoded size = %d bytes, want %d (16-bit body, not u8's 2)", got, want)
	}
}
