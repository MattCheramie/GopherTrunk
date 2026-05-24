package sdr

import (
	"context"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// DefaultWatchdogInterval is the polling cadence used when the caller
// doesn't override it. 30s is short enough to catch a transient USB
// drop within a single failure cycle but long enough that the periodic
// USB re-enumerate doesn't show up as background load on slow hubs.
const DefaultWatchdogInterval = 30 * time.Second

// RunWatchdog ticks every interval, re-enumerates every registered
// driver, and acts on serial-level state changes against the pool:
//
//   - A serial that the pool holds but the enumerate does NOT see
//     transitions to "missing" and surfaces a KindSDRDetached event
//     (the pool's API/TUI/web snapshot consumers see the gap).
//   - A serial that was missing in the previous tick and is now back
//     in the enumerate triggers Pool.Reacquire so the freshly re-
//     enumerated USB handle replaces the dead one before the next
//     consumer touches it.
//
// The watchdog only acts on the *transition*: a device that was always
// present and is still present is left alone (no spurious reacquires
// on healthy hardware), and a device that's been missing for many
// ticks waits for the actual reappear before any work happens.
//
// Used by the daemon to keep idle voice / control SDRs warm across
// flaky USB cycles without waiting for the next consumer to surface
// the failure. The in-stream IQ-death retry (ccdecoder retry loop,
// VoicePool.Bind reacquire) still owns the in-use case. See issue #345.
//
// Returns ctx.Err() on shutdown. Pass interval <= 0 to disable the
// watchdog entirely (returns ctx.Err() after ctx cancels, no ticks).
func (p *Pool) RunWatchdog(ctx context.Context, interval time.Duration, sampleRateHz uint32) error {
	if interval <= 0 {
		<-ctx.Done()
		return ctx.Err()
	}
	tick := time.NewTicker(interval)
	defer tick.Stop()

	// missing tracks serials the watchdog has seen as absent from the
	// latest enumerate. Membership flips to true on the first missing
	// tick and back to false (delete) on a reappear, at which point a
	// Reacquire fires. Owned by this goroutine only.
	missing := map[string]bool{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			p.watchdogTick(missing, sampleRateHz)
		}
	}
}

func (p *Pool) watchdogTick(missing map[string]bool, sampleRateHz uint32) {
	infos, errs := EnumerateAll()
	for _, err := range errs {
		// Enumerate failures are usually transient (permissions
		// race, USB bus busy); log and move on so a single bad
		// driver doesn't stall the watchdog.
		p.log.Debug("sdr: watchdog enumerate error", "err", err)
	}
	present := make(map[string]struct{}, len(infos))
	for _, info := range infos {
		present[info.Serial] = struct{}{}
	}

	// Snapshot the pool's expected serials under the read lock so we
	// don't race with concurrent Open / Close / Reacquire.
	p.mu.RLock()
	expected := make([]string, 0, len(p.entries))
	for _, e := range p.entries {
		expected = append(expected, e.Info.Serial)
	}
	p.mu.RUnlock()

	for _, serial := range expected {
		_, here := present[serial]
		if !here {
			// Pool expects this serial, enumerate doesn't see it.
			if !missing[serial] {
				missing[serial] = true
				p.log.Warn("sdr: watchdog: device missing from USB enumerate",
					"serial", serial)
				if entry := p.FindBySerial(serial); entry != nil {
					p.publish(events.KindSDRDetached, entry.Snapshot(false))
				}
			}
			continue
		}
		if missing[serial] {
			// Was gone, now back — re-acquire so the next consumer
			// touches a live handle instead of the stale one.
			delete(missing, serial)
			p.log.Info("sdr: watchdog: device reappeared; reacquiring",
				"serial", serial)
			if _, err := p.Reacquire(serial, sampleRateHz); err != nil {
				// Reacquire is also fired by the in-stream retry
				// paths and by the next consumer; a watchdog
				// failure here is not terminal — log and move on
				// so the next tick or consumer retries.
				p.log.Warn("sdr: watchdog: reacquire failed",
					"serial", serial, "err", err)
			}
		}
	}
}
