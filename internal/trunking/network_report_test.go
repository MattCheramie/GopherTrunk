package trunking

import (
	"strings"
	"testing"
)

// fullReport mirrors the user's reference site: WACN BEE00 / SYSTEM 2C2 /
// NAC 2C1, RFSS 1 / Site 1, a primary + secondary CC, one neighbour, one band.
func fullReport() NetworkReport {
	return NetworkReport{
		Name:     "Main Site",
		Protocol: "p25",
		WACN:     0xBEE00,
		SystemID: 0x2C2,
		NAC:      0x2C1,
		Sites: []ReportSite{{
			RFSS:      1,
			Site:      1,
			PrimaryCC: ReportChannel{ChannelID: 2, ChannelNumber: 1620, DownlinkHz: 450125000, UplinkHz: 460687500},
			SecondaryCC: []ReportChannel{
				{ChannelID: 2, ChannelNumber: 1692, DownlinkHz: 450575000, UplinkHz: 460575000},
			},
			Neighbors: []ReportNeighbor{
				{RFSS: 1, Site: 7, Channel: ReportChannel{ChannelID: 2, ChannelNumber: 1754, DownlinkHz: 450962500, UplinkHz: 461437500}},
			},
		}},
		Bands: []ReportBand{
			{ChannelID: 2, BaseHz: 440000000, SpacingHz: 6250, BandwidthHz: 12500, TxOffsetHz: -6400000},
		},
	}
}

func TestRenderNetworkReportTokens(t *testing.T) {
	out := FormatNetworkReport(fullReport())
	for _, want := range []string{
		"P25 Network Configuration — Main Site",
		"WACN:BEE00[781824]",
		"SYSTEM:2C2[706]",
		"NAC:2C1[705]",
		"Current Site",
		"RFSS:1[1] SITE:1[1]",
		"PRI CONTROL CHANNEL:2-1620 DOWNLINK:450.125000 MHz UPLINK:460.687500 MHz",
		"SEC CONTROL CHANNEL:2-1692 DOWNLINK:450.575000 MHz",
		"NEIGHBOR RFSS:1[1] SITE:7[7] CHANNEL:2-1754 DOWNLINK:450.962500 MHz UPLINK:461.437500 MHz",
		"Frequency Bands",
		"BAND:2 FDMA BASE:440.000000 MHz BANDWIDTH:12.5 kHz SPACING:6.25 kHz OFFSET:-6.400000 MHz",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q\n--- full report ---\n%s", want, out)
		}
	}
}

func TestRenderNetworkReportSuppressesEmptySections(t *testing.T) {
	// Only identity known — no site channels, no neighbours, no bands.
	out := FormatNetworkReport(NetworkReport{
		Protocol: "p25", WACN: 0xBEE00, SystemID: 0x2C2, NAC: 0x2C1,
		Sites: []ReportSite{{RFSS: 1, Site: 1}},
	})
	if strings.Contains(out, "Frequency Bands") {
		t.Errorf("empty band plan should be suppressed:\n%s", out)
	}
	if strings.Contains(out, "CONTROL CHANNEL") || strings.Contains(out, "NEIGHBOR") {
		t.Errorf("empty channels/neighbours should be suppressed:\n%s", out)
	}
	if !strings.Contains(out, "(unnamed)") {
		t.Errorf("blank name should render as (unnamed):\n%s", out)
	}
}

func TestRenderNetworkReportHuntFrequencyOnlyChannel(t *testing.T) {
	// A hunt-discovered control channel carries only a resolved frequency (no
	// band-plan coordinates), so the id-number token must be omitted.
	out := FormatNetworkReport(NetworkReport{
		Protocol: "p25",
		Sites:    []ReportSite{{RFSS: 1, Site: 1, PrimaryCC: ReportChannel{DownlinkHz: 450125000}}},
	})
	if !strings.Contains(out, "PRI CONTROL CHANNEL DOWNLINK:450.125000 MHz") {
		t.Errorf("frequency-only primary CC should omit the id-number token:\n%s", out)
	}
}

func TestReportFromTopologyMapsFieldsAndUplink(t *testing.T) {
	snap := &TopologySnapshot{
		SystemName: "Main Site",
		Protocol:   "p25",
		WACN:       0xBEE00,
		SystemID:   0x2C2,
		NAC:        0x2C1,
		RFSS:       1,
		Site:       1,
		LRA:        0,
		PrimaryCC:  &TopoChannelRef{ChannelID: 2, ChannelNumber: 1620, FrequencyHz: 450125000},
		Neighbors:  []TopoNeighborRef{{RFSS: 1, Site: 7, ChannelID: 2, ChannelNumber: 1754, FrequencyHz: 450962500}},
		BandPlan:   []TopoBandPlanSlot{{ChannelID: 2, BaseHz: 440000000, SpacingHz: 6250, TxOffsetHz: -6400000}},
	}
	r := ReportFromTopology(snap)
	if len(r.Sites) != 1 {
		t.Fatalf("Sites = %d, want 1", len(r.Sites))
	}
	site := r.Sites[0]
	// Uplink = downlink + band TxOffset (450125000 - 6400000 = 443725000).
	if site.PrimaryCC.UplinkHz != 443725000 {
		t.Errorf("primary uplink = %d, want 443725000", site.PrimaryCC.UplinkHz)
	}
	if r.NAC != 0x2C1 || r.WACN != 0xBEE00 || r.Name != "Main Site" {
		t.Errorf("identity not mapped: %+v", r)
	}
	if len(site.Neighbors) != 1 || site.Neighbors[0].Channel.UplinkHz != 444562500 {
		t.Errorf("neighbour uplink not derived: %+v", site.Neighbors)
	}
}

// TestReportFromTopologyResolvesNeighbourDownlinkFromBandPlan is the regression
// for the blank "Neighbor sites" downlink: a P25 ADJ_STS_BCST reports a neighbour
// as a (channel id, channel number) pair with NO frequency, so the report must
// compute the downlink from the matching IDEN_UP band plan (base + number *
// spacing) — the same math a voice grant uses. Fails before the fix (downlink 0).
func TestReportFromTopologyResolvesNeighbourDownlinkFromBandPlan(t *testing.T) {
	snap := &TopologySnapshot{
		SystemName: "Main Site",
		Protocol:   "p25",
		BandPlan:   []TopoBandPlanSlot{{ChannelID: 2, BaseHz: 450_000_000, SpacingHz: 12_500, TxOffsetHz: 10_000_000}},
		// FrequencyHz deliberately 0 — the neighbour's band hadn't resolved when
		// the ADJ_STS was heard; the report must fill it in from the band plan.
		Neighbors: []TopoNeighborRef{{RFSS: 1, Site: 7, ChannelID: 2, ChannelNumber: 77}},
	}
	r := ReportFromTopology(snap)
	if len(r.Sites) != 1 || len(r.Sites[0].Neighbors) != 1 {
		t.Fatalf("expected one neighbour: %+v", r.Sites)
	}
	// downlink = 450_000_000 + 77*12_500 = 450_962_500; uplink = +10 MHz.
	n := r.Sites[0].Neighbors[0].Channel
	if n.DownlinkHz != 450_962_500 {
		t.Errorf("neighbour downlink = %d, want 450962500 (base + number*spacing)", n.DownlinkHz)
	}
	if n.UplinkHz != 460_962_500 {
		t.Errorf("neighbour uplink = %d, want 460962500 (downlink + tx offset)", n.UplinkHz)
	}
}

// TestReportFromTopologyDirectUplink covers a protocol (TETRA) that resolves the
// uplink directly from its own duplex spacing, with no band plan: the explicit
// TopoChannelRef.UplinkHz must flow through to the report and the rendered
// UPLINK line, not be dropped for want of a band-plan transmit offset.
func TestReportFromTopologyDirectUplink(t *testing.T) {
	snap := &TopologySnapshot{
		SystemName: "TETRA Cell",
		Protocol:   "tetra",
		// Offset-corrected duplex pair (§21.4.4.1): 469.88125 MHz DL / 459.88125 UL.
		PrimaryCC: &TopoChannelRef{ChannelNumber: 3595, FrequencyHz: 469881250, UplinkHz: 459881250},
	}
	r := ReportFromTopology(snap)
	if len(r.Sites) != 1 {
		t.Fatalf("Sites = %d, want 1", len(r.Sites))
	}
	if got := r.Sites[0].PrimaryCC.UplinkHz; got != 459881250 {
		t.Errorf("primary uplink = %d, want 459881250 (direct, no band plan)", got)
	}
	var buf strings.Builder
	RenderNetworkReport(&buf, r)
	if !strings.Contains(buf.String(), "UPLINK:459.881250 MHz") {
		t.Errorf("rendered report missing direct TETRA uplink:\n%s", buf.String())
	}
}
