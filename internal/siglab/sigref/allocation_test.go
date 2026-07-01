package sigref

import "testing"

func TestAllocationLookup(t *testing.T) {
	cases := []struct {
		hz          uint32
		wantService string
		wantFound   bool
	}{
		{1_090_000_000, "ADS-B 1090ES", true},
		{915_000_000, "ISM 900 MHz", true},
		{462_600_000, "FRS / GMRS", true},
		{162_500_000, "NOAA weather", true}, // narrowest-wins over VHF public safety
		{98_500_000, "FM broadcast", true},
		{751_000_000, "Cellular 700 MHz DL", true},
		{703_000_000, "Cellular 700 MHz", true}, // 700 MHz lower blocks (698–746)
		{5_000_000, "", false},                  // unallocated in the table
	}
	for _, c := range cases {
		a, ok := Lookup(c.hz)
		if ok != c.wantFound {
			t.Errorf("Lookup(%d) found = %v, want %v", c.hz, ok, c.wantFound)
			continue
		}
		if ok && a.Service != c.wantService {
			t.Errorf("Lookup(%d) = %q, want %q", c.hz, a.Service, c.wantService)
		}
	}
}

func TestAllocationLookupNarrowestWins(t *testing.T) {
	// 162.5 MHz falls in both NOAA weather (162.4–162.55) and the broad VHF
	// public-safety band (162–174). The narrower one must win.
	a, ok := Lookup(162_500_000)
	if !ok || a.Service != "NOAA weather" {
		t.Fatalf("Lookup(162.5 MHz) = %+v (ok=%v), want NOAA weather", a, ok)
	}
}

func TestRankWidebandByBandwidthAndAllocation(t *testing.T) {
	cases := []struct {
		name    string
		obs     Observation
		wantTop string
	}{
		{
			"10 MHz OFDM at 751 MHz ⇒ cellular, not WiFi",
			Observation{CenterFreqHz: 751_000_000, ChannelBWHz: 10_000_000},
			"LTE/5G NR (10 MHz)",
		},
		{
			"20 MHz OFDM at 2.45 GHz ⇒ WiFi",
			Observation{CenterFreqHz: 2_450_000_000, ChannelBWHz: 20_000_000},
			"WiFi 802.11 (20 MHz)",
		},
		{
			"20 MHz OFDM at 1.94 GHz PCS ⇒ cellular",
			Observation{CenterFreqHz: 1_940_000_000, ChannelBWHz: 20_000_000},
			"LTE/5G NR (20 MHz)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Rank(tc.obs, 3)
			if len(m) == 0 {
				t.Fatal("no matches")
			}
			if m[0].Entry.DisplayName != tc.wantTop {
				t.Errorf("top = %q (%.2f), want %q", m[0].Entry.DisplayName, m[0].Score, tc.wantTop)
			}
			if m[0].Entry.Decodable {
				t.Errorf("wideband match %q marked decodable; must be false", m[0].Entry.DisplayName)
			}
		})
	}
}

// TestNarrowbandRankingUnaffectedByWideband guards that adding the wideband
// catalog + allocation-weighted scoring did not perturb a narrowband signature
// even when a centre frequency is supplied.
func TestNarrowbandRankingUnaffectedByWideband(t *testing.T) {
	// A P25 control channel at 851.0125 MHz: 4800 sym/s C4FM, 12.5 kHz.
	obs := Observation{
		SymbolRateHz: 4800, SymbolRateConf: 0.9, ModClass: "fsk4", Levels: 4,
		ChannelBWHz: 12500, CenterFreqHz: 851_012_500,
	}
	m := Rank(obs, 3)
	if len(m) == 0 || m[0].Entry.Protocol != "p25" {
		t.Fatalf("P25 signature top match = %+v, want p25", m)
	}
}
