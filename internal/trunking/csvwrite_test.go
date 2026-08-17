package trunking

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

// The point of the export is that the file loads back, so the round trip is
// the test: write → LoadCSV into a fresh DB → compare field for field. A
// column that stops loading fails here rather than in an operator's config.
func TestWriteTalkgroupCSVRoundTripsThroughLoadCSV(t *testing.T) {
	want := []*TalkGroup{
		{ID: 1020545, AlphaTag: "TAC-1", Description: "tactical", Tag: "ops",
			Group: "police", Mode: "D", Priority: 3, Scan: true},
		{ID: 900, AlphaTag: "DISPATCH", Group: "fire", Lockout: true, Scan: false},
		// Nothing but an id: an operator can name a talkgroup later.
		{ID: 42, Scan: true},
	}

	var buf bytes.Buffer
	if err := WriteTalkgroupCSV(&buf, want); err != nil {
		t.Fatal(err)
	}

	db := NewTalkgroupDB()
	n, err := db.LoadCSV(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(want) {
		t.Fatalf("loaded %d rows, wrote %d", n, len(want))
	}
	for _, w := range want {
		got := db.Lookup(w.ID)
		if got == nil {
			t.Fatalf("talkgroup %d did not survive the round trip", w.ID)
		}
		if got.AlphaTag != w.AlphaTag || got.Description != w.Description ||
			got.Tag != w.Tag || got.Group != w.Group || got.Mode != w.Mode ||
			got.Priority != w.Priority || got.Lockout != w.Lockout || got.Scan != w.Scan {
			t.Errorf("talkgroup %d round-tripped as %+v, want %+v", w.ID, *got, *w)
		}
	}
}

func TestWriteRIDCSVRoundTripsThroughLoadCSV(t *testing.T) {
	want := []*RID{
		{ID: 1005736, Alias: "CPL-SMITH", Description: "patrol", Tag: "unit",
			Group: "police", Owner: "badge-41", Priority: 2, Watch: true, Icon: "car"},
		{ID: 77, Alias: "OLD-RADIO", Lockout: true, Watch: false},
		{ID: 5, Watch: true},
	}

	var buf bytes.Buffer
	if err := WriteRIDCSV(&buf, want); err != nil {
		t.Fatal(err)
	}

	db := NewRIDDB()
	n, err := db.LoadCSV(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(want) {
		t.Fatalf("loaded %d rows, wrote %d", n, len(want))
	}
	for _, w := range want {
		got := db.Lookup(w.ID)
		if got == nil {
			t.Fatalf("rid %d did not survive the round trip", w.ID)
		}
		if got.Alias != w.Alias || got.Description != w.Description ||
			got.Tag != w.Tag || got.Group != w.Group || got.Owner != w.Owner ||
			got.Priority != w.Priority || got.Lockout != w.Lockout ||
			got.Watch != w.Watch || got.Icon != w.Icon {
			t.Errorf("rid %d round-tripped as %+v, want %+v", w.ID, *got, *w)
		}
	}
}

// The header has to match what `gophertrunk import` writes, or a file exported
// from a running daemon is not interchangeable with an imported one.
func TestTalkgroupCSVHeaderMatchesTheImporter(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTalkgroupCSV(&buf, nil); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	const want = "Decimal,Hex,Mode,Alpha Tag,Description,Tag,Group,Priority,Lockout,Scan"
	if got != want {
		t.Errorf("header = %q, want %q", got, want)
	}
}

// An empty catalogue must still produce a loadable file, not a zero-length one.
func TestWriteCSVEmptyCatalogueEmitsHeaderOnly(t *testing.T) {
	var tgBuf, ridBuf bytes.Buffer
	if err := WriteTalkgroupCSV(&tgBuf, nil); err != nil {
		t.Fatal(err)
	}
	if err := WriteRIDCSV(&ridBuf, nil); err != nil {
		t.Fatal(err)
	}
	for name, buf := range map[string]*bytes.Buffer{"talkgroup": &tgBuf, "rid": &ridBuf} {
		rows, err := csv.NewReader(bytes.NewReader(buf.Bytes())).ReadAll()
		if err != nil {
			t.Fatalf("%s export is not valid CSV: %v", name, err)
		}
		if len(rows) != 1 {
			t.Errorf("%s export has %d rows, want header only", name, len(rows))
		}
	}
}

// Catalogues are maps, so an export must impose an order or every run produces
// a different file and diffing two exports is useless.
func TestWriteCSVIsSortedByID(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteRIDCSV(&buf, []*RID{{ID: 30}, {ID: 10}, {ID: 20}}); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(buf.Bytes())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, r := range rows[1:] {
		ids = append(ids, r[0])
	}
	if strings.Join(ids, ",") != "10,20,30" {
		t.Errorf("row order = %v, want ascending id", ids)
	}
}
