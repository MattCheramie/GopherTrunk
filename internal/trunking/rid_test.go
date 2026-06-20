package trunking

import (
	"strings"
	"testing"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func TestRIDDBLookup(t *testing.T) {
	d := NewRIDDB()
	d.Add(&RID{ID: 207545, Alias: "CPL-SMITH", Owner: "Cpl. Smith", Watch: true})
	if r := d.Lookup(207545); r == nil || r.Alias != "CPL-SMITH" {
		t.Errorf("Lookup(207545) = %+v", r)
	}
	if r := d.Lookup(99999); r != nil {
		t.Errorf("Lookup(99999) should be nil, got %+v", r)
	}
	if got := d.Len(); got != 1 {
		t.Errorf("Len = %d, want 1", got)
	}
}

func TestRIDDBUpdateFields(t *testing.T) {
	d := NewRIDDB()
	d.Add(&RID{ID: 100, Alias: "BEFORE", Watch: true})
	ok := d.UpdateFields(100, func(r *RID) {
		r.Alias = "AFTER"
		r.Priority = 5
	})
	if !ok {
		t.Fatal("UpdateFields(100) returned false on known RID")
	}
	r := d.Lookup(100)
	if r.Alias != "AFTER" || r.Priority != 5 {
		t.Errorf("after update = %+v", r)
	}
	if ok := d.UpdateFields(999, func(*RID) {}); ok {
		t.Error("UpdateFields(999) should return false for unknown RID")
	}
}

func TestRIDDBDelete(t *testing.T) {
	d := NewRIDDB()
	d.Add(&RID{ID: 1})
	if !d.Delete(1) {
		t.Error("Delete(1) returned false")
	}
	if d.Delete(1) {
		t.Error("Delete(1) on absent RID should return false")
	}
	if d.Len() != 0 {
		t.Errorf("Len after delete = %d, want 0", d.Len())
	}
}

func TestLoadRIDCSV(t *testing.T) {
	csv := `Decimal,Alias,Description,Tag,Group,Owner,Priority,Lockout,Watch,Icon
207545,CPL-SMITH,Patrol corporal,Patrol,Bossier PD,Cpl. Smith,2,,,badge
207546,LOCKED,Stolen radio,,,,L,,,
207547,EXPLICIT,Decommissioned,,,,3,Y,,
207548,NOWATCH,,,,,,,no,
`
	d := NewRIDDB()
	n, err := d.LoadCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if n != 4 {
		t.Errorf("loaded %d, want 4", n)
	}
	smith := d.Lookup(207545)
	if smith == nil {
		t.Fatal("CPL-SMITH not loaded")
	}
	if smith.Alias != "CPL-SMITH" || smith.Priority != 2 || smith.Tag != "Patrol" ||
		smith.Group != "Bossier PD" || smith.Owner != "Cpl. Smith" || smith.Icon != "badge" ||
		!smith.Watch {
		t.Errorf("CPL-SMITH = %+v", smith)
	}

	lk := d.Lookup(207546)
	if lk == nil || !lk.Lockout {
		t.Errorf("Priority=L should set Lockout: %+v", lk)
	}

	ex := d.Lookup(207547)
	if ex == nil || !ex.Lockout || ex.Priority != 3 {
		t.Errorf("EXPLICIT = %+v", ex)
	}

	nw := d.Lookup(207548)
	if nw == nil || nw.Watch {
		t.Errorf("NOWATCH should have Watch=false: %+v", nw)
	}
}

func TestLoadRIDCSVAcceptsAlphaTagAndDECColumns(t *testing.T) {
	csv := `DEC,Alpha Tag
42,FORTYTWO
`
	d := NewRIDDB()
	n, err := d.LoadCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if n != 1 {
		t.Fatalf("loaded %d, want 1", n)
	}
	if r := d.Lookup(42); r == nil || r.Alias != "FORTYTWO" {
		t.Errorf("Lookup(42) = %+v", r)
	}
}

func TestLoadRIDCSVAcceptsIDColumn(t *testing.T) {
	csv := `ID,Alias
99,NINETYNINE
`
	d := NewRIDDB()
	n, err := d.LoadCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if n != 1 {
		t.Fatalf("loaded %d, want 1", n)
	}
}

func TestLoadRIDCSVRequiresIDColumn(t *testing.T) {
	csv := `Hex,Alias
2A,FORTYTWO
`
	d := NewRIDDB()
	_, err := d.LoadCSV(strings.NewReader(csv))
	if err == nil {
		t.Error("expected error for missing Decimal/DEC/ID column")
	}
}

func TestLoadRIDCSVDefaultsWatchTrue(t *testing.T) {
	// Bare CSV with no Watch column → every loaded RID watches by
	// default. This mirrors TalkgroupDB.LoadCSV's default-Scan=true.
	csv := `Decimal,Alias
1,A
2,B
`
	d := NewRIDDB()
	if _, err := d.LoadCSV(strings.NewReader(csv)); err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if r := d.Lookup(1); r == nil || !r.Watch {
		t.Errorf("Lookup(1) Watch = false, want true: %+v", r)
	}
	if r := d.Lookup(2); r == nil || !r.Watch {
		t.Errorf("Lookup(2) Watch = false, want true: %+v", r)
	}
}

func TestLoadRIDCSVEmptyFile(t *testing.T) {
	d := NewRIDDB()
	n, err := d.LoadCSV(strings.NewReader(""))
	if err != nil {
		t.Fatalf("LoadCSV(empty): %v", err)
	}
	if n != 0 {
		t.Errorf("loaded %d from empty file, want 0", n)
	}
}

func TestLoadRIDJSON(t *testing.T) {
	js := `[
		{"id":1,"alias":"FIRST"},
		{"id":2,"alias":"SECOND","watch":false},
		{"id":3,"alias":"THIRD","priority":7,"owner":"Sgt. Jones","group":"Fire"}
	]`
	d := NewRIDDB()
	n, err := d.LoadJSON(strings.NewReader(js))
	if err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if n != 3 {
		t.Errorf("loaded %d, want 3", n)
	}
	if r := d.Lookup(1); r == nil || !r.Watch || r.Alias != "FIRST" {
		t.Errorf("Lookup(1) = %+v", r)
	}
	if r := d.Lookup(2); r == nil || r.Watch {
		t.Errorf("Lookup(2) explicit watch=false: %+v", r)
	}
	if r := d.Lookup(3); r == nil || r.Priority != 7 || r.Owner != "Sgt. Jones" || r.Group != "Fire" {
		t.Errorf("Lookup(3) = %+v", r)
	}
}

func TestLoadRIDJSONInvalid(t *testing.T) {
	d := NewRIDDB()
	if _, err := d.LoadJSON(strings.NewReader("{not json")); err == nil {
		t.Error("expected error for malformed JSON")
	}
}

// TestLoadRIDCSVUTF16 proves a SDRTrunk-style UTF-16LE-with-BOM export is
// transcoded to clean UTF-8 instead of read as interleaved-NUL mojibake.
func TestLoadRIDCSVUTF16(t *testing.T) {
	const csv = "Decimal,Alias\n900007,DISPATCH\n"
	enc := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder()
	b, _, err := transform.Bytes(enc, []byte(csv))
	if err != nil {
		t.Fatalf("encode utf16: %v", err)
	}
	d := NewRIDDB()
	if _, err := d.LoadCSV(strings.NewReader(string(b))); err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	r := d.Lookup(900007)
	if r == nil || r.Alias != "DISPATCH" {
		t.Errorf("utf16 import = %+v, want Alias=DISPATCH", r)
	}
}

// TestLoadRIDStripsMojibake proves non-ASCII junk in alias fields is dropped
// rather than surfaced verbatim in the UI (issue #711).
func TestLoadRIDStripsMojibake(t *testing.T) {
	csv := "Decimal,Alias\n900007,DIS中PATCH\n"
	d := NewRIDDB()
	if _, err := d.LoadCSV(strings.NewReader(csv)); err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if r := d.Lookup(900007); r == nil || r.Alias != "DISPATCH" {
		t.Errorf("csv mojibake = %+v, want Alias=DISPATCH", r)
	}

	js := `[{"id":900008,"alias":"UN中IT","owner":"Sgt"}]`
	d2 := NewRIDDB()
	if _, err := d2.LoadJSON(strings.NewReader(js)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if r := d2.Lookup(900008); r == nil || r.Alias != "UNIT" || r.Owner != "Sgt" {
		t.Errorf("json mojibake = %+v, want Alias=UNIT Owner=Sgt", r)
	}
}
