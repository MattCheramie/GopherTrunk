package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
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
	written, _, err := captureToFile(context.Background(), path, siglab.FormatF32, feedChunks(5, 300), 1000, 1.0, nil)
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
	written, _, err := captureToFile(context.Background(), path, siglab.FormatU8, feedChunks(4, 300), 1000, 1.0, nil)
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

func TestCaptureToFileCS16(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cap.raw")
	written, _, err := captureToFile(context.Background(), path, siglab.FormatS16, feedChunks(4, 300), 1000, 1.0, nil)
	if err != nil {
		t.Fatalf("captureToFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != written*4 { // cs16 = 4 bytes/sample
		t.Errorf("file size = %d, want %d", info.Size(), written*4)
	}
}

func TestCaptureToFileNarrowband(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nb.raw")
	// Decimate a 48 kHz input to a ~6 kHz channel at zero offset. Feed enough
	// input (target = 48000 input samples) so the DDC produces output.
	ddc := ccdecoder.NewDownconverterWithOffset(48000, 6000, 0)
	written, _, err := captureToFile(context.Background(), path, siglab.FormatS16, feedChunks(160, 320), 48000, 1.0, ddc)
	if err != nil {
		t.Fatalf("captureToFile: %v", err)
	}
	// 48000 input samples / 8 decimation ≈ 6000 output samples — far fewer than
	// the input, and non-zero.
	if written == 0 || written > 48000/4 {
		t.Fatalf("narrowband written = %d, want a decimated (~6000) count", written)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != written*4 {
		t.Errorf("file size = %d, want %d", info.Size(), written*4)
	}
}

func TestCaptureToFileStreamEndsEarly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "short.cfile")
	// Only 600 samples available but target is 1000 → returns the partial
	// capture plus an error so the caller can report it.
	written, _, err := captureToFile(context.Background(), path, siglab.FormatF32, feedChunks(2, 300), 1000, 1.0, nil)
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
	_, _, err := captureToFile(ctx, path, siglab.FormatF32, feedChunks(5, 300), 1_000_000, 1.0, nil)
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

// gatedWriter blocks every Write until release is closed, then records the
// bytes. It models a stalled disk so a test can prove the capture drain is
// decoupled from the write.
type gatedWriter struct {
	release chan struct{}
	mu      sync.Mutex
	buf     []byte
}

func (w *gatedWriter) Write(p []byte) (int, error) {
	<-w.release
	w.mu.Lock()
	w.buf = append(w.buf, p...)
	w.mu.Unlock()
	return len(p), nil
}

func (w *gatedWriter) len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.buf)
}

// TestCaptureStreamDrainsDespiteBlockedWriter proves the fix: even when every
// disk write is blocked, the goroutine draining src keeps running and empties
// the source into the deep write buffer. On a live SDR that is what stops the
// driver from dropping chunks when storage stalls. Fails first: a synchronous
// write in the drain goroutine would leave src full until the writer unblocks.
func TestCaptureStreamDrainsDespiteBlockedWriter(t *testing.T) {
	const n, size = 100, 50 // fits captureWriterDepth
	w := &gatedWriter{release: make(chan struct{})}
	src := feedChunks(n, size)

	type res struct {
		written int64
		err     error
	}
	done := make(chan res, 1)
	go func() {
		wr, _, err := captureStream(context.Background(), w, siglab.FormatF32, src, uint32(n*size), 1.0, nil)
		done <- res{wr, err}
	}()

	// src must drain while the writer is still blocked.
	deadline := time.After(2 * time.Second)
	for len(src) > 0 {
		select {
		case <-deadline:
			t.Fatalf("src still has %d chunks while the writer is blocked — the write is not decoupled from the drain", len(src))
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if w.len() != 0 {
		t.Fatalf("writer wrote %d bytes while gated — it should be blocked", w.len())
	}

	close(w.release)
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("captureStream: %v", r.err)
		}
		if r.written != int64(n*size) {
			t.Errorf("written = %d, want %d", r.written, n*size)
		}
		if got := w.len(); got != n*size*8 {
			t.Errorf("wrote %d bytes, want %d (f32)", got, n*size*8)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("captureStream did not finish after releasing the writer")
	}
}

// errWriter fails every write.
type errWriter struct{ err error }

func (w errWriter) Write(p []byte) (int, error) { return 0, w.err }

func TestCaptureStreamSurfacesWriteError(t *testing.T) {
	src := feedChunks(10, 50)
	_, _, err := captureStream(context.Background(), errWriter{err: errors.New("disk full")}, siglab.FormatF32, src, 500, 1.0, nil)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("want a surfaced write error, got %v", err)
	}
}

func TestCaptureDropCounter(t *testing.T) {
	c := &captureDropCounter{serial: "ABC"}
	c.observe(sdr.Info{Serial: "ABC"})
	c.observe(sdr.Info{Serial: "OTHER"}) // different device: ignored
	c.observe(sdr.Info{Serial: "ABC"})
	if got := c.count(); got != 2 {
		t.Errorf("count = %d, want 2 (drops for OTHER must not be counted)", got)
	}
}

func TestSerialsOf(t *testing.T) {
	got := serialsOf([]sdr.Info{{Serial: "AAA"}, {Serial: "BBB"}})
	if got != "AAA, BBB" {
		t.Errorf("serialsOf = %q, want \"AAA, BBB\"", got)
	}
}
