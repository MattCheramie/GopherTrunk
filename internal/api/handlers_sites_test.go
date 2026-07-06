package api

import (
	"encoding/json"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeSites is a stand-in SitesProvider for the handler test.
type fakeSites struct{ sites []trunking.SiteInfo }

func (f fakeSites) Sites() []trunking.SiteInfo { return f.sites }

func TestListSitesUnionAndNameMerge(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	// One discovered site (rfss 1 / site 1) that is also named in config,
	// and one configured-but-unseen site (rfss 2 / site 9).
	discovered := fakeSites{sites: []trunking.SiteInfo{
		{System: "MMR", RFSSID: 1, SiteID: 1, ControlChannelHz: 420012500, WACN: 0xBEE00, SystemID: 0x123,
			ControlChannelTSBKErrorRate: 8.0, ControlChannelTSBKCount: 250},
	}}
	systems := []trunking.System{{
		Name: "MMR",
		Sites: []trunking.ConfiguredSite{
			{RFSS: 1, Site: 1, Name: "Mt Anakie"},
			{RFSS: 2, Site: 9, Name: "Murradoc Hill"},
		},
	}}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: systems, Sites: discovered})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/sites")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Sites []SiteDTO `json:"sites"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Sites) != 2 {
		t.Fatalf("got %d sites, want 2 (union): %+v", len(body.Sites), body.Sites)
	}

	// Sorted by (system, rfss, site): (1,1) then (2,9).
	first := body.Sites[0]
	if first.RFSSID != 1 || first.SiteID != 1 {
		t.Fatalf("first site = %+v, want rfss 1 / site 1", first)
	}
	if first.Name != "Mt Anakie" {
		t.Errorf("discovered site name not merged: %q", first.Name)
	}
	if first.ControlChannelHz != 420012500 {
		t.Errorf("discovered control_channel_hz = %d, want 420012500", first.ControlChannelHz)
	}
	// Per-site decode quality surfaces from the TSBK stats (issue #858).
	if first.ControlChannelTSBKErrorRate == nil {
		t.Fatalf("discovered site missing control_channel_tsbk_error_rate")
	}
	if *first.ControlChannelTSBKErrorRate != 8.0 {
		t.Errorf("control_channel_tsbk_error_rate = %v, want 8.0", *first.ControlChannelTSBKErrorRate)
	}
	if first.ControlChannelTSBKCount != 250 {
		t.Errorf("control_channel_tsbk_count = %d, want 250", first.ControlChannelTSBKCount)
	}
	if first.ControlChannelDecodeQuality != "poor" {
		t.Errorf("control_channel_decode_quality = %q, want %q", first.ControlChannelDecodeQuality, "poor")
	}

	second := body.Sites[1]
	if second.RFSSID != 2 || second.SiteID != 9 || second.Name != "Murradoc Hill" {
		t.Fatalf("configured-only site wrong: %+v", second)
	}
	if second.ControlChannelHz != 0 {
		t.Errorf("unseen site should have no control_channel_hz, got %d", second.ControlChannelHz)
	}
	// A site with no TSBK data (count 0) omits the decode-quality fields
	// rather than reporting a misleading 0% / "clean".
	if second.ControlChannelTSBKErrorRate != nil {
		t.Errorf("unseen site should have no tsbk_error_rate, got %v", *second.ControlChannelTSBKErrorRate)
	}
	if second.ControlChannelTSBKCount != 0 || second.ControlChannelDecodeQuality != "" {
		t.Errorf("unseen site should have no tsbk quality, got count=%d quality=%q",
			second.ControlChannelTSBKCount, second.ControlChannelDecodeQuality)
	}
}

// fakeSitesTopo is a SitesProvider that also implements NetworkTopologyProvider,
// returning discovered rows plus one system's topology snapshot.
type fakeSitesTopo struct {
	sites  []trunking.SiteInfo
	system string
	snap   *trunking.TopologySnapshot
}

func (f fakeSitesTopo) Sites() []trunking.SiteInfo { return f.sites }
func (f fakeSitesTopo) Topology(system string) (*trunking.TopologySnapshot, bool) {
	if system == f.system && f.snap != nil {
		return f.snap, true
	}
	return nil, false
}

// TestListSitesNeighbors checks that the live topology's adjacent sites are
// overlaid onto the camped-site row of GET /api/v1/sites (issue #864), with
// band-plan-resolved downlink frequencies, and that a non-camped site of the
// same system carries no neighbours.
func TestListSitesNeighbors(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	// Camped site (rfss 1 / site 1) advertises two neighbours; a second
	// discovered site (rfss 1 / site 2) of the same system is not camped.
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
	prov := fakeSitesTopo{
		system: "MMR",
		snap:   snap,
		sites: []trunking.SiteInfo{
			{System: "MMR", RFSSID: 1, SiteID: 1, ControlChannelHz: 450_012_500},
			{System: "MMR", RFSSID: 1, SiteID: 2, ControlChannelHz: 450_137_500},
		},
	}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: []trunking.System{{Name: "MMR"}}, Sites: prov})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/sites")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Sites []SiteDTO `json:"sites"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Sites) != 2 {
		t.Fatalf("got %d sites, want 2: %+v", len(body.Sites), body.Sites)
	}

	// Sorted by (system, rfss, site): camped site (1,1) first.
	camped := body.Sites[0]
	if camped.RFSSID != 1 || camped.SiteID != 1 {
		t.Fatalf("first site = %+v, want rfss 1 / site 1", camped)
	}
	if len(camped.Neighbors) != 2 {
		t.Fatalf("camped site Neighbors = %d, want 2: %+v", len(camped.Neighbors), camped.Neighbors)
	}
	// Sorted by (RFSS, Site): Site 7 before Site 9.
	if camped.Neighbors[0].Site != 7 || camped.Neighbors[1].Site != 9 {
		t.Errorf("neighbors not sorted by site: %+v", camped.Neighbors)
	}
	n := camped.Neighbors[0]
	if n.DownlinkHz != 450_962_500 {
		t.Errorf("neighbour downlink_hz = %d, want 450962500", n.DownlinkHz)
	}
	// Uplink = downlink + band-plan tx offset (+10 MHz).
	if n.UplinkHz != 460_962_500 {
		t.Errorf("neighbour uplink_hz = %d, want 460962500 (downlink + 10 MHz offset)", n.UplinkHz)
	}

	// The non-camped site of the same system did not broadcast the list.
	other := body.Sites[1]
	if other.RFSSID != 1 || other.SiteID != 2 {
		t.Fatalf("second site = %+v, want rfss 1 / site 2", other)
	}
	if len(other.Neighbors) != 0 {
		t.Errorf("non-camped site should carry no neighbours, got %+v", other.Neighbors)
	}
}

// TestListSitesNoProvider confirms the route degrades gracefully when no
// site tracker is wired (empty list, not a 500).
func TestListSitesNoProvider(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/sites")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Sites []SiteDTO `json:"sites"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Sites) != 0 {
		t.Fatalf("want empty sites, got %+v", body.Sites)
	}
}
