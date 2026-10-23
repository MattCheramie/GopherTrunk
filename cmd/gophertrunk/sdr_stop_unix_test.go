//go:build unix

package main

import (
	"io"
	"syscall"
	"testing"
	"time"
)

// TestSDRStopContextEndsOnTermAndHup: `pkill gophertrunk` (SIGTERM) and a
// closed terminal (SIGHUP) stop a live SDR command like Ctrl-C does, with its
// summary, instead of killing it outright (#1256).
func TestSDRStopContextEndsOnTermAndHup(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGHUP} {
		ctx, stop := sdrStopContext("test", io.Discard)
		if err := syscall.Kill(syscall.Getpid(), sig); err != nil {
			t.Fatal(err)
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
			t.Fatalf("%v did not end the context", sig)
		}
		stop()
	}
}
