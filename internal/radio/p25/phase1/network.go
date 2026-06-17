package phase1

import "sync"

// P25 trunked-system topology model.
//
// A P25 site repeats a set of status-broadcast TSBKs — Network Status
// (0x3B), RFSS Status (0x3A), Secondary Control Channel (0x39), and
// Adjacent Site Status (0x3C) — that together describe the system's
// identity and neighbours. NetworkModel accumulates them into a
// queryable NetworkConfig, the rough equivalent of SDRtrunk's P25
// network-configuration monitor. It is fed from the control channel's
// dispatchTSBK and snapshot-read by the daemon / API.

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

// NetworkModel is the thread-safe accumulator behind NetworkConfig.
// The zero value is ready to use.
type NetworkModel struct {
	mu  sync.Mutex
	cfg NetworkConfig
}

// ApplyNetworkStatus folds a Network Status Broadcast (0x3B) in.
func (m *NetworkModel) ApplyNetworkStatus(n NetworkStatusBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.WACN = n.WACN
	m.cfg.SystemID = n.SystemID
	if n.LRA != 0 {
		m.cfg.LRA = n.LRA
	}
}

// ApplyRFSSStatus folds an RFSS Status Broadcast (0x3A) in.
func (m *NetworkModel) ApplyRFSSStatus(r RFSSStatusBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.SystemID = r.SystemID
	m.cfg.RFSS = r.RFSS
	m.cfg.Site = r.Site
	m.cfg.LRA = r.LRA
}

// ApplySecondaryControlChannel folds a Secondary Control Channel
// Broadcast (0x39) in, de-duplicating by channel.
func (m *NetworkModel) ApplySecondaryControlChannel(s SecondaryControlChannelBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.Secondary = upsertChannel(m.cfg.Secondary, Channel{s.ChannelAID, s.ChannelANumber})
	if s.ChannelBID != 0 || s.ChannelBNumber != 0 {
		m.cfg.Secondary = upsertChannel(m.cfg.Secondary, Channel{s.ChannelBID, s.ChannelBNumber})
	}
}

// ApplyAdjacentSite folds an Adjacent Site Status Broadcast (0x3C) in,
// de-duplicating by (RFSS, Site).
func (m *NetworkModel) ApplyAdjacentSite(a AdjacentSiteStatusBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ns := NeighborSite{RFSS: a.RFSS, Site: a.Site, ChannelID: a.ChannelID, ChannelNumber: a.ChannelNumber}
	for i, e := range m.cfg.Neighbors {
		if e.RFSS == ns.RFSS && e.Site == ns.Site {
			m.cfg.Neighbors[i] = ns
			return
		}
	}
	m.cfg.Neighbors = append(m.cfg.Neighbors, ns)
}

// Snapshot returns a deep copy of the accumulated topology.
func (m *NetworkModel) Snapshot() NetworkConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.cfg
	out.Secondary = append([]Channel(nil), m.cfg.Secondary...)
	out.Neighbors = append([]NeighborSite(nil), m.cfg.Neighbors...)
	return out
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
