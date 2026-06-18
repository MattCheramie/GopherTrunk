package phase1

import "testing"

func TestBandPlanSnapshot(t *testing.T) {
	var bp BandPlan
	if len(bp.Snapshot()) != 0 {
		t.Fatalf("empty band plan should snapshot to nothing")
	}
	// Apply out of ID order; Snapshot must return ascending channel-ID order.
	// Each slot is applied corroborationMin (2) times so it clears the
	// corroboration gate Snapshot enforces.
	bp.Apply(IdentifierUpdate{ChannelID: 3, BaseHz: 853_000_000, SpacingHz: 12_500})
	bp.Apply(IdentifierUpdate{ChannelID: 3, BaseHz: 853_000_000, SpacingHz: 12_500})
	bp.Apply(IdentifierUpdate{ChannelID: 1, BaseHz: 851_000_000, SpacingHz: 12_500, AccessTDMA: true})
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

	// A slot seen only once must not surface (one-shot CRC false positive).
	bp.Apply(IdentifierUpdate{ChannelID: 5, BaseHz: 136_000_000, SpacingHz: 6_250})
	if got := bp.Snapshot(); len(got) != 2 {
		t.Errorf("one-shot slot leaked into snapshot: len = %d, want 2", len(got))
	}
	// ...but it still resolves a live grant via Frequency (not corroboration-gated).
	if _, err := bp.Frequency(5, 10); err != nil {
		t.Errorf("Frequency on a once-seen slot should resolve: %v", err)
	}
}
