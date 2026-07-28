package trunking

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// GrantTracker keeps a bounded, most-recent-first log of the voice-channel
// grants GopherTrunk decodes off the control channel — the same
// events.KindGrant stream the SSE feed (/api/v1/events) carries live, made
// available as a pollable snapshot behind GET /api/v1/grants.
//
// It exists for issue #915: a consumer building a telemetry database wants the
// source RID off the control-channel grant the way SDRtrunk's grant log
// exposes it, without holding a long-lived SSE connection. The grant carries
// SourceID + GroupID + FrequencyHz + encryption at call-setup time for every
// protocol, so this is a protocol-agnostic "who was granted what, when" feed —
// distinct from the completed-call webhook, whose source RID depends on a
// separate voice-side decode landing.
//
// Grants are logged exactly as decoded (no source backfill): a grant whose
// TSBK carried src=0 is stored with src=0, mirroring what a control-channel
// grant log actually saw. The ring buffer is fixed-capacity, so a busy system
// (thousands of grants/hour) never grows the log without bound — the oldest
// entries fall off. The tracker is a non-blocking bus subscriber (the bus
// drops to a slow subscriber rather than stalling the publisher), and its
// per-event work is a single ring append, so it cannot become the slow
// consumer in practice.
type GrantTracker struct {
	bus *events.Bus
	now func() time.Time

	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	mu   sync.Mutex
	ring []Grant // fixed-capacity ring buffer
	head int     // index of the next write
	size int     // number of valid entries (<= len(ring))
	seq  uint64  // total grants observed since start (monotonic)
}

// GrantTrackerOptions configure a GrantTracker.
type GrantTrackerOptions struct {
	// Bus is the event bus to subscribe to. Required.
	Bus *events.Bus
	// Capacity is how many recent grants the ring retains. Default
	// DefaultGrantLogCapacity.
	Capacity int
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// DefaultGrantLogCapacity is the ring size when Capacity is unset. A few
// thousand entries covers many minutes of a busy system while staying well
// under a megabyte of retained grants.
const DefaultGrantLogCapacity = 4096

// NewGrantTracker validates opts and returns a tracker that has already
// subscribed to the bus. Call Run to start draining.
func NewGrantTracker(opts GrantTrackerOptions) (*GrantTracker, error) {
	if opts.Bus == nil {
		return nil, errors.New("trunking: GrantTracker requires an events.Bus")
	}
	if opts.Capacity <= 0 {
		opts.Capacity = DefaultGrantLogCapacity
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	t := &GrantTracker{
		bus:     opts.Bus,
		now:     opts.Now,
		runDone: make(chan struct{}),
		ring:    make([]Grant, opts.Capacity),
	}
	t.sub = opts.Bus.Subscribe()
	return t, nil
}

// Run drains grant events until ctx cancels or the bus closes.
func (t *GrantTracker) Run(ctx context.Context) error {
	defer close(t.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-t.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind == events.KindGrant {
				if g, ok := ev.Payload.(Grant); ok {
					t.record(g)
				}
			}
		}
	}
}

// record appends one grant to the ring, evicting the oldest when full.
func (t *GrantTracker) record(g Grant) {
	// Stamp a receive time if the publisher left At zero, so the log always
	// carries a usable timestamp. Clone PatchedGroups so a later mutation of
	// the caller's slice can't alias into the retained entry.
	if g.At.IsZero() {
		g.At = t.now()
	}
	if len(g.PatchedGroups) > 0 {
		g.PatchedGroups = append([]uint32(nil), g.PatchedGroups...)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ring[t.head] = g
	t.head = (t.head + 1) % len(t.ring)
	if t.size < len(t.ring) {
		t.size++
	}
	t.seq++
}

// Snapshot returns up to limit of the most-recent grants, newest first. A
// limit <= 0 returns every retained grant. The returned slice is a copy; the
// caller may retain or mutate it freely.
func (t *GrantTracker) Snapshot(limit int) []Grant {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := t.size
	if limit > 0 && limit < n {
		n = limit
	}
	out := make([]Grant, 0, n)
	// The newest entry sits at head-1; walk backwards n entries.
	for i := 0; i < n; i++ {
		idx := (t.head - 1 - i + len(t.ring)) % len(t.ring)
		out = append(out, t.ring[idx])
	}
	return out
}

// Len returns the number of grants currently retained in the ring.
func (t *GrantTracker) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.size
}

// Observed returns the total number of grants seen since start (including
// those evicted from the ring) — a simple throughput counter.
func (t *GrantTracker) Observed() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.seq
}

// Close releases the bus subscription and waits for Run to drain.
func (t *GrantTracker) Close() error {
	t.closeOnce.Do(func() {
		t.sub.Close()
		select {
		case <-t.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
