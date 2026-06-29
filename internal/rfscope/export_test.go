package rfscope

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
	"gopkg.in/yaml.v3"
)

func exportFixture() *Scene {
	sc := hierFixture()
	sc.Source = "fixture.cfile"
	sc.NoiseFloorDbFS = -92
	sc.Channels = []Channel{
		{FreqHz: 450_100_000, DominantClass: survey.ClassC4FM, BurstCount: 2, DutyCycle: 0.15, OccupancyPct: 15, BurstRatePerMin: 12, PeriodSec: 1.0},
	}
	sc.Anomalies = []Anomaly{{Severity: "warn", Kind: "intermittent", FreqHz: 450_100_000, Detail: "low duty cycle"}}
	return sc
}

func TestParseFormatRoundTrip(t *testing.T) {
	t.Parallel()
	cases := map[string]Format{
		"json": FormatJSON, "jsonl": FormatJSONL, "ndjson": FormatJSONL,
		"yaml": FormatYAML, "yml": FormatYAML, "csv": FormatCSV,
		"summary": FormatSummary, "txt": FormatSummary, "": FormatJSON,
	}
	for in, want := range cases {
		got, err := ParseFormat(in)
		if err != nil || got != want {
			t.Fatalf("ParseFormat(%q) = %v,%v want %v", in, got, err, want)
		}
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Fatalf("want error for unknown format")
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := Write(&buf, exportFixture(), FormatJSON); err != nil {
		t.Fatalf("Write: %v", err)
	}
	var got Scene
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Source != "fixture.cfile" || len(got.Bursts) != 3 {
		t.Fatalf("json round-trip mismatch: %+v", got)
	}
}

func TestWriteYAML(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := Write(&buf, exportFixture(), FormatYAML); err != nil {
		t.Fatalf("Write: %v", err)
	}
	var got Scene
	if err := yaml.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	if len(got.Bursts) != 3 {
		t.Fatalf("yaml round-trip: got %d bursts", len(got.Bursts))
	}
}

func TestWriteJSONL(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := Write(&buf, exportFixture(), FormatJSONL); err != nil {
		t.Fatalf("Write: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// 1 scene + 3 bursts + 1 channel + 1 anomaly = 6 records.
	if len(lines) != 6 {
		t.Fatalf("jsonl line count: got %d want 6\n%s", len(lines), buf.String())
	}
	var first jsonlRec
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil || first.Kind != "scene" {
		t.Fatalf("first record kind: %q err=%v", first.Kind, err)
	}
}

func TestWriteCSV(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := Write(&buf, exportFixture(), FormatCSV); err != nil {
		t.Fatalf("Write: %v", err)
	}
	rows, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("csv read: %v", err)
	}
	if len(rows) != 4 { // header + 3 bursts
		t.Fatalf("csv rows: got %d want 4", len(rows))
	}
	if rows[0][1] != "freq_hz" {
		t.Fatalf("csv header: %v", rows[0])
	}
}

func TestWriteSummary(t *testing.T) {
	t.Parallel()
	sc := exportFixture()
	// Populate the hierarchy so the summary renders it.
	_ = (hierarchyAnalyzer{}).Analyze(context.Background(), sc, nil)
	var buf bytes.Buffer
	if err := Write(&buf, sc, FormatSummary); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"RF Scene", "fixture.cfile", "Protocol hierarchy", "Channels:", "Expert info:", "intermittent"} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary missing %q:\n%s", want, out)
		}
	}
}
