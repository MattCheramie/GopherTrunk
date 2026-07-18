package siglab

import (
	"bytes"
	"math"
	"testing"
)

// TestCaptureWriterMatchesEncodeCapture proves the streaming CaptureWriter emits
// bytes identical to EncodeCapture of the whole capture, for every headerless
// format and regardless of how the IQ is chunked. This is what lets the live
// capture stream to disk (bounded RAM) without changing the on-disk result.
func TestCaptureWriterMatchesEncodeCapture(t *testing.T) {
	iq := make([]complex64, 5000)
	for i := range iq {
		// Include out-of-range values so the clamping paths are exercised.
		iq[i] = complex(float32(i%13)/6-1, float32(i%7)/3-1)
	}
	formats := []SampleFormat{FormatU8, FormatF32, FormatS16, FormatWAV}
	chunkings := [][]int{
		{5000},    // single chunk
		{1, 4999}, // tiny first chunk
		{1000, 1000, 1000, 1000, 1000},
		{999, 1, 2500, 0, 1500}, // includes an empty chunk
	}
	for _, format := range formats {
		want := EncodeCapture(iq, format)
		for _, sizes := range chunkings {
			var buf bytes.Buffer
			enc := NewCaptureWriter(&buf, format)
			off := 0
			var total int64
			for _, n := range sizes {
				if err := enc.Write(iq[off : off+n]); err != nil {
					t.Fatalf("format %v: Write: %v", format, err)
				}
				off += n
				total += int64(n)
			}
			if enc.Samples() != total {
				t.Errorf("format %v chunks %v: Samples() = %d, want %d", format, sizes, enc.Samples(), total)
			}
			if !bytes.Equal(buf.Bytes(), want) {
				t.Errorf("format %v chunks %v: streamed %d bytes != EncodeCapture %d bytes",
					format, sizes, buf.Len(), len(want))
			}
		}
	}
}

// TestStreamDownconverterMatchesOneShot proves the streaming narrowband DDC,
// driven chunk-by-chunk, produces the same output as the one-shot Downconvert of
// the whole signal — the property that lets narrowband capture stream without
// buffering the full wideband grab.
func TestStreamDownconverterMatchesOneShot(t *testing.T) {
	const (
		inRate   = 2_400_000.0
		offset   = 100_000.0
		targetBW = 50_000.0
	)
	iq := make([]complex64, 240_000) // 0.1 s
	for i := range iq {
		// A tone offset from DC so the mixer + decimator actually do work.
		th := 2 * math.Pi * offset * float64(i) / inRate
		iq[i] = complex(float32(math.Cos(th)), float32(math.Sin(th)))
	}

	want, wantRate := Downconvert(iq, inRate, offset, targetBW)

	sd := NewStreamDownconverter(inRate, offset, targetBW)
	if sd.OutRateHz() != wantRate {
		t.Fatalf("OutRateHz = %g, want %g", sd.OutRateHz(), wantRate)
	}
	var got []complex64
	for _, chunk := range chunkComplex(iq, 7) {
		out := sd.Process(chunk)
		got = append(got, out...) // copy out before the next Process reuses it
	}
	if len(got) != len(want) {
		t.Fatalf("streamed %d samples, one-shot %d samples", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sample %d: streamed %v != one-shot %v", i, got[i], want[i])
		}
	}
}

func chunkComplex(iq []complex64, n int) [][]complex64 {
	if n < 1 {
		n = 1
	}
	size := (len(iq) + n - 1) / n
	var out [][]complex64
	for i := 0; i < len(iq); i += size {
		end := i + size
		if end > len(iq) {
			end = len(iq)
		}
		out = append(out, iq[i:end])
	}
	return out
}
