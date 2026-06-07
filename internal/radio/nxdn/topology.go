package nxdn

import "sync"

// TopologyConfig is a snapshot of the NXDN single-site identity accumulated
// from RCCH Site Information. NXDN doesn't broadcast adjacent sites in a form
// this decoder accumulates, so only identity is surfaced.
type TopologyConfig struct {
	SystemID   uint16
	SiteID     uint16
	LocationID uint16
}

// topologyModel is the mutex-guarded accumulator behind TopologyConfig.
type topologyModel struct {
	mu  sync.Mutex
	cfg TopologyConfig
}

// applySiteInfo records the site identity (first non-zero of each field wins).
func (m *topologyModel) applySiteInfo(systemID, siteID, locationID uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cfg.SystemID == 0 && systemID != 0 {
		m.cfg.SystemID = systemID
	}
	if m.cfg.SiteID == 0 && siteID != 0 {
		m.cfg.SiteID = siteID
	}
	if m.cfg.LocationID == 0 && locationID != 0 {
		m.cfg.LocationID = locationID
	}
}

func (m *topologyModel) snapshot() TopologyConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}
