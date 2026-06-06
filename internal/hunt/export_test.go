package hunt

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// sampleSystem is a fully-populated discovery used across the exporter tests.
func sampleSystem() *DiscoveredSystem {
	at := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	return &DiscoveredSystem{
		Name:     "Example P25 System",
		Protocol: "p25",
		WACN:     0xBEE99,
		SystemID: 0x49A,
		NAC:      0x4D2,
		County:   "Example County",
		Location: "Example City, AZ",
		Identity: map[string]any{"WACN": uint64(0xBEE99), "SystemID": uint64(0x49A), "NAC": uint64(0x4D2)},
		Sites: []DiscoveredSite{{
			RFSS: 1, SiteID: 1, SiteName: "Tower Alpha", County: "Example",
			ControlChannels: []DiscoveredChannel{
				{FrequencyHz: 851012500, IsControl: true, Confidence: 0.9},
				{FrequencyHz: 851262500, IsControl: true},
			},
		}},
		Talkgroups: []DiscoveredTalkgroup{
			{Dec: 1000, Hex: "3e8", Count: 5, FirstSeen: at},
			{Dec: 1001, Hex: "3e9", Count: 1, Encrypted: true, FirstSeen: at},
		},
		Confidence: 0.9,
	}
}

func TestParseFormat(t *testing.T) {
	cases := map[string]Format{
		"bundle": FormatBundle, "csv": FormatBundle, "": FormatBundle,
		"trunk-recorder": FormatTrunkRecorder, "tr": FormatTrunkRecorder,
		"rr": FormatRR, "radioreference": FormatRR,
	}
	for in, want := range cases {
		got, err := ParseFormat(in)
		if err != nil {
			t.Errorf("ParseFormat(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseFormat(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := ParseFormat("nope"); err == nil {
		t.Error("ParseFormat(nope) expected error")
	}
}

func TestWriteBundle_StructureAndContent(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sampleSystem(), FormatBundle, nil); err != nil {
		t.Fatalf("Write bundle: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"# Section: metadata",
		"name,Example P25 System",
		"protocol,p25",
		"sysid,49A",
		"wacn,BEE99",
		"# Section: sites",
		"rfss,site_id,site_name,county,frequencies",
		"851.0125c|851.2625c",
		"# Section: talkgroups",
		"1000,3e8,D",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bundle missing %q\n--- output ---\n%s", want, out)
		}
	}
	// Location contains a comma → must be quoted by the CSV writer.
	if !strings.Contains(out, `"Example City, AZ"`) {
		t.Errorf("bundle did not quote comma-bearing location:\n%s", out)
	}
}

func TestWriteTrunkRecorder_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sampleSystem(), FormatTrunkRecorder, nil); err != nil {
		t.Fatalf("Write trunk-recorder: %v", err)
	}
	var cfg struct {
		Systems []struct {
			ShortName       string  `json:"shortName"`
			Type            string  `json:"type"`
			ControlChannels []int64 `json:"control_channels"`
		} `json:"systems"`
	}
	if err := json.Unmarshal(buf.Bytes(), &cfg); err != nil {
		t.Fatalf("trunk-recorder output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(cfg.Systems) != 1 {
		t.Fatalf("len(systems) = %d, want 1", len(cfg.Systems))
	}
	s := cfg.Systems[0]
	if s.Type != "p25" {
		t.Errorf("type = %q, want p25", s.Type)
	}
	if s.ShortName != "example-p25-system" {
		t.Errorf("shortName = %q, want example-p25-system", s.ShortName)
	}
	if len(s.ControlChannels) != 2 || s.ControlChannels[0] != 851012500 {
		t.Errorf("control_channels = %v, want [851012500 851262500]", s.ControlChannels)
	}
}

func TestWriteRR_WithAndWithoutHints(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sampleSystem(), FormatRR, nil); err != nil {
		t.Fatalf("Write rr: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"# RadioReference submission: Example P25 System",
		"BEE99",
		"Tower Alpha",
		"851.0125",
		rrSubmitURL,
		"No matching system was found",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RR package missing %q", want)
		}
	}

	buf.Reset()
	hints := []DuplicateHint{{SID: 1234, Name: "Existing Sys", Reason: "WACN+SYSID", Confidence: 0.95}}
	if err := Write(&buf, sampleSystem(), FormatRR, hints); err != nil {
		t.Fatalf("Write rr w/ hints: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, "Possible existing systems") || !strings.Contains(out, "SID **1234**") {
		t.Errorf("RR package did not surface duplicate hint:\n%s", out)
	}
}

func TestFormatMHz(t *testing.T) {
	cases := map[uint32]string{
		851012500: "851.0125",
		854000000: "854",
		853512500: "853.5125",
	}
	for hz, want := range cases {
		if got := formatMHz(hz); got != want {
			t.Errorf("formatMHz(%d) = %q, want %q", hz, got, want)
		}
	}
}
