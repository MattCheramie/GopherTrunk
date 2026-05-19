package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite testdata/*.golden.json from current parser output")

// TestParseSystem runs the parser against pre-extracted positioned-text
// fixtures. The PDF library is intentionally NOT invoked at test time
// so parser correctness stays decoupled from upstream PDF-extraction
// stability. Regenerate the .extracted.json fixtures with
// `go run ./cmd/gophertrunk/internal/extract` (or the equivalent
// hand-tool) when bumping ledongthuc/pdf.
func TestParseSystem(t *testing.T) {
	cases := []struct {
		name            string
		extractedFile   string
		goldenFile      string
		wantName        string
		wantSysID       string
		wantWACN        string
		wantMinSites    int
		wantMinTGs      int
		wantContainsTag string
	}{
		{
			name:            "maricopa",
			extractedFile:   "testdata/import_maricopa.extracted.json",
			goldenFile:      "testdata/import_maricopa.golden.json",
			wantName:        "Maricopa County",
			wantSysID:       "49A",
			wantWACN:        "BEE00",
			wantMinSites:    15,
			wantMinTGs:      40,
			wantContainsTag: "Law Dispatch",
		},
		{
			name:            "rwc",
			extractedFile:   "testdata/import_rwc.extracted.json",
			goldenFile:      "testdata/import_rwc.golden.json",
			wantName:        "Regional Wireless Cooperative (RWC)",
			wantSysID:       "534",
			wantWACN:        "BEE08",
			wantMinSites:    18,
			wantMinTGs:      100,
			wantContainsTag: "Interop",
		},
		{
			name:            "mmr",
			extractedFile:   "testdata/import_mmr.extracted.json",
			goldenFile:      "testdata/import_mmr.golden.json",
			wantName:        "Metropolitan Mobile Radio (MMR) System",
			wantSysID:       "164",
			wantWACN:        "BEE00",
			wantMinSites:    40,
			wantMinTGs:      500,
			wantContainsTag: "Law Dispatch",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(tc.extractedFile)
			if err != nil {
				t.Fatalf("open %s: %v", tc.extractedFile, err)
			}
			defer f.Close()
			rows, err := loadParseRowsJSON(f)
			if err != nil {
				t.Fatalf("decode %s: %v", tc.extractedFile, err)
			}

			sys, err := parseSystem(rows)
			if err != nil {
				t.Fatalf("parseSystem: %v", err)
			}

			if sys.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", sys.Name, tc.wantName)
			}
			if sys.SysID != tc.wantSysID {
				t.Errorf("SysID = %q, want %q", sys.SysID, tc.wantSysID)
			}
			if sys.WACN != tc.wantWACN {
				t.Errorf("WACN = %q, want %q", sys.WACN, tc.wantWACN)
			}
			if len(sys.Sites) < tc.wantMinSites {
				t.Errorf("Sites = %d, want >= %d", len(sys.Sites), tc.wantMinSites)
			}
			if len(sys.Talkgroups) < tc.wantMinTGs {
				t.Errorf("Talkgroups = %d, want >= %d", len(sys.Talkgroups), tc.wantMinTGs)
			}

			// Every site has at least one frequency. CC requirement
			// is system-wide rather than per-site: some MMR sites
			// are voice-only repeaters with no control channel.
			totalCC := 0
			for i, site := range sys.Sites {
				if len(site.Frequencies) == 0 {
					t.Errorf("Sites[%d] %q has no frequencies", i, site.SiteName)
				}
				for _, f := range site.Frequencies {
					if f.ControlChannel {
						totalCC++
					}
				}
			}
			if totalCC == 0 {
				t.Errorf("system has zero control-channel-capable frequencies across all sites")
			}

			// At least one talkgroup carries the canonical Tag we expect.
			seen := false
			for _, tg := range sys.Talkgroups {
				if tg.Tag == tc.wantContainsTag {
					seen = true
					break
				}
			}
			if !seen {
				t.Errorf("no talkgroup tagged %q found (parser tag split likely wrong)", tc.wantContainsTag)
			}

			// All talkgroups have DEC ≤ 65535 (P25 limit) and Hex matches DEC.
			for i, tg := range sys.Talkgroups {
				if tg.Dec > 65535 {
					t.Errorf("Talkgroups[%d] DEC %d > 65535 (P25 limit)", i, tg.Dec)
				}
				if tg.Mode != "D" && tg.Mode != "A" {
					t.Errorf("Talkgroups[%d] Mode = %q, want D or A", i, tg.Mode)
				}
				if tg.AlphaTag == "" {
					t.Errorf("Talkgroups[%d] DEC=%d has empty AlphaTag", i, tg.Dec)
				}
			}

			// Golden file round-trip.
			if *updateGolden {
				writeGolden(t, tc.goldenFile, sys)
				return
			}
			compareGolden(t, tc.goldenFile, sys)
		})
	}
}

func writeGolden(t *testing.T, path string, sys parsedSystem) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(sys); err != nil {
		t.Fatal(err)
	}
}

func compareGolden(t *testing.T, path string, sys parsedSystem) {
	t.Helper()
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file %s missing; rerun with -update-golden: %v", path, err)
	}
	var goldenSys parsedSystem
	if err := json.Unmarshal(want, &goldenSys); err != nil {
		t.Fatalf("decode golden %s: %v", path, err)
	}
	got, err := json.MarshalIndent(sys, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.MarshalIndent(goldenSys, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(wantJSON) {
		t.Errorf("parsed system != golden; rerun with -update-golden after inspecting the diff.\nfirst differing line:\n%s", firstDiff(string(wantJSON), string(got)))
	}
}

func firstDiff(a, b string) string {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(la) && i < len(lb); i++ {
		if la[i] != lb[i] {
			return "  want: " + la[i] + "\n  got:  " + lb[i]
		}
	}
	if len(la) != len(lb) {
		return "line count differs"
	}
	return "(identical)"
}

// TestSplitTalkgroupTail unit-tests the most-error-prone helper without
// going through the full parser.
func TestSplitTalkgroupTail(t *testing.T) {
	cases := []struct {
		in        string
		wantAlpha string
		wantDesc  string
		wantTag   string
	}{
		// Common patterns. Alpha/Description splits are inherently
		// heuristic when the PDF lacks column separators — we assert
		// the Tag column (the only one the daemon uses for scanner
		// behaviour) and the rough Alpha shape.
		{"MCSO Ops Operations with outside agencies Interop", "MCSO Ops", "Operations with outside agencies", "Interop"},
		{"MCSO D1 SE D1 Southeast Law Dispatch", "MCSO D1 SE D1", "Southeast", "Law Dispatch"},
		{"Probation 1101 Probation Officers Corrections", "Probation 1101", "Probation Officers", "Corrections"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			a, d, g := splitTalkgroupTail(tc.in)
			if a != tc.wantAlpha {
				t.Errorf("alpha = %q, want %q", a, tc.wantAlpha)
			}
			if d != tc.wantDesc {
				t.Errorf("desc = %q, want %q", d, tc.wantDesc)
			}
			if g != tc.wantTag {
				t.Errorf("tag = %q, want %q", g, tc.wantTag)
			}
		})
	}
}

func TestPDFTextNeedsShift(t *testing.T) {
	cases := []struct {
		name string
		rows []rawRow
		want bool
	}{
		{"system-name anchor", []rawRow{{Text: "System Name: Foo"}}, false},
		{"sites-and-frequencies anchor", []rawRow{{Text: "Sites and Frequencies"}}, false},
		{"wacn anchor", []rawRow{{Text: "WACN: BEE08"}}, false},
		{"last-updated anchor", []rawRow{{Text: "Last Updated: June 2024"}}, false},
		{"talkgroups anchor", []rawRow{{Text: "Talkgroups"}}, false},
		{"shifted MMR-style", []rawRow{{Text: "nystem;i|meU hetropolit|n"}}, true},
		{"shifted maricopa-style", []rawRow{{Text: "2FWNHTUF\x05(TZSY^"}}, true},
		{"empty", []rawRow{}, true},
		{"anchor not in first 50 rows", append(
			make([]rawRow, 60),
			rawRow{Text: "System Name: Late"},
		), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pdfTextNeedsShift(tc.rows)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDecodeShift_LiteralSpacesPreserved(t *testing.T) {
	// Maricopa-style raw bytes for "Maricopa County Menu". The
	// gap before "Menu" is a literal 0x20 followed by an encoded
	// 0x05 — the literal space must not be shifted to ';'.
	in := "2FWNHTUF\x05(TZSY^ \x052JSZ"
	want := "Maricopa County  Menu"
	got := decodeShift(in)
	if got != want {
		t.Errorf("decodeShift = %q, want %q", got, want)
	}
}

func TestParseMetaLine_Variants(t *testing.T) {
	cases := []struct {
		in        string
		wantName  string
		wantLoc   string
		wantSysID string
		wantWACN  string
	}{
		{in: "System Name: Foo", wantName: "Foo"},
		{in: "System Name : Foo", wantName: "Foo"},
		{in: "  System Name:   Foo  ", wantName: "Foo"},
		{in: "SYSTEM NAME: Foo", wantName: "Foo"},
		{in: "system name: Foo Bar", wantName: "Foo Bar"},
		{in: "Location: Phoenix, AZ", wantLoc: "Phoenix, AZ"},
		{in: "System ID: Sysid: 49A WACN: BEE00", wantSysID: "49A", wantWACN: "BEE00"},
		{in: "SYSTEM ID: sysid: 49A wacn: BEE00", wantSysID: "49A", wantWACN: "BEE00"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			var sys parsedSystem
			parseMetaLine(tc.in, &sys)
			if sys.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", sys.Name, tc.wantName)
			}
			if sys.Location != tc.wantLoc {
				t.Errorf("Location = %q, want %q", sys.Location, tc.wantLoc)
			}
			if sys.SysID != tc.wantSysID {
				t.Errorf("SysID = %q, want %q", sys.SysID, tc.wantSysID)
			}
			if sys.WACN != tc.wantWACN {
				t.Errorf("WACN = %q, want %q", sys.WACN, tc.wantWACN)
			}
		})
	}
}

func TestParseSystem_PageTitleFallback(t *testing.T) {
	// No "System Name:" line — only the "<Name> Menu" banner.
	// Followed by a minimum-viable Sites section so the parser
	// doesn't trip the "neither sites nor talkgroups" check.
	rows := []parseRow{
		{Y: 700, Page: 1, Text: "Maricopa County  Menu"},
		{Y: 650, Page: 1, Text: "Sites and Frequencies"},
		{Y: 600, Page: 1, Text: "1 (1) 016 (10) Oatman Mountain Maricopa 769.54375c 770.04375"},
	}
	sys, err := parseSystem(rows)
	if err != nil {
		t.Fatalf("parseSystem: %v", err)
	}
	if sys.Name != "Maricopa County" {
		t.Errorf("Name = %q, want %q", sys.Name, "Maricopa County")
	}
	if len(sys.Sites) != 1 {
		t.Fatalf("Sites = %d, want 1", len(sys.Sites))
	}
}

func TestParseSystem_NoSystemNameDumpsRows(t *testing.T) {
	// 50 rows, no system name anywhere, no page-title banner.
	rows := make([]parseRow, 50)
	for i := range rows {
		rows[i] = parseRow{Y: float64(700 - i), Page: 1, Text: fmt.Sprintf("row-%02d-content", i)}
	}
	_, err := parseSystem(rows)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no System Name found") {
		t.Errorf("error missing canonical phrase: %s", msg)
	}
	if !strings.Contains(msg, "first extracted rows") {
		t.Errorf("error missing diagnostic dump header: %s", msg)
	}
	if !strings.Contains(msg, "row-00-content") {
		t.Errorf("error missing first row text: %s", msg)
	}
	if strings.Contains(msg, "row-30-content") {
		t.Errorf("error should cap at 30 rows but contains row-30: %s", msg)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("abc", 10); got != "abc" {
		t.Errorf("short string truncated: %q", got)
	}
	if got := truncateRunes("abcdefghij", 5); got != "abcde…" {
		t.Errorf("truncate = %q, want abcde…", got)
	}
}

// TestParseFreqList sanity-checks freq parsing and CC marker handling.
func TestParseFreqList(t *testing.T) {
	got := parseFreqList("769.54375c 770.04375 851.075 99.999c")
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (99.999 below P25 band, should drop)", len(got))
	}
	if !got[0].ControlChannel {
		t.Error("first freq should be CC")
	}
	if got[1].ControlChannel {
		t.Error("second freq should NOT be CC")
	}
	if got[0].Hz != 769543750 {
		t.Errorf("first Hz = %d, want 769543750", got[0].Hz)
	}
}
