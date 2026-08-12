package configbuilder

import (
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

func TestTalkgroupCSVRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tg.csv")
	rows := []TalkgroupCSVRow{
		{Decimal: 101, AlphaTag: "PD Disp", Description: "Police", Tag: "Law", Group: "PD", Mode: "D"},
		{Decimal: 202, AlphaTag: "FD Ops"},
	}
	if err := WriteTalkgroupCSV(path, rows); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadTalkgroupCSV(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != 2 || got[0].Decimal != 101 || got[0].AlphaTag != "PD Disp" || got[0].Mode != "D" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got[1].Decimal != 202 || got[1].AlphaTag != "FD Ops" {
		t.Fatalf("row1 mismatch: %+v", got[1])
	}
}

func TestHzToDisplay(t *testing.T) {
	cases := map[uint32]string{
		0:         "0",
		851037500: "851.0375 MHz",
		852000000: "852 MHz",
		161975000: "161.975 MHz",
	}
	for hz, want := range cases {
		if got := HzToDisplay(hz); got != want {
			t.Errorf("HzToDisplay(%d) = %q, want %q", hz, got, want)
		}
	}
}

func TestParseFreqList(t *testing.T) {
	if got := ParseFreqList("851.0375\n852.2625"); len(got) != 2 || got[0] != 851037500 || got[1] != 852262500 {
		t.Fatalf("MHz parse: %v", got)
	}
	if got := ParseFreqList("851037500, 852262500"); len(got) != 2 || got[0] != 851037500 {
		t.Fatalf("Hz parse: %v", got)
	}
	if got := ParseFreqList("851.0375 MHz; junk; 0"); len(got) != 1 || got[0] != 851037500 {
		t.Fatalf("mixed parse: %v", got)
	}
	if got := ParseHz("154.25"); got != 154250000 {
		t.Fatalf("ParseHz: %d", got)
	}
	if got := ParseHz("nonsense"); got != 0 {
		t.Fatalf("ParseHz junk: %d", got)
	}
}

func TestSectionOf(t *testing.T) {
	cases := map[string]string{
		"trunking.systems[0]: name required": "trunking",
		"sdr.sample_rate must be ...":        "sdr",
		"scanner.conventional[1]: bad":       "scanner",
		"web.tabs: unknown":                  "web",
	}
	for msg, want := range cases {
		if got := SectionOf(msg); got != want {
			t.Errorf("SectionOf(%q) = %q, want %q", msg, got, want)
		}
	}
}

func TestRRToSystemConfig(t *testing.T) {
	full := radioreference.FullSystem{
		Name:     "Metro",
		Protocol: "p25",
		Sites: []radioreference.SiteDetail{
			{ControlChannels: []uint32{851_037_500, 851_062_500}},
			{ControlChannels: []uint32{851_037_500, 851_087_500}}, // dup 851_037_500 across sites
		},
		Talkgroups: []radioreference.TalkgroupDetail{
			{Dec: 101, AlphaTag: "PD", Mode: "D"},
		},
	}
	sys, rows := RRToSystemConfig(full)
	if sys.Name != "Metro" || sys.Protocol != "p25" {
		t.Fatalf("identity: %+v", sys)
	}
	if len(sys.ControlChannels) != 3 {
		t.Fatalf("expected 3 deduped CCs, got %v", sys.ControlChannels)
	}
	if len(rows) != 1 || rows[0].Decimal != 101 || rows[0].Mode != "D" {
		t.Fatalf("talkgroups: %+v", rows)
	}
}

// TestRRToSystemConfigSites confirms named RR sites become SiteConfig entries
// carrying name + coordinates (so an imported system maps its sites), and a
// site with no description is skipped.
func TestRRToSystemConfigSites(t *testing.T) {
	full := radioreference.FullSystem{
		Name:     "Metro",
		Protocol: "p25",
		Sites: []radioreference.SiteDetail{
			{RFSS: 1, SiteNumber: 3, Description: "Downtown", Latitude: 30.27, Longitude: -97.74,
				ControlChannels: []uint32{851_037_500}},
			{RFSS: 1, SiteNumber: 4, Description: "", ControlChannels: []uint32{851_062_500}}, // no name → skipped
		},
	}
	sys, _ := RRToSystemConfig(full)
	if len(sys.Sites) != 1 {
		t.Fatalf("expected 1 named site, got %+v", sys.Sites)
	}
	s := sys.Sites[0]
	if s.RFSS != 1 || s.Site != 3 || s.Name != "Downtown" || s.Latitude != 30.27 || s.Longitude != -97.74 {
		t.Errorf("site config = %+v", s)
	}
}
