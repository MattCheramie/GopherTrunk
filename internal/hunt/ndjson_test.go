package hunt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func TestNDJSONSinkRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survey.ndjson")
	sink, err := OpenNDJSONSink(path)
	if err != nil {
		t.Fatalf("OpenNDJSONSink: %v", err)
	}
	freqs := []uint32{451_000_000, 452_125_000, 453_250_000}
	for _, f := range freqs {
		if err := sink.Write(DetectedSignal{FreqHz: f, Class: survey.ClassNBFM}); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// One JSON object per line.
	raw, _ := os.ReadFile(path)
	if got := countLines(raw); got != len(freqs) {
		t.Errorf("line count = %d, want %d", got, len(freqs))
	}

	set, err := LoadSurveyedFreqs(path)
	if err != nil {
		t.Fatalf("LoadSurveyedFreqs: %v", err)
	}
	if len(set) != len(freqs) {
		t.Fatalf("loaded %d freqs, want %d", len(set), len(freqs))
	}
	for _, f := range freqs {
		if _, ok := set[f]; !ok {
			t.Errorf("freq %d missing from resume set", f)
		}
	}
}

func TestLoadSurveyedFreqs_MissingFileAndJunk(t *testing.T) {
	// Missing file ⇒ empty set, no error.
	set, err := LoadSurveyedFreqs(filepath.Join(t.TempDir(), "nope.ndjson"))
	if err != nil || len(set) != 0 {
		t.Fatalf("missing file: set=%v err=%v, want empty/no error", set, err)
	}

	// A half-written trailing line (crash) is skipped, valid lines still load.
	path := filepath.Join(t.TempDir(), "partial.ndjson")
	os.WriteFile(path, []byte(`{"freq_hz":451000000}`+"\n"+`{"freq_hz":4520`), 0o644)
	set, err = LoadSurveyedFreqs(path)
	if err != nil {
		t.Fatalf("LoadSurveyedFreqs: %v", err)
	}
	if len(set) != 1 {
		t.Errorf("loaded %d, want 1 (junk line skipped)", len(set))
	}
}

// TestRunLiveSurvey_ResumeSkips verifies a resumed survey skips candidates whose
// frequency is in SkipFreqs — they are never tuned and produce no signal.
func TestRunLiveSurvey_ResumeSkips(t *testing.T) {
	const a, b = 451_000_000, 452_000_000
	// Two candidates; map gives each a flat (analog-ish) buffer.
	buf := make([]complex64, 48_000)
	for i := range buf {
		buf[i] = complex(0.2, 0)
	}
	src := NewMappedIQSource(map[uint32][]complex64{a: buf, b: buf}, 48_000)

	var seen []uint32
	sv, _, err := RunLiveSurvey(context.Background(), LiveHuntOptions{
		Source:       src,
		Candidates:   []uint32{a, b},
		DwellSeconds: float64(len(buf)) / 48_000,
		ClassifyOnly: true,
		SkipFreqs:    map[uint32]struct{}{a: {}}, // resume: a already done
		OnSignal:     func(ds DetectedSignal) { seen = append(seen, ds.FreqHz) },
	})
	if err != nil {
		t.Fatalf("RunLiveSurvey: %v", err)
	}
	if len(sv.Signals) != 1 || sv.Signals[0].FreqHz != b {
		t.Errorf("signals = %+v, want only %d", sv.Signals, b)
	}
	if len(seen) != 1 || seen[0] != b {
		t.Errorf("OnSignal freqs = %v, want [%d]", seen, b)
	}
	for _, f := range src.TuneCalls() {
		if f == a {
			t.Errorf("skipped freq %d was tuned", a)
		}
	}
}

func countLines(b []byte) int {
	n := 0
	for _, c := range b {
		if c == '\n' {
			n++
		}
	}
	return n
}
