package phase1

import "testing"

func TestBandPlanSnapshot(t *testing.T) {
	var bp BandPlan
	if len(bp.Snapshot()) != 0 {
		t.Fatalf("empty band plan should snapshot to nothing")
	}
	// Apply out of ID order; Snapshot must return ascending channel-ID order.
	bp.Apply(IdentifierUpdate{ChannelID: 3, BaseHz: 853_000_000, SpacingHz: 12_500})
	bp.Apply(IdentifierUpdate{ChannelID: 1, BaseHz: 851_000_000, SpacingHz: 12_500, AccessTDMA: true})

	got := bp.Snapshot()
	if len(got) != 2 {
		t.Fatalf("len(snapshot) = %d, want 2", len(got))
	}
	if got[0].ChannelID != 1 || got[1].ChannelID != 3 {
		t.Errorf("snapshot order = %d,%d, want 1,3", got[0].ChannelID, got[1].ChannelID)
	}
	if got[0].BaseHz != 851_000_000 || !got[0].AccessTDMA {
		t.Errorf("slot 1 = %+v, want base 851M TDMA", got[0])
	}
}
