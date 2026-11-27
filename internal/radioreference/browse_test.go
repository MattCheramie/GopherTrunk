package radioreference

import "testing"

func TestMhzToHz(t *testing.T) {
	cases := map[string]uint32{
		"851.0125": 851_012_500,
		"460.5":    460_500_000,
		" 154.25 ": 154_250_000,
		"":         0,
		"abc":      0,
		"-5":       0,
		// Overflow guard: a value beyond the uint32 Hz ceiling (~4294.97
		// MHz) returns 0 rather than wrapping to a bogus frequency.
		"99999999": 0,
	}
	for in, want := range cases {
		if got := mhzToHz(in); got != want {
			t.Errorf("mhzToHz(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestProtocolFromType(t *testing.T) {
	cases := []struct {
		sType, flavor, want string
	}{
		// #1316: a Phase II system has a Phase 1 (FDMA) control channel;
		// p25 follows its TDMA voice grants. p25-phase2 is the TDMA CC
		// pipeline and never locks on these systems.
		{"Project 25", "Phase II", "p25"},
		{"Project 25 Phase II", "", "p25"},
		{"Project 25", "Standard", "p25"},
		{"DMR", "Tier III", "dmr"},
		{"DMR", "Tier II", "dmr-tier2"},
		{"MotoTRBO Connect Plus", "", "dmr"},
		{"NXDN", "", "nxdn"},
		{"EDACS", "Standard", "edacs"},
		{"LTR Standard", "", "ltr"},
		{"TETRA", "", "tetra"},
		{"Motorola Type II SmartZone", "", "motorola"},
		{"Something Unknown", "", "p25"},
	}
	for _, tc := range cases {
		if got := protocolFromType(tc.sType, tc.flavor); got != tc.want {
			t.Errorf("protocolFromType(%q,%q) = %q, want %q", tc.sType, tc.flavor, got, tc.want)
		}
	}
}
