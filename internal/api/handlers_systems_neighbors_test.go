package api

import (
	"encoding/json"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeTopoProvider is a SitesProvider that also implements
// NetworkTopologyProvider, returning a fixed snapshot for one system.
type fakeTopoProvider struct {
	system string
	snap   *trunking.TopologySnapshot
}

func (f fakeTopoProvider) Sites() []trunking.SiteInfo { return nil }
func (f fakeTopoProvider) Topology(system string) (*trunking.TopologySnapshot, bool) {
	if system == f.system && f.snap != nil {
		return f.snap, true
	}
	return nil, false
}

// TestGetSystemNeighbors checks that the live topology's adjacent sites are
// overlaid onto SystemDTO.Neighbors for both the detail and list endpoints,
// with band-plan-resolved downlink frequencies.
func TestGetSystemNeighbors(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	snap := &trunking.TopologySnapshot{
		WACN:     0xBEE00,
		SystemID: 0x2C2,
		RFSS:     1,
		Site:     1,
		BandPlan: []trunking.TopoBandPlanSlot{
			{ChannelID: 2, BaseHz: 450_000_000, SpacingHz: 12_500, TxOffsetHz: 10_000_000},
		},
		Neighbors: []trunking.TopoNeighborRef{
			// Site 9 first in the slice; expect it sorted after Site 7.
			{RFSS: 1, Site: 9, ChannelID: 2, ChannelNumber: 1654, FrequencyHz: 450_337_500},
			{RFSS: 1, Site: 7, ChannelID: 2, ChannelNumber: 1754, FrequencyHz: 450_962_500},
		},
	}
	prov := fakeTopoProvider{system: "MMR", snap: snap}
	systems := []trunking.System{{Name: "MMR"}}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: systems, Sites: prov})
	defer teardown()

	// Detail endpoint.
	resp := mustGet(t, base+"/api/v1/systems/MMR")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var dto SystemDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(dto.Neighbors) != 2 {
		t.Fatalf("Neighbors = %d, want 2: %+v", len(dto.Neighbors), dto.Neighbors)
	}
	// Sorted by (RFSS, Site): Site 7 before Site 9.
	if dto.Neighbors[0].Site != 7 || dto.Neighbors[1].Site != 9 {
		t.Errorf("neighbors not sorted by site: %+v", dto.Neighbors)
	}
	n := dto.Neighbors[0]
	if n.DownlinkHz != 450_962_500 {
		t.Errorf("downlink = %d, want 450962500", n.DownlinkHz)
	}
	// Uplink = downlink + band-plan tx offset (+10 MHz).
	if n.UplinkHz != 460_962_500 {
		t.Errorf("uplink = %d, want 460962500 (downlink + 10 MHz offset)", n.UplinkHz)
	}
	if n.SiteHex != "7" || n.RFSSHex != "1" {
		t.Errorf("hex renderings unexpected: rfss=%q site=%q", n.RFSSHex, n.SiteHex)
	}

	// List endpoint surfaces the same neighbours.
	listResp := mustGet(t, base+"/api/v1/systems")
	defer listResp.Body.Close()
	var list struct {
		Systems []SystemDTO `json:"systems"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Systems) != 1 || len(list.Systems[0].Neighbors) != 2 {
		t.Fatalf("list neighbors missing: %+v", list.Systems)
	}
}
