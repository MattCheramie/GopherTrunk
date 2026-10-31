package sdr

import (
	"sync"
	"sync/atomic"
)

// iqDropObserver, when set, is invoked once per IQ chunk a driver
// discards because the primary consumer fell behind. It is process-wide
// because SDR backends self-register via init() with a zero-value driver,
// so there is no constructor seam to thread a per-device sink through.
// The daemon installs an observer once during bring-up; tools and tests
// that don't wire metrics leave it nil. The atomic pointer keeps the
// hot-path read lock-free and safe against the one-time set. Issue #402:
// live chunk drops were silent before this hook, so live IQ loss could
// not be told apart from RF problems.
var iqDropObserver atomic.Pointer[func(Info)]

// SetIQDropObserver installs fn as the process-wide IQ-drop observer. fn
// is called (on the dropping driver's reaper goroutine) once per dropped
// chunk with the device's Info, so the caller can record the
// iq_underruns_total metric and emit a rate-limited warning. Passing nil
// clears the observer. Safe to call concurrently with active streams.
func SetIQDropObserver(fn func(Info)) {
	if fn == nil {
		iqDropObserver.Store(nil)
		return
	}
	iqDropObserver.Store(&fn)
}

// NotifyIQDrop reports that the device described by info dropped one IQ
// chunk. Backends call it from their stream reaper's overrun branch. It
// is a no-op when no observer is installed. Safe for any goroutine.
func NotifyIQDrop(info Info) {
	if v, ok := iqDropCounts.Load(info.Serial); ok {
		v.(*atomic.Uint64).Add(1)
	} else {
		v, _ := iqDropCounts.LoadOrStore(info.Serial, new(atomic.Uint64))
		v.(*atomic.Uint64).Add(1)
	}
	if obs := iqDropObserver.Load(); obs != nil {
		(*obs)(info)
	}
}

// iqDropCounts holds the cumulative chunks each device serial has dropped
// (serial -> *atomic.Uint64), observer or not.
var iqDropCounts sync.Map

// IQDropCount returns the cumulative IQ chunks the device with this serial
// has dropped since the process started (0 for a serial that never dropped).
// A consumer that needs the loss over an interval — a recording that must
// say whether its IQ was complete (issue #1288) — reads it at both ends and
// takes the difference.
func IQDropCount(serial string) uint64 {
	if v, ok := iqDropCounts.Load(serial); ok {
		return v.(*atomic.Uint64).Load()
	}
	return 0
}
