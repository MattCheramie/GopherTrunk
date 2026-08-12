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

// TestParseSitesSkipsEmpty confirms a metadata-only <siteList> with no
// usable frequency is dropped rather than surfaced as a zero-channel site.
func TestParseSitesSkipsEmpty(t *testing.T) {
	raw := []byte(`
	<siteList><rfss>1</rfss><siteNumber>1</siteNumber><siteDescr>NoFreqs</siteDescr></siteList>
	<siteList>
	  <rfss>1</rfss><siteNumber>2</siteNumber><siteDescr>Good</siteDescr>
	  <siteFreq><freq>851.0125</freq><use>d</use></siteFreq>
	</siteList>`)
	sites := parseSites(raw)
	if len(sites) != 1 {
		t.Fatalf("expected 1 site (empty dropped), got %d", len(sites))
	}
	if sites[0].Description != "Good" {
		t.Fatalf("kept the wrong site: %+v", sites[0])
	}
}

func TestProtocolFromType(t *testing.T) {
	cases := []struct {
		sType, flavor, want string
	}{
		{"Project 25", "Phase II", "p25-phase2"},
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

func TestParseSites(t *testing.T) {
	raw := []byte(`
	<siteList>
	  <rfss>1</rfss><siteNumber>3</siteNumber><siteDescr>Downtown</siteDescr><cName>Travis</cName>
	  <siteFreq><freq>851.0125</freq><use>d</use></siteFreq>
	  <siteFreq><freq>851.2625</freq><use></use></siteFreq>
	  <siteFreq><freq>851.5125</freq><use>a</use></siteFreq>
	</siteList>
	<siteList>
	  <rfss>1</rfss><siteNumber>4</siteNumber><siteDescr>North</siteDescr>
	  <siteFreq><freq>852.0125</freq><use>d</use></siteFreq>
	</siteList>`)
	sites := parseSites(raw)
	if len(sites) != 2 {
		t.Fatalf("expected 2 sites, got %d", len(sites))
	}
	s0 := sites[0]
	if s0.RFSS != 1 || s0.SiteNumber != 3 || s0.Description != "Downtown" || s0.County != "Travis" {
		t.Errorf("site0 identity wrong: %+v", s0)
	}
	if len(s0.Frequencies) != 3 {
		t.Errorf("site0 expected 3 freqs, got %v", s0.Frequencies)
	}
	if len(s0.ControlChannels) != 2 || s0.ControlChannels[0] != 851_012_500 {
		t.Errorf("site0 control channels wrong: %v", s0.ControlChannels)
	}
}

// TestParseSitesCoordinates confirms RadioReference site coordinates
// (siteLat/siteLng) are captured, and a missing/invalid leaf degrades to 0.
func TestParseSitesCoordinates(t *testing.T) {
	raw := []byte(`
	<siteList>
	  <rfss>1</rfss><siteNumber>3</siteNumber><siteDescr>Downtown</siteDescr>
	  <siteLat>30.2672</siteLat><siteLng>-97.7431</siteLng>
	  <siteFreq><freq>851.0125</freq><use>d</use></siteFreq>
	</siteList>
	<siteList>
	  <rfss>1</rfss><siteNumber>4</siteNumber><siteDescr>NoGeo</siteDescr>
	  <siteFreq><freq>852.0125</freq><use>d</use></siteFreq>
	</siteList>`)
	sites := parseSites(raw)
	if len(sites) != 2 {
		t.Fatalf("expected 2 sites, got %d", len(sites))
	}
	if sites[0].Latitude != 30.2672 || sites[0].Longitude != -97.7431 {
		t.Errorf("site0 coords = (%v,%v), want (30.2672,-97.7431)", sites[0].Latitude, sites[0].Longitude)
	}
	if sites[1].Latitude != 0 || sites[1].Longitude != 0 {
		t.Errorf("site1 (no geo) coords = (%v,%v), want (0,0)", sites[1].Latitude, sites[1].Longitude)
	}
}

func TestParseTalkgroups(t *testing.T) {
	raw := []byte(`
	<tgList><tgDec>101</tgDec><tgAlpha>PD Disp</tgAlpha><tgDescr>Police Dispatch</tgDescr><tag>Law Dispatch</tag><mode>D</mode><enc>0</enc></tgList>
	<tgList><tgDec>202</tgDec><tgAlpha>FD Ops</tgAlpha><mode>A</mode><enc>1</enc></tgList>
	<tgList><tgDec>0</tgDec><tgAlpha>ignored</tgAlpha></tgList>`)
	tgs := parseTalkgroups(raw)
	if len(tgs) != 2 {
		t.Fatalf("expected 2 talkgroups (zero-dec skipped), got %d", len(tgs))
	}
	if tgs[0].Dec != 101 || tgs[0].AlphaTag != "PD Disp" || tgs[0].Mode != "D" || tgs[0].Encrypted {
		t.Errorf("tg0 wrong: %+v", tgs[0])
	}
	if tgs[1].Dec != 202 || tgs[1].Mode != "A" || !tgs[1].Encrypted {
		t.Errorf("tg1 wrong: %+v", tgs[1])
	}
}
