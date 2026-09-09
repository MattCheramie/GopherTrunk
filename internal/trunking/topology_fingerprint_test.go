package trunking

import "testing"

// TestTopologyFingerprint pins the edge-trigger contract: identical material
// content hashes equal; a change to any material field (identity, secondary CC,
// neighbor, band plan) changes the hash; display-only metadata does not; and a
// nil snapshot is stable.
func TestTopologyFingerprint(t *testing.T) {
	base := func() *TopologySnapshot {
		return &TopologySnapshot{
			WACN: 0xBEE00, SystemID: 0x123, NAC: 0x293, RFSS: 1, Site: 2,
			PrimaryCC: &TopoChannelRef{ChannelID: 1, ChannelNumber: 100, FrequencyHz: 851_000_000},
			Neighbors: []TopoNeighborRef{{RFSS: 1, Site: 7, FrequencyHz: 851_012_500}},
			BandPlan:  []TopoBandPlanSlot{{ChannelID: 1, BaseHz: 851_000_000, SpacingHz: 12500}},
		}
	}

	if (*TopologySnapshot)(nil).Fingerprint() != (*TopologySnapshot)(nil).Fingerprint() {
		t.Fatal("nil fingerprint not stable")
	}
	if base().Fingerprint() != base().Fingerprint() {
		t.Fatal("identical content must hash equal")
	}

	// Display-only metadata must NOT affect the fingerprint.
	withMeta := base()
	withMeta.SystemName = "Renamed"
	withMeta.Protocol = "p25"
	if withMeta.Fingerprint() != base().Fingerprint() {
		t.Error("display metadata changed the fingerprint")
	}

	// Each material mutation must change the hash.
	muts := map[string]func(*TopologySnapshot){
		"site":       func(s *TopologySnapshot) { s.Site = 9 },
		"wacn":       func(s *TopologySnapshot) { s.WACN = 0xBEE01 },
		"primary_cc": func(s *TopologySnapshot) { s.PrimaryCC.FrequencyHz = 851_025_000 },
		"neighbor":   func(s *TopologySnapshot) { s.Neighbors = append(s.Neighbors, TopoNeighborRef{RFSS: 2, Site: 3}) },
		"bandplan":   func(s *TopologySnapshot) { s.BandPlan[0].SpacingHz = 6250 },
	}
	baseFP := base().Fingerprint()
	for name, mut := range muts {
		s := base()
		mut(s)
		if s.Fingerprint() == baseFP {
			t.Errorf("mutation %q did not change the fingerprint", name)
		}
	}
}
