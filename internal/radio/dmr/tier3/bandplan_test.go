package tier3

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestLinearBandPlanWithLCNOffset(t *testing.T) {
	// 1-indexed site: LCN 1 → 851.000 MHz, LCN 2 → 851.025 MHz.
	bp := LinearBandPlan{BaseHz: 851_000_000, SpacingHz: 25_000, Offset: 1}
	if hz, err := bp.Frequency(1); err != nil || hz != 851_000_000 {
		t.Errorf("Frequency(1) = %d, %v; want 851_000_000, nil", hz, err)
	}
	if hz, err := bp.Frequency(2); err != nil || hz != 851_025_000 {
		t.Errorf("Frequency(2) = %d, %v; want 851_025_000", hz, err)
	}
}

func TestLinearBandPlanRejectsBelowOffset(t *testing.T) {
	bp := LinearBandPlan{BaseHz: 851_000_000, SpacingHz: 25_000, Offset: 1}
	_, err := bp.Frequency(0) // LCN 0 below offset 1
	if !errors.Is(err, ErrUnknownLCN) {
		t.Errorf("err = %v, want ErrUnknownLCN", err)
	}
}

func TestLinearBandPlanRejectsZeroSpacing(t *testing.T) {
	if _, err := (LinearBandPlan{BaseHz: 1, SpacingHz: 0}).Frequency(1); err == nil {
		t.Error("zero spacing should error")
	}
}

func TestTableBandPlan(t *testing.T) {
	bp := TableBandPlan{1: 154_115_000, 5: 154_205_000}
	if hz, _ := bp.Frequency(1); hz != 154_115_000 {
		t.Errorf("LCN 1 → %d", hz)
	}
	if hz, _ := bp.Frequency(5); hz != 154_205_000 {
		t.Errorf("LCN 5 → %d", hz)
	}
	if _, err := bp.Frequency(99); !errors.Is(err, ErrUnknownLCN) {
		t.Errorf("missing LCN: err = %v, want ErrUnknownLCN", err)
	}
}

func TestLinearBandPlanRejectsOverflow(t *testing.T) {
	// 4.2 GHz base + 127 × 1 MHz = 4.327 GHz, just past uint32 (≈4.294 GHz).
	bp := LinearBandPlan{BaseHz: 4_200_000_000, SpacingHz: 1_000_000}
	if _, err := bp.Frequency(127); err == nil {
		t.Error("expected overflow error for >4.29 GHz resolved frequency")
	}
}

func TestResolverFromPlanNil(t *testing.T) {
	if r := ResolverFromPlan(nil); r != nil {
		t.Errorf("ResolverFromPlan(nil) = %v, want nil", r)
	}
	// An empty plan (neither linear nor table) also resolves to nil so
	// the caller falls back to the no-bandplan drop behaviour.
	if r := ResolverFromPlan(&trunking.DMRBandPlan{}); r != nil {
		t.Errorf("ResolverFromPlan(empty) = %v, want nil", r)
	}
}

func TestResolverFromPlanLinear(t *testing.T) {
	r := ResolverFromPlan(&trunking.DMRBandPlan{
		Linear: &trunking.DMRLinearBandPlan{BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 1},
	})
	if r == nil {
		t.Fatal("ResolverFromPlan(linear) = nil")
	}
	if _, ok := r.(LinearBandPlan); !ok {
		t.Fatalf("resolver type = %T, want LinearBandPlan", r)
	}
	// LCN 8 on a 1-indexed 25 kHz grid: 866.000 MHz + 7×25 kHz = 866.175 MHz.
	if hz, err := r.Frequency(8); err != nil || hz != 866_175_000 {
		t.Errorf("Frequency(8) = %d, %v; want 866_175_000, nil", hz, err)
	}
}

func TestResolverFromPlanTable(t *testing.T) {
	r := ResolverFromPlan(&trunking.DMRBandPlan{
		Table: []trunking.DMRBandPlanTableEntry{
			{LCN: 1, FreqHz: 462_550_000},
			{LCN: 4, FreqHz: 462_575_000},
		},
	})
	if r == nil {
		t.Fatal("ResolverFromPlan(table) = nil")
	}
	if _, ok := r.(TableBandPlan); !ok {
		t.Fatalf("resolver type = %T, want TableBandPlan", r)
	}
	if hz, err := r.Frequency(4); err != nil || hz != 462_575_000 {
		t.Errorf("Frequency(4) = %d, %v; want 462_575_000, nil", hz, err)
	}
	if _, err := r.Frequency(9); !errors.Is(err, ErrUnknownLCN) {
		t.Errorf("Frequency(9) err = %v, want ErrUnknownLCN", err)
	}
}

func TestResolverFromPlanLinearWinsWhenBothSet(t *testing.T) {
	// Config validation forbids both, but ResolverFromPlan defensively
	// prefers Linear so a malformed plan still yields a usable resolver.
	r := ResolverFromPlan(&trunking.DMRBandPlan{
		Linear: &trunking.DMRLinearBandPlan{BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 1},
		Table:  []trunking.DMRBandPlanTableEntry{{LCN: 1, FreqHz: 1}},
	})
	if _, ok := r.(LinearBandPlan); !ok {
		t.Fatalf("resolver type = %T, want LinearBandPlan", r)
	}
}
