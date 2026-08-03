package usb

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// defaultBulkStallTimeout is how long a bulk-IN stream may go without
// delivering a single packet before the stall watchdog acts. A real
// Airspy fills a 16 KiB URB in well under a millisecond even at its
// slowest rate, so seconds of dead air unambiguously means the endpoint
// has halted — 2 s never false-positives on a live stream.
const defaultBulkStallTimeout = 2 * time.Second

// bulkStallEnv overrides the stall window (milliseconds). 0 disables the
// watchdog entirely.
const bulkStallEnv = "GT_USB_BULK_STALL_MS"

// readPipeTimeoutEnv is the opt-in that switches the darwin reaper from
// the blocking ReadPipe to ReadPipeTO with a bounded no-data timeout
// (milliseconds). A halted endpoint then returns kIOReturnTimeout so the
// reaper dies and the stream surfaces EOF instead of hanging. Off by
// default; the stall watchdog covers the default read path.
const readPipeTimeoutEnv = "GT_USB_READPIPE_TIMEOUT_MS"

// bulkStallTimeout resolves the effective stall window from the
// environment, falling back to defaultBulkStallTimeout. 0 disables the
// watchdog; a negative or unparseable value falls back to the default.
func bulkStallTimeout() time.Duration {
	v := os.Getenv(bulkStallEnv)
	if v == "" {
		return defaultBulkStallTimeout
	}
	ms, err := strconv.Atoi(v)
	if err != nil || ms < 0 {
		return defaultBulkStallTimeout
	}
	return time.Duration(ms) * time.Millisecond
}

// bulkWatchTickInterval bounds the watchdog poll/telemetry cadence: half
// the stall window (so a stall is caught within ~1.5× the timeout) but no
// coarser than 1 s so the RTLSDR_DEBUG_USB telemetry stays readable.
func bulkWatchTickInterval(stall time.Duration) time.Duration {
	iv := stall / 2
	if iv > time.Second || iv <= 0 {
		iv = time.Second
	}
	return iv
}

// readPipeTimeoutMs returns the configured ReadPipeTO no-data timeout and
// whether the opt-in is active. A missing, zero, negative, or unparseable
// value leaves the reaper on the default blocking ReadPipe.
func readPipeTimeoutMs() (uint32, bool) {
	v := os.Getenv(readPipeTimeoutEnv)
	if v == "" {
		return 0, false
	}
	ms, err := strconv.Atoi(v)
	if err != nil || ms <= 0 {
		return 0, false
	}
	return uint32(ms), true
}

// bulkStallWatch is the platform-neutral core of the bulk-IN stall
// detector wired into the macOS transport (usb_darwin.go). It tracks
// when the reaper last delivered a packet plus per-slot throughput so a
// supervisor goroutine can notice a silently-frozen endpoint: the device
// stops sending, the darwin backend's blocking ReadPipe never returns,
// and the stream wedges with no error, no EOF, and no drop — the macOS
// Airspy freeze. None of the existing safety nets catch that case
// (onStreamDead needs the reapers to *error*, the overrun drop needs
// onPacket to *block*, and Pool.RunWatchdog only reacts to a device that
// vanishes from USB enumerate), so this watch converts the silent hang
// into a real stream-death the driver can surface.
//
// The timing and counter logic is deliberately OS-independent and
// clock-injectable so it is unit-tested on any CI runner without darwin
// hardware (bulk_stall_test.go). usb_darwin.go supplies the real clock,
// ticker, and AbortPipe-based stall action.
type bulkStallWatch struct {
	startNanos   atomic.Int64 // unix nanos when the stream armed
	lastActivity atomic.Int64 // unix nanos of the most recent successful read
	totalURBs    atomic.Uint64
	totalBytes   atomic.Uint64
	stopped      atomic.Bool
	fired        atomic.Bool

	mu      sync.Mutex
	perSlot []uint64
}

// newBulkStallWatch returns a watch sized for the given number of reaper
// slots. slots may be zero (per-slot accounting is simply skipped).
func newBulkStallWatch(slots int) *bulkStallWatch {
	if slots < 0 {
		slots = 0
	}
	return &bulkStallWatch{perSlot: make([]uint64, slots)}
}

// start seeds the activity clock so a stream that never delivers a first
// packet is still caught stallTimeout after the seed instant.
func (w *bulkStallWatch) start(atNanos int64) {
	w.startNanos.Store(atNanos)
	w.lastActivity.Store(atNanos)
}

// markActivity records one successful read of n bytes on the given slot,
// using the wall clock. Called on the reaper hot path.
func (w *bulkStallWatch) markActivity(slot, n int) {
	w.markActivityAt(time.Now().UnixNano(), slot, n)
}

// markActivityAt is the clock-injected form used by tests.
func (w *bulkStallWatch) markActivityAt(atNanos int64, slot, n int) {
	w.lastActivity.Store(atNanos)
	w.totalURBs.Add(1)
	if n > 0 {
		w.totalBytes.Add(uint64(n))
	}
	w.mu.Lock()
	if slot >= 0 && slot < len(w.perSlot) {
		w.perSlot[slot]++
	}
	w.mu.Unlock()
}

// stop disarms the watch so a normal teardown (StopBulkIn/Close) never
// trips the stall path.
func (w *bulkStallWatch) stop() { w.stopped.Store(true) }

// shouldFire reports whether the stream has been idle longer than
// stallTimeout while the watch is still armed (not stopped, not already
// fired). Pure — the seam the unit test drives with an injected clock.
func (w *bulkStallWatch) shouldFire(nowNanos, stallTimeoutNanos int64) bool {
	if w.stopped.Load() || w.fired.Load() {
		return false
	}
	return nowNanos-w.lastActivity.Load() > stallTimeoutNanos
}

// run polls tick, optionally emits telemetry via onTick, and fires
// onStall exactly once when the stream stalls, then returns. now supplies
// the current unix-nanos clock; now/tick/stop are injected by tests and
// satisfied by the runtime clock + a time.Ticker in production.
func (w *bulkStallWatch) run(now func() int64, tick <-chan time.Time, stop <-chan struct{}, stallTimeoutNanos int64, onTick func(), onStall func()) {
	for {
		select {
		case <-stop:
			return
		case <-tick:
			// Sample the clock AT tick receipt, before onTick(). onTick is a
			// test-only hook that hands control back to the driver, which may
			// advance an injected clock for the next tick; reading now() after it
			// would let that next value decide THIS tick's stall (a false early
			// fire — the flake behind the CI hang in the stall-watch test). In
			// production onTick is nil, so this is behaviour-identical there.
			atTick := now()
			if onTick != nil {
				onTick()
			}
			if w.shouldFire(atTick, stallTimeoutNanos) {
				if w.fired.CompareAndSwap(false, true) {
					if onStall != nil {
						onStall()
					}
				}
				return
			}
		}
	}
}

// bulkStallStats is a point-in-time snapshot of the watch counters.
type bulkStallStats struct {
	elapsed time.Duration
	idle    time.Duration
	urbs    uint64
	bytes   uint64
	perSlot []uint64
}

// stats snapshots the counters relative to nowNanos.
func (w *bulkStallWatch) stats(nowNanos int64) bulkStallStats {
	w.mu.Lock()
	per := append([]uint64(nil), w.perSlot...)
	w.mu.Unlock()
	last := w.lastActivity.Load()
	start := w.startNanos.Load()
	return bulkStallStats{
		elapsed: time.Duration(nowNanos - start),
		idle:    time.Duration(nowNanos - last),
		urbs:    w.totalURBs.Load(),
		bytes:   w.totalBytes.Load(),
		perSlot: per,
	}
}

// telemetryLine renders a compact one-line summary for the periodic
// RTLSDR_DEBUG_USB bulk trace: elapsed, URB count, MB transferred,
// throughput, current idle gap, and the per-slot completion spread. Kept
// here (untagged) so its formatting is exercised by the portable test.
func (s bulkStallStats) telemetryLine() string {
	mb := float64(s.bytes) / (1024 * 1024)
	var mbps float64
	if s.elapsed > 0 {
		mbps = mb / s.elapsed.Seconds()
	}
	return fmt.Sprintf("elapsed=%s urbs=%d mb=%.2f throughput=%.2fMB/s idle=%s slots=%s",
		s.elapsed.Round(time.Millisecond), s.urbs, mb, mbps,
		s.idle.Round(time.Millisecond), formatSlotSpread(s.perSlot))
}

// stalledLine renders the one-shot "bulk-IN stalled" summary emitted when
// the watch fires: how long the stream was idle, how much data it carried
// before the freeze, and the per-slot completion spread that localises
// whether every reaper died together or one slot starved.
func (s bulkStallStats) stalledLine(stallTimeout time.Duration) string {
	mb := float64(s.bytes) / (1024 * 1024)
	return fmt.Sprintf("idle=%s (threshold=%s) after elapsed=%s urbs=%d mb=%.2f slots=%s",
		s.idle.Round(time.Millisecond), stallTimeout,
		s.elapsed.Round(time.Millisecond), s.urbs, mb, formatSlotSpread(s.perSlot))
}

// formatSlotSpread renders per-slot completion counts compactly. When the
// slots are perfectly balanced it collapses to "N×count"; otherwise it
// lists min/max so an uneven reaper (one slot starved while others cycled)
// is visible at a glance.
func formatSlotSpread(perSlot []uint64) string {
	if len(perSlot) == 0 {
		return "[]"
	}
	min, max := perSlot[0], perSlot[0]
	for _, v := range perSlot[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if min == max {
		return fmt.Sprintf("%dx%d", len(perSlot), min)
	}
	return fmt.Sprintf("%d slots min=%d max=%d", len(perSlot), min, max)
}
