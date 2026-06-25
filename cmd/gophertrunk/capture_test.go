package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// feedChunks streams n chunks of the given size onto a fresh channel and
// closes it, so captureToFile sees a finite IQ source.
func feedChunks(n, size int) <-chan []complex64 {
	src := make(chan []complex64, n)
	for i := 0; i < n; i++ {
		chunk := make([]complex64, size)
		for j := range chunk {
			chunk[j] = complex(float32(i), float32(j))
		}
		src <- chunk
	}
	close(src)
	return src
}

func TestCaptureToFileF32(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cap.cfile")
	// 1000-sample target at rate=1000, 1 s; 5×300 = 1500 ≥ 1000.
	written, err := captureToFile(context.Background(), path, siglab.FormatF32, feedChunks(5, 300), 1000, 1.0)
	if err != nil {
		t.Fatalf("captureToFile: %v", err)
	}
	if written < 1000 {
		t.Fatalf("written = %d, want ≥ 1000", written)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != written*8 { // f32 = 8 bytes/sample
		t.Errorf("file size = %d, want %d", info.Size(), written*8)
	}
}

func TestCaptureToFileU8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cap.bin")
	written, err := captureToFile(context.Background(), path, siglab.FormatU8, feedChunks(4, 300), 1000, 1.0)
	if err != nil {
		t.Fatalf("captureToFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != written*2 { // u8 = 2 bytes/sample
		t.Errorf("file size = %d, want %d", info.Size(), written*2)
	}
}

func TestCaptureToFileStreamEndsEarly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "short.cfile")
	// Only 600 samples available but target is 1000 → returns the partial
	// capture plus an error so the caller can report it.
	written, err := captureToFile(context.Background(), path, siglab.FormatF32, feedChunks(2, 300), 1000, 1.0)
	if err == nil {
		t.Fatalf("expected an error when the stream ends before the target")
	}
	if written != 600 {
		t.Errorf("written = %d, want 600", written)
	}
}

func TestCaptureToFileContextCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cancelled.cfile")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before the first read
	_, err := captureToFile(ctx, path, siglab.FormatF32, feedChunks(5, 300), 1_000_000, 1.0)
	if err == nil {
		t.Fatalf("expected ctx.Err() when the context is already cancelled")
	}
}

func TestCaptureSampleRateHint(t *testing.T) {
	cases := []struct {
		name     string
		rateHz   uint32
		protocol string
		wantHint bool
	}{
		{"default rate, p25 — silent", 2_400_000, "p25", false},
		{"2.5 MS/s, p25 — silent (recommended R2 rate)", 2_500_000, "p25", false},
		{"exactly at threshold — silent", 4_000_000, "p25", false},
		{"just above threshold — fires", 4_000_001, "p25", true},
		{"10 MS/s, p25 — fires (the #771 trap)", 10_000_000, "p25", true},
		{"6 MS/s, dmr — fires", 6_000_000, "dmr", true},
		{"10 MS/s, mixed-case protocol — fires", 10_000_000, "P25", true},
		{"10 MS/s, no protocol — fires (the #771 repro command)", 10_000_000, "", true},
		{"10 MS/s, explicit non-narrowband — silent", 10_000_000, "wfm", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := captureSampleRateHint(tc.rateHz, tc.protocol)
			if tc.wantHint && got == "" {
				t.Fatalf("captureSampleRateHint(%d, %q) = \"\", want a hint", tc.rateHz, tc.protocol)
			}
			if !tc.wantHint && got != "" {
				t.Fatalf("captureSampleRateHint(%d, %q) = %q, want \"\"", tc.rateHz, tc.protocol, got)
			}
			if tc.wantHint {
				if !strings.Contains(got, "#771") {
					t.Errorf("hint missing issue reference: %q", got)
				}
				if !strings.Contains(got, "Choosing a sample rate") {
					t.Errorf("hint missing doc pointer: %q", got)
				}
			}
		})
	}
}

func TestSerialsOf(t *testing.T) {
	got := serialsOf([]sdr.Info{{Serial: "AAA"}, {Serial: "BBB"}})
	if got != "AAA, BBB" {
		t.Errorf("serialsOf = %q, want \"AAA, BBB\"", got)
	}
}
