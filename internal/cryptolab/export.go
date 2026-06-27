package cryptolab

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format selects how a Result is serialized. Mirrors the siglab export
// surface (text summary plus structured json/jsonl/yaml/csv).
type Format int

const (
	// FormatText is the human-readable summary (default).
	FormatText Format = iota
	// FormatJSON is a single pretty JSON object.
	FormatJSON
	// FormatJSONL is one JSON object per finding, then a summary object —
	// convenient for streaming/grep over large survivor sets.
	FormatJSONL
	// FormatYAML is a single YAML document.
	FormatYAML
	// FormatCSV is one row per finding (label,score,detail-json).
	FormatCSV
)

// ParseFormat maps a CLI string to a Format.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "jsonl":
		return FormatJSONL, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "csv":
		return FormatCSV, nil
	default:
		return FormatText, fmt.Errorf("unknown output format %q (text|json|jsonl|yaml|csv)", s)
	}
}

// WriteResult renders r to w in the requested format.
func WriteResult(w io.Writer, r *Result, f Format) error {
	if r == nil {
		return nil
	}
	switch f {
	case FormatText:
		return writeText(w, r)
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	case FormatJSONL:
		return writeJSONL(w, r)
	case FormatYAML:
		return yaml.NewEncoder(w).Encode(r)
	case FormatCSV:
		return writeCSV(w, r)
	default:
		return fmt.Errorf("cryptolab: unhandled format %d", f)
	}
}

func writeText(w io.Writer, r *Result) error {
	var b strings.Builder
	fmt.Fprintf(&b, "cryptolab %s/%s\n", r.Tool, r.Mode)
	if r.Summary != "" {
		fmt.Fprintf(&b, "  %s\n", r.Summary)
	}
	for _, f := range r.Fields {
		fmt.Fprintf(&b, "  %-22s %v\n", f.Key+":", f.Value)
	}
	if len(r.Findings) > 0 {
		fmt.Fprintf(&b, "  findings (%d):\n", len(r.Findings))
		for i, fd := range r.Findings {
			fmt.Fprintf(&b, "    %2d. %-28s score=%.4g\n", i+1, fd.Label, fd.Score)
		}
	}
	for _, n := range r.Notes {
		fmt.Fprintf(&b, "  note: %s\n", n)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeJSONL(w io.Writer, r *Result) error {
	enc := json.NewEncoder(w)
	for _, fd := range r.Findings {
		if err := enc.Encode(fd); err != nil {
			return err
		}
	}
	// Trailing summary object (findings omitted to avoid duplication).
	summary := *r
	summary.Findings = nil
	return enc.Encode(summary)
}

func writeCSV(w io.Writer, r *Result) error {
	if _, err := io.WriteString(w, "label,score,detail\n"); err != nil {
		return err
	}
	for _, fd := range r.Findings {
		detail := ""
		if len(fd.Detail) > 0 {
			b, err := json.Marshal(fd.Detail)
			if err != nil {
				return err
			}
			detail = strings.ReplaceAll(string(b), `"`, `""`)
		}
		if _, err := fmt.Fprintf(w, "%s,%g,\"%s\"\n", csvEscape(fd.Label), fd.Score, detail); err != nil {
			return err
		}
	}
	return nil
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
