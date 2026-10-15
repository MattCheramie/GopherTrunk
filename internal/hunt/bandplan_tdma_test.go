package hunt

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// The hunt band plan resolved a grant or neighbour on a TDMA ID to carrier n
// instead of n / slots-per-carrier (issue #1273); the slot count must survive
// accumulation from the decoder's topology snapshot.
func TestAccumulateTDMABandPlanResolvesToCarrier(t *testing.T) {
	sys := &DiscoveredSystem{}
	r := fakeResult(851_012_500, nil)
	r.Topology = &siglab.TopologySnapshot{
		RFSS: 1, Site: 1,
		Neighbors: []siglab.NeighborRef{{RFSS: 1, Site: 2, ChannelID: 3, ChannelNumber: 9}},
		BandPlan: []siglab.BandPlanSlot{{ChannelID: 3, BaseHz: 762_006_250, SpacingHz: 12_500,
			AccessTDMA: true, SlotsPerCarrier: 2}},
	}
	Accumulate(sys, Observation{Protocol: "p25", Confidence: 0.8, Result: r})

	if len(sys.BandPlan) != 1 || sys.BandPlan[0].SlotsPerCarrier != 2 {
		t.Fatalf("band plan lost the TDMA slot count: %+v", sys.BandPlan)
	}
	want := uint32(762_006_250 + 4*12_500) // carrier 9/2 = 4
	if got := grantFreq(sys, siglab.GrantRecord{ChannelID: 3, ChannelNum: 9}); got != want {
		t.Errorf("grant on TDMA channel 9 = %d Hz, want %d", got, want)
	}
	sys.sortAll()
	if got := sys.Sites[0].Neighbors[0].FrequencyHz; got != want {
		t.Errorf("neighbour on TDMA channel 9 = %d Hz, want %d", got, want)
	}
}
