package usb

import (
	"sync/atomic"
	"testing"
	"time"
)

const testStallTimeoutNanos = int64(2 * time.Second)

// TestBulkStallWatchFiresOnceAfterTimeout is the core regression for the
// macOS Airspy silent freeze: a bulk-IN stream that stops delivering must
// be detected. The watch is driven with an injected clock and manual
// ticks so the timing is deterministic and hardware-free.
func TestBulkStallWatchFiresOnceAfterTimeout(t *testing.T) {
	w := newBulkStallWatch(4)
	w.start(0)

	var now atomic.Int64
	nowFn := func() int64 { return now.Load() }
	tick := make(chan time.Time)
	stop := make(chan struct{})
	ticked := make(chan struct{}, 1)
	stalled := make(chan struct{}, 4)
	onTick := func() { ticked <- struct{}{} }
	onStall := func() { stalled <- struct{}{} }

	go w.run(nowFn, tick, stop, testStallTimeoutNanos, onTick, onStall)

	// A tick below the threshold must NOT fire.
	now.Store(int64(1 * time.Second))
	tick <- time.Time{}
	<-ticked
	select {
	case <-stalled:
		t.Fatal("stall fired before the timeout elapsed")
	default:
	}

	// A tick past the threshold fires exactly once and returns.
	now.Store(int64(3 * time.Second))
	tick <- time.Time{}
	<-ticked
	select {
	case <-stalled:
	case <-time.After(time.Second):
		t.Fatal("stall did not fire after the idle window exceeded the timeout")
	}

	// The run loop has returned; the fired guard prevents a second fire.
	if w.shouldFire(int64(10*time.Second), testStallTimeoutNanos) {
		t.Fatal("shouldFire returned true after the watch already fired")
	}
	close(stop)
}

// TestBulkStallWatchStaysQuietWhileActive proves a healthy stream — one
// that keeps delivering packets — never trips the stall path, and that
// the throughput counters accumulate as data flows.
func TestBulkStallWatchStaysQuietWhileActive(t *testing.T) {
	w := newBulkStallWatch(2)
	w.start(0)

	var now atomic.Int64
	nowFn := func() int64 { return now.Load() }
	tick := make(chan time.Time)
	stop := make(chan struct{})
	ticked := make(chan struct{}, 1)
	fired := make(chan struct{}, 1)
	onTick := func() { ticked <- struct{}{} }
	onStall := func() { fired <- struct{}{} }

	go w.run(nowFn, tick, stop, testStallTimeoutNanos, onTick, onStall)

	// Advance well past a single stall window, but keep delivering data
	// just before each tick so the idle gap never exceeds the timeout.
	for i := int64(1); i <= 10; i++ {
		at := i * int64(time.Second)
		w.markActivityAt(at, int(i%2), 16*1024)
		now.Store(at)
		tick <- time.Time{}
		<-ticked
		select {
		case <-fired:
			t.Fatalf("stall fired on a live stream at t=%ds", i)
		default:
		}
	}
	close(stop)

	if got := w.totalURBs.Load(); got != 10 {
		t.Fatalf("totalURBs = %d, want 10", got)
	}
	if got := w.totalBytes.Load(); got != 10*16*1024 {
		t.Fatalf("totalBytes = %d, want %d", got, 10*16*1024)
	}
	s := w.stats(int64(10 * time.Second))
	if len(s.perSlot) != 2 || s.perSlot[0]+s.perSlot[1] != 10 {
		t.Fatalf("per-slot counts = %v, want two slots summing to 10", s.perSlot)
	}
}

// TestBulkStallWatchStopDisarms confirms a normal teardown disarms the
// watch so a quiescent-but-intentionally-stopped stream never fires.
func TestBulkStallWatchStopDisarms(t *testing.T) {
	w := newBulkStallWatch(1)
	w.start(0)
	w.stop()
	if w.shouldFire(int64(1*time.Hour), testStallTimeoutNanos) {
		t.Fatal("shouldFire returned true after stop() disarmed the watch")
	}
}

func TestBulkStallTimeoutEnv(t *testing.T) {
	t.Setenv(bulkStallEnv, "")
	if got := bulkStallTimeout(); got != defaultBulkStallTimeout {
		t.Errorf("unset: got %s, want %s", got, defaultBulkStallTimeout)
	}
	t.Setenv(bulkStallEnv, "500")
	if got := bulkStallTimeout(); got != 500*time.Millisecond {
		t.Errorf("500: got %s, want 500ms", got)
	}
	t.Setenv(bulkStallEnv, "0")
	if got := bulkStallTimeout(); got != 0 {
		t.Errorf("0 (disabled): got %s, want 0", got)
	}
	t.Setenv(bulkStallEnv, "garbage")
	if got := bulkStallTimeout(); got != defaultBulkStallTimeout {
		t.Errorf("garbage: got %s, want default", got)
	}
}

func TestReadPipeTimeoutEnv(t *testing.T) {
	t.Setenv(readPipeTimeoutEnv, "")
	if ms, ok := readPipeTimeoutMs(); ok || ms != 0 {
		t.Errorf("unset: got (%d,%v), want (0,false)", ms, ok)
	}
	t.Setenv(readPipeTimeoutEnv, "200")
	if ms, ok := readPipeTimeoutMs(); !ok || ms != 200 {
		t.Errorf("200: got (%d,%v), want (200,true)", ms, ok)
	}
	t.Setenv(readPipeTimeoutEnv, "0")
	if _, ok := readPipeTimeoutMs(); ok {
		t.Error("0 should not enable ReadPipeTO")
	}
}

func TestBulkWatchTickInterval(t *testing.T) {
	if got := bulkWatchTickInterval(2 * time.Second); got != time.Second {
		t.Errorf("2s stall: got %s, want 1s", got)
	}
	if got := bulkWatchTickInterval(400 * time.Millisecond); got != 200*time.Millisecond {
		t.Errorf("400ms stall: got %s, want 200ms", got)
	}
	if got := bulkWatchTickInterval(10 * time.Second); got != time.Second {
		t.Errorf("10s stall: got %s, want capped 1s", got)
	}
}

func TestFormatSlotSpread(t *testing.T) {
	tests := []struct {
		name    string
		perSlot []uint64
		want    string
	}{
		{"empty", nil, "[]"},
		{"balanced", []uint64{7, 7, 7}, "3x7"},
		{"uneven", []uint64{7, 0, 3}, "3 slots min=0 max=7"},
	}
	for _, tt := range tests {
		if got := formatSlotSpread(tt.perSlot); got != tt.want {
			t.Errorf("%s: formatSlotSpread(%v) = %q, want %q", tt.name, tt.perSlot, got, tt.want)
		}
	}
}
