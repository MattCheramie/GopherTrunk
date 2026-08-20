package main

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

func TestParseIQCaptureSpec(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    iqCaptureSpec
		wantErr string // substring; empty = expect success
	}{
		{
			name: "empty input yields zero spec (flag not set)",
			in:   "",
			want: iqCaptureSpec{},
		},
		{
			name: "minimum required keys, default format",
			in:   "serial=76361606,path=mmr.cfile,seconds=10",
			want: iqCaptureSpec{Serial: "76361606", Path: "mmr.cfile", Seconds: 10, Format: "f32"},
		},
		{
			name: "explicit f32 format",
			in:   "serial=ABC,path=x.cfile,seconds=5,format=f32",
			want: iqCaptureSpec{Serial: "ABC", Path: "x.cfile", Seconds: 5, Format: "f32"},
		},
		{
			name: "u8 format",
			in:   "serial=ABC,path=x.bin,seconds=5,format=u8",
			want: iqCaptureSpec{Serial: "ABC", Path: "x.bin", Seconds: 5, Format: "u8"},
		},
		{
			name: "decimate factor parsed",
			in:   "serial=ABC,path=x.cs16,seconds=5,format=cs16,decimate=5",
			want: iqCaptureSpec{Serial: "ABC", Path: "x.cs16", Seconds: 5, Format: "cs16", Decimate: 5},
		},
		{
			name:    "decimate zero rejected",
			in:      "serial=A,path=x,seconds=1,decimate=0",
			wantErr: "decimate",
		},
		{
			name:    "decimate negative rejected",
			in:      "serial=A,path=x,seconds=1,decimate=-2",
			wantErr: "decimate",
		},
		{
			name:    "decimate non-integer rejected",
			in:      "serial=A,path=x,seconds=1,decimate=2.5",
			wantErr: "decimate",
		},
		{
			name: "whitespace around keys + values is trimmed",
			in:   " serial = AAA , path = / tmp / iq , seconds = 3 ",
			want: iqCaptureSpec{Serial: "AAA", Path: "/ tmp / iq", Seconds: 3, Format: "f32"},
		},
		{
			name:    "missing serial",
			in:      "path=x,seconds=1",
			wantErr: "serial",
		},
		{
			name:    "missing path",
			in:      "serial=A,seconds=1",
			wantErr: "path",
		},
		{
			name:    "missing seconds",
			in:      "serial=A,path=x",
			wantErr: "seconds",
		},
		{
			name:    "zero seconds rejected",
			in:      "serial=A,path=x,seconds=0",
			wantErr: "positive",
		},
		{
			name:    "negative seconds rejected",
			in:      "serial=A,path=x,seconds=-1",
			wantErr: "positive",
		},
		{
			name:    "bad format rejected",
			in:      "serial=A,path=x,seconds=1,format=wav",
			wantErr: "format",
		},
		{
			name:    "unknown key rejected",
			in:      "serial=A,path=x,seconds=1,wat=1",
			wantErr: "unknown key",
		},
		{
			name:    "malformed key=value rejected",
			in:      "serial=A,pathx,seconds=1",
			wantErr: "malformed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseIQCaptureSpec(tc.in)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (spec=%+v)", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q missing substring %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("spec = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestCaptureEncoderPassThroughIsByteIdentical pins that a nil decimator makes
// captureEncoder.write byte-for-byte identical to encoding through
// siglab.CaptureWriter directly — an undecimated capture must be unchanged.
func TestCaptureEncoderPassThroughIsByteIdentical(t *testing.T) {
	chunk := []complex64{
		complex(0.1, -0.2), complex(0.3, 0.4), complex(-0.5, 0.6), complex(0.7, -0.8),
	}
	var buf bytes.Buffer
	enc := newCaptureEncoder(&buf, siglab.FormatF32, nil)
	n, err := enc.write(chunk)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != len(chunk) {
		t.Errorf("written = %d, want %d (pass-through writes every sample)", n, len(chunk))
	}
	want := siglab.EncodeCapture(chunk, siglab.FormatF32)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("pass-through bytes differ from siglab.EncodeCapture (len %d vs %d)", buf.Len(), len(want))
	}
}

// TestCaptureEncoderDecimates is the failing-first regression for software
// decimation: with a down-converter set, captureEncoder must (1) write ~1/N of
// the input samples and (2) preserve an in-band tone while rejecting an
// out-of-band one (the anti-alias filter). Without the decimation wiring the
// encoder writes every sample at the full rate and this fails.
func TestCaptureEncoderDecimates(t *testing.T) {
	const (
		baseRate = 1_000_000.0
		factor   = 5
		total    = 40000 // input samples
		chunkLen = 4096
	)
	// Decimated Nyquist is baseRate/factor/2 = 100 kHz.
	inBandHz := 20_000.0   // passes
	outBandHz := 300_000.0 // rejected (> 100 kHz)

	run := func(toneHz float64) (writtenSamples int, meanMag float64) {
		var buf bytes.Buffer
		ddc := newCaptureDecimator(uint32(baseRate), factor)
		if ddc == nil {
			t.Fatal("newCaptureDecimator returned nil for factor 5")
		}
		enc := newCaptureEncoder(&buf, siglab.FormatF32, ddc)
		phase := 0.0
		dPhase := 2 * math.Pi * toneHz / baseRate
		for off := 0; off < total; off += chunkLen {
			n := chunkLen
			if off+n > total {
				n = total - off
			}
			chunk := make([]complex64, n)
			for i := range chunk {
				chunk[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
				phase += dPhase
			}
			w, err := enc.write(chunk)
			if err != nil {
				t.Fatalf("write: %v", err)
			}
			writtenSamples += w
		}
		// Decode the written f32 bytes back to complex64.
		dec, bytesPer := siglab.FormatF32.Decoder()
		out := make([]complex64, buf.Len()/bytesPer)
		dec(buf.Bytes(), out)
		// Skip the filter warm-up (first taps-worth) before measuring energy.
		const warm = 200
		var sum float64
		var cnt int
		for i := warm; i < len(out); i++ {
			sum += math.Hypot(float64(real(out[i])), float64(imag(out[i])))
			cnt++
		}
		if cnt > 0 {
			meanMag = sum / float64(cnt)
		}
		return writtenSamples, meanMag
	}

	written, inMag := run(inBandHz)
	// ~1/5 of the input, allowing a few samples of filter-state slack.
	wantApprox := total / factor
	if written < wantApprox-16 || written > wantApprox+16 {
		t.Errorf("decimated sample count = %d, want ~%d (input %d / %d)", written, wantApprox, total, factor)
	}
	if inMag < 0.7 || inMag > 1.3 {
		t.Errorf("in-band tone mean magnitude = %.3f, want ~1.0 (passband should be flat)", inMag)
	}

	_, outMag := run(outBandHz)
	if outMag > 0.1 {
		t.Errorf("out-of-band tone mean magnitude = %.3f, want < 0.1 (anti-alias stopband)", outMag)
	}
}

// TestNewCaptureDecimatorPassThrough confirms factor <= 1 (and an unknown base
// rate) yields no decimator, so the capture stays on the full-rate path.
func TestNewCaptureDecimatorPassThrough(t *testing.T) {
	if d := newCaptureDecimator(1_000_000, 1); d != nil {
		t.Error("factor 1 should yield no decimator (full-rate pass-through)")
	}
	if d := newCaptureDecimator(1_000_000, 0); d != nil {
		t.Error("factor 0 should yield no decimator")
	}
	if d := newCaptureDecimator(0, 5); d != nil {
		t.Error("unknown base rate should yield no decimator")
	}
	d := newCaptureDecimator(1_000_000, 5)
	if d == nil {
		t.Fatal("factor 5 should yield a decimator")
	}
	if got := d.OutRateHz(); math.Abs(got-200_000) > 1 {
		t.Errorf("OutRateHz = %.1f, want 200000 (1 MHz / 5)", got)
	}
}

// TestEncodeF32RoundTripsThroughReplay locks down the wire shape: the
// f32 cfile bytes the capture writer emits must round-trip cleanly
// through replay.go's decodeF32Replay so the captured file can be
// fed directly into `gophertrunk replay -format f32` without surprises.
func TestEncodeF32RoundTripsThroughReplay(t *testing.T) {
	in := []complex64{
		complex(0.0, 0.0),
		complex(0.25, -0.25),
		complex(-0.5, 0.75),
		complex(1.0, -1.0),
	}
	buf := siglab.EncodeCapture(in, siglab.FormatF32)

	out := make([]complex64, len(in))
	decF32, _ := siglab.FormatF32.Decoder()
	decF32(buf, out)
	for i := range in {
		if out[i] != in[i] {
			t.Errorf("[%d] decoded %v, want %v", i, out[i], in[i])
		}
	}
}

// TestEncodeU8RoundTripsThroughReplay verifies the u8 path encodes
// inversely to replay.go's decodeU8Replay (within the 8-bit
// quantisation step, ~1/128 ≈ 0.008).
func TestEncodeU8RoundTripsThroughReplay(t *testing.T) {
	in := []complex64{
		complex(0.0, 0.0),
		complex(0.25, -0.25),
		complex(-0.5, 0.5),
		complex(0.99, -0.99),
	}
	buf := siglab.EncodeCapture(in, siglab.FormatU8)

	out := make([]complex64, len(in))
	decU8, _ := siglab.FormatU8.Decoder()
	decU8(buf, out)
	const tol = 1.0 / 127.5 // one quantisation step
	for i := range in {
		dR := math.Abs(float64(real(out[i]) - real(in[i])))
		dI := math.Abs(float64(imag(out[i]) - imag(in[i])))
		if dR > tol || dI > tol {
			t.Errorf("[%d] decoded %v, want ~%v (tol=%v)", i, out[i], in[i], tol)
		}
	}
}

// TestEncodeU8Clips checks the saturating-clip behaviour: out-of-range
// inputs (|x| > 1) shouldn't wrap, they should saturate at the u8
// endpoints. Real SDR samples land in [-1, +1] but a synthesised
// test signal might exceed that.
func TestEncodeU8Clips(t *testing.T) {
	in := []complex64{
		complex(5.0, -5.0),
		complex(-10.0, 10.0),
	}
	buf := siglab.EncodeCapture(in, siglab.FormatU8)
	if buf[0] != 255 || buf[1] != 0 {
		t.Errorf("positive sat = (%d,%d), want (255,0)", buf[0], buf[1])
	}
	if buf[2] != 0 || buf[3] != 255 {
		t.Errorf("negative sat = (%d,%d), want (0,255)", buf[2], buf[3])
	}
}
