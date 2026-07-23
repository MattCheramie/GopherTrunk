package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func sampleSurvey() *hunt.SignalSurvey {
	return &hunt.SignalSurvey{Signals: []hunt.DetectedSignal{
		{FreqHz: 462_562_500, Class: survey.ClassNBFM, OccupiedBwHz: 12_500, Name: "nbfm", Service: "FRS / GMRS"},
		{FreqHz: 851_012_500, Class: survey.ClassTrunkControl, OccupiedBwHz: 12_500, Name: "P25 Phase 1",
			Trunking: &hunt.TrunkingRef{Protocol: "p25", Locked: true}},
		{FreqHz: 751_000_000, Class: survey.ClassWideband, OccupiedBwHz: 10_000_000, Name: "LTE/5G NR (10 MHz)", Wideband: true},
	}}
}

func TestSelectSurveySignal(t *testing.T) {
	sv := sampleSurvey()

	// By index.
	if s, err := selectSurveySignal(sv, "#1"); err != nil || s.FreqHz != 851_012_500 {
		t.Errorf("#1 = %+v, err %v; want the P25 row", s, err)
	}
	// By frequency in MHz (nearest within tolerance).
	if s, err := selectSurveySignal(sv, "462.5625"); err != nil || s.Name != "nbfm" {
		t.Errorf("462.5625 = %+v, err %v; want the FRS row", s, err)
	}
	// Wideband row matches even a few hundred kHz off-centre (tolerance scales
	// with its bandwidth).
	if s, err := selectSurveySignal(sv, "752.0"); err != nil || !s.Wideband {
		t.Errorf("752.0 = %+v, err %v; want the wideband row", s, err)
	}
	// Out-of-range index and a far-off frequency both error.
	if _, err := selectSurveySignal(sv, "#9"); err == nil {
		t.Error("#9: want out-of-range error")
	}
	if _, err := selectSurveySignal(sv, "100.0"); err == nil {
		t.Error("100.0 MHz: want no-match error")
	}
}

func TestLoadSurveyJSONRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "survey.json")
	raw, _ := json.Marshal(sampleSurvey())
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	sv, err := loadSurveyJSON(path)
	if err != nil {
		t.Fatalf("loadSurveyJSON: %v", err)
	}
	if len(sv.Signals) != 3 {
		t.Errorf("loaded %d signals, want 3", len(sv.Signals))
	}
	if _, err := loadSurveyJSON(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing file: want error")
	}
}

func TestCaptureSignalToFile(t *testing.T) {
	dir := t.TempDir()
	// A float32 IQ fixture for the mock f32 driver to replay.
	const fixtureSamples = 8192
	iq := make([]complex64, fixtureSamples)
	buf := siglab.EncodeCapture(iq, siglab.FormatF32)
	fixture := filepath.Join(dir, "fixture.cfile")
	if err := os.WriteFile(fixture, buf, 0o644); err != nil {
		t.Fatal(err)
	}

	dev, err := (&sdr.MockFloat32Driver{Files: []string{fixture}}).Open(0)
	if err != nil {
		t.Fatalf("open mock: %v", err)
	}
	defer dev.Close()

	out := filepath.Join(dir, "out.cfile")
	const rate = 48_000
	written, err := captureSignalToFile(dev, 751_000_000, rate, -1, 0, out, 0.05) // ~2400 samples
	if err != nil {
		t.Fatalf("captureSignalToFile: %v", err)
	}
	if written < int64(0.05*rate) {
		t.Errorf("wrote %d samples, want ≥ %d", written, int64(0.05*rate))
	}
	fi, err := os.Stat(out)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("capture file missing/empty: %v", err)
	}
}
