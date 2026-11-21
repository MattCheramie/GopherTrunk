package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// misnestedConfig is the shape of a real support case (18 Nov 2026): asked
// to switch on the debug log, raw sidecars and per-call voice IQ, the
// operator pasted the three sections under the trunking system. The file
// loaded, the daemon ran, and none of the three took effect ("the log and iq
// folder were left empty").
const misnestedConfig = `sdr:
  sample_rate: 2400000
  devices:
    - serial: "00000001"
      role: wideband
      center_freq_hz: 867475000
      channels:
        - frequency_hz: 867475000
          system: "p25-site"
trunking:
  systems:
    - name: "p25-site"
      protocol: p25
      control_channels:
        - 867475000
      log:
        file:
          enabled: true
          path: "../logs/debug.log"
      recordings:
        write_raw: true
        mbe_files: true
      baseband:
        voice_iq_debug:
          enabled: true
          dir: "../iq/voice-debug/"
recordings:
  dir: "recordings"
`

func TestUnknownKeysReportsSectionsNestedUnderASystem(t *testing.T) {
	got := UnknownKeys([]byte(misnestedConfig))
	want := []UnknownKey{
		{Line: 16, Key: "log", In: "trunking.systems[]", TopLevel: true},
		{Line: 20, Key: "recordings", In: "trunking.systems[]", TopLevel: true},
		{Line: 23, Key: "baseband", In: "trunking.systems[]", TopLevel: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d unknown keys %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("key %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	msg := got[0].String()
	for _, part := range []string{"line 16", `"log"`, "trunking.systems[]", "ignored", "top-level section", "left margin"} {
		if !strings.Contains(msg, part) {
			t.Errorf("message %q does not mention %q", msg, part)
		}
	}
	// The lenient load is what hid it: the file is otherwise valid.
	var cfg Config
	if err := Unmarshal([]byte(misnestedConfig), &cfg); err != nil {
		t.Fatalf("the mis-nested file must still load: %v", err)
	}
	if cfg.Log.File.Enabled || cfg.Baseband.VoiceIQDebug.Enabled {
		t.Fatal("the nested sections took effect; the fixture no longer shows the silent drop")
	}
}

func TestUnknownKeysReportsMisspelledAndTopLevelKeys(t *testing.T) {
	doc := "recording:\n  dir: x\nsdr:\n  sampel_rate: 2400000\n  devices:\n    - serial: \"1\"\n      gian: 200\n"
	got := UnknownKeys([]byte(doc))
	want := []UnknownKey{
		{Line: 1, Key: "recording"},
		{Line: 4, Key: "sampel_rate", In: "sdr"},
		{Line: 7, Key: "gian", In: "sdr.devices[]"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("key %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if s := got[0].String(); !strings.Contains(s, "at the top level") || strings.Contains(s, "left margin") {
		t.Errorf("top-level message %q", s)
	}
	if s := got[1].String(); !strings.Contains(s, "under sdr") || !strings.Contains(s, "misspelled") {
		t.Errorf("misspelling message %q", s)
	}
}

// TestUnknownKeysQuietOnShippedConfigs keeps the check free of false alarms:
// every config the repository ships is clean. Every setting is a plain
// struct field (no inline maps, no custom unmarshalers), so the strict
// decode behind UnknownKeys knows exactly the keys Load reads, and a
// warning always means the operator's file has a key nothing reads.
func TestUnknownKeysQuietOnShippedConfigs(t *testing.T) {
	files := []string{"../../config.example.yaml"}
	samples, err := filepath.Glob("../../samples/*/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, samples...)
	if len(files) < 2 {
		t.Fatalf("found only %v; the sample configs moved", files)
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, k := range UnknownKeys(data) {
			t.Errorf("%s: %s", f, k)
		}
	}
}

func TestUnknownKeysNilForUnparseableOrMissing(t *testing.T) {
	if got := UnknownKeys([]byte("sdr: [unclosed")); got != nil {
		t.Errorf("unparseable document: got %+v, want nil (Load reports the syntax error)", got)
	}
	if got := UnknownKeysInFile(""); got != nil {
		t.Errorf("empty path: got %+v", got)
	}
	if got := UnknownKeysInFile(filepath.Join(t.TempDir(), "absent.yaml")); got != nil {
		t.Errorf("missing file: got %+v", got)
	}
}
