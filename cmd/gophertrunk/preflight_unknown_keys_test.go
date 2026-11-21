package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUnknownKeyWarningsNameTheMisplacedSection: the start-up warning for a
// section pasted under a trunking system names the line, the key, where it
// was found and what to do. A clean file and no file are silent.
func TestUnknownKeyWarningsNameTheMisplacedSection(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "config.yaml")
	doc := "trunking:\n  systems:\n    - name: s\n      protocol: p25\n      control_channels: [851000000]\n      recordings:\n        write_raw: true\n"
	if err := os.WriteFile(bad, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	got := unknownKeyWarnings(bad)
	if len(got) != 1 {
		t.Fatalf("warnings = %q, want one", got)
	}
	for _, part := range []string{"config line 6", `"recordings"`, "trunking.systems[]", "ignored", "left margin"} {
		if !strings.Contains(got[0], part) {
			t.Errorf("warning %q does not mention %q", got[0], part)
		}
	}

	good := filepath.Join(dir, "good.yaml")
	if err := os.WriteFile(good, []byte("recordings:\n  write_raw: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if w := unknownKeyWarnings(good); len(w) != 0 {
		t.Errorf("clean file warned: %q", w)
	}
	if w := unknownKeyWarnings(""); len(w) != 0 {
		t.Errorf("no config file warned: %q", w)
	}
}
