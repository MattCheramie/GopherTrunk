//go:build !windows

package sidecar

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Opening a FIFO for reading BLOCKS until a writer attaches. Doing it inline
// would hang daemon startup for as long as the sidecar takes to come up — or
// forever if it never does — so the open is raced against the connect timeout.
func TestOpenFIFOTimesOutWithNoWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iq.fifo")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}

	drv := New([]Spec{{
		Transport: "unix_pipe", DataAddr: path, SampleRateHz: 1,
		ConnectTimeout: 200 * time.Millisecond,
	}}, testLogger())

	start := time.Now()
	if _, err := drv.Open(0); err == nil {
		t.Fatal("Open succeeded on a FIFO with no writer")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("Open blocked for %v — the FIFO open is not bounded", elapsed)
	}
}

func TestStreamIQOverAFIFO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iq.fifo")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}

	// Attach the writer first so the reader's open completes. Opening for
	// write blocks on a reader in turn, so it goes on its own goroutine.
	writerReady := make(chan *os.File, 1)
	go func() {
		w, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			close(writerReady)
			return
		}
		writerReady <- w
	}()

	drv := New([]Spec{{
		Transport: "unix_pipe", DataAddr: path, SampleRateHz: 2_400_000,
		ConnectTimeout: 3 * time.Second,
	}}, testLogger())
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	w := <-writerReady
	if w == nil {
		t.Fatal("FIFO writer did not open")
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}

	want := []complex64{complex(0.5, -0.25), complex(-0.125, 0.75)}
	if _, err := w.Write(cs16Bytes(want...)); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-ch:
		if len(got) != len(want) {
			t.Fatalf("chunk len = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if absDiff(got[i], want[i]) > 1e-3 {
				t.Errorf("sample %d = %v, want ~%v", i, got[i], want[i])
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no samples over the FIFO within 3s")
	}
}
