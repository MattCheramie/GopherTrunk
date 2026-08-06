package nxdn

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestLinearBandPlanResolves(t *testing.T) {
	// Channel-1-indexed site: channel 1 → BaseHz, +SpacingHz per channel.
	b := LinearBandPlan{BaseHz: 461_000_000, SpacingHz: 12_500, Offset: 1}
	cases := []struct {
		ch   uint16
		want uint32
	}{
		{1, 461_000_000},
		{2, 461_012_500},
		{5, 461_050_000},
	}
	for _, tc := range cases {
		got, err := b.Frequency(tc.ch)
		if err != nil {
			t.Errorf("Frequency(%d) error: %v", tc.ch, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Frequency(%d) = %d, want %d", tc.ch, got, tc.want)
		}
	}
}

func TestLinearBandPlanRejectsBelowOffset(t *testing.T) {
	b := LinearBandPlan{BaseHz: 461_000_000, SpacingHz: 12_500, Offset: 1}
	if _, err := b.Frequency(0); !errors.Is(err, ErrUnknownChannel) {
		t.Errorf("Frequency(0) err = %v, want ErrUnknownChannel", err)
	}
}

func TestLinearBandPlanNeedsSpacing(t *testing.T) {
	b := LinearBandPlan{BaseHz: 461_000_000, Offset: 1}
	if _, err := b.Frequency(1); err == nil {
		t.Error("Frequency with zero SpacingHz should error")
	}
}

func TestTableBandPlanResolves(t *testing.T) {
	tbl := TableBandPlan{3: 462_000_000, 7: 462_500_000}
	if got, err := tbl.Frequency(7); err != nil || got != 462_500_000 {
		t.Errorf("Frequency(7) = %d, %v; want 462500000, nil", got, err)
	}
	if _, err := tbl.Frequency(4); !errors.Is(err, ErrUnknownChannel) {
		t.Errorf("Frequency(4) err = %v, want ErrUnknownChannel", err)
	}
}

func TestResolverFromPlan(t *testing.T) {
	if ResolverFromPlan(nil) != nil {
		t.Error("nil plan should yield nil resolver")
	}
	if ResolverFromPlan(&trunking.NXDNBandPlan{}) != nil {
		t.Error("empty plan should yield nil resolver")
	}
	lin := ResolverFromPlan(&trunking.NXDNBandPlan{
		Linear: &trunking.NXDNLinearBandPlan{BaseHz: 461_000_000, SpacingHz: 12_500, Offset: 1},
	})
	if lin == nil {
		t.Fatal("linear plan should yield a resolver")
	}
	if got, _ := lin.Frequency(2); got != 461_012_500 {
		t.Errorf("linear resolver Frequency(2) = %d, want 461012500", got)
	}
	tbl := ResolverFromPlan(&trunking.NXDNBandPlan{
		Table: []trunking.NXDNBandPlanTableEntry{{Channel: 9, FreqHz: 463_000_000}},
	})
	if tbl == nil {
		t.Fatal("table plan should yield a resolver")
	}
	if got, _ := tbl.Frequency(9); got != 463_000_000 {
		t.Errorf("table resolver Frequency(9) = %d, want 463000000", got)
	}
}
