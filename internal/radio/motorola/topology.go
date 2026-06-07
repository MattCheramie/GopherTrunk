package motorola

import "sync"

// NeighborSite is one adjacent site advertised by an Adjacent Site Status OSW,
// de-duplicated by SiteID.
type NeighborSite struct {
	SiteID uint16
	LCN    uint16
}

// TopologyConfig is a snapshot of the Motorola Type II / SmartZone system
// topology accumulated over a run: the opaque system identifier and the
// adjacent sites it advertised. Motorola has no RFSS concept.
type TopologyConfig struct {
	SystemID  uint16
	Neighbors []NeighborSite
}

// topologyModel is the mutex-guarded accumulator behind TopologyConfig — fed
// from Ingest (decode goroutine), snapshotted at EOF (engine goroutine).
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

// applyAdjacent folds an adjacent-site announcement, de-duplicating by SiteID.
func (m *topologyModel) applyAdjacent(a AdjacentSite) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := NeighborSite{SiteID: a.SiteID, LCN: a.LCN}
	for i := range m.cfg.Neighbors {
		if m.cfg.Neighbors[i].SiteID == n.SiteID {
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
