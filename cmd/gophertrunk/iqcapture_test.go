package main

import (
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
