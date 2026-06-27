package trunking

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// SiteInfo records one P25 site the control-channel decoder has
// identified over the air. It is the row type of the SiteTracker's
// snapshot — the data behind GET /api/v1/sites (issue #698). Name is
// not set here; the API layer merges operator-supplied names from the
// system config onto matching rows.
type SiteInfo struct {
	System           string `json:"system"`
	RFSSID           uint8  `json:"rfss_id"`
	SiteID           uint8  `json:"site_id"`
	ControlChannelHz uint32 `json:"control_channel_hz,omitempty"`
	// ControlChannelCarrierOffsetHz is the demod's measured carrier offset (Hz)
	// of the locked control carrier relative to the tuned centre, as of LastSeen.
	// A large value flags a control lock sitting well off the configured
	// frequency — e.g. an adjacent site bleeding through (issue #815).
	ControlChannelCarrierOffsetHz int32     `json:"control_channel_carrier_offset_hz,omitempty"`
	WACN                          uint32    `json:"wacn,omitempty"`
	SystemID                      uint16    `json:"system_id,omitempty"`
	FirstSeen                     time.Time `json:"first_seen"`
	LastSeen                      time.Time `json:"last_seen"`
}

// SiteTracker maintains a live table of the P25 sites GT has decoded
// from the control channel. It subscribes to the events bus and folds
// in every events.KindSiteUpdate (published when the decoder parses an
// RFSS Status Broadcast), keyed by (system, RFSS, site). Unlike the
// camped-site snapshot in p25/phase1.NetworkModel — which only knows
// the current site — the tracker accumulates every site seen as the
// decoder hops control channels, so a multi-site system surfaces all
// of its sites over time. Entries never expire: a system's site roster
// is small and stable, and operators want the full list even for sites
// that have gone quiet.
type SiteTracker struct {
	bus *events.Bus
	now func() time.Time

	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	mu    sync.Mutex
	sites map[siteKey]*SiteInfo
	// topo holds the latest full topology snapshot per system name, so the
	// live network-configuration report (GET /api/v1/systems/{name}/report)
	// can be rendered without reaching into the decoder. It reflects the most
	// recently camped site of each system (identity + that site's control
	// channels, neighbours, and the system band plan).
	topo map[string]*TopologySnapshot
}

// siteKey identifies a site within the tracker table.
type siteKey struct {
	system string
	rfss   uint8
	site   uint8
}

// SiteTrackerOptions configure a tracker.
type SiteTrackerOptions struct {
	Bus *events.Bus
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// NewSiteTracker validates opts and returns a tracker that has already
// subscribed to the bus.
func NewSiteTracker(opts SiteTrackerOptions) (*SiteTracker, error) {
	if opts.Bus == nil {
		return nil, errors.New("trunking: SiteTracker requires an events.Bus")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	t := &SiteTracker{
		bus:     opts.Bus,
		now:     opts.Now,
		runDone: make(chan struct{}),
		sites:   make(map[siteKey]*SiteInfo),
		topo:    make(map[string]*TopologySnapshot),
	}
	t.sub = opts.Bus.Subscribe()
	return t, nil
}

// Run drains site-update events until ctx cancels or the bus closes.
func (t *SiteTracker) Run(ctx context.Context) error {
	defer close(t.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-t.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind == events.KindSiteUpdate {
				if u, ok := ev.Payload.(SiteUpdate); ok {
					t.observe(u)
				}
			}
		}
	}
}

// observe records (or refreshes) a discovered site.
func (t *SiteTracker) observe(u SiteUpdate) {
	key := siteKey{system: u.System, rfss: u.RFSSID, site: u.SiteID}
	t.mu.Lock()
	defer t.mu.Unlock()
	s := t.sites[key]
	if s == nil {
		s = &SiteInfo{
			System:    u.System,
			RFSSID:    u.RFSSID,
			SiteID:    u.SiteID,
			FirstSeen: t.now(),
		}
		t.sites[key] = s
	}
	if u.ControlChannelHz != 0 {
		s.ControlChannelHz = u.ControlChannelHz
	}
	// Always refresh the live carrier offset (zero is a valid reading — on
	// frequency, or a demod path with no AFC stage), so the surfaced value
	// tracks the most recent lock (issue #815).
	s.ControlChannelCarrierOffsetHz = u.ControlChannelCarrierOffsetHz
	if u.WACN != 0 {
		s.WACN = u.WACN
	}
	if u.SystemID != 0 {
		s.SystemID = u.SystemID
	}
	if u.Topology != nil && !u.Topology.Empty() {
		t.topo[u.System] = u.Topology
	}
	s.LastSeen = t.now()
}

// Report renders the human-readable network-configuration report for the named
// system from its most recent topology snapshot. The bool is false when no
// topology has been observed for the system yet.
func (t *SiteTracker) Report(system string) (string, bool) {
	t.mu.Lock()
	snap := t.topo[system]
	t.mu.Unlock()
	if snap == nil {
		return "", false
	}
	return FormatNetworkReport(ReportFromTopology(snap)), true
}

// Topology returns the most recent topology snapshot for the named system, or
// (nil, false) when none has been observed yet. observe replaces the stored
// pointer wholesale rather than mutating it in place (each update carries the
// decoder's freshly built snapshot), so the returned value is safe to read
// without further copying. Used by the API to overlay the live neighbour list
// onto a system's DTO.
func (t *SiteTracker) Topology(system string) (*TopologySnapshot, bool) {
	t.mu.Lock()
	snap := t.topo[system]
	t.mu.Unlock()
	return snap, snap != nil
}

// Snapshot returns every tracked site, ordered by system then RFSS/Site
// so the listing is stable across calls.
func (t *SiteTracker) Snapshot() []SiteInfo {
	t.mu.Lock()
	out := make([]SiteInfo, 0, len(t.sites))
	for _, s := range t.sites {
		out = append(out, *s)
	}
	t.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].System != out[j].System {
			return out[i].System < out[j].System
		}
		if out[i].RFSSID != out[j].RFSSID {
			return out[i].RFSSID < out[j].RFSSID
		}
		return out[i].SiteID < out[j].SiteID
	})
	return out
}

// Len returns the number of tracked sites.
func (t *SiteTracker) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.sites)
}

// Close releases the bus subscription and waits for Run to drain.
func (t *SiteTracker) Close() error {
	t.closeOnce.Do(func() {
		t.sub.Close()
		select {
		case <-t.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
