package sigref

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/blind"
)

// TestDecodableRowsCoverProtocols is a drift guard: every decodable row must
// carry a positive symbol rate and bandwidth, and the catalog must include the
// core trunking protocols by name.
func TestDecodableRowsValid(t *testing.T) {
	want := []string{"p25", "p25-phase2", "dmr", "nxdn", "dpmr", "edacs", "motorola", "ltr", "mpt1327", "tetra", "ysf", "dstar"}
	for _, p := range want {
		e, ok := ByProtocol(p)
		if !ok {
			t.Errorf("missing decodable entry for %q", p)
			continue
		}
		if e.SymbolRateHz <= 0 || e.ChannelBWHz <= 0 {
			t.Errorf("%s: symbol rate %.0f / BW %.0f, want both > 0", p, e.SymbolRateHz, e.ChannelBWHz)
		}
	}
	for _, e := range Decodable() {
		if !e.Decodable || e.Protocol == "" {
			t.Errorf("Decodable() returned a non-decodable/blank-protocol row: %+v", e)
		}
	}
}

func TestRankNamesSignatures(t *testing.T) {
	cases := []struct {
		name    string
		obs     Observation
		wantTop string
	}{
		{
			"tetra signature",
			Observation{SymbolRateHz: 18000, SymbolRateConf: 0.9, ModClass: blind.ModPSK, Levels: 4, ChannelBWHz: 25000},
			"tetra",
		},
		{
			"p25p1 signature",
			Observation{SymbolRateHz: 4800, SymbolRateConf: 0.9, ModClass: blind.ModFSK4, Levels: 4, ChannelBWHz: 12500},
			"p25",
		},
		{
			"dpmr signature (2400, 4FSK, narrow)",
			Observation{SymbolRateHz: 2400, SymbolRateConf: 0.9, ModClass: blind.ModFSK4, Levels: 4, ChannelBWHz: 6250},
			"dpmr",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Rank(tc.obs, 3)
			if len(m) == 0 {
				t.Fatal("no matches")
			}
			if m[0].Entry.Protocol != tc.wantTop {
				t.Errorf("top match = %q (%.2f), want %q; why=%q", m[0].Entry.Protocol, m[0].Score, tc.wantTop, m[0].Why)
			}
			if m[0].Score <= 0 || m[0].Score > 1 {
				t.Errorf("score %.3f out of (0,1]", m[0].Score)
			}
		})
	}
}

func TestRankSparseObservation(t *testing.T) {
	// Symbol rate alone should still rank — and not crash on missing fields.
	m := Rank(Observation{SymbolRateHz: 18000, SymbolRateConf: 0.8}, 1)
	if len(m) == 0 || m[0].Entry.Protocol != "tetra" {
		t.Fatalf("symbol-rate-only TETRA: got %+v", m)
	}
}
