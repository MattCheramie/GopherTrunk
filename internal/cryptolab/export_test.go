package cryptolab

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want Format
		err  bool
	}{
		{"", FormatText, false},
		{"text", FormatText, false},
		{"JSON", FormatJSON, false},
		{"jsonl", FormatJSONL, false},
		{"yml", FormatYAML, false},
		{"csv", FormatCSV, false},
		{"toml", FormatText, true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := ParseFormat(tc.in)
			if tc.err {
				if err == nil {
					t.Fatalf("ParseFormat(%q): want error", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFormat(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseFormat(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func sampleResult() *Result {
	r := &Result{Tool: "brute", Mode: "xor", Summary: "recovered 1 key"}
	r.AddField("keyspace", 256)
	r.AddFinding("key=0x42", 0.93, map[string]any{"key": "42"})
	r.Note("weak single-byte XOR only")
	return r
}

func TestWriteResultFormats(t *testing.T) {
	t.Parallel()
	for _, f := range []Format{FormatText, FormatJSON, FormatJSONL, FormatYAML, FormatCSV} {
		var buf bytes.Buffer
		if err := WriteResult(&buf, sampleResult(), f); err != nil {
			t.Fatalf("WriteResult(%d): %v", f, err)
		}
		if buf.Len() == 0 {
			t.Fatalf("WriteResult(%d): empty output", f)
		}
	}
}

func TestWriteResultText(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatText); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"brute/xor", "recovered 1 key", "keyspace:", "key=0x42", "note:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("text output missing %q:\n%s", want, out)
		}
	}
}
