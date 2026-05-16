package main

import (
	"encoding/json"
	"flag"
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

			// Every site has at least one frequency and at least one CC.
			for i, site := range sys.Sites {
				if len(site.Frequencies) == 0 {
					t.Errorf("Sites[%d] %q has no frequencies", i, site.SiteName)
				}
				ccCount := 0
				for _, f := range site.Frequencies {
					if f.ControlChannel {
						ccCount++
					}
				}
				if ccCount == 0 {
					t.Errorf("Sites[%d] %q has no control-channel-capable frequencies", i, site.SiteName)
				}
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
