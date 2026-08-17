package sidecar

import (
	"log/slog"
	"sync/atomic"
	"time"
)

// dropWarnInterval rate-limits the shed-chunk warning. A stream that is
// shedding is shedding continuously, so one line per interval carrying the
// running counts says everything a line per drop would, without drowning the
// log.
const dropWarnInterval = 5 * time.Second

// dropThrottle counts chunks shed because the DSP consumer fell behind, and
// datagrams discarded because their length was not a whole number of samples.
// Both are silent data loss otherwise: the decode just gets worse and nothing
// says why.
type dropThrottle struct {
	log  *slog.Logger
	addr string

	drops    atomic.Int64
	misalign atomic.Int64
	last     time.Time
}

func newDropThrottle(log *slog.Logger, addr string) *dropThrottle {
	return &dropThrottle{log: log, addr: addr, last: time.Now()}
}

func (t *dropThrottle) dropped() {
	t.drops.Add(1)
	t.maybeWarn(false)
}

func (t *dropThrottle) misaligned() {
	t.misalign.Add(1)
	t.maybeWarn(false)
}

// flush emits any trailing counts at teardown so a short stream's losses are
// not swallowed by the interval.
func (t *dropThrottle) flush() { t.maybeWarn(true) }

func (t *dropThrottle) maybeWarn(force bool) {
	drops, misalign := t.drops.Load(), t.misalign.Load()
	if drops == 0 && misalign == 0 {
		return
	}
	now := time.Now()
	if !force && now.Sub(t.last) < dropWarnInterval {
		return
	}
	t.last = now
	switch {
	case misalign > 0 && drops > 0:
		t.log.Warn("sidecar: shedding IQ — the decode consumer is not keeping up, and "+
			"some datagrams were not a whole number of samples (check the sidecar's "+
			"format matches sdr.sidecar[].format)",
			"addr", t.addr, "host_drops", drops, "misaligned_datagrams", misalign)
	case misalign > 0:
		t.log.Warn("sidecar: datagrams discarded — their length is not a whole number "+
			"of samples, so the sidecar's sample format does not match "+
			"sdr.sidecar[].format",
			"addr", t.addr, "misaligned_datagrams", misalign)
	default:
		t.log.Warn("sidecar: shedding IQ — the decode consumer is not draining the "+
			"stream, so the oldest queued chunk is dropped to keep latency low. "+
			"Reduce CPU load or lower the sidecar's sample rate.",
			"addr", t.addr, "host_drops", drops)
	}
}
