package main

import (
	"bytes"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
)

// TestHuntBundleRoundTrip is the load-bearing guarantee of the export: a
// DiscoveredSystem rendered as a bundle must re-import cleanly through the
// importer's parseCSVStream with its sites, control channels and talkgroups
// intact. This couples the hunt exporter to the importer so the two formats
// can't silently drift apart.
func TestHuntBundleRoundTrip(t *testing.T) {
	sys := &hunt.DiscoveredSystem{
		Name:     "Roundtrip P25",
		Protocol: "p25",
		WACN:     0xBEE99,
		SystemID: 0x49A,
		County:   "Example County",
		Location: "Example City, AZ",
		Sites: []hunt.DiscoveredSite{{
			RFSS: 1, SiteID: 2, SiteName: "Tower Alpha", County: "Example",
			ControlChannels: []hunt.DiscoveredChannel{
				{FrequencyHz: 851012500, IsControl: true},
				{FrequencyHz: 851262500, IsControl: true},
			},
		}},
		Talkgroups: []hunt.DiscoveredTalkgroup{
			{Dec: 1000, Hex: "3e8", Count: 3},
			{Dec: 1001, Hex: "3e9", Count: 1},
		},
	}

	var buf bytes.Buffer
	if err := hunt.Write(&buf, sys, hunt.FormatBundle, nil); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	got, err := parseCSVStream(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("re-import bundle failed: %v\n--- bundle ---\n%s", err, buf.String())
	}

	if got.Name != "Roundtrip P25" {
		t.Errorf("Name = %q, want Roundtrip P25", got.Name)
	}
	if got.Protocol != "p25" {
		t.Errorf("Protocol = %q, want p25", got.Protocol)
	}
	if got.County != "Example County" {
		t.Errorf("County = %q, want Example County", got.County)
	}
	if got.Location != "Example City, AZ" {
		t.Errorf("Location = %q, want \"Example City, AZ\"", got.Location)
	}
	if len(got.Sites) != 1 {
		t.Fatalf("len(Sites) = %d, want 1", len(got.Sites))
	}
	st := got.Sites[0]
	if st.RFSS != 1 || st.SiteID != 2 {
		t.Errorf("site = RFSS %d Site %d, want 1/2", st.RFSS, st.SiteID)
	}
	if len(st.Frequencies) != 2 {
		t.Fatalf("len(Frequencies) = %d, want 2", len(st.Frequencies))
	}
	if st.Frequencies[0].Hz != 851012500 || !st.Frequencies[0].ControlChannel {
		t.Errorf("freq[0] = %+v, want 851012500 control", st.Frequencies[0])
	}
	if len(got.Talkgroups) != 2 {
		t.Fatalf("len(Talkgroups) = %d, want 2", len(got.Talkgroups))
	}
	if got.Talkgroups[0].Dec != 1000 {
		t.Errorf("tg[0].Dec = %d, want 1000", got.Talkgroups[0].Dec)
	}
}

func TestDiscoveredToParsed(t *testing.T) {
	sys := &hunt.DiscoveredSystem{
		Protocol: "dmr",
		SystemID: 0x10,
		Sites: []hunt.DiscoveredSite{{
			RFSS: 0, SiteID: 0,
			ControlChannels: []hunt.DiscoveredChannel{{FrequencyHz: 851000000, IsControl: true}},
		}},
		Talkgroups: []hunt.DiscoveredTalkgroup{{Dec: 5, Hex: "5", Encrypted: true}},
	}
	ps := discoveredToParsed(sys)
	if ps.SysID != "10" {
		t.Errorf("SysID = %q, want 10", ps.SysID)
	}
	if len(ps.Sites) != 1 || ps.Sites[0].SiteName == "" || !ps.Sites[0].Include {
		t.Errorf("site mapping wrong: %+v", ps.Sites)
	}
	if len(ps.Talkgroups) != 1 || !ps.Talkgroups[0].Encrypted || ps.Talkgroups[0].Mode != "D" {
		t.Errorf("talkgroup mapping wrong: %+v", ps.Talkgroups)
	}
}
