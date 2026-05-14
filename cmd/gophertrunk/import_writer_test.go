package main

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

const seedConfig = `# Existing configuration with comments that must survive a merge.
log:
  level: info
  format: text

api:
  http_addr: "127.0.0.1:8080"

# Trunking systems already configured by the operator.
trunking:
  systems:
    - name: "Pre-existing"
      protocol: p25
      control_channels:
        - 851000000
      talkgroup_file: "/etc/gophertrunk/pre.csv"
`

func TestMergeIntoConfig_AddsSystem(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(seedConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	sys := sampleParsedSystem()
	res, err := mergeIntoConfig([]parsedSystem{sys}, mergeOptions{ConfigPath: cfgPath})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	// Existing system survives, new system appended.
	merged := string(res.ConfigYAML)
	if !strings.Contains(merged, "Pre-existing") {
		t.Errorf("seed system lost from merged YAML")
	}
	if !strings.Contains(merged, sys.Name) {
		t.Errorf("new system %q missing from merged YAML:\n%s", sys.Name, merged)
	}
	// Comments survive.
	if !strings.Contains(merged, "# Existing configuration") {
		t.Errorf("head comment lost in merge:\n%s", merged)
	}

	// Files actually exist on disk after non-dry-run.
	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf("config.yaml missing after merge: %v", err)
	}
	if len(res.CSVs) != 1 {
		t.Fatalf("expected 1 CSV, got %d", len(res.CSVs))
	}
	if _, err := os.Stat(res.CSVs[0].Path); err != nil {
		t.Errorf("CSV missing after merge: %v", err)
	}

	// Daemon's config loader accepts the merged file.
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load on merged file failed: %v", err)
	}
	if len(cfg.Trunking.Systems) != 2 {
		t.Errorf("Trunking.Systems = %d, want 2", len(cfg.Trunking.Systems))
	}

	// Daemon's talkgroup loader accepts the CSV.
	db := trunking.NewTalkgroupDB()
	count, err := db.LoadCSVFile(res.CSVs[0].Path)
	if err != nil {
		t.Fatalf("TalkgroupDB.LoadCSV: %v", err)
	}
	if count != len(sys.Talkgroups) {
		t.Errorf("loaded %d talkgroups, expected %d", count, len(sys.Talkgroups))
	}
	// Spot-check first talkgroup ID round-trip.
	if tg := db.Lookup(sys.Talkgroups[0].Dec); tg == nil {
		t.Errorf("talkgroup %d not found in DB after CSV reload", sys.Talkgroups[0].Dec)
	} else if tg.AlphaTag != sys.Talkgroups[0].AlphaTag {
		t.Errorf("AlphaTag round-trip = %q, want %q", tg.AlphaTag, sys.Talkgroups[0].AlphaTag)
	}
}

func TestMergeIntoConfig_NameCollisionRequiresForce(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(seedConfig), 0o644)

	sys := sampleParsedSystem()
	sys.Name = "Pre-existing" // same name as seed
	_, err := mergeIntoConfig([]parsedSystem{sys}, mergeOptions{ConfigPath: cfgPath})
	if err == nil {
		t.Fatal("expected collision error without --force, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error message = %q, want it to mention 'already exists'", err.Error())
	}

	// With force, it should succeed and overwrite.
	res, err := mergeIntoConfig([]parsedSystem{sys}, mergeOptions{ConfigPath: cfgPath, Force: true})
	if err != nil {
		t.Fatalf("merge with --force failed: %v", err)
	}
	// Should still have exactly one "Pre-existing" entry (replaced, not duplicated).
	occurrences := strings.Count(string(res.ConfigYAML), "name: Pre-existing")
	if occurrences != 1 {
		t.Errorf("name: Pre-existing appears %d times, want 1", occurrences)
	}
}

func TestMergeIntoConfig_DryRun(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(seedConfig), 0o644)
	originalBytes, _ := os.ReadFile(cfgPath)

	sys := sampleParsedSystem()
	res, err := mergeIntoConfig([]parsedSystem{sys}, mergeOptions{ConfigPath: cfgPath, DryRun: true})
	if err != nil {
		t.Fatalf("dry-run merge: %v", err)
	}
	if len(res.ConfigYAML) == 0 {
		t.Error("dry-run should still produce ConfigYAML buffer")
	}
	if len(res.CSVs) == 0 {
		t.Error("dry-run should still produce CSV buffers")
	}

	// File on disk MUST be unchanged.
	after, _ := os.ReadFile(cfgPath)
	if !bytes.Equal(originalBytes, after) {
		t.Error("dry-run modified config.yaml on disk")
	}
	// CSV must NOT exist.
	if _, err := os.Stat(res.CSVs[0].Path); err == nil {
		t.Error("dry-run created CSV file on disk")
	}
}

func TestMergeIntoConfig_EmptyConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	// File doesn't exist — merge should synthesise the structure.

	sys := sampleParsedSystem()
	_, err := mergeIntoConfig([]parsedSystem{sys}, mergeOptions{ConfigPath: cfgPath})
	if err != nil {
		t.Fatalf("merge into missing config: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(cfg.Trunking.Systems) != 1 {
		t.Errorf("Trunking.Systems = %d, want 1", len(cfg.Trunking.Systems))
	}
}

func TestBuildSlug(t *testing.T) {
	cases := []struct {
		name, sysid, want string
	}{
		{"Maricopa County", "49A", "maricopa-county-49a"},
		{"Regional Wireless Cooperative (RWC)", "534", "regional-wireless-cooperative-rwc-534"},
		{"  Trim  Me  ", "FF", "trim-me-ff"},
	}
	for _, tc := range cases {
		got := buildSlug(tc.name, tc.sysid)
		if got != tc.want {
			t.Errorf("buildSlug(%q,%q) = %q, want %q", tc.name, tc.sysid, got, tc.want)
		}
	}
}

func TestBuildTalkgroupCSV_LoaderRoundTrip(t *testing.T) {
	sys := sampleParsedSystem()
	b, err := buildTalkgroupCSV(sys)
	if err != nil {
		t.Fatal(err)
	}
	// Parse via stdlib csv to verify header order.
	r := csv.NewReader(bytes.NewReader(b))
	header, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}
	wantCols := []string{"Decimal", "Hex", "Mode", "Alpha Tag", "Description", "Tag", "Group", "Priority", "Lockout", "Scan"}
	if len(header) != len(wantCols) {
		t.Fatalf("header len = %d, want %d", len(header), len(wantCols))
	}
	for i, c := range wantCols {
		if header[i] != c {
			t.Errorf("header[%d] = %q, want %q", i, header[i], c)
		}
	}
}

// sampleParsedSystem returns a small, valid parsedSystem for writer tests.
func sampleParsedSystem() parsedSystem {
	return parsedSystem{
		Name:       "Test System",
		Protocol:   "p25",
		SysID:      "ABC",
		WACN:       "BEE99",
		SystemType: "Project 25 Phase II",
		Sites: []parsedSite{
			{
				RFSS: 1, SiteID: 1, SiteName: "Hill A", Cty: "Test",
				Include: true,
				Frequencies: []parsedFreq{
					{Hz: 851012500, ControlChannel: true},
					{Hz: 851262500, ControlChannel: false},
				},
			},
			{
				RFSS: 1, SiteID: 2, SiteName: "Hill B", Cty: "Test",
				Include: true,
				Frequencies: []parsedFreq{
					{Hz: 852012500, ControlChannel: true},
				},
			},
		},
		Talkgroups: []parsedTalkgroup{
			{Dec: 1000, Hex: "3e8", Mode: "D", AlphaTag: "OPS 1", Description: "Operations", Tag: "Law Dispatch", Group: "Police", Scan: true},
			{Dec: 1001, Hex: "3e9", Mode: "D", AlphaTag: "OPS 2", Description: "Tactical", Tag: "Law Tac", Group: "Police", Scan: true},
		},
	}
}
