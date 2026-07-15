package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// TestStreamDecodeEmitsLiveJSONLEvents pins issue #314: the `replay -stream`
// core decodes a raw IQ stream read from an arbitrary reader (stdin in the CLI)
// and emits each decoded event as its own JSON line as it is produced — the
// contract an OpenWebRX+-style consumer piping IQ in needs. Uses the committed
// pre-centred 48 kHz f32 P25 control-channel fixture as the "stream".
func TestStreamDecodeEmitsLiveJSONLEvents(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "mmr-s9-cc.cfile"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	proto, err := siglab.ParseProtocolCLI("p25p1")
	if err != nil {
		t.Fatalf("parse protocol: %v", err)
	}
	cfg := siglab.Config{
		Protocol:             proto,
		SystemName:           "stream-test",
		SampleRateHz:         48_000,
		Format:               siglab.FormatF32,
		CollectReceiverState: true,
	}

	var buf bytes.Buffer
	if err := streamDecode(bytes.NewReader(raw), "stdin", cfg, &buf); err != nil {
		t.Fatalf("streamDecode: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("no events emitted from the stream")
	}

	// Every line must be a standalone, well-formed EventRecord JSON object —
	// line-delimited JSON, not a single buffered array.
	sawLock := false
	for i, ln := range lines {
		var ev siglab.EventRecord
		if err := json.Unmarshal([]byte(ln), &ev); err != nil {
			t.Fatalf("line %d is not valid EventRecord JSON: %v\n%s", i, err, ln)
		}
		if ev.Kind == "" {
			t.Errorf("line %d has empty kind: %s", i, ln)
		}
		if ev.Kind == "cc.locked" {
			sawLock = true
		}
	}
	// A real decode off the piped stream must reach control-channel lock —
	// proving the stream path actually decodes, not just echoes bytes.
	if !sawLock {
		t.Errorf("expected a cc.locked event from the decoded stream; got %d events without one", len(lines))
	}
}

// TestOpenIQInput covers the stdin sentinel, a real file, and a missing file.
func TestOpenIQInput(t *testing.T) {
	// "-" resolves to os.Stdin with a no-op closer.
	r, closeIn, err := openIQInput("-")
	if err != nil {
		t.Fatalf("openIQInput(-): %v", err)
	}
	if r != os.Stdin {
		t.Errorf("openIQInput(-) reader = %v, want os.Stdin", r)
	}
	if err := closeIn(); err != nil {
		t.Errorf("stdin closer returned %v, want nil", err)
	}

	// A real file opens and reads back its bytes.
	path := filepath.Join(t.TempDir(), "iq.bin")
	want := []byte{1, 2, 3, 4}
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}
	fr, closeF, err := openIQInput(path)
	if err != nil {
		t.Fatalf("openIQInput(file): %v", err)
	}
	got := make([]byte, len(want))
	if _, err := fr.Read(got); err != nil {
		t.Fatalf("read file reader: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("read %v, want %v", got, want)
	}
	if err := closeF(); err != nil {
		t.Errorf("file closer: %v", err)
	}

	// A missing file surfaces the open error.
	if _, _, err := openIQInput(filepath.Join(t.TempDir(), "nope.bin")); err == nil {
		t.Error("openIQInput(missing) returned nil error, want an open error")
	}
}

func TestIQSourceName(t *testing.T) {
	if got := iqSourceName("-"); got != "stdin" {
		t.Errorf("iqSourceName(-) = %q, want stdin", got)
	}
	if got := iqSourceName("/tmp/cap.cfile"); got != "/tmp/cap.cfile" {
		t.Errorf("iqSourceName(path) = %q, want the path", got)
	}
}
