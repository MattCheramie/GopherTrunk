package siglab

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// sampleResult is a fixed Result used to exercise every exporter.
func sampleResult() *Result {
	pass := true
	return &Result{
		Source:           "synth.cfile",
		Protocol:         "p25",
		SampleRateHz:     48000,
		PipelineRateHz:   48000,
		DurationSec:      5,
		TotalSamples:     240000,
		Symbols:          24000,
		EffectiveBaud:    4800,
		ExpectedBaud:     4800,
		BaudDeviationPct: 0,
		Locked:           true,
		LockLatencySec:   0.5,
		Lock:             &LockInfo{FrequencyHz: 851000000, Fields: map[string]any{"NAC": uint16(0x293)}},
		Grants: []GrantRecord{
			{OffsetSec: 1.2, GroupID: 100, SourceID: 5, FrequencyHz: 852000000},
		},
		Events: []EventRecord{
			{Seq: 0, OffsetSec: 0.5, Kind: "cc.locked", Fields: map[string]any{"NAC": uint16(0x293)}},
			{Seq: 1, OffsetSec: 1.2, Kind: "grant", Fields: map[string]any{"GroupID": uint32(100)}},
		},
		EventCounts:  map[string]int{"cc.locked": 1, "grant": 1},
		DecodeErrors: map[string]int{"nid-bch": 3},
		Verdict:      &Verdict{Pass: pass, Checks: []string{"ok: lock = true"}},
	}
}

func TestExportJSONRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatJSON); err != nil {
		t.Fatalf("WriteResult JSON: %v", err)
	}
	var got Result
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, buf.String())
	}
	if got.Protocol != "p25" || !got.Locked || got.Symbols != 24000 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.Lock == nil || got.Lock.FrequencyHz != 851000000 {
		t.Errorf("lock not preserved: %+v", got.Lock)
	}
}

func TestExportYAMLRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatYAML); err != nil {
		t.Fatalf("WriteResult YAML: %v", err)
	}
	var got Result
	if err := yaml.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, buf.String())
	}
	if got.Protocol != "p25" || len(got.Grants) != 1 {
		t.Errorf("yaml round-trip mismatch: %+v", got)
	}
}

func TestExportJSONL(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatJSONL); err != nil {
		t.Fatalf("WriteResult JSONL: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// 2 events + 1 summary line.
	if len(lines) != 3 {
		t.Fatalf("JSONL line count = %d, want 3:\n%s", len(lines), buf.String())
	}
	var last jsonlLine
	if err := json.Unmarshal([]byte(lines[2]), &last); err != nil {
		t.Fatalf("summary line: %v", err)
	}
	if last.Type != "summary" || last.Summary == nil {
		t.Errorf("last line not a summary: %s", lines[2])
	}
	if last.Summary != nil && len(last.Summary.Events) != 0 {
		t.Error("summary line should elide Events")
	}
}

func TestExportCSVSummary(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatCSVSummary); err != nil {
		t.Fatalf("WriteResult CSV: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("CSV summary lines = %d, want 2:\n%s", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "protocol") || !strings.Contains(lines[0], "effective_baud") {
		t.Errorf("header missing columns: %s", lines[0])
	}
	if !strings.Contains(lines[1], "p25") {
		t.Errorf("data row missing protocol: %s", lines[1])
	}
}

func TestExportCSVEvents(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatCSVEvents); err != nil {
		t.Fatalf("WriteResult CSV events: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// header + 2 events.
	if len(lines) != 3 {
		t.Fatalf("CSV events lines = %d, want 3:\n%s", len(lines), buf.String())
	}
	if !strings.Contains(lines[1], "cc.locked") {
		t.Errorf("first event row wrong: %s", lines[1])
	}
}

func TestParseFormat(t *testing.T) {
	cases := map[string]Format{
		"json": FormatJSON, "jsonl": FormatJSONL, "yaml": FormatYAML,
		"csv": FormatCSVSummary, "csv-events": FormatCSVEvents, "": FormatTextSummary,
	}
	for in, want := range cases {
		got, err := ParseFormat(in)
		if err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseFormat("bogus"); err == nil {
		t.Error("expected error for bogus format")
	}
}
