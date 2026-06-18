package phase1

import (
	"sort"
	"sync"
)

// P25 trunked-system topology model.
//
// A P25 site repeats a set of status-broadcast TSBKs — Network Status
// (0x3B), RFSS Status (0x3A), Secondary Control Channel (0x39), and
// Adjacent Site Status (0x3C) — that together describe the system's
// identity and neighbours. NetworkModel accumulates them into a
// queryable NetworkConfig, the rough equivalent of SDRtrunk's P25
// network-configuration monitor. It is fed from the control channel's
// dispatchTSBK and snapshot-read by the daemon / API.
//
// Accumulation is corroborated rather than last-write-wins: every TSBK
// that survives the trellis + CRC gate still occasionally carries a
// corrupt-but-CRC-passing payload (the control channel evaluates a grid
// of NID/TSBK alignment hypotheses per frame, which multiplies that
// exposure), and a single such frame must not poison the reported
// identity or inject a phantom band-plan slot / neighbour. Identity
// scalars (WACN/SysID/RFSS/Site/LRA) are resolved by majority vote — a
// lone observation is still surfaced, but a repeated correct value
// out-votes a one-shot wrong one. Neighbours must be seen at least
// corroborationMin times before Snapshot surfaces them.

// corroborationMin is how many independent sightings a neighbour (and, in
// BandPlan.Snapshot, a band-plan slot) needs before it is trusted. Two is
// enough to reject a one-shot CRC false positive while a real, repeating
// broadcast clears the bar within a second or two of dwell.
const corroborationMin = 2

// Channel is a (band-plan ID, channel number) pair.
type Channel struct {
	ChannelID     uint8
	ChannelNumber uint16
}

// NeighborSite is one adjacent site a radio may roam to.
type NeighborSite struct {
	RFSS          uint8
	Site          uint8
	ChannelID     uint8
	ChannelNumber uint16
}

// NetworkConfig is a snapshot of the accumulated system topology.
type NetworkConfig struct {
	WACN      uint32 // 20-bit Wide-Area Communication Network ID
	SystemID  uint16 // 12-bit System ID
	RFSS      uint8  // RF Sub-System ID of the camped site
	Site      uint8  // Site ID of the camped site
	LRA       uint8  // Location Registration Area
	Secondary []Channel
	Neighbors []NeighborSite
}

// neighborKey identifies a neighbour by (RFSS, Site).
type neighborKey struct {
	RFSS uint8
	Site uint8
}

// NetworkModel is the thread-safe accumulator behind NetworkConfig.
// The zero value is ready to use.
type NetworkModel struct {
	mu sync.Mutex

	// Identity vote tallies — Snapshot reports the most-observed value.
	wacnVotes  map[uint32]int
	sysidVotes map[uint16]int
	rfssVotes  map[uint8]int
	siteVotes  map[uint8]int
	lraVotes   map[uint8]int

	// Secondary control channels — de-duplicated on first sight (a
	// secondary CC is low-risk and not part of the corroboration scope).
	secondary []Channel

	// Neighbours — sighting count + latest channel info per (RFSS, Site).
	neighborVotes map[neighborKey]int
	neighborData  map[neighborKey]NeighborSite
}

// ensure lazily initialises the vote maps so the zero value is usable.
func (m *NetworkModel) ensure() {
	if m.wacnVotes == nil {
		m.wacnVotes = map[uint32]int{}
		m.sysidVotes = map[uint16]int{}
		m.rfssVotes = map[uint8]int{}
		m.siteVotes = map[uint8]int{}
		m.lraVotes = map[uint8]int{}
		m.neighborVotes = map[neighborKey]int{}
		m.neighborData = map[neighborKey]NeighborSite{}
	}
}

// ApplyNetworkStatus folds a Network Status Broadcast (0x3B) in.
func (m *NetworkModel) ApplyNetworkStatus(n NetworkStatusBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	// WACN/SysID are meaningless at zero (absent), so a zero never gets a
	// vote — it cannot out-rank a real value seen the same number of times.
	if n.WACN != 0 {
		m.wacnVotes[n.WACN]++
	}
	if n.SystemID != 0 {
		m.sysidVotes[n.SystemID]++
	}
	if n.LRA != 0 {
		m.lraVotes[n.LRA]++
	}
}

// ApplyRFSSStatus folds an RFSS Status Broadcast (0x3A) in.
func (m *NetworkModel) ApplyRFSSStatus(r RFSSStatusBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	if r.SystemID != 0 {
		m.sysidVotes[r.SystemID]++
	}
	// RFSS and Site of zero are valid (RFSS 0 is a real, common value), so
	// they are always counted.
	m.rfssVotes[r.RFSS]++
	m.siteVotes[r.Site]++
	if r.LRA != 0 {
		m.lraVotes[r.LRA]++
	}
}

// ApplySecondaryControlChannel folds a Secondary Control Channel
// Broadcast (0x39) in, de-duplicating by channel.
func (m *NetworkModel) ApplySecondaryControlChannel(s SecondaryControlChannelBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secondary = upsertChannel(m.secondary, Channel{s.ChannelAID, s.ChannelANumber})
	if s.ChannelBID != 0 || s.ChannelBNumber != 0 {
		m.secondary = upsertChannel(m.secondary, Channel{s.ChannelBID, s.ChannelBNumber})
	}
}

// ApplyAdjacentSite folds an Adjacent Site Status Broadcast (0x3C) in,
// counting sightings per (RFSS, Site) and keeping the latest channel info.
func (m *NetworkModel) ApplyAdjacentSite(a AdjacentSiteStatusBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	key := neighborKey{RFSS: a.RFSS, Site: a.Site}
	m.neighborVotes[key]++
	m.neighborData[key] = NeighborSite{RFSS: a.RFSS, Site: a.Site, ChannelID: a.ChannelID, ChannelNumber: a.ChannelNumber}
}

// Snapshot returns a deep copy of the accumulated topology. Identity
// fields are the most-observed value; neighbours appear only once seen at
// least corroborationMin times, sorted by (RFSS, Site) for stable output.
func (m *NetworkModel) Snapshot() NetworkConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := NetworkConfig{
		WACN:      majority(m.wacnVotes),
		SystemID:  majority(m.sysidVotes),
		RFSS:      majority(m.rfssVotes),
		Site:      majority(m.siteVotes),
		LRA:       majority(m.lraVotes),
		Secondary: append([]Channel(nil), m.secondary...),
	}
	for key, c := range m.neighborVotes {
		if c >= corroborationMin {
			out.Neighbors = append(out.Neighbors, m.neighborData[key])
		}
	}
	sort.Slice(out.Neighbors, func(i, j int) bool {
		if out.Neighbors[i].RFSS != out.Neighbors[j].RFSS {
			return out.Neighbors[i].RFSS < out.Neighbors[j].RFSS
		}
		return out.Neighbors[i].Site < out.Neighbors[j].Site
	})
	return out
}

// majority returns the most-observed key in a vote tally, or the zero
// value when the tally is empty. Ties are resolved by the larger key so
// the result is deterministic regardless of map iteration order.
func majority[K ~uint8 | ~uint16 | ~uint32](votes map[K]int) K {
	var best K
	bestCount := 0
	for k, c := range votes {
		if c > bestCount || (c == bestCount && k > best) {
			best, bestCount = k, c
		}
	}
	return best
}

// upsertChannel appends ch to chans if not already present.
func upsertChannel(chans []Channel, ch Channel) []Channel {
	for _, e := range chans {
		if e == ch {
			return chans
		}
	}
	return append(chans, ch)
}
