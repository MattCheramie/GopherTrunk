package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseCSVStream_HappyPath(t *testing.T) {
	const in = `# Section: metadata
key,value
name,Test
protocol,p25
sysid,1A
wacn,BEE99

# Section: sites
rfss,site_id,site_name,county,frequencies
1,1,Hill A,Test,851.0125c|851.2625|852.0125c
1,2,Hill B,Test,853.0125c

# Section: talkgroups
decimal,hex,mode,alpha_tag,description,tag,group,priority,lockout,scan
1000,3e8,D,OPS,Operations,Law Dispatch,Police,1,,Y
1001,,D,TAC,Tactical,Law Tac,Police,,,Y
9999,,A,ANALOG,Analog,Multi-Tac,Common,,Y,N
`
	sys, err := parseCSVStream(strings.NewReader(in))
	if err != nil {
		t.Fatalf("parseCSVStream: %v", err)
	}
	if sys.Name != "Test" {
		t.Errorf("Name = %q, want Test", sys.Name)
	}
	if sys.Protocol != "p25" {
		t.Errorf("Protocol = %q, want p25", sys.Protocol)
	}
	if len(sys.Sites) != 2 {
		t.Fatalf("Sites = %d, want 2", len(sys.Sites))
	}
	if sys.Sites[0].SiteName != "Hill A" {
		t.Errorf("first site name = %q", sys.Sites[0].SiteName)
	}
	if len(sys.Sites[0].Frequencies) != 3 {
		t.Errorf("first site freqs = %d, want 3", len(sys.Sites[0].Frequencies))
	}
	if !sys.Sites[0].Frequencies[0].ControlChannel {
		t.Errorf("first freq should be control channel")
	}
	if sys.Sites[0].Frequencies[1].ControlChannel {
		t.Errorf("second freq should NOT be control channel")
	}
	if sys.Sites[0].Frequencies[0].Hz != 851012500 {
		t.Errorf("first Hz = %d, want 851012500", sys.Sites[0].Frequencies[0].Hz)
	}
	if !sys.Sites[0].Include {
		t.Errorf("Include should default to true")
	}

	if len(sys.Talkgroups) != 3 {
		t.Fatalf("Talkgroups = %d, want 3", len(sys.Talkgroups))
	}
	if sys.Talkgroups[1].Hex != "3e9" {
		t.Errorf("auto-computed hex = %q, want 3e9", sys.Talkgroups[1].Hex)
	}
	if sys.Talkgroups[0].Priority != 1 {
		t.Errorf("priority = %d, want 1", sys.Talkgroups[0].Priority)
	}
	if sys.Talkgroups[2].Scan {
		t.Errorf("third talkgroup Scan should be false")
	}
	if !sys.Talkgroups[2].Lockout {
		t.Errorf("third talkgroup Lockout should be true")
	}
}

func TestParseCSVStream_OrderIndependent(t *testing.T) {
	// Sections in reversed order — should still parse.
	const in = `# Section: talkgroups
decimal,alpha_tag
1000,OPS

# Section: sites
site_name,frequencies
Hill,851.0125c

# Section: metadata
key,value
name,Reordered
protocol,p25
`
	sys, err := parseCSVStream(strings.NewReader(in))
	if err != nil {
		t.Fatalf("parseCSVStream: %v", err)
	}
	if sys.Name != "Reordered" {
		t.Errorf("Name = %q", sys.Name)
	}
	if len(sys.Sites) != 1 || len(sys.Talkgroups) != 1 {
		t.Errorf("Sites=%d Talkgroups=%d", len(sys.Sites), len(sys.Talkgroups))
	}
}

func TestParseCSVStream_MissingName(t *testing.T) {
	const in = `# Section: metadata
key,value
protocol,p25

# Section: sites
site_name,frequencies
Hill,851.0125c
`
	_, err := parseCSVStream(strings.NewReader(in))
	if err == nil || !strings.Contains(err.Error(), "missing metadata.name") {
		t.Errorf("expected name-required error, got %v", err)
	}
}

func TestParseCSVStream_UnknownSection(t *testing.T) {
	const in = `# Section: metadata
key,value
name,X
protocol,p25

# Section: gibberish
foo,bar
`
	_, err := parseCSVStream(strings.NewReader(in))
	if err == nil || !strings.Contains(err.Error(), "unknown section") {
		t.Errorf("expected unknown-section error, got %v", err)
	}
}

func TestParseCSVStream_QuotedFieldWithComma(t *testing.T) {
	const in = `# Section: metadata
key,value
name,"Quoted, Name"
protocol,p25

# Section: talkgroups
decimal,alpha_tag,description
1000,OPS,"Description, with comma"
`
	sys, err := parseCSVStream(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if sys.Name != "Quoted, Name" {
		t.Errorf("Name = %q, want 'Quoted, Name'", sys.Name)
	}
	if sys.Talkgroups[0].Description != "Description, with comma" {
		t.Errorf("desc = %q", sys.Talkgroups[0].Description)
	}
}

func TestParseCSVStream_AliasedHeaders(t *testing.T) {
	// Use Trunk-Recorder-style headers ("Alpha Tag", "DEC", etc.).
	const in = `# Section: metadata
field,val
name,Aliased
protocol,p25

# Section: talkgroups
DEC,Alpha Tag,Description,Category,Priority
1000,OPS,Operations,Police,3
`
	sys, err := parseCSVStream(strings.NewReader(in))
	if err != nil {
		t.Fatalf("aliased headers failed: %v", err)
	}
	if sys.Name != "Aliased" {
		t.Errorf("Name = %q", sys.Name)
	}
	if sys.Talkgroups[0].Tag != "Police" {
		t.Errorf("Tag = %q (Category alias should map to Tag)", sys.Talkgroups[0].Tag)
	}
	if sys.Talkgroups[0].Priority != 3 {
		t.Errorf("Priority = %d", sys.Talkgroups[0].Priority)
	}
}

func TestParseCSVFrequencies_Separators(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"851.0125c|851.2625|852.0125c", 3},
		{"851.0125c;851.2625;852.0125", 3},  // semicolons
		{"851.0125c 851.2625 852.0125c", 3}, // spaces
		{"", 0},
	}
	for _, tc := range cases {
		got, err := parseCSVFrequencies(tc.in)
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if len(got) != tc.want {
			t.Errorf("%q: got %d, want %d", tc.in, len(got), tc.want)
		}
	}
}

func TestParseCSVFile_ExampleFixture(t *testing.T) {
	sys, err := parseCSVFile("../../samples/rr-import/example.csv", csvImportOpts{})
	if err != nil {
		t.Fatalf("parseCSVFile: %v", err)
	}
	if sys.Name != "Example P25 System" {
		t.Errorf("Name = %q", sys.Name)
	}
	if len(sys.Sites) != 2 {
		t.Errorf("Sites = %d, want 2", len(sys.Sites))
	}
	if len(sys.Talkgroups) != 6 {
		t.Errorf("Talkgroups = %d, want 6", len(sys.Talkgroups))
	}
	// Last talkgroup: analog, lockout, scan=false (verifies bool parsing).
	last := sys.Talkgroups[5]
	if last.Mode != "A" {
		t.Errorf("last Mode = %q, want A", last.Mode)
	}
	if !last.Lockout {
		t.Errorf("last Lockout should be true")
	}
	if last.Scan {
		t.Errorf("last Scan should be false")
	}
}

func TestParseCSVStream_EndToEndMerge(t *testing.T) {
	// The whole point: a CSV-imported system should flow through the
	// same writer the PDF importer uses.
	const in = `# Section: metadata
key,value
name,CSV E2E
protocol,p25

# Section: sites
site_name,frequencies
HillX,851.0125c|851.2625c

# Section: talkgroups
decimal,alpha_tag,tag
1000,OPS,Law Dispatch
1001,TAC,Law Tac
`
	sys, err := parseCSVStream(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfgPath := dir + "/config.yaml"
	res, err := mergeIntoConfig([]parsedSystem{sys}, mergeOptions{ConfigPath: cfgPath})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(res.CSVs) != 1 {
		t.Errorf("expected 1 CSV output, got %d", len(res.CSVs))
	}
}

func TestLooksLikeRRNativeCSV(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{
			name: "rr native header",
			in:   "DEC,HEX,Mode,Alpha Tag,Description,Tag,Group\n1000,3e8,D,OPS,Operations,Law Dispatch,Police\n",
			want: true,
		},
		{
			name: "lower-case columns",
			in:   "decimal,hex,mode,alpha_tag,description,tag,group\n1000,3e8,D,OPS,Ops,Law Dispatch,Police\n",
			want: true,
		},
		{
			name: "bundle with section marker",
			in:   "# Section: metadata\nkey,value\nname,Foo\n# Section: talkgroups\ndecimal,alpha_tag\n1000,OPS\n",
			want: false,
		},
		{
			name: "blank file",
			in:   "",
			want: false,
		},
		{
			name: "single column",
			in:   "name\nfoo\n",
			want: false,
		},
		{
			name: "unrelated csv",
			in:   "a,b,c,d,e,f\n1,2,3,4,5,6\n",
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := looksLikeRRNativeCSV([]byte(tc.in))
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseRRNativeCSVStream(t *testing.T) {
	const in = `DEC,HEX,Mode,Alpha Tag,Description,Tag,Group
1000,3e8,D,OPS,Operations,Law Dispatch,Police
1001,3e9,D,TAC1,Tactical 1,Law Tac,Police
1002,,D,FIRE,Fire Dispatch,Fire Dispatch,Fire
`
	sys, err := parseRRNativeCSVStream(strings.NewReader(in), csvImportOpts{Name: "Test", SysID: "49A"})
	if err != nil {
		t.Fatalf("parseRRNativeCSVStream: %v", err)
	}
	if sys.Name != "Test" {
		t.Errorf("Name = %q, want Test", sys.Name)
	}
	if sys.SysID != "49A" {
		t.Errorf("SysID = %q, want 49A", sys.SysID)
	}
	if sys.Protocol != "p25" {
		t.Errorf("Protocol = %q, want p25", sys.Protocol)
	}
	if len(sys.Talkgroups) != 3 {
		t.Fatalf("Talkgroups = %d, want 3", len(sys.Talkgroups))
	}
	if sys.Talkgroups[2].Hex != "3ea" {
		t.Errorf("third hex auto-fill = %q, want 3ea", sys.Talkgroups[2].Hex)
	}
	if len(sys.Sites) != 0 {
		t.Errorf("Sites should be empty for RR native CSV, got %d", len(sys.Sites))
	}
}

func TestParseRRNativeCSVStream_MissingName(t *testing.T) {
	const in = "DEC,HEX,Mode,Alpha Tag,Description,Tag,Group\n1000,3e8,D,OPS,Operations,Law Dispatch,Police\n"
	_, err := parseRRNativeCSVStream(strings.NewReader(in), csvImportOpts{})
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if !strings.Contains(err.Error(), "missing system name") {
		t.Errorf("error %q missing expected phrase", err.Error())
	}
}

func TestParseCSVFile_RRNativeWithFilenameStem(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/maricopa-49A.csv"
	body := "DEC,HEX,Mode,Alpha Tag,Description,Tag,Group\n1000,3e8,D,OPS,Operations,Law Dispatch,Police\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sys, err := parseCSVFile(path, csvImportOpts{})
	if err != nil {
		t.Fatalf("parseCSVFile: %v", err)
	}
	if sys.Name != "maricopa-49A" {
		t.Errorf("Name = %q, want maricopa-49A (filename stem fallback)", sys.Name)
	}
	if len(sys.Talkgroups) != 1 {
		t.Errorf("Talkgroups = %d, want 1", len(sys.Talkgroups))
	}
}

func TestParseCSVFile_BundleStillWorks(t *testing.T) {
	// parseCSVFile must continue to dispatch bundle files to
	// parseCSVStream — the sniffer should reject them.
	dir := t.TempDir()
	path := dir + "/bundle.csv"
	body := `# Section: metadata
key,value
name,BundleTest
protocol,p25

# Section: talkgroups
decimal,alpha_tag
1000,OPS
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sys, err := parseCSVFile(path, csvImportOpts{})
	if err != nil {
		t.Fatalf("parseCSVFile: %v", err)
	}
	if sys.Name != "BundleTest" {
		t.Errorf("Name = %q, want BundleTest", sys.Name)
	}
}
