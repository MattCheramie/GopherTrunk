package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

type stuckCloseDevice struct {
	sdr.Device
	release chan struct{}
}

func (d *stuckCloseDevice) Close() error { <-d.release; return nil }

// TestCloseSDRBoundedGivesUpOnStuckClose: a close that never returns (a USB
// transfer stalled on a phone, #1256) used to keep wmbus alive after Ctrl-C,
// holding the dongle. The bounded close must return after its timeout and say
// why.
func TestCloseSDRBoundedGivesUpOnStuckClose(t *testing.T) {
	dev := &stuckCloseDevice{release: make(chan struct{})}
	defer close(dev.release)
	var out bytes.Buffer
	start := time.Now()
	closeSDRBounded(dev, 50*time.Millisecond, &out, "wmbus")
	if el := time.Since(start); el > 2*time.Second {
		t.Fatalf("closeSDRBounded blocked %s on a stuck close", el)
	}
	if !strings.Contains(out.String(), "wmbus: the SDR did not close within 50ms") {
		t.Fatalf("output = %q", out.String())
	}
}

// TestCloseSDRBoundedQuietOnNormalClose: a close that completes says nothing.
func TestCloseSDRBoundedQuietOnNormalClose(t *testing.T) {
	dev := &stuckCloseDevice{release: make(chan struct{})}
	close(dev.release)
	var out bytes.Buffer
	closeSDRBounded(dev, time.Second, &out, "wmbus")
	if out.Len() != 0 {
		t.Fatalf("output = %q, want none", out.String())
	}
}
