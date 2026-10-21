package wmbus

import (
	"strings"
	"testing"
	"unicode"
)

// TestManufacturerVariableRecordPrintsHex: Qundis heat cost allocators (#1256
// field run, QDS 315671xx) send a variable-length record with VIF 0xFF (the
// manufacturer-specific VIF) whose LVAR says "text" but whose bytes are
// binary. It printed as raw bytes: control characters and broken UTF-8 on the
// terminal ("VIF 0xFF VIFE 5F /<binary>"). Binary data must print as hex, and
// the VIF is named for what it is.
func TestManufacturerVariableRecordPrintsHex(t *testing.T) {
	payload := []byte{0x1F, 0x00, 0x5F, 0x0A, 0x35, 0x39, 0x5E, 0x8C, 0x3C, 0x3F, 0x35, 0xC8}
	rec := append([]byte{0x0D, 0xFF, 0x5F, byte(len(payload))}, payload...)
	rs, err := ParseRecords(rec)
	if err != nil || len(rs) != 1 {
		t.Fatalf("ParseRecords = %v, %v", rs, err)
	}
	got := rs[0].String()
	for _, r := range got {
		if !unicode.IsPrint(r) || r == unicode.ReplacementChar {
			t.Fatalf("record prints a non-printable character %q: %q", r, got)
		}
	}
	if want := "manufacturer-specific VIFE 5F 1F005F0A35395E8C3C3F35C8"; got != want {
		t.Fatalf("record = %q, want %q", got, want)
	}
}

// TestVariableRecordKeepsPrintableText: a genuine ASCII variable-length
// record (last character first on air) still prints as text.
func TestVariableRecordKeepsPrintableText(t *testing.T) {
	rec := []byte{0x0D, 0x7C, 0x01, 'V', 0x05, 'O', 'L', 'L', 'E', 'H'}
	rs, err := ParseRecords(rec)
	if err != nil || len(rs) != 1 {
		t.Fatalf("ParseRecords = %v, %v", rs, err)
	}
	if got := rs[0].String(); !strings.HasSuffix(got, " HELLO") {
		t.Fatalf("record = %q, want text HELLO", got)
	}
}
