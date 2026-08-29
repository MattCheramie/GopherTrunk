package motorola

import "sync"

// NeighborSite is one control channel advertised by the OSW stream
// beyond the one being decoded: an alternate CC of this site
// (Adjacent=false) or an adjacent site's CC (Adjacent=true).
// De-duplicated by channel number — SmartNet OSW broadcasts carry no
// numeric site ID for these, only the channel.
type NeighborSite struct {
	LCN      uint16
	Adjacent bool
}

// TopologyConfig is a snapshot of the SmartNet / SmartZone system
// topology accumulated over a run: the system identifier and the
// alternate / adjacent control channels it advertised. Motorola has
// no RFSS concept.
type TopologyConfig struct {
	SystemID  uint16
	Neighbors []NeighborSite
}

// topologyModel is the mutex-guarded accumulator behind TopologyConfig — fed
// from the OSW sequencer (decode goroutine), snapshotted at EOF (engine
// goroutine).
type topologyModel struct {
	mu  sync.Mutex
	cfg TopologyConfig
}

// applySystemID records the system identifier (first non-zero wins).
func (m *topologyModel) applySystemID(id uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cfg.SystemID == 0 && id != 0 {
		m.cfg.SystemID = id
	}
}

// applyNeighbor folds an advertised control channel, de-duplicating
// by channel number.
func (m *topologyModel) applyNeighbor(n NeighborSite) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.cfg.Neighbors {
		if m.cfg.Neighbors[i].LCN == n.LCN {
			m.cfg.Neighbors[i] = n
			return
		}
	}
	m.cfg.Neighbors = append(m.cfg.Neighbors, n)
}

// snapshot returns a deep copy of the accumulated topology.
func (m *topologyModel) snapshot() TopologyConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.cfg
	out.Neighbors = append([]NeighborSite(nil), m.cfg.Neighbors...)
	return out
}
