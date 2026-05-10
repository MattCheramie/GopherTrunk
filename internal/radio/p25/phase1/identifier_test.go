package phase1

import (
	"errors"
	"testing"
)

func TestIdentifierUpdateRoundTrip700MHz(t *testing.T) {
	// Realistic 700/800 MHz Phase 1 fixture: ChannelID 1, base 851.0
	// MHz, 12.5 kHz spacing + bandwidth, -39 MHz transmit offset.
	in := IdentifierUpdate{
		ChannelID:   1,
		BandwidthHz: 12_500,
		SpacingHz:   12_500,
		TxOffsetHz:  -39_000_000,
		BaseHz:      851_000_000,
	}
	out := ParseIdentifierUpdate(AssembleIdentifierUpdate(in))
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestIdentifierUpdateRoundTripPositiveOffset(t *testing.T) {
	// VHF-style fixture with a positive offset to exercise the sign
	// bit. Spacing is 6.25 kHz; bandwidth differs from spacing.
	in := IdentifierUpdate{
		ChannelID:   5,
		BandwidthHz: 6_250,
		SpacingHz:   6_250,
		TxOffsetHz:  5_000_000,
		BaseHz:      154_000_000,
	}
	out := ParseIdentifierUpdate(AssembleIdentifierUpdate(in))
	if out != in {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

func TestBandPlanResolvesKnownChannel(t *testing.T) {
	bp := &BandPlan{}
	bp.Apply(IdentifierUpdate{
		ChannelID: 1,
		SpacingHz: 12_500,
		BaseHz:    851_000_000,
	})

	// Channel 0 → base. Channel 16 → base + 16 × 12.5 kHz = 851.2 MHz.
	if hz, err := bp.Frequency(1, 0); err != nil || hz != 851_000_000 {
		t.Errorf("Frequency(1,0) = %d, %v; want 851_000_000, nil", hz, err)
	}
	if hz, err := bp.Frequency(1, 16); err != nil || hz != 851_200_000 {
		t.Errorf("Frequency(1,16) = %d, %v; want 851_200_000, nil", hz, err)
	}
}

func TestBandPlanUnknownChannelID(t *testing.T) {
	bp := &BandPlan{}
	_, err := bp.Frequency(7, 0)
	if !errors.Is(err, ErrUnknownChannelID) {
		t.Errorf("Frequency on empty plan: err = %v, want ErrUnknownChannelID", err)
	}
	if bp.Known(7) {
		t.Error("Known(7) on empty plan should be false")
	}
}

func TestBandPlanReplacesSlotOnNewIdentifierUpdate(t *testing.T) {
	bp := &BandPlan{}
	bp.Apply(IdentifierUpdate{ChannelID: 2, SpacingHz: 12_500, BaseHz: 851_000_000})
	bp.Apply(IdentifierUpdate{ChannelID: 2, SpacingHz: 25_000, BaseHz: 852_000_000})

	hz, err := bp.Frequency(2, 4)
	if err != nil {
		t.Fatalf("Frequency: %v", err)
	}
	if hz != 852_100_000 {
		t.Errorf("hz = %d, want 852_100_000 (uses replaced slot)", hz)
	}
}

func TestBandPlanRejectsOverflow(t *testing.T) {
	bp := &BandPlan{}
	bp.Apply(IdentifierUpdate{ChannelID: 0, SpacingHz: 1_000_000, BaseHz: 4_000_000_000})
	if _, err := bp.Frequency(0, 1000); err == nil {
		t.Error("expected overflow error for >4.29 GHz resolved frequency")
	}
}
