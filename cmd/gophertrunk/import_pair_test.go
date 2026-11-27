package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// A RadioReference talkgroup export, in the header RadioReference writes.
const rrTalkgroupCSV = `Decimal,Hex,Alpha Tag,Mode,Description,Tag,Category
1000,3e8,OPS,D,Operations,Law Dispatch,Police
1001,3e9,TAC1,D,Tactical 1,Law Tac,Police
1002,3ea,FIRE,D,Fire Dispatch,Fire Dispatch,Fire
`

func rrPairSources(t *testing.T, tgName, sitesName string) []api.ImportSource {
	t.Helper()
	sites, err := os.ReadFile("testdata/rr_trs_sites_2951.csv") // the export attached to #849
	if err != nil {
		t.Fatal(err)
	}
	return []api.ImportSource{
		{Filename: tgName, Kind: api.ImportSourceCSV, Data: []byte(rrTalkgroupCSV)},
		{Filename: sitesName, Kind: api.ImportSourceCSV, Data: sites},
	}
}

// Issue #1403: a system's RadioReference talkgroup CSV and sites CSV,
// selected together, imported as two systems: one with talkgroups and no
// control channels, the other with control channels and no talkgroups. They
// must preview as ONE system with both.
func TestImportJoinsRRTalkgroupAndSitesCSVs(t *testing.T) {
	imp := &daemonImporter{}
	for _, order := range [][2]int{{0, 1}, {1, 0}} {
		src := rrPairSources(t, "trs_tg_2951.csv", "trs_sites_2951.csv")
		previews, err := imp.ParseAll([]api.ImportSource{src[order[0]], src[order[1]]})
		if err != nil {
			t.Fatalf("ParseAll: %v", err)
		}
		if len(previews) != 1 {
			t.Fatalf("upload order %v: %d systems, want the two exports joined into 1: %+v", order, len(previews), previews)
		}
		p := previews[0]
		if len(p.ControlChannels) == 0 || len(p.Talkgroups) != 3 {
			t.Errorf("joined system has %d control channels and %d talkgroups, want both", len(p.ControlChannels), len(p.Talkgroups))
		}
		if p.Name != "trs_tg_2951" {
			t.Errorf("Name = %q, want the talkgroup export's (trs_tg_2951)", p.Name)
		}
		for _, f := range []string{"trs_tg_2951.csv", "trs_sites_2951.csv"} {
			if !strings.Contains(p.SourcePath, f) {
				t.Errorf("SourcePath %q does not name %s", p.SourcePath, f)
			}
		}
	}
}

// The commit must merge what the preview showed: one system in config.yaml,
// with control channels and a talkgroup file.
func TestImportCommitWritesOneSystemForRRCSVPair(t *testing.T) {
	parsed, err := parseImportSources(rrPairSources(t, "trs_tg_2951.csv", "trs_sites_2951.csv"))
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	res, err := mergeIntoConfig(parsed, mergeOptions{ConfigPath: cfgPath, CSVDir: filepath.Dir(cfgPath)})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(cfg.Trunking.Systems); n != 1 {
		t.Fatalf("config has %d systems, want 1", n)
	}
	sys := cfg.Trunking.Systems[0]
	if len(sys.ControlChannels) == 0 || sys.TalkgroupFile == "" || len(res.CSVs) != 1 {
		t.Errorf("system has %d control channels, talkgroup file %q, %d CSVs written; want all three",
			len(sys.ControlChannels), sys.TalkgroupFile, len(res.CSVs))
	}
}

func TestPairRRNativeCSVs(t *testing.T) {
	tg := func(name, sid string) parsedSystem {
		return parsedSystem{Name: name, rrCSV: rrCSVTalkgroups, rrSID: sid, Talkgroups: []parsedTalkgroup{{Dec: 1}}}
	}
	sites := func(name, sid string) parsedSystem {
		return parsedSystem{Name: name, rrCSV: rrCSVSites, rrSID: sid, Sites: []parsedSite{{SiteName: name}}}
	}
	bundle := parsedSystem{Name: "bundle", Talkgroups: []parsedTalkgroup{{Dec: 9}}}

	cases := []struct {
		name string
		in   []parsedSystem
		want []string // "name:sites/talkgroups" of each result
	}{
		{"same id", []parsedSystem{tg("A", "7715"), sites("A-sites", "7715")}, []string{"A:1/1"}},
		{"renamed files, one of each", []parsedSystem{sites("s", ""), tg("t", "")}, []string{"t:1/1"}},
		{"one id missing", []parsedSystem{tg("t", "7715"), sites("s", "")}, []string{"t:1/1"}},
		{"two different systems", []parsedSystem{tg("A", "7715"), sites("B", "2951")}, []string{"A:0/1", "B:1/0"}},
		{"two systems, both pairs", []parsedSystem{tg("A", "1"), tg("B", "2"), sites("B-s", "2"), sites("A-s", "1")}, []string{"A:1/1", "B:1/1"}},
		{"ambiguous without ids", []parsedSystem{tg("A", ""), tg("B", ""), sites("s", "")}, []string{"A:0/1", "B:0/1", "s:1/0"}},
		{"a lone export", []parsedSystem{tg("A", "7715")}, []string{"A:0/1"}},
		{"a bundle is never half a pair", []parsedSystem{bundle, sites("s", "")}, []string{"bundle:0/1", "s:1/0"}},
	}
	for _, tc := range cases {
		var got []string
		for _, s := range pairRRNativeCSVs(tc.in) {
			got = append(got, s.Name+":"+itoa(len(s.Sites))+"/"+itoa(len(s.Talkgroups)))
		}
		if strings.Join(got, " ") != strings.Join(tc.want, " ") {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func itoa(n int) string { return string(rune('0' + n)) }

func TestRRExportSID(t *testing.T) {
	cases := map[string]string{
		"trs_tg_7715.csv":               "7715",
		"/tmp/x/trs_sites_2951.csv":     "2951",
		"TRS_TG_12 (1).csv":             "12",
		"gophertrunk-import-123456.csv": "",
		"my-talkgroups.csv":             "",
	}
	for name, want := range cases {
		if got := rrExportSID(name); got != want {
			t.Errorf("rrExportSID(%q) = %q, want %q", name, got, want)
		}
	}
	// An upload is parsed from a tempfile; the id comes from the name it was
	// uploaded under.
	if got := rrExportSID("/tmp/gophertrunk-import-1.csv", "trs_tg_7715"); got != "7715" {
		t.Errorf("fallback name: got %q, want 7715", got)
	}
}
