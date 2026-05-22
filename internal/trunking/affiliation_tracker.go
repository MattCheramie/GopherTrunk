package trunking

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// UnitActivity records the last-observed talkgroup activity of one
// radio unit. It is the row type of the affiliation tracker's
// snapshot — the "who is on which talkgroup" view SDRtrunk surfaces.
type UnitActivity struct {
	RadioID   uint32 `json:"radio_id"`
	Talkgroup uint32 `json:"talkgroup"`
	System    string `json:"system"`
	Protocol  string `json:"protocol"`
	// Explicit is true when the association came from a decoded
	// affiliation message (P25 Group Affiliation Response); false when
	// it was observed from a voice-channel grant naming the unit as
	// the source. Both are ground truth — a grant proves the unit is
	// transmitting on that talkgroup — but the distinction is useful
	// in the UI.
	Explicit bool `json:"explicit"`
	// Registered is true once the unit has been seen in a unit
	// registration message.
	Registered bool      `json:"registered"`
	LastSeen   time.Time `json:"last_seen"`
}

// AffiliationTracker maintains a live, protocol-agnostic table of which
// radio units are active on which talkgroups. It subscribes to the
// events bus and updates the table from three sources:
//
//   - KindGrant — the grant's SourceID is transmitting on its GroupID.
//   - KindAffiliation — an explicit decoded affiliation message.
//   - KindUnitRegistration — the unit registered on a site.
//
// Because grants carry SourceID + GroupID for every protocol, the
// tracker works uniformly across P25, DMR (all tiers and vendors),
// NXDN and the rest — no per-protocol decoding is required. Entries
// expire after a configurable idle TTL.
type AffiliationTracker struct {
	bus *events.Bus
	now func() time.Time
	ttl time.Duration

	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	mu    sync.Mutex
	units map[uint32]*UnitActivity // keyed by RadioID
}

// AffiliationTrackerOptions configure a tracker.
type AffiliationTrackerOptions struct {
	Bus *events.Bus
	// TTL is how long a unit stays in the table after it was last
	// seen. Default 30 minutes.
	TTL time.Duration
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// NewAffiliationTracker validates opts and returns a tracker that has
// already subscribed to the bus.
func NewAffiliationTracker(opts AffiliationTrackerOptions) (*AffiliationTracker, error) {
	if opts.Bus == nil {
		return nil, errors.New("trunking: AffiliationTracker requires an events.Bus")
	}
	if opts.TTL <= 0 {
		opts.TTL = 30 * time.Minute
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	t := &AffiliationTracker{
		bus:     opts.Bus,
		now:     opts.Now,
		ttl:     opts.TTL,
		runDone: make(chan struct{}),
		units:   make(map[uint32]*UnitActivity),
	}
	t.sub = opts.Bus.Subscribe()
	return t, nil
}

// Run drains grant / affiliation / registration events and sweeps
// expired units until ctx cancels or the bus closes.
func (t *AffiliationTracker) Run(ctx context.Context) error {
	defer close(t.runDone)
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			t.sweep()
		case ev, ok := <-t.sub.C:
			if !ok {
				return nil
			}
			t.handle(ev)
		}
	}
}

func (t *AffiliationTracker) handle(ev events.Event) {
	switch ev.Kind {
	case events.KindGrant:
		if g, ok := ev.Payload.(Grant); ok && g.SourceID != 0 {
			t.observe(g.SourceID, g.GroupID, g.System, g.Protocol, false, false)
		}
	case events.KindAffiliation:
		if a, ok := ev.Payload.(Affiliation); ok && a.SourceID != 0 &&
			a.Response == AffiliationAccepted {
			t.observe(a.SourceID, a.GroupID, a.System, a.Protocol, true, false)
		}
	case events.KindUnitRegistration:
		if r, ok := ev.Payload.(UnitRegistration); ok && r.SourceID != 0 &&
			r.Response == RegistrationAccepted {
			t.observe(r.SourceID, 0, r.System, r.Protocol, false, true)
		}
	}
}

// observe records (or refreshes) a unit's activity. A talkgroup of 0
// (a bare registration) refreshes the unit without clobbering a known
// talkgroup association.
func (t *AffiliationTracker) observe(radioID, talkgroup uint32, system, protocol string, explicit, registered bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	u := t.units[radioID]
	if u == nil {
		u = &UnitActivity{RadioID: radioID}
		t.units[radioID] = u
	}
	if talkgroup != 0 {
		u.Talkgroup = talkgroup
		u.Explicit = explicit
	}
	if system != "" {
		u.System = system
	}
	if protocol != "" {
		u.Protocol = protocol
	}
	if registered {
		u.Registered = true
	}
	u.LastSeen = t.now()
}

// sweep drops units idle longer than the TTL.
func (t *AffiliationTracker) sweep() {
	cutoff := t.now().Add(-t.ttl)
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, u := range t.units {
		if u.LastSeen.Before(cutoff) {
			delete(t.units, id)
		}
	}
}

// Snapshot returns every tracked unit, most-recently-seen first.
func (t *AffiliationTracker) Snapshot() []UnitActivity {
	t.mu.Lock()
	out := make([]UnitActivity, 0, len(t.units))
	for _, u := range t.units {
		out = append(out, *u)
	}
	t.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastSeen.After(out[j].LastSeen)
	})
	return out
}

// UnitsOnTalkgroup returns the radio IDs currently associated with the
// given talkgroup.
func (t *AffiliationTracker) UnitsOnTalkgroup(talkgroup uint32) []uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var out []uint32
	for id, u := range t.units {
		if u.Talkgroup == talkgroup {
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Len returns the number of tracked units.
func (t *AffiliationTracker) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.units)
}

// Close releases the bus subscription and waits for Run to drain.
func (t *AffiliationTracker) Close() error {
	t.closeOnce.Do(func() {
		t.sub.Close()
		select {
		case <-t.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
