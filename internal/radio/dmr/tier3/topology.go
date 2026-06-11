package tier3

import "sync"

// NeighborSite is one adjacent site advertised by a C_BCAST Adjacent_Site
// announcement (CSBKO 0x28, anncd_type 6), de-duplicated by SiteID.
type NeighborSite struct {
	SystemID  uint16
	SiteID    uint16
	LCN       uint16
	ColorCode uint8
}

// TopologyConfig is a snapshot of the system topology a DMR Tier III control
// channel accumulates over a run: the camped site's identity and the adjacent
// sites it advertised. It is mapped to the protocol-neutral
// trunking.TopologySnapshot by the ccdecoder pipeline.
type TopologyConfig struct {
	SystemID  uint16
	RFSS      uint8
	Site      uint8
	ColorCode uint8
	Neighbors []NeighborSite
}

// topologyModel is the mutex-guarded accumulator behind TopologyConfig. The
// control channel feeds it from the CSBK dispatch (decode goroutine); the
// signal-lab snapshots it at EOF (engine goroutine), so access is locked. The
// zero value is ready to use.
type topologyModel struct {
	mu  sync.Mutex
	cfg TopologyConfig
}

// applySystemInfo folds a vendor (Motorola Cap+) System Information CSBK:
// the camped site's SystemID/RFSS/Site. First non-zero values win.
func (m *topologyModel) applySystemInfo(si SystemInfoBroadcast) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if si.SystemID != 0 {
		m.cfg.SystemID = si.SystemID
	}
	if si.RFSSID != 0 {
		m.cfg.RFSS = si.RFSSID
	}
	if si.SiteID != 0 {
		m.cfg.Site = si.SiteID
	}
}

// applySiteParams folds the RFSS + Site number from a C_BCAST
// Gen_Site_Params announcement without disturbing the SystemID established
// by the Aloha beacon (whose raw identity field is the authoritative lock
// key). First non-zero values win.
func (m *topologyModel) applySiteParams(rfss, site uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cfg.RFSS == 0 && rfss != 0 {
		m.cfg.RFSS = rfss
	}
	if m.cfg.Site == 0 && site != 0 {
		m.cfg.Site = site
	}
}

// applyIdentity folds the SystemID + ColorCode seen on an Aloha / C_BCAST
// Gen_Site_Params CSBK.
func (m *topologyModel) applyIdentity(systemID uint16, colorCode uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cfg.SystemID == 0 && systemID != 0 {
		m.cfg.SystemID = systemID
	}
	if m.cfg.ColorCode == 0 && colorCode != 0 {
		m.cfg.ColorCode = colorCode
	}
}

// applyAdjacent folds a C_BCAST Adjacent_Site announcement, de-duplicating
// by SiteID (last write wins for a given site's details).
func (m *topologyModel) applyAdjacent(a AdjacentSiteStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := NeighborSite{SystemID: a.SystemID, SiteID: a.SiteID, LCN: a.LCN, ColorCode: a.ColorCode}
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
