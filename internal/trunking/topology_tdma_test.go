package trunking

import "testing"

// A TDMA ID's channel number counts logical channels (issue #1273): the
// carrier is number / SlotsPerCarrier, as in the decoders' BandPlan.Frequency.
func TestTopoBandPlanSlotChannelHzDividesTDMA(t *testing.T) {
	const base, spacing = 762_006_250, 12_500
	for _, tc := range []struct {
		slots uint8
		num   uint16
		want  uint32
	}{
		{0, 11, base + 11*spacing}, // FDMA: unchanged
		{1, 11, base + 11*spacing},
		{2, 11, base + 5*spacing}, // carrier 5, slot 2
		{4, 11, base + 2*spacing}, // carrier 2, slot 4
	} {
		s := TopoBandPlanSlot{ChannelID: 3, BaseHz: base, SpacingHz: spacing, SlotsPerCarrier: tc.slots}
		if got := s.ChannelHz(tc.num); got != tc.want {
			t.Errorf("slots=%d num=%d: ChannelHz = %d, want %d", tc.slots, tc.num, got, tc.want)
		}
	}
	if got := (TopoBandPlanSlot{SpacingHz: spacing}).ChannelHz(5); got != 0 {
		t.Errorf("no base: ChannelHz = %d, want 0", got)
	}
}

// The network report resolved a neighbour on a 2-slot TDMA ID to carrier n
// instead of n/2, and the explicit uplink the same way (issue #1273).
func TestReportFromTopologyDividesTDMAChannelNumbers(t *testing.T) {
	snap := &TopologySnapshot{
		Protocol: "p25",
		BandPlan: []TopoBandPlanSlot{{ChannelID: 3, BaseHz: 762_006_250, SpacingHz: 12_500,
			AccessTDMA: true, SlotsPerCarrier: 2}},
		Neighbors: []TopoNeighborRef{{RFSS: 1, Site: 7, ChannelID: 3, ChannelNumber: 11,
			UplinkChannelID: 3, UplinkChannelNumber: 21}},
	}
	r := ReportFromTopology(snap)
	if len(r.Sites) != 1 || len(r.Sites[0].Neighbors) != 1 {
		t.Fatalf("expected one neighbour: %+v", r.Sites)
	}
	c := r.Sites[0].Neighbors[0].Channel
	if want := uint32(762_006_250 + 5*12_500); c.DownlinkHz != want {
		t.Errorf("TDMA neighbour downlink = %d, want %d (carrier 11/2 = 5)", c.DownlinkHz, want)
	}
	if want := uint32(762_006_250 + 10*12_500); c.UplinkHz != want {
		t.Errorf("TDMA neighbour explicit uplink = %d, want %d (carrier 21/2 = 10)", c.UplinkHz, want)
	}
}
