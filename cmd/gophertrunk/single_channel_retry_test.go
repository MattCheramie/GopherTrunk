package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// flakyTuner fails its first n SetCenterFreq calls with the error the
// #1184 rig logged, then succeeds.
type flakyTuner struct {
	mu    sync.Mutex
	fails int
	calls int
}

func (f *flakyTuner) SetCenterFreq(hz uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.fails {
		return errors.New("rtlsdr: SetCenterFreq(462562500): r82xx SetFreq: setPLL(466132500): tried chunk sizes 16,8,4; all stalled: after 1 retry on stall: rtl2832u: I2CWrite addr=0x34: broken pipe")
	}
	return nil
}

func (f *flakyTuner) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func withFastRetryBackoff(t *testing.T) {
	t.Helper()
	old := singleChannelRetryBackoff
	singleChannelRetryBackoff = func(int) time.Duration { return time.Millisecond }
	t.Cleanup(func() { singleChannelRetryBackoff = old })
}

// TestTuneWithRetryRecoversFromStartupStall is the #1184 regression: the
// FleetSync receiver's first tune came back EPIPE (an RTL-SDR
// control-pipe stall at start-up) and the daemon abandoned the receiver
// for its whole lifetime. A tune that fails must be retried until it
// succeeds.
func TestTuneWithRetryRecoversFromStartupStall(t *testing.T) {
	withFastRetryBackoff(t)
	tuner := &flakyTuner{fails: 3}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tuneWithRetry(ctx, log, "fleetsync", tuner, 462_562_500); err != nil {
		t.Fatalf("tuneWithRetry: %v", err)
	}
	if got := tuner.count(); got != 4 {
		t.Fatalf("SetCenterFreq called %d times, want 4 (3 failures + the success)", got)
	}
}

// TestTuneWithRetryStopsWhenContextEnds pins that a permanently failing
// tune does not spin past shutdown: the retry loop returns ctx.Err().
func TestTuneWithRetryStopsWhenContextEnds(t *testing.T) {
	withFastRetryBackoff(t)
	tuner := &flakyTuner{fails: 1 << 30}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := tuneWithRetry(ctx, log, "mdc1200", tuner, 154_000_000)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context deadline", err)
	}
	if tuner.count() < 2 {
		t.Fatalf("SetCenterFreq called %d times, want repeated retries before the deadline", tuner.count())
	}
}

// TestSingleChannelRetryBackoffIsBoundedAndGrows pins the schedule the
// helpers sleep on: doubling from 500 ms, capped at 30 s, never zero.
func TestSingleChannelRetryBackoffIsBoundedAndGrows(t *testing.T) {
	prev := time.Duration(0)
	for attempt := 0; attempt < 12; attempt++ {
		d := singleChannelRetryBackoff(attempt)
		if d <= 0 {
			t.Fatalf("attempt %d: backoff %v", attempt, d)
		}
		if d < prev {
			t.Fatalf("attempt %d: backoff %v shrank from %v", attempt, d, prev)
		}
		if d > 30*time.Second {
			t.Fatalf("attempt %d: backoff %v exceeds the 30 s ceiling", attempt, d)
		}
		prev = d
	}
	if singleChannelRetryBackoff(0) != 500*time.Millisecond {
		t.Fatalf("first backoff = %v, want 500ms", singleChannelRetryBackoff(0))
	}
}
