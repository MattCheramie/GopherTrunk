package phase1

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// A p25_band_plan seed for a TDMA channel ID must divide like an on-air
// IDEN_UP_TDMA (issue #1273). Before slots_per_carrier existed a seed could
// only be FDMA, so a site whose TDMA ID was never announced resolved every
// Phase 2 grant to carrier n instead of n/slots.
func TestIdentifierUpdateFromSeedTDMA(t *testing.T) {
	var bp BandPlan
	bp.Apply(IdentifierUpdateFromSeed(trunking.P25BandPlanEntry{
		ChannelID: 3, BaseHz: 762_006_250, SpacingHz: 12_500, TxOffsetHz: 30_000_000,
		BandwidthHz: 12_500, SlotsPerCarrier: 2,
	}))
	hz, err := bp.Frequency(3, 11)
	if err != nil {
		t.Fatal(err)
	}
	if want := uint32(762_006_250 + 5*12_500); hz != want {
		t.Errorf("seeded TDMA Frequency(3, 11) = %d, want %d (carrier 11/2 = 5)", hz, want)
	}
	if ts := bp.Timeslot(3, 11); ts != 2 {
		t.Errorf("seeded TDMA Timeslot(3, 11) = %d, want 2", ts)
	}
	u := IdentifierUpdateFromSeed(trunking.P25BandPlanEntry{ChannelID: 3, SlotsPerCarrier: 2})
	if !u.AccessTDMA {
		t.Error("a 2-slot seed must be AccessTDMA so its grants route to the Phase 2 voice chain")
	}

	// FDMA seeds are unchanged.
	var fd BandPlan
	fd.Apply(IdentifierUpdateFromSeed(trunking.P25BandPlanEntry{ChannelID: 2, BaseHz: 851_006_250, SpacingHz: 12_500}))
	if hz, _ := fd.Frequency(2, 11); hz != 851_006_250+11*12_500 {
		t.Errorf("FDMA seed Frequency(2, 11) = %d, want %d", hz, 851_006_250+11*12_500)
	}
	if IdentifierUpdateFromSeed(trunking.P25BandPlanEntry{ChannelID: 2}).AccessTDMA {
		t.Error("an FDMA seed must not be AccessTDMA")
	}
}
