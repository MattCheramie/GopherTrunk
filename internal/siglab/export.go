package siglab

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Format selects an output encoding for a Result.
type Format int

const (
	// FormatTextSummary is the human-readable summary (handled by the
	// caller's renderer, not WriteResult). Defined so callers share one
	// Format enum.
	FormatTextSummary Format = iota
	// FormatJSON is a single indented JSON object.
	FormatJSON
	// FormatJSONL is one JSON object per line: every captured event, then a
	// final summary object tagged "summary".
	FormatJSONL
	// FormatYAML is a single YAML document.
	FormatYAML
	// FormatCSVSummary is a two-line CSV (header + one summary row).
	FormatCSVSummary
	// FormatCSVEvents is a CSV with one row per captured event.
	FormatCSVEvents
)

// ParseFormat maps a -format flag value to a Format. csv defaults to the
// summary shape; use "csv-events" for the per-event shape.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "", "text":
		return FormatTextSummary, nil
	case "json":
		return FormatJSON, nil
	case "jsonl":
		return FormatJSONL, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "csv", "csv-summary":
		return FormatCSVSummary, nil
	case "csv-events":
		return FormatCSVEvents, nil
	default:
		return FormatTextSummary, fmt.Errorf("siglab: unknown format %q (want json|jsonl|yaml|csv|csv-events)", s)
	}
}

// WriteResult serializes r to w in the requested format. FormatTextSummary
// is not handled here (the subcommands own their text rendering); it returns
// an error so a miswired caller is caught.
func WriteResult(w io.Writer, r *Result, f Format) error {
	switch f {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	case FormatJSONL:
		return writeJSONL(w, r)
	case FormatYAML:
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		if err := enc.Encode(r); err != nil {
			return err
		}
		return enc.Close()
	case FormatCSVSummary:
		return writeCSVSummary(w, r)
	case FormatCSVEvents:
		return writeCSVEvents(w, r)
	case FormatTextSummary:
		return fmt.Errorf("siglab: text summary is rendered by the caller, not WriteResult")
	default:
		return fmt.Errorf("siglab: unsupported format %d", f)
	}
}

// writeJSONL emits one JSON line per captured event followed by a summary
// line. The summary reuses the Result with its (already-streamed) Events
// elided so the trailing object stays compact.
func writeJSONL(w io.Writer, r *Result) error {
	enc := json.NewEncoder(w)
	for _, ev := range r.Events {
		if err := enc.Encode(jsonlLine{Type: "event", Event: &ev}); err != nil {
			return err
		}
	}
	summary := *r
	summary.Events = nil
	return enc.Encode(jsonlLine{Type: "summary", Summary: &summary})
}

type jsonlLine struct {
	Type    string       `json:"type"`
	Event   *EventRecord `json:"event,omitempty"`
	Summary *Result      `json:"summary,omitempty"`
}

// writeCSVSummary writes a stable single-row summary CSV.
func writeCSVSummary(w io.Writer, r *Result) error {
	cw := csv.NewWriter(w)
	header := []string{
		"source", "protocol", "sample_rate_hz", "pipeline_rate_hz", "duration_sec",
		"total_samples", "symbols", "effective_baud", "expected_baud", "baud_deviation_pct",
		"locked", "lock_latency_sec", "lock_frequency_hz", "grants", "decode_errors", "verdict_pass",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	lockFreq := ""
	if r.Lock != nil {
		lockFreq = strconv.FormatUint(uint64(r.Lock.FrequencyHz), 10)
	}
	verdict := ""
	if r.Verdict != nil {
		verdict = strconv.FormatBool(r.Verdict.Pass)
	}
	row := []string{
		r.Source, r.Protocol,
		ftoa(r.SampleRateHz), ftoa(r.PipelineRateHz), ftoa(r.DurationSec),
		strconv.FormatInt(r.TotalSamples, 10), strconv.FormatInt(r.Symbols, 10),
		ftoa(r.EffectiveBaud), ftoa(r.ExpectedBaud), ftoa(r.BaudDeviationPct),
		strconv.FormatBool(r.Locked), ftoa(r.LockLatencySec), lockFreq,
		strconv.Itoa(len(r.Grants)), strconv.Itoa(sumCounts(r.DecodeErrors)), verdict,
	}
	if err := cw.Write(row); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

// writeCSVEvents writes one row per captured event; the protocol-specific
// payload fields are JSON-encoded into a single stable column so the column
// set is identical across all protocols.
func writeCSVEvents(w io.Writer, r *Result) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"seq", "offset_sec", "kind", "fields_json"}); err != nil {
		return err
	}
	for _, ev := range r.Events {
		fields, err := json.Marshal(ev.Fields)
		if err != nil {
			return err
		}
		row := []string{
			strconv.Itoa(ev.Seq), ftoa(ev.OffsetSec), ev.Kind, string(fields),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func sumCounts(m map[string]int) int {
	total := 0
	for _, n := range m {
		total += n
	}
	return total
}

// SortedDecodeErrors returns the decode-error stages in a stable order, for
// deterministic text/CSV rendering by callers.
func SortedDecodeErrors(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
